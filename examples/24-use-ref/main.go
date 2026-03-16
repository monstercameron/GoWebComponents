//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func useRefExample() ui.Node {
	visible := ui.UseState(0)
	counterRef := ui.UseRef(visible.Get())
	renders := ui.UseRef(0)
	renders.Set(renders.Get() + 1)

	incrementRef := ui.UseEvent(func() {
		counterRef.Set(counterRef.Get() + 1)
	})
	syncRef := ui.UseEvent(func() {
		visible.Set(counterRef.Get())
	})

	return shared.ExamplePage(
		"ui.UseRef",
		"Persistent mutable values without forcing rerenders",
		"The first button mutates a ref only, so the visible state does not change. The second button copies the ref value into state so the UI catches up on demand. The render counter also lives in a ref.",
		shared.ExamplePanel("Mutable ref",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Press 'Increment ref only' a few times. Nothing visible changes until you explicitly sync the ref into component state.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-4"},
				shared.ExampleButton("Increment ref only", incrementRef),
				shared.ExampleButton("Sync ref to view", syncRef),
				shared.ExampleStat("Visible count", fmt.Sprintf("%d", visible.Get())),
				shared.ExampleStat("Ref count", fmt.Sprintf("%d", counterRef.Get())),
				shared.ExampleStat("Render count", fmt.Sprintf("%d", renders.Get())),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useRefExample), "#app")
	select {}
}
