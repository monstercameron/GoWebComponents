//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"reflect"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

type Attrs = map[string]interface{}
type Element = runtime.Element
type GoEvent = runtime.GoEvent

func toProps(parseAttrs Attrs) html.Props {
	if parseAttrs == nil {
		return html.Props{}
	}
	return html.Props{Raw: parseAttrs}
}

func normalizeChildren(parseChildren ...interface{}) []ui.Node {
	parseResult := make([]ui.Node, 0, len(parseChildren))
	for _, parseChild := range parseChildren {
		switch parseValue := parseChild.(type) {
		case nil:
			continue
		case ui.Node:
			parseResult = append(parseResult, parseValue)
		case string:
			parseResult = append(parseResult, html.Text(parseValue))
		case fmt.Stringer:
			parseResult = append(parseResult, html.Text(parseValue.String()))
		case []ui.Node:
			parseResult = append(parseResult, parseValue...)
		case []interface{}:
			parseResult = append(parseResult, normalizeChildren(parseValue...)...)
		default:
			parseResult = append(parseResult, html.Text(fmt.Sprint(parseValue)))
		}
	}
	return parseResult
}

func tag(parseName string, parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return html.Tag(parseName, toProps(parseAttrs), normalizeChildren(parseChildren...)...)
}

func Text(parseValue interface{}) *Element { return html.Text(fmt.Sprint(parseValue)) }
func Div(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("div", parseAttrs, parseChildren...)
}
func Form(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("form", parseAttrs, parseChildren...)
}
func Button(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("button", parseAttrs, parseChildren...)
}
func H1(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("h1", parseAttrs, parseChildren...)
}
func H2(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("h2", parseAttrs, parseChildren...)
}
func H3(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("h3", parseAttrs, parseChildren...)
}
func P(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("p", parseAttrs, parseChildren...)
}
func Span(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("span", parseAttrs, parseChildren...)
}
func Label(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("label", parseAttrs, parseChildren...)
}
func Select(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("select", parseAttrs, parseChildren...)
}
func Option(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("option", parseAttrs, parseChildren...)
}
func Input(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("input", parseAttrs, parseChildren...)
}

func UseState[T any](parseInitialValue T) (func() T, func(interface{})) {
	return runtime.GoUseStateGlobal(parseInitialValue)
}
func UseEffect(parseEffect func() func(), parseDeps ...interface{}) {
	runtime.GoUseEffectGlobal(parseEffect, parseDeps...)
}
func UseMemo[T any](parseCompute func() T, parseDeps ...interface{}) T {
	parseValue := runtime.GoUseMemoGlobalTyped(func() interface{} {
		return parseCompute()
	}, reflect.TypeOf((*T)(nil)).Elem(), parseDeps...)
	return parseValue.(T)
}
func UseId() string                             { return runtime.GoUseIdGlobal() }
func GoUseFunc(parseFn interface{}) interface{} { return runtime.GoUseFunc(parseFn) }
func UseFetch(parseUrl string, parseOptions ...interface{}) (func() runtime.FetchState, func()) {
	return runtime.GoUseFetch(parseUrl, parseOptions...)
}

func To(parseRoot *Element, parseSelector string) { ui.Render(parseRoot, parseSelector) }
