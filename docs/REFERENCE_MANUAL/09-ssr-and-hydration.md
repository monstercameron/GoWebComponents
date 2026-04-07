# 09 SSR And Hydration

Use this chapter when your app renders HTML before wasm starts and then resumes that tree in the browser.

It is the right chapter for:

- request-time HTML rendering with `ui.RenderToString(...)`
- inline or sidecar bootstrap transfer with the public `ui.SSRBootstrap` helpers
- hydration reuse with `ui.Hydrate(...)` and routed resume with `router.HydrateMount(...)`
- route-data reuse, cache seeds, form defaults, and session hints during resume

Use another chapter instead when:

- you need route registration and guard semantics first: go to [08 Routing](08-routing.md)
- you need fetch-cache ownership and offline replay first: go to [07 Data Loading And Mutations](07-data-loading-and-mutations.md)
- you need forms and server-owned mutation flow in depth: go to [11 Forms Accessibility And I18n](11-forms-accessibility-and-i18n.md)
- you need deployment, manifests, or PWA packaging: go to [13 Assets Deployment And PWA](13-assets-deployment-and-pwa.md)

## Overview

The SSR and hydration model is reuse-first and explicitly typed.

The normal path is:

1. render HTML on the server with `ui.RenderToString(...)`
2. build one `ui.SSRBootstrap` payload with only the client-visible state needed for resume
3. emit that payload inline with `ui.RenderBootstrapScript(...)` or as a sidecar reference with `ui.RenderBootstrapReferenceScript(...)`
4. read it in the browser through `ui.ReadBootstrapScript(...)` or `ui.ReadBootstrapReferenceScript(...)`
5. call `ui.Hydrate(...)` so the runtime reuses matching DOM where possible
6. if the app is routed, let `router.HydrateMount(...)` attach the router after the current route tree is already hydrated

Keep one ownership boundary clear:

- the server owns request auth, request-only data, HTML generation, redirects, and CSRF
- the bootstrap payload only transfers public resume data
- the client owns later navigation, later local state, and later data refreshes

## Stability Note

The main SSR and hydration entrypoints are `Stable`:

- `ui.RenderToString(...)`
- `ui.RenderToStringObserved(...)`
- `ui.Hydrate(...)`
- `router.HydrateMount(...)`
- `ui.RenderBootstrapScript(...)`
- `ui.ReadBootstrapScript(...)`
- `ui.RegisterRouteBootstrapData(...)`
- `ui.RegisterFormBootstrapDefaults(...)`
- `ui.RegisterCacheBootstrapSeed(...)`
- `ui.RegisterSessionBootstrapHint(...)`

Important advanced surfaces:

- alternative bootstrap transports beyond the documented JSON inline path, including sidecar JSON and CBOR, remain part of the more advanced SSR transport boundary
- hydration strictness, mismatch recovery, and route-loader reuse are shipped but still deserve extra care because they combine runtime, router, and bootstrap ownership rules
- older docs may still mention `ObserveSSR` or `AnalyzeSSRBootstrapSize`; the current exported code surface uses `ui.RegisterSSRObserver(...)` and `ui.InspectSSRBootstrapSize(...)`

## Minimal Example

Start with the smallest request-time render: build the node and turn it into HTML on the server.

```go
package main

import (
	"fmt"
	"net/http"

	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// renderLandingPage returns one server-rendered page tree.
func renderLandingPage() ui.Node {
	return h.Main(
		h.Class("mx-auto max-w-3xl space-y-4 p-6"),
		h.H1("Request-time HTML"),
		h.P("This markup exists before the browser loads wasm."),
	)
}

// handleLandingPage renders HTML per request and writes the final document response.
func handleLandingPage(getW http.ResponseWriter, getR *http.Request) {
	getMarkup, getErr := ui.RenderToString(renderLandingPage())
	if getErr != nil {
		http.Error(getW, getErr.Error(), http.StatusInternalServerError)
		return
	}

	getW.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(getW, "<!doctype html><html><body><div id=\"app\">%s</div></body></html>", getMarkup)
}
```

