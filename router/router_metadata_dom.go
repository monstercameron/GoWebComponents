//go:build js && wasm

package router

import (
	"strings"
	"syscall/js"
)

// applyRouteMetadata is an internal router helper.
func (parseR *Router) applyRouteMetadata(parseOption Options) {
	applyRouteTitle(&parseR.metadataState, parseOption.Title)
	applyRouteMetaTag("description", parseOption.Description)
	applyRouteCanonical(parseOption.CanonicalURL)
}

// applyRouteTitle is an internal router helper.
func applyRouteTitle(parseState *routeMetadataState, parseTitle string) {
	parseDoc := js.Global().Get("document")
	if !parseDoc.Truthy() {
		return
	}
	initializeRouteMetadataState(parseState, parseDoc)

	parseTrimmed := strings.TrimSpace(parseTitle)
	if parseTrimmed == "" {
		if parseState != nil && parseState.titleManaged {
			if parseTitleElement := ensureManagedTitleElement(parseDoc, false); parseTitleElement.Truthy() {
				if strings.TrimSpace(parseState.baseTitle) == "" {
					removeElement(parseTitleElement)
				} else {
					parseTitleElement.Set("textContent", parseState.baseTitle)
					if parseTitleElement.Get("removeAttribute").Truthy() {
						parseTitleElement.Call("removeAttribute", managedMetadataAttr)
					}
				}
			}
			parseDoc.Set("title", parseState.baseTitle)
			parseState.titleManaged = false
		}
		return
	}

	parseTitleElement2 := ensureManagedTitleElement(parseDoc, true)
	if parseTitleElement2.Truthy() {
		parseTitleElement2.Set("textContent", parseTrimmed)
		parseTitleElement2.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
	}
	parseDoc.Set("title", parseTrimmed)
	if parseState != nil {
		parseState.titleManaged = true
	}
}

// applyRouteMetaTag is an internal router helper.
func applyRouteMetaTag(parseName, parseContent string) {
	parseDoc := js.Global().Get("document")
	if !parseDoc.Truthy() {
		return
	}
	parseHead := getHeadElement(parseDoc)
	if !parseHead.Truthy() {
		return
	}
	parseElement := ensureManagedHeadElement(parseDoc, parseHead, `meta[name="`+parseName+`"]`, "meta", func(parseNode js.Value) {
		setElementAttribute(parseNode, "name", parseName)
	})
	parseTrimmed := strings.TrimSpace(parseContent)
	if parseTrimmed == "" {
		if parseElement.Truthy() {
			removeElement(parseElement)
		}
		return
	}
	if !parseElement.Truthy() {
		return
	}
	setElementAttribute(parseElement, managedMetadataAttr, managedMetadataValue)
	setElementAttribute(parseElement, "content", parseTrimmed)
}

// applyRouteCanonical is an internal router helper.
func applyRouteCanonical(parseHref string) {
	parseDoc := js.Global().Get("document")
	if !parseDoc.Truthy() {
		return
	}
	parseHead := getHeadElement(parseDoc)
	if !parseHead.Truthy() {
		return
	}
	parseElement := ensureManagedHeadElement(parseDoc, parseHead, `link[rel="canonical"]`, "link", func(parseNode js.Value) {
		setElementAttribute(parseNode, "rel", "canonical")
	})
	parseTrimmed := strings.TrimSpace(parseHref)
	if parseTrimmed == "" {
		if parseElement.Truthy() {
			removeElement(parseElement)
		}
		return
	}
	if !parseElement.Truthy() {
		return
	}
	setElementAttribute(parseElement, managedMetadataAttr, managedMetadataValue)
	setElementAttribute(parseElement, "href", parseTrimmed)
}

// initializeRouteMetadataState is an internal router helper.
func initializeRouteMetadataState(parseState *routeMetadataState, parseDoc js.Value) {
	if parseState == nil || parseState.baseTitleCaptured {
		return
	}
	parseState.baseTitleCaptured = true
	parseState.baseTitle = parseDoc.Get("title").String()
	if parseTitle := findManagedHeadElement(parseDoc, `title[`+managedMetadataAttr+`="`+managedMetadataValue+`"]`); parseTitle.Truthy() {
		parseState.baseTitle = ""
		parseState.titleManaged = true
	}
}

// ensureManagedTitleElement is an internal router helper.
func ensureManagedTitleElement(parseDoc js.Value, isCreate bool) js.Value {
	if parseElement := findManagedHeadElement(parseDoc, `title[`+managedMetadataAttr+`="`+managedMetadataValue+`"]`); parseElement.Truthy() {
		return parseElement
	}
	parseTitles := querySelectorAll(parseDoc, "title")
	if len(parseTitles) == 1 {
		setElementAttribute(parseTitles[0], managedMetadataAttr, managedMetadataValue)
		return parseTitles[0]
	}
	if !isCreate {
		return js.Null()
	}
	parseHead := getHeadElement(parseDoc)
	if !parseHead.Truthy() {
		return js.Null()
	}
	parseCreateElement := parseDoc.Get("createElement")
	if parseCreateElement.IsUndefined() || parseCreateElement.IsNull() || !parseCreateElement.Truthy() {
		return js.Null()
	}
	parseAppendChild := parseHead.Get("appendChild")
	if parseAppendChild.IsUndefined() || parseAppendChild.IsNull() || !parseAppendChild.Truthy() {
		return js.Null()
	}
	parseElement2 := parseDoc.Call("createElement", "title")
	setElementAttribute(parseElement2, managedMetadataAttr, managedMetadataValue)
	appendChildElement(parseHead, parseElement2)
	return parseElement2
}

