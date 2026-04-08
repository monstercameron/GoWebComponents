//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func renderExample() ui.Node {
	parseClicks := ui.UseState(0)
	parseIncrement := ui.UseEvent(func() {
		parseClicks.Update(func(parsePrev int) int { return parsePrev + 1 })
	})

	return shared.ExamplePage(
		"ui.Render",
		"ui.Render(ui.CreateElement(App), exampleboot.GetExampleMountSelector())",
		"This is the smallest browser-mount flow: define a component, wrap it with ui.CreateElement, and mount it into the current example host selector with ui.Render.",
		shared.ExamplePanel("Mounted App",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("The button below works only because main mounted this component tree with ui.Render.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-4"},
				shared.ExampleButton("Increment mounted state", parseIncrement),
				shared.ExampleStat("Clicks", fmt.Sprintf("%d", parseClicks.Get())),
			),
		),
		shared.ExamplePanel("main()",
			shared.ExampleCode(
				"func main() {",
				"    exampleboot.RenderExampleRoot(ui.CreateElement(renderExample))",
				"    exampleboot.WaitExampleRuntime()",
				"}",
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(renderExample))
	exampleboot.WaitExampleRuntime()
}
