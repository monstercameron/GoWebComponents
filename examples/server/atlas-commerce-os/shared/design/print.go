package design

import "github.com/monstercameron/GoWebComponents/v5/css"

// PRINT — the most on-brand thing in this package, and it was missing.
//
// Atlas's entire metaphor is printed paperwork: dock tags, manifests, lane placards,
// receipts, transfers, purchase orders. A receipt that prints as a screenshot of a
// dark web app with a navigation rail down the left is the metaphor breaking at
// exactly the moment it should pay off, because a printed receipt is not an imitation
// of the document — it IS the document.
//
// # Print is implemented as a third theme, not as a second stylesheet
//
// The package doc says "adding a third theme means adding a third :root block". This
// is that third block. Under @media print the token names are rewritten to ink on
// white, and every primitive in the package becomes print-correct without being
// touched — the same mechanism that makes dark mode free. That is the whole argument
// for semantic token names: --atlas-ink means "the color you write with", and on
// paper what you write with is black.
//
// # The trap that makes a naive print stylesheet worse than none
//
// Browsers do NOT print background-color or background-image by default (Chrome's
// "Background graphics" checkbox is off, and it is off in most print pipelines). So
// every FILLED element in this design system silently loses its fill on paper while
// keeping its foreground color:
//
//   - StatusChip(ToneException) is paper-on-oxide. Fill dropped -> white on white.
//     The one thing the design system exists to make impossible to miss becomes
//     invisible, and it is invisible only on paper, so nobody sees it in review.
//   - The lane placard is placard-fg on placard-bg. Fill dropped -> the signature
//     element prints as a blank rectangle.
//   - The placard posture chips are paper-on-tone. Same failure.
//   - ButtonPrimary is signal-fg on signal. Same failure (moot — buttons do not print).
//
// The fix is NOT `print-color-adjust: exact`. Forcing background printing spends the
// user's toner on decoration, and an oxide chip at 5.7:1 on a colour printer is 2:1 on
// the office monochrome laser that actually prints warehouse paperwork. The fix is to
// re-express every filled element as an OUTLINED one for print, which the design
// system can do losslessly because it already forbids color from being the only
// channel: "a chip whose meaning is carried by hue alone fails for a colorblind
// operator AND in a printed manifest" (status.go). Print is where that rule is cashed.
//
// So: each filled primitive carries its own @media print block, next to the fill it
// has to undo. They are not collected here, deliberately — a print override that lives
// in a different file from the rule it overrides is an override that rots.
//
// # What this file owns
//
// Only what must be true of the whole document: the page box, the token rewrite, the
// element baseline, table pagination, and link/heading break behavior. Everything
// component-shaped lives with its component.

// printQuery is the print media query.
//
// GAP NOTE: css.MediaQuery has typed constructors for MinW/MaxW and a Dark constant
// but nothing for print or reduced-motion, so both go through css.RawMedia. A typed
// css.Print and css.ReducedMotion would remove the two remaining raw queries in this
// package; a typo in a raw media string is a silently dead block, not a compile error,
// which is exactly the failure mode the typed constructors exist to prevent.
const printQuery css.MediaQuery = "print"

