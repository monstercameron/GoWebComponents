//go:build js && wasm
// +build js,wasm

package render_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	render "github.com/monstercameron/GoWebComponents/test/render"
	base "github.com/monstercameron/GoWebComponents/testkit/render"
	"github.com/monstercameron/GoWebComponents/ui"
)

func parityCounterApp() ui.Node {
	count := ui.UseState(0)
	increment := ui.UseEvent(func() {
		count.Update(func(previous int) int {
			return previous + 1
		})
	})

	return html.Div(html.Props{ID: "parity-root"},
		html.P(html.Props{ID: "count-output"}, html.Text(fmt.Sprintf("Count: %d", count.Get()))),
		html.Button(html.Props{ID: "increment-button", Type: "button", OnClick: increment}, html.Text("Increment")),
	)
}

func TestPreferredRenderWrappersMatchCompatibilityAliasBehavior(t *testing.T) {
	preferred := render.New(t, render.WithQueuedScheduler())
	preferred.Render(ui.CreateElement(parityCounterApp))
	preferred.ByRole("button", "Increment").Click()
	preferredText := preferred.ByID("count-output").Text()
	preferred.Cleanup()

	compat := base.New(t, base.WithQueuedScheduler())
	compat.Render(ui.CreateElement(parityCounterApp))
	compat.ByRole("button", "Increment").Click()
	compatText := compat.ByID("count-output").Text()
	compat.Cleanup()

	if preferredText != compatText {
		t.Fatalf("expected preferred wrapper and compatibility alias to match, got preferred=%q compat=%q", preferredText, compatText)
	}
}
