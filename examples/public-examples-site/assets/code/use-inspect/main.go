//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// inspectExample demonstrates ui.UseInspect (Svelte-style $inspect): it logs a labeled value's
// initial state and every change through a swappable sink. Here SetInspectSink routes those records
// into an on-screen log instead of the console.
func inspectExample() ui.Node {
	parseCount := ui.UseState(0)
	parseLogs := ui.UseState([]string{})

	// Install the sink once: route inspect records into the visible log.
	ui.UseEffect(func() func() {
		return ui.SetInspectSink(func(parseMessage string) {
			parseLogs.Update(func(parsePrev []string) []string {
				return append(append([]string{}, parsePrev...), parseMessage)
			})
		})
	}, "install-inspect-sink")

	// Logs "count: 0" on mount, then "count: <old> -> <new>" on every change.
	ui.UseInspect("count", parseCount.Get())

	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrev int) int { return parsePrev + 1 })
	})

	parseLogNodes := make([]ui.Node, 0, len(parseLogs.Get()))
	for _, parseLine := range parseLogs.Get() {
		parseLogNodes = append(parseLogNodes, html.Div(html.Props{Class: "font-mono text-sm text-emerald-300"}, html.Text(parseLine)))
	}

	return shared.ExamplePage(
		"ui.UseInspect",
		"Svelte-style $inspect through a swappable sink",
		"UseInspect logs a labeled value's initial state and every subsequent change. SetInspectSink swaps where those records go — devtools, a test buffer, the console, or (here) an on-screen panel. In production you can silence it.",
		shared.ExamplePanel("Inspected value",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap items-center gap-4"},
				shared.ExampleButton("Increment", parseIncrement),
				shared.ExampleStat("count", fmt.Sprintf("%d", parseCount.Get())),
			),
			html.Div(html.Props{Class: "mt-6 space-y-1 rounded-lg bg-slate-950 p-4"}, parseLogNodes...),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(inspectExample))
	exampleboot.WaitExampleRuntime()
}
