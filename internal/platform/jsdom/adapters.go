//go:build js && wasm
// +build js,wasm

package jsdom

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// WASMDOMNode wraps a js.Value representing a DOM node
type WASMDOMNode struct {
	value js.Value
}

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
	document js.Value
}

func NewWASMDOMAdapter() *WASMDOMAdapter {
	return &WASMDOMAdapter{
		document: js.Global().Get("document"),
	}
}

func (a *WASMDOMAdapter) CreateElement(tag string) runtime.DOMNode {
	// Check if document is available
	if a.document.IsNull() || a.document.IsUndefined() {
		// Document not available - return null node
		return &WASMDOMNode{value: js.Null()}
	}

	elem := a.document.Call("createElement", tag)
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

	textNode := a.document.Call("createTextNode", text)
	if textNode.IsNull() || textNode.IsUndefined() {
		return &WASMDOMNode{value: js.Null()}
	}
	return &WASMDOMNode{value: textNode}
}

func (a *WASMDOMAdapter) SetAttribute(node runtime.DOMNode, name, value string) {
	if node == nil || node.IsNull() {
		return // Silently ignore - node doesn't exist yet or is a text node
	}
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		if !wasmNode.value.IsNull() && !wasmNode.value.IsUndefined() {
			wasmNode.value.Call("setAttribute", name, value)
		}
	}
}

func (a *WASMDOMAdapter) RemoveAttribute(node runtime.DOMNode, name string) {
	if node == nil || node.IsNull() {
		return
	}
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		if !wasmNode.value.IsNull() && !wasmNode.value.IsUndefined() {
			wasmNode.value.Call("removeAttribute", name)
		}
	}
}

func (a *WASMDOMAdapter) SetProperty(node runtime.DOMNode, name string, value interface{}) {
	if node == nil || node.IsNull() {
		return
	}
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		if !wasmNode.value.IsNull() && !wasmNode.value.IsUndefined() {
			wasmNode.value.Set(name, value)
		}
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
	if parent == nil || parent.IsNull() || child == nil || child.IsNull() {
		return
	}
	parentNode, ok1 := parent.(*WASMDOMNode)
	childNode, ok2 := child.(*WASMDOMNode)
	if ok1 && ok2 && !parentNode.value.IsNull() && !childNode.value.IsNull() {
		parentNode.value.Call("appendChild", childNode.value)
	}
}

func (a *WASMDOMAdapter) RemoveChild(parent, child runtime.DOMNode) {
	if parent == nil || parent.IsNull() || child == nil || child.IsNull() {
		return
	}
	parentNode, ok1 := parent.(*WASMDOMNode)
	childNode, ok2 := child.(*WASMDOMNode)
	if ok1 && ok2 && !parentNode.value.IsNull() && !childNode.value.IsNull() {
		parentNode.value.Call("removeChild", childNode.value)
	}
}

func (a *WASMDOMAdapter) InsertBefore(parent, newNode, referenceNode runtime.DOMNode) {
	parentN, ok1 := parent.(*WASMDOMNode)
	newN, ok2 := newNode.(*WASMDOMNode)
	refN, ok3 := referenceNode.(*WASMDOMNode)
	if ok1 && ok2 && ok3 {
		parentN.value.Call("insertBefore", newN.value, refN.value)
	}
}

func (a *WASMDOMAdapter) ReplaceChild(parent, newNode, oldNode runtime.DOMNode) {
	parentN, ok1 := parent.(*WASMDOMNode)
	newN, ok2 := newNode.(*WASMDOMNode)
	oldN, ok3 := oldNode.(*WASMDOMNode)
	if ok1 && ok2 && ok3 {
		parentN.value.Call("replaceChild", newN.value, oldN.value)
	}
}

func (a *WASMDOMAdapter) QuerySelector(selector string) interface{} {
	result := a.document.Call("querySelector", selector)
	if result.IsNull() || result.IsUndefined() {
		return nil
	}
	return &WASMDOMNode{value: result}
}

func (a *WASMDOMAdapter) QuerySelectorAll(selector string) []runtime.DOMNode {
	nodeList := a.document.Call("querySelectorAll", selector)
	length := nodeList.Get("length").Int()

	nodes := make([]runtime.DOMNode, length)
	for i := 0; i < length; i++ {
		nodes[i] = &WASMDOMNode{value: nodeList.Call("item", i)}
	}
	return nodes
}

