//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
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
	parseEnvironment := ui.UseState("Staging")
	parseResource := fetch.UseResource(func(parseCtx context.Context) (deploymentPreview, error) {
		select {
		case <-time.After(900 * time.Millisecond):
		case <-parseCtx.Done():
			return deploymentPreview{}, parseCtx.Err()
		}

		if parseEnvironment.Get() == "Production" {
			return deploymentPreview{Environment: "Production", Nodes: 12, Window: "02:00 UTC"}, nil
		}
		return deploymentPreview{Environment: "Staging", Nodes: 4, Window: "Now"}, nil
	}, parseEnvironment.Get())

	parseShowStaging := ui.UseEvent(func() { parseEnvironment.Set("Staging") })
	parseShowProduction := ui.UseEvent(func() { parseEnvironment.Set("Production") })
	parseReload := ui.UseEvent(func() { parseResource.Reload() })
	parseCancel := ui.UseEvent(func() { parseResource.Cancel() })

	parseState := parseResource.Get()
	parseStatus := "Idle"
	if parseState.Loading {
		parseStatus = "Loading"
	} else if parseState.Error != nil {
		parseStatus = "Error"
	} else if parseState.Ready {
		parseStatus = "Ready"
	}

	parseWindow := "-"
	parseNodes := "-"
	if parseState.Ready {
		parseWindow = parseState.Value.Window
		parseNodes = fmt.Sprintf("%d", parseState.Value.Nodes)
	}

	return shared.ExamplePage(
		"fetch.UseResource",
		"Load typed async values with cancellation and reload support",
		"UseResource is the higher-level hook for Go loaders. It returns typed values, exposes cancellation, and re-runs when dependency values change.",
		shared.ExamplePanel("Resource controls",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Staging", parseShowStaging),
				shared.ExampleButton("Production", parseShowProduction),
				shared.ExampleButton("Reload", parseReload),
				shared.ExampleButton("Cancel", parseCancel),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Status", parseStatus),
				shared.ExampleStat("Environment", parseEnvironment.Get()),
				shared.ExampleStat("Nodes", parseNodes),
				shared.ExampleStat("Window", parseWindow),
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
