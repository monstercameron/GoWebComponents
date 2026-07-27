//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/css"
)

// declBody harvests the emitted CSS for a freshly folded rule-set and returns the
// declaration body of its single block, so a test can assert on the WHOLE
// declaration rather than a substring of it. Substring assertions are exactly what
// let the multi-property Transition bug survive: "contains 120ms" was true of the
// broken output too.
func declBody(parseT *testing.T, parseRules ...css.Rule) string {
	parseT.Helper()
	css.Reset()
	parseClass := css.New(parseRules...)
	if parseClass == "" {
		parseT.Fatal("rule-set folded to no class")
	}
	parseCSS := css.Harvest()
	parseOpen := "." + string(parseClass) + "{"
	parseStart := strings.Index(parseCSS, parseOpen)
	if parseStart < 0 {
		parseT.Fatalf("class %q not found in harvested CSS:\n%s", parseClass, parseCSS)
	}
	parseRest := parseCSS[parseStart+len(parseOpen):]
	parseEnd := strings.Index(parseRest, "}")
	if parseEnd < 0 {
		parseT.Fatalf("unterminated block for %q:\n%s", parseClass, parseCSS)
	}
	return parseRest[:parseEnd]
}

// --- Bug 1: the transition shorthand's comma is per-transition ------------------

// TestTransitionSinglePropertyIsUnchanged pins the single-property shorthand
// byte-for-byte. It is the compatibility half of the multi-property fix: the
// expansion must not perturb the single-property serialization, because that would
// change every existing transition class hash.
func TestTransitionSinglePropertyIsUnchanged(parseT *testing.T) {
	parseCases := []struct {
		rule css.Rule
		want string
	}{
		{css.Transition(css.PropOpacity, css.Ms(120), css.Ease), "transition:opacity 120ms ease;"},
		{css.Transition(css.PropAll, css.Ms(120), css.Ease), "transition:all 120ms ease;"},
		{css.Transition(css.PropTransform, css.S(0.2), css.EaseOut), "transition:transform 0.2s ease-out;"},
		{
			css.Transition(css.PropShadow, css.Ms(80), css.CubicBezier(0.4, 0, 0.2, 1)),
			"transition:box-shadow 80ms cubic-bezier(0.4,0,0.2,1);",
		},
	}
	for _, parseCase := range parseCases {
		if parseGot := declBody(parseT, parseCase.rule); parseGot != parseCase.want {
			parseT.Errorf("single-property transition:\n got %q\nwant %q", parseGot, parseCase.want)
		}
	}
}

// TestTransitionMultiPropertyExpandsPerTransition is the regression test for the
// PropColors bug. The `transition` shorthand is a comma-separated list of whole
// transitions, so the old concatenation
//
//	transition: color, background-color, border-color, fill, stroke 120ms ease
//
// declared four transitions at the default 0s plus one 120ms transition on `stroke`
// — the preset animated nothing visible. The fixed form distributes the timing over
// every property. The assertion is on the FULL declaration, not a substring.
func TestTransitionMultiPropertyExpandsPerTransition(parseT *testing.T) {
	parseGot := declBody(parseT, css.Transition(css.PropColors, css.Ms(120), css.Ease))
	parseWant := "transition:" +
		"color 120ms ease," +
		"background-color 120ms ease," +
		"border-color 120ms ease," +
		"fill 120ms ease," +
		"stroke 120ms ease;"
	if parseGot != parseWant {
		parseT.Fatalf("multi-property transition:\n got %q\nwant %q", parseGot, parseWant)
	}
	// Belt and braces: the broken shape must be gone, and every property must carry
	// its own duration (five of them, not one).
	if strings.Contains(parseGot, "color, background-color") {
		parseT.Errorf("bare property list survived: %q", parseGot)
	}
	if parseN := strings.Count(parseGot, "120ms"); parseN != 5 {
		parseT.Errorf("expected 5 durations (one per property), got %d in %q", parseN, parseGot)
	}
	if parseN := strings.Count(parseGot, "ease"); parseN != 5 {
		parseT.Errorf("expected 5 easings (one per property), got %d in %q", parseN, parseGot)
	}
}

