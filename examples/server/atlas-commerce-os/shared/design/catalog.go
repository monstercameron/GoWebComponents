package design

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/css"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// THE CATALOG MANIFEST — the storefront's answer, and the reason this file exists.
//
// This file is the longest argument in the package because it is the one a future
// contributor is most likely to "fix" back into the thing it replaced. Read the whole
// header before changing the shape.
//
// # The gap
//
// The package provides no Card, for reasons argued at length in surfaces.go, and that
// is correct. But it also provided no replacement for a PRODUCT CATALOG, and a catalog
// is the one Atlas surface that is neither a queue nor a document. Built from what the
// package already had — Surface plus Cluster — /shop comes out as twenty-four bordered
// boxes at identical visual weight: exactly the disease the design system was written
// to cure, repainted in manila. An absence with no alternative is not discipline, it is
// a hole, and a hole in a design system gets filled by whoever is on that ticket.
//
// # Why this is not a card grid
//
// Six reasons, in descending order of how much they cost:
//
//  1. Nothing aligns. Twenty-four prices in twenty-four boxes cannot be compared, and
//     comparing prices is most of what browsing a catalog IS. In a manifest the price
//     column is a column: magnitude is readable as a shape down the page.
//  2. Availability gets buried. Atlas's entire thesis is warehouse-aware availability —
//     not "in stock" but "can this be where I need it, when I need it". In a card that
//     fact is the fourth line of a paragraph; here it is a labelled column with a
//     hairline in front of it, which is the most promoted position on the row.
//  3. The orphan problem. Four products in a three-column grid leaves a row with one
//     item and two holes, and there is no CSS fix — only a fudge (center the last row,
//     stretch it, add a ghost card). Full-width rows make the failure mode structurally
//     impossible: four rows is four rows, and so is five.
//  4. Uniform weight again. Twenty-four bordered radiused boxes have twenty-four
//     identical frames, so nothing can be emphasized. Rows share hairlines, so a row can
//     be emphasized by its CONTENT (an oxide value, a heavier chip) without competing
//     with a frame.
//  5. Density. A card is ~280px tall; a manifest line is ~96px. A twenty-four product
//     catalog is one scroll instead of four, and scanning beats paging.
//  6. It is off-brand. A catalog IS a manifest. The domain already hands us the right
//     metaphor and the package already has the vocabulary for it.
//
// # Why it is not a <table> either
//
// The obvious move, and the one the package author first suggested, was to reuse the
// Table machinery with a thumbnail column. That is nearly right and it is wrong in three
// specific ways, all of which come down to a buyer scanning differently from an operator:
//
//   - An operator reads DOWN a column, comparing forty values to find the outlier. A
//     table exists for that. A buyer reads ACROSS a row, one product at a time, and
//     needs one thing from the row before moving on. The row, not the column, is the
//     unit of attention — so the row is the element that should be styleable, hoverable
//     and clickable, and in a table the row is the one thing you cannot make a link.
//   - Narrow viewports. A table's answer to "too many columns" is TableScroll, i.e.
//     horizontal scroll, and that is the right answer for a 14-column operator manifest
//     on a desktop console. The storefront is the one Atlas surface that is genuinely
//     going to be opened on a phone, and horizontally scrolling a product catalog is
//     hostile. A grid row can restack its own tracks; a table row cannot.
//   - Semantics. A catalog is a list of links, not tabular data. <ul><li><a> announces
//     "list, 24 items" and gives one tab stop per product; a table announces "table, 24
//     rows, 6 columns" and puts a screen-reader user into cell-by-cell navigation for a
//     task that has no cross-referencing in it. Table markup is a claim that the
//     relationship between rows and columns is meaningful. Here it is not: nobody
//     compares thumbnails.
//
// So: a MANIFEST ROW. A list, laid out on a shared CSS grid so the columns still align
// down the page, with a labelled header strip above it so the columns are still named,
// carrying a thumbnail and one action. It reads as a table, behaves as a list, and
// reflows as neither.
//
// # What keeps it honest
//
// The header strip and the row share one track definition (catalogTracks). If they
// drifted, the labels would stop naming the columns underneath them and the whole
// "reads as a table" claim would quietly become false — so the tracks are defined once
// and both callers take them. That single shared function is the load-bearing part of
// this file.

// --- availability -------------------------------------------------------------

// Availability is the buyer-facing answer to Atlas's central question. It is domain
// vocabulary, and like [Posture] it maps onto the four visual [StatusTone] semantics
// through one function, so the catalog can grow states without the palette growing hues.
//
// It exists so the catalog cannot be handed a color. A caller passes "this is inbound"
// and the design system decides what that looks like; there is no way to ask for a green
// chip on an out-of-stock line.
type Availability int

const (
	// AvailStocked — on hand in the hub that serves this buyer. Ships on the promise date.
	AvailStocked Availability = iota
	// AvailNearby — on hand, but in another hub. Still a yes, on a longer lane.
	AvailNearby
	// AvailInbound — not on hand, on a lane, has a promise date. A yes with a wait.
	AvailInbound
	// AvailNone — no stock and no promise. The action changes to a notification.
	AvailNone
)

const availCount = int(AvailNone) + 1

// Label is the printed availability text: short, uppercase, buyer vocabulary rather than
// operator vocabulary. An operator says "ON HAND 4"; a buyer needs "IN STOCK".
func (parseAvail Availability) Label() string {
	switch parseAvail {
	case AvailNearby:
		return "NEARBY HUB"
	case AvailInbound:
		return "INBOUND"
	case AvailNone:
		return "NOT STOCKED"
	default:
		return "IN STOCK"
	}
}

// Tone maps availability onto the four visual semantics.
//
// The interesting mapping is AvailNone -> ToneNeutral, and it is deliberate. The
// tempting choice is ToneException, because out-of-stock is the row a buyer most wants
// to skip. It is the wrong choice for two reasons:
//
//  1. ToneException is FILLED (see status.go), and filled is the design system's
//     scarcest resource. A catalog filtered to a thin category can easily be twelve
//     unstocked lines, and twelve solid oxide blocks is a wall — the exact
//     everything-shouting failure the whole system exists to prevent. Exception has to
//     stay rare to stay loud.
//  2. Oxide means "a human must act on this". Nothing has gone wrong when a product is
//     not stocked; no discrepancy was found and nobody has to reconcile anything. Using
//     the exception hue for "unavailable" would teach operators — who read the same
//     palette on the console — that oxide sometimes means nothing.
//
// The unavailable line is de-emphasized instead: neutral chip, graphite price, and a
// different action label. Absence of emphasis is a signal too, and it is the cheap one.
func (parseAvail Availability) Tone() StatusTone {
	switch parseAvail {
	case AvailNearby, AvailInbound:
		return TonePending
	case AvailNone:
		return ToneNeutral
	default:
		return ToneVerified
	}
}

