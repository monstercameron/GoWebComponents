//go:build js && wasm
// +build js,wasm

package jsdom

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// WASMDOMNode wraps a js.Value representing a DOM node
type WASMDOMNode struct {
	value js.Value
}

var _ runtime.DOMNode = (*WASMDOMNode)(nil)

func NewWASMDOMNode(value js.Value) runtime.DOMNode {
	return &WASMDOMNode{value: value}
}

func (n *WASMDOMNode) IsNull() bool {
	return n.value.IsNull() || n.value.IsUndefined()
}

func (n *WASMDOMNode) Equals(other runtime.DOMNode) bool {
	if otherNode, ok := other.(*WASMDOMNode); ok {
		return n.value.Equal(otherNode.value)
	}
	return false
}

func (n *WASMDOMNode) Value() js.Value {
	return n.value
}

// WASMDOMAdapter implements DOMAdapter for browser/WASM
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

func NewWASMDOMAdapter() *WASMDOMAdapter {
	doc := js.Global().Get("document")
	// Pre-cache DOM prototype methods
	elemProto := js.Global().Get("Element").Get("prototype")
	batchSetAttributes := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		node := args[0]
		attrs := args[1]
		keys := js.Global().Get("Object").Call("keys", attrs)
		length := keys.Get("length").Int()
		for i := 0; i < length; i++ {
			key := keys.Index(i).String()
			node.Call("setAttribute", key, attrs.Get(key).String())
		}
		return nil
	})
	return &WASMDOMAdapter{
		document: doc,
		// Bind methods to document to ensure correct 'this' context when Invoked
		createElement:    doc.Get("createElement").Call("bind", doc),
		createTextNode:   doc.Get("createTextNode").Call("bind", doc),
		querySelector:    doc.Get("querySelector").Call("bind", doc),
		querySelectorAll: doc.Get("querySelectorAll").Call("bind", doc),
		getElementByID:   doc.Get("getElementById").Call("bind", doc),
		getByClassName:   doc.Get("getElementsByClassName").Call("bind", doc),
		getByTagName:     doc.Get("getElementsByTagName").Call("bind", doc),
		createFragment:   doc.Get("createDocumentFragment").Call("bind", doc),
		// Cache element methods (not bound, will use Call)
		appendChild:         elemProto.Get("appendChild"),
		removeChild:         elemProto.Get("removeChild"),
		setAttribute:        elemProto.Get("setAttribute"),
		removeAttribute:     elemProto.Get("removeAttribute"),
		insertBefore:        elemProto.Get("insertBefore"),
		replaceChild:        elemProto.Get("replaceChild"),
		addEventListener:    elemProto.Get("addEventListener"),
		removeEventListener: elemProto.Get("removeEventListener"),
		batchSetAttributes:  batchSetAttributes,
	}
}

func (a *WASMDOMAdapter) CreateElement(tag string) runtime.DOMNode {
	// Check if document is available
	if a.document.IsNull() || a.document.IsUndefined() {
		// Document not available - return null node
		return &WASMDOMNode{value: js.Null()}
	}

	// Use Invoke on the cached function instead of Call on the document
	// This saves a property lookup on every call
	elem := a.createElement.Invoke(tag)
	if elem.IsNull() || elem.IsUndefined() {
		// This shouldn't happen, but handle it gracefully
		return &WASMDOMNode{value: js.Null()}
	}
	return &WASMDOMNode{value: elem}
}

func (a *WASMDOMAdapter) CreateTextNode(text string) runtime.DOMNode {
	// Check if document is available
	if a.document.IsNull() || a.document.IsUndefined() {
		return &WASMDOMNode{value: js.Null()}
	}

	// Use Invoke on the cached function
	textNode := a.createTextNode.Invoke(text)
	if textNode.IsNull() || textNode.IsUndefined() {
		return &WASMDOMNode{value: js.Null()}
	}
	return &WASMDOMNode{value: textNode}
}

func (a *WASMDOMAdapter) SetAttribute(node runtime.DOMNode, name, value string) {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		// Use cached method for better performance
		a.setAttribute.Call("call", wasmNode.value, name, value)
	}
}

func (a *WASMDOMAdapter) RemoveAttribute(node runtime.DOMNode, name string) {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		a.removeAttribute.Call("call", wasmNode.value, name)
	}
}

