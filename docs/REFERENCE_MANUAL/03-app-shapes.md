# 03 App Shapes

Use this chapter when you need to choose the right application shape before you start spreading examples or APIs across a real codebase.

It is the right chapter for:

- deciding between client-only, routed, SSR, forms-heavy, or static/PWA-first apps
- matching the current public package surface to the product shape
- choosing the first maintained example that fits the work
- understanding when to keep the app small and when to widen the surface deliberately

Use another chapter instead when:

- you already know the app shape and only need launcher commands: go to [02 GWC Workflows](02-gwc-workflows.md)
- you already know the next package to adopt and want API-level guidance: go to the matching package chapter
- you are organizing a larger product app or team boundary: go to [14 Scaling Large Codebases](14-scaling-large-codebases.md)

## Overview

GoWebComponents is intentionally incremental.

The current supported shapes are:

- client-only app
- routed SPA
- SSR and hydration app
- forms-heavy business app
- static or PWA-oriented app

The right question is not "which feature can I enable first." The right question is "who owns the app lifecycle and where does correctness live."

## Stability Note

These app shapes are built from a mix of `Stable` entrypoints and feature areas with deeper operational rules.

Practical defaults:

- client-only app: safest first shape
- routed SPA: safe default when the app genuinely needs multiple screens
- SSR and hydration: stable core entrypoints, but operationally stricter
- forms-heavy flows: stable building blocks, but server-ownership decisions matter more than helper selection
- static/PWA shape: companion-package-oriented and integration-heavy by design

## Shape Chooser

Use this table first.

| Shape | Best fit | Start with | Add next | First examples |
| --- | --- | --- | --- | --- |
| Client-only | one screen or a small browser-only workflow | `ui`, `html`, `gwc dev` | `router` only if the app grows past one screen | `counter`, `ui-render`, `use-state`, `use-effect` |
| Routed SPA | multiple screens, params, loaders, guards, route metadata | `ui`, `html`, `router` | `state` and `fetch` when shared state or typed route data appear | `55-hash-router`, `56-browser-router`, `60-route-loaders`, `64-nested-layout-routes` |
| SSR and hydration | request-time HTML, route-aware first paint, hydration reuse | `ui.RenderToString`, `ui.Hydrate`, `router.HydrateMount` | typed bootstrap ownership, same-origin APIs, observability | `70-render-to-string`, `71-hydrate`, `72-router-hydrate-mount`, `18-ssr-server-routing` |
| Forms-heavy | authoritative mutations, validation round-trips, pending UX, redirects, uploads | `ui.UseForm` or server-owned HTML form flow | route revalidation, cache invalidation, auth/session rules | `51-use-form`, `79-form-accessibility`, `87-ssr-secure-forms` |
| Static/PWA-oriented | static-hosted delivery, installability, cache planning, offline-capable workflows | client-only or prerendered shell plus `pwa` helpers | service-worker ownership, cache plans, mutation replay | `97-pwa-installability`, `97-pwa-offline-cache`, `97-pwa-multi-client`, `102-static-export-site` |

## Minimal Example: Client-Only App

Choose this shape when one browser-mounted shell and local state are enough.

```go
package main

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

// renderClientOnlyApp renders the smallest useful product-shaped browser app.
func renderClientOnlyApp() ui.Node {
	storeCount := ui.UseState(0)
	handleUserIncrement := ui.UseEvent(func() {
		storeCount.Update(func(getPreviousCount int) int {
			return getPreviousCount + 1
		})
	})

	return Main(
		Class("mx-auto max-w-xl space-y-4 p-6"),
		H1("Client-only app"),
		P(Textf("Current count: %d", storeCount.Get())),
		Button(Type("button"), OnClick(handleUserIncrement), "Increment"),
	)
}

// main mounts the client-only app into the browser DOM.
func main() {
	ui.Render(ui.CreateElement(renderClientOnlyApp, nil), "#app")
	utils.WaitForever()
}
```

Use this shape when:

- the app is still one screen
- local hooks are enough
- the first value is speed of iteration, not route structure or request-time HTML

Do not force this shape past its limits:

