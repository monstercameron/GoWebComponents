//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

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

func Text(value interface{}) *Element { return html.Text(fmt.Sprint(value)) }
func CreateElement(component interface{}, props ...interface{}) *Element {
	return ui.CreateElement(component, props...)
}
func Div(attrs Attrs, children ...interface{}) *Element { return tag("div", attrs, children...) }
func Section(attrs Attrs, children ...interface{}) *Element {
	return tag("section", attrs, children...)
}
func Article(attrs Attrs, children ...interface{}) *Element {
	return tag("article", attrs, children...)
}
func Footer(attrs Attrs, children ...interface{}) *Element { return tag("footer", attrs, children...) }
func Nav(attrs Attrs, children ...interface{}) *Element    { return tag("nav", attrs, children...) }
func Form(attrs Attrs, children ...interface{}) *Element   { return tag("form", attrs, children...) }
func A(attrs Attrs, children ...interface{}) *Element      { return tag("a", attrs, children...) }
func Button(attrs Attrs, children ...interface{}) *Element { return tag("button", attrs, children...) }
func Code(attrs Attrs, children ...interface{}) *Element   { return tag("code", attrs, children...) }
func Pre(attrs Attrs, children ...interface{}) *Element    { return tag("pre", attrs, children...) }
func H1(attrs Attrs, children ...interface{}) *Element     { return tag("h1", attrs, children...) }
func H2(attrs Attrs, children ...interface{}) *Element     { return tag("h2", attrs, children...) }
func H3(attrs Attrs, children ...interface{}) *Element     { return tag("h3", attrs, children...) }
func H4(attrs Attrs, children ...interface{}) *Element     { return tag("h4", attrs, children...) }
func P(attrs Attrs, children ...interface{}) *Element      { return tag("p", attrs, children...) }
func Span(attrs Attrs, children ...interface{}) *Element   { return tag("span", attrs, children...) }
func Strong(attrs Attrs, children ...interface{}) *Element { return tag("strong", attrs, children...) }
func Ul(attrs Attrs, children ...interface{}) *Element     { return tag("ul", attrs, children...) }
func Li(attrs Attrs, children ...interface{}) *Element     { return tag("li", attrs, children...) }
func Label(attrs Attrs, children ...interface{}) *Element  { return tag("label", attrs, children...) }
func Select(attrs Attrs, children ...interface{}) *Element { return tag("select", attrs, children...) }
func Option(attrs Attrs, children ...interface{}) *Element { return tag("option", attrs, children...) }
func Textarea(attrs Attrs, children ...interface{}) *Element {
	return tag("textarea", attrs, children...)
}
func Input(attrs Attrs, children ...interface{}) *Element { return tag("input", attrs, children...) }
func Img(attrs Attrs, children ...interface{}) *Element   { return tag("img", attrs, children...) }
func Br(attrs Attrs, children ...interface{}) *Element    { return tag("br", attrs, children...) }

func UseState[T any](initialValue T) (func() T, func(interface{})) {
	return runtime.GoUseStateGlobal(initialValue)
}
func UseEffect(effect func() func(), deps ...interface{}) { runtime.GoUseEffectGlobal(effect, deps...) }
func UseMemo(compute func() interface{}, deps ...interface{}) interface{} {
	return runtime.GoUseMemoGlobal(compute, deps...)
}
func UseCallback(fn interface{}, deps ...interface{}) interface{} {
	return runtime.GoUseCallbackGlobal(fn, deps...)
}
func UseId() string                        { return runtime.GoUseIdGlobal() }
func GoUseFunc(fn interface{}) interface{} { return runtime.GoUseFunc(fn) }
func UseFetch(url string, options ...interface{}) (func() runtime.FetchState, func()) {
	return runtime.GoUseFetch(url, options...)
}
