//go:build js && wasm

package render_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/testkit/render"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// TestUseMountLifecycle proves UseMount runs its fn exactly once on mount, does
// NOT re-run on a re-render, and runs its cleanup on unmount (G38).
func TestUseMountLifecycle(parseT *testing.T) {
	var parseMounts, parseCleanups int

	parseChild := func() ui.Node {
		ui.UseMount(func() func() {
			parseMounts++
			return func() { parseCleanups++ }
		})
		return html.P(html.Props{ID: "mount-child"}, html.Text("child"))
	}

	parseParent := func() ui.Node {
		parseShow := ui.UseState(true)
		parseTick := ui.UseState(0)
		parseHide := ui.UseEvent(func() { parseShow.Set(false) })
		parseRerender := ui.UseEvent(func() {
			parseTick.Update(func(parsePrev int) int { return parsePrev + 1 })
		})
		return html.Div(html.Props{ID: "mount-parent"},
			html.Button(html.Props{ID: "rerender", Type: "button", OnClick: parseRerender}, html.Text("rerender")),
			html.Button(html.Props{ID: "hide", Type: "button", OnClick: parseHide}, html.Text("hide")),
			html.If(parseShow.Get(), ui.CreateElement(parseChild)),
		)
	}

	parseFixture := render.New(parseT)
	parseFixture.Render(ui.CreateElement(parseParent))
	parseFixture.Flush()

	if parseMounts != 1 {
		parseT.Fatalf("expected exactly 1 mount, got %d", parseMounts)
	}

	// Re-render the child (parent state change) — UseMount must NOT run again.
	parseFixture.ClickByID("rerender")
	parseFixture.Flush()
	if parseMounts != 1 {
		parseT.Fatalf("UseMount must not re-run on re-render, mounts=%d", parseMounts)
	}
	if parseCleanups != 0 {
		parseT.Fatalf("no cleanup expected before unmount, got %d", parseCleanups)
	}

	// Unmount the child — cleanup must run exactly once.
	parseFixture.ClickByID("hide")
	parseFixture.Flush()
	if parseCleanups != 1 {
		parseT.Fatalf("expected exactly 1 cleanup on unmount, got %d", parseCleanups)
	}
}
