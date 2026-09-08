//go:build js && wasm
// +build js,wasm

package jsdom

import (
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

func installBenchmarkDOM() func() {
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parseArrayCtor := parseGlobal.Get("Array")
	parseReflectObj := parseGlobal.Get("Reflect")

	parsePrevDoc := parseGlobal.Get("document")
	parsePrevElement := parseGlobal.Get("Element")

	parseMakeNode := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseTag := ""
		if len(parseArgs) > 0 {
			parseTag = parseArgs[0].String()
		}
		parseNode := parseObjectCtor.New()
		parseNode.Set("tagName", parseTag)
		parseNode.Set("attributes", parseObjectCtor.New())
		parseNode.Set("children", parseArrayCtor.New())
		parseNode.Set("parentNode", js.Null())
		return parseNode
	})

	parseAppendChild := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseChild := parseArgs2[0]
		if parseChild.Get("isFragment").Truthy() {
			parseFragmentChildren := parseChild.Get("children")
			parseChildren := parseThis2.Get("children")
			parseLength := parseFragmentChildren.Get("length").Int()
			for parseI := 0; parseI < parseLength; parseI++ {
				parseFragmentChild := parseFragmentChildren.Index(parseI)
				parseChildren.Call("push", parseFragmentChild)
				parseFragmentChild.Set("parentNode", parseThis2)
			}
			parseChild.Set("children", parseArrayCtor.New())
			return parseChild
		}
		parseThis2.Get("children").Call("push", parseChild)
		parseChild.Set("parentNode", parseThis2)
		return parseArgs2[0]
	})
	parseAppendNodes := js.FuncOf(func(parseThisAppend js.Value, parseArgsAppend []js.Value) interface{} {
		for _, parseChild := range parseArgsAppend {
			if parseChild.Get("isFragment").Truthy() {
				parseFragmentChildren := parseChild.Get("children")
				parseLength := parseFragmentChildren.Get("length").Int()
				for parseI := 0; parseI < parseLength; parseI++ {
					parseFragmentChild := parseFragmentChildren.Index(parseI)
					parseThisAppend.Get("children").Call("push", parseFragmentChild)
					parseFragmentChild.Set("parentNode", parseThisAppend)
				}
				parseChild.Set("children", parseArrayCtor.New())
				continue
			}
			parseThisAppend.Get("children").Call("push", parseChild)
			parseChild.Set("parentNode", parseThisAppend)
		}
		return nil
	})
	parseRemoveChild := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		parseChildren2 := parseThis3.Get("children")
		parseLength2 := parseChildren2.Get("length").Int()
		for parseI2 := 0; parseI2 < parseLength2; parseI2++ {
			if parseChildren2.Index(parseI2).Equal(parseArgs3[0]) {
				parseChildren2.Call("splice", parseI2, 1)
				parseArgs3[0].Set("parentNode", js.Null())
				break
			}
		}
		return parseArgs3[0]
	})
	parseRemoveNode := js.FuncOf(func(parseThisRemove js.Value, parseArgsRemove []js.Value) interface{} {
		parseParent := parseThisRemove.Get("parentNode")
		if parseParent.IsNull() || parseParent.IsUndefined() {
			return nil
		}
		parseChildren := parseParent.Get("children")
		parseLength := parseChildren.Get("length").Int()
		for parseI := 0; parseI < parseLength; parseI++ {
			if parseChildren.Index(parseI).Equal(parseThisRemove) {
				parseChildren.Call("splice", parseI, 1)
				parseThisRemove.Set("parentNode", js.Null())
				break
			}
		}
		return nil
	})
	setAttribute := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		parseThis4.Get("attributes").Set(parseArgs4[0].String(), parseArgs4[1].String())
		return nil
	})
	parseRemoveAttribute := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		parseReflectObj.Call("deleteProperty", parseThis5.Get("attributes"), parseArgs5[0].String())
		return nil
	})
	parseInsertBefore := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
		parseThis6.Get("children").Call("push", parseArgs6[0])
		parseArgs6[0].Set("parentNode", parseThis6)
		return parseArgs6[0]
	})
	parseBeforeNode := js.FuncOf(func(parseThisBefore js.Value, parseArgsBefore []js.Value) interface{} {
		parseParent := parseThisBefore.Get("parentNode")
		if parseParent.IsNull() || parseParent.IsUndefined() {
			return nil
		}
		parseChildren := parseParent.Get("children")
		parseLength := parseChildren.Get("length").Int()
		parseRefIndex := parseLength
		for parseI := 0; parseI < parseLength; parseI++ {
			if parseChildren.Index(parseI).Equal(parseThisBefore) {
				parseRefIndex = parseI
				break
			}
		}
		for _, parseChild := range parseArgsBefore {
			parseChildren.Call("splice", parseRefIndex, 0, parseChild)
			parseChild.Set("parentNode", parseParent)
			parseRefIndex++
		}
		return nil
	})
	parseReplaceChild := js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} {
		parseChildren3 := parseThis7.Get("children")
		parseLength3 := parseChildren3.Get("length").Int()
		for parseI3 := 0; parseI3 < parseLength3; parseI3++ {
			if parseChildren3.Index(parseI3).Equal(parseArgs7[1]) {
				parseChildren3.SetIndex(parseI3, parseArgs7[0])
				parseArgs7[0].Set("parentNode", parseThis7)
				parseArgs7[1].Set("parentNode", js.Null())
				return parseArgs7[1]
			}
		}
		return parseArgs7[1]
	})
	parseReplaceWith := js.FuncOf(func(parseThisReplace js.Value, parseArgsReplace []js.Value) interface{} {
		parseParent := parseThisReplace.Get("parentNode")
		if parseParent.IsNull() || parseParent.IsUndefined() {
			return nil
		}
		parseChildren := parseParent.Get("children")
		parseLength := parseChildren.Get("length").Int()
		for parseI := 0; parseI < parseLength; parseI++ {
			if parseChildren.Index(parseI).Equal(parseThisReplace) {
				parseChildren.SetIndex(parseI, parseArgsReplace[0])
				parseArgsReplace[0].Set("parentNode", parseParent)
				parseThisReplace.Set("parentNode", js.Null())
				break
			}
		}
		return nil
	})
	parseAddEventListener := js.FuncOf(func(parseThis8 js.Value, parseArgs8 []js.Value) interface{} {
		return nil
	})
	parseRemoveEventListener := js.FuncOf(func(parseThis9 js.Value, parseArgs9 []js.Value) interface{} {
		return nil
	})
	parseStoreNodeMethods := func(parseNode js.Value) {
		parseNode.Set("appendChild", parseAppendChild)
		parseNode.Set("append", parseAppendNodes)
		parseNode.Set("removeChild", parseRemoveChild)
		parseNode.Set("remove", parseRemoveNode)
		parseNode.Set("insertBefore", parseInsertBefore)
		parseNode.Set("before", parseBeforeNode)
		parseNode.Set("replaceChild", parseReplaceChild)
		parseNode.Set("replaceWith", parseReplaceWith)
	}

	parseElementCtor := js.FuncOf(func(parseThis10 js.Value, parseArgs10 []js.Value) interface{} {
		return parseObjectCtor.New()
	})
	parseProto := parseObjectCtor.New()
	parseProto.Set("appendChild", parseAppendChild)
	parseProto.Set("append", parseAppendNodes)
	parseProto.Set("removeChild", parseRemoveChild)
	parseProto.Set("remove", parseRemoveNode)
	parseProto.Set("setAttribute", setAttribute)
	parseProto.Set("removeAttribute", parseRemoveAttribute)
	parseProto.Set("insertBefore", parseInsertBefore)
	parseProto.Set("before", parseBeforeNode)
	parseProto.Set("replaceChild", parseReplaceChild)
	parseProto.Set("replaceWith", parseReplaceWith)
	parseProto.Set("addEventListener", parseAddEventListener)
	parseProto.Set("removeEventListener", parseRemoveEventListener)
	parseElementCtor.Set("prototype", parseProto)

	parseDocCreateElement := js.FuncOf(func(parseThis11 js.Value, parseArgs11 []js.Value) interface{} {
		parseNode := parseMakeNode.Invoke(parseArgs11[0].String())
		parseStoreNodeMethods(parseNode)
		return parseNode
	})
	parseDocCreateTextNode := js.FuncOf(func(parseThis12 js.Value, parseArgs12 []js.Value) interface{} {
		parseNode2 := parseObjectCtor.New()
		parseNode2.Set("textContent", parseArgs12[0].String())
		parseNode2.Set("parentNode", js.Null())
		parseStoreNodeMethods(parseNode2)
		return parseNode2
	})
	parseDocCreateFragment := js.FuncOf(func(parseThis13 js.Value, parseArgs13 []js.Value) interface{} {
		parseFrag := parseObjectCtor.New()
		parseFrag.Set("isFragment", true)
		parseFrag.Set("children", parseArrayCtor.New())
		parseFrag.Set("appendChild", parseAppendChild)
		parseFrag.Set("append", parseAppendNodes)
		parseFrag.Set("parentNode", js.Null())
		return parseFrag
	})
	parseDocQuerySelector := js.FuncOf(func(parseThis14 js.Value, parseArgs14 []js.Value) interface{} {
		parseNode := parseMakeNode.Invoke("div")
		parseStoreNodeMethods(parseNode)
		return parseNode
	})
	parseDocQuerySelectorAll := js.FuncOf(func(parseThis15 js.Value, parseArgs15 []js.Value) interface{} {
		parseList := parseArrayCtor.New()
		parseNode := parseMakeNode.Invoke("div")
		parseStoreNodeMethods(parseNode)
		parseList.Call("push", parseNode)
		parseList.Set("item", js.FuncOf(func(parseThis16 js.Value, parseArgs16 []js.Value) interface{} {
			return parseThis16.Index(parseArgs16[0].Int())
		}))
		return parseList
	})
	parseDocGetElementById := js.FuncOf(func(parseThis17 js.Value, parseArgs17 []js.Value) interface{} {
		parseNode := parseMakeNode.Invoke("div")
		parseStoreNodeMethods(parseNode)
		return parseNode
	})
	parseDocGetElementsByClassName := js.FuncOf(func(parseThis18 js.Value, parseArgs18 []js.Value) interface{} {
		parseList2 := parseArrayCtor.New()
		parseNode := parseMakeNode.Invoke("div")
		parseStoreNodeMethods(parseNode)
		parseList2.Call("push", parseNode)
		parseList2.Set("item", js.FuncOf(func(parseThis19 js.Value, parseArgs19 []js.Value) interface{} {
			return parseThis19.Index(parseArgs19[0].Int())
		}))
		return parseList2
	})
	parseDocGetElementsByTagName := js.FuncOf(func(parseThis20 js.Value, parseArgs20 []js.Value) interface{} {
		parseList3 := parseArrayCtor.New()
		parseNode := parseMakeNode.Invoke("div")
		parseStoreNodeMethods(parseNode)
		parseList3.Call("push", parseNode)
		parseList3.Set("item", js.FuncOf(func(parseThis21 js.Value, parseArgs21 []js.Value) interface{} {
			return parseThis21.Index(parseArgs21[0].Int())
		}))
		return parseList3
	})
	parseDoc := parseObjectCtor.New()
	parseDoc.Set("createElement", parseDocCreateElement)
	parseDoc.Set("createTextNode", parseDocCreateTextNode)
	parseDoc.Set("createDocumentFragment", parseDocCreateFragment)
	parseDoc.Set("querySelector", parseDocQuerySelector)
	parseDoc.Set("querySelectorAll", parseDocQuerySelectorAll)
	parseDoc.Set("getElementById", parseDocGetElementById)
	parseDoc.Set("getElementsByClassName", parseDocGetElementsByClassName)
	parseDoc.Set("getElementsByTagName", parseDocGetElementsByTagName)

	parseGlobal.Set("document", parseDoc)
	parseGlobal.Set("Element", parseElementCtor)

	return func() {
		parseGlobal.Set("document", parsePrevDoc)
		parseGlobal.Set("Element", parsePrevElement)
		parseMakeNode.Release()
		parseAppendChild.Release()
		parseAppendNodes.Release()
		parseRemoveChild.Release()
		parseRemoveNode.Release()
		setAttribute.Release()
		parseRemoveAttribute.Release()
		parseInsertBefore.Release()
		parseBeforeNode.Release()
		parseReplaceChild.Release()
		parseReplaceWith.Release()
		parseAddEventListener.Release()
		parseRemoveEventListener.Release()
		parseDocCreateElement.Release()
		parseDocCreateTextNode.Release()
		parseDocCreateFragment.Release()
		parseDocQuerySelector.Release()
		parseDocQuerySelectorAll.Release()
		parseDocGetElementById.Release()
		parseDocGetElementsByClassName.Release()
		parseDocGetElementsByTagName.Release()
		parseElementCtor.Release()
	}
}

