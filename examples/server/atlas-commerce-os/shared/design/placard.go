package design

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/css"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// THE LANE PLACARD — Atlas's signature element.
//
// Everything else in this design system is deliberately quiet: flat paper, hairlines,
// graphite meta, one heavy rule per page. All of the saved boldness is spent here, in
// one device, because a design system needs exactly one memorable thing and it should
// be the thing that carries the product's actual thesis.
//
// Atlas's thesis is warehouse-aware availability: not "is this in stock" but "can this
// be where you need it, when you need it". That question is a route, a date and a
// posture — origin hub, destination hub, promise date, and whether the lane is moving.
// The placard IS that sentence, set as freight paperwork: mono codes on an inked bar
// with a punched hole and a perforated tear edge.
//
// It appears on product availability, transfers, purchase orders and receiving, which
// is what makes it a signature rather than an ornament: the same object shows the same
// four facts everywhere, so an operator learns to read it once.
//
// # Why this is the ONE bold element
//
// The failure being fixed was uniform weight — four nested boxes all shouting equally,
// which reads as nothing shouting. Hierarchy is a budget. Spending it on one device
// that encodes a real domain fact buys a page that can be scanned in one glance:
// placard first, then the heavy title rule, then tables, then meta. Spending it on
// three ornamental accents buys nothing, which is where Atlas started.

// Posture is the state of a freight lane. It is domain vocabulary — the words an
// operator uses — and it maps onto the visual StatusTone semantics via Tone(). Keeping
// those two enums separate is the useful bit: the domain can grow new postures without
// the design system growing new hues.
type Posture int

const (
	// PostureOnLane — moving, expected to hit the promise date.
	PostureOnLane Posture = iota
	// PostureHeld — stopped at a hub. Not yet a failure, but a human should know.
	PostureHeld
	// PostureShort — the quantity will not be met. An exception.
	PostureShort
	// PostureClosed — delivered and reconciled. Closed clean.
	PostureClosed
)

// Label is the placard's printed posture text: short, uppercase, operator vocabulary.
func (parsePosture Posture) Label() string {
	switch parsePosture {
	case PostureHeld:
		return "HELD"
	case PostureShort:
		return "SHORT"
	case PostureClosed:
		return "CLOSED"
	default:
		return "ON LANE"
	}
}

// Tone maps a domain posture onto the four visual status semantics. This function is
// the entire bridge between "what the warehouse says" and "what color the screen
// uses", which is why it is one switch in one place.
func (parsePosture Posture) Tone() StatusTone {
	switch parsePosture {
	case PostureHeld:
		return TonePending
	case PostureShort:
		return ToneException
	case PostureClosed:
		return ToneVerified
	default:
		return TonePending
	}
}

// --- the bar ------------------------------------------------------------------

var placardBarBundle = clip(css.Rules(
	css.Position.Relative,
	Cluster(Space5),
	css.Justify.Between,
	css.Bg(PlacardBg()),
	css.TextColor(PlacardFg()),
	css.Rounded(RadiusTag),
	css.PaddingY(Space3),
	css.Raw("padding-right", string(Space4)),
	// Leading room for the punched hole. Dock tags have a hole; the hole is what makes
	// this read as a physical tag rather than as a dark banner.
	css.Raw("padding-left", "2.5rem"),
	// The perforation strip is a pseudo-element bled 1px past the bottom edge, so
	// overflow must clip to the 2px radius.
	css.Raw("overflow", "hidden"),
	// Extra bottom room so the perforation does not sit under the text.
	css.Raw("padding-bottom", "1.25rem"),

	// The placard is the one surface where the global lane-blue focus ring fails.
	// Lane on placard-bg measures 2.11:1 in light mode (dark blue on an inked bar),
	// so a link or button composed into a placard body would be focusable with an
	// effectively invisible ring — and only in light mode, which is exactly the kind
	// of bug that ships. Inside the bar the ring switches to placard-fg (16:1 light,
	// 13:1 dark) and moves INSET, because an outset ring on a full-bleed bar would
	// land half on the paper outside it, where near-paper is invisible.
	//
	// Scoped as a descendant rule so it covers anything a caller nests here,
	// including elements this design system has never seen. LanePlacard itself
	// renders nothing focusable; PlacardBar is the documented composition path, and
	// this is the cost of documenting one.
	css.Descendant(css.Sel(":focus-visible"),
		css.Outline(ManifestRuleWidth, PlacardFg()),
		css.OutlineOffset(css.Px(-2)),
	),

	// The punched hole: a paper-colored disc on the leading edge.
	//
	// Paper-colored, not transparent, because a real cut-out would need the parent's
	// background and the placard may sit on Paper or inside SurfaceFlush. Referencing
	// the token means the hole follows the theme; it is the one reason the docs say to
	// place a placard directly on Paper.
	css.Before(
		css.Position.Absolute,
		css.Raw("left", "0.9rem"),
		css.Raw("top", "0.95rem"),
		css.W(css.Px(11)),
		css.H(css.Px(11)),
		css.Rounded(css.Percent(50)),
		css.Bg(Paper()),
		css.Raw("pointer-events", "none"),
	),

	// PRINT: the bar becomes a stamped outline.
	//
	// Two reasons, and the second one is why this is not merely damage control:
	//
	//  1. Mechanical. Browsers do not print background-color, so an inked bar prints as
	//     placard-fg text on nothing — a blank region where the signature element was.
	//     print.go already flips placard-bg to white and placard-fg to black; this
	//     block supplies the keyline that has to replace the fill, or the placard prints
	//     as four floating codes with no object around them.
	//  2. Design. A real dock tag is not a solid black rectangle. It is a stiff card
	//     carrying printed codes, a punched hole and a perforated stub — an OUTLINE. The
	//     screen renders it inverted because a screen has no card stock to imply and
	//     paper does. So this is not a degraded placard; it is the placard's original
	//     form, and the 2px keyline is the card edge.
	//
	// A solid black bar would also be the largest toner cost on a page that gets printed
	// by the hundred, which is a real objection in a warehouse.
	css.Media(printQuery, css.Rules(
		css.Border(ManifestRuleWidth, PlacardFg()),
		// The punch loses its point once the bar is white-on-white — a white disc on
		// white paper is nothing — so it is dropped and the leading indent comes back to
		// a normal gutter. The perforation is handled separately (see below), because it
		// lives on ::after and has to become a line rather than disappear.
		css.Raw("padding-left", string(Space4)),
		css.Raw("padding-bottom", string(Space3)),
		css.Before(css.Display.None),
	)...),
))

