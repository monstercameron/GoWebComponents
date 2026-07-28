package design

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/css"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// The page shell: the frame every Atlas route renders inside.
//
// Atlas has two shells because it has two audiences, and conflating them was part of
// what made the old design incoherent:
//
//   - The CONSOLE (internal: inventory, receiving, transfers, purchase orders) is an
//     operator tool. It gets a persistent left rail, because an operator navigates
//     dozens of times per session and needs the map always visible.
//   - The STOREFRONT (public: landing, catalog, availability) is a document. It gets
//     a thin sticky header, because a shopper navigates rarely and wants the page.
//
// The old navigation rendered as four stacked pill boxes spanning the full width,
// which is the worst of both: it consumed the most valuable vertical space on the
// page (the top) to show four items, and it gave no sense of place. ConsoleRail
// replaces it.

var consoleShellBundle = clip(css.Rules(
	css.Display.Grid,
	// GAP NOTE: grid-template-columns / rows / area have no typed constructors, so
	// every grid definition in this package goes through css.Raw. This is the single
	// largest gap in the typed layer for real layout work.
	css.Raw("grid-template-columns", "var("+TokenRailWidth+") minmax(0,1fr)"),
	css.MinHeight(css.Vh(100)),
	css.Bg(Paper()),
	css.TextColor(Ink()),
	// Below the rail breakpoint the grid becomes a single column and the rail turns
	// into a horizontal strip above the content (see ConsoleRail). Note this is the
	// ONLY media query needed for the shell: minmax(0,1fr) plus the rail's own
	// narrow-mode rules do the rest.
	css.Media(css.MaxW(breakRail), css.Raw("grid-template-columns", "minmax(0,1fr)")),
	// Print collapses the same way. The rail sets display:none in print, but a
	// display:none grid ITEM still leaves its 232px track behind, so the document
	// would print with a blank left margin the width of a navigation bar nobody can
	// click. Collapsing the template here is the other half of that fix, and it is
	// why the two rules live in two files that have to agree.
	// min-height:100vh means "at least one full viewport" on screen and "at least one full
	// PAGE" on paper, so a two-line document prints a mostly blank sheet and a document
	// that ends near a page boundary prints a trailing blank one. The viewport floor is a
	// screen concern; paper has its own.
	css.Media(printQuery, css.Rules(
		css.Raw("grid-template-columns", "minmax(0,1fr)"),
		css.MinHeight(css.Auto),
	)...),
))

// ConsoleShell is the internal two-column frame: rail, then content column.
//
//	html.Div(html.Props{Class: design.Class(design.ConsoleShell())},
//	    html.Aside(html.Props{Class: design.Class(design.ConsoleRail())}, …),
//	    html.Main(html.Props{Class: design.Class(design.ContentColumn())}, …),
//	)
func ConsoleShell() []css.Rule { return consoleShellBundle }

// --- the rail as a placard column ---------------------------------------------
//
// "Left rail plus content column" is the most common application layout in
// existence, and until this section existed nothing about Atlas's rail was Atlas's.
// The typography and the lane placard carried the whole identity while the frame —
// the largest persistent element on every internal screen — free-rode on them.
//
// Three marks fix that, and the constraint on all three is that the placard has to
// stay the one bold thing. So the rail gets IDENTITY, not VOLUME:
//
//  1. A perforated trailing edge (railPerforation). The rail's right edge is bitten
//     by a column of paper-colored half-circles: the same tear-edge device as the
//     placard, rotated 90°, so the frame reads as a torn strip of manifest stock
//     rather than as a panel. It costs one background-image on an element that
//     already had a background, no DOM and no asset, and it inverts with the theme
//     for free because the discs are painted in var(--atlas-paper).
//  2. Hub codes in mono down the rail ([RailCode]). Every rail item carries its route
//     code at the trailing edge, set in Data/StepMicro. This is the package's own
//     fixed-advance-width argument applied to navigation: a column of mono codes
//     aligns, so the rail reads as a column of manifest lines. It is also real Atlas
//     information — an operator says "RCV" and "PO", not "receiving screen".
//  3. A rail plate ([RailPlate]) at the top: the console's own hub, printed as an
//     eyebrow plus a mono code and closed by a hairline. It is the rail's "who and
//     where am I" line, which is what a placard column has at its head.
//
// And Chanel's rule, applied: something came OFF to pay for it. The rail's items
// were 2px-rounded pills with a filled hover — the rounded-box vocabulary this
// design system explicitly rejects everywhere else. They are square-cut now
// (RadiusNone), which reads as ruled lines in a column instead of as a stack of
// chips, and which lets the current item's fill run cleanly to the torn edge.
//
// What was considered and rejected: an inked rail (a dark bar down the left) reads
// as a second placard and would have flatly competed with the real one; a 2px ink
// rule under the rail plate would have put a second heavy mark on every screen
// beside PageHead's. Both are the "add volume" answer. The perforation is the "add
// specificity" answer, and it is free.

