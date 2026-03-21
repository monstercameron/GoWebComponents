//go:build js && wasm
// +build js,wasm

package routertest

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	appRouter "github.com/monstercameron/GoWebComponents/router"
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
