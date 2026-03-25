package runtime

// HTML element constructors
// These create virtual DOM elements for standard HTML tags

// Document Structure Elements

// Html creates an html element.
func Html(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("html", props, children...)
}

// Head creates a head element.
func Head(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("head", props, children...)
}

// Body creates a body element.
func Body(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("body", props, children...)
}

// Title creates a title element.
func Title(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("title", props, children...)
}

// Meta creates a meta element.
func Meta(props map[string]interface{}) *Element {
	return CreateElement("meta", props)
}

// Link creates a link element.
func Link(props map[string]interface{}) *Element {
	return CreateElement("link", props)
}

// Style creates a style element.
func Style(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("style", props, children...)
}

// Script creates a script element.
func Script(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("script", props, children...)
}

// Semantic Structure Elements

// Header creates a header element.
func Header(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("header", props, children...)
}

// Nav creates a nav element.
func Nav(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("nav", props, children...)
}

// Main creates a main element.
func Main(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("main", props, children...)
}

// Footer creates a footer element.
func Footer(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("footer", props, children...)
}

// Section creates a section element.
func Section(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("section", props, children...)
}

// Article creates an article element.
func Article(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("article", props, children...)
}

// Aside creates an aside element.
func Aside(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("aside", props, children...)
}

// Content Grouping

// Div creates a div element.
func Div(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("div", props, children...)
}

// P creates a p element.
func P(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("p", props, children...)
}

// Span creates a span element.
func Span(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("span", props, children...)
}

// Pre creates a pre element.
func Pre(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("pre", props, children...)
}

// Blockquote creates a blockquote element.
func Blockquote(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("blockquote", props, children...)
}

// Hr creates an hr element.
func Hr(props map[string]interface{}) *Element {
	return CreateElement("hr", props)
}

// Text Content

// H1 creates an h1 element.
func H1(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("h1", props, children...)
}

// H2 creates an h2 element.
func H2(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("h2", props, children...)
}

// H3 creates an h3 element.
func H3(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("h3", props, children...)
}

// H4 creates an h4 element.
func H4(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("h4", props, children...)
}

// H5 creates an h5 element.
func H5(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("h5", props, children...)
}

// H6 creates an h6 element.
func H6(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("h6", props, children...)
}

// Lists

// Ul creates a ul element.
func Ul(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("ul", props, children...)
}

// Ol creates an ol element.
func Ol(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("ol", props, children...)
}

// Li creates a li element.
func Li(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("li", props, children...)
}

// Dl creates a dl element.
func Dl(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("dl", props, children...)
}

// Dt creates a dt element.
func Dt(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("dt", props, children...)
}

// Dd creates a dd element.
func Dd(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("dd", props, children...)
}

// Inline Text

// A creates an a element.
func A(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("a", props, children...)
}

// Strong creates a strong element.
func Strong(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("strong", props, children...)
}

// Em creates an em element.
func Em(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("em", props, children...)
}

// Code creates a code element.
func Code(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("code", props, children...)
}

// Small creates a small element.
func Small(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("small", props, children...)
}

// Mark creates a mark element.
func Mark(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("mark", props, children...)
}

// Del creates a del element.
func Del(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("del", props, children...)
}

// Ins creates an ins element.
func Ins(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("ins", props, children...)
}

// Sub creates a sub element.
func Sub(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("sub", props, children...)
}

// Sup creates a sup element.
func Sup(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("sup", props, children...)
}

// Br creates a br element.
func Br(props map[string]interface{}) *Element {
	return CreateElement("br", props)
}

// Forms

// Form creates a form element.
func Form(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("form", props, children...)
}

// Label creates a label element.
func Label(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("label", props, children...)
}

// Input creates an input element.
func Input(props map[string]interface{}) *Element {
	return CreateElement("input", props)
}

// Button creates a button element.
func Button(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("button", props, children...)
}

// Select creates a select element.
func Select(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("select", props, children...)
}

// Option creates an option element.
func Option(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("option", props, children...)
}

// Textarea creates a textarea element.
func Textarea(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("textarea", props, children...)
}

// Fieldset creates a fieldset element.
func Fieldset(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("fieldset", props, children...)
}

