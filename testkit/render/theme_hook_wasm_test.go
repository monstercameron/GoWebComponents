//go:build js && wasm

package render_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/testkit/render"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func themeToggleApp() ui.Node {
	parseTheme, parseSetTheme := ui.UseTheme("dark")
	parseToggle := ui.UseEvent(func() {
		if parseTheme == "dark" {
			parseSetTheme("light")
			return
		}
		parseSetTheme("dark")
	})
	return html.Div(html.Props{ID: "theme-root"},
		html.Button(html.Props{ID: "toggle-theme", Type: "button", OnClick: parseToggle}, html.Text("toggle")),
		html.P(html.Props{ID: "theme-label"}, html.Text(parseTheme)),
	)
}

// TestUseThemeReactiveSwitch proves UseTheme seeds its default, re-renders the
// subscriber when the theme changes, and that an EXTERNAL SetTheme also drives
// the re-render.
func TestUseThemeReactiveSwitch(parseT *testing.T) {
	parseFixture := render.New(parseT)
	parseFixture.Render(ui.CreateElement(themeToggleApp))
	parseFixture.Flush()

	if parseGot := parseFixture.ByID("theme-label").Text(); parseGot != "dark" {
		parseT.Fatalf("initial theme = %q, want dark (the default)", parseGot)
	}

	// Switch via the hook's setter.
	parseFixture.ClickByID("toggle-theme")
	parseFixture.Flush()
	if parseGot := parseFixture.ByID("theme-label").Text(); parseGot != "light" {
		parseT.Fatalf("after toggle theme = %q, want light", parseGot)
	}

	// Switch from OUTSIDE the component via SetTheme — subscriber must re-render.
	ui.SetTheme("dark")
	parseFixture.Flush()
	if parseGot := parseFixture.ByID("theme-label").Text(); parseGot != "dark" {
		parseT.Fatalf("after external SetTheme theme = %q, want dark", parseGot)
	}
	if parseGot := ui.CurrentTheme(); parseGot != "dark" {
		parseT.Fatalf("CurrentTheme = %q, want dark", parseGot)
	}
}
