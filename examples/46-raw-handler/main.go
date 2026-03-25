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

func rawHandlerExample() ui.Node {
	wrappedCount := ui.UseState(0)
	rawCount := ui.UseState(0)
	lastMode := ui.UseState("No clicks yet")

	wrapped := ui.UseEvent(func() {
		wrappedCount.Update(func(previous int) int { return previous + 1 })
		lastMode.Set("ui.UseEvent")
	})
	prebuiltRaw := ui.UseEvent(func() {
		rawCount.Update(func(previous int) int { return previous + 1 })
		lastMode.Set("ui.WrapHandler")
	})
	raw := ui.WrapHandler(prebuiltRaw.Value())

	return shared.ExamplePage(
		"ui.WrapHandler",
		"Forward a raw function value instead of using the event hook wrapper",
		"WrapHandler exists for edge cases and interop when you already have a handler value and need to pass it through unchanged. The preferred default is still ui.UseEvent.",
		shared.ExamplePanel("Wrapped versus raw",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Both buttons work. The left button passes the handler directly from ui.UseEvent. The right button forwards an already-created handler value through ui.WrapHandler without wrapping it again.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Increment via ui.UseEvent", wrapped),
				html.Button(html.Props{OnClick: raw, Class: "rounded-full border border-amber-500/40 bg-amber-500/10 px-5 py-3 font-semibold text-amber-100 hover:bg-amber-500/20"}, html.Text("Increment via ui.WrapHandler")),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Wrapped clicks", fmt.Sprintf("%d", wrappedCount.Get())),
				shared.ExampleStat("Raw clicks", fmt.Sprintf("%d", rawCount.Get())),
				shared.ExampleStat("Last path", lastMode.Get()),
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
	ui.Render(ui.CreateElement(rawHandlerExample), "#app")
	select {}
}
