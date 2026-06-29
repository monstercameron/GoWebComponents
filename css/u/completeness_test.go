//go:build !js || !wasm

package u_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/css"
	"github.com/monstercameron/GoWebComponents/v4/css/u"
)

// TestUtilityCompletenessMatrix proves every utility CATEGORY the engine claims to cover
// actually folds to a real class and emits non-empty CSS — the completeness matrix the audit
// asked for (B4). Each row exercises one category through the css registry end-to-end, so a
// regression that silently drops a category (emitting nothing) fails here.
func TestUtilityCompletenessMatrix(parseT *testing.T) {
	parseMatrix := []struct {
		category string
		build    func() css.Sheet
	}{
		{"display", func() css.Sheet { return css.New(u.Flex) }},
		{"flex-align", func() css.Sheet { return css.New(u.ItemsCenter, u.JustifyBetween) }},
		{"spacing-padding", func() css.Sheet { return css.New(u.Pad(u.Spacing4)) }},
		{"spacing-margin", func() css.Sheet { return css.New(u.M(u.Spacing2)) }},
		{"spacing-gap", func() css.Sheet { return css.New(u.Gap(u.Spacing4)) }},
		{"sizing", func() css.Sheet { return css.New(u.W(u.Spacing8), u.H(u.Spacing8)) }},
		{"color", func() css.Sheet { return css.New(u.BgToken("primary"), u.TextToken("fg")) }},
		{"typography", func() css.Sheet { return css.New(u.TextSize(u.TextBase)) }},
		{"radius", func() css.Sheet { return css.New(u.Rounded(u.RadiusMd)) }},
		{"effects-opacity", func() css.Sheet { return css.New(u.Opacity(50)) }},
		{"variant-hover", func() css.Sheet { return css.New(u.Hover(u.Opacity(80))...) }},
		{"variant-focus", func() css.Sheet { return css.New(u.Focus(u.Opacity(90))...) }},
		{"variant-dark", func() css.Sheet { return css.New(u.Dark(u.Opacity(70))...) }},
		{"responsive-sm", func() css.Sheet { return css.New(u.Sm(u.Opacity(60))...) }},
		{"responsive-lg", func() css.Sheet { return css.New(u.Lg(u.Opacity(40))...) }},
		{"important", func() css.Sheet { return css.New(u.Important(u.Opacity(30))...) }},
	}

	for _, parseRow := range parseMatrix {
		parseT.Run(parseRow.category, func(parseT *testing.T) {
			css.Reset()
			parseSheet := parseRow.build()
			if parseSheet.String() == "" {
				parseT.Fatalf("category %q produced no class", parseRow.category)
			}
			parseCSS := css.Harvest()
			if strings.TrimSpace(parseCSS) == "" {
				parseT.Fatalf("category %q emitted no CSS", parseRow.category)
			}
			if !strings.Contains(parseCSS, parseSheet.String()) {
				parseT.Fatalf("category %q: harvested CSS does not contain its class %q", parseRow.category, parseSheet.String())
			}
		})
	}
}

// TestRegistryNoDoubleEmit proves the registry emits a given class exactly once no matter how
// many times the same utility set is folded — the dedup guarantee that keeps the SSR stylesheet
// from ballooning when a style is reused across a render loop (B4).
func TestRegistryNoDoubleEmit(parseT *testing.T) {
	css.Reset()

	parseFirst := css.New(u.Pad(u.Spacing4))
	parseSecond := css.New(u.Pad(u.Spacing4)) // identical set, folded again
	if parseFirst.String() != parseSecond.String() {
		parseT.Fatalf("identical utility sets should fold to the same class, got %q vs %q", parseFirst, parseSecond)
	}

	parseClasses := css.HarvestedClasses()
	parseCount := 0
	for _, parseClass := range parseClasses {
		if parseClass == parseFirst.String() {
			parseCount++
		}
	}
	if parseCount != 1 {
		parseT.Fatalf("class %q should be emitted exactly once, emitted %d times (classes: %v)", parseFirst, parseCount, parseClasses)
	}
}
