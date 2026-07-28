//go:build !(js && wasm)

package design

// CONTRAST, MEASURED.
//
// The package asserted token POLARITY between themes — paper darkens, ink lightens —
// and nowhere asserted RATIOS. Polarity is not legibility: a palette can invert
// perfectly and still be unreadable, and the dark palette in particular had never been
// checked against anything but an eyeball. Two real failures were sitting in it.
//
// This file computes WCAG 2.1 contrast in Go, from the hex values in tokens.go, for
// every meaningful (foreground, background) pair in BOTH themes. No screenshot, no
// browser, no external tool: the palette is data, the formula is arithmetic, so this is
// a unit test and it runs in CI on every change to a hex digit.
//
// It is an INTERNAL test (package design, not design_test) for one reason: the palettes
// are unexported, and exporting them so a black-box test could read them would grow the
// public API to serve a test. An internal test file is the language's answer to that.
//
// # What it found
//
//  1. ButtonPrimary was ink-on-signal at 1.36:1 in DARK mode (9.18:1 in light). Ink
//     inverts between themes; safety yellow does not. Fixed by adding TokenSignalFg,
//     pinned near-black in both themes -> 9.18 / 10.79.
//  2. Input borders were on the hairline at 1.50:1, and an unfilled text field is
//     identified by its border and nothing else. Fixed by adding TokenEdge -> 3.46 /
//     3.62 against paper, 3.15 / 3.33 against paper-sunk.
//  3. The global lane-blue focus ring measured 2.11:1 against the LIGHT placard bar
//     (dark blue on near-black), so a control composed into a placard would be
//     focusable with an invisible ring, in one theme only. Fixed inside PlacardBar with
//     a scoped placard-fg inset ring -> 16.03 / 13.06.
//
// In all three the fix was a token or a scoped rule, never a lowered threshold.
//
// # How the thresholds are assigned
//
// Every pair is classified, and every pair carries a written reason. The reason field is
// asserted non-empty, which is the mechanism that stops a future contributor from
// quietly parking a failing pair in the lenient class: to do that you have to write down
// why, in this file, next to the number.
//
//   - classBodyText — 4.5:1. WCAG 2.1 AA 1.4.3 for text below 18.66px bold / 24px.
//     Almost everything in Atlas is 11-17px, so this is the default.
//   - classLargeText — 3.0:1. Reserved, and DELIBERATELY EMPTY: see
//     TestNoTextPairNeedsTheLargeTextRelaxation. Every text pair in this palette clears
//     4.5 even where it only owes 3.0, so no primitive's legibility depends on its font
//     size. That is a stronger property than AA and worth keeping.
//   - classBoundary — 3.0:1. WCAG 2.1 AA 1.4.11, for the visual information REQUIRED to
//     identify a control or a state: a field border, a focus ring, a current-page
//     marker, the heavy manifest rule.
//   - classDecorative — 1.25:1 plus cross-theme parity. 1.4.11 explicitly excludes
//     "purely decorative" graphics, and a hairline between two table rows carries no
//     information the row layout does not already carry. Pushing it to 3.0 would put a
//     mid-grey line under every row and turn printed paperwork into a spreadsheet grid.
//     The floor still proves the line renders at all, and the parity test proves the two
//     themes are equally legible rather than one being quietly worse.
//   - classFilledControl — 1.5:1 floor. A filled button's fill-versus-page ratio is not
//     a 1.4.11 requirement when the control is identified by a high-contrast label
//     inside it, which ButtonPrimary is (9.18:1 / 10.79:1). The number is measured and
//     pinned anyway so that a change to signal or paper trips this test and forces the
//     exemption to be re-read rather than inherited.

import (
	"math"
	"sort"
	"strings"
	"testing"
)

// --- the WCAG formulas --------------------------------------------------------