- once the app needs clean URLs, params, or multiple screens, move to a routed shape
- once unrelated branches need the same state, move shared ownership into `state`

## Production-Shaped Example: Routed SPA

Choose this shape when the app needs multiple screens, route params, clean navigation, or route-owned data loading.

```go
package main

import (
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type routePageProps struct {
	Title   string
	Summary string
}

// renderRoutePage renders one routed screen while navigation stays router-owned.
func renderRoutePage(getProps routePageProps) ui.Node {
	getNavigate := router.UseNavigate()
	return html.Main(
		html.Props{Class: "mx-auto max-w-2xl space-y-4 p-6"},
		html.H1(html.Text(getProps.Title)),
		html.P(html.Text(getProps.Summary)),
		html.Button(
			html.Props{Type: "button", OnClick: ui.UseEvent(func() { getNavigate.Navigate("/pricing") })},
			html.Text("Go to pricing"),
		),
	)
}

// buildRoutePage returns the router element for one page.
func buildRoutePage(getTitle, getSummary string) *router.Element {
	return ui.CreateElement(renderRoutePage, routePageProps{
		Title:   getTitle,
		Summary: getSummary,
	})
}

// main mounts the browser-router app into the current HTML shell.
func main() {
	storeRouter := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: "/"})
	storeRouter.Register("/", func(router.Attrs) *router.Element {
		return buildRoutePage("Home", "Use a routed SPA when clean navigation and multiple screens matter.")
	})
	storeRouter.Register("/pricing", func(router.Attrs) *router.Element {
		return buildRoutePage("Pricing", "History-router apps need server rewrites for deep links.")
	})
	storeRouter.Register("*", func(router.Attrs) *router.Element {
		return buildRoutePage("Not found", "Keep an explicit catch-all route inside the app.")
	})
	storeRouter.Mount("#app")
	utils.WaitForever()
}
```

Use this shape when:

- route identity matters
- users need deep links or navigable screen structure
- the app will grow into loaders, guards, metadata, or nested layouts

Keep these rules in mind:

- prefer one router tree, not multiple disconnected apps
- browser-router apps need server rewrites
- hash-router apps are still a valid static-hosting fallback

## Scale-Up Example: Single-Shell Product App

Once a product app owns marketing, auth, and workspace routes together, scale by route-family shell ownership instead of splitting the app into unrelated runtimes.

Reference route tree:

```text
/
  (root-shell)
    /
    /features
    /pricing
    /auth/signin
    /auth/signup
    /app/inbox
    /app/projects/:id
    /app/settings
```

One practical package split is:

```text
my-app/
|-- cmd/web/
|-- internal/app/
|-- internal/route/public/
|-- internal/route/auth/
|-- internal/route/workspace/
|-- internal/feature/
\-- static/
```

Why this shape scales:

- one runtime owns providers, cache, i18n, and diagnostics once
- public, auth, and workspace branches get separate route-family ownership
- the authenticated boundary stays route-level instead of becoming a second HTML shell

This is the right next step when a routed SPA becomes a real product app.

## SSR And Hydration App

Choose this shape when the first response must contain request-time HTML and the browser should resume over matching markup.

```go
package main

import (
	"net/http"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderSSRPage renders one request-time HTML tree for the current request.
func renderSSRPage() ui.Node {
	return Main(
		Class("mx-auto max-w-2xl space-y-4 p-6"),
		H1("SSR app"),
		P("Use SSR when the first response must already contain useful HTML."),
	)
}

// handlePageIndex writes one server-rendered HTML response.
func handlePageIndex(writeResponse http.ResponseWriter, getRequest *http.Request) {
	renderMarkup, renderErr := ui.RenderToString(ui.CreateElement(renderSSRPage, nil))
	if renderErr != nil {
		http.Error(writeResponse, renderErr.Error(), http.StatusInternalServerError)
		return
	}
	writeResponse.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = writeResponse.Write([]byte("<!doctype html><html><body>" + renderMarkup + "</body></html>"))
}
```

Choose SSR when:

- request-time HTML matters for the product
- route entry needs server-owned data or auth context
- hydration reuse is part of the experience, not an optional afterthought

