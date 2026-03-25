//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func useRefExample() ui.Node {
	parseVisible := ui.UseState(0)
	parseCounterRef := ui.UseRef(parseVisible.Get())
	parseRenders := ui.UseRef(0)
	parseRenders.Set(parseRenders.Get() + 1)

	parseIncrementRef := ui.UseEvent(func() {
		parseCounterRef.Set(parseCounterRef.Get() + 1)
	})
	parseSyncRef := ui.UseEvent(func() {
		parseVisible.Set(parseCounterRef.Get())
	})

	return shared.ExamplePage(
		"ui.UseRef",
		"Persistent mutable values without forcing rerenders",
		"The first button mutates a ref only, so the visible state does not change. The second button copies the ref value into state so the UI catches up on demand. The render counter also lives in a ref.",
		shared.ExamplePanel("Mutable ref",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Press 'Increment ref only' a few times. Nothing visible changes until you explicitly sync the ref into component state.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-4"},
				shared.ExampleButton("Increment ref only", parseIncrementRef),
				shared.ExampleButton("Sync ref to view", parseSyncRef),
				shared.ExampleStat("Visible count", fmt.Sprintf("%d", parseVisible.Get())),
				shared.ExampleStat("Ref count", fmt.Sprintf("%d", parseCounterRef.Get())),
				shared.ExampleStat("Render count", fmt.Sprintf("%d", parseRenders.Get())),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useRefExample), "#app")
	select {}
}
