//go:build js && wasm
// +build js,wasm

package jsdom

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// WASMDOMNode wraps a js.Value representing a DOM node.
type WASMDOMNode struct {
	value js.Value
}

var _ runtime.DOMNode = (*WASMDOMNode)(nil)

// NewWASMDOMNode wraps a raw js.Value as a runtime.DOMNode.
func NewWASMDOMNode(parseValue js.Value) runtime.DOMNode {
	return &WASMDOMNode{value: parseValue}
}

func (parseN *WASMDOMNode) IsNull() bool {
	return parseN.value.IsNull() || parseN.value.IsUndefined()
}

func (parseN *WASMDOMNode) Equals(parseOther runtime.DOMNode) bool {
	if parseOtherNode, parseOk := parseOther.(*WASMDOMNode); parseOk {
		return parseN.value.Equal(parseOtherNode.value)
	}
	return false
}

func (parseN *WASMDOMNode) Value() js.Value {
	return parseN.value
}

// WASMDOMAdapter implements runtime.DOMAdapter for browser/WASM.
type WASMDOMAdapter struct {
	document         js.Value
	createElement    js.Value
	createTextNode   js.Value
	querySelector    js.Value
	querySelectorAll js.Value
	getElementByID   js.Value
	getByClassName   js.Value
	getByTagName     js.Value
	// Cached methods for performance
	appendChild         js.Value
	removeChild         js.Value
	setAttribute        js.Value
	removeAttribute     js.Value
	insertBefore        js.Value
	replaceChild        js.Value
	addEventListener    js.Value
	removeEventListener js.Value
	createFragment      js.Value
	batchSetAttributes  js.Func
	// Batch operation support
	fragmentPool []js.Value
	batchStack   []wasmBatchState
}

type wasmBatchState struct {
	parent   *WASMDOMNode
	fragment js.Value
}

var _ runtime.DOMAdapter = (*WASMDOMAdapter)(nil)

// NewWASMDOMAdapter creates a DOM adapter backed by the browser document.
func NewWASMDOMAdapter() *WASMDOMAdapter {
	parseDoc := js.Global().Get("document")
	// Pre-cache DOM prototype methods
	parseElemProto := js.Global().Get("Element").Get("prototype")
	parseBatchSetAttributes := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseNode := parseArgs[0]
		parseAttrs := parseArgs[1]
		parseKeys := js.Global().Get("Object").Call("keys", parseAttrs)
		parseLength := parseKeys.Get("length").Int()
		for parseI := 0; parseI < parseLength; parseI++ {
			parseKey := parseKeys.Index(parseI).String()
			parseNode.Call("setAttribute", parseKey, parseAttrs.Get(parseKey).String())
		}
		return nil
	})
	return &WASMDOMAdapter{
		document: parseDoc,
		// Bind methods to document to ensure correct 'this' context when Invoked
		createElement:    parseDoc.Get("createElement").Call("bind", parseDoc),
		createTextNode:   parseDoc.Get("createTextNode").Call("bind", parseDoc),
		querySelector:    parseDoc.Get("querySelector").Call("bind", parseDoc),
		querySelectorAll: parseDoc.Get("querySelectorAll").Call("bind", parseDoc),
		getElementByID:   parseDoc.Get("getElementById").Call("bind", parseDoc),
		getByClassName:   parseDoc.Get("getElementsByClassName").Call("bind", parseDoc),
		getByTagName:     parseDoc.Get("getElementsByTagName").Call("bind", parseDoc),
		createFragment:   parseDoc.Get("createDocumentFragment").Call("bind", parseDoc),
		// Cache element methods (not bound, will use Call)
		appendChild:         parseElemProto.Get("appendChild"),
		removeChild:         parseElemProto.Get("removeChild"),
		setAttribute:        parseElemProto.Get("setAttribute"),
		removeAttribute:     parseElemProto.Get("removeAttribute"),
		insertBefore:        parseElemProto.Get("insertBefore"),
		replaceChild:        parseElemProto.Get("replaceChild"),
		addEventListener:    parseElemProto.Get("addEventListener"),
		removeEventListener: parseElemProto.Get("removeEventListener"),
		batchSetAttributes:  parseBatchSetAttributes,
	}
}

