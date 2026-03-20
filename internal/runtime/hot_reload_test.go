package runtime

import (
	"reflect"
	"strings"
	"testing"
)

var hotReloadDiagnosticUsesRef bool

func hotReloadDiagnosticComponent() *Element {
	GoUseState[int](nil, 1)
	if hotReloadDiagnosticUsesRef {
		GoUseRef("mismatch")
	} else {
		GoUseId()
	}
	return CreateElement("div", nil, "ok")
}

func TestRenderFunctionComponentReportsHotReloadFallbackDiagnostic(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	rt := &Runtime{}
	root := &Fiber{typeOf: "ROOT"}

	hotReloadDiagnosticUsesRef = false
	initialFiber := &Fiber{typeOf: hotReloadDiagnosticComponent, parent: root}
	element, handled, next := rt.renderFunctionComponent(initialFiber)
	if handled || next != nil || element == nil {
		t.Fatalf("expected initial render to succeed, got handled=%v next=%v element=%v", handled, next, element)
	}

	snapshot := captureHotReloadComponentSnapshot(initialFiber)
	if snapshot == nil {
		t.Fatal("expected initial hot reload snapshot")
	}

	rt.RestoreHotReloadSnapshot(HotReloadSnapshot{Components: []HotReloadComponentSnapshot{*snapshot}})

	hotReloadDiagnosticUsesRef = true
	reloadFiber := &Fiber{typeOf: hotReloadDiagnosticComponent, parent: root}
	element, handled, next = rt.renderFunctionComponent(reloadFiber)
	if handled || next != nil || element == nil {
		t.Fatalf("expected fallback render to succeed, got handled=%v next=%v element=%v", handled, next, element)
	}

	diagnostics := GetDiagnostics()
	if len(diagnostics) != 1 {
		t.Fatalf("expected one hot reload diagnostic, got %d", len(diagnostics))
	}
	if diagnostics[0].Severity != DiagnosticWarning {
		t.Fatalf("expected warning diagnostic, got %+v", diagnostics[0])
	}
	if diagnostics[0].Classification != DiagnosticUnsupportedRecover {
		t.Fatalf("expected unsupported-recovered classification, got %+v", diagnostics[0])
	}
	if !strings.Contains(diagnostics[0].Message, "hot reload fell back to remount") {
		t.Fatalf("expected fallback diagnostic message, got %+v", diagnostics[0])
	}
	if !strings.Contains(diagnostics[0].Message, "hook order changed") {
		t.Fatalf("expected hook-order reason in diagnostic, got %+v", diagnostics[0])
	}
	if diagnostics[0].Path != "hotReloadDiagnosticComponent" {
		t.Fatalf("expected component path in diagnostic, got %+v", diagnostics[0])
	}
	if len(diagnostics[0].ComponentStack) != 1 || diagnostics[0].ComponentStack[0] != "hotReloadDiagnosticComponent" {
		t.Fatalf("expected component stack in diagnostic, got %+v", diagnostics[0])
	}

	logs := GetLogs()
	if len(logs) != 1 {
		t.Fatalf("expected one diagnostic log, got %d", len(logs))
	}
	if logs[0].Classification != DiagnosticUnsupportedRecover {
		t.Fatalf("expected unsupported-recovered log classification, got %+v", logs[0])
	}
	if logs[0].Fields["path"] != "hotReloadDiagnosticComponent" {
		t.Fatalf("expected diagnostic log path, got %+v", logs[0])
	}
	if logs[0].Fields["component_stack"] != "hotReloadDiagnosticComponent" {
		t.Fatalf("expected diagnostic log component stack, got %+v", logs[0])
	}

	hotReloadDiagnosticUsesRef = false
}

var hotReloadSerializableMigrationAddsEffect bool

func hotReloadSerializableMigrationComponent() *Element {
	GoUseState[int](nil, 7)
	if hotReloadSerializableMigrationAddsEffect {
		GoUseEffect(func() func() { return nil }, nil)
	}
	GoUseId()
	return CreateElement("div", nil, "ok")
}

func TestRenderFunctionComponentPreservesSerializableStateAcrossEffectShapeChange(t *testing.T) {
	rt := &Runtime{}
	root := &Fiber{typeOf: "ROOT"}

	hotReloadSerializableMigrationAddsEffect = false
	initialFiber := &Fiber{typeOf: hotReloadSerializableMigrationComponent, parent: root}
	element, handled, next := rt.renderFunctionComponent(initialFiber)
	if handled || next != nil || element == nil {
		t.Fatalf("expected initial render to succeed, got handled=%v next=%v element=%v", handled, next, element)
	}
	initialID := initialFiber.hooks.ids[0]

	snapshot := captureHotReloadComponentSnapshot(initialFiber)
	if snapshot == nil {
		t.Fatal("expected initial hot reload snapshot")
	}

	rt.RestoreHotReloadSnapshot(HotReloadSnapshot{Components: []HotReloadComponentSnapshot{*snapshot}})

	hotReloadSerializableMigrationAddsEffect = true
	reloadFiber := &Fiber{typeOf: hotReloadSerializableMigrationComponent, parent: root}
	element, handled, next = rt.renderFunctionComponent(reloadFiber)
	if handled || next != nil || element == nil {
		t.Fatalf("expected reload render to succeed, got handled=%v next=%v element=%v", handled, next, element)
	}

	restoredState, ok := reloadFiber.hooks.states[0].(int)
	if !ok || restoredState != 7 {
		t.Fatalf("expected restored state 7 after effect insertion, got %#v", reloadFiber.hooks.states)
	}
	if got := reloadFiber.hooks.ids[0]; got != initialID {
		t.Fatalf("expected restored id %q, got %q", initialID, got)
	}
}

type hotReloadPrepareWrapper struct {
	released *int
}

func (w hotReloadPrepareWrapper) Release() {
	if w.released != nil {
		*w.released += 1
	}
}

func TestPrepareForHotReloadRunsCleanupsAndReleasesWrappers(t *testing.T) {
	released := 0
	cleanupRuns := 0
	rt := &Runtime{}
	leaf := &Fiber{
		typeOf: "div",
		hooks: &Hooks{
			cleanups: []func(){func() { cleanupRuns++ }},
			funcs:    []funcHandlerValue{{wrapper: hotReloadPrepareWrapper{released: &released}}},
		},
	}
	root := &Fiber{typeOf: "ROOT", child: leaf}
	leaf.parent = root
	rt.currentRoot = root
	rt.updateScheduled = true

	rt.PrepareForHotReload()

	if cleanupRuns != 1 {
		t.Fatalf("expected one cleanup run, got %d", cleanupRuns)
	}
	if released != 1 {
		t.Fatalf("expected one wrapper release, got %d", released)
	}
	if len(leaf.hooks.cleanups) != 1 || leaf.hooks.cleanups[0] != nil {
		t.Fatalf("expected cleanups to be cleared, got %#v", leaf.hooks.cleanups)
	}
	if !reflect.DeepEqual(leaf.hooks.funcs[0], funcHandlerValue{}) {
		t.Fatalf("expected function wrapper to be cleared, got %#v", leaf.hooks.funcs[0])
	}
	if rt.updateScheduled {
		t.Fatal("expected prepare for hot reload to clear update scheduling")
	}
}
