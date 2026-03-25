//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/hotreload"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const sessionAtomID = "catalog-protected-routes-session"

const (
	sessionUnknown         = "unknown"
	sessionAuthenticated   = "authenticated"
	sessionUnauthenticated = "unauthenticated"
)

type demoSession struct {
	Status         string
	Subject        string
	CanViewBilling bool
}

type workspaceSummary struct {
	Subject      string
	Revision     int
	Projects     int
	Alerts       int
	LastSnapshot string
}

var liveSession = demoSession{Status: sessionUnauthenticated}
var workspaceRevision int32

func workspaceSummaryKey(subject string) string {
	trimmed := strings.TrimSpace(subject)
	if trimmed == "" {
		return ""
	}
	return "workspace:summary:" + trimmed
}

func loadWorkspaceSummary(ctx context.Context, subject string) (workspaceSummary, error) {
	select {
	case <-ctx.Done():
		return workspaceSummary{}, ctx.Err()
	case <-time.After(70 * time.Millisecond):
	}

	revision := int(atomic.AddInt32(&workspaceRevision, 1))
	return workspaceSummary{
		Subject:      subject,
		Revision:     revision,
		Projects:     6 + (revision % 3),
		Alerts:       1 + (revision % 2),
		LastSnapshot: time.Now().Format("15:04:05"),
	}, nil
}

func setLiveSession(atom state.Atom[demoSession], next demoSession) {
	liveSession = next
	atom.Set(next)
}

func homePageView() ui.Node {
	nav := router.UseNavigate()
	sessionAtom := state.UseAtom(sessionAtomID, liveSession)
	session := sessionAtom.Get()

	return shared.ExamplePage(
		"Protected Routes",
		"router guards, return-to helpers, and fetch.LoadCached",
		"Demonstrate the current shipped protected-route pattern: synchronous guard redirects for known signed-out sessions, manual authorizing and unauthorized UI, safe return_to preservation, and shared cache reuse between a route loader and a component reader.",
		shared.ExamplePanel("Entry states",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Use these launch buttons to enter the same protected route from different auth states. Signed-out navigation redirects through /login with a bounded return_to value. Unknown auth intentionally lands on the protected route first so the page can render manual authorizing UI while the session resolves.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Open signed out", ui.UseEvent(func() {
					setLiveSession(sessionAtom, demoSession{Status: sessionUnauthenticated})
					nav.Navigate("/workspace")
				})),
				shared.ExampleButton("Open while resolving", ui.UseEvent(func() {
					setLiveSession(sessionAtom, demoSession{Status: sessionUnknown})
					nav.Navigate("/workspace")
				})),
				shared.ExampleButton("Open signed in", ui.UseEvent(func() {
					setLiveSession(sessionAtom, demoSession{Status: sessionAuthenticated, Subject: "atlas-admin", CanViewBilling: false})
					nav.Navigate("/workspace")
				})),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Session", session.Status),
				shared.ExampleStat("Subject", emptyFallback(session.Subject, "guest")),
				shared.ExampleStat("Billing claim", fmt.Sprintf("%t", session.CanViewBilling)),
			),
		),
		shared.ExamplePanel("Security boundary",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("The guard is a UX gate, not authorization proof. The example keeps the redirect, authorizing shell, and unauthorized fallback on the client, but a real server still has to enforce access on SSR responses, API handlers, and mutations.")),
			shared.ExampleCode(
				`returnTo := router.PreserveReturnTo(ctx.Path, ctx.Query.Values())`,
				`return router.RedirectNavigation("/login?" + url.Values{router.ReturnToParam: {returnTo}}.Encode())`,
				`summary, err := fetch.LoadCached(ctx, cacheKey, loader, fetch.CacheOptions{StaleAfter: 20 * time.Second})`,
			),
		),
	)
}

func homePage(router.Attrs) *router.Element {
	return ui.CreateElement(homePageView)
}

func loginPageView() ui.Node {
	nav := router.UseNavigate()
	query := router.UseQuery()
	sessionAtom := state.UseAtom(sessionAtomID, liveSession)
	session := sessionAtom.Get()
	returnTo := router.ReadReturnTo(query.Values(), "/workspace")

	return shared.ExamplePage(
		"Protected Route Login",
		"router.ReadReturnTo and manual redirect recovery",
		"This route is the current unauthorized fallback target. It reads the bounded return_to payload, lets the user choose a claim set, and uses replacement navigation so the transient login step does not linger in history after sign-in.",
		shared.ExamplePanel("Redirect recovery",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Current return target: "+returnTo)),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Sign in and continue", ui.UseEvent(func() {
					setLiveSession(sessionAtom, demoSession{Status: sessionAuthenticated, Subject: "atlas-admin", CanViewBilling: false})
					nav.Replace(returnTo)
				})),
				shared.ExampleButton("Sign in with billing access", ui.UseEvent(func() {
					setLiveSession(sessionAtom, demoSession{Status: sessionAuthenticated, Subject: "atlas-finance", CanViewBilling: true})
					nav.Replace(returnTo)
				})),
				shared.ExampleButton("Stay signed out", ui.UseEvent(func() {
					setLiveSession(sessionAtom, demoSession{Status: sessionUnauthenticated})
					nav.Replace("/")
				})),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Session", session.Status),
				shared.ExampleStat("Subject", emptyFallback(session.Subject, "guest")),
				shared.ExampleStat("Billing claim", fmt.Sprintf("%t", session.CanViewBilling)),
			),
		),
	)
}