func clampAvail(parseAvail Availability) Availability {
	if parseAvail < 0 || int(parseAvail) >= availCount {
		return AvailNone
	}
	return parseAvail
}

// --- the shared track definition ----------------------------------------------

// catalogTracks is the column geometry, and it is shared by the header strip and every
// row precisely so they cannot drift. Change it here or the labels stop naming the
// columns they sit over.
//
// The desktop tracks, and why each is the width it is:
//
//	4rem              thumbnail — square, big enough to recognize a product, small
//	                  enough that twelve rows still fit a screen
//	minmax(0,1fr)     identity — title, SKU, category, summary. The 0 minimum is
//	                  load-bearing: a long unbroken SKU would otherwise set the track's
//	                  min-content width and blow the grid out (see Stack's comment)
//	13rem             availability — the promoted column
//	7rem              price
//	6rem              action label
//
// GAP NOTE: grid-template-columns, grid-template-areas and grid-area have no typed
// constructors, so every line of this geometry is css.Raw. This is the single biggest
// hole in the typed layer for real layout work and it is why this file has more Raw in
// it than the rest of the package combined.
func catalogTracks() []css.Rule {
	return css.Rules(
		css.Display.Grid,
		css.Raw("grid-template-columns", "4rem minmax(0,1fr) 13rem 7rem 6rem"),
		css.Raw("grid-template-areas", `"thumb id avail price action"`),
		css.Gap(Space4),
		// start, not center. Top-aligned matches the table's vertical-align:top and is
		// what paperwork does: the price aligns with the title it belongs to rather than
		// floating in the middle of a three-line identity block.
		css.Items.Start,
		css.MinWidth(css.Zero),

		// Narrow: the five tracks become two columns and three rows. The thumbnail keeps
		// its own column beside the identity (an image above a title wastes the height
		// that a phone has least of), and availability then price/action stack under
		// both, full width.
		//
		// This is the reflow a <table> cannot do, and it is the concrete reason this is
		// not a table.
		css.Media(css.MaxW(breakRail),
			css.Raw("grid-template-columns", "3.5rem minmax(0,1fr)"),
			css.Raw("grid-template-areas", `"thumb id" "avail avail" "price action"`),
			css.Gap(Space3),
			css.Items.Center,
		),
		css.Media(css.MaxW(breakNarrow),
			css.Raw("grid-template-columns", "3rem minmax(0,1fr)"),
			css.Gap(Space2),
		),
	)
}

// --- the manifest container ---------------------------------------------------

var catalogManifestBundle = clip(css.Rules(
	// A real <ul>, with the bullets off. The list SEMANTICS are the point (a screen
	// reader announces the item count before the user commits to reading twenty-four
	// products); the list APPEARANCE is not.
	//
	// list-style:none on a <ul> famously removes the list role in Safari + VoiceOver.
	// The documented fix is role="list" on the element, which the Catalog component
	// sets — see its doc comment. If you hand-roll this bundle, set it yourself.
	css.Raw("list-style", "none"),
	css.Raw("margin", "0"),
	css.Raw("padding", "0"),
	css.Display.Flex,
	css.FlexDir.Col,
	css.MinWidth(css.Zero),
	css.W(css.Full),
))

// CatalogManifest is the <ul> holding the product lines. Apply it to the list element;
// the rows carry their own hairlines, so there is no gap here — the lines are ruled, not
// spaced, which is what makes the column reads work.
//
// Put it inside a [SurfaceFlush] so the rows reach the sheet edge, exactly as a table
// does. Do NOT put it inside a padded [Surface]: a manifest inside a 16px gutter wastes
// the alignment it was chosen for.
func CatalogManifest() []css.Rule { return catalogManifestBundle }

var catalogHeaderBundle = clip(css.Rules(
	catalogTracks(),
	// The same voice as a table header, for the same reason: a column label and a table
	// header do the same job, so they read as one document.
	Display(StepMicro),
	css.TextColor(Graphite()),
	css.PaddingX(Space4),
	css.Raw("padding-top", string(Space3)),
	css.Raw("padding-bottom", string(Space2)),
	// The heavy 2px ink rule. This one mark is what makes a <ul> read as a manifest
	// rather than as a stack of links, and it is the same rule that closes a table
	// header — which is the whole trick of this primitive: table legibility, list
	// semantics.
	css.Raw("border-bottom", manifestRule()),
	// Below the rail breakpoint the tracks restack, so a five-label header strip is
	// labelling columns that no longer exist. Same call as RailGroupLabel: a mark that
	// stops doing its job in a layout is removed in that layout.
	css.Media(css.MaxW(breakRail), css.Display.None),
))

// CatalogHeaderStrip is the labelled column header above the manifest, sharing the row's
// exact track geometry. Hidden below 960px, where the tracks restack.
//
// It is not decoration. Without it the availability column is an unexplained pair of
// mono strings, and the "reads as a table" claim this primitive makes is false.
func CatalogHeaderStrip() []css.Rule { return catalogHeaderBundle }

// --- the row ------------------------------------------------------------------

// catalogActionMark is the attribute the row's hover rule reaches for.
//
// The action label has to react to hover on the ROW, not on itself, which means the row
// bundle must select a descendant. Selecting it by generated class name would couple two
// bundles through a hash; selecting it by element type would catch every span in the row.
// A data attribute is the third option: greppable, stable, and it makes the coupling
// visible in the markup instead of implicit in the CSS.
const catalogActionMark = "data-atlas-catalog-action"

var catalogRowBundle = clip(css.Rules(
	catalogTracks(),
	css.PaddingX(Space4),
	css.PaddingY(Space4),
	css.Raw("border-bottom", hairlineRule()),
	css.TextColor(Ink()),
	css.Raw("text-decoration", "none"),
	css.Cursor.Pointer,
	css.Position.Relative,
	withMotion(css.PropColors, css.Ms(120)),

	// NO zebra, and that is a decision rather than an omission.
	//
	// Table earns its stripes because its rows are ~34px tall and 12+ columns wide, where
	// tracking one row across the screen is the real failure. A catalog line is ~96px
	// tall and five columns wide; the hairline already bounds it unambiguously, and a
	// stripe on a 96px block stops being a tracking aid and becomes a second surface
	// color — which is the nested-box look arriving through the back door.
	css.Hover(css.Bg(PaperSunk())),

	// Hover also darkens the action label and underlines it, so the affordance responds
	// to the row rather than only to itself. Specificity 0-1-2 against the action
	// bundle's own 0-1-0, which is the intended direction: the row owns the state.
	css.Hover(
		css.Descendant(css.AttrSel(catalogActionMark),
			css.TextColor(Ink()),
			css.Raw("text-decoration", "underline"),
			css.Raw("text-underline-offset", "0.18em"),
		)...,
	),

	// Inset ring. The row is full-bleed inside SurfaceFlush, so an outset ring would be
	// clipped by the sheet's overflow:hidden on the first and last rows — visible on the
	// middle twenty-two and gone on the two you are most likely to tab to first.
	css.FocusVisible(
		css.Outline(ManifestRuleWidth, Lane()),
		css.OutlineOffset(css.Px(-2)),
	),

	// PRINT: a printed catalog is a price list. Keep the row, drop the interactive
	// affordances (the pointer cursor is meaningless and the row must not break across
	// the fold), and let the action label remove itself.
	css.Media(printQuery,
		css.Raw("break-inside", "avoid"),
		css.Raw("page-break-inside", "avoid"),
		css.PaddingX(css.Zero),
		css.PaddingY(Space3),
	),
))

// CatalogRow is the manifest line, for callers composing their own row body. Apply it to
// an <a> — the row IS the link.
//
// [CatalogLine] is the shape you almost always want; see its doc for why the row is a
// link rather than a container with a button in it.
func CatalogRow() []css.Rule { return catalogRowBundle }

// --- the thumbnail ------------------------------------------------------------

var catalogThumbBase = css.Rules(
	css.Raw("grid-area", "thumb"),
	css.W(css.Full),
	// A square, enforced by aspect-ratio rather than by a matching height, so the box
	// stays square when the track width changes at the two breakpoints. One declaration
	// instead of three media queries.
	css.Raw("aspect-ratio", "1 / 1"),
	css.Bg(PaperSunk()),
	css.Border(HairlineWidth, Hairline()),
	css.Rounded(RadiusTag),
	css.Raw("overflow", "hidden"),
)

var catalogThumbBundle = clip(css.Rules(
	catalogThumbBase,
	css.Display.Block,
	// object-fit:cover so a non-square source crops instead of distorting. A stretched
	// product photo is worse than no photo — it misrepresents the product.
	css.Raw("object-fit", "cover"),
))

// CatalogThumb is the product image cell: a square on pressed paper with a hairline,
// cropped rather than stretched. Apply it to an <img>.
//
// It is pressed paper (not white) and it has a hairline, because an image with no frame
// on a manila page reads as a hole in the page. Give the <img> width and height
// attributes so the row does not reflow when the image loads, and loading="lazy" — a
// catalog is the one Atlas surface with twenty-four images below the fold.
func CatalogThumb() []css.Rule { return catalogThumbBundle }

var catalogThumbPlateBundle = clip(css.Rules(
	catalogThumbBase,
	css.Display.Flex,
	css.Items.Center,
	css.Justify.Center,
	Data(StepMicro),
	css.FontWeight.Bold,
	css.TextColor(Graphite()),
	css.Raw("text-align", "center"),
	css.Raw("word-break", "break-all"),
	css.Raw("padding", "2px"),
	// Tight leading so a two-line wrapped code still fits the 64px plate.
	css.LineHeight(css.Number("1.1")),
))

// CatalogThumbPlate is the thumbnail slot when there IS no image: a bin label.
//
// This matters more than it sounds, because Atlas's product data has no image field —
// repository.Product is SKU, slug, title, category, price, status, summary. A catalog
// primitive whose thumbnail column only works once somebody ships an image pipeline is a
// primitive that ships as an empty grey column.
//
// The alternative to a grey box with a picture icon is the one a warehouse actually uses:
// print the code on a plate and stick it on the bin. So the empty state of the image cell
// is the SKU's distinguishing segment, set in mono on pressed paper — which is legible,
// scannable, tells the reader something true, and stops looking like a broken image. When
// real photography arrives, the same slot takes [CatalogThumb] and nothing else changes.
func CatalogThumbPlate() []css.Rule { return catalogThumbPlateBundle }

// --- the identity cell --------------------------------------------------------

var catalogIdentityBundle = clip(css.Rules(
	css.Raw("grid-area", "id"),
	css.Display.Flex,
	css.FlexDir.Col,
	css.Gap(css.Length("3px")),
	css.MinWidth(css.Zero),
))

// CatalogIdentity is the "what is this" cell: title, then the SKU/category line, then the
// summary.
func CatalogIdentity() []css.Rule { return catalogIdentityBundle }

var catalogTitleBundle = clip(css.Rules(
	// PROSE, not Display, and this is the one place in the package where that will look
	// like a violation of the type-role rule. It is not, and the distinction is worth
	// getting right because it generalizes.
	//
	// Display NAMES A REGION of the page — a page title, a section title, a column
	// header, a button label. It is condensed, heavy and uppercase, which caps out at
	// about six words before it stops being readable. A product title is not a region
	// name, it is CONTENT: it is the longest, most-read, most-varied string on the row,
	// twenty-four of them are stacked, and set in condensed uppercase they would be a
	// wall AND would visually outrank the actual page title above them.
	//
	// So: Prose, at the base step, semibold. Weight carries the emphasis, and weight is
	// the cheap channel — it costs no color, no box and no case change.
	Prose(StepBase),
	css.FontWeight.Semibold,
	css.TextColor(Ink()),
	css.Raw("text-wrap", "balance"),
	css.MinWidth(css.Zero),
))

// CatalogTitle is the product name: proportional, semibold, ink. See the bundle for why
// it is Prose rather than Display — that argument is the one most likely to be
// "corrected" by someone applying the display rule mechanically.
func CatalogTitle() []css.Rule { return catalogTitleBundle }

