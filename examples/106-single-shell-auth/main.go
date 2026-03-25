//go:build js && wasm
// +build js,wasm

package main

import (
	"net/url"
	"strings"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const shellSessionAtomID = "catalog-single-shell-auth-session"

const (
	sessionGuest         = "guest"
	sessionAuthenticated = "authenticated"
)

type shellSession struct {
	Status      string
	Subject     string
	Workspace   string
	SessionMode string
}

var liveShellSession = shellSession{
	Status:      sessionGuest,
	Subject:     "guest",
	Workspace:   "public-site",
	SessionMode: "marketing",
}

var (
	marketingRoute = router.MustDefineRoute("/")
	pricingRoute   = router.MustDefineRoute("/pricing")
	signInRoute    = router.MustDefineRoute("/signin")
	workspaceRoute = router.MustDefineRoute("/workspace")
)

func setShellSession(atom state.Atom[shellSession], next shellSession) {
	liveShellSession = next
	atom.Set(next)
}

func sessionSubject(session shellSession) string {
	subject := strings.TrimSpace(session.Subject)
	if subject == "" {
		return "guest"
	}
	return subject
}

func shellNavigationPanel() ui.Node {
	nav := router.UseNavigate()
	sessionAtom := state.UseAtom(shellSessionAtomID, liveShellSession)
	session := sessionAtom.Get()
	inspection := router.InspectCurrentRoute()

	return shared.ExamplePanel("One running shell",
		html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("These routes all live inside one mounted WASM client. Marketing pages, sign-in, and the protected workspace change through router navigation and shared auth state instead of bouncing through server-rendered detour pages.")),
		html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
			shared.ExampleButton("Marketing home", ui.UseEvent(func() {
				nav.Navigate(marketingRoute.MustPath(nil))
			})),
			shared.ExampleButton("Pricing", ui.UseEvent(func() {
				nav.Navigate(pricingRoute.MustPath(nil))
			})),
			shared.ExampleButton("Sign in", ui.UseEvent(func() {
				nav.Navigate(signInRoute.MustPath(nil))
			})),
			shared.ExampleButton("Workspace", ui.UseEvent(func() {
				nav.Navigate(workspaceRoute.MustPath(nil))
			})),
		),
		html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
			shared.ExampleStat("Current path", inspection.Path),
			shared.ExampleStat("Session", session.Status),
			shared.ExampleStat("Subject", sessionSubject(session)),
			shared.ExampleStat("Shell mode", session.SessionMode),
		),
	)
}

func singleShellPage(title, feature, summary string, content ...ui.Node) ui.Node {
	panels := make([]ui.Node, 0, len(content)+1)
	panels = append(panels, shellNavigationPanel())
	panels = append(panels, content...)
	return shared.ExamplePage(title, feature, summary, panels...)
}

func marketingPageView() ui.Node {
	nav := router.UseNavigate()
	sessionAtom := state.UseAtom(shellSessionAtomID, liveShellSession)
	session := sessionAtom.Get()

	return singleShellPage(
		"Single-shell auth",
		"public marketing, sign-in, and a guarded workspace in one WASM client",
		"This example demonstrates the runtime-owned product-shell pattern: public routes stay in the same client as sign-in and the protected workspace, and the router moves between them by updating session state plus route location instead of reloading the document.",
		shared.ExamplePanel("Public marketing route",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("This is the anonymous marketing surface. Jump directly to the protected workspace to watch the router preserve a bounded return target and redirect into the sign-in route without leaving the running shell.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Open pricing", ui.UseEvent(func() {
					nav.Navigate(pricingRoute.MustPath(nil))
				})),
				shared.ExampleButton("Go to sign in", ui.UseEvent(func() {
					nav.Navigate(signInRoute.MustPath(nil))
				})),
				shared.ExampleButton("Try protected workspace", ui.UseEvent(func() {
					nav.Navigate(workspaceRoute.MustPath(nil))
				})),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Current session", session.Status),
				shared.ExampleStat("Workspace name", session.Workspace),
				shared.ExampleStat("Guard target", workspaceRoute.MustPath(nil)),
			),
		),
		shared.ExamplePanel("What the example proves",
			html.Ul(html.Props{Class: "mt-3 grid gap-3 text-sm leading-7 text-slate-300"},
				html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("Marketing pages remain first-class routes inside the same router that later owns sign-in and workspace navigation.")),
				html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("Protected entry uses a normal route guard plus router.PreserveReturnTo(...) rather than a hard-coded auth redirect path.")),
				html.Li(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 px-4 py-3"}, html.Text("Sign-in updates a shared auth atom and then replaces navigation directly into the protected route, so there is no full page reload or second app shell.")),
			),
		),
	)
}

func marketingPage(router.Attrs) *router.Element {
	return ui.CreateElement(marketingPageView)
}

func pricingPageView() ui.Node {
	nav := router.UseNavigate()
	return singleShellPage(
		"Single-shell pricing",
		"marketing routes can coexist with auth routes and workspace routes",
		"The pricing page is still just another client-owned route in the same running shell. Public marketing links can stay live even while the same router later owns the sign-in and workspace flow.",
		shared.ExamplePanel("Pricing route",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("A single-shell product does not need separate HTML entrypoints for every auth state. Pricing stays public, and sign-in can still continue into the same protected route contract after the user chooses to proceed.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Return to marketing", ui.UseEvent(func() {
					nav.Navigate(marketingRoute.MustPath(nil))
				})),
				shared.ExampleButton("Continue to sign in", ui.UseEvent(func() {
					nav.Navigate(signInRoute.MustPath(nil))
				})),
				shared.ExampleButton("Request workspace", ui.UseEvent(func() {
					nav.Navigate(workspaceRoute.MustPath(nil))
				})),
			),
		),
	)
}

