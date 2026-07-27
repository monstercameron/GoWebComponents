package css

import (
	"strconv"
	"strings"
)

// Positioning offsets, scrolling, stacking, the flex item properties, and the whole
// grid vocabulary. Every Position.Absolute / Position.Sticky rule needs at least one
// inset offset, and grid had no typed surface at all, so these were the two largest
// css.Raw sinks in real layout code.

// --- inset offsets --------------------------------------------------------------

// Top / Right / Bottom / Left set a single positioning offset. Negative values come
// from the Length constructors (Px(-1)), so nothing here has to parse a sign.
func Top(v Length) Rule    { return decl("top", string(v)) }
func Right(v Length) Rule  { return decl("right", string(v)) }
func Bottom(v Length) Rule { return decl("bottom", string(v)) }
func Left(v Length) Rule   { return decl("left", string(v)) }

// Inset sets all four offsets at once (`inset: 0` is the "fill the positioned
// ancestor" idiom).
func Inset(v Length) Rule { return decl("inset", string(v)) }

// InsetX / InsetY set the horizontal / vertical offset pairs as longhands, the same
// shape as PaddingX / MarginY. They are emitted as physical longhands rather than
// `inset-inline`/`inset-block` so they do not flip under a right-to-left writing mode
// — matching PaddingX/MarginX, which are physical too.
func InsetX(v Length) Rule {
	return Rule{decls: []declaration{{"left", string(v)}, {"right", string(v)}}}
}
func InsetY(v Length) Rule {
	return Rule{decls: []declaration{{"top", string(v)}, {"bottom", string(v)}}}
}

// ZIndex sets the stacking order. It takes an int, not a Length, because z-index is
// an integer property: a "1px" z-index is simply dropped by the browser.
func ZIndex(n int) Rule { return decl("z-index", strconv.Itoa(n)) }

// --- overflow & scrolling -------------------------------------------------------

type overflowProp struct {
	Visible Rule
	Hidden  Rule
	Scroll  Rule
	Auto    Rule
	// Clip clips like Hidden but creates no scroll container, so a clipped box
	// cannot be scrolled programmatically or by a focus jump.
	Clip Rule
}

func overflowNamespace(property string) overflowProp {
	return overflowProp{
		Visible: decl(property, "visible"),
		Hidden:  decl(property, "hidden"),
		Scroll:  decl(property, "scroll"),
		Auto:    decl(property, "auto"),
		Clip:    decl(property, "clip"),
	}
}

// Overflow / OverflowX / OverflowY are the typed namespaces for the overflow
// properties: css.OverflowY.Auto.
//
// Accessibility note: a box with Overflow.Auto or .Scroll is a scroll container, and
// a scroll container that is not focusable cannot be scrolled by keyboard. Pair it
// with html.Props{TabIndex: html.TabIndexZero} (see that field's comment).
var (
	Overflow  = overflowNamespace("overflow")
	OverflowX = overflowNamespace("overflow-x")
	OverflowY = overflowNamespace("overflow-y")
)

type overscrollProp struct {
	Auto    Rule
	Contain Rule
	None    Rule
}

func overscrollNamespace(property string) overscrollProp {
	return overscrollProp{
		Auto:    decl(property, "auto"),
		Contain: decl(property, "contain"),
		None:    decl(property, "none"),
	}
}

// OverscrollBehavior / -X / -Y are the typed namespaces for overscroll-behavior.
// Contain stops a scroll that reaches the end of this box from chaining to the page
// behind it (and suppresses pull-to-refresh) — the fix for a horizontally scrolling
// table that drags the whole document sideways.
var (
	OverscrollBehavior  = overscrollNamespace("overscroll-behavior")
	OverscrollBehaviorX = overscrollNamespace("overscroll-behavior-x")
	OverscrollBehaviorY = overscrollNamespace("overscroll-behavior-y")
)

// ScrollPadding / ScrollPaddingTop inset the scrollport for scroll-snapping and
// fragment/focus jumps — how a sticky header stops covering the element that a
// same-page anchor just scrolled to.
func ScrollPadding(v Length) Rule    { return decl("scroll-padding", string(v)) }
func ScrollPaddingTop(v Length) Rule { return decl("scroll-padding-top", string(v)) }

// --- interaction ----------------------------------------------------------------

type pointerEventsProp struct {
	Auto Rule
	None Rule
}

