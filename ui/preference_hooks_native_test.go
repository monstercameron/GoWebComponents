package ui_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// preferenceProbe renders the preference-hook values so a server render can
// assert the documented defaults on builds without media-query support.
func preferenceProbe(_ struct{}) ui.Node {
	parseReduced := ui.UsePrefersReducedMotion()
	parseScheme := ui.UsePrefersColorScheme()
	return html.Div(html.Props{ID: "prefs"},
		html.Text(fmt.Sprintf("motion=%v scheme=%s", parseReduced, parseScheme)),
	)
}

// TestPreferenceHooksDefaultWithoutMediaQueries pins the native/SSR contract:
// with no matchMedia available the hooks return reduced-motion=false and the
// light color scheme, and rendering does not error (the effect is a no-op).
func TestPreferenceHooksDefaultWithoutMediaQueries(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(ui.CreateElement(preferenceProbe, struct{}{}))
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "motion=false") {
		parseT.Fatalf("expected reduced-motion default false, got: %s", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "scheme=light") {
		parseT.Fatalf("expected color-scheme default light, got: %s", parseMarkup)
	}
}

// TestColorSchemeConstants documents the public ColorScheme values.
func TestColorSchemeConstants(parseT *testing.T) {
	if ui.ColorSchemeLight != "light" || ui.ColorSchemeDark != "dark" {
		parseT.Fatalf("unexpected ColorScheme constants: %q %q", ui.ColorSchemeLight, ui.ColorSchemeDark)
	}
}