Why this is the right first SSR step:

- server rendering stays ordinary Go `net/http`
- `RenderToString(...)` is just HTML generation, not a full server framework
- you can verify the server HTML before adding any resume state at all

## Production-Shaped Example

For a real resumable page, render HTML, register typed bootstrap payloads, emit the bootstrap script, and hydrate against the same tree in the browser.

```go
package server

import (
	"fmt"
	"net/http"

	"github.com/monstercameron/GoWebComponents/ui"
)

type productPageData struct {
	Revision int
	Items    []string
}

// buildProductsBootstrap assembles the public resume payload for the current request.
func buildProductsBootstrap() (ui.SSRBootstrap, error) {
	getBootstrap := ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{Path: "/products"},
	}

	if getErr := ui.RegisterRouteBootstrapData(&getBootstrap, "catalog", "/products", productPageData{
		Revision: 1,
		Items:    []string{"starter", "scale", "enterprise"},
	}); getErr != nil {
		return ui.SSRBootstrap{}, getErr
	}
	if getErr := ui.RegisterSessionBootstrapHint(&getBootstrap, "viewer", map[string]string{
		"mode": "anonymous",
	}); getErr != nil {
		return ui.SSRBootstrap{}, getErr
	}

	return getBootstrap, nil
}

// handleProductsPage renders the request-time HTML and the inline bootstrap payload together.
func handleProductsPage(getW http.ResponseWriter, getR *http.Request) {
	getBootstrap, getErr := buildProductsBootstrap()
	if getErr != nil {
		http.Error(getW, getErr.Error(), http.StatusInternalServerError)
		return
	}

	getMarkup, getErr := ui.RenderToString(renderProductsPage())
	if getErr != nil {
		http.Error(getW, getErr.Error(), http.StatusInternalServerError)
		return
	}
	getScript, getErr := ui.RenderBootstrapScript(getBootstrap, "")
	if getErr != nil {
		http.Error(getW, getErr.Error(), http.StatusInternalServerError)
		return
	}

	getW.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(getW, "<!doctype html><html><body><div id=\"app\">%s</div>%s</body></html>", getMarkup, getScript)
}
```

```go
package main

import (
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

// main restores the inline bootstrap payload and hydrates the existing DOM tree.
func main() {
	getBootstrap, getErr := ui.ReadBootstrapScript("")
	if getErr != nil {
		panic(getErr)
	}

	_, getErr = ui.Hydrate(renderProductsPage(), "#app", ui.HydrationOptions{
		Bootstrap: getBootstrap,
	})
	if getErr != nil {
		panic(getErr)
	}
	utils.WaitForever()
}
```

Why this is the production-shaped SSR baseline:

- the server decides exactly what resume data crosses into the client
- typed payload helpers keep route data and session hints explicit instead of hiding them in one ad hoc blob
- hydration resumes the same tree instead of client-rendering a second copy from scratch

## Scale-Up Example

In a larger routed app, move heavier bootstrap payloads into sidecars and reuse route data on the first hydrated loader pass before falling back to the normal client path.

