package design

import "github.com/monstercameron/GoWebComponents/v5/css"

// This file is the single source of truth for Atlas's visual constants. Two kinds
// of constant live here and the difference matters:
//
//   - CSS custom properties (the Token* names) — anything a THEME is allowed to
//     change at runtime: colors, font stacks, the rail width. They are emitted once
//     into :root by Install and read back through the typed accessors below, so a
//     theme override rewrites one declaration instead of invalidating every class.
//   - Go constants (Space*, Radius*, Hairline*) — anything a theme must NOT change,
//     because changing it would change the design rather than re-skin it. A dark
//     Atlas is still Atlas; an Atlas with 16px radii and 40px gutters is not.
//
// That split is the useful discipline. If everything is a custom property you have
// built a configuration format, not a design system.

// --- custom-property names ----------------------------------------------------

// The token names are exported so a theme can override them by name without
// importing anything else from this package:
//
//	css.Global(`[data-theme="dusk"]`, css.Raw(design.TokenPaper, "#1A1D22"))
//
// Names are semantic, not literal: TokenInk is "the color you write with", not
// "black". That is what lets the dark theme invert paper and ink while every
// primitive keeps working untouched.
const (
	// Structure. These five carry the entire layout; a page built from only these
	// is already recognizably Atlas.
	TokenPaper     = "--atlas-paper"      // cool manila stock — the page itself
	TokenPaperSunk = "--atlas-paper-sunk" // recessed surfaces and table stripes
	TokenInk       = "--atlas-ink"        // primary text and heavy rules
	TokenGraphite  = "--atlas-graphite"   // secondary text, meta, labels
	TokenHairline  = "--atlas-hairline"   // 1px paperwork rules (decorative separation)
	TokenEdge      = "--atlas-edge"       // 1px CONTROL boundaries (must hold 3:1)

	// Structural accent. Lane blue is the placard/structure hue: links, focus
	// rings, "in the system" states. It is allowed to be structural because it is
	// low-chroma enough to sit under text without competing with it.
	TokenLane = "--atlas-lane"

	// Semantics. These three are NOT part of the palette in the decorative sense —
	// see the accessor comments. Overriding them in a theme is fine; using them for
	// ornament is the failure this package exists to prevent.
	TokenSignal = "--atlas-signal" // safety yellow — the one primary action
	TokenOxide  = "--atlas-oxide"  // exceptions, discrepancies, flags
	TokenVerify = "--atlas-verify" // approved, confirmed, closed clean

	// Derived. The lane placard needs a bar/label pair that is NOT simply ink on
	// paper in dark mode (a full-brightness bar in a dark UI is a flashbang), so it
	// gets its own two tokens and the dark theme picks a deep lane blue instead.
	// Add a derived token when a component needs a decision the palette cannot
	// make; do not smuggle the decision into the component as a literal.
	TokenPlacardBg = "--atlas-placard-bg"
	TokenPlacardFg = "--atlas-placard-fg"

	// TokenSignalFg is the second derived pair, and it exists because of a real
	// contrast failure that TestTokenContrastHoldsAAInBothThemes caught:
	// ButtonPrimary was ink-on-signal, and `ink` inverts between themes while
	// `signal` does not — safety yellow is a light color in BOTH themes. So the
	// dark theme rendered a #E8E7E1 label on a #F0C244 button at 1.36:1, which is
	// unreadable, while the light theme was fine at 9.18:1. See buttonPrimaryBundle.
	//
	// The lesson generalizes: any pair where ONE side inverts and the other does not
	// needs its own foreground token. Polarity inversion is not a safe default.
	TokenSignalFg = "--atlas-signal-fg"

	// Type. Font stacks are tokens so a future webfont can be dropped in via one
	// @font-face + one :root override, with no primitive edited. There are no
	// webfonts today: this repo ships zero @font-face rules and no CDN is
	// permitted, so all three stacks are system fonts.
	TokenFontDisplay = "--atlas-font-display"
	TokenFontBody    = "--atlas-font-body"
	TokenFontData    = "--atlas-font-data"

	// Layout. The console rail width is a token because the rail and the content
	// column must agree on it, and because a density preference is a plausible
	// future theme axis.
	TokenRailWidth = "--atlas-rail-width"
)

