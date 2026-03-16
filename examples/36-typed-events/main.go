//go:build js && wasm
// +build js,wasm

package main

import (
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func typedEventsExample() ui.Node {
	message := ui.UseState("No events yet")
	text := ui.UseState("")

	onClick := ui.UseEvent(func(event ui.MouseEvent) {
		event.PreventDefault()
		message.Set("MouseEvent handled")
	})
	onInput := ui.UseEvent(func(event ui.InputEvent) {
		text.Set(event.GetValue())
		message.Set("InputEvent: value captured")
	})
	onKeyDown := ui.UseEvent(func(event ui.KeyboardEvent) {
		message.Set("KeyboardEvent: key=" + event.GetKey())
	})
	onSubmit := ui.UseEvent(func(event ui.FormEvent) {
		event.PreventDefault()
		message.Set("FormEvent: submit prevented")
	})

	return shared.ExamplePage(
		"Typed Events",
		"Use ui.UseEvent with the right public event type",
		"The same helper wraps different event kinds, but the handler signatures stay explicit and readable for the target interaction.",
		shared.ExamplePanel("Event handlers",
			html.Form(html.Props{OnSubmit: onSubmit, Class: "mt-3 grid gap-4"},
				html.Input(html.Props{Value: text.Get(), OnInput: onInput, OnKeyDown: onKeyDown, Placeholder: "Type and press keys", Class: "w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
				html.Div(html.Props{Class: "flex flex-wrap gap-3"},
					html.Button(html.Props{OnClick: onClick, Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80", Type: "button"}, html.Text("Click handler")),
					html.Button(html.Props{Type: "submit", Class: "rounded-full border border-white/10 bg-white/5 px-5 py-3 font-semibold text-white hover:bg-white/10"}, html.Text("Submit form")),
				),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Typed value", text.Get()),
				shared.ExampleStat("Last event", message.Get()),
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
	ui.Render(ui.CreateElement(typedEventsExample), "#app")
	select {}
}
