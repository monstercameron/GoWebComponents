//go:build js && wasm
// +build js,wasm

package router

import (
	"syscall/js"
	"testing"
)

func installRouterBrowserEnv(t *testing.T) {
	t.Helper()
	global := js.Global()
	objectCtor := global.Get("Object")
	arrayCtor := global.Get("Array")
	reflectObj := global.Get("Reflect")

	prevDoc := global.Get("document")
	prevElement := global.Get("Element")
	prevWindow := global.Get("window")
	prevHistory := global.Get("history")
	prevLocation := global.Get("location")

	makeNode := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		tag := ""
		if len(args) > 0 {
			tag = args[0].String()
		}
		node := objectCtor.New()
		node.Set("tagName", tag)
		node.Set("attributes", objectCtor.New())
		node.Set("children", arrayCtor.New())
		return node
	})

	appendChild := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		this.Get("children").Call("push", args[0])
		return args[0]
	})
	removeChild := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		children := this.Get("children")
		length := children.Get("length").Int()
		for index := 0; index < length; index++ {
			if children.Index(index).Equal(args[0]) {
				children.Call("splice", index, 1)
				break
			}
		}
		return args[0]
	})
	setAttribute := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		this.Get("attributes").Set(args[0].String(), args[1].String())
		return nil
	})
	removeAttribute := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		reflectObj.Call("deleteProperty", this.Get("attributes"), args[0].String())
		return nil
	})
	insertBefore := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		this.Get("children").Call("push", args[0])
		return args[0]
	})
	replaceChild := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		children := this.Get("children")
		length := children.Get("length").Int()
		for index := 0; index < length; index++ {
			if children.Index(index).Equal(args[1]) {
				children.SetIndex(index, args[0])
				break
			}
		}
		return args[1]
	})
	addEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	removeEventListener := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })

	elementCtor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return objectCtor.New()
	})
	proto := objectCtor.New()
	proto.Set("appendChild", appendChild)
	proto.Set("removeChild", removeChild)
	proto.Set("setAttribute", setAttribute)
	proto.Set("removeAttribute", removeAttribute)
	proto.Set("insertBefore", insertBefore)
	proto.Set("replaceChild", replaceChild)
	proto.Set("addEventListener", addEventListener)
	proto.Set("removeEventListener", removeEventListener)
	elementCtor.Set("prototype", proto)

	docCreateElement := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return makeNode.Invoke(args[0].String())
	})
	docCreateTextNode := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		node := objectCtor.New()
		node.Set("textContent", args[0].String())
		return node
	})
	docCreateFragment := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		frag := objectCtor.New()
		frag.Set("isFragment", true)
		frag.Set("children", arrayCtor.New())
		frag.Set("appendChild", appendChild)
		return frag
	})
	docQuerySelector := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return makeNode.Invoke("div")
	})
	doc := objectCtor.New()
	doc.Set("createElement", docCreateElement)
	doc.Set("createTextNode", docCreateTextNode)
	doc.Set("createDocumentFragment", docCreateFragment)
	doc.Set("querySelector", docQuerySelector)

	location := objectCtor.New()
	location.Set("hash", "")
	location.Set("pathname", "/")
	locationReplace := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		location.Set("hash", args[0].String())
		return nil
	})
	location.Set("replace", locationReplace)

	history := objectCtor.New()
	pushState := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 2 {
			location.Set("pathname", args[2].String())
		}
		return nil
	})
	replaceState := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 2 {
			location.Set("pathname", args[2].String())
		}
		return nil
	})
	history.Set("pushState", pushState)
	history.Set("replaceState", replaceState)

	window := objectCtor.New()
	window.Set("addEventListener", addEventListener)
	window.Set("removeEventListener", removeEventListener)
	window.Set("document", doc)
	window.Set("history", history)
	window.Set("location", location)

	global.Set("document", doc)
	global.Set("Element", elementCtor)
	global.Set("window", window)
	global.Set("history", history)
	global.Set("location", location)

	t.Cleanup(func() {
		global.Set("document", prevDoc)
		global.Set("Element", prevElement)
		global.Set("window", prevWindow)
		global.Set("history", prevHistory)
		global.Set("location", prevLocation)
		makeNode.Release()
		appendChild.Release()
		removeChild.Release()
		setAttribute.Release()
		removeAttribute.Release()
		insertBefore.Release()
		replaceChild.Release()
		addEventListener.Release()
		removeEventListener.Release()
		elementCtor.Release()
		docCreateElement.Release()
		docCreateTextNode.Release()
		docCreateFragment.Release()
		docQuerySelector.Release()
		locationReplace.Release()
		pushState.Release()
		replaceState.Release()
	})
}
