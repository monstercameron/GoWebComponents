//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"reflect"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

type Attrs = map[string]interface{}
type Element = runtime.Element
type GoEvent = runtime.GoEvent

func toProps(attrs Attrs) html.Props {
	if attrs == nil {
		return html.Props{}
	}
	return html.Props{Raw: attrs}
}

func normalizeChildren(children ...interface{}) []ui.Node {
	result := make([]ui.Node, 0, len(children))
	for _, child := range children {
		switch value := child.(type) {
		case nil:
			continue
		case ui.Node:
			result = append(result, value)
		case string:
			result = append(result, html.Text(value))
		case fmt.Stringer:
			result = append(result, html.Text(value.String()))
		case []ui.Node:
			result = append(result, value...)
		case []interface{}:
			result = append(result, normalizeChildren(value...)...)
		default:
			result = append(result, html.Text(fmt.Sprint(value)))
		}
	}
	return result
}

func tag(name string, attrs Attrs, children ...interface{}) *Element {
	return html.Tag(name, toProps(attrs), normalizeChildren(children...)...)
}

func Text(value interface{}) *Element                      { return html.Text(fmt.Sprint(value)) }
func Div(attrs Attrs, children ...interface{}) *Element    { return tag("div", attrs, children...) }
func Form(attrs Attrs, children ...interface{}) *Element   { return tag("form", attrs, children...) }
func Button(attrs Attrs, children ...interface{}) *Element { return tag("button", attrs, children...) }
func H1(attrs Attrs, children ...interface{}) *Element     { return tag("h1", attrs, children...) }
func H2(attrs Attrs, children ...interface{}) *Element     { return tag("h2", attrs, children...) }
func H3(attrs Attrs, children ...interface{}) *Element     { return tag("h3", attrs, children...) }
func P(attrs Attrs, children ...interface{}) *Element      { return tag("p", attrs, children...) }
func Span(attrs Attrs, children ...interface{}) *Element   { return tag("span", attrs, children...) }
func Label(attrs Attrs, children ...interface{}) *Element  { return tag("label", attrs, children...) }
func Select(attrs Attrs, children ...interface{}) *Element { return tag("select", attrs, children...) }
func Option(attrs Attrs, children ...interface{}) *Element { return tag("option", attrs, children...) }
func Input(attrs Attrs, children ...interface{}) *Element  { return tag("input", attrs, children...) }

func UseState[T any](initialValue T) (func() T, func(interface{})) {
	return runtime.GoUseStateGlobal(initialValue)
}
func UseEffect(effect func() func(), deps ...interface{}) { runtime.GoUseEffectGlobal(effect, deps...) }
func UseMemo[T any](compute func() T, deps ...interface{}) T {
	value := runtime.GoUseMemoGlobalTyped(func() interface{} {
		return compute()
	}, reflect.TypeOf((*T)(nil)).Elem(), deps...)
	return value.(T)
}
func UseId() string                        { return runtime.GoUseIdGlobal() }
func GoUseFunc(fn interface{}) interface{} { return runtime.GoUseFunc(fn) }
func UseFetch(url string, options ...interface{}) (func() runtime.FetchState, func()) {
	return runtime.GoUseFetch(url, options...)
}

func To(root *Element, selector string) { ui.Render(root, selector) }