// TokenNames returns every custom property this design system defines, in
// declaration order. Install emits exactly this set, and
// TestEveryReferencedTokenIsDefined walks it to prove no primitive references a
// name that :root never defines.
//
// That test exists because of a specific, expensive class of bug: var(--typo)
// resolves to nothing, the declaration is dropped, and the element silently
// inherits — so a misspelled token looks like a layout bug, not a typo. A
// var(--name, fallback) would hide it even better. This package uses no fallbacks
// and proves the names instead.
func TokenNames() []string {
	return []string{
		TokenPaper, TokenPaperSunk, TokenInk, TokenGraphite, TokenHairline, TokenEdge,
		TokenLane, TokenSignal, TokenOxide, TokenVerify,
		TokenPlacardBg, TokenPlacardFg, TokenSignalFg,
		TokenFontDisplay, TokenFontBody, TokenFontData,
		TokenRailWidth,
	}
}

// --- typed color accessors ----------------------------------------------------
//
// Every accessor returns css.Var(...), i.e. var(--atlas-x) — never a literal hex.
// Primitives that hard-code hex cannot be themed, and a design system whose
// primitives cannot be themed will grow a second, divergent set of primitives the
// first time someone needs a dark mode.

// Paper is the page surface: cool manila stock. Deliberately not cream — cream
// plus a serif plus terracotta is a different (and by now very familiar) look, and
// Atlas is industrial paperwork, not a stationery brand.
func Paper() css.Color { return css.Var(TokenPaper) }

// PaperSunk is the recessed shade of paper: table zebra stripes, the console rail,
// input-ish regions. It is one step of value, not a new hue, so a stripe reads as
// "same paper, pressed" rather than as a second surface.
func PaperSunk() css.Color { return css.Var(TokenPaperSunk) }

// Ink is primary text and heavy rules. Pair it with Paper for body copy.
func Ink() css.Color { return css.Var(TokenInk) }

// Graphite is secondary text: meta, labels, units, timestamps, table headers. It
// is the only de-emphasis tool in the system — there is no opacity scale, because
// translucent text over a zebra stripe changes contrast per row.
func Graphite() css.Color { return css.Var(TokenGraphite) }

// Hairline is the 1px rule color. Nearly every separation in Atlas is a hairline;
// see Divider for why that is the answer instead of another box.
//
// Hairline is DECORATIVE separation and deliberately sits below the 3:1 non-text
// contrast threshold (it measures ~1.5:1 against paper in both themes). That is
// correct and intentional: a rule between two table rows carries no information the
// row layout does not already carry, so WCAG 1.4.11's "purely decorative" exclusion
// applies, and a 3:1 separator would be a mid-grey line under every row — the page
// would read as a grid of cells rather than as printed paperwork.
//
// The moment a 1px line becomes the thing that tells a user "this is a control",
// that exclusion stops applying. Use [Edge] there instead. TestTokenContrastHoldsAA
// pins both halves of this split.
func Hairline() css.Color { return css.Var(TokenHairline) }

// Edge is the CONTROL boundary color: the 1px border that identifies an input, a
// select or a textarea as something you can type into.
//
// It exists because Hairline was doing both jobs and could only be tuned for one.
// WCAG 1.4.11 requires 3:1 for "visual information required to identify user
// interface components", and an unfilled text field IS identified by its border and
// nothing else — so a 1.5:1 hairline field is, measurably, an invisible control. It
// looked fine in review because a reviewer already knows where the fields are.
//
// Edge measures 3.46:1 against paper and 3.15:1 against paper-sunk in light, and
// 3.62:1 / 3.33:1 in dark. It is NOT used for paperwork rules; using it there would
// undo the quiet the hairline buys.
func Edge() css.Color { return css.Var(TokenEdge) }

// Lane is placard blue: structure, links, focus rings, "pending / in the system".
// It is the one hue allowed to appear for non-status reasons, and only for
// structure — a link, a focus ring, a current-page marker. Not for filling shapes.
func Lane() css.Color { return css.Var(TokenLane) }

