//go:build js && wasm
// +build js,wasm

package jsdom

import (
	"syscall/js"
	"testing"
)

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

func TestWASMDOMAdapter_AppendChildFallsBackToAppendChild(parseT *testing.T) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseArrayCtor := js.Global().Get("Array")
	parseObjectCtor := js.Global().Get("Object")
	parseParent := parseObjectCtor.New()
	parseChild := parseObjectCtor.New()
	parseChildren := parseArrayCtor.New()
	parseParent.Set("children", parseChildren)
	parseParent.Set("append", js.Undefined())

	parseAppendChild := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return nil
		}
		parseThis.Get("children").Call("push", parseArgs[0])
		parseArgs[0].Set("parentNode", parseThis)
		return parseArgs[0]
	})
	parseParent.Set("appendChild", parseAppendChild)
	parseT.Cleanup(parseAppendChild.Release)

	parseAdapter := NewWASMDOMAdapter()
	parseAdapter.AppendChild(&WASMDOMNode{value: parseParent}, &WASMDOMNode{value: parseChild})

	if parseGot := parseChildren.Get("length").Int(); parseGot != 1 {
		parseT.Fatalf("expected appendChild fallback to add one child, got %d", parseGot)
	}
	if !parseChild.Get("parentNode").Equal(parseParent) {
		parseT.Fatal("expected appendChild fallback to set parentNode")
	}
}

func TestWASMDOMAdapter_EndBatchFallsBackToAppendChild(parseT *testing.T) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseArrayCtor := js.Global().Get("Array")
	parseObjectCtor := js.Global().Get("Object")
	parseParent := parseObjectCtor.New()
	parseFirst := parseObjectCtor.New()
	parseSecond := parseObjectCtor.New()
	parseChildren := parseArrayCtor.New()
	parseParent.Set("children", parseChildren)
	parseParent.Set("append", js.Undefined())

	parseAppendChild := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return nil
		}
		parseThis.Get("children").Call("push", parseArgs[0])
		parseArgs[0].Set("parentNode", parseThis)
		return parseArgs[0]
	})
	parseParent.Set("appendChild", parseAppendChild)
	parseT.Cleanup(parseAppendChild.Release)

	parseAdapter := NewWASMDOMAdapter()
	parseParentNode := &WASMDOMNode{value: parseParent}
	parseAdapter.BeginBatch(parseParentNode)
	parseAdapter.AppendChild(parseParentNode, &WASMDOMNode{value: parseFirst})
	parseAdapter.AppendChild(parseParentNode, &WASMDOMNode{value: parseSecond})
	parseAdapter.EndBatch()

	if parseGot := parseChildren.Get("length").Int(); parseGot != 2 {
		parseT.Fatalf("expected batched appendChild fallback to add two children, got %d", parseGot)
	}
	if !parseFirst.Get("parentNode").Equal(parseParent) || !parseSecond.Get("parentNode").Equal(parseParent) {
		parseT.Fatal("expected batched appendChild fallback to set parentNode on each child")
	}
}
