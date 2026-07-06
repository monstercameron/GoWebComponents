//go:build js && wasm
// +build js,wasm

package jsdom

import (
	"fmt"
	"html"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// WASMDOMNode wraps a js.Value representing a DOM node.
type WASMDOMNode struct {
	value js.Value
	// batchID caches the node's JS-side attr-batch registry id (0 = not yet
	// registered; ids start at 1). See attr_batch.go.
	batchID int
}

var _ runtime.DOMNode = (*WASMDOMNode)(nil)

// NewWASMDOMNode wraps a raw js.Value as a runtime.DOMNode.
func NewWASMDOMNode(parseValue js.Value) runtime.DOMNode {
	return &WASMDOMNode{value: parseValue}
}

func (parseN *WASMDOMNode) IsNull() bool {
	return parseN == nil || parseN.value.IsNull() || parseN.value.IsUndefined()
}

// liveWasmNode resolves a runtime.DOMNode to a *WASMDOMNode only when it is a
// live element — a wrong concrete type, a typed-nil pointer, or a wrapper around
// a null/undefined js.Value all yield ok=false. Reaching through .value on any
// of those panics in syscall/js ("not an object"), which would crash the whole
// app from the render/commit path, so every adapter method that dereferences
// .value must gate on this rather than the bare type assertion.
func liveWasmNode(parseNode runtime.DOMNode) (*WASMDOMNode, bool) {
	parseWasmNode, parseOk := parseNode.(*WASMDOMNode)
	if !parseOk || parseWasmNode.IsNull() {
		return nil, false
	}
	return parseWasmNode, true
}

func (parseN *WASMDOMNode) Equals(parseOther runtime.DOMNode) bool {
	if parseN == nil {
		return runtime.IsDOMNodeNull(parseOther)
	}
	if runtime.IsDOMNodeNull(parseOther) {
		return false
	}
	if parseOtherNode, parseOk := parseOther.(*WASMDOMNode); parseOk && parseOtherNode != nil {
		return parseN.value.Equal(parseOtherNode.value)
	}
	return false
}

func (parseN *WASMDOMNode) Value() js.Value {
	return parseN.value
}

// Focus implements runtime.Focuser by calling the element's focus() method. It is
// a no-op when the underlying value is absent or not focusable.
func (parseN *WASMDOMNode) Focus() {
	if parseN == nil || !parseN.value.Truthy() {
		return
	}
	if parseFn := parseN.value.Get("focus"); parseFn.Type() == js.TypeFunction {
		parseN.value.Call("focus")
	}
}

// callMethod invokes a zero/one-arg DOM method on the element when present, no-op otherwise.
// Shared by the imperative-handle helpers below so each stays a one-liner like Focus.
func (parseN *WASMDOMNode) callMethod(parseName string, parseArgs ...any) {
	if parseN == nil || !parseN.value.Truthy() {
		return
	}
	if parseFn := parseN.value.Get(parseName); parseFn.Type() == js.TypeFunction {
		parseN.value.Call(parseName, parseArgs...)
	}
}

// Blur implements runtime.Blurrer by calling the element's blur() method.
func (parseN *WASMDOMNode) Blur() { parseN.callMethod("blur") }

// Click implements runtime.Clicker by calling the element's click() method (synthesizes a click).
func (parseN *WASMDOMNode) Click() { parseN.callMethod("click") }

// ScrollIntoView implements runtime.ScrollIntoViewer. An empty behavior uses the browser default;
// "smooth"/"instant"/"auto" map to scrollIntoView({behavior}).
func (parseN *WASMDOMNode) ScrollIntoView(parseBehavior string) {
	if parseBehavior == "" {
		parseN.callMethod("scrollIntoView")
		return
	}
	parseOptions := js.Global().Get("Object").New()
	parseOptions.Set("behavior", parseBehavior)
	parseN.callMethod("scrollIntoView", parseOptions)
}

// WASMDOMAdapter implements runtime.DOMAdapter for browser/WASM.
type WASMDOMAdapter struct {
	document                js.Value
	isDocumentBound         bool
	createElement           js.Value
	isCreateElementBound    bool
	createElementNS         js.Value
	isCreateElementNSBound  bool
	createTextNode          js.Value
	isCreateTextNodeBound   bool
	querySelector           js.Value
	isQuerySelectorBound    bool
	querySelectorAll        js.Value
	isQuerySelectorAllBound bool
	getElementByID          js.Value
	isGetElementByIDBound   bool
	getByClassName          js.Value
	isGetByClassNameBound   bool
	getByTagName            js.Value
	isGetByTagNameBound     bool
	storeTemplate           js.Value
	storeTemplateContent    js.Value
	isStoreTemplateBound    bool
	// Batch operation support
	batchStack             []wasmBatchState
	storeBatchChildrenPool [][]interface{}
	// appendChecked is true once we have verified that DOM nodes support the
	// multi-arg append() method; appendFast is true if they do (false = use
	// appendChild fallback).  Both are set on the first appendDOMChildren call.
	appendChecked bool
	appendFast    bool
	// Cross-node attribute update batching state (see attr_batch.go).
	attrBatchActive   bool
	attrHelpersBound  bool
	attrHelpersFailed bool
	attrBatchPayload  strings.Builder
	attrRegisterNode  js.Value
	attrFlushBatch    js.Value
}

type wasmBatchState struct {
	parent   *WASMDOMNode
	children []interface{}
}

var _ runtime.DOMAdapter = (*WASMDOMAdapter)(nil)

// NewWASMDOMAdapter creates a DOM adapter backed by the browser document.
func NewWASMDOMAdapter() *WASMDOMAdapter {
	return &WASMDOMAdapter{}
}

func (parseA *WASMDOMAdapter) getDocument() js.Value {
	if parseA == nil {
		return js.Undefined()
	}
	if parseA.isDocumentBound && !parseA.document.IsNull() && !parseA.document.IsUndefined() {
		return parseA.document
	}
	parseA.document = js.Global().Get("document")
	parseA.isDocumentBound = true
	return parseA.document
}

func bindWASMDocumentMethod(parseDoc js.Value, parseName string) js.Value {
	if parseDoc.IsNull() || parseDoc.IsUndefined() {
		return js.Undefined()
	}
	parseMethod := parseDoc.Get(parseName)
	if parseMethod.IsNull() || parseMethod.IsUndefined() || parseMethod.Type() != js.TypeFunction {
		return js.Undefined()
	}
	return parseMethod.Call("bind", parseDoc)
}

func (parseA *WASMDOMAdapter) getCreateElement() js.Value {
	if parseA.isCreateElementBound && !parseA.createElement.IsNull() && !parseA.createElement.IsUndefined() {
		return parseA.createElement
	}
	parseA.createElement = bindWASMDocumentMethod(parseA.getDocument(), "createElement")
	parseA.isCreateElementBound = true
	return parseA.createElement
}

// svgNamespace is the namespace URI every SVG element must be created under;
// elements created with document.createElement (the HTML namespace) never paint
// as SVG, so the reconciler routes SVG tags through createElementNS instead.
const svgNamespace = "http://www.w3.org/2000/svg"

// svgTagSet lists the SVG-only element names the reconciler may encounter. These
// names don't collide with HTML elements, so a tag-name lookup unambiguously
// identifies an SVG node without tracking the parent namespace. (Names also
// shared with HTML — a, title, script, style, text — are intentionally omitted so
// HTML usage is never misrouted; inline icon sets don't rely on them.)
var svgTagSet = map[string]struct{}{
	"svg": {}, "g": {}, "defs": {}, "use": {}, "symbol": {}, "marker": {},
	"path": {}, "rect": {}, "circle": {}, "ellipse": {}, "line": {},
	"polyline": {}, "polygon": {}, "tspan": {}, "clippath": {}, "mask": {},
	"pattern": {}, "image": {}, "foreignobject": {},
	"lineargradient": {}, "radialgradient": {}, "stop": {},
}

// isSVGTag reports whether a tag must be created in the SVG namespace.
func isSVGTag(parseTag string) bool {
	_, parseOK := svgTagSet[strings.ToLower(strings.TrimSpace(parseTag))]
	return parseOK
}

func (parseA *WASMDOMAdapter) getCreateElementNS() js.Value {
	if parseA.isCreateElementNSBound && !parseA.createElementNS.IsNull() && !parseA.createElementNS.IsUndefined() {
		return parseA.createElementNS
	}
	parseA.createElementNS = bindWASMDocumentMethod(parseA.getDocument(), "createElementNS")
	parseA.isCreateElementNSBound = true
	return parseA.createElementNS
}

// createHostElement builds one raw DOM element, choosing createElementNS for SVG
// tags and createElement for everything else. Returns js.Null on failure.
func (parseA *WASMDOMAdapter) createHostElement(parseTag string) js.Value {
	if isSVGTag(parseTag) {
		parseCreateNS := parseA.getCreateElementNS()
		if !parseCreateNS.IsNull() && !parseCreateNS.IsUndefined() {
			parseElem := parseCreateNS.Invoke(svgNamespace, parseTag)
			if !parseElem.IsNull() && !parseElem.IsUndefined() {
				return parseElem
			}
		}
		// Fall through to createElement if createElementNS is unavailable.
	}
	parseCreate := parseA.getCreateElement()
	if parseCreate.IsNull() || parseCreate.IsUndefined() {
		return js.Null()
	}
	parseElem := parseCreate.Invoke(parseTag)
	if parseElem.IsNull() || parseElem.IsUndefined() {
		return js.Null()
	}
	return parseElem
}

func (parseA *WASMDOMAdapter) getCreateTextNode() js.Value {
	if parseA.isCreateTextNodeBound && !parseA.createTextNode.IsNull() && !parseA.createTextNode.IsUndefined() {
		return parseA.createTextNode
	}
	parseA.createTextNode = bindWASMDocumentMethod(parseA.getDocument(), "createTextNode")
	parseA.isCreateTextNodeBound = true
	return parseA.createTextNode
}

func (parseA *WASMDOMAdapter) getQuerySelector() js.Value {
	if parseA.isQuerySelectorBound && !parseA.querySelector.IsNull() && !parseA.querySelector.IsUndefined() {
		return parseA.querySelector
	}
	parseA.querySelector = bindWASMDocumentMethod(parseA.getDocument(), "querySelector")
	parseA.isQuerySelectorBound = true
	return parseA.querySelector
}

func (parseA *WASMDOMAdapter) getQuerySelectorAll() js.Value {
	if parseA.isQuerySelectorAllBound && !parseA.querySelectorAll.IsNull() && !parseA.querySelectorAll.IsUndefined() {
		return parseA.querySelectorAll
	}
	parseA.querySelectorAll = bindWASMDocumentMethod(parseA.getDocument(), "querySelectorAll")
	parseA.isQuerySelectorAllBound = true
	return parseA.querySelectorAll
}

func (parseA *WASMDOMAdapter) getElementByIdMethod() js.Value {
	if parseA.isGetElementByIDBound && !parseA.getElementByID.IsNull() && !parseA.getElementByID.IsUndefined() {
		return parseA.getElementByID
	}
	parseA.getElementByID = bindWASMDocumentMethod(parseA.getDocument(), "getElementById")
	parseA.isGetElementByIDBound = true
	return parseA.getElementByID
}

func (parseA *WASMDOMAdapter) getElementsByClassNameMethod() js.Value {
	if parseA.isGetByClassNameBound && !parseA.getByClassName.IsNull() && !parseA.getByClassName.IsUndefined() {
		return parseA.getByClassName
	}
	parseA.getByClassName = bindWASMDocumentMethod(parseA.getDocument(), "getElementsByClassName")
	parseA.isGetByClassNameBound = true
	return parseA.getByClassName
}

func (parseA *WASMDOMAdapter) getElementsByTagNameMethod() js.Value {
	if parseA.isGetByTagNameBound && !parseA.getByTagName.IsNull() && !parseA.getByTagName.IsUndefined() {
		return parseA.getByTagName
	}
	parseA.getByTagName = bindWASMDocumentMethod(parseA.getDocument(), "getElementsByTagName")
	parseA.isGetByTagNameBound = true
	return parseA.getByTagName
}

func (parseA *WASMDOMAdapter) ensureStoreTemplate() bool {
	if parseA.isStoreTemplateBound &&
		!parseA.storeTemplate.IsNull() && !parseA.storeTemplate.IsUndefined() &&
		!parseA.storeTemplateContent.IsNull() && !parseA.storeTemplateContent.IsUndefined() {
		return true
	}
	parseCreateElement := parseA.getCreateElement()
	if parseCreateElement.IsNull() || parseCreateElement.IsUndefined() {
		return false
	}
	parseA.storeTemplate = parseCreateElement.Invoke("template")
	if parseA.storeTemplate.IsNull() || parseA.storeTemplate.IsUndefined() {
		return false
	}
	parseA.storeTemplateContent = parseA.storeTemplate.Get("content")
	parseA.isStoreTemplateBound = true
	return !parseA.storeTemplateContent.IsNull() && !parseA.storeTemplateContent.IsUndefined()
}

func (parseA *WASMDOMAdapter) CreateElement(parseTag string) runtime.DOMNode {
	// createHostElement picks createElementNS for SVG tags (which never paint when
	// built in the HTML namespace) and createElement otherwise.
	parseElem := parseA.createHostElement(parseTag)
	if parseElem.IsNull() || parseElem.IsUndefined() {
		// Document not available, or creation failed - return null node
		return &WASMDOMNode{value: js.Null()}
	}
	return &WASMDOMNode{value: parseElem}
}

func (parseA *WASMDOMAdapter) CreateTextNode(parseText string) runtime.DOMNode {
	parseCreateTextNode := parseA.getCreateTextNode()
	if parseCreateTextNode.IsNull() || parseCreateTextNode.IsUndefined() {
		return &WASMDOMNode{value: js.Null()}
	}

	// Use Invoke on the cached function
	parseTextNode := parseCreateTextNode.Invoke(parseText)
	if parseTextNode.IsNull() || parseTextNode.IsUndefined() {
		return &WASMDOMNode{value: js.Null()}
	}
	return &WASMDOMNode{value: parseTextNode}
}

// CreatePreparedElement creates one compact host element using one conservative template fast path when it is safe to do so.
func (parseA *WASMDOMAdapter) CreatePreparedElement(parseTag string, parseAttrs []runtime.HostAttr, parseText string) runtime.DOMNode {
	parseCreateElement := parseA.getCreateElement()
	if parseCreateElement.IsNull() || parseCreateElement.IsUndefined() {
		return &WASMDOMNode{value: js.Null()}
	}
	// SVG elements must be created in the SVG namespace, and the HTML-string
	// template fast path below can't produce SVG nodes — build them directly.
	if isSVGTag(parseTag) {
		parseElem := parseA.createHostElement(parseTag)
		if parseElem.IsNull() || parseElem.IsUndefined() {
			return &WASMDOMNode{value: js.Null()}
		}
		getNode := &WASMDOMNode{value: parseElem}
		if len(parseAttrs) > 0 {
			parseA.BatchSetAttributes(getNode, buildHostAttrMap(parseAttrs))
		}
		if parseText != "" {
			parseElem.Set("textContent", parseText)
		}
		return getNode
	}
	if len(parseAttrs) == 0 {
		parseNode := parseCreateElement.Invoke(parseTag)
		if parseNode.IsNull() || parseNode.IsUndefined() {
			return &WASMDOMNode{value: js.Null()}
		}
		if parseText != "" {
			parseNode.Set("textContent", parseText)
		}
		return &WASMDOMNode{value: parseNode}
	}
	if getHTML, hasHTML := buildHostElementHTML(parseTag, parseAttrs, parseText); hasHTML {
		if !parseA.ensureStoreTemplate() {
			parseNode := parseCreateElement.Invoke(parseTag)
			if parseNode.IsNull() || parseNode.IsUndefined() {
				return &WASMDOMNode{value: js.Null()}
			}
			getNode := &WASMDOMNode{value: parseNode}
			parseA.BatchSetAttributes(getNode, buildHostAttrMap(parseAttrs))
			if parseText != "" {
				parseNode.Set("textContent", parseText)
			}
			return getNode
		}
		parseA.storeTemplate.Set("innerHTML", getHTML)
		parseNode := parseA.storeTemplateContent.Get("firstChild")
		if !parseNode.IsNull() && !parseNode.IsUndefined() {
			return &WASMDOMNode{value: parseNode}
		}
		// The HTML parser drops some elements from template content (html, head,
		// body, frame) — fall through to direct createElement instead of handing
		// the reconciler a null node.
	}
	parseNode := parseCreateElement.Invoke(parseTag)
	if parseNode.IsNull() || parseNode.IsUndefined() {
		return &WASMDOMNode{value: js.Null()}
	}
	getNode := &WASMDOMNode{value: parseNode}
	parseA.BatchSetAttributes(getNode, buildHostAttrMap(parseAttrs))
	if parseText != "" {
		parseNode.Set("textContent", parseText)
	}
	return getNode
}

// CreateHTMLFragment parses several serialized sibling subtrees through the
// shared template element in one bridge call and returns the template
// CONTENT node; its children are the parsed roots (still detached — walking
// then appending them moves each out of the fragment).
func (parseA *WASMDOMAdapter) CreateHTMLFragment(parseHTML string) runtime.DOMNode {
	if parseHTML == "" || !parseA.ensureStoreTemplate() {
		return &WASMDOMNode{value: js.Null()}
	}
	parseA.storeTemplate.Set("innerHTML", parseHTML)
	return &WASMDOMNode{value: parseA.storeTemplateContent}
}

// CreateHTMLSubtree parses one serialized HTML subtree through the shared
// template element in a single bridge call and returns its root node (still
// detached; appending it later moves it out of the template content).
func (parseA *WASMDOMAdapter) CreateHTMLSubtree(parseHTML string) runtime.DOMNode {
	if parseHTML == "" || !parseA.ensureStoreTemplate() {
		return &WASMDOMNode{value: js.Null()}
	}
	parseA.storeTemplate.Set("innerHTML", parseHTML)
	parseNode := parseA.storeTemplateContent.Get("firstChild")
	if parseNode.IsNull() || parseNode.IsUndefined() {
		return &WASMDOMNode{value: js.Null()}
	}
	return &WASMDOMNode{value: parseNode}
}

func (parseA *WASMDOMAdapter) SetAttribute(parseNode runtime.DOMNode, parseName, parseValue string) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		// Block javascript:/vbscript: URLs in href/src/action so a user-controlled
		// URL rendered through the normal element API cannot execute on click —
		// parity with the SSR serializer's sanitizer.
		parseValue = runtime.SanitizeURLAttributeValue(parseName, parseValue)
		if parseA.queueAttrWrite(parseWasmNode, 'a', parseName, parseValue) {
			return
		}
		parseWasmNode.value.Call("setAttribute", parseName, parseValue)
	}
}

