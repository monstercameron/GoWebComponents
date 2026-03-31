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
func FromProps(parseProps Props) interface{} {
	return propsInput{value: parseProps}
}

// Tag builds an arbitrary host element from mixed prop options and children.
func Tag(parseName string, parseArgs ...interface{}) ui.Node {
	parseProps, parseChildren := splitArgs(parseArgs...)
	return html.Tag(parseName, parseProps, parseChildren...)
}

// Fragment groups mixed children without introducing a host element.
func Fragment(parseArgs ...interface{}) ui.Node {
	return html.Fragment(html.Children(parseArgs...)...)
}

// A delegates to [html.A].
func A(parseArgs ...interface{}) ui.Node { return Tag("a", parseArgs...) }

// Article delegates to [html.Article].
func Article(parseArgs ...interface{}) ui.Node { return Tag("article", parseArgs...) }

// Body delegates to [html.Body].
func Body(parseArgs ...interface{}) ui.Node { return Tag("body", parseArgs...) }

// Button delegates to [html.Button].
func Button(parseArgs ...interface{}) ui.Node { return Tag("button", parseArgs...) }

// Br delegates to [html.Br].
func Br(parseArgs ...interface{}) ui.Node { return Tag("br", parseArgs...) }

// Code delegates to [html.Code].
func Code(parseArgs ...interface{}) ui.Node { return Tag("code", parseArgs...) }

// Details delegates to [html.Details].
func Details(parseArgs ...interface{}) ui.Node { return Tag("details", parseArgs...) }

// Div delegates to [html.Div].
func Div(parseArgs ...interface{}) ui.Node { return Tag("div", parseArgs...) }

// Form delegates to [html.Form].
func Form(parseArgs ...interface{}) ui.Node { return Tag("form", parseArgs...) }

// H1 delegates to [html.H1].
func H1(parseArgs ...interface{}) ui.Node { return Tag("h1", parseArgs...) }

// H2 delegates to [html.H2].
func H2(parseArgs ...interface{}) ui.Node { return Tag("h2", parseArgs...) }

// H3 delegates to [html.H3].
func H3(parseArgs ...interface{}) ui.Node { return Tag("h3", parseArgs...) }

// Head delegates to [html.Head].
func Head(parseArgs ...interface{}) ui.Node { return Tag("head", parseArgs...) }

// Header delegates to [html.Header].
func Header(parseArgs ...interface{}) ui.Node { return Tag("header", parseArgs...) }

// Hr delegates to [html.Hr].
func Hr(parseArgs ...interface{}) ui.Node { return Tag("hr", parseArgs...) }

// Html delegates to [html.Html].
func Html(parseArgs ...interface{}) ui.Node { return Tag("html", parseArgs...) }

// Img delegates to [html.Img].
func Img(parseArgs ...interface{}) ui.Node { return Tag("img", parseArgs...) }

// Input delegates to [html.Input].
func Input(parseArgs ...interface{}) ui.Node { return Tag("input", parseArgs...) }

// Label delegates to [html.Label].
func Label(parseArgs ...interface{}) ui.Node { return Tag("label", parseArgs...) }

// Li delegates to [html.Li].
func Li(parseArgs ...interface{}) ui.Node { return Tag("li", parseArgs...) }

// Main delegates to [html.Main].
func Main(parseArgs ...interface{}) ui.Node { return Tag("main", parseArgs...) }

// Mark delegates to [html.Mark].
func Mark(parseArgs ...interface{}) ui.Node { return Tag("mark", parseArgs...) }

// Meta delegates to [html.Meta].
func Meta(parseArgs ...interface{}) ui.Node { return Tag("meta", parseArgs...) }

// NoScript delegates to [html.NoScript].
func NoScript(parseArgs ...interface{}) ui.Node { return Tag("noscript", parseArgs...) }

// Option delegates to [html.Option].
func Option(parseArgs ...interface{}) ui.Node { return Tag("option", parseArgs...) }

// P delegates to [html.P].
func P(parseArgs ...interface{}) ui.Node { return Tag("p", parseArgs...) }

