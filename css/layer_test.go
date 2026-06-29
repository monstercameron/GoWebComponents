//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/css"
)

// TestLayerWrapsHashedClass proves Layer emits the folded class inside an
// @layer block (CSS4).
func TestLayerWrapsHashedClass(parseT *testing.T) {
	css.Reset()
	parseSheet := css.Layer("components", css.Raw("display", "flex"))
	parseOut := css.Harvest()
	parseExpected := "@layer components{." + string(parseSheet) + "{display:flex"
	if !strings.Contains(parseOut, parseExpected) {
		parseT.Fatalf("expected %q in %q", parseExpected, parseOut)
	}
}

// TestLayerEmptyNameFallsBackToNew proves Layer("") behaves like New (no @layer).
func TestLayerEmptyNameFallsBackToNew(parseT *testing.T) {
	css.Reset()
	css.Layer("", css.Raw("color", "red"))
	parseOut := css.Harvest()
	if strings.Contains(parseOut, "@layer") {
		parseT.Fatalf("empty layer name must not emit @layer, got %q", parseOut)
	}
	if !strings.Contains(parseOut, "color:red") {
		parseT.Fatalf("expected the rule to still emit, got %q", parseOut)
	}
}

// TestDeclareLayersEmitsOrderStatement proves DeclareLayers emits an ordering
// statement.
func TestDeclareLayersEmitsOrderStatement(parseT *testing.T) {
	css.Reset()
	css.DeclareLayers("base", "components", "overrides")
	parseOut := css.Harvest()
	if !strings.Contains(parseOut, "@layer base,components,overrides;") {
		parseT.Fatalf("expected layer order statement, got %q", parseOut)
	}
}

// TestDeclareLayersDedupes proves an identical declaration emits once.
func TestDeclareLayersDedupes(parseT *testing.T) {
	css.Reset()
	css.DeclareLayers("a", "b")
	parseFirst := len(css.HarvestedClasses())
	css.DeclareLayers("a", "b")
	if parseSecond := len(css.HarvestedClasses()); parseSecond != parseFirst {
		parseT.Fatalf("duplicate DeclareLayers must not re-emit, %d -> %d", parseFirst, parseSecond)
	}
}

// TestLayerGlobalEmitsGlobalRuleInLayer proves LayerGlobal wraps a global
// selector in @layer — the typed override-layer use case.
func TestLayerGlobalEmitsGlobalRuleInLayer(parseT *testing.T) {
	css.Reset()
	css.LayerGlobal("overrides", `[data-theme="light"] .card`, css.Raw("background", "white"))
	parseOut := css.Harvest()
	if !strings.Contains(parseOut, `@layer overrides{[data-theme="light"] .card{background:white`) {
		parseT.Fatalf("expected global rule wrapped in @layer overrides, got %q", parseOut)
	}
	if strings.Contains(parseOut, ".c-") {
		parseT.Fatalf("LayerGlobal must not hash the selector, got %q", parseOut)
	}
}

// TestDeclareLayersEmptyIsNoop proves a zero-arg call emits nothing.
func TestDeclareLayersEmptyIsNoop(parseT *testing.T) {
	css.Reset()
	css.DeclareLayers()
	if parseOut := css.Harvest(); parseOut != "" {
		parseT.Fatalf("expected no emission, got %q", parseOut)
	}
}
