//go:build !js || !wasm

package render

import "testing"

func TestWithQueuedSchedulerOptionMutatesConfig(parseT *testing.T) {
	parseCfg := config{synchronous: true}
	WithQueuedScheduler()(&parseCfg)
	if parseCfg.synchronous {
		parseT.Fatalf("expected queued scheduler option to disable synchronous mode")
	}
	if parseContract := ParallelSafetyContract(); parseContract == "" {
		parseT.Fatalf("expected non-empty parallel safety contract message")
	}
}

func TestNativeFixtureAndQueryNodeStubMethods(parseT *testing.T) {
	parseFixture := &Fixture{}
	parseFixture.Render(nil)
	parseFixture.Rerender(nil)
	parseFixture.Flush()
	parseFixture.FlushTimers()
	parseFixture.Stabilize()
	parseFixture.Cleanup()
	parseFixture.DispatchByID("row-1", "click", Event{Value: "x", Checked: true, Key: "Enter", KeyCode: 13})
	parseFixture.ClickByID("row-1")
	parseFixture.InputByID("row-1", "input")
	parseFixture.ChangeByID("row-1", "change")
	parseFixture.SubmitByID("form-1")
	parseFixture.BuildOverlaySurfaces()
	parseFixture.BuildOverlayEscapeSurfaceID()
	parseFixture.BuildOverlayOutsideSurfaceID()
	parseFixture.BuildOverlayFocusSurfaceID()
	parseFixture.BuildOverlayScrollLockActive()
	parseFixture.BuildOverlayBodyOverflow()
	parseFixture.BuildOverlayPortalTargetID("overlay")
	parseFixture.HandleOverlayOutsideClick("overlay")
	parseFixture.BuildDiagnostics()
	parseFixture.BuildLogs()
	parseFixture.BuildRenderCounts()
	parseFixture.BuildWarningDiagnostics()
	parseFixture.BuildWarningLogs()
	parseFixture.ApplyDiagnosticCode("GWC-EXAMPLE")
	parseFixture.ApplyDiagnosticMessage("example")
	parseFixture.ApplyLogCode("GWC-EXAMPLE")
	parseFixture.ApplyLogMessage("example")
	parseFixture.ApplyRenderCountMax("counterHarnessApp", 1)
	parseFixture.ApplyRenderRerenderMax("counterHarnessApp", 1)
	parseFixture.ApplyWarningCountMax(0)
	parseFixture.ApplyWarningNone()

	if parseFixture.Container() != nil {
		parseT.Fatalf("expected nil container node in native stub")
	}
	if parseFixture.Target() != nil {
		parseT.Fatalf("expected nil target in native stub")
	}
	if parseFixture.ByRole("button", "save") != nil || len(parseFixture.AllByRole("button")) != 0 {
		parseT.Fatalf("expected no role matches in native stub")
	}
	if parseFixture.ByLabel("Search Catalog") != nil || parseFixture.ByDescription("Type to filter") != nil || parseFixture.ByLiveRegion("polite", "Saved") != nil {
		parseT.Fatalf("expected no accessibility-first query matches in native stub")
	}
	if parseFixture.ApplyByRole("button", "save") != nil || parseFixture.ApplyByLabel("Search Catalog") != nil || parseFixture.ApplyByDescription("Type to filter") != nil || parseFixture.ApplyByLiveRegion("polite", "Saved") != nil {
		parseT.Fatalf("expected no accessibility-first assertion matches in native stub")
	}
	if parseFixture.ByID("row-1") != nil || parseFixture.ByText("save") != nil || len(parseFixture.AllByTag("div")) != 0 {
		parseT.Fatalf("expected no node matches in native stub")
	}
	if parseFixture.Text() != "" {
		parseT.Fatalf("expected empty fixture text in native stub")
	}
	if len(parseFixture.BuildOverlaySurfaces()) != 0 || parseFixture.BuildOverlayEscapeSurfaceID() != "" || parseFixture.BuildOverlayOutsideSurfaceID() != "" || parseFixture.BuildOverlayFocusSurfaceID() != "" || parseFixture.BuildOverlayScrollLockActive() || parseFixture.BuildOverlayBodyOverflow() != "" || parseFixture.BuildOverlayPortalTargetID("overlay") != "" || parseFixture.HandleOverlayOutsideClick("overlay") {
		parseT.Fatalf("expected empty overlay helper defaults in native stub")
	}
	parseDiagnosticCode := parseFixture.ApplyDiagnosticCode("GWC-EXAMPLE")
	parseDiagnosticMessage := parseFixture.ApplyDiagnosticMessage("example")
	parseLogCode := parseFixture.ApplyLogCode("GWC-EXAMPLE")
	parseLogMessage := parseFixture.ApplyLogMessage("example")
	if len(parseFixture.BuildDiagnostics()) != 0 || len(parseFixture.BuildLogs()) != 0 || parseDiagnosticCode.Code != "" || parseDiagnosticCode.Message != "" || len(parseDiagnosticCode.ComponentStack) != 0 || len(parseDiagnosticCode.Fields) != 0 || parseDiagnosticMessage.Code != "" || parseDiagnosticMessage.Message != "" || len(parseDiagnosticMessage.ComponentStack) != 0 || len(parseDiagnosticMessage.Fields) != 0 || parseLogCode.Code != "" || parseLogCode.Message != "" || len(parseLogCode.Fields) != 0 || parseLogMessage.Code != "" || parseLogMessage.Message != "" || len(parseLogMessage.Fields) != 0 {
		parseT.Fatalf("expected empty diagnostics and logs helper defaults in native stub")
	}
	parseRenderCount := parseFixture.ApplyRenderCountMax("counterHarnessApp", 1)
	parseRerenderCount := parseFixture.ApplyRenderRerenderMax("counterHarnessApp", 1)
	if len(parseFixture.BuildRenderCounts()) != 0 || len(parseFixture.BuildWarningDiagnostics()) != 0 || len(parseFixture.BuildWarningLogs()) != 0 || parseRenderCount.Name != "" || parseRenderCount.Path != "" || parseRenderCount.RenderCount != 0 || parseRenderCount.RerenderCount != 0 || parseRenderCount.LastTrigger != "" || parseRenderCount.TotalRenderDurationNs != 0 || parseRenderCount.AverageRenderDurationNs != 0 || parseRerenderCount.Name != "" || parseRerenderCount.Path != "" || parseRerenderCount.RenderCount != 0 || parseRerenderCount.RerenderCount != 0 || parseRerenderCount.LastTrigger != "" || parseRerenderCount.TotalRenderDurationNs != 0 || parseRerenderCount.AverageRenderDurationNs != 0 {
		parseT.Fatalf("expected empty render-count and warning helper defaults in native stub")
	}

	parseNode := &QueryNode{}
	parseNode.Dispatch("click", Event{})
	parseNode.Click()
	parseNode.Input("value")
	parseNode.Change("value")
	parseNode.Submit()

	if parseNode.Exists() {
		parseT.Fatalf("expected native query node to report not found")
	}
	if parseNode.Tag() != "" || parseNode.Name() != "" || parseNode.Text() != "" || parseNode.Attr("id") != "" || parseNode.Property("x") != nil {
		parseT.Fatalf("expected empty native query node values")
	}
	if parseChildren := parseNode.Children(); parseChildren != nil {
		parseT.Fatalf("expected nil children in native query node")
	}
}
