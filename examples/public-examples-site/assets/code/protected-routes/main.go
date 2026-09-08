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

	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	"github.com/monstercameron/GoWebComponents/v6/fetch"
	"github.com/monstercameron/GoWebComponents/v6/hotreload"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/state"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
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

func workspaceSummaryKey(parseSubject string) string {
	parseTrimmed := strings.TrimSpace(parseSubject)
	if parseTrimmed == "" {
		return ""
	}
	return "workspace:summary:" + parseTrimmed
}

func loadWorkspaceSummary(parseCtx context.Context, parseSubject string) (workspaceSummary, error) {
	select {
	case <-parseCtx.Done():
		return workspaceSummary{}, parseCtx.Err()
	case <-time.After(70 * time.Millisecond):
	}

	parseRevision := int(atomic.AddInt32(&workspaceRevision, 1))
	return workspaceSummary{
		Subject:      parseSubject,
		Revision:     parseRevision,
		Projects:     6 + (parseRevision % 3),
		Alerts:       1 + (parseRevision % 2),
		LastSnapshot: time.Now().Format("15:04:05"),
	}, nil
}

func setLiveSession(parseAtom state.Atom[demoSession], parseNext demoSession) {
	liveSession = parseNext
	parseAtom.Set(parseNext)
}

func homePageView() ui.Node {
	parseNav := router.UseNavigate()
	parseSessionAtom := state.UseAtom(sessionAtomID, liveSession)
	parseSession := parseSessionAtom.Get()

	return shared.ExamplePage(
		"Protected Routes",
		"router guards, return-to helpers, and fetch.LoadCached",
		"Demonstrate the current shipped protected-route pattern: synchronous guard redirects for known signed-out sessions, manual authorizing and unauthorized UI, safe return_to preservation, and shared cache reuse between a route loader and a component reader.",
		shared.ExamplePanel("Entry states",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Use these launch buttons to enter the same protected route from different auth states. Signed-out navigation redirects through /login with a bounded return_to value. Unknown auth intentionally lands on the protected route first so the page can render manual authorizing UI while the session resolves.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Open signed out", ui.UseEvent(func() {
					setLiveSession(parseSessionAtom, demoSession{Status: sessionUnauthenticated})
					parseNav.Navigate("/workspace")
				})),
				shared.ExampleButton("Open while resolving", ui.UseEvent(func() {
					setLiveSession(parseSessionAtom, demoSession{Status: sessionUnknown})
					parseNav.Navigate("/workspace")
				})),
				shared.ExampleButton("Open signed in", ui.UseEvent(func() {
					setLiveSession(parseSessionAtom, demoSession{Status: sessionAuthenticated, Subject: "atlas-admin", CanViewBilling: false})
					parseNav.Navigate("/workspace")
				})),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Session", parseSession.Status),
				shared.ExampleStat("Subject", emptyFallback(parseSession.Subject, "guest")),
				shared.ExampleStat("Billing claim", fmt.Sprintf("%t", parseSession.CanViewBilling)),
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
	parseNav := router.UseNavigate()
	parseQuery := router.UseQuery()
	parseSessionAtom := state.UseAtom(sessionAtomID, liveSession)
	parseSession := parseSessionAtom.Get()
	parseReturnTo := router.ReadReturnTo(parseQuery.Values(), "/workspace")

	return shared.ExamplePage(
		"Protected Route Login",
		"router.ReadReturnTo and manual redirect recovery",
		"This route is the current unauthorized fallback target. It reads the bounded return_to payload, lets the user choose a claim set, and uses replacement navigation so the transient login step does not linger in history after sign-in.",
		shared.ExamplePanel("Redirect recovery",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Current return target: "+parseReturnTo)),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Sign in and continue", ui.UseEvent(func() {
					setLiveSession(parseSessionAtom, demoSession{Status: sessionAuthenticated, Subject: "atlas-admin", CanViewBilling: false})
					parseNav.Replace(parseReturnTo)
				})),
				shared.ExampleButton("Sign in with billing access", ui.UseEvent(func() {
					setLiveSession(parseSessionAtom, demoSession{Status: sessionAuthenticated, Subject: "atlas-finance", CanViewBilling: true})
					parseNav.Replace(parseReturnTo)
				})),
				shared.ExampleButton("Stay signed out", ui.UseEvent(func() {
					setLiveSession(parseSessionAtom, demoSession{Status: sessionUnauthenticated})
					parseNav.Replace("/")
				})),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Session", parseSession.Status),
				shared.ExampleStat("Subject", emptyFallback(parseSession.Subject, "guest")),
				shared.ExampleStat("Billing claim", fmt.Sprintf("%t", parseSession.CanViewBilling)),
			),
		),
	)
}

