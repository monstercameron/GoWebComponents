//go:build js && wasm

package render_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/testkit/render"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func counterExampleApp() ui.Node {
	parseCount := ui.UseState(0)
	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrevious int) int {
			return parsePrevious + 1
		})
	})

	return html.Div(html.Props{ID: "counter-example"},
		html.P(html.Props{ID: "count-output"}, html.Text(fmt.Sprintf("Count: %d", parseCount.Get()))),
		html.Button(html.Props{ID: "increment-button", Type: "button", OnClick: parseIncrement}, html.Text("Increment")),
	)
}

func saveFlowExampleApp() ui.Node {
	parseStatus := ui.UseState("Idle")
	handleSave := ui.UseEvent(func() {
		ui.StartTransition(func() {
			parseStatus.Set("Saved")
		})
	})

	return html.Div(html.Props{ID: "save-example"},
		html.Button(html.Props{ID: "save-button", Type: "button", OnClick: handleSave}, html.Text("Save profile")),
		html.P(html.Props{ID: "save-status"}, html.Text(parseStatus.Get())),
	)
}

func TestConsumerRenderPattern_ComponentState(parseT *testing.T) {
	parseFixture := render.New(parseT)
	parseFixture.Render(ui.CreateElement(counterExampleApp))

	parseFixture.ByRole("button", "Increment").Click()

	if parseGot := parseFixture.ByID("count-output").Text(); parseGot != "Count: 1" {
		parseT.Fatalf("expected count output to update, got %q", parseGot)
	}
}

func TestConsumerRenderPattern_QueuedIntegrationFlow(parseT *testing.T) {
	parseFixture := render.New(parseT, render.WithQueuedScheduler())
	parseFixture.Render(ui.CreateElement(saveFlowExampleApp))

	parseFixture.ByRole("button", "Save profile").Click()

	if parseGot := parseFixture.ByID("save-status").Text(); parseGot != "Saved" {
		parseT.Fatalf("expected queued save flow to settle, got %q", parseGot)
	}
}
