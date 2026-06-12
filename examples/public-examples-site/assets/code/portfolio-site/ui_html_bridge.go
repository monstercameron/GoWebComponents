//go:build js && wasm

package main

import (
	"fmt"

	gwcfetch "github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type MouseEvent = ui.MouseEvent
type InputEvent = ui.InputEvent
type ChangeEvent = ui.ChangeEvent
type KeyboardEvent = ui.KeyboardEvent
type FormEvent = ui.FormEvent

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
func CreateElement(parseComponent interface{}, parseProps ...interface{}) *Element {
	return ui.CreateElement(parseComponent, parseProps...)
}
func Div(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("div", parseAttrs, parseChildren...)
}
func Section(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("section", parseAttrs, parseChildren...)
}
func Article(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("article", parseAttrs, parseChildren...)
}
func Footer(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("footer", parseAttrs, parseChildren...)
}
func Nav(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("nav", parseAttrs, parseChildren...)
}
func Form(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("form", parseAttrs, parseChildren...)
}
func A(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("a", parseAttrs, parseChildren...)
}
func Button(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("button", parseAttrs, parseChildren...)
}
func Code(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("code", parseAttrs, parseChildren...)
}
func Pre(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("pre", parseAttrs, parseChildren...)
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
func H4(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("h4", parseAttrs, parseChildren...)
}
func P(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("p", parseAttrs, parseChildren...)
}
func Span(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("span", parseAttrs, parseChildren...)
}
func Strong(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("strong", parseAttrs, parseChildren...)
}
func Ul(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("ul", parseAttrs, parseChildren...)
}
func Li(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("li", parseAttrs, parseChildren...)
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
func Textarea(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("textarea", parseAttrs, parseChildren...)
}
func Input(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("input", parseAttrs, parseChildren...)
}
func Iframe(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("iframe", parseAttrs, parseChildren...)
}
func Img(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("img", parseAttrs, parseChildren...)
}
func Br(parseAttrs Attrs, parseChildren ...interface{}) *Element {
	return tag("br", parseAttrs, parseChildren...)
}

func UseState[T any](parseInitialValue T) (func() T, func(interface{})) {
	parseState := ui.UseState(parseInitialValue)
	return parseState.Get, func(parseValue interface{}) {
		if parseUpdater, parseOk := parseValue.(func(T) T); parseOk {
			parseState.Update(parseUpdater)
			return
		}
		parseCast, parseOk2 := parseValue.(T)
		if parseOk2 {
			parseState.Set(parseCast)
		}
	}
}
func UseEffect(parseEffect func() func(), parseDeps ...interface{}) {
	ui.UseEffect(parseEffect, parseDeps...)
}
func UseMemo(parseCompute func() interface{}, parseDeps ...interface{}) interface{} {
	return ui.UseMemo(parseCompute, parseDeps...)
}
func UseCallback(parseFn interface{}, parseDeps ...interface{}) interface{} {
	return ui.UseCallback(parseFn, parseDeps...)
}
func UseId() string                            { return ui.UseId() }
func UseEvent(parseFn interface{}) interface{} { return ui.UseEvent(parseFn).Value() }
func UseFetch(parseUrl string, parseOptions ...interface{}) (func() gwcfetch.State, func()) {
	parseFetchOptions := make([]gwcfetch.Options, 0, len(parseOptions))
	for _, parseOption := range parseOptions {
		parseCast, parseOk := parseOption.(gwcfetch.Options)
		if parseOk {
			parseFetchOptions = append(parseFetchOptions, parseCast)
		}
	}
	parseResource := gwcfetch.UseFetch(parseUrl, parseFetchOptions...)
	return parseResource.Get, parseResource.Refetch
}