var catalogIdentityMetaBundle = clip(css.Rules(
	Cluster(Space2),
	Data(StepMicro),
	css.TextColor(Graphite()),
	css.FontWeight.Semibold,
	css.FontVariantNumeric.TabularNums,
	// A SKU is a code; a ligature or a proportional digit inside a code hides a
	// character. Same reasoning as NumericCell.
	css.Raw("font-variant-ligatures", "none"),
))

// CatalogIdentityMeta is the mono line under the title: SKU, category, any other code.
// Set it as a Cluster of spans rather than one pre-joined string, so the separator is a
// styling decision and the SKU stays selectable on its own.
func CatalogIdentityMeta() []css.Rule { return catalogIdentityMetaBundle }

var catalogSummaryBundle = clip(css.Rules(
	Prose(StepFine),
	css.TextColor(Graphite()),
	css.MaxWidth(css.RawLength("62ch")),

	// Clamped to two lines. This is the rule that keeps a manifest a manifest: without
	// it a product with a 60-word summary makes one row four times taller than its
	// neighbours, the price column stops being scannable, and the whole density argument
	// for not using cards evaporates. A caller cannot be trusted to keep summaries short
	// — they come out of a CMS.
	//
	// GAP NOTE: the -webkit-box line-clamp idiom has no typed constructor and is four
	// css.Raw declarations. It is also, despite the prefix, the interoperable way to do
	// this — every current engine implements it, and the unprefixed `line-clamp` is not
	// yet safe to rely on alone.
	css.Raw("display", "-webkit-box"),
	css.Raw("-webkit-line-clamp", "2"),
	css.Raw("-webkit-box-orient", "vertical"),
	css.Raw("overflow", "hidden"),

	// Below the rail breakpoint the identity cell shares a row with the thumbnail and
	// there is no width to spend on a summary. The title and the codes are what a buyer
	// scans on a phone; the summary is on the product page one tap away.
	css.Media(css.MaxW(breakRail), css.Display.None),
))

// CatalogSummary is the product blurb: prose, graphite, clamped to two lines, hidden on
// narrow viewports. The clamp is not cosmetic — see the bundle.
func CatalogSummary() []css.Rule { return catalogSummaryBundle }

// --- the availability cell ----------------------------------------------------

var catalogAvailabilityBundle = clip(css.Rules(
	css.Raw("grid-area", "avail"),
	css.Display.Flex,
	css.FlexDir.Col,
	css.Items.Start,
	css.Gap(css.Length("4px")),
	css.MinWidth(css.Zero),

	// A leading hairline, and this is the promotion Atlas's thesis is owed. The rule
	// splits the row into "what this is" and "whether you can have it" — the two
	// questions a buyer actually has, in that order — and it is a hairline rather than a
	// box because the package's answer to "these need separating" is a rule (see
	// surfaces.go). It is the only structural mark inside a row.
	css.Raw("border-left", hairlineRule()),
	css.Raw("padding-left", string(Space4)),
	// align-self:stretch against the grid's align-items:start, so the rule spans the full
	// row height rather than only the height of the chip and the promise line. A rule two
	// lines tall beside a three-line identity block reads as a fragment of a border, not
	// as a divider — the first render of this looked like a rendering artifact.
	css.Raw("align-self", "stretch"),

	// Narrow: the cell is now a full-width band under the identity, so the separator
	// rotates from a leading rule to a rule above it, and the two facts sit side by side
	// instead of stacked. Same mark, same meaning, correct axis.
	css.Media(css.MaxW(breakRail),
		css.Raw("border-left", "0"),
		css.Raw("padding-left", "0"),
		css.Raw("border-top", hairlineRule()),
		css.Raw("padding-top", string(Space2)),
		css.FlexDir.Row,
		css.Items.Center,
		css.Raw("flex-wrap", "wrap"),
		css.Gap(Space2),
	),
))

// CatalogAvailability is the promoted "can I have it" cell: a [StatusChip] over a mono
// hub/promise line. It carries a leading hairline on wide viewports and a rule above it
// on narrow ones.
//
// This column is the reason the whole primitive exists. If you are tempted to move it
// after price, or to drop it into the summary, re-read the file header: Atlas's product
// is not a catalog, it is an answer about availability, and the catalog is where that
// answer either shows up or does not.
func CatalogAvailability() []css.Rule { return catalogAvailabilityBundle }

var catalogPromiseBundle = clip(css.Rules(
	Data(StepMicro),
	css.TextColor(Graphite()),
	css.FontVariantNumeric.TabularNums,
	css.Raw("font-variant-ligatures", "none"),
	css.Raw("white-space", "nowrap"),
))

// CatalogPromise is the mono hub-and-date line under the availability chip:
// "IL-HUB · 2026-08-04". Both are machine facts, so Data — and being mono and tabular
// they align down the column, which is what makes twelve promise dates comparable at a
// glance. Format the string yourself; this package does not know Atlas's date format.
func CatalogPromise() []css.Rule { return catalogPromiseBundle }

// --- price and action ---------------------------------------------------------

var catalogPriceBundle = clip(css.Rules(
	css.Raw("grid-area", "price"),
	// StepLede, one step above the title's size, and mono. A price is the single most
	// compared value on the surface, so it gets the one size bump on the row — and mono
	// plus tabular figures is what lets "$1,240.00" and "$980.00" line up their decimal
	// points down twenty-four rows.
	Data(StepLede),
	css.FontWeight.Semibold,
	css.TextColor(Ink()),
	css.Raw("text-align", "right"),
	css.FontVariantNumeric.TabularNums,
	css.Raw("font-variant-ligatures", "none"),
	css.Raw("white-space", "nowrap"),
	// Narrow: the price shares a line with the action, so it goes back to the leading
	// edge. Right-aligning it there would push it into the middle of the row.
	css.Media(css.MaxW(breakRail), css.Raw("text-align", "left")),
))

// CatalogPrice is the price cell: mono, tabular, one step up, right-aligned into a
// column.
//
// For a line the buyer cannot buy, compose it with StatusValue(ToneNeutral) rather than
// adding a second bundle — the price of something unavailable is context, not content,
// and graphite is the package's only de-emphasis tool. [CatalogLine] does this for you.
func CatalogPrice() []css.Rule { return catalogPriceBundle }

