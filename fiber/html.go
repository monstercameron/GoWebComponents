package fiber

// Type aliases for cleaner attribute syntax
type Attrs map[string]interface{}
type Attributes map[string]interface{}

// HTML Element Aliases - These functions provide convenient aliases for createElement
// with pre-filled element names, making the code more readable and JSX-like.

// Document Structure Elements
func Html(props Attrs, children ...interface{}) *Element {
	return createElement("html", props, children...)
}

func Head(props Attrs, children ...interface{}) *Element {
	return createElement("head", props, children...)
}

func Body(props Attrs, children ...interface{}) *Element {
	return createElement("body", props, children...)
}

func Title(props Attrs, children ...interface{}) *Element {
	return createElement("title", props, children...)
}

func Meta(props Attrs) *Element {
	return createElement("meta", props)
}

func Link(props Attrs) *Element {
	return createElement("link", props)
}

func Style(props Attrs, children ...interface{}) *Element {
	return createElement("style", props, children...)
}

func Script(props Attrs, children ...interface{}) *Element {
	return createElement("script", props, children...)
}

// Semantic Structure Elements
func Header(props Attrs, children ...interface{}) *Element {
	return createElement("header", props, children...)
}

func Nav(props Attrs, children ...interface{}) *Element {
	return createElement("nav", props, children...)
}

func Main(props Attrs, children ...interface{}) *Element {
	return createElement("main", props, children...)
}

func Section(props Attrs, children ...interface{}) *Element {
	return createElement("section", props, children...)
}

func Article(props Attrs, children ...interface{}) *Element {
	return createElement("article", props, children...)
}

func Aside(props Attrs, children ...interface{}) *Element {
	return createElement("aside", props, children...)
}

func Footer(props Attrs, children ...interface{}) *Element {
	return createElement("footer", props, children...)
}

// Layout Elements
func Div(props Attrs, children ...interface{}) *Element {
	return createElement("div", props, children...)
}

func Span(props Attrs, children ...interface{}) *Element {
	return createElement("span", props, children...)
}

func P(props Attrs, children ...interface{}) *Element {
	return createElement("p", props, children...)
}

func Br(props Attrs) *Element {
	return createElement("br", props)
}

func Hr(props Attrs) *Element {
	return createElement("hr", props)
}

// Heading Elements
func H1(props Attrs, children ...interface{}) *Element {
	return createElement("h1", props, children...)
}

func H2(props Attrs, children ...interface{}) *Element {
	return createElement("h2", props, children...)
}

func H3(props Attrs, children ...interface{}) *Element {
	return createElement("h3", props, children...)
}

func H4(props Attrs, children ...interface{}) *Element {
	return createElement("h4", props, children...)
}

func H5(props Attrs, children ...interface{}) *Element {
	return createElement("h5", props, children...)
}

func H6(props Attrs, children ...interface{}) *Element {
	return createElement("h6", props, children...)
}

// Text Content Elements
func Strong(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("strong", props, children...)
}

func Em(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("em", props, children...)
}

func Small(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("small", props, children...)
}

func Mark(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("mark", props, children...)
}

func Del(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("del", props, children...)
}

func Ins(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("ins", props, children...)
}

func Sub(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("sub", props, children...)
}

func Sup(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("sup", props, children...)
}

func Code(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("code", props, children...)
}

func Pre(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("pre", props, children...)
}

func Kbd(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("kbd", props, children...)
}

func Samp(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("samp", props, children...)
}

// Quote Elements
func Blockquote(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("blockquote", props, children...)
}

func Q(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("q", props, children...)
}

func Cite(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("cite", props, children...)
}

// List Elements
func Ul(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("ul", props, children...)
}

func Ol(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("ol", props, children...)
}

func Li(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("li", props, children...)
}

func Dl(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("dl", props, children...)
}

func Dt(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("dt", props, children...)
}

func Dd(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("dd", props, children...)
}

// Link Elements
func A(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("a", props, children...)
}

// Media Elements
func Img(props map[string]interface{}) *Element {
	return createElement("img", props)
}

func Video(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("video", props, children...)
}

func Audio(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("audio", props, children...)
}