// TestTransitionMultiPropertyPreservesFunctionCommas proves the expansion splits on
// the LIST commas only. A cubic-bezier easing is full of commas; splitting on those
// would shred the easing into fragments and invalidate the declaration.
func TestTransitionMultiPropertyPreservesFunctionCommas(parseT *testing.T) {
	parseEase := css.CubicBezier(0.2, 0, 0.2, 1)
	parseGot := declBody(parseT, css.Transition(
		css.TransitionProps(css.PropOpacity, css.PropTransform),
		css.Ms(160), parseEase,
	))
	parseWant := "transition:opacity 160ms cubic-bezier(0.2,0,0.2,1)," +
		"transform 160ms cubic-bezier(0.2,0,0.2,1);"
	if parseGot != parseWant {
		parseT.Fatalf("TransitionProps expansion:\n got %q\nwant %q", parseGot, parseWant)
	}
}

// TestTransitionPropDoesNotDoubleSplit pins that a Prop() name containing a comma
// (someone hand-rolling a list) expands the same way the presets do.
func TestTransitionPropListExpands(parseT *testing.T) {
	parseGot := declBody(parseT, css.Transition(css.Prop("width, height"), css.Ms(50), css.Linear))
	parseWant := "transition:width 50ms linear,height 50ms linear;"
	if parseGot != parseWant {
		parseT.Fatalf("Prop list expansion:\n got %q\nwant %q", parseGot, parseWant)
	}
}

// TestTransitionLonghandsEmitsThreeDeclarations pins the longhand alternative, which
// is what a reduced-motion override needs: TransitionDuration can then replace only
// the duration, which is impossible against the packed shorthand value.
func TestTransitionLonghandsEmitsThreeDeclarations(parseT *testing.T) {
	parseGot := declBody(parseT, css.TransitionLonghands(css.PropColors, css.Ms(120), css.Ease))
	parseWant := "transition-duration:120ms;" +
		"transition-property:color, background-color, border-color, fill, stroke;" +
		"transition-timing-function:ease;"
	if parseGot != parseWant {
		parseT.Fatalf("longhand transition:\n got %q\nwant %q", parseGot, parseWant)
	}
}

// TestReducedMotionOverrideBeatsLonghandDuration proves the reduced-motion pattern
// works without !important: the unconditional block and the @media block are separate
// scopes, and canonicalize sorts at-rules after the empty one, so the media block
// lands last and wins at equal specificity.
func TestReducedMotionOverrideBeatsLonghandDuration(parseT *testing.T) {
	css.Reset()
	parseClass := css.New(css.Rules(
		css.TransitionLonghands(css.PropColors, css.Ms(120), css.Ease),
		css.Media(css.ReducedMotion, css.TransitionDuration(css.RawDuration("0.01ms"))),
	)...)
	parseCSS := css.Harvest()
	if !strings.Contains(parseCSS, "@media (prefers-reduced-motion:reduce){."+string(parseClass)+"{transition-duration:0.01ms;}}") {
		parseT.Fatalf("reduced-motion override not emitted as expected:\n%s", parseCSS)
	}
	if strings.Index(parseCSS, "@media") < strings.Index(parseCSS, "transition-property") {
		parseT.Fatalf("media block must be emitted after the unconditional block:\n%s", parseCSS)
	}
}

// --- shadows (the RawShadow doc reference, now real) ---------------------------

