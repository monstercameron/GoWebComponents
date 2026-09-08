//go:build js && wasm

package render

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func staticHarnessApp() ui.Node {
	return html.Div(html.Props{ID: "fixture-root"},
		html.H1(html.Props{ID: "main-heading"}, html.Text("Hello Harness")),
		html.Button(html.Props{ID: "primary-button", Type: "button"}, html.Text("Save")),
		html.Span(html.Props{ID: "search-label"}, html.Text("Search Catalog")),
		html.P(html.Props{ID: "search-description"}, html.Text("Type to filter")),
		html.Input(html.Props{ID: "search-input", Type: "text", Raw: map[string]interface{}{"aria-labelledby": "search-label", "aria-describedby": "search-description"}}),
		html.Div(html.Props{ID: "save-live-region", Role: "status", Raw: map[string]interface{}{"aria-live": "polite"}}, html.Text("Saved profile")),
	)
}

func counterHarnessApp() ui.Node {
	parseCount := ui.UseState(0)
	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrevious int) int {
			return parsePrevious + 1
		})
	})

	return html.Div(html.Props{ID: "counter-root"},
		html.P(html.Props{ID: "count-label"}, html.Text(fmt.Sprintf("Count: %d", parseCount.Get()))),
		html.Button(html.Props{ID: "increment", Type: "button", OnClick: parseIncrement}, html.Text("Increment")),
	)
}

func inputHarnessApp() ui.Node {
	parseValue := ui.UseState("")
	handleInput := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseValue.Set(parseEvent.GetValue())
	})

	return html.Div(html.Props{ID: "input-root"},
		html.Input(html.Props{ID: "name-input", Type: "text", OnInput: handleInput}),
		html.P(html.Props{ID: "name-value"}, html.Text("Value: "+parseValue.Get())),
	)
}

func transitionHarnessApp() ui.Node {
	parseCount := ui.UseState(0)
	handleClick := ui.UseEvent(func() {
		ui.StartTransition(func() {
			parseCount.Update(func(parsePrevious int) int {
				return parsePrevious + 1
			})
		})
	})

	return html.Div(html.Props{ID: "transition-root"},
		html.P(html.Props{ID: "transition-count"}, html.Text(fmt.Sprintf("Transition Count: %d", parseCount.Get()))),
		html.Button(html.Props{ID: "transition-increment", Type: "button", OnClick: handleClick}, html.Text("Increment Later")),
	)
}

// overlayHarnessApp renders a nested modal overlay fixture through a shared portal root.
func overlayHarnessApp() ui.Node {
	isParentOpen := ui.UseState(false)
	isChildOpen := ui.UseState(false)

	handleOpenParent := ui.UseEvent(func() {
		isParentOpen.Set(true)
	})
	handleOpenChild := ui.UseEvent(func() {
		if isParentOpen.Get() {
			isChildOpen.Set(true)
		}
	})

	return html.Div(html.Props{ID: "overlay-shell"},
		html.Button(html.Props{ID: "open-parent", Type: "button", OnClick: handleOpenParent}, html.Text("Open parent")),
		html.Button(html.Props{ID: "open-child", Type: "button", OnClick: handleOpenChild}, html.Text("Open child")),
		html.Div(html.Props{ID: "portal-root"}),
		ui.Overlay(ui.OverlayProps{
			Open:                isParentOpen.Get(),
			Target:              ui.PortalTarget{Selector: "#portal-root"},
			SurfaceID:           "parent-overlay",
			Modal:               true,
			Backdrop:            true,
			CloseOnEscape:       true,
			CloseOnOutsideClick: true,
			LockScroll:          true,
			OnDismiss: func() {
				isChildOpen.Set(false)
				isParentOpen.Set(false)
			},
			Child: html.Div(html.Props{ID: "parent-content"}, html.Text("Parent overlay")),
		}),
		ui.Overlay(ui.OverlayProps{
			Open:                isParentOpen.Get() && isChildOpen.Get(),
			Target:              ui.PortalTarget{Selector: "#portal-root"},
			SurfaceID:           "child-overlay",
			Modal:               true,
			Backdrop:            true,
			CloseOnEscape:       true,
			CloseOnOutsideClick: true,
			LockScroll:          true,
			OnDismiss: func() {
				isChildOpen.Set(false)
			},
			Child: html.Div(html.Props{ID: "child-content"}, html.Text("Child overlay")),
		}),
	)
}