Avoid SSR by reflex when:

- the app is small and client-rendered delivery is simpler
- the product does not benefit enough from request-time HTML to justify the added server boundary

## Forms-Heavy App

Choose this shape when the mutation path matters more than the rendering novelty.

The default business-app posture is:

- client owns local field state
- server owns validation authority and mutation success
- the same workflow must handle field errors, pending UX, redirects, and uploads coherently

Start here:

- `ui.UseForm[T]` for hydrated local form state
- `examples/public/use-form` for the local form owner
- `examples/server/server-side-rendering-secure-forms` for request-time forms, CSRF, server validation, multipart uploads, and redirects

Use this shape when:

- the product is mostly CRUD, review, approval, or internal business workflow
- server validation or normalization is part of correctness
- redirect-after-submit and field-error mapping matter

Common upgrade path:

- start local with `ui.UseForm[T]`
- keep the server authoritative
- add route revalidation or targeted cache invalidation after success

## Static Or PWA-Oriented App

Choose this shape when the delivery model matters as much as the component tree.

```go
package main

import "github.com/monstercameron/GoWebComponents/pwa"

// buildInstallabilityManifest returns the app-owned manifest for installable delivery.
func buildInstallabilityManifest() pwa.Manifest {
	return pwa.Manifest{
		Name:            "Static App",
		ShortName:       "Static",
		StartURL:        "/",
		Scope:           "/",
		Display:         pwa.ManifestDisplayStandalone,
		ThemeColor:      "#0f172a",
		BackgroundColor: "#08111d",
	}
}
```

Choose this shape when:

- the app is static-hosted or prerendered
- installability, service-worker registration, or offline cache planning matters
- you want a client-only or prerendered shell with explicit browser-platform ownership

Keep the boundary explicit:

- the framework ships helpers
- the application still owns the service-worker script, route fallback policy, and rollout decisions

## Shape Reference

Use these defaults to avoid over-engineering the first version.

| Question | Default answer |
| --- | --- |
| one screen with local state only | client-only app |
| multiple screens and clean URLs | routed SPA |
| request-time HTML or auth-aware first paint | SSR and hydration app |
| business records, validation round-trips, file uploads | forms-heavy app |
| static hosting, installability, offline shell, cache planning | static/PWA-oriented app |

## Design Notes And Boundaries

Keep these rules in mind when choosing the shape:

- start from the smallest shape that fits the product
- do not adopt SSR, routing, or PWA features just because the framework ships them
- route structure, state ownership, and mutation ownership should follow product needs, not example curiosity
- one runtime and one router tree are preferred for product apps that mix marketing and workspace flows
- static/PWA support is integration-oriented; the framework helps, but the application still owns service-worker and offline policy

## Common Failure Modes

- starting with SSR when a client-only or routed shape would be simpler and more robust
- staying client-only after the app clearly needs route identity and deep links
- splitting one product app into multiple shells instead of using one routed runtime
- treating forms as a purely client-side concern when the server actually owns correctness
- assuming static hosting and installability automatically imply good offline route policy
- copying the largest integrated example before confirming the smallest matching app shape

## Validation

Use the smallest commands that prove the chosen shape.

Client-only:

```powershell
go run ./tools/gwc dev -app .\examples\public\counter\main.go
```

Routed SPA:

```powershell
go run ./tools/gwc dev -app .\examples\public\browser-router\main.go
go run ./tools/gwc verify -app .\examples\public\browser-router\main.go -root .\examples\public\browser-router
```

SSR and hydration:

```powershell
go run ./tools/gwc verify -app .\examples\server\server-side-rendering-routing\main.go -root .\examples\server\server-side-rendering-routing
```

Forms-heavy:

```powershell
go run ./examples/server/server-side-rendering-secure-forms
```

Static/PWA-oriented:

```powershell
go run ./tools/gwc dev -app .\examples\public\progressive-web-app-installability\main.go
```

## Topic Pagination
Topic 3 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [02 GWC Workflows](02-gwc-workflows.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [04 UI Rendering And Hooks](04-ui-rendering-and-hooks.md)
