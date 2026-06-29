//go:build js && wasm

package render_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/state"
	"github.com/monstercameron/GoWebComponents/v4/testkit/render"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// TestGlobalAtomSetDedupSkipsNoOpRerender proves GlobalAtom.Set with an unchanged
// value does NOT re-render subscribers, while a changed value does (matching
// UseState's dedup behavior).
func TestGlobalAtomSetDedupSkipsNoOpRerender(parseT *testing.T) {
	const parseID = "test:ga:dedup"
	parseRenders := 0
	parseApp := func() ui.Node {
		parseRenders++
		parseValue := state.UseAtom(parseID, "init")
		return html.P(html.Props{ID: "dedup-label"}, html.Text(parseValue.Get()))
	}

	parseExternal := state.NewGlobalAtom(parseID, "init")
	parseFixture := render.New(parseT)
	parseFixture.Render(ui.CreateElement(parseApp))
	parseFixture.Flush()
	parseAfterMount := parseRenders

	// No-op set: same value → no re-render.
	parseExternal.Set("init")
	parseFixture.Flush()
	if parseRenders != parseAfterMount {
		parseT.Fatalf("no-op Set re-rendered: %d -> %d", parseAfterMount, parseRenders)
	}

	// Changed set: must re-render and update.
	parseExternal.Set("changed")
	parseFixture.Flush()
	if parseRenders <= parseAfterMount {
		parseT.Fatalf("changed Set did not re-render (renders=%d)", parseRenders)
	}
	if parseGot := parseFixture.ByID("dedup-label").Text(); parseGot != "changed" {
		parseT.Fatalf("label = %q, want changed", parseGot)
	}
}
