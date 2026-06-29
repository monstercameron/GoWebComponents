//go:build js && wasm

package ui

import (
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/platform/jsdom"
)

// TestBindElementEventManagesListener proves bindElementEvent adds a listener,
// dispatches to the Go handler, and removes the listener on unbind.
func TestBindElementEventManagesListener(parseT *testing.T) {
	parseElement := js.Global().Get("Object").New()
	var parseAddedType, parseRemovedType string
	var parseStoredListener js.Value

	parseAdd := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		if len(parseArgs) >= 2 {
			parseAddedType = parseArgs[0].String()
			parseStoredListener = parseArgs[1]
		}
		return nil
	})
	defer parseAdd.Release()
	parseRemove := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		if len(parseArgs) >= 1 {
			parseRemovedType = parseArgs[0].String()
		}
		return nil
	})
	defer parseRemove.Release()
	parseElement.Set("addEventListener", parseAdd)
	parseElement.Set("removeEventListener", parseRemove)

	parseFired := 0
	parseUnbind := bindElementEvent(parseElement, "wheel", func(Event) { parseFired++ })
	if parseAddedType != "wheel" {
		parseT.Fatalf("expected addEventListener('wheel'), got %q", parseAddedType)
	}
	if parseStoredListener.Type() == js.TypeFunction {
		parseStoredListener.Invoke(js.Global().Get("Object").New())
	}
	if parseFired != 1 {
		parseT.Fatalf("expected handler to fire once, got %d", parseFired)
	}
	if parseUnbind == nil {
		parseT.Fatal("expected a non-nil unbind")
	}
	parseUnbind()
	if parseRemovedType != "wheel" {
		parseT.Fatalf("expected removeEventListener('wheel') on unbind, got %q", parseRemovedType)
	}
}

// TestBindElementEventNilHandlerIsNoBind proves a nil handler binds nothing.
func TestBindElementEventNilHandlerIsNoBind(parseT *testing.T) {
	parseElement := js.Global().Get("Object").New()
	if parseUnbind := bindElementEvent(parseElement, "wheel", nil); parseUnbind != nil {
		parseT.Fatal("nil handler must not bind (nil unbind expected)")
	}
}

// TestElementValueForRefBridgesNode proves a DOMRef pointing at a live node
// yields its js.Value, and an empty ref yields not-ok.
func TestElementValueForRefBridgesNode(parseT *testing.T) {
	parseDOMValue := js.Global().Get("Object").New()
	parseDOMValue.Set("marker", 42)
	parseRef := DOMRef{box: &domRefBox{node: jsdom.NewWASMDOMNode(parseDOMValue)}}

	parseValue, parseOk := elementValueForRef(parseRef)
	if !parseOk {
		parseT.Fatal("expected a usable js.Value for a live ref")
	}
	if parseValue.Get("marker").Int() != 42 {
		parseT.Fatal("bridged value is not the original node")
	}

	if _, parseEmptyOk := elementValueForRef(DOMRef{}); parseEmptyOk {
		parseT.Fatal("empty ref must not yield a value")
	}
}

// TestViewTransitionFallbackCallsApply proves that without startViewTransition
// (absent in the test host) the callback still runs synchronously.
func TestViewTransitionFallbackCallsApply(parseT *testing.T) {
	parseCalled := false
	ViewTransition(func() { parseCalled = true })
	if !parseCalled {
		parseT.Fatal("ViewTransition must fall back to calling apply directly")
	}
}

// TestDefaultReadOnlineStatusReturnsBool proves the wasm online read does not
// panic and returns a usable value in the test host.
func TestDefaultReadOnlineStatusReturnsBool(parseT *testing.T) {
	_ = defaultReadOnlineStatus()
}
