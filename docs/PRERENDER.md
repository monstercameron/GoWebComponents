# Prerender

This page defines the current prerender contract for GoWebComponents.

Use it when you need build-time HTML output for docs, marketing pages, or hybrid sites and want to understand how prerender differs from request-time SSR.

## At A Glance

- Prerender in this repo means rendering HTML ahead of time with the same native SSR primitives used by request-time SSR.
- The shipped surface today is the render and bootstrap API plus the `prerender.Export(...)` file-emission companion package: `ui.RenderToString(...)`, `ui.RenderToStringObserved(...)`, `ui.MarshalSSRBootstrap(...)`, `ui.MarshalSSRBootstrapBinary(...)`, `ui.RenderBootstrapScript(...)`, `ui.RenderBootstrapReferenceScript(...)`, and `prerender.Export(...)`.
- The repo does not yet ship a generic public site exporter that discovers routes, copies assets, and hashes manifests for arbitrary applications.
- Use prerender when a route can be rendered from build-time data and does not need request-bound auth, session, or CSRF context.
- Use [HYDRATION.md](HYDRATION.md) when the main question is resume behavior after static HTML is emitted.

## Quick API Chooser

Use this route when the question is about:

- build-time HTML generation for one known page: `ui.RenderToString(...)`
- SSR timing and bootstrap size metrics during prerender generation: `ui.RenderToStringObserved(...)`, `ui.MarshalSSRBootstrapObserved(...)`, `ui.MarshalSSRBootstrapBinaryObserved(...)`
- inline bootstrap payloads embedded directly into HTML: `ui.RenderBootstrapScript(...)`
- external JSON or CBOR sidecar payloads: `ui.RenderBootstrapReferenceScript(...)`, `ui.MarshalSSRBootstrap(...)`, `ui.MarshalSSRBootstrapBinary(...)`
- writing prerendered HTML files and optional bootstrap sidecars: `prerender.Export(...)`
- realistic reference implementations: `examples/17-ssr-routing`, `examples/18-ssr-server-routing`, and `examples/102-static-export-site`

## Current Shipped Slice

What is already real in this repo today:

- native HTML rendering on non-browser targets through `ui.RenderToString(...)`
- observed SSR rendering and bootstrap serialization for timing and payload metrics
- inline and referenced bootstrap script generation for hydrated pages
- bootstrap JSON and binary decoding on the wasm client
- a first-party prerender file writer through `prerender.Export(...)`
- example SSR servers and tests that prove route rendering, bootstrap emission, and hydration reuse

What remains outside the public core surface today:

- generic route enumeration for arbitrary apps
- full asset copying and manifest orchestration for arbitrary apps
- selective activation primitives beyond the normal hydration contract

## Example Shape

```go
package main

import (
	"os"

	. "github.com/atdiar/gowebcomponents/html/shorthand"
	"github.com/atdiar/gowebcomponents/ui"
)

func renderPage() (string, string, error) {
	node := Html(
		Body(
			Main(
				H1("Docs"),
				P("Generated at build time."),
			),
		),
	)

	markup, err := ui.RenderToString(node)
	if err != nil {
		return "", "", err
	}

	bootstrap, err := ui.RenderBootstrapReferenceScript(ui.SSRBootstrapReference{
		URL:    "/bootstrap/home.cbor",
		Format: ui.SSRBootstrapFormatCBOR,
	}, "")
	if err != nil {
		return "", "", err
	}

	return markup, bootstrap, nil
}

func writePage() error {
	markup, bootstrap, err := renderPage()
	if err != nil {
		return err
	}
	return os.WriteFile("dist/index.html", []byte(markup+bootstrap), 0644)
}

func main() {
	_ = writePage()
}
```

The remaining application-owned pieces are route discovery, asset copying, and cache-aware rebuild logic around the file writer.

## Prerender-To-Files Pipeline

The repo now ships a small first-party companion package for the file-emission step.

`prerender.Export(...)`:

- computes one stable HTML output path per route
- optionally computes a matching bootstrap sidecar path under `/bootstrap/...`
- creates the output directories
- writes the rendered HTML and optional bootstrap payload bytes to disk

The application still owns:

- route enumeration
- page rendering and layout composition
- copied static assets and hashed manifests
- any deployment-specific post-processing

Example:

