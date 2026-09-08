//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

func typedEventsExample() ui.Node {
	parseMessage := ui.UseState("No events yet")
	parseText := ui.UseState("")

	parseOnClick := ui.UseEvent(func(parseEvent ui.MouseEvent) {
		parseEvent.PreventDefault()
		parseMessage.Set("MouseEvent handled")
	})
	parseOnInput := ui.UseEvent(func(parseEvent2 ui.InputEvent) {
		parseText.Set(parseEvent2.GetValue())
		parseMessage.Set("InputEvent: value captured")
	})
	parseOnKeyDown := ui.UseEvent(func(parseEvent3 ui.KeyboardEvent) {
		parseMessage.Set("KeyboardEvent: key=" + parseEvent3.GetKey())
	})
	parseOnSubmit := ui.UseEvent(func(parseEvent4 ui.FormEvent) {
		parseEvent4.PreventDefault()
		parseMessage.Set("FormEvent: submit prevented")
	})

	return shared.ExamplePage(
		"Typed Events",
		"Use ui.UseEvent with the right public event type",
		"The same helper wraps different event kinds, but the handler signatures stay explicit and readable for the target interaction.",
		shared.ExamplePanel("Event handlers",
			html.Form(html.Props{OnSubmit: parseOnSubmit, Class: "mt-3 grid gap-4"},
				html.Input(html.Props{Value: parseText.Get(), OnInput: parseOnInput, OnKeyDown: parseOnKeyDown, Placeholder: "Type and press keys", Class: "w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
				html.Div(html.Props{Class: "flex flex-wrap gap-3"},
					html.Button(html.Props{OnClick: parseOnClick, Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80", Type: "button"}, html.Text("Click handler")),
					html.Button(html.Props{Type: "submit", Class: "rounded-full border border-white/10 bg-white/5 px-5 py-3 font-semibold text-white hover:bg-white/10"}, html.Text("Submit form")),
				),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Typed value", parseText.Get()),
				shared.ExampleStat("Last event", parseMessage.Get()),
			),
			shared.ExampleCode(
				"ui.UseEvent(func(event ui.InputEvent) { ... })",
				"ui.UseEvent(func(event ui.KeyboardEvent) { ... })",
				"ui.UseEvent(func(event ui.FormEvent) { ... })",
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(typedEventsExample))
	exampleboot.WaitExampleRuntime()
}
