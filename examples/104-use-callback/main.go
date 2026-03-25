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

func useCallbackExample() ui.Node {
	total := ui.UseState(12)
	step := ui.UseState(2)
	draft := ui.UseState("")
	renders := ui.UseRef(0)
	renders.Set(renders.Get() + 1)

	increment := ui.UseCallback(func() {
		total.Update(func(previous int) int {
			return previous + step.Get()
		})
	}, step.Get())

	callbackRefreshes := ui.UseState(1)
	lastStep := ui.UseRef(step.Get())
	ui.UseEffect(func() func() {
		currentStep := step.Get()
		if currentStep != lastStep.Get() {
			lastStep.Set(currentStep)
			callbackRefreshes.Update(func(previous int) int {
				return previous + 1
			})
		}
		return nil
	}, increment)

	updateDraft := ui.UseEvent(func(event ui.InputEvent) {
		draft.Set(event.GetValue())
	})
	changeStep := ui.UseEvent(func(event ui.InputEvent) {
		next := 1
		fmt.Sscanf(event.GetValue(), "%d", &next)
		if next < 1 {
			next = 1
		}
		if next > 9 {
			next = 9
		}
		step.Set(next)
	})
	runIncrement := ui.UseEvent(func(event ui.MouseEvent) {
		event.PreventDefault()
		increment()
	})

	return shared.ExamplePage(
		"ui.UseCallback",
		"Keep a callback stable until its real dependencies change",
		"UseCallback is for child props, subscriptions, and effect dependencies that should not refresh on unrelated rerenders. Keep ordinary inline handlers for simple local wiring, and wrap DOM events with ui.UseEvent when the callback needs an event argument.",
		shared.ExamplePanel("Callback stability",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
				html.Div(html.Props{},
					html.Label(html.Props{For: "use-callback-draft", Class: "text-sm font-semibold text-slate-200"}, html.Text("Unrelated draft text")),
					html.Input(html.Props{
						ID:          "use-callback-draft",
						Value:       draft.Get(),
						OnInput:     updateDraft,
						Placeholder: "Type here to rerender without changing step",
						Class:       "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
					}),
				),
				html.Div(html.Props{},
					html.Label(html.Props{For: "use-callback-step", Class: "text-sm font-semibold text-slate-200"}, html.Text("Step dependency")),
					html.Input(html.Props{
						ID:      "use-callback-step",
						Type:    "number",
						Min:     "1",
						Max:     "9",
						Value:   fmt.Sprintf("%d", step.Get()),
						OnInput: changeStep,
						Class:   "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
					}),
				),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Current total", fmt.Sprintf("%d", total.Get())),
				shared.ExampleStat("Component renders", fmt.Sprintf("%d", renders.Get())),
				shared.ExampleStat("UseCallback refreshes", fmt.Sprintf("%d", callbackRefreshes.Get())),
				shared.ExampleStat("Inline handler policy", "recreated every render"),
			),
			html.P(html.Props{Class: "mt-5 text-sm leading-7 text-slate-300"}, html.Text("Typing in the draft box rerenders the component, but the memoized increment callback stays stable because its only dependency is step. Changing step is the moment when UseCallback refreshes the closure.")),
			html.Div(html.Props{Class: "mt-5 flex flex-wrap gap-3"},
				shared.ExampleButton("Increment total", runIncrement),
			),
		),
		shared.ExamplePanel("Where each tool fits",
			html.Ul(html.Props{Class: "mt-3 grid gap-3 text-sm leading-7 text-slate-300"},
				html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("Use an ordinary inline closure when the handler is local, cheap, and no child or effect depends on its identity.")),
				html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("Use ui.UseCallback(...) when a child prop, subscription, or effect dependency should stay stable until specific values change.")),
				html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("Use ui.UseEvent(...) for DOM handlers so the callback has the right event type and event lifecycle boundary.")),
			),
			shared.ExampleCode(
				`increment := ui.UseCallback(func() { total.Update(func(previous int) int { return previous + step.Get() }) }, step.Get())`,
				`onClick := ui.UseEvent(func(event ui.MouseEvent) { increment() })`,
				`html.Button(html.Props{OnClick: onClick}, html.Text("Increment"))`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useCallbackExample), "#app")
	select {}
}
