package shorthand

import (
	"time"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

type Props = html.Props
type PropOption = html.PropOption
type SwitchBranch = html.SwitchBranch

type propsInput struct {
	value Props
}

// FromProps injects a full Props value into a mixed-argument shorthand call.
func FromProps(props Props) interface{} {
	return propsInput{value: props}
}

// Tag builds an arbitrary host element from mixed prop options and children.
func Tag(name string, args ...interface{}) ui.Node {
	props, children := splitArgs(args...)
	return html.Tag(name, props, children...)
}

// Fragment groups mixed children without introducing a host element.
func Fragment(args ...interface{}) ui.Node {
	return html.Fragment(html.Children(args...)...)
}

// A delegates to [html.A].
func A(args ...interface{}) ui.Node { return Tag("a", args...) }

// Article delegates to [html.Article].
func Article(args ...interface{}) ui.Node { return Tag("article", args...) }

// Body delegates to [html.Body].
func Body(args ...interface{}) ui.Node { return Tag("body", args...) }

// Button delegates to [html.Button].
func Button(args ...interface{}) ui.Node { return Tag("button", args...) }

// Br delegates to [html.Br].
func Br(args ...interface{}) ui.Node { return Tag("br", args...) }

// Code delegates to [html.Code].
func Code(args ...interface{}) ui.Node { return Tag("code", args...) }

// Details delegates to [html.Details].
func Details(args ...interface{}) ui.Node { return Tag("details", args...) }

// Div delegates to [html.Div].
func Div(args ...interface{}) ui.Node { return Tag("div", args...) }

// Form delegates to [html.Form].
func Form(args ...interface{}) ui.Node { return Tag("form", args...) }

// H1 delegates to [html.H1].
func H1(args ...interface{}) ui.Node { return Tag("h1", args...) }

// H2 delegates to [html.H2].
func H2(args ...interface{}) ui.Node { return Tag("h2", args...) }

// H3 delegates to [html.H3].
func H3(args ...interface{}) ui.Node { return Tag("h3", args...) }

// Head delegates to [html.Head].
func Head(args ...interface{}) ui.Node { return Tag("head", args...) }

// Header delegates to [html.Header].
func Header(args ...interface{}) ui.Node { return Tag("header", args...) }

// Hr delegates to [html.Hr].
func Hr(args ...interface{}) ui.Node { return Tag("hr", args...) }

// Html delegates to [html.Html].
func Html(args ...interface{}) ui.Node { return Tag("html", args...) }

// Img delegates to [html.Img].
func Img(args ...interface{}) ui.Node { return Tag("img", args...) }

// Input delegates to [html.Input].
func Input(args ...interface{}) ui.Node { return Tag("input", args...) }

// Label delegates to [html.Label].
func Label(args ...interface{}) ui.Node { return Tag("label", args...) }

// Li delegates to [html.Li].
func Li(args ...interface{}) ui.Node { return Tag("li", args...) }

// Main delegates to [html.Main].
func Main(args ...interface{}) ui.Node { return Tag("main", args...) }

// Mark delegates to [html.Mark].
func Mark(args ...interface{}) ui.Node { return Tag("mark", args...) }

// Meta delegates to [html.Meta].
func Meta(args ...interface{}) ui.Node { return Tag("meta", args...) }

// NoScript delegates to [html.NoScript].
func NoScript(args ...interface{}) ui.Node { return Tag("noscript", args...) }

// Option delegates to [html.Option].
func Option(args ...interface{}) ui.Node { return Tag("option", args...) }

// P delegates to [html.P].
func P(args ...interface{}) ui.Node { return Tag("p", args...) }

// Pre delegates to [html.Pre].
func Pre(args ...interface{}) ui.Node { return Tag("pre", args...) }

// Script delegates to [html.Script].
func Script(args ...interface{}) ui.Node { return Tag("script", args...) }

// Section delegates to [html.Section].
func Section(args ...interface{}) ui.Node { return Tag("section", args...) }

// Select delegates to [html.Select].
func Select(args ...interface{}) ui.Node { return Tag("select", args...) }

// Span delegates to [html.Span].
func Span(args ...interface{}) ui.Node { return Tag("span", args...) }

// Summary delegates to [html.Summary].
func Summary(args ...interface{}) ui.Node { return Tag("summary", args...) }

// Table delegates to [html.Table].
func Table(args ...interface{}) ui.Node { return Tag("table", args...) }

// Tbody delegates to [html.Tbody].
func Tbody(args ...interface{}) ui.Node { return Tag("tbody", args...) }

// Td delegates to [html.Td].
func Td(args ...interface{}) ui.Node { return Tag("td", args...) }

// Th delegates to [html.Th].
func Th(args ...interface{}) ui.Node { return Tag("th", args...) }

// Thead delegates to [html.Thead].
func Thead(args ...interface{}) ui.Node { return Tag("thead", args...) }

// Tr delegates to [html.Tr].
func Tr(args ...interface{}) ui.Node { return Tag("tr", args...) }

// Ul delegates to [html.Ul].
func Ul(args ...interface{}) ui.Node { return Tag("ul", args...) }

// Text delegates to [html.Text].
func Text(content interface{}) ui.Node { return html.Text(content) }

// Textf delegates to [html.Textf].
func Textf(format string, args ...interface{}) ui.Node { return html.Textf(format, args...) }

// TextIf delegates to [html.TextIf].
func TextIf(condition bool, content interface{}) ui.Node { return html.TextIf(condition, content) }

// Children delegates to [html.Children].
func Children(values ...interface{}) []ui.Node { return html.Children(values...) }

// When delegates to [html.When].
func When(condition bool, className string) string { return html.When(condition, className) }

// ClassNames delegates to [html.ClassNames].
func ClassNames(parts ...interface{}) string { return html.ClassNames(parts...) }

// If delegates to [html.If].
func If(condition bool, node ui.Node) ui.Node { return html.If(condition, node) }

// IfElse delegates to [html.IfElse].
func IfElse(condition bool, whenTrue ui.Node, whenFalse ui.Node) ui.Node {
	return html.IfElse(condition, whenTrue, whenFalse)
}

// Unless delegates to [html.Unless].
func Unless(condition bool, node ui.Node) ui.Node { return html.Unless(condition, node) }

// WithKey delegates to [html.WithKey].
func WithKey(node ui.Node, key interface{}) ui.Node { return html.WithKey(node, key) }

// Case delegates to [html.Case].
func Case(value interface{}, node ui.Node) SwitchBranch { return html.Case(value, node) }

// Default delegates to [html.Default].
func Default(node ui.Node) SwitchBranch { return html.Default(node) }

// Switch delegates to [html.Switch].
func Switch(value interface{}, branches ...SwitchBranch) ui.Node {
	return html.Switch(value, branches...)
}

// PropsOf delegates to [html.PropsOf].
func PropsOf(options ...PropOption) Props { return html.PropsOf(options...) }

// WithProps delegates to [html.WithProps].
func WithProps(base Props, options ...PropOption) Props { return html.WithProps(base, options...) }

// Map delegates to [html.Map].
func Map[T any](items []T, render func(T) ui.Node) []ui.Node { return html.Map(items, render) }

// MapKeyed delegates to [html.MapKeyed].
func MapKeyed[T any](items []T, key func(T) interface{}, render func(T) ui.Node) []ui.Node {
	return html.MapKeyed(items, key, render)
}

// FlatMap delegates to [html.FlatMap].
func FlatMap[T any](items []T, render func(T) []ui.Node) []ui.Node {
	return html.FlatMap(items, render)
}

// FilterMap delegates to [html.FilterMap].
func FilterMap[T any](items []T, render func(T) (ui.Node, bool)) []ui.Node {
	return html.FilterMap(items, render)
}

// Join delegates to [html.Join].
func Join(separator ui.Node, nodes ...ui.Node) []ui.Node { return html.Join(separator, nodes...) }

// Maybe delegates to [html.Maybe].
func Maybe[T any](value *T, render func(T) ui.Node) ui.Node { return html.Maybe(value, render) }

// OrElse delegates to [html.OrElse].
func OrElse[T any](value *T, fallback T) T { return html.OrElse(value, fallback) }

// Coalesce delegates to [html.Coalesce].
func Coalesce[T any](values ...*T) *T { return html.Coalesce(values...) }

// ID delegates to [html.ID].
func ID(value string) PropOption { return html.ID(value) }

// Class delegates to [html.Class].
func Class(value string) PropOption { return html.Class(value) }

// For delegates to [html.For].
func For(value string) PropOption { return html.For(value) }

// Name delegates to [html.Name].
func Name(value string) PropOption { return html.Name(value) }

// Title delegates to [html.Title].
func Title(value string) PropOption { return html.Title(value) }

// Value delegates to [html.Value].
func Value(value string) PropOption { return html.Value(value) }

// Placeholder delegates to [html.Placeholder].
func Placeholder(value string) PropOption { return html.Placeholder(value) }

// Type delegates to [html.Type].
func Type(value string) PropOption { return html.Type(value) }

// Href delegates to [html.Href].
func Href(value string) PropOption { return html.Href(value) }

// Src delegates to [html.Src].
func Src(value string) PropOption { return html.Src(value) }

// Role delegates to [html.Role].
func Role(value string) PropOption { return html.Role(value) }

// Rows delegates to [html.Rows].
func Rows(value int) PropOption { return html.Rows(value) }

// TabIndex delegates to [html.TabIndex].
func TabIndex(value int) PropOption { return html.TabIndex(value) }

// Disabled delegates to [html.Disabled].
func Disabled(values ...bool) PropOption { return html.Disabled(values...) }

// Checked delegates to [html.Checked].
func Checked(values ...bool) PropOption { return html.Checked(values...) }

// Selected delegates to [html.Selected].
func Selected(values ...bool) PropOption { return html.Selected(values...) }

// Required delegates to [html.Required].
func Required(values ...bool) PropOption { return html.Required(values...) }

// ReadOnly delegates to [html.ReadOnly].
func ReadOnly(values ...bool) PropOption { return html.ReadOnly(values...) }

// AutoFocus delegates to [html.AutoFocus].
func AutoFocus(values ...bool) PropOption { return html.AutoFocus(values...) }

// DisabledIf delegates to [html.DisabledIf].
func DisabledIf(condition bool) PropOption { return html.DisabledIf(condition) }

// ReadOnlyIf delegates to [html.ReadOnlyIf].
func ReadOnlyIf(condition bool) PropOption { return html.ReadOnlyIf(condition) }

// SelectedIf delegates to [html.SelectedIf].
func SelectedIf(condition bool) PropOption { return html.SelectedIf(condition) }

// Style delegates to [html.Style].
func Style(values map[string]string) PropOption { return html.Style(values) }

// Data delegates to [html.Data].
func Data(name string, value string) PropOption { return html.Data(name, value) }

// Dataset delegates to [html.Dataset].
func Dataset(values map[string]string) PropOption { return html.Dataset(values) }

// Aria delegates to [html.Aria].
func Aria(name string, value string) PropOption { return html.Aria(name, value) }

// AriaSet delegates to [html.AriaSet].
func AriaSet(values map[string]string) PropOption { return html.AriaSet(values) }

// Attr delegates to [html.Attr].
func Attr(key string, value interface{}) PropOption { return html.Attr(key, value) }

// Attrs delegates to [html.Attrs].
func Attrs(values map[string]interface{}) PropOption { return html.Attrs(values) }

// OnClick delegates to [html.OnClick].
func OnClick(callback interface{}) PropOption { return html.OnClick(callback) }

// OnInput delegates to [html.OnInput].
func OnInput(callback interface{}) PropOption { return html.OnInput(callback) }

// OnChange delegates to [html.OnChange].
func OnChange(callback interface{}) PropOption { return html.OnChange(callback) }

// OnSubmit delegates to [html.OnSubmit].
func OnSubmit(callback interface{}) PropOption { return html.OnSubmit(callback) }

// OnKeyDown delegates to [html.OnKeyDown].
func OnKeyDown(callback interface{}) PropOption { return html.OnKeyDown(callback) }

// OnKeyUp delegates to [html.OnKeyUp].
func OnKeyUp(callback interface{}) PropOption { return html.OnKeyUp(callback) }

// OnMouseUp delegates to [html.OnMouseUp].
func OnMouseUp(callback interface{}) PropOption { return html.OnMouseUp(callback) }

// OnFocus delegates to [html.OnFocus].
func OnFocus(callback interface{}) PropOption { return html.OnFocus(callback) }

// OnBlur delegates to [html.OnBlur].
func OnBlur(callback interface{}) PropOption { return html.OnBlur(callback) }

// Prevent delegates to [html.Prevent].
func Prevent(callback interface{}) interface{} { return html.Prevent(callback) }

// Stop delegates to [html.Stop].
func Stop(callback interface{}) interface{} { return html.Stop(callback) }

// Debounce delegates to [html.Debounce].
func Debounce(delay time.Duration, callback interface{}) interface{} {
	return html.Debounce(delay, callback)
}

// Throttle delegates to [html.Throttle].
func Throttle(interval time.Duration, callback interface{}) interface{} {
	return html.Throttle(interval, callback)
}

func splitArgs(args ...interface{}) (Props, []ui.Node) {
	var props Props
	childInputs := make([]interface{}, 0, len(args))
	for _, arg := range args {
		switch typed := arg.(type) {
		case nil:
			continue
		case propsInput:
			props = typed.value
		case PropOption:
			props = html.WithProps(props, typed)
		case []PropOption:
			props = html.WithProps(props, typed...)
		default:
			childInputs = append(childInputs, arg)
		}
	}
	return props, html.Children(childInputs...)
}
