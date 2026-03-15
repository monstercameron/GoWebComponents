//go:build js && wasm
// +build js,wasm

package ui

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

type noOpScheduler struct{}

func (noOpScheduler) RequestIdleCallback(callback func(runtime.Deadline)) {}

func (noOpScheduler) SetTimeout(callback func(), delay int) {}

func installUIHookContext(t *testing.T) {
	t.Helper()
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	t.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})
}

func TestCreateElementReturnsExistingNode(t *testing.T) {
	existing := runtime.Div(map[string]interface{}{"id": "existing"})
	if got := CreateElement(existing); got != existing {
		t.Fatal("expected CreateElement to return existing node unchanged")
	}
}

func TestCreateElementAcceptsComponentFunctions(t *testing.T) {
	type props struct {
		Label string
	}

	withoutProps := func() Node {
		return Text("plain")
	}
	withProps := func(input props) Node {
		return Text(input.Label)
	}

	if node := CreateElement(withoutProps); node == nil {
		t.Fatal("expected zero-argument component to produce a node")
	}
	if node := CreateElement(withProps, props{Label: "hello"}); node == nil {
		t.Fatal("expected props component to produce a node")
	}
}

func TestFragmentAndTextHelpers(t *testing.T) {
	first := Text("first")
	second := Text("second")

	fragment := Fragment(first, second)
	if fragment == nil {
		t.Fatal("expected fragment")
	}
	if fragment.Type != "FRAGMENT" {
		t.Fatalf("expected fragment type, got %#v", fragment.Type)
	}
	if len(fragment.Children) != 2 {
		t.Fatalf("expected two fragment children, got %d", len(fragment.Children))
	}
	if first.TextContent != "first" || second.TextContent != "second" {
		t.Fatal("expected text helper to preserve text content")
	}
}

func TestPublicHooksWrappers(t *testing.T) {
	installUIHookContext(t)

	state := UseState(1)
	if state.Get() != 1 {
		t.Fatalf("expected initial state, got %d", state.Get())
	}
	state.Set(3)
	if state.Get() != 3 {
		t.Fatalf("expected updated state, got %d", state.Get())
	}
	state.Update(func(prev int) int { return prev + 4 })
	if state.Get() != 7 {
		t.Fatalf("expected updated state after updater, got %d", state.Get())
	}

	computed := UseMemo(func() int { return 9 }, "dep")
	if computed != 9 {
		t.Fatalf("expected memoized value 9, got %d", computed)
	}

	callback := UseCallback(func() int { return 11 }, "dep")
	if callback() != 11 {
		t.Fatalf("expected callback wrapper to preserve function value")
	}

	ref := UseRef("start")
	if ref.Get() != "start" {
		t.Fatalf("expected initial ref value, got %q", ref.Get())
	}
	ref.Set("done")
	if ref.Get() != "done" {
		t.Fatalf("expected updated ref value, got %q", ref.Get())
	}

	id := UseId()
	if id == "" {
		t.Fatal("expected non-empty id")
	}
}

func TestRefAndHandlerHelpers(t *testing.T) {
	var empty Ref[int]
	if empty.Get() != 0 {
		t.Fatalf("expected zero value from nil ref, got %d", empty.Get())
	}
	empty.Set(42)
	if empty.Get() != 0 {
		t.Fatal("expected nil ref Set to remain a no-op")
	}

	handler := RawHandler("wrapped")
	if handler.Value() != "wrapped" {
		t.Fatalf("expected raw handler value, got %#v", handler.Value())
	}
}
