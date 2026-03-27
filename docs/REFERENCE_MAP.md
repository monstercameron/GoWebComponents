# Reference Map

This page cross-links major concepts to the current public APIs, runnable examples, production caveats, and related docs.

Use it when you know the problem you are solving but do not want to guess which package docs or examples cover it best.

## At A Glance

- Start here when you already know the feature area and want the shortest path to the right public package, example, and deeper doc.
- Prefer the public package surface named here over `internal/` implementation details or example-only scaffolding.
- Use [START_HERE.md](START_HERE.md) when you need an adoption order. Use this page when you need a problem-to-surface lookup.

## Quick Starting Points

If you are trying to:

- build a client-only UI: start with Rendering And Local State
- wire shared state or async data: start with State And Data
- add routes, loaders, or guards: start with Routing
- server-render and resume in the browser: start with SSR And Hydration
- reason about worker-backed rendering or background compute: start with Worker And Parallel Rendering
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

- [HTML_SUGAR.md](HTML_SUGAR.md)
- [ACCESSIBILITY.md](ACCESSIBILITY.md)
- [START_HERE.md](START_HERE.md)

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
- [examples/28-scaling-local-state-with-use-reducer](../examples/28-scaling-local-state-with-use-reducer)
- [examples/75-use-state](../examples/75-use-state)
- [examples/76-use-effect](../examples/76-use-effect)

Production caveats:

- Keep render-time logic deterministic across server and client when the same component participates in SSR and hydration.
- Prefer typed events and typed props over raw attribute maps or ad hoc event plumbing.

Related docs:

