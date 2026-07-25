//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

func useCallbackExample() ui.Node {
	parseTotal := ui.UseState(12)
	parseStep := ui.UseState(2)
	parseDraft := ui.UseState("")
	parseRenders := ui.UseRef(0)
	parseRenders.Set(parseRenders.Get() + 1)

	parseIncrement := ui.UseCallback(func() {
		parseTotal.Update(func(parsePrevious int) int {
			return parsePrevious + parseStep.Get()
		})
	}, parseStep.Get())

	parseCallbackRefreshes := ui.UseState(1)
	parseLastStep := ui.UseRef(parseStep.Get())
	ui.UseEffect(func() func() {
		parseCurrentStep := parseStep.Get()
		if parseCurrentStep != parseLastStep.Get() {
			parseLastStep.Set(parseCurrentStep)
			parseCallbackRefreshes.Update(func(parsePrevious2 int) int {
				return parsePrevious2 + 1
			})
		}
		return nil
	}, parseIncrement)

	parseUpdateDraft := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseDraft.Set(parseEvent.GetValue())
	})
	parseChangeStep := ui.UseEvent(func(parseEvent2 ui.InputEvent) {
		parseNext := 1
		fmt.Sscanf(parseEvent2.GetValue(), "%d", &parseNext)
		if parseNext < 1 {
			parseNext = 1
		}
		if parseNext > 9 {
			parseNext = 9
		}
		parseStep.Set(parseNext)
	})
	parseRunIncrement := ui.UseEvent(func(parseEvent3 ui.MouseEvent) {
		parseEvent3.PreventDefault()
		parseIncrement()
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
						Value:       parseDraft.Get(),
						OnInput:     parseUpdateDraft,
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
						Value:   fmt.Sprintf("%d", parseStep.Get()),
						OnInput: parseChangeStep,
						Class:   "mt-2 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100",
					}),
				),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Current total", fmt.Sprintf("%d", parseTotal.Get())),
				shared.ExampleStat("Component renders", fmt.Sprintf("%d", parseRenders.Get())),
				shared.ExampleStat("UseCallback refreshes", fmt.Sprintf("%d", parseCallbackRefreshes.Get())),
				shared.ExampleStat("Inline handler policy", "recreated every render"),
			),
			html.P(html.Props{Class: "mt-5 text-sm leading-7 text-slate-300"}, html.Text("Typing in the draft box rerenders the component, but the memoized increment callback stays stable because its only dependency is step. Changing step is the moment when UseCallback refreshes the closure.")),
			html.Div(html.Props{Class: "mt-5 flex flex-wrap gap-3"},
				shared.ExampleButton("Increment total", parseRunIncrement),
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
	exampleboot.RenderExampleRoot(ui.CreateElement(useCallbackExample))
	exampleboot.WaitExampleRuntime()
}