```go
package server

import (
	"fmt"
	"net/http"

	"github.com/monstercameron/GoWebComponents/ui"
)

// handleDocsPage emits a sidecar bootstrap reference instead of inlining a larger payload.
func handleDocsPage(getW http.ResponseWriter, getR *http.Request) {
	getBootstrap := ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{Path: "/docs/getting-started"},
	}
	_ = ui.RegisterRouteBootstrapData(&getBootstrap, "docs", "/docs/getting-started", map[string]any{
		"revision": 3,
		"items":    []string{"Install", "Render", "Hydrate"},
	})

	getPayload, getErr := ui.MarshalSSRBootstrapBinary(getBootstrap)
	if getErr != nil {
		http.Error(getW, getErr.Error(), http.StatusInternalServerError)
		return
	}
	getRefScript, getErr := ui.RenderBootstrapReferenceScript(ui.SSRBootstrapReference{
		URL:    "/bootstrap/docs-getting-started.cbor",
		Format: ui.SSRBootstrapFormatCBOR,
	}, "")
	if getErr != nil {
		http.Error(getW, getErr.Error(), http.StatusInternalServerError)
		return
	}

	storeBootstrapSidecar("/bootstrap/docs-getting-started.cbor", getPayload)
	getMarkup, _ := ui.RenderToString(renderDocsPage())
	_, _ = fmt.Fprintf(getW, "<!doctype html><html><body><div id=\"app\">%s</div>%s</body></html>", getMarkup, getRefScript)
}
```

```go
package client

import (
	"context"

	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type docsBootstrapData struct {
	Revision int      `json:"revision"`
	Items    []string `json:"items"`
}

var (
	initialBootstrap     ui.SSRBootstrap
	initialDocsData      docsBootstrapData
	hasInitialDocsData   bool
	usedInitialDocsData  bool
)

// consumeInitialDocsData returns the bootstrapped route data only once for the initial hydrated route.
func consumeInitialDocsData(getRouteCtx router.RouteContext) (docsBootstrapData, bool) {
	if !hasInitialDocsData || usedInitialDocsData {
		return docsBootstrapData{}, false
	}
	if initialBootstrap.Route.Path != getRouteCtx.Path {
		return docsBootstrapData{}, false
	}
	usedInitialDocsData = true
	return initialDocsData, true
}

// buildDocsLoader reuses bootstrap route data on first resume and falls back to the normal client loader later.
func buildDocsLoader(getCtx context.Context, getRouteCtx router.RouteContext) (router.Attrs, error) {
	if getData, getOk := consumeInitialDocsData(getRouteCtx); getOk {
		return router.Attrs{"revision": getData.Revision, "items": getData.Items, "source": "bootstrap"}, nil
	}
	_ = getCtx
	return router.Attrs{"revision": 4, "items": []string{"Client refresh"}, "source": "live loader"}, nil
}

// main restores a sidecar bootstrap payload, hydrates the router tree, then attaches router listeners.
func main() {
	getBootstrapRef, getErr := ui.ReadBootstrapReferenceScript("")
	if getErr != nil {
		panic(getErr)
	}
	initialBootstrap, getErr = ui.ReadBootstrapReference(getBootstrapRef)
	if getErr != nil {
		panic(getErr)
	}

	getValue, getOk, getErr := ui.ReadRouteBootstrapData[docsBootstrapData](initialBootstrap, "docs", "/docs/getting-started")
	if getErr == nil && getOk {
		initialDocsData = getValue.Value
		hasInitialDocsData = true
	}

	getRouter := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: "/docs/getting-started"})
	getRouter.Register("/docs/:section", renderDocsRoute, router.Options{Loader: buildDocsLoader})

	getRoot := ui.CreateElement(func() ui.Node { return getRouter.Current() })
	_, getErr = ui.Hydrate(getRoot, "#app", ui.HydrationOptions{Bootstrap: initialBootstrap})
	if getErr != nil {
		panic(getErr)
	}
	getRouter.HydrateMount("#app")
	utils.WaitForever()
}
```

Why this scales:

- sidecar payloads keep the HTML response smaller when the resume data grows
- route bootstrap data is consumed once for the initial hydrated pass, then normal client loaders take over
- the router attaches after hydration instead of replacing the already-resumed entry route

## Bootstrap Transfer Rules

Use `ui.SSRBootstrap` intentionally:

- `Route`: initial route path, query, and params
- `Atoms`: runtime-owned atom snapshot restore
- `Data`: app-owned or package-owned bootstrap payloads
- `I18n`: locale and initial message subset
- `IDSeed`: hydration-safe `UseId()` continuity

