//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/fetch"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

const (
	manualGreetingURL = "data:text/plain,hello%20from%20manual%20fetch"
	manualOpsURL      = "data:text/plain,ops%20payload%20loaded"
)

func runManualFetch(parseUrl string, parseStatus ui.State[string], parseLoading ui.State[bool]) {
	parseLoading.Set(true)
	parseStatus.Set("Request started...")
	parseResultChan := fetch.Fetch(parseUrl, fetch.Options{})

	go func() {
		parseResult := <-parseResultChan
		fetch.ReturnChannel(parseResultChan)
		parseLoading.Set(false)
		if parseResult.Err != nil {
			parseStatus.Set("Error: " + parseResult.Err.Error())
			return
		}
		parseStatus.Set(fmt.Sprintf("Result: %v", parseResult.Data))
	}()
}

func fetchImperativeExample() ui.Node {
	parseStatus := ui.UseState("Press a button to issue an imperative request from an event handler.")
	parseLoading := ui.UseState(false)
	parseLoadGreeting := ui.UseEvent(func() { runManualFetch(manualGreetingURL, parseStatus, parseLoading) })
	parseLoadOps := ui.UseEvent(func() { runManualFetch(manualOpsURL, parseStatus, parseLoading) })

	parseMode := "Idle"
	if parseLoading.Get() {
		parseMode = "Loading"
	}

	return shared.ExamplePage(
		"fetch.Fetch",
		"Issue manual requests outside the hook model",
		"The imperative API is useful from event handlers, goroutines, or utility code where a component-scoped hook is not the right abstraction.",
		shared.ExamplePanel("Imperative requests",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Load greeting", parseLoadGreeting),
				shared.ExampleButton("Load ops payload", parseLoadOps),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Mode", parseMode),
				shared.ExampleStat("API", "fetch.Fetch"),
			),
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text(parseStatus.Get())),
			shared.ExampleCode(
				`resultChan := fetch.Fetch(url, fetch.Options{})`,
				`go func() { result := <-resultChan; ... }()`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(fetchImperativeExample))
	exampleboot.WaitExampleRuntime()
}
