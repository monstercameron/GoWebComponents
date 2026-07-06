//go:build js && wasm
// +build js,wasm

package jsdom

import (
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

func TestWASMDOMNodeNullHelpersTreatTypedNilAndNullWrapperAsEquivalent(parseT *testing.T) {
	var parseTypedNil runtime.DOMNode = (*WASMDOMNode)(nil)
	parseNullWrapper := runtime.DOMNode(&WASMDOMNode{value: js.Null()})
	parseConcrete := runtime.DOMNode(&WASMDOMNode{value: js.Global().Get("Object").New()})

	if !runtime.IsDOMNodeNull(parseTypedNil) {
		parseT.Fatal("expected typed nil wasm DOM node to report null")
	}
	if !runtime.IsDOMNodeNull(parseNullWrapper) {
		parseT.Fatal("expected null js.Value wrapper to report null")
	}
	if runtime.IsDOMNodeNull(parseConcrete) {
		parseT.Fatal("expected concrete wasm DOM node to report non-null")
	}
	if !runtime.IsSameDOMNode(parseTypedNil, parseNullWrapper) {
		parseT.Fatal("expected typed nil and null-wrapper wasm DOM nodes to compare equal")
	}
}

// TestSetTextContentToleratesNullAndTypedNilNodes pins that the text adapters do
// not panic on a typed-nil node or a null/undefined-valued node. A typed-nil
// (*WASMDOMNode) passes the type assertion but nil-derefs on .value, and .Set on a
// null/undefined js.Value panics with "not an object"; either would tear down the
// whole app from the render/commit path where detached nodes are common.
func TestSetTextContentToleratesNullAndTypedNilNodes(parseT *testing.T) {
	parseAdapter := &WASMDOMAdapter{}
	var parseTypedNil runtime.DOMNode = (*WASMDOMNode)(nil)
	parseNullNode := runtime.DOMNode(&WASMDOMNode{value: js.Null()})
	parseUndefNode := runtime.DOMNode(&WASMDOMNode{value: js.Undefined()})

	for _, parseNode := range []runtime.DOMNode{parseTypedNil, parseNullNode, parseUndefNode} {
		// Must be a no-op, not a panic.
		parseAdapter.SetTextContent(parseNode, "x")
		if parseGot := parseAdapter.GetTextContent(parseNode); parseGot != "" {
			parseT.Fatalf("expected empty text for absent node, got %q", parseGot)
		}
	}

	// A concrete object-backed node still round-trips (textContent is a plain
	// property on any JS object, so this needs no document shim).
	parseReal := runtime.DOMNode(&WASMDOMNode{value: js.Global().Get("Object").New()})
	parseAdapter.SetTextContent(parseReal, "hello")
	if parseGot := parseAdapter.GetTextContent(parseReal); parseGot != "hello" {
		parseT.Fatalf("expected round-tripped text 'hello', got %q", parseGot)
	}
}

func TestWASMDOMNodeEqualsUsesUnderlyingJSIdentity(parseT *testing.T) {
	parseValue := js.Global().Get("Object").New()
	parseSame := &WASMDOMNode{value: parseValue}
	parseAlsoSame := &WASMDOMNode{value: parseValue}
	parseDifferent := &WASMDOMNode{value: js.Global().Get("Object").New()}

	if !parseSame.Equals(parseAlsoSame) {
		parseT.Fatal("expected equal wrappers for the same underlying js value")
	}
	if parseSame.Equals(parseDifferent) {
		parseT.Fatal("expected different js values to compare unequal")
	}
}