// PrimaryActionOnly is safety yellow, and the name is awkward on purpose.
//
// There is exactly ONE primary action per view, and this is its color. Written at
// a call site that is not that action — css.Bg(design.PrimaryActionOnly()) on a
// badge, an icon, a chart series — the name reads as a lie, which is the whole
// mechanism. The previous design used this hue as one of three ornamental accents
// and consequently had no primary action anywhere.
//
// In practice you should not need this at all: use ButtonPrimary, which is the
// only primitive that spends it.
func PrimaryActionOnly() css.Color { return css.Var(TokenSignal) }

// StatusException is oxide: a discrepancy, a short shipment, a failed check, a
// flag. It is a STATUS SEMANTIC, never decoration — the name says "exception" and
// not "red" so that using it for a decorative accent reads as a false claim about
// the data. Prefer StatusChip / StatusValue with ToneException, which is how a
// status is supposed to reach the screen.
func StatusException() css.Color { return css.Var(TokenOxide) }

// StatusVerified is verify green: approved, confirmed, received clean, closed
// clean. Same contract as StatusException — a semantic, not a palette entry. Green
// on an Atlas screen is a claim that something was checked and passed.
func StatusVerified() css.Color { return css.Var(TokenVerify) }

// PlacardBg and PlacardFg are the lane placard's bar and text colors.
//
// They are exported so a caller composing a custom placard body (see PlacardBar) stays
// on these two tokens rather than reaching for Ink/Paper. That distinction matters
// precisely in dark mode: light-mode placard-bg IS ink, so Ink() would look correct in
// review and then render a full-brightness bar in a dark UI.
func PlacardBg() css.Color { return css.Var(TokenPlacardBg) }
func PlacardFg() css.Color { return css.Var(TokenPlacardFg) }

// SignalFgOnly is the label color for the one primary action, and like
// PrimaryActionOnly the name is awkward on purpose: it is the ONLY correct
// foreground for a signal-yellow fill and it is correct for nothing else.
//
// Do not substitute Ink(). Ink inverts between themes and signal does not, so
// Ink-on-signal measures 9.18:1 in light and 1.36:1 in dark. This token is pinned
// near-black in BOTH themes, which is the whole point of it existing.
func SignalFgOnly() css.Color { return css.Var(TokenSignalFg) }

// RailWidth is the console rail's width as a typed length, for callers that need
// to reserve or offset by it.
func RailWidth() css.Length { return css.VarLength(TokenRailWidth) }

// --- light values -------------------------------------------------------------

// The literal hex values live in exactly two places: lightTokenValues and
// darkTokenValues. Nowhere else in the package is a hex color written.
func lightTokenValues() map[string]string {
	return map[string]string{
		TokenPaper:     "#F2F1ED",
		TokenPaperSunk: "#E8E7E1",
		TokenInk:       "#14161A",
		TokenGraphite:  "#5C626B",
		TokenHairline:  "#C9C7C0",
		// A warm grey from the same family as the hairline, two steps darker, so a
		// field border reads as "same paperwork, but this one you write on".
		TokenEdge:   "#838178",
		TokenLane:   "#1B4B8F",
		TokenSignal: "#E8B004",
		TokenOxide:  "#A63A21",
		TokenVerify: "#2F6B4F",

		// Light mode's placard is literally an inked bar on paper — the dock tag.
		TokenPlacardBg: "#14161A",
		TokenPlacardFg: "#F2F1ED",

		// Deliberately identical to light ink, and deliberately NOT written as
		// TokenInk's value by reference: the whole reason this token exists is that
		// it must NOT follow ink into the dark theme.
		TokenSignalFg: "#14161A",

		TokenFontDisplay: stackDisplay,
		TokenFontBody:    stackBody,
		TokenFontData:    stackData,

		TokenRailWidth: "232px",
	}
}

