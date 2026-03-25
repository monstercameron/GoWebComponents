//go:build js && wasm
// +build js,wasm

package render_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	render "github.com/monstercameron/GoWebComponents/test/render"
	"github.com/monstercameron/GoWebComponents/ui"
)

func counterExampleApp() ui.Node {
	count := ui.UseState(0)
	increment := ui.UseEvent(func() {
		count.Update(func(previous int) int {
			return previous + 1
		})
	})

	return html.Div(html.Props{ID: "counter-example"},
		html.P(html.Props{ID: "count-output"}, html.Text(fmt.Sprintf("Count: %d", count.Get()))),
		html.Button(html.Props{ID: "increment-button", Type: "button", OnClick: increment}, html.Text("Increment")),
	)
}

func saveFlowExampleApp() ui.Node {
	status := ui.UseState("Idle")
	handleSave := ui.UseEvent(func() {
		ui.StartTransition(func() {
			status.Set("Saved")
		})
	})

	return html.Div(html.Props{ID: "save-example"},
		html.Button(html.Props{ID: "save-button", Type: "button", OnClick: handleSave}, html.Text("Save profile")),
		html.P(html.Props{ID: "save-status"}, html.Text(status.Get())),
	)
}

func TestConsumerRenderPattern_ComponentState(t *testing.T) {
	fixture := render.New(t)
	fixture.Render(ui.CreateElement(counterExampleApp))

	fixture.ByRole("button", "Increment").Click()

	if got := fixture.ByID("count-output").Text(); got != "Count: 1" {
		t.Fatalf("expected count output to update, got %q", got)
	}
}

func TestConsumerRenderPattern_QueuedIntegrationFlow(t *testing.T) {
	fixture := render.New(t, render.WithQueuedScheduler())
	fixture.Render(ui.CreateElement(saveFlowExampleApp))

	fixture.ByRole("button", "Save profile").Click()

	if got := fixture.ByID("save-status").Text(); got != "Saved" {
		t.Fatalf("expected queued save flow to settle, got %q", got)
	}
}
