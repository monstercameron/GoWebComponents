package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func ChangedCounterPanel() ui.Node {
	const changedSubtreeVersion = "v1"

	parseCount := ui.UseState(0)
	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrev int) int { return parsePrev + 1 })
	})

	return html.Div(html.Props{Class: "bg-slate-900 p-5 rounded-xl border border-amber-500/40", ID: "changed-panel"},
		html.H2(html.Props{Class: "text-lg font-semibold text-amber-300 mb-2"}, html.Text("Changed sibling subtree")),
		html.P(html.Props{Class: "text-sm text-slate-300 mb-3", ID: "changed-version"}, html.Text("Changed subtree version: "+changedSubtreeVersion)),
		html.P(html.Props{Class: "text-sm text-slate-300 mb-3"}, html.Text("When this component changes, it should remount and lose only its own local state.")),
		html.P(html.Props{Class: "font-mono text-base", ID: "changed-count"}, html.Text(fmt.Sprintf("Changed count: %d", parseCount.Get()))),
		html.Button(html.Props{
			Class:   "mt-3 bg-amber-400 hover:bg-amber-500 text-black font-semibold px-4 py-2 rounded",
			ID:      "changed-increment",
			OnClick: parseIncrement,
		}, html.Text("Increment Changed Counter")),
	)
}