func loginPage(router.Attrs) *router.Element {
	return ui.CreateElement(loginPageView)
}

func workspacePageView(props router.Attrs) ui.Node {
	nav := router.UseNavigate()
	search := router.UseSearchParams()
	revalidator := router.UseRevalidator()
	sessionAtom := state.UseAtom(sessionAtomID, liveSession)
	session := sessionAtom.Get()

	cacheKey, _ := props["cacheKey"].(string)
	loaderSummary, _ := props["summary"].(workspaceSummary)
	tab := search.Get("tab")
	if tab == "" {
		tab = "overview"
	}

	resource := fetch.UseCachedResource(cacheKey, func(ctx context.Context) (workspaceSummary, error) {
		return loadWorkspaceSummary(ctx, session.Subject)
	}, fetch.CacheOptions{
		StaleAfter:   20 * time.Second,
		MaxAge:       90 * time.Second,
		DisposeAfter: 3 * time.Minute,
	})
	cacheState := resource.Get()

	resolveSignedIn := ui.UseEvent(func() {
		setLiveSession(sessionAtom, demoSession{Status: sessionAuthenticated, Subject: "atlas-admin", CanViewBilling: false})
		revalidator.Revalidate()
	})
	resolveGuest := ui.UseEvent(func() {
		setLiveSession(sessionAtom, demoSession{Status: sessionUnauthenticated})
		values := url.Values{}
		values.Set(router.ReturnToParam, router.PreserveReturnTo("/workspace", search.Values()))
		nav.Replace("/login?" + values.Encode())
	})
	showOverview := ui.UseEvent(func() { search.Replace("tab", "overview") })
	showBilling := ui.UseEvent(func() { search.Replace("tab", "billing") })
	grantBilling := ui.UseEvent(func() {
		setLiveSession(sessionAtom, demoSession{Status: sessionAuthenticated, Subject: session.Subject, CanViewBilling: true})
	})
	refreshBoth := ui.UseEvent(func() {
		if cacheKey != "" {
			fetch.InvalidateResource(cacheKey)
		}
		revalidator.Revalidate()
	})
	disposeAndRefresh := ui.UseEvent(func() {
		resource.Dispose()
		revalidator.Revalidate()
	})
	signOut := ui.UseEvent(func() {
		setLiveSession(sessionAtom, demoSession{Status: sessionUnauthenticated})
		values := url.Values{}
		values.Set(router.ReturnToParam, router.PreserveReturnTo("/workspace", search.Values()))
		nav.Replace("/login?" + values.Encode())
	})

	if session.Status == sessionUnknown {
		return shared.ExamplePage(
			"Protected Workspace",
			"manual authorizing UI",
			"The router currently ships synchronous guards only, so pending-auth UI is still application-owned. This page intentionally renders an authorizing shell while the auth state is unresolved, then reruns the route loader once the session becomes concrete.",
			shared.ExamplePanel("Authorizing",
				html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Session state is still unknown, so the route holds on to explicit authorizing UI instead of guessing. Pick an outcome below to complete the flow.")),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					shared.ExampleButton("Restore signed-in session", resolveSignedIn),
					shared.ExampleButton("Resolve as guest", resolveGuest),
				),
			),
		)
	}

	if tab == "billing" && !session.CanViewBilling {
		return shared.ExamplePage(
			"Protected Workspace",
			"manual unauthorized fallback",
			"This route stays mounted, but the billing section renders explicit unauthorized content because the current user lacks the claim required for that subsection.",
			shared.ExamplePanel("Unauthorized billing section",
				html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Router-level Unauthorized content is still future work. Today the app renders its own fallback for forbidden subsections, keeps the rest of the shell intact, and lets the user request a different claim set or navigate elsewhere.")),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					shared.ExampleButton("Back to overview", showOverview),
					shared.ExampleButton("Grant billing claim", grantBilling),
					shared.ExampleButton("Sign out", signOut),
				),
			),
		)
	}

	cacheRevision := "cold"
	cacheSnapshot := "none"
	if cacheState.Ready {
		cacheRevision = fmt.Sprintf("%d", cacheState.Value.Revision)
		cacheSnapshot = cacheState.Value.LastSnapshot
	}

	return shared.ExamplePage(
		"Protected Workspace",
		"guarded loader and shared cache",
		"The route loader seeds the shared cache through fetch.LoadCached(...), and the component reads the same key through fetch.UseCachedResource(...). That keeps route-scoped data and component-level cache state aligned without a second fetch path.",
		shared.ExamplePanel("Protected shell",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Overview tab", showOverview),
				shared.ExampleButton("Billing tab", showBilling),
				shared.ExampleButton("Revalidate route + cache", refreshBoth),
				shared.ExampleButton("Dispose cache entry", disposeAndRefresh),
				shared.ExampleButton("Sign out", signOut),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Active tab", tab),
				shared.ExampleStat("Subject", session.Subject),
				shared.ExampleStat("Loader revision", fmt.Sprintf("%d", loaderSummary.Revision)),
				shared.ExampleStat("Shared cache revision", cacheRevision),
			),
			html.P(html.Props{Class: "mt-4 leading-7 text-slate-300"}, html.Text("The route loader and component reader share key "+emptyFallback(cacheKey, "<none>")+". If the route loader already filled it, the widget below starts ready instead of kicking off a second request.")),
		),
		shared.ExamplePanel("Shared cache widget",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-5"},
				shared.ExampleStat("Ready", fmt.Sprintf("%t", cacheState.Ready)),
				shared.ExampleStat("Loading", fmt.Sprintf("%t", cacheState.Loading)),
				shared.ExampleStat("Stale", fmt.Sprintf("%t", cacheState.Stale)),
				shared.ExampleStat("Projects", fmt.Sprintf("%d", cacheState.Value.Projects)),
				shared.ExampleStat("Snapshot", cacheSnapshot),
			),
			html.P(html.Props{Class: "mt-4 leading-7 text-slate-300"}, html.Text("Cache policy here uses StaleAfter for background refresh, MaxAge for hard expiry, and DisposeAfter for idle cleanup. The explicit Dispose action demonstrates how long-lived apps can clear entries intentionally instead of waiting for eventual eviction.")),
		),
		shared.ExamplePanel("Security guidance",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Nothing on this page makes the browser authoritative. The same account and billing checks still belong on the server for HTML responses, data endpoints, and mutations. The client route gate only improves navigation behavior, pending UX, and cache reuse once the server has already decided what is allowed.")),
			shared.ExampleCode(
				`summary, err := fetch.LoadCached(ctx, cacheKey, loadWorkspaceSummary, fetch.CacheOptions{StaleAfter: 20 * time.Second, MaxAge: 90 * time.Second, DisposeAfter: 3 * time.Minute})`,
				`resource := fetch.UseCachedResource(cacheKey, loadWorkspaceSummary, fetch.CacheOptions{StaleAfter: 20 * time.Second, MaxAge: 90 * time.Second, DisposeAfter: 3 * time.Minute})`,
			),
		),
	)
}