func (parseA *WASMDOMAdapter) CreateElement(parseTag string) runtime.DOMNode {
	// Check if document is available
	if parseA.document.IsNull() || parseA.document.IsUndefined() {
		// Document not available - return null node
		return &WASMDOMNode{value: js.Null()}
	}

	// Use Invoke on the cached function instead of Call on the document
	// This saves a property lookup on every call
	parseElem := parseA.createElement.Invoke(parseTag)
	if parseElem.IsNull() || parseElem.IsUndefined() {
		// This shouldn't happen, but handle it gracefully
		return &WASMDOMNode{value: js.Null()}
	}
	return &WASMDOMNode{value: parseElem}
}

func (parseA *WASMDOMAdapter) CreateTextNode(parseText string) runtime.DOMNode {
	// Check if document is available
	if parseA.document.IsNull() || parseA.document.IsUndefined() {
		return &WASMDOMNode{value: js.Null()}
	}

	// Use Invoke on the cached function
	parseTextNode := parseA.createTextNode.Invoke(parseText)
	if parseTextNode.IsNull() || parseTextNode.IsUndefined() {
		return &WASMDOMNode{value: js.Null()}
	}
	return &WASMDOMNode{value: parseTextNode}
}

func (parseA *WASMDOMAdapter) SetAttribute(parseNode runtime.DOMNode, parseName, parseValue string) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		// Use cached method for better performance
		parseA.setAttribute.Call("call", parseWasmNode.value, parseName, parseValue)
	}
}

func (parseA *WASMDOMAdapter) RemoveAttribute(parseNode runtime.DOMNode, parseName string) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		parseA.removeAttribute.Call("call", parseWasmNode.value, parseName)
	}
}

func (parseA *WASMDOMAdapter) SetProperty(parseNode runtime.DOMNode, parseName string, parseValue interface{}) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		// Direct property set (fastest path)
		parseWasmNode.value.Set(parseName, parseValue)
	}
}