- [WORKFLOWS.md](WORKFLOWS.md#build-a-client-only-app)
- [WALKTHROUGHS.md](WALKTHROUGHS.md#small-spa)
- [SCALING_LOCAL_STATE_WITH_USE_REDUCER.md](SCALING_LOCAL_STATE_WITH_USE_REDUCER.md)

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

- [WORKFLOWS.md](WORKFLOWS.md#test-a-component-or-app-flow)
- [WALKTHROUGHS.md](WALKTHROUGHS.md#data-heavy-dashboard)
- [CONFIGURATION.md](CONFIGURATION.md)

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

- [WORKFLOWS.md](WORKFLOWS.md#add-routing)
- [WALKTHROUGHS.md](WALKTHROUGHS.md#static-hosted-app)
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

- [MIGRATIONS.md](MIGRATIONS.md)
- [SERVER_INTEGRATION.md](SERVER_INTEGRATION.md)
- [OBSERVABILITY.md](OBSERVABILITY.md)
- [WORKFLOWS.md](WORKFLOWS.md#add-ssr-and-hydration)
- [WALKTHROUGHS.md](WALKTHROUGHS.md#server-rendered-app)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md#hydration-mismatch-warnings)

## Worker And Parallel Rendering

Public API:

- current shipped worker surface: `interop.OpenWorker`, `interop.OpenGoWASMWorker`, `interop.OpenWorkerPool`, `interop.OpenMessageChannel`, `interop.OpenSharedBuffer`
- current explicit narrow-update surface: `ui.ReactiveRegion`, `state.Select`, `state.UseAtom`, `state.UseDerived`
- current public parallel-region shell surface: `ui.RegisterParallelRegion`, `ui.ParallelRegion`, `ui.BuildParallelRegionSourceIDs`
- proposed worker-backed render architecture: design only, documented in `MULTITHREADED_RUNTIME.md`
- execution backlog for the proposed worker-backed render runtime: `MULTITHREADED_RUNTIME_TODO.md`

Runnable examples:

- [examples/108-parallel-region-basic](../examples/108-parallel-region-basic)
- [examples/109-parallel-region-grid](../examples/109-parallel-region-grid)
- [examples/110-parallel-region-diagnostics](../examples/110-parallel-region-diagnostics)
- [examples/91-worker-text-index](../examples/91-worker-text-index)
- [examples/100-ai-chat-wizard](../examples/100-ai-chat-wizard)

Production caveats:

- `ui.ParallelRegion(...)` is a shipped local-first public shell: browser rerenders dispatch validated snapshots into runtime2, while default public DOM commit ownership remains on the main thread
- DOM ownership stays on the main thread even in the proposed multithreaded runtime
- shared-memory transport requires cross-origin isolation, but the design keeps a message-passing fallback path

Related docs:

- [WORKERS.md](WORKERS.md)
- [FINE_GRAINED_REACTIVITY.md](FINE_GRAINED_REACTIVITY.md)
- [MULTITHREADED_RUNTIME.md](MULTITHREADED_RUNTIME.md)
- [MULTITHREADED_RUNTIME_TODO.md](MULTITHREADED_RUNTIME_TODO.md)
- [PARALLEL_REGION_AUTHORING.md](PARALLEL_REGION_AUTHORING.md)
- [PARALLEL_REGION_TROUBLESHOOTING.md](PARALLEL_REGION_TROUBLESHOOTING.md)
- [STATE_ARCHITECTURE.md](STATE_ARCHITECTURE.md)
- [SCHEDULING.md](SCHEDULING.md)
- [HYDRATION.md](HYDRATION.md)

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

- [ACTIONABLE_ERRORS.md](ACTIONABLE_ERRORS.md)
- [OBSERVABILITY.md](OBSERVABILITY.md)
- [LOGGING.md](LOGGING.md)
- [PRERENDER.md](PRERENDER.md)
- [ASSETS.md](ASSETS.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
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

- [PWA.md](PWA.md)
- [OFFLINE_MUTATIONS.md](OFFLINE_MUTATIONS.md)
- [ASSETS.md](ASSETS.md)
- [BROWSER_SUPPORT.md](BROWSER_SUPPORT.md)
- [CROSS_TAB.md](CROSS_TAB.md)

## Multi-Client Coordination

Public API:

- `interop.OpenCrossTabChannel`, `interop.CrossTabChannel.Publish`, `interop.CrossTabChannel.Subscribe`
- `interop.OpenSecondaryWindowChannel`, `interop.WindowOpenerChannel`, `interop.WindowChannel.Publish`, `interop.WindowChannel.Subscribe`
- proposed typed message layer in [MULTI_CLIENTS.md](MULTI_CLIENTS.md)

Runnable examples:

- [examples/94-cross-tab-sync](../examples/94-cross-tab-sync)
- [examples/95-multi-window-console](../examples/95-multi-window-console)

Production caveats:

- keep each client sovereign and isolate it to its own tab, popup, or embedded document root
- treat one client or the server as authoritative per topic instead of replicating arbitrary mutable state
- prefer `intent` and `invalidate` messages over broad cross-client state mirroring

Related docs:

- [CROSS_TAB.md](CROSS_TAB.md)
- [MULTI_SURFACE.md](MULTI_SURFACE.md)
- [MULTI_CLIENTS.md](MULTI_CLIENTS.md)

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

- [PRERENDER.md](PRERENDER.md)
- [ASSETS.md](ASSETS.md)
- [HYDRATION.md](HYDRATION.md)
- [SERVER_INTEGRATION.md](SERVER_INTEGRATION.md)
- [HEAD_MANAGEMENT.md](HEAD_MANAGEMENT.md)

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

- [ADOPTION.md](ADOPTION.md)
- [COMPARISONS.md](COMPARISONS.md)
- [ECOSYSTEM.md](ECOSYSTEM.md)
- [API_POLICY.md](API_POLICY.md)
- [FRAMEWORK_SCOPE.md](FRAMEWORK_SCOPE.md)
- [HEAD_MANAGEMENT.md](HEAD_MANAGEMENT.md)
- [FORMS.md](FORMS.md)
- [WORKFLOWS.md](WORKFLOWS.md#ship-a-production-wasm-build)

## Review Checklist

- does each section point to current public APIs instead of internal implementation files or stale compatibility layers
- are the runnable examples still the best maintained examples for that feature area
- are production caveats explicit enough to stop readers from overgeneralizing a demo into a production contract
- do the deeper-doc links match the current mirrored doc titles and current repo guidance
