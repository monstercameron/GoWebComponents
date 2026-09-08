//go:build js && wasm
// +build js,wasm

package jsdom

import (
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
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

// TestSetTextContentPreservesSingleTextNodeIdentity verifies the hot update path
// mutates an existing Text child rather than replacing it through the parent's
// textContent property.
func TestSetTextContentPreservesSingleTextNodeIdentity(parseT *testing.T) {
	parseAdapter := &WASMDOMAdapter{}
	parseObject := js.Global().Get("Object")
	parseTextNode := parseObject.New()
	parseTextNode.Set("nodeType", 3)
	parseTextNode.Set("nodeValue", "before")
	parseTextNode.Set("nextSibling", js.Null())
	parseElement := parseObject.New()
	parseElement.Set("nodeType", 1)
	parseElement.Set("firstChild", parseTextNode)
	parseElement.Set("textContent", "before")

	parseAdapter.SetTextContent(&WASMDOMNode{value: parseElement}, "after")

	if parseGot := parseTextNode.Get("nodeValue").String(); parseGot != "after" {
		parseT.Fatalf("expected existing text child to update in place, got %q", parseGot)
	}
	if parseGot := parseElement.Get("textContent").String(); parseGot != "before" {
		parseT.Fatalf("single-text fast path replaced parent textContent, got %q", parseGot)
	}
}

// TestSetTextContentFallsBackForMixedChildren verifies the identity-preserving
// optimization never changes Element.textContent semantics for mixed content.
func TestSetTextContentFallsBackForMixedChildren(parseT *testing.T) {
	parseAdapter := &WASMDOMAdapter{}
	parseObject := js.Global().Get("Object")
	parseTextNode := parseObject.New()
	parseTextNode.Set("nodeType", 3)
	parseTextNode.Set("nodeValue", "before")
	parseTextNode.Set("nextSibling", parseObject.New())
	parseElement := parseObject.New()
	parseElement.Set("nodeType", 1)
	parseElement.Set("firstChild", parseTextNode)
	parseElement.Set("textContent", "before")

	parseAdapter.SetTextContent(&WASMDOMNode{value: parseElement}, "after")

	if parseGot := parseElement.Get("textContent").String(); parseGot != "after" {
		parseT.Fatalf("expected mixed-content fallback to set parent textContent, got %q", parseGot)
	}
	if parseGot := parseTextNode.Get("nodeValue").String(); parseGot != "before" {
		parseT.Fatalf("mixed-content fallback mutated the first child directly, got %q", parseGot)
	}
}

// TestSetTextContentBatchDefersAndPreservesIdentity verifies a commit batch
// turns many text writes into one flush without exposing partial DOM state or
// replacing an existing single Text child.
func TestSetTextContentBatchDefersAndPreservesIdentity(parseT *testing.T) {
	// The node-based wasm test runner has globalThis but no browser `window`;
	// expose the usual alias so the browser helper can install normally.
	parseWindow := js.Global().Get("window")
	if parseWindow.IsUndefined() {
		js.Global().Set("window", js.Global())
		defer js.Global().Delete("window")
	}
	parseAdapter := &WASMDOMAdapter{}
	parseObject := js.Global().Get("Object")
	parseTextNode := parseObject.New()
	parseTextNode.Set("nodeType", 3)
	parseTextNode.Set("nodeValue", "before")
	parseTextNode.Set("nextSibling", js.Null())
	parseElement := parseObject.New()
	parseElement.Set("nodeType", 1)
	parseElement.Set("firstChild", parseTextNode)
	parseElement.Set("textContent", "before")

	parseAdapter.BeginAttrUpdateBatch()
	parseAdapter.SetTextContent(&WASMDOMNode{value: parseElement}, "after")
	if parseGot := parseTextNode.Get("nodeValue").String(); parseGot != "before" {
		parseT.Fatalf("expected text update to remain buffered before flush, got %q", parseGot)
	}
	parseAdapter.EndAttrUpdateBatch()

	if parseGot := parseTextNode.Get("nodeValue").String(); parseGot != "after" {
		parseT.Fatalf("expected existing text child to update during flush, got %q", parseGot)
	}
	if parseGot := parseElement.Get("textContent").String(); parseGot != "before" {
		parseT.Fatalf("batched single-text update replaced parent textContent, got %q", parseGot)
	}
}

func TestRemoveChildBatchDefersAndChecksParent(parseT *testing.T) {
	parseWindow := js.Global().Get("window")
	if parseWindow.IsUndefined() {
		js.Global().Set("window", js.Global())
		defer js.Global().Delete("window")
	}
	parseAdapter := &WASMDOMAdapter{}
	parseObject := js.Global().Get("Object")
	parseParent := parseObject.New()
	parseChild := parseObject.New()
	parseChild.Set("parentNode", parseParent)
	wasRemoved := false
	parseRemove := js.FuncOf(func(js.Value, []js.Value) any {
		wasRemoved = true
		return nil
	})
	defer parseRemove.Release()
	parseChild.Set("remove", parseRemove)

	parseAdapter.BeginAttrUpdateBatch()
	parseAdapter.RemoveChild(&WASMDOMNode{value: parseParent}, &WASMDOMNode{value: parseChild})
	if wasRemoved {
		parseT.Fatal("expected removal to remain buffered before flush")
	}
	parseAdapter.EndAttrUpdateBatch()
	if !wasRemoved {
		parseT.Fatal("expected matching-parent child to be removed during flush")
	}

	wasRemoved = false
	parseAdapter.BeginAttrUpdateBatch()
	parseAdapter.RemoveChild(&WASMDOMNode{value: parseObject.New()}, &WASMDOMNode{value: parseChild})
	parseAdapter.EndAttrUpdateBatch()
	if wasRemoved {
		parseT.Fatal("expected stale-parent removal to remain a no-op")
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