func (a *WASMDOMAdapter) GetElementById(id string) runtime.DOMNode {
	result := a.document.Call("getElementById", id)
	if result.IsNull() || result.IsUndefined() {
		return nil
	}
	return &WASMDOMNode{value: result}
}

func (a *WASMDOMAdapter) GetElementsByClassName(className string) []runtime.DOMNode {
	htmlCollection := a.document.Call("getElementsByClassName", className)
	length := htmlCollection.Get("length").Int()

	nodes := make([]runtime.DOMNode, length)
	for i := 0; i < length; i++ {
		nodes[i] = &WASMDOMNode{value: htmlCollection.Call("item", i)}
	}
	return nodes
}

func (a *WASMDOMAdapter) GetElementsByTagName(tagName string) []runtime.DOMNode {
	htmlCollection := a.document.Call("getElementsByTagName", tagName)
	length := htmlCollection.Get("length").Int()

	nodes := make([]runtime.DOMNode, length)
	for i := 0; i < length; i++ {
		nodes[i] = &WASMDOMNode{value: htmlCollection.Call("item", i)}
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
			nodes[i] = &WASMDOMNode{value: children.Call("item", i)}
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
		for property, value := range styles {
			style.Set(property, value)
		}
	}
}

func (a *WASMDOMAdapter) WrapFunction(fn interface{}) interface{} {
	fmt.Printf("WrapFunction called for %T\n", fn)
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		fmt.Printf("Wrapped function executed for %T\n", fn)
		switch f := fn.(type) {
		case func():
			f()
		case func(string):
			if len(args) > 0 {
				event := args[0]
				fmt.Printf("WrapFunction: func(string) called with args len %d\n", len(args))
				target := event.Get("target")
				if !target.IsNull() && !target.IsUndefined() {
					value := target.Get("value")
					fmt.Printf("WrapFunction: target found, value type: %s\n", value.Type())
					if !value.IsNull() && !value.IsUndefined() {
						strVal := value.String()
						fmt.Printf("WrapFunction: calling f with '%s'\n", strVal)
						f(strVal)
					} else {
						// Fallback for elements without value (like buttons)
						f("")
					}
				} else {
					// Fallback if no target
					f("")
				}
			} else {
				fmt.Println("WrapFunction: func(string) called with 0 args")
			}
		case func(js.Value):
			if len(args) > 0 {
				f(args[0])
			}
		case func() error:
			f()
		case func(js.Value) error:
			if len(args) > 0 {
				f(args[0])
			}
		case func(runtime.GoEvent):
			if len(args) > 0 {
				f(runtime.NewGoEvent(args[0]))
			}
		case func(runtime.GoEvent) error:
			if len(args) > 0 {
				f(runtime.NewGoEvent(args[0]))
			}
		}
		return nil
	})
}

// WASMEventAdapter implements EventAdapter for browser/WASM
type WASMEventAdapter struct{}

func NewWASMEventAdapter() *WASMEventAdapter {
	return &WASMEventAdapter{}
}

// wasmEventHandler wraps a js.Func for event handling
type wasmEventHandler struct {
	fn     js.Func
	goFunc func(runtime.Event)
}

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
			wasmNode.value.Call("addEventListener", eventType, wasmHandler.fn)
		}
	}
}

func (a *WASMEventAdapter) RemoveEventListener(node runtime.DOMNode, eventType string, handler runtime.EventHandler) {
	if wasmNode, ok := node.(*WASMDOMNode); ok {
		if wasmHandler, ok := handler.(*wasmEventHandler); ok {
			wasmNode.value.Call("removeEventListener", eventType, wasmHandler.fn)
		}
	}
}

// wasmEvent implements Event for browser/WASM
type wasmEvent struct {
	value js.Value
}

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
		fmt.Println("RequestIdleCallback invoked")
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
	if delay == 0 {
		// Call immediately for delay 0 to make updates synchronous for tests
		callback()
	} else {
		var jsFn js.Func
		jsFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			callback()
			// Release after callback executes
			jsFn.Release()
			return nil
		})

		s.window.Call("setTimeout", jsFn, delay)
	}
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