// printTokenValues is the third palette: ink on white.
//
// Note which names are and are not rewritten, because the choices carry the print
// design:
//
//   - paper -> pure white, not the manila #F2F1ED. On screen manila is the stock; on
//     paper the stock is whatever is in the tray, and painting a background over it
//     is both wasteful and (since backgrounds do not print) a lie.
//   - paper-sunk -> a very light grey. It survives as a table zebra stripe IF the user
//     turns background graphics on, and costs nothing when they do not. It is light
//     enough that ink-on-sunk stays black-on-near-white either way.
//   - ink -> pure black, graphite -> a real grey rather than a blue-grey. Monochrome
//     lasers dither hue, and a dithered 11px label is mud.
//   - hairline and edge -> mid greys that survive halftoning. #C9C7C0 prints as
//     nothing on a 600dpi laser; the rules that make a manifest a manifest have to be
//     visibly darker on paper than they are on a screen.
//   - lane -> black. A blue link on monochrome paper is a grey link, and the underline
//     already carries "this was a link". Keeping the hue would only cost contrast.
//   - The three semantics KEEP their hues, lightly darkened for halftone. On a colour
//     printer they still work; on a monochrome one they dither to distinguishable
//     greys (oxide dark, verify mid, signal light) which is a real third channel after
//     the word and the border weight.
//   - placard-bg -> white and placard-fg -> black: the placard inverts to a stamped
//     outline rather than a solid bar. See placard.go for why an inked bar is the wrong
//     thing to print.
func printTokenValues() map[string]string {
	return map[string]string{
		TokenPaper:     "#FFFFFF",
		TokenPaperSunk: "#F4F4F2",
		TokenInk:       "#000000",
		TokenGraphite:  "#404040",
		TokenHairline:  "#8C8C8C",
		TokenEdge:      "#5A5A5A",
		TokenLane:      "#000000",
		TokenSignal:    "#B8860B",
		TokenOxide:     "#8C2D14",
		TokenVerify:    "#1F5138",
		TokenPlacardBg: "#FFFFFF",
		TokenPlacardFg: "#000000",
		TokenSignalFg:  "#000000",
	}
}

// installPrint emits the print layer. Called from Install, after the dark override, so
// that within :root the print block is emitted last and wins over both — @media blocks
// at equal specificity resolve by order, and a print rendering must beat a
// prefers-color-scheme:dark rendering (a dark-mode user printing a receipt still wants
// ink on paper, and the two queries can both match).
func installPrint() {
	css.Global(":root", css.Media(printQuery, declarations(printTokenValues())...)...)

	// The page box. Freight paperwork has a generous margin because it gets punched,
	// stapled and filed; 14mm is roughly a half inch, which clears a two-hole punch.
	//
	// GAP NOTE: @page is not a selector, and the css package has no at-rule
	// constructor for it. css.Global emits its selector verbatim, so passing "@page"
	// produces a valid `@page{margin:14mm;}` — a documented abuse of Global rather
	// than a supported path. A css.Page(rules…) constructor would make this honest.
	css.Global("@page", css.Raw("margin", "14mm"))

	css.Global("html", css.Media(printQuery,
		// The screen baseline sets scroll-padding-top for sticky headers. Paper has no
		// sticky headers and no scrolling; leaving it in adds 5rem of dead space at
		// every anchor target.
		css.Raw("scroll-padding-top", "0"),
	)...)

	css.Global("body", css.Media(printQuery,
		css.Bg(Paper()),
		css.TextColor(Ink()),
		// 100vh on paper means "at least one full page even if the document is two
		// lines", which prints a trailing blank page often enough to be a support call.
		css.MinHeight(css.Auto),
		// Print at the base step rather than the screen's — paper is read at arm's
		// length, and 15px at 600dpi is smaller than 15px at 96dpi.
		css.FontSize(css.RawLength("10.5pt")),
		css.LineHeight(css.Number("1.4")),
		// orphans/widows are the two properties that stop a paragraph leaving one line
		// alone at a page break. They exist for exactly this and are ignored on screen.
		css.Raw("orphans", "3"),
		css.Raw("widows", "3"),
	)...)

	// Table pagination — the rules that make a long manifest print like a manifest.
	//
	// display:table-header-group makes the browser REPEAT <thead> at the top of every
	// printed page. That is the single most valuable print rule for this app: a
	// 200-row discrepancy manifest that spills onto page three with no column headers
	// is unreadable, and it is the paper equivalent of the sticky header the table
	// already has on screen. It also has to be restated because the screen rules give
	// thead cells position:sticky, and a sticky header inside a print context can
	// paint over the first row of every page.
	css.Global("thead", css.Media(printQuery, css.Raw("display", "table-header-group"))...)
	css.Global("tfoot", css.Media(printQuery, css.Raw("display", "table-footer-group"))...)
	css.Global("thead th", css.Media(printQuery,
		css.Position.Static,
		css.Bg(css.Transparent),
	)...)

	// break-inside:avoid on the ROW, not on the table. Avoiding a break inside the
	// whole table would push a 200-row manifest onto a fresh page and then break it
	// anyway (the fragmentation engine gives up when the box cannot fit), which is
	// worse than not asking. Rows and cells are small enough to honor it, so a row
	// never shears in half across the fold — which is the actual requirement.
	css.Global("tr", css.Media(printQuery,
		css.Raw("break-inside", "avoid"),
		css.Raw("page-break-inside", "avoid"), // legacy alias; some print engines only take this
	)...)
	css.Global("td,th", css.Media(printQuery, css.Raw("break-inside", "avoid"))...)

	// A heading stranded at the bottom of a page with its content overleaf is the
	// classic print defect. break-after:avoid binds a title to what follows it.
	css.Global("h1,h2,h3", css.Media(printQuery,
		css.Raw("break-after", "avoid"),
		css.Raw("break-inside", "avoid"),
	)...)

	// Interactive chrome that cannot exist on paper. This is a very short list on
	// purpose: everything else opts out through ScreenOnly, because the design system
	// cannot tell a filter bar (chrome) from a filled-in receiving form (content), and
	// guessing wrong erases an operator's typed values from their own printout.
	css.Global("button,[role=\"button\"]", css.Media(printQuery, css.Display.None)...)

	// A form field still prints — a filled-in transfer form is a document — but it
	// prints as a ruled blank rather than as a control: no rounded corner, no fill,
	// and a real black keyline so a handwritten entry has somewhere to sit.
	css.Global("input,select,textarea", css.Media(printQuery,
		css.Bg(css.Transparent),
		css.TextColor(Ink()),
		css.Border(HairlineWidth, Ink()),
		css.Rounded(RadiusNone),
	)...)
}

