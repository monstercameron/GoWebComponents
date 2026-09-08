package design

import "github.com/monstercameron/GoWebComponents/v6/css"

// This file is the most important file in the package, because in Atlas the type
// system IS the information architecture. See the package doc for the argument; this
// file is its implementation.
//
// Three roles, one scale. A caller picks a role (what IS this string?) and a step
// (how loud is it?) and gets size, weight, line-height, tracking, case and figure
// style as one decision. The alternative — exposing FontSize/FontWeight/Tracking
// separately — is how you end up with 40 slightly different headings, because every
// call site re-decides four properties under time pressure.

// TypeStep is a position on the type scale. It is a closed enum, not a length: a
// caller cannot invent a 19px step, which keeps the number of distinct emitted
// classes bounded (7 steps x 3 roles = 21 possible folds) and keeps the page
// looking like one document.
type TypeStep int

// The scale. Seven steps, and the names say what each is FOR rather than how big it
// is, because "small" is a comparison and "the size of a table cell" is a decision.
const (
	StepMicro   TypeStep = iota // eyebrows, table headers, chips, field labels
	StepFine                    // meta, table cells, hints — the dense default
	StepBase                    // body copy
	StepLede                    // opening paragraph, placard values, stat figures
	StepSubhead                 // section titles
	StepHead                    // page titles
	StepBanner                  // storefront hero — at most one per page
)

const stepCount = int(StepBanner) + 1

// String returns the step's identifier, for a data-* attribute, a debug dump or a
// test name. Not user-visible copy.
func (parseStep TypeStep) String() string {
	switch parseStep {
	case StepMicro:
		return "Micro"
	case StepFine:
		return "Fine"
	case StepBase:
		return "Base"
	case StepLede:
		return "Lede"
	case StepSubhead:
		return "Subhead"
	case StepHead:
		return "Head"
	case StepBanner:
		return "Banner"
	default:
		return "Unknown"
	}
}

// metrics is one row of the scale. Line-height and tracking differ per ROLE at the
// same step, which is the part a naive scale gets wrong: uppercase condensed
// display type needs positive tracking at small sizes and negative tracking at
// large ones, while mono needs almost none at any size, and prose needs a much
// looser line-height than either.
type metrics struct {
	size css.Length

	displayLineHeight css.Number
	displayTracking   css.Length

	proseLineHeight css.Number

	dataLineHeight css.Number
	dataTracking   css.Length
}

// The scale table. Sizes are in rem so the browser's font-size setting scales the
// whole system; the two largest steps are clamp()ed against the viewport so a long
// page title cannot force a horizontal scrollbar at 380px. That single clamp is what
// makes the display role responsive without a media query anywhere.
var scaleTable = [stepCount]metrics{
	StepMicro: {
		size:              "0.6875rem", // 11px
		displayLineHeight: css.Number("1.25"),
		displayTracking:   "0.1em", // uppercase at 11px is unreadable without air
		proseLineHeight:   css.Number("1.45"),
		dataLineHeight:    css.Number("1.3"),
		dataTracking:      "0.06em",
	},
	StepFine: {
		size:              "0.8125rem", // 13px — the density Atlas actually runs at
		displayLineHeight: css.Number("1.3"),
		displayTracking:   "0.07em",
		proseLineHeight:   css.Number("1.5"),
		dataLineHeight:    css.Number("1.35"),
		dataTracking:      "0.02em",
	},
	StepBase: {
		size:              "0.9375rem", // 15px
		displayLineHeight: css.Number("1.25"),
		displayTracking:   "0.05em",
		proseLineHeight:   css.Number("1.6"),
		dataLineHeight:    css.Number("1.4"),
		dataTracking:      "0.01em",
	},
	StepLede: {
		size:              "1.0625rem", // 17px
		displayLineHeight: css.Number("1.2"),
		displayTracking:   "0.03em",
		proseLineHeight:   css.Number("1.55"),
		dataLineHeight:    css.Number("1.35"),
		dataTracking:      "0.01em",
	},
	StepSubhead: {
		size:              "1.3125rem", // 21px
		displayLineHeight: css.Number("1.12"),
		displayTracking:   "0.01em",
		proseLineHeight:   css.Number("1.35"),
		dataLineHeight:    css.Number("1.25"),
		dataTracking:      "0",
	},
	StepHead: {
		// Fluid: 24px floor, 30px ceiling. Below ~380px the floor still fits.
		size:              "clamp(1.5rem,4.6vw,1.875rem)",
		displayLineHeight: css.Number("1.02"),
		displayTracking:   "-0.01em", // condensed heavy type tightens as it grows
		proseLineHeight:   css.Number("1.2"),
		dataLineHeight:    css.Number("1.15"),
		dataTracking:      "-0.005em",
	},
	StepBanner: {
		size:              "clamp(2rem,7vw,2.75rem)",
		displayLineHeight: css.Number("0.96"), // headline leading can go sub-1
		displayTracking:   "-0.02em",
		proseLineHeight:   css.Number("1.1"),
		dataLineHeight:    css.Number("1.05"),
		dataTracking:      "-0.01em",
	},
}

// stepMetrics looks up a step, clamping out-of-range values to StepBase. A caller
// can construct TypeStep(99) — Go enums are not closed at the type level — and the
// honest failure mode for that is "renders as body text", not "renders at 0px".
func stepMetrics(parseStep TypeStep) metrics {
	if parseStep < 0 || int(parseStep) >= stepCount {
		return scaleTable[StepBase]
	}
	return scaleTable[parseStep]
}

// --- the three roles ----------------------------------------------------------
//
// The 21 (role x step) bundles are assembled once per process into these tables.
// They are NOT emitted at init — emission happens when Class folds them — which is
// what keeps the package correct across css.Reset() in tests: an init-time fold
// would hand out class names whose CSS a Reset had already thrown away.
//
// These are var initializers rather than an init() func on purpose. Go runs every
// init() AFTER all package-level variable initialization, so a bundle further down
// this file that composes Display(StepMicro) would have captured an empty slice.
// Var initializers get dependency-ordered instead, transitively through the
// functions they call.
var (
	displayBundles = buildRoleBundles(displayRules)
	proseBundles   = buildRoleBundles(proseRules)
	dataBundles    = buildRoleBundles(dataRules)
)

func buildRoleBundles(parseBuild func(metrics) []css.Rule) [stepCount][]css.Rule {
	var parseOut [stepCount][]css.Rule
	for parseIndex := 0; parseIndex < stepCount; parseIndex++ {
		parseOut[parseIndex] = clip(parseBuild(scaleTable[parseIndex]))
	}
	return parseOut
}

func displayRules(parseM metrics) []css.Rule {
	return css.Rules(
		fontFamily(TokenFontDisplay),
		css.FontSize(parseM.size),
		css.FontWeight.Bold,
		css.LineHeight(parseM.displayLineHeight),
		css.Tracking(parseM.displayTracking),
		css.TextTransform.Uppercase,
	)
}

func proseRules(parseM metrics) []css.Rule {
	return css.Rules(
		fontFamily(TokenFontBody),
		css.FontSize(parseM.size),
		css.FontWeight.Normal,
		css.LineHeight(parseM.proseLineHeight),
		css.Tracking(css.Zero),
		css.TextTransform.None,
	)
}

func dataRules(parseM metrics) []css.Rule {
	return css.Rules(
		fontFamily(TokenFontData),
		css.FontSize(parseM.size),
		// Medium, not normal: mono faces render optically lighter than the
		// proportional body face at the same size, and a SKU that looks fainter than
		// the sentence around it reads as less certain than the sentence.
		css.FontWeight.Medium,
		css.LineHeight(parseM.dataLineHeight),
		css.Tracking(parseM.dataTracking),
		css.TextTransform.None,
		// Tabular figures are the reason mono is here at all: a column of quantities
		// has to align, and "1,041" under "1,042" has to differ in exactly one
		// glyph slot.
		css.FontVariantNumeric.TabularNums,
	)
}

