//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/css"
)

// TestLayerComposesWithVariant probes Layer + a pseudo-state variant: the layer
// at-rule and the &:hover selector must both appear and nest correctly.
func TestLayerComposesWithVariant(parseT *testing.T) {
	css.Reset()
	parseSheet := css.Layer("components", css.Hover(css.Raw("color", "red"))...)
	parseOut := css.Harvest()
	parseExpected := "@layer components{." + string(parseSheet) + ":hover{color:red"
	if !strings.Contains(parseOut, parseExpected) {
		parseT.Fatalf("Layer+Hover: expected %q in %q", parseExpected, parseOut)
	}
}

// TestGlobalComposesWithMedia probes a global selector wrapped in @media.
func TestGlobalComposesWithMedia(parseT *testing.T) {
	css.Reset()
	css.Global("body", css.Media(css.MinW(768), css.Raw("margin", "0"))...)
	parseOut := css.Harvest()
	if !strings.Contains(parseOut, "@media") || !strings.Contains(parseOut, "body{margin:0") {
		parseT.Fatalf("Global+Media: expected media-wrapped body rule, got %q", parseOut)
	}
}

// TestLayerGlobalWithAncestorVariant probes LayerGlobal carrying a Within
// ancestor selector — the override-layer + theme-attr combination.
func TestLayerGlobalWithAncestorVariant(parseT *testing.T) {
	css.Reset()
	css.LayerGlobal("overrides", ".card", css.DataTheme("light", css.Raw("background", "white"))...)
	parseOut := css.Harvest()
	if !strings.Contains(parseOut, "@layer overrides{") {
		parseT.Fatalf("expected @layer overrides wrapper, got %q", parseOut)
	}
	if !strings.Contains(parseOut, `[data-theme="light"]`) || !strings.Contains(parseOut, "background:white") {
		parseT.Fatalf("expected data-theme-scoped .card override, got %q", parseOut)
	}
}
