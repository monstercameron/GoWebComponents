# Start Here

This page is the recommended path for new GoWebComponents adopters.

Use it when you want the shortest route from evaluation to a production-shaped app without guessing which docs reflect the current public surface.

## Recommended Adoption Path

1. Read the root [README.md](../README.md) for the current public package surface, build requirements, and repo-level commands.
2. Read [API_POLICY.md](API_POLICY.md) so you know which packages are stable, which surfaces are experimental, and how upgrades are handled.
3. Read [WORKFLOWS.md](WORKFLOWS.md) and pick the workflow that matches your app: client-only, router-based, SSR, testing, production wasm build, or hydration debugging.
4. Use [examples/README.md](../examples/README.md) to jump into the smallest runnable example that demonstrates the feature you need.
5. Use [WALKTHROUGHS.md](WALKTHROUGHS.md) when you want a realistic app shape instead of a single isolated feature demo.
6. Use [TROUBLESHOOTING.md](TROUBLESHOOTING.md) when the wasm build, dev server, hydration flow, or router setup is not behaving as expected.

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
- SSR or hydration: `examples/70-render-to-string`, `examples/71-hydrate`, `examples/72-router-hydrate-mount`, `examples/18-ssr-server-routing`, `examples/87-ssr-secure-forms`
- Debugging and diagnostics: `examples/66-devtools-panel`, `examples/69-devtools-diagnostics`, `examples/86-atlas-commerce-os`

## Production-Oriented Defaults

If you are starting a new application today, these defaults are the least surprising path:

- Start with `ui` and `html` for browser rendering.
- Add `router` only once you need multiple pages, params, loaders, or route metadata.
- Add `state` for shared state instead of building custom registries.
- Use `fetch.UseResource[T]` for typed async data and `fetch.UseFetch` only when you need lower-level request control.
- Add SSR only when your app needs request-time HTML, route-aware bootstrapping, or hydration reuse.
- Validate production flows with both Go tests and Playwright browser tests.

## Related Docs

- [README.md](../README.md)
- [WORKFLOWS.md](WORKFLOWS.md)
- [WALKTHROUGHS.md](WALKTHROUGHS.md)
- [REFERENCE_MAP.md](REFERENCE_MAP.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- [examples/README.md](../examples/README.md)