func (a *WASMDOMAdapter) SetProperty(node runtime.DOMNode, name string, value interface{}) {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		// Direct property set (fastest path)
		wasmNode.value.Set(name, value)
	}
}

func (a *WASMDOMAdapter) GetProperty(node runtime.DOMNode, name string) interface{} {
	if node == nil || node.IsNull() {
		return nil
	}
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		if !wasmNode.value.IsNull() && !wasmNode.value.IsUndefined() {
			return wasmNode.value.Get(name)
		}
	}
	return nil
}

func (a *WASMDOMAdapter) AppendChild(parent, child runtime.DOMNode) {
	// Fast path: skip nil checks when nodes are valid
	parentNode, ok1 := parent.(*WASMDOMNode)
	childNode, ok2 := child.(*WASMDOMNode)
	if !ok1 || !ok2 {
		return
	}

	// If in batch mode for this specific parent, append to the top-most fragment.
	if depth := len(a.batchStack); depth > 0 {
		state := a.batchStack[depth-1]
		if state.parent == parentNode {
			a.appendChild.Call("call", state.fragment, childNode.value)
			return
		}
	}

	// Use cached method via Call (faster than method lookup each time)
	a.appendChild.Call("call", parentNode.value, childNode.value)
}

func (a *WASMDOMAdapter) RemoveChild(parent, child runtime.DOMNode) {
	parentNode, ok1 := parent.(*WASMDOMNode)
	childNode, ok2 := child.(*WASMDOMNode)
	if !ok1 || !ok2 {
		return
	}
	// Use cached method
	a.removeChild.Call("call", parentNode.value, childNode.value)
}

func (a *WASMDOMAdapter) InsertBefore(parent, newNode, referenceNode runtime.DOMNode) {
	parentN, ok1 := parent.(*WASMDOMNode)
	newN, ok2 := newNode.(*WASMDOMNode)
	refN, ok3 := referenceNode.(*WASMDOMNode)
	if ok1 && ok2 && ok3 {
		a.insertBefore.Call("call", parentN.value, newN.value, refN.value)
	}
}

func (a *WASMDOMAdapter) ReplaceChild(parent, newNode, oldNode runtime.DOMNode) {
	parentN, ok1 := parent.(*WASMDOMNode)
	newN, ok2 := newNode.(*WASMDOMNode)
	oldN, ok3 := oldNode.(*WASMDOMNode)
	if ok1 && ok2 && ok3 {
		a.replaceChild.Call("call", parentN.value, newN.value, oldN.value)
	}
}

func (a *WASMDOMAdapter) QuerySelector(selector string) interface{} {
	result := a.querySelector.Invoke(selector)
	if result.IsNull() || result.IsUndefined() {
		return nil
	}
	return &WASMDOMNode{value: result}
}

func (a *WASMDOMAdapter) QuerySelectorAll(selector string) []runtime.DOMNode {
	nodeList := a.querySelectorAll.Invoke(selector)
	length := nodeList.Get("length").Int()

	nodes := make([]runtime.DOMNode, length)
	for i := 0; i < length; i++ {
		nodes[i] = &WASMDOMNode{value: nodeList.Index(i)}
	}
	return nodes
}

func (a *WASMDOMAdapter) GetElementById(id string) runtime.DOMNode {
	result := a.getElementByID.Invoke(id)
	if result.IsNull() || result.IsUndefined() {
		return nil
	}
	return &WASMDOMNode{value: result}
}

func (a *WASMDOMAdapter) GetElementsByClassName(className string) []runtime.DOMNode {
	htmlCollection := a.getByClassName.Invoke(className)
	length := htmlCollection.Get("length").Int()

	nodes := make([]runtime.DOMNode, length)
	for i := 0; i < length; i++ {
		nodes[i] = &WASMDOMNode{value: htmlCollection.Index(i)}
	}
	return nodes
}

func (a *WASMDOMAdapter) GetElementsByTagName(tagName string) []runtime.DOMNode {
	htmlCollection := a.getByTagName.Invoke(tagName)
	length := htmlCollection.Get("length").Int()

	nodes := make([]runtime.DOMNode, length)
	for i := 0; i < length; i++ {
		nodes[i] = &WASMDOMNode{value: htmlCollection.Index(i)}
	}
	return nodes
}

func (a *WASMDOMAdapter) SetInnerHTML(node runtime.DOMNode, html string) {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		wasmNode.value.Set("innerHTML", html)
	}
}

