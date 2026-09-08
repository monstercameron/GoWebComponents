//go:build !(js && wasm)

package design_test

// These tests assert on the CSS the package actually EMITS, not on the Go values it
// builds. That distinction is the whole point: a design system's contract is the
// stylesheet, and a bundle that composes beautifully in Go but emits a dropped
// declaration is broken in the only place that matters.
//
// The native buffer sink makes this cheap — css.Harvest() returns everything emitted
// since the last css.Reset() — so each test resets, folds exactly what it cares
// about, and reads the text back. This mirrors css/css_test.go and css/global_test.go.

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/css"
	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/design"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// reset gives each test a clean process-wide registry and sink.
//
// Note what this proves incidentally: because every design bundle folds lazily (no
// init-time css.New), a Reset followed by a re-fold produces identical class names and
// re-emits identical CSS. An init-time fold would hand out class names whose CSS the
// Reset had discarded, and the whole suite would silently assert on an empty sink.
func reset(t *testing.T) {
	t.Helper()
	css.Reset()
}

// --- tokens -------------------------------------------------------------------

func TestInstallEmitsEveryTokenIntoRoot(t *testing.T) {
	reset(t)
	design.Install()
	parseCSS := css.Harvest()

	parseRoot := blockFor(t, parseCSS, ":root{")
	for _, parseToken := range design.TokenNames() {
		if !strings.Contains(parseRoot, parseToken+":") {
			t.Errorf("token %s is not declared in the :root block", parseToken)
		}
	}

	// Spot-check the actual values, so a paste error in the palette is caught rather
	// than merely "some value is present".
	for _, parseWant := range []string{
		"--atlas-paper:#F2F1ED",
		"--atlas-paper-sunk:#E8E7E1",
		"--atlas-ink:#14161A",
		"--atlas-graphite:#5C626B",
		"--atlas-hairline:#C9C7C0",
		"--atlas-lane:#1B4B8F",
		"--atlas-signal:#E8B004",
		"--atlas-oxide:#A63A21",
		"--atlas-verify:#2F6B4F",
		"--atlas-rail-width:232px",
	} {
		if !strings.Contains(parseRoot, parseWant) {
			t.Errorf("missing token declaration %q in :root", parseWant)
		}
	}

	// System fonts only. A webfont would need an @font-face, and there is no CDN.
	if strings.Contains(parseCSS, "@font-face") || strings.Contains(parseCSS, "http") {
		t.Error("design system must not reference external fonts or URLs")
	}
}

func TestDarkModeOverridesTheSameTokenNames(t *testing.T) {
	reset(t)
	design.Install()
	parseCSS := css.Harvest()

	const parseQuery = "@media (prefers-color-scheme:dark){:root{"
	parseIndex := strings.Index(parseCSS, parseQuery)
	if parseIndex < 0 {
		t.Fatalf("no dark-mode :root override emitted:\n%s", parseCSS)
	}
	parseDark := parseCSS[parseIndex+len(parseQuery):]
	parseDark = parseDark[:strings.Index(parseDark, "}")]

	// Theming works by rewriting the SAME names. Every color token must be overridden
	// (fonts and the rail width intentionally are not — they are not theme-dependent).
	for _, parseToken := range []string{
		design.TokenPaper, design.TokenPaperSunk, design.TokenInk, design.TokenGraphite,
		design.TokenHairline, design.TokenLane, design.TokenSignal, design.TokenOxide,
		design.TokenVerify, design.TokenPlacardBg, design.TokenPlacardFg,
	} {
		if !strings.Contains(parseDark, parseToken+":") {
			t.Errorf("dark theme does not override %s", parseToken)
		}
	}

	// Ink and paper must actually swap polarity, otherwise "dark mode" is cosmetic.
	if !strings.Contains(parseDark, design.TokenPaper+":#16181C") {
		t.Error("dark paper is not dark")
	}
	if !strings.Contains(parseDark, design.TokenInk+":#E8E7E1") {
		t.Error("dark ink is not light")
	}
}

// TestNoThemeLockRegression is the guard against the specific bug this design system
// replaced: example-shell.css hard-set `color-scheme: dark` and used !important
// gradients, so the app rendered near-black while reporting THEME: light and nothing
// it emitted could win.
func TestNoThemeLockRegression(t *testing.T) {
	reset(t)
	design.Install()
	foldEveryPrimitive()
	parseCSS := css.Harvest()

	if strings.Contains(parseCSS, "!important") {
		t.Error("design system must emit no !important: it is what made the old theme unfightable")
	}
	// Careful with the substring: "prefers-color-scheme:dark" (the media query, which
	// is correct and required) also contains "color-scheme:dark". Match the property
	// at the start of a declaration instead.
	if regexp.MustCompile(`[{;]color-scheme:dark`).MatchString(parseCSS) {
		t.Error("color-scheme must be `light dark`, never locked to one scheme")
	}
	if !strings.Contains(parseCSS, "color-scheme:light dark") {
		t.Error("expected `color-scheme: light dark` so the UA follows the user preference")
	}
	// Gradients: there are exactly two in the system, both perforations (the placard's tear
	// edge and the rail's punched trailing edge), and both are radial. A linear-gradient
	// anywhere is the old example-shell.css look coming back.
	//
	// This assertion used to read `Contains("gradient") && !Contains("repeating-radial-
	// gradient")`, which passed for the wrong reason: it only proved that SOME
	// repeating-radial-gradient existed, so a linear-gradient added beside it would have
	// sailed through. It also pinned the broken form of the perforation — the repeating
	// variant fills its tile solid and never rendered a scallop (see placard.go). Matching
	// every gradient function by name is the assertion that was meant.
	for _, parseMatch := range regexp.MustCompile(`[a-z-]*gradient\(`).FindAllString(parseCSS, -1) {
		if parseMatch != "radial-gradient(" {
			t.Errorf("unexpected gradient %q: the only gradients in this system are the "+
				"placard and rail perforations, both radial-gradient", parseMatch)
		}
	}
}

