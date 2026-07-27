package css

import "strings"

// Typography properties beyond size/weight/tracking: alignment, wrapping,
// decoration, and font-family (which css/global.go's own doc comment already
// advertised as css.Font(css.SansStack) before this file existed).

// --- font-family ----------------------------------------------------------------

// FontStack is a typed font-family value: an ordered fallback list, a var()
// reference to a token, or one of the built-in system stacks.
type FontStack string

func (s FontStack) String() string { return string(s) }

// The built-in stacks. font-family IS a comma-separated fallback list — that is its
// entire grammar, and unlike the transition shorthand nothing is appended after the
// list, so a plain comma join is correct. `ui-*` leads each stack so the browser can
// pick the platform UI face before falling back to named families.
const (
	SansStack  FontStack = "ui-sans-serif,system-ui,-apple-system,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif"
	SerifStack FontStack = "ui-serif,Georgia,Cambria,'Times New Roman',Times,serif"
	MonoStack  FontStack = "ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,'Liberation Mono','Courier New',monospace"
)

// FontStackOf builds a fallback list from family names, quoting any name that needs
// it and dropping characters that could close the quote or end the declaration —
// so a family name from a user profile cannot inject CSS:
//
//	FontStackOf("Atlas Grotesk", "system-ui", "sans-serif")
//	  -> "'Atlas Grotesk',system-ui,sans-serif"
func FontStackOf(parseFamilies ...string) FontStack {
	parts := make([]string, 0, len(parseFamilies))
	for _, family := range parseFamilies {
		if name := sanitizeFontFamily(family); name != "" {
			parts = append(parts, name)
		}
	}
	return FontStack(strings.Join(parts, ","))
}

// sanitizeFontFamily keeps letters, digits, spaces, '-' and '_' (everything a real
// family name needs), then quotes the result when it contains a space or leads with a
// digit — the two cases where an unquoted family name is invalid CSS.
func sanitizeFontFamily(parseFamily string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(parseFamily) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == ' ':
			b.WriteRune(r)
		}
	}
	name := strings.TrimSpace(b.String())
	if name == "" {
		return ""
	}
	if strings.ContainsRune(name, ' ') || (name[0] >= '0' && name[0] <= '9') {
		return "'" + name + "'"
	}
	return name
}

// VarFontStack references a custom property holding a font stack — the typed sibling
// of Var/VarLength for the font-family position, so a design token flows into Font
// without dropping to Raw. Same name sanitizing as Var.
func VarFontStack(parseName string) FontStack { return FontStack(varExpr(parseName)) }

// RawFontStack is the escape hatch for a family list the constructors do not cover.
func RawFontStack(parseValue string) FontStack { return FontStack(parseValue) }

// Font sets font-family from a typed stack: css.Font(css.SansStack).
//
// There is deliberately NO `font` shorthand constructor. The `font` shorthand RESETS
// every font longhand it omits (line-height included) and ends in a comma-separated
// family list, which makes it both destructive and the same comma hazard the
// Transition grammar note describes. Use Font + FontSize + FontWeight + LineHeight.
func Font(stack FontStack) Rule { return decl("font-family", string(stack)) }

type fontStyleProp struct {
	Normal  Rule
	Italic  Rule
	Oblique Rule
}

// FontStyle is the typed namespace for font-style.
var FontStyle = fontStyleProp{
	Normal:  decl("font-style", "normal"),
	Italic:  decl("font-style", "italic"),
	Oblique: decl("font-style", "oblique"),
}

type fontVariantLigaturesProp struct {
	Normal Rule
	// None disables ligatures. Required for tabular/monospaced data columns, where
	// an "fi" or "ffi" ligature changes the advance width and breaks the alignment
	// that tabular-nums is there to guarantee.
	None          Rule
	NoCommon      Rule
	Contextual    Rule
	NoContextual  Rule
	Discretionary Rule
}

// FontVariantLigatures is the typed namespace for font-variant-ligatures.
var FontVariantLigatures = fontVariantLigaturesProp{
	Normal:        decl("font-variant-ligatures", "normal"),
	None:          decl("font-variant-ligatures", "none"),
	NoCommon:      decl("font-variant-ligatures", "no-common-ligatures"),
	Contextual:    decl("font-variant-ligatures", "contextual"),
	NoContextual:  decl("font-variant-ligatures", "no-contextual"),
	Discretionary: decl("font-variant-ligatures", "discretionary-ligatures"),
}

// --- alignment ------------------------------------------------------------------

type textAlignProp struct {
	Left    Rule
	Center  Rule
	Right   Rule
	Justify Rule
	// Start / End follow the writing direction, so they stay correct in a
	// right-to-left locale where Left/Right do not.
	Start Rule
	End   Rule
}

// TextAlign is the typed namespace for text-align: css.TextAlign.Right.
var TextAlign = textAlignProp{
	Left:    decl("text-align", "left"),
	Center:  decl("text-align", "center"),
	Right:   decl("text-align", "right"),
	Justify: decl("text-align", "justify"),
	Start:   decl("text-align", "start"),
	End:     decl("text-align", "end"),
}

type verticalAlignProp struct {
	Baseline   Rule
	Top        Rule
	Middle     Rule
	Bottom     Rule
	TextTop    Rule
	TextBottom Rule
	Sub        Rule
	Super      Rule
}

