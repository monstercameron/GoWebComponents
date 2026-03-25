package runtime

// HTML element constructors
// These create virtual DOM elements for standard HTML tags

// Document Structure Elements

// Html creates an html element.
func Html(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("html", parseProps, parseChildren...)
}

// Head creates a head element.
func Head(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("head", parseProps, parseChildren...)
}

// Body creates a body element.
func Body(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("body", parseProps, parseChildren...)
}

// Title creates a title element.
func Title(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("title", parseProps, parseChildren...)
}

// Meta creates a meta element.
func Meta(parseProps map[string]interface{}) *Element {
	return CreateElement("meta", parseProps)
}

// Link creates a link element.
func Link(parseProps map[string]interface{}) *Element {
	return CreateElement("link", parseProps)
}

// Style creates a style element.
func Style(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("style", parseProps, parseChildren...)
}

// Script creates a script element.
func Script(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("script", parseProps, parseChildren...)
}

// Semantic Structure Elements

// Header creates a header element.
func Header(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("header", parseProps, parseChildren...)
}

// Nav creates a nav element.
func Nav(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("nav", parseProps, parseChildren...)
}

// Main creates a main element.
func Main(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("main", parseProps, parseChildren...)
}

// Footer creates a footer element.
func Footer(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("footer", parseProps, parseChildren...)
}

// Section creates a section element.
func Section(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("section", parseProps, parseChildren...)
}

// Article creates an article element.
func Article(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("article", parseProps, parseChildren...)
}

// Aside creates an aside element.
func Aside(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("aside", parseProps, parseChildren...)
}

// Content Grouping

// Div creates a div element.
func Div(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("div", parseProps, parseChildren...)
}

// P creates a p element.
func P(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("p", parseProps, parseChildren...)
}

// Span creates a span element.
func Span(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("span", parseProps, parseChildren...)
}

// Pre creates a pre element.
func Pre(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("pre", parseProps, parseChildren...)
}

// Blockquote creates a blockquote element.
func Blockquote(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("blockquote", parseProps, parseChildren...)
}

// Hr creates an hr element.
func Hr(parseProps map[string]interface{}) *Element {
	return CreateElement("hr", parseProps)
}

// Text Content

// H1 creates an h1 element.
func H1(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("h1", parseProps, parseChildren...)
}

// H2 creates an h2 element.
func H2(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("h2", parseProps, parseChildren...)
}

// H3 creates an h3 element.
func H3(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("h3", parseProps, parseChildren...)
}

// H4 creates an h4 element.
func H4(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("h4", parseProps, parseChildren...)
}

// H5 creates an h5 element.
func H5(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("h5", parseProps, parseChildren...)
}

// H6 creates an h6 element.
func H6(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("h6", parseProps, parseChildren...)
}

// Lists

// Ul creates a ul element.
func Ul(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("ul", parseProps, parseChildren...)
}

// Ol creates an ol element.
func Ol(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("ol", parseProps, parseChildren...)
}

// Li creates a li element.
func Li(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("li", parseProps, parseChildren...)
}

// Dl creates a dl element.
func Dl(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("dl", parseProps, parseChildren...)
}

// Dt creates a dt element.
func Dt(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("dt", parseProps, parseChildren...)
}

// Dd creates a dd element.
func Dd(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("dd", parseProps, parseChildren...)
}

// Inline Text

// A creates an a element.
func A(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("a", parseProps, parseChildren...)
}

// Strong creates a strong element.
func Strong(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("strong", parseProps, parseChildren...)
}

// Em creates an em element.
func Em(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("em", parseProps, parseChildren...)
}

// Code creates a code element.
func Code(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("code", parseProps, parseChildren...)
}

// Small creates a small element.
func Small(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("small", parseProps, parseChildren...)
}

// Mark creates a mark element.
func Mark(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("mark", parseProps, parseChildren...)
}

// Del creates a del element.
func Del(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("del", parseProps, parseChildren...)
}

// Ins creates an ins element.
func Ins(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("ins", parseProps, parseChildren...)
}

// Sub creates a sub element.
func Sub(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("sub", parseProps, parseChildren...)
}

// Sup creates a sup element.
func Sup(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("sup", parseProps, parseChildren...)
}

// Br creates a br element.
func Br(parseProps map[string]interface{}) *Element {
	return CreateElement("br", parseProps)
}

// Forms

// Form creates a form element.
func Form(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("form", parseProps, parseChildren...)
}

// Label creates a label element.
func Label(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("label", parseProps, parseChildren...)
}

// Input creates an input element.
func Input(parseProps map[string]interface{}) *Element {
	return CreateElement("input", parseProps)
}

// Button creates a button element.
func Button(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("button", parseProps, parseChildren...)
}

// Select creates a select element.
func Select(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("select", parseProps, parseChildren...)
}

// Option creates an option element.
func Option(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("option", parseProps, parseChildren...)
}

// Textarea creates a textarea element.
func Textarea(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("textarea", parseProps, parseChildren...)
}

// Fieldset creates a fieldset element.
func Fieldset(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("fieldset", parseProps, parseChildren...)
}