func loginPage(router.Attrs) *router.Element {
	return ui.CreateElement(loginPageView)
}

func workspacePageView(parseProps router.Attrs) ui.Node {
	parseNav := router.UseNavigate()
	parseSearch := router.UseSearchParams()
	parseRevalidator := router.UseRevalidator()
	parseSessionAtom := state.UseAtom(sessionAtomID, liveSession)
	parseSession := parseSessionAtom.Get()

	cacheKey, _ := parseProps["cacheKey"].(string)
	parseLoaderSummary, _ := parseProps["summary"].(workspaceSummary)
	parseTab := parseSearch.Get("tab")
	if parseTab == "" {
		parseTab = "overview"
	}

	parseResource := fetch.UseCachedResource(cacheKey, func(parseCtx context.Context) (workspaceSummary, error) {
		return loadWorkspaceSummary(parseCtx, parseSession.Subject)
	}, fetch.CacheOptions{
		StaleAfter:   20 * time.Second,
		MaxAge:       90 * time.Second,
		DisposeAfter: 3 * time.Minute,
	})
	cacheState := parseResource.Get()

	parseResolveSignedIn := ui.UseEvent(func() {
		setLiveSession(parseSessionAtom, demoSession{Status: sessionAuthenticated, Subject: "atlas-admin", CanViewBilling: false})
		parseRevalidator.Revalidate()
	})
	parseResolveGuest := ui.UseEvent(func() {
		setLiveSession(parseSessionAtom, demoSession{Status: sessionUnauthenticated})
		parseValues := url.Values{}
		parseValues.Set(router.ReturnToParam, router.PreserveReturnTo("/workspace", parseSearch.Values()))
		parseNav.Replace("/login?" + parseValues.Encode())
	})
	parseShowOverview := ui.UseEvent(func() { parseSearch.Replace("tab", "overview") })
	parseShowBilling := ui.UseEvent(func() { parseSearch.Replace("tab", "billing") })
	parseGrantBilling := ui.UseEvent(func() {
		setLiveSession(parseSessionAtom, demoSession{Status: sessionAuthenticated, Subject: parseSession.Subject, CanViewBilling: true})
	})
	parseRefreshBoth := ui.UseEvent(func() {
		if cacheKey != "" {
			fetch.InvalidateResource(cacheKey)
		}
		parseRevalidator.Revalidate()
	})
	parseDisposeAndRefresh := ui.UseEvent(func() {
		parseResource.Dispose()
		parseRevalidator.Revalidate()
	})
	parseSignOut := ui.UseEvent(func() {
		setLiveSession(parseSessionAtom, demoSession{Status: sessionUnauthenticated})
		parseValues2 := url.Values{}
		parseValues2.Set(router.ReturnToParam, router.PreserveReturnTo("/workspace", parseSearch.Values()))
		parseNav.Replace("/login?" + parseValues2.Encode())
	})

	if parseSession.Status == sessionUnknown {
		return shared.ExamplePage(
			"Protected Workspace",
			"manual authorizing UI",
			"The router currently ships synchronous guards only, so pending-auth UI is still application-owned. This page intentionally renders an authorizing shell while the auth state is unresolved, then reruns the route loader once the session becomes concrete.",
			shared.ExamplePanel("Authorizing",
				html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Session state is still unknown, so the route holds on to explicit authorizing UI instead of guessing. Pick an outcome below to complete the flow.")),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					shared.ExampleButton("Restore signed-in session", parseResolveSignedIn),
					shared.ExampleButton("Resolve as guest", parseResolveGuest),
				),
			),
		)
	}

	if parseTab == "billing" && !parseSession.CanViewBilling {
		return shared.ExamplePage(
			"Protected Workspace",
			"manual unauthorized fallback",
			"This route stays mounted, but the billing section renders explicit unauthorized content because the current user lacks the claim required for that subsection.",
			shared.ExamplePanel("Unauthorized billing section",
				html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Router-level Unauthorized content is still future work. Today the app renders its own fallback for forbidden subsections, keeps the rest of the shell intact, and lets the user request a different claim set or navigate elsewhere.")),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					shared.ExampleButton("Back to overview", parseShowOverview),
					shared.ExampleButton("Grant billing claim", parseGrantBilling),
					shared.ExampleButton("Sign out", parseSignOut),
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
				shared.ExampleButton("Overview tab", parseShowOverview),
				shared.ExampleButton("Billing tab", parseShowBilling),
				shared.ExampleButton("Revalidate route + cache", parseRefreshBoth),
				shared.ExampleButton("Dispose cache entry", parseDisposeAndRefresh),
				shared.ExampleButton("Sign out", parseSignOut),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("Active tab", parseTab),
				shared.ExampleStat("Subject", parseSession.Subject),
				shared.ExampleStat("Loader revision", fmt.Sprintf("%d", parseLoaderSummary.Revision)),
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

func workspacePage(parseProps router.Attrs) *router.Element {
	return ui.CreateElement(func() ui.Node {
		return workspacePageView(parseProps)
	})
}

func routeLoading(parseProps router.Attrs) *router.Element {
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

func routeError(parseProps router.Attrs) *router.Element {
	parseMessage, _ := parseProps["error"].(string)
	return ui.CreateElement(func() ui.Node {
		return shared.ExamplePage(
			"Protected Workspace",
			"route error state",
			"Route loader failures stay explicit and user-visible.",
			shared.ExamplePanel("Error",
				html.P(html.Props{Class: "mt-3 leading-7 text-rose-200"}, html.Text(emptyFallback(parseMessage, "Route load failed"))),
			),
		)
	})
}

func protectedGuard(parseCtx router.RouteContext) router.GuardResult {
	if liveSession.Status != sessionUnauthenticated {
		return router.AllowNavigation()
	}
	parseValues := url.Values{}
	parseValues.Set(router.ReturnToParam, router.PreserveReturnTo(parseCtx.Path, parseCtx.Query.Values()))
	return router.RedirectNavigation("/login?" + parseValues.Encode())
}

func workspaceLoader(parseCtx context.Context, parseRouteCtx router.RouteContext) (router.Attrs, error) {
	if liveSession.Status != sessionAuthenticated {
		return router.Attrs{}, nil
	}

	cacheKey := workspaceSummaryKey(liveSession.Subject)
	parseSummary, parseErr := fetch.LoadCached(parseCtx, cacheKey, func(parseLoadCtx context.Context) (workspaceSummary, error) {
		return loadWorkspaceSummary(parseLoadCtx, liveSession.Subject)
	}, fetch.CacheOptions{
		StaleAfter:   20 * time.Second,
		MaxAge:       90 * time.Second,
		DisposeAfter: 3 * time.Minute,
	})
	if parseErr != nil {
		return nil, parseErr
	}
	return router.Attrs{
		"cacheKey": cacheKey,
		"summary":  parseSummary,
	}, nil
}

func emptyFallback(parseValue, parseFallback string) string {
	parseTrimmed := strings.TrimSpace(parseValue)
	if parseTrimmed == "" {
		return parseFallback
	}
	return parseTrimmed
}

func main() {
	utils.DisableAllDebug()
	hotreload.Enable()
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	parseR.Register("/", homePage)
	parseR.Register("/login", loginPage)
	parseR.Register("/workspace", workspacePage, router.Options{
		BeforeEnter: protectedGuard,
		Loader:      workspaceLoader,
		Loading:     routeLoading,
		Error:       routeError,
	})
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