// GetAttribute reports one attribute value from one wasm DOM node.
func (parseA *WASMDOMAdapter) GetAttribute(parseNode runtime.DOMNode, parseName string) string {
	parseWasmNode, parseOk := parseNode.(*WASMDOMNode)
	if !parseOk || runtime.IsDOMNodeNull(parseWasmNode) {
		return ""
	}
	parseAttributeValue := parseWasmNode.value.Call("getAttribute", parseName)
	if parseAttributeValue.IsNull() || parseAttributeValue.IsUndefined() {
		return ""
	}
	return parseAttributeValue.String()
}

func (parseA *WASMDOMAdapter) RemoveAttribute(parseNode runtime.DOMNode, parseName string) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		if parseA.queueAttrWrite(parseWasmNode, 'r', parseName, "") {
			return
		}
		parseWasmNode.value.Call("removeAttribute", parseName)
	}
}

func (parseA *WASMDOMAdapter) SetProperty(parseNode runtime.DOMNode, parseName string, parseValue interface{}) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		// Direct property set (fastest path)
		parseWasmNode.value.Set(parseName, parseValue)
	}
}

func (parseA *WASMDOMAdapter) AddPassiveEventListener(parseNode runtime.DOMNode, parseEventType string, parseHandler any) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		parseOptions := js.Global().Get("Object").New()
		parseOptions.Set("passive", true)
		parseWasmNode.value.Call("addEventListener", parseEventType, parseHandler, parseOptions)
	}
}