// railPerforation is the torn trailing edge: a column of punched holes down the rail's
// right side.
//
// It is a background layer on the rail itself rather than a pseudo-element, so it needs no
// stacking context and survives the rail's own overflow-y:auto. background-attachment
// defaults to `scroll`, which for an element with internal scrolling means the layer is
// painted against the padding box and does NOT slide with the nav items — correct, because
// a torn edge belongs to the sheet, not to its contents.
//
// # Why the hole has a rim, when the placard's does not
//
// The placard's scallops are paper on ink: 16:1, so a flat disc reads instantly. The rail
// is paper-SUNK, and paper on paper-sunk is 1.10:1 — one step of value. The first version
// of this used a plain paper disc and it was, measurably and visibly, invisible; the rail's
// edge rendered as a plain hairline and the whole "frame reads as a placard column" claim
// was decoration that never arrived on screen.
//
// So each hole is drawn as a paper-colored disc with a HAIRLINE rim, which is what a punch
// through stock actually looks like and which lands the mark at exactly the visibility of
// every other rule in the system — the package's already-calibrated "quiet but present"
// level. It is two extra color stops, no extra element, and it stays on tokens, so it
// inverts with the theme like everything else.
//
// (See placard.go for why this is radial-gradient with an explicit `circle 3.5px` ending
// shape and not repeating-radial-gradient. The repeating form does not tile a motif; it
// paints concentric bands, and it silently fills the tile.)
//
// GAP NOTE: background-image / background-size / background-repeat /
// background-position have no typed constructors, so all four go through css.Raw.
// Gradients are the single biggest omission in the typed layer for signature devices.
const railPunchTile = "radial-gradient(circle 3.5px at 100% 50%, " +
	"var(" + TokenPaper + ") 0 62%, var(" + TokenHairline + ") 66% 96%, transparent 100%)"

var railPerforation = css.Rules(
	css.Raw("background-image", railPunchTile),
	css.Raw("background-size", "7px 11px"),
	css.Raw("background-repeat", "repeat-y"),
	css.Raw("background-position", "right top"),
)