func BenchmarkWASMDOMAdapterCreateElement(parseB *testing.B) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseNode := parseAdapter.CreateElement("div")
		if parseNode == nil || parseNode.IsNull() {
			parseB.Fatal("expected DOM node")
		}
	}
}

func BenchmarkWASMDOMAdapterSetAttribute(parseB *testing.B) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseNode := parseAdapter.CreateElement("div")
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseAdapter.SetAttribute(parseNode, "data-id", "42")
	}
}

func BenchmarkWASMDOMAdapterAppendChild(parseB *testing.B) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseParent := parseAdapter.CreateElement("div")
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseChild := parseAdapter.CreateElement("span")
		parseAdapter.AppendChild(parseParent, parseChild)
	}
}

func BenchmarkWASMDOMAdapterQuerySelector(parseB *testing.B) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseNode := parseAdapter.QuerySelector("#app")
		if parseNode == nil {
			parseB.Fatal("expected DOM node")
		}
	}
}

func BenchmarkWASMDOMAdapterGetElementById(parseB *testing.B) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseNode := parseAdapter.GetElementById("app")
		if parseNode == nil || parseNode.IsNull() {
			parseB.Fatal("expected DOM node")
		}
	}
}

func BenchmarkWASMDOMAdapterQuerySelectorAll(parseB *testing.B) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseNodes := parseAdapter.QuerySelectorAll(".item")
		if len(parseNodes) == 0 {
			parseB.Fatal("expected DOM nodes")
		}
	}
}

