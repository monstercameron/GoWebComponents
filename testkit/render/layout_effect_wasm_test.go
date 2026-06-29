//go:build js && wasm

package render_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/testkit/render"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// TestLayoutEffectRunsBeforePassiveEffect proves UseLayoutEffect runs before
// UseEffect even when declared AFTER it in source — ordering is by effect kind,
// not call order (G36).
func TestLayoutEffectRunsBeforePassiveEffect(parseT *testing.T) {
	var parseOrder []string

	parseApp := func() ui.Node {
		// Passive declared first, layout second — layout must still run first.
		ui.UseEffect(func() func() {
			parseOrder = append(parseOrder, "passive")
			return nil
		}, "once")
		ui.UseLayoutEffect(func() func() {
			parseOrder = append(parseOrder, "layout")
			return nil
		}, "once")
		return html.Div(html.Props{ID: "le-box"}, html.Text("ready"))
	}

	parseFixture := render.New(parseT)
	parseFixture.Render(ui.CreateElement(parseApp))
	parseFixture.Flush()

	if len(parseOrder) != 2 || parseOrder[0] != "layout" || parseOrder[1] != "passive" {
		parseT.Fatalf("expected [layout passive], got %v", parseOrder)
	}
}

// TestMultipleLayoutEffectsPreserveDeclarationOrder proves layout effects run in
// their declared order among themselves.
func TestMultipleLayoutEffectsPreserveDeclarationOrder(parseT *testing.T) {
	var parseOrder []string

	parseApp := func() ui.Node {
		ui.UseLayoutEffect(func() func() { parseOrder = append(parseOrder, "L1"); return nil }, "a")
		ui.UseLayoutEffect(func() func() { parseOrder = append(parseOrder, "L2"); return nil }, "b")
		return html.Div(html.Props{ID: "le-multi"}, html.Text("x"))
	}

	parseFixture := render.New(parseT)
	parseFixture.Render(ui.CreateElement(parseApp))
	parseFixture.Flush()

	if len(parseOrder) != 2 || parseOrder[0] != "L1" || parseOrder[1] != "L2" {
		parseT.Fatalf("expected [L1 L2], got %v", parseOrder)
	}
}

// TestLayoutEffectCleanupRunsOnDepChange proves a layout effect's cleanup runs
// before it re-runs when its dependency changes.
func TestLayoutEffectCleanupRunsOnDepChange(parseT *testing.T) {
	var parseOrder []string

	parseApp := func() ui.Node {
		parseTick := ui.UseState(0)
		ui.UseLayoutEffect(func() func() {
			parseOrder = append(parseOrder, "run")
			return func() { parseOrder = append(parseOrder, "cleanup") }
		}, parseTick.Get())
		parseBump := ui.UseEvent(func() {
			parseTick.Update(func(parsePrev int) int { return parsePrev + 1 })
		})
		return html.Div(html.Props{ID: "le-clean"},
			html.Button(html.Props{ID: "bump", Type: "button", OnClick: parseBump}, html.Text("bump")),
		)
	}

	parseFixture := render.New(parseT)
	parseFixture.Render(ui.CreateElement(parseApp))
	parseFixture.Flush()
	parseFixture.ClickByID("bump")
	parseFixture.Flush()

	// First run, then on dep change: cleanup of the previous, then re-run.
	if len(parseOrder) < 3 || parseOrder[0] != "run" || parseOrder[1] != "cleanup" || parseOrder[2] != "run" {
		parseT.Fatalf("expected [run cleanup run ...], got %v", parseOrder)
	}
}
