package runtime

import (
	"reflect"
	"strings"
	"testing"
)

var hotReloadDiagnosticUsesRef bool

func hotReloadDiagnosticComponent() *Element {
	GoUseState(nil, 1)
	if hotReloadDiagnosticUsesRef {
		GoUseRef("mismatch")
	} else {
		GoUseId()
	}
	return CreateElement("div", nil, "ok")
}

func TestRenderFunctionComponentReportsHotReloadFallbackDiagnostic(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseRt := &Runtime{}
	parseRoot := &Fiber{typeOf: "ROOT"}

	hotReloadDiagnosticUsesRef = false
	parseInitialFiber := &Fiber{typeOf: hotReloadDiagnosticComponent, parent: parseRoot}
	parseElement, parseHandled, parseNext := parseRt.renderFunctionComponent(parseInitialFiber)
	if parseHandled || parseNext != nil || parseElement == nil {
		parseT.Fatalf("expected initial render to succeed, got handled=%v next=%v element=%v", parseHandled, parseNext, parseElement)
	}

	parseSnapshot := captureHotReloadComponentSnapshot(parseInitialFiber)
	if parseSnapshot == nil {
		parseT.Fatal("expected initial hot reload snapshot")
	}

	parseRt.RestoreHotReloadSnapshot(HotReloadSnapshot{Components: []HotReloadComponentSnapshot{*parseSnapshot}})

	hotReloadDiagnosticUsesRef = true
	parseReloadFiber := &Fiber{typeOf: hotReloadDiagnosticComponent, parent: parseRoot}
	parseElement, parseHandled, parseNext = parseRt.renderFunctionComponent(parseReloadFiber)
	if parseHandled || parseNext != nil || parseElement == nil {
		parseT.Fatalf("expected fallback render to succeed, got handled=%v next=%v element=%v", parseHandled, parseNext, parseElement)
	}

	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) != 1 {
		parseT.Fatalf("expected one hot reload diagnostic, got %d", len(parseDiagnostics))
	}
	if parseDiagnostics[0].Severity != DiagnosticWarning {
		parseT.Fatalf("expected warning diagnostic, got %+v", parseDiagnostics[0])
	}
	if parseDiagnostics[0].Classification != DiagnosticUnsupportedRecover {
		parseT.Fatalf("expected unsupported-recovered classification, got %+v", parseDiagnostics[0])
	}
	if !strings.Contains(parseDiagnostics[0].Message, "hot reload fell back to remount") {
		parseT.Fatalf("expected fallback diagnostic message, got %+v", parseDiagnostics[0])
	}
	if !strings.Contains(parseDiagnostics[0].Message, "hook order changed") {
		parseT.Fatalf("expected hook-order reason in diagnostic, got %+v", parseDiagnostics[0])
	}
	if parseDiagnostics[0].Path != "hotReloadDiagnosticComponent" {
		parseT.Fatalf("expected component path in diagnostic, got %+v", parseDiagnostics[0])
	}
	if len(parseDiagnostics[0].ComponentStack) != 1 || parseDiagnostics[0].ComponentStack[0] != "hotReloadDiagnosticComponent" {
		parseT.Fatalf("expected component stack in diagnostic, got %+v", parseDiagnostics[0])
	}

	parseLogs := GetLogs()
	if len(parseLogs) != 1 {
		parseT.Fatalf("expected one diagnostic log, got %d", len(parseLogs))
	}
	if parseLogs[0].Classification != DiagnosticUnsupportedRecover {
		parseT.Fatalf("expected unsupported-recovered log classification, got %+v", parseLogs[0])
	}
	if parseLogs[0].Fields["path"] != "hotReloadDiagnosticComponent" {
		parseT.Fatalf("expected diagnostic log path, got %+v", parseLogs[0])
	}
	if parseLogs[0].Fields["component_stack"] != "hotReloadDiagnosticComponent" {
		parseT.Fatalf("expected diagnostic log component stack, got %+v", parseLogs[0])
	}

	hotReloadDiagnosticUsesRef = false
}

var hotReloadSerializableMigrationAddsEffect bool

func hotReloadSerializableMigrationComponent() *Element {
	GoUseState(nil, 7)
	if hotReloadSerializableMigrationAddsEffect {
		GoUseEffect(func() func() { return nil }, nil)
	}
	GoUseId()
	return CreateElement("div", nil, "ok")
}