// PointerEvents is the typed namespace for pointer-events. None makes a box
// invisible to the mouse so clicks fall through to what is underneath — required for
// decorative overlays (a gradient scrim, a perforation strip) that must not eat the
// clicks of the content they sit over.
var PointerEvents = pointerEventsProp{
	Auto: decl("pointer-events", "auto"),
	None: decl("pointer-events", "none"),
}

type appearanceProp struct {
	Auto Rule
	None Rule
}

// Appearance is the typed namespace for appearance. None strips the platform widget
// look from a control (the native <select> arrow, the iOS <input> inner shadow) so a
// design system can style one box primitive and share it across input and select.
// It is emitted unprefixed; every current browser supports the unprefixed property.
var Appearance = appearanceProp{
	Auto: decl("appearance", "auto"),
	None: decl("appearance", "none"),
}

type colorSchemeProp struct {
	Normal Rule
	Light  Rule
	Dark   Rule
	// LightDark declares that BOTH schemes are supported, so UA-rendered chrome
	// (scrollbars, form controls, the canvas behind the page) follows the user's
	// preference. Hard-setting Light or Dark locks that chrome regardless of the
	// theme the app actually paints, which is the classic "dark scrollbars on a
	// light page" bug.
	LightDark Rule
}

// ColorScheme is the typed namespace for the color-scheme property.
var ColorScheme = colorSchemeProp{
	Normal:    decl("color-scheme", "normal"),
	Light:     decl("color-scheme", "light"),
	Dark:      decl("color-scheme", "dark"),
	LightDark: decl("color-scheme", "light dark"),
}

// --- flex items -----------------------------------------------------------------

type flexWrapProp struct {
	Wrap        Rule
	NoWrap      Rule
	WrapReverse Rule
}

// FlexWrap is the typed namespace for flex-wrap.
var FlexWrap = flexWrapProp{
	Wrap:        decl("flex-wrap", "wrap"),
	NoWrap:      decl("flex-wrap", "nowrap"),
	WrapReverse: decl("flex-wrap", "wrap-reverse"),
}

// Flex sets the flex shorthand from its three typed parts: grow, shrink, basis.
// The three-value form is spelled out on purpose — the one-value form (`flex: 1`)
// resets basis to 0%, which is a different layout from `flex: 1 1 auto`, and the
// difference is the single most common flexbox surprise.
//
//	Flex(Num(0), Num(0), Auto) -> flex: 0 0 auto   // size to content, never flex
//	Flex(Num(1), Num(1), Zero) -> flex: 1 1 0      // share space equally
func Flex(grow, shrink Number, basis Length) Rule {
	return decl("flex", string(grow)+" "+string(shrink)+" "+string(basis))
}

// FlexGrow / FlexShrink / FlexBasis set the individual longhands.
func FlexGrow(n Number) Rule   { return decl("flex-grow", string(n)) }
func FlexShrink(n Number) Rule { return decl("flex-shrink", string(n)) }
func FlexBasis(v Length) Rule  { return decl("flex-basis", string(v)) }

// Order overrides a flex/grid item's visual position. It is deliberately separate
// from the DOM order it overrides: reordering visually without reordering the DOM
// desynchronizes tab order from reading order, so use it only where both still agree.
func Order(parseN int) Rule { return decl("order", strconv.Itoa(parseN)) }

// RowGap / ColumnGap set the axis gaps individually (Gap sets both).
func RowGap(v Length) Rule    { return decl("row-gap", string(v)) }
func ColumnGap(v Length) Rule { return decl("column-gap", string(v)) }

// --- per-item alignment ---------------------------------------------------------

type selfAlignProp struct {
	Auto     Rule
	Start    Rule
	Center   Rule
	End      Rule
	Stretch  Rule
	Baseline Rule
}

func selfAlignNamespace(property string) selfAlignProp {
	return selfAlignProp{
		Auto:     decl(property, "auto"),
		Start:    decl(property, "start"),
		Center:   decl(property, "center"),
		End:      decl(property, "end"),
		Stretch:  decl(property, "stretch"),
		Baseline: decl(property, "baseline"),
	}
}

// AlignSelf / JustifySelf override the container's alignment for one item — how a
// grid child stops stretching to its row's full height (AlignSelf.Start) without the
// container losing stretch for everything else.
//
// These use the CSS-Align `start`/`end` keywords rather than the flexbox-era
// `flex-start`/`flex-end` used by the older Items/Justify namespaces, because
// `flex-start` is not valid in a grid context while `start` works in both.
var (
	AlignSelf   = selfAlignNamespace("align-self")
	JustifySelf = selfAlignNamespace("justify-self")
)

