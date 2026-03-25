//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func nowMillis() float64 {
	parsePerformance := js.Global().Get("performance")
	if parsePerformance.Truthy() {
		return parsePerformance.Call("now").Float()
	}
	return 0
}

func writeMetric(parseElementID, parseValue string) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseNode := parseDocument.Call("getElementById", parseElementID)
	if !parseNode.Truthy() {
		return
	}
	parseNode.Set("textContent", parseValue)
}

func hydrateIsland(parseName string, parseSelector string, parseNode ui.Node) {
	_, _ = ui.Hydrate(parseNode, parseSelector, ui.HydrationOptions{
		Observability: ui.SSRObservabilityOptions{
			CorrelationID: "static-islands:" + parseName,
			OnEvent: func(parseEvent ui.SSRObservation) {
				if parseEvent.Hydration == nil {
					return
				}
				writeMetric("metric-"+parseName+"-hydration", fmt.Sprintf("%.2f ms", float64(parseEvent.Hydration.DurationNs)/1_000_000.0))
			},
		},
	})
}

func main() {
	utils.DisableAllDebug()
	parseStarted := nowMillis()
	hydrateIsland("newsletter", "#newsletter-island", ui.CreateElement(newsletterIsland))
	hydrateIsland("quote", "#quote-island", ui.CreateElement(quoteIsland))
	writeMetric("metric-startup-total", fmt.Sprintf("%.2f ms", nowMillis()-parseStarted))
	select {}
}