func (a *WASMDOMAdapter) GetInnerHTML(node runtime.DOMNode) string {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		return wasmNode.value.Get("innerHTML").String()
	}
	return ""
}

func (a *WASMDOMAdapter) SetTextContent(node runtime.DOMNode, text string) {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		wasmNode.value.Set("textContent", text)
	}
}

func (a *WASMDOMAdapter) GetTextContent(node runtime.DOMNode) string {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		return wasmNode.value.Get("textContent").String()
	}
	return ""
}

func (a *WASMDOMAdapter) AddClass(node runtime.DOMNode, className string) {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		classList := wasmNode.value.Get("classList")
		classList.Call("add", className)
	}
}

func (a *WASMDOMAdapter) RemoveClass(node runtime.DOMNode, className string) {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		classList := wasmNode.value.Get("classList")
		classList.Call("remove", className)
	}
}

func (a *WASMDOMAdapter) ToggleClass(node runtime.DOMNode, className string) {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		classList := wasmNode.value.Get("classList")
		classList.Call("toggle", className)
	}
}

func (a *WASMDOMAdapter) GetParent(node runtime.DOMNode) runtime.DOMNode {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		parent := wasmNode.value.Get("parentNode")
		if !parent.IsNull() && !parent.IsUndefined() {
			return &WASMDOMNode{value: parent}
		}
	}
	return nil
}

func (a *WASMDOMAdapter) GetChildren(node runtime.DOMNode) []runtime.DOMNode {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		children := wasmNode.value.Get("children")
		length := children.Get("length").Int()

		nodes := make([]runtime.DOMNode, length)
		for i := 0; i < length; i++ {
			nodes[i] = &WASMDOMNode{value: children.Index(i)}
		}
		return nodes
	}
	return nil
}

func (a *WASMDOMAdapter) GetFirstChild(node runtime.DOMNode) runtime.DOMNode {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		firstChild := wasmNode.value.Get("firstChild")
		if !firstChild.IsNull() && !firstChild.IsUndefined() {
			return &WASMDOMNode{value: firstChild}
		}
	}
	return nil
}

func (a *WASMDOMAdapter) GetNextSibling(node runtime.DOMNode) runtime.DOMNode {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		nextSibling := wasmNode.value.Get("nextSibling")
		if !nextSibling.IsNull() && !nextSibling.IsUndefined() {
			return &WASMDOMNode{value: nextSibling}
		}
	}
	return nil
}

func (a *WASMDOMAdapter) SetStyle(node runtime.DOMNode, property, value string) {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		style := wasmNode.value.Get("style")
		style.Set(property, value)
	}
}

func (a *WASMDOMAdapter) SetStyles(node runtime.DOMNode, styles map[string]string) {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		style := wasmNode.value.Get("style")
		// Batch style updates by caching style object
		for property, value := range styles {
			style.Set(property, value)
		}
	}
}

// BeginBatch starts batching DOM operations for a parent node
func (a *WASMDOMAdapter) BeginBatch(parent runtime.DOMNode) {
	if parentNode, ok := parent.(*WASMDOMNode); ok {
		depth := len(a.batchStack)
		var fragment js.Value
		if depth < len(a.fragmentPool) {
			fragment = a.fragmentPool[depth]
		} else {
			fragment = a.createFragment.Invoke()
			a.fragmentPool = append(a.fragmentPool, fragment)
		}
		a.batchStack = append(a.batchStack, wasmBatchState{parent: parentNode, fragment: fragment})
	}
}

// EndBatch commits all batched operations
func (a *WASMDOMAdapter) EndBatch() {
	depth := len(a.batchStack)
	if depth == 0 {
		return
	}

	state := a.batchStack[depth-1]
	a.batchStack = a.batchStack[:depth-1]
	if state.parent != nil {
		// Single DOM call to append all children
		a.appendChild.Call("call", state.parent.value, state.fragment)
	}
}

// BatchSetAttributes sets multiple attributes in a single boundary crossing
func (a *WASMDOMAdapter) BatchSetAttributes(node runtime.DOMNode, attrs map[string]string) {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		if len(attrs) == 0 {
			return
		}

		payload := js.Global().Get("Object").New()
		for name, value := range attrs {
			payload.Set(name, value)
		}
		a.batchSetAttributes.Invoke(wasmNode.value, payload)
	}
}

