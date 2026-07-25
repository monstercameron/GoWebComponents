//go:build !js || !wasm

package routertest

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/testkit/render"
)

func TestNativeRouterFixtureStubMethods(parseT *testing.T) {
	parseFixture := &Fixture{}

	parseFixture.Register("/users/:id", nil)
	parseFixture.SetPath("/users/1")
	parseFixture.Render()
	parseFixture.Navigate("/users/2")
	parseFixture.Replace("/users/2?tab=history")
	parseFixture.DispatchByID("route-input", "oninput", render.Event{Value: "Cam"})
	parseFixture.ClickByID("route-button")
	parseFixture.InputByID("route-input", "Cam")
	parseFixture.ChangeByID("route-input", "Cam")
	parseFixture.SubmitByID("route-form")
	parseFixture.Cleanup()

	parseInspection := parseFixture.Inspect()
	if parseInspection.Path != "" || parseInspection.Query != nil || parseInspection.Params != nil {
		parseT.Fatalf("expected zero-value inspection in native stub, got %+v", parseInspection)
	}
	if parseFixture.Path() != "" || parseFixture.Query() != nil || parseFixture.Params() != nil || parseFixture.Router() != nil {
		parseT.Fatalf("expected zero-value navigation state in native stub")
	}
	if parseFixture.ByID("route") != nil || parseFixture.ByText("route") != nil {
		parseT.Fatalf("expected no route nodes in native stub")
	}
	if parseFixture.ByRole("button", "Save") != nil || len(parseFixture.AllByRole("button")) != 0 || parseFixture.ByLabel("Name") != nil || parseFixture.ByDescription("Type your name") != nil || parseFixture.ByLiveRegion("polite", "Saved") != nil {
		parseT.Fatalf("expected no accessibility route nodes in native stub")
	}
	if parseFixture.ApplyByRole("button", "Save") != nil || parseFixture.ApplyByLabel("Name") != nil || parseFixture.ApplyByDescription("Type your name") != nil || parseFixture.ApplyByLiveRegion("polite", "Saved") != nil {
		parseT.Fatalf("expected no accessibility assertion matches in native stub")
	}
	if parseFixture.Text() != "" {
		parseT.Fatalf("expected empty fixture text in native stub")
	}
}