func (parseA *WASMDOMAdapter) GetProperty(parseNode runtime.DOMNode, parseName string) interface{} {
	if parseNode == nil || parseNode.IsNull() {
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

	// If in batch mode for this specific parent, append to the top-most fragment.
	if parseDepth := len(parseA.batchStack); parseDepth > 0 {
		parseState := parseA.batchStack[parseDepth-1]
		if parseState.parent == parseParentNode {
			parseA.appendChild.Call("call", parseState.fragment, parseChildNode.value)
			return
		}
	}

	// Use cached method via Call (faster than method lookup each time)
	parseA.appendChild.Call("call", parseParentNode.value, parseChildNode.value)
}

func (parseA *WASMDOMAdapter) RemoveChild(parseParent, parseChild runtime.DOMNode) {
	parseParentNode, parseOk1 := parseParent.(*WASMDOMNode)
	parseChildNode, parseOk2 := parseChild.(*WASMDOMNode)
	if !parseOk1 || !parseOk2 {
		return
	}
	// Use cached method
	parseA.removeChild.Call("call", parseParentNode.value, parseChildNode.value)
}

func (parseA *WASMDOMAdapter) InsertBefore(parseParent, parseNewNode, parseReferenceNode runtime.DOMNode) {
	parseParentN, parseOk1 := parseParent.(*WASMDOMNode)
	parseNewN, parseOk2 := parseNewNode.(*WASMDOMNode)
	parseRefN, parseOk3 := parseReferenceNode.(*WASMDOMNode)
	if parseOk1 && parseOk2 && parseOk3 {
		parseA.insertBefore.Call("call", parseParentN.value, parseNewN.value, parseRefN.value)
	}
}

func (parseA *WASMDOMAdapter) ReplaceChild(parseParent, parseNewNode, parseOldNode runtime.DOMNode) {
	parseParentN, parseOk1 := parseParent.(*WASMDOMNode)
	parseNewN, parseOk2 := parseNewNode.(*WASMDOMNode)
	parseOldN, parseOk3 := parseOldNode.(*WASMDOMNode)
	if parseOk1 && parseOk2 && parseOk3 {
		parseA.replaceChild.Call("call", parseParentN.value, parseNewN.value, parseOldN.value)
	}
}

func (parseA *WASMDOMAdapter) QuerySelector(parseSelector string) interface{} {
	parseResult := parseA.querySelector.Invoke(parseSelector)
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
	parseNodeList := parseA.querySelectorAll.Invoke(parseSelector)
	parseLength := parseNodeList.Get("length").Int()

	parseNodes := make([]runtime.DOMNode, parseLength)
	for parseI := 0; parseI < parseLength; parseI++ {
		parseNodes[parseI] = &WASMDOMNode{value: parseNodeList.Index(parseI)}
	}
	return parseNodes
}

func (parseA *WASMDOMAdapter) GetElementById(parseId string) runtime.DOMNode {
	parseResult := parseA.getElementByID.Invoke(parseId)
	if parseResult.IsNull() || parseResult.IsUndefined() {
		return nil
	}
	return &WASMDOMNode{value: parseResult}
}

func (parseA *WASMDOMAdapter) GetElementsByClassName(parseClassName string) []runtime.DOMNode {
	parseHtmlCollection := parseA.getByClassName.Invoke(parseClassName)
	parseLength := parseHtmlCollection.Get("length").Int()

	parseNodes := make([]runtime.DOMNode, parseLength)
	for parseI := 0; parseI < parseLength; parseI++ {
		parseNodes[parseI] = &WASMDOMNode{value: parseHtmlCollection.Index(parseI)}
	}
	return parseNodes
}

func (parseA *WASMDOMAdapter) GetElementsByTagName(parseTagName string) []runtime.DOMNode {
	parseHtmlCollection := parseA.getByTagName.Invoke(parseTagName)
	parseLength := parseHtmlCollection.Get("length").Int()

	parseNodes := make([]runtime.DOMNode, parseLength)
	for parseI := 0; parseI < parseLength; parseI++ {
		parseNodes[parseI] = &WASMDOMNode{value: parseHtmlCollection.Index(parseI)}
	}
	return parseNodes
}

func (parseA *WASMDOMAdapter) SetInnerHTML(parseNode runtime.DOMNode, parseHtml string) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		parseWasmNode.value.Set("innerHTML", parseHtml)
	}
}

func (parseA *WASMDOMAdapter) GetInnerHTML(parseNode runtime.DOMNode) string {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		return parseWasmNode.value.Get("innerHTML").String()
	}
	return ""
}

func (parseA *WASMDOMAdapter) SetTextContent(parseNode runtime.DOMNode, parseText string) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		parseWasmNode.value.Set("textContent", parseText)
	}
}

func (parseA *WASMDOMAdapter) GetTextContent(parseNode runtime.DOMNode) string {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		return parseWasmNode.value.Get("textContent").String()
	}
	return ""
}

func (parseA *WASMDOMAdapter) AddClass(parseNode runtime.DOMNode, parseClassName string) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		parseClassList := parseWasmNode.value.Get("classList")
		parseClassList.Call("add", parseClassName)
	}
}

func (parseA *WASMDOMAdapter) RemoveClass(parseNode runtime.DOMNode, parseClassName string) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		parseClassList := parseWasmNode.value.Get("classList")
		parseClassList.Call("remove", parseClassName)
	}
}

func (parseA *WASMDOMAdapter) ToggleClass(parseNode runtime.DOMNode, parseClassName string) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
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
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		parseStyle := parseWasmNode.value.Get("style")
		parseStyle.Set(parseProperty, parseValue)
	}
}

