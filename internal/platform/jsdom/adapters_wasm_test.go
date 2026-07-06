//go:build js && wasm
// +build js,wasm

package jsdom

import (
	"strings"
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

func TestWASMDOMAdapter_LazilyBindsDocumentMethods(parseT *testing.T) {
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parsePrevDoc := parseGlobal.Get("document")

	parseCreateElementCalls := 0
	parseCreateTextNodeCalls := 0
	parseQuerySelectorCalls := 0
	parseQuerySelectorAllCalls := 0
	parseGetElementByIDCalls := 0
	parseGetElementsByClassNameCalls := 0
	parseGetElementsByTagNameCalls := 0

	parseNewNode := func(parseTag string) js.Value {
		parseNode := parseObjectCtor.New()
		parseNode.Set("tagName", parseTag)
		parseNode.Set("attributes", parseObjectCtor.New())
		parseNode.Set("parentNode", js.Null())
		return parseNode
	}
	parseNewNodeList := func(parseTag string) js.Value {
		parseList := parseGlobal.Get("Array").New()
		parseList.Call("push", parseNewNode(parseTag))
		return parseList
	}
	parseSetAttribute := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseThis.Get("attributes").Set(parseArgs[0].String(), parseArgs[1].String())
		return nil
	})
	parseCreateElement := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseCreateElementCalls++
		parseTag := ""
		if len(parseArgs) > 0 {
			parseTag = parseArgs[0].String()
		}
		parseNode := parseNewNode(parseTag)
		parseNode.Set("setAttribute", parseSetAttribute)
		if parseTag == "template" {
			parseContent := parseObjectCtor.New()
			parseChild := parseObjectCtor.New()
			parseChild.Set("tagName", "prepared")
			parseContent.Set("firstChild", parseChild)
			parseNode.Set("content", parseContent)
		}
		return parseNode
	})
	parseCreateTextNode := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseCreateTextNodeCalls++
		parseNode := parseObjectCtor.New()
		if len(parseArgs) > 0 {
			parseNode.Set("textContent", parseArgs[0].String())
		}
		return parseNode
	})
	parseQuerySelector := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseQuerySelectorCalls++
		return parseNewNode("query")
	})
	parseQuerySelectorAll := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseQuerySelectorAllCalls++
		return parseNewNodeList("query-all")
	})
	parseGetElementByID := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseGetElementByIDCalls++
		return parseNewNode("by-id")
	})
	parseGetElementsByClassName := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseGetElementsByClassNameCalls++
		return parseNewNodeList("by-class")
	})
	parseGetElementsByTagName := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseGetElementsByTagNameCalls++
		return parseNewNodeList("by-tag")
	})
	parseT.Cleanup(func() {
		parseGlobal.Set("document", parsePrevDoc)
		parseSetAttribute.Release()
		parseCreateElement.Release()
		parseCreateTextNode.Release()
		parseQuerySelector.Release()
		parseQuerySelectorAll.Release()
		parseGetElementByID.Release()
		parseGetElementsByClassName.Release()
		parseGetElementsByTagName.Release()
	})

	parseDoc := parseObjectCtor.New()
	parseDoc.Set("createElement", parseCreateElement)
	parseDoc.Set("createTextNode", parseCreateTextNode)
	parseDoc.Set("querySelector", parseQuerySelector)
	parseDoc.Set("querySelectorAll", parseQuerySelectorAll)
	parseDoc.Set("getElementById", parseGetElementByID)
	parseDoc.Set("getElementsByClassName", parseGetElementsByClassName)
	parseDoc.Set("getElementsByTagName", parseGetElementsByTagName)
	parseGlobal.Set("document", parseDoc)

	parseAdapter := NewWASMDOMAdapter()
	if parseAdapter.isDocumentBound ||
		parseAdapter.isCreateElementBound ||
		parseAdapter.isCreateTextNodeBound ||
		parseAdapter.isQuerySelectorBound ||
		parseAdapter.isQuerySelectorAllBound ||
		parseAdapter.isGetElementByIDBound ||
		parseAdapter.isGetByClassNameBound ||
		parseAdapter.isGetByTagNameBound {
		parseT.Fatal("expected constructor to leave document methods unbound")
	}
	if parseCreateElementCalls != 0 ||
		parseCreateTextNodeCalls != 0 ||
		parseQuerySelectorCalls != 0 ||
		parseQuerySelectorAllCalls != 0 ||
		parseGetElementByIDCalls != 0 ||
		parseGetElementsByClassNameCalls != 0 ||
		parseGetElementsByTagNameCalls != 0 {
		parseT.Fatalf(
			"constructor called DOM methods: create=%d text=%d query=%d queryAll=%d id=%d class=%d tag=%d",
			parseCreateElementCalls,
			parseCreateTextNodeCalls,
			parseQuerySelectorCalls,
			parseQuerySelectorAllCalls,
			parseGetElementByIDCalls,
			parseGetElementsByClassNameCalls,
			parseGetElementsByTagNameCalls,
		)
	}

	parseText := parseAdapter.CreateTextNode("hello").(*WASMDOMNode)
	if parseText.value.Get("textContent").String() != "hello" {
		parseT.Fatalf("expected text node content to be hello, got %q", parseText.value.Get("textContent").String())
	}
	if parseCreateTextNodeCalls != 1 || parseCreateElementCalls != 0 {
		parseT.Fatalf("CreateTextNode should bind only text creation, got create=%d text=%d", parseCreateElementCalls, parseCreateTextNodeCalls)
	}
	if !parseAdapter.isDocumentBound || !parseAdapter.isCreateTextNodeBound || parseAdapter.isCreateElementBound {
		parseT.Fatal("expected text creation to bind document and createTextNode only")
	}

	parseElement := parseAdapter.CreateElement("div").(*WASMDOMNode)
	if parseElement.value.Get("tagName").String() != "div" {
		parseT.Fatalf("expected created element tag div, got %q", parseElement.value.Get("tagName").String())
	}
	if parseCreateElementCalls != 1 {
		parseT.Fatalf("expected one createElement invocation, got %d", parseCreateElementCalls)
	}
	if !parseAdapter.isCreateElementBound {
		parseT.Fatal("expected CreateElement to bind createElement")
	}
	parseBoundCreateElement := parseAdapter.createElement
	parseAdapter.CreateElement("section")
	if !parseAdapter.createElement.Equal(parseBoundCreateElement) {
		parseT.Fatal("expected repeated CreateElement to reuse the bound createElement method")
	}

	parsePrepared := parseAdapter.CreatePreparedElement("span", []runtime.HostAttr{{Name: "data-id", Value: "42"}}, "ready")
	if parsePrepared == nil || parsePrepared.IsNull() {
		parseT.Fatal("expected prepared element from lazy template path")
	}
	if parseCreateElementCalls != 3 {
		parseT.Fatalf("expected template to be created lazily on prepared path, got %d createElement calls", parseCreateElementCalls)
	}
	if !parseAdapter.isStoreTemplateBound {
		parseT.Fatal("expected prepared element path to bind the template lazily")
	}

	parseAdapter.QuerySelector("#app")
	if parseQuerySelectorCalls != 1 || !parseAdapter.isQuerySelectorBound {
		parseT.Fatalf("expected querySelector to bind on first query, calls=%d bound=%v", parseQuerySelectorCalls, parseAdapter.isQuerySelectorBound)
	}
	parseBoundQuerySelector := parseAdapter.querySelector
	parseAdapter.QuerySelector("#app")
	if !parseAdapter.querySelector.Equal(parseBoundQuerySelector) {
		parseT.Fatal("expected repeated QuerySelector to reuse the bound querySelector method")
	}
	if parseQuerySelectorCalls != 2 {
		parseT.Fatalf("expected repeated QuerySelector to call querySelector twice, got %d", parseQuerySelectorCalls)
	}

	if parseNodes := parseAdapter.QuerySelectorAll(".item"); len(parseNodes) != 1 || !parseAdapter.isQuerySelectorAllBound || parseQuerySelectorAllCalls != 1 {
		parseT.Fatalf("expected querySelectorAll to bind on first use, nodes=%d bound=%v calls=%d", len(parseNodes), parseAdapter.isQuerySelectorAllBound, parseQuerySelectorAllCalls)
	}
	if parseNode := parseAdapter.GetElementById("app"); parseNode == nil || parseNode.IsNull() || !parseAdapter.isGetElementByIDBound || parseGetElementByIDCalls != 1 {
		parseT.Fatalf("expected getElementById to bind on first use, node=%v bound=%v calls=%d", parseNode, parseAdapter.isGetElementByIDBound, parseGetElementByIDCalls)
	}
	if parseNodes := parseAdapter.GetElementsByClassName("item"); len(parseNodes) != 1 || !parseAdapter.isGetByClassNameBound || parseGetElementsByClassNameCalls != 1 {
		parseT.Fatalf("expected getElementsByClassName to bind on first use, nodes=%d bound=%v calls=%d", len(parseNodes), parseAdapter.isGetByClassNameBound, parseGetElementsByClassNameCalls)
	}
	if parseNodes := parseAdapter.GetElementsByTagName("li"); len(parseNodes) != 1 || !parseAdapter.isGetByTagNameBound || parseGetElementsByTagNameCalls != 1 {
		parseT.Fatalf("expected getElementsByTagName to bind on first use, nodes=%d bound=%v calls=%d", len(parseNodes), parseAdapter.isGetByTagNameBound, parseGetElementsByTagNameCalls)
	}
}

