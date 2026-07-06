// Package hydrationprobe holds a deterministic component shared by the server
// (native RenderToString) and the client (wasm Hydrate) sides of the #37 hydration
// e2e. Because its markup is identical on the server and the client's first render,
// a correct hydration ADOPTS the server DOM with zero fallbacks/discards — which the
// e2e asserts against the framework-reported hydration metrics in a real browser.
package hydrationprobe

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// HydrationProbe renders a small, deterministic tree. It uses an initial UseState
// value rather than anything time/random-derived so the server and client first
// renders are byte-identical.
func HydrationProbe() ui.Node {
	parseCount := ui.UseState(7)
	return html.Div(html.Props{ID: "probe"},
		html.P(html.Props{ID: "probe-label"}, html.Text("Hydration Probe")),
		html.P(html.Props{ID: "probe-count"}, html.Text(fmt.Sprintf("count: %d", parseCount.Get()))),
	)
}