func pricingPage(router.Attrs) *router.Element {
	return ui.CreateElement(pricingPageView)
}

func signInPageView() ui.Node {
	nav := router.UseNavigate()
	query := router.UseQuery()
	sessionAtom := state.UseAtom(shellSessionAtomID, liveShellSession)
	returnTo := router.ReadReturnTo(query.Values(), workspaceRoute.MustPath(nil))

	signInOperator := ui.UseEvent(func() {
		setShellSession(sessionAtom, shellSession{
			Status:      sessionAuthenticated,
			Subject:     "atlas-operator",
			Workspace:   "warehouse-ops",
			SessionMode: "workspace",
		})
		nav.Replace(returnTo)
	})

	signInFinance := ui.UseEvent(func() {
		setShellSession(sessionAtom, shellSession{
			Status:      sessionAuthenticated,
			Subject:     "atlas-finance",
			Workspace:   "finance-review",
			SessionMode: "workspace",
		})
		nav.Replace(returnTo)
	})

	return singleShellPage(
		"Single-shell sign-in",
		"route recovery with one client-owned shell",
		"The sign-in route reads the bounded return target, updates the shared auth session, and replaces navigation straight into the protected workspace. The shell never hands off to a second HTML app or full-page redirect.",
		shared.ExamplePanel("Return-to recovery",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Current bounded return target: "+returnTo)),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Sign in as operator", signInOperator),
				shared.ExampleButton("Sign in as finance reviewer", signInFinance),
				shared.ExampleButton("Back to marketing", ui.UseEvent(func() {
					nav.Replace(marketingRoute.MustPath(nil))
				})),
			),
			shared.ExampleCode(
				`values.Set(router.ReturnToParam, router.PreserveReturnTo(ctx.Path, ctx.Query.Values()))`,
				`returnTo := router.ReadReturnTo(query.Values(), "/workspace")`,
				`nav.Replace(returnTo)`,
			),
		),
	)
}

func signInPage(router.Attrs) *router.Element {
	return ui.CreateElement(signInPageView)
}

func workspacePageView() ui.Node {
	nav := router.UseNavigate()
	sessionAtom := state.UseAtom(shellSessionAtomID, liveShellSession)
	session := sessionAtom.Get()

	signOut := ui.UseEvent(func() {
		setShellSession(sessionAtom, shellSession{
			Status:      sessionGuest,
			Subject:     "guest",
			Workspace:   "public-site",
			SessionMode: "marketing",
		})
		nav.Replace(marketingRoute.MustPath(nil))
	})

	return singleShellPage(
		"Protected workspace",
		"guarded route target in the same router tree",
		"The workspace route is protected by a route guard, but it still lives in the same client-owned router tree as the marketing and sign-in pages. After sign-in, the shell simply replaces the path and continues rendering under the authenticated state.",
		shared.ExamplePanel("Authenticated workspace route",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("No second shell booted to show this screen. The same client instance updated the auth atom, replaced the route, and kept navigation inside the running session-aware app surface.")),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Signed in as", sessionSubject(session)),
				shared.ExampleStat("Workspace", session.Workspace),
				shared.ExampleStat("Session mode", session.SessionMode),
			),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Open pricing without leaving shell", ui.UseEvent(func() {
					nav.Navigate(pricingRoute.MustPath(nil))
				})),
				shared.ExampleButton("Sign out to marketing", signOut),
			),
		),
		shared.ExamplePanel("Route contract usage",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("The example also uses the new router.RouteContract helpers so route registration, redirect targets, and button navigation all reuse one route definition instead of repeating string literals throughout the flow.")),
			shared.ExampleCode(
				`var workspaceRoute = router.MustDefineRoute("/workspace")`,
				`nav.Navigate(workspaceRoute.MustPath(nil))`,
				`return router.RedirectNavigation(signInRoute.MustHref(nil, values))`,
			),
		),
	)
}

func workspacePage(router.Attrs) *router.Element {
	return ui.CreateElement(workspacePageView)
}

func notFoundPageView() ui.Node {
	nav := router.UseNavigate()
	return shared.ExamplePage(
		"Single-shell auth",
		"route fallback",
		"This route is not part of the example flow.",
		shared.ExamplePanel("Unknown route",
			html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Use the button below to jump back into the marketing route and continue the single-shell auth flow.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Return to marketing", ui.UseEvent(func() {
					nav.Replace(marketingRoute.MustPath(nil))
				})),
			),
		),
	)
}

func notFoundPage(router.Attrs) *router.Element {
	return ui.CreateElement(notFoundPageView)
}

func protectedWorkspaceGuard(ctx router.RouteContext) router.GuardResult {
	if liveShellSession.Status == sessionAuthenticated {
		return router.AllowNavigation()
	}

	values := url.Values{}
	values.Set(router.ReturnToParam, router.PreserveReturnTo(ctx.Path, ctx.Query.Values()))
	return router.RedirectNavigation(signInRoute.MustHref(nil, values))
}

func main() {
	utils.DisableAllDebug()

	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: marketingRoute.Pattern()})
	r.Register(marketingRoute.Pattern(), marketingPage)
	r.Register(pricingRoute.Pattern(), pricingPage)
	r.Register(signInRoute.Pattern(), signInPage)
	r.Register(workspaceRoute.Pattern(), workspacePage, router.Options{
		BeforeEnter: protectedWorkspaceGuard,
	})
	r.Register("*", notFoundPage)
	r.Mount("#app")
	select {}
}
