package html

import (
	"maps"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// DataAttribute names one data-* attribute (Name is the suffix after
// "data-") for the zero-allocation Props.DataAttr field.
type DataAttribute struct {
	Name  string
	Value string
}

// Props contains the common HTML attributes and event handlers supported by the typed builders.
type Props struct {
	ID    string
	Class string
	Key   string
	Slot  string
	Title string
	Type  string
	Name  string
	// Value is the input/option value.  An empty string is silently omitted by
	// toRuntimeProps; use html.Value("") (the PropOption) or Raw["value"]="" to
	// explicitly emit an empty controlled-input value.
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
	Pattern      string
	Lang         string
	Dir          string
	Width        string
	Height       string
	Loading      string

	Rows int
	Cols int
	// TabIndex of 0 is silently omitted by toRuntimeProps; use html.TabIndex(0)
	// (the PropOption) or Raw["tabIndex"]=0 to make tabindex=0 explicit.
	TabIndex  int
	MaxLength int
	MinLength int
	ColSpan   int
	RowSpan   int

	Checked   bool
	Disabled  bool
	Selected  bool
	Required  bool
	ReadOnly  bool
	Hidden    bool
	Multiple  bool
	AutoFocus bool
	Open      bool

	Style map[string]string
	Data  map[string]string
	Aria  map[string]string
	Raw   map[string]any

	// DataAttr sets one data-* attribute without allocating a Data map — the
	// dominant single-attribute case (e.g. a per-row id) costs zero
	// allocations this way. Name is the part after "data-". Combine with the
	// Data map for additional attributes; both land in the same
	// deterministically sorted segment.
	DataAttr DataAttribute

	// Text sets the element's entire text content directly. The equivalent
	// html.Text(...) child form allocates a full throwaway Element that Tag
	// discards after extracting the string — one of the highest-frequency
	// allocation sites in text-heavy trees. Ignored when children are passed
	// or when empty (use a Text child for explicit empty text).
	Text string

	OnClick         ui.Handler
	OnInput         ui.Handler
	OnChange        ui.Handler
	OnSubmit        ui.Handler
	OnKeyDown       ui.Handler
	OnKeyUp         ui.Handler
	OnMouseUp       ui.Handler
	OnMouseDown     ui.Handler
	OnMouseEnter    ui.Handler
	OnMouseLeave    ui.Handler
	OnDoubleClick   ui.Handler
	OnContextMenu   ui.Handler
	OnWheel         ui.Handler
	OnTransitionEnd ui.Handler
	OnAnimationEnd  ui.Handler
	OnLoad          ui.Handler
	OnError         ui.Handler
	OnPointerDown   ui.Handler
	OnPointerMove   ui.Handler
	OnPointerUp     ui.Handler
	OnTouchStart    ui.Handler
	OnTouchMove     ui.Handler
	OnTouchEnd      ui.Handler
	OnDragStart     ui.Handler
	OnDragOver      ui.Handler
	OnDrop          ui.Handler
	OnDragEnd       ui.Handler
	OnFocus         ui.Handler
	OnBlur          ui.Handler
	OnScroll        ui.Handler
}

// CustomElementProps makes attribute-versus-property intent explicit for
// browser-defined custom elements and web components.
type CustomElementProps struct {
	Props      Props
	Attributes map[string]string
	Presence   map[string]bool
	Properties map[string]any
}

const customElementPropertyPrefix = "__gwc_prop__:"
const parallelRegionClickSlotDataKey = "gwc-parallel-click-slot"

// Tag creates a node for an arbitrary HTML tag name.
func Tag(parseName string, parseProps Props, parseChildren ...ui.Node) ui.Node {
	// The event scan runs exactly once per element; both the compact
	// disqualification check and the map-lane fallback consume it.
	parseEvents := runtimeEventProps(parseProps)
	if parseKey, parseAttrs, isCompact := toRuntimeCompactProps(parseProps, parseEvents); isCompact {
		if parseProps.Text != "" && len(parseChildren) == 0 && parseName != "TEXT_ELEMENT" && parseName != "FRAGMENT" {
			return runtime.CreateElementCompactHostOwnedText(parseName, parseKey, parseAttrs, parseProps.Text)
		}
		if len(parseChildren) == 1 && parseName != "TEXT_ELEMENT" && parseName != "FRAGMENT" {
			if parseText, hasText := runtime.PlainTextContent(parseChildren[0]); hasText {
				return runtime.CreateElementCompactHostOwnedText(parseName, parseKey, parseAttrs, parseText)
			}
		}
		return runtime.CreateElementCompactHostOwned(parseName, parseKey, parseAttrs, toInterfaces(parseChildren)...)
	}
	if parseProps.Text != "" && len(parseChildren) == 0 {
		return runtime.CreateElementOwned(parseName, toRuntimePropsWithEvents(parseProps, parseEvents), any(Text(parseProps.Text)))
	}
	return runtime.CreateElementOwned(parseName, toRuntimePropsWithEvents(parseProps, parseEvents), toInterfaces(parseChildren)...)
}

// Link creates a typed link element.
func Link(parseProps Props) ui.Node {
	return Tag("link", parseProps)
}

// CustomElement creates a browser-defined custom element with explicit
// attribute and property channels.
func CustomElement(parseName string, parseProps CustomElementProps, parseChildren ...ui.Node) ui.Node {
	parseChildValues := toInterfaces(parseChildren)
	parseCount := len(parseProps.Attributes) + len(parseProps.Presence) + len(parseProps.Properties)
	if parseCount == 0 {
		parseEvents := runtimeEventProps(parseProps.Props)
		if parseKey, parseAttrs, isCompact := toRuntimeCompactProps(parseProps.Props, parseEvents); isCompact {
			return runtime.CreateElementCompactHostOwned(parseName, parseKey, parseAttrs, parseChildValues...)
		}
		return runtime.CreateElementOwned(parseName, toRuntimePropsWithEvents(parseProps.Props, parseEvents), parseChildValues...)
	}
	// Extra attribute/presence/property channels are rare; the map-based lane
	// keeps their merge semantics and CreateElementOwned re-derives the
	// compact attribute view when the merged payload allows it.
	parseValues := toRuntimeProps(parseProps.Props)
	if parseValues == nil {
		parseValues = make(map[string]any, parseCount)
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
		// Reject DOM sinks that parse their value as markup: Properties flows to a
		// literal JS property set (element[name]=value), so element.innerHTML =
		// userValue would execute injected markup, bypassing every escaping/
		// sanitization path in the framework. Use html.RawHTML (which runs through
		// sanitize) for trusted markup instead.
		if isUnsafeCustomElementProperty(parseKey3) {
			continue
		}
		parseValues[customElementPropertyPrefix+parseKey3] = parseValue2
	}
	return runtime.CreateElementOwned(parseName, parseValues, parseChildValues...)
}

// isUnsafeCustomElementProperty reports whether a custom-element property name is
// a DOM sink that parses its value as markup. Setting these via Properties (a
// literal element[name]=value) would execute injected HTML, bypassing all
// escaping/sanitization — use html.RawHTML for trusted markup instead.
func isUnsafeCustomElementProperty(parseName string) bool {
	switch strings.ToLower(strings.TrimSpace(parseName)) {
	case "innerhtml", "outerhtml", "insertadjacenthtml":
		return true
	default:
		return false
	}
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

// Iframe creates an iframe element.
func Iframe(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("iframe", parseProps, parseChildren...)
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

// Ol creates an ol ordered-list element.
func Ol(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("ol", parseProps, parseChildren...)
}

// Tfoot creates a tfoot table section.
func Tfoot(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("tfoot", parseProps, parseChildren...)
}

// Caption creates a table caption element.
func Caption(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("caption", parseProps, parseChildren...)
}

// Colgroup creates a colgroup table element.
func Colgroup(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("colgroup", parseProps, parseChildren...)
}

// Col creates a col table element.
func Col(parseProps Props) ui.Node { return Tag("col", parseProps) }

// Video creates a video media element.
func Video(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("video", parseProps, parseChildren...)
}

// Audio creates an audio media element.
func Audio(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("audio", parseProps, parseChildren...)
}

// Source creates a source media element.
func Source(parseProps Props) ui.Node { return Tag("source", parseProps) }

// Track creates a track media element.
func Track(parseProps Props) ui.Node { return Tag("track", parseProps) }

// Canvas creates a canvas element.
func Canvas(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("canvas", parseProps, parseChildren...)
}

// Optgroup creates an optgroup element.
func Optgroup(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("optgroup", parseProps, parseChildren...)
}

// Datalist creates a datalist element.
func Datalist(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("datalist", parseProps, parseChildren...)
}

// Output creates an output element.
func Output(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("output", parseProps, parseChildren...)
}

// Progress creates a progress element.
func Progress(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("progress", parseProps, parseChildren...)
}

// Meter creates a meter element.
func Meter(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("meter", parseProps, parseChildren...)
}

// Figure creates a figure element.
func Figure(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("figure", parseProps, parseChildren...)
}

// Figcaption creates a figcaption element.
func Figcaption(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("figcaption", parseProps, parseChildren...)
}

// Picture creates a picture element.
func Picture(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("picture", parseProps, parseChildren...)
}

// Abbr creates an abbr element.
func Abbr(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("abbr", parseProps, parseChildren...)
}

// Kbd creates a kbd element.
func Kbd(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("kbd", parseProps, parseChildren...)
}

// Sub creates a sub element.
func Sub(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("sub", parseProps, parseChildren...)
}

// Sup creates a sup element.
func Sup(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("sup", parseProps, parseChildren...)
}

// Del creates a del element.
func Del(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("del", parseProps, parseChildren...)
}

// Ins creates an ins element.
func Ins(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("ins", parseProps, parseChildren...)
}

// B creates a b element.
func B(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("b", parseProps, parseChildren...)
}

// I creates an i element.
func I(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("i", parseProps, parseChildren...)
}

// U creates a u element.
func U(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("u", parseProps, parseChildren...)
}

// Svg creates an svg element with the SVG namespace attribute set by default.
func Svg(parseProps Props, parseChildren ...ui.Node) ui.Node {
	if parseProps.Raw == nil {
		parseProps.Raw = map[string]any{"xmlns": "http://www.w3.org/2000/svg"}
		return Tag("svg", parseProps, parseChildren...)
	}
	if _, parseHasXMLNS := parseProps.Raw["xmlns"]; !parseHasXMLNS {
		parseProps.Raw = maps.Clone(parseProps.Raw)
		parseProps.Raw["xmlns"] = "http://www.w3.org/2000/svg"
	}
	return Tag("svg", parseProps, parseChildren...)
}

// Path creates an svg path element.
func Path(parseProps Props) ui.Node { return Tag("path", parseProps) }

// Circle creates an svg circle element.
func Circle(parseProps Props) ui.Node { return Tag("circle", parseProps) }

// Rect creates an svg rect element.
func Rect(parseProps Props) ui.Node { return Tag("rect", parseProps) }

// G creates an svg group element.
func G(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("g", parseProps, parseChildren...)
}

// Line creates an svg line element.
func Line(parseProps Props) ui.Node { return Tag("line", parseProps) }

// Polyline creates an svg polyline element.
func Polyline(parseProps Props) ui.Node { return Tag("polyline", parseProps) }

// Polygon creates an svg polygon element.
func Polygon(parseProps Props) ui.Node { return Tag("polygon", parseProps) }

// Defs creates an svg defs element.
func Defs(parseProps Props, parseChildren ...ui.Node) ui.Node {
	return Tag("defs", parseProps, parseChildren...)
}

// Use creates an svg use element.
func Use(parseProps Props) ui.Node { return Tag("use", parseProps) }

// toRuntimeCompactProps normalizes typed Props that contain only string
// attributes (plus an optional key) into the typed fast lane: a reconciliation
// key and a deterministic compact attribute slice, with no props map at all.
// parseEvents is the caller's single runtimeEventProps scan.
func toRuntimeCompactProps(parseProps Props, parseEvents []eventProp) (string, []runtime.HostAttr, bool) {
	if parseProps.Value != "" ||
		parseProps.Rows != 0 ||
		parseProps.Cols != 0 ||
		parseProps.TabIndex != 0 ||
		parseProps.MaxLength != 0 ||
		parseProps.MinLength != 0 ||
		parseProps.ColSpan != 0 ||
		parseProps.RowSpan != 0 ||
		parseProps.Checked ||
		parseProps.Disabled ||
		parseProps.Selected ||
		parseProps.Required ||
		parseProps.ReadOnly ||
		parseProps.Hidden ||
		parseProps.Multiple ||
		parseProps.AutoFocus ||
		parseProps.Open ||
		parseProps.Style != nil ||
		len(parseProps.Raw) != 0 ||
		len(parseEvents) != 0 {
		return "", nil, false
	}

	// The prop/attr name table (compactStringAttrNames) pairs positionally
	// with this stack array; both must stay index-aligned. Building the array
	// in place keeps the hot constructor free of helper-call copies and
	// closures.
	parseStringValues := [...]string{
		parseProps.ID,
		parseProps.Class,
		parseProps.Slot,
		parseProps.Title,
		parseProps.Type,
		parseProps.Name,
		parseProps.Placeholder,
		parseProps.Accept,
		parseProps.Href,
		parseProps.Src,
		parseProps.Alt,
		parseProps.For,
		parseProps.Role,
		parseProps.Target,
		parseProps.Rel,
		parseProps.As,
		parseProps.Action,
		parseProps.Method,
		parseProps.EncType,
		parseProps.AutoComplete,
		parseProps.Min,
		parseProps.Max,
		parseProps.Step,
		parseProps.Pattern,
		parseProps.Lang,
		parseProps.Dir,
		parseProps.Width,
		parseProps.Height,
		parseProps.Loading,
	}
	parseAttrCount := len(parseProps.Data) + len(parseProps.Aria)
	if parseProps.DataAttr.Name != "" {
		parseAttrCount++
	}
	for _, parseValue := range parseStringValues {
		if parseValue != "" {
			parseAttrCount++
		}
	}

	if parseAttrCount == 0 {
		return parseProps.Key, nil, true
	}

	parseAttrs := make([]runtime.HostAttr, 0, parseAttrCount)
	for parseIndex, parseValue := range parseStringValues {
		if parseValue == "" {
			continue
		}
		parseAttrs = append(parseAttrs, runtime.HostAttr{Name: compactStringAttrNames[parseIndex].attrName, Value: parseValue})
	}

	// Data/Aria segments come from map iteration; sort each segment so the
	// attribute order is deterministic for a given payload (the reconciler
	// compares fast-lane attribute slices positionally).
	parseDataStart := len(parseAttrs)
	// DataAttr and the Data map can both target the same data-* name. Emitting both
	// would produce a DUPLICATE data-<name> attribute in this fast-lane slice (the
	// map path in toRuntimePropsWithEvents lets Data overwrite DataAttr). Resolve the
	// collision the SAME way here — the Data map wins — so both build paths agree and
	// no duplicate attribute is emitted. (Data[name] on a nil map is a safe zero read.)
	if parseProps.DataAttr.Name != "" {
		if _, parseDataCollision := parseProps.Data[parseProps.DataAttr.Name]; !parseDataCollision {
			parseAttrs = append(parseAttrs, runtime.HostAttr{Name: internPrefixedAttrName(&dataAttrNameCache, "data-", parseProps.DataAttr.Name), Value: parseProps.DataAttr.Value})
		}
	}
	for parseKey, parseValue := range parseProps.Data {
		parseAttrs = append(parseAttrs, runtime.HostAttr{Name: internPrefixedAttrName(&dataAttrNameCache, "data-", parseKey), Value: parseValue})
	}
	sortHostAttrSegment(parseAttrs[parseDataStart:])
	parseAriaStart := len(parseAttrs)
	for parseKey2, parseValue2 := range parseProps.Aria {
		parseAttrs = append(parseAttrs, runtime.HostAttr{Name: internPrefixedAttrName(&ariaAttrNameCache, "aria-", parseKey2), Value: parseValue2})
	}
	sortHostAttrSegment(parseAttrs[parseAriaStart:])

	return parseProps.Key, parseAttrs, true
}

// dataAttrNameCache / ariaAttrNameCache intern prefixed attribute names
// ("row-id" -> "data-row-id"). Data/Aria keys are developer-authored and
// highly repetitive, so the concat allocated one string per attribute per
// element per render (measured ~10% of a bailout pass's allocations).
var (
	dataAttrNameCache attrNameCache
	ariaAttrNameCache attrNameCache
)

// attrNameCache is a size-capped intern table. The cap only exists so an app
// that (unusually) generates unique attribute NAMES per render cannot grow
// the cache without bound; past the cap new names just pay the concat.
type attrNameCache struct {
	entries sync.Map
	size    atomic.Int32
}

const maxAttrNameCacheEntries = 4096

// internPrefixedAttrName returns the cached prefixed form of one attribute key.
func internPrefixedAttrName(parseCache *attrNameCache, parsePrefix, parseKey string) string {
	if parseCached, parseOk := parseCache.entries.Load(parseKey); parseOk {
		return parseCached.(string)
	}
	parseName := parsePrefix + parseKey
	if parseCache.size.Load() < maxAttrNameCacheEntries {
		if _, parseExisted := parseCache.entries.LoadOrStore(parseKey, parseName); !parseExisted {
			parseCache.size.Add(1)
		}
	}
	return parseName
}

// sortHostAttrSegment insertion-sorts one small attribute segment by name;
// data-/aria- maps rarely exceed a handful of entries.
func sortHostAttrSegment(parseAttrs []runtime.HostAttr) {
	if len(parseAttrs) < 2 {
		return
	}
	for parseIndex := 1; parseIndex < len(parseAttrs); parseIndex++ {
		parseAttr := parseAttrs[parseIndex]
		parseSlot := parseIndex
		for parseSlot > 0 && parseAttrs[parseSlot-1].Name > parseAttr.Name {
			parseAttrs[parseSlot] = parseAttrs[parseSlot-1]
			parseSlot--
		}
		parseAttrs[parseSlot] = parseAttr
	}
}

// compactStringAttrNames maps each compactStringAttrValues slot to its runtime
// prop name and DOM attribute name, in the same order attributes were emitted
// historically (id, class, then the long tail).
var compactStringAttrNames = [...]struct {
	propName string
	attrName string
}{
	{"id", "id"},
	{"class", "class"},
	{"slot", "slot"},
	{"title", "title"},
	{"type", "type"},
	{"name", "name"},
	{"placeholder", "placeholder"},
	{"accept", "accept"},
	{"href", "href"},
	{"src", "src"},
	{"alt", "alt"},
	{"htmlFor", "for"},
	{"role", "role"},
	{"target", "target"},
	{"rel", "rel"},
	{"as", "as"},
	{"action", "action"},
	{"method", "method"},
	{"enctype", "enctype"},
	{"autocomplete", "autocomplete"},
	{"min", "min"},
	{"max", "max"},
	{"step", "step"},
	{"pattern", "pattern"},
	{"lang", "lang"},
	{"dir", "dir"},
	{"width", "width"},
	{"height", "height"},
	{"loading", "loading"},
}


// toRuntimeProps is a core package helper.
func toRuntimeProps(parseProps Props) map[string]any {
	return toRuntimePropsWithEvents(parseProps, runtimeEventProps(parseProps))
}

// toRuntimePropsWithEvents is toRuntimeProps with the event scan hoisted to
// the caller, so construction paths that already scanned events don't pay for
// a second pass.
func toRuntimePropsWithEvents(parseProps Props, parseEvents []eventProp) map[string]any {
	parseCount := len(parseProps.Data) + len(parseProps.Aria) + len(parseProps.Raw) + len(parseEvents)
	if parseProps.DataAttr.Name != "" {
		parseCount++
	}
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
	if parseProps.Pattern != "" {
		parseCount++
	}
	if parseProps.Lang != "" {
		parseCount++
	}
	if parseProps.Dir != "" {
		parseCount++
	}
	if parseProps.Width != "" {
		parseCount++
	}
	if parseProps.Height != "" {
		parseCount++
	}
	if parseProps.Loading != "" {
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
	if parseProps.MaxLength != 0 {
		parseCount++
	}
	if parseProps.MinLength != 0 {
		parseCount++
	}
	if parseProps.ColSpan != 0 {
		parseCount++
	}
	if parseProps.RowSpan != 0 {
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
	if parseProps.Open {
		parseCount++
	}
	if parseProps.Style != nil {
		parseCount++
	}

	if parseCount == 0 {
		return nil
	}

	parseValues := make(map[string]any, parseCount)
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
	if parseProps.Pattern != "" {
		parseValues["pattern"] = parseProps.Pattern
	}
	if parseProps.Lang != "" {
		parseValues["lang"] = parseProps.Lang
	}
	if parseProps.Dir != "" {
		parseValues["dir"] = parseProps.Dir
	}
	if parseProps.Width != "" {
		parseValues["width"] = parseProps.Width
	}
	if parseProps.Height != "" {
		parseValues["height"] = parseProps.Height
	}
	if parseProps.Loading != "" {
		parseValues["loading"] = parseProps.Loading
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
	if parseProps.MaxLength != 0 {
		parseValues["maxLength"] = parseProps.MaxLength
	}
	if parseProps.MinLength != 0 {
		parseValues["minLength"] = parseProps.MinLength
	}
	if parseProps.ColSpan != 0 {
		parseValues["colSpan"] = parseProps.ColSpan
	}
	if parseProps.RowSpan != 0 {
		parseValues["rowSpan"] = parseProps.RowSpan
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
	if parseProps.Open {
		parseValues["open"] = true
	}
	if parseProps.Style != nil {
		parseValues["style"] = parseProps.Style
	}

	if parseProps.DataAttr.Name != "" {
		parseValues[internPrefixedAttrName(&dataAttrNameCache, "data-", parseProps.DataAttr.Name)] = parseProps.DataAttr.Value
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
	for _, parseEvent := range parseEvents {
		parseValues[parseEvent.name] = parseEvent.value
	}
	if len(parseProps.Raw) != 0 {
		maps.Copy(parseValues, parseProps.Raw)
	}

	return parseValues
}

type eventProp struct {
	name  string
	value any
}

func runtimeEventProps(parseProps Props) []eventProp {
	var parseEvents []eventProp
	appendEventProp := func(parseName string, parseHandler ui.Handler) {
		if parseValue := parseHandler.Value(); parseValue != nil {
			parseEvents = append(parseEvents, eventProp{name: parseName, value: parseValue})
		}
	}
	appendEventProp("onclick", parseProps.OnClick)
	appendEventProp("oninput", parseProps.OnInput)
	appendEventProp("onchange", parseProps.OnChange)
	appendEventProp("onsubmit", parseProps.OnSubmit)
	appendEventProp("onkeydown", parseProps.OnKeyDown)
	appendEventProp("onkeyup", parseProps.OnKeyUp)
	appendEventProp("onmouseup", parseProps.OnMouseUp)
	appendEventProp("onmousedown", parseProps.OnMouseDown)
	appendEventProp("onmouseenter", parseProps.OnMouseEnter)
	appendEventProp("onmouseleave", parseProps.OnMouseLeave)
	appendEventProp("ondblclick", parseProps.OnDoubleClick)
	appendEventProp("oncontextmenu", parseProps.OnContextMenu)
	appendEventProp("onwheel", parseProps.OnWheel)
	appendEventProp("ontransitionend", parseProps.OnTransitionEnd)
	appendEventProp("onanimationend", parseProps.OnAnimationEnd)
	appendEventProp("onload", parseProps.OnLoad)
	appendEventProp("onerror", parseProps.OnError)
	appendEventProp("onpointerdown", parseProps.OnPointerDown)
	appendEventProp("onpointermove", parseProps.OnPointerMove)
	appendEventProp("onpointerup", parseProps.OnPointerUp)
	appendEventProp("ontouchstart", parseProps.OnTouchStart)
	appendEventProp("ontouchmove", parseProps.OnTouchMove)
	appendEventProp("ontouchend", parseProps.OnTouchEnd)
	appendEventProp("ondragstart", parseProps.OnDragStart)
	appendEventProp("ondragover", parseProps.OnDragOver)
	appendEventProp("ondrop", parseProps.OnDrop)
	appendEventProp("ondragend", parseProps.OnDragEnd)
	appendEventProp("onfocus", parseProps.OnFocus)
	appendEventProp("onblur", parseProps.OnBlur)
	appendEventProp("onscroll", parseProps.OnScroll)
	return parseEvents
}

// toInterfaces is a core package helper.
func toInterfaces(parseChildren []ui.Node) []any {
	if len(parseChildren) == 0 {
		return nil
	}

	parseValues := make([]any, 0, len(parseChildren))
	for _, parseChild := range parseChildren {
		parseValues = append(parseValues, parseChild)
	}

	return parseValues
}