func TestWASMEventAdapter_LazilyBindsElementPrototype(parseT *testing.T) {
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parsePrevElement := parseGlobal.Get("Element")

	parseAddCalls := 0
	parseRemoveCalls := 0
	parseAddEventListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseAddCalls++
		parseThis.Set("lastEventType", parseArgs[0].String())
		return nil
	})
	parseRemoveEventListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseRemoveCalls++
		parseThis.Set("removedEventType", parseArgs[0].String())
		return nil
	})
	parseElementCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return parseObjectCtor.New()
	})
	parseT.Cleanup(func() {
		parseGlobal.Set("Element", parsePrevElement)
		parseAddEventListener.Release()
		parseRemoveEventListener.Release()
		parseElementCtor.Release()
	})

	parseProto := parseObjectCtor.New()
	parseProto.Set("addEventListener", parseAddEventListener)
	parseProto.Set("removeEventListener", parseRemoveEventListener)
	parseElementCtor.Set("prototype", parseProto)
	parseGlobal.Set("Element", parseElementCtor)

	parseAdapter := NewWASMEventAdapter()
	if parseAdapter.isMethodsBound {
		parseT.Fatal("expected constructor to leave event methods unbound")
	}
	if parseAddCalls != 0 || parseRemoveCalls != 0 {
		parseT.Fatalf("constructor called event methods: add=%d remove=%d", parseAddCalls, parseRemoveCalls)
	}

	parseNode := &WASMDOMNode{value: parseObjectCtor.New()}
	parseHandler := parseAdapter.CreateEventHandler(func(runtime.Event) {})
	defer parseAdapter.ReleaseEventHandler(parseHandler)

	parseAdapter.AddEventListener(parseNode, "click", parseHandler)
	if !parseAdapter.isMethodsBound {
		parseT.Fatal("expected AddEventListener to bind event methods")
	}
	if parseAddCalls != 1 {
		parseT.Fatalf("expected one addEventListener call, got %d", parseAddCalls)
	}
	if parseNode.value.Get("lastEventType").String() != "click" {
		parseT.Fatalf("expected click listener to be attached, got %q", parseNode.value.Get("lastEventType").String())
	}

	parseAdapter.RemoveEventListener(parseNode, "click", parseHandler)
	if parseRemoveCalls != 1 {
		parseT.Fatalf("expected one removeEventListener call, got %d", parseRemoveCalls)
	}
	if parseNode.value.Get("removedEventType").String() != "click" {
		parseT.Fatalf("expected click listener to be removed, got %q", parseNode.value.Get("removedEventType").String())
	}
}

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

func TestBuildHostElementHTMLSanitizesURLAttributes(parseT *testing.T) {
	getHTML, parseOK := buildHostElementHTML("a", []runtime.HostAttr{
		{Name: "href", Value: "javascript:alert(1)"},
		{Name: "class", Value: "link"},
	}, "click")
	if !parseOK {
		parseT.Fatalf("expected template fast path to accept a safe tag/attr set")
	}
	if strings.Contains(getHTML, "javascript:") {
		parseT.Fatalf("template fast path must block javascript: URLs, got %q", getHTML)
	}
	if !strings.Contains(getHTML, `class="link"`) || !strings.Contains(getHTML, ">click</a>") {
		parseT.Fatalf("unexpected serialized element: %q", getHTML)
	}
}