// srgbChannel linearizes one 8-bit sRGB channel per WCAG 2.1's definition of relative
// luminance. The 0.04045 knee and the 2.4 exponent are the specification's, not an
// approximation of it — a naive (c/255)^2.2 gamma is off by enough to flip a borderline
// pair, which is exactly the kind of pair this file exists to catch.
func srgbChannel(parseByte float64) float64 {
	parseC := parseByte / 255.0
	if parseC <= 0.04045 {
		return parseC / 12.92
	}
	return math.Pow((parseC+0.055)/1.055, 2.4)
}

type rgbColor struct{ r, g, b float64 }

// parseHex reads "#RRGGBB". The design system writes six-digit hex and nothing else —
// there is no alpha anywhere in the palette, which is itself a design decision (a
// translucent token would have a different contrast on every surface it landed on, which
// is precisely why status.go fills with solid color instead of a wash).
func parseHex(t *testing.T, parseValue string) rgbColor {
	t.Helper()
	if len(parseValue) != 7 || parseValue[0] != '#' {
		t.Fatalf("token value %q is not #RRGGBB; this test cannot measure it", parseValue)
	}
	parseNibble := func(parseChar byte) float64 {
		switch {
		case parseChar >= '0' && parseChar <= '9':
			return float64(parseChar - '0')
		case parseChar >= 'a' && parseChar <= 'f':
			return float64(parseChar-'a') + 10
		case parseChar >= 'A' && parseChar <= 'F':
			return float64(parseChar-'A') + 10
		}
		t.Fatalf("token value %q contains a non-hex digit", parseValue)
		return 0
	}
	return rgbColor{
		r: parseNibble(parseValue[1])*16 + parseNibble(parseValue[2]),
		g: parseNibble(parseValue[3])*16 + parseNibble(parseValue[4]),
		b: parseNibble(parseValue[5])*16 + parseNibble(parseValue[6]),
	}
}

func relativeLuminance(parseColor rgbColor) float64 {
	return 0.2126*srgbChannel(parseColor.r) +
		0.7152*srgbChannel(parseColor.g) +
		0.0722*srgbChannel(parseColor.b)
}

// contrastRatio is WCAG 2.1's (L1+0.05)/(L2+0.05), lighter over darker. It is symmetric,
// which is why a pair like paper-on-oxide and oxide-on-paper measure identically and only
// one of them needs to be listed.
func contrastRatio(parseA, parseB rgbColor) float64 {
	parseLightest, parseDarkest := relativeLuminance(parseA), relativeLuminance(parseB)
	if parseLightest < parseDarkest {
		parseLightest, parseDarkest = parseDarkest, parseLightest
	}
	return (parseLightest + 0.05) / (parseDarkest + 0.05)
}

// compositeOver alpha-composites an opaque foreground at the given opacity over an opaque
// background.
//
// It blends in the 8-bit sRGB space rather than in linear light because that is what CSS
// `opacity` actually does (compositing happens on non-premultiplied sRGB values unless a
// color-interpolation hint says otherwise), so this matches what the browser paints. It
// exists because the lane placard's field labels are drawn at opacity 0.7 and an
// unmeasured translucent label is exactly the kind of thing that silently drops below AA.
func compositeOver(parseFg rgbColor, parseAlpha float64, parseBg rgbColor) rgbColor {
	return rgbColor{
		r: parseAlpha*parseFg.r + (1-parseAlpha)*parseBg.r,
		g: parseAlpha*parseFg.g + (1-parseAlpha)*parseBg.g,
		b: parseAlpha*parseFg.b + (1-parseAlpha)*parseBg.b,
	}
}

// --- the pair table -----------------------------------------------------------

type contrastClass int

const (
	classBodyText contrastClass = iota
	classLargeText
	classBoundary
	classDecorative
	classFilledControl
)

func (parseClass contrastClass) minimum() float64 {
	switch parseClass {
	case classLargeText, classBoundary:
		return 3.0
	case classDecorative:
		return 1.25
	case classFilledControl:
		return 1.5
	default:
		return 4.5
	}
}

func (parseClass contrastClass) String() string {
	switch parseClass {
	case classLargeText:
		return "large-text"
	case classBoundary:
		return "boundary"
	case classDecorative:
		return "decorative"
	case classFilledControl:
		return "filled-control"
	default:
		return "body-text"
	}
}

