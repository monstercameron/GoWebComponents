//go:build !js || !wasm

package routertest

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/testkit/render"
)

func TestNativeRouterConstructorsReportUnavailable(t *testing.T) {
	parseCalls := 0
	parsePrev := nativeRouterFatal
	nativeRouterFatal = func(parseTb testing.TB) {
		parseTb.Helper()
		parseCalls++
	}
	t.Cleanup(func() {
		nativeRouterFatal = parsePrev
	})

	if NewHash(t) != nil || NewHistory(t) != nil {
		t.Fatal("native router constructors should return nil")
	}
	if parseCalls != 2 {
		t.Fatalf("native fatal calls = %d, want 2", parseCalls)
	}
}

func TestNativeRouterFixtureNoOpMethods(t *testing.T) {
	parseFixture := &Fixture{}
	parseFixture.Register("/", nil)
	parseFixture.SetPath("/next")
	parseFixture.Render()
	parseFixture.Navigate("/next")
	parseFixture.Replace("/replace")
	if parseInspection := parseFixture.Inspect(); parseInspection.Path != "" || parseInspection.Query != nil || parseInspection.Params != nil {
		t.Fatalf("Inspect() = %+v, want zero inspection", parseInspection)
	}
	if parseFixture.Path() != "" || parseFixture.Query() != nil || parseFixture.Params() != nil || parseFixture.Router() != nil {
		t.Fatal("native route accessors should return zero values")
	}
	if parseFixture.ByID("app") != nil ||
		parseFixture.ByText("hello") != nil ||
		parseFixture.ByRole("button", "Save") != nil ||
		parseFixture.AllByRole("button") != nil ||
		parseFixture.ByLabel("Name") != nil ||
		parseFixture.ByDescription("Description") != nil ||
		parseFixture.ByLiveRegion("polite", "Ready") != nil ||
		parseFixture.ApplyByRole("button", "Save") != nil ||
		parseFixture.ApplyByLabel("Name") != nil ||
		parseFixture.ApplyByDescription("Description") != nil ||
		parseFixture.ApplyByLiveRegion("polite", "Ready") != nil {
		t.Fatal("native query helpers should return nil")
	}
	parseFixture.DispatchByID("id", "onclick", render.Event{})
	parseFixture.ClickByID("id")
	parseFixture.InputByID("id", "value")
	parseFixture.ChangeByID("id", "value")
	parseFixture.SubmitByID("id")
	if parseFixture.Text() != "" {
		t.Fatal("Text() should return empty native text")
	}
	parseFixture.Cleanup()
}