func BenchmarkWASMDOMAdapterSetPropertyString(parseB *testing.B) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseNode := parseAdapter.CreateElement("input")
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseAdapter.SetProperty(parseNode, "value", "payload")
	}
}

func BenchmarkWASMDOMAdapterBatchAppend16(parseB *testing.B) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseParent := parseAdapter.CreateElement("div")
	parseChildren := make([]runtime.DOMNode, 16)
	for parseI := range parseChildren {
		parseChildren[parseI] = parseAdapter.CreateElement("span")
	}

	parseB.ReportAllocs()
	for parseI2 := 0; parseI2 < parseB.N; parseI2++ {
		parseAdapter.BeginBatch(parseParent)
		for _, parseChild := range parseChildren {
			parseAdapter.AppendChild(parseParent, parseChild)
		}
		parseAdapter.EndBatch()
	}
}

// BenchmarkWASMDOMAdapterAppendChildCurrentVsLegacy compares the current append-based mutator path against the previous appendChild-returning bridge call.
func BenchmarkWASMDOMAdapterAppendChildCurrentVsLegacy(parseB *testing.B) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseB.Run("current_append", func(parseB *testing.B) {
		parseB.ReportAllocs()
		for parseI := 0; parseI < parseB.N; parseI++ {
			parseParent := parseAdapter.CreateElement("div")
			parseChild := parseAdapter.CreateElement("span")
			parseAdapter.AppendChild(parseParent, parseChild)
		}
	})
	parseB.Run("legacy_appendChild", func(parseB *testing.B) {
		parseB.ReportAllocs()
		for parseI := 0; parseI < parseB.N; parseI++ {
			parseParent := parseAdapter.CreateElement("div").(*WASMDOMNode)
			parseChild := parseAdapter.CreateElement("span").(*WASMDOMNode)
			parseParent.value.Call("appendChild", parseChild.value)
		}
	})
}

