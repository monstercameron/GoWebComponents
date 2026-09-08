//go:build js && wasm

package render_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	render "github.com/monstercameron/GoWebComponents/v6/test/render"
	base "github.com/monstercameron/GoWebComponents/v6/testkit/render"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func parityCounterApp() ui.Node {
	parseCount := ui.UseState(0)
	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrevious int) int {
			return parsePrevious + 1
		})
	})

	return html.Div(html.Props{ID: "parity-root"},
		html.P(html.Props{ID: "count-output"}, html.Text(fmt.Sprintf("Count: %d", parseCount.Get()))),
		html.Button(html.Props{ID: "increment-button", Type: "button", OnClick: parseIncrement}, html.Text("Increment")),
	)
}

func TestPreferredRenderWrappersMatchCompatibilityAliasBehavior(parseT *testing.T) {
	parsePreferred := render.New(parseT, render.WithQueuedScheduler())
	parsePreferred.Render(ui.CreateElement(parityCounterApp))
	parsePreferred.ByRole("button", "Increment").Click()
	parsePreferredText := parsePreferred.ByID("count-output").Text()
	parsePreferred.Cleanup()

	parseCompat := base.New(parseT, base.WithQueuedScheduler())
	parseCompat.Render(ui.CreateElement(parityCounterApp))
	parseCompat.ByRole("button", "Increment").Click()
	parseCompatText := parseCompat.ByID("count-output").Text()
	parseCompat.Cleanup()

	if parsePreferredText != parseCompatText {
		parseT.Fatalf("expected preferred wrapper and compatibility alias to match, got preferred=%q compat=%q", parsePreferredText, parseCompatText)
	}
}
