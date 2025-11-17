package runtime

// HTML element constructors
// These create virtual DOM elements for standard HTML tags

// Document Structure Elements

func Html(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("html", props, children...)
}

func Head(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("head", props, children...)
}

func Body(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("body", props, children...)
}

func Title(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("title", props, children...)
}

func Meta(props map[string]interface{}) *Element {
	return CreateElement("meta", props)
}

func Link(props map[string]interface{}) *Element {
	return CreateElement("link", props)
}

func Style(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("style", props, children...)
}

func Script(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("script", props, children...)
}

// Semantic Structure Elements

func Header(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("header", props, children...)
}

func Nav(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("nav", props, children...)
}

func Main(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("main", props, children...)
}

func Footer(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("footer", props, children...)
}

func Section(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("section", props, children...)
}

func Article(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("article", props, children...)
}

func Aside(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("aside", props, children...)
}

// Content Grouping

func Div(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("div", props, children...)
}

func P(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("p", props, children...)
}

func Span(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("span", props, children...)
}

func Pre(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("pre", props, children...)
}

func Blockquote(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("blockquote", props, children...)
}

func Hr(props map[string]interface{}) *Element {
	return CreateElement("hr", props)
}

// Text Content

func H1(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("h1", props, children...)
}

func H2(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("h2", props, children...)
}

func H3(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("h3", props, children...)
}

func H4(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("h4", props, children...)
}

func H5(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("h5", props, children...)
}

func H6(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("h6", props, children...)
}

// Lists

func Ul(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("ul", props, children...)
}

func Ol(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("ol", props, children...)
}

func Li(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("li", props, children...)
}

func Dl(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("dl", props, children...)
}

func Dt(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("dt", props, children...)
}

func Dd(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("dd", props, children...)
}

// Inline Text

func A(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("a", props, children...)
}

func Strong(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("strong", props, children...)
}

func Em(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("em", props, children...)
}

func Code(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("code", props, children...)
}

func Small(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("small", props, children...)
}

func Mark(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("mark", props, children...)
}

func Del(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("del", props, children...)
}

func Ins(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("ins", props, children...)
}

func Sub(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("sub", props, children...)
}

func Sup(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("sup", props, children...)
}

func Br(props map[string]interface{}) *Element {
	return CreateElement("br", props)
}

// Forms

func Form(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("form", props, children...)
}

func Label(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("label", props, children...)
}

func Input(props map[string]interface{}) *Element {
	return CreateElement("input", props)
}

func Button(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("button", props, children...)
}

func Select(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("select", props, children...)
}

func Option(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("option", props, children...)
}

func Textarea(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("textarea", props, children...)
}

func Fieldset(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("fieldset", props, children...)
}

func Legend(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("legend", props, children...)
}

// Tables

func Table(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("table", props, children...)
}

func Thead(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("thead", props, children...)
}

func Tbody(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("tbody", props, children...)
}

func Tfoot(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("tfoot", props, children...)
}

func Tr(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("tr", props, children...)
}

func Th(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("th", props, children...)
}

func Td(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("td", props, children...)
}

func Caption(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("caption", props, children...)
}

func Colgroup(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("colgroup", props, children...)
}

func Col(props map[string]interface{}) *Element {
	return CreateElement("col", props)
}

// Media

func Img(props map[string]interface{}) *Element {
	return CreateElement("img", props)
}

func Video(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("video", props, children...)
}

func Audio(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("audio", props, children...)
}

func Source(props map[string]interface{}) *Element {
	return CreateElement("source", props)
}

func Picture(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("picture", props, children...)
}

func Canvas(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("canvas", props, children...)
}

func Svg(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("svg", props, children...)
}

func Path(props map[string]interface{}) *Element {
	return CreateElement("path", props)
}

func Circle(props map[string]interface{}) *Element {
	return CreateElement("circle", props)
}

func Rect(props map[string]interface{}) *Element {
	return CreateElement("rect", props)
}

func Line(props map[string]interface{}) *Element {
	return CreateElement("line", props)
}

func Polygon(props map[string]interface{}) *Element {
	return CreateElement("polygon", props)
}

func G(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("g", props, children...)
}

// Interactive

func Details(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("details", props, children...)
}

