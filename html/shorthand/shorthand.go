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

func A(args ...interface{}) ui.Node        { return Tag("a", args...) }
func Article(args ...interface{}) ui.Node  { return Tag("article", args...) }
func Body(args ...interface{}) ui.Node     { return Tag("body", args...) }
func Button(args ...interface{}) ui.Node   { return Tag("button", args...) }
func Br(args ...interface{}) ui.Node       { return Tag("br", args...) }
func Code(args ...interface{}) ui.Node     { return Tag("code", args...) }
func Details(args ...interface{}) ui.Node  { return Tag("details", args...) }
func Div(args ...interface{}) ui.Node      { return Tag("div", args...) }
func Form(args ...interface{}) ui.Node     { return Tag("form", args...) }
func H1(args ...interface{}) ui.Node       { return Tag("h1", args...) }
func H2(args ...interface{}) ui.Node       { return Tag("h2", args...) }
func H3(args ...interface{}) ui.Node       { return Tag("h3", args...) }
func Head(args ...interface{}) ui.Node     { return Tag("head", args...) }
func Header(args ...interface{}) ui.Node   { return Tag("header", args...) }
func Hr(args ...interface{}) ui.Node       { return Tag("hr", args...) }
func Html(args ...interface{}) ui.Node     { return Tag("html", args...) }
func Img(args ...interface{}) ui.Node      { return Tag("img", args...) }
func Input(args ...interface{}) ui.Node    { return Tag("input", args...) }
func Label(args ...interface{}) ui.Node    { return Tag("label", args...) }
func Li(args ...interface{}) ui.Node       { return Tag("li", args...) }
func Main(args ...interface{}) ui.Node     { return Tag("main", args...) }
func Mark(args ...interface{}) ui.Node     { return Tag("mark", args...) }
func Meta(args ...interface{}) ui.Node     { return Tag("meta", args...) }
func NoScript(args ...interface{}) ui.Node { return Tag("noscript", args...) }
func Option(args ...interface{}) ui.Node   { return Tag("option", args...) }
func P(args ...interface{}) ui.Node        { return Tag("p", args...) }
func Pre(args ...interface{}) ui.Node      { return Tag("pre", args...) }
func Script(args ...interface{}) ui.Node   { return Tag("script", args...) }
func Section(args ...interface{}) ui.Node  { return Tag("section", args...) }
func Select(args ...interface{}) ui.Node   { return Tag("select", args...) }
func Span(args ...interface{}) ui.Node     { return Tag("span", args...) }
func Summary(args ...interface{}) ui.Node  { return Tag("summary", args...) }
func Table(args ...interface{}) ui.Node    { return Tag("table", args...) }
func Tbody(args ...interface{}) ui.Node    { return Tag("tbody", args...) }
func Td(args ...interface{}) ui.Node       { return Tag("td", args...) }
func Th(args ...interface{}) ui.Node       { return Tag("th", args...) }
func Thead(args ...interface{}) ui.Node    { return Tag("thead", args...) }
func Tr(args ...interface{}) ui.Node       { return Tag("tr", args...) }
func Ul(args ...interface{}) ui.Node       { return Tag("ul", args...) }

func Text(content interface{}) ui.Node                   { return html.Text(content) }
func Textf(format string, args ...interface{}) ui.Node   { return html.Textf(format, args...) }
func TextIf(condition bool, content interface{}) ui.Node { return html.TextIf(condition, content) }
func Children(values ...interface{}) []ui.Node           { return html.Children(values...) }
func When(condition bool, className string) string       { return html.When(condition, className) }
func ClassNames(parts ...interface{}) string             { return html.ClassNames(parts...) }
func If(condition bool, node ui.Node) ui.Node            { return html.If(condition, node) }
func IfElse(condition bool, whenTrue ui.Node, whenFalse ui.Node) ui.Node {
	return html.IfElse(condition, whenTrue, whenFalse)
}
func Unless(condition bool, node ui.Node) ui.Node       { return html.Unless(condition, node) }
func WithKey(node ui.Node, key interface{}) ui.Node     { return html.WithKey(node, key) }
func Case(value interface{}, node ui.Node) SwitchBranch { return html.Case(value, node) }
func Default(node ui.Node) SwitchBranch                 { return html.Default(node) }
func Switch(value interface{}, branches ...SwitchBranch) ui.Node {
	return html.Switch(value, branches...)
}
func PropsOf(options ...PropOption) Props               { return html.PropsOf(options...) }
func WithProps(base Props, options ...PropOption) Props { return html.WithProps(base, options...) }