func TestFixtureRenderAndQuery(parseT *testing.T) {
	parseFixture := New(parseT)
	parseFixture.Render(ui.CreateElement(staticHarnessApp))

	parseHeading := parseFixture.ByID("main-heading")
	if parseHeading == nil || parseHeading.Text() != "Hello Harness" {
		parseT.Fatalf("expected heading text, got %#v", parseHeading)
	}
	if parseMatch := parseFixture.ByText("Save"); parseMatch == nil || parseMatch.Tag() != "button" {
		parseT.Fatalf("expected to find button by text, got %#v", parseMatch)
	}
	parseButtons := parseFixture.AllByTag("button")
	if len(parseButtons) != 1 {
		parseT.Fatalf("expected one button, got %d", len(parseButtons))
	}
	if parseButtons[0].Attr("id") != "primary-button" {
		parseT.Fatalf("expected queried button id to be preserved, got %q", parseButtons[0].Attr("id"))
	}
	if parseButton := parseFixture.ByRole("button", "Save"); parseButton == nil || parseButton.Attr("id") != "primary-button" {
		parseT.Fatalf("expected role query to find the button, got %#v", parseButton)
	}
	if parseField := parseFixture.ByRole("textbox", "Search Catalog"); parseField == nil || parseField.Attr("id") != "search-input" {
		parseT.Fatalf("expected role query to resolve aria-labelledby accessible name, got %#v", parseField)
	}
	if parseName := parseFixture.ByID("search-input").Name(); parseName != "Search Catalog" {
		parseT.Fatalf("expected stable accessible name, got %q", parseName)
	}
	if parseTextboxes := parseFixture.AllByRole("textbox"); len(parseTextboxes) != 1 {
		parseT.Fatalf("expected one textbox role match, got %d", len(parseTextboxes))
	}
}

// TestFixtureAccessibilityFirstQueriesAndAssertions validates label, description, and live-region helpers.
func TestFixtureAccessibilityFirstQueriesAndAssertions(parseT *testing.T) {
	parseFixture := New(parseT)
	parseFixture.Render(ui.CreateElement(staticHarnessApp))

	if parseField := parseFixture.ByLabel("Search Catalog"); parseField == nil || parseField.Attr("id") != "search-input" {
		parseT.Fatalf("expected label query to find search input, got %#v", parseField)
	}
	if parseField2 := parseFixture.ByDescription("Type to filter"); parseField2 == nil || parseField2.Attr("id") != "search-input" {
		parseT.Fatalf("expected description query to find search input, got %#v", parseField2)
	}
	if parseRegion := parseFixture.ByLiveRegion("polite", "Saved profile"); parseRegion == nil || parseRegion.Attr("id") != "save-live-region" {
		parseT.Fatalf("expected live-region query to find save status, got %#v", parseRegion)
	}

	if parseField3 := parseFixture.ApplyByLabel("Search Catalog"); parseField3.Attr("id") != "search-input" {
		parseT.Fatalf("expected ApplyByLabel to return search input, got %#v", parseField3)
	}
	if parseField4 := parseFixture.ApplyByDescription("Type to filter"); parseField4.Attr("id") != "search-input" {
		parseT.Fatalf("expected ApplyByDescription to return search input, got %#v", parseField4)
	}
	if parseRegion2 := parseFixture.ApplyByLiveRegion("polite", "Saved profile"); parseRegion2.Attr("id") != "save-live-region" {
		parseT.Fatalf("expected ApplyByLiveRegion to return save live region, got %#v", parseRegion2)
	}
	if parseButton := parseFixture.ApplyByRole("button", "Save"); parseButton.Attr("id") != "primary-button" {
		parseT.Fatalf("expected ApplyByRole to return primary button, got %#v", parseButton)
	}
}

func TestFixtureExposesRenderedPropertiesForStateUpdates(parseT *testing.T) {
	parseFixture := New(parseT)
	parseFixture.Render(ui.CreateElement(counterHarnessApp))

	if parseGot := parseFixture.ByID("count-label").Text(); parseGot != "Count: 0" {
		parseT.Fatalf("expected initial count label, got %q", parseGot)
	}

	parseHandler, parseOk := parseFixture.ByID("increment").Property("onclick").(func())
	if !parseOk {
		parseT.Fatalf("expected onclick property with func() signature, got %T", parseFixture.ByID("increment").Property("onclick"))
	}
	parseHandler()

	if parseGot2 := parseFixture.ByID("count-label").Text(); parseGot2 != "Count: 1" {
		parseT.Fatalf("expected updated count label after invoking onclick, got %q", parseGot2)
	}
}

func TestFixtureRerenderReplacesTree(parseT *testing.T) {
	parseFixture := New(parseT)
	parseFixture.Render(ui.CreateElement(staticHarnessApp))
	parseFixture.Rerender(html.Div(html.Props{ID: "replacement"}, html.Text("Replaced")))

	if parseFixture.ByID("main-heading") != nil {
		parseT.Fatal("expected original tree to be replaced")
	}
	parseReplacement := parseFixture.ByID("replacement")
	if parseReplacement == nil || parseReplacement.Text() != "Replaced" {
		parseT.Fatalf("expected rerendered replacement tree, got %#v", parseReplacement)
	}
}

