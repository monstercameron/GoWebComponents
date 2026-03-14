//go:build js && wasm
// +build js,wasm

package jsdom

import (
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func installBenchmarkDOM() func() {
	global := js.Global()
	objectCtor := global.Get("Object")
	arrayCtor := global.Get("Array")
	reflectObj := global.Get("Reflect")

	prevDoc := global.Get("document")
	prevElement := global.Get("Element")

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
		for i := 0; i < length; i++ {
			if children.Index(i).Equal(args[0]) {
				children.Call("splice", i, 1)
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

	elementCtor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return objectCtor.New()
	})
	proto := objectCtor.New()
	proto.Set("appendChild", appendChild)
	proto.Set("removeChild", removeChild)
	proto.Set("setAttribute", setAttribute)
	proto.Set("removeAttribute", removeAttribute)
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

	global.Set("document", doc)
	global.Set("Element", elementCtor)

	return func() {
		global.Set("document", prevDoc)
		global.Set("Element", prevElement)
		makeNode.Release()
		appendChild.Release()
		removeChild.Release()
		setAttribute.Release()
		removeAttribute.Release()
		docCreateElement.Release()
		docCreateTextNode.Release()
		docCreateFragment.Release()
		docQuerySelector.Release()
		elementCtor.Release()
	}
}

func BenchmarkWASMDOMAdapterCreateElement(b *testing.B) {
	cleanup := installBenchmarkDOM()
	defer cleanup()

	adapter := NewWASMDOMAdapter()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		node := adapter.CreateElement("div")
		if node == nil || node.IsNull() {
			b.Fatal("expected DOM node")
		}
	}
}

func BenchmarkWASMDOMAdapterSetAttribute(b *testing.B) {
	cleanup := installBenchmarkDOM()
	defer cleanup()

	adapter := NewWASMDOMAdapter()
	node := adapter.CreateElement("div")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		adapter.SetAttribute(node, "data-id", "42")
	}
}

func BenchmarkWASMDOMAdapterAppendChild(b *testing.B) {
	cleanup := installBenchmarkDOM()
	defer cleanup()

	adapter := NewWASMDOMAdapter()
	parent := adapter.CreateElement("div")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		child := adapter.CreateElement("span")
		adapter.AppendChild(parent, child)
	}
}

func BenchmarkWASMDOMAdapterWrapFunctionNoArgsInvoke(b *testing.B) {
	cleanup := installBenchmarkDOM()
	defer cleanup()

	adapter := NewWASMDOMAdapter()
	wrapped := adapter.WrapFunction(func() {}).(js.Func)
	defer wrapped.Release()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		wrapped.Invoke()
	}
}

func BenchmarkWASMDOMAdapterWrapFunctionStringInvoke(b *testing.B) {
	cleanup := installBenchmarkDOM()
	defer cleanup()

	adapter := NewWASMDOMAdapter()
	wrapped := adapter.WrapFunction(func(string) {}).(js.Func)
	defer wrapped.Release()

	target := js.Global().Get("Object").New()
	target.Set("value", "abc")
	event := js.Global().Get("Object").New()
	event.Set("target", target)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		wrapped.Invoke(event)
	}
}

func legacyWrapFunction(fn interface{}) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		switch f := fn.(type) {
		case func():
			f()
		case func(string):
			if len(args) > 0 {
				event := args[0]
				target := event.Get("target")
				if !target.IsNull() && !target.IsUndefined() {
					value := target.Get("value")
					if !value.IsNull() && !value.IsUndefined() {
						strVal := value.String()
						f(strVal)
					} else {
						f("")
					}
				} else {
					f("")
				}
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

func BenchmarkLegacyWrapFunctionNoArgsInvoke(b *testing.B) {
	wrapped := legacyWrapFunction(func() {})
	defer wrapped.Release()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		wrapped.Invoke()
	}
}

func BenchmarkLegacyWrapFunctionStringInvoke(b *testing.B) {
	target := js.Global().Get("Object").New()
	target.Set("value", "abc")
	event := js.Global().Get("Object").New()
	event.Set("target", target)

	wrapped := legacyWrapFunction(func(string) {})
	defer wrapped.Release()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		wrapped.Invoke(event)
	}
}