Prefer the typed payload helpers over one generic `Data` blob:

- `RegisterRouteBootstrapData(...)`
- `RegisterFormBootstrapDefaults(...)`
- `RegisterCacheBootstrapSeed(...)`
- `RegisterSessionBootstrapHint(...)`
- `ReadBootstrapPayload(...)`
- `ReadRouteBootstrapData(...)`
- `ReadFormBootstrapDefaults(...)`
- `ReadCacheBootstrapSeed(...)`
- `ReadSessionBootstrapHint(...)`

Practical rules:

- transfer only public resume data
- keep secrets and raw auth credentials out of bootstrap completely
- filter payloads by owner and scope instead of flattening everything into one map
- prefer JSON-shaped values unless both writer and reader deliberately own a stronger encoding contract

## Hydration Reuse Rules

Use `ui.Hydrate(...)` when:

- matching HTML already exists in the DOM
- the client should reuse that DOM where possible
- bootstrap data should restore atoms, IDs, or route-scoped resume values first

Use strict hydration when:

- tests or development should fail fast on mismatches
- you want subtree fallback and warnings to become hard failures

Keep these runtime boundaries in mind:

- text or attribute mismatches warn and let the client own the final DOM
- structural mismatches fall back per subtree instead of restarting the whole app by default
- uncontrolled form values are preserved conservatively on reused nodes during resume
- effects and subscriptions wait until hydration commit finishes

## Server Integration Rules

The canonical first-party server shape is still normal Go `net/http`:

- request middleware for correlation IDs, recovery, auth, and CSRF
- request-time SSR handlers using `RenderToString(...)`
- bootstrap emission through inline or sidecar helpers
- same-origin JSON endpoints and form handlers
- static asset serving for `.wasm`, `wasm_exec.js`, CSS, and worker assets

Keep these boundaries clear:

- SSR is request-scoped; do not cache request-owned state in globals
- form and mutation authority stays on the server
- same-origin endpoints make auth and hydration reuse easier to keep consistent
- prerender is a separate output mode for build-time HTML, not request-time SSR

## Auth Session And Configuration Boundaries

Treat auth, session, and startup configuration as separate classes of data:

- session hints in bootstrap should describe client-visible state, never credentials
- auth enforcement stays on same-origin handlers, route loaders, and server-owned redirects
- feature flags and runtime configuration that cross into bootstrap should be typed, non-secret, and safe to view in page source
- environment layering, secret lookup, CSP, and compliance policy remain application-owned server concerns

Practical rule:

- if a browser does not need to read a value directly, do not bootstrap it
- if a route decision depends on secret policy, keep the decision on the server and bootstrap only the result

## Streaming Server Functions And Server-Interactive Direction

The manual treats these as bounded advanced directions rather than the default runtime shape:

- streaming SSR is a targeted future-facing delivery mode for explicit loader and buffering contracts, not a replacement for the stable request-render-hydrate path
- server functions should stay behind app-owned transport and auth boundaries when used
- server-interactive or server-owned live UI remains a narrow experiment until latency, offline behavior, backpressure, and reconnection semantics are proven for real apps

For production defaults today:

- use request-time SSR plus typed bootstrap for first paint
- use same-origin loaders, actions, forms, and fetch helpers for later reads and writes
- add streaming or richer server-owned interaction only behind bounded app abstractions

## Observability And Size Budgets

The current exported observability surface is:

- `ui.RegisterSSRObserver(...)`
- `ui.RenderToStringObserved(...)`
- `ui.RenderBootstrapScriptObserved(...)`
- `ui.InspectSSRBootstrapSize(...)`
- `ui.NewSSRBootstrapBudget()`
- `ui.InspectBootstrapPayloads(...)`

Use that slice when:

- you need request-level render or bootstrap timing
- you want warnings before inline bootstrap scripts quietly become too large
- you need to inspect which typed payloads are crossing the server-client boundary

## API Family Reference

