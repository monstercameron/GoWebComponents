//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v5/examples/shared"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/state"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

const hydrationTopicAtom = "catalog-hydrate-topic"

func statCard(parseLabel, parseValue string) ui.Node {
	return shared.ExampleStat(parseLabel, parseValue)
}

func actionButton(parseLabel string, parseHandler ui.Handler) ui.Node {
	return shared.ExampleButton(parseLabel, parseHandler)
}

func hydrateExample() ui.Node {
	parseTopic := state.UseAtom(hydrationTopicAtom, "client-fallback")
	parseCount := ui.UseState(2)
	parseIncrement := ui.UseEvent(func() { parseCount.Update(func(parsePrevious int) int { return parsePrevious + 1 }) })
	parseToggleTopic := ui.UseEvent(func() {
		if parseTopic.Get() == "server-rendered" {
			parseTopic.Set("client-hydrated")
			return
		}
		parseTopic.Set("server-rendered")
	})

	return shared.ExamplePage(
		"Hydration",
		"ui.Hydrate",
		"Resume prerendered HTML, restore bootstrap state, and keep the mounted tree interactive.",
		shared.ExamplePanel("Hydrated State",
			html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-2"},
				statCard("Bootstrap Topic", parseTopic.Get()),
				statCard("Counter", fmt.Sprintf("%d", parseCount.Get())),
			),
		),
		shared.ExamplePanel("Actions",
			html.Div(html.Props{Class: "flex flex-wrap gap-2"},
				actionButton("Increment Counter", parseIncrement),
				actionButton("Toggle Topic", parseToggleTopic),
			),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("The HTML arrived before wasm. These controls prove the resumed tree stayed live instead of remounting from scratch.")),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	if _, parseErr := exampleboot.ApplyExampleHydration(ui.CreateElement(hydrateExample), ui.HydrationOptions{ScriptID: ui.DefaultBootstrapScriptID}); parseErr != nil {
		exampleboot.RenderExampleRoot(ui.CreateElement(hydrateExample))
	}
	exampleboot.WaitExampleRuntime()
}