func (a *WASMDOMAdapter) WrapFunction(fn interface{}) interface{} {
	switch f := fn.(type) {
	case func():
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			f()
			return nil
		})
	case func(string):
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) == 0 {
				f("")
				return nil
			}
			target := args[0].Get("target")
			if target.IsNull() || target.IsUndefined() {
				f("")
				return nil
			}
			value := target.Get("value")
			if value.IsNull() || value.IsUndefined() {
				f("")
				return nil
			}
			f(value.String())
			return nil
		})
	case func(js.Value):
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				f(args[0])
			}
			return nil
		})
	case func() error:
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			f()
			return nil
		})
	case func(js.Value) error:
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				f(args[0])
			}
			return nil
		})
	case func(runtime.GoEvent):
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				f(runtime.NewGoEvent(args[0]))
			}
			return nil
		})
	case func(runtime.GoEvent) error:
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				f(runtime.NewGoEvent(args[0]))
			}
			return nil
		})
	default:
		return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return nil
		})
	}
}

// WASMEventAdapter implements EventAdapter for browser/WASM
type WASMEventAdapter struct {
	addEventListener    js.Value
	removeEventListener js.Value
}

var _ runtime.EventAdapter = (*WASMEventAdapter)(nil)

func NewWASMEventAdapter() *WASMEventAdapter {
	elemProto := js.Global().Get("Element").Get("prototype")
	return &WASMEventAdapter{
		addEventListener:    elemProto.Get("addEventListener"),
		removeEventListener: elemProto.Get("removeEventListener"),
	}
}

// wasmEventHandler wraps a js.Func for event handling
type wasmEventHandler struct {
	fn     js.Func
	goFunc func(runtime.Event)
}

var _ runtime.EventHandler = (*wasmEventHandler)(nil)

func (h *wasmEventHandler) Release() {
	h.fn.Release()
}

func (a *WASMEventAdapter) CreateEventHandler(fn func(runtime.Event)) runtime.EventHandler {
	// Create a wasmEvent wrapper
	jsFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			event := &wasmEvent{value: args[0]}
			fn(event)
		}
		return nil
	})

	return &wasmEventHandler{
		fn:     jsFn,
		goFunc: fn,
	}
}

func (a *WASMEventAdapter) ReleaseEventHandler(handler runtime.EventHandler) {
	handler.Release()
}

func (a *WASMEventAdapter) AddEventListener(node runtime.DOMNode, eventType string, handler runtime.EventHandler) {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		if wasmHandler, ok := handler.(*wasmEventHandler); ok {
			a.addEventListener.Call("call", wasmNode.value, eventType, wasmHandler.fn)
		}
	}
}

func (a *WASMEventAdapter) RemoveEventListener(node runtime.DOMNode, eventType string, handler runtime.EventHandler) {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		if wasmHandler, ok := handler.(*wasmEventHandler); ok {
			a.removeEventListener.Call("call", wasmNode.value, eventType, wasmHandler.fn)
		}
	}
}

// wasmEvent implements Event for browser/WASM
type wasmEvent struct {
	value js.Value
}

var _ runtime.Event = (*wasmEvent)(nil)

func (e *wasmEvent) PreventDefault() {
	e.value.Call("preventDefault")
}

func (e *wasmEvent) StopPropagation() {
	e.value.Call("stopPropagation")
}

func (e *wasmEvent) GetValue() string {
	target := e.value.Get("target")
	if !target.IsNull() && !target.IsUndefined() {
		value := target.Get("value")
		if !value.IsNull() && !value.IsUndefined() {
			return value.String()
		}
	}
	return ""
}

func (e *wasmEvent) GetTarget() runtime.DOMNode {
	target := e.value.Get("target")
	if !target.IsNull() && !target.IsUndefined() {
		return &WASMDOMNode{value: target}
	}
	return nil
}

func (e *wasmEvent) GetKeyCode() int {
	keyCode := e.value.Get("keyCode")
	if !keyCode.IsNull() && !keyCode.IsUndefined() {
		return keyCode.Int()
	}
	return 0
}

func (e *wasmEvent) GetKey() string {
	key := e.value.Get("key")
	if !key.IsNull() && !key.IsUndefined() {
		return key.String()
	}
	return ""
}

func (e *wasmEvent) IsChecked() bool {
	target := e.value.Get("target")
	if !target.IsNull() && !target.IsUndefined() {
		checked := target.Get("checked")
		if !checked.IsNull() && !checked.IsUndefined() {
			return checked.Bool()
		}
	}
	return false
}

