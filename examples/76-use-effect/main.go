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

func logEffectAction(parseAction string, parseDetails ...interface{}) {
	parseArgs := append([]interface{}{"[ui.UseEffect demo]", parseAction}, parseDetails...)
	js.Global().Get("console").Call("log", parseArgs...)
}

func useEffectExample() ui.Node {
	parseSelectedMode := ui.UseState("draft")
	parseEffectRuns := ui.UseState(0)
	parseCleanupRuns := ui.UseState(0)
	parseStatus := ui.UseState("Waiting for effect to synchronize state")

	ui.UseEffect(func() func() {
		logEffectAction("mounted", map[string]interface{}{
			"mode": parseSelectedMode.Get(),
		})
		return nil
	}, "effect-demo-mounted")

	setDraft := ui.UseEvent(func() {
		parsePrevious := parseSelectedMode.Get()
		parseSelectedMode.Set("draft")
		logEffectAction("draft mode selected", map[string]interface{}{
			"previousMode": parsePrevious,
			"nextMode":     "draft",
		})
	})
	setReview := ui.UseEvent(func() {
		parsePrevious2 := parseSelectedMode.Get()
		parseSelectedMode.Set("review")
		logEffectAction("review mode selected", map[string]interface{}{
			"previousMode": parsePrevious2,
			"nextMode":     "review",
		})
	})
	setShip := ui.UseEvent(func() {
		parsePrevious3 := parseSelectedMode.Get()
		parseSelectedMode.Set("ship")
		logEffectAction("ship mode selected", map[string]interface{}{
			"previousMode": parsePrevious3,
			"nextMode":     "ship",
		})
	})

	ui.UseEffect(func() func() {
		parseCurrentMode := parseSelectedMode.Get()
		parseNextEffectRun := parseEffectRuns.Get() + 1
		parseEffectRuns.Update(func(parsePrevious4 int) int { return parsePrevious4 + 1 })
		parseStatus.Set("Effect synchronized mode: " + parseCurrentMode)
		logEffectAction("effect setup", map[string]interface{}{
			"mode":        parseCurrentMode,
			"effectRun":   parseNextEffectRun,
			"cleanupRuns": parseCleanupRuns.Get(),
		})

		parseDocument := js.Global().Get("document")
		parsePreviousTitle := ""
		parseNextTitle := "ui.UseEffect demo - " + parseCurrentMode
		if parseDocument.Truthy() {
			parsePreviousTitle = parseDocument.Get("title").String()
			parseDocument.Set("title", parseNextTitle)
			logEffectAction("document title updated", map[string]interface{}{
				"previousTitle": parsePreviousTitle,
				"nextTitle":     parseNextTitle,
			})
		}

		var parseTimeoutFn js.Func
		parseTimeoutFn = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseStatus.Set("Effect timer completed for mode: " + parseCurrentMode)
			logEffectAction("timer completed", map[string]interface{}{
				"mode": parseCurrentMode,
			})
			return nil
		})
		parseTimeoutID := js.Global().Call("setTimeout", parseTimeoutFn, 900)
		logEffectAction("timer scheduled", map[string]interface{}{
			"mode":       parseCurrentMode,
			"delayMs":    900,
			"timeoutRef": parseTimeoutID.Int(),
		})

		return func() {
			js.Global().Call("clearTimeout", parseTimeoutID)
			parseTimeoutFn.Release()
			parseCleanupRuns.Update(func(parsePrevious5 int) int { return parsePrevious5 + 1 })
			logEffectAction("cleanup running", map[string]interface{}{
				"mode":             parseCurrentMode,
				"restoredTitle":    parsePreviousTitle,
				"nextCleanupCount": parseCleanupRuns.Get() + 1,
			})
			if parseDocument.Truthy() {
				parseDocument.Set("title", parsePreviousTitle)
			}
		}
	}, parseSelectedMode.Get())

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
				shared.ExampleStat("Mode", parseSelectedMode.Get()),
				shared.ExampleStat("Effect runs", fmt.Sprintf("%d", parseEffectRuns.Get())),
				shared.ExampleStat("Cleanup runs", fmt.Sprintf("%d", parseCleanupRuns.Get())),
			),
		),
		shared.ExamplePanel("Observed side effect",
			html.Div(html.Props{Class: "mt-3 rounded-2xl border border-white/10 bg-slate-950/45 p-5"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Status")),
				html.P(html.Props{Class: "mt-3 text-xl font-semibold text-white"}, html.Text(parseStatus.Get())),
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