func workspacePage(props router.Attrs) *router.Element {
	return ui.CreateElement(func() ui.Node {
		return workspacePageView(props)
	})
}

func routeLoading(props router.Attrs) *router.Element {
	return ui.CreateElement(func() ui.Node {
		return shared.ExamplePage(
			"Protected Workspace",
			"route loading state",
			"The protected route loader is filling the shared cache before the final page renders.",
			shared.ExamplePanel("Loading",
				html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Loading protected route data...")),
			),
		)
	})
}

func routeError(props router.Attrs) *router.Element {
	message, _ := props["error"].(string)
	return ui.CreateElement(func() ui.Node {
		return shared.ExamplePage(
			"Protected Workspace",
			"route error state",
			"Route loader failures stay explicit and user-visible.",
			shared.ExamplePanel("Error",
				html.P(html.Props{Class: "mt-3 leading-7 text-rose-200"}, html.Text(emptyFallback(message, "Route load failed"))),
			),
		)
	})
}

func protectedGuard(ctx router.RouteContext) router.GuardResult {
	if liveSession.Status != sessionUnauthenticated {
		return router.AllowNavigation()
	}
	values := url.Values{}
	values.Set(router.ReturnToParam, router.PreserveReturnTo(ctx.Path, ctx.Query.Values()))
	return router.RedirectNavigation("/login?" + values.Encode())
}

func workspaceLoader(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
	if liveSession.Status != sessionAuthenticated {
		return router.Attrs{}, nil
	}

	cacheKey := workspaceSummaryKey(liveSession.Subject)
	summary, err := fetch.LoadCached(ctx, cacheKey, func(loadCtx context.Context) (workspaceSummary, error) {
		return loadWorkspaceSummary(loadCtx, liveSession.Subject)
	}, fetch.CacheOptions{
		StaleAfter:   20 * time.Second,
		MaxAge:       90 * time.Second,
		DisposeAfter: 3 * time.Minute,
	})
	if err != nil {
		return nil, err
	}
	return router.Attrs{
		"cacheKey": cacheKey,
		"summary":  summary,
	}, nil
}

func emptyFallback(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func main() {
	utils.DisableAllDebug()
	hotreload.Enable()
	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	r.Register("/", homePage)
	r.Register("/login", loginPage)
	r.Register("/workspace", workspacePage, router.Options{
		BeforeEnter: protectedGuard,
		Loader:      workspaceLoader,
		Loading:     routeLoading,
		Error:       routeError,
	})
	r.Mount("#app")
	select {}
}
