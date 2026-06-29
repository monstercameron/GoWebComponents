//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

func useTaskExample() ui.Node {
	parseProgress := ui.UseState(0)
	parseTask := ui.UseTask(func(parseCtx context.Context) (string, error) {
		for parseStep := 1; parseStep <= 5; parseStep++ {
			select {
			case <-parseCtx.Done():
				return "", parseCtx.Err()
			case <-time.After(280 * time.Millisecond):
			}
			parseProgress.Set(parseStep * 20)
		}
		return "Artifact published", nil
	})

	parseStart := ui.UseEvent(func() {
		parseProgress.Set(0)
		parseTask.Start()
	})
	parseCancel := ui.UseEvent(func() { parseTask.Cancel() })
	reset := ui.UseEvent(func() {
		parseTask.Cancel()
		parseProgress.Set(0)
	})

	parseState := parseTask.Get()
	parseResult := "No result yet"
	if parseState.Ready {
		parseResult = parseState.Value
	} else if parseState.Error != nil {
		parseResult = parseState.Error.Error()
	}

	return shared.ExamplePage(
		"ui.UseTask",
		"Start and cancel a typed background job from UI actions",
		"UseTask is for explicit jobs, not automatic data loading. The task only runs when Start is called, tracks cancellation, and exposes a typed terminal value or error.",
		shared.ExamplePanel("Task lifecycle",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Start task", parseStart),
				shared.ExampleButton("Cancel task", parseCancel),
				shared.ExampleButton("Reset", reset),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-5"},
				shared.ExampleStat("Progress", fmt.Sprintf("%d%%", parseProgress.Get())),
				shared.ExampleStat("Running", fmt.Sprintf("%t", parseState.Running)),
				shared.ExampleStat("Ready", fmt.Sprintf("%t", parseState.Ready)),
				shared.ExampleStat("Cancelled", fmt.Sprintf("%t", parseState.Cancelled)),
				shared.ExampleStat("Started", fmt.Sprintf("%t", parseState.Started)),
			),
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text("Result: "+parseResult)),
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
	exampleboot.RenderExampleRoot(ui.CreateElement(useTaskExample))
	exampleboot.WaitExampleRuntime()
}