func Summary(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("summary", props, children...)
}

func Dialog(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("dialog", props, children...)
}

func Menu(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("menu", props, children...)
}

// Embedded Content

func Iframe(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("iframe", props, children...)
}

func Embed(props map[string]interface{}) *Element {
	return CreateElement("embed", props)
}

func Object(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("object", props, children...)
}

func Param(props map[string]interface{}) *Element {
	return CreateElement("param", props)
}

// Additional Elements

func Time(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("time", props, children...)
}

func Progress(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("progress", props, children...)
}

func Meter(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("meter", props, children...)
}

func Output(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("output", props, children...)
}

func Data(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("data", props, children...)
}

func Wbr(props map[string]interface{}) *Element {
	return CreateElement("wbr", props)
}

func Abbr(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("abbr", props, children...)
}

func Address(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("address", props, children...)
}

func Cite(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("cite", props, children...)
}

func Kbd(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("kbd", props, children...)
}

func Samp(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("samp", props, children...)
}

func Var(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("var", props, children...)
}

func Q(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("q", props, children...)
}

func Dfn(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("dfn", props, children...)
}

func B(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("b", props, children...)
}

func I(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("i", props, children...)
}

func U(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("u", props, children...)
}

func S(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("s", props, children...)
}

func Bdi(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("bdi", props, children...)
}

func Bdo(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("bdo", props, children...)
}

func Ruby(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("ruby", props, children...)
}

func Rt(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("rt", props, children...)
}

func Rp(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("rp", props, children...)
}

func Figure(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("figure", props, children...)
}

func Figcaption(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("figcaption", props, children...)
}

// Additional missing elements

func Track(props map[string]interface{}) *Element {
	return CreateElement("track", props)
}

func Optgroup(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("optgroup", props, children...)
}

func Datalist(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("datalist", props, children...)
}

func Hgroup(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("hgroup", props, children...)
}

func Portal(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("portal", props, children...)
}

func Template(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("template", props, children...)
}

func Slot(props map[string]interface{}, children ...interface{}) *Element {
	return CreateElement("slot", props, children...)
}

// Helper functions for common patterns

func ClassProps(class string) map[string]interface{} {
	return map[string]interface{}{"class": class}
}

func IdProps(id string) map[string]interface{} {
	return map[string]interface{}{"id": id}
}

func HrefProps(href string) map[string]interface{} {
	return map[string]interface{}{"href": href}
}

func SrcProps(src string) map[string]interface{} {
	return map[string]interface{}{"src": src}
}

func StyleProps(style string) map[string]interface{} {
	return map[string]interface{}{"style": style}
}

func TypeProps(typ string) map[string]interface{} {
	return map[string]interface{}{"type": typ}
}

func ValueProps(value interface{}) map[string]interface{} {
	return map[string]interface{}{"value": value}
}

func PlaceholderProps(placeholder string) map[string]interface{} {
	return map[string]interface{}{"placeholder": placeholder}
}

func InputTypeProps(typ string) map[string]interface{} {
	return map[string]interface{}{"type": typ}
}

func ClassIdProps(class, id string) map[string]interface{} {
	return map[string]interface{}{"class": class, "id": id}
}

func EmptyProps() map[string]interface{} {
	return map[string]interface{}{}
}

// Component helpers

func WithComponents(tagName string, props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	// Convert componentRefs to interfaces for CreateElement
	children := make([]interface{}, len(componentRefs))
	for i, ref := range componentRefs {
		children[i] = ref
	}
	return CreateElement(tagName, props, children...)
}

func DivWithComponents(props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	// Convert componentRefs to interfaces
	children := make([]interface{}, len(componentRefs))
	for i, ref := range componentRefs {
		children[i] = ref
	}
	return Div(props, children...)
}

func SectionWithComponents(props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	// Convert componentRefs to interfaces
	children := make([]interface{}, len(componentRefs))
	for i, ref := range componentRefs {
		children[i] = ref
	}
	return Section(props, children...)
}

func MainWithComponents(props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	// Convert componentRefs to interfaces
	children := make([]interface{}, len(componentRefs))
	for i, ref := range componentRefs {
		children[i] = ref
	}
	return Main(props, children...)
}