// TestEveryReferencedTokenIsDefined is the defense against the silent-typo failure
// mode: var(--atlas-hairlne) resolves to nothing, the declaration is dropped, and the
// element inherits — so a one-character typo looks like a layout bug. Because this
// package uses no var() fallbacks, every referenced name must exist in :root.
func TestEveryReferencedTokenIsDefined(t *testing.T) {
	reset(t)
	design.Install()
	foldEveryPrimitive()
	parseCSS := css.Harvest()

	parseDefined := map[string]bool{}
	for _, parseToken := range design.TokenNames() {
		parseDefined[parseToken] = true
	}

	parseRefs := regexp.MustCompile(`var\((--[a-zA-Z0-9_-]+)`).FindAllStringSubmatch(parseCSS, -1)
	if len(parseRefs) == 0 {
		t.Fatal("no var() references found; the primitives are not token-driven")
	}
	for _, parseMatch := range parseRefs {
		if !parseDefined[parseMatch[1]] {
			t.Errorf("primitive references %s, which TokenNames/:root never defines", parseMatch[1])
		}
	}
}

// TestPrimitivesUseNoLiteralColors proves the primitives are themeable: a hex color
// anywhere outside the two :root blocks is a primitive that a theme cannot reach.
func TestPrimitivesUseNoLiteralColors(t *testing.T) {
	reset(t)
	foldEveryPrimitive() // note: Install is NOT called, so no :root block is emitted
	parseCSS := css.Harvest()

	if parseMatches := regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b`).FindAllString(parseCSS, -1); len(parseMatches) > 0 {
		t.Errorf("primitives contain literal colors %v; every color must be a var() token", parseMatches)
	}
}

// --- determinism --------------------------------------------------------------

// TestClassNamesAreDeterministicAcrossRuns folds every primitive twice, from a clean
// registry each time, and requires identical class names.
//
// This matters more than it looks: class names are content hashes, and SSR emits the
// server's names into the document while the client re-folds the same bundles and
// seeds its registry from those names. If a fold were order-dependent (a map iteration
// leaking into the canonical serialization, say), the client would mint different
// names, fail to recognize the server's rules, re-inject all of them, and every
// element would carry a class the stylesheet never defined.
func TestClassNamesAreDeterministicAcrossRuns(t *testing.T) {
	reset(t)
	design.Install()
	parseFirst := foldEveryPrimitive()
	parseFirstCSS := css.Harvest()

	reset(t)
	design.Install()
	parseSecond := foldEveryPrimitive()
	parseSecondCSS := css.Harvest()

	if len(parseFirst) != len(parseSecond) {
		t.Fatalf("fold count changed between runs: %d vs %d", len(parseFirst), len(parseSecond))
	}
	for parseName, parseClass := range parseFirst {
		if parseSecond[parseName] != parseClass {
			t.Errorf("%s: class name is not deterministic: %q then %q", parseName, parseClass, parseSecond[parseName])
		}
		if parseClass == "" {
			t.Errorf("%s: folded to an empty class (the bundle is empty)", parseName)
		}
	}
	if parseFirstCSS != parseSecondCSS {
		t.Error("emitted CSS differs between two identical runs")
	}
}

// TestDistinctPrimitivesFoldToDistinctClasses catches the copy-paste failure where two
// primitives are accidentally identical — e.g. ButtonSecondary and ButtonQuiet ending
// up the same because a colour override was lost in an edit.
func TestDistinctPrimitivesFoldToDistinctClasses(t *testing.T) {
	reset(t)
	parseFolds := foldEveryPrimitive()

	parseByClass := map[string][]string{}
	for parseName, parseClass := range parseFolds {
		parseByClass[parseClass] = append(parseByClass[parseClass], parseName)
	}
	for parseClass, parseNames := range parseByClass {
		if len(parseNames) > 1 {
			sort.Strings(parseNames)
			t.Errorf("primitives %v all fold to %s — they are byte-identical", parseNames, parseClass)
		}
	}
}

// TestBundlesAreImmutableToCallers proves the clip() discipline: appending to a
// returned bundle must not mutate the shared package-level slice.
func TestBundlesAreImmutableToCallers(t *testing.T) {
	reset(t)
	parseBefore := design.Class(design.Surface())

	parseHijacked := append(design.Surface(), css.Bg(css.Red500))
	if len(parseHijacked) == 0 {
		t.Fatal("Surface bundle is empty")
	}

	if parseAfter := design.Class(design.Surface()); parseAfter != parseBefore {
		t.Errorf("appending to a returned bundle mutated it: %q -> %q", parseBefore, parseAfter)
	}
}

// --- accessibility floor ------------------------------------------------------

func TestFocusVisibleIsEmittedGloballyAndPerControl(t *testing.T) {
	reset(t)
	design.Install()
	if !strings.Contains(css.Harvest(), ":focus-visible{outline:2px solid var(--atlas-lane)") {
		t.Errorf("no global :focus-visible ring emitted:\n%s", css.Harvest())
	}

	// Every interactive primitive must carry its own visible focus state too, so a
	// component that overrides `outline` locally cannot silently drop the ring.
	for parseName, parseBundle := range map[string][]css.Rule{
		"ButtonPrimary":   design.ButtonPrimary(),
		"ButtonSecondary": design.ButtonSecondary(),
		"ButtonQuiet":     design.ButtonQuiet(),
		"Input":           design.Input(),
		"InputData":       design.InputData(),
		"Link":            design.Link(),
		"RailLink":        design.RailLink(),
		"RailLinkCurrent": design.RailLinkCurrent(),
	} {
		reset(t)
		parseClass := design.Class(parseBundle)
		parseCSS := css.Harvest()
		if !strings.Contains(parseCSS, "."+parseClass+":focus-visible{") {
			t.Errorf("%s emits no :focus-visible rule:\n%s", parseName, parseCSS)
		}
		if !strings.Contains(parseCSS, "outline:2px solid var(--atlas-lane)") {
			t.Errorf("%s focus ring is not the lane-blue 2px outline", parseName)
		}
	}
}

// TestEveryTransitionCarriesAReducedMotionOverride counts declarations rather than
// inspecting a list, so a NEW primitive that adds a transition without going through
// withMotion fails this test automatically. That is the property worth having: the
// test guards the rule, not today's call sites.
func TestEveryTransitionCarriesAReducedMotionOverride(t *testing.T) {
	reset(t)
	design.Install()
	foldEveryPrimitive()
	parseCSS := css.Harvest()

	parseBase := 0
	parseReduced := 0
	for _, parseBlock := range strings.SplitAfter(parseCSS, "}") {
		if !strings.Contains(parseBlock, "transition-duration:") &&
			!strings.Contains(parseBlock, "transition:") {
			continue
		}
		if strings.Contains(parseBlock, "prefers-reduced-motion") {
			continue
		}
		parseBase++
	}
	for _, parseBlock := range strings.Split(parseCSS, "@media (prefers-reduced-motion: reduce){")[1:] {
		if strings.Contains(parseBlock, "transition-duration:0.01ms") {
			parseReduced++
		}
	}
	if parseBase == 0 {
		t.Fatal("no transitions emitted at all; the counting test would pass vacuously")
	}
	if parseReduced < parseBase {
		t.Errorf("%d transitions but only %d reduced-motion overrides: a primitive added motion without withMotion", parseBase, parseReduced)
	}
}

// --- print --------------------------------------------------------------------

// TestPrintLayerEmitsPaperworkRules is the regression net for a layer nobody looks at.
//
// A print stylesheet is uniquely prone to rotting: it is invisible in every screenshot,
// in every browser tab and in every review, so a rule that stops being emitted is
// discovered by an operator holding a bad printout weeks later. Asserting on the emitted
// CSS is the only cheap way to know it is still there.
//
// contrast_test.go covers the print PALETTE. This covers the print STRUCTURE.
func TestPrintLayerEmitsPaperworkRules(t *testing.T) {
	reset(t)
	design.Install()
	parseGlobalCSS := css.Harvest()

	for _, parseWant := range []string{
		// The page box.
		"@page{margin:14mm;}",
		// The third token block: ink on white.
		"@media print{:root{",
		"--atlas-paper:#FFFFFF",
		"--atlas-ink:#000000",
		// The single most valuable print rule in the package: a repeated table header, so
		// a manifest that spills onto page three still has column meanings.
		"@media print{thead{display:table-header-group;}}",
		// A row must not shear across the fold. Both the modern property and the legacy
		// alias, because some print engines only implement the alias.
		"break-inside:avoid",
		"page-break-inside:avoid",
		// A heading must not strand itself at the bottom of a page.
		"break-after:avoid",
		// Orphan/widow control, which only exists for paged media.
		"orphans:3",
		"widows:3",
	} {
		if !strings.Contains(parseGlobalCSS, parseWant) {
			t.Errorf("print layer is missing %q", parseWant)
		}
	}

	// Per-primitive print overrides. These are NOT in the global layer, and the reason is
	// the cascade fact that cost a whole probe run: `@media print{button{display:none}}`
	// is 0-0-1 and loses to a button bundle's own `.c-x{display:inline-flex}` at 0-1-0.
	// An at-rule does not raise specificity, so a styled element can only be re-styled for
	// print by the bundle that styled it. Each of these asserts that the override lives
	// with its primitive.
	for _, parseCase := range []struct {
		name   string
		bundle []css.Rule
		want   []string
	}{
		{"ConsoleRail", design.ConsoleRail(), []string{"display:none"}},
		{"ButtonPrimary", design.ButtonPrimary(), []string{"display:none"}},
		{"ScreenOnly", design.ScreenOnly(), []string{"display:none"}},
		{"CatalogActionLabel", design.CatalogActionLabel(), []string{"display:none"}},
		// The scroll container is the expensive one to get wrong: overflow-x:auto does
		// not scroll on paper, it CLIPS, so a wide manifest would print with its right
		// columns silently missing and look complete.
		{"TableScroll", design.TableScroll(), []string{"overflow:visible"}},
		// Browsers do not print background-color, so every FILLED element must re-form
		// as an outline or its paper-colored text lands on white paper.
		{"StatusChipException", design.StatusChip(design.ToneException), []string{
			"background-color:transparent",
			"border:2px solid var(--atlas-oxide)",
		}},
		{"PlacardBar", design.PlacardBar(), []string{
			"border:2px solid var(--atlas-placard-fg)",
			// The perforation cannot survive as paper-on-paper discs, so it becomes a
			// printed tear line.
			"border-top:2px dotted var(--atlas-placard-fg)",
		}},
		// The rail is display:none in print, but a hidden grid ITEM still leaves its
		// 232px track, so the shell has to collapse too. Two files that must agree.
		{"ConsoleShell", design.ConsoleShell(), []string{"grid-template-columns:minmax(0,1fr)"}},
	} {
		reset(t)
		parseClass := design.Class(parseCase.bundle)
		parseCSS := css.Harvest()
		parsePrintBlocks := strings.Split(parseCSS, "@media print{")
		if len(parsePrintBlocks) < 2 {
			t.Errorf("%s emits no @media print block at all:\n%s", parseCase.name, parseCSS)
			continue
		}
		parsePrint := strings.Join(parsePrintBlocks[1:], "@media print{")
		if !strings.Contains(parsePrint, parseClass) {
			t.Errorf("%s's print block does not target its own class %s", parseCase.name, parseClass)
		}
		for _, parseWant := range parseCase.want {
			if !strings.Contains(parsePrint, parseWant) {
				t.Errorf("%s's print block is missing %q:\n%s", parseCase.name, parseWant, parsePrint)
			}
		}
	}

	// PrintOnly is the inverse and has to be default-hidden: visible-then-hidden-on-screen
	// would leak into any context matching neither query.
	reset(t)
	parsePrintOnly := design.Class(design.PrintOnly())
	parseCSS := css.Harvest()
	if !strings.Contains(parseCSS, "."+parsePrintOnly+"{display:none;}") {
		t.Errorf("PrintOnly must be hidden by default (fail-safe), got:\n%s", parseCSS)
	}
	if !strings.Contains(parseCSS, "@media print{."+parsePrintOnly+"{display:block;}}") {
		t.Errorf("PrintOnly does not reveal itself in print:\n%s", parseCSS)
	}
}

// --- responsiveness -----------------------------------------------------------

func TestShellCollapsesAndGuttersTightenOnNarrowViewports(t *testing.T) {
	reset(t)
	parseShell := design.Class(design.ConsoleShell())
	parseRail := design.Class(design.ConsoleRail())
	parseColumn := design.Class(design.ContentColumn())
	parseCSS := css.Harvest()

	if !strings.Contains(parseCSS, "@media (max-width:960px){."+parseShell+"{grid-template-columns:minmax(0,1fr);}") {
		t.Errorf("console shell does not collapse to one column:\n%s", parseCSS)
	}
	if !strings.Contains(parseCSS, "@media (max-width:960px){."+parseRail+"{") {
		t.Error("console rail has no narrow-mode rules")
	}
	if !strings.Contains(parseCSS, "@media (max-width:520px){."+parseColumn+"{") {
		t.Error("content column gutters do not tighten below 520px")
	}
}

// TestLargeTypeStepsAreFluid pins the mechanism that makes a page title survive a
// 380px viewport without any media query: clamp() against vw.
func TestLargeTypeStepsAreFluid(t *testing.T) {
	reset(t)
	design.Class(design.PageTitle())
	if !strings.Contains(css.Harvest(), "font-size:clamp(1.5rem,4.6vw,1.875rem)") {
		t.Errorf("page title is not fluid:\n%s", css.Harvest())
	}

	reset(t)
	design.Class(design.Display(design.StepBanner))
	if !strings.Contains(css.Harvest(), "font-size:clamp(2rem,7vw,2.75rem)") {
		t.Errorf("banner step is not fluid:\n%s", css.Harvest())
	}
}

// --- the type roles -----------------------------------------------------------

// TestTypeRolesBindTheRightFamily is the mechanical half of "every machine fact is
// mono". If Data ever stops resolving to the mono stack, the information architecture
// described in the package doc quietly stops existing.
func TestTypeRolesBindTheRightFamily(t *testing.T) {
	for _, parseCase := range []struct {
		name   string
		bundle []css.Rule
		family string
	}{
		{"Display", design.Display(design.StepHead), "var(--atlas-font-display)"},
		{"Prose", design.Prose(design.StepBase), "var(--atlas-font-body)"},
		{"Data", design.Data(design.StepFine), "var(--atlas-font-data)"},
	} {
		reset(t)
		design.Class(parseCase.bundle)
		parseCSS := css.Harvest()
		if !strings.Contains(parseCSS, "font-family:"+parseCase.family) {
			t.Errorf("%s does not bind %s:\n%s", parseCase.name, parseCase.family, parseCSS)
		}
	}

	// Data must be tabular, or a column of quantities does not align — which is the
	// functional reason mono was chosen at all.
	reset(t)
	design.Class(design.Data(design.StepFine))
	if !strings.Contains(css.Harvest(), "font-variant-numeric:tabular-nums") {
		t.Error("Data role is not tabular")
	}

	// Display must be uppercase; that is what makes it a label rather than a heading.
	reset(t)
	design.Class(design.Display(design.StepMicro))
	if !strings.Contains(css.Harvest(), "text-transform:uppercase") {
		t.Error("Display role is not uppercase")
	}
}

func TestOutOfRangeStepsAndTonesDegradeSafely(t *testing.T) {
	reset(t)
	if parseGot, parseWant := design.Class(design.Display(design.TypeStep(99))), design.Class(design.Display(design.StepBase)); parseGot != parseWant {
		t.Errorf("out-of-range TypeStep should clamp to StepBase, got %q want %q", parseGot, parseWant)
	}
	if parseGot, parseWant := design.Class(design.StatusChip(design.StatusTone(-3))), design.Class(design.StatusChip(design.ToneNeutral)); parseGot != parseWant {
		t.Errorf("out-of-range StatusTone should clamp to ToneNeutral, got %q want %q", parseGot, parseWant)
	}
}

// --- status -------------------------------------------------------------------

// TestStatusTonesMapToTheRightSemanticToken pins the semantic contract: each tone
// resolves to its own token, and only ToneException is filled.
func TestStatusTonesMapToTheRightSemanticToken(t *testing.T) {
	for _, parseCase := range []struct {
		tone  design.StatusTone
		token string
	}{
		{design.ToneNeutral, "var(--atlas-graphite)"},
		{design.TonePending, "var(--atlas-lane)"},
		{design.ToneVerified, "var(--atlas-verify)"},
		{design.ToneException, "var(--atlas-oxide)"},
	} {
		reset(t)
		design.Class(design.StatusChip(parseCase.tone))
		parseCSS := css.Harvest()
		if !strings.Contains(parseCSS, parseCase.token) {
			t.Errorf("StatusChip(%s) does not use %s:\n%s", parseCase.tone, parseCase.token, parseCSS)
		}
	}

	// Only exceptions are filled. That asymmetry IS the hierarchy: in a 40-row queue
	// the exceptions are solid blocks and everything else is a quiet outline.
	reset(t)
	design.Class(design.StatusChip(design.ToneException))
	if !strings.Contains(css.Harvest(), "background-color:var(--atlas-oxide)") {
		t.Error("ToneException chip must be filled")
	}
	for _, parseTone := range []design.StatusTone{design.ToneNeutral, design.TonePending, design.ToneVerified} {
		reset(t)
		design.Class(design.StatusChip(parseTone))
		if !strings.Contains(css.Harvest(), "background-color:transparent") {
			t.Errorf("StatusChip(%s) must be outlined, not filled", parseTone)
		}
	}
}

func TestPostureMapsToTone(t *testing.T) {
	for _, parseCase := range []struct {
		posture design.Posture
		tone    design.StatusTone
		label   string
	}{
		{design.PostureOnLane, design.TonePending, "ON LANE"},
		{design.PostureHeld, design.TonePending, "HELD"},
		{design.PostureShort, design.ToneException, "SHORT"},
		{design.PostureClosed, design.ToneVerified, "CLOSED"},
	} {
		if parseGot := parseCase.posture.Tone(); parseGot != parseCase.tone {
			t.Errorf("%s.Tone() = %v, want %v", parseCase.label, parseGot, parseCase.tone)
		}
		if parseGot := parseCase.posture.Label(); parseGot != parseCase.label {
			t.Errorf("Label() = %q, want %q", parseGot, parseCase.label)
		}
	}
}

// --- table --------------------------------------------------------------------

func TestTableEmitsDenseHairlineManifestRules(t *testing.T) {
	reset(t)
	parseClass := design.Class(design.Table())
	parseCSS := css.Harvest()

	for _, parseWant := range []string{
		// Mono cells by default: a table in Atlas is a manifest.
		"font-family:var(--atlas-font-data)",
		// Header closed by the heavy ink rule; body rows by hairlines.
		"." + parseClass + " thead th{",
		"border-bottom:2px solid var(--atlas-ink)",
		"." + parseClass + " tbody td{",
		"border-bottom:1px solid var(--atlas-hairline)",
		// Zebra from paper-sunk, not from a translucent tint.
		"." + parseClass + " tbody tr:nth-child(odd){background-color:var(--atlas-paper-sunk);}",
		// Sticky header, so a 200-row queue keeps its column meanings.
		"position:sticky",
	} {
		if !strings.Contains(parseCSS, parseWant) {
			t.Errorf("table CSS missing %q:\n%s", parseWant, parseCSS)
		}
	}
}

// TestRowHoverOutranksZebra is the regression test for the cascade trap documented in
// table.go: within one folded class, blocks are emitted in SORTED selector order, so
// ":hover" is written before ":nth-child" and source order cannot break the tie. Hover
// has to win on specificity, which is why it targets the cells of a hovered row.
func TestRowHoverOutranksZebra(t *testing.T) {
	reset(t)
	parseClass := design.Class(design.Table())
	parseCSS := css.Harvest()

	parseHover := "." + parseClass + " tbody tr:hover td{"
	parseZebra := "." + parseClass + " tbody tr:nth-child(odd){"
	parseHoverAt := strings.Index(parseCSS, parseHover)
	parseZebraAt := strings.Index(parseCSS, parseZebra)
	if parseHoverAt < 0 || parseZebraAt < 0 {
		t.Fatalf("expected both hover and zebra rules:\n%s", parseCSS)
	}
	// Documenting the trap in an assertion: hover really is emitted FIRST, so if it
	// relied on source order it would lose on every odd row.
	if parseHoverAt > parseZebraAt {
		t.Fatal("emission order changed; re-derive the specificity argument in table.go")
	}
	// The win is structural: tbody tr:hover td (0-1-3) beats tbody tr:nth-child (0-1-2).
	if !strings.Contains(parseCSS, parseHover+"background-color:var(--atlas-paper);}") {
		t.Error("row hover does not target the cells of the hovered row, so zebra will win")
	}
}

// TestCellModifiersOutrankTable pins the specificity-doubling trick. A cell modifier is
// a bare class on a <td>; the Table bundle styles cells through descendant selectors,
// which outrank a bare class. Doubling the class ("&&") puts the modifier at 0-2-0, so
// it wins on every cell — including a <th>, where the table sets text-align:left and
// NumericCell has to override it.
func TestCellModifiersOutrankTable(t *testing.T) {
	for parseName, parseBundle := range map[string][]css.Rule{
		"NumericCell": design.NumericCell(),
		"ProseCell":   design.ProseCell(),
		"CellMeta":    design.CellMeta(),
	} {
		reset(t)
		parseClass := design.Class(parseBundle)
		parseCSS := css.Harvest()
		parseDoubled := "." + parseClass + "." + parseClass + "{"
		if !strings.Contains(parseCSS, parseDoubled) {
			t.Errorf("%s is not specificity-doubled; it will lose to the table's descendant rules:\n%s", parseName, parseCSS)
		}
	}

	// And the property that actually needs to win does: NumericCell right-aligns even
	// though `& tbody th` sets text-align:left.
	reset(t)
	design.Class(design.NumericCell())
	if !strings.Contains(css.Harvest(), "text-align:right") {
		t.Error("NumericCell does not right-align")
	}
}

// --- the signature element ----------------------------------------------------

func TestLanePlacardRendersTheDockTag(t *testing.T) {
	reset(t)
	design.Install()

	parseHTML := renderOnce(t, design.LanePlacard(design.PlacardSpec{
		OriginHub: "NJ-HUB",
		DestHub:   "IL-HUB",
		Promise:   "2026-08-04",
		LaneID:    "LN-4471",
		Posture:   design.PostureShort,
	}))

	for _, parseWant := range []string{
		"NJ-HUB", "IL-HUB", "2026-08-04", "PROMISE", "LANE", "LN-4471", "SHORT",
		`role="group"`,
		// The graphic arrow must not be announced; the label carries the direction.
		`aria-hidden="true"`,
		// The spelled-out sentence, because DOM order alone reads as
		// "NJ-HUB IL-HUB PROMISE …" and loses the direction entirely.
		`aria-label="Lane NJ-HUB to IL-HUB, promise 2026-08-04, lane id LN-4471, posture SHORT"`,
	} {
		if !strings.Contains(parseHTML, parseWant) {
			t.Errorf("placard HTML missing %q:\n%s", parseWant, parseHTML)
		}
	}

	parseCSS := css.Harvest()
	for _, parseWant := range []string{
		// Inked bar via the derived placard tokens, not via ink/paper directly.
		"background-color:var(--atlas-placard-bg)",
		"color:var(--atlas-placard-fg)",
		// The punched hole and the perforated tear edge.
		"::before{",
		"::after{",
		// The tear edge. Pinned as an exact string because the FORM matters, not just the
		// presence of a gradient: the original `repeating-radial-gradient(... 0 3.5px,
		// transparent 3.5px)` repeats its stop list outward and fills the tile solid, so
		// the placard shipped with a flat bottom edge and dead CSS. A non-repeating
		// gradient with an explicit `circle 3.5px` ending shape is what tiles one disc.
		"radial-gradient(circle 3.5px at 50% 100%, var(--atlas-paper) 0 96%, transparent 100%)",
		// Hub codes are machine facts, so mono.
		"font-family:var(--atlas-font-data)",
	} {
		if !strings.Contains(parseCSS, parseWant) {
			t.Errorf("placard CSS missing %q:\n%s", parseWant, parseCSS)
		}
	}
}