// tokenPair is one measured pair. fgAlpha of 0 means "opaque"; any other value composites
// the foreground over the background first.
type tokenPair struct {
	fg      string
	bg      string
	fgAlpha float64
	class   contrastClass
	// where names the primitive(s) that put these two colors together. If you cannot
	// name one, the pair is not meaningful and does not belong in this table.
	where string
	// why justifies the CLASS, not the pair. Asserted non-empty. For anything below
	// 4.5 this is the argument a reviewer will read instead of the number.
	why string
}

// meaningfulPairs is every (foreground, background) combination this design system
// actually paints, with the primitive that paints it.
//
// "Meaningful" is doing work here. The cross product of thirteen color tokens is 156
// ordered pairs, almost all of which never touch — asserting on all of them would fail on
// combinations no primitive can produce (oxide on verify, placard-bg on signal) and the
// only way to make it pass would be to flatten the palette into thirteen mutually
// contrasting colors, i.e. to destroy it. So the table is derived from the primitives,
// and TestEveryColorTokenIsMeasured enforces that no token escapes it.
func meaningfulPairs() []tokenPair {
	return []tokenPair{
		// --- body text ---------------------------------------------------------
		{fg: TokenInk, bg: TokenPaper, class: classBodyText,
			where: "body copy, Prose/Data at every step, PageTitle, table cells",
			why:   "11-17px text; AA 1.4.3 body threshold"},
		{fg: TokenInk, bg: TokenPaperSunk, class: classBodyText,
			where: "table zebra rows, text inside Recess, rail hover label",
			why:   "same text, one step of paper value down"},
		{fg: TokenGraphite, bg: TokenPaper, class: classBodyText,
			where: "Eyebrow, FieldLabel, FieldHint, CellMeta, table headers, RailCode",
			why:   "graphite is the only de-emphasis tool; it still has to be read"},
		{fg: TokenGraphite, bg: TokenPaperSunk, class: classBodyText,
			where: "RailLink rest state, disabled button label, CellMeta on a zebra row",
			why:   "the rail is paper-sunk, so this is the rail's default text pair"},
		{fg: TokenLane, bg: TokenPaper, class: classBodyText,
			where: "Link, ButtonSecondary label, StatusChip(TonePending), CatalogActionLabel",
			why:   "link text, read at 13-15px"},
		{fg: TokenLane, bg: TokenPaperSunk, class: classBodyText,
			where: "a Link inside Recess, a pending chip on a zebra row",
			why:   "same, one step down"},
		{fg: TokenOxide, bg: TokenPaper, class: classBodyText,
			where: "FieldError, StatusValue(ToneException), the print exception chip",
			why:   "the exception message is the text most important to read"},
		{fg: TokenOxide, bg: TokenPaperSunk, class: classBodyText,
			where: "FieldError inside Recess, an exception value on a zebra row",
			why:   "FieldError lives in forms, and forms live in Recess"},
		{fg: TokenVerify, bg: TokenPaper, class: classBodyText,
			where: "StatusChip(ToneVerified), StatusValue(ToneVerified), AvailStocked",
			why:   "verified is a claim a reader has to be able to read"},
		{fg: TokenVerify, bg: TokenPaperSunk, class: classBodyText,
			where: "a verified chip on a zebra row",
			why:   "same, one step down"},
		{fg: TokenPlacardFg, bg: TokenPlacardBg, class: classBodyText,
			where: "LanePlacard hub codes, PROMISE and LANE values",
			why:   "the signature element's whole content"},
		{fg: TokenPaper, bg: TokenOxide, class: classBodyText,
			where: "StatusChip(ToneException) — the one filled tone",
			why:   "reversed pair; the filled chip's label"},
		{fg: TokenPaper, bg: TokenLane, class: classBodyText,
			where: "ButtonSecondary hover, ::selection, placard posture chip (pending)",
			why:   "reversed pair; a full inversion on hover has to stay readable"},
		{fg: TokenPaper, bg: TokenVerify, class: classBodyText,
			where: "placard posture chip (CLOSED)",
			why:   "reversed pair"},
		{fg: TokenPaper, bg: TokenGraphite, class: classBodyText,
			where: "placard posture chip (neutral tone)",
			why:   "reversed pair"},
		{fg: TokenSignalFg, bg: TokenSignal, class: classBodyText,
			where: "ButtonPrimary label and hover keyline",
			why:   "the one primary action's label; this pair is why TokenSignalFg exists",
		},
		{fg: TokenPlacardFg, bg: TokenPlacardBg, fgAlpha: 0.7, class: classBodyText,
			where: "LanePlacard field labels (PROMISE, LANE) at opacity 0.7",
			why:   "11px text through a CSS opacity; composited, then measured"},

		// --- boundaries and state markers --------------------------------------
		{fg: TokenEdge, bg: TokenPaper, class: classBoundary,
			where: "Input / InputData border",
			why:   "1.4.11: an unfilled text field is identified by its border alone"},
		{fg: TokenEdge, bg: TokenPaperSunk, class: classBoundary,
			where: "Input inside Recess — the filter bar and every form region",
			why:   "the field's own paper against the pressed paper it sits on"},
		{fg: TokenLane, bg: TokenPaper, class: classBoundary,
			where: ":focus-visible ring, ButtonSecondary border",
			why:   "1.4.11: a focus ring is a state indicator"},
		{fg: TokenLane, bg: TokenPaperSunk, class: classBoundary,
			where: "focus ring inside the rail, RailLinkCurrent's inset lane bar",
			why:   "1.4.11: the current-page marker identifies a state"},
		{fg: TokenPlacardFg, bg: TokenPlacardBg, class: classBoundary,
			where: "PlacardBar's scoped inset focus ring, and its print keyline",
			why:   "1.4.11; lane measured 2.11:1 here in light mode, hence the override"},
		{fg: TokenInk, bg: TokenPaper, class: classBoundary,
			where: "ManifestRule, PageHead's rule, the table header's 2px rule",
			why:   "1.4.11 graphical object: the heavy rule is structure, not ornament"},

		// --- decorative separation ---------------------------------------------
		{fg: TokenHairline, bg: TokenPaper, class: classDecorative,
			where: "Divider, Surface border, table row rules, CatalogRow rules",
			why: "1.4.11 excludes purely decorative graphics; a row rule carries no " +
				"information the row layout does not already carry, and a 3:1 separator " +
				"under every row turns paperwork into a spreadsheet grid. Controls use Edge."},
		{fg: TokenHairline, bg: TokenPaperSunk, class: classDecorative,
			where: "a hairline crossing a zebra row or sitting inside the rail",
			why:   "same rule, against the sunk shade — the worse of the two cases"},

		// --- measured exemptions -----------------------------------------------
		{fg: TokenSignal, bg: TokenPaper, class: classFilledControl,
			where: "ButtonPrimary's fill against the page",
			why: "1.4.11 does not require a filled control's fill to contrast with the " +
				"page when the control is identified by a high-contrast label inside it, " +
				"which this is (signal-fg on signal, 9.18:1 light / 10.79:1 dark), and no " +
				"STATE is distinguished by the fill. Reaching 3:1 against manila would " +
				"take signal to roughly #7A5C00 — no longer safety yellow, and it would " +
				"force a light label, reversing the documented dark-on-bright decision in " +
				"controls.go. Measured and pinned so a palette change re-opens this note."},
	}
}

