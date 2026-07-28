package design

import "github.com/monstercameron/GoWebComponents/v5/css"

// Status — the one place saturated color is allowed, and the API is shaped so it
// cannot be used anywhere else.
//
// # Why the enum, and not a color argument
//
// The old design had three saturated hues on screen at once — an orange eyebrow, a
// teal badge, a yellow button — chosen independently at three call sites. None of
// them meant anything, so the reader learned nothing from color, and the one thing
// color is uniquely good at (making an exception impossible to miss in a list of 40
// rows) was spent on decoration.
//
// The mechanism that prevents a repeat is the type signature. StatusChip takes a
// StatusTone, not a css.Color. There is no way to ask this design system for "a
// green chip"; you can only ask for a chip that says the thing was VERIFIED. If your
// state does not map to one of the four tones, it is not a status, and it should be
// Graphite text.
//
// # Only exceptions are filled
//
// The four tones do not have four equally loud treatments. Neutral, Pending and
// Verified are outlined — a hairline in the tone with matching text on plain paper.
// ToneException is FILLED — solid oxide with paper text.
//
// That asymmetry is the hierarchy. In a 40-row queue, the four exceptions are solid
// blocks and the other 36 rows are quiet outlines, so the eye lands on the work
// without reading a word. Four equally saturated pills would be the old design again
// with better colors.
//
// # Where "color is never the only channel" gets cashed
//
// StatusChip's doc has always said the label carries the meaning and the hue is a
// second channel. Print is where that stops being a nice sentiment: browsers do not
// print background-color, so the filled exception chip would land white-on-white on
// paper. Because the label already carries the meaning, the print override can drop the
// fill and re-emphasize with border WEIGHT instead, and lose nothing. A design system
// that had let hue be the only channel would have had no move available here.

// StatusTone is the closed set of states Atlas colors. Four, deliberately: a fifth
// tone means a fifth hue, and the point of a semantic palette is that a reader can
// learn it in one screen.
type StatusTone int

const (
	// ToneNeutral — no claim. A draft, an unstarted step, an informational tag.
	ToneNeutral StatusTone = iota
	// TonePending — in the system, awaited. In transit, queued, submitted, on order.
	// Lane blue, because "the system is handling it" is structural, not alarming.
	TonePending
	// ToneVerified — checked and passed. Received clean, approved, reconciled, closed.
	ToneVerified
	// ToneException — needs a human. Discrepancy, short shipment, failed check,
	// overdue promise, negative on-hand. The only filled tone.
	ToneException
)

const toneCount = int(ToneException) + 1

// String returns the tone's lowercase identifier. It is intended for a data-*
// attribute or a test, NOT as user-visible copy: the label a user reads is domain
// vocabulary ("SHORT 4", "RECEIVED CLEAN"), which the caller supplies. A design
// system should not be inventing the words on your screen.
func (parseTone StatusTone) String() string {
	switch parseTone {
	case TonePending:
		return "pending"
	case ToneVerified:
		return "verified"
	case ToneException:
		return "exception"
	default:
		return "neutral"
	}
}

// toneColor is the internal tone -> token mapping. This is the ONLY function in the
// package that turns a tone into a hue, which is what makes the semantic contract
// enforceable by reading one function.
func toneColor(parseTone StatusTone) css.Color {
	switch parseTone {
	case TonePending:
		return Lane()
	case ToneVerified:
		return StatusVerified()
	case ToneException:
		return StatusException()
	default:
		return Graphite()
	}
}

var statusChipBase = css.Rules(
	css.Display.InlineFlex,
	css.Items.Center,
	css.Gap(Space1),
	// A chip is a machine-reported state, so it is Data: mono, uppercase-tracked,
	// micro. It reads as something the system printed, not something someone wrote.
	Data(StepMicro),
	css.TextTransform.Uppercase,
	css.FontWeight.Semibold,
	css.Raw("padding-left", string(Space2)),
	css.Raw("padding-right", string(Space2)),
	css.Raw("padding-top", "2px"),
	css.Raw("padding-bottom", "2px"),
	css.Rounded(RadiusTag),
	css.Raw("white-space", "nowrap"),
	css.Border(HairlineWidth, css.CurrentCo),
)