func TestLanePlacardOmitsEmptyFields(t *testing.T) {
	reset(t)
	parseHTML := renderOnce(t, design.LanePlacard(design.PlacardSpec{
		OriginHub: "TX-HUB",
		DestHub:   "IL-HUB",
		Posture:   design.PostureOnLane,
	}))
	if strings.Contains(parseHTML, "PROMISE") || strings.Contains(parseHTML, ">LANE<") {
		t.Errorf("empty placard fields should be omitted, not rendered blank:\n%s", parseHTML)
	}
	if !strings.Contains(parseHTML, `aria-label="Lane TX-HUB to IL-HUB, posture ON LANE"`) {
		t.Errorf("accessible label should skip absent fields:\n%s", parseHTML)
	}
}

func TestPlacardLabelOverrideWins(t *testing.T) {
	reset(t)
	parseHTML := renderOnce(t, design.LanePlacard(design.PlacardSpec{
		OriginHub: "NJ-HUB", DestHub: "IL-HUB", Label: "Backhaul, no promise date",
	}))
	if !strings.Contains(parseHTML, `aria-label="Backhaul, no promise date"`) {
		t.Errorf("Label override not honored:\n%s", parseHTML)
	}
}

// --- SSR shape ----------------------------------------------------------------

// TestStyleBlockIsSSRReady proves the emitted CSS actually reaches a document: the
// native sink serializes into a <style data-gwc-css> block whose data attribute lists
// the class names, which is what the wasm client reads back to suppress re-injection.
func TestStyleBlockIsSSRReady(t *testing.T) {
	reset(t)
	design.Install()
	parseClass := design.Class(design.Surface())

	parseBlock := css.StyleBlock()
	if !strings.HasPrefix(parseBlock, `<style data-gwc-css="`) {
		t.Fatalf("unexpected style block prefix: %.60s", parseBlock)
	}
	if !strings.Contains(parseBlock, parseClass) {
		t.Error("style block does not list the folded class name")
	}
	if !strings.Contains(parseBlock, "--atlas-paper:#F2F1ED") {
		t.Error("style block does not carry the token palette")
	}
	// A <style> element that could be terminated by its own content is an XSS vector;
	// the css package hardens every emission. Assert the property holds end to end.
	if strings.Contains(parseBlock[len(`<style data-gwc-css="`):], "</style>"+"") &&
		!strings.HasSuffix(parseBlock, "</style>") {
		t.Error("style block content is not breakout-safe")
	}
}