// Pre delegates to [html.Pre].
func Pre(parseArgs ...interface{}) ui.Node { return Tag("pre", parseArgs...) }

// Script delegates to [html.Script].
func Script(parseArgs ...interface{}) ui.Node { return Tag("script", parseArgs...) }

// Section delegates to [html.Section].
func Section(parseArgs ...interface{}) ui.Node { return Tag("section", parseArgs...) }

// Select delegates to [html.Select].
func Select(parseArgs ...interface{}) ui.Node { return Tag("select", parseArgs...) }

// Span delegates to [html.Span].
func Span(parseArgs ...interface{}) ui.Node { return Tag("span", parseArgs...) }

// Summary delegates to [html.Summary].
func Summary(parseArgs ...interface{}) ui.Node { return Tag("summary", parseArgs...) }

// Table delegates to [html.Table].
func Table(parseArgs ...interface{}) ui.Node { return Tag("table", parseArgs...) }

// Tbody delegates to [html.Tbody].
func Tbody(parseArgs ...interface{}) ui.Node { return Tag("tbody", parseArgs...) }

// Td delegates to [html.Td].
func Td(parseArgs ...interface{}) ui.Node { return Tag("td", parseArgs...) }

// Th delegates to [html.Th].
func Th(parseArgs ...interface{}) ui.Node { return Tag("th", parseArgs...) }

// Thead delegates to [html.Thead].
func Thead(parseArgs ...interface{}) ui.Node { return Tag("thead", parseArgs...) }

// Tr delegates to [html.Tr].
func Tr(parseArgs ...interface{}) ui.Node { return Tag("tr", parseArgs...) }

// Ul delegates to [html.Ul].
func Ul(parseArgs ...interface{}) ui.Node { return Tag("ul", parseArgs...) }

// Text delegates to [html.Text].
func Text(parseContent interface{}) ui.Node { return html.Text(parseContent) }

// Textf delegates to [html.Textf].
func Textf(format string, parseArgs ...interface{}) ui.Node { return html.Textf(format, parseArgs...) }

// TextIf delegates to [html.TextIf].
func TextIf(isCondition bool, parseContent interface{}) ui.Node {
	return html.TextIf(isCondition, parseContent)
}

// Children delegates to [html.Children].
func Children(parseValues ...interface{}) []ui.Node { return html.Children(parseValues...) }

// When delegates to [html.When].
func When(isCondition bool, parseClassName string) string {
	return html.When(isCondition, parseClassName)
}

// ClassNames delegates to [html.ClassNames].
func ClassNames(parseParts ...interface{}) string { return html.ClassNames(parseParts...) }

// If delegates to [html.If].
func If(isCondition bool, parseNode ui.Node) ui.Node { return html.If(isCondition, parseNode) }

// IfElse delegates to [html.IfElse].
func IfElse(isCondition bool, parseWhenTrue ui.Node, parseWhenFalse ui.Node) ui.Node {
	return html.IfElse(isCondition, parseWhenTrue, parseWhenFalse)
}

// Unless delegates to [html.Unless].
func Unless(isCondition bool, parseNode ui.Node) ui.Node { return html.Unless(isCondition, parseNode) }

// WithKey delegates to [html.WithKey].
func WithKey(parseNode ui.Node, parseKey interface{}) ui.Node {
	return html.WithKey(parseNode, parseKey)
}

// Case delegates to [html.Case].
func Case(parseValue interface{}, parseNode ui.Node) SwitchBranch {
	return html.Case(parseValue, parseNode)
}

// Default delegates to [html.Default].
func Default(parseNode ui.Node) SwitchBranch { return html.Default(parseNode) }

// Switch delegates to [html.Switch].
func Switch(parseValue interface{}, parseBranches ...SwitchBranch) ui.Node {
	return html.Switch(parseValue, parseBranches...)
}

// PropsOf delegates to [html.PropsOf].
func PropsOf(parseOptions ...PropOption) Props { return html.PropsOf(parseOptions...) }

// WithProps delegates to [html.WithProps].
func WithProps(parseBase Props, parseOptions ...PropOption) Props {
	return html.WithProps(parseBase, parseOptions...)
}

