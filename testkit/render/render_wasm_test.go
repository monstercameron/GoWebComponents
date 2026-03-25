//go:build js && wasm
// +build js,wasm

package render

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
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
	count := ui.UseState(0)
	increment := ui.UseEvent(func() {
		count.Update(func(previous int) int {
			return previous + 1
		})
	})

	return html.Div(html.Props{ID: "counter-root"},
		html.P(html.Props{ID: "count-label"}, html.Text(fmt.Sprintf("Count: %d", count.Get()))),
		html.Button(html.Props{ID: "increment", Type: "button", OnClick: increment}, html.Text("Increment")),
	)
}

func inputHarnessApp() ui.Node {
	value := ui.UseState("")
	handleInput := ui.UseEvent(func(event ui.InputEvent) {
		value.Set(event.GetValue())
	})

	return html.Div(html.Props{ID: "input-root"},
		html.Input(html.Props{ID: "name-input", Type: "text", OnInput: handleInput}),
		html.P(html.Props{ID: "name-value"}, html.Text("Value: "+value.Get())),
	)
}

func transitionHarnessApp() ui.Node {
	count := ui.UseState(0)
	handleClick := ui.UseEvent(func() {
		ui.StartTransition(func() {
			count.Update(func(previous int) int {
				return previous + 1
			})
		})
	})

	return html.Div(html.Props{ID: "transition-root"},
		html.P(html.Props{ID: "transition-count"}, html.Text(fmt.Sprintf("Transition Count: %d", count.Get()))),
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

func TestFixtureRenderAndQuery(t *testing.T) {
	fixture := New(t)
	fixture.Render(ui.CreateElement(staticHarnessApp))

	heading := fixture.ByID("main-heading")
	if heading == nil || heading.Text() != "Hello Harness" {
		t.Fatalf("expected heading text, got %#v", heading)
	}
	if match := fixture.ByText("Save"); match == nil || match.Tag() != "button" {
		t.Fatalf("expected to find button by text, got %#v", match)
	}
	buttons := fixture.AllByTag("button")
	if len(buttons) != 1 {
		t.Fatalf("expected one button, got %d", len(buttons))
	}
	if buttons[0].Attr("id") != "primary-button" {
		t.Fatalf("expected queried button id to be preserved, got %q", buttons[0].Attr("id"))
	}
	if button := fixture.ByRole("button", "Save"); button == nil || button.Attr("id") != "primary-button" {
		t.Fatalf("expected role query to find the button, got %#v", button)
	}
	if field := fixture.ByRole("textbox", "Search Catalog"); field == nil || field.Attr("id") != "search-input" {
		t.Fatalf("expected role query to resolve aria-labelledby accessible name, got %#v", field)
	}
	if name := fixture.ByID("search-input").Name(); name != "Search Catalog" {
		t.Fatalf("expected stable accessible name, got %q", name)
	}
	if textboxes := fixture.AllByRole("textbox"); len(textboxes) != 1 {
		t.Fatalf("expected one textbox role match, got %d", len(textboxes))
	}
}

// TestFixtureAccessibilityFirstQueriesAndAssertions validates label, description, and live-region helpers.
func TestFixtureAccessibilityFirstQueriesAndAssertions(t *testing.T) {
	fixture := New(t)
	fixture.Render(ui.CreateElement(staticHarnessApp))

	if field := fixture.ByLabel("Search Catalog"); field == nil || field.Attr("id") != "search-input" {
		t.Fatalf("expected label query to find search input, got %#v", field)
	}
	if field := fixture.ByDescription("Type to filter"); field == nil || field.Attr("id") != "search-input" {
		t.Fatalf("expected description query to find search input, got %#v", field)
	}
	if region := fixture.ByLiveRegion("polite", "Saved profile"); region == nil || region.Attr("id") != "save-live-region" {
		t.Fatalf("expected live-region query to find save status, got %#v", region)
	}

	if field := fixture.ApplyByLabel("Search Catalog"); field.Attr("id") != "search-input" {
		t.Fatalf("expected ApplyByLabel to return search input, got %#v", field)
	}
	if field := fixture.ApplyByDescription("Type to filter"); field.Attr("id") != "search-input" {
		t.Fatalf("expected ApplyByDescription to return search input, got %#v", field)
	}
	if region := fixture.ApplyByLiveRegion("polite", "Saved profile"); region.Attr("id") != "save-live-region" {
		t.Fatalf("expected ApplyByLiveRegion to return save live region, got %#v", region)
	}
	if button := fixture.ApplyByRole("button", "Save"); button.Attr("id") != "primary-button" {
		t.Fatalf("expected ApplyByRole to return primary button, got %#v", button)
	}
}

func TestFixtureExposesRenderedPropertiesForStateUpdates(t *testing.T) {
	fixture := New(t)
	fixture.Render(ui.CreateElement(counterHarnessApp))

	if got := fixture.ByID("count-label").Text(); got != "Count: 0" {
		t.Fatalf("expected initial count label, got %q", got)
	}

	handler, ok := fixture.ByID("increment").Property("onclick").(func())
	if !ok {
		t.Fatalf("expected onclick property with func() signature, got %T", fixture.ByID("increment").Property("onclick"))
	}
	handler()

	if got := fixture.ByID("count-label").Text(); got != "Count: 1" {
		t.Fatalf("expected updated count label after invoking onclick, got %q", got)
	}
}

func TestFixtureRerenderReplacesTree(t *testing.T) {
	fixture := New(t)
	fixture.Render(ui.CreateElement(staticHarnessApp))
	fixture.Rerender(html.Div(html.Props{ID: "replacement"}, html.Text("Replaced")))

	if fixture.ByID("main-heading") != nil {
		t.Fatal("expected original tree to be replaced")
	}
	replacement := fixture.ByID("replacement")
	if replacement == nil || replacement.Text() != "Replaced" {
		t.Fatalf("expected rerendered replacement tree, got %#v", replacement)
	}
}

func TestFixtureInputHelperDispatchesAndSettles(t *testing.T) {
	fixture := New(t)
	fixture.Render(ui.CreateElement(inputHarnessApp))

	fixture.InputByID("name-input", "Cam")

	if got := fixture.ByID("name-value").Text(); got != "Value: Cam" {
		t.Fatalf("expected input helper to update rendered value, got %q", got)
	}
}

func TestFixtureQueuedSchedulerClickHelperStabilizesTransitionWork(t *testing.T) {
	fixture := New(t, WithQueuedScheduler())
	fixture.Render(ui.CreateElement(transitionHarnessApp))

	fixture.ClickByID("transition-increment")

	if got := fixture.ByID("transition-count").Text(); got != "Transition Count: 1" {
		t.Fatalf("expected queued click helper to stabilize transition work, got %q", got)
	}
}

// TestFixtureOverlayHelpersTrackStackAndOutsideDismiss validates overlay helper coverage for portal-backed stacks.
func TestFixtureOverlayHelpersTrackStackAndOutsideDismiss(t *testing.T) {
	fixture := New(t)
	fixture.Render(ui.CreateElement(overlayHarnessApp))

	if surfaces := fixture.BuildOverlaySurfaces(); len(surfaces) != 0 {
		t.Fatalf("expected no overlays before opening, got %+v", surfaces)
	}

	fixture.ClickByID("open-parent")
	fixture.ClickByID("open-child")

	surfaces := fixture.BuildOverlaySurfaces()
	if len(surfaces) != 2 {
		t.Fatalf("expected two overlay surfaces, got %+v", surfaces)
	}
	if fixture.BuildOverlayEscapeSurfaceID() != "child-overlay" {
		t.Fatalf("expected child overlay to own escape handling, got %q", fixture.BuildOverlayEscapeSurfaceID())
	}
	if fixture.BuildOverlayOutsideSurfaceID() != "child-overlay" {
		t.Fatalf("expected child overlay to own outside-click handling, got %q", fixture.BuildOverlayOutsideSurfaceID())
	}
	if fixture.BuildOverlayFocusSurfaceID() != "child-overlay" {
		t.Fatalf("expected child overlay to own trap focus, got %q", fixture.BuildOverlayFocusSurfaceID())
	}
	if fixture.BuildOverlayPortalTargetID("parent-overlay") != "portal-root" || fixture.BuildOverlayPortalTargetID("child-overlay") != "portal-root" {
		t.Fatalf("expected both overlays to render into portal-root, got parent=%q child=%q", fixture.BuildOverlayPortalTargetID("parent-overlay"), fixture.BuildOverlayPortalTargetID("child-overlay"))
	}
	if !fixture.BuildOverlayScrollLockActive() {
		t.Fatalf("expected scroll lock helper to report active while overlays are open; overflow=%q surfaces=%+v", fixture.BuildOverlayBodyOverflow(), surfaces)
	}

	if !fixture.HandleOverlayOutsideClick("child-overlay") {
		t.Fatal("expected child overlay outside-click helper to dispatch through backdrop")
	}
	if fixture.ByID("child-overlay") != nil {
		t.Fatalf("expected child overlay to close after outside click")
	}
	if fixture.ByID("parent-overlay") == nil {
		t.Fatalf("expected parent overlay to remain open after child outside click")
	}
	if fixture.BuildOverlayEscapeSurfaceID() != "parent-overlay" || fixture.BuildOverlayFocusSurfaceID() != "parent-overlay" {
		t.Fatalf("expected parent overlay to inherit top ownership after child dismissal, got escape=%q focus=%q", fixture.BuildOverlayEscapeSurfaceID(), fixture.BuildOverlayFocusSurfaceID())
	}

	if !fixture.HandleOverlayOutsideClick("parent-overlay") {
		t.Fatal("expected parent overlay outside-click helper to dispatch through backdrop")
	}
	if fixture.ByID("parent-overlay") != nil {
		t.Fatalf("expected parent overlay to close after outside click")
	}
	if fixture.BuildOverlayScrollLockActive() {
		t.Fatalf("expected scroll lock helper to report inactive after all overlays close; overflow=%q surfaces=%+v", fixture.BuildOverlayBodyOverflow(), fixture.BuildOverlaySurfaces())
	}
}

func TestFixtureDiagnosticsAndLogsHelpersExposeStructuredAssertions(t *testing.T) {
	fixture := New(t)
	fixture.Render(ui.CreateElement(staticHarnessApp))

	runtime.ReportDiagnostic("runtime", runtime.DiagnosticWarning, "hydration fell back to client rendering for <div>: DOM node <span> did not match expected <div>")
	runtime.ReportLogWithFields("router", runtime.LogError, runtime.DiagnosticCorrectness, "route loader failed", "", nil)

	diagnostics := fixture.BuildDiagnostics()
	if len(diagnostics) == 0 {
		t.Fatalf("expected diagnostics helper to expose runtime diagnostics")
	}
	logs := fixture.BuildLogs()
	if len(logs) == 0 {
		t.Fatalf("expected logs helper to expose runtime logs")
	}

	diagnostic := fixture.ApplyDiagnosticCode("GWC-HYDRATION-FALLBACK")
	if diagnostic.Source != "runtime" || !diagnostic.Recoverable {
		t.Fatalf("expected hydration fallback diagnostic metadata, got %+v", diagnostic)
	}
	if match := fixture.ApplyDiagnosticMessage("hydration fell back"); match.Code != "GWC-HYDRATION-FALLBACK" {
		t.Fatalf("expected diagnostic message assertion to resolve hydration fallback code, got %+v", match)
	}

	log := fixture.ApplyLogCode("GWC-ROUTER-LOADER-FAILED")
	if log.Domain != "router" || log.Level != string(runtime.LogError) {
		t.Fatalf("expected router loader log metadata, got %+v", log)
	}
	if match := fixture.ApplyLogMessage("route loader failed"); match.Code != "GWC-ROUTER-LOADER-FAILED" {
		t.Fatalf("expected log message assertion to resolve loader failure code, got %+v", match)
	}
}
