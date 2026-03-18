# Reference Map

This page cross-links major concepts to the current public APIs, runnable examples, production caveats, and related docs.

Use it when you know the problem you are solving but do not want to guess which package docs or examples cover it best.

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

- [WORKFLOWS.md](WORKFLOWS.md#build-a-client-only-app)
- [WALKTHROUGHS.md](WALKTHROUGHS.md#small-spa)

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

Production caveats:

- Server markup and first client render must match closely enough for safe hydration reuse.
- Route data, IDs, metadata, and conditional branches should be validated after resume.

Related docs:

- [MIGRATIONS.md](MIGRATIONS.md)
- [WORKFLOWS.md](WORKFLOWS.md#add-ssr-and-hydration)
- [WALKTHROUGHS.md](WALKTHROUGHS.md#server-rendered-app)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md#hydration-mismatch-warnings)

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

- [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- [test/README.md](../test/README.md)
- [tools/README.md](../tools/README.md)