type contentAlignProp struct {
	Normal   Rule
	Start    Rule
	Center   Rule
	End      Rule
	Between  Rule
	Around   Rule
	Evenly   Rule
	Stretch  Rule
	Baseline Rule
}

// AlignContent distributes a multi-line flex container's lines, or a grid's rows,
// within the container's own height.
//
// It gets its own namespace rather than reusing selfAlignNamespace because the value
// grammars genuinely differ: align-content accepts the space-* distributions but NOT
// `auto`, while align-self/justify-self accept `auto` and no distributions. Sharing
// one struct would have put a silently-invalid `AlignContent.Auto` in autocomplete,
// which is precisely the class of mistake a typed CSS layer exists to prevent.
var AlignContent = contentAlignProp{
	Normal:   decl("align-content", "normal"),
	Start:    decl("align-content", "start"),
	Center:   decl("align-content", "center"),
	End:      decl("align-content", "end"),
	Between:  decl("align-content", "space-between"),
	Around:   decl("align-content", "space-around"),
	Evenly:   decl("align-content", "space-evenly"),
	Stretch:  decl("align-content", "stretch"),
	Baseline: decl("align-content", "baseline"),
}

type itemsAlignProp struct {
	Normal   Rule
	Start    Rule
	Center   Rule
	End      Rule
	Stretch  Rule
	Baseline Rule
}

// JustifyItems is the container-side default for every item's justify-self.
var JustifyItems = itemsAlignProp{
	Normal:   decl("justify-items", "normal"),
	Start:    decl("justify-items", "start"),
	Center:   decl("justify-items", "center"),
	End:      decl("justify-items", "end"),
	Stretch:  decl("justify-items", "stretch"),
	Baseline: decl("justify-items", "baseline"),
}

// --- grid tracks ----------------------------------------------------------------

// Track is one entry in a grid track list (a column width or a row height). Build
// one with Fr, TrackLen, MinMax, Repeat, FitContent, or the keyword constants.
type Track string

func (t Track) String() string { return string(t) }

const (
	// TrackAuto sizes the track to its content, then absorbs leftover space.
	TrackAuto Track = "auto"
	// TrackMinContent / TrackMaxContent size to the smallest / largest the content
	// can be without overflowing.
	TrackMinContent Track = "min-content"
	TrackMaxContent Track = "max-content"
)

// Fr is a fractional track: Fr(1) -> "1fr". The fr unit only exists inside a grid
// track list, which is why it is a Track constructor and not a Length one.
func Fr(n float64) Track { return Track(trimFloat(n) + "fr") }

// TrackLen makes a fixed-size track from a typed Length: TrackLen(Rem(16)).
func TrackLen(v Length) Track { return Track(v) }

// MinMax builds minmax(min, max). The commas here are *function arguments*, not a
// track-list separator, so they are safe inside a joined track list.
//
//	MinMax(TrackLen(Zero), Fr(1)) -> minmax(0,1fr)
//
// minmax(0,1fr) rather than a bare 1fr is the fix for a grid column that refuses to
// shrink below its content: 1fr's implicit minimum is auto (i.e. min-content).
func MinMax(min, max Track) Track {
	return Track("minmax(" + string(min) + "," + string(max) + ")")
}

// FitContent builds fit-content(limit): size to content, but never past limit.
func FitContent(limit Length) Track {
	return Track("fit-content(" + string(limit) + ")")
}

// Repeat repeats a track pattern a fixed number of times: Repeat(3, Fr(1)).
func Repeat(count int, tracks ...Track) Track {
	return Track("repeat(" + strconv.Itoa(count) + "," + joinTracks(tracks) + ")")
}

// RepeatFit / RepeatFill are repeat(auto-fit, …) / repeat(auto-fill, …): as many
// tracks as fit. auto-fit collapses the empty ones (so the filled tracks stretch),
// auto-fill keeps them (so the grid holds its rhythm). This is the whole
// no-media-query responsive-card-grid idiom:
//
//	GridCols(RepeatFit(MinMax(TrackLen(Rem(16)), Fr(1))))
func RepeatFit(tracks ...Track) Track {
	return Track("repeat(auto-fit," + joinTracks(tracks) + ")")
}
func RepeatFill(tracks ...Track) Track {
	return Track("repeat(auto-fill," + joinTracks(tracks) + ")")
}

