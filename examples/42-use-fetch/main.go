//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
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
	currentURL := ui.UseState(teamFeedURL)
	resource := fetch.UseFetch(currentURL.Get())
	state := resource.Get()

	showTeam := ui.UseEvent(func() { currentURL.Set(teamFeedURL) })
	showMetrics := ui.UseEvent(func() { currentURL.Set(metricsFeedURL) })
	refetch := ui.UseEvent(func() { resource.Refetch() })

	status := "Idle"
	if state.Loading {
		status = "Loading"
	} else if state.Error != "" {
		status = "Error"
	} else if state.Data != nil {
		status = "Ready"
	}

	payload := "No response yet"
	if state.Data != nil {
		payload = fmt.Sprint(state.Data)
	}

	return shared.ExamplePage(
		"fetch.UseFetch",
		"Track low-level browser fetch state inside a component",
		"UseFetch is the lightest hook in the package: give it a URL, read the raw loading and error state, and parse the response body yourself.",
		shared.ExamplePanel("Request controls",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Team payload", showTeam),
				shared.ExampleButton("Metrics payload", showMetrics),
				shared.ExampleButton("Refetch", refetch),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Status", status),
				shared.ExampleStat("Error", state.Error),
				shared.ExampleStat("URL kind", map[bool]string{true: "Team", false: "Metrics"}[currentURL.Get() == teamFeedURL]),
			),
		),
		shared.ExamplePanel("Raw response data",
			shared.ExampleCode(payload),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useFetchExample), "#app")
	select {}
}
