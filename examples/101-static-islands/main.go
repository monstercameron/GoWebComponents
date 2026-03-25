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
	performance := js.Global().Get("performance")
	if performance.Truthy() {
		return performance.Call("now").Float()
	}
	return 0
}

func writeMetric(elementID, value string) {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	node := document.Call("getElementById", elementID)
	if !node.Truthy() {
		return
	}
	node.Set("textContent", value)
}

func hydrateIsland(name string, selector string, node ui.Node) {
	_, _ = ui.Hydrate(node, selector, ui.HydrationOptions{
		Observability: ui.SSRObservabilityOptions{
			CorrelationID: "static-islands:" + name,
			OnEvent: func(event ui.SSRObservation) {
				if event.Hydration == nil {
					return
				}
				writeMetric("metric-"+name+"-hydration", fmt.Sprintf("%.2f ms", float64(event.Hydration.DurationNs)/1_000_000.0))
			},
		},
	})
}

func main() {
	utils.DisableAllDebug()
	started := nowMillis()
	hydrateIsland("newsletter", "#newsletter-island", ui.CreateElement(newsletterIsland))
	hydrateIsland("quote", "#quote-island", ui.CreateElement(quoteIsland))
	writeMetric("metric-startup-total", fmt.Sprintf("%.2f ms", nowMillis()-started))
	select {}
}