var catalogActionBundle = clip(css.Rules(
	css.Raw("grid-area", "action"),
	css.Display.InlineFlex,
	css.Items.Center,
	css.Justify.End,
	css.Gap(Space1),
	// Display micro: this is a label naming an action, the same voice as a button label
	// and a table header, which is what keeps the interface reading as one document.
	Display(StepMicro),
	css.TextColor(Lane()),
	css.Raw("white-space", "nowrap"),
	css.W(css.Full),

	// PRINT: the affordance is a lie on paper. A price list with "VIEW →" at the end of
	// every line is noise, and the row is not clickable in a filing cabinet.
	css.Media(printQuery, css.Display.None),
))

// CatalogActionLabel is the trailing affordance: "VIEW →", "NOTIFY ME". It is a <span>,
// never a <button> — see [CatalogLine] for the argument. It darkens and underlines when
// the row is hovered, driven from [CatalogRow], and it removes itself in print.
func CatalogActionLabel() []css.Rule { return catalogActionBundle }

// --- empty and filtered states ------------------------------------------------

var catalogResultNoteBundle = clip(css.Rules(
	Cluster(Space2),
	Data(StepFine),
	css.TextColor(Graphite()),
	css.PaddingX(Space4),
	css.Raw("padding-top", string(Space3)),
	css.Raw("padding-bottom", string(Space2)),
))

// CatalogResultNote is the count line above the manifest: "14 OF 62 LINES · CATEGORY
// DESKS". It is Data because a count and a filter expression are machine facts.
//
// It exists because a filtered result set that does not say it is filtered is the most
// common way a catalog lies to a buyer: twelve products where there are sixty-two, with
// nothing on screen to say why, reads as "this shop is nearly empty" rather than as "your
// filter is narrow". The count is the cheapest possible fix and it belongs to the
// manifest, not to the filter bar — the filter bar is what the buyer already stopped
// looking at.
func CatalogResultNote() []css.Rule { return catalogResultNoteBundle }

var catalogEmptyBundle = clip(css.Rules(
	css.Display.Flex,
	css.FlexDir.Col,
	css.Items.Center,
	css.Gap(Space3),
	css.Raw("text-align", "center"),
	css.PaddingY(Space7),
	css.PaddingX(Space4),
	// Pressed paper, no border, no radius. It is the same sheet with nothing printed on
	// it, which is exactly what an empty manifest is — NOT a card announcing its own
	// emptiness. A bordered empty-state box is the nested-card move at its most
	// pointless: a frame around nothing.
	css.Bg(PaperSunk()),
	css.MinWidth(css.Zero),
))

// CatalogEmpty is the void: no results, no rows, nothing printed on the sheet.
//
// The empty result set is a state a catalog is in constantly (every over-narrow filter
// produces one) and it is the state most likely to be left as a bare string, so it is
// part of the primitive rather than left to the caller. [Catalog] renders it
// automatically when there are no items, which is the point — a caller cannot forget a
// state the component owns.
func CatalogEmpty() []css.Rule { return catalogEmptyBundle }

// There is deliberately NO CatalogEmptyTitle. The empty state's heading is
// [SectionTitle], and the first draft of this file did ship a separate bundle for it —
// which TestDistinctPrimitivesFoldToDistinctClasses immediately caught as byte-identical
// to SectionTitle, because it was.
//
// That is the test doing its actual job. A design system with two names for one rule-set
// has two places to change it, and they will diverge. Display at StepSubhead is correct
// here for the reason CatalogTitle is NOT Display: "NO LINES ON THIS MANIFEST" names a
// region of the page, it is short, and there is exactly one of it.

var catalogEmptyBodyBundle = clip(css.Rules(
	Prose(StepBase),
	css.TextColor(Graphite()),
	css.MaxWidth(css.RawLength("46ch")),
))

// CatalogEmptyBody is the sentence under the empty state's heading: what happened and
// what to do about it. Prose, because it is a sentence someone wrote.
func CatalogEmptyBody() []css.Rule { return catalogEmptyBodyBundle }

// --- the components -----------------------------------------------------------

// CatalogItem is one product line. Every string is caller-formatted, for the same reason
// PlacardSpec's are: this package does not know Atlas's currency format, date format or
// SKU scheme, and a design package that starts formatting domain values stops being
// reusable.
//
// The field set is deliberately Atlas's real product shape (repository.Product plus the
// warehouse columns from InventoryRow) rather than a generic one, because a catalog
// primitive that cannot express warehouse-aware availability is not a catalog primitive
// for THIS product.
type CatalogItem struct {
	// Href is where the row points. A row with no Href renders as a non-interactive
	// line rather than as a dead link — see CatalogLine.
	Href string

	// Title is the product name, e.g. "Meridian Height-Adjustable Desk". The longest and
	// most-read string on the row.
	Title string

	// SKU is the stock code, e.g. "SKU-40192". Set in mono; also supplies the thumbnail
	// plate's code when there is no image.
	SKU string

	// Category is the taxonomy label, e.g. "Desks". Rendered uppercase by the mono meta
	// line; pass it in whatever case your data holds.
	Category string

	// Price is the formatted price, e.g. "$1,240.00". Atlas stores cents; format them at
	// the call site, where the locale is known.
	Price string

	// Summary is one or two sentences. Clamped to two rendered lines and hidden on narrow
	// viewports, so put the distinguishing detail first.
	Summary string

	// Avail is the buyer-facing availability state. It drives the chip tone, the chip
	// label, the price emphasis and the action label — one field, four consequences,
	// which is what keeps an unavailable line from being rendered as an available one.
	Avail Availability

	// Hub is the warehouse code the promise is made from, e.g. "IL-HUB". Optional.
	Hub string

	// Promise is the formatted promise date, e.g. "2026-08-04". Optional. Prefer an
	// ISO-ish sortable form: it is mono and tabular, so a column of them aligns.
	Promise string

	// ThumbSrc and ThumbAlt are the product image. When ThumbSrc is empty the slot falls
	// back to a SKU plate (see CatalogThumbPlate), which is why this primitive works
	// against Atlas's current image-less data.
	ThumbSrc string
	ThumbAlt string

	// ActionLabel overrides the generated affordance text ("VIEW", "NOTIFY ME"). Leave
	// it empty unless the generated word is wrong for the context.
	ActionLabel string

	// Label overrides the generated accessible name for the row. Leave it empty unless
	// the generated sentence is wrong.
	Label string
}

