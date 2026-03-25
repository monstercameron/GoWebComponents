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

func logStateAction(parseAction string, parseDetails ...interface{}) {
	parseArgs := append([]interface{}{"[ui.UseState demo]", parseAction}, parseDetails...)
	js.Global().Get("console").Call("log", parseArgs...)
}

func useStateExample() ui.Node {
	parseCounter := ui.UseState(2)
	parseMessage := ui.UseState("Ship the feature-isolated catalog")

	ui.UseEffect(func() func() {
		logStateAction("mounted", map[string]interface{}{
			"counter": parseCounter.Get(),
			"message": parseMessage.Get(),
		})
		return nil
	}, "state-demo-mounted")

	parseIncrement := ui.UseEvent(func() {
		parsePrevious := parseCounter.Get()
		parseCounter.Update(func(parsePrevious5 int) int { return parsePrevious5 + 1 })
		logStateAction("increment clicked", map[string]interface{}{
			"previousCounter": parsePrevious,
			"nextCounter":     parsePrevious + 1,
		})
	})
	parseDecrement := ui.UseEvent(func() {
		parsePrevious2 := parseCounter.Get()
		parseCounter.Update(func(parsePrevious6 int) int { return parsePrevious6 - 1 })
		logStateAction("decrement clicked", map[string]interface{}{
			"previousCounter": parsePrevious2,
			"nextCounter":     parsePrevious2 - 1,
		})
	})
	reset := ui.UseEvent(func() {
		parsePreviousCounter := parseCounter.Get()
		parsePreviousMessage := parseMessage.Get()
		parseCounter.Set(2)
		parseMessage.Set("Ship the feature-isolated catalog")
		logStateAction("reset clicked", map[string]interface{}{
			"previousCounter": parsePreviousCounter,
			"nextCounter":     2,
			"previousMessage": parsePreviousMessage,
			"nextMessage":     "Ship the feature-isolated catalog",
		})
	})
	setAlpha := ui.UseEvent(func() {
		parsePrevious3 := parseMessage.Get()
		parseMessage.Set("Audit every example page")
		logStateAction("set audit message clicked", map[string]interface{}{
			"previousMessage": parsePrevious3,
			"nextMessage":     "Audit every example page",
		})
	})
	setBeta := ui.UseEvent(func() {
		parsePrevious4 := parseMessage.Get()
		parseMessage.Set("Document the implementation details")
		logStateAction("set docs message clicked", map[string]interface{}{
			"previousMessage": parsePrevious4,
			"nextMessage":     "Document the implementation details",
		})
	})

	return shared.ExamplePage(
		"ui.UseState",
		"Create local reactive state with typed get, set, and update helpers",
		"UseState is the core local state primitive. This example shows direct Set calls for replacing values and Update calls for deriving the next value from the previous state safely.",
		shared.ExamplePanel("State transitions",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("The counter demonstrates numeric update flows, while the message state demonstrates direct replacement of a typed string value.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("-1", parseDecrement),
				shared.ExampleButton("+1", parseIncrement),
				shared.ExampleButton("Reset", reset),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Counter", fmt.Sprintf("%d", parseCounter.Get())),
				shared.ExampleStat("Message length", fmt.Sprintf("%d chars", len(parseMessage.Get()))),
			),
		),
		shared.ExamplePanel("String state",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Set audit message", setAlpha),
				shared.ExampleButton("Set docs message", setBeta),
			),
			html.Div(html.Props{Class: "mt-6 rounded-2xl border border-white/10 bg-slate-950/45 p-5"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Current message")),
				html.P(html.Props{Class: "mt-3 text-xl font-semibold text-white"}, html.Text(parseMessage.Get())),
			),
			shared.ExampleCode(
				"counter := ui.UseState(2)",
				"counter.Set(10)",
				"counter.Update(func(previous int) int { return previous + 1 })",
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useStateExample), "#app")
	select {}
}
