# Prerender

This page defines the current intended static prerender story for GoWebComponents.

Use it when you need build-time HTML output for docs, marketing pages, or hybrid sites and want to understand how prerender differs from request-time SSR.

## First-Class Output Mode

Static prerender is an intended first-class output mode for the project, but it is not the same thing as request-time SSR.

The intended distinction is:

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

The intended model is:

- core rendering primitives stay in the current runtime and `ui.RenderToString(...)` surface
- prerender orchestration belongs in a companion build or export workflow, not in ad hoc app scripts scattered across examples

This means prerender should be treated as a supported output mode, but the route enumeration, file emission, asset copying, and rebuild orchestration can live in dedicated build tooling instead of bloating the core runtime API.

## Route Enumeration

The intended prerender route model is explicit.

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

The intended output layout should be predictable across static hosts.

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

The intended activation model is:

- purely static pages may remain static with no client resume
- interactive pages may hydrate the full page shell
- hybrid pages may prerender broad layout and content while only specific client-owned sections attach behavior after load

Until selective activation primitives exist, applications should assume prerendered pages either:

- remain fully static, or
- hydrate through the same full-page hydration contract used by SSR output

Prerender tooling should not invent a separate bootstrap or resume model for static exports.

## Invalidation And Rebuild Guidance

Prerender correctness depends on rebuild discipline.

The intended rebuild triggers are:

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

More detailed file-emission tooling, example apps, and asset-pipeline helpers remain separate backlog work.
