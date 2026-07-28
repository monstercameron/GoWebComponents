package design

import "github.com/monstercameron/GoWebComponents/v5/css"

// Surfaces, separation and spacing — and the one deliberate absence that shapes the
// rest of this design system.
//
// # There is no Card, Panel, or nested-surface primitive, and there never will be
//
// The failure this replaces was rounded dark boxes nested four deep, every one with
// the same border, the same radius and the same shadow. That is not a styling
// accident, it is what happens when a design system ships a Card: a Card is a
// container, containers accept containers, and every nesting level costs 16px of
// padding and a border while adding zero information. Four levels in, the reader has
// four identical frames and no idea which one matters.
//
// So the entire surface vocabulary here is:
//
//   - Surface — ONE flat paper plane with a hairline. Use it once per region.
//   - SurfaceFlush — the same plane with no padding, for a table that should bleed
//     to its own hairline instead of floating inside a gutter.
//   - Recess — a texture change, not a box: no border, no radius, just pressed paper.
//   - Divider — a hairline rule. This is the answer to "these two things need
//     separating".
//   - Stack / Cluster — space. This is the other answer.
//
// If something inside a Surface needs to be visually separate, it gets a Divider or
// a Stack gap. If it feels like it needs its own border, the real problem is almost
// always that the parent Surface is doing two jobs and should be two Surfaces, side
// by side, at the same level.
//
// # If you came here looking for a Card to build a product grid
//
// You want [Catalog], in catalog.go. That is the one case where "no Card" needed a
// positive answer rather than an argument: a catalog is neither a queue (which is a
// Table) nor a document (which is a Surface), and built from Surface + Cluster it comes
// out as two dozen boxes at identical weight — exactly the failure this file exists to
// prevent, reached by following this file's own rules. The answer is a manifest of
// full-width rows, which also makes the four-items-in-a-three-column-grid orphan
// structurally impossible. Read catalog.go's header before reaching for a box.

var surfaceBundle = clip(css.Rules(
	css.Bg(Paper()),
	css.Border(HairlineWidth, Hairline()),
	css.Rounded(RadiusTag),
	css.Padding(Space4),
	// NO box-shadow, at any elevation, anywhere in this design system.
	//
	// A shadow claims the element floats above the page. Four nested shadowed boxes
	// claim four elevations that do not exist, and on a dark background a shadow is
	// invisible anyway — which is precisely how the old design ended up with heavy
	// shadow declarations that cost paint time and communicated nothing. Paper does
	// not float. It is on the desk, and a hairline is where one sheet ends.
	css.Media(css.MaxW(breakNarrow), css.Padding(Space3)),
))

// Surface is the one flat paper plane: hairline border, 2px radius, no shadow, no
// gradient. Use it once per page region.
//
// Do not nest it. See the file comment for why the alternatives (Divider, Stack) are
// the right instinct instead.
func Surface() []css.Rule { return surfaceBundle }

var surfaceFlushBundle = clip(css.Rules(
	css.Bg(Paper()),
	css.Border(HairlineWidth, Hairline()),
	css.Rounded(RadiusTag),
	// overflow:hidden so the child table's own zebra and header rules clip to the
	// 2px radius instead of squaring off the corners.
	css.Raw("overflow", "hidden"),
))

// SurfaceFlush is Surface without padding, for content that should reach the
// hairline: a data table, a full-width list, a lane placard row.
//
// This is not a second card style. It is the same plane with the gutter removed,
// because a dense table inside 16px of padding wastes the density it was chosen for.
func SurfaceFlush() []css.Rule { return surfaceFlushBundle }

var recessBundle = clip(css.Rules(
	css.Bg(PaperSunk()),
	css.Padding(Space3),
	// Deliberately NO border and NO radius. Those two properties are what make a
	// rectangle read as a card, and this is not a card — it is the same sheet of
	// paper, pressed. Take them away and the eye reads a region rather than an
	// object, which is exactly the difference between "here is the filter area" and
	// "here is a fourth nested box".
	css.Rounded(RadiusNone),
	css.Media(css.MaxW(breakNarrow), css.Padding(Space2)),
))

// Recess is pressed paper: a filter bar, a raw payload, a diff, a form region.
//
// Use it for content that is INPUT to the page rather than output of it. Do not use
// it to group things a second time inside a Surface — that is the nested-card move
// wearing a different background, and the reason Recess has no border or radius is
// to make that misuse look wrong immediately.
func Recess() []css.Rule { return recessBundle }

// A note on the two border declarations below, because it looks like a conflict and
// is not: the reset (`border`) and the rule (`border-top`) land in the same block,
// and css/rule.go's canonicalize sorts declarations by PROPERTY NAME, so "border"
// always precedes "border-top" and the longhand always wins. That is a guarantee, not
// luck — but it is a guarantee about alphabetical order, so if you ever need
// `border-bottom` to beat `border-top` in one block, it will not work. Emit them in
// separate scopes or use one declaration.
//
// The reset is here because Divider is meant to work on a bare <hr>, which carries a
// UA border that Preflight does not remove.
var dividerBundle = clip(css.Rules(
	css.Display.Block,
	css.W(css.Full),
	css.H(css.Zero),
	css.Raw("border", "none"),
	css.Raw("border-top", hairlineRule()),
	css.MarginY(Space4),
))

// Divider is a hairline rule, for an <hr> or a plain div.
//
// This is the primary separation tool in Atlas, and it is worth being explicit about
// why it beats a second box: a rule costs one pixel and says "these are different
// sections of the same thing", while a box costs a border, a radius, two gutters and
// says "these are different things". Inside one Surface, the first statement is
// almost always the true one.
func Divider() []css.Rule { return dividerBundle }

var manifestRuleBundle = clip(css.Rules(
	css.Display.Block,
	css.W(css.Full),
	css.H(css.Zero),
	css.Raw("border", "none"),
	css.Raw("border-top", manifestRule()),
	css.MarginY(Space3),
))

// ManifestRule is the heavy 2px ink rule: under a page title, above a total, closing
// a section. It is the design system's only "loud" structural mark besides the lane
// placard, and it is what makes a screen read as a printed manifest.
//
// One per region at most. Two heavy rules in the same view cancel each other out and
// you are back to uniform weight.
func ManifestRule() []css.Rule { return manifestRuleBundle }

// --- spacing primitives -------------------------------------------------------

// Stack is a vertical flow with a uniform gap: the other answer to "these need
// separating".
//
// Gap, not margin, deliberately. Margins collapse, they double up between siblings,
// they need :last-child resets, and they leak out of their container. A flex gap does
// none of that, so vertical rhythm becomes a property of the container — one place to
// change it — instead of a property each child has to remember.
//
// Pass a Space constant. Any css.Length compiles, but an off-scale value mints
// another hashed class and puts the page slightly out of rhythm.
func Stack(parseGap css.Length) []css.Rule {
	return css.Rules(
		css.Display.Flex,
		css.FlexDir.Col,
		css.Gap(parseGap),
		// min-width:0 so a Stack inside a grid column can actually shrink. Without
		// it, a wide mono string (a long SKU, a URL) sets the column's min-content
		// width and blows out the whole grid — the single most common cause of an
		// unexplained horizontal scrollbar in a CSS grid layout.
		css.MinWidth(css.Zero),
	)
}

// Cluster is a horizontal group that wraps: a button row, a chip row, a set of meta
// fields.
//
// It wraps by default rather than scrolling or overflowing, which is most of what
// makes this design system survive 380px without per-component media queries.
func Cluster(parseGap css.Length) []css.Rule {
	return css.Rules(
		css.Display.Flex,
		css.FlexDir.Row,
		css.Raw("flex-wrap", "wrap"),
		css.Items.Center,
		css.Gap(parseGap),
		css.MinWidth(css.Zero),
	)
}

// SplitRow is a Cluster whose children push apart: a title on the left, actions on
// the right, collapsing to a wrapped stack when there is no room.
//
// It exists because "header row with actions" appears on nearly every Atlas page and
// hand-rolling it invites a justify-between that does not wrap, which is a horizontal
// scrollbar at 380px.
func SplitRow(parseGap css.Length) []css.Rule {
	return css.Rules(
		Cluster(parseGap),
		css.Justify.Between,
		css.Items.End,
	)
}