// CatalogSpec is a whole manifest, including the states a hand-rolled catalog forgets.
type CatalogSpec struct {
	// Items are the lines to render, already filtered and sorted by the caller.
	Items []CatalogItem

	// TotalCount is how many lines exist BEFORE filtering. When it is greater than
	// len(Items) the result note reads "14 of 62"; when it is zero or equal the note
	// omits the total. This is the field that stops a filtered catalog from looking like
	// an empty shop.
	TotalCount int

	// FilterSummary is a formatted description of the active filter, e.g.
	// `CATEGORY DESKS · IN STOCK`. Shown in the result note and in the empty state, where
	// it is the difference between "nothing exists" and "nothing matches what you asked".
	FilterSummary string

	// Label is the accessible name of the list, e.g. "Product catalog".
	Label string

	// EmptyTitle and EmptyBody override the empty state's copy. Supply them: the defaults
	// are deliberately generic, and the words on your screen are your domain's, not this
	// package's.
	EmptyTitle string
	EmptyBody  string

	// EmptyActionLabel and EmptyActionHref add one recovery action to the empty state —
	// almost always "clear the filter". Both must be set or the action is omitted.
	EmptyActionLabel string
	EmptyActionHref  string
}

// Catalog renders a whole product manifest: the result note, the labelled header strip
// and one line per item — or the empty state when there are none.
//
//	design.Catalog(design.CatalogSpec{
//	    Items:         lines,
//	    TotalCount:    62,
//	    FilterSummary: "CATEGORY DESKS",
//	    Label:         "Product catalog",
//	    EmptyTitle:    "No lines match this filter",
//	    EmptyBody:     "Widen the category or clear the stock filter to see the full manifest.",
//	    EmptyActionLabel: "Clear filters",
//	    EmptyActionHref:  "/shop",
//	})
//
// It is a component rather than a bundle for the same reason LanePlacard is: the
// structure IS the element. The header strip has to share the rows' track geometry, the
// list needs role="list" to survive list-style:none in Safari, and the empty and filtered
// states have to exist. Three call sites reassembling that by hand is three chances to
// ship a catalog that is a stack of links with no columns and a blank page when a filter
// misses.
//
// The three states this closes, which the brief called out and which hand-rolled
// catalogs always drop:
//
//   - EMPTY: no items at all -> the void, with copy and one recovery action.
//   - FILTERED: fewer items than TotalCount -> a result note saying so, so a narrow
//     filter does not read as an empty shop.
//   - ORPHANS: structurally impossible. Rows are full-width, so four items is four rows.
//     There is no last-row-with-two-holes case to handle, which is the whole reason a
//     3-column grid was rejected.
func Catalog(parseSpec CatalogSpec) ui.Node {
	parseChildren := make([]ui.Node, 0, len(parseSpec.Items)+3)

	if parseNote := catalogNoteText(parseSpec); parseNote != "" {
		parseChildren = append(parseChildren,
			html.Div(html.Props{Class: Class(catalogResultNoteBundle)}, html.Text(parseNote)))
	}

	if len(parseSpec.Items) == 0 {
		parseChildren = append(parseChildren, catalogEmptyBlock(parseSpec))
		return html.Section(html.Props{Class: Class(Stack(Space1))}, parseChildren...)
	}

	parseChildren = append(parseChildren,
		html.Div(html.Props{Class: Class(catalogHeaderBundle), Aria: map[string]string{"hidden": "true"}},
			// aria-hidden, because the header strip is a VISUAL column key. Each row
			// already carries its own accessible name spelling out price and
			// availability, so announcing five orphan column labels before the list would
			// be noise with nothing to attach to — a list is not a grid and there is no
			// row/column relationship for a screen reader to use them in.
			html.Span(html.Props{}, html.Text("")),
			html.Span(html.Props{}, html.Text("Product")),
			html.Span(html.Props{}, html.Text("Availability")),
			html.Span(html.Props{Class: Class(catalogHeaderPriceBundle)}, html.Text("Price")),
			html.Span(html.Props{}, html.Text("")),
		))

	parseLines := make([]ui.Node, 0, len(parseSpec.Items))
	for _, parseItem := range parseSpec.Items {
		parseLines = append(parseLines, CatalogLine(parseItem))
	}

	parseListProps := html.Props{
		Class: Class(catalogManifestBundle),
		// role="list" restores the list role that list-style:none removes in Safari with
		// VoiceOver. Without it the item count — the single most useful thing a screen
		// reader can say before a user commits to twenty-four products — is not announced.
		Role: "list",
	}
	if parseLabel := strings.TrimSpace(parseSpec.Label); parseLabel != "" {
		parseListProps.Aria = map[string]string{"label": parseLabel}
	}

	parseChildren = append(parseChildren, html.Ul(parseListProps, parseLines...))
	return html.Section(html.Props{Class: Class(Stack(Space1))}, parseChildren...)
}

var catalogHeaderPriceBundle = clip(css.Rules(
	css.Raw("grid-area", "price"),
	css.Raw("text-align", "right"),
))