// --- dark values --------------------------------------------------------------
//
// The dark theme is a value inversion of the SAME semantics, not a different
// design. Paper darkens, ink lightens, hairline lifts off the background, and the
// three semantic hues are lightened just enough to hold contrast against dark
// paper — a #A63A21 oxide on a #16181C paper is unreadable, so oxide becomes a
// lighter tint of the same hue rather than a different color.
//
// Note what is NOT here: no gradients, no color-scheme lock, no !important. The
// old example-shell.css did all three and the result was an app that rendered
// near-black while reporting THEME: light, because nothing the app emitted could
// beat it.
func darkTokenValues() map[string]string {
	return map[string]string{
		TokenPaper:     "#16181C",
		TokenPaperSunk: "#1D2025",
		TokenInk:       "#E8E7E1",
		TokenGraphite:  "#9AA0A8",
		TokenHairline:  "#343841",
		// The dark edge is a cool grey (matching the dark hairline's hue) lifted far
		// enough off dark paper to clear 3:1. Note that edge and hairline invert
		// TOGETHER — the light pair is darker than paper, the dark pair is lighter —
		// which is what keeps "a rule is quieter than a field border" true in both.
		TokenEdge:   "#6A717E",
		TokenLane:   "#8AB0EA",
		TokenSignal: "#F0C244",
		TokenOxide:  "#E48267",
		TokenVerify: "#79B795",

		// A full-brightness bar in a dark UI is a flashbang, so the dark placard is
		// a deep lane blue with near-paper text instead of an inverted ink bar. This
		// is exactly why the placard got its own derived tokens.
		TokenPlacardBg: "#0E2545",
		TokenPlacardFg: "#E9EDF4",

		// NOT inverted. Safety yellow is a light color in both themes, so its label
		// stays dark in both. This single line is the fix for a 1.36:1 primary button.
		TokenSignalFg: "#14161A",
	}
}

// --- font stacks --------------------------------------------------------------
//
// System fonts only. Three roles, chosen for what is actually installed on the
// platforms Atlas runs on rather than for what would be nice:
//
//   - Display leads with the condensed faces because a condensed heavy uppercase title
//     is the dock-placard voice and is the one place this design system spends weight.
//     The Segoe UI Variable Display / Segoe UI / system-ui tail is the graceful
//     fallback: not condensed, but still the right *weight* and tracking.
//   - Body is the platform UI text face. Segoe UI Variable Text is optically sized
//     for small text, which is what most of Atlas is.
//   - Data leads with Cascadia Mono / Consolas on Windows, SF Mono / Menlo on
//     macOS, then ui-monospace. Every stack ends in a generic family so there is
//     always a resolution.
//
// KNOWN LIMITATION, and it is not fixable here: no webfont is permitted in this repo, and
// `font-stretch: condensed` is ignored by system faces that have no width axis (Segoe UI
// Variable has none), so on a platform with no condensed face installed the display voice
// degrades to plain bold uppercase. That is a real loss of identity and it is accepted.
//
// What the stack CAN do is stop degrading all the way to plain on platforms that do ship a
// condensed face under a different name, which the original three-entry stack did:
//
//   - Liberation Sans Narrow — the metric-compatible Arial Narrow clone, present on most
//     Linux distributions that ship the Liberation family.
//   - DejaVu Sans Condensed — the other common Linux condensed face, and the fallback of
//     last resort on minimal images.
//   - Avenir Next Condensed — ships with macOS. Humanist rather than grotesque, so the
//     placard voice shifts slightly, but a condensed heavy uppercase Avenir is far closer
//     to the intent than non-condensed Helvetica.
//   - Roboto Condensed — Android and ChromeOS.
//
// Ordering is "closest to the intended grotesque first, generic last". Naming faces that
// may be absent costs nothing: an unavailable family is skipped, not downloaded.
const (
	stackDisplay = `"Arial Narrow",Haettenschweiler,"Liberation Sans Narrow",` +
		`"DejaVu Sans Condensed","Avenir Next Condensed","Roboto Condensed",` +
		`"Segoe UI Variable Display","Segoe UI",system-ui,sans-serif`
	stackBody = `"Segoe UI Variable Text","Segoe UI",system-ui,sans-serif`
	stackData = `"Cascadia Mono",Consolas,"SF Mono",Menlo,ui-monospace,monospace`
)

