//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

func rawHandlerExample() ui.Node {
	parseWrappedCount := ui.UseState(0)
	parseRawCount := ui.UseState(0)
	parseLastMode := ui.UseState("No clicks yet")

	parseWrapped := ui.UseEvent(func() {
		parseWrappedCount.Update(func(parsePrevious int) int { return parsePrevious + 1 })
		parseLastMode.Set("ui.UseEvent")
	})
	parsePrebuiltRaw := ui.UseEvent(func() {
		parseRawCount.Update(func(parsePrevious2 int) int { return parsePrevious2 + 1 })
		parseLastMode.Set("ui.WrapHandler")
	})
	parseRaw := ui.WrapHandler(parsePrebuiltRaw.Value())

	return shared.ExamplePage(
		"ui.WrapHandler",
		"Forward a raw function value instead of using the event hook wrapper",
		"WrapHandler exists for edge cases and interop when you already have a handler value and need to pass it through unchanged. The preferred default is still ui.UseEvent.",
		shared.ExamplePanel("Wrapped versus raw",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Both buttons work. The left button passes the handler directly from ui.UseEvent. The right button forwards an already-created handler value through ui.WrapHandler without wrapping it again.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Increment via ui.UseEvent", parseWrapped),
				html.Button(html.Props{OnClick: parseRaw, Class: "rounded-full border border-amber-500/40 bg-amber-500/10 px-5 py-3 font-semibold text-amber-100 hover:bg-amber-500/20"}, html.Text("Increment via ui.WrapHandler")),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Wrapped clicks", fmt.Sprintf("%d", parseWrappedCount.Get())),
				shared.ExampleStat("Raw clicks", fmt.Sprintf("%d", parseRawCount.Get())),
				shared.ExampleStat("Last path", parseLastMode.Get()),
			),
			shared.ExampleCode(
				`preferred := ui.UseEvent(func() { ... })`,
				`escapeHatch := ui.WrapHandler(preferred.Value())`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(rawHandlerExample))
	exampleboot.WaitExampleRuntime()
}