func Source(props map[string]interface{}) *Element {
	return createElement("source", props)
}

func Track(props map[string]interface{}) *Element {
	return createElement("track", props)
}

func Canvas(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("canvas", props, children...)
}

func Svg(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("svg", props, children...)
}

// Form Elements
func Form(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("form", props, children...)
}

func Input(props map[string]interface{}) *Element {
	return createElement("input", props)
}

func Textarea(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("textarea", props, children...)
}

func Button(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("button", props, children...)
}

func Select(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("select", props, children...)
}

func Option(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("option", props, children...)
}

func Optgroup(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("optgroup", props, children...)
}

func Label(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("label", props, children...)
}

func Fieldset(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("fieldset", props, children...)
}

func Legend(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("legend", props, children...)
}

func Datalist(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("datalist", props, children...)
}

func Output(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("output", props, children...)
}

func Progress(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("progress", props, children...)
}

func Meter(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("meter", props, children...)
}

// Table Elements
func Table(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("table", props, children...)
}

func Thead(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("thead", props, children...)
}

func Tbody(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("tbody", props, children...)
}

func Tfoot(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("tfoot", props, children...)
}

func Tr(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("tr", props, children...)
}

func Th(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("th", props, children...)
}

func Td(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("td", props, children...)
}

func Caption(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("caption", props, children...)
}

func Colgroup(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("colgroup", props, children...)
}

func Col(props map[string]interface{}) *Element {
	return createElement("col", props)
}

// Interactive Elements
func Details(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("details", props, children...)
}

func Summary(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("summary", props, children...)
}

func Dialog(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("dialog", props, children...)
}

// Content Sectioning Elements
func Address(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("address", props, children...)
}

func Hgroup(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("hgroup", props, children...)
}

// Time Elements
func Time(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("time", props, children...)
}

// Ruby Annotation Elements
func Ruby(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("ruby", props, children...)
}

func Rt(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("rt", props, children...)
}

func Rp(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("rp", props, children...)
}

// Definition Elements
func Dfn(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("dfn", props, children...)
}

func Abbr(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("abbr", props, children...)
}

// Generic Container Elements
func Figure(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("figure", props, children...)
}

func Figcaption(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("figcaption", props, children...)
}

func Data(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("data", props, children...)
}

func Var(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("var", props, children...)
}

func Wbr(props map[string]interface{}) *Element {
	return createElement("wbr", props)
}

func Bdi(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("bdi", props, children...)
}

func Bdo(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("bdo", props, children...)
}

// Embedded Content Elements
func Embed(props map[string]interface{}) *Element {
	return createElement("embed", props)
}

func Object(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("object", props, children...)
}

func Param(props map[string]interface{}) *Element {
	return createElement("param", props)
}

func Picture(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("picture", props, children...)
}

func Portal(props map[string]interface{}) *Element {
	return createElement("portal", props)
}

// Template Elements
func Template(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("template", props, children...)
}

func Slot(props map[string]interface{}, children ...interface{}) *Element {
	return createElement("slot", props, children...)
}

// Helper function for nil props (common case)
func NilProps() map[string]interface{} {
	return nil
}

// Helper function for class-only props (very common case)
func ClassProps(className string) map[string]interface{} {
	return map[string]interface{}{"class": className}
}

// Helper function for id-only props
func IdProps(id string) map[string]interface{} {
	return map[string]interface{}{"id": id}
}

// Helper function for href-only props (links)
func HrefProps(href string) map[string]interface{} {
	return map[string]interface{}{"href": href}
}

// Helper function to create elements with component references as children
// This makes it easy to pass component functions as children
func WithComponents(tagName string, props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	// Convert component references to interfaces
	children := make([]interface{}, len(componentRefs))
	for i, comp := range componentRefs {
		children[i] = comp
	}
	return createElement(tagName, props, children...)
}

// Helper functions for common patterns with component references
func DivWithComponents(props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	return WithComponents("div", props, componentRefs...)
}

func SectionWithComponents(props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	return WithComponents("section", props, componentRefs...)
}

func MainWithComponents(props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	return WithComponents("main", props, componentRefs...)
}