func TestFixtureInputHelperDispatchesAndSettles(parseT *testing.T) {
	parseFixture := New(parseT)
	parseFixture.Render(ui.CreateElement(inputHarnessApp))

	parseFixture.InputByID("name-input", "Cam")

	if parseGot := parseFixture.ByID("name-value").Text(); parseGot != "Value: Cam" {
		parseT.Fatalf("expected input helper to update rendered value, got %q", parseGot)
	}
}

func TestFixtureQueuedSchedulerClickHelperStabilizesTransitionWork(parseT *testing.T) {
	parseFixture := New(parseT, WithQueuedScheduler())
	parseFixture.Render(ui.CreateElement(transitionHarnessApp))

	parseFixture.ClickByID("transition-increment")

	if parseGot := parseFixture.ByID("transition-count").Text(); parseGot != "Transition Count: 1" {
		parseT.Fatalf("expected queued click helper to stabilize transition work, got %q", parseGot)
	}
}

// TestFixtureOverlayHelpersTrackStackAndOutsideDismiss validates overlay helper coverage for portal-backed stacks.
func TestFixtureOverlayHelpersTrackStackAndOutsideDismiss(parseT *testing.T) {
	parseFixture := New(parseT)
	parseFixture.Render(ui.CreateElement(overlayHarnessApp))

	if parseSurfaces := parseFixture.BuildOverlaySurfaces(); len(parseSurfaces) != 0 {
		parseT.Fatalf("expected no overlays before opening, got %+v", parseSurfaces)
	}

	parseFixture.ClickByID("open-parent")
	parseFixture.ClickByID("open-child")

	parseSurfaces2 := parseFixture.BuildOverlaySurfaces()
	if len(parseSurfaces2) != 2 {
		parseT.Fatalf("expected two overlay surfaces, got %+v", parseSurfaces2)
	}
	if parseFixture.BuildOverlayEscapeSurfaceID() != "child-overlay" {
		parseT.Fatalf("expected child overlay to own escape handling, got %q", parseFixture.BuildOverlayEscapeSurfaceID())
	}
	if parseFixture.BuildOverlayOutsideSurfaceID() != "child-overlay" {
		parseT.Fatalf("expected child overlay to own outside-click handling, got %q", parseFixture.BuildOverlayOutsideSurfaceID())
	}
	if parseFixture.BuildOverlayFocusSurfaceID() != "child-overlay" {
		parseT.Fatalf("expected child overlay to own trap focus, got %q", parseFixture.BuildOverlayFocusSurfaceID())
	}
	if parseFixture.BuildOverlayPortalTargetID("parent-overlay") != "portal-root" || parseFixture.BuildOverlayPortalTargetID("child-overlay") != "portal-root" {
		parseT.Fatalf("expected both overlays to render into portal-root, got parent=%q child=%q", parseFixture.BuildOverlayPortalTargetID("parent-overlay"), parseFixture.BuildOverlayPortalTargetID("child-overlay"))
	}
	if !parseFixture.BuildOverlayScrollLockActive() {
		parseT.Fatalf("expected scroll lock helper to report active while overlays are open; overflow=%q surfaces=%+v", parseFixture.BuildOverlayBodyOverflow(), parseSurfaces2)
	}

	if !parseFixture.HandleOverlayOutsideClick("child-overlay") {
		parseT.Fatal("expected child overlay outside-click helper to dispatch through backdrop")
	}
	if parseFixture.ByID("child-overlay") != nil {
		parseT.Fatalf("expected child overlay to close after outside click")
	}
	if parseFixture.ByID("parent-overlay") == nil {
		parseT.Fatalf("expected parent overlay to remain open after child outside click")
	}
	if parseFixture.BuildOverlayEscapeSurfaceID() != "parent-overlay" || parseFixture.BuildOverlayFocusSurfaceID() != "parent-overlay" {
		parseT.Fatalf("expected parent overlay to inherit top ownership after child dismissal, got escape=%q focus=%q", parseFixture.BuildOverlayEscapeSurfaceID(), parseFixture.BuildOverlayFocusSurfaceID())
	}

	if !parseFixture.HandleOverlayOutsideClick("parent-overlay") {
		parseT.Fatal("expected parent overlay outside-click helper to dispatch through backdrop")
	}
	if parseFixture.ByID("parent-overlay") != nil {
		parseT.Fatalf("expected parent overlay to close after outside click")
	}
	if parseFixture.BuildOverlayScrollLockActive() {
		parseT.Fatalf("expected scroll lock helper to report inactive after all overlays close; overflow=%q surfaces=%+v", parseFixture.BuildOverlayBodyOverflow(), parseFixture.BuildOverlaySurfaces())
	}
}