func TestRenderFunctionComponentPreservesSerializableStateAcrossEffectShapeChange(parseT *testing.T) {
	parseRt := &Runtime{}
	parseRoot := &Fiber{typeOf: "ROOT"}

	hotReloadSerializableMigrationAddsEffect = false
	parseInitialFiber := &Fiber{typeOf: hotReloadSerializableMigrationComponent, parent: parseRoot}
	parseElement, parseHandled, parseNext := parseRt.renderFunctionComponent(parseInitialFiber)
	if parseHandled || parseNext != nil || parseElement == nil {
		parseT.Fatalf("expected initial render to succeed, got handled=%v next=%v element=%v", parseHandled, parseNext, parseElement)
	}
	parseInitialID := parseInitialFiber.hooks.ids[0]

	parseSnapshot := captureHotReloadComponentSnapshot(parseInitialFiber)
	if parseSnapshot == nil {
		parseT.Fatal("expected initial hot reload snapshot")
	}

	parseRt.RestoreHotReloadSnapshot(HotReloadSnapshot{Components: []HotReloadComponentSnapshot{*parseSnapshot}})

	hotReloadSerializableMigrationAddsEffect = true
	parseReloadFiber := &Fiber{typeOf: hotReloadSerializableMigrationComponent, parent: parseRoot}
	parseElement, parseHandled, parseNext = parseRt.renderFunctionComponent(parseReloadFiber)
	if parseHandled || parseNext != nil || parseElement == nil {
		parseT.Fatalf("expected reload render to succeed, got handled=%v next=%v element=%v", parseHandled, parseNext, parseElement)
	}

	parseRestoredState, parseOk := parseReloadFiber.hooks.states[0].(int)
	if !parseOk || parseRestoredState != 7 {
		parseT.Fatalf("expected restored state 7 after effect insertion, got %#v", parseReloadFiber.hooks.states)
	}
	if parseGot := parseReloadFiber.hooks.ids[0]; parseGot != parseInitialID {
		parseT.Fatalf("expected restored id %q, got %q", parseInitialID, parseGot)
	}
}

type hotReloadPrepareWrapper struct {
	released *int
}

func (parseW hotReloadPrepareWrapper) Release() {
	if parseW.released != nil {
		*parseW.released += 1
	}
}

func TestPrepareForHotReloadRunsCleanupsAndReleasesWrappers(parseT *testing.T) {
	parseReleased := 0
	parseCleanupRuns := 0
	parseRt := &Runtime{}
	parseLeaf := &Fiber{
		typeOf: "div",
		hooks: &Hooks{
			cleanups: []func(){func() { parseCleanupRuns++ }},
			funcs:    []funcHandlerValue{{wrapper: hotReloadPrepareWrapper{released: &parseReleased}}},
		},
	}
	parseRoot := &Fiber{typeOf: "ROOT", child: parseLeaf}
	parseLeaf.parent = parseRoot
	parseRt.currentRoot = parseRoot
	parseRt.updateScheduled = true

	parseRt.PrepareForHotReload()

	if parseCleanupRuns != 1 {
		parseT.Fatalf("expected one cleanup run, got %d", parseCleanupRuns)
	}
	if parseReleased != 1 {
		parseT.Fatalf("expected one wrapper release, got %d", parseReleased)
	}
	if len(parseLeaf.hooks.cleanups) != 1 || parseLeaf.hooks.cleanups[0] != nil {
		parseT.Fatalf("expected cleanups to be cleared, got %#v", parseLeaf.hooks.cleanups)
	}
	if !reflect.DeepEqual(parseLeaf.hooks.funcs[0], funcHandlerValue{}) {
		parseT.Fatalf("expected function wrapper to be cleared, got %#v", parseLeaf.hooks.funcs[0])
	}
	if parseRt.updateScheduled {
		parseT.Fatal("expected prepare for hot reload to clear update scheduling")
	}
}