var consoleRailBundle = clip(css.Rules(
	// Sticky, not fixed. `position: fixed` takes the rail out of flow, so the
	// content column has to be manually offset by the rail width and the two can
	// drift apart; sticky inside a grid column keeps the grid as the single source
	// of truth for the geometry. It also degrades correctly on a short viewport,
	// where a fixed rail would clip its own overflow.
	css.Position.Sticky,
	css.Raw("top", "0"),
	css.Raw("align-self", "start"),
	css.H(css.Vh(100)),
	css.Raw("overflow-y", "auto"),
	css.Bg(PaperSunk()),
	css.Raw("border-right", hairlineRule()),
	railPerforation,
	css.Display.Flex,
	css.FlexDir.Col,
	css.Gap(Space1),
	css.PaddingY(Space4),
	// Extra trailing room so a nav label never runs under the perforation strip.
	css.Raw("padding-left", string(Space3)),
	css.Raw("padding-right", "1.1rem"),

	// Narrow mode: the rail becomes a horizontally scrolling strip. It stays a
	// single element with the same children — no duplicate mobile navigation to keep
	// in sync, which is where the old design's separate mobile drawer diverged from
	// the desktop nav.
	//
	// The perforation rotates with it: a horizontal strip's torn edge is its BOTTOM
	// edge, so the disc column becomes a disc row (repeat-x, discs pinned to
	// "50% 100%"). Leaving it on the right edge would put a vertical tear mark in the
	// middle of a horizontal bar, which reads as damage rather than as a tear.
	css.Media(css.MaxW(breakRail),
		css.Position.Static,
		css.H(css.Auto),
		css.Raw("border-right", "0"),
		css.Raw("border-bottom", hairlineRule()),
		css.Raw("background-image",
			"radial-gradient(circle 3.5px at 50% 100%, var("+TokenPaper+") 0 62%, "+
				"var("+TokenHairline+") 66% 96%, transparent 100%)"),
		css.Raw("background-size", "11px 7px"),
		css.Raw("background-repeat", "repeat-x"),
		css.Raw("background-position", "left bottom"),
		css.FlexDir.Row,
		css.Items.Center,
		css.Gap(Space1),
		css.Raw("overflow-x", "auto"),
		css.Raw("overflow-y", "hidden"),
		css.Raw("white-space", "nowrap"),
		css.Raw("padding-left", string(Space3)),
		css.Raw("padding-right", string(Space3)),
		css.Raw("padding-top", string(Space2)),
		// Room under the items for the torn bottom edge.
		css.Raw("padding-bottom", "0.85rem"),
	),

	// A printed page has no navigation. The rail is pure interactive chrome, so it
	// leaves the paper entirely — see print.go for why that is a per-primitive
	// decision rather than a global `aside { display: none }`.
	css.Media(printQuery, css.Display.None),
))

// ConsoleRail is the persistent left navigation rail: a torn-edge column of manifest
// lines. Below 960px it becomes a horizontally scrolling strip with the same markup
// and the tear edge on the bottom. It is hidden in print.
//
// Compose it as RailPlate, then RailGroupLabel / RailLink+RailCode groups:
//
//	html.Aside(html.Props{Class: design.Class(design.ConsoleRail())},
//	    design.RailPlateBlock("Console", "IL-HUB"),
//	    html.Div(html.Props{Class: design.Class(design.RailGroupLabel())}, html.Text("Inbound")),
//	    html.A(html.Props{Href: "…", Class: design.Class(design.RailLink())},
//	        html.Span(html.Props{}, html.Text("Receiving")),
//	        html.Span(html.Props{Class: design.Class(design.RailCode())}, html.Text("RCV")),
//	    ),
//	)
func ConsoleRail() []css.Rule { return consoleRailBundle }

var railPlateBundle = clip(css.Rules(
	css.Display.Flex,
	css.FlexDir.Col,
	css.Gap(css.Length("2px")),
	css.MinWidth(css.Zero),
	css.PaddingX(Space2),
	css.Raw("padding-bottom", string(Space3)),
	css.Raw("margin-bottom", string(Space2)),
	// A hairline, NOT the 2px manifest rule. The plate is the rail's head and it
	// wants a baseline, but a second heavy ink rule on every screen would halve the
	// value of the one under the page title — two loud marks in one view cancel and
	// you are back at uniform weight.
	css.Raw("border-bottom", hairlineRule()),
	css.Raw("flex", "0 0 auto"),

	// In the horizontal strip the plate becomes the leading item, so its rule turns
	// into a trailing hairline and it stops eating vertical space.
	css.Media(css.MaxW(breakRail),
		css.Raw("border-bottom", "0"),
		css.Raw("border-right", hairlineRule()),
		css.Raw("padding-bottom", "0"),
		css.Raw("margin-bottom", "0"),
		css.Raw("padding-right", string(Space3)),
		css.Raw("margin-right", string(Space1)),
	),
))

// RailPlate is the rail's head block: which console this is and which hub it is
// speaking for. Pair [Eyebrow] with a [Data] code inside it, or call
// [RailPlateBlock] for the assembled shape.
//
// It carries the rail's only structural mark besides the perforation, and that mark
// is a hairline on purpose. See the bundle for the argument against a 2px rule here.
func RailPlate() []css.Rule { return railPlateBundle }

var railPlateCodeBundle = clip(css.Rules(
	Data(StepFine),
	css.FontWeight.Bold,
	css.TextColor(Ink()),
	css.Raw("white-space", "nowrap"),
	css.Tracking(css.Ems(0.02)),
))

