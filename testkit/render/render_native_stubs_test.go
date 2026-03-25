//go:build !js || !wasm
// +build !js !wasm

package render

import "testing"

func TestWithQueuedSchedulerOptionMutatesConfig(t *testing.T) {
	cfg := config{synchronous: true}
	WithQueuedScheduler()(&cfg)
	if cfg.synchronous {
		t.Fatalf("expected queued scheduler option to disable synchronous mode")
	}
	if contract := ParallelSafetyContract(); contract == "" {
		t.Fatalf("expected non-empty parallel safety contract message")
	}
}

func TestNativeFixtureAndQueryNodeStubMethods(t *testing.T) {
	fixture := &Fixture{}
	fixture.Render(nil)
	fixture.Rerender(nil)
	fixture.Flush()
	fixture.FlushTimers()
	fixture.Stabilize()
	fixture.Cleanup()
	fixture.DispatchByID("row-1", "click", Event{Value: "x", Checked: true, Key: "Enter", KeyCode: 13})
	fixture.ClickByID("row-1")
	fixture.InputByID("row-1", "input")
	fixture.ChangeByID("row-1", "change")
	fixture.SubmitByID("form-1")
	fixture.BuildOverlaySurfaces()
	fixture.BuildOverlayEscapeSurfaceID()
	fixture.BuildOverlayOutsideSurfaceID()
	fixture.BuildOverlayFocusSurfaceID()
	fixture.BuildOverlayScrollLockActive()
	fixture.BuildOverlayBodyOverflow()
	fixture.BuildOverlayPortalTargetID("overlay")
	fixture.HandleOverlayOutsideClick("overlay")
	fixture.BuildDiagnostics()
	fixture.BuildLogs()
	fixture.ApplyDiagnosticCode("GWC-EXAMPLE")
	fixture.ApplyDiagnosticMessage("example")
	fixture.ApplyLogCode("GWC-EXAMPLE")
	fixture.ApplyLogMessage("example")

	if fixture.Container() != nil {
		t.Fatalf("expected nil container node in native stub")
	}
	if fixture.Target() != nil {
		t.Fatalf("expected nil target in native stub")
	}
	if fixture.ByRole("button", "save") != nil || len(fixture.AllByRole("button")) != 0 {
		t.Fatalf("expected no role matches in native stub")
	}
	if fixture.ByLabel("Search Catalog") != nil || fixture.ByDescription("Type to filter") != nil || fixture.ByLiveRegion("polite", "Saved") != nil {
		t.Fatalf("expected no accessibility-first query matches in native stub")
	}
	if fixture.ApplyByRole("button", "save") != nil || fixture.ApplyByLabel("Search Catalog") != nil || fixture.ApplyByDescription("Type to filter") != nil || fixture.ApplyByLiveRegion("polite", "Saved") != nil {
		t.Fatalf("expected no accessibility-first assertion matches in native stub")
	}
	if fixture.ByID("row-1") != nil || fixture.ByText("save") != nil || len(fixture.AllByTag("div")) != 0 {
		t.Fatalf("expected no node matches in native stub")
	}
	if fixture.Text() != "" {
		t.Fatalf("expected empty fixture text in native stub")
	}
	if len(fixture.BuildOverlaySurfaces()) != 0 || fixture.BuildOverlayEscapeSurfaceID() != "" || fixture.BuildOverlayOutsideSurfaceID() != "" || fixture.BuildOverlayFocusSurfaceID() != "" || fixture.BuildOverlayScrollLockActive() || fixture.BuildOverlayBodyOverflow() != "" || fixture.BuildOverlayPortalTargetID("overlay") != "" || fixture.HandleOverlayOutsideClick("overlay") {
		t.Fatalf("expected empty overlay helper defaults in native stub")
	}
	diagnosticCode := fixture.ApplyDiagnosticCode("GWC-EXAMPLE")
	diagnosticMessage := fixture.ApplyDiagnosticMessage("example")
	logCode := fixture.ApplyLogCode("GWC-EXAMPLE")
	logMessage := fixture.ApplyLogMessage("example")
	if len(fixture.BuildDiagnostics()) != 0 || len(fixture.BuildLogs()) != 0 || diagnosticCode.Code != "" || diagnosticCode.Message != "" || len(diagnosticCode.ComponentStack) != 0 || len(diagnosticCode.Fields) != 0 || diagnosticMessage.Code != "" || diagnosticMessage.Message != "" || len(diagnosticMessage.ComponentStack) != 0 || len(diagnosticMessage.Fields) != 0 || logCode.Code != "" || logCode.Message != "" || len(logCode.Fields) != 0 || logMessage.Code != "" || logMessage.Message != "" || len(logMessage.Fields) != 0 {
		t.Fatalf("expected empty diagnostics and logs helper defaults in native stub")
	}

	node := &QueryNode{}
	node.Dispatch("click", Event{})
	node.Click()
	node.Input("value")
	node.Change("value")
	node.Submit()

	if node.Exists() {
		t.Fatalf("expected native query node to report not found")
	}
	if node.Tag() != "" || node.Name() != "" || node.Text() != "" || node.Attr("id") != "" || node.Property("x") != nil {
		t.Fatalf("expected empty native query node values")
	}
	if children := node.Children(); children != nil {
		t.Fatalf("expected nil children in native query node")
	}
}