// --- helpers ------------------------------------------------------------------

// renderOnce serializes a node to HTML.
//
// Native SSR is single-flight per process: internal/runtime/reconciler.go holds
// currentFiber in package globals, so concurrent RenderToString calls trip
// GWC-RUNTIME-HOOK-THREADING. Every render in this suite therefore goes through here,
// sequentially, and no test in this file calls t.Parallel().
func renderOnce(t *testing.T, parseNode ui.Node) string {
	t.Helper()
	parseHTML, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		t.Fatalf("RenderToString: %v", parseErr)
	}
	return parseHTML
}

// blockFor returns the text of the first CSS block starting with parsePrefix.
func blockFor(t *testing.T, parseCSS string, parsePrefix string) string {
	t.Helper()
	parseIndex := strings.Index(parseCSS, parsePrefix)
	if parseIndex < 0 {
		t.Fatalf("no block found for %q in:\n%s", parsePrefix, parseCSS)
	}
	parseRest := parseCSS[parseIndex+len(parsePrefix):]
	parseEnd := strings.Index(parseRest, "}")
	if parseEnd < 0 {
		t.Fatalf("unterminated block for %q", parsePrefix)
	}
	return parseRest[:parseEnd]
}

// foldEveryPrimitive folds the entire exported surface and returns name -> class.
//
// Keeping one exhaustive list means the determinism, immutability, no-literal-color and
// reduced-motion tests all automatically cover any primitive added later — as long as
// it is added here, which is the one thing a reviewer has to check.
func foldEveryPrimitive() map[string]string {
	parseBundles := map[string][]css.Rule{
		"ConsoleShell":          design.ConsoleShell(),
		"ConsoleRail":           design.ConsoleRail(),
		"RailGroupLabel":        design.RailGroupLabel(),
		"RailLink":              design.RailLink(),
		"RailLinkCurrent":       design.RailLinkCurrent(),
		"RailPlate":             design.RailPlate(),
		"RailPlateCode":         design.RailPlateCode(),
		"RailCode":              design.RailCode(),
		"ContentColumn":         design.ContentColumn(),
		"StorefrontHeader":      design.StorefrontHeader(),
		"StorefrontHeaderInner": design.StorefrontHeaderInner(),
		"StorefrontMain":        design.StorefrontMain(),
		"PageHead":              design.PageHead(),
		"PageTitle":             design.PageTitle(),
		"SectionTitle":          design.SectionTitle(),
		"Eyebrow":               design.Eyebrow(),
		"Measure":               design.Measure(),
		"Surface":               design.Surface(),
		"SurfaceFlush":          design.SurfaceFlush(),
		"Recess":                design.Recess(),
		"Divider":               design.Divider(),
		"ManifestRule":          design.ManifestRule(),
		"Stack":                 design.Stack(design.Space4),
		"Cluster":               design.Cluster(design.Space3),
		"SplitRow":              design.SplitRow(design.Space4),
		"Table":                 design.Table(),
		"TableScroll":           design.TableScroll(),
		"NumericCell":           design.NumericCell(),
		"ProseCell":             design.ProseCell(),
		"CellMeta":              design.CellMeta(),
		"ButtonPrimary":         design.ButtonPrimary(),
		"ButtonSecondary":       design.ButtonSecondary(),
		"ButtonQuiet":           design.ButtonQuiet(),
		"Link":                  design.Link(),
		"Field":                 design.Field(),
		"FieldLabel":            design.FieldLabel(),
		"Input":                 design.Input(),
		"InputData":             design.InputData(),
		"FieldHint":             design.FieldHint(),
		"FieldError":            design.FieldError(),
		"PlacardBar":            design.PlacardBar(),
		"StatusChipNeutral":     design.StatusChip(design.ToneNeutral),
		"StatusChipPending":     design.StatusChip(design.TonePending),
		"StatusChipVerified":    design.StatusChip(design.ToneVerified),
		"StatusChipException":   design.StatusChip(design.ToneException),
		"StatusValueNeutral":    design.StatusValue(design.ToneNeutral),
		"StatusValuePending":    design.StatusValue(design.TonePending),
		"StatusValueVerified":   design.StatusValue(design.ToneVerified),
		"StatusValueException":  design.StatusValue(design.ToneException),

		// The catalog manifest — the storefront's answer to "no Card". Every cell bundle
		// is listed so the determinism, immutability, no-literal-color and
		// reduced-motion guarantees cover the newest and largest primitive too.
		"CatalogManifest":     design.CatalogManifest(),
		"CatalogHeaderStrip":  design.CatalogHeaderStrip(),
		"CatalogRow":          design.CatalogRow(),
		"CatalogThumb":        design.CatalogThumb(),
		"CatalogThumbPlate":   design.CatalogThumbPlate(),
		"CatalogIdentity":     design.CatalogIdentity(),
		"CatalogTitle":        design.CatalogTitle(),
		"CatalogIdentityMeta": design.CatalogIdentityMeta(),
		"CatalogSummary":      design.CatalogSummary(),
		"CatalogAvailability": design.CatalogAvailability(),
		"CatalogPromise":      design.CatalogPromise(),
		"CatalogPrice":        design.CatalogPrice(),
		"CatalogActionLabel":  design.CatalogActionLabel(),
		"CatalogResultNote":   design.CatalogResultNote(),
		"CatalogEmpty":        design.CatalogEmpty(),
		"CatalogEmptyBody":    design.CatalogEmptyBody(),

		// Print/screen visibility.
		"ScreenOnly": design.ScreenOnly(),
		"PrintOnly":  design.PrintOnly(),
	}
	for parseIndex := design.StepMicro; parseIndex <= design.StepBanner; parseIndex++ {
		parseBundles["Display"+parseIndex.String()] = design.Display(parseIndex)
		parseBundles["Prose"+parseIndex.String()] = design.Prose(parseIndex)
		parseBundles["Data"+parseIndex.String()] = design.Data(parseIndex)
	}

	// Fold in sorted name order. Class names are content hashes and are therefore
	// order-independent, but the SINK records emission order, so folding in Go's
	// randomized map-iteration order would make css.Harvest() differ run to run and
	// the determinism test would fail on the harness rather than on the package.
	parseNames := make([]string, 0, len(parseBundles))
	for parseName := range parseBundles {
		parseNames = append(parseNames, parseName)
	}
	sort.Strings(parseNames)

	parseFolds := make(map[string]string, len(parseBundles))
	for _, parseName := range parseNames {
		parseFolds[parseName] = design.Class(parseBundles[parseName])
	}
	return parseFolds
}