// CatalogLine renders one product manifest line.
//
// # Why the row is a link and the action is a span
//
// The row is a single <a> laid out as a grid, and the trailing "VIEW →" is a <span>
// inside it, not a <button>. Three alternatives were considered:
//
//   - Row as a container, title as the link, plus a real action button. Correct HTML, but
//     it gives a mouse user an 800x96 target of which only the title is clickable, which
//     everyone finds infuriating and which every catalog on the web has therefore
//     abandoned.
//   - Row as a link WITH a nested button. Invalid: interactive content cannot nest inside
//     an <a>. Browsers recover unpredictably and keyboard order gets strange.
//   - Row as a link, action as a label. One tab stop per product, whole row clickable,
//     no nested interactive content, and the label still tells the reader what the row
//     will do — which is the only job it had.
//
// The third is what this is. A future contributor wanting an "Add to cart" button in the
// row should note that adding it requires unpicking the row-as-link decision, and that
// the right place for a second action is the product page, not twenty-four rows.
//
// # Accessibility
//
// The row's accessible name is a spelled-out sentence ("Meridian Desk, SKU-40192,
// $1,240.00, IN STOCK from IL-HUB, promise 2026-08-04"). Read in DOM order the row would
// announce as a run of disconnected fragments, and the "·" separators would be read as
// punctuation noise, so the label is written explicitly — the same cost the lane placard
// pays for being graphic. The "·" glyphs and the "→" are aria-hidden.
func CatalogLine(parseItem CatalogItem) ui.Node {
	parseAvail := clampAvail(parseItem.Avail)

	parseIdentity := []ui.Node{
		html.Span(html.Props{Class: Class(catalogTitleBundle)}, html.Text(parseItem.Title)),
	}
	if parseMeta := catalogMetaNodes(parseItem); len(parseMeta) > 0 {
		parseIdentity = append(parseIdentity,
			html.Div(html.Props{Class: Class(catalogIdentityMetaBundle)}, parseMeta...))
	}
	if strings.TrimSpace(parseItem.Summary) != "" {
		parseIdentity = append(parseIdentity,
			html.P(html.Props{Class: Class(catalogSummaryBundle)}, html.Text(parseItem.Summary)))
	}

	parseAvailability := []ui.Node{
		html.Span(html.Props{Class: Class(statusChipBundles[clampTone(parseAvail.Tone())])},
			html.Text(parseAvail.Label())),
	}
	if parsePromise := catalogPromiseText(parseItem); parsePromise != "" {
		parseAvailability = append(parseAvailability,
			html.Span(html.Props{Class: Class(catalogPromiseBundle)}, html.Text(parsePromise)))
	}

	// An unavailable line's price is context rather than content, so it drops to graphite
	// through the existing semantic API instead of through a second price bundle. Note
	// there is still no way to ask this primitive for a color.
	parsePriceClass := Class(catalogPriceBundle)
	if parseAvail == AvailNone {
		parsePriceClass = Class(catalogPriceBundle, StatusValue(ToneNeutral))
	}

	parseRowChildren := []ui.Node{
		catalogThumbNode(parseItem),
		html.Div(html.Props{Class: Class(catalogIdentityBundle)}, parseIdentity...),
		html.Div(html.Props{Class: Class(catalogAvailabilityBundle)}, parseAvailability...),
		html.Span(html.Props{Class: parsePriceClass}, html.Text(parseItem.Price)),
		html.Span(
			html.Props{
				Class: Class(catalogActionBundle),
				Raw:   map[string]any{catalogActionMark: "true"},
			},
			html.Text(catalogActionText(parseItem, parseAvail)),
			html.Span(html.Props{Aria: map[string]string{"hidden": "true"}}, html.Text("→")),
		),
	}

	parseAnchorProps := html.Props{
		Class: Class(catalogRowBundle),
		Href:  parseItem.Href,
		Aria:  map[string]string{"label": catalogAccessibleLabel(parseItem, parseAvail)},
	}

	return html.Li(html.Props{Role: "listitem"},
		html.A(parseAnchorProps, parseRowChildren...),
	)
}

// catalogThumbNode picks the image or the SKU plate. The plate is not a placeholder for
// a missing feature — see CatalogThumbPlate.
func catalogThumbNode(parseItem CatalogItem) ui.Node {
	if strings.TrimSpace(parseItem.ThumbSrc) != "" {
		return html.Img(html.Props{
			Class: Class(catalogThumbBundle),
			Src:   parseItem.ThumbSrc,
			// The alt text falls back to the title rather than to "" — an image that IS
			// the product needs a name, and an empty alt would make the row's thumbnail
			// invisible to a screen reader that is looking for the product.
			Alt: catalogThumbAlt(parseItem),
			// Explicit intrinsic size so the row does not reflow as twenty-four images
			// arrive, and lazy loading because a catalog is the one Atlas surface with
			// most of its images below the fold.
			Width:   "64",
			Height:  "64",
			Loading: "lazy",
		})
	}
	return html.Div(
		html.Props{
			Class: Class(catalogThumbPlateBundle),
			// The plate repeats the SKU, which the meta line already announces, so it is
			// hidden from assistive technology rather than read twice.
			Aria: map[string]string{"hidden": "true"},
		},
		html.Text(catalogPlateCode(parseItem.SKU)),
	)
}

// catalogPlateCode reduces a SKU to its distinguishing tail so it fits a 64px plate.
//
// "SKU-40192" becomes "40192": the "SKU-" prefix is on every line and therefore carries
// no information, which is the same reason a warehouse bin label prints the number and
// not the word. Codes with no separator are kept whole and wrap (word-break:break-all).
func catalogPlateCode(parseSKU string) string {
	parseSKU = strings.TrimSpace(parseSKU)
	if parseSKU == "" {
		return "—"
	}
	if parseIndex := strings.LastIndex(parseSKU, "-"); parseIndex >= 0 && parseIndex < len(parseSKU)-1 {
		return parseSKU[parseIndex+1:]
	}
	return parseSKU
}

func catalogThumbAlt(parseItem CatalogItem) string {
	if parseAlt := strings.TrimSpace(parseItem.ThumbAlt); parseAlt != "" {
		return parseAlt
	}
	return parseItem.Title
}

// catalogMetaNodes builds the mono SKU/category line, separating the parts with an
// aria-hidden middot so a screen reader does not read "middle dot".
func catalogMetaNodes(parseItem CatalogItem) []ui.Node {
	parseParts := make([]string, 0, 2)
	if parseSKU := strings.TrimSpace(parseItem.SKU); parseSKU != "" {
		parseParts = append(parseParts, parseSKU)
	}
	if parseCategory := strings.TrimSpace(parseItem.Category); parseCategory != "" {
		parseParts = append(parseParts, strings.ToUpper(parseCategory))
	}
	parseNodes := make([]ui.Node, 0, len(parseParts)*2)
	for parseIndex, parsePart := range parseParts {
		if parseIndex > 0 {
			parseNodes = append(parseNodes,
				html.Span(html.Props{Aria: map[string]string{"hidden": "true"}}, html.Text("·")))
		}
		parseNodes = append(parseNodes, html.Span(html.Props{}, html.Text(parsePart)))
	}
	return parseNodes
}

// catalogPromiseText joins the hub and the promise date into the one mono line under the
// chip. Either may be absent; an absent field is omitted rather than rendered blank, the
// same contract PlacardSpec has.
func catalogPromiseText(parseItem CatalogItem) string {
	parseHub := strings.TrimSpace(parseItem.Hub)
	parsePromise := strings.TrimSpace(parseItem.Promise)
	switch {
	case parseHub != "" && parsePromise != "":
		return parseHub + " · " + parsePromise
	case parseHub != "":
		return parseHub
	default:
		return parsePromise
	}
}