// --- print/screen visibility --------------------------------------------------

var screenOnlyBundle = clip(css.Rules(
	css.Media(printQuery, css.Display.None),
))

// ScreenOnly hides an element in print. It is the escape hatch that lets this package
// keep its global print layer tiny.
//
// Use it on chrome: a filter bar, a pagination row, a toolbar, a "show all" affordance,
// a live search box, a toast region. Do NOT use it on a form the operator filled in —
// the whole point of printing a receiving document is the values that were typed into
// it, and the design system has no way to tell those two Recess blocks apart. That
// judgement is the caller's, which is why this is a primitive and not a heuristic.
//
//	html.Div(html.Props{Class: design.Class(design.Recess(), design.Cluster(design.Space3), design.ScreenOnly())}, filters…)
func ScreenOnly() []css.Rule { return screenOnlyBundle }

var printOnlyBundle = clip(css.Rules(
	css.Display.None,
	// Note the direction: hidden by default, revealed under print. Written the other
	// way round (visible, then hidden on screen via a `screen` media query) it would
	// be visible in any context that matches neither query — a screen reader's own
	// media context, a `print` preview in an unusual engine, an environment where the
	// UA does not support media queries at all. Default-hidden fails safe.
	css.Media(printQuery, css.Display.Block),
))

// PrintOnly reveals an element only in print.
//
// Freight paperwork has a footer the screen does not need and the paper cannot do
// without: who printed it, when, from which hub, and the document id. A printed page
// that cannot be traced back to its source is not paperwork, it is a photocopy.
//
//	html.Div(html.Props{Class: design.Class(design.PrintOnly(), design.Data(design.StepMicro))},
//	    html.Text("RCV-ILLINOIS-001 · IL-HUB · PRINTED 2026-07-26"))
func PrintOnly() []css.Rule { return printOnlyBundle }