func TestFixtureDiagnosticsAndLogsHelpersExposeStructuredAssertions(parseT *testing.T) {
	parseFixture := New(parseT)
	parseFixture.Render(ui.CreateElement(staticHarnessApp))

	runtime.ReportDiagnostic("runtime", runtime.DiagnosticWarning, "hydration fell back to client rendering for <div>: DOM node <span> did not match expected <div>")
	runtime.ReportLogWithFields("router", runtime.LogError, runtime.DiagnosticCorrectness, "route loader failed", "", nil)

	parseDiagnostics := parseFixture.BuildDiagnostics()
	if len(parseDiagnostics) == 0 {
		parseT.Fatalf("expected diagnostics helper to expose runtime diagnostics")
	}
	parseLogs := parseFixture.BuildLogs()
	if len(parseLogs) == 0 {
		parseT.Fatalf("expected logs helper to expose runtime logs")
	}

	parseDiagnostic := parseFixture.ApplyDiagnosticCode("GWC-HYDRATION-FALLBACK")
	if parseDiagnostic.Source != "runtime" || !parseDiagnostic.Recoverable {
		parseT.Fatalf("expected hydration fallback diagnostic metadata, got %+v", parseDiagnostic)
	}
	if parseMatch := parseFixture.ApplyDiagnosticMessage("hydration fell back"); parseMatch.Code != "GWC-HYDRATION-FALLBACK" {
		parseT.Fatalf("expected diagnostic message assertion to resolve hydration fallback code, got %+v", parseMatch)
	}

	parseLog := parseFixture.ApplyLogCode("GWC-ROUTER-LOADER-FAILED")
	if parseLog.Domain != "router" || parseLog.Level != string(runtime.LogError) {
		parseT.Fatalf("expected router loader log metadata, got %+v", parseLog)
	}
	if parseMatch2 := parseFixture.ApplyLogMessage("route loader failed"); parseMatch2.Code != "GWC-ROUTER-LOADER-FAILED" {
		parseT.Fatalf("expected log message assertion to resolve loader failure code, got %+v", parseMatch2)
	}
}

// TestFixtureRenderCountAndWarningAssertions validates render-count and warning assertion helpers.
func TestFixtureRenderCountAndWarningAssertions(parseT *testing.T) {
	parseFixture := New(parseT)
	parseFixture.Render(ui.CreateElement(counterHarnessApp))
	parseFixture.ClickByID("increment")

	parseSignals := parseFixture.BuildRenderCounts()
	if len(parseSignals) == 0 {
		parseT.Fatalf("expected render-count helper to expose profiling traces")
	}
	parseSignal := RenderCountSignal{}
	for _, parseCandidate := range parseSignals {
		if parseCandidate.RerenderCount > 0 {
			parseSignal = parseCandidate
			break
		}
	}
	if parseSignal.Name == "" && parseSignal.Path == "" {
		parseT.Fatalf("expected one render-count signal with rerenders, got %+v", parseSignals)
	}
	parseComponent := parseSignal.Name
	if parseComponent == "" {
		parseComponent = parseSignal.Path
	}
	parseRenderSignal := parseFixture.ApplyRenderCountMax(parseComponent, parseSignal.RenderCount)
	if parseRenderSignal.RenderCount != parseSignal.RenderCount {
		parseT.Fatalf("expected render-count assertion helper to resolve %q, got %+v", parseComponent, parseRenderSignal)
	}
	parseRerenderSignal := parseFixture.ApplyRenderRerenderMax(parseComponent, parseSignal.RerenderCount)
	if parseRerenderSignal.RerenderCount != parseSignal.RerenderCount {
		parseT.Fatalf("expected rerender assertion helper to resolve %q, got %+v", parseComponent, parseRerenderSignal)
	}

	runtime.ClearDiagnostics()
	runtime.ClearLogs()
	parseFixture.ApplyWarningNone()

	runtime.ReportDiagnostic("runtime", runtime.DiagnosticWarning, "test warning for fixture warning assertions")
	runtime.ReportLogWithFields("runtime", runtime.LogWarn, runtime.DiagnosticCorrectness, "test warn log for fixture warning assertions", "", nil)
	runtime.ReportDiagnostic("runtime", runtime.DiagnosticInfo, "test info diagnostic that should be excluded from warning counts")
	runtime.ReportLogWithFields("runtime", runtime.LogError, runtime.DiagnosticCorrectness, "test error log that should be excluded from warning counts", "", nil)

	parseWarnings := parseFixture.BuildWarningDiagnostics()
	if len(parseWarnings) != 1 {
		parseT.Fatalf("expected one warning diagnostic, got %+v", parseWarnings)
	}
	parseWarnLogs := parseFixture.BuildWarningLogs()
	if len(parseWarnLogs) != 1 {
		parseT.Fatalf("expected one warn-level log, got %+v", parseWarnLogs)
	}
	parseFixture.ApplyWarningCountMax(2)
}