// Legend creates a legend element.
func Legend(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("legend", parseProps, parseChildren...)
}

// Tables

// Table creates a table element.
func Table(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("table", parseProps, parseChildren...)
}

// Thead creates a thead element.
func Thead(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("thead", parseProps, parseChildren...)
}

// Tbody creates a tbody element.
func Tbody(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("tbody", parseProps, parseChildren...)
}

// Tfoot creates a tfoot element.
func Tfoot(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("tfoot", parseProps, parseChildren...)
}

// Tr creates a tr element.
func Tr(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("tr", parseProps, parseChildren...)
}

// Th creates a th element.
func Th(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("th", parseProps, parseChildren...)
}

// Td creates a td element.
func Td(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("td", parseProps, parseChildren...)
}

// Caption creates a caption element.
func Caption(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("caption", parseProps, parseChildren...)
}

// Colgroup creates a colgroup element.
func Colgroup(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("colgroup", parseProps, parseChildren...)
}

// Col creates a col element.
func Col(parseProps map[string]interface{}) *Element {
	return CreateElement("col", parseProps)
}

// Media

// Img creates an img element.
func Img(parseProps map[string]interface{}) *Element {
	return CreateElement("img", parseProps)
}

// Video creates a video element.
func Video(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("video", parseProps, parseChildren...)
}

// Audio creates an audio element.
func Audio(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("audio", parseProps, parseChildren...)
}

// Source creates a source element.
func Source(parseProps map[string]interface{}) *Element {
	return CreateElement("source", parseProps)
}

// Picture creates a picture element.
func Picture(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("picture", parseProps, parseChildren...)
}

// Canvas creates a canvas element.
func Canvas(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("canvas", parseProps, parseChildren...)
}

// Svg creates an svg element.
func Svg(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("svg", parseProps, parseChildren...)
}

// Path creates a path element.
func Path(parseProps map[string]interface{}) *Element {
	return CreateElement("path", parseProps)
}

// Circle creates a circle element.
func Circle(parseProps map[string]interface{}) *Element {
	return CreateElement("circle", parseProps)
}

// Rect creates a rect element.
func Rect(parseProps map[string]interface{}) *Element {
	return CreateElement("rect", parseProps)
}

// Line creates a line element.
func Line(parseProps map[string]interface{}) *Element {
	return CreateElement("line", parseProps)
}

// Polygon creates a polygon element.
func Polygon(parseProps map[string]interface{}) *Element {
	return CreateElement("polygon", parseProps)
}

// G creates a g (SVG group) element.
func G(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("g", parseProps, parseChildren...)
}

// Interactive

// Details creates a details element.
func Details(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("details", parseProps, parseChildren...)
}

// Summary creates a summary element.
func Summary(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("summary", parseProps, parseChildren...)
}

// Dialog creates a dialog element.
func Dialog(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("dialog", parseProps, parseChildren...)
}

// Menu creates a menu element.
func Menu(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("menu", parseProps, parseChildren...)
}

// Embedded Content

// Iframe creates an iframe element.
func Iframe(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("iframe", parseProps, parseChildren...)
}

// Embed creates an embed element.
func Embed(parseProps map[string]interface{}) *Element {
	return CreateElement("embed", parseProps)
}

// Object creates an object element.
func Object(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("object", parseProps, parseChildren...)
}

// Param creates a param element.
func Param(parseProps map[string]interface{}) *Element {
	return CreateElement("param", parseProps)
}

// Additional Elements

// Time creates a time element.
func Time(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("time", parseProps, parseChildren...)
}

// Progress creates a progress element.
func Progress(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("progress", parseProps, parseChildren...)
}

// Meter creates a meter element.
func Meter(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("meter", parseProps, parseChildren...)
}

// Output creates an output element.
func Output(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("output", parseProps, parseChildren...)
}

// Data creates a data element.
func Data(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("data", parseProps, parseChildren...)
}

// Wbr creates a wbr element.
func Wbr(parseProps map[string]interface{}) *Element {
	return CreateElement("wbr", parseProps)
}

// Abbr creates an abbr element.
func Abbr(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("abbr", parseProps, parseChildren...)
}

// Address creates an address element.
func Address(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("address", parseProps, parseChildren...)
}

// Cite creates a cite element.
func Cite(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("cite", parseProps, parseChildren...)
}

// Kbd creates a kbd element.
func Kbd(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("kbd", parseProps, parseChildren...)
}

// Samp creates a samp element.
func Samp(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("samp", parseProps, parseChildren...)
}

// Var creates a var element.
func Var(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("var", parseProps, parseChildren...)
}

// Q creates a q element.
func Q(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("q", parseProps, parseChildren...)
}

// Dfn creates a dfn element.
func Dfn(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("dfn", parseProps, parseChildren...)
}

// B creates a b element.
func B(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("b", parseProps, parseChildren...)
}

// I creates an i element.
func I(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("i", parseProps, parseChildren...)
}

// U creates a u element.
func U(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("u", parseProps, parseChildren...)
}

// S creates an s element.
func S(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("s", parseProps, parseChildren...)
}

// Bdi creates a bdi element.
func Bdi(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("bdi", parseProps, parseChildren...)
}