func (parseA *WASMDOMAdapter) RemovePassiveEventListener(parseNode runtime.DOMNode, parseEventType string, parseHandler any) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		parseWasmNode.value.Call("removeEventListener", parseEventType, parseHandler)
	}
}

func (parseA *WASMDOMAdapter) GetProperty(parseNode runtime.DOMNode, parseName string) interface{} {
	if runtime.IsDOMNodeNull(parseNode) {
		return nil
	}
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		if !parseWasmNode.value.IsNull() && !parseWasmNode.value.IsUndefined() {
			return parseWasmNode.value.Get(parseName)
		}
	}
	return nil
}

func (parseA *WASMDOMAdapter) AppendChild(parseParent, parseChild runtime.DOMNode) {
	// Fast path: skip nil checks when nodes are valid
	parseParentNode, parseOk1 := parseParent.(*WASMDOMNode)
	parseChildNode, parseOk2 := parseChild.(*WASMDOMNode)
	if !parseOk1 || !parseOk2 {
		return
	}

	// If in batch mode for this specific parent, buffer nodes locally and flush once at EndBatch.
	if parseDepth := len(parseA.batchStack); parseDepth > 0 {
		parseState := &parseA.batchStack[parseDepth-1]
		if parseState.parent == parseParentNode {
			parseState.children = append(parseState.children, parseChildNode.value)
			return
		}
	}

	parseA.appendDOMChildren(parseParentNode.value, parseChildNode.value)
}

