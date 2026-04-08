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
2. Read [API_POLICY.md](api-stability-and-support-policy.md) so you know which packages are stable, which surfaces are experimental, and how upgrades are handled.
3. Read [WORKFLOWS.md](common-workflows.md) and pick the workflow that matches your app: client-only, router-based, SSR, testing, production wasm build, or hydration debugging.
4. Use [examples/README.md](../examples/README.md) to jump into the smallest runnable example that demonstrates the feature you need.
5. Use [WALKTHROUGHS.md](end-to-end-walkthroughs.md) when you want a realistic app shape instead of a single isolated feature demo.
6. Use [TROUBLESHOOTING.md](troubleshooting.md) when the wasm build, dev server, hydration flow, or router setup is not behaving as expected.
7. Use [ONBOARDING.md](onboarding-and-bootstrap.md) when you want the repoâ€™s explicit starter path, prerequisites, or inner-loop workflow without reverse-engineering examples.

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

- Small client-only app: `examples/public/counter`, `examples/public/ui-render`, `examples/public/use-state`, `examples/public/use-effect`
- Forms and input-heavy UI: `examples/public/text-input`, `examples/public/use-form`, `examples/public/html-forms`, `examples/public/form-accessibility`
- Shared state and fetch: `examples/public/use-atom`, `examples/public/use-derived`, `examples/public/use-resource`, `examples/public/use-cached-resource`
- Client-side routing: `examples/public/hash-router`, `examples/public/browser-router`, `examples/public/route-loaders`, `examples/public/nested-layout-routes`
- SSR or hydration: `examples/server/render-to-string`, `examples/public/hydration`, `examples/public/router-hydrate-mount`, `examples/server/server-side-rendering-routing`, `examples/server/server-side-rendering-secure-forms`, `docs/HYDRATION.md`, `docs/SERVER_INTEGRATION.md`
- Browser interop and web components: `examples/public/browser-interop`, `examples/public/worker-text-index`, `examples/public/cross-tab-sync`, `examples/public/multi-window-console`, `examples/public/web-components`, `docs/INTEROP.md`, `docs/CUSTOM_ELEMENTS.md`, `docs/WORKERS.md`, `docs/OFFLINE_MUTATIONS.md`, `docs/CROSS_TAB.md`, `docs/MULTI_SURFACE.md`
- Debugging and diagnostics: `examples/public/devtools-panel`, `examples/public/devtools-diagnostics`, `examples/server/atlas-commerce-os`

## Production-Oriented Defaults

If you are starting a new application today, these defaults are the least surprising path:

- Start with `ui` and `html` for browser rendering.
- Add `router` only once you need multiple pages, params, loaders, or route metadata.
- Add `state` for shared state instead of building custom registries.
- Use `fetch.UseResource[T]` for typed async data and `fetch.UseFetch` only when you need lower-level request control.
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
- [WORKFLOWS.md](common-workflows.md)
- [CACHE.md](shared-cache-design.md)
- [CUSTOM_ELEMENTS.md](custom-elements.md)
- [CROSS_TAB.md](cross-tab-synchronization.md)
- [ERROR_BOUNDARIES.md](error-boundaries.md)
- [INTEROP.md](interop-guidance.md)
- [MULTI_SURFACE.md](multi-surface-coordination.md)
- [OFFLINE_MUTATIONS.md](offline-mutation-queueing.md)
- [PRODUCTION_CORRECTNESS.md](production-correctness.md)
- [SCHEDULING.md](scheduling.md)
- [HYDRATION.md](hydration.md)
- [SERVER_INTEGRATION.md](server-integration.md)
- [OBSERVABILITY.md](observability.md)
- [LOGGING.md](logging.md)
- [PRERENDER.md](prerender.md)
- [ASSETS.md](assets.md)
- [BROWSER_SUPPORT.md](browser-support.md)
- [PWA.md](pwa-and-offline-app-support.md)
- [SECURITY.md](security-compliance-and-governance.md)
- [CONFIGURATION.md](runtime-configuration-and-feature-flags.md)
- [WASM_RELEASES.md](wasm-build-profiles-and-release-engineering.md)
- [BUILD_EXPERIMENTS.md](wasm-build-experiments.md)
- [ONBOARDING.md](onboarding-and-bootstrap.md)
- [STATE_TRANSFER.md](state-transfer.md)
- [STREAMING_SSR.md](streaming-ssr.md)
- [WORKERS.md](worker-guidance.md)
- [ROUTER_AUTH.md](router-async-guards-and-auth-design.md)
- [WALKTHROUGHS.md](end-to-end-walkthroughs.md)
- [REFERENCE_MAP.md](reference-map.md)
- [TROUBLESHOOTING.md](troubleshooting.md)
- [examples/README.md](../examples/README.md)