func TestShadowConstructorsAndRawShadow(parseT *testing.T) {
	if parseGot := declBody(parseT, css.Shadow(css.RawShadow("0 0 0 3px currentColor"))); parseGot != "box-shadow:0 0 0 3px currentColor;" {
		parseT.Errorf("RawShadow: got %q", parseGot)
	}
	if parseGot := declBody(parseT, css.Shadow(css.ShadowOf(css.Zero, css.Px(1), css.Px(2), css.Zero, css.RGBA(0, 0, 0, 0.05)))); parseGot != "box-shadow:0 1px 2px 0 rgba(0,0,0,0.05);" {
		parseT.Errorf("ShadowOf: got %q", parseGot)
	}
	// box-shadow's comma list is a list of complete values, so a comma join is the
	// whole grammar here — no per-item timing to distribute (contrast Transition).
	if parseGot := declBody(parseT, css.Shadow(css.Shadows(css.ShadowSm, css.ShadowInset(css.Zero, css.Px(1), css.Zero, css.Zero, css.White)))); parseGot != "box-shadow:0 1px 2px 0 rgba(0,0,0,0.05),inset 0 1px 0 0 #ffffff;" {
		parseT.Errorf("Shadows: got %q", parseGot)
	}
	// A "none" among real shadows would invalidate the whole declaration.
	if parseGot := declBody(parseT, css.Shadow(css.Shadows(css.ShadowNone, css.ShadowSm))); parseGot != "box-shadow:0 1px 2px 0 rgba(0,0,0,0.05);" {
		parseT.Errorf("Shadows dropping ShadowNone: got %q", parseGot)
	}
}

// --- side-specific borders -----------------------------------------------------

func TestSideSpecificBorders(parseT *testing.T) {
	if parseGot := declBody(parseT, css.BorderBottom(css.Px(1), css.Slate200)); parseGot != "border-bottom:1px solid #e2e8f0;" {
		parseT.Errorf("BorderBottom: got %q", parseGot)
	}
	if parseGot := declBody(parseT, css.BorderX(css.Px(2), css.Black)); parseGot != "border-left:2px solid #000000;border-right:2px solid #000000;" {
		parseT.Errorf("BorderX: got %q", parseGot)
	}
	// Shorthand-then-longhand: "border-top" sorts before "border-top-style", so the
	// style longhand reliably overrides the side shorthand (documented in rule.go).
	parseGot := declBody(parseT,
		css.BorderTop(css.Px(1), css.Black),
		css.BorderTopStyle(css.LineDashed),
	)
	if parseGot != "border-top:1px solid #000000;border-top-style:dashed;" {
		parseT.Errorf("border shorthand/longhand order: got %q", parseGot)
	}
}

// --- grid ----------------------------------------------------------------------

func TestGridConstructors(parseT *testing.T) {
	// A track list is SPACE separated; a comma between tracks invalidates it.
	if parseGot := declBody(parseT, css.GridCols(css.TrackLen(css.Rem(14)), css.MinMax(css.TrackLen(css.Zero), css.Fr(1)))); parseGot != "grid-template-columns:14rem minmax(0,1fr);" {
		parseT.Errorf("GridCols: got %q", parseGot)
	}
	if parseGot := declBody(parseT, css.GridCols(css.RepeatFit(css.MinMax(css.TrackLen(css.Rem(16)), css.Fr(1))))); parseGot != "grid-template-columns:repeat(auto-fit,minmax(16rem,1fr));" {
		parseT.Errorf("RepeatFit: got %q", parseGot)
	}
	if parseGot := declBody(parseT, css.GridRows(css.Repeat(3, css.TrackAuto))); parseGot != "grid-template-rows:repeat(3,auto);" {
		parseT.Errorf("Repeat: got %q", parseGot)
	}
	if parseGot := declBody(parseT, css.GridAreas("rail head", "rail body")); parseGot != `grid-template-areas:"rail head" "rail body";` {
		parseT.Errorf("GridAreas: got %q", parseGot)
	}
	if parseGot := declBody(parseT, css.GridColumn(css.GridRange(css.GridLineAt(1), css.GridLineAt(-1)))); parseGot != "grid-column:1 / -1;" {
		parseT.Errorf("GridColumn range: got %q", parseGot)
	}
	if parseGot := declBody(parseT, css.GridRow(css.GridSpan(2))); parseGot != "grid-row:span 2;" {
		parseT.Errorf("GridRow span: got %q", parseGot)
	}
	if parseGot := declBody(parseT, css.AlignSelf.Start, css.JustifySelf.End); parseGot != "align-self:start;justify-self:end;" {
		parseT.Errorf("self alignment: got %q", parseGot)
	}
}