func (parseA *WASMDOMAdapter) RemoveChild(parseParent, parseChild runtime.DOMNode) {
	parseParentNode, parseOk1 := parseParent.(*WASMDOMNode)
	parseChildNode, parseOk2 := parseChild.(*WASMDOMNode)
	if !parseOk1 || !parseOk2 {
		return
	}
	parseChildParent := parseChildNode.value.Get("parentNode")
	if parseChildParent.IsNull() || parseChildParent.IsUndefined() || !parseChildParent.Equal(parseParentNode.value) {
		return
	}
	parseChildNode.value.Call("remove")
}

func (parseA *WASMDOMAdapter) InsertBefore(parseParent, parseNewNode, parseReferenceNode runtime.DOMNode) {
	parseParentN, parseOk1 := parseParent.(*WASMDOMNode)
	parseNewN, parseOk2 := parseNewNode.(*WASMDOMNode)
	parseRefN, parseOk3 := parseReferenceNode.(*WASMDOMNode)
	if parseOk1 && parseOk2 && parseOk3 {
		parseRefParent := parseRefN.value.Get("parentNode")
		if parseRefParent.IsNull() || parseRefParent.IsUndefined() || !parseRefParent.Equal(parseParentN.value) {
			return
		}
		parseRefN.value.Call("before", parseNewN.value)
	}
}

func (parseA *WASMDOMAdapter) ReplaceChild(parseParent, parseNewNode, parseOldNode runtime.DOMNode) {
	parseParentN, parseOk1 := parseParent.(*WASMDOMNode)
	parseNewN, parseOk2 := parseNewNode.(*WASMDOMNode)
	parseOldN, parseOk3 := parseOldNode.(*WASMDOMNode)
	if parseOk1 && parseOk2 && parseOk3 {
		parseOldParent := parseOldN.value.Get("parentNode")
		if parseOldParent.IsNull() || parseOldParent.IsUndefined() || !parseOldParent.Equal(parseParentN.value) {
			return
		}
		parseOldN.value.Call("replaceWith", parseNewN.value)
	}
}

// ReplaceChildren replaces one parent child list in one DOM bridge call.
func (parseA *WASMDOMAdapter) ReplaceChildren(parseParent runtime.DOMNode, parseChildren []runtime.DOMNode) {
	parseParentNode, parseOk := parseParent.(*WASMDOMNode)
	if !parseOk {
		return
	}
	if len(parseChildren) == 0 {
		parseParentNode.value.Call("replaceChildren")
		return
	}
	getArgs := parseA.getBatchChildren()
	for _, parseChild := range parseChildren {
		parseChildNode, parseChildOk := parseChild.(*WASMDOMNode)
		if !parseChildOk {
			parseA.storeBatchChildren(getArgs)
			return
		}
		getArgs = append(getArgs, parseChildNode.value)
	}
	parseParentNode.value.Call("replaceChildren", getArgs...)
	parseA.storeBatchChildren(getArgs)
}

func (parseA *WASMDOMAdapter) QuerySelector(parseSelector string) interface{} {
	parseQuerySelector := parseA.getQuerySelector()
	if parseQuerySelector.IsNull() || parseQuerySelector.IsUndefined() {
		return nil
	}
	parseResult := parseQuerySelector.Invoke(parseSelector)
	if parseResult.IsNull() || parseResult.IsUndefined() {
		return nil
	}
	return &WASMDOMNode{value: parseResult}
}

func (parseA *WASMDOMAdapter) ResolveNode(parseValue interface{}) runtime.DOMNode {
	switch parseTyped := parseValue.(type) {
	case *WASMDOMNode:
		return parseTyped
	case runtime.DOMNode:
		return parseTyped
	case js.Value:
		if parseTyped.IsNull() || parseTyped.IsUndefined() {
			return nil
		}
		return &WASMDOMNode{value: parseTyped}
	default:
		return nil
	}
}

func (parseA *WASMDOMAdapter) QuerySelectorAll(parseSelector string) []runtime.DOMNode {
	parseQuerySelectorAll := parseA.getQuerySelectorAll()
	if parseQuerySelectorAll.IsNull() || parseQuerySelectorAll.IsUndefined() {
		return nil
	}
	parseNodeList := parseQuerySelectorAll.Invoke(parseSelector)
	parseLength := parseNodeList.Get("length").Int()

	parseNodes := make([]runtime.DOMNode, parseLength)
	for parseI := 0; parseI < parseLength; parseI++ {
		parseNodes[parseI] = &WASMDOMNode{value: parseNodeList.Index(parseI)}
	}
	return parseNodes
}

func (parseA *WASMDOMAdapter) GetElementById(parseId string) runtime.DOMNode {
	parseGetElementByID := parseA.getElementByIdMethod()
	if parseGetElementByID.IsNull() || parseGetElementByID.IsUndefined() {
		return nil
	}
	parseResult := parseGetElementByID.Invoke(parseId)
	if parseResult.IsNull() || parseResult.IsUndefined() {
		return nil
	}
	return &WASMDOMNode{value: parseResult}
}