func TestPrepareForHotReloadReportsPendingFetchRestartActivity(parseT *testing.T) {
	ClearLogs()
	defer ClearLogs()

	parseRt := &Runtime{}
	parseLeaf := &Fiber{
		typeOf: "div",
		hooks: &Hooks{
			fetches: []fetchValue{{
				url:   "/api/orders",
				state: FetchState{Loading: true},
			}},
		},
	}
	parseRoot := &Fiber{typeOf: "ROOT", child: parseLeaf}
	parseLeaf.parent = parseRoot
	parseRt.currentRoot = parseRoot

	parseRt.PrepareForHotReload()

	parseLogs := GetLogs()
	if len(parseLogs) != 1 {
		parseT.Fatalf("expected one hot reload activity log, got %d", len(parseLogs))
	}
	if parseLogs[0].Domain != "hotreload" {
		parseT.Fatalf("expected hotreload log domain, got %+v", parseLogs[0])
	}
	if parseLogs[0].Message != "pending fetch will restart on hot reload" {
		parseT.Fatalf("expected fetch restart message, got %+v", parseLogs[0])
	}
	if parseLogs[0].Fields["url"] != "/api/orders" {
		parseT.Fatalf("expected fetch url field, got %+v", parseLogs[0])
	}
}

func TestSelectiveHotReloadRestoreRemountsChangedSubtreeOnly(parseT *testing.T) {
	parseRt := &Runtime{}
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseApp := &Fiber{typeOf: NewComponentType("example/App", "App", "example/App", nil, nil), parent: parseRoot, hooks: &Hooks{signature: []string{"state"}, states: []any{1, nil}}}
	parseChanged := &Fiber{typeOf: NewComponentType("example/Changed", "Changed", "example/Changed", nil, nil), parent: parseApp, hooks: &Hooks{signature: []string{"state"}, states: []any{2, nil}}}
	parseStable := &Fiber{typeOf: NewComponentType("example/Stable", "Stable", "example/Stable", nil, nil), parent: parseApp, hooks: &Hooks{signature: []string{"state"}, states: []any{3, nil}}}
	parseRoot.child = parseApp
	parseApp.child = parseChanged
	parseChanged.sibling = parseStable

	parseSnapshot := HotReloadSnapshot{}
	captureHotReloadComponentSnapshots(parseRoot, &parseSnapshot.Components)
	parseDecision := parseRt.RestoreHotReloadSnapshotWithPlan(parseSnapshot, HotReloadRestorePlan{
		Selective:         true,
		ChangedIdentities: []string{"example/Changed"},
	})
	if parseDecision.Strategy != "selective" {
		parseT.Fatalf("expected selective restore strategy, got %+v", parseDecision)
	}

	parseReloadRoot := &Fiber{typeOf: "ROOT"}
	parseReloadApp := &Fiber{typeOf: NewComponentType("example/App", "App", "example/App", nil, nil), parent: parseReloadRoot}
	parseReloadChanged := &Fiber{typeOf: NewComponentType("example/Changed", "Changed", "example/Changed", nil, nil), parent: parseReloadApp}
	parseReloadStable := &Fiber{typeOf: NewComponentType("example/Stable", "Stable", "example/Stable", nil, nil), parent: parseReloadApp}
	parseReloadRoot.child = parseReloadApp
	parseReloadApp.child = parseReloadChanged
	parseReloadChanged.sibling = parseReloadStable

	if parseGot := parseRt.matchingHotReloadComponentSnapshot(parseReloadApp); parseGot == nil || parseGot.Signature.identityKey() != "example/App" {
		parseT.Fatalf("expected app snapshot to be preserved, got %#v", parseGot)
	}
	if parseGot2 := parseRt.matchingHotReloadComponentSnapshot(parseReloadChanged); parseGot2 != nil {
		parseT.Fatalf("expected changed subtree to remount without snapshot, got %#v", parseGot2)
	}
	if parseGot3 := parseRt.matchingHotReloadComponentSnapshot(parseReloadStable); parseGot3 == nil || parseGot3.Signature.identityKey() != "example/Stable" {
		parseT.Fatalf("expected unchanged sibling snapshot to survive, got %#v", parseGot3)
	}
}

func TestSelectiveHotReloadRestoreFallsBackWhenPathsAreMissing(parseT *testing.T) {
	parseRt := &Runtime{}
	parseDecision := parseRt.RestoreHotReloadSnapshotWithPlan(HotReloadSnapshot{Components: []HotReloadComponentSnapshot{{
		Signature: ComponentSignature{Kind: "component", Name: "App", QualifiedName: "example/App"},
	}}}, HotReloadRestorePlan{Selective: true, ChangedIdentities: []string{"example/App"}})
	if parseDecision.Strategy != "legacy" {
		parseT.Fatalf("expected legacy fallback strategy, got %+v", parseDecision)
	}
	if parseDecision.UnsafeReason == "" {
		parseT.Fatalf("expected unsafe fallback reason, got %+v", parseDecision)
	}
}
