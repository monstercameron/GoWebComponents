//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	"github.com/monstercameron/GoWebComponents/hotreload"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func StableCounterPanel() ui.Node {
	parseCount := ui.UseState(0)
	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrev int) int { return parsePrev + 1 })
	})

	return html.Div(html.Props{Class: "bg-slate-900 p-5 rounded-xl border border-emerald-500/40", ID: "stable-panel"},
		html.H2(html.Props{Class: "text-lg font-semibold text-emerald-300 mb-2"}, html.Text("Stable sibling subtree")),
		html.P(html.Props{Class: "text-sm text-slate-300 mb-3"}, html.Text("This subtree should preserve its local state when a different sibling component changes.")),
		html.P(html.Props{Class: "font-mono text-base", ID: "stable-count"}, html.Text(fmt.Sprintf("Stable count: %d", parseCount.Get()))),
		html.Button(html.Props{
			Class:   "mt-3 bg-emerald-500 hover:bg-emerald-600 text-black font-semibold px-4 py-2 rounded",
			ID:      "stable-increment",
			OnClick: parseIncrement,
		}, html.Text("Increment Stable Counter")),
	)
}

// App demonstrates selective preserve/remount hot reload behavior.
func App() ui.Node {
	return html.Div(html.Props{Class: "p-8 space-y-6"},
		html.Div(html.Props{Class: "space-y-3"},
			html.H1(html.Props{Class: "text-2xl font-bold", ID: "hot-reload-heading"}, html.Text("Hot Reload Development Server")),
			html.P(html.Props{Class: "text-slate-300 max-w-3xl"},
				html.Text("Edit only the changed sibling component in this file while the standalone dev server is running. The stable sibling should preserve its counter state, while the changed sibling should remount and reset its local state after the rebuild."),
			),
		),
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-2"},
			ui.CreateElement(StableCounterPanel),
			ui.CreateElement(ChangedCounterPanel),
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