// BenchmarkWASMDOMAdapterBatchAppend16CurrentVsLegacy compares the current single-call append batch flush against the previous fragment-plus-appendChild loop.
func BenchmarkWASMDOMAdapterBatchAppend16CurrentVsLegacy(parseB *testing.B) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseDocument := js.Global().Get("document")
	parseB.Run("current_append", func(parseB *testing.B) {
		parseB.ReportAllocs()
		for parseI := 0; parseI < parseB.N; parseI++ {
			parseParent := parseAdapter.CreateElement("div")
			parseChildren := make([]runtime.DOMNode, 16)
			for parseChildIndex := range parseChildren {
				parseChildren[parseChildIndex] = parseAdapter.CreateElement("span")
			}
			parseAdapter.BeginBatch(parseParent)
			for _, parseChild := range parseChildren {
				parseAdapter.AppendChild(parseParent, parseChild)
			}
			parseAdapter.EndBatch()
		}
	})
	parseB.Run("legacy_fragment_appendChild", func(parseB *testing.B) {
		parseB.ReportAllocs()
		for parseI := 0; parseI < parseB.N; parseI++ {
			parseParent := parseAdapter.CreateElement("div").(*WASMDOMNode)
			parseChildren := make([]runtime.DOMNode, 16)
			for parseChildIndex := range parseChildren {
				parseChildren[parseChildIndex] = parseAdapter.CreateElement("span")
			}
			parseFragment := parseDocument.Call("createDocumentFragment")
			for _, parseChild := range parseChildren {
				parseFragment.Call("appendChild", parseChild.(*WASMDOMNode).value)
			}
			parseParent.value.Call("appendChild", parseFragment)
		}
	})
}

