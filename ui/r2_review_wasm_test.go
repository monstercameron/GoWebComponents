//go:build js && wasm

package ui

import (
	"context"
	"syscall/js"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// r2FakeElementNode is a DOMNode backed by a recording JS element, for hooks that bind to a DOMRef.
type r2FakeElementNode struct {
	value js.Value
}

// IsNull reports false: the fake element is always live.
func (parseN r2FakeElementNode) IsNull() bool { return false }

// Equals reports identity with another fake element.
func (parseN r2FakeElementNode) Equals(parseOther runtime.DOMNode) bool {
	parseCast, parseOk := parseOther.(r2FakeElementNode)
	return parseOk && parseCast.value.Equal(parseN.value)
}

// Value returns the recording JS element.
func (parseN r2FakeElementNode) Value() js.Value { return parseN.value }

// newR2RecordingElement builds a JS element whose addEventListener stores listeners by type.
func newR2RecordingElement(parseT *testing.T) (js.Value, map[string]js.Value) {
	parseT.Helper()
	parseListeners := map[string]js.Value{}
	parseElement := js.Global().Get("Object").New()
	parseAdd := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		parseListeners[parseArgs[0].String()] = parseArgs[1]
		return nil
	})
	parseRemove := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any { return nil })
	parseT.Cleanup(func() {
		parseAdd.Release()
		parseRemove.Release()
	})
	parseElement.Set("addEventListener", parseAdd)
	parseElement.Set("removeEventListener", parseRemove)
	return parseElement, parseListeners
}

// TestUsePointerEventsHandlerSeesLatestRender proves a pointer listener bound at mount dispatches to
// the closure from the most recent render, not the first one.
func TestUsePointerEventsHandlerSeesLatestRender(parseT *testing.T) {
	parseElement, parseListeners := newR2RecordingElement(parseT)
	var parseSetCount func(int)
	parseSeen := -1
	parseFlush := mountLatestClosureProbe(parseT, func() Node {
		parseCount := UseState(0)
		parseSetCount = parseCount.Set
		parseValue := parseCount.Get()
		parseRef := UseDOMRef()
		parseRef.box.node = r2FakeElementNode{value: parseElement}
		UsePointerEvents(parseRef, PointerHandlers{OnPointerDown: func(Event) { parseSeen = parseValue }})
		return runtime.CreateElement("span", map[string]any{})
	})
	parseSetCount(5)
	parseFlush()
	parseListener, parseOk := parseListeners["pointerdown"]
	if !parseOk {
		parseT.Fatal("pointerdown listener was not bound")
	}
	parseListener.Invoke(js.Global().Get("Object").New())
	if parseSeen != 5 {
		parseT.Fatalf("pointer handler saw count %d, want 5 (stale first-render closure)", parseSeen)
	}
}

// TestUseWheelHandlerSeesLatestRender is the same latest-closure proof for the wheel listener.
func TestUseWheelHandlerSeesLatestRender(parseT *testing.T) {
	parseElement, parseListeners := newR2RecordingElement(parseT)
	var parseSetCount func(int)
	parseSeen := -1
	parseFlush := mountLatestClosureProbe(parseT, func() Node {
		parseCount := UseState(0)
		parseSetCount = parseCount.Set
		parseValue := parseCount.Get()
		parseRef := UseDOMRef()
		parseRef.box.node = r2FakeElementNode{value: parseElement}
		UseWheel(parseRef, func(Event) { parseSeen = parseValue })
		return runtime.CreateElement("span", map[string]any{})
	})
	parseSetCount(5)
	parseFlush()
	parseListeners["wheel"].Invoke(js.Global().Get("Object").New())
	if parseSeen != 5 {
		parseT.Fatalf("wheel handler saw count %d, want 5 (stale first-render closure)", parseSeen)
	}
}

// TestUseTimerTriggerSurvivesRerenders proves the timer trigger measures from mount: re-renders
// before the delay elapses must not restart it.
func TestUseTimerTriggerSurvivesRerenders(parseT *testing.T) {
	var parseSetTick func(int)
	parseFired := false
	parseFlush := mountLatestClosureProbe(parseT, func() Node {
		parseTick := UseState(0)
		parseSetTick = parseTick.Set
		_ = parseTick.Get()
		parseFired = UseTimerTrigger(120 * time.Millisecond)
		return runtime.CreateElement("span", map[string]any{})
	})
	for parseIndex := 1; parseIndex <= 8; parseIndex++ {
		time.Sleep(25 * time.Millisecond)
		parseSetTick(parseIndex)
		parseFlush()
	}
	if !parseFired {
		parseT.Fatal("timer trigger never fired: each re-render restarted the delay")
	}
}

