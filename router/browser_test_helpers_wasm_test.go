//go:build js && wasm
// +build js,wasm

package router

import (
	"strings"
	"syscall/js"
	"testing"
)

func installRouterBrowserEnv(t testing.TB) {
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
	prevInitialized := initialized
	var decorateNode func(js.Value)

	makeNode := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		tag := ""
		if len(args) > 0 {
			tag = args[0].String()
		}
		node := objectCtor.New()
		node.Set("tagName", tag)
		node.Set("attributes", objectCtor.New())
		node.Set("children", arrayCtor.New())
		if decorateNode != nil {
			decorateNode(node)
		}
		return node
	})

	appendChild := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		child := args[0]
		if child.Get("isFragment").Truthy() {
			fragmentChildren := child.Get("children")
			children := this.Get("children")
			length := fragmentChildren.Get("length").Int()
			for i := 0; i < length; i++ {
				children.Call("push", fragmentChildren.Index(i))
			}
			child.Set("children", arrayCtor.New())
			return child
		}
		child.Set("parentNode", this)
		this.Get("children").Call("push", child)
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
	getAttribute := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		value := this.Get("attributes").Get(args[0].String())
		if value.IsUndefined() {
			return js.Null()
		}
		return value
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
	removeNode := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		parent := this.Get("parentNode")
		if !parent.Truthy() {
			return nil
		}
		children := parent.Get("children")
		length := children.Get("length").Int()
		for index := 0; index < length; index++ {
			if children.Index(index).Equal(this) {
				children.Call("splice", index, 1)
				break
			}
		}
		this.Set("parentNode", js.Null())
		return nil
	})
	decorateNode = func(node js.Value) {
		node.Set("appendChild", appendChild)
		node.Set("removeChild", removeChild)
		node.Set("setAttribute", setAttribute)
		node.Set("removeAttribute", removeAttribute)
		node.Set("getAttribute", getAttribute)
		node.Set("insertBefore", insertBefore)
		node.Set("replaceChild", replaceChild)
		node.Set("addEventListener", addEventListener)
		node.Set("removeEventListener", removeEventListener)
		node.Set("remove", removeNode)
	}

	elementCtor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return objectCtor.New()
	})
	proto := objectCtor.New()
	proto.Set("appendChild", appendChild)
	proto.Set("removeChild", removeChild)
	proto.Set("setAttribute", setAttribute)
	proto.Set("removeAttribute", removeAttribute)
	proto.Set("getAttribute", getAttribute)
	proto.Set("insertBefore", insertBefore)
	proto.Set("replaceChild", replaceChild)
	proto.Set("addEventListener", addEventListener)
	proto.Set("removeEventListener", removeEventListener)
	proto.Set("remove", removeNode)
	elementCtor.Set("prototype", proto)

	findHeadChild := func(head js.Value, tag string, attrName string, attrValue string) js.Value {
		children := head.Get("children")
		length := children.Get("length").Int()
		for index := 0; index < length; index++ {
			child := children.Index(index)
			if !strings.EqualFold(child.Get("tagName").String(), tag) {
				continue
			}
			if child.Get("attributes").Get(attrName).String() == attrValue {
				return child
			}
		}
		return js.Null()
	}

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
	head := makeNode.Invoke("head")
	body := makeNode.Invoke("body")
	docQuerySelector := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		selector := args[0].String()
		switch selector {
		case "head":
			return head
		case `meta[name="description"]`:
			return findHeadChild(head, "meta", "name", "description")
		case `link[rel="canonical"]`:
			return findHeadChild(head, "link", "rel", "canonical")
		default:
			return makeNode.Invoke("div")
		}
	})
	docQuerySelectorAll := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		list := arrayCtor.New()
		list.Call("push", makeNode.Invoke("div"))
		list.Set("item", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return this.Index(args[0].Int())
		}))
		return list
	})
	docGetElementByID := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		node := makeNode.Invoke("div")
		if len(args) > 0 {
			node.Set("id", args[0].String())
		}
		return node
	})
	docGetElementsByClassName := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		list := arrayCtor.New()
		list.Call("push", makeNode.Invoke("div"))
		list.Set("item", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return this.Index(args[0].Int())
		}))
		return list
	})
	docGetElementsByTagName := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		list := arrayCtor.New()
		list.Call("push", makeNode.Invoke("div"))
		list.Set("item", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			return this.Index(args[0].Int())
		}))
		return list
	})
	doc := objectCtor.New()
	doc.Set("createElement", docCreateElement)
	doc.Set("createTextNode", docCreateTextNode)
	doc.Set("createDocumentFragment", docCreateFragment)
	doc.Set("querySelector", docQuerySelector)
	doc.Set("querySelectorAll", docQuerySelectorAll)
	doc.Set("getElementById", docGetElementByID)
	doc.Set("getElementsByClassName", docGetElementsByClassName)
	doc.Set("getElementsByTagName", docGetElementsByTagName)
	doc.Set("head", head)
	doc.Set("body", body)
	doc.Set("title", "")

	location := objectCtor.New()
	location.Set("hash", "")
	location.Set("pathname", "/")
	location.Set("search", "")
	location.Set("reload", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	locationReplace := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		next := args[0].String()
		if len(next) > 0 && next[0] != '#' {
			location.Set("hash", "#"+next)
		} else {
			location.Set("hash", next)
		}
		if index := strings.Index(next, "?"); index >= 0 {
			location.Set("search", next[index:])
		} else {
			location.Set("search", "")
		}
		return nil
	})
	location.Set("replace", locationReplace)

	history := objectCtor.New()
	pushState := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 2 {
			next := args[2].String()
			if idx := strings.Index(next, "?"); idx >= 0 {
				location.Set("pathname", next[:idx])
				location.Set("search", next[idx:])
			} else {
				location.Set("pathname", next)
				location.Set("search", "")
			}
		}
		return nil
	})
	replaceState := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 2 {
			next := args[2].String()
			if idx := strings.Index(next, "?"); idx >= 0 {
				location.Set("pathname", next[:idx])
				location.Set("search", next[idx:])
			} else {
				location.Set("pathname", next)
				location.Set("search", "")
			}
		}
		return nil
	})
	history.Set("pushState", pushState)
	history.Set("replaceState", replaceState)

	storage := objectCtor.New()
	storageData := objectCtor.New()
	storage.Set("setItem", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		storageData.Set(args[0].String(), args[1].String())
		return nil
	}))
	storage.Set("getItem", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		value := storageData.Get(args[0].String())
		if value.IsUndefined() {
			return js.Null()
		}
		return value
	}))
	storage.Set("removeItem", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		reflectObj.Call("deleteProperty", storageData, args[0].String())
		return nil
	}))

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
	initialized = false

	t.Cleanup(func() {
		global.Set("document", prevDoc)
		global.Set("Element", prevElement)
		global.Set("window", prevWindow)
		global.Set("history", prevHistory)
		global.Set("location", prevLocation)
		initialized = prevInitialized
		makeNode.Release()
		appendChild.Release()
		removeChild.Release()
		setAttribute.Release()
		removeAttribute.Release()
		getAttribute.Release()
		insertBefore.Release()
		replaceChild.Release()
		addEventListener.Release()
		removeEventListener.Release()
		removeNode.Release()
		elementCtor.Release()
		docCreateElement.Release()
		docCreateTextNode.Release()
		docCreateFragment.Release()
		docQuerySelector.Release()
		docQuerySelectorAll.Release()
		docGetElementByID.Release()
		docGetElementsByClassName.Release()
		docGetElementsByTagName.Release()
		locationReplace.Release()
		pushState.Release()
		replaceState.Release()
	})
}
