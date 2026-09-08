//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

var catalog = []string{
	"router hydration", "render pipeline", "memoized search", "typed events", "portal overlays",
	"derived atoms", "lazy subtree", "error boundary", "transition lane", "resource cache",
}

func useDeferredValueExample() ui.Node {
	parseQuery := ui.UseState("")
	parseDeferred := ui.UseDeferredValue(parseQuery.Get())
	parseUpdate := ui.UseEvent(func(parseEvent ui.InputEvent) { parseQuery.Set(parseEvent.GetValue()) })

	parseResults := make([]ui.Node, 0, len(catalog))
	for _, parseItem := range catalog {
		if parseDeferred != "" && !strings.Contains(parseItem, strings.ToLower(parseDeferred)) {
			continue
		}
		parseResults = append(parseResults, html.Li(html.Props{Class: "rounded-xl border border-white/10 bg-slate-950/50 px-4 py-3"}, html.Text(parseItem)))
	}

	return shared.ExamplePage(
		"ui.UseDeferredValue",
		"Keep rendering the last committed value until a transition catches up",
		"The input changes immediately, but the result list follows the deferred value so fast updates do not need to block the rest of the UI.",
		shared.ExamplePanel("Deferred search",
			html.Input(html.Props{Value: parseQuery.Get(), OnInput: parseUpdate, Placeholder: "Filter the feature list", Class: "mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Immediate", parseQuery.Get()),
				shared.ExampleStat("Deferred", parseDeferred),
			),
			html.P(html.Props{Class: "mt-6 text-slate-400"}, html.Text(fmt.Sprintf("Showing %d results from the deferred value", len(parseResults)))),
			html.Ul(html.Props{Class: "mt-4 grid gap-3"}, parseResults...),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(useDeferredValueExample))
	exampleboot.WaitExampleRuntime()
}
