//go:build js && wasm

package render_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/state"
	"github.com/monstercameron/GoWebComponents/v6/testkit/render"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

const globalAtomIntegrationID = "test:ga:integration:status"

func globalAtomReaderApp() ui.Node {
	parseStatus := state.UseAtom(globalAtomIntegrationID, "initial")
	return html.Div(html.Props{ID: "ga-box"}, html.Text(parseStatus.Get()))
}

// TestGlobalAtomDrivesUseAtomRerender is the G39 integration proof: a write made
// from OUTSIDE the component tree (through a state.GlobalAtom handle, as a global
// event handler or background goroutine would) re-renders a component subscribed
// to the same id via UseAtom — no render-phase capture variable required.
func TestGlobalAtomDrivesUseAtomRerender(parseT *testing.T) {
	parseExternal := state.NewGlobalAtom(globalAtomIntegrationID, "initial")

	parseFixture := render.New(parseT)
	parseFixture.Render(ui.CreateElement(globalAtomReaderApp))

	if parseGot := parseFixture.ByID("ga-box").Text(); parseGot != "initial" {
		parseT.Fatalf("expected initial seeded value, got %q", parseGot)
	}

	// External write from outside any render/event.
	parseExternal.Set("changed-from-outside")
	parseFixture.Flush()

	if parseGot := parseFixture.ByID("ga-box").Text(); parseGot != "changed-from-outside" {
		parseT.Fatalf("external GlobalAtom write did not re-render UseAtom subscriber, got %q", parseGot)
	}
}

// TestGlobalAtomPreRenderWriteSurvives proves a write made BEFORE the component
// first renders is observed by the first UseAtom read (no silent drop).
func TestGlobalAtomPreRenderWriteSurvives(parseT *testing.T) {
	const parseID = "test:ga:integration:prerender"

	parseFixture := render.New(parseT)
	parseExternal := state.NewGlobalAtom(parseID, "seed")
	// Write before the component using parseID has rendered for the first time.
	parseExternal.Set("written-pre-render")

	parseFixture.Render(ui.CreateElement(func() ui.Node {
		parseValue := state.UseAtom(parseID, "seed")
		return html.Div(html.Props{ID: "pre-box"}, html.Text(parseValue.Get()))
	}))

	if parseGot := parseFixture.ByID("pre-box").Text(); parseGot != "written-pre-render" {
		parseT.Fatalf("pre-render write was dropped, got %q", parseGot)
	}
}
