//go:build js && wasm
// +build js,wasm

package jsdom

import "testing"

func TestWASMDOMAdapter_NestedBatchesKeepParentBoundaries(t *testing.T) {
	cleanup := installBenchmarkDOM()
	defer cleanup()

	adapter := NewWASMDOMAdapter()
	outerParent := adapter.CreateElement("div").(*WASMDOMNode)
	outerA := adapter.CreateElement("h1")
	innerParent := adapter.CreateElement("section").(*WASMDOMNode)
	innerA := adapter.CreateElement("button")
	innerB := adapter.CreateElement("button")
	outerB := adapter.CreateElement("p")

	adapter.BeginBatch(outerParent)
	adapter.AppendChild(outerParent, outerA)
	adapter.AppendChild(outerParent, innerParent)
	adapter.BeginBatch(innerParent)
	adapter.AppendChild(innerParent, innerA)
	adapter.AppendChild(innerParent, innerB)
	adapter.EndBatch()
	adapter.AppendChild(outerParent, outerB)
	adapter.EndBatch()

	if got := outerParent.value.Get("children").Get("length").Int(); got != 3 {
		t.Fatalf("expected 3 outer children, got %d", got)
	}
	if got := innerParent.value.Get("children").Get("length").Int(); got != 2 {
		t.Fatalf("expected 2 inner children, got %d", got)
	}
	if first := outerParent.value.Get("children").Index(0).Get("tagName").String(); first != "h1" {
		t.Fatalf("expected first outer child to be h1, got %q", first)
	}
	if second := outerParent.value.Get("children").Index(1).Get("tagName").String(); second != "section" {
		t.Fatalf("expected second outer child to be section, got %q", second)
	}
	if third := outerParent.value.Get("children").Index(2).Get("tagName").String(); third != "p" {
		t.Fatalf("expected third outer child to be p, got %q", third)
	}
}

func TestWASMDOMAdapter_FragmentReuseDoesNotReplayChildren(t *testing.T) {
	cleanup := installBenchmarkDOM()
	defer cleanup()

	adapter := NewWASMDOMAdapter()
	parent := adapter.CreateElement("div").(*WASMDOMNode)
	first := adapter.CreateElement("span")
	second := adapter.CreateElement("span")

	adapter.BeginBatch(parent)
	adapter.AppendChild(parent, first)
	adapter.EndBatch()

	adapter.BeginBatch(parent)
	adapter.AppendChild(parent, second)
	adapter.EndBatch()

	children := parent.value.Get("children")
	if got := children.Get("length").Int(); got != 2 {
		t.Fatalf("expected 2 children after fragment reuse, got %d", got)
	}
	if firstTag := children.Index(0).Get("tagName").String(); firstTag != "span" {
		t.Fatalf("expected first child tag span, got %q", firstTag)
	}
	if secondTag := children.Index(1).Get("tagName").String(); secondTag != "span" {
		t.Fatalf("expected second child tag span, got %q", secondTag)
	}
}