```go
package main

import (
	"github.com/monstercameron/GoWebComponents/prerender"
	"github.com/monstercameron/GoWebComponents/ui"
)

func exportSite() error {
	_, err := prerender.Export("dist", []prerender.Route{
		{
			Path:            "/",
			BootstrapFormat: ui.SSRBootstrapFormatJSON,
			Build: func(target prerender.Target) (prerender.RouteOutput, error) {
				markup, err := ui.RenderToString(renderHome())
				if err != nil {
					return prerender.RouteOutput{}, err
				}
				payload, err := ui.MarshalSSRBootstrap(ui.SSRBootstrap{
					State: map[string]interface{}{"screen": "home"},
				})
				if err != nil {
					return prerender.RouteOutput{}, err
				}
				ref, err := ui.RenderBootstrapReferenceScript(ui.SSRBootstrapReference{
					URL:    target.BootstrapURL,
					Format: ui.SSRBootstrapFormatJSON,
				}, "")
				if err != nil {
					return prerender.RouteOutput{}, err
				}
				return prerender.RouteOutput{
					HTML:      "<!doctype html><html><body>" + markup + ref + "</body></html>",
					Bootstrap: payload,
				}, nil
			},
		},
		{
			Path: "/docs/getting-started",
			Build: func(target prerender.Target) (prerender.RouteOutput, error) {
				markup, err := ui.RenderToString(renderDocs())
				if err != nil {
					return prerender.RouteOutput{}, err
				}
				return prerender.RouteOutput{
					HTML: "<!doctype html><html><body>" + markup + "</body></html>",
				}, nil
			},
		},
	})
	return err
}
```

The emitted layout follows the current route convention:

- `/` -> `dist/index.html`
- `/docs/getting-started` -> `dist/docs/getting-started/index.html`
- sidecar payloads -> `dist/bootstrap/...` with `.json` or `.cbor` extensions

This closes the file-writing gap without claiming that route discovery or arbitrary asset copying is solved generically for every app.

## First-Class Output Mode

Static prerender is a supported output mode for the project, but it is not the same thing as request-time SSR.

The distinction is:

- request-time SSR renders per incoming request and can depend on request auth, sessions, CSRF state, and dynamic server data
- prerender renders at build time to files and targets pages whose initial HTML can be produced ahead of time

Prerender is the right fit for:

- docs sites
- marketing pages
- product catalogs or public content that can tolerate build-time freshness
- hybrid apps where some routes are static and others still require request-time SSR

Prerender is not the right fit for:

- per-user authenticated shells
- workflows that require request-bound security context
- routes whose correctness depends on per-request mutation or live server state

## Tooling Boundary

The current model is:

- core rendering primitives stay in the current runtime and `ui.RenderToString(...)` surface
- prerender orchestration belongs in a companion build or export workflow, not in ad hoc app scripts scattered across examples

This means prerender is supported, but the route enumeration, file emission, asset copying, and rebuild orchestration can live in dedicated build tooling instead of bloating the core runtime API.

Today that boundary is visible in the repo itself:

- `ui` ships the render and bootstrap primitives
- the SSR examples prove end-to-end render, payload, and hydration behavior
- repo tooling contains example-specific static shell generation, but not a generic public exporter for arbitrary apps

## Route Enumeration

The prerender route model should stay explicit.

Prerender tooling should support:

- a fixed list of canonical routes for small docs or marketing sites
- parameter expansion hooks for routes such as `/blog/:slug` or `/docs/:section/:page`
- optional generator-style enumeration from CMS data, filesystem content, or application-owned manifests
- fallback declarations for routes that cannot be known at build time and therefore stay request-time or client-only

The important constraints are:

- prerender should only emit canonical paths, not every redirect source or alias
- route expansion must be application-owned when it depends on external data or protected APIs
- any route that requires request auth, per-request session state, or request-specific CSRF material should be excluded from prerender enumeration

## Output Conventions

The output layout should be predictable across static hosts.

The baseline convention is:

- one HTML file per resolved route
- one sidecar bootstrap payload per HTML file when transferred state is needed
- one shared asset manifest for hashed wasm, JS helpers, CSS, images, and copied static files
- one copied static-asset tree for files that are not fingerprinted by the build

The exact directory names can live in build tooling, but the contract should preserve these properties:

- generated HTML can resolve assets through a manifest instead of hard-coded filenames
- bootstrap payloads can be emitted inline or as sidecars without changing the hydration contract
- base-path deployment can rewrite one output root instead of every component or route template

## Client Resume Expectations

Prerendered output does not imply that every page hydrates.

The activation model should be:

- purely static pages may remain static with no client resume
- interactive pages may hydrate the full page shell
- hybrid pages may prerender broad layout and content while only specific client-owned sections attach behavior after load

Until selective activation primitives exist, applications should assume prerendered pages either:

