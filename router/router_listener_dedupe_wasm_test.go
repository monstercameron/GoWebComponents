//go:build js && wasm

package router

import (
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// TestListenerRenderDedupesPopstateAndHashchange pins #82 finding (3): a single hash
// navigation under a history router fires BOTH popstate and hashchange; the router
// must render once for that pair, not twice. A later navigation to a different URL
// still renders (the dedup only collapses the immediate same-URL double-fire).
func TestListenerRenderDedupesPopstateAndHashchange(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseRouter := NewHistoryRouter(RouterOptions{})
	parseT.Cleanup(func() { parseRouter.teardownHistoryListener() })

	parseRenders := 0
	parseComponent := func(parseAttrs Attrs) *Element {
		parseRenders++
		return runtime.Div(nil, runtime.Text("x"))
	}
	parseRouter.Register("/a", parseComponent)
	parseRouter.Register("/b", parseComponent)

	parseRouter.Navigate("/a")
	parseWindow := js.Global().Get("window")

	// One hash nav → popstate + hashchange for the SAME url. The second must dedupe.
	parseRenders = 0
	parseWindow.Call("dispatchEvent", browserEventPop)
	parseAfterPopstate := parseRenders
	if parseAfterPopstate == 0 {
		parseT.Fatal("popstate should have rendered at least once")
	}
	parseWindow.Call("dispatchEvent", browserEventHash)
	if parseRenders != parseAfterPopstate {
		parseT.Fatalf("hashchange after same-URL popstate must be deduped, it added %d render(s)", parseRenders-parseAfterPopstate)
	}

	// A genuine navigation to a different URL must still render via the listener.
	parseRouter.Navigate("/b")
	parseRenders = 0
	parseWindow.Call("dispatchEvent", browserEventPop)
	if parseRenders == 0 {
		parseT.Fatal("a listener event at a new URL must render (dedup must not block real navigation)")
	}
}