// --- the tests ----------------------------------------------------------------

// TestTokenContrastHoldsAAInBothThemes is the main assertion. Run it with -v to get the
// full measured table, which is the artifact to paste into a design review.
func TestTokenContrastHoldsAAInBothThemes(t *testing.T) {
	for _, parseTheme := range themesUnderTest() {
		t.Run(parseTheme.name, func(t *testing.T) {
			t.Logf("%-46s %-16s %-6s %-5s %s", "PAIR", "CLASS", "RATIO", "MIN", "RESULT")
			for _, parsePair := range meaningfulPairs() {
				parseRatio := measurePair(t, parseTheme, parsePair)
				parseMin := parsePair.class.minimum()
				parseVerdict := "ok"
				if parseRatio < parseMin {
					parseVerdict = "FAIL"
				}
				t.Logf("%-46s %-16s %6.2f %5.2f %s",
					pairName(parsePair), parsePair.class, parseRatio, parseMin, parseVerdict)

				if strings.TrimSpace(parsePair.why) == "" {
					t.Errorf("%s: every pair must carry a written reason for its class", pairName(parsePair))
				}
				if strings.TrimSpace(parsePair.where) == "" {
					t.Errorf("%s: every pair must name the primitive that paints it", pairName(parsePair))
				}
				if parseRatio < parseMin {
					t.Errorf("%s theme: %s measures %.2f:1, below the %.2f:1 floor for %s (%s).\n"+
						"Fix the TOKEN in tokens.go, not this threshold.",
						parseTheme.name, pairName(parsePair), parseRatio, parseMin,
						parsePair.class, parsePair.where)
				}
			}
		})
	}
}