// ensureManagedHeadElement is an internal router helper.
func ensureManagedHeadElement(parseDoc js.Value, parseHead js.Value, parseSelector string, parseTag string, parseInitialize func(js.Value)) js.Value {
	parseManagedSelector := parseSelector + `[` + managedMetadataAttr + `="` + managedMetadataValue + `"]`
	if parseElement := findManagedHeadElement(parseDoc, parseManagedSelector); parseElement.Truthy() {
		return parseElement
	}
	parseMatches := querySelectorAll(parseDoc, parseSelector)
	if len(parseMatches) == 1 {
		setElementAttribute(parseMatches[0], managedMetadataAttr, managedMetadataValue)
		if parseInitialize != nil {
			parseInitialize(parseMatches[0])
		}
		return parseMatches[0]
	}
	parseCreateElement := parseDoc.Get("createElement")
	if parseCreateElement.IsUndefined() || parseCreateElement.IsNull() || !parseCreateElement.Truthy() {
		return js.Null()
	}
	parseAppendChild := parseHead.Get("appendChild")
	if parseAppendChild.IsUndefined() || parseAppendChild.IsNull() || !parseAppendChild.Truthy() {
		return js.Null()
	}
	parseElement2 := parseDoc.Call("createElement", parseTag)
	setElementAttribute(parseElement2, managedMetadataAttr, managedMetadataValue)
	if parseInitialize != nil {
		parseInitialize(parseElement2)
	}
	appendChildElement(parseHead, parseElement2)
	return parseElement2
}

// findManagedHeadElement is an internal router helper.
func findManagedHeadElement(parseDoc js.Value, parseSelector string) js.Value {
	parseMatches := querySelectorAll(parseDoc, parseSelector)
	if len(parseMatches) == 0 {
		return js.Null()
	}
	for _, parseExtra := range parseMatches[1:] {
		removeElement(parseExtra)
	}
	return parseMatches[0]
}

// querySelectorAll is an internal router helper.
func querySelectorAll(parseDoc js.Value, parseSelector string) []js.Value {
	if parseDoc.IsUndefined() || parseDoc.IsNull() || !parseDoc.Truthy() {
		return nil
	}
	parseQueryAll := parseDoc.Get("querySelectorAll")
	if parseQueryAll.IsUndefined() || parseQueryAll.IsNull() || !parseQueryAll.Truthy() {
		return nil
	}
	parseList := parseDoc.Call("querySelectorAll", parseSelector)
	if parseList.IsUndefined() || parseList.IsNull() || !parseList.Truthy() {
		return nil
	}
	parseLength := parseList.Get("length").Int()
	if parseLength == 0 {
		return nil
	}
	parseMatches := make([]js.Value, 0, parseLength)
	for parseIndex := 0; parseIndex < parseLength; parseIndex++ {
		parseNode := parseList.Call("item", parseIndex)
		if !parseNode.Truthy() {
			parseNode = parseList.Index(parseIndex)
		}
		if parseNode.Truthy() {
			parseMatches = append(parseMatches, parseNode)
		}
	}
	return parseMatches
}

// removeElement is an internal router helper.
func removeElement(parseNode js.Value) {
	if !parseNode.Truthy() {
		return
	}
	if parseNode.Get("remove").Truthy() {
		parseNode.Call("remove")
		return
	}
	parseParent := parseNode.Get("parentNode")
	if parseParent.Truthy() {
		parseParent.Call("removeChild", parseNode)
	}
}

// getHeadElement is an internal router helper.
func getHeadElement(parseDoc js.Value) js.Value {
	if parseDoc.IsUndefined() || parseDoc.IsNull() || !parseDoc.Truthy() {
		return js.Null()
	}
	parseHead := parseDoc.Get("head")
	if parseHead.Truthy() {
		return parseHead
	}
	parseQuery := parseDoc.Get("querySelector")
	if parseQuery.IsUndefined() || parseQuery.IsNull() || !parseQuery.Truthy() {
		return js.Null()
	}
	return parseDoc.Call("querySelector", "head")
}

// setElementAttribute is an internal router helper.
func setElementAttribute(parseNode js.Value, parseName, parseValue string) bool {
	if parseNode.IsUndefined() || parseNode.IsNull() || !parseNode.Truthy() {
		return false
	}
	parseMethod := parseNode.Get("setAttribute")
	if parseMethod.IsUndefined() || parseMethod.IsNull() || !parseMethod.Truthy() {
		return false
	}
	parseNode.Call("setAttribute", parseName, parseValue)
	return true
}

// appendChildElement is an internal router helper.
func appendChildElement(parseParent js.Value, parseChild js.Value) bool {
	if parseParent.IsUndefined() || parseParent.IsNull() || !parseParent.Truthy() {
		return false
	}
	parseMethod := parseParent.Get("appendChild")
	if parseMethod.IsUndefined() || parseMethod.IsNull() || !parseMethod.Truthy() {
		return false
	}
	parseParent.Call("appendChild", parseChild)
	return true
}
