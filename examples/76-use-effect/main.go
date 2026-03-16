//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func logEffectAction(action string, details ...interface{}) {
	args := append([]interface{}{"[ui.UseEffect demo]", action}, details...)
	js.Global().Get("console").Call("log", args...)
}

func useEffectExample() ui.Node {
	selectedMode := ui.UseState("draft")
	effectRuns := ui.UseState(0)
	cleanupRuns := ui.UseState(0)
	status := ui.UseState("Waiting for effect to synchronize state")

	ui.UseEffect(func() func() {
		logEffectAction("mounted", map[string]interface{}{
			"mode": selectedMode.Get(),
		})
		return nil
	}, "effect-demo-mounted")

	setDraft := ui.UseEvent(func() {
		previous := selectedMode.Get()
		selectedMode.Set("draft")
		logEffectAction("draft mode selected", map[string]interface{}{
			"previousMode": previous,
			"nextMode":     "draft",
		})
	})
	setReview := ui.UseEvent(func() {
		previous := selectedMode.Get()
		selectedMode.Set("review")
		logEffectAction("review mode selected", map[string]interface{}{
			"previousMode": previous,
			"nextMode":     "review",
		})
	})
	setShip := ui.UseEvent(func() {
		previous := selectedMode.Get()
		selectedMode.Set("ship")
		logEffectAction("ship mode selected", map[string]interface{}{
			"previousMode": previous,
			"nextMode":     "ship",
		})
	})

	ui.UseEffect(func() func() {
		currentMode := selectedMode.Get()
		nextEffectRun := effectRuns.Get() + 1
		effectRuns.Update(func(previous int) int { return previous + 1 })
		status.Set("Effect synchronized mode: " + currentMode)
		logEffectAction("effect setup", map[string]interface{}{
			"mode":        currentMode,
			"effectRun":   nextEffectRun,
			"cleanupRuns": cleanupRuns.Get(),
		})

		document := js.Global().Get("document")
		previousTitle := ""
		nextTitle := "ui.UseEffect demo - " + currentMode
		if document.Truthy() {
			previousTitle = document.Get("title").String()
			document.Set("title", nextTitle)
			logEffectAction("document title updated", map[string]interface{}{
				"previousTitle": previousTitle,
				"nextTitle":     nextTitle,
			})
		}

		var timeoutFn js.Func
		timeoutFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			status.Set("Effect timer completed for mode: " + currentMode)
			logEffectAction("timer completed", map[string]interface{}{
				"mode": currentMode,
			})
			return nil
		})
		timeoutID := js.Global().Call("setTimeout", timeoutFn, 900)
		logEffectAction("timer scheduled", map[string]interface{}{
			"mode":       currentMode,
			"delayMs":    900,
			"timeoutRef": timeoutID.Int(),
		})

		return func() {
			js.Global().Call("clearTimeout", timeoutID)
			timeoutFn.Release()
			cleanupRuns.Update(func(previous int) int { return previous + 1 })
			logEffectAction("cleanup running", map[string]interface{}{
				"mode":             currentMode,
				"restoredTitle":    previousTitle,
				"nextCleanupCount": cleanupRuns.Get() + 1,
			})
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