// WASMScheduler implements Scheduler for browser/WASM
type WASMScheduler struct {
	window js.Value
}

var _ runtime.Scheduler = (*WASMScheduler)(nil)

func NewWASMScheduler() *WASMScheduler {
	return &WASMScheduler{
		window: js.Global(),
	}
}

func (s *WASMScheduler) RequestIdleCallback(callback func(runtime.Deadline)) {
	// Wrap the callback - release after execution
	var jsFn js.Func
	jsFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		deadline := &wasmDeadline{}
		if len(args) > 0 {
			deadline.value = args[0]
		}
		callback(deadline)
		// Release after callback executes
		jsFn.Release()
		return nil
	})

	// Check if requestIdleCallback is available
	if s.window.Get("requestIdleCallback").Truthy() {
		s.window.Call("requestIdleCallback", jsFn)
	} else {
		// Fallback to setTimeout
		s.window.Call("setTimeout", jsFn, 0)
	}
}

func (s *WASMScheduler) SetTimeout(callback func(), delay int) {
	// Always use setTimeout (even for delay 0) to allow goroutines to run
	// This ensures that goroutines calling setState can enqueue updates before workLoop processes them
	var jsFn js.Func
	jsFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		callback()
		// Release after callback executes
		jsFn.Release()
		return nil
	})

	s.window.Call("setTimeout", jsFn, delay)
}

func (s *WASMScheduler) CancelIdleCallback(id interface{}) {
	if s.window.Get("cancelIdleCallback").Truthy() {
		s.window.Call("cancelIdleCallback", id)
	}
}

// wasmDeadline implements Deadline for browser/WASM
type wasmDeadline struct {
	value js.Value
}

var _ runtime.Deadline = (*wasmDeadline)(nil)

func (d *wasmDeadline) TimeRemaining() float64 {
	if d.value.IsUndefined() || d.value.IsNull() {
		return 50.0 // Default to 50ms
	}
	if d.value.Get("timeRemaining").Truthy() {
		return d.value.Call("timeRemaining").Float()
	}
	return 50.0
}

func (d *wasmDeadline) DidTimeout() bool {
	if d.value.IsUndefined() || d.value.IsNull() {
		return false
	}
	return d.value.Get("didTimeout").Bool()
}

// WASMBrowserState implements BrowserState for browser/WASM
type WASMBrowserState struct {
	window js.Value
}

var _ runtime.BrowserState = (*WASMBrowserState)(nil)

func NewWASMBrowserState() *WASMBrowserState {
	return &WASMBrowserState{
		window: js.Global(),
	}
}

func (b *WASMBrowserState) PushState(state interface{}, title, url string) {
	history := b.window.Get("history")
	history.Call("pushState", state, title, url)
}

func (b *WASMBrowserState) ReplaceState(state interface{}, title, url string) {
	history := b.window.Get("history")
	history.Call("replaceState", state, title, url)
}

func (b *WASMBrowserState) GetCurrentPath() string {
	location := b.window.Get("location")
	return location.Get("pathname").String()
}

func (b *WASMBrowserState) OnPopState(callback func(path string)) {
	b.window.Call("addEventListener", "popstate", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		location := b.window.Get("location")
		path := location.Get("pathname").String()
		callback(path)
		return nil
	}))
}

func (b *WASMBrowserState) SetItem(key, value string) error {
	storage := b.window.Get("localStorage")
	if !storage.IsNull() && !storage.IsUndefined() {
		storage.Call("setItem", key, value)
	}
	return nil
}

func (b *WASMBrowserState) GetItem(key string) (string, bool) {
	storage := b.window.Get("localStorage")
	if storage.IsNull() || storage.IsUndefined() {
		return "", false
	}

	value := storage.Call("getItem", key)
	if value.IsNull() {
		return "", false
	}
	return value.String(), true
}

func (b *WASMBrowserState) RemoveItem(key string) {
	storage := b.window.Get("localStorage")
	if !storage.IsNull() && !storage.IsUndefined() {
		storage.Call("removeItem", key)
	}
}

func (b *WASMBrowserState) GetHash() string {
	location := b.window.Get("location")
	return location.Get("hash").String()
}

func (b *WASMBrowserState) SetHash(hash string) {
	location := b.window.Get("location")
	location.Set("hash", hash)
}

func (b *WASMBrowserState) Reload() {
	location := b.window.Get("location")
	location.Call("reload")
}
