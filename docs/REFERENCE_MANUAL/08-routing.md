# 08 Routing

Use this chapter when you are building route trees, reading params or query state, wiring loaders, or enforcing route-level navigation rules.

It is the right chapter for:

- choosing between hash routing and history routing
- using params, query helpers, route contracts, and imperative navigation cleanly
- wiring route loaders, revalidation, redirects, metadata, layouts, and guards
- understanding how routing stays inside one client shell as the app grows

Use another chapter instead when:

- you need raw `ui` component mechanics first: go to [04 UI Rendering And Hooks](04-ui-rendering-and-hooks.md)
- you need async data ownership first: go to [07 Data Loading And Mutations](07-data-loading-and-mutations.md)
- you need SSR hydration attach and bootstrap details in depth: go to [09 SSR And Hydration](09-ssr-and-hydration.md)
- you need auth, form, or locale behavior inside routes rather than the router itself: go to [11 Forms Accessibility And I18n](11-forms-accessibility-and-i18n.md)

## Overview

The router model is intentionally explicit.

Start with these core pieces:

- `router.NewHashRouter(...)` or `router.NewHistoryRouter(...)`
- `Register(...)` to declare routes
- `Mount(...)` to attach the router
- `router.UseNavigate()` for imperative navigation from handlers
- `router.UseParams()`, `router.UseQuery()`, and `router.UseSearchParams()` inside matched route components

Then add advanced route lifecycle only when the screen needs it:

- `Loader` for route-owned entry data
- `UseRevalidator()` for rerunning the current route loader
- `BeforeEnter` or `BeforeLeave` plus async variants for route-level navigation policy
- `Layout: true` plus `router.GetOutlet()` for nested route shells
- route metadata through `Title`, `Description`, and `CanonicalURL`

One practical split matters early:

- use hash routing when you want browser-only deploy simplicity
- use history routing when you want clean URLs and can provide server rewrites for deep links

## Stability Note

Core routing is `Stable`:

- `NewHashRouter`
- `NewHistoryRouter`
- `Register`
- `Mount`
- `UseNavigate`
- `UseParams`
- `UseQuery`
- `UseSearchParams`
- redirects through `router.Options{Redirect: ...}`
- nested layouts through `Layout: true` and `router.GetOutlet()`
- route contracts through `DefineRoute`, `MustDefineRoute`, `Path`, and `Href`

Important advanced routing lifecycle:

- route loaders, `UseRouteData`, and `UseRevalidator()` are part of the advanced router data surface and should be treated carefully
- guards, return-to helpers, route-managed metadata, and hydration-heavy router attach flows are also part of that advanced lifecycle boundary
- these surfaces are shipped and documented, but the repo policy still treats this slice as more changeable than the core route registration and navigation surface

## Minimal Example

Start with one router, a couple of routes, and imperative navigation from a normal event handler.

```go
package main

import (
	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type routePageProps struct {
	Title   string
	Summary string
}

// renderRoutePage renders one simple matched route and navigates through the router handle.
func renderRoutePage(getProps routePageProps) ui.Node {
	getNav := router.UseNavigate()

	return h.Main(
		h.Class("mx-auto max-w-xl space-y-4 p-6"),
		h.H1(getProps.Title),
		h.P(getProps.Summary),
		h.Div(
			h.Class("flex gap-3"),
			h.Button(h.Type("button"), h.OnClick(ui.UseEvent(func() { getNav.Navigate("/") })), "Home"),
			h.Button(h.Type("button"), h.OnClick(ui.UseEvent(func() { getNav.Navigate("/pricing") })), "Pricing"),
		),
	)
}

// buildRoutePage adapts one plain route description into a router component.
func buildRoutePage(getTitle string, getSummary string) *router.Element {
	return ui.CreateElement(renderRoutePage, routePageProps{Title: getTitle, Summary: getSummary})
}

// main registers a minimal hash-router tree and mounts it into the page.
func main() {
	getRouter := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	getRouter.Register("/", func(router.Attrs) *router.Element {
		return buildRoutePage("Router home", "Hash routing works well when you do not want server rewrites.")
	})
	getRouter.Register("/pricing", func(router.Attrs) *router.Element {
		return buildRoutePage("Pricing route", "This is another matched page in the same router tree.")
	})
	getRouter.Register("*", func(router.Attrs) *router.Element {
		return buildRoutePage("Not found", "Catch-all routes still matter inside a client router.")
	})
	getRouter.Mount("#app")
	utils.WaitForever()
}
```