// TestGridAreasCannotBreakOutOfTheQuotedRow proves an area row cannot terminate its
// CSS string and inject a declaration — the same safety-by-construction property the
// other typed constructors have.
func TestGridAreasCannotBreakOutOfTheQuotedRow(parseT *testing.T) {
	parseGot := declBody(parseT, css.GridAreas(`a"; color:red; x:"b`))
	if strings.Contains(parseGot, "color:red") {
		parseT.Fatalf("GridAreas allowed an injected declaration: %q", parseGot)
	}
	if strings.Count(parseGot, `"`) != 2 {
		parseT.Fatalf("GridAreas emitted an unbalanced quoted row: %q", parseGot)
	}
}

// --- gradients / background ----------------------------------------------------

// TestPerforationDeviceIsFullyTyped rebuilds the exact background stack a torn-stub
// perforation needs — the case that was 100% css.Raw before these constructors.
func TestPerforationDeviceIsFullyTyped(parseT *testing.T) {
	parsePaper := css.Var("atlas-paper")
	parseGot := declBody(parseT,
		css.BgImage(css.RepeatingRadialGradient(
			css.CircleAt(css.Percent(50), css.Percent(100)),
			css.StopSpan(parsePaper, css.Zero, css.RawLength("3.5px")),
			css.StopAt(css.Transparent, css.RawLength("3.5px")),
		)),
		css.BgSize(css.BgSizeXY(css.Px(11), css.Px(7))),
		css.BgRepeat.RepeatX,
		css.PointerEvents.None,
	)
	parseWant := "background-image:repeating-radial-gradient(circle at 50% 100%," +
		"var(--atlas-paper) 0 3.5px,transparent 3.5px);" +
		"background-repeat:repeat-x;" +
		"background-size:11px 7px;" +
		"pointer-events:none;"
	if parseGot != parseWant {
		parseT.Fatalf("perforation device:\n got %q\nwant %q", parseGot, parseWant)
	}
}

func TestGradientConstructors(parseT *testing.T) {
	if parseGot := declBody(parseT, css.BgImage(css.LinearGradientTo(css.ToBottom, css.Stop(css.Black), css.Stop(css.Transparent)))); parseGot != "background-image:linear-gradient(to bottom,#000000,transparent);" {
		parseT.Errorf("LinearGradientTo: got %q", parseGot)
	}
	if parseGot := declBody(parseT, css.BgImage(css.LinearGradient(css.Deg(180), css.StopAt(css.White, css.Percent(40))))); parseGot != "background-image:linear-gradient(180deg,#ffffff 40%);" {
		parseT.Errorf("LinearGradient: got %q", parseGot)
	}
	// background-image is a genuine comma-separated LAYER list, so a comma join is
	// correct here (unlike the transition shorthand).
	if parseGot := declBody(parseT, css.BgImage(css.URLImage("/a.png"), css.NoImage)); parseGot != `background-image:url("/a.png"),none;` {
		parseT.Errorf("BgImage layers: got %q", parseGot)
	}
	// A path cannot close the url() string: the quote is escaped, so the injected
	// text stays inside the CSS string instead of becoming declarations.
	if parseGot := declBody(parseT, css.BgImage(css.URLImage(`x") ;color:red;background:url("y`))); parseGot != `background-image:url("x\") ;color:red;background:url(\"y");` {
		parseT.Errorf("URLImage escaping: got %q", parseGot)
	}
}