func BenchmarkWASMEventAdapterAddRemoveListener(parseB *testing.B) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseEventAdapter := NewWASMEventAdapter()
	parseNode := parseAdapter.CreateElement("button")
	parseHandler := parseEventAdapter.CreateEventHandler(func(runtime.Event) {})
	defer parseEventAdapter.ReleaseEventHandler(parseHandler)

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseEventAdapter.AddEventListener(parseNode, "click", parseHandler)
		parseEventAdapter.RemoveEventListener(parseNode, "click", parseHandler)
	}
}

func BenchmarkWASMDOMAdapterWrapFunctionNoArgsInvoke(parseB *testing.B) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseWrapped := parseAdapter.WrapFunction(func() {}).(js.Func)
	defer parseWrapped.Release()

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseWrapped.Invoke()
	}
}

func BenchmarkWASMDOMAdapterWrapFunctionStringInvoke(parseB *testing.B) {
	parseCleanup := installBenchmarkDOM()
	defer parseCleanup()

	parseAdapter := NewWASMDOMAdapter()
	parseWrapped := parseAdapter.WrapFunction(func(string) {}).(js.Func)
	defer parseWrapped.Release()

	parseTarget := js.Global().Get("Object").New()
	parseTarget.Set("value", "abc")
	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("target", parseTarget)

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseWrapped.Invoke(parseEvent)
	}
}

func legacyWrapFunction(parseFn interface{}) js.Func {
	return js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		switch parseF := parseFn.(type) {
		case func():
			parseF()
		case func(string):
			if len(parseArgs) > 0 {
				parseEvent := parseArgs[0]
				parseTarget := parseEvent.Get("target")
				if !parseTarget.IsNull() && !parseTarget.IsUndefined() {
					parseValue := parseTarget.Get("value")
					if !parseValue.IsNull() && !parseValue.IsUndefined() {
						parseStrVal := parseValue.String()
						parseF(parseStrVal)
					} else {
						parseF("")
					}
				} else {
					parseF("")
				}
			}
		case func(js.Value):
			if len(parseArgs) > 0 {
				parseF(parseArgs[0])
			}
		case func() error:
			parseF()
		case func(js.Value) error:
			if len(parseArgs) > 0 {
				parseF(parseArgs[0])
			}
		case func(runtime.GoEvent):
			if len(parseArgs) > 0 {
				parseF(runtime.NewGoEvent(parseArgs[0]))
			}
		case func(runtime.GoEvent) error:
			if len(parseArgs) > 0 {
				parseF(runtime.NewGoEvent(parseArgs[0]))
			}
		}
		return nil
	})
}

func BenchmarkLegacyWrapFunctionNoArgsInvoke(parseB *testing.B) {
	parseWrapped := legacyWrapFunction(func() {})
	defer parseWrapped.Release()

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseWrapped.Invoke()
	}
}

func BenchmarkLegacyWrapFunctionStringInvoke(parseB *testing.B) {
	parseTarget := js.Global().Get("Object").New()
	parseTarget.Set("value", "abc")
	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("target", parseTarget)

	parseWrapped := legacyWrapFunction(func(string) {})
	defer parseWrapped.Release()

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseWrapped.Invoke(parseEvent)
	}
}
