//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
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
	return Div(Class("min-h-screen bg-[linear-gradient(180deg,#08111d_0%,#0b1523_100%)] text-slate-100"),
		Div(Class("mx-auto max-w-6xl px-6 py-12"),
			Div(Class("rounded-[2rem] border border-white/10 bg-slate-950/80 p-8 shadow-2xl"),
				P(Class("text-xs uppercase tracking-[0.35em] text-cyan-300"), Text("Static islands")),
				H1(Class("mt-4 text-5xl font-black tracking-tight text-white"), Text("Scoped hydration surfaces")),
				P(Class("mt-4 max-w-4xl text-lg leading-8 text-slate-300"), Text("The standalone example hydrates two narrow islands inside a larger prerendered page. This preview keeps the same island components, but mounts them inside one shared teaching shell so they can still run inside the public examples site.")),
				Div(Class("mt-8 grid gap-6 lg:grid-cols-2"),
					ui.CreateElement(newsletterIsland),
					ui.CreateElement(quoteIsland),
				),
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
