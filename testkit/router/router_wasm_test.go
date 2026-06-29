//go:build js && wasm

package routertest

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	appRouter "github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

func userRoute(_ appRouter.Attrs) *appRouter.Element {
	parseParams := appRouter.UseParams()
	parseQuery := appRouter.UseQuery()
	return html.Div(html.Props{ID: "user-route"},
		html.Text(fmt.Sprintf("user:%s|tab:%s", parseParams.Get("id"), parseQuery.Get("tab"))),
	)
}

func homeRoute(_ appRouter.Attrs) *appRouter.Element {
	return html.Div(html.Props{ID: "home-route"}, html.Text("home"))
}

func historyRoute(_ appRouter.Attrs) *appRouter.Element {
	parseParams := appRouter.UseParams()
	return html.Div(html.Props{ID: "history-route"}, html.Text("history:"+parseParams.Get("slug")))
}

func accessibilityRoute(_ appRouter.Attrs) *appRouter.Element {
	parseState := ui.UseState("")
	handleInput := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseState.Set(parseEvent.GetValue())
	})
	handleSave := ui.UseEvent(func() {
		parseState.Set("Saved")
	})

	return html.Div(html.Props{ID: "accessibility-route"},
		html.Span(html.Props{ID: "route-name-label"}, html.Text("Name")),
		html.P(html.Props{ID: "route-name-description"}, html.Text("Type your name")),
		html.Input(html.Props{ID: "route-name-input", Type: "text", OnInput: handleInput, Raw: map[string]interface{}{
			"aria-labelledby":  "route-name-label",
			"aria-describedby": "route-name-description",
		}}),
		html.Button(html.Props{ID: "route-save", Type: "button", OnClick: handleSave}, html.Text("Save route")),
		html.Div(html.Props{ID: "route-live", Role: "status", Raw: map[string]interface{}{"aria-live": "polite"}}, html.Text(parseState.Get())),
	)
}

func TestHashFixtureSetsInitialPathAndExposesParamsAndQuery(parseT *testing.T) {
	parseFixture := NewHash(parseT)
	parseFixture.Register("/", homeRoute)
	parseFixture.Register("/users/:id", userRoute)

	parseFixture.SetPath("/users/42?tab=settings")

	if parseGot := parseFixture.Path(); parseGot != "/users/42" {
		parseT.Fatalf("expected current path /users/42, got %q", parseGot)
	}
	if parseGot2 := parseFixture.Params()["id"]; parseGot2 != "42" {
		parseT.Fatalf("expected id param 42, got %q", parseGot2)
	}
	if parseGot3 := parseFixture.Query().Get("tab"); parseGot3 != "settings" {
		parseT.Fatalf("expected tab query settings, got %q", parseGot3)
	}
	if parseMatch := parseFixture.ByID("user-route"); parseMatch == nil || parseMatch.Text() != "user:42|tab:settings" {
		parseT.Fatalf("expected rendered user route text, got %#v", parseMatch)
	}
}

func TestHistoryFixtureNavigatesAndRendersCurrentRoute(parseT *testing.T) {
	parseFixture := NewHistory(parseT)
	parseFixture.Register("/", homeRoute)
	parseFixture.Register("/docs/:slug", historyRoute)
	parseFixture.SetPath("/")

	parseFixture.Navigate("/docs/getting-started")

	if parseGot := parseFixture.Path(); parseGot != "/docs/getting-started" {
		parseT.Fatalf("expected current path /docs/getting-started, got %q", parseGot)
	}
	if parseGot2 := parseFixture.Params()["slug"]; parseGot2 != "getting-started" {
		parseT.Fatalf("expected slug param getting-started, got %q", parseGot2)
	}
}

func TestRouterFixtureDelegatesAccessibilityQueriesAndEvents(parseT *testing.T) {
	parseFixture := NewHash(parseT)
	parseFixture.Register("/a11y", accessibilityRoute)
	parseFixture.SetPath("/a11y")

	if parseButton := parseFixture.ByRole("button", "Save route"); parseButton == nil || parseButton.Attr("id") != "route-save" {
		parseT.Fatalf("expected role delegation to match route save button, got %#v", parseButton)
	}
	if parseFields := parseFixture.AllByRole("textbox"); len(parseFields) != 1 || parseFields[0].Attr("id") != "route-name-input" {
		parseT.Fatalf("expected one textbox role match from router fixture, got %#v", parseFields)
	}
	if parseField := parseFixture.ByLabel("Name"); parseField == nil || parseField.Attr("id") != "route-name-input" {
		parseT.Fatalf("expected label delegation to find route input, got %#v", parseField)
	}
	if parseField2 := parseFixture.ByDescription("Type your name"); parseField2 == nil || parseField2.Attr("id") != "route-name-input" {
		parseT.Fatalf("expected description delegation to find route input, got %#v", parseField2)
	}
	if parseFixture.ByLiveRegion("polite", "Saved") != nil {
		parseT.Fatalf("expected live-region text to be empty before interaction")
	}

	parseFixture.InputByID("route-name-input", "Cam")
	if parseGot := parseFixture.ByID("route-name-input").Attr("value"); parseGot != "Cam" {
		parseT.Fatalf("expected delegated input helper to set route input value, got %q", parseGot)
	}

	parseFixture.ClickByID("route-save")
	if parseRegion := parseFixture.ApplyByLiveRegion("polite", "Saved"); parseRegion.Attr("id") != "route-live" {
		parseT.Fatalf("expected live-region assertion delegation to find route status, got %#v", parseRegion)
	}
	if parseField3 := parseFixture.ApplyByLabel("Name"); parseField3.Attr("id") != "route-name-input" {
		parseT.Fatalf("expected ApplyByLabel delegation to return route input, got %#v", parseField3)
	}
	if parseButton2 := parseFixture.ApplyByRole("button", "Save route"); parseButton2.Attr("id") != "route-save" {
		parseT.Fatalf("expected ApplyByRole delegation to return route save button, got %#v", parseButton2)
	}
}