Why this is the right first step:

- route registration stays explicit
- `UseNavigate()` keeps handler-driven navigation inside normal UI code
- the hash router avoids server rewrite requirements while the app is still small

## Production-Shaped Example

Once a route owns entry data and URL-driven state, use params, query helpers, a route loader, and manual revalidation together.

```go
package reports

import (
	"context"
	"fmt"
	"time"

	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

// RenderReportsPage reads params, query state, and loader data from the matched route.
func RenderReportsPage() ui.Node {
	getParams := router.UseParams()
	getQuery := router.UseQuery()
	getSearch := router.UseSearchParams()
	getRevalidator := router.UseRevalidator()
	getRouteData := router.UseRouteData()

	getReportID := getParams.Get("id")
	getTab := getQuery.Get("tab")
	if getTab == "" {
		getTab = "overview"
	}

	getRevision, _ := getRouteData["revision"].(string)
	getLoadedAt, _ := getRouteData["loadedAt"].(string)

	handleUserOverview := ui.UseEvent(func() {
		getSearch.Replace("tab", "overview")
	})
	handleUserActivity := ui.UseEvent(func() {
		getSearch.Replace("tab", "activity")
	})
	handleUserReload := ui.UseEvent(func() {
		getRevalidator.Revalidate()
	})

	return h.Section(
		h.Class("space-y-4 rounded-2xl border border-slate-200 bg-white p-5"),
		h.H2("Loader-backed report route"),
		h.Div(
			h.Class("flex flex-wrap gap-3"),
			h.Button(h.Type("button"), h.OnClick(handleUserOverview), "Overview tab"),
			h.Button(h.Type("button"), h.OnClick(handleUserActivity), "Activity tab"),
			h.Button(h.Type("button"), h.OnClick(handleUserReload), "Revalidate"),
		),
		h.P(h.Textf("report=%s tab=%s revision=%s loadedAt=%s loading=%t", getReportID, getTab, getRevision, getLoadedAt, getRevalidator.Loading())),
	)
}

// BuildReportsLoader returns route-owned attrs before the final page renders.
func BuildReportsLoader(getCtx context.Context, getRouteCtx router.RouteContext) (router.Attrs, error) {
	select {
	case <-getCtx.Done():
		return nil, getCtx.Err()
	case <-time.After(250 * time.Millisecond):
	}

	return router.Attrs{
		"revision": fmt.Sprintf("r-%s", getRouteCtx.Params.Get("id")),
		"loadedAt": time.Now().Format("15:04:05.000"),
	}, nil
}
```

```go
package main

import (
	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"

	"my-app/internal/reports"
)

// renderRouteLoadingPage renders a focused route-level loading fallback while the loader is in flight.
func renderRouteLoadingPage(router.Attrs) *router.Element {
	return h.P("Loading route data...")
}

// renderRouteErrorPage renders a focused route-level error fallback.
func renderRouteErrorPage(getProps router.Attrs) *router.Element {
	getMessage, _ := getProps["error"].(string)
	return h.P(h.Textf("Route load failed: %s", getMessage))
}

// main mounts a history router whose report route owns its entry data and metadata.
func main() {
	getRouter := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: "/reports/7"})
	getRouter.Register("/reports/:id", func(router.Attrs) *router.Element {
		return ui.CreateElement(reports.RenderReportsPage, nil)
	}, router.Options{
		Loader:       reports.BuildReportsLoader,
		Loading:      renderRouteLoadingPage,
		Error:        renderRouteErrorPage,
		Title:        "Reports",
		Description:  "Operations report detail",
		CanonicalURL: "https://example.com/reports",
	})
	getRouter.Mount("#app")
	utils.WaitForever()
}
```

Why this is the right next layer:

- params and query stay tied to the URL instead of hidden local state
- the loader owns route entry data explicitly
- revalidation reruns the same route loader without inventing a second refresh path

## Scale-Up Example

In a larger app, scale routing by using route contracts, nested shells, and one route-level auth boundary instead of multiple disconnected app roots.

```go
package routes

import (
	"net/url"

	"github.com/monstercameron/GoWebComponents/router"
)

var (
	MarketingRoute = router.MustDefineRoute("/")
	SignInRoute    = router.MustDefineRoute("/signin")
	AppRoute       = router.MustDefineRoute("/app")
	ProjectRoute   = router.MustDefineRoute("/app/projects/:projectID")
)

// BuildAppGuard preserves one bounded return target before redirecting unauthenticated users.
func BuildAppGuard(getIsAuthenticated func() bool) router.GuardFunc {
	return func(getRouteCtx router.RouteContext) router.GuardResult {
		if getIsAuthenticated() {
			return router.AllowNavigation()
		}
		getValues := url.Values{}
		getValues.Set(router.ReturnToParam, router.PreserveReturnTo(getRouteCtx.Path, getRouteCtx.Query.Values()))
		return router.RedirectNavigation(SignInRoute.MustHref(nil, getValues))
	}
}
```