// The perforated tear edge.
//
// Paper-colored half-discs tiled along the bottom edge bite scallops out of the ink — the
// torn stub of a dock tag. It is a background image on a 7px strip, so it costs no extra
// DOM and no image asset, and because the disc color is var(--atlas-paper) it inverts with
// the theme for free.
//
// BUG FIXED HERE (this device had never actually rendered). The original was:
//
//	repeating-radial-gradient(circle at 50% 100%, var(--atlas-paper) 0 3.5px, transparent 3.5px)
//
// A repeating-radial-gradient repeats its whole stop list outward as concentric bands, so
// "paper 0->3.5px, transparent at 3.5px" produces paper 0-3.5px, then the list restarts
// and paints paper 3.5-7px, and so on forever. The transparent stop is zero-width and the
// tile comes out SOLID PAPER — a 7px paper-colored strip along the bottom of the bar,
// which against a paper page is indistinguishable from the bar simply being 7px shorter.
// The placard has been shipping with a flat bottom edge and dead CSS behind it, and it
// looked fine in review because a flat dark bar looks deliberate.
//
// The fix is a NON-repeating radial-gradient with an explicit ending shape — `circle
// 3.5px` — so the stops are relative to that radius and the tile holds exactly one disc.
// Tiling is then background-size's job, which is what it was for. Lesson worth keeping:
// `repeating-*-gradient` is for concentric/striped bands, never for tiling a motif;
// tile a plain gradient with background-size instead.
//
// The 96%/100% stop pair rather than a hard 100% stop is deliberate too: a hard stop on a
// curve aliases into visible stair-steps at 1x, and 4% of 3.5px is a sub-pixel feather.
//
// GAP NOTE: background-image, background-size, background-repeat, and inset properties
// (top/left/right/bottom) have no typed constructors in the css package, so the whole
// perforation is css.Raw. Gradients are the biggest single omission — they are how most
// signature visual devices get built.
// In PRINT the perforation has to change technique rather than change value. The discs
// are painted in var(--atlas-paper), which print.go rewrites to white — so on paper the
// tear edge would be white half-circles biting into a white bar, i.e. nothing at all,
// and the placard would print as a plain box with an unexplained 7px gap at the bottom.
//
// A printed tear line is a dotted rule. That is what a perforation looks like once it is
// printed rather than torn, so the ::after strip drops its gradient and becomes a 2px
// dotted line in the placard's own foreground color, inset to the card edge. Same device,
// same meaning, drawn the way a press would draw it.
// Note the shape: css.After takes ...Rule, so a media query cannot be nested INSIDE
// it. The print variant is therefore a second, sibling After() wrapped in the query.
// Within one folded class blocks are emitted sorted by (at-rule, selector), and ""
// sorts before "@media print", so the print block lands last and wins on order at
// equal specificity — the same mechanism withMotion relies on for reduced motion.
var placardPerforation = css.Rules(
	css.After(
		css.Position.Absolute,
		css.Raw("left", "0"),
		css.Raw("right", "0"),
		css.Raw("bottom", "-1px"),
		css.H(css.Px(7)),
		css.Raw("background-image",
			"radial-gradient(circle 3.5px at 50% 100%, var("+TokenPaper+") 0 96%, transparent 100%)"),
		css.Raw("background-size", "11px 7px"),
		css.Raw("background-repeat", "repeat-x"),
		css.Raw("pointer-events", "none"),
	),
	css.Media(printQuery, css.After(
		css.Raw("background-image", "none"),
		css.H(css.Zero),
		css.Raw("bottom", "0.5rem"),
		css.Raw("left", "0.75rem"),
		css.Raw("right", "0.75rem"),
		css.Raw("border-top", string(ManifestRuleWidth)+" dotted var("+TokenPlacardFg+")"),
	)...),
)

