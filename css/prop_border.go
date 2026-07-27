package css

// Side-specific borders and the table box properties.
//
// Border(width, color) only ever emitted the four-sided `border` shorthand, so every
// one-sided hairline in a document-style design — a rule under a table row, a rule
// beside a nav rail, a divider above a footer — had to go through css.Raw. These are
// the typed forms.

// --- border sides -------------------------------------------------------------

// BorderTop / BorderRight / BorderBottom / BorderLeft set one side's border to a
// solid line of the given width and color, matching Border's solid default:
//
//	BorderBottom(Px(1), Slate200) -> border-bottom: 1px solid #e2e8f0
//
// For a non-solid line, pair a side constructor with BorderStyle (or the
// side-specific style namespaces): canonicalize sorts declarations by property name
// within a block, and "border-bottom" sorts before "border-bottom-style", so the
// longhand reliably overrides the shorthand it follows.
func BorderTop(width Length, c Color) Rule    { return decl("border-top", solidLine(width, c)) }
func BorderRight(width Length, c Color) Rule  { return decl("border-right", solidLine(width, c)) }
func BorderBottom(width Length, c Color) Rule { return decl("border-bottom", solidLine(width, c)) }
func BorderLeft(width Length, c Color) Rule   { return decl("border-left", solidLine(width, c)) }

// BorderX / BorderY set the horizontal / vertical border pairs as longhands, the
// same shape as PaddingX / MarginY.
func BorderX(width Length, c Color) Rule {
	return Rule{decls: []declaration{
		{"border-left", solidLine(width, c)},
		{"border-right", solidLine(width, c)},
	}}
}
func BorderY(width Length, c Color) Rule {
	return Rule{decls: []declaration{
		{"border-top", solidLine(width, c)},
		{"border-bottom", solidLine(width, c)},
	}}
}

// solidLine builds the `<width> solid <color>` border/outline value shared by
// Border, the side constructors, and Outline. A Color may contain commas
// (rgba(0,0,0,.5)), but they are nested inside the function's parentheses and the
// border shorthand is NOT a comma-separated list, so plain concatenation is correct
// here — unlike the transition shorthand. See Transition for that contrast.
func solidLine(width Length, c Color) string {
	return string(width) + " solid " + string(c)
}

// --- border-style -------------------------------------------------------------

// LineStyle is a typed border/outline line style.
type LineStyle string

func (s LineStyle) String() string { return string(s) }

const (
	LineSolid  LineStyle = "solid"
	LineDashed LineStyle = "dashed"
	LineDotted LineStyle = "dotted"
	LineDouble LineStyle = "double"
	LineGroove LineStyle = "groove"
	LineRidge  LineStyle = "ridge"
	// LineNone removes the border and collapses its width to 0; LineHidden does the
	// same except in a collapsed table border conflict, where it wins outright.
	LineNone   LineStyle = "none"
	LineHidden LineStyle = "hidden"
)

// BorderStyle sets border-style for all four sides from a typed LineStyle.
func BorderStyle(s LineStyle) Rule { return decl("border-style", string(s)) }

// BorderTopStyle / BorderRightStyle / BorderBottomStyle / BorderLeftStyle set one
// side's line style.
func BorderTopStyle(s LineStyle) Rule    { return decl("border-top-style", string(s)) }
func BorderRightStyle(s LineStyle) Rule  { return decl("border-right-style", string(s)) }
func BorderBottomStyle(s LineStyle) Rule { return decl("border-bottom-style", string(s)) }
func BorderLeftStyle(s LineStyle) Rule   { return decl("border-left-style", string(s)) }

// BorderColor / BorderWidth set the color / width of all four sides without
// restating the whole shorthand.
func BorderColor(c Color) Rule      { return decl("border-color", string(c)) }
func BorderWidth(v Length) Rule     { return decl("border-width", string(v)) }
func OutlineStyle(s LineStyle) Rule { return decl("outline-style", string(s)) }

// --- corner radii --------------------------------------------------------------

// RoundedTop / RoundedBottom / RoundedLeft / RoundedRight round one edge's pair of
// corners — the "tab" and "card stub" shapes, which Rounded (all four) cannot express.
func RoundedTop(v Length) Rule {
	return Rule{decls: []declaration{
		{"border-top-left-radius", string(v)},
		{"border-top-right-radius", string(v)},
	}}
}
func RoundedBottom(v Length) Rule {
	return Rule{decls: []declaration{
		{"border-bottom-left-radius", string(v)},
		{"border-bottom-right-radius", string(v)},
	}}
}
func RoundedLeft(v Length) Rule {
	return Rule{decls: []declaration{
		{"border-top-left-radius", string(v)},
		{"border-bottom-left-radius", string(v)},
	}}
}
func RoundedRight(v Length) Rule {
	return Rule{decls: []declaration{
		{"border-top-right-radius", string(v)},
		{"border-bottom-right-radius", string(v)},
	}}
}

// --- tables --------------------------------------------------------------------

type borderCollapseProp struct {
	Collapse Rule
	Separate Rule
}

// BorderCollapse is the typed namespace for border-collapse. `collapse` is what
// makes a table's hairlines share a single line rather than doubling at every cell
// boundary — the difference between a printed manifest and a spreadsheet.
var BorderCollapse = borderCollapseProp{
	Collapse: decl("border-collapse", "collapse"),
	Separate: decl("border-collapse", "separate"),
}

// BorderSpacing sets border-spacing (only meaningful under BorderCollapse.Separate).
func BorderSpacing(v Length) Rule { return decl("border-spacing", string(v)) }

type tableLayoutProp struct {
	Auto  Rule
	Fixed Rule
}

// TableLayout is the typed namespace for table-layout. `fixed` sizes columns from
// the first row only (fast, predictable); `auto` measures all content.
var TableLayout = tableLayoutProp{
	Auto:  decl("table-layout", "auto"),
	Fixed: decl("table-layout", "fixed"),
}
