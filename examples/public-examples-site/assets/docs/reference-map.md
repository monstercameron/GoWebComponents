# Reference Map

This page cross-links major concepts to the current public APIs, runnable examples, production caveats, and related docs.

Use it when you know the problem you are solving but do not want to guess which package docs or examples cover it best.

## At A Glance

- Start here when you already know the feature area and want the shortest path to the right public package, example, and deeper doc.
- Prefer the public package surface named here over `internal/` implementation details or example-only scaffolding.
- Use [start-here.md](start-here.md) when you need an adoption order. Use this page when you need a problem-to-surface lookup.

## Quick Starting Points

If you are trying to:

- build a client-only UI: start with Rendering And Local State
- wire shared state or async data: start with State And Data
- add routes, loaders, or guards: start with Routing
- server-render and resume in the browser: start with SSR And Hydration
- inspect runtime behavior or structured diagnostics: start with Diagnostics And Debugging
- add installability, service workers, or offline helpers: start with PWA And Offline Support

## HTML Authoring

Public API:

- `html.Div`, `html.Button`, `html.Input`, `html.Tag`
- `html/shorthand` helpers such as `Div`, `Button`, `Class`, `Text`, `Textf`, `If`

Runnable examples:

- [examples/public/ui-render](../examples/public/ui-render)
- [examples/public/create-element](../examples/public/create-element)
- [examples/public/html-forms](../examples/public/html-forms)

Production caveats:

- Prefer typed builders and shorthand helpers over unstructured raw prop maps when the goal is maintainable UI code.
- Keep server-rendered and hydrated markup deterministic when the same authoring path runs on both sides.

Related docs:

- [HTML_SUGAR.md](html-sugar.md)
- [ACCESSIBILITY.md](accessibility-guidance.md)
- [START_HERE.md](start-here.md)

## Rendering And Local State

Public API:

- `ui.Render`, `ui.CreateElement`, `ui.Fragment`
- `ui.UseState`, `ui.UseReducer`, `ui.UseEffect`, `ui.UseRef`, `ui.UsePrevious`, `ui.UseId`

Runnable examples:

- [examples/public/ui-render](../examples/public/ui-render)
- [examples/public/create-element](../examples/public/create-element)
- [examples/public/fragment](../examples/public/fragment)
- [examples/public/use-ref](../examples/public/use-ref)
- [examples/public/use-reducer](../examples/public/use-reducer)
- [examples/public/use-state](../examples/public/use-state)
- [examples/public/use-effect](../examples/public/use-effect)

Production caveats:

- Keep render-time logic deterministic across server and client when the same component participates in SSR and hydration.
- Prefer typed events and typed props over raw attribute maps or ad hoc event plumbing.

Related docs:

- [WORKFLOWS.md](common-workflows.md#build-a-client-only-app)
- [WALKTHROUGHS.md](end-to-end-walkthroughs.md#small-spa)

## State And Data

Public API:

- `state.UseAtom`, `state.UseComputed`, `state.UseDerived`
- `state.ExportSnapshot`, `state.ImportSnapshot`, `state.SaveSnapshot`, `state.RestoreSnapshot`
- `fetch.UseFetch`, `fetch.UseResource[T]`, `fetch.UseCachedResource[T]`, `fetch.Fetch`

Runnable examples:

- [examples/public/use-atom](../examples/public/use-atom)
- [examples/public/use-computed](../examples/public/use-computed)
- [examples/public/use-derived](../examples/public/use-derived)
- [examples/public/snapshot-export-import](../examples/public/snapshot-export-import)
- [examples/public/snapshot-storage](../examples/public/snapshot-storage)
- [examples/public/use-fetch](../examples/public/use-fetch)
- [examples/public/use-resource](../examples/public/use-resource)
- [examples/public/use-cached-resource](../examples/public/use-cached-resource)

Production caveats:

- Keep snapshot payloads serialization-safe.
- Use typed resources for most async data flows and reserve lower-level fetch hooks for cases that need extra request control.

Related docs:

- [WORKFLOWS.md](common-workflows.md#test-a-component-or-app-flow)
- [WALKTHROUGHS.md](end-to-end-walkthroughs.md#data-heavy-dashboard)
- [CONFIGURATION.md](runtime-configuration-and-feature-flags.md)

## Routing

Public API:

- `router.NewHashRouter`, `router.NewRouter`, `Register`, `Mount`
- `router.UseNavigate`, `router.UseParams`, `router.UseQuery`, `router.UseSearchParams`, `router.UseRevalidator`
- `router.Outlet`, `router.Options{Loader: ...}`, redirects, guards, and metadata

Runnable examples:

- [examples/public/hash-router](../examples/public/hash-router)
- [examples/public/browser-router](../examples/public/browser-router)
- [examples/public/use-navigate](../examples/public/use-navigate)
- [examples/public/route-params](../examples/public/route-params)
- [examples/public/route-query](../examples/public/route-query)
- [examples/public/route-loaders](../examples/public/route-loaders)
- [examples/public/nested-layout-routes](../examples/public/nested-layout-routes)
- [examples/public/router-guards](../examples/public/router-guards)

Production caveats:

- Browser-router apps need server-side rewrites for deep links.
- Hash-router apps avoid server rewrites but change URL structure and direct-link behavior.

Related docs:

- [WORKFLOWS.md](common-workflows.md#add-routing)
- [WALKTHROUGHS.md](end-to-end-walkthroughs.md#static-hosted-app)
- [router/README.md](../router/README.md)

## SSR And Hydration

Public API:

- `ui.RenderToString`, `ui.Hydrate`
- `router.HydrateMount`
- bootstrap helpers documented by the SSR examples and migration guide

Runnable examples:

- [examples/server/render-to-string](../examples/server/render-to-string)
- [examples/public/hydration](../examples/public/hydration)
- [examples/public/router-hydrate-mount](../examples/public/router-hydrate-mount)
- [examples/server/server-side-rendering-bootstrap](../examples/server/server-side-rendering-bootstrap)
- [examples/public/server-side-rendering-route-data-reuse](../examples/public/server-side-rendering-route-data-reuse)
- [examples/server/server-side-rendering-routing](../examples/server/server-side-rendering-routing)
- [examples/server/server-side-rendering-secure-forms](../examples/server/server-side-rendering-secure-forms)

Production caveats:

- Server markup and first client render must match closely enough for safe hydration reuse.
- Route data, IDs, metadata, and conditional branches should be validated after resume.

Related docs:

- [MIGRATIONS.md](migration-guide.md)
- [SERVER_INTEGRATION.md](server-integration.md)
- [OBSERVABILITY.md](observability.md)
- [WORKFLOWS.md](common-workflows.md#add-ssr-and-hydration)
- [WALKTHROUGHS.md](end-to-end-walkthroughs.md#server-rendered-app)
- [TROUBLESHOOTING.md](troubleshooting.md#hydration-mismatch-warnings)

## Diagnostics And Debugging

Public API:

- `devtools.Panel`, `devtools.UseSnapshot`, `devtools.SnapshotNow`
- diagnostics surfaced through the devtools package and example catalog

Runnable examples:

- [examples/public/devtools-panel](../examples/public/devtools-panel)
- [examples/public/use-snapshot](../examples/public/use-snapshot)
- [examples/public/snapshot-now](../examples/public/snapshot-now)
- [examples/public/devtools-diagnostics](../examples/public/devtools-diagnostics)
- [examples/server/atlas-commerce-os](../examples/server/atlas-commerce-os)

Production caveats:

- Keep diagnostics payloads free of secrets.
- Validate behavior under both dev-server and production-like serving conditions.

Related docs:

- [ACTIONABLE_ERRORS.md](actionable-errors-and-diagnostics.md)
- [OBSERVABILITY.md](observability.md)
- [LOGGING.md](logging.md)
- [PRERENDER.md](prerender.md)
- [ASSETS.md](assets.md)
- [TROUBLESHOOTING.md](troubleshooting.md)
- [test/README.md](../test/README.md)
- [tools/README.md](../tools/README.md)

## PWA And Offline Support

Public API:

- `pwa.Manifest`, `pwa.MarshalManifestJSON`, `Manifest.Validate`
- `pwa.ObserveInstallability`, `pwa.RegisterServiceWorker`
- `pwa.ParseWasmReleaseManifestJSON`, `pwa.BuildServiceWorkerAssetPlan`, `pwa.BuildCacheStoragePlan`, `pwa.OpenCacheStorageManager`
- `pwa.InspectDiagnostics`, `pwa.MutationQueueDiagnosticsSource`
- `fetch.OpenMutationQueue`

Runnable examples:

- [examples/server/server-side-rendering-routing](../examples/server/server-side-rendering-routing)
- [examples/server/atlas-commerce-os](../examples/server/atlas-commerce-os)

Production caveats:

- the framework ships explicit helpers, but applications still own service-worker code, route fallback policy, and durable-data retention rules
- offline writes, read caching, and installability should be treated as separate concerns rather than one generic offline mode

Related docs:

- [PWA.md](pwa-and-offline-app-support.md)
- [OFFLINE_MUTATIONS.md](offline-mutation-queueing.md)
- [ASSETS.md](assets.md)
- [BROWSER_SUPPORT.md](browser-support.md)
- [CROSS_TAB.md](cross-tab-synchronization.md)

## Multi-Client Coordination

Public API:

- `interop.OpenCrossTabChannel`, `interop.CrossTabChannel.Publish`, `interop.CrossTabChannel.Subscribe`
- `interop.OpenSecondaryWindowChannel`, `interop.WindowOpenerChannel`, `interop.WindowChannel.Publish`, `interop.WindowChannel.Subscribe`
- proposed typed message layer in [MULTI_CLIENTS.md](multi-client-coordination.md)

Runnable examples:

- [examples/public/cross-tab-sync](../examples/public/cross-tab-sync)
- [examples/public/multi-window-console](../examples/public/multi-window-console)

Production caveats:

- keep each client sovereign and isolate it to its own tab, popup, or embedded document root
- treat one client or the server as authoritative per topic instead of replicating arbitrary mutable state
- prefer `intent` and `invalidate` messages over broad cross-client state mirroring

Related docs:

- [CROSS_TAB.md](cross-tab-synchronization.md)
- [MULTI_SURFACE.md](multi-surface-coordination.md)
- [MULTI_CLIENTS.md](multi-client-coordination.md)

## Prerender And Assets

Public API:

- `ui.RenderToString`
- hydration and bootstrap helpers when prerendered pages also resume in the browser
- application-owned manifest or asset-reference helpers

Runnable examples:

- [examples/server/render-to-string](../examples/server/render-to-string)
- [examples/public/hydration](../examples/public/hydration)
- [examples/server/server-side-rendering-bootstrap](../examples/server/server-side-rendering-bootstrap)

Production caveats:

- prerender route enumeration must exclude request-bound or authenticated routes
- generated HTML should resolve hashed assets and base paths through one documented manifest convention
- static pages may remain static, but hydrated pages still follow the normal hydration contract

Related docs:

- [PRERENDER.md](prerender.md)
- [ASSETS.md](assets.md)
- [HYDRATION.md](hydration.md)
- [SERVER_INTEGRATION.md](server-integration.md)
- [HEAD_MANAGEMENT.md](head-management-and-seo-surface.md)

## Ecosystem And Companion Packages

Public API:

- stable public packages are the base extension surface: `ui`, `html`, `state`, `fetch`, `router`
- supported companion integrations may layer on documented package boundaries without requiring runtime internals
- there is no framework-wide plugin registry or directive syntax in the current product surface

Runnable examples:

- [examples/public/devtools-panel](../examples/public/devtools-panel)
- [examples/public/devtools-diagnostics](../examples/public/devtools-diagnostics)
- [examples/server/server-side-rendering-bootstrap](../examples/server/server-side-rendering-bootstrap)
- [examples/server/server-side-rendering-secure-forms](../examples/server/server-side-rendering-secure-forms)

Production caveats:

- prefer companion packages over runtime-internal hooks when adding higher-level integrations
- avoid depending on `internal/` or undocumented runtime behavior for ecosystem packages
- add subsystem-specific hooks only when composition on public APIs is no longer sufficient

Related docs:

- [ADOPTION.md](adoption-baseline.md)
- [COMPARISONS.md](framework-comparisons.md)
- [ECOSYSTEM.md](ecosystem-and-extension-model.md)
- [API_POLICY.md](api-stability-and-support-policy.md)
- [FRAMEWORK_SCOPE.md](framework-scope.md)
- [HEAD_MANAGEMENT.md](head-management-and-seo-surface.md)
- [FORMS.md](forms.md)
- [WORKFLOWS.md](common-workflows.md#ship-a-production-wasm-build)

## Review Checklist

- does each section point to current public APIs instead of internal implementation files or stale compatibility layers
- are the runnable examples still the best maintained examples for that feature area
- are production caveats explicit enough to stop readers from overgeneralizing a demo into a production contract
- do the deeper-doc links match the current mirrored doc titles and current repo guidance