var statusChipBundles = buildStatusChips()

func buildStatusChips() [toneCount][]css.Rule {
	var parseOut [toneCount][]css.Rule
	for parseIndex := 0; parseIndex < toneCount; parseIndex++ {
		parseTone := StatusTone(parseIndex)
		if parseTone == ToneException {
			// Filled. Paper-on-oxide, not oxide-on-wash: a tinted wash would need an
			// alpha blend against whatever surface it lands on (paper, paper-sunk on
			// odd rows, the ink placard), and its contrast would differ in each one.
			// Solid ink-on-color has one contrast ratio everywhere.
			parseOut[parseIndex] = clip(css.Rules(
				statusChipBase,
				css.Bg(toneColor(parseTone)),
				css.TextColor(Paper()),
				css.Border(HairlineWidth, toneColor(parseTone)),
				// PRINT: un-fill. Browsers do not print background-color by default, so
				// on paper this chip would keep its paper-colored TEXT and lose its oxide
				// FILL — white on white. The single element this design system exists to
				// make unmissable would be the only one that disappears, and only on
				// paper, where nobody looks during review.
				//
				// It re-forms as the loudest OUTLINE the system can draw: the 2px
				// manifest rule in oxide with oxide text. Weight, not fill, carries the
				// emphasis — which works on a monochrome laser too, where oxide and
				// verify dither to similar greys but 2px and 1px do not.
				css.Media(printQuery,
					css.Bg(css.Transparent),
					css.TextColor(StatusException()),
					css.Border(ManifestRuleWidth, StatusException()),
				),
			))
			continue
		}
		parseOut[parseIndex] = clip(css.Rules(
			statusChipBase,
			css.Bg(css.Transparent),
			// currentColor in the border (see statusChipBase) means one declaration
			// sets both text and border. Fewer declarations is not the point —
			// impossible-to-desync is.
			css.TextColor(toneColor(parseTone)),
		))
	}
	return parseOut
}

// StatusChip is the status badge, driven by a semantic tone rather than a color.
//
//	html.Span(html.Props{Class: design.Class(design.StatusChip(design.ToneException))}, "SHORT 4")
//
// Supply the label yourself — the words are domain vocabulary, and Atlas's are better
// than any generic set ("SHORT 4", "HELD AT HUB", "CLOSED CLEAN"). Color is a second
// channel on top of the words, never the only channel: a chip whose meaning is
// carried by hue alone fails for a colorblind operator and in a printed manifest.
func StatusChip(parseTone StatusTone) []css.Rule {
	return statusChipBundles[clampTone(parseTone)]
}

var statusValueBundles = buildStatusValues()

func buildStatusValues() [toneCount][]css.Rule {
	var parseOut [toneCount][]css.Rule
	for parseIndex := 0; parseIndex < toneCount; parseIndex++ {
		parseOut[parseIndex] = clip(css.Rules(
			css.TextColor(toneColor(StatusTone(parseIndex))),
			css.FontWeight.Semibold,
		))
	}
	return parseOut
}

// StatusValue tones a bare value rather than wrapping it in a chip: a negative
// on-hand quantity in a table cell, an overdue promise date, a variance figure.
//
// It sets color and weight only, so it composes over a table cell without disturbing
// the cell's alignment or family. Use it when the number IS the status — a chip beside
// a number that already says "-4" is redundant, and a dense manifest cannot afford
// redundancy.
//
// Same contract as everything in this file: a tone, never a color.
func StatusValue(parseTone StatusTone) []css.Rule {
	return statusValueBundles[clampTone(parseTone)]
}

func clampTone(parseTone StatusTone) int {
	if parseTone < 0 || int(parseTone) >= toneCount {
		return int(ToneNeutral)
	}
	return int(parseTone)
}