// joinTracks joins a track list with SPACES. A grid track list is
// whitespace-separated — a comma between tracks is a syntax error that invalidates
// the whole declaration — so this must never become a comma join.
func joinTracks(tracks []Track) string {
	parts := make([]string, 0, len(tracks))
	for _, t := range tracks {
		if t == "" {
			continue
		}
		parts = append(parts, string(t))
	}
	return strings.Join(parts, " ")
}

// GridCols / GridRows set grid-template-columns / grid-template-rows from a typed
// track list:
//
//	GridCols(TrackLen(Rem(14)), MinMax(TrackLen(Zero), Fr(1)))
//	  -> grid-template-columns: 14rem minmax(0,1fr)
func GridCols(tracks ...Track) Rule { return decl("grid-template-columns", joinTracks(tracks)) }
func GridRows(tracks ...Track) Rule { return decl("grid-template-rows", joinTracks(tracks)) }

// GridAutoRows / GridAutoColumns size the implicit tracks the grid creates for items
// beyond the explicit template.
func GridAutoRows(tracks ...Track) Rule { return decl("grid-auto-rows", joinTracks(tracks)) }
func GridAutoColumns(tracks ...Track) Rule {
	return decl("grid-auto-columns", joinTracks(tracks))
}

// GridAreas sets grid-template-areas from one quoted string per row:
//
//	GridAreas("rail head", "rail body") ->
//	  grid-template-areas: "rail head" "rail body"
//
// The rows are whitespace-separated quoted strings, not a comma list. Each row is
// sanitized to area-name characters (letters, digits, '-', '_', '.', and spaces) so a
// quote in a row string cannot terminate the string and inject a declaration; '.'
// survives because it is grid's own "empty cell" token. Rows are NOT padded or
// validated for equal cell counts — a ragged template is invalid CSS and the browser
// drops the whole declaration, which is the loud failure you want.
func GridAreas(parseRows ...string) Rule {
	parts := make([]string, 0, len(parseRows))
	for _, row := range parseRows {
		parts = append(parts, "\""+sanitizeAreaRow(row)+"\"")
	}
	return decl("grid-template-areas", strings.Join(parts, " "))
}

// sanitizeAreaRow keeps only characters legal in a grid-template-areas row.
func sanitizeAreaRow(parseRow string) string {
	var b strings.Builder
	for _, r := range parseRow {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.', r == ' ':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// GridArea places an item into a named area from GridAreas: GridArea("rail").
func GridArea(parseName string) Rule {
	return decl("grid-area", cssIdentSanitize(parseName, ""))
}

// --- grid placement -------------------------------------------------------------

// GridPlacement is a typed grid-column / grid-row value. Build one with GridLineAt,
// GridSpan, GridLineName, GridRange, or GridAuto.
type GridPlacement string

func (p GridPlacement) String() string { return string(p) }

// GridAuto lets the auto-placement algorithm pick the track.
const GridAuto GridPlacement = "auto"

// GridLineAt names a numbered grid line. Negative indices count back from the end,
// so GridLineAt(-1) is the last line — the typed way to say "to the far edge"
// without knowing the column count.
func GridLineAt(parseN int) GridPlacement { return GridPlacement(strconv.Itoa(parseN)) }

// GridSpan spans a number of tracks from wherever the item lands: GridSpan(2).
func GridSpan(parseN int) GridPlacement {
	return GridPlacement("span " + strconv.Itoa(parseN))
}

// GridLineName references a named grid line (from a [name] entry in a track list).
func GridLineName(parseName string) GridPlacement {
	return GridPlacement(cssIdentSanitize(parseName, ""))
}

// GridRange combines a start and an end placement: GridRange(GridLineAt(1),
// GridLineAt(-1)) -> "1 / -1". The separator is a slash, not a comma; grid-column
// takes exactly one start/end pair, so there is no list to get wrong.
func GridRange(start, end GridPlacement) GridPlacement {
	return GridPlacement(string(start) + " / " + string(end))
}

// GridColumn / GridRow place an item on the column / row axis.
//
//	GridColumn(GridRange(GridLineAt(1), GridLineAt(-1)))  // full-bleed row
//	GridRow(GridSpan(2))                                  // two rows tall
func GridColumn(p GridPlacement) Rule { return decl("grid-column", string(p)) }
func GridRow(p GridPlacement) Rule    { return decl("grid-row", string(p)) }