```go
package shell

import (
	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

// RenderWorkspaceLayout keeps app-level chrome mounted while nested child routes swap through the outlet.
func RenderWorkspaceLayout() ui.Node {
	return h.Main(
		h.Class("min-h-screen bg-slate-50"),
		h.Header(h.Class("border-b bg-white p-4"), h.H1("Workspace shell")),
		h.Section(h.Class("p-6"), router.GetOutlet()),
	)
}

// RenderProjectPage reads the typed route parameter inside the nested workspace shell.
func RenderProjectPage() ui.Node {
	getParams := router.UseParams()
	return h.Article(
		h.Class("rounded-2xl border border-slate-200 bg-white p-5"),
		h.H2("Project detail"),
		h.P(h.Textf("projectID=%s", getParams.Get("projectID"))),
	)
}
```

```go
package main

import (
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"

	"my-app/internal/routes"
	"my-app/internal/shell"
)

// main mounts one product-shaped history router with contracts, a nested shell, and one guarded branch.
func main() {
	getRouter := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: routes.MarketingRoute.MustPath(nil)})
	getRouter.Register(routes.MarketingRoute.MustPath(nil), func(router.Attrs) *router.Element { return ui.CreateElement(renderMarketingPage, nil) })
	getRouter.Register(routes.SignInRoute.MustPath(nil), func(router.Attrs) *router.Element { return ui.CreateElement(renderSignInPage, nil) })
	getRouter.Register(routes.AppRoute.MustPath(nil), func(router.Attrs) *router.Element {
		return ui.CreateElement(shell.RenderWorkspaceLayout, nil)
	}, router.Options{
		Layout:      true,
		BeforeEnter: routes.BuildAppGuard(getIsAuthenticated),
	})
	getRouter.Register(routes.ProjectRoute.Pattern(), func(router.Attrs) *router.Element {
		return ui.CreateElement(shell.RenderProjectPage, nil)
	}, router.Options{Title: "Project detail"})
	getRouter.Mount("#app")
	utils.WaitForever()
}
```

Why this scales:

- contracts remove string drift between registration, redirects, and button navigation
- one shell stays mounted while nested routes swap through `GetOutlet()`
- auth stays at the route-branch boundary instead of splitting the product into multiple HTML shells

## Params, Query, And Contracts

Use `router.UseParams()` when:

- the route path captures identifiers such as `:id` or `:slug`
- typed helpers like `Int(...)` or `Bool(...)` make the route easier to read

Use `router.UseQuery()` when:

- the component only needs to read the current query string

Use `router.UseSearchParams()` when:

- the component should write query state through navigation helpers such as `Set(...)`, `Replace(...)`, `Delete(...)`, or `Navigate(...)`

Use route contracts when:

- the app is large enough that repeated string paths are starting to drift
- redirects, links, tests, and route registration should all reuse one validated definition

Practical rule:

- params identify the resource
- query state identifies the current view of that resource
- contracts keep the route definition itself from drifting across the codebase

## Metadata And Hydration Attach

Route-managed metadata in the router is intentionally narrow:

- `Title`
- `Description`
- `CanonicalURL`

Use it for route-local document title, description, and canonical URL updates. Broader head composition stays outside core routing.

For SSR-resumed routed apps, the router also provides `HydrateMount(...)` and `HydrateMountElement(...)` so it can attach listeners after the current route has already been hydrated.

Use that pattern when:

- the initial route tree was already hydrated with `ui.Hydrate(...)`
- you want the router to start listening without immediately replacing the current route

## Auth Guards And Head Ownership

Keep auth and metadata boundaries narrow:

- route guards decide whether a route may continue, but the server still owns session truth and authorization policy
- return-to targets should stay same-origin, normalized, and route-contract-aware
- authorizing, guard-pending, and unauthorized states should be deliberate UX states, not accidental blank screens
- router metadata is intentionally limited to title, description, and canonical URL
- broader head composition such as resource hints, Open Graph tags, or app-wide document policy belongs in a dedicated head layer instead of the router itself

