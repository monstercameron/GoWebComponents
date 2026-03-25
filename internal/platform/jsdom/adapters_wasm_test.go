//go:build js && wasm
// +build js,wasm

package jsdom

import "testing"

func TestWASMDOMAdapter_NestedBatchesKeepParentBoundaries(parseT *testing.T) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseOuterParent := parseAdapter.CreateElement("div").(*WASMDOMNode)
	parseOuterA := parseAdapter.CreateElement("h1")
	parseInnerParent := parseAdapter.CreateElement("section").(*WASMDOMNode)
	parseInnerA := parseAdapter.CreateElement("button")
	parseInnerB := parseAdapter.CreateElement("button")
	parseOuterB := parseAdapter.CreateElement("p")

	parseAdapter.BeginBatch(parseOuterParent)
	parseAdapter.AppendChild(parseOuterParent, parseOuterA)
	parseAdapter.AppendChild(parseOuterParent, parseInnerParent)
	parseAdapter.BeginBatch(parseInnerParent)
	parseAdapter.AppendChild(parseInnerParent, parseInnerA)
	parseAdapter.AppendChild(parseInnerParent, parseInnerB)
	parseAdapter.EndBatch()
	parseAdapter.AppendChild(parseOuterParent, parseOuterB)
	parseAdapter.EndBatch()

	if parseGot := parseOuterParent.value.Get("children").Get("length").Int(); parseGot != 3 {
		parseT.Fatalf("expected 3 outer children, got %d", parseGot)
	}
	if parseGot2 := parseInnerParent.value.Get("children").Get("length").Int(); parseGot2 != 2 {
		parseT.Fatalf("expected 2 inner children, got %d", parseGot2)
	}
	if parseFirst := parseOuterParent.value.Get("children").Index(0).Get("tagName").String(); parseFirst != "h1" {
		parseT.Fatalf("expected first outer child to be h1, got %q", parseFirst)
	}
	if parseSecond := parseOuterParent.value.Get("children").Index(1).Get("tagName").String(); parseSecond != "section" {
		parseT.Fatalf("expected second outer child to be section, got %q", parseSecond)
	}
	if parseThird := parseOuterParent.value.Get("children").Index(2).Get("tagName").String(); parseThird != "p" {
		parseT.Fatalf("expected third outer child to be p, got %q", parseThird)
	}
}

func TestWASMDOMAdapter_FragmentReuseDoesNotReplayChildren(parseT *testing.T) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseParent := parseAdapter.CreateElement("div").(*WASMDOMNode)
	parseFirst := parseAdapter.CreateElement("span")
	parseSecond := parseAdapter.CreateElement("span")

	parseAdapter.BeginBatch(parseParent)
	parseAdapter.AppendChild(parseParent, parseFirst)
	parseAdapter.EndBatch()

	parseAdapter.BeginBatch(parseParent)
	parseAdapter.AppendChild(parseParent, parseSecond)
	parseAdapter.EndBatch()

	parseChildren := parseParent.value.Get("children")
	if parseGot := parseChildren.Get("length").Int(); parseGot != 2 {
		parseT.Fatalf("expected 2 children after fragment reuse, got %d", parseGot)
	}
	if parseFirstTag := parseChildren.Index(0).Get("tagName").String(); parseFirstTag != "span" {
		parseT.Fatalf("expected first child tag span, got %q", parseFirstTag)
	}
	if parseSecondTag := parseChildren.Index(1).Get("tagName").String(); parseSecondTag != "span" {
		parseT.Fatalf("expected second child tag span, got %q", parseSecondTag)
	}
}