func (parseA *WASMDOMAdapter) GetElementsByClassName(parseClassName string) []runtime.DOMNode {
	parseGetByClassName := parseA.getElementsByClassNameMethod()
	if parseGetByClassName.IsNull() || parseGetByClassName.IsUndefined() {
		return nil
	}
	parseHtmlCollection := parseGetByClassName.Invoke(parseClassName)
	parseLength := parseHtmlCollection.Get("length").Int()

	parseNodes := make([]runtime.DOMNode, parseLength)
	for parseI := 0; parseI < parseLength; parseI++ {
		parseNodes[parseI] = &WASMDOMNode{value: parseHtmlCollection.Index(parseI)}
	}
	return parseNodes
}

func (parseA *WASMDOMAdapter) GetElementsByTagName(parseTagName string) []runtime.DOMNode {
	parseGetByTagName := parseA.getElementsByTagNameMethod()
	if parseGetByTagName.IsNull() || parseGetByTagName.IsUndefined() {
		return nil
	}
	parseHtmlCollection := parseGetByTagName.Invoke(parseTagName)
	parseLength := parseHtmlCollection.Get("length").Int()

	parseNodes := make([]runtime.DOMNode, parseLength)
	for parseI := 0; parseI < parseLength; parseI++ {
		parseNodes[parseI] = &WASMDOMNode{value: parseHtmlCollection.Index(parseI)}
	}
	return parseNodes
}

func (parseA *WASMDOMAdapter) GetInnerHTML(parseNode runtime.DOMNode) string {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		return parseWasmNode.value.Get("innerHTML").String()
	}
	return ""
}

func (parseA *WASMDOMAdapter) SetTextContent(parseNode runtime.DOMNode, parseText string) {
	// IsNull is nil-receiver-safe and also covers a null/undefined js.Value: a
	// typed-nil (*WASMDOMNode) passes the type assertion but nil-derefs on .value,
	// and .Set on a null/undefined value panics with "not an object". Both crash
	// the whole app from the render/commit path, so guard before touching JS.
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk && !parseWasmNode.IsNull() {
		parseWasmNode.value.Set("textContent", parseText)
	}
}

func (parseA *WASMDOMAdapter) GetTextContent(parseNode runtime.DOMNode) string {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk && !parseWasmNode.IsNull() {
		return parseWasmNode.value.Get("textContent").String()
	}
	return ""
}

func (parseA *WASMDOMAdapter) AddClass(parseNode runtime.DOMNode, parseClassName string) {
	if parseWasmNode, parseOk := liveWasmNode(parseNode); parseOk {
		parseClassList := parseWasmNode.value.Get("classList")
		parseClassList.Call("add", parseClassName)
	}
}

func (parseA *WASMDOMAdapter) RemoveClass(parseNode runtime.DOMNode, parseClassName string) {
	if parseWasmNode, parseOk := liveWasmNode(parseNode); parseOk {
		parseClassList := parseWasmNode.value.Get("classList")
		parseClassList.Call("remove", parseClassName)
	}
}

func (parseA *WASMDOMAdapter) ToggleClass(parseNode runtime.DOMNode, parseClassName string) {
	if parseWasmNode, parseOk := liveWasmNode(parseNode); parseOk {
		parseClassList := parseWasmNode.value.Get("classList")
		parseClassList.Call("toggle", parseClassName)
	}
}

func (parseA *WASMDOMAdapter) GetParent(parseNode runtime.DOMNode) runtime.DOMNode {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		parseParent := parseWasmNode.value.Get("parentNode")
		if !parseParent.IsNull() && !parseParent.IsUndefined() {
			return &WASMDOMNode{value: parseParent}
		}
	}
	return nil
}

func (parseA *WASMDOMAdapter) GetChildren(parseNode runtime.DOMNode) []runtime.DOMNode {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		parseChildren := parseWasmNode.value.Get("children")
		parseLength := parseChildren.Get("length").Int()

		parseNodes := make([]runtime.DOMNode, parseLength)
		for parseI := 0; parseI < parseLength; parseI++ {
			parseNodes[parseI] = &WASMDOMNode{value: parseChildren.Index(parseI)}
		}
		return parseNodes
	}
	return nil
}

func (parseA *WASMDOMAdapter) GetFirstChild(parseNode runtime.DOMNode) runtime.DOMNode {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		parseFirstChild := parseWasmNode.value.Get("firstChild")
		if !parseFirstChild.IsNull() && !parseFirstChild.IsUndefined() {
			return &WASMDOMNode{value: parseFirstChild}
		}
	}
	return nil
}

func (parseA *WASMDOMAdapter) GetNextSibling(parseNode runtime.DOMNode) runtime.DOMNode {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		parseNextSibling := parseWasmNode.value.Get("nextSibling")
		if !parseNextSibling.IsNull() && !parseNextSibling.IsUndefined() {
			return &WASMDOMNode{value: parseNextSibling}
		}
	}
	return nil
}

func (parseA *WASMDOMAdapter) SetStyle(parseNode runtime.DOMNode, parseProperty, parseValue string) {
	if parseWasmNode, parseOk := liveWasmNode(parseNode); parseOk {
		parseStyle := parseWasmNode.value.Get("style")
		parseStyle.Set(parseProperty, parseValue)
	}
}

func (parseA *WASMDOMAdapter) SetStyles(parseNode runtime.DOMNode, parseStyles map[string]string) {
	if parseWasmNode, parseOk := liveWasmNode(parseNode); parseOk {
		parseStyle := parseWasmNode.value.Get("style")
		// Batch style updates by caching style object
		for parseProperty, parseValue := range parseStyles {
			parseStyle.Set(parseProperty, parseValue)
		}
	}
}

// BeginBatch starts batching DOM operations for a parent node
func (parseA *WASMDOMAdapter) BeginBatch(parseParent runtime.DOMNode) {
	if parseParentNode, parseOk := parseParent.(*WASMDOMNode); parseOk {
		parseA.batchStack = append(parseA.batchStack, wasmBatchState{parent: parseParentNode, children: parseA.getBatchChildren()})
	}
}

// EndBatch commits all batched operations
func (parseA *WASMDOMAdapter) EndBatch() {
	parseDepth := len(parseA.batchStack)
	if parseDepth == 0 {
		return
	}

	parseState := parseA.batchStack[parseDepth-1]
	parseA.batchStack = parseA.batchStack[:parseDepth-1]
	if parseState.parent != nil && len(parseState.children) > 0 {
		parseA.appendDOMChildren(parseState.parent.value, parseState.children...)
	}
	parseA.storeBatchChildren(parseState.children)
}