- remain fully static, or
- hydrate through the same full-page hydration contract used by SSR output

Prerender tooling should not invent a separate bootstrap or resume model for static exports.

## Islands And Selective Activation Model

The first-class islands model for this project is explicit multi-root ownership, not magical per-node hydration priority.

The ownership split is:

- static region: prerendered HTML that never receives a framework-owned browser root
- island root: one explicit DOM container that a browser-owned `ui.Hydrate(...)` or `ui.Render(...)` call targets
- route shell: the outer page or application shell that may itself stay static, hydrate once, or host nested island roots

The intended activation rules are:

- prerender the whole page for first paint
- keep purely presentational regions static with no browser root at all
- hydrate an island with `ui.Hydrate(...)` when the server already emitted matching HTML for that island
- mount an island with `ui.Render(...)` when the prerendered page emitted only a placeholder shell for that region

This means selective activation is achieved through several explicit roots that each use the normal render or hydration contract. It is not a separate hidden transport or scheduler.

### Composition With Routing

Route ownership stays explicit.

Supported compositions are:

- static route plus local islands where navigation is plain document navigation and only selected panels or widgets activate in the browser
- prerendered route shell plus one hydrated browser router root when the full route tree should become client-owned after startup
- prerendered route shell plus smaller island roots for selected interactive regions when a full browser-router takeover is unnecessary

The important boundary is:

- one DOM subtree has one browser root owner
- an outer router root must not also try to hydrate a nested island subtree that a second root will own directly
- shared route metadata, canonical URLs, and asset references still belong to the prerendered document regardless of how many islands activate later

### Composition With Async Boundaries

`ui.AsyncBoundary(...)` and `ui.Lazy(...)` compose inside an island after that island activates. They are not island discovery primitives by themselves.

Use this split:

- islands decide which regions get a browser root at all
- async boundaries decide how a given activated island reveals deferred work, loading states, or retries
- error boundaries remain local recovery for the activated island subtree, not for the static document around it

This keeps selective activation and deferred loading separate instead of overloading `ui.Lazy(...)` to mean both.

### Correctness Guarantees

Each activated island keeps the normal framework guarantees for its own root:

- matching server HTML can be reused through `ui.Hydrate(...)`
- text or attribute mismatches warn and let the client commit own the final DOM
- structural mismatches fall back only for that island root, not for unrelated static siblings or other islands
- static regions outside the island remain untouched by that island runtime

Shared state between islands is application-owned. If two islands need common data, that data must come from:

- duplicated public bootstrap payloads
- shared browser-readable config
- app-owned fetch or cache layers
- other explicit browser coordination

There is no hidden cross-root atom or event-replay system implied by the islands model.

### Current Non-Goals

This definition does not claim:

- automatic island discovery from arbitrary component trees
- implicit event replay across not-yet-activated islands
- framework-owned hydration prioritization across many island roots
- one universal export layout that chooses islands automatically for arbitrary apps

Those remain higher-level tooling or runtime evolution topics. The first-class model here is explicit static-region versus island-root ownership with normal render and hydration semantics at each activated root.

## Invalidation And Rebuild Guidance

Prerender correctness depends on rebuild discipline.

The rebuild triggers should be:

- content changes for the route itself
- shared layout or component changes that affect emitted HTML
- asset-manifest changes that alter referenced filenames or hashes
- route-data source changes for any route expanded during the build

In local development, prerender tooling should prefer incremental rebuilds for the changed route set plus any shared-layout dependents.

In CI or release pipelines, full rebuilds are acceptable when dependency tracking is too coarse, but generated output should still be deterministic from the same inputs.

## Relationship To Hydration

Prerendered output may still hydrate in the browser when the route needs client interaction, router ownership, or stateful islands.

The important rule is that prerendered HTML still follows the same hydration contract as request-time SSR when hydration is enabled:

- matching HTML is reusable
- bootstrap state must stay serialization-safe
- pages that do not need activation may remain static instead of hydrating automatically

For hydrated prerendered routes, the practical choices today are:

- inline bootstrap scripts for smaller payloads
- referenced JSON or CBOR sidecars when payload size or cache behavior matters
- the same DOM reuse and mismatch fallback behavior documented in `HYDRATION.md`

## Review Checklist

- does the page clearly distinguish shipped render/bootstrap primitives from still-app-owned export orchestration
- are prerender-safe routes separated from request-bound authenticated or mutation-driven routes
- does the output guidance preserve the existing hydration contract instead of inventing a second resume model
- are route enumeration, asset copying, and rebuild invalidation described as exporter responsibilities rather than core runtime behavior