// Legend creates a legend element.
func Legend(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("legend", props, children...)
}

// Tables

// Table creates a table element.
func Table(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("table", props, children...)
}

// Thead creates a thead element.
func Thead(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("thead", props, children...)
}

// Tbody creates a tbody element.
func Tbody(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("tbody", props, children...)
}

// Tfoot creates a tfoot element.
func Tfoot(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("tfoot", props, children...)
}

// Tr creates a tr element.
func Tr(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("tr", props, children...)
}

// Th creates a th element.
func Th(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("th", props, children...)
}

// Td creates a td element.
func Td(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("td", props, children...)
}

// Caption creates a caption element.
func Caption(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("caption", props, children...)
}

// Colgroup creates a colgroup element.
func Colgroup(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("colgroup", props, children...)
}

// Col creates a col element.
func Col(props map[string]interface{}) *Element {
	return CreateElement("col", props)
}

// Media

// Img creates an img element.
func Img(props map[string]interface{}) *Element {
	return CreateElement("img", props)
}

// Video creates a video element.
func Video(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("video", props, children...)
}

// Audio creates an audio element.
func Audio(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("audio", props, children...)
}

// Source creates a source element.
func Source(props map[string]interface{}) *Element {
	return CreateElement("source", props)
}

// Picture creates a picture element.
func Picture(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("picture", props, children...)
}

// Canvas creates a canvas element.
func Canvas(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("canvas", props, children...)
}

// Svg creates an svg element.
func Svg(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("svg", props, children...)
}

// Path creates a path element.
func Path(props map[string]interface{}) *Element {
	return CreateElement("path", props)
}

// Circle creates a circle element.
func Circle(props map[string]interface{}) *Element {
	return CreateElement("circle", props)
}

// Rect creates a rect element.
func Rect(props map[string]interface{}) *Element {
	return CreateElement("rect", props)
}

// Line creates a line element.
func Line(props map[string]interface{}) *Element {
	return CreateElement("line", props)
}

// Polygon creates a polygon element.
func Polygon(props map[string]interface{}) *Element {
	return CreateElement("polygon", props)
}

// G creates a g (SVG group) element.
func G(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("g", props, children...)
}

// Interactive

// Details creates a details element.
func Details(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("details", props, children...)
}

// Summary creates a summary element.
func Summary(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("summary", props, children...)
}

// Dialog creates a dialog element.
func Dialog(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("dialog", props, children...)
}

// Menu creates a menu element.
func Menu(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("menu", props, children...)
}

// Embedded Content

// Iframe creates an iframe element.
func Iframe(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("iframe", props, children...)
}

// Embed creates an embed element.
func Embed(props map[string]interface{}) *Element {
	return CreateElement("embed", props)
}

// Object creates an object element.
func Object(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("object", props, children...)
}

// Param creates a param element.
func Param(props map[string]interface{}) *Element {
	return CreateElement("param", props)
}

// Additional Elements

// Time creates a time element.
func Time(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("time", props, children...)
}

// Progress creates a progress element.
func Progress(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("progress", props, children...)
}

// Meter creates a meter element.
func Meter(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("meter", props, children...)
}

// Output creates an output element.
func Output(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("output", props, children...)
}

// Data creates a data element.
func Data(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("data", props, children...)
}

// Wbr creates a wbr element.
func Wbr(props map[string]interface{}) *Element {
	return CreateElement("wbr", props)
}

// Abbr creates an abbr element.
func Abbr(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("abbr", props, children...)
}

// Address creates an address element.
func Address(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("address", props, children...)
}

// Cite creates a cite element.
func Cite(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("cite", props, children...)
}

// Kbd creates a kbd element.
func Kbd(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("kbd", props, children...)
}

// Samp creates a samp element.
func Samp(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("samp", props, children...)
}

// Var creates a var element.
func Var(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("var", props, children...)
}

// Q creates a q element.
func Q(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("q", props, children...)
}

// Dfn creates a dfn element.
func Dfn(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("dfn", props, children...)
}

// B creates a b element.
func B(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("b", props, children...)
}

// I creates an i element.
func I(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("i", props, children...)
}