// RailPlateCode is the hub code on the rail plate: mono, ink, slightly tracked. It is
// the loudest thing in the rail and it is still only 13px — the rail's job is to be
// unmistakably Atlas, not to be seen first.
func RailPlateCode() []css.Rule { return railPlateCodeBundle }

var railCodeBundle = clip(css.Rules(
	Data(StepMicro),
	css.TextColor(Graphite()),
	css.FontWeight.Semibold,
	css.Raw("white-space", "nowrap"),
	css.Raw("flex", "0 0 auto"),
	// Tabular + no ligatures for the same reason NumericCell has them: these are
	// codes, and a ligature or a proportional digit inside a code hides a character.
	css.FontVariantNumeric.TabularNums,
	css.Raw("font-variant-ligatures", "none"),
	// In the horizontal strip the codes sit AFTER each label on one line, which
	// roughly doubles the strip's width for information the label already carries.
	// Same call as RailGroupLabel: a mark that stops doing its job in a layout is
	// removed in that layout rather than kept for consistency.
	css.Media(css.MaxW(breakRail), css.Display.None),
))

// RailCode is the mono route code at the trailing edge of a rail item ("RCV", "PO",
// "INV", "XFER"). Hidden in the narrow horizontal strip.
//
// This is the mark that makes the rail read as a manifest column rather than as a
// generic nav: RailLink is already justify-between, so a label plus a RailCode puts
// every code in the same trailing column, and a column of fixed-advance-width codes
// aligns. It is the same argument the package makes for putting SKUs in mono, applied
// to the frame.
func RailCode() []css.Rule { return railCodeBundle }

// RailPlateBlock assembles the rail head: an eyebrow naming the console and a mono
// hub code under it.
//
//	design.RailPlateBlock("Operations console", "IL-HUB")
//
// It is a component for the same reason LanePlacard is: the two-line eyebrow/code
// pairing is the thing that makes the rail read as a placard column, and three call
// sites reassembling it by hand is three chances to ship a rail with a bold label and
// no code, which is just a nav header again. Both strings are caller-supplied — the
// design system does not know which hub you are standing in.
func RailPlateBlock(parseName string, parseHubCode string) ui.Node {
	parseChildren := make([]ui.Node, 0, 2)
	if strings.TrimSpace(parseName) != "" {
		parseChildren = append(parseChildren,
			html.Span(html.Props{Class: Class(eyebrowBundle)}, html.Text(parseName)))
	}
	if strings.TrimSpace(parseHubCode) != "" {
		parseChildren = append(parseChildren,
			html.Span(html.Props{Class: Class(railPlateCodeBundle)}, html.Text(parseHubCode)))
	}
	return html.Div(html.Props{Class: Class(railPlateBundle)}, parseChildren...)
}

var railGroupLabelBundle = clip(css.Rules(
	Display(StepMicro),
	css.TextColor(Graphite()),
	css.PaddingX(Space2),
	css.Raw("padding-top", string(Space3)),
	css.Raw("padding-bottom", string(Space1)),
	// In narrow mode the rail is horizontal, and a group label in a horizontal strip
	// is noise — the groups are no longer stacked, so the label labels nothing.
	css.Media(css.MaxW(breakRail), css.Display.None),
))

// RailGroupLabel is a section label inside the rail ("INVENTORY", "INBOUND",
// "SETTINGS"). It hides itself in narrow mode, where grouping no longer reads.
func RailGroupLabel() []css.Rule { return railGroupLabelBundle }

var railLinkBase = css.Rules(
	css.Display.Flex,
	css.Items.Center,
	css.Justify.Between,
	css.Gap(Space2),
	css.PaddingX(Space2),
	css.PaddingY(Space2),
	// Square-cut, not a 2px pill. This is the "something came off" half of the rail
	// rework: a rounded, filled nav item is the rounded-box vocabulary the rest of
	// this design system refuses (see surfaces.go), and eight of them stacked read as
	// a stack of chips. Square items read as ruled lines in a column, which is what
	// the perforated edge is claiming the rail is.
	css.Rounded(RadiusNone),
	// Nav labels are words a person reads, not machine facts, so Prose — even though
	// they sit next to plenty of mono. Getting this backwards is the classic
	// over-application of the mono rule: a monospace navigation reads as a terminal,
	// not as a place.
	Prose(StepFine),
	css.FontWeight.Medium,
	css.TextColor(Graphite()),
	css.Cursor.Pointer,
	css.Raw("text-decoration", "none"),
	css.Raw("flex", "0 0 auto"),
	withMotion(css.PropColors, css.Ms(120)),
	css.Hover(css.Bg(Paper()), css.TextColor(Ink())),
	css.FocusVisible(css.Outline(ManifestRuleWidth, Lane()), css.OutlineOffset(css.Px(-2))),
)

