//go:build js && wasm

package render_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/testkit/render"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// TestNewHooksComposeInOneComponent exercises many of the v3.3–v3.4 hooks in a
// single component to catch hook-ordering / coexistence regressions (stable hook
// positions across re-render).
func TestNewHooksComposeInOneComponent(parseT *testing.T) {
	parseMounts := 0
	parseApp := func() ui.Node {
		parseCount := ui.UseState(0)
		ui.UseMount(func() func() { parseMounts++; return nil })
		parseWide := ui.UseMediaQuery("(min-width: 768px)")
		parseTheme, parseSetTheme := ui.UseTheme("dark")
		ui.UseLayoutEffect(func() func() { return nil }, parseCount.Get())
		parseBump := ui.UseEvent(func() {
			parseCount.Update(func(parsePrev int) int { return parsePrev + 1 })
			if parseTheme == "dark" {
				parseSetTheme("light")
			}
		})
		return html.Div(html.Props{ID: "compose"},
			html.Button(html.Props{ID: "bump", Type: "button", OnClick: parseBump}, html.Text("bump")),
			html.P(html.Props{ID: "out"}, html.Text(fmt.Sprintf("c=%d wide=%t theme=%s", parseCount.Get(), parseWide, parseTheme))),
		)
	}

	parseFixture := render.New(parseT)
	parseFixture.Render(ui.CreateElement(parseApp))
	parseFixture.Flush()
	if parseMounts != 1 {
		parseT.Fatalf("UseMount fired %d times, want 1", parseMounts)
	}
	parseBefore := parseFixture.ByID("out").Text()

	parseFixture.ClickByID("bump")
	parseFixture.Flush()
	parseAfter := parseFixture.ByID("out").Text()

	if parseMounts != 1 {
		parseT.Fatalf("UseMount re-fired on re-render: %d", parseMounts)
	}
	if parseBefore == parseAfter {
		parseT.Fatalf("expected output to change after bump, still %q", parseAfter)
	}
}
