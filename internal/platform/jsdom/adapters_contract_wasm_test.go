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

// TestClassAndStyleMutatorsTolerateNullAndTypedNilNodes pins #80: the class and
// style write paths — which run on every commit and routinely see detached or
// not-yet-created nodes — must no-op rather than panic on a typed-nil or
// null/undefined-valued node. Reaching classList/style through .value on such a
// node panics in syscall/js and tears down the whole app from the render path.
func TestClassAndStyleMutatorsTolerateNullAndTypedNilNodes(parseT *testing.T) {
	parseAdapter := &WASMDOMAdapter{}
	var parseTypedNil runtime.DOMNode = (*WASMDOMNode)(nil)
	parseNullNode := runtime.DOMNode(&WASMDOMNode{value: js.Null()})
	parseUndefNode := runtime.DOMNode(&WASMDOMNode{value: js.Undefined()})

	for _, parseNode := range []runtime.DOMNode{parseTypedNil, parseNullNode, parseUndefNode} {
		// Each must be a no-op, not a panic.
		parseAdapter.AddClass(parseNode, "a")
		parseAdapter.RemoveClass(parseNode, "a")
		parseAdapter.ToggleClass(parseNode, "a")
		parseAdapter.SetStyle(parseNode, "color", "red")
		parseAdapter.SetStyles(parseNode, map[string]string{"color": "red"})
	}
}

// TestAttributePropertyAndTreeReadsTolerateNullAndTypedNilNodes pins the rest of
// the #80 batch: the attribute/property/event-listener writers and the tree
// reads all dereference .value on the receiver, so a typed-nil or
// null/undefined-valued node must yield a no-op / zero value rather than a
// syscall/js panic that tears down the app from the render/commit path.
func TestAttributePropertyAndTreeReadsTolerateNullAndTypedNilNodes(parseT *testing.T) {
	parseAdapter := &WASMDOMAdapter{}
	var parseTypedNil runtime.DOMNode = (*WASMDOMNode)(nil)
	parseNullNode := runtime.DOMNode(&WASMDOMNode{value: js.Null()})
	parseUndefNode := runtime.DOMNode(&WASMDOMNode{value: js.Undefined()})
	parseHandler := js.FuncOf(func(js.Value, []js.Value) any { return nil })
	defer parseHandler.Release()

	for _, parseNode := range []runtime.DOMNode{parseTypedNil, parseNullNode, parseUndefNode} {
		// Writes: no-op, not a panic.
		parseAdapter.SetAttribute(parseNode, "id", "x")
		parseAdapter.RemoveAttribute(parseNode, "id")
		parseAdapter.SetProperty(parseNode, "value", "x")
		parseAdapter.AddPassiveEventListener(parseNode, "click", parseHandler)
		parseAdapter.RemovePassiveEventListener(parseNode, "click", parseHandler)

		// Reads: zero value, not a panic.
		if parseGot := parseAdapter.GetInnerHTML(parseNode); parseGot != "" {
			parseT.Fatalf("expected empty innerHTML for absent node, got %q", parseGot)
		}
		if parseGot := parseAdapter.GetChildren(parseNode); parseGot != nil {
			parseT.Fatalf("expected nil children for absent node, got %v", parseGot)
		}
		if parseGot := parseAdapter.GetParent(parseNode); parseGot != nil {
			parseT.Fatalf("expected nil parent for absent node, got %v", parseGot)
		}
		if parseGot := parseAdapter.GetFirstChild(parseNode); parseGot != nil {
			parseT.Fatalf("expected nil firstChild for absent node, got %v", parseGot)
		}
		if parseGot := parseAdapter.GetNextSibling(parseNode); parseGot != nil {
			parseT.Fatalf("expected nil nextSibling for absent node, got %v", parseGot)
		}
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
