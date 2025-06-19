package fiber

import "reflect"

// Type aliases for cleaner attribute syntax
type Attrs map[string]interface{}
type Attributes map[string]interface{}

// createElementWithStringSupport is a performance-optimized wrapper around createElement
// that automatically converts raw strings to Text elements for better DX
func createElementWithStringSupport(typ interface{}, props map[string]interface{}, children ...interface{}) *Element {
	if len(children) == 0 {
		return createElement(typ, props, children...)
	}

	// Process children with string conversion (less common path)
	processedChildren := make([]interface{}, 0, len(children))
	for _, child := range children {
		if child == nil {
			continue
		}

		if str, ok := child.(string); ok {
			processedChildren = append(processedChildren, Text(str))
		} else if reflect.TypeOf(child).Kind() == reflect.Func {
			// This is a component function, wrap it in an element
			processedChildren = append(processedChildren, CreateElement(child, nil))
		} else {
			processedChildren = append(processedChildren, child)
		}
	}

	return createElement(typ, props, processedChildren...)
}

// Element factory for common HTML elements - optimized for performance
// This reduces function call overhead by inlining the most frequently used elements
type ElementFactory struct {
	// Pre-allocated element types to avoid string allocations
	divType    string
	spanType   string
	pType      string
	buttonType string
	inputType  string
	h1Type     string
	h2Type     string
	h3Type     string
}

// Global element factory instance
var elementFactory = &ElementFactory{
	divType:    "div",
	spanType:   "span",
	pType:      "p",
	buttonType: "button",
	inputType:  "input",
	h1Type:     "h1",
	h2Type:     "h2",
	h3Type:     "h3",
}

// Pre-allocated common prop maps to avoid repeated allocations
// These are the most frequently used prop patterns in web applications
var (
	// Empty props - most common case
	emptyProps = map[string]interface{}{}

	// Common CSS class patterns - pre-allocated to avoid map creation overhead
	commonClassProps = map[string]map[string]interface{}{
		"container":    {"class": "container"},
		"btn":          {"class": "btn"},
		"btn-primary":  {"class": "btn-primary"},
		"form-control": {"class": "form-control"},
		"nav":          {"class": "nav"},
		"header":       {"class": "header"},
		"footer":       {"class": "footer"},
		"content":      {"class": "content"},
	}

	// Common input type patterns
	commonInputProps = map[string]map[string]interface{}{
		"text":     {"type": "text"},
		"password": {"type": "password"},
		"email":    {"type": "email"},
		"number":   {"type": "number"},
		"submit":   {"type": "submit"},
		"button":   {"type": "button"},
		"checkbox": {"type": "checkbox"},
		"radio":    {"type": "radio"},
	}
)

// Optimized inline functions for most common elements
// These avoid the function call overhead of createElement for hot paths

// HTML Element Aliases - These functions provide convenient aliases for createElement
// with pre-filled element names, making the code more readable and JSX-like.

// Document Structure Elements
func Html(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("html", props, children...)
}

func Head(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("head", props, children...)
}

func Body(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("body", props, children...)
}

func Title(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("title", props, children...)
}

func Meta(props Attrs) *Element {
	return createElement("meta", props)
}

func Link(props Attrs) *Element {
	return createElement("link", props)
}

func Style(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("style", props, children...)
}

func Script(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("script", props, children...)
}

// Semantic Structure Elements
func Header(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("header", props, children...)
}

func Nav(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("nav", props, children...)
}

func Main(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("main", props, children...)
}

func Section(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("section", props, children...)
}

func Article(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("article", props, children...)
}

func Aside(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("aside", props, children...)
}

func Footer(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("footer", props, children...)
}

// Layout Elements - Optimized for performance (most frequently used)
func Div(props Attrs, children ...interface{}) *Element {
	// Inline optimization for the most common element with auto string wrapping
	return createElementWithStringSupport(elementFactory.divType, props, children...)
}

