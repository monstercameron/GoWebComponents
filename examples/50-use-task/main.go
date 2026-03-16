//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func useTaskExample() ui.Node {
	progress := ui.UseState(0)
	task := ui.UseTask(func(ctx context.Context) (string, error) {
		for step := 1; step <= 5; step++ {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(280 * time.Millisecond):
			}
			progress.Set(step * 20)
		}
		return "Artifact published", nil
	})

	start := ui.UseEvent(func() {
		progress.Set(0)
		task.Start()
	})
	cancel := ui.UseEvent(func() { task.Cancel() })
	reset := ui.UseEvent(func() {
		task.Cancel()
		progress.Set(0)
	})

	state := task.Get()
	result := "No result yet"
	if state.Ready {
		result = state.Value
	} else if state.Error != nil {
		result = state.Error.Error()
	}

	return shared.ExamplePage(
		"ui.UseTask",
		"Start and cancel a typed background job from UI actions",
		"UseTask is for explicit jobs, not automatic data loading. The task only runs when Start is called, tracks cancellation, and exposes a typed terminal value or error.",
		shared.ExamplePanel("Task lifecycle",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Start task", start),
				shared.ExampleButton("Cancel task", cancel),
				shared.ExampleButton("Reset", reset),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-5"},
				shared.ExampleStat("Progress", fmt.Sprintf("%d%%", progress.Get())),
				shared.ExampleStat("Running", fmt.Sprintf("%t", state.Running)),
				shared.ExampleStat("Ready", fmt.Sprintf("%t", state.Ready)),
				shared.ExampleStat("Cancelled", fmt.Sprintf("%t", state.Cancelled)),
				shared.ExampleStat("Started", fmt.Sprintf("%t", state.Started)),
			),
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text("Result: "+result)),
			shared.ExampleCode(
				`task := ui.UseTask(func(ctx context.Context) (string, error) { ... })`,
				`task.Start()`,
				`task.Cancel()`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useTaskExample), "#app")
	select {}
}