// TestMaskEmitsPrefixedAndStandard pins that the mask constructors emit the -webkit-
// longhand as well, and that canonicalize's property-name sort puts the standard
// property last so it wins wherever it is supported.
func TestMaskEmitsPrefixedAndStandard(parseT *testing.T) {
	parseGot := declBody(parseT,
		css.MaskImage(css.LinearGradientTo(css.ToBottom, css.Stop(css.Black), css.Stop(css.Transparent))),
		css.MaskRepeat.NoRepeat,
	)
	parseWant := "-webkit-mask-image:linear-gradient(to bottom,#000000,transparent);" +
		"-webkit-mask-repeat:no-repeat;" +
		"mask-image:linear-gradient(to bottom,#000000,transparent);" +
		"mask-repeat:no-repeat;"
	if parseGot != parseWant {
		parseT.Fatalf("mask:\n got %q\nwant %q", parseGot, parseWant)
	}
}

// --- inset / overflow / stacking ------------------------------------------------

func TestInsetAndBoxConstructors(parseT *testing.T) {
	parseGot := declBody(parseT,
		css.Position.Absolute,
		css.Left(css.Zero), css.Right(css.Zero), css.Bottom(css.Px(-1)),
		css.ZIndex(20),
	)
	parseWant := "bottom:-1px;left:0;position:absolute;right:0;z-index:20;"
	if parseGot != parseWant {
		parseT.Fatalf("inset offsets:\n got %q\nwant %q", parseGot, parseWant)
	}
	if parseGot := declBody(parseT, css.Inset(css.Zero)); parseGot != "inset:0;" {
		parseT.Errorf("Inset: got %q", parseGot)
	}
	if parseGot := declBody(parseT, css.InsetY(css.Px(4))); parseGot != "bottom:4px;top:4px;" {
		parseT.Errorf("InsetY: got %q", parseGot)
	}
	if parseGot := declBody(parseT, css.OverflowX.Auto, css.OverflowY.Hidden, css.OverscrollBehaviorX.Contain); parseGot != "overflow-x:auto;overflow-y:hidden;overscroll-behavior-x:contain;" {
		parseT.Errorf("overflow: got %q", parseGot)
	}
	if parseGot := declBody(parseT, css.Flex(css.Num(0), css.Num(0), css.Auto), css.FlexWrap.NoWrap); parseGot != "flex:0 0 auto;flex-wrap:nowrap;" {
		parseT.Errorf("flex: got %q", parseGot)
	}
	if parseGot := declBody(parseT, css.Appearance.None, css.ColorScheme.LightDark, css.BorderCollapse.Collapse, css.TableLayout.Fixed); parseGot != "appearance:none;border-collapse:collapse;color-scheme:light dark;table-layout:fixed;" {
		parseT.Errorf("misc namespaces: got %q", parseGot)
	}
}

// --- typography -----------------------------------------------------------------

func TestTypographyConstructors(parseT *testing.T) {
	if parseGot := declBody(parseT, css.Font(css.VarFontStack("atlas-sans"))); parseGot != "font-family:var(--atlas-sans);" {
		parseT.Errorf("Font/VarFontStack: got %q", parseGot)
	}
	if parseGot := string(css.FontStackOf("Atlas Grotesk", "system-ui", "sans-serif")); parseGot != "'Atlas Grotesk',system-ui,sans-serif" {
		parseT.Errorf("FontStackOf: got %q", parseGot)
	}
	// A family name is often profile/config driven; it must not be able to close the
	// quote or end the declaration.
	if parseGot := string(css.FontStackOf(`Evil'; color:red; font-family:'x`)); strings.Contains(parseGot, ";") {
		parseT.Errorf("FontStackOf allowed injection: got %q", parseGot)
	}
	parseGot := declBody(parseT,
		css.TextAlign.Right,
		css.WhiteSpace.NoWrap,
		css.TextWrap.Balance,
		css.VerticalAlign.Top,
		css.TextDecoration.Underline,
		css.TextDecorationThickness(css.Px(1)),
		css.TextUnderlineOffset(css.Ems(0.2)),
		css.FontVariantLigatures.None,
	)
	parseWant := "font-variant-ligatures:none;" +
		"text-align:right;" +
		"text-decoration-line:underline;" +
		"text-decoration-thickness:1px;" +
		"text-underline-offset:0.2em;" +
		"text-wrap:balance;" +
		"vertical-align:top;" +
		"white-space:nowrap;"
	if parseGot != parseWant {
		parseT.Fatalf("typography:\n got %q\nwant %q", parseGot, parseWant)
	}
}

