# Start Here

This page is the recommended path for new GoWebComponents adopters.

Use it when you want the shortest route from evaluation to a production-shaped app without guessing which docs reflect the current public surface.

## Current Status

Shipped today:

- a stable application-authoring surface centered on `ui`, `html`, `state`, `fetch`, and `router`
- companion guidance for `devtools`, `interop`, SSR or hydration, server integration, testing, and troubleshooting
- a large example catalog that spans isolated API demos, integrated client apps, SSR flows, and the Atlas reference app

Not shipped today:

- one single mandatory starter template for every app shape
- a requirement that every adopter use SSR, routing, or companion packages on day one

The shortest successful path is still incremental: start from the smallest surface that fits your app, then layer in routing, fetch, shared state, SSR, or diagnostics only when your app actually needs them.

## Recommended Adoption Path

1. Read the root [README.md](../README.md) for the current public package surface, build requirements, and repo-level commands.
2. Read [API_POLICY.md](API_POLICY.md) so you know which packages are stable, which surfaces are experimental, and how upgrades are handled.
3. Read [WORKFLOWS.md](WORKFLOWS.md) and pick the workflow that matches your app: client-only, router-based, SSR, testing, production wasm build, or hydration debugging.
4. Use [examples/README.md](../examples/README.md) to jump into the smallest runnable example that demonstrates the feature you need.
5. Use [WALKTHROUGHS.md](WALKTHROUGHS.md) when you want a realistic app shape instead of a single isolated feature demo.
6. Use [TROUBLESHOOTING.md](TROUBLESHOOTING.md) when the wasm build, dev server, hydration flow, or router setup is not behaving as expected.
7. Use [ONBOARDING.md](ONBOARDING.md) when you want the repo’s explicit starter path, prerequisites, or inner-loop workflow without reverse-engineering examples.

## Preferred Modern Surface

For new code, treat these packages as the supported application-authoring surface:

- `ui`: components, hooks, rendering, hydration, async helpers, events, portals, and form helpers
- `html`: typed HTML builders and DOM props
- `state`: shared state, computed values, derived state, and snapshots
- `fetch`: fetch hooks, typed resources, and imperative fetch flows
- `interop`: typed browser API wrappers and dynamic module helpers for cases that still need browser-specific interop
- `router`: hash and browser routers, params, queries, guards, metadata, loaders, and hydration-aware mounting
- `devtools`: diagnostics, snapshots, and embedded inspection UI

Avoid treating repo layout such as `internal/runtime` as application API. Those directories are useful when debugging the framework itself, but they are not the supported authoring path for apps.

## First Example Picks

Choose the first example based on the app shape you are building:

- Small client-only app: `examples/01-counter`, `examples/21-ui-render`, `examples/75-use-state`, `examples/76-use-effect`
- Forms and input-heavy UI: `examples/02-text-input`, `examples/51-use-form`, `examples/53-html-forms`, `examples/79-form-accessibility`
- Shared state and fetch: `examples/37-use-atom`, `examples/39-use-derived`, `examples/43-use-resource`, `examples/44-use-cached-resource`
- Client-side routing: `examples/55-hash-router`, `examples/56-browser-router`, `examples/60-route-loaders`, `examples/64-nested-layout-routes`
- SSR or hydration: `examples/70-render-to-string`, `examples/71-hydrate`, `examples/72-router-hydrate-mount`, `examples/18-ssr-server-routing`, `examples/87-ssr-secure-forms`, `docs/HYDRATION.md`, `docs/SERVER_INTEGRATION.md`
- Browser interop and web components: `examples/90-browser-interop`, `examples/91-worker-text-index`, `examples/94-cross-tab-sync`, `examples/95-multi-window-console`, `examples/88-web-components`, `docs/INTEROP.md`, `docs/CUSTOM_ELEMENTS.md`, `docs/WORKERS.md`, `docs/OFFLINE_MUTATIONS.md`, `docs/CROSS_TAB.md`, `docs/MULTI_SURFACE.md`
- Debugging and diagnostics: `examples/66-devtools-panel`, `examples/69-devtools-diagnostics`, `examples/86-atlas-commerce-os`

## Production-Oriented Defaults

If you are starting a new application today, these defaults are the least surprising path:

- Start with `ui` and `html` for browser rendering.
- Add `router` only once you need multiple pages, params, loaders, or route metadata.
- Add `state` for shared state instead of building custom registries.
- Use [STATE_ARCHITECTURE.md](STATE_ARCHITECTURE.md) once the app needs a consistent split between local hooks, reducers, context, atoms, selectors, and snapshots.
- Use `fetch.UseResource[T]` for typed async data and `fetch.UseFetch` only when you need lower-level request control.
- Use [DATA_LOADING_AND_MUTATION_ARCHITECTURE.md](DATA_LOADING_AND_MUTATION_ARCHITECTURE.md) once the app needs a consistent split between route loaders, typed resources, shared cache, form submits, optimistic updates, and offline replay.
- Use [AUTH_AND_SESSION_INTEGRATION.md](AUTH_AND_SESSION_INTEGRATION.md) when the app needs one practical answer for cookie sessions, SSR auth hints, guarded routes, same-origin APIs, and logout handling.
- Use [BUSINESS_APP_FORM_RECIPES.md](BUSINESS_APP_FORM_RECIPES.md) when internal-tool forms need one practical path for validation, pending UX, server errors, redirects, uploads, and authoritative submit handling.
- Add SSR only when your app needs request-time HTML, route-aware bootstrapping, or hydration reuse.
- Validate production flows with both Go tests and Playwright browser tests.

## Current Boundary

This page is an entrypoint, not a full architecture spec.

It does claim:

- the recommended starting surface is the documented public package set, not repo internals
- examples and workflows are the fastest path to the current supported app shapes
- troubleshooting, testing, and SSR guidance already exist and should be used early instead of reverse-engineering the repo

It does not claim:

- that every application needs all public packages
- that the experimental docs define the default adoption path
- that reading every document in `docs/` is the best first move for new adopters

## Related Docs

- [README.md](../README.md)
- [WORKFLOWS.md](WORKFLOWS.md)
- [CACHE.md](CACHE.md)
- [DATA_LOADING_AND_MUTATION_ARCHITECTURE.md](DATA_LOADING_AND_MUTATION_ARCHITECTURE.md)
- [AUTH_AND_SESSION_INTEGRATION.md](AUTH_AND_SESSION_INTEGRATION.md)
- [BUSINESS_APP_FORM_RECIPES.md](BUSINESS_APP_FORM_RECIPES.md)
- [STATE_ARCHITECTURE.md](STATE_ARCHITECTURE.md)
- [CUSTOM_ELEMENTS.md](CUSTOM_ELEMENTS.md)
- [CROSS_TAB.md](CROSS_TAB.md)
- [ERROR_BOUNDARIES.md](ERROR_BOUNDARIES.md)
- [INTEROP.md](INTEROP.md)
- [MULTI_SURFACE.md](MULTI_SURFACE.md)
- [OFFLINE_MUTATIONS.md](OFFLINE_MUTATIONS.md)
- [PRODUCTION_CORRECTNESS.md](PRODUCTION_CORRECTNESS.md)
- [SCHEDULING.md](SCHEDULING.md)
- [HYDRATION.md](HYDRATION.md)
- [SERVER_INTEGRATION.md](SERVER_INTEGRATION.md)
- [OBSERVABILITY.md](OBSERVABILITY.md)
- [LOGGING.md](LOGGING.md)
- [PRERENDER.md](PRERENDER.md)
- [ASSETS.md](ASSETS.md)
- [BROWSER_SUPPORT.md](BROWSER_SUPPORT.md)
- [PWA.md](PWA.md)
- [SECURITY.md](SECURITY.md)
- [CONFIGURATION.md](CONFIGURATION.md)
- [WASM_RELEASES.md](WASM_RELEASES.md)
- [BUILD_EXPERIMENTS.md](BUILD_EXPERIMENTS.md)
- [ONBOARDING.md](ONBOARDING.md)
- [STATE_TRANSFER.md](STATE_TRANSFER.md)
- [STREAMING_SSR.md](STREAMING_SSR.md)
- [WORKERS.md](WORKERS.md)
- [ROUTER_AUTH.md](ROUTER_AUTH.md)
- [WALKTHROUGHS.md](WALKTHROUGHS.md)
- [REFERENCE_MAP.md](REFERENCE_MAP.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- [examples/README.md](../examples/README.md)