// catalogActionText picks the affordance word from the availability state.
//
// The action changes with availability because that is the honest thing for it to do: a
// line you cannot buy leads somewhere different from a line you can, and a row that says
// "VIEW" on an unstocked product spends a click to deliver a disappointment. This is also
// the second channel that keeps AvailNone legible without spending the exception hue on
// it — the word changes, not just the color.
func catalogActionText(parseItem CatalogItem, parseAvail Availability) string {
	if parseLabel := strings.TrimSpace(parseItem.ActionLabel); parseLabel != "" {
		return parseLabel
	}
	if parseAvail == AvailNone {
		return "NOTIFY ME"
	}
	return "VIEW"
}

// catalogAccessibleLabel spells the row out as a sentence.
//
// Read in DOM order a row announces as "Meridian Desk SKU-40192 DESKS <summary> IN STOCK
// IL-HUB · 2026-08-04 $1,240.00 VIEW", which is a pile of fragments with no relationships
// and no indication that IL-HUB has anything to do with the date. Writing the label is
// the cost of a graphic row, and it is not optional — the same argument
// placardAccessibleLabel makes.
func catalogAccessibleLabel(parseItem CatalogItem, parseAvail Availability) string {
	if parseLabel := strings.TrimSpace(parseItem.Label); parseLabel != "" {
		return parseLabel
	}
	var parseBuilder strings.Builder
	parseBuilder.WriteString(parseItem.Title)
	if parseSKU := strings.TrimSpace(parseItem.SKU); parseSKU != "" {
		parseBuilder.WriteString(", ")
		parseBuilder.WriteString(parseSKU)
	}
	if parsePrice := strings.TrimSpace(parseItem.Price); parsePrice != "" {
		parseBuilder.WriteString(", ")
		parseBuilder.WriteString(parsePrice)
	}
	parseBuilder.WriteString(", ")
	parseBuilder.WriteString(parseAvail.Label())
	if parseHub := strings.TrimSpace(parseItem.Hub); parseHub != "" {
		parseBuilder.WriteString(" from ")
		parseBuilder.WriteString(parseHub)
	}
	if parsePromise := strings.TrimSpace(parseItem.Promise); parsePromise != "" {
		parseBuilder.WriteString(", promise ")
		parseBuilder.WriteString(parsePromise)
	}
	return parseBuilder.String()
}

// catalogNoteText formats the result count.
//
// It reports "N OF M LINES" only when the set is actually narrowed, because a note that
// says "62 of 62" on every unfiltered visit is chrome the reader learns to ignore — and
// then does not see on the one visit where it said 12.
func catalogNoteText(parseSpec CatalogSpec) string {
	parseShown := len(parseSpec.Items)
	var parseBuilder strings.Builder
	switch {
	case parseSpec.TotalCount > parseShown:
		parseBuilder.WriteString(itoa(parseShown))
		parseBuilder.WriteString(" OF ")
		parseBuilder.WriteString(itoa(parseSpec.TotalCount))
		parseBuilder.WriteString(" LINES")
	case parseShown > 0:
		parseBuilder.WriteString(itoa(parseShown))
		parseBuilder.WriteString(" LINES")
	}
	if parseFilter := strings.TrimSpace(parseSpec.FilterSummary); parseFilter != "" {
		if parseBuilder.Len() > 0 {
			parseBuilder.WriteString(" · ")
		}
		parseBuilder.WriteString(parseFilter)
	}
	return parseBuilder.String()
}

// catalogEmptyBlock renders the void.
//
// The default copy is intentionally bland. status.go's rule holds here too — "a design
// system should not be inventing the words on your screen" — but an empty state with NO
// words is a blank region that reads as a broken page, so the package ships a neutral
// sentence and the doc on CatalogSpec tells you to replace it.
func catalogEmptyBlock(parseSpec CatalogSpec) ui.Node {
	parseTitle := strings.TrimSpace(parseSpec.EmptyTitle)
	if parseTitle == "" {
		parseTitle = "No lines on this manifest"
	}
	parseBody := strings.TrimSpace(parseSpec.EmptyBody)
	if parseBody == "" {
		parseBody = "Nothing matches the current filter. Widen it or clear it to see the full catalog."
	}

	parseChildren := []ui.Node{
		html.H2(html.Props{Class: Class(sectionTitleBundle)}, html.Text(parseTitle)),
		html.P(html.Props{Class: Class(catalogEmptyBodyBundle)}, html.Text(parseBody)),
	}
	if parseFilter := strings.TrimSpace(parseSpec.FilterSummary); parseFilter != "" {
		// Echoing the active filter inside the void is the difference between "this shop
		// is empty" and "you asked for something narrow". It is the same information as
		// the result note, and it has to appear twice because in the empty case there is
		// no manifest under the note for the reader to have already been looking at.
		parseChildren = append(parseChildren,
			html.Span(html.Props{Class: Class(catalogPromiseBundle)}, html.Text(parseFilter)))
	}
	if parseLabel := strings.TrimSpace(parseSpec.EmptyActionLabel); parseLabel != "" &&
		strings.TrimSpace(parseSpec.EmptyActionHref) != "" {
		parseChildren = append(parseChildren,
			html.A(html.Props{
				Href:  parseSpec.EmptyActionHref,
				Class: Class(buttonSecondaryBundle),
			}, html.Text(parseLabel)))
	}
	return html.Div(html.Props{Class: Class(catalogEmptyBundle)}, parseChildren...)
}

// itoa is a local non-negative int formatter.
//
// strconv would pull the whole package in for one call in a design system that otherwise
// imports only css, html and ui. Counts are non-negative by construction (len of a
// slice, a row total), and the negative branch is a guard rather than a case.
func itoa(parseValue int) string {
	if parseValue == 0 {
		return "0"
	}
	parseNegative := parseValue < 0
	if parseNegative {
		parseValue = -parseValue
	}
	var parseDigits [20]byte
	parseIndex := len(parseDigits)
	for parseValue > 0 {
		parseIndex--
		parseDigits[parseIndex] = byte('0' + parseValue%10)
		parseValue /= 10
	}
	if parseNegative {
		parseIndex--
		parseDigits[parseIndex] = '-'
	}
	return string(parseDigits[parseIndex:])
}