// U creates a u element.
func U(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("u", props, children...)
}

// S creates an s element.
func S(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("s", props, children...)
}

// Bdi creates a bdi element.
func Bdi(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("bdi", props, children...)
}

// Bdo creates a bdo element.
func Bdo(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("bdo", props, children...)
}

// Ruby creates a ruby element.
func Ruby(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("ruby", props, children...)
}

// Rt creates an rt element.
func Rt(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("rt", props, children...)
}

// Rp creates an rp element.
func Rp(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("rp", props, children...)
}

// Figure creates a figure element.
func Figure(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("figure", props, children...)
}

// Figcaption creates a figcaption element.
func Figcaption(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("figcaption", props, children...)
}

// Additional missing elements

// Track creates a track element.
func Track(props map[string]interface{}) *Element {
	return CreateElement("track", props)
}

// Optgroup creates an optgroup element.
func Optgroup(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("optgroup", props, children...)
}

// Datalist creates a datalist element.
func Datalist(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("datalist", props, children...)
}

// Hgroup creates an hgroup element.
func Hgroup(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("hgroup", props, children...)
}

// Portal creates a portal element.
func Portal(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("portal", props, children...)
}

// Template creates a template element.
func Template(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("template", props, children...)
}

// Slot creates a slot element.
func Slot(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("slot", props, children...)
}

// Helper functions for common patterns

// ClassProps returns a props map with the given CSS class.
func ClassProps(class string) map[string]interface{} {
	return map[string]interface{}{"class": class}
}

// IdProps returns a props map with the given element ID.
func IdProps(id string) map[string]interface{} {
	return map[string]interface{}{"id": id}
}

// HrefProps returns a props map with the given href attribute.
func HrefProps(href string) map[string]interface{} {
	return map[string]interface{}{"href": href}
}

// SrcProps returns a props map with the given src attribute.
func SrcProps(src string) map[string]interface{} {
	return map[string]interface{}{"src": src}
}

// StyleProps returns a props map with the given inline style string.
func StyleProps(style string) map[string]interface{} {
	return map[string]interface{}{"style": style}
}

// TypeProps returns a props map with the given type attribute.
func TypeProps(typ string) map[string]interface{} {
	return map[string]interface{}{"type": typ}
}

// ValueProps returns a props map with the given value.
func ValueProps(value interface{}) map[string]interface{} {
	return map[string]interface{}{"value": value}
}

// PlaceholderProps returns a props map with the given placeholder text.
func PlaceholderProps(placeholder string) map[string]interface{} {
	return map[string]interface{}{"placeholder": placeholder}
}

// InputTypeProps returns a props map with the given input type.
func InputTypeProps(typ string) map[string]interface{} {
	return map[string]interface{}{"type": typ}
}

// ClassIdProps returns a props map with the given class and id.
func ClassIdProps(class, id string) map[string]interface{} {
	return map[string]interface{}{"class": class, "id": id}
}

// EmptyProps returns an empty props map.
func EmptyProps() map[string]interface{} {
	return map[string]interface{}{}
}

// Component helpers

// WithComponents creates an element with the given tag and renders component refs as children.
func WithComponents(tagName string, props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	children := componentRefsToChildren(componentRefs)
	return CreateElement(tagName, props, children...)
}

// DivWithComponents creates a div element and renders component refs as children.
func DivWithComponents(props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	children := componentRefsToChildren(componentRefs)
	return Div(props, children...)
}

// SectionWithComponents creates a section element and renders component refs as children.
func SectionWithComponents(props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	children := componentRefsToChildren(componentRefs)
	return Section(props, children...)
}

// MainWithComponents creates a main element and renders component refs as children.
func MainWithComponents(props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	children := componentRefsToChildren(componentRefs)
	return Main(props, children...)
}

func componentRefsToChildren(componentRefs []func(map[string]interface{}) *Element) []interface{} {
	if len(componentRefs) == 0 {
		return emptyChildren
	}

	children := make([]interface{}, len(componentRefs))
	for i, ref := range componentRefs {
		children[i] = &Element{
			Type:     ref,
			Children: emptyChildren,
		}
	}
	return children
}