// Display is the condensed, heavy, uppercase role: page titles, section titles,
// eyebrows, button labels, table headers. It NAMES a region of the page.
//
// Use it for labels, never for content. A sentence set in condensed uppercase is
// unreadable past about six words, which is a useful natural limit: if the string
// does not fit the role, the string is prose.
//
// (The name shadows nothing in this package. It is unrelated to css.Display, the
// display property.)
func Display(parseStep TypeStep) []css.Rule {
	return displayBundles[clampStep(parseStep)]
}

// Prose is the proportional role: every sentence a human wrote to be read.
//
// The brief calls this role "Body". It is exported as Prose because design.Body()
// reads like the <body> element, and because "prose" names the thing being styled —
// running text — which is the test a caller should apply: if it is a sentence, it is
// Prose; if it is a machine fact, it is Data.
func Prose(parseStep TypeStep) []css.Rule {
	return proseBundles[clampStep(parseStep)]
}

// Data is the monospace role, and it is the load-bearing rule of this design system:
// EVERY machine fact is Data. SKU, hub code, lane id, promise date, ETA, quantity,
// price, status code, order number, any identifier.
//
// Mono here is not a stylistic tic, it is three functional guarantees:
//
//   - Fixed advance width, so a column of SKUs or quantities aligns even without a
//     table wrapper, and a wrapping id breaks at a predictable place.
//   - Combined with tabular figures, one changed digit occupies exactly one glyph
//     slot, so a transposition (1042 vs 1024) is visible rather than plausible.
//   - It marks provenance. A mono string is something the system knows; a
//     proportional string is something a person said. That distinction is the
//     difference between "SHORT 4 UNITS" as a computed discrepancy and as a note
//     somebody typed, and in a warehouse app it is the distinction that matters.
//
// If you are reaching for Data on a sentence, or Prose on an identifier, the
// hierarchy will read as noise. There is no third option for either.
func Data(parseStep TypeStep) []css.Rule {
	return dataBundles[clampStep(parseStep)]
}

func clampStep(parseStep TypeStep) int {
	if parseStep < 0 || int(parseStep) >= stepCount {
		return int(StepBase)
	}
	return int(parseStep)
}

// --- derived type primitives --------------------------------------------------

// Note the composition idiom: css.Rules takes ...any and unpacks a []css.Rule
// argument, so a whole bundle is passed as ONE argument (not spread). That is how
// every composite primitive in this package builds on another.
var eyebrowBundle = clip(css.Rules(
	Display(StepMicro),
	css.TextColor(Graphite()),
	css.Display.Block,
))

// Eyebrow is the small uppercase label above a title: "RECEIVING", "PROMISE DATE",
// "LANE 4471". It is graphite, never a hue.
//
// The previous design used an orange eyebrow, a teal badge and a yellow button on
// the same screen — three saturated accents with no relationship, which is how a
// page ends up with three competing focal points and therefore none. An eyebrow's
// job is to be legible and quiet; if it needs a color, what it actually needs is to
// be a StatusChip.
func Eyebrow() []css.Rule { return eyebrowBundle }

var measureBundle = clip(css.Rules(
	css.MaxWidth(css.RawLength("68ch")),
))

// Measure caps a text block at ~68 characters, the readable line length.
//
// It exists so the content column can be uncapped. A max-width on the column would
// make every page a centered strip of content with dead space beside it — the
// generic dashboard look — while an uncapped column with Measure on its prose gets
// full-width tables AND readable paragraphs. Apply it to paragraphs, not to
// containers.
//
// The `ch` unit has no typed constructor, so this goes through css.RawLength — which
// is the typed escape hatch, not css.Raw, so the value still flows into MaxWidth as
// a Length.
func Measure() []css.Rule { return measureBundle }
