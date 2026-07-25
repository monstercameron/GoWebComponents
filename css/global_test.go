//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/css"
)

// TestGlobalEmitsLiteralElementSelector proves Global emits an un-prefixed
// element rule (no hashed class) into the buffer sink (CSS1).
func TestGlobalEmitsLiteralElementSelector(parseT *testing.T) {
	css.Reset()
	css.Global("body", css.Raw("margin", "0"))
	parseOut := css.Harvest()
	if !strings.Contains(parseOut, "body{margin:0") {
		parseT.Fatalf("expected literal body rule, got %q", parseOut)
	}
	if strings.Contains(parseOut, ".c-") {
		parseT.Fatalf("Global must not scope under a hashed class, got %q", parseOut)
	}
}

// TestGlobalEmitsSemanticClassUnhashed proves a stable semantic class name
// survives verbatim — the theme engine / e2e selectors depend on it.
func TestGlobalEmitsSemanticClassUnhashed(parseT *testing.T) {
	css.Reset()
	css.Global(".nav-link", css.Raw("display", "flex"))
	parseOut := css.Harvest()
	if !strings.Contains(parseOut, ".nav-link{display:flex") {
		parseT.Fatalf("expected literal .nav-link rule, got %q", parseOut)
	}
}

// TestRootEmitsCustomPropertyPalette proves Root emits a :root token block (CSS2).
func TestRootEmitsCustomPropertyPalette(parseT *testing.T) {
	css.Reset()
	css.Root(css.Raw("--accent", "#4f46e5"), css.Raw("--radius", "12px"))
	parseOut := css.Harvest()
	if !strings.Contains(parseOut, ":root{") ||
		!strings.Contains(parseOut, "--accent:#4f46e5") ||
		!strings.Contains(parseOut, "--radius:12px") {
		parseT.Fatalf("expected :root token palette, got %q", parseOut)
	}
}

// TestGlobalVariantComposesAgainstSelector proves Hover composes against the
// literal selector (`.btn:hover`), not a hashed class.
func TestGlobalVariantComposesAgainstSelector(parseT *testing.T) {
	css.Reset()
	css.Global(".btn", css.Hover(css.Raw("color", "red"))...)
	parseOut := css.Harvest()
	if !strings.Contains(parseOut, ".btn:hover{color:red") {
		parseT.Fatalf("expected .btn:hover rule, got %q", parseOut)
	}
}

// TestGlobalDedupes proves identical Global calls emit exactly once.
func TestGlobalDedupes(parseT *testing.T) {
	css.Reset()
	parseBefore := len(css.HarvestedClasses())
	css.Global("h3", css.Raw("font-weight", "700"))
	parseAfterFirst := len(css.HarvestedClasses())
	css.Global("h3", css.Raw("font-weight", "700"))
	parseAfterSecond := len(css.HarvestedClasses())
	if parseAfterFirst != parseBefore+1 {
		parseT.Fatalf("first Global should add one entry, got %d -> %d", parseBefore, parseAfterFirst)
	}
	if parseAfterSecond != parseAfterFirst {
		parseT.Fatalf("duplicate Global must not re-emit, got %d -> %d", parseAfterFirst, parseAfterSecond)
	}
}

// TestWithinAncestorState proves Within emits `<ancestor> .c-xxx` (CSS3).
func TestWithinAncestorState(parseT *testing.T) {
	css.Reset()
	parseSheet := css.New(css.Within(`[data-theme="light"]`, css.Raw("color", "black"))...)
	parseOut := css.Harvest()
	parseExpected := `[data-theme="light"] .` + string(parseSheet) + "{color:black"
	if !strings.Contains(parseOut, parseExpected) {
		parseT.Fatalf("expected %q in %q", parseExpected, parseOut)
	}
}

// TestDataThemeConvenience proves DataTheme wraps the data-theme attribute.
func TestDataThemeConvenience(parseT *testing.T) {
	css.Reset()
	css.New(css.DataTheme("light", css.Raw("color", "black"))...)
	parseOut := css.Harvest()
	if !strings.Contains(parseOut, `[data-theme="light"] .`) {
		parseT.Fatalf("expected data-theme ancestor selector, got %q", parseOut)
	}
}

// TestInjectIsIdempotentByID proves Inject records the first CSS for an id and
// ignores later calls for the same id (G30).
func TestInjectIsIdempotentByID(parseT *testing.T) {
	css.Reset()
	css.Inject("app-fonts", "@font-face{font-family:A}")
	css.Inject("app-fonts", "@font-face{font-family:B}") // same id, ignored
	parseOut := css.Harvest()
	if !strings.Contains(parseOut, "font-family:A") {
		parseT.Fatalf("expected first injected CSS, got %q", parseOut)
	}
	if strings.Contains(parseOut, "font-family:B") {
		parseT.Fatalf("second Inject for same id must be ignored, got %q", parseOut)
	}
}

// TestGlobalHardensStyleBreakout proves emitted global CSS cannot terminate the
// <style> element (security boundary shared with New).
func TestGlobalHardensStyleBreakout(parseT *testing.T) {
	css.Reset()
	css.Global("body", css.Raw("content", `"</style><script>x</script>"`))
	parseOut := css.Harvest()
	if strings.Contains(parseOut, "</style>") {
		parseT.Fatalf("global CSS must be hardened against </style> breakout, got %q", parseOut)
	}
}

// TestGlobalEmptyInputsAreNoops proves blank selector / empty rules / blank
// inject id are safe no-ops.
func TestGlobalEmptyInputsAreNoops(parseT *testing.T) {
	css.Reset()
	css.Global("", css.Raw("x", "y"))
	css.Global("body")
	css.Root()
	css.Inject("", "x")
	css.Inject("id", "")
	if parseOut := css.Harvest(); parseOut != "" {
		parseT.Fatalf("expected no emission from empty inputs, got %q", parseOut)
	}
}