// appendDOMChildren appends one or more child nodes, falling back to appendChild when append is unavailable.
// The append capability is probed once and cached on the adapter so subsequent calls skip the property lookup.
func (parseA *WASMDOMAdapter) appendDOMChildren(parseParent js.Value, parseChildren ...interface{}) {
	if parseParent.IsNull() || parseParent.IsUndefined() || len(parseChildren) == 0 {
		return
	}
	if !parseA.appendChecked {
		parseAppend := parseParent.Get("append")
		parseA.appendFast = !parseAppend.IsNull() && !parseAppend.IsUndefined() && parseAppend.Type() == js.TypeFunction
		parseA.appendChecked = true
	}
	if parseA.appendFast {
		parseParent.Call("append", parseChildren...)
		return
	}
	parseAppendChild := parseParent.Get("appendChild")
	if parseAppendChild.IsNull() || parseAppendChild.IsUndefined() || parseAppendChild.Type() != js.TypeFunction {
		return
	}
	for _, parseChild := range parseChildren {
		if parseChild == nil {
			continue
		}
		parseParent.Call("appendChild", parseChild)
	}
}

// getBatchChildren returns one reusable DOM argument buffer for append-style bridge calls.
func (parseA *WASMDOMAdapter) getBatchChildren() []interface{} {
	if parseA == nil {
		return make([]interface{}, 0, 8)
	}
	getPoolIndex := len(parseA.storeBatchChildrenPool) - 1
	if getPoolIndex < 0 {
		return make([]interface{}, 0, 8)
	}
	getChildren := parseA.storeBatchChildrenPool[getPoolIndex]
	parseA.storeBatchChildrenPool = parseA.storeBatchChildrenPool[:getPoolIndex]
	return getChildren[:0]
}

// storeBatchChildren stores one DOM argument buffer for later append-style bridge-call reuse.
func (parseA *WASMDOMAdapter) storeBatchChildren(parseChildren []interface{}) {
	if parseA == nil || parseChildren == nil {
		return
	}
	for getIndex := range parseChildren {
		parseChildren[getIndex] = nil
	}
	parseA.storeBatchChildrenPool = append(parseA.storeBatchChildrenPool, parseChildren[:0])
}

// BatchSetAttributes sets multiple attributes without paying one extra Go callback hop.
func (parseA *WASMDOMAdapter) BatchSetAttributes(parseNode runtime.DOMNode, parseAttrs map[string]string) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		if len(parseAttrs) == 0 {
			return
		}
		for parseName, parseValue := range parseAttrs {
			parseValue = runtime.SanitizeURLAttributeValue(parseName, parseValue)
			parseWasmNode.value.Call("setAttribute", parseName, parseValue)
		}
	}
}

// buildHostAttrMap converts one compact host-attr slice into one string map for shared batching code.
func buildHostAttrMap(parseAttrs []runtime.HostAttr) map[string]string {
	if len(parseAttrs) == 0 {
		return nil
	}
	getAttrs := make(map[string]string, len(parseAttrs))
	for _, parseAttr := range parseAttrs {
		getAttrs[parseAttr.Name] = parseAttr.Value
	}
	return getAttrs
}

// buildHostElementHTML formats one compact host element as one HTML string when the tag and attrs are safe.
func buildHostElementHTML(parseTag string, parseAttrs []runtime.HostAttr, parseText string) (string, bool) {
	var parseBuilder strings.Builder
	if !buildHostElementHTMLInto(&parseBuilder, parseTag, parseAttrs, parseText) {
		return "", false
	}
	return parseBuilder.String(), true
}

// buildHostElementHTMLInto appends one compact host element HTML fragment into parseBuilder when the tag and attrs are safe.
func buildHostElementHTMLInto(parseBuilder *strings.Builder, parseTag string, parseAttrs []runtime.HostAttr, parseText string) bool {
	if parseBuilder == nil || !hasSafeHostHTMLTag(parseTag) {
		return false
	}
	if parseText != "" && hasVoidHTMLElementTag(parseTag) {
		return false
	}
	parseBuilder.WriteByte('<')
	parseBuilder.WriteString(parseTag)
	for _, parseAttr := range parseAttrs {
		if !hasSafeHostHTMLAttrName(parseAttr.Name) {
			return false
		}
		parseBuilder.WriteByte(' ')
		parseBuilder.WriteString(parseAttr.Name)
		parseBuilder.WriteString(`="`)
		// Same javascript:/vbscript: URL blocking as SetAttribute — the template
		// fast path must not be the one lane where a hostile href survives.
		parseBuilder.WriteString(html.EscapeString(runtime.SanitizeURLAttributeValue(parseAttr.Name, parseAttr.Value)))
		parseBuilder.WriteByte('"')
	}
	parseBuilder.WriteByte('>')
	if parseText != "" {
		parseBuilder.WriteString(html.EscapeString(parseText))
	}
	if !hasVoidHTMLElementTag(parseTag) {
		parseBuilder.WriteString("</")
		parseBuilder.WriteString(parseTag)
		parseBuilder.WriteByte('>')
	}
	return true
}

// hasSafeHostHTMLTag reports whether one tag can safely round-trip through the conservative template fast path.
func hasSafeHostHTMLTag(parseTag string) bool {
	if strings.TrimSpace(parseTag) == "" {
		return false
	}
	for _, parseRune := range parseTag {
		switch {
		case parseRune >= 'a' && parseRune <= 'z':
		case parseRune >= 'A' && parseRune <= 'Z':
		case parseRune >= '0' && parseRune <= '9':
		case parseRune == '-':
		default:
			return false
		}
	}
	switch strings.ToLower(parseTag) {
	case "svg", "math":
		return false
	default:
		return true
	}
}

// hasSafeHostHTMLAttrName reports whether one attr name can safely round-trip through the conservative template fast path.
func hasSafeHostHTMLAttrName(parseName string) bool {
	if strings.TrimSpace(parseName) == "" {
		return false
	}
	for _, parseRune := range parseName {
		switch {
		case parseRune >= 'a' && parseRune <= 'z':
		case parseRune >= 'A' && parseRune <= 'Z':
		case parseRune >= '0' && parseRune <= '9':
		case parseRune == '-', parseRune == '_', parseRune == ':':
		default:
			return false
		}
	}
	return true
}

