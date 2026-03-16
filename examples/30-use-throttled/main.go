//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func useThrottledExample() ui.Node {
	value := ui.UseState(0)
	throttled := ui.UseThrottled(value.Get(), 400*time.Millisecond)
	increment := ui.UseEvent(func() { value.Update(func(prev int) int { return prev + 1 }) })

	status := "Synced"
	if throttled.Pending() {
		status = "Rate limited"
	}

	return shared.ExamplePage(
		"ui.UseThrottled",
		"Rate-limit a fast-changing derived value",
		"Throttle is useful when rapid updates should still flow regularly, just not on every single event.",
		shared.ExamplePanel("Throttled counter",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-4"},
				shared.ExampleButton("Increment quickly", increment),
				shared.ExampleStat("Immediate", fmt.Sprintf("%d", value.Get())),
				shared.ExampleStat("Throttled", fmt.Sprintf("%d", throttled.Get())),
				shared.ExampleStat("State", status),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useThrottledExample), "#app")
	select {}
}