// Map delegates to [html.Map].
func Map[T any](parseItems []T, render func(T) ui.Node) []ui.Node {
	return html.Map(parseItems, render)
}

// MapKeyed delegates to [html.MapKeyed].
func MapKeyed[T any](parseItems []T, parseKey func(T) interface{}, render func(T) ui.Node) []ui.Node {
	return html.MapKeyed(parseItems, parseKey, render)
}

// FlatMap delegates to [html.FlatMap].
func FlatMap[T any](parseItems []T, render func(T) []ui.Node) []ui.Node {
	return html.FlatMap(parseItems, render)
}

// FilterMap delegates to [html.FilterMap].
func FilterMap[T any](parseItems []T, render func(T) (ui.Node, bool)) []ui.Node {
	return html.FilterMap(parseItems, render)
}

// Join delegates to [html.Join].
func Join(parseSeparator ui.Node, parseNodes ...ui.Node) []ui.Node {
	return html.Join(parseSeparator, parseNodes...)
}

// Maybe delegates to [html.Maybe].
func Maybe[T any](parseValue *T, render func(T) ui.Node) ui.Node {
	return html.Maybe(parseValue, render)
}

// OrElse delegates to [html.OrElse].
func OrElse[T any](parseValue *T, parseFallback T) T { return html.OrElse(parseValue, parseFallback) }

// Coalesce delegates to [html.Coalesce].
func Coalesce[T any](parseValues ...*T) *T { return html.Coalesce(parseValues...) }

// ID delegates to [html.ID].
func ID(parseValue string) PropOption { return html.ID(parseValue) }

// Class delegates to [html.Class].
func Class(parseValue string) PropOption { return html.Class(parseValue) }

// For delegates to [html.For].
func For(parseValue string) PropOption { return html.For(parseValue) }

// Name delegates to [html.Name].
func Name(parseValue string) PropOption { return html.Name(parseValue) }

// Title delegates to [html.Title].
func Title(parseValue string) PropOption { return html.Title(parseValue) }

// Value delegates to [html.Value].
func Value(parseValue string) PropOption { return html.Value(parseValue) }

// Placeholder delegates to [html.Placeholder].
func Placeholder(parseValue string) PropOption { return html.Placeholder(parseValue) }

// Type delegates to [html.Type].
func Type(parseValue string) PropOption { return html.Type(parseValue) }

// Href delegates to [html.Href].
func Href(parseValue string) PropOption { return html.Href(parseValue) }

// Src delegates to [html.Src].
func Src(parseValue string) PropOption { return html.Src(parseValue) }

// Role delegates to [html.Role].
func Role(parseValue string) PropOption { return html.Role(parseValue) }

// Rows delegates to [html.Rows].
func Rows(parseValue int) PropOption { return html.Rows(parseValue) }

// TabIndex delegates to [html.TabIndex].
func TabIndex(parseValue int) PropOption { return html.TabIndex(parseValue) }

// Disabled delegates to [html.Disabled].
func Disabled(parseValues ...bool) PropOption { return html.Disabled(parseValues...) }

// Checked delegates to [html.Checked].
func Checked(parseValues ...bool) PropOption { return html.Checked(parseValues...) }

// Selected delegates to [html.Selected].
func Selected(parseValues ...bool) PropOption { return html.Selected(parseValues...) }

// Required delegates to [html.Required].
func Required(parseValues ...bool) PropOption { return html.Required(parseValues...) }

// ReadOnly delegates to [html.ReadOnly].
func ReadOnly(parseValues ...bool) PropOption { return html.ReadOnly(parseValues...) }

// AutoFocus delegates to [html.AutoFocus].
func AutoFocus(parseValues ...bool) PropOption { return html.AutoFocus(parseValues...) }

// DisabledIf delegates to [html.DisabledIf].
func DisabledIf(isCondition bool) PropOption { return html.DisabledIf(isCondition) }

// ReadOnlyIf delegates to [html.ReadOnlyIf].
func ReadOnlyIf(isCondition bool) PropOption { return html.ReadOnlyIf(isCondition) }