// hasVoidHTMLElementTag reports whether one tag omits its closing tag in HTML parsing mode.
func hasVoidHTMLElementTag(parseTag string) bool {
	switch strings.ToLower(strings.TrimSpace(parseTag)) {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

func (parseA *WASMDOMAdapter) WrapFunction(parseFn interface{}) interface{} {
	switch parseF := parseFn.(type) {
	case func():
		return js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			defer runtime.RecoverContainedPanic("dom", "wrapped callback")
			parseF()
			runtime.FlushGlobalDiscreteWork()
			return nil
		})
	case func(string):
		return js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
			defer runtime.RecoverContainedPanic("dom", "wrapped callback")
			defer runtime.FlushGlobalDiscreteWork()
			if len(parseArgs2) == 0 {
				parseF("")
				return nil
			}
			parseTarget := parseArgs2[0].Get("target")
			if parseTarget.IsNull() || parseTarget.IsUndefined() {
				parseF("")
				return nil
			}
			parseValue := parseTarget.Get("value")
			if parseValue.IsNull() || parseValue.IsUndefined() {
				parseF("")
				return nil
			}
			parseF(parseValue.String())
			return nil
		})
	case func(js.Value):
		return js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			defer runtime.RecoverContainedPanic("dom", "wrapped callback")
			if len(parseArgs3) > 0 {
				parseF(parseArgs3[0])
				runtime.FlushGlobalDiscreteWork()
			}
			return nil
		})
	case func() error:
		return js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			defer runtime.RecoverContainedPanic("dom", "wrapped callback")
			parseF()
			runtime.FlushGlobalDiscreteWork()
			return nil
		})
	case func(js.Value) error:
		return js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			defer runtime.RecoverContainedPanic("dom", "wrapped callback")
			if len(parseArgs5) > 0 {
				parseF(parseArgs5[0])
				runtime.FlushGlobalDiscreteWork()
			}
			return nil
		})
	case func(runtime.GoEvent):
		return js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
			defer runtime.RecoverContainedPanic("dom", "wrapped callback")
			if len(parseArgs6) > 0 {
				parseF(runtime.NewGoEvent(parseArgs6[0]))
				runtime.FlushGlobalDiscreteWork()
			}
			return nil
		})
	case func(runtime.GoEvent) error:
		return js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} {
			defer runtime.RecoverContainedPanic("dom", "wrapped callback")
			if len(parseArgs7) > 0 {
				parseF(runtime.NewGoEvent(parseArgs7[0]))
				runtime.FlushGlobalDiscreteWork()
			}
			return nil
		})
	default:
		return js.FuncOf(func(parseThis8 js.Value, parseArgs8 []js.Value) interface{} {
			return nil
		})
	}
}

// WASMEventAdapter implements runtime.EventAdapter for browser/WASM.
type WASMEventAdapter struct {
	addEventListener    js.Value
	removeEventListener js.Value
	isMethodsBound      bool
}

var _ runtime.EventAdapter = (*WASMEventAdapter)(nil)

// NewWASMEventAdapter creates an event adapter backed by browser DOM listeners.
func NewWASMEventAdapter() *WASMEventAdapter {
	return &WASMEventAdapter{}
}

func (parseA *WASMEventAdapter) ensureEventMethods() bool {
	if parseA == nil {
		return false
	}
	if parseA.isMethodsBound &&
		!parseA.addEventListener.IsNull() && !parseA.addEventListener.IsUndefined() &&
		!parseA.removeEventListener.IsNull() && !parseA.removeEventListener.IsUndefined() {
		return true
	}
	parseElement := js.Global().Get("Element")
	if parseElement.IsNull() || parseElement.IsUndefined() {
		return false
	}
	parseElemProto := parseElement.Get("prototype")
	if parseElemProto.IsNull() || parseElemProto.IsUndefined() {
		return false
	}
	parseAddEventListener := parseElemProto.Get("addEventListener")
	parseRemoveEventListener := parseElemProto.Get("removeEventListener")
	if parseAddEventListener.IsNull() || parseAddEventListener.IsUndefined() || parseAddEventListener.Type() != js.TypeFunction {
		return false
	}
	if parseRemoveEventListener.IsNull() || parseRemoveEventListener.IsUndefined() || parseRemoveEventListener.Type() != js.TypeFunction {
		return false
	}
	parseA.addEventListener = parseAddEventListener
	parseA.removeEventListener = parseRemoveEventListener
	parseA.isMethodsBound = true
	return true
}

// wasmEventHandler wraps a js.Func for event handling.
type wasmEventHandler struct {
	fn     js.Func
	goFunc func(runtime.Event)
}

var _ runtime.EventHandler = (*wasmEventHandler)(nil)

func (parseH *wasmEventHandler) Release() {
	parseH.fn.Release()
}

func (parseA *WASMEventAdapter) CreateEventHandler(parseFn func(runtime.Event)) runtime.EventHandler {
	// Create a wasmEvent wrapper
	parseJsFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer runtime.RecoverContainedPanic("dom", "event handler bridge")
		if len(parseArgs) > 0 {
			parseEvent := &wasmEvent{value: parseArgs[0]}
			parseFn(parseEvent)
		}
		return nil
	})

	return &wasmEventHandler{
		fn:     parseJsFn,
		goFunc: parseFn,
	}
}

func (parseA *WASMEventAdapter) ReleaseEventHandler(parseHandler runtime.EventHandler) {
	parseHandler.Release()
}

func (parseA *WASMEventAdapter) AddEventListener(parseNode runtime.DOMNode, parseEventType string, parseHandler runtime.EventHandler) {
	if !parseA.ensureEventMethods() {
		return
	}
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		if parseWasmHandler, parseOk2 := parseHandler.(*wasmEventHandler); parseOk2 {
			parseA.addEventListener.Call("call", parseWasmNode.value, parseEventType, parseWasmHandler.fn)
		}
	}
}

func (parseA *WASMEventAdapter) RemoveEventListener(parseNode runtime.DOMNode, parseEventType string, parseHandler runtime.EventHandler) {
	if !parseA.ensureEventMethods() {
		return
	}
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		if parseWasmHandler, parseOk2 := parseHandler.(*wasmEventHandler); parseOk2 {
			parseA.removeEventListener.Call("call", parseWasmNode.value, parseEventType, parseWasmHandler.fn)
		}
	}
}

// wasmEvent implements runtime.Event for browser/WASM.
type wasmEvent struct {
	value js.Value
}

var _ runtime.Event = (*wasmEvent)(nil)

func (parseE *wasmEvent) PreventDefault() {
	parseE.value.Call("preventDefault")
}

func (parseE *wasmEvent) StopPropagation() {
	parseE.value.Call("stopPropagation")
}

func (parseE *wasmEvent) GetValue() string {
	parseTarget := parseE.value.Get("target")
	if !parseTarget.IsNull() && !parseTarget.IsUndefined() {
		parseValue := parseTarget.Get("value")
		if !parseValue.IsNull() && !parseValue.IsUndefined() {
			return parseValue.String()
		}
	}
	return ""
}

func (parseE *wasmEvent) GetTarget() runtime.DOMNode {
	parseTarget := parseE.value.Get("target")
	if !parseTarget.IsNull() && !parseTarget.IsUndefined() {
		return &WASMDOMNode{value: parseTarget}
	}
	return nil
}

func (parseE *wasmEvent) GetKeyCode() int {
	parseKeyCode := parseE.value.Get("keyCode")
	if !parseKeyCode.IsNull() && !parseKeyCode.IsUndefined() {
		return parseKeyCode.Int()
	}
	return 0
}

func (parseE *wasmEvent) GetKey() string {
	parseKey := parseE.value.Get("key")
	if !parseKey.IsNull() && !parseKey.IsUndefined() {
		return parseKey.String()
	}
	return ""
}

