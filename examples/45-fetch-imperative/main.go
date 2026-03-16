//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const (
	manualGreetingURL = "data:text/plain,hello%20from%20manual%20fetch"
	manualOpsURL      = "data:text/plain,ops%20payload%20loaded"
)

func runManualFetch(url string, status ui.State[string], loading ui.State[bool]) {
	loading.Set(true)
	status.Set("Request started...")
	resultChan := fetch.Fetch(url, fetch.Options{})

	go func() {
		result := <-resultChan
		fetch.ReturnChannel(resultChan)
		loading.Set(false)
		if result.Err != nil {
			status.Set("Error: " + result.Err.Error())
			return
		}
		status.Set(fmt.Sprintf("Result: %v", result.Data))
	}()
}

func fetchImperativeExample() ui.Node {
	status := ui.UseState("Press a button to issue an imperative request from an event handler.")
	loading := ui.UseState(false)
	loadGreeting := ui.UseEvent(func() { runManualFetch(manualGreetingURL, status, loading) })
	loadOps := ui.UseEvent(func() { runManualFetch(manualOpsURL, status, loading) })

	mode := "Idle"
	if loading.Get() {
		mode = "Loading"
	}

	return shared.ExamplePage(
		"fetch.Fetch",
		"Issue manual requests outside the hook model",
		"The imperative API is useful from event handlers, goroutines, or utility code where a component-scoped hook is not the right abstraction.",
		shared.ExamplePanel("Imperative requests",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Load greeting", loadGreeting),
				shared.ExampleButton("Load ops payload", loadOps),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Mode", mode),
				shared.ExampleStat("API", "fetch.Fetch"),
			),
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text(status.Get())),
			shared.ExampleCode(
				`resultChan := fetch.Fetch(url, fetch.Options{})`,
				`go func() { result := <-resultChan; ... }()`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(fetchImperativeExample), "#app")
	select {}
}