// SelectedIf delegates to [html.SelectedIf].
func SelectedIf(isCondition bool) PropOption { return html.SelectedIf(isCondition) }

// Style delegates to [html.Style].
func Style(parseValues map[string]string) PropOption { return html.Style(parseValues) }

// Data delegates to [html.Data].
func Data(parseName string, parseValue string) PropOption { return html.Data(parseName, parseValue) }

// Dataset delegates to [html.Dataset].
func Dataset(parseValues map[string]string) PropOption { return html.Dataset(parseValues) }

// Aria delegates to [html.Aria].
func Aria(parseName string, parseValue string) PropOption { return html.Aria(parseName, parseValue) }

// AriaSet delegates to [html.AriaSet].
func AriaSet(parseValues map[string]string) PropOption { return html.AriaSet(parseValues) }

// Attr delegates to [html.Attr].
func Attr(parseKey string, parseValue interface{}) PropOption { return html.Attr(parseKey, parseValue) }

// Attrs delegates to [html.Attrs].
func Attrs(parseValues map[string]interface{}) PropOption { return html.Attrs(parseValues) }

// OnClick delegates to [html.OnClick].
func OnClick(parseCallback interface{}) PropOption { return html.OnClick(parseCallback) }

// OnInput delegates to [html.OnInput].
func OnInput(parseCallback interface{}) PropOption { return html.OnInput(parseCallback) }

// OnChange delegates to [html.OnChange].
func OnChange(parseCallback interface{}) PropOption { return html.OnChange(parseCallback) }

// OnSubmit delegates to [html.OnSubmit].
func OnSubmit(parseCallback interface{}) PropOption { return html.OnSubmit(parseCallback) }

// OnKeyDown delegates to [html.OnKeyDown].
func OnKeyDown(parseCallback interface{}) PropOption { return html.OnKeyDown(parseCallback) }

// OnKeyUp delegates to [html.OnKeyUp].
func OnKeyUp(parseCallback interface{}) PropOption { return html.OnKeyUp(parseCallback) }

// OnMouseUp delegates to [html.OnMouseUp].
func OnMouseUp(parseCallback interface{}) PropOption   { return html.OnMouseUp(parseCallback) }
func OnMouseDown(parseCallback interface{}) PropOption { return html.OnMouseDown(parseCallback) }

// OnFocus delegates to [html.OnFocus].
func OnFocus(parseCallback interface{}) PropOption { return html.OnFocus(parseCallback) }

// OnBlur delegates to [html.OnBlur].
func OnBlur(parseCallback interface{}) PropOption { return html.OnBlur(parseCallback) }

// OnScroll delegates to [html.OnScroll].
func OnScroll(parseCallback interface{}) PropOption { return html.OnScroll(parseCallback) }

// Prevent delegates to [html.Prevent].
func Prevent(parseCallback interface{}) interface{} { return html.Prevent(parseCallback) }

// Stop delegates to [html.Stop].
func Stop(parseCallback interface{}) interface{} { return html.Stop(parseCallback) }

// Debounce delegates to [html.Debounce].
func Debounce(parseDelay time.Duration, parseCallback interface{}) interface{} {
	return html.Debounce(parseDelay, parseCallback)
}

// Throttle delegates to [html.Throttle].
func Throttle(parseInterval time.Duration, parseCallback interface{}) interface{} {
	return html.Throttle(parseInterval, parseCallback)
}

// splitArgs is a core package helper.
func splitArgs(parseArgs ...interface{}) (Props, []ui.Node) {
	var parseProps Props
	parseChildInputs := make([]interface{}, 0, len(parseArgs))
	for _, parseArg := range parseArgs {
		switch parseTyped := parseArg.(type) {
		case nil:
			continue
		case propsInput:
			parseProps = parseTyped.value
		case PropOption:
			parseProps = html.WithProps(parseProps, parseTyped)
		case []PropOption:
			parseProps = html.WithProps(parseProps, parseTyped...)
		default:
			parseChildInputs = append(parseChildInputs, parseArg)
		}
	}
	return parseProps, html.Children(parseChildInputs...)
}