func Span(props Attrs, children ...interface{}) *Element {
	// Inline optimization for common inline element with auto string wrapping
	return createElementWithStringSupport(elementFactory.spanType, props, children...)
}

func P(props Attrs, children ...interface{}) *Element {
	// Inline optimization for common text element with auto string wrapping
	return createElementWithStringSupport(elementFactory.pType, props, children...)
}

func Br(props Attrs) *Element {
	return createElement("br", props)
}

func Hr(props Attrs) *Element {
	return createElement("hr", props)
}

// Heading Elements - Optimized for common headings
func H1(props Attrs, children ...interface{}) *Element {
	// Inline optimization for most common heading with auto string wrapping
	return createElementWithStringSupport(elementFactory.h1Type, props, children...)
}

func H2(props Attrs, children ...interface{}) *Element {
	// Inline optimization for common heading with auto string wrapping
	return createElementWithStringSupport(elementFactory.h2Type, props, children...)
}

func H3(props Attrs, children ...interface{}) *Element {
	// Inline optimization for common heading with auto string wrapping
	return createElementWithStringSupport(elementFactory.h3Type, props, children...)
}

func H4(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("h4", props, children...)
}

func H5(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("h5", props, children...)
}

func H6(props Attrs, children ...interface{}) *Element {
	return createElementWithStringSupport("h6", props, children...)
}

// Text Content Elements
func Strong(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("strong", props, children...)
}

func Em(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("em", props, children...)
}

func Small(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("small", props, children...)
}

func Mark(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("mark", props, children...)
}

func Del(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("del", props, children...)
}

func Ins(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("ins", props, children...)
}

func Sub(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("sub", props, children...)
}

func Sup(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("sup", props, children...)
}

func Code(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("code", props, children...)
}

func Pre(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("pre", props, children...)
}

func Kbd(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("kbd", props, children...)
}

func Samp(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("samp", props, children...)
}

// Quote Elements
func Blockquote(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("blockquote", props, children...)
}

func Q(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("q", props, children...)
}

func Cite(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("cite", props, children...)
}

// List Elements
func Ul(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("ul", props, children...)
}

func Ol(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("ol", props, children...)
}

func Li(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("li", props, children...)
}

func Dl(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("dl", props, children...)
}

func Dt(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("dt", props, children...)
}

func Dd(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("dd", props, children...)
}

// Link Elements
func A(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("a", props, children...)
}

// Media Elements
func Img(props map[string]interface{}) *Element {
	return createElement("img", props)
}

func Video(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("video", props, children...)
}

func Audio(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("audio", props, children...)
}

func Source(props map[string]interface{}) *Element {
	return createElement("source", props)
}

func Track(props map[string]interface{}) *Element {
	return createElement("track", props)
}

func Canvas(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("canvas", props, children...)
}

func Svg(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("svg", props, children...)
}

// Form Elements
func Form(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("form", props, children...)
}

func Input(props map[string]interface{}) *Element {
	// Inline optimization for common form element
	return createElement(elementFactory.inputType, props)
}

func Textarea(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("textarea", props, children...)
}

func Button(props map[string]interface{}, children ...interface{}) *Element {
	// Inline optimization for common interactive element with auto string wrapping
	return createElementWithStringSupport(elementFactory.buttonType, props, children...)
}

func Select(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("select", props, children...)
}

func Option(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("option", props, children...)
}

func Optgroup(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("optgroup", props, children...)
}

func Label(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("label", props, children...)
}

func Fieldset(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("fieldset", props, children...)
}

func Legend(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("legend", props, children...)
}

func Datalist(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("datalist", props, children...)
}

func Output(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("output", props, children...)
}

func Progress(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("progress", props, children...)
}

func Meter(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("meter", props, children...)
}

// Table Elements
func Table(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("table", props, children...)
}

func Thead(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("thead", props, children...)
}

func Tbody(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("tbody", props, children...)
}

func Tfoot(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("tfoot", props, children...)
}

