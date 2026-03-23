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

- [examples/21-ui-render](../examples/21-ui-render)
- [examples/22-create-element](../examples/22-create-element)
- [examples/53-html-forms](../examples/53-html-forms)

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

- [examples/21-ui-render](../examples/21-ui-render)
- [examples/22-create-element](../examples/22-create-element)
- [examples/23-fragment](../examples/23-fragment)
- [examples/24-use-ref](../examples/24-use-ref)
- [examples/28-use-reducer](../examples/28-use-reducer)
- [examples/75-use-state](../examples/75-use-state)
- [examples/76-use-effect](../examples/76-use-effect)

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

- [examples/37-use-atom](../examples/37-use-atom)
- [examples/38-use-computed](../examples/38-use-computed)
- [examples/39-use-derived](../examples/39-use-derived)
- [examples/40-snapshot-export-import](../examples/40-snapshot-export-import)
- [examples/41-snapshot-storage](../examples/41-snapshot-storage)
- [examples/42-use-fetch](../examples/42-use-fetch)
- [examples/43-use-resource](../examples/43-use-resource)
- [examples/44-use-cached-resource](../examples/44-use-cached-resource)

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

- [examples/55-hash-router](../examples/55-hash-router)
- [examples/56-browser-router](../examples/56-browser-router)
- [examples/57-use-navigate](../examples/57-use-navigate)
- [examples/58-route-params](../examples/58-route-params)
- [examples/59-route-query](../examples/59-route-query)
- [examples/60-route-loaders](../examples/60-route-loaders)
- [examples/64-nested-layout-routes](../examples/64-nested-layout-routes)
- [examples/65-router-guards](../examples/65-router-guards)

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

- [examples/70-render-to-string](../examples/70-render-to-string)
- [examples/71-hydrate](../examples/71-hydrate)
- [examples/72-router-hydrate-mount](../examples/72-router-hydrate-mount)
- [examples/73-ssr-bootstrap](../examples/73-ssr-bootstrap)
- [examples/74-ssr-route-data-reuse](../examples/74-ssr-route-data-reuse)
- [examples/18-ssr-server-routing](../examples/18-ssr-server-routing)
- [examples/87-ssr-secure-forms](../examples/87-ssr-secure-forms)

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

- [examples/66-devtools-panel](../examples/66-devtools-panel)
- [examples/67-use-snapshot](../examples/67-use-snapshot)
- [examples/68-snapshot-now](../examples/68-snapshot-now)
- [examples/69-devtools-diagnostics](../examples/69-devtools-diagnostics)
- [examples/86-atlas-commerce-os](../examples/86-atlas-commerce-os)

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

- [examples/18-ssr-server-routing](../examples/18-ssr-server-routing)
- [examples/86-atlas-commerce-os](../examples/86-atlas-commerce-os)

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

- [examples/94-cross-tab-sync](../examples/94-cross-tab-sync)
- [examples/95-multi-window-console](../examples/95-multi-window-console)

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

- [examples/70-render-to-string](../examples/70-render-to-string)
- [examples/71-hydrate](../examples/71-hydrate)
- [examples/73-ssr-bootstrap](../examples/73-ssr-bootstrap)

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

- [examples/66-devtools-panel](../examples/66-devtools-panel)
- [examples/69-devtools-diagnostics](../examples/69-devtools-diagnostics)
- [examples/73-ssr-bootstrap](../examples/73-ssr-bootstrap)
- [examples/87-ssr-secure-forms](../examples/87-ssr-secure-forms)

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
