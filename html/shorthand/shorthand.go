package shorthand

import (
	"maps"
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
func FromProps(parseProps Props) any {
	return propsInput{value: parseProps}
}

// Tag builds an arbitrary host element from mixed prop options and children.
func Tag(parseName string, parseArgs ...any) ui.Node {
	parseProps, parseChildren := splitArgs(parseArgs...)
	return html.Tag(parseName, parseProps, parseChildren...)
}

// Fragment groups mixed children without introducing a host element.
func Fragment(parseArgs ...any) ui.Node {
	return html.Fragment(html.Children(parseArgs...)...)
}

// A delegates to [html.A].
func A(parseArgs ...any) ui.Node { return Tag("a", parseArgs...) }

// Article delegates to [html.Article].
func Article(parseArgs ...any) ui.Node { return Tag("article", parseArgs...) }

// Aside delegates to [html.Aside].
func Aside(parseArgs ...any) ui.Node { return Tag("aside", parseArgs...) }

// Blockquote delegates to [html.Blockquote].
func Blockquote(parseArgs ...any) ui.Node { return Tag("blockquote", parseArgs...) }

// Body delegates to [html.Body].
func Body(parseArgs ...any) ui.Node { return Tag("body", parseArgs...) }

// Button delegates to [html.Button].
func Button(parseArgs ...any) ui.Node { return Tag("button", parseArgs...) }

// Br delegates to [html.Br].
func Br(parseArgs ...any) ui.Node { return Tag("br", parseArgs...) }

// Code delegates to [html.Code].
func Code(parseArgs ...any) ui.Node { return Tag("code", parseArgs...) }

// Details delegates to [html.Details].
func Details(parseArgs ...any) ui.Node { return Tag("details", parseArgs...) }

// Dialog delegates to [html.Dialog].
func Dialog(parseArgs ...any) ui.Node { return Tag("dialog", parseArgs...) }

// Div delegates to [html.Div].
func Div(parseArgs ...any) ui.Node { return Tag("div", parseArgs...) }

// Em delegates to [html.Em].
func Em(parseArgs ...any) ui.Node { return Tag("em", parseArgs...) }

// Fieldset delegates to [html.Fieldset].
func Fieldset(parseArgs ...any) ui.Node { return Tag("fieldset", parseArgs...) }

// Footer delegates to [html.Footer].
func Footer(parseArgs ...any) ui.Node { return Tag("footer", parseArgs...) }

// Form delegates to [html.Form].
func Form(parseArgs ...any) ui.Node { return Tag("form", parseArgs...) }

// H1 delegates to [html.H1].
func H1(parseArgs ...any) ui.Node { return Tag("h1", parseArgs...) }

// H2 delegates to [html.H2].
func H2(parseArgs ...any) ui.Node { return Tag("h2", parseArgs...) }

// H3 delegates to [html.H3].
func H3(parseArgs ...any) ui.Node { return Tag("h3", parseArgs...) }

// H4 delegates to [html.H4].
func H4(parseArgs ...any) ui.Node { return Tag("h4", parseArgs...) }

// H5 delegates to [html.H5].
func H5(parseArgs ...any) ui.Node { return Tag("h5", parseArgs...) }

// H6 delegates to [html.H6].
func H6(parseArgs ...any) ui.Node { return Tag("h6", parseArgs...) }

// Head delegates to [html.Head].
func Head(parseArgs ...any) ui.Node { return Tag("head", parseArgs...) }

// Header delegates to [html.Header].
func Header(parseArgs ...any) ui.Node { return Tag("header", parseArgs...) }

// Hr delegates to [html.Hr].
func Hr(parseArgs ...any) ui.Node { return Tag("hr", parseArgs...) }

// Html delegates to [html.Html].
func Html(parseArgs ...any) ui.Node { return Tag("html", parseArgs...) }

// HiddenInput creates a hidden input element with the given name and value,
// delegating to [html.HiddenInput].
func HiddenInput(parseName string, parseValue string) ui.Node {
	return html.HiddenInput(parseName, parseValue)
}

// Img delegates to [html.Img].
func Img(parseArgs ...any) ui.Node { return Tag("img", parseArgs...) }

// Iframe delegates to [html.Iframe].
func Iframe(parseArgs ...any) ui.Node { return Tag("iframe", parseArgs...) }

// Input delegates to [html.Input].
func Input(parseArgs ...any) ui.Node { return Tag("input", parseArgs...) }

// Label delegates to [html.Label].
func Label(parseArgs ...any) ui.Node { return Tag("label", parseArgs...) }

// Legend delegates to [html.Legend].
func Legend(parseArgs ...any) ui.Node { return Tag("legend", parseArgs...) }

// Li delegates to [html.Li].
func Li(parseArgs ...any) ui.Node { return Tag("li", parseArgs...) }

// Main delegates to [html.Main].
func Main(parseArgs ...any) ui.Node { return Tag("main", parseArgs...) }

// Mark delegates to [html.Mark].
func Mark(parseArgs ...any) ui.Node { return Tag("mark", parseArgs...) }

// Meta delegates to [html.Meta].
func Meta(parseArgs ...any) ui.Node { return Tag("meta", parseArgs...) }

// Nav delegates to [html.Nav].
func Nav(parseArgs ...any) ui.Node { return Tag("nav", parseArgs...) }

// NoScript delegates to [html.NoScript].
func NoScript(parseArgs ...any) ui.Node { return Tag("noscript", parseArgs...) }

// Option delegates to [html.Option].
func Option(parseArgs ...any) ui.Node { return Tag("option", parseArgs...) }

// P delegates to [html.P].
func P(parseArgs ...any) ui.Node { return Tag("p", parseArgs...) }

// Pre delegates to [html.Pre].
func Pre(parseArgs ...any) ui.Node { return Tag("pre", parseArgs...) }

// Script delegates to [html.Script].
func Script(parseArgs ...any) ui.Node { return Tag("script", parseArgs...) }

// Section delegates to [html.Section].
func Section(parseArgs ...any) ui.Node { return Tag("section", parseArgs...) }

// Select delegates to [html.Select].
func Select(parseArgs ...any) ui.Node { return Tag("select", parseArgs...) }

// Small delegates to [html.Small].
func Small(parseArgs ...any) ui.Node { return Tag("small", parseArgs...) }

// Span delegates to [html.Span].
func Span(parseArgs ...any) ui.Node { return Tag("span", parseArgs...) }

// Strong delegates to [html.Strong].
func Strong(parseArgs ...any) ui.Node { return Tag("strong", parseArgs...) }

// Summary delegates to [html.Summary].
func Summary(parseArgs ...any) ui.Node { return Tag("summary", parseArgs...) }

// Table delegates to [html.Table].
func Table(parseArgs ...any) ui.Node { return Tag("table", parseArgs...) }

// Tbody delegates to [html.Tbody].
func Tbody(parseArgs ...any) ui.Node { return Tag("tbody", parseArgs...) }

// Td delegates to [html.Td].
func Td(parseArgs ...any) ui.Node { return Tag("td", parseArgs...) }

// Th delegates to [html.Th].
func Th(parseArgs ...any) ui.Node { return Tag("th", parseArgs...) }

// Thead delegates to [html.Thead].
func Thead(parseArgs ...any) ui.Node { return Tag("thead", parseArgs...) }

// Textarea delegates to [html.Textarea].
func Textarea(parseArgs ...any) ui.Node { return Tag("textarea", parseArgs...) }

// Time delegates to [html.Time].
func Time(parseArgs ...any) ui.Node { return Tag("time", parseArgs...) }

// Tr delegates to [html.Tr].
func Tr(parseArgs ...any) ui.Node { return Tag("tr", parseArgs...) }

// Ul delegates to [html.Ul].
func Ul(parseArgs ...any) ui.Node { return Tag("ul", parseArgs...) }

// Ol delegates to [html.Ol].
func Ol(parseArgs ...any) ui.Node { return Tag("ol", parseArgs...) }

// Tfoot delegates to [html.Tfoot].
func Tfoot(parseArgs ...any) ui.Node { return Tag("tfoot", parseArgs...) }

// Caption delegates to [html.Caption].
func Caption(parseArgs ...any) ui.Node { return Tag("caption", parseArgs...) }

// Colgroup delegates to [html.Colgroup].
func Colgroup(parseArgs ...any) ui.Node { return Tag("colgroup", parseArgs...) }

// Col delegates to [html.Col].
func Col(parseArgs ...any) ui.Node { return Tag("col", parseArgs...) }

// Video delegates to [html.Video].
func Video(parseArgs ...any) ui.Node { return Tag("video", parseArgs...) }

// Audio delegates to [html.Audio].
func Audio(parseArgs ...any) ui.Node { return Tag("audio", parseArgs...) }

// Source delegates to [html.Source].
func Source(parseArgs ...any) ui.Node { return Tag("source", parseArgs...) }

// Track delegates to [html.Track].
func Track(parseArgs ...any) ui.Node { return Tag("track", parseArgs...) }

// Canvas delegates to [html.Canvas].
func Canvas(parseArgs ...any) ui.Node { return Tag("canvas", parseArgs...) }

// Optgroup delegates to [html.Optgroup].
func Optgroup(parseArgs ...any) ui.Node { return Tag("optgroup", parseArgs...) }

// Datalist delegates to [html.Datalist].
func Datalist(parseArgs ...any) ui.Node { return Tag("datalist", parseArgs...) }

// Output delegates to [html.Output].
func Output(parseArgs ...any) ui.Node { return Tag("output", parseArgs...) }

// Progress delegates to [html.Progress].
func Progress(parseArgs ...any) ui.Node { return Tag("progress", parseArgs...) }

// Meter delegates to [html.Meter].
func Meter(parseArgs ...any) ui.Node { return Tag("meter", parseArgs...) }

// Figure delegates to [html.Figure].
func Figure(parseArgs ...any) ui.Node { return Tag("figure", parseArgs...) }

// Figcaption delegates to [html.Figcaption].
func Figcaption(parseArgs ...any) ui.Node { return Tag("figcaption", parseArgs...) }

// Picture delegates to [html.Picture].
func Picture(parseArgs ...any) ui.Node { return Tag("picture", parseArgs...) }

// Abbr delegates to [html.Abbr].
func Abbr(parseArgs ...any) ui.Node { return Tag("abbr", parseArgs...) }

// Kbd delegates to [html.Kbd].
func Kbd(parseArgs ...any) ui.Node { return Tag("kbd", parseArgs...) }

// Sub delegates to [html.Sub].
func Sub(parseArgs ...any) ui.Node { return Tag("sub", parseArgs...) }

// Sup delegates to [html.Sup].
func Sup(parseArgs ...any) ui.Node { return Tag("sup", parseArgs...) }

// Del delegates to [html.Del].
func Del(parseArgs ...any) ui.Node { return Tag("del", parseArgs...) }

// Ins delegates to [html.Ins].
func Ins(parseArgs ...any) ui.Node { return Tag("ins", parseArgs...) }

// B delegates to [html.B].
func B(parseArgs ...any) ui.Node { return Tag("b", parseArgs...) }

// I delegates to [html.I].
func I(parseArgs ...any) ui.Node { return Tag("i", parseArgs...) }

// U delegates to [html.U].
func U(parseArgs ...any) ui.Node { return Tag("u", parseArgs...) }

// Svg delegates to [html.Svg].
func Svg(parseArgs ...any) ui.Node {
	parseProps, parseChildren := splitArgs(parseArgs...)
	return html.Svg(parseProps, parseChildren...)
}

// Path delegates to [html.Path].
func Path(parseArgs ...any) ui.Node { return Tag("path", parseArgs...) }

// Circle delegates to [html.Circle].
func Circle(parseArgs ...any) ui.Node { return Tag("circle", parseArgs...) }

// Rect delegates to [html.Rect].
func Rect(parseArgs ...any) ui.Node { return Tag("rect", parseArgs...) }

// G delegates to [html.G].
func G(parseArgs ...any) ui.Node { return Tag("g", parseArgs...) }

// Line delegates to [html.Line].
func Line(parseArgs ...any) ui.Node { return Tag("line", parseArgs...) }

// Polyline delegates to [html.Polyline].
func Polyline(parseArgs ...any) ui.Node { return Tag("polyline", parseArgs...) }

// Polygon delegates to [html.Polygon].
func Polygon(parseArgs ...any) ui.Node { return Tag("polygon", parseArgs...) }

// Defs delegates to [html.Defs].
func Defs(parseArgs ...any) ui.Node { return Tag("defs", parseArgs...) }

// Use delegates to [html.Use].
func Use(parseArgs ...any) ui.Node { return Tag("use", parseArgs...) }

// Text delegates to [html.Text].
func Text(parseContent any) ui.Node { return html.Text(parseContent) }

// Textf delegates to [html.Textf].
func Textf(format string, parseArgs ...any) ui.Node { return html.Textf(format, parseArgs...) }

// TextIf delegates to [html.TextIf].
func TextIf(isCondition bool, parseContent any) ui.Node {
	return html.TextIf(isCondition, parseContent)
}

// Children delegates to [html.Children].
func Children(parseValues ...any) []ui.Node { return html.Children(parseValues...) }

// When delegates to [html.When].
func When(isCondition bool, parseClassName string) string {
	return html.When(isCondition, parseClassName)
}

// ClassNames delegates to [html.ClassNames].
func ClassNames(parseParts ...any) string { return html.ClassNames(parseParts...) }

// ClassMap delegates to [html.ClassMap].
func ClassMap(parseValues map[string]bool) string { return html.ClassMap(parseValues) }

// If delegates to [html.If].
func If(isCondition bool, parseNode ui.Node) ui.Node { return html.If(isCondition, parseNode) }

// IfElse delegates to [html.IfElse].
func IfElse(isCondition bool, parseWhenTrue ui.Node, parseWhenFalse ui.Node) ui.Node {
	return html.IfElse(isCondition, parseWhenTrue, parseWhenFalse)
}

// Unless delegates to [html.Unless].
func Unless(isCondition bool, parseNode ui.Node) ui.Node { return html.Unless(isCondition, parseNode) }

// WithKey delegates to [html.WithKey].
func WithKey(parseNode ui.Node, parseKey any) ui.Node {
	return html.WithKey(parseNode, parseKey)
}

// Case delegates to [html.Case].
func Case(parseValue any, parseNode ui.Node) SwitchBranch {
	return html.Case(parseValue, parseNode)
}

// Default delegates to [html.Default].
func Default(parseNode ui.Node) SwitchBranch { return html.Default(parseNode) }

// Switch delegates to [html.Switch].
func Switch(parseValue any, parseBranches ...SwitchBranch) ui.Node {
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
func MapKeyed[T any](parseItems []T, parseKey func(T) any, render func(T) ui.Node) []ui.Node {
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

// CondBranch aliases [html.CondBranch].
type CondBranch = html.CondBranch

// MapIndexed delegates to [html.MapIndexed].
func MapIndexed[T any](parseItems []T, render func(parseIndex int, parseItem T) ui.Node) []ui.Node {
	return html.MapIndexed(parseItems, render)
}

// MapKeyedIndexed delegates to [html.MapKeyedIndexed].
func MapKeyedIndexed[T any](parseItems []T, parseKey func(parseIndex int, parseItem T) any, render func(parseIndex int, parseItem T) ui.Node) []ui.Node {
	return html.MapKeyedIndexed(parseItems, parseKey, render)
}

// MapOr delegates to [html.MapOr].
func MapOr[T any](parseItems []T, render func(T) ui.Node, parseFallback ui.Node) ui.Node {
	return html.MapOr(parseItems, render, parseFallback)
}

// MapKeyedOr delegates to [html.MapKeyedOr].
func MapKeyedOr[T any](parseItems []T, parseKey func(T) any, render func(T) ui.Node, parseFallback ui.Node) ui.Node {
	return html.MapKeyedOr(parseItems, parseKey, render, parseFallback)
}

// Range delegates to [html.Range].
func Range(parseCount int, render func(parseIndex int) ui.Node) []ui.Node {
	return html.Range(parseCount, render)
}

// Repeat delegates to [html.Repeat].
func Repeat(parseCount int, parseNode ui.Node) []ui.Node { return html.Repeat(parseCount, parseNode) }

// MaybeOr delegates to [html.MaybeOr].
func MaybeOr[T any](parseValue *T, render func(T) ui.Node, parseFallback ui.Node) ui.Node {
	return html.MaybeOr(parseValue, render, parseFallback)
}

// Match delegates to [html.Match].
func Match(isCondition bool, parseNode ui.Node) CondBranch { return html.Match(isCondition, parseNode) }

// Otherwise delegates to [html.Otherwise].
func Otherwise(parseNode ui.Node) CondBranch { return html.Otherwise(parseNode) }

// Cond delegates to [html.Cond].
func Cond(parseBranches ...CondBranch) ui.Node { return html.Cond(parseBranches...) }

// MarkdownRenderOptions aliases [html.MarkdownRenderOptions].
type MarkdownRenderOptions = html.MarkdownRenderOptions

// AttrIf delegates to [html.AttrIf].
func AttrIf(isCondition bool, parseKey string, parseValue any) PropOption {
	return html.AttrIf(isCondition, parseKey, parseValue)
}

// ClassIf delegates to [html.ClassIf].
func ClassIf(isCondition bool, parseClass string) PropOption {
	return html.ClassIf(isCondition, parseClass)
}

// StyleIf delegates to [html.StyleIf].
func StyleIf(isCondition bool, parseValues map[string]string) PropOption {
	return html.StyleIf(isCondition, parseValues)
}

// StyleVar delegates to [html.StyleVar].
func StyleVar(parseName string, parseValue string) PropOption {
	return html.StyleVar(parseName, parseValue)
}

// MergeProps delegates to [html.MergeProps].
func MergeProps(parseBase Props, parseOverride Props) Props {
	return html.MergeProps(parseBase, parseOverride)
}

// DefaultProps delegates to [html.DefaultProps].
func DefaultProps(parseProps Props, parseDefaults Props) Props {
	return html.DefaultProps(parseProps, parseDefaults)
}

// Markdown delegates to [html.Markdown].
func Markdown(parseSource string, parseOptions ...MarkdownRenderOptions) []ui.Node {
	return html.Markdown(parseSource, parseOptions...)
}

// TextLines delegates to [html.TextLines].
func TextLines(parseText string) []ui.Node { return html.TextLines(parseText) }

// Show delegates to [html.Show].
func Show(isCondition bool, parseNode ui.Node) ui.Node { return html.Show(isCondition, parseNode) }

// WithChildren delegates to [html.WithChildren].
func WithChildren(parseNode ui.Node, parseChildren ...ui.Node) ui.Node {
	return html.WithChildren(parseNode, parseChildren...)
}

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

// Alt delegates to [html.Alt].
func Alt(parseValue string) PropOption { return html.Alt(parseValue) }

// Role delegates to [html.Role].
func Role(parseValue string) PropOption { return html.Role(parseValue) }

// Lang delegates to [html.Lang].
func Lang(parseValue string) PropOption { return html.Lang(parseValue) }

// Dir delegates to [html.Dir].
func Dir(parseValue string) PropOption { return html.Dir(parseValue) }

// Target delegates to [html.Target].
func Target(parseValue string) PropOption { return html.Target(parseValue) }

// Rel delegates to [html.Rel].
func Rel(parseValue string) PropOption { return html.Rel(parseValue) }

// Accept delegates to [html.Accept].
func Accept(parseValue string) PropOption { return html.Accept(parseValue) }

// AutoComplete delegates to [html.AutoComplete].
func AutoComplete(parseValue string) PropOption { return html.AutoComplete(parseValue) }

// Min delegates to [html.Min].
func Min(parseValue string) PropOption { return html.Min(parseValue) }

// Max delegates to [html.Max].
func Max(parseValue string) PropOption { return html.Max(parseValue) }

// Step delegates to [html.Step].
func Step(parseValue string) PropOption { return html.Step(parseValue) }

// Pattern delegates to [html.Pattern].
func Pattern(parseValue string) PropOption { return html.Pattern(parseValue) }

// MaxLength delegates to [html.MaxLength].
func MaxLength(parseValue int) PropOption { return html.MaxLength(parseValue) }

// MinLength delegates to [html.MinLength].
func MinLength(parseValue int) PropOption { return html.MinLength(parseValue) }

// ColSpan delegates to [html.ColSpan].
func ColSpan(parseValue int) PropOption { return html.ColSpan(parseValue) }

// RowSpan delegates to [html.RowSpan].
func RowSpan(parseValue int) PropOption { return html.RowSpan(parseValue) }

// Width delegates to [html.Width].
func Width(parseValue string) PropOption { return html.Width(parseValue) }

// Height delegates to [html.Height].
func Height(parseValue string) PropOption { return html.Height(parseValue) }

// Loading delegates to [html.Loading].
func Loading(parseValue string) PropOption { return html.Loading(parseValue) }

// Rows delegates to [html.Rows].
func Rows(parseValue int) PropOption { return html.Rows(parseValue) }

// Cols delegates to [html.Cols].
func Cols(parseValue int) PropOption { return html.Cols(parseValue) }

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

// Multiple delegates to [html.Multiple].
func Multiple(parseValues ...bool) PropOption { return html.Multiple(parseValues...) }

// Open delegates to [html.Open].
func Open(parseValues ...bool) PropOption { return html.Open(parseValues...) }

// Hidden delegates to [html.Hidden].
func Hidden(parseValues ...bool) PropOption { return html.Hidden(parseValues...) }

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
func Attr(parseKey string, parseValue any) PropOption { return html.Attr(parseKey, parseValue) }

// Attrs delegates to [html.Attrs].
func Attrs(parseValues map[string]any) PropOption { return html.Attrs(parseValues) }

// OnClick delegates to [html.OnClick].
func OnClick(parseCallback any) PropOption { return html.OnClick(parseCallback) }

// OnClickParallel delegates to [html.OnClickParallel].
func OnClickParallel(parseSlotID string, parseCallback any) PropOption {
	return html.OnClickParallel(parseSlotID, parseCallback)
}

// OnInput delegates to [html.OnInput].
func OnInput(parseCallback any) PropOption { return html.OnInput(parseCallback) }

// OnChange delegates to [html.OnChange].
func OnChange(parseCallback any) PropOption { return html.OnChange(parseCallback) }

// OnSubmit delegates to [html.OnSubmit].
func OnSubmit(parseCallback any) PropOption { return html.OnSubmit(parseCallback) }

// OnKeyDown delegates to [html.OnKeyDown].
func OnKeyDown(parseCallback any) PropOption { return html.OnKeyDown(parseCallback) }

// OnKeyUp delegates to [html.OnKeyUp].
func OnKeyUp(parseCallback any) PropOption { return html.OnKeyUp(parseCallback) }

// OnMouseUp delegates to [html.OnMouseUp].
func OnMouseUp(parseCallback any) PropOption   { return html.OnMouseUp(parseCallback) }
func OnMouseDown(parseCallback any) PropOption { return html.OnMouseDown(parseCallback) }

// OnFocus delegates to [html.OnFocus].
func OnFocus(parseCallback any) PropOption { return html.OnFocus(parseCallback) }

// OnBlur delegates to [html.OnBlur].
func OnBlur(parseCallback any) PropOption { return html.OnBlur(parseCallback) }

// OnScroll delegates to [html.OnScroll].
func OnScroll(parseCallback any) PropOption { return html.OnScroll(parseCallback) }

// OnPointerDown delegates to [html.OnPointerDown].
func OnPointerDown(parseCallback any) PropOption { return html.OnPointerDown(parseCallback) }

// OnPointerMove delegates to [html.OnPointerMove].
func OnPointerMove(parseCallback any) PropOption { return html.OnPointerMove(parseCallback) }

// OnPointerUp delegates to [html.OnPointerUp].
func OnPointerUp(parseCallback any) PropOption { return html.OnPointerUp(parseCallback) }

// OnTouchStart delegates to [html.OnTouchStart].
func OnTouchStart(parseCallback any) PropOption { return html.OnTouchStart(parseCallback) }

// OnTouchMove delegates to [html.OnTouchMove].
func OnTouchMove(parseCallback any) PropOption { return html.OnTouchMove(parseCallback) }

// OnTouchEnd delegates to [html.OnTouchEnd].
func OnTouchEnd(parseCallback any) PropOption { return html.OnTouchEnd(parseCallback) }

// OnDragStart delegates to [html.OnDragStart].
func OnDragStart(parseCallback any) PropOption { return html.OnDragStart(parseCallback) }

// OnDragOver delegates to [html.OnDragOver].
func OnDragOver(parseCallback any) PropOption { return html.OnDragOver(parseCallback) }

// OnDrop delegates to [html.OnDrop].
func OnDrop(parseCallback any) PropOption { return html.OnDrop(parseCallback) }

// OnDragEnd delegates to [html.OnDragEnd].
func OnDragEnd(parseCallback any) PropOption { return html.OnDragEnd(parseCallback) }

// OnMouseEnter delegates to [html.OnMouseEnter].
func OnMouseEnter(parseCallback any) PropOption { return html.OnMouseEnter(parseCallback) }

// OnMouseLeave delegates to [html.OnMouseLeave].
func OnMouseLeave(parseCallback any) PropOption { return html.OnMouseLeave(parseCallback) }

// OnDoubleClick delegates to [html.OnDoubleClick].
func OnDoubleClick(parseCallback any) PropOption { return html.OnDoubleClick(parseCallback) }

// OnContextMenu delegates to [html.OnContextMenu].
func OnContextMenu(parseCallback any) PropOption { return html.OnContextMenu(parseCallback) }

// OnWheel delegates to [html.OnWheel].
func OnWheel(parseCallback any) PropOption { return html.OnWheel(parseCallback) }

// OnTransitionEnd delegates to [html.OnTransitionEnd].
func OnTransitionEnd(parseCallback any) PropOption { return html.OnTransitionEnd(parseCallback) }

// OnAnimationEnd delegates to [html.OnAnimationEnd].
func OnAnimationEnd(parseCallback any) PropOption { return html.OnAnimationEnd(parseCallback) }

// OnLoad delegates to [html.OnLoad].
func OnLoad(parseCallback any) PropOption { return html.OnLoad(parseCallback) }

// OnError delegates to [html.OnError].
func OnError(parseCallback any) PropOption { return html.OnError(parseCallback) }

// Passive delegates to [html.Passive].
func Passive(parseCallback any) any { return html.Passive(parseCallback) }

// Prevent delegates to [html.Prevent].
func Prevent(parseCallback any) any { return html.Prevent(parseCallback) }

// Stop delegates to [html.Stop].
func Stop(parseCallback any) any { return html.Stop(parseCallback) }

// Debounce delegates to [html.Debounce].
func Debounce(parseDelay time.Duration, parseCallback any) any {
	return html.Debounce(parseDelay, parseCallback)
}

// Throttle delegates to [html.Throttle].
func Throttle(parseInterval time.Duration, parseCallback any) any {
	return html.Throttle(parseInterval, parseCallback)
}

// splitArgs is a core package helper.
//
// A bare Props value is merged onto the accumulated state rather than replacing
// it, so that earlier PropOption values (e.g. OnClick handlers) are not dropped
// when a later Props{} argument sets only a subset of fields.  Later values win
// on conflict, matching last-write-wins semantics across the whole argument list.
func splitArgs(parseArgs ...any) (Props, []ui.Node) {
	var parseProps Props
	parseChildInputs := make([]any, 0, len(parseArgs))
	for _, parseArg := range parseArgs {
		switch parseTyped := parseArg.(type) {
		case nil:
			continue
		case Props:
			parseProps = mergeProps(parseProps, parseTyped)
		case *Props:
			if parseTyped != nil {
				parseProps = mergeProps(parseProps, *parseTyped)
			}
		case propsInput:
			parseProps = mergeProps(parseProps, parseTyped.value)
		case PropOption:
			// parseProps is a fresh local (and mergeProps copies any adopted
			// maps), so options apply in place — WithProps would pay one full
			// Props clone per option argument.
			html.ApplyPropOptions(&parseProps, parseTyped)
		case []PropOption:
			html.ApplyPropOptions(&parseProps, parseTyped...)
		default:
			parseChildInputs = append(parseChildInputs, parseArg)
		}
	}
	return parseProps, html.Children(parseChildInputs...)
}

// mergeProps copies the non-zero fields of parseIncoming over parseBase and
// returns the result.  It is a pure value merge with no side effects: event
// handlers are copied as-is rather than re-registered through PropOption
// constructors (which on the WASM build register hooks and therefore must only
// run inside a component render).  Later values win on conflict; map fields
// are merged key-wise with incoming entries overriding base entries.
func mergeProps(parseBase Props, parseIncoming Props) Props {
	return html.MergeProps(parseBase, parseIncoming)
}

// mergeStringMap merges parseIncoming over parseBase without mutating either.
func mergeStringMap(parseBase map[string]string, parseIncoming map[string]string) map[string]string {
	if len(parseIncoming) == 0 {
		return parseBase
	}
	parseMerged := make(map[string]string, len(parseBase)+len(parseIncoming))
	maps.Copy(parseMerged, parseBase)
	maps.Copy(parseMerged, parseIncoming)
	return parseMerged
}
