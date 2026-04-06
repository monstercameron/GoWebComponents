package html

import (
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

// Props contains the common HTML attributes and event handlers supported by the typed builders.
type Props struct {
	ID           string
	Class        string
	Key          string
	Slot         string
	Title        string
	Type         string
	Name         string
	Value        string
	Placeholder  string
	Accept       string
	Href         string
	Src          string
	Alt          string
	For          string
	Role         string
	Target       string
	Rel          string
	As           string
	Action       string
	Method       string
	EncType      string
	AutoComplete string
	Min          string
	Max          string
	Step         string

	Rows     int
	Cols     int
	TabIndex int

	Checked   bool
	Disabled  bool
	Selected  bool
	Required  bool
	ReadOnly  bool
	Hidden    bool
	Multiple  bool
	AutoFocus bool

	Style map[string]string
	Data  map[string]string
	Aria  map[string]string
	Raw   map[string]interface{}

	OnClick     ui.Handler
	OnInput     ui.Handler
	OnChange    ui.Handler
	OnSubmit    ui.Handler
	OnKeyDown   ui.Handler
	OnKeyUp     ui.Handler
	OnMouseUp   ui.Handler
	OnMouseDown ui.Handler
	OnFocus     ui.Handler
	OnBlur      ui.Handler
	OnScroll    ui.Handler
}

// CustomElementProps makes attribute-versus-property intent explicit for
// browser-defined custom elements and web components.
type CustomElementProps struct {
	Props      Props
	Attributes map[string]string
	Presence   map[string]bool
	Properties map[string]interface{}
}

const customElementPropertyPrefix = "__gwc_prop__:"
const parallelRegionClickSlotDataKey = "gwc-parallel-click-slot"

// Tag creates a node for an arbitrary HTML tag name.
func Tag(parseName string, parseProps Props, parseChildren ...ui.Node) ui.Node {
	return runtime.CreateElement(parseName, toRuntimeProps(parseProps), toInterfaces(parseChildren)...)
}

// Link creates a typed link element.
func Link(parseProps Props) ui.Node {
	return Tag("link", parseProps)
}

// CustomElement creates a browser-defined custom element with explicit
// attribute and property channels.
func CustomElement(parseName string, parseProps CustomElementProps, parseChildren ...ui.Node) ui.Node {
	parseValues := toRuntimeProps(parseProps.Props)
	parseCount := len(parseProps.Attributes) + len(parseProps.Presence) + len(parseProps.Properties)
	if parseCount == 0 {
		return runtime.CreateElement(parseName, parseValues, toInterfaces(parseChildren)...)
	}
	if parseValues == nil {
		parseValues = make(map[string]interface{}, parseCount)
	}
	for parseKey, parseValue := range parseProps.Attributes {
		parseValues[parseKey] = parseValue
	}
	for parseKey2, parseEnabled := range parseProps.Presence {
		if parseEnabled {
			parseValues[parseKey2] = ""
		}
	}
	for parseKey3, parseValue2 := range parseProps.Properties {
		parseValues[customElementPropertyPrefix+parseKey3] = parseValue2
	}
	return runtime.CreateElement(parseName, parseValues, toInterfaces(parseChildren)...)
}

// Fragment groups children without introducing an extra host element.
func Fragment(parseChildren ...ui.Node) ui.Node {
	return ui.Fragment(parseChildren...)
}

// A creates an anchor element.
func A(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("a", parseProps, parseChildren...)
}

// Article creates an article element.
func Article(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("article", parseProps, parseChildren...)
}

// Aside creates an aside element.
func Aside(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("aside", parseProps, parseChildren...)
}

// Blockquote creates a blockquote element.
func Blockquote(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("blockquote", parseProps, parseChildren...)
}

// Br creates a br line-break element.
func Br(parseProps Props) ui.Node {
	return Tag("br", parseProps)
}

// Button creates a button element.
func Button(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("button", parseProps, parseChildren...)
}

// Body creates a body element.
func Body(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("body", parseProps, parseChildren...)
}

// Code creates a code element.
func Code(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("code", parseProps, parseChildren...)
}

// Details creates a details disclosure element.
func Details(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("details", parseProps, parseChildren...)
}

// Dialog creates a dialog element.
func Dialog(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("dialog", parseProps, parseChildren...)
}

// Div creates a div element.
func Div(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("div", parseProps, parseChildren...)
}

// Em creates an em emphasis element.
func Em(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("em", parseProps, parseChildren...)
}

// Fieldset creates a fieldset element.
func Fieldset(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("fieldset", parseProps, parseChildren...)
}

// Footer creates a footer element.
func Footer(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("footer", parseProps, parseChildren...)
}

// Form creates a form element.
func Form(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("form", parseProps, parseChildren...)
}

// H1 creates an h1 heading element.
func H1(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("h1", parseProps, parseChildren...)
}

// H2 creates an h2 heading element.
func H2(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("h2", parseProps, parseChildren...)
}

// H3 creates an h3 heading element.
func H3(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("h3", parseProps, parseChildren...)
}

// H4 creates an h4 heading element.
func H4(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("h4", parseProps, parseChildren...)
}

// H5 creates an h5 heading element.
func H5(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("h5", parseProps, parseChildren...)
}

// H6 creates an h6 heading element.
func H6(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("h6", parseProps, parseChildren...)
}

// Head creates a head element.
func Head(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("head", parseProps, parseChildren...)
}

// Header creates a header element.
func Header(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("header", parseProps, parseChildren...)
}

// Html creates an html root element.
func Html(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("html", parseProps, parseChildren...)
}

// Hr creates an hr horizontal rule element.
func Hr(parseProps Props) ui.Node {
	return Tag("hr", parseProps)
}

// Img creates an img image element.
func Img(parseProps Props) ui.Node {
	return Tag("img", parseProps)
}

// Input creates an input element.
func Input(parseProps Props) ui.Node {
	return Tag("input", parseProps)
}

// HiddenInput creates a hidden input element with the given name and value.
func HiddenInput(parseName string, parseValue string) ui.Node {
	return Input(Props{Type: "hidden", Name: parseName, Value: parseValue})
}

// Label creates a label element.
func Label(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("label", parseProps, parseChildren...)
}

// Legend creates a legend element.
func Legend(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("legend", parseProps, parseChildren...)
}

// Li creates an li list-item element.
func Li(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("li", parseProps, parseChildren...)
}

// Main creates a main element.
func Main(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("main", parseProps, parseChildren...)
}

// Mark creates a mark element.
func Mark(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("mark", parseProps, parseChildren...)
}

// Meta creates a meta element.
func Meta(parseProps Props) ui.Node {
	return Tag("meta", parseProps)
}

// Nav creates a nav element.
func Nav(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("nav", parseProps, parseChildren...)
}

// NoScript creates a noscript element.
func NoScript(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("noscript", parseProps, parseChildren...)
}

// Preload creates a link[rel=preload] element for the given href and as type.
func Preload(parseHref, parseAs string) ui.Node {
	return Link(Props{Rel: "preload", Href: parseHref, As: parseAs})
}

// ModulePreload creates a link[rel=modulepreload] element for module scripts.
func ModulePreload(parseHref string) ui.Node {
	return Link(Props{Rel: "modulepreload", Href: parseHref, As: "script"})
}

// Prefetch creates a link[rel=prefetch] element for the given href.
func Prefetch(parseHref string) ui.Node {
	return Link(Props{Rel: "prefetch", Href: parseHref})
}

// Preconnect creates a link[rel=preconnect] element for the given href.
func Preconnect(parseHref string) ui.Node {
	return Link(Props{Rel: "preconnect", Href: parseHref})
}

// DNSPrefetch creates a link[rel=dns-prefetch] element for the given href.
func DNSPrefetch(parseHref string) ui.Node {
	return Link(Props{Rel: "dns-prefetch", Href: parseHref})
}

// Option creates an option element.
func Option(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("option", parseProps, parseChildren...)
}

// P creates a p paragraph element.
func P(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("p", parseProps, parseChildren...)
}

// Pre creates a pre preformatted element.
func Pre(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("pre", parseProps, parseChildren...)
}

// Script creates a script element.
func Script(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("script", parseProps, parseChildren...)
}

// Section creates a section element.
func Section(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("section", parseProps, parseChildren...)
}

// Select creates a select element.
func Select(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("select", parseProps, parseChildren...)
}

// Small creates a small element.
func Small(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("small", parseProps, parseChildren...)
}

// Span creates a span element.
func Span(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("span", parseProps, parseChildren...)
}

// Strong creates a strong element.
func Strong(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("strong", parseProps, parseChildren...)
}

// Summary creates a summary element.
func Summary(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("summary", parseProps, parseChildren...)
}

// Table creates a table element.
func Table(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("table", parseProps, parseChildren...)
}

// Tbody creates a tbody element.
func Tbody(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("tbody", parseProps, parseChildren...)
}

// Td creates a td table-data element.
func Td(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("td", parseProps, parseChildren...)
}

// Th creates a th table-header element.
func Th(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("th", parseProps, parseChildren...)
}

// Thead creates a thead element.
func Thead(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("thead", parseProps, parseChildren...)
}

// Textarea creates a textarea element.
func Textarea(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("textarea", parseProps, parseChildren...)
}

// Time creates a time element.
func Time(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("time", parseProps, parseChildren...)
}

// Tr creates a tr table-row element.
func Tr(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("tr", parseProps, parseChildren...)
}

// Ul creates a ul unordered-list element.
func Ul(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("ul", parseProps, parseChildren...)
}

// toRuntimeProps is a core package helper.
func toRuntimeProps(parseProps Props) map[string]interface{} {
	parseOnClick := parseProps.OnClick.Value()
	parseOnInput := parseProps.OnInput.Value()
	parseOnChange := parseProps.OnChange.Value()
	parseOnSubmit := parseProps.OnSubmit.Value()
	parseOnKeyDown := parseProps.OnKeyDown.Value()
	parseOnKeyUp := parseProps.OnKeyUp.Value()
	parseOnMouseUp := parseProps.OnMouseUp.Value()
	parseOnMouseDown := parseProps.OnMouseDown.Value()
	parseOnFocus := parseProps.OnFocus.Value()
	parseOnBlur := parseProps.OnBlur.Value()
	parseOnScroll := parseProps.OnScroll.Value()

	parseCount := len(parseProps.Data) + len(parseProps.Aria) + len(parseProps.Raw)
	if parseProps.ID != "" {
		parseCount++
	}
	if parseProps.Class != "" {
		parseCount++
	}
	if parseProps.Key != "" {
		parseCount++
	}
	if parseProps.Slot != "" {
		parseCount++
	}
	if parseProps.Title != "" {
		parseCount++
	}
	if parseProps.Type != "" {
		parseCount++
	}
	if parseProps.Name != "" {
		parseCount++
	}
	if parseProps.Value != "" {
		parseCount++
	}
	if parseProps.Placeholder != "" {
		parseCount++
	}
	if parseProps.Accept != "" {
		parseCount++
	}
	if parseProps.Href != "" {
		parseCount++
	}
	if parseProps.Src != "" {
		parseCount++
	}
	if parseProps.Alt != "" {
		parseCount++
	}
	if parseProps.For != "" {
		parseCount++
	}
	if parseProps.Role != "" {
		parseCount++
	}
	if parseProps.Target != "" {
		parseCount++
	}
	if parseProps.Rel != "" {
		parseCount++
	}
	if parseProps.As != "" {
		parseCount++
	}
	if parseProps.Action != "" {
		parseCount++
	}
	if parseProps.Method != "" {
		parseCount++
	}
	if parseProps.EncType != "" {
		parseCount++
	}
	if parseProps.AutoComplete != "" {
		parseCount++
	}
	if parseProps.Min != "" {
		parseCount++
	}
	if parseProps.Max != "" {
		parseCount++
	}
	if parseProps.Step != "" {
		parseCount++
	}
	if parseProps.Rows != 0 {
		parseCount++
	}
	if parseProps.Cols != 0 {
		parseCount++
	}
	if parseProps.TabIndex != 0 {
		parseCount++
	}
	if parseProps.Checked {
		parseCount++
	}
	if parseProps.Disabled {
		parseCount++
	}
	if parseProps.Selected {
		parseCount++
	}
	if parseProps.Required {
		parseCount++
	}
	if parseProps.ReadOnly {
		parseCount++
	}
	if parseProps.Hidden {
		parseCount++
	}
	if parseProps.Multiple {
		parseCount++
	}
	if parseProps.AutoFocus {
		parseCount++
	}
	if parseProps.Style != nil {
		parseCount++
	}
	if parseOnClick != nil {
		parseCount++
	}
	if parseOnInput != nil {
		parseCount++
	}
	if parseOnChange != nil {
		parseCount++
	}
	if parseOnSubmit != nil {
		parseCount++
	}
	if parseOnKeyDown != nil {
		parseCount++
	}
	if parseOnKeyUp != nil {
		parseCount++
	}
	if parseOnMouseUp != nil {
		parseCount++
	}
	if parseOnMouseDown != nil {
		parseCount++
	}
	if parseOnFocus != nil {
		parseCount++
	}
	if parseOnBlur != nil {
		parseCount++
	}
	if parseOnScroll != nil {
		parseCount++
	}

	if parseCount == 0 {
		return nil
	}

	parseValues := make(map[string]interface{}, parseCount)
	if parseProps.ID != "" {
		parseValues["id"] = parseProps.ID
	}
	if parseProps.Class != "" {
		parseValues["class"] = parseProps.Class
	}
	if parseProps.Key != "" {
		parseValues["key"] = parseProps.Key
	}
	if parseProps.Slot != "" {
		parseValues["slot"] = parseProps.Slot
	}
	if parseProps.Title != "" {
		parseValues["title"] = parseProps.Title
	}
	if parseProps.Type != "" {
		parseValues["type"] = parseProps.Type
	}
	if parseProps.Name != "" {
		parseValues["name"] = parseProps.Name
	}
	if parseProps.Value != "" {
		parseValues["value"] = parseProps.Value
	}
	if parseProps.Placeholder != "" {
		parseValues["placeholder"] = parseProps.Placeholder
	}
	if parseProps.Accept != "" {
		parseValues["accept"] = parseProps.Accept
	}
	if parseProps.Href != "" {
		parseValues["href"] = parseProps.Href
	}
	if parseProps.Src != "" {
		parseValues["src"] = parseProps.Src
	}
	if parseProps.Alt != "" {
		parseValues["alt"] = parseProps.Alt
	}
	if parseProps.For != "" {
		parseValues["htmlFor"] = parseProps.For
	}
	if parseProps.Role != "" {
		parseValues["role"] = parseProps.Role
	}
	if parseProps.Target != "" {
		parseValues["target"] = parseProps.Target
	}
	if parseProps.Rel != "" {
		parseValues["rel"] = parseProps.Rel
	}
	if parseProps.As != "" {
		parseValues["as"] = parseProps.As
	}
	if parseProps.Action != "" {
		parseValues["action"] = parseProps.Action
	}
	if parseProps.Method != "" {
		parseValues["method"] = parseProps.Method
	}
	if parseProps.EncType != "" {
		parseValues["enctype"] = parseProps.EncType
	}
	if parseProps.AutoComplete != "" {
		parseValues["autocomplete"] = parseProps.AutoComplete
	}
	if parseProps.Min != "" {
		parseValues["min"] = parseProps.Min
	}
	if parseProps.Max != "" {
		parseValues["max"] = parseProps.Max
	}
	if parseProps.Step != "" {
		parseValues["step"] = parseProps.Step
	}
	if parseProps.Rows != 0 {
		parseValues["rows"] = parseProps.Rows
	}
	if parseProps.Cols != 0 {
		parseValues["cols"] = parseProps.Cols
	}
	if parseProps.TabIndex != 0 {
		parseValues["tabIndex"] = parseProps.TabIndex
	}
	if parseProps.Checked {
		parseValues["checked"] = true
	}
	if parseProps.Disabled {
		parseValues["disabled"] = true
	}
	if parseProps.Selected {
		parseValues["selected"] = true
	}
	if parseProps.Required {
		parseValues["required"] = true
	}
	if parseProps.ReadOnly {
		parseValues["readOnly"] = true
	}
	if parseProps.Hidden {
		parseValues["hidden"] = true
	}
	if parseProps.Multiple {
		parseValues["multiple"] = true
	}
	if parseProps.AutoFocus {
		parseValues["autofocus"] = true
	}
	if parseProps.Style != nil {
		parseValues["style"] = parseProps.Style
	}

	if len(parseProps.Data) != 0 {
		for parseKey, parseValue := range parseProps.Data {
			parseValues["data-"+parseKey] = parseValue
		}
	}
	if len(parseProps.Aria) != 0 {
		for parseKey2, parseValue2 := range parseProps.Aria {
			parseValues["aria-"+parseKey2] = parseValue2
		}
	}
	if parseOnClick != nil {
		parseValues["onclick"] = parseOnClick
	}
	if parseOnInput != nil {
		parseValues["oninput"] = parseOnInput
	}
	if parseOnChange != nil {
		parseValues["onchange"] = parseOnChange
	}
	if parseOnSubmit != nil {
		parseValues["onsubmit"] = parseOnSubmit
	}
	if parseOnKeyDown != nil {
		parseValues["onkeydown"] = parseOnKeyDown
	}
	if parseOnKeyUp != nil {
		parseValues["onkeyup"] = parseOnKeyUp
	}
	if parseOnMouseUp != nil {
		parseValues["onmouseup"] = parseOnMouseUp
	}
	if parseOnMouseDown != nil {
		parseValues["onmousedown"] = parseOnMouseDown
	}
	if parseOnFocus != nil {
		parseValues["onfocus"] = parseOnFocus
	}
	if parseOnBlur != nil {
		parseValues["onblur"] = parseOnBlur
	}
	if parseOnScroll != nil {
		parseValues["onscroll"] = parseOnScroll
	}
	if len(parseProps.Raw) != 0 {
		for parseKey3, parseValue3 := range parseProps.Raw {
			parseValues[parseKey3] = parseValue3
		}
	}

	return parseValues
}

// toInterfaces is a core package helper.
func toInterfaces(parseChildren []ui.Node) []interface{} {
	if len(parseChildren) == 0 {
		return nil
	}

	parseValues := make([]interface{}, 0, len(parseChildren))
	for _, parseChild := range parseChildren {
		parseValues = append(parseValues, parseChild)
	}

	return parseValues
}
