//go:build js && wasm
// +build js,wasm

package dom

import (
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// Type aliases for cleaner attribute syntax
type Attrs = map[string]interface{}
type Attributes = map[string]interface{}
type Element = runtime.Element

// Re-export all HTML element constructors from fiber package
// This provides a clean, organized API while maintaining backward compatibility

// Document Structure Elements

func Html(props Attrs, children ...interface{}) *Element {
	return runtime.Html(props, children...)
}

func Head(props Attrs, children ...interface{}) *Element {
	return runtime.Head(props, children...)
}

func Body(props Attrs, children ...interface{}) *Element {
	return runtime.Body(props, children...)
}

func Title(props Attrs, children ...interface{}) *Element {
	return runtime.Title(props, children...)
}

func Meta(props Attrs) *Element {
	return runtime.Meta(props)
}

func Link(props Attrs) *Element {
	return runtime.Link(props)
}

func Style(props Attrs, children ...interface{}) *Element {
	return runtime.Style(props, children...)
}

func Script(props Attrs, children ...interface{}) *Element {
	return runtime.Script(props, children...)
}

// Semantic Structure Elements

func Header(props Attrs, children ...interface{}) *Element {
	return runtime.Header(props, children...)
}

func Nav(props Attrs, children ...interface{}) *Element {
	return runtime.Nav(props, children...)
}

func Main(props Attrs, children ...interface{}) *Element {
	return runtime.Main(props, children...)
}

func Section(props Attrs, children ...interface{}) *Element {
	return runtime.Section(props, children...)
}

func Article(props Attrs, children ...interface{}) *Element {
	return runtime.Article(props, children...)
}

func Aside(props Attrs, children ...interface{}) *Element {
	return runtime.Aside(props, children...)
}

func Footer(props Attrs, children ...interface{}) *Element {
	return runtime.Footer(props, children...)
}

// Layout Elements

func Div(props Attrs, children ...interface{}) *Element {
	return runtime.Div(props, children...)
}

func Span(props Attrs, children ...interface{}) *Element {
	return runtime.Span(props, children...)
}

func P(props Attrs, children ...interface{}) *Element {
	return runtime.P(props, children...)
}

func Br(props Attrs) *Element {
	return runtime.Br(props)
}

func Hr(props Attrs) *Element {
	return runtime.Hr(props)
}

// Heading Elements

func H1(props Attrs, children ...interface{}) *Element {
	return runtime.H1(props, children...)
}

func H2(props Attrs, children ...interface{}) *Element {
	return runtime.H2(props, children...)
}

func H3(props Attrs, children ...interface{}) *Element {
	return runtime.H3(props, children...)
}

func H4(props Attrs, children ...interface{}) *Element {
	return runtime.H4(props, children...)
}

func H5(props Attrs, children ...interface{}) *Element {
	return runtime.H5(props, children...)
}

func H6(props Attrs, children ...interface{}) *Element {
	return runtime.H6(props, children...)
}

// Text Content Elements

func Strong(props Attrs, children ...interface{}) *Element {
	return runtime.Strong(props, children...)
}

func Em(props Attrs, children ...interface{}) *Element {
	return runtime.Em(props, children...)
}

func Small(props Attrs, children ...interface{}) *Element {
	return runtime.Small(props, children...)
}

func Mark(props Attrs, children ...interface{}) *Element {
	return runtime.Mark(props, children...)
}

func Del(props Attrs, children ...interface{}) *Element {
	return runtime.Del(props, children...)
}

func Ins(props Attrs, children ...interface{}) *Element {
	return runtime.Ins(props, children...)
}

func Sub(props Attrs, children ...interface{}) *Element {
	return runtime.Sub(props, children...)
}

func Sup(props Attrs, children ...interface{}) *Element {
	return runtime.Sup(props, children...)
}

func Code(props Attrs, children ...interface{}) *Element {
	return runtime.Code(props, children...)
}

func Pre(props Attrs, children ...interface{}) *Element {
	return runtime.Pre(props, children...)
}

func Kbd(props Attrs, children ...interface{}) *Element {
	return runtime.Kbd(props, children...)
}

func Samp(props Attrs, children ...interface{}) *Element {
	return runtime.Samp(props, children...)
}

// Quote Elements

func Blockquote(props Attrs, children ...interface{}) *Element {
	return runtime.Blockquote(props, children...)
}

func Q(props Attrs, children ...interface{}) *Element {
	return runtime.Q(props, children...)
}

func Cite(props Attrs, children ...interface{}) *Element {
	return runtime.Cite(props, children...)
}

// List Elements

func Ul(props Attrs, children ...interface{}) *Element {
	return runtime.Ul(props, children...)
}

func Ol(props Attrs, children ...interface{}) *Element {
	return runtime.Ol(props, children...)
}

func Li(props Attrs, children ...interface{}) *Element {
	return runtime.Li(props, children...)
}

func Dl(props Attrs, children ...interface{}) *Element {
	return runtime.Dl(props, children...)
}

func Dt(props Attrs, children ...interface{}) *Element {
	return runtime.Dt(props, children...)
}

func Dd(props Attrs, children ...interface{}) *Element {
	return runtime.Dd(props, children...)
}

// Link Elements

func A(props Attrs, children ...interface{}) *Element {
	return runtime.A(props, children...)
}

// Media Elements

func Img(props Attrs) *Element {
	return runtime.Img(props)
}

func Video(props Attrs, children ...interface{}) *Element {
	return runtime.Video(props, children...)
}

func Audio(props Attrs, children ...interface{}) *Element {
	return runtime.Audio(props, children...)
}

func Source(props Attrs) *Element {
	return runtime.Source(props)
}

func Track(props Attrs) *Element {
	return runtime.Track(props)
}

func Canvas(props Attrs, children ...interface{}) *Element {
	return runtime.Canvas(props, children...)
}

func Svg(props Attrs, children ...interface{}) *Element {
	return runtime.Svg(props, children...)
}

// Form Elements

func Form(props Attrs, children ...interface{}) *Element {
	return runtime.Form(props, children...)
}

func Input(props Attrs) *Element {
	return runtime.Input(props)
}

func Textarea(props Attrs, children ...interface{}) *Element {
	return runtime.Textarea(props, children...)
}

func Button(props Attrs, children ...interface{}) *Element {
	return runtime.Button(props, children...)
}

func Select(props Attrs, children ...interface{}) *Element {
	return runtime.Select(props, children...)
}

func Option(props Attrs, children ...interface{}) *Element {
	return runtime.Option(props, children...)
}

func Optgroup(props Attrs, children ...interface{}) *Element {
	return runtime.Optgroup(props, children...)
}

func Label(props Attrs, children ...interface{}) *Element {
	return runtime.Label(props, children...)
}

func Fieldset(props Attrs, children ...interface{}) *Element {
	return runtime.Fieldset(props, children...)
}

func Legend(props Attrs, children ...interface{}) *Element {
	return runtime.Legend(props, children...)
}

func Datalist(props Attrs, children ...interface{}) *Element {
	return runtime.Datalist(props, children...)
}

func Output(props Attrs, children ...interface{}) *Element {
	return runtime.Output(props, children...)
}

func Progress(props Attrs, children ...interface{}) *Element {
	return runtime.Progress(props, children...)
}

func Meter(props Attrs, children ...interface{}) *Element {
	return runtime.Meter(props, children...)
}

// Table Elements

func Table(props Attrs, children ...interface{}) *Element {
	return runtime.Table(props, children...)
}

func Thead(props Attrs, children ...interface{}) *Element {
	return runtime.Thead(props, children...)
}

func Tbody(props Attrs, children ...interface{}) *Element {
	return runtime.Tbody(props, children...)
}

func Tfoot(props Attrs, children ...interface{}) *Element {
	return runtime.Tfoot(props, children...)
}

func Tr(props Attrs, children ...interface{}) *Element {
	return runtime.Tr(props, children...)
}

func Th(props Attrs, children ...interface{}) *Element {
	return runtime.Th(props, children...)
}

func Td(props Attrs, children ...interface{}) *Element {
	return runtime.Td(props, children...)
}

func Caption(props Attrs, children ...interface{}) *Element {
	return runtime.Caption(props, children...)
}

func Colgroup(props Attrs, children ...interface{}) *Element {
	return runtime.Colgroup(props, children...)
}

func Col(props Attrs) *Element {
	return runtime.Col(props)
}

// Interactive Elements

func Details(props Attrs, children ...interface{}) *Element {
	return runtime.Details(props, children...)
}

func Summary(props Attrs, children ...interface{}) *Element {
	return runtime.Summary(props, children...)
}

func Dialog(props Attrs, children ...interface{}) *Element {
	return runtime.Dialog(props, children...)
}

// Content Sectioning Elements

func Address(props Attrs, children ...interface{}) *Element {
	return runtime.Address(props, children...)
}

func Hgroup(props Attrs, children ...interface{}) *Element {
	return runtime.Hgroup(props, children...)
}

// Time Elements

func Time(props Attrs, children ...interface{}) *Element {
	return runtime.Time(props, children...)
}

// Ruby Annotation Elements

func Ruby(props Attrs, children ...interface{}) *Element {
	return runtime.Ruby(props, children...)
}

func Rt(props Attrs, children ...interface{}) *Element {
	return runtime.Rt(props, children...)
}

func Rp(props Attrs, children ...interface{}) *Element {
	return runtime.Rp(props, children...)
}

// Definition Elements

func Dfn(props Attrs, children ...interface{}) *Element {
	return runtime.Dfn(props, children...)
}

func Abbr(props Attrs, children ...interface{}) *Element {
	return runtime.Abbr(props, children...)
}

// Generic Container Elements

func Figure(props Attrs, children ...interface{}) *Element {
	return runtime.Figure(props, children...)
}

func Figcaption(props Attrs, children ...interface{}) *Element {
	return runtime.Figcaption(props, children...)
}

func Data(props Attrs, children ...interface{}) *Element {
	return runtime.Data(props, children...)
}

func Var(props Attrs, children ...interface{}) *Element {
	return runtime.Var(props, children...)
}

func Wbr(props Attrs) *Element {
	return runtime.Wbr(props)
}

func Bdi(props Attrs, children ...interface{}) *Element {
	return runtime.Bdi(props, children...)
}

func Bdo(props Attrs, children ...interface{}) *Element {
	return runtime.Bdo(props, children...)
}

// Embedded Content Elements

func Embed(props Attrs) *Element {
	return runtime.Embed(props)
}

func Object(props Attrs, children ...interface{}) *Element {
	return runtime.Object(props, children...)
}

func Param(props Attrs) *Element {
	return runtime.Param(props)
}

func Picture(props Attrs, children ...interface{}) *Element {
	return runtime.Picture(props, children...)
}

func Portal(props Attrs) *Element {
	return runtime.Portal(props)
}

// Template Elements

func Template(props Attrs, children ...interface{}) *Element {
	return runtime.Template(props, children...)
}

func Slot(props Attrs, children ...interface{}) *Element {
	return runtime.Slot(props, children...)
}

// Helper Functions

// NilProps returns a nil props map - useful for elements without attributes.
func NilProps() map[string]interface{} {
	return nil
}

// ClassProps creates a props map with only a class attribute.
// Uses pre-allocated maps for common CSS classes to reduce allocations.
func ClassProps(className string) map[string]interface{} {
	return runtime.ClassProps(className)
}

// IdProps creates a props map with only an id attribute.
func IdProps(id string) map[string]interface{} {
	return runtime.IdProps(id)
}

// HrefProps creates a props map with only an href attribute.
func HrefProps(href string) map[string]interface{} {
	return runtime.HrefProps(href)
}

// InputTypeProps creates a props map with only a type attribute.
// Uses pre-allocated maps for common input types to reduce allocations.
func InputTypeProps(inputType string) map[string]interface{} {
	return runtime.InputTypeProps(inputType)
}

// ClassIdProps creates a props map with both class and id attributes.
func ClassIdProps(className, id string) map[string]interface{} {
	return runtime.ClassIdProps(className, id)
}

// EmptyProps returns an empty props map.
func EmptyProps() map[string]interface{} {
	return runtime.EmptyProps()
}

// WithComponents creates an element with component references as children.
func WithComponents(tagName string, props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	return runtime.WithComponents(tagName, props, componentRefs...)
}

// DivWithComponents creates a div element with component references as children.
func DivWithComponents(props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	return runtime.DivWithComponents(props, componentRefs...)
}

// SectionWithComponents creates a section element with component references as children.
func SectionWithComponents(props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	return runtime.SectionWithComponents(props, componentRefs...)
}

// MainWithComponents creates a main element with component references as children.
func MainWithComponents(props map[string]interface{}, componentRefs ...func(map[string]interface{}) *Element) *Element {
	return runtime.MainWithComponents(props, componentRefs...)
}