func (parseA *WASMDOMAdapter) SetStyles(parseNode runtime.DOMNode, parseStyles map[string]string) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
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
		parseDepth := len(parseA.batchStack)
		var parseFragment js.Value
		if parseDepth < len(parseA.fragmentPool) {
			parseFragment = parseA.fragmentPool[parseDepth]
		} else {
			parseFragment = parseA.createFragment.Invoke()
			parseA.fragmentPool = append(parseA.fragmentPool, parseFragment)
		}
		parseA.batchStack = append(parseA.batchStack, wasmBatchState{parent: parseParentNode, fragment: parseFragment})
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
	if parseState.parent != nil {
		// Single DOM call to append all children
		parseA.appendChild.Call("call", parseState.parent.value, parseState.fragment)
	}
}

// BatchSetAttributes sets multiple attributes in a single boundary crossing
func (parseA *WASMDOMAdapter) BatchSetAttributes(parseNode runtime.DOMNode, parseAttrs map[string]string) {
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		if len(parseAttrs) == 0 {
			return
		}

		parsePayload := js.Global().Get("Object").New()
		for parseName, parseValue := range parseAttrs {
			parsePayload.Set(parseName, parseValue)
		}
		parseA.batchSetAttributes.Invoke(parseWasmNode.value, parsePayload)
	}
}

func (parseA *WASMDOMAdapter) WrapFunction(parseFn interface{}) interface{} {
	switch parseF := parseFn.(type) {
	case func():
		return js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseF()
			return nil
		})
	case func(string):
		return js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
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
			if len(parseArgs3) > 0 {
				parseF(parseArgs3[0])
			}
			return nil
		})
	case func() error:
		return js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
			parseF()
			return nil
		})
	case func(js.Value) error:
		return js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
			if len(parseArgs5) > 0 {
				parseF(parseArgs5[0])
			}
			return nil
		})
	case func(runtime.GoEvent):
		return js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
			if len(parseArgs6) > 0 {
				parseF(runtime.NewGoEvent(parseArgs6[0]))
			}
			return nil
		})
	case func(runtime.GoEvent) error:
		return js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} {
			if len(parseArgs7) > 0 {
				parseF(runtime.NewGoEvent(parseArgs7[0]))
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
}

var _ runtime.EventAdapter = (*WASMEventAdapter)(nil)

// NewWASMEventAdapter creates an event adapter backed by browser DOM listeners.
func NewWASMEventAdapter() *WASMEventAdapter {
	parseElemProto := js.Global().Get("Element").Get("prototype")
	return &WASMEventAdapter{
		addEventListener:    parseElemProto.Get("addEventListener"),
		removeEventListener: parseElemProto.Get("removeEventListener"),
	}
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
	if parseWasmNode, parseOk := parseNode.(*WASMDOMNode); parseOk {
		if parseWasmHandler, parseOk2 := parseHandler.(*wasmEventHandler); parseOk2 {
			parseA.addEventListener.Call("call", parseWasmNode.value, parseEventType, parseWasmHandler.fn)
		}
	}
}

func (parseA *WASMEventAdapter) RemoveEventListener(parseNode runtime.DOMNode, parseEventType string, parseHandler runtime.EventHandler) {
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
		parseDeadline := &wasmDeadline{}
		if len(parseArgs) > 0 {
			parseDeadline.value = parseArgs[0]
		}
		parseCallback(parseDeadline)
		// Release after callback executes
		parseJsFn.Release()
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
		parseCallback()
		// Release after callback executes
		parseJsFn.Release()
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
	window js.Value
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
	parseB.window.Call("addEventListener", "popstate", js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseLocation := parseB.window.Get("location")
		parsePath := parseLocation.Get("pathname").String()
		parseCallback(parsePath)
		return nil
	}))
}

func (parseB *WASMBrowserState) SetItem(parseKey, parseValue string) error {
	parseStorage := parseB.window.Get("localStorage")
	if !parseStorage.IsNull() && !parseStorage.IsUndefined() {
		parseStorage.Call("setItem", parseKey, parseValue)
	}
	return nil
}

func (parseB *WASMBrowserState) GetItem(parseKey string) (string, bool) {
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