// VerticalAlign is the typed namespace for vertical-align — the property that
// controls where a table cell's content sits when rows have unequal heights.
var VerticalAlign = verticalAlignProp{
	Baseline:   decl("vertical-align", "baseline"),
	Top:        decl("vertical-align", "top"),
	Middle:     decl("vertical-align", "middle"),
	Bottom:     decl("vertical-align", "bottom"),
	TextTop:    decl("vertical-align", "text-top"),
	TextBottom: decl("vertical-align", "text-bottom"),
	Sub:        decl("vertical-align", "sub"),
	Super:      decl("vertical-align", "super"),
}

// --- wrapping -------------------------------------------------------------------

type whiteSpaceProp struct {
	Normal Rule
	// NoWrap keeps a run of text on one line. On a table cell it is what stops a
	// numeric column from wrapping mid-figure; on a flex/grid child it is also the
	// reason the child can overflow, so pair it with MinWidth(Zero) or Overflow.
	NoWrap      Rule
	Pre         Rule
	PreWrap     Rule
	PreLine     Rule
	BreakSpaces Rule
}

// WhiteSpace is the typed namespace for white-space.
var WhiteSpace = whiteSpaceProp{
	Normal:      decl("white-space", "normal"),
	NoWrap:      decl("white-space", "nowrap"),
	Pre:         decl("white-space", "pre"),
	PreWrap:     decl("white-space", "pre-wrap"),
	PreLine:     decl("white-space", "pre-line"),
	BreakSpaces: decl("white-space", "break-spaces"),
}

type textWrapProp struct {
	Wrap   Rule
	NoWrap Rule
	// Balance evens out the line lengths of a short block (headings, pull quotes);
	// Pretty only fixes the last line (orphans) and is cheap enough for body copy.
	Balance Rule
	Pretty  Rule
	Stable  Rule
}

// TextWrap is the typed namespace for text-wrap. Unsupported values are ignored by
// older browsers, so these are safe progressive enhancement.
var TextWrap = textWrapProp{
	Wrap:    decl("text-wrap", "wrap"),
	NoWrap:  decl("text-wrap", "nowrap"),
	Balance: decl("text-wrap", "balance"),
	Pretty:  decl("text-wrap", "pretty"),
	Stable:  decl("text-wrap", "stable"),
}

type overflowWrapProp struct {
	Normal    Rule
	Anywhere  Rule
	BreakWord Rule
}

// OverflowWrap is the typed namespace for overflow-wrap — the escape valve for an
// unbreakable token (a URL, a hash) that would otherwise overflow its container.
var OverflowWrap = overflowWrapProp{
	Normal:    decl("overflow-wrap", "normal"),
	Anywhere:  decl("overflow-wrap", "anywhere"),
	BreakWord: decl("overflow-wrap", "break-word"),
}

// TextOverflowEllipsis truncates overflowing text with an ellipsis. It only takes
// effect on a single-line, non-wrapping, overflow-hidden box, so all three
// declarations are emitted together — the property on its own is a silent no-op, and
// that is the most common reason "text-overflow: ellipsis doesn't work".
func TextOverflowEllipsis() Rule {
	return Rule{decls: []declaration{
		{"overflow", "hidden"},
		{"text-overflow", "ellipsis"},
		{"white-space", "nowrap"},
	}}
}

// --- decoration -----------------------------------------------------------------

type textDecorationProp struct {
	None        Rule
	Underline   Rule
	Overline    Rule
	LineThrough Rule
}

// TextDecoration is the typed namespace for text-decoration-line. It sets the
// LONGHAND, not the shorthand: the shorthand resets thickness and offset, so
// TextDecoration.Underline composed with TextDecorationThickness would have raced
// against declaration ordering. As longhands they compose unconditionally.
var TextDecoration = textDecorationProp{
	None:        decl("text-decoration-line", "none"),
	Underline:   decl("text-decoration-line", "underline"),
	Overline:    decl("text-decoration-line", "overline"),
	LineThrough: decl("text-decoration-line", "line-through"),
}

// TextDecorationThickness / TextUnderlineOffset tune an underline's weight and
// distance from the baseline — the difference between a link underline that reads as
// typography and one that reads as a browser default.
func TextDecorationThickness(v Length) Rule {
	return decl("text-decoration-thickness", string(v))
}
func TextUnderlineOffset(v Length) Rule { return decl("text-underline-offset", string(v)) }

// TextDecorationColor sets the line color independently of the text color.
func TextDecorationColor(c Color) Rule { return decl("text-decoration-color", string(c)) }

type textDecorationStyleProp struct {
	Solid  Rule
	Double Rule
	Dotted Rule
	Dashed Rule
	Wavy   Rule
}

// TextDecorationStyle is the typed namespace for text-decoration-style.
var TextDecorationStyle = textDecorationStyleProp{
	Solid:  decl("text-decoration-style", "solid"),
	Double: decl("text-decoration-style", "double"),
	Dotted: decl("text-decoration-style", "dotted"),
	Dashed: decl("text-decoration-style", "dashed"),
	Wavy:   decl("text-decoration-style", "wavy"),
}

// --- misc typography ------------------------------------------------------------

// TextIndent sets text-indent.
func TextIndent(v Length) Rule { return decl("text-indent", string(v)) }

// WordSpacing sets word-spacing (letter-spacing is Tracking).
func WordSpacing(v Length) Rule { return decl("word-spacing", string(v)) }