func Tr(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("tr", props, children...)
}

func Th(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("th", props, children...)
}

func Td(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("td", props, children...)
}

func Caption(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("caption", props, children...)
}

func Colgroup(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("colgroup", props, children...)
}

func Col(props map[string]interface{}) *Element {
	return createElement("col", props)
}

// Interactive Elements
func Details(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("details", props, children...)
}

func Summary(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("summary", props, children...)
}

func Dialog(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("dialog", props, children...)
}

// Content Sectioning Elements
func Address(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("address", props, children...)
}

func Hgroup(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("hgroup", props, children...)
}

// Time Elements
func Time(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("time", props, children...)
}

// Ruby Annotation Elements
func Ruby(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("ruby", props, children...)
}

func Rt(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("rt", props, children...)
}

func Rp(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("rp", props, children...)
}

// Definition Elements
func Dfn(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("dfn", props, children...)
}

func Abbr(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("abbr", props, children...)
}

// Generic Container Elements
func Figure(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("figure", props, children...)
}

func Figcaption(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("figcaption", props, children...)
}

func Data(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("data", props, children...)
}

func Var(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("var", props, children...)
}

func Wbr(props map[string]interface{}) *Element {
	return createElement("wbr", props)
}

func Bdi(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("bdi", props, children...)
}

func Bdo(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("bdo", props, children...)
}

// Embedded Content Elements
func Embed(props map[string]interface{}) *Element {
	return createElement("embed", props)
}

func Object(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("object", props, children...)
}

func Param(props map[string]interface{}) *Element {
	return createElement("param", props)
}

func Picture(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("picture", props, children...)
}

func Portal(props map[string]interface{}) *Element {
	return createElement("portal", props)
}

// Template Elements
func Template(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("template", props, children...)
}

func Slot(props map[string]interface{}, children ...interface{}) *Element {
	return createElementWithStringSupport("slot", props, children...)
}

// Helper function for nil props (common case)
func NilProps() map[string]interface{} {
	return nil
}

// Optimized helper function for class-only props (very common case)
// Uses pre-allocated maps for common CSS classes to avoid allocations
func ClassProps(className string) map[string]interface{} {
	// Fast path: check if we have a pre-allocated map for this class
	if preallocated, exists := commonClassProps[className]; exists {
		return preallocated
	}
	// Fallback: create new map for uncommon classes
	return map[string]interface{}{"class": className}
}

// Helper function for id-only props - optimized with capacity hint
func IdProps(id string) map[string]interface{} {
	// Pre-size map to avoid growth during insertion
	props := make(map[string]interface{}, 1)
	props["id"] = id
	return props
}

// Helper function for href-only props (links) - optimized with capacity hint
func HrefProps(href string) map[string]interface{} {
	// Pre-size map to avoid growth during insertion
	props := make(map[string]interface{}, 1)
	props["href"] = href
	return props
}

// New optimized helper for common input types
func InputTypeProps(inputType string) map[string]interface{} {
	// Fast path: check if we have a pre-allocated map for this input type
	if preallocated, exists := commonInputProps[inputType]; exists {
		return preallocated
	}
	// Fallback: create new map for uncommon input types
	props := make(map[string]interface{}, 1)
	props["type"] = inputType
	return props
}

// Helper for combined class and id props (common pattern)
func ClassIdProps(className, id string) map[string]interface{} {
	// Pre-size map for exactly 2 properties
	props := make(map[string]interface{}, 2)
	props["class"] = className
	props["id"] = id
	return props
}

// Helper for empty props (avoids nil checks in some cases)
func EmptyProps() map[string]interface{} {
	return emptyProps
}

// Helper function to create elements with component references as children
// This makes it easy to pass component functions as children
func WithComponents(tagName string, props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	// Convert component references to interfaces
	children := make([]interface{}, len(componentRefs))
	for i, comp := range componentRefs {
		children[i] = comp
	}
	return createElementWithStringSupport(tagName, props, children...)
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