// --- custom properties ----------------------------------------------------------

// TestCustomPropertyDeclarationAndReferenceAgree is the point of the Custom*
// constructors: the declaration and the css.Var reference normalize through the same
// helper, so they cannot drift apart on the "--" prefix.
func TestCustomPropertyDeclarationAndReferenceAgree(parseT *testing.T) {
	for _, parseName := range []string{"accent", "--accent", "-accent"} {
		parseGot := declBody(parseT, css.CustomColor(parseName, css.Hex("4f46e5")))
		if parseGot != "--accent:#4f46e5;" {
			parseT.Errorf("CustomColor(%q): got %q", parseName, parseGot)
		}
		if parseRef := string(css.Var(parseName)); parseRef != "var(--accent)" {
			parseT.Errorf("Var(%q): got %q, does not match the declaration", parseName, parseRef)
		}
	}
	parseGot := declBody(parseT,
		css.CustomLength("radius", css.Px(12)),
		css.CustomDuration("fast", css.Ms(120)),
		css.CustomNumber("lead", css.Num(1.4)),
		css.Custom("rule", "1px solid currentColor"),
	)
	parseWant := "--fast:120ms;--lead:1.4;--radius:12px;--rule:1px solid currentColor;"
	if parseGot != parseWant {
		parseT.Fatalf("custom properties:\n got %q\nwant %q", parseGot, parseWant)
	}
	// The NAME is sanitized even though the value is author-trusted.
	if parseGot := declBody(parseT, css.Custom("evil}body{display:none", "1px")); strings.Contains(parseGot, "}") {
		parseT.Fatalf("Custom allowed a name breakout: %q", parseGot)
	}
}

// --- accessibility media queries ------------------------------------------------

// TestAccessibilityMediaQueriesAreTypedConstants pins the exact query text. A typo in
// a RawMedia string produces a syntactically valid block that never matches, so these
// are the queries that most need to be unmisspellable.
func TestAccessibilityMediaQueriesAreTypedConstants(parseT *testing.T) {
	parseCases := map[css.MediaQuery]string{
		css.ReducedMotion: "(prefers-reduced-motion:reduce)",
		css.MotionOK:      "(prefers-reduced-motion:no-preference)",
		css.ContrastMore:  "(prefers-contrast:more)",
		css.ContrastLess:  "(prefers-contrast:less)",
		css.ForcedColors:  "(forced-colors:active)",
		css.Light:         "(prefers-color-scheme:light)",
	}
	for parseQuery, parseWant := range parseCases {
		if string(parseQuery) != parseWant {
			parseT.Errorf("media query: got %q want %q", parseQuery, parseWant)
		}
	}
	if parseGot := string(css.MediaAll(css.MinW(768), css.Dark)); parseGot != "(min-width:768px) and (prefers-color-scheme:dark)" {
		parseT.Errorf("MediaAll: got %q", parseGot)
	}
	css.Reset()
	parseClass := css.New(css.Media(css.ForcedColors, css.BorderStyle(css.LineSolid))...)
	if !strings.Contains(css.Harvest(), "@media (forced-colors:active){."+string(parseClass)+"{border-style:solid;}}") {
		parseT.Fatalf("forced-colors block not emitted:\n%s", css.Harvest())
	}
}

// --- new value constructors -----------------------------------------------------

