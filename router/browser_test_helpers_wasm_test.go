//go:build js && wasm
// +build js,wasm

package router

import (
	"strings"
	"syscall/js"
	"testing"
)

func installRouterBrowserEnv(parseT testing.TB) {
	parseT.Helper()
	parseGlobal := js.Global()
	parseObjectCtor := parseGlobal.Get("Object")
	parseArrayCtor := parseGlobal.Get("Array")
	parseReflectObj := parseGlobal.Get("Reflect")

	parsePrevDoc := parseGlobal.Get("document")
	parsePrevElement := parseGlobal.Get("Element")
	parsePrevWindow := parseGlobal.Get("window")
	parsePrevHistory := parseGlobal.Get("history")
	parsePrevLocation := parseGlobal.Get("location")
	parsePrevInitialized := routerRuntimeInitialized
	var parseDecorateNode func(js.Value)
	parseListenerStore := parseObjectCtor.New()

	parseMakeNode := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseTag := ""
		if len(parseArgs) > 0 {
			parseTag = parseArgs[0].String()
		}
		parseNode := parseObjectCtor.New()
		parseNode.Set("tagName", parseTag)
		parseNode.Set("attributes", parseObjectCtor.New())
		parseNode.Set("children", parseArrayCtor.New())
		if parseDecorateNode != nil {
			parseDecorateNode(parseNode)
		}
		return parseNode
	})

	parseAppendChild := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseChild := parseArgs2[0]
		if parseChild.Get("isFragment").Truthy() {
			parseFragmentChildren := parseChild.Get("children")
			parseChildren := parseThis2.Get("children")
			parseLength := parseFragmentChildren.Get("length").Int()
			for parseI := 0; parseI < parseLength; parseI++ {
				parseChildren.Call("push", parseFragmentChildren.Index(parseI))
			}
			parseChild.Set("children", parseArrayCtor.New())
			return parseChild
		}
		parseChild.Set("parentNode", parseThis2)
		parseThis2.Get("children").Call("push", parseChild)
		return parseArgs2[0]
	})
	parseRemoveChild := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		parseChildren2 := parseThis3.Get("children")
		parseLength2 := parseChildren2.Get("length").Int()
		for parseIndex := 0; parseIndex < parseLength2; parseIndex++ {
			if parseChildren2.Index(parseIndex).Equal(parseArgs3[0]) {
				parseChildren2.Call("splice", parseIndex, 1)
				break
			}
		}
		return parseArgs3[0]
	})
	setAttribute := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		parseThis4.Get("attributes").Set(parseArgs4[0].String(), parseArgs4[1].String())
		return nil
	})
	parseRemoveAttribute := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		parseReflectObj.Call("deleteProperty", parseThis5.Get("attributes"), parseArgs5[0].String())
		return nil
	})
	getAttribute := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
		parseValue := parseThis6.Get("attributes").Get(parseArgs6[0].String())
		if parseValue.IsUndefined() {
			return js.Null()
		}
		return parseValue
	})
	parseInsertBefore := js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} {
		parseThis7.Get("children").Call("push", parseArgs7[0])
		return parseArgs7[0]
	})
	parseReplaceChild := js.FuncOf(func(parseThis8 js.Value, parseArgs8 []js.Value) interface{} {
		parseChildren3 := parseThis8.Get("children")
		parseLength3 := parseChildren3.Get("length").Int()
		for parseIndex2 := 0; parseIndex2 < parseLength3; parseIndex2++ {
			if parseChildren3.Index(parseIndex2).Equal(parseArgs8[1]) {
				parseChildren3.SetIndex(parseIndex2, parseArgs8[0])
				break
			}
		}
		return parseArgs8[1]
	})
	parseAddEventListener := js.FuncOf(func(parseThis9 js.Value, parseArgs9 []js.Value) interface{} {
		if len(parseArgs9) < 2 {
			return nil
		}
		parseEventType := parseArgs9[0].String()
		parseHandler := parseArgs9[1]
		parseListeners := parseListenerStore.Get(parseEventType)
		if !parseListeners.Truthy() {
			parseListeners = parseArrayCtor.New()
			parseListenerStore.Set(parseEventType, parseListeners)
		}
		parseListeners.Call("push", parseHandler)
		return nil
	})
	parseRemoveEventListener := js.FuncOf(func(parseThis10 js.Value, parseArgs10 []js.Value) interface{} {
		if len(parseArgs10) < 2 {
			return nil
		}
		parseEventType2 := parseArgs10[0].String()
		parseHandler2 := parseArgs10[1]
		parseListeners2 := parseListenerStore.Get(parseEventType2)
		if !parseListeners2.Truthy() {
			return nil
		}
		parseLength4 := parseListeners2.Get("length").Int()
		for parseIndex3 := 0; parseIndex3 < parseLength4; parseIndex3++ {
			if parseListeners2.Index(parseIndex3).Equal(parseHandler2) {
				parseListeners2.Call("splice", parseIndex3, 1)
				break
			}
		}
		return nil
	})
	parseEmitEvent := func(parseEventType3 string) {
		parseListeners3 := parseListenerStore.Get(parseEventType3)
		if !parseListeners3.Truthy() {
			return
		}
		parseLength5 := parseListeners3.Get("length").Int()
		for parseIndex4 := 0; parseIndex4 < parseLength5; parseIndex4++ {
			parseHandler3 := parseListeners3.Index(parseIndex4)
			if parseHandler3.Truthy() {
				parseHandler3.Invoke()
			}
		}
	}
	parseDispatchEvent := js.FuncOf(func(parseThis11 js.Value, parseArgs11 []js.Value) interface{} {
		if len(parseArgs11) == 0 {
			return false
		}
		parseEventType3 := ""
		if parseArgs11[0].Type() == js.TypeString {
			parseEventType3 = strings.TrimSpace(parseArgs11[0].String())
		} else {
			parseEventType3 = strings.TrimSpace(parseArgs11[0].Get("type").String())
		}
		if parseEventType3 == "" {
			return false
		}
		parseEmitEvent(parseEventType3)
		return true
	})
	parseRemoveNode := js.FuncOf(func(parseThis11 js.Value, parseArgs11 []js.Value) interface{} {
		parseParent := parseThis11.Get("parentNode")
		if !parseParent.Truthy() {
			return nil
		}
		parseChildren4 := parseParent.Get("children")
		parseLength6 := parseChildren4.Get("length").Int()
		for parseIndex5 := 0; parseIndex5 < parseLength6; parseIndex5++ {
			if parseChildren4.Index(parseIndex5).Equal(parseThis11) {
				parseChildren4.Call("splice", parseIndex5, 1)
				break
			}
		}
		parseThis11.Set("parentNode", js.Null())
		return nil
	})
	parseDecorateNode = func(parseNode4 js.Value) {
		parseNode4.Set("appendChild", parseAppendChild)
		parseNode4.Set("removeChild", parseRemoveChild)
		parseNode4.Set("setAttribute", setAttribute)
		parseNode4.Set("removeAttribute", parseRemoveAttribute)
		parseNode4.Set("getAttribute", getAttribute)
		parseNode4.Set("insertBefore", parseInsertBefore)
		parseNode4.Set("replaceChild", parseReplaceChild)
		parseNode4.Set("addEventListener", parseAddEventListener)
		parseNode4.Set("removeEventListener", parseRemoveEventListener)
		parseNode4.Set("remove", parseRemoveNode)
	}

	parseElementCtor := js.FuncOf(func(parseThis12 js.Value, parseArgs12 []js.Value) interface{} {
		return parseObjectCtor.New()
	})
	parseProto := parseObjectCtor.New()
	parseProto.Set("appendChild", parseAppendChild)
	parseProto.Set("removeChild", parseRemoveChild)
	parseProto.Set("setAttribute", setAttribute)
	parseProto.Set("removeAttribute", parseRemoveAttribute)
	parseProto.Set("getAttribute", getAttribute)
	parseProto.Set("insertBefore", parseInsertBefore)
	parseProto.Set("replaceChild", parseReplaceChild)
	parseProto.Set("addEventListener", parseAddEventListener)
	parseProto.Set("removeEventListener", parseRemoveEventListener)
	parseProto.Set("remove", parseRemoveNode)
	parseElementCtor.Set("prototype", parseProto)

	parseFindHeadChild := func(parseHead2 js.Value, parseTag2 string, parseAttrName string, parseAttrValue string) js.Value {
		parseChildren5 := parseHead2.Get("children")
		parseLength7 := parseChildren5.Get("length").Int()
		for parseIndex6 := 0; parseIndex6 < parseLength7; parseIndex6++ {
			parseChild2 := parseChildren5.Index(parseIndex6)
			if !strings.EqualFold(parseChild2.Get("tagName").String(), parseTag2) {
				continue
			}
			if parseChild2.Get("attributes").Get(parseAttrName).String() == parseAttrValue {
				return parseChild2
			}
		}
		return js.Null()
	}
	parseFindHeadChildren := func(parseHead3 js.Value, parseTag3 string, parseAttrName2 string, parseAttrValue2 string, isManagedOnly bool) js.Value {
		parseList := parseArrayCtor.New()
		parseChildren6 := parseHead3.Get("children")
		parseLength8 := parseChildren6.Get("length").Int()
		for parseIndex7 := 0; parseIndex7 < parseLength8; parseIndex7++ {
			parseChild3 := parseChildren6.Index(parseIndex7)
			if !strings.EqualFold(parseChild3.Get("tagName").String(), parseTag3) {
				continue
			}
			if parseAttrName2 != "" && parseChild3.Get("attributes").Get(parseAttrName2).String() != parseAttrValue2 {
				continue
			}
			if isManagedOnly && parseChild3.Get("attributes").Get(managedMetadataAttr).String() != managedMetadataValue {
				continue
			}
			parseList.Call("push", parseChild3)
		}
		parseList.Set("item", js.FuncOf(func(parseThis13 js.Value, parseArgs13 []js.Value) interface{} {
			return parseThis13.Index(parseArgs13[0].Int())
		}))
		return parseList
	}

	parseDocCreateElement := js.FuncOf(func(parseThis14 js.Value, parseArgs14 []js.Value) interface{} {
		return parseMakeNode.Invoke(parseArgs14[0].String())
	})
	parseDocCreateTextNode := js.FuncOf(func(parseThis15 js.Value, parseArgs15 []js.Value) interface{} {
		parseNode2 := parseObjectCtor.New()
		parseNode2.Set("textContent", parseArgs15[0].String())
		return parseNode2
	})
	parseDocCreateFragment := js.FuncOf(func(parseThis16 js.Value, parseArgs16 []js.Value) interface{} {
		parseFrag := parseObjectCtor.New()
		parseFrag.Set("isFragment", true)
		parseFrag.Set("children", parseArrayCtor.New())
		parseFrag.Set("appendChild", parseAppendChild)
		return parseFrag
	})
	parseHead := parseMakeNode.Invoke("head")
	parseBody := parseMakeNode.Invoke("body")
	parseDocQuerySelector := js.FuncOf(func(parseThis17 js.Value, parseArgs17 []js.Value) interface{} {
		parseSelector := parseArgs17[0].String()
		switch parseSelector {
		case "head":
			return parseHead
		case "title":
			return parseFindHeadChild(parseHead, "title", "", "")
		case `title[data-gwc-router-managed="true"]`:
			parseChildren7 := parseFindHeadChildren(parseHead, "title", "", "", true)
			if parseChildren7.Get("length").Int() > 0 {
				return parseChildren7.Index(0)
			}
			return js.Null()
		case `meta[name="description"]`:
			return parseFindHeadChild(parseHead, "meta", "name", "description")
		case `meta[name="description"][data-gwc-router-managed="true"]`:
			parseChildren8 := parseFindHeadChildren(parseHead, "meta", "name", "description", true)
			if parseChildren8.Get("length").Int() > 0 {
				return parseChildren8.Index(0)
			}
			return js.Null()
		case `link[rel="canonical"]`:
			return parseFindHeadChild(parseHead, "link", "rel", "canonical")
		case `link[rel="canonical"][data-gwc-router-managed="true"]`:
			parseChildren9 := parseFindHeadChildren(parseHead, "link", "rel", "canonical", true)
			if parseChildren9.Get("length").Int() > 0 {
				return parseChildren9.Index(0)
			}
			return js.Null()
		default:
			return parseMakeNode.Invoke("div")
		}
	})
	parseDocQuerySelectorAll := js.FuncOf(func(parseThis18 js.Value, parseArgs18 []js.Value) interface{} {
		switch parseArgs18[0].String() {
		case "title":
			return parseFindHeadChildren(parseHead, "title", "", "", false)
		case `title[data-gwc-router-managed="true"]`:
			return parseFindHeadChildren(parseHead, "title", "", "", true)
		case `meta[name="description"]`:
			return parseFindHeadChildren(parseHead, "meta", "name", "description", false)
		case `meta[name="description"][data-gwc-router-managed="true"]`:
			return parseFindHeadChildren(parseHead, "meta", "name", "description", true)
		case `link[rel="canonical"]`:
			return parseFindHeadChildren(parseHead, "link", "rel", "canonical", false)
		case `link[rel="canonical"][data-gwc-router-managed="true"]`:
			return parseFindHeadChildren(parseHead, "link", "rel", "canonical", true)
		default:
			parseList2 := parseArrayCtor.New()
			parseList2.Set("item", js.FuncOf(func(parseThis19 js.Value, parseArgs19 []js.Value) interface{} {
				return parseThis19.Index(parseArgs19[0].Int())
			}))
			return parseList2
		}
	})
	parseDocGetElementByID := js.FuncOf(func(parseThis20 js.Value, parseArgs20 []js.Value) interface{} {
		parseNode3 := parseMakeNode.Invoke("div")
		if len(parseArgs20) > 0 {
			parseNode3.Set("id", parseArgs20[0].String())
		}
		return parseNode3
	})
	parseDocGetElementsByClassName := js.FuncOf(func(parseThis21 js.Value, parseArgs21 []js.Value) interface{} {
		parseList3 := parseArrayCtor.New()
		parseList3.Call("push", parseMakeNode.Invoke("div"))
		parseList3.Set("item", js.FuncOf(func(parseThis22 js.Value, parseArgs22 []js.Value) interface{} {
			return parseThis22.Index(parseArgs22[0].Int())
		}))
		return parseList3
	})
	parseDocGetElementsByTagName := js.FuncOf(func(parseThis23 js.Value, parseArgs23 []js.Value) interface{} {
		parseList4 := parseArrayCtor.New()
		parseList4.Call("push", parseMakeNode.Invoke("div"))
		parseList4.Set("item", js.FuncOf(func(parseThis24 js.Value, parseArgs24 []js.Value) interface{} {
			return parseThis24.Index(parseArgs24[0].Int())
		}))
		return parseList4
	})
	parseDoc := parseObjectCtor.New()
	parseDoc.Set("createElement", parseDocCreateElement)
	parseDoc.Set("createTextNode", parseDocCreateTextNode)
	parseDoc.Set("createDocumentFragment", parseDocCreateFragment)
	parseDoc.Set("querySelector", parseDocQuerySelector)
	parseDoc.Set("querySelectorAll", parseDocQuerySelectorAll)
	parseDoc.Set("getElementById", parseDocGetElementByID)
	parseDoc.Set("getElementsByClassName", parseDocGetElementsByClassName)
	parseDoc.Set("getElementsByTagName", parseDocGetElementsByTagName)
	parseDoc.Set("head", parseHead)
	parseDoc.Set("body", parseBody)
	parseDoc.Set("title", "")

	parseLocation := parseObjectCtor.New()
	parseLocation.Set("hash", "")
	parseLocation.Set("pathname", "/")
	parseLocation.Set("search", "")
	parseLocation.Set("reload", js.FuncOf(func(parseThis25 js.Value, parseArgs25 []js.Value) interface{} { return nil }))
	parseLocationReplace := js.FuncOf(func(parseThis26 js.Value, parseArgs26 []js.Value) interface{} {
		parseNext := parseArgs26[0].String()
		if len(parseNext) > 0 && parseNext[0] != '#' {
			parseLocation.Set("hash", "#"+parseNext)
		} else {
			parseLocation.Set("hash", parseNext)
		}
		if parseIndex8 := strings.Index(parseNext, "?"); parseIndex8 >= 0 {
			parseLocation.Set("search", parseNext[parseIndex8:])
		} else {
			parseLocation.Set("search", "")
		}
		return nil
	})
	parseLocation.Set("replace", parseLocationReplace)

	parseHistory := parseObjectCtor.New()
	parseHistoryEntries := []string{"/"}
	parseHistoryIndex := 0
	applyHistoryTarget := func(parseTarget string) {
		parsePathTarget := parseTarget
		parseHashTarget := ""
		if parseIdx := strings.Index(parseTarget, "#"); parseIdx >= 0 {
			parsePathTarget = parseTarget[:parseIdx]
			parseHashTarget = parseTarget[parseIdx:]
		}
		if parseIdx := strings.Index(parsePathTarget, "?"); parseIdx >= 0 {
			parseLocation.Set("pathname", parsePathTarget[:parseIdx])
			parseLocation.Set("search", parsePathTarget[parseIdx:])
		} else {
			parseLocation.Set("pathname", parsePathTarget)
			parseLocation.Set("search", "")
		}
		parseLocation.Set("hash", parseHashTarget)
	}
	parsePushState := js.FuncOf(func(parseThis27 js.Value, parseArgs27 []js.Value) interface{} {
		if len(parseArgs27) > 2 {
			parseNext2 := parseArgs27[2].String()
			if parseHistoryIndex < len(parseHistoryEntries)-1 {
				parseHistoryEntries = append([]string(nil), parseHistoryEntries[:parseHistoryIndex+1]...)
			}
			parseHistoryEntries = append(parseHistoryEntries, parseNext2)
			parseHistoryIndex = len(parseHistoryEntries) - 1
			applyHistoryTarget(parseNext2)
		}
		return nil
	})
	parseReplaceState := js.FuncOf(func(parseThis28 js.Value, parseArgs28 []js.Value) interface{} {
		if len(parseArgs28) > 2 {
			parseNext3 := parseArgs28[2].String()
			if len(parseHistoryEntries) == 0 {
				parseHistoryEntries = append(parseHistoryEntries, parseNext3)
				parseHistoryIndex = 0
			} else {
				parseHistoryEntries[parseHistoryIndex] = parseNext3
			}
			applyHistoryTarget(parseNext3)
		}
		return nil
	})
	parseBack := js.FuncOf(func(parseThis29 js.Value, parseArgs29 []js.Value) interface{} {
		if parseHistoryIndex == 0 {
			return nil
		}
		parseHistoryIndex--
		applyHistoryTarget(parseHistoryEntries[parseHistoryIndex])
		parseEmitEvent(browserEventPop)
		return nil
	})
	parseForward := js.FuncOf(func(parseThis30 js.Value, parseArgs30 []js.Value) interface{} {
		if parseHistoryIndex >= len(parseHistoryEntries)-1 {
			return nil
		}
		parseHistoryIndex++
		applyHistoryTarget(parseHistoryEntries[parseHistoryIndex])
		parseEmitEvent(browserEventPop)
		return nil
	})
	parseHistory.Set("pushState", parsePushState)
	parseHistory.Set("replaceState", parseReplaceState)
	parseHistory.Set("back", parseBack)
	parseHistory.Set("forward", parseForward)

	parseStorage := parseObjectCtor.New()
	parseStorageData := parseObjectCtor.New()
	parseStorage.Set("setItem", js.FuncOf(func(parseThis31 js.Value, parseArgs31 []js.Value) interface{} {
		parseStorageData.Set(parseArgs31[0].String(), parseArgs31[1].String())
		return nil
	}))
	parseStorage.Set("getItem", js.FuncOf(func(parseThis32 js.Value, parseArgs32 []js.Value) interface{} {
		parseValue2 := parseStorageData.Get(parseArgs32[0].String())
		if parseValue2.IsUndefined() {
			return js.Null()
		}
		return parseValue2
	}))
	parseStorage.Set("removeItem", js.FuncOf(func(parseThis33 js.Value, parseArgs33 []js.Value) interface{} {
		parseReflectObj.Call("deleteProperty", parseStorageData, parseArgs33[0].String())
		return nil
	}))

	parseWindow := parseObjectCtor.New()
	parseWindow.Set("addEventListener", parseAddEventListener)
	parseWindow.Set("removeEventListener", parseRemoveEventListener)
	parseWindow.Set("dispatchEvent", parseDispatchEvent)
	parseWindow.Set("document", parseDoc)
	parseWindow.Set("history", parseHistory)
	parseWindow.Set("location", parseLocation)

	parseGlobal.Set("document", parseDoc)
	parseGlobal.Set("Element", parseElementCtor)
	parseGlobal.Set("window", parseWindow)
	parseGlobal.Set("history", parseHistory)
	parseGlobal.Set("location", parseLocation)
	routerRuntimeInitialized = false

	parseT.Cleanup(func() {
		parseGlobal.Set("document", parsePrevDoc)
		parseGlobal.Set("Element", parsePrevElement)
		parseGlobal.Set("window", parsePrevWindow)
		parseGlobal.Set("history", parsePrevHistory)
		parseGlobal.Set("location", parsePrevLocation)
		routerRuntimeInitialized = parsePrevInitialized
		parseMakeNode.Release()
		parseAppendChild.Release()
		parseRemoveChild.Release()
		setAttribute.Release()
		parseRemoveAttribute.Release()
		getAttribute.Release()
		parseInsertBefore.Release()
		parseReplaceChild.Release()
		parseAddEventListener.Release()
		parseRemoveEventListener.Release()
		parseRemoveNode.Release()
		parseElementCtor.Release()
		parseDocCreateElement.Release()
		parseDocCreateTextNode.Release()
		parseDocCreateFragment.Release()
		parseDocQuerySelector.Release()
		parseDocQuerySelectorAll.Release()
		parseDocGetElementByID.Release()
		parseDocGetElementsByClassName.Release()
		parseDocGetElementsByTagName.Release()
		parseLocationReplace.Release()
		parsePushState.Release()
		parseReplaceState.Release()
		parseBack.Release()
		parseForward.Release()
	})
}
