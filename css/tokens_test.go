//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/css"
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
