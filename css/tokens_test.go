//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/css"
)

// TestDesignTokenAndVariableRoundTrip proves the full CSS custom-property /
// design-token loop (CSS2): define a :root palette with Root+Raw, reference the
// tokens in typed values via Var, and confirm the emitted CSS uses var(--token)
// — so a runtime element.style.setProperty on :root reskins every reference
// without regenerating any hashed classes.
func TestDesignTokenAndVariableRoundTrip(parseT *testing.T) {
	css.Reset()

	// 1. Token palette on :root.
	css.Root(
		css.Raw("--accent", "#4f46e5"),
		css.Raw("--radius", "12px"),
	)

	// 2. A component class that REFERENCES the tokens (not literal values).
	parseSheet := css.New(
		css.Bg(css.Var("accent")),
		css.Raw("border-radius", string(css.Var("radius"))),
	)

	parseOut := css.Harvest()

	if !strings.Contains(parseOut, ":root{") ||
		!strings.Contains(parseOut, "--accent:#4f46e5") ||
		!strings.Contains(parseOut, "--radius:12px") {
		parseT.Fatalf("expected :root token palette, got %q", parseOut)
	}
	if !strings.Contains(parseOut, "."+string(parseSheet)+"{") ||
		!strings.Contains(parseOut, "var(--accent)") ||
		!strings.Contains(parseOut, "var(--radius)") {
		parseT.Fatalf("expected component class to reference var(--token), got %q", parseOut)
	}
}

// TestThemeRootRulesEmitsCustomProperties proves the typed Theme drives a :root
// custom-property palette (CSS2 Theme->:root bridge).
func TestThemeRootRulesEmitsCustomProperties(parseT *testing.T) {
	css.Reset()
	css.EmitThemeTokens(css.DefaultTheme())
	parseOut := css.Harvest()

	for _, parseFragment := range []string{
		":root{",
		"--color-slate-900:",
		"--color-white:",
		"--space-4:",
		"--text-lg:",
		"--radius-md:",
	} {
		if !strings.Contains(parseOut, parseFragment) {
			parseT.Fatalf("expected %q in theme tokens, got %q", parseFragment, parseOut)
		}
	}
	// Breakpoints are intentionally NOT emitted as custom properties.
	if strings.Contains(parseOut, "--bp-") || strings.Contains(parseOut, "--breakpoint") {
		parseT.Fatalf("breakpoints should not be emitted as :root vars, got %q", parseOut)
	}
}

// TestThemeRootRulesComposeWithVarAndDataTheme proves a theme's tokens can be
// referenced via Var and scoped under a data-theme ancestor (a light override).
func TestThemeRootRulesComposeWithVarAndDataTheme(parseT *testing.T) {
	css.Reset()
	parseLight := css.DefaultTheme()
	parseLight.Colors = map[string]css.Color{"bg": css.White}
	parseLight.Spacing = nil
	parseLight.FontSizes = nil
	parseLight.Radii = nil

	css.New(css.DataTheme("light", parseLight.RootRules()...)...)
	parseOut := css.Harvest()
	if !strings.Contains(parseOut, `[data-theme="light"] .`) || !strings.Contains(parseOut, "--color-bg:") {
		parseT.Fatalf("expected data-theme-scoped token, got %q", parseOut)
	}
}

// TestVarSanitizesAndPrefixes proves Var normalizes a token name to a safe
// custom-property reference.
func TestVarSanitizesAndPrefixes(parseT *testing.T) {
	if parseGot := string(css.Var("accent")); parseGot != "var(--accent)" {
		parseT.Fatalf("Var(accent) = %q, want var(--accent)", parseGot)
	}
	if parseGot := string(css.Var("--accent")); parseGot != "var(--accent)" {
		parseT.Fatalf("Var(--accent) = %q, want var(--accent)", parseGot)
	}
	// Injection attempt is stripped to identifier chars.
	if parseGot := string(css.Var("accent);x:y")); !strings.HasPrefix(parseGot, "var(--accent") || strings.Contains(parseGot, ";") {
		parseT.Fatalf("Var must sanitize, got %q", parseGot)
	}
}

// TestThemeRootRulesHandlesPartialAndEmptyThemes proves a custom Theme with only
// some scales set (nil maps elsewhere) emits just those tokens without panicking,
// and a fully-empty Theme emits nothing — the common "define only my colors"
// usage.
func TestThemeRootRulesHandlesPartialAndEmptyThemes(parseT *testing.T) {
	parsePartial := css.Theme{Colors: map[string]css.Color{"brand": css.Color("#abc")}}
	parseRules := parsePartial.RootRules()
	if len(parseRules) != 1 {
		parseT.Fatalf("partial theme RootRules = %d rules, want 1", len(parseRules))
	}
	if len((css.Theme{}).RootRules()) != 0 {
		parseT.Fatal("empty theme must emit no token rules")
	}
}