// TestNoTextPairNeedsTheLargeTextRelaxation pins a property stronger than AA: no text
// pair in this palette gets to lean on the 3:1 large-text allowance.
//
// It matters because font size is not stable. A step gets bumped, a clamp() floor gets
// lowered for a narrow viewport, a caller uses StepFine where the palette was checked at
// StepHead — and a pair that only ever cleared 3:1 silently becomes a 4.5:1 failure with
// no palette change to blame it on. Holding every text pair at 4.5 makes the type scale
// and the palette independent, which is worth the small amount of freedom it costs.
func TestNoTextPairNeedsTheLargeTextRelaxation(t *testing.T) {
	for _, parseTheme := range themesUnderTest() {
		for _, parsePair := range meaningfulPairs() {
			if parsePair.class != classBodyText && parsePair.class != classLargeText {
				continue
			}
			if parseRatio := measurePair(t, parseTheme, parsePair); parseRatio < 4.5 {
				t.Errorf("%s theme: %s measures %.2f:1 and would only pass as LARGE text; "+
					"this palette holds 4.5 for every text pair so the type scale and the "+
					"palette stay independent", parseTheme.name, pairName(parsePair), parseRatio)
			}
		}
	}
}

// TestDecorativeHairlineKeepsPolarityAndParity is what replaces a 3:1 assertion on the
// decorative separators: instead of demanding a ratio the design does not want, it demands
// that the two themes be EQUALLY legible.
//
// A hairline at 1.50:1 in light and 1.05:1 in dark would be a real dark-mode regression
// that a per-theme floor of 1.25 would not catch on the light side. Parity catches it.
func TestDecorativeHairlineKeepsPolarityAndParity(t *testing.T) {
	const parseTolerance = 0.35
	parseThemes := themesUnderTest()

	for _, parsePair := range meaningfulPairs() {
		if parsePair.class != classDecorative {
			continue
		}
		parseByTheme := map[string]float64{}
		for _, parseTheme := range parseThemes {
			parseByTheme[parseTheme.name] = measurePair(t, parseTheme, parsePair)
		}
		parseLight, parseDark := parseByTheme["light"], parseByTheme["dark"]
		if parseDelta := math.Abs(parseLight - parseDark); parseDelta > parseTolerance {
			t.Errorf("%s: light %.2f:1 vs dark %.2f:1 (delta %.2f) — one theme's separators "+
				"are materially weaker than the other's",
				pairName(parsePair), parseLight, parseDark, parseDelta)
		}
		t.Logf("%s: light %.2f:1, dark %.2f:1 — parity held", pairName(parsePair), parseLight, parseDark)
	}

	// And the polarity itself: the light hairline must be DARKER than light paper while
	// the dark hairline must be LIGHTER than dark paper. A hairline that failed to invert
	// would still pass a symmetric ratio check, because contrastRatio does not care which
	// side is lighter.
	parseLightHairline := relativeLuminance(parseHex(t, lightTokenValues()[TokenHairline]))
	parseLightPaper := relativeLuminance(parseHex(t, lightTokenValues()[TokenPaper]))
	if parseLightHairline >= parseLightPaper {
		t.Error("light hairline is not darker than light paper: the rule would be invisible ink")
	}
	parseDarkHairline := relativeLuminance(parseHex(t, darkTokenValues()[TokenHairline]))
	parseDarkPaper := relativeLuminance(parseHex(t, darkTokenValues()[TokenPaper]))
	if parseDarkHairline <= parseDarkPaper {
		t.Error("dark hairline is not lighter than dark paper: the theme did not invert")
	}

	// Edge must be strictly stronger than hairline in BOTH themes. That ordering is the
	// entire justification for having two tokens: if a future palette edit made the field
	// border quieter than a row rule, the split would be pointless and the WCAG argument
	// in Edge's doc comment would be false.
	for _, parseTheme := range parseThemes {
		parseEdge := contrastRatio(
			parseHex(t, parseTheme.values[TokenEdge]),
			parseHex(t, parseTheme.values[TokenPaper]))
		parseHairline := contrastRatio(
			parseHex(t, parseTheme.values[TokenHairline]),
			parseHex(t, parseTheme.values[TokenPaper]))
		if parseEdge <= parseHairline {
			t.Errorf("%s theme: edge (%.2f:1) is not stronger than hairline (%.2f:1); "+
				"the two-token split has no meaning", parseTheme.name, parseEdge, parseHairline)
		}
	}
}