func Map[T any](items []T, render func(T) ui.Node) []ui.Node { return html.Map(items, render) }
func MapKeyed[T any](items []T, key func(T) interface{}, render func(T) ui.Node) []ui.Node {
	return html.MapKeyed(items, key, render)
}
func FlatMap[T any](items []T, render func(T) []ui.Node) []ui.Node {
	return html.FlatMap(items, render)
}
func FilterMap[T any](items []T, render func(T) (ui.Node, bool)) []ui.Node {
	return html.FilterMap(items, render)
}
func Join(separator ui.Node, nodes ...ui.Node) []ui.Node    { return html.Join(separator, nodes...) }
func Maybe[T any](value *T, render func(T) ui.Node) ui.Node { return html.Maybe(value, render) }
func OrElse[T any](value *T, fallback T) T                  { return html.OrElse(value, fallback) }
func Coalesce[T any](values ...*T) *T                       { return html.Coalesce(values...) }

func ID(value string) PropOption                     { return html.ID(value) }
func Class(value string) PropOption                  { return html.Class(value) }
func For(value string) PropOption                    { return html.For(value) }
func Name(value string) PropOption                   { return html.Name(value) }
func Title(value string) PropOption                  { return html.Title(value) }
func Value(value string) PropOption                  { return html.Value(value) }
func Placeholder(value string) PropOption            { return html.Placeholder(value) }
func Type(value string) PropOption                   { return html.Type(value) }
func Href(value string) PropOption                   { return html.Href(value) }
func Src(value string) PropOption                    { return html.Src(value) }
func Role(value string) PropOption                   { return html.Role(value) }
func Rows(value int) PropOption                      { return html.Rows(value) }
func TabIndex(value int) PropOption                  { return html.TabIndex(value) }
func Disabled(values ...bool) PropOption             { return html.Disabled(values...) }
func Checked(values ...bool) PropOption              { return html.Checked(values...) }
func Selected(values ...bool) PropOption             { return html.Selected(values...) }
func Required(values ...bool) PropOption             { return html.Required(values...) }
func ReadOnly(values ...bool) PropOption             { return html.ReadOnly(values...) }
func AutoFocus(values ...bool) PropOption            { return html.AutoFocus(values...) }
func DisabledIf(condition bool) PropOption           { return html.DisabledIf(condition) }
func ReadOnlyIf(condition bool) PropOption           { return html.ReadOnlyIf(condition) }
func SelectedIf(condition bool) PropOption           { return html.SelectedIf(condition) }
func Style(values map[string]string) PropOption      { return html.Style(values) }
func Data(name string, value string) PropOption      { return html.Data(name, value) }
func Dataset(values map[string]string) PropOption    { return html.Dataset(values) }
func Aria(name string, value string) PropOption      { return html.Aria(name, value) }
func AriaSet(values map[string]string) PropOption    { return html.AriaSet(values) }
func Attr(key string, value interface{}) PropOption  { return html.Attr(key, value) }
func Attrs(values map[string]interface{}) PropOption { return html.Attrs(values) }
func OnClick(callback interface{}) PropOption        { return html.OnClick(callback) }
func OnInput(callback interface{}) PropOption        { return html.OnInput(callback) }
func OnChange(callback interface{}) PropOption       { return html.OnChange(callback) }
func OnSubmit(callback interface{}) PropOption       { return html.OnSubmit(callback) }
func OnKeyDown(callback interface{}) PropOption      { return html.OnKeyDown(callback) }
func OnKeyUp(callback interface{}) PropOption        { return html.OnKeyUp(callback) }
func OnMouseUp(callback interface{}) PropOption      { return html.OnMouseUp(callback) }
func OnFocus(callback interface{}) PropOption        { return html.OnFocus(callback) }
func OnBlur(callback interface{}) PropOption         { return html.OnBlur(callback) }

func Prevent(callback interface{}) interface{} { return html.Prevent(callback) }
func Stop(callback interface{}) interface{}    { return html.Stop(callback) }
func Debounce(delay time.Duration, callback interface{}) interface{} {
	return html.Debounce(delay, callback)
}
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
