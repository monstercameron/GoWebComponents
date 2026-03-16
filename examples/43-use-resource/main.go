//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type deploymentPreview struct {
	Environment string
	Nodes       int
	Window      string
}

func useResourceExample() ui.Node {
	environment := ui.UseState("Staging")
	resource := fetch.UseResource(func(ctx context.Context) (deploymentPreview, error) {
		select {
		case <-time.After(900 * time.Millisecond):
		case <-ctx.Done():
			return deploymentPreview{}, ctx.Err()
		}

		if environment.Get() == "Production" {
			return deploymentPreview{Environment: "Production", Nodes: 12, Window: "02:00 UTC"}, nil
		}
		return deploymentPreview{Environment: "Staging", Nodes: 4, Window: "Now"}, nil
	}, environment.Get())

	showStaging := ui.UseEvent(func() { environment.Set("Staging") })
	showProduction := ui.UseEvent(func() { environment.Set("Production") })
	reload := ui.UseEvent(func() { resource.Reload() })
	cancel := ui.UseEvent(func() { resource.Cancel() })

	state := resource.Get()
	status := "Idle"
	if state.Loading {
		status = "Loading"
	} else if state.Error != nil {
		status = "Error"
	} else if state.Ready {
		status = "Ready"
	}

	window := "-"
	nodes := "-"
	if state.Ready {
		window = state.Value.Window
		nodes = fmt.Sprintf("%d", state.Value.Nodes)
	}

	return shared.ExamplePage(
		"fetch.UseResource",
		"Load typed async values with cancellation and reload support",
		"UseResource is the higher-level hook for Go loaders. It returns typed values, exposes cancellation, and re-runs when dependency values change.",
		shared.ExamplePanel("Resource controls",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Staging", showStaging),
				shared.ExampleButton("Production", showProduction),
				shared.ExampleButton("Reload", reload),
				shared.ExampleButton("Cancel", cancel),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Status", status),
				shared.ExampleStat("Environment", environment.Get()),
				shared.ExampleStat("Nodes", nodes),
				shared.ExampleStat("Window", window),
			),
		),
		shared.ExamplePanel("Typed loader",
			shared.ExampleCode(
				`resource := fetch.UseResource(func(ctx context.Context) (deploymentPreview, error) { ... }, environment.Get())`,
				`state := resource.Get()`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useResourceExample), "#app")
	select {}
}