//go:build js && wasm

package render_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/testkit/render"
	"github.com/monstercameron/GoWebComponents/ui"
)

type mapRow struct {
	id   string
	name string
}

// TestMapKeyedComponentAllowsPerRowHooksInLoop is the definitive G1 proof: each
// row declares its OWN UseState + OnClick directly inside the loop render func
// (no hand-extracted row component). MapKeyedComponent gives each row its own
// fiber, so the per-row hook state is isolated and persists across renders by
// key — clicking one row's button never disturbs another's.
func TestMapKeyedComponentAllowsPerRowHooksInLoop(parseT *testing.T) {
	parseRows := []mapRow{{"a", "Alpha"}, {"b", "Beta"}, {"c", "Gamma"}}

	parseApp := func() ui.Node {
		parseChildren := html.MapKeyedComponent(parseRows,
			func(parseRow mapRow) any { return parseRow.id },
			func(parseRow mapRow) ui.Node {
				// Per-row hooks declared INSIDE the loop — legal here.
				parseCount := ui.UseState(0)
				parseInc := ui.UseEvent(func() {
					parseCount.Update(func(parsePrev int) int { return parsePrev + 1 })
				})
				return html.Div(html.Props{ID: "row-" + parseRow.id},
					html.Button(html.Props{ID: "btn-" + parseRow.id, Type: "button", OnClick: parseInc}, html.Text("inc")),
					html.P(html.Props{ID: "count-" + parseRow.id}, html.Text(fmt.Sprintf("%d", parseCount.Get()))),
				)
			})
		return html.Div(html.Props{ID: "list"}, parseChildren...)
	}

	parseFixture := render.New(parseT)
	parseFixture.Render(ui.CreateElement(parseApp))
	parseFixture.Flush()

	parseCountOf := func(parseID string) string {
		return parseFixture.ByID("count-" + parseID).Text()
	}

	if parseCountOf("a") != "0" || parseCountOf("b") != "0" || parseCountOf("c") != "0" {
		parseT.Fatalf("initial counts not zero: a=%s b=%s c=%s", parseCountOf("a"), parseCountOf("b"), parseCountOf("c"))
	}

	// Increment row A twice.
	parseFixture.ClickByID("btn-a")
	parseFixture.Flush()
	parseFixture.ClickByID("btn-a")
	parseFixture.Flush()

	// Increment row B once.
	parseFixture.ClickByID("btn-b")
	parseFixture.Flush()

	if parseCountOf("a") != "2" {
		parseT.Fatalf("row A count = %s, want 2 (isolated per-row hook)", parseCountOf("a"))
	}
	if parseCountOf("b") != "1" {
		parseT.Fatalf("row B count = %s, want 1", parseCountOf("b"))
	}
	if parseCountOf("c") != "0" {
		parseT.Fatalf("row C count = %s, want 0 (untouched)", parseCountOf("c"))
	}
}
