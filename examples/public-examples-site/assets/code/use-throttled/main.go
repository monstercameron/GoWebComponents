//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func useThrottledExample() ui.Node {
	parseValue := ui.UseState(0)
	parseThrottled := ui.UseThrottled(parseValue.Get(), 400*time.Millisecond)
	parseIncrement := ui.UseEvent(func() { parseValue.Update(func(parsePrev int) int { return parsePrev + 1 }) })

	parseStatus := "Synced"
	if parseThrottled.Pending() {
		parseStatus = "Rate limited"
	}

	return shared.ExamplePage(
		"ui.UseThrottled",
		"Rate-limit a fast-changing derived value",
		"Throttle is useful when rapid updates should still flow regularly, just not on every single event.",
		shared.ExamplePanel("Throttled counter",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-4"},
				shared.ExampleButton("Increment quickly", parseIncrement),
				shared.ExampleStat("Immediate", fmt.Sprintf("%d", parseValue.Get())),
				shared.ExampleStat("Throttled", fmt.Sprintf("%d", parseThrottled.Get())),
				shared.ExampleStat("State", parseStatus),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(useThrottledExample))
	exampleboot.WaitExampleRuntime()
}
