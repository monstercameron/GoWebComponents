//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func useEffectExample() ui.Node {
	selectedMode := ui.UseState("draft")
	effectRuns := ui.UseState(0)
	cleanupRuns := ui.UseState(0)
	status := ui.UseState("Waiting for effect to synchronize state")

	setDraft := ui.UseEvent(func() { selectedMode.Set("draft") })
	setReview := ui.UseEvent(func() { selectedMode.Set("review") })
	setShip := ui.UseEvent(func() { selectedMode.Set("ship") })

	ui.UseEffect(func() func() {
		effectRuns.Update(func(previous int) int { return previous + 1 })
		status.Set("Effect synchronized mode: " + selectedMode.Get())

		document := js.Global().Get("document")
		previousTitle := ""
		if document.Truthy() {
			previousTitle = document.Get("title").String()
			document.Set("title", "ui.UseEffect demo - "+selectedMode.Get())
		}

		var timeoutFn js.Func
		timeoutFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			status.Set("Effect timer completed for mode: " + selectedMode.Get())
			return nil
		})
		timeoutID := js.Global().Call("setTimeout", timeoutFn, 900)

		return func() {
			js.Global().Call("clearTimeout", timeoutID)
			timeoutFn.Release()
			cleanupRuns.Update(func(previous int) int { return previous + 1 })
			if document.Truthy() {
				document.Set("title", previousTitle)
			}
		}
	}, selectedMode.Get())

	return shared.ExamplePage(
		"ui.UseEffect",
		"Run side effects after render and clean them up when dependencies change",
		"UseEffect is for non-render work like timers, subscriptions, and DOM integration. This demo reruns the effect when the selected mode changes, updates document.title, schedules a timer, and records cleanup runs when dependencies change.",
		shared.ExamplePanel("Dependency-driven effect",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Switch modes to trigger a cleanup for the previous effect and a fresh setup for the new dependency value.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Draft", setDraft),
				shared.ExampleButton("Review", setReview),
				shared.ExampleButton("Ship", setShip),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Mode", selectedMode.Get()),
				shared.ExampleStat("Effect runs", fmt.Sprintf("%d", effectRuns.Get())),
				shared.ExampleStat("Cleanup runs", fmt.Sprintf("%d", cleanupRuns.Get())),
			),
		),
		shared.ExamplePanel("Observed side effect",
			html.Div(html.Props{Class: "mt-3 rounded-2xl border border-white/10 bg-slate-950/45 p-5"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Status")),
				html.P(html.Props{Class: "mt-3 text-xl font-semibold text-white"}, html.Text(status.Get())),
				html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("Open the browser tab title as you switch modes. The effect updates it during setup and restores the previous title during cleanup.")),
			),
			shared.ExampleCode(
				"ui.UseEffect(func() func() {",
				"    timeout := js.Global().Call(\"setTimeout\", callback, 900)",
				"    return func() { js.Global().Call(\"clearTimeout\", timeout) }",
				"}, selectedMode.Get())",
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useEffectExample), "#app")
	select {}
}