var placardFullBundle = clip(css.Rules(
	placardBarBundle,
	placardPerforation,
))

// PlacardBar is the inked dock-tag bar with the punched hole and the perforated tear
// edge, for callers composing their own placard body. LanePlacard is the shape you
// almost always want.
//
// Place it directly on Paper — the punch and the perforation are painted in the paper
// token, so on a Recess background they will read a shade off.
func PlacardBar() []css.Rule { return placardFullBundle }

// --- internals ----------------------------------------------------------------

var placardRouteBundle = clip(css.Rules(
	Cluster(Space3),
	css.Raw("flex-wrap", "nowrap"),
))

var placardHubBundle = clip(css.Rules(
	Data(StepLede),
	css.FontWeight.Bold,
	css.Raw("white-space", "nowrap"),
	css.Tracking(css.Ems(0.02)),
))

var placardArrowBundle = clip(css.Rules(
	Data(StepLede),
	// The arrow is graphic, not informational — the accessible label spells out
	// "to" — so it is dimmed rather than given equal weight to the hub codes.
	css.OpacityNum(css.Num(0.55)),
))

var placardFieldBundle = clip(css.Rules(
	css.Display.Flex,
	css.FlexDir.Col,
	css.Gap(css.Length("2px")),
	css.MinWidth(css.Zero),
))

var placardFieldLabelBundle = clip(css.Rules(
	Display(StepMicro),
	css.OpacityNum(css.Num(0.7)),
))

var placardFieldValueBundle = clip(css.Rules(
	Data(StepFine),
	css.FontWeight.Semibold,
	css.Raw("white-space", "nowrap"),
))

// placardPostureChip is a placard-local variant of the status chip.
//
// StatusChip's outlined tones are tuned for dark text on light paper; on the inked bar
// they would be low-contrast, and ToneException's oxide fill against the ink is muddy.
// So on the placard every posture is FILLED with its tone color and labelled in paper —
// uniform, high contrast, readable on both the light and dark bar.
//
// This is the right kind of exception: a documented, contained variant for one context,
// derived from the same four tones. It is not a new palette entry, and it is not a
// free-form color.
var placardPostureBundles = buildPlacardPostures()

func buildPlacardPostures() [toneCount][]css.Rule {
	var parseOut [toneCount][]css.Rule
	for parseIndex := 0; parseIndex < toneCount; parseIndex++ {
		parseOut[parseIndex] = clip(css.Rules(
			css.Display.InlineFlex,
			css.Items.Center,
			Data(StepMicro),
			css.TextTransform.Uppercase,
			css.FontWeight.Bold,
			css.Raw("padding-left", string(Space2)),
			css.Raw("padding-right", string(Space2)),
			css.Raw("padding-top", "3px"),
			css.Raw("padding-bottom", "3px"),
			css.Rounded(RadiusTag),
			css.Raw("white-space", "nowrap"),
			css.Bg(toneColor(StatusTone(parseIndex))),
			css.TextColor(Paper()),
			// PRINT: same un-fill as StatusChip, same reason — paper-on-tone becomes
			// white-on-white once the browser drops the background. The posture is the
			// one word on the placard that says whether anything is wrong, so it must
			// not be the word that vanishes on the printed copy.
			css.Media(printQuery,
				css.Bg(css.Transparent),
				css.TextColor(toneColor(StatusTone(parseIndex))),
				css.Border(HairlineWidth, toneColor(StatusTone(parseIndex))),
			),
		))
	}
	return parseOut
}

// --- the component ------------------------------------------------------------