// --- spacing scale ------------------------------------------------------------

// The spacing scale. Seven steps, roughly 1.5x, expressed in rem so a user's
// browser font-size scales the layout with the text — a px gutter next to rem text
// is how a page breaks at 200% zoom.
//
// These are css.Length constants rather than a closed enum type, so they drop
// straight into any css constructor: css.Padding(design.Space4). The cost of that
// ergonomics is that css.Gap(css.Px(13)) still compiles — the scale has teeth in
// review, not in the type checker. Passing an off-scale length also mints a new
// hashed class, so an unbounded set of ad-hoc values grows the registry.
const (
	Space1 css.Length = "0.25rem" // 4px  — icon/label gaps, chip padding
	Space2 css.Length = "0.5rem"  // 8px  — dense table cell padding
	Space3 css.Length = "0.75rem" // 12px — control padding, rail item padding
	Space4 css.Length = "1rem"    // 16px — surface padding, default stack gap
	Space5 css.Length = "1.5rem"  // 24px — between page sections
	Space6 css.Length = "2rem"    // 32px — page gutters
	Space7 css.Length = "3rem"    // 48px — between major page regions
)

// --- rules and radii ----------------------------------------------------------

const (
	// HairlineWidth is the 1px rule. It is a constant rather than a token because a
	// design where separators are 3px is a different design.
	HairlineWidth css.Length = "1px"

	// ManifestRuleWidth is the heavy rule: under a table header, above a total. It
	// is the one place ink gets thick, and it is what makes a table read as a
	// printed manifest rather than as a grid of divs.
	ManifestRuleWidth css.Length = "2px"

	// RadiusNone and RadiusTag are the ONLY two radii, and RadiusTag is 2px.
	//
	// This is a direct response to the failure being fixed: rounded dark boxes
	// nested four deep, every one at the same weight. A 2px radius reads as cut
	// paper. A 20px radius reads as a card, cards invite nesting, and nesting at
	// uniform weight destroys hierarchy. There is no radius scale to reach for.
	RadiusNone css.Length = "0"
	RadiusTag  css.Length = "2px"
)

// --- breakpoints --------------------------------------------------------------

// Two breakpoints, both max-width, because the desktop console is the primary
// target and the narrow layouts are reductions of it.
//
// The quality floor is a usable layout at ~380px: the rail becomes a horizontally
// scrolling strip, gutters drop to Space3, tables scroll inside TableScroll, and
// the two largest type steps are clamp()ed against the viewport so a page title
// cannot force a horizontal scrollbar.
const (
	breakRail   = 960 // below this the console rail stops being a rail
	breakNarrow = 520 // below this gutters and paddings tighten
)

// clip returns rules with cap == len.
//
// Every exported bundle is built once at init and returned by value. Without the
// clip, a caller writing append(design.Surface(), extra) could write into the
// shared bundle's spare capacity and silently change Surface for the whole
// process. With cap == len, append always allocates a copy. It is one expression
// and it removes an entire category of impossible-to-find bug.
func clip(parseRules []css.Rule) []css.Rule {
	return parseRules[:len(parseRules):len(parseRules)]
}

// Class folds one or more design bundles into a single hashed class name, for
// html.Props{Class: ...} call sites — which is how Atlas's markup is written.
//
//	html.Props{Class: design.Class(design.Surface(), design.Stack(design.Space4))}
//
// Folding everything into ONE class (rather than concatenating several class
// names) is deliberate: conflicts between bundles then resolve by argument order,
// last one wins, which a reader can see. Two class names on one element would
// resolve by stylesheet emission order instead, which nobody can see.
//
// The parameter is []css.Rule rather than ...any so the call site stays typed. To
// add a one-off rule, wrap it: design.Class(design.Surface(), []css.Rule{css.MarginY(design.Space5)}).
func Class(parseBundles ...[]css.Rule) string {
	var parseAll []css.Rule
	for _, parseBundle := range parseBundles {
		parseAll = append(parseAll, parseBundle...)
	}
	return css.New(parseAll...).String()
}
