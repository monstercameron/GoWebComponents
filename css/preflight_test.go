//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/css"
)

// TestPreflightEmitsResetRules proves Preflight emits the base reset (CSS5).
func TestPreflightEmitsResetRules(parseT *testing.T) {
	css.Reset()
	css.Preflight()
	parseOut := css.Harvest()
	for _, parseFragment := range []string{
		"box-sizing:border-box", "*{margin:0", "img,picture,video,canvas,svg{", "button{cursor:pointer",
	} {
		if !strings.Contains(parseOut, parseFragment) {
			parseT.Fatalf("expected %q in preflight, got %q", parseFragment, parseOut)
		}
	}
	if strings.Contains(parseOut, ".c-") {
		parseT.Fatalf("preflight must emit global (un-hashed) rules, got %q", parseOut)
	}
}

// TestPreflightInLayerWrapsInLayer proves PreflightInLayer scopes the reset to a
// named cascade layer (CSS4+CSS5).
func TestPreflightInLayerWrapsInLayer(parseT *testing.T) {
	css.Reset()
	css.PreflightInLayer("base")
	parseOut := css.Harvest()
	if !strings.Contains(parseOut, "@layer base{*") && !strings.Contains(parseOut, "@layer base{html") &&
		!strings.Contains(parseOut, "@layer base{body") {
		parseT.Fatalf("expected reset rules wrapped in @layer base, got %q", parseOut)
	}
}

// TestPreflightIsIdempotent proves repeated calls do not duplicate emission.
func TestPreflightIsIdempotent(parseT *testing.T) {
	css.Reset()
	css.Preflight()
	parseFirst := len(css.HarvestedClasses())
	css.Preflight()
	if parseSecond := len(css.HarvestedClasses()); parseSecond != parseFirst {
		parseT.Fatalf("Preflight must be idempotent, %d -> %d", parseFirst, parseSecond)
	}
}

// TestCriticalCSSExtractAndSeedRoundTrip proves the CSS6 extract→seed loop: emit
// rules, extract a <style> block containing them, then Seed the classes so a
// re-emit is suppressed.
func TestCriticalCSSExtractAndSeedRoundTrip(parseT *testing.T) {
	css.Reset()
	parseSheet := css.New(css.Raw("color", "rebeccapurple"))

	parseBlock := css.CriticalCSS()
	if !strings.Contains(parseBlock, "<style data-gwc-css=") {
		parseT.Fatalf("expected a <style data-gwc-css> block, got %q", parseBlock)
	}
	if !strings.Contains(parseBlock, "color:rebeccapurple") {
		parseT.Fatalf("critical CSS missing the emitted rule, got %q", parseBlock)
	}

	// Simulate a fresh client: reset, seed the class, then re-fold the same rules.
	css.Reset()
	css.Seed(string(parseSheet))
	if css.New(css.Raw("color", "rebeccapurple")) != parseSheet {
		parseT.Fatal("re-folding the same rules must yield the same class")
	}
	if css.Harvest() != "" {
		parseT.Fatalf("seeded class must not re-emit, got %q", css.Harvest())
	}
}

// TestCriticalCSSEmptyWhenNothingEmitted proves the extraction is empty on a
// clean registry.
func TestCriticalCSSEmptyWhenNothingEmitted(parseT *testing.T) {
	css.Reset()
	if parseBlock := css.CriticalCSS(); parseBlock != "" {
		parseT.Fatalf("expected empty critical CSS, got %q", parseBlock)
	}
}