Use this table before you widen the SSR or hydration pipeline.

| API family | Representative APIs | Stability | Use it when | Prefer something else when |
| --- | --- | --- | --- | --- |
| Server render entrypoints | `RenderToString`, `RenderToStringObserved` | `Stable` | the server should emit HTML before wasm starts | the page is fully client-only |
| Browser resume entrypoints | `Hydrate`, `HydrateInto` | `Stable` | matching HTML already exists and the client should reuse it | the page should client-render from scratch |
| Routed resume attach | `router.HydrateMount`, `router.HydrateMountElement` | `Stable` entrypoints with deeper lifecycle rules | a routed shell should attach after the initial hydrated route is already present | the app is not routed or can use plain `Mount` |
| Inline bootstrap transport | `RenderBootstrapScript`, `ReadBootstrapScript`, `MarshalSSRBootstrap`, `UnmarshalSSRBootstrap` | `Stable` default | the payload is small enough to live inline | the bootstrap payload is large enough to bloat HTML |
| Sidecar bootstrap transport | `RenderBootstrapReferenceScript`, `ReadBootstrapReferenceScript`, `ReadBootstrapReference`, `MarshalSSRBootstrapBinary`, `UnmarshalSSRBootstrapBinary` | advanced SSR transport surface | sidecar JSON or CBOR is better than inline HTML bloat | the inline JSON payload is still small and simple |
| Typed bootstrap payloads | `RegisterRouteBootstrapData`, `RegisterFormBootstrapDefaults`, `RegisterCacheBootstrapSeed`, `RegisterSessionBootstrapHint`, matching `Read...` helpers | `Stable` public transfer helpers | the app wants explicit scope and reuse policy for resume data | the page does not need resume data for that slice |
| Transfer inspection | `InspectBootstrapPayloads`, `InspectSSRBootstrapSize`, `NewSSRBootstrapBudget` | `Stable` helper surface | payload size or ownership needs review before release | the app has no bootstrap payload at all |
| SSR observability | `RegisterSSRObserver`, `RenderToStringObserved`, `RenderBootstrapScriptObserved` | shipped public observability surface | request-level render or bootstrap events need metrics or logs | you only need plain render output |

## Design Notes And Boundaries

Keep these SSR rules in mind:

- SSR is request-owned HTML generation, not a replacement for server auth or validation
- bootstrap is a public transfer channel, not a secret channel
- route-data reuse during hydration should be explicit and one-time, not a hidden permanent loader bypass
- sidecar transport choice is an operational concern; it does not loosen the public-data boundary
- prerender is for build-time pages; request-time SSR is for request-scoped pages
- hydration continuity matters, but structural correctness matters more than forcing reuse at all costs

## Common Failure Modes

- serializing secrets or raw session material into bootstrap data
- using SSR globals to hold request-owned data
- emitting large inline bootstrap payloads without measuring size budgets
- assuming hydration will silently fix structural markup drift that should really be corrected at the source
- bootstrapping route data without defining when the client should stop trusting it
- attaching the router before hydration has resumed the current route tree
- treating prerender and request-time SSR as if they had the same freshness and security boundaries

## Validation

Use the smallest examples that prove the SSR or hydration slice you are adopting.

Server render and inline bootstrap:

```powershell
go run .\examples\70-render-to-string
go run .\examples\73-ssr-bootstrap
```

Hydration and route-data reuse:

```powershell
go run ./tools/gwc dev -app .\examples\71-hydrate\main.go
go run ./tools/gwc dev -app .\examples\74-ssr-route-data-reuse\main.go
go run ./tools/gwc dev -app .\examples\72-router-hydrate-mount\main.go
```

Production-shaped server integration:

```powershell
go run .\examples\18-ssr-server-routing
go run .\examples\87-ssr-secure-forms
```

## Topic Pagination
Topic 9 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [08 Routing](08-routing.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [10 Browser Interop And Workers](10-browser-interop-and-workers.md)
