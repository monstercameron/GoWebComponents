//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const (
	teamFeedURL    = "data:application/json,%7B%22team%22%3A%22alpha%22%2C%22status%22%3A%22ready%22%7D"
	metricsFeedURL = "data:application/json,%7B%22visitors%22%3A1280%2C%22trend%22%3A%22up%22%7D"
)

func useFetchExample() ui.Node {
	parseCurrentURL := ui.UseState(teamFeedURL)
	parseResource := fetch.UseFetch(parseCurrentURL.Get())
	parseState := parseResource.Get()

	parseShowTeam := ui.UseEvent(func() { parseCurrentURL.Set(teamFeedURL) })
	parseShowMetrics := ui.UseEvent(func() { parseCurrentURL.Set(metricsFeedURL) })
	parseRefetch := ui.UseEvent(func() { parseResource.Refetch() })

	parseStatus := "Idle"
	if parseState.Loading {
		parseStatus = "Loading"
	} else if parseState.Error != "" {
		parseStatus = "Error"
	} else if parseState.Data != nil {
		parseStatus = "Ready"
	}

	parsePayload := "No response yet"
	if parseState.Data != nil {
		parsePayload = fmt.Sprint(parseState.Data)
	}

	return shared.ExamplePage(
		"fetch.UseFetch",
		"Track low-level browser fetch state inside a component",
		"UseFetch is the lightest hook in the package: give it a URL, read the raw loading and error state, and parse the response body yourself.",
		shared.ExamplePanel("Request controls",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Team payload", parseShowTeam),
				shared.ExampleButton("Metrics payload", parseShowMetrics),
				shared.ExampleButton("Refetch", parseRefetch),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Status", parseStatus),
				shared.ExampleStat("Error", parseState.Error),
				shared.ExampleStat("URL kind", map[bool]string{true: "Team", false: "Metrics"}[parseCurrentURL.Get() == teamFeedURL]),
			),
		),
		shared.ExamplePanel("Raw response data",
			shared.ExampleCode(parsePayload),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(useFetchExample))
	exampleboot.WaitExampleRuntime()
}
