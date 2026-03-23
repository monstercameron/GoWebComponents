# Prerender

This page defines the current prerender contract for GoWebComponents.

Use it when you need build-time HTML output for docs, marketing pages, or hybrid sites and want to understand how prerender differs from request-time SSR.

## At A Glance

- Prerender in this repo means rendering HTML ahead of time with the same native SSR primitives used by request-time SSR.
- The shipped surface today is the render and bootstrap API: `ui.RenderToString(...)`, `ui.RenderToStringObserved(...)`, `ui.MarshalSSRBootstrap(...)`, `ui.MarshalSSRBootstrapBinary(...)`, `ui.RenderBootstrapScript(...)`, and `ui.RenderBootstrapReferenceScript(...)`.
- The repo does not yet ship a generic public site exporter that discovers routes, writes every HTML file, and copies assets for arbitrary applications.
- Use prerender when a route can be rendered from build-time data and does not need request-bound auth, session, or CSRF context.
- Use [HYDRATION.md](HYDRATION.md) when the main question is resume behavior after static HTML is emitted.

## Quick API Chooser

Use this route when the question is about:

- build-time HTML generation for one known page: `ui.RenderToString(...)`
- SSR timing and bootstrap size metrics during prerender generation: `ui.RenderToStringObserved(...)`, `ui.MarshalSSRBootstrapObserved(...)`, `ui.MarshalSSRBootstrapBinaryObserved(...)`
- inline bootstrap payloads embedded directly into HTML: `ui.RenderBootstrapScript(...)`
- external JSON or CBOR sidecar payloads: `ui.RenderBootstrapReferenceScript(...)`, `ui.MarshalSSRBootstrap(...)`, `ui.MarshalSSRBootstrapBinary(...)`
- realistic reference implementations: `examples/17-ssr-routing` and `examples/18-ssr-server-routing`

## Current Shipped Slice

What is already real in this repo today:

- native HTML rendering on non-browser targets through `ui.RenderToString(...)`
- observed SSR rendering and bootstrap serialization for timing and payload metrics
- inline and referenced bootstrap script generation for hydrated pages
- bootstrap JSON and binary decoding on the wasm client
- example SSR servers and tests that prove route rendering, bootstrap emission, and hydration reuse

What remains outside the public core surface today:

- generic route enumeration for arbitrary apps
- a first-class public exporter that writes one file per route and copies assets
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

The missing piece is the exporter around that snippet: route discovery, file naming, asset copying, and cache-aware rebuild logic are still application-owned.

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