func TestLengthAndColorValueAdditions(parseT *testing.T) {
	parseCases := []struct{ got, want string }{
		{string(css.Ch(68)), "68ch"},
		{string(css.Clamp(css.Rem(1), css.Vw(2.5), css.Rem(1.5))), "clamp(1rem,2.5vw,1.5rem)"},
		{string(css.MinLen(css.Full, css.Rem(40))), "min(100%,40rem)"},
		{string(css.MaxLen(css.Rem(40))), "40rem"}, // single value needs no wrapper
		{string(css.ColorMix(css.Sky500, css.White, 20)), "color-mix(in oklab,#0ea5e9 20%,#ffffff)"},
		{string(css.ColorMix(css.Sky500, css.White, 400)), "color-mix(in oklab,#0ea5e9 100%,#ffffff)"},
		{string(css.ColorMix(css.Sky500, css.White, -5)), "color-mix(in oklab,#0ea5e9 0%,#ffffff)"},
	}
	for _, parseCase := range parseCases {
		if parseCase.got != parseCase.want {
			parseT.Errorf("value: got %q want %q", parseCase.got, parseCase.want)
		}
	}
}

// --- documented (not fixed) canonicalize behaviours -----------------------------

// TestFoldSortsBlocksByAtRuleAndSelector confirms behaviour #1 documented on
// canonicalize: blocks are ordered by (at-rule, selector), so source order cannot
// break a specificity tie inside one folded class. This is REQUIRED — New(a,b) and
// New(b,a) must mint the same class — so it is pinned, not fixed.
func TestFoldSortsBlocksByAtRuleAndSelector(parseT *testing.T) {
	css.Reset()
	parseForward := css.New(css.Rules(
		css.Hover(css.Bg(css.Black)),
		css.Focus(css.Bg(css.White)),
	)...)
	parseCSSForward := css.Harvest()

	css.Reset()
	parseReverse := css.New(css.Rules(
		css.Focus(css.Bg(css.White)),
		css.Hover(css.Bg(css.Black)),
	)...)
	parseCSSReverse := css.Harvest()

	if parseForward != parseReverse {
		parseT.Fatalf("reordering rules changed the class: %q vs %q", parseForward, parseReverse)
	}
	if parseCSSForward != parseCSSReverse {
		parseT.Fatalf("reordering rules changed the emitted CSS:\n%s\nvs\n%s", parseCSSForward, parseCSSReverse)
	}
	// ":focus" sorts before ":hover", regardless of authoring order.
	if strings.Index(parseCSSForward, ":focus") > strings.Index(parseCSSForward, ":hover") {
		parseT.Fatalf("blocks are not selector-sorted:\n%s", parseCSSForward)
	}
}

// TestFoldSortsDeclarationsByPropertyName confirms behaviour #2 documented on
// canonicalize: declarations sort by property name inside a block. Shorthand-then-
// longhand therefore works by construction, but border-bottom can never beat
// border-top within one block — authoring order is irrelevant either way.
func TestFoldSortsDeclarationsByPropertyName(parseT *testing.T) {
	// Shorthand before its own longhand: "border" < "border-color".
	if parseGot := declBody(parseT,
		css.BorderColor(css.Red500),
		css.Border(css.Px(1), css.Black),
	); parseGot != "border:1px solid #000000;border-color:#ef4444;" {
		parseT.Fatalf("shorthand/longhand order: got %q", parseGot)
	}
	// Sibling longhands: border-bottom is emitted first whichever order it is
	// written in, so it cannot override border-top here. Documented, not fixed —
	// use one declaration per side, or separate scopes.
	parseBottomFirst := declBody(parseT,
		css.BorderBottom(css.Px(2), css.Red500),
		css.BorderTop(css.Px(1), css.Black),
	)
	parseTopFirst := declBody(parseT,
		css.BorderTop(css.Px(1), css.Black),
		css.BorderBottom(css.Px(2), css.Red500),
	)
	if parseBottomFirst != parseTopFirst {
		parseT.Fatalf("declaration order leaked authoring order: %q vs %q", parseBottomFirst, parseTopFirst)
	}
	if parseTopFirst != "border-bottom:2px solid #ef4444;border-top:1px solid #000000;" {
		parseT.Fatalf("unexpected declaration order: %q", parseTopFirst)
	}
}