// PlacardSpec is the placard's content. Every string is caller-formatted: this design
// system does not know Atlas's date format, its hub-code casing or its lane-id scheme,
// and a design package that starts formatting domain values becomes impossible to
// reuse.
type PlacardSpec struct {
	// OriginHub and DestHub are hub codes, e.g. "NJ-HUB" and "IL-HUB". Short and
	// uppercase; they are set in mono at StepLede and are the loudest text on the page.
	OriginHub string
	DestHub   string

	// Promise is the promise date, already formatted, e.g. "2026-08-04". Prefer an
	// ISO-ish sortable form: it is mono and tabular, so a column of them aligns.
	Promise string

	// LaneID is the freight lane identifier, e.g. "LN-4471". Optional; the LANE field
	// is omitted when empty rather than rendered blank.
	LaneID string

	// Posture is the lane state. It drives both the printed label and the tone.
	Posture Posture

	// Label overrides the generated accessible label. Leave it empty unless the
	// generated sentence is wrong for the context.
	Label string
}

// LanePlacard renders the signature dock tag: ORIGIN → DEST, promise date, lane id and
// posture, on an inked perforated bar.
//
//	design.LanePlacard(design.PlacardSpec{
//	    OriginHub: "NJ-HUB",
//	    DestHub:   "IL-HUB",
//	    Promise:   "2026-08-04",
//	    LaneID:    "LN-4471",
//	    Posture:   design.PostureOnLane,
//	})
//
// It is provided as a component rather than as a bundle because the structure is the
// element: the hole, the perforation, the route/meta/posture ordering and the
// accessible label all have to hold together, and four call sites reassembling that by
// hand is four chances to ship a placard that reads as a dark banner.
//
// Accessibility: the bar is a role="group" carrying a spelled-out label ("Lane NJ-HUB
// to IL-HUB, promise 2026-08-04, posture ON LANE"), and the "→" glyph is aria-hidden.
// A screen reader gets the sentence; a sighted operator gets the tag.
func LanePlacard(parseSpec PlacardSpec) ui.Node {
	parseChildren := []ui.Node{
		html.Div(html.Props{Class: Class(placardRouteBundle)},
			html.Span(html.Props{Class: Class(placardHubBundle)}, html.Text(parseSpec.OriginHub)),
			html.Span(
				html.Props{Class: Class(placardArrowBundle), Aria: map[string]string{"hidden": "true"}},
				html.Text("→"),
			),
			html.Span(html.Props{Class: Class(placardHubBundle)}, html.Text(parseSpec.DestHub)),
		),
	}

	if strings.TrimSpace(parseSpec.Promise) != "" {
		parseChildren = append(parseChildren, placardField("PROMISE", parseSpec.Promise))
	}
	if strings.TrimSpace(parseSpec.LaneID) != "" {
		parseChildren = append(parseChildren, placardField("LANE", parseSpec.LaneID))
	}

	parseChildren = append(parseChildren, html.Span(
		html.Props{Class: Class(placardPostureBundles[clampTone(parseSpec.Posture.Tone())])},
		html.Text(parseSpec.Posture.Label()),
	))

	return html.Div(
		html.Props{
			Class: Class(placardFullBundle),
			Role:  "group",
			Aria:  map[string]string{"label": placardAccessibleLabel(parseSpec)},
		},
		parseChildren...,
	)
}

func placardField(parseLabel string, parseValue string) ui.Node {
	return html.Div(html.Props{Class: Class(placardFieldBundle)},
		html.Span(html.Props{Class: Class(placardFieldLabelBundle)}, html.Text(parseLabel)),
		html.Span(html.Props{Class: Class(placardFieldValueBundle)}, html.Text(parseValue)),
	)
}

// placardAccessibleLabel spells the placard out as a sentence.
//
// The visual placard is a set of codes and a glyph; read aloud in DOM order that is
// "NJ-HUB IL-HUB PROMISE 2026-08-04 LANE LN-4471 ON LANE", which is not a sentence and
// loses the direction entirely. Writing the label explicitly is the cost of having a
// graphic signature element, and it is not optional.
func placardAccessibleLabel(parseSpec PlacardSpec) string {
	if parseLabel := strings.TrimSpace(parseSpec.Label); parseLabel != "" {
		return parseLabel
	}
	var parseBuilder strings.Builder
	parseBuilder.WriteString("Lane ")
	parseBuilder.WriteString(parseSpec.OriginHub)
	parseBuilder.WriteString(" to ")
	parseBuilder.WriteString(parseSpec.DestHub)
	if parseSpec.Promise != "" {
		parseBuilder.WriteString(", promise ")
		parseBuilder.WriteString(parseSpec.Promise)
	}
	if parseSpec.LaneID != "" {
		parseBuilder.WriteString(", lane id ")
		parseBuilder.WriteString(parseSpec.LaneID)
	}
	parseBuilder.WriteString(", posture ")
	parseBuilder.WriteString(parseSpec.Posture.Label())
	return parseBuilder.String()
}