func (parseE *wasmEvent) IsChecked() bool {
	parseTarget := parseE.value.Get("target")
	if !parseTarget.IsNull() && !parseTarget.IsUndefined() {
		parseChecked := parseTarget.Get("checked")
		if !parseChecked.IsNull() && !parseChecked.IsUndefined() {
			return parseChecked.Bool()
		}
	}
	return false
}

// WASMScheduler implements runtime.Scheduler for browser/WASM.
type WASMScheduler struct {
	window js.Value
}

var _ runtime.Scheduler = (*WASMScheduler)(nil)

// NewWASMScheduler creates a scheduler backed by browser idle callbacks and timeouts.
func NewWASMScheduler() *WASMScheduler {
	return &WASMScheduler{
		window: js.Global(),
	}
}

func (parseS *WASMScheduler) RequestIdleCallback(parseCallback func(runtime.Deadline)) {
	// Wrap the callback - release after execution
	var parseJsFn js.Func
	parseJsFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer parseJsFn.Release()
		defer runtime.RecoverContainedPanic("scheduler", "idle callback")
		parseDeadline := &wasmDeadline{}
		if len(parseArgs) > 0 {
			parseDeadline.value = parseArgs[0]
		}
		parseCallback(parseDeadline)
		return nil
	})

	// Check if requestIdleCallback is available
	if parseS.window.Get("requestIdleCallback").Truthy() {
		parseS.window.Call("requestIdleCallback", parseJsFn)
	} else {
		// Fallback to setTimeout
		parseS.window.Call("setTimeout", parseJsFn, 0)
	}
}

func (parseS *WASMScheduler) SetTimeout(parseCallback func(), parseDelay int) {
	// Always use setTimeout (even for delay 0) to allow goroutines to run
	// This ensures that goroutines calling setState can enqueue updates before workLoop processes them
	var parseJsFn js.Func
	parseJsFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer parseJsFn.Release()
		defer runtime.RecoverContainedPanic("scheduler", "timeout callback")
		parseCallback()
		return nil
	})

	parseS.window.Call("setTimeout", parseJsFn, parseDelay)
}

func (parseS *WASMScheduler) CancelIdleCallback(parseId interface{}) {
	if parseS.window.Get("cancelIdleCallback").Truthy() {
		parseS.window.Call("cancelIdleCallback", parseId)
	}
}

// wasmDeadline implements runtime.Deadline for browser/WASM.
type wasmDeadline struct {
	value js.Value
}

var _ runtime.Deadline = (*wasmDeadline)(nil)

func (parseD *wasmDeadline) TimeRemaining() float64 {
	if parseD.value.IsUndefined() || parseD.value.IsNull() {
		return 50.0 // Default to 50ms
	}
	if parseD.value.Get("timeRemaining").Truthy() {
		return parseD.value.Call("timeRemaining").Float()
	}
	return 50.0
}

func (parseD *wasmDeadline) DidTimeout() bool {
	if parseD.value.IsUndefined() || parseD.value.IsNull() {
		return false
	}
	return parseD.value.Get("didTimeout").Bool()
}

// WASMBrowserState implements runtime.BrowserState for browser/WASM.
type WASMBrowserState struct {
	window             js.Value
	popStateHandle     js.Func // retained so it can be released and is not GC'd
	popStateRegistered bool    // true once a popstate listener has been set
}

var _ runtime.BrowserState = (*WASMBrowserState)(nil)

// NewWASMBrowserState creates a browser state adapter backed by window and history.
func NewWASMBrowserState() *WASMBrowserState {
	return &WASMBrowserState{
		window: js.Global(),
	}
}

func (parseB *WASMBrowserState) PushState(parseState interface{}, parseTitle, parseUrl string) {
	parseHistory := parseB.window.Get("history")
	parseHistory.Call("pushState", parseState, parseTitle, parseUrl)
}

func (parseB *WASMBrowserState) ReplaceState(parseState interface{}, parseTitle, parseUrl string) {
	parseHistory := parseB.window.Get("history")
	parseHistory.Call("replaceState", parseState, parseTitle, parseUrl)
}

func (parseB *WASMBrowserState) GetCurrentPath() string {
	parseLocation := parseB.window.Get("location")
	return parseLocation.Get("pathname").String()
}

func (parseB *WASMBrowserState) OnPopState(parseCallback func(path string)) {
	// Release any previously registered popstate listener before replacing it.
	if parseB.popStateRegistered {
		parseB.window.Call("removeEventListener", "popstate", parseB.popStateHandle)
		parseB.popStateHandle.Release()
		parseB.popStateRegistered = false
	}
	parseB.popStateHandle = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer runtime.RecoverContainedPanic("browser", "popstate listener")
		parseLocation := parseB.window.Get("location")
		parsePath := parseLocation.Get("pathname").String()
		parseCallback(parsePath)
		return nil
	})
	parseB.popStateRegistered = true
	parseB.window.Call("addEventListener", "popstate", parseB.popStateHandle)
}

func (parseB *WASMBrowserState) SetItem(parseKey, parseValue string) (getErr error) {
	// localStorage.setItem throws (quota exceeded, private-mode restrictions);
	// syscall/js surfaces that as a Go panic — convert it to the error this
	// method already promises instead of crashing the app on a full store.
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			getErr = fmt.Errorf("jsdom: localStorage setItem %q failed: %v", parseKey, parseRecovered)
		}
	}()
	parseStorage := parseB.window.Get("localStorage")
	if !parseStorage.IsNull() && !parseStorage.IsUndefined() {
		parseStorage.Call("setItem", parseKey, parseValue)
	}
	return nil
}

func (parseB *WASMBrowserState) GetItem(parseKey string) (getValue string, getOK bool) {
	// getItem can throw under blocked-storage policies; treat that as absent.
	defer func() {
		if recover() != nil {
			getValue, getOK = "", false
		}
	}()
	parseStorage := parseB.window.Get("localStorage")
	if parseStorage.IsNull() || parseStorage.IsUndefined() {
		return "", false
	}

	parseValue := parseStorage.Call("getItem", parseKey)
	if parseValue.IsNull() {
		return "", false
	}
	return parseValue.String(), true
}

func (parseB *WASMBrowserState) RemoveItem(parseKey string) {
	// removeItem can throw under blocked-storage policies; removal is best-effort.
	defer func() { _ = recover() }()
	parseStorage := parseB.window.Get("localStorage")
	if !parseStorage.IsNull() && !parseStorage.IsUndefined() {
		parseStorage.Call("removeItem", parseKey)
	}
}

func (parseB *WASMBrowserState) GetHash() string {
	parseLocation := parseB.window.Get("location")
	return parseLocation.Get("hash").String()
}

func (parseB *WASMBrowserState) SetHash(parseHash string) {
	parseLocation := parseB.window.Get("location")
	parseLocation.Set("hash", parseHash)
}

func (parseB *WASMBrowserState) Reload() {
	parseLocation := parseB.window.Get("location")
	parseLocation.Call("reload")
}
