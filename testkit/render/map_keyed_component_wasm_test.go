//go:build js && wasm

package render_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/testkit/render"
	"github.com/monstercameron/GoWebComponents/v4/ui"
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

// TestMapKeyedComponentStatePersistsThroughReorderAndRemoval is the hard G1
// correctness test: per-row hook state must follow the KEY when the list is
// reordered AND when it shrinks (the variable-length case that breaks naive
// in-loop hooks). Row state must not "slide" to a neighbor when an earlier row
// is removed.
func TestMapKeyedComponentStatePersistsThroughReorderAndRemoval(parseT *testing.T) {
	parseRows := []mapRow{{"a", "Alpha"}, {"b", "Beta"}, {"c", "Gamma"}}

	parseApp := func() ui.Node {
		parseOrder := ui.UseState(parseRows)
		parseReorder := ui.UseEvent(func() {
			parseOrder.Set([]mapRow{{"c", "Gamma"}, {"a", "Alpha"}, {"b", "Beta"}})
		})
		parseRemoveA := ui.UseEvent(func() {
			parseCurrent := parseOrder.Get()
			parseNext := make([]mapRow, 0, len(parseCurrent))
			for _, parseRow := range parseCurrent {
				if parseRow.id != "a" {
					parseNext = append(parseNext, parseRow)
				}
			}
			parseOrder.Set(parseNext)
		})

		parseChildren := html.MapKeyedComponent(parseOrder.Get(),
			func(parseRow mapRow) any { return parseRow.id },
			func(parseRow mapRow) ui.Node {
				parseCount := ui.UseState(0)
				parseInc := ui.UseEvent(func() {
					parseCount.Update(func(parsePrev int) int { return parsePrev + 1 })
				})
				return html.Div(html.Props{ID: "r-" + parseRow.id},
					html.Button(html.Props{ID: "i-" + parseRow.id, Type: "button", OnClick: parseInc}, html.Text("inc")),
					html.P(html.Props{ID: "c-" + parseRow.id}, html.Text(fmt.Sprintf("%d", parseCount.Get()))),
				)
			})
		return html.Div(html.Props{ID: "list2"},
			html.Button(html.Props{ID: "reorder", Type: "button", OnClick: parseReorder}, html.Text("reorder")),
			html.Button(html.Props{ID: "remove-a", Type: "button", OnClick: parseRemoveA}, html.Text("remove a")),
			html.Div(html.Props{ID: "rows"}, parseChildren...),
		)
	}

	parseFixture := render.New(parseT)
	parseFixture.Render(ui.CreateElement(parseApp))
	parseFixture.Flush()

	parseCountOf := func(parseID string) string { return parseFixture.ByID("c-" + parseID).Text() }

	// Give each row a distinct count: a=1, b=2, c=3.
	parseFixture.ClickByID("i-a")
	parseFixture.ClickByID("i-b")
	parseFixture.ClickByID("i-b")
	parseFixture.ClickByID("i-c")
	parseFixture.ClickByID("i-c")
	parseFixture.ClickByID("i-c")
	parseFixture.Flush()
	if parseCountOf("a") != "1" || parseCountOf("b") != "2" || parseCountOf("c") != "3" {
		parseT.Fatalf("setup counts wrong: a=%s b=%s c=%s", parseCountOf("a"), parseCountOf("b"), parseCountOf("c"))
	}

	// Reorder to [c, a, b] — counts must follow keys, not positions.
	parseFixture.ClickByID("reorder")
	parseFixture.Flush()
	if parseCountOf("a") != "1" || parseCountOf("b") != "2" || parseCountOf("c") != "3" {
		parseT.Fatalf("after reorder counts slid: a=%s b=%s c=%s", parseCountOf("a"), parseCountOf("b"), parseCountOf("c"))
	}

	// Remove row A (list shrinks) — remaining rows keep their own counts.
	parseFixture.ClickByID("remove-a")
	parseFixture.Flush()
	if parseCountOf("b") != "2" || parseCountOf("c") != "3" {
		parseT.Fatalf("after removal counts slid: b=%s c=%s (want 2,3)", parseCountOf("b"), parseCountOf("c"))
	}
}