// Bdo creates a bdo element.
func Bdo(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("bdo", parseProps, parseChildren...)
}

// Ruby creates a ruby element.
func Ruby(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("ruby", parseProps, parseChildren...)
}

// Rt creates an rt element.
func Rt(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("rt", parseProps, parseChildren...)
}

// Rp creates an rp element.
func Rp(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("rp", parseProps, parseChildren...)
}

// Figure creates a figure element.
func Figure(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("figure", parseProps, parseChildren...)
}

// Figcaption creates a figcaption element.
func Figcaption(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("figcaption", parseProps, parseChildren...)
}

// Additional missing elements

// Track creates a track element.
func Track(parseProps map[string]interface{}) *Element {
	return CreateElement("track", parseProps)
}

// Optgroup creates an optgroup element.
func Optgroup(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("optgroup", parseProps, parseChildren...)
}

// Datalist creates a datalist element.
func Datalist(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("datalist", parseProps, parseChildren...)
}

// Hgroup creates an hgroup element.
func Hgroup(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("hgroup", parseProps, parseChildren...)
}

// Portal creates a portal element.
func Portal(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("portal", parseProps, parseChildren...)
}

// Template creates a template element.
func Template(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("template", parseProps, parseChildren...)
}

// Slot creates a slot element.
func Slot(parseProps map[string]interface{}, parseChildren ...interface{}) *Element {
	return CreateElement("slot", parseProps, parseChildren...)
}

// Helper functions for common patterns

// ClassProps returns a props map with the given CSS class.
func ClassProps(parseClass string) map[string]interface{} {
	return map[string]interface{}{"class": parseClass}
}

// IdProps returns a props map with the given element ID.
func IdProps(parseId string) map[string]interface{} {
	return map[string]interface{}{"id": parseId}
}

// HrefProps returns a props map with the given href attribute.
func HrefProps(parseHref string) map[string]interface{} {
	return map[string]interface{}{"href": parseHref}
}

// SrcProps returns a props map with the given src attribute.
func SrcProps(parseSrc string) map[string]interface{} {
	return map[string]interface{}{"src": parseSrc}
}

// StyleProps returns a props map with the given inline style string.
func StyleProps(parseStyle string) map[string]interface{} {
	return map[string]interface{}{"style": parseStyle}
}

// TypeProps returns a props map with the given type attribute.
func TypeProps(parseTyp string) map[string]interface{} {
	return map[string]interface{}{"type": parseTyp}
}

// ValueProps returns a props map with the given value.
func ValueProps(parseValue interface{}) map[string]interface{} {
	return map[string]interface{}{"value": parseValue}
}

// PlaceholderProps returns a props map with the given placeholder text.
func PlaceholderProps(parsePlaceholder string) map[string]interface{} {
	return map[string]interface{}{"placeholder": parsePlaceholder}
}

// InputTypeProps returns a props map with the given input type.
func InputTypeProps(parseTyp string) map[string]interface{} {
	return map[string]interface{}{"type": parseTyp}
}

// ClassIdProps returns a props map with the given class and id.
func ClassIdProps(parseClass, parseId string) map[string]interface{} {
	return map[string]interface{}{"class": parseClass, "id": parseId}
}

// EmptyProps returns an empty props map.
func EmptyProps() map[string]interface{} {
	return map[string]interface{}{}
}

// Component helpers

// WithComponents creates an element with the given tag and renders component refs as children.
func WithComponents(parseTagName string, parseProps map[string]interface{}, parseComponentRefs ...func(map[string]interface{}) *Element) *Element {
	parseChildren := componentRefsToChildren(parseComponentRefs)
	return CreateElement(parseTagName, parseProps, parseChildren...)
}

// DivWithComponents creates a div element and renders component refs as children.
func DivWithComponents(parseProps map[string]interface{}, parseComponentRefs ...func(map[string]interface{}) *Element) *Element {
	parseChildren := componentRefsToChildren(parseComponentRefs)
	return Div(parseProps, parseChildren...)
}

// SectionWithComponents creates a section element and renders component refs as children.
func SectionWithComponents(parseProps map[string]interface{}, parseComponentRefs ...func(map[string]interface{}) *Element) *Element {
	parseChildren := componentRefsToChildren(parseComponentRefs)
	return Section(parseProps, parseChildren...)
}

// MainWithComponents creates a main element and renders component refs as children.
func MainWithComponents(parseProps map[string]interface{}, parseComponentRefs ...func(map[string]interface{}) *Element) *Element {
	parseChildren := componentRefsToChildren(parseComponentRefs)
	return Main(parseProps, parseChildren...)
}

// componentRefsToChildren is a core package helper.
func componentRefsToChildren(parseComponentRefs []func(map[string]interface{}) *Element) []interface{} {
	if len(parseComponentRefs) == 0 {
		return emptyChildren
	}

	parseChildren := make([]interface{}, len(parseComponentRefs))
	for parseI, parseRef := range parseComponentRefs {
		parseChildren[parseI] = &Element{
			Type:     parseRef,
			Children: emptyChildren,
		}
	}
	return parseChildren
}
