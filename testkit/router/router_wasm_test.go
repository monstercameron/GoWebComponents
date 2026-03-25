//go:build js && wasm
// +build js,wasm

package routertest

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	appRouter "github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

func userRoute(_ appRouter.Attrs) *appRouter.Element {
	params := appRouter.UseParams()
	query := appRouter.UseQuery()
	return html.Div(html.Props{ID: "user-route"},
		html.Text(fmt.Sprintf("user:%s|tab:%s", params.Get("id"), query.Get("tab"))),
	)
}

func homeRoute(_ appRouter.Attrs) *appRouter.Element {
	return html.Div(html.Props{ID: "home-route"}, html.Text("home"))
}

func historyRoute(_ appRouter.Attrs) *appRouter.Element {
	params := appRouter.UseParams()
	return html.Div(html.Props{ID: "history-route"}, html.Text("history:"+params.Get("slug")))
}

func accessibilityRoute(_ appRouter.Attrs) *appRouter.Element {
	state := ui.UseState("")
	handleInput := ui.UseEvent(func(event ui.InputEvent) {
		state.Set(event.GetValue())
	})
	handleSave := ui.UseEvent(func() {
		state.Set("Saved")
	})

	return html.Div(html.Props{ID: "accessibility-route"},
		html.Span(html.Props{ID: "route-name-label"}, html.Text("Name")),
		html.P(html.Props{ID: "route-name-description"}, html.Text("Type your name")),
		html.Input(html.Props{ID: "route-name-input", Type: "text", OnInput: handleInput, Raw: map[string]interface{}{
			"aria-labelledby":  "route-name-label",
			"aria-describedby": "route-name-description",
		}}),
		html.Button(html.Props{ID: "route-save", Type: "button", OnClick: handleSave}, html.Text("Save route")),
		html.Div(html.Props{ID: "route-live", Role: "status", Raw: map[string]interface{}{"aria-live": "polite"}}, html.Text(state.Get())),
	)
}

func TestHashFixtureSetsInitialPathAndExposesParamsAndQuery(t *testing.T) {
	fixture := NewHash(t)
	fixture.Register("/", homeRoute)
	fixture.Register("/users/:id", userRoute)

	fixture.SetPath("/users/42?tab=settings")

	if got := fixture.Path(); got != "/users/42" {
		t.Fatalf("expected current path /users/42, got %q", got)
	}
	if got := fixture.Params()["id"]; got != "42" {
		t.Fatalf("expected id param 42, got %q", got)
	}
	if got := fixture.Query().Get("tab"); got != "settings" {
		t.Fatalf("expected tab query settings, got %q", got)
	}
	if match := fixture.ByID("user-route"); match == nil || match.Text() != "user:42|tab:settings" {
		t.Fatalf("expected rendered user route text, got %#v", match)
	}
}

func TestHistoryFixtureNavigatesAndRendersCurrentRoute(t *testing.T) {
	fixture := NewHistory(t)
	fixture.Register("/", homeRoute)
	fixture.Register("/docs/:slug", historyRoute)
	fixture.SetPath("/")

	fixture.Navigate("/docs/getting-started")

	if got := fixture.Path(); got != "/docs/getting-started" {
		t.Fatalf("expected current path /docs/getting-started, got %q", got)
	}
	if got := fixture.Params()["slug"]; got != "getting-started" {
		t.Fatalf("expected slug param getting-started, got %q", got)
	}
}

func TestRouterFixtureDelegatesAccessibilityQueriesAndEvents(t *testing.T) {
	fixture := NewHash(t)
	fixture.Register("/a11y", accessibilityRoute)
	fixture.SetPath("/a11y")

	if button := fixture.ByRole("button", "Save route"); button == nil || button.Attr("id") != "route-save" {
		t.Fatalf("expected role delegation to match route save button, got %#v", button)
	}
	if fields := fixture.AllByRole("textbox"); len(fields) != 1 || fields[0].Attr("id") != "route-name-input" {
		t.Fatalf("expected one textbox role match from router fixture, got %#v", fields)
	}
	if field := fixture.ByLabel("Name"); field == nil || field.Attr("id") != "route-name-input" {
		t.Fatalf("expected label delegation to find route input, got %#v", field)
	}
	if field := fixture.ByDescription("Type your name"); field == nil || field.Attr("id") != "route-name-input" {
		t.Fatalf("expected description delegation to find route input, got %#v", field)
	}
	if fixture.ByLiveRegion("polite", "Saved") != nil {
		t.Fatalf("expected live-region text to be empty before interaction")
	}

	fixture.InputByID("route-name-input", "Cam")
	if got := fixture.ByID("route-name-input").Attr("value"); got != "Cam" {
		t.Fatalf("expected delegated input helper to set route input value, got %q", got)
	}

	fixture.ClickByID("route-save")
	if region := fixture.ApplyByLiveRegion("polite", "Saved"); region.Attr("id") != "route-live" {
		t.Fatalf("expected live-region assertion delegation to find route status, got %#v", region)
	}
	if field := fixture.ApplyByLabel("Name"); field.Attr("id") != "route-name-input" {
		t.Fatalf("expected ApplyByLabel delegation to return route input, got %#v", field)
	}
	if button := fixture.ApplyByRole("button", "Save route"); button.Attr("id") != "route-save" {
		t.Fatalf("expected ApplyByRole delegation to return route save button, got %#v", button)
	}
}
