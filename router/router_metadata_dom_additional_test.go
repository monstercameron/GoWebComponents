//go:build js && wasm
// +build js,wasm

package router

import (
	"syscall/js"
	"testing"
)

// TestGetHeadElementWasmFallsBackToQuerySelector verifies metadata helpers can recover the head element through querySelector when document.head is unavailable.
func TestGetHeadElementWasmFallsBackToQuerySelector(parseT *testing.T) {
	parseDoc := js.Global().Get("Object").New()
	parseHead := js.Global().Get("Object").New()
	parseQuerySelector := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) != 1 || parseArgs[0].String() != "head" {
			parseT.Fatalf("expected head lookup, got %v", parseArgs)
		}
		return parseHead
	})
	defer parseQuerySelector.Release()
	parseDoc.Set("head", js.Undefined())
	parseDoc.Set("querySelector", parseQuerySelector)

	parseResolved := getHeadElement(parseDoc)
	if !parseResolved.Equal(parseHead) {
		parseT.Fatal("expected getHeadElement to fall back to querySelector(\"head\")")
	}
}

// TestRemoveElementWasmFallsBackToParentRemoveChild verifies metadata cleanup still works when the node does not expose remove().
func TestRemoveElementWasmFallsBackToParentRemoveChild(parseT *testing.T) {
	parseParent := js.Global().Get("Object").New()
	parseNode := js.Global().Get("Object").New()
	parseRemoved := js.Null()
	parseRemoveChild := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) != 1 {
			parseT.Fatalf("expected one child to remove, got %d args", len(parseArgs))
		}
		parseRemoved = parseArgs[0]
		return nil
	})
	defer parseRemoveChild.Release()
	parseParent.Set("removeChild", parseRemoveChild)
	parseNode.Set("remove", js.Undefined())
	parseNode.Set("parentNode", parseParent)

	removeElement(parseNode)
	if !parseRemoved.Equal(parseNode) {
		parseT.Fatal("expected removeElement to delegate to parentNode.removeChild when remove() is unavailable")
	}
}