// TestUseTaskCtxParentCancelSettlesState proves cancelling the parent context leaves the task in a
// settled (not forever Running) state.
func TestUseTaskCtxParentCancelSettlesState(parseT *testing.T) {
	installUIHookContext(parseT)
	parseParent, parseCancelParent := context.WithCancel(context.Background())
	parseTask := UseTaskCtx(parseParent, func(parseCtx context.Context) (string, error) {
		<-parseCtx.Done()
		return "", parseCtx.Err()
	})
	parseTask.Start()
	parseCancelParent()
	waitAsyncCondition(parseT, "task to stop running after parent cancel", func() bool { return !parseTask.Get().Running })
	if !parseTask.Get().Cancelled {
		parseT.Fatalf("expected Cancelled after parent cancel, got %+v", parseTask.Get())
	}
}

// TestCompositeTypeaheadIgnoresModifierShortcuts proves Ctrl/Meta/Alt chords are left to the browser.
func TestCompositeTypeaheadIgnoresModifierShortcuts(parseT *testing.T) {
	installUIHookContext(parseT)
	parseNav := UseCompositeNavigation([]CompositeItem{{ID: "a", Text: "Alpha"}, {ID: "r", Text: "Reload"}})
	parsePrevented := 0
	parsePreventFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		parsePrevented++
		return nil
	})
	parseT.Cleanup(parsePreventFn.Release)
	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("key", "r")
	parseEvent.Set("ctrlKey", true)
	parseEvent.Set("preventDefault", parsePreventFn)
	parseNav.OnKeyDown(runtime.NewGoEvent(parseEvent))
	if parsePrevented != 0 {
		parseT.Fatalf("Ctrl+R was swallowed by typeahead (preventDefault called %d times)", parsePrevented)
	}
	if parseNav.ActiveIndex() != 0 {
		parseT.Fatalf("Ctrl+R moved the active item to %d", parseNav.ActiveIndex())
	}
}

// TestUseFocusTrapFocusesFirstFocusableNotContainer proves that with no initial/fallback selector
// the trap leaves focus on the first focusable child rather than stealing it back to the container.
func TestUseFocusTrapFocusesFirstFocusableNotContainer(parseT *testing.T) {
	parseFocusOrder := []string{}
	parseFuncs := []js.Func{}
	parseT.Cleanup(func() {
		for _, parseFn := range parseFuncs {
			parseFn.Release()
		}
	})
	parseNoop := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any { return nil })
	parseFuncs = append(parseFuncs, parseNoop)
	newFocusable := func(parseName string) js.Value {
		parseEl := js.Global().Get("Object").New()
		parseFocus := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
			parseFocusOrder = append(parseFocusOrder, parseName)
			return nil
		})
		parseFuncs = append(parseFuncs, parseFocus)
		parseEl.Set("focus", parseFocus)
		return parseEl
	}
	parseContainer := newFocusable("container")
	parseFirst := newFocusable("first")
	parseList := js.Global().Get("Array").New(parseFirst)
	parseQuery := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any { return parseList })
	parseFuncs = append(parseFuncs, parseQuery)
	parseContainer.Set("querySelectorAll", parseQuery)

	parseDocument := js.Global().Get("Object").New()
	parseGetByID := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any { return parseContainer })
	parseFuncs = append(parseFuncs, parseGetByID)
	parseDocument.Set("getElementById", parseGetByID)
	parseDocument.Set("addEventListener", parseNoop)
	parseDocument.Set("removeEventListener", parseNoop)
	parsePrevDocument := js.Global().Get("document")
	js.Global().Set("document", parseDocument)
	parseT.Cleanup(func() { js.Global().Set("document", parsePrevDocument) })

	mountLatestClosureProbe(parseT, func() Node {
		UseFocusTrap(FocusTrapOptions{ContainerSelector: "#dialog", Active: true})
		return runtime.CreateElement("span", map[string]any{})
	})
	if len(parseFocusOrder) == 0 || parseFocusOrder[len(parseFocusOrder)-1] != "first" {
		parseT.Fatalf("focus order %v: want the first focusable focused last", parseFocusOrder)
	}
}

// TestUseTaskRunnerPanicSettlesWithError proves a panicking task body is contained and surfaces as
// an Error instead of leaving the task Running forever.
func TestUseTaskRunnerPanicSettlesWithError(parseT *testing.T) {
	installUIHookContext(parseT)
	parseTask := UseTask(func(context.Context) (string, error) { panic("task boom") })
	parseTask.Start()
	waitAsyncCondition(parseT, "panicked task to stop running", func() bool { return !parseTask.Get().Running })
	if parseTask.Get().Error == nil || parseTask.Get().Ready {
		parseT.Fatalf("expected an Error state after a panicking task, got %+v", parseTask.Get())
	}
}