// TestEveryColorTokenIsMeasured is the coverage gate, and it is the reason this file does
// not rot.
//
// A design system grows tokens. Without this test a new one arrives, gets used in a
// primitive, and is never measured — the suite stays green and the palette quietly stops
// being verified. With it, adding a color token to TokenNames() fails the build until it
// appears in meaningfulPairs() with a named primitive and a written reason. That is the
// forcing function; the arithmetic above is only the tool.
func TestEveryColorTokenIsMeasured(t *testing.T) {
	parseMeasured := map[string]bool{}
	for _, parsePair := range meaningfulPairs() {
		parseMeasured[parsePair.fg] = true
		parseMeasured[parsePair.bg] = true
	}

	parseMissing := []string{}
	for _, parseToken := range TokenNames() {
		parseValue, parseOK := lightTokenValues()[parseToken]
		if !parseOK || !strings.HasPrefix(parseValue, "#") {
			continue // font stacks and the rail width are not colors
		}
		if !parseMeasured[parseToken] {
			parseMissing = append(parseMissing, parseToken)
		}
	}
	sort.Strings(parseMissing)
	if len(parseMissing) > 0 {
		t.Errorf("color tokens %v are never measured for contrast. Add each to "+
			"meaningfulPairs() against the surface it is actually painted on, with the "+
			"primitive named and the threshold class justified.", parseMissing)
	}

	// The dark theme must override every color the light theme defines. This overlaps
	// TestDarkModeOverridesTheSameTokenNames in design_test.go, but that test reads the
	// EMITTED CSS while this one reads the maps — and a token present in the map but
	// missing from the dark map would make measurePair silently fall back to the light
	// value and report a dark ratio that does not exist. Measuring the wrong palette and
	// passing is worse than failing.
	parseDark := darkTokenValues()
	for _, parseToken := range TokenNames() {
		parseValue, parseOK := lightTokenValues()[parseToken]
		if !parseOK || !strings.HasPrefix(parseValue, "#") {
			continue
		}
		if _, parseHas := parseDark[parseToken]; !parseHas {
			t.Errorf("%s has no dark-theme value, so its dark contrast cannot be measured", parseToken)
		}
	}
}

