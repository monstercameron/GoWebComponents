//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/hotreload"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func StableCounterPanel() ui.Node {
	parseCount := ui.UseState(0)
	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrev int) int { return parsePrev + 1 })
	})

	return html.Div(html.Props{Class: "rounded-[20px] border border-emerald-500/20 bg-emerald-500/10 p-5", ID: "stable-panel"},
		html.H2(html.Props{Class: "mb-2 text-lg font-semibold text-emerald-100"}, html.Text("Stable sibling subtree")),
		html.P(html.Props{Class: "mb-3 text-sm text-emerald-50/80"}, html.Text("This subtree should preserve its local state when a different sibling component changes.")),
		html.P(html.Props{Class: "font-mono text-base", ID: "stable-count"}, html.Text(fmt.Sprintf("Stable count: %d", parseCount.Get()))),
		html.Button(html.Props{
			Class:   "mt-3 rounded-2xl border border-emerald-300/30 bg-emerald-300/15 px-4 py-2 text-sm font-medium text-emerald-100 transition-all duration-200 hover:-translate-y-0.5 hover:bg-emerald-300/20 active:translate-y-0 active:scale-95",
			ID:      "stable-increment",
			OnClick: parseIncrement,
		}, html.Text("Increment Stable Counter")),
	)
}

// App demonstrates selective preserve/remount hot reload behavior.
func App() ui.Node {
	return shared.ExamplePage(
		"Hot Reload",
		"hotreload.Enable",
		"Change one sibling component during development and compare preserved state versus remounted state after rebuild.",
		shared.ExamplePanel("Behavior",
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text("The stable sibling should keep its count. The edited sibling should remount and reset after the hot-reload rebuild.")),
		),
		shared.ExamplePanel("Compare Panels",
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-2"},
				ui.CreateElement(StableCounterPanel),
				ui.CreateElement(ChangedCounterPanel),
			),
		),
	)
}

func main() {
	hotreload.Enable()

	parseApp := ui.CreateElement(App)
	exampleboot.RenderExampleRoot(parseApp)

	// Keep the Go program running
	exampleboot.WaitExampleRuntime()
}