var railLinkBundle = clip(railLinkBase)

var railLinkCurrentBundle = clip(css.Rules(
	railLinkBase,
	css.Bg(Paper()),
	css.TextColor(Ink()),
	css.FontWeight.Semibold,
	// The current-page marker is an inset lane-blue bar on the leading edge, drawn
	// with an inset box-shadow rather than a border-left so it costs no layout — a
	// border would shift the label 3px relative to every other rail item.
	//
	// GAP NOTE: css.ShadowToken's own doc comment says "use the preset tokens or
	// RawShadow", but RawShadow does not exist in the package. The typed workaround
	// is a css.ShadowToken conversion, which at least keeps the value in the typed
	// position instead of falling through to css.Raw.
	css.Shadow(css.ShadowToken("inset 3px 0 0 var("+TokenLane+")")),
	// Note what is NOT here: a rule forcing the child RailCode to ink. It would have
	// to be a descendant selector to beat RailCode's own bare class, and a blunt
	// `& *` in a design system is how a bundle silently recolors something a caller
	// nested inside it three months later. The lane bar plus the semibold label
	// already answer "am I here"; the code stays graphite in every state, which makes
	// the code column uniform and therefore scannable.
	// In the horizontal strip an inset LEFT bar reads as an arbitrary tick, so the
	// marker moves to the bottom edge, where it reads as a tab.
	css.Media(css.MaxW(breakRail),
		css.Shadow(css.ShadowToken("inset 0 -3px 0 var("+TokenLane+")")),
	),
))

// RailLink and RailLinkCurrent are the rail's navigation items. Exactly two states:
// there is no hover-only "active-ish" third style, because the only question a rail
// item answers is "am I here or not".
//
// Pair RailLinkCurrent with aria-current="page" on the element — the inset bar is a
// visual marker and carries no semantics on its own.
func RailLink() []css.Rule        { return railLinkBundle }
func RailLinkCurrent() []css.Rule { return railLinkCurrentBundle }

var contentColumnBundle = clip(css.Rules(
	// min-width:0 is load-bearing: without it a wide mono string or an unwrapped
	// table sets this grid column's min-content width, the column refuses to shrink,
	// and the whole page gets a horizontal scrollbar that appears to come from
	// nowhere. minmax(0,1fr) on the shell plus min-width:0 here is the pair that
	// makes a grid layout actually shrinkable.
	css.MinWidth(css.Zero),
	css.Display.Flex,
	css.FlexDir.Col,
	css.Gap(Space5),
	css.PaddingX(Space6),
	css.PaddingY(Space5),
	// No max-width. The content column fills the available width, because Atlas's
	// content is tables and a centered strip with dead space beside it is both the
	// generic dashboard look and actively worse for a 14-column manifest. Readable
	// line length is handled where it belongs — on the prose, via Measure.
	css.Media(css.MaxW(breakNarrow),
		css.PaddingX(Space3),
		css.PaddingY(Space4),
		css.Gap(Space4),
	),
	// On paper the page box (@page margin: 14mm) already supplies the gutter, so the
	// column's own 32px would be a second margin inside the first — the printed
	// document would sit in a narrow strip and a wide manifest would fit even less.
	css.Media(printQuery,
		css.PaddingX(css.Zero),
		css.PaddingY(css.Zero),
		css.Gap(Space4),
	),
))

// ContentColumn is the console's content area: full-width, uncapped, with gutters
// that tighten on narrow viewports. Cap paragraph width with Measure, not this.
func ContentColumn() []css.Rule { return contentColumnBundle }

// --- storefront ---------------------------------------------------------------

var storefrontHeaderBundle = clip(css.Rules(
	css.Position.Sticky,
	css.Raw("top", "0"),
	css.Raw("z-index", "20"),
	css.Bg(Paper()),
	css.Raw("border-bottom", hairlineRule()),
	// No backdrop-filter, no translucency. A blurred translucent header over a
	// scrolling table is expensive to composite and makes the sticky table header
	// underneath it unreadable at the moment it matters most.
	//
	// PRINT: a sticky header prints once, at the top, which is fine — but position:sticky
	// inside a print fragmentation context is under-specified and some engines paint it
	// over the first rows of every page. Pinning it static costs nothing and removes the
	// variability.
	css.Media(printQuery,
		css.Position.Static,
		css.Raw("border-bottom", manifestRule()),
	),
))

// StorefrontHeader is the public shell's sticky header bar. Pair it with
// StorefrontHeaderInner for the width-constrained row inside.
func StorefrontHeader() []css.Rule { return storefrontHeaderBundle }

var storefrontHeaderInnerBundle = clip(css.Rules(
	Cluster(Space4),
	css.Justify.Between,
	css.MaxWidth(css.Px(1240)),
	css.MarginX(css.Auto),
	css.PaddingX(Space6),
	css.PaddingY(Space3),
	css.W(css.Full),
	css.Media(css.MaxW(breakNarrow), css.PaddingX(Space3), css.PaddingY(Space2)),
))

// StorefrontHeaderInner is the centered row inside StorefrontHeader. Unlike the
// console's content column this one IS capped: the storefront is a document, and a
// document's header should align with the document.
func StorefrontHeaderInner() []css.Rule { return storefrontHeaderInnerBundle }

var storefrontMainBundle = clip(css.Rules(
	css.MaxWidth(css.Px(1240)),
	css.MarginX(css.Auto),
	css.W(css.Full),
	css.Display.Flex,
	css.FlexDir.Col,
	css.Gap(Space7),
	css.PaddingX(Space6),
	css.PaddingY(Space6),
	css.MinWidth(css.Zero),
	css.Media(css.MaxW(breakNarrow),
		css.PaddingX(Space3),
		css.PaddingY(Space5),
		css.Gap(Space5),
	),
	// Same argument as ContentColumn: @page owns the paper's margin. The 1240px cap also
	// goes, because on paper the medium already caps the measure.
	css.Media(printQuery,
		css.PaddingX(css.Zero),
		css.PaddingY(css.Zero),
		css.MaxWidth(css.RawLength("none")), // max-width takes `none`, not `auto`
		css.Gap(Space5),
	),
))

// StorefrontMain is the public content column: capped, centered, with generous
// between-section spacing (Space7, versus the console's Space5 — the storefront is
// read, the console is scanned).
func StorefrontMain() []css.Rule { return storefrontMainBundle }

// --- page head ----------------------------------------------------------------

var pageHeadBundle = clip(css.Rules(
	css.Display.Flex,
	css.FlexDir.Col,
	css.Gap(Space1),
	css.Raw("padding-bottom", string(Space3)),
	css.Raw("border-bottom", manifestRule()),
	css.MinWidth(css.Zero),
))

// PageHead is the title block at the top of a route: eyebrow, title, and the heavy
// ink rule that closes it.
//
// The rule is what makes the page start. It is the one heavy mark most Atlas pages
// get, and pairing it with PageTitle means a route cannot accidentally ship a title
// that floats without a baseline.
func PageHead() []css.Rule { return pageHeadBundle }

var pageTitleBundle = clip(css.Rules(
	Display(StepHead),
	css.TextColor(Ink()),
	// text-wrap:balance keeps a two-line condensed title from leaving one orphan
	// word, which at this weight and tracking is very visible.
	css.Raw("text-wrap", "balance"),
))

// PageTitle is the route title: condensed, heavy, uppercase, fluid between 24px and
// 30px so it survives a 380px viewport without a media query.
func PageTitle() []css.Rule { return pageTitleBundle }

var sectionTitleBundle = clip(css.Rules(
	Display(StepSubhead),
	css.TextColor(Ink()),
))

// SectionTitle is a heading inside a page region. There is no third heading level:
// if a page needs one, it needs to be two pages or two Surfaces.
func SectionTitle() []css.Rule { return sectionTitleBundle }