// TestPrintPaletteHoldsAA runs the same table against the print token block.
//
// Print is a third theme (see print.go), so it owes the same guarantees — and it is the
// theme nobody looks at, which is exactly why it needs a test rather than a review. It
// also has a constraint the screen themes do not: office laser printers are monochrome,
// so a pair that passes on hue alone will not pass on paper. Pinning it at the same AA
// thresholds against a white ground is the closest a unit test gets to that.
func TestPrintPaletteHoldsAA(t *testing.T) {
	parseTheme := themeUnderTest{name: "print", values: mergeTheme(lightTokenValues(), printTokenValues())}
	for _, parsePair := range meaningfulPairs() {
		parseRatio := measurePair(t, parseTheme, parsePair)
		parseMin := parsePair.class.minimum()
		// The one class that does not apply on paper: a filled control's fill is not
		// printed at all (browsers drop background-color, and buttons are display:none in
		// the print layer), so measuring signal against white paper measures a thing that
		// never appears.
		if parsePair.class == classFilledControl {
			t.Logf("%-46s skipped on paper: the fill is not printed", pairName(parsePair))
			continue
		}
		t.Logf("%-46s %-16s %6.2f %5.2f", pairName(parsePair), parsePair.class, parseRatio, parseMin)
		if parseRatio < parseMin {
			t.Errorf("print palette: %s measures %.2f:1, below %.2f:1 for %s (%s)",
				pairName(parsePair), parseRatio, parseMin, parsePair.class, parsePair.where)
		}
	}
}

// --- helpers ------------------------------------------------------------------

type themeUnderTest struct {
	name   string
	values map[string]string
}

func themesUnderTest() []themeUnderTest {
	return []themeUnderTest{
		{name: "light", values: lightTokenValues()},
		{name: "dark", values: mergeTheme(lightTokenValues(), darkTokenValues())},
	}
}

// mergeTheme layers an override map over a base, which is how the browser resolves the
// dark and print :root blocks: they rewrite SOME names and inherit the rest. Measuring
// the dark theme against darkTokenValues() alone would fail to find a token the dark
// block forgot to override, because the lookup would come back empty rather than come
// back light.
func mergeTheme(parseBase, parseOverride map[string]string) map[string]string {
	parseOut := make(map[string]string, len(parseBase))
	for parseKey, parseValue := range parseBase {
		parseOut[parseKey] = parseValue
	}
	for parseKey, parseValue := range parseOverride {
		parseOut[parseKey] = parseValue
	}
	return parseOut
}

func measurePair(t *testing.T, parseTheme themeUnderTest, parsePair tokenPair) float64 {
	t.Helper()
	parseFgHex, parseOK := parseTheme.values[parsePair.fg]
	if !parseOK {
		t.Fatalf("%s theme has no value for %s", parseTheme.name, parsePair.fg)
	}
	parseBgHex, parseOK := parseTheme.values[parsePair.bg]
	if !parseOK {
		t.Fatalf("%s theme has no value for %s", parseTheme.name, parsePair.bg)
	}
	parseFg := parseHex(t, parseFgHex)
	parseBg := parseHex(t, parseBgHex)
	if parsePair.fgAlpha > 0 && parsePair.fgAlpha < 1 {
		parseFg = compositeOver(parseFg, parsePair.fgAlpha, parseBg)
	}
	return contrastRatio(parseFg, parseBg)
}

func pairName(parsePair tokenPair) string {
	parseName := strings.TrimPrefix(parsePair.fg, "--atlas-")
	if parsePair.fgAlpha > 0 && parsePair.fgAlpha < 1 {
		parseName += "@" + trimFloat(parsePair.fgAlpha)
	}
	return parseName + " on " + strings.TrimPrefix(parsePair.bg, "--atlas-")
}

func trimFloat(parseValue float64) string {
	parseOut := []byte("0.")
	parseHundredths := int(math.Round(parseValue * 100))
	parseOut = append(parseOut, byte('0'+parseHundredths/10))
	if parseHundredths%10 != 0 {
		parseOut = append(parseOut, byte('0'+parseHundredths%10))
	}
	return string(parseOut)
}