## API Family Reference

Use this table before you add more route machinery.

| API family | Representative APIs | Stability | Use it when | Prefer something else when |
| --- | --- | --- | --- | --- |
| Router creation and mount | `NewHashRouter`, `NewHistoryRouter`, `Register`, `Mount`, `Current` | `Stable` | you need the base route tree and mount lifecycle | never; these are the entrypoints |
| Imperative navigation | `UseNavigate`, `Navigator.Navigate`, `Navigator.Replace`, top-level `Navigate`, `NavigateReplace` | `Stable` | event handlers should push or replace routes programmatically | a normal link is enough |
| Params and query reads | `UseParams`, `UseQuery`, `UseSearchParams` | `Stable` | the route component should read or write URL-derived state | the value is purely local UI state |
| Route contracts | `DefineRoute`, `MustDefineRoute`, `Path`, `Href`, `PathFor`, `HrefFor` | `Stable` | larger apps need validated reverse routing | the app is still small and string literals remain obvious |
| Redirects | `router.Options{Redirect: ...}`, `RedirectNavigation` | core redirects are `Stable` | a matched route should hand off to another route cleanly | the page should stay mounted and render a manual fallback |
| Nested layouts | `router.Options{Layout: true}`, `GetOutlet` | `Stable` | one shell should stay mounted while child routes swap | there is no shared shell between the routes |
| Loader-backed routes | `Loader`, `UseRouteData`, `UseRevalidator` | advanced router lifecycle | the route owns first-load data and refresh should rerun the same loader | the data is local to one component |
| Guards and return-to recovery | `BeforeEnter`, `BeforeLeave`, async guard variants, `PreserveReturnTo`, `ReadReturnTo` | advanced router lifecycle | navigation policy belongs at the route boundary | the restriction is purely in-page UI, not route entry |
| Route metadata | `Title`, `Description`, `CanonicalURL` | advanced router lifecycle | the route should own narrow SEO-safe metadata | broader head composition needs a dedicated head system |
| Hydration-aware attach | `HydrateMount`, `HydrateMountElement` | advanced hydration-aware routing surface | the route tree was already hydrated and the router should attach without replacing the initial route | the app is client-only and can use normal `Mount` |

## Design Notes And Boundaries

Keep these routing rules in mind:

- one router tree should own the product shell whenever possible
- history routing needs server rewrites; hash routing does not
- route loaders own route entry data, not every async read in the app
- guarded routes improve client navigation flow, but the server still owns authorization
- route metadata in core is intentionally narrow and should not become a second general head-management system
- contracts are a runtime-first answer to route drift, not a code-generation mandate

## Common Failure Modes

- using history routing without server rewrites for deep links
- storing route state in local hooks instead of the URL even when it should be shareable and reload-safe
- inventing a second loader path instead of reusing `UseRouteData()` and `UseRevalidator()`
- scattering string route literals through registration, links, redirects, and tests once the app is large
- putting auth logic only in the client guard and forgetting server enforcement
- using nested layouts without defining clear shell ownership boundaries
- forcing route-level loaders to own deeply local panel data that should stay component-owned

## Validation

Use the smallest examples that prove the routing family you are adopting.

Core router setup, navigation, params, and query:

```powershell
go run ./tools/gwc dev -app .\examples\56-browser-router\main.go
go run ./tools/gwc dev -app .\examples\57-use-navigate\main.go
go run ./tools/gwc dev -app .\examples\58-route-params\main.go
go run ./tools/gwc dev -app .\examples\59-route-query\main.go
```

Loader-backed routes, revalidation, metadata, and redirects:

```powershell
go run ./tools/gwc dev -app .\examples\60-route-loaders\main.go
go run ./tools/gwc dev -app .\examples\61-use-revalidator\main.go
go run ./tools/gwc dev -app .\examples\62-router-redirects\main.go
go run ./tools/gwc dev -app .\examples\63-router-metadata\main.go
```

Nested shells, guards, and hydration-aware attach:

```powershell
go run ./tools/gwc dev -app .\examples\64-nested-layout-routes\main.go
go run ./tools/gwc dev -app .\examples\65-router-guards\main.go
go run ./tools/gwc dev -app .\examples\72-router-hydrate-mount\main.go
go run ./tools/gwc dev -app .\examples\106-single-shell-auth\main.go
```

## Topic Pagination
Topic 8 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [07 Data Loading And Mutations](07-data-loading-and-mutations.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [09 SSR And Hydration](09-ssr-and-hydration.md)
