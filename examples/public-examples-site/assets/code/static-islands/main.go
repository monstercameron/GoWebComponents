//go:build js && wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
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

// hasStaticIslandHosts reports whether the standalone prerendered island containers exist in the current document.
func hasStaticIslandHosts() bool {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return false
	}
	return parseDocument.Call("getElementById", "newsletter-island").Truthy() && parseDocument.Call("getElementById", "quote-island").Truthy()
}

// renderStaticIslandsPreviewRoot renders a learning-friendly fallback when the dedicated prerendered island hosts are unavailable.
func renderStaticIslandsPreviewRoot() ui.Node {
	return shared.ExamplePage(
		"Static Islands",
		"ui.Hydrate",
		"Hydrate two narrow interactive islands inside a larger prerendered page instead of waking the whole document.",
		shared.ExamplePanel("Islands",
			Div(ClassStr("grid gap-6 lg:grid-cols-2"),
				ui.CreateElement(newsletterIsland),
				ui.CreateElement(quoteIsland),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	if exampleboot.HasExampleMountSelector() || !hasStaticIslandHosts() {
		exampleboot.RenderExampleRoot(ui.CreateElement(renderStaticIslandsPreviewRoot))
		exampleboot.WaitExampleRuntime()
		return
	}
	parseStarted := nowMillis()
	hydrateIsland("newsletter", "#newsletter-island", ui.CreateElement(newsletterIsland))
	hydrateIsland("quote", "#quote-island", ui.CreateElement(quoteIsland))
	writeMetric("metric-startup-total", fmt.Sprintf("%.2f ms", nowMillis()-parseStarted))
	exampleboot.WaitExampleRuntime()
}
