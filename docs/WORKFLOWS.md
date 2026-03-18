# Common Workflows

This page reorganizes the documentation around common developer tasks instead of package-by-package reading order.

Use it as the main task-oriented index for the current repo.

## Build A Client-Only App

Use this path when the app is mounted in the browser and does not need request-time HTML.

Public API:

- `ui.Render(...)`
- `ui.UseState`, `ui.UseEffect`, `ui.UseReducer`, `ui.UseEvent`
- `html.*` builders

Runnable examples:

- [examples/21-ui-render](../examples/21-ui-render)
- [examples/22-create-element](../examples/22-create-element)
- [examples/75-use-state](../examples/75-use-state)
- [examples/76-use-effect](../examples/76-use-effect)

Production caveats:

- Ship the generated wasm binary together with the matching `wasm_exec.js` from your active Go toolchain.
- Keep browser-only APIs behind browser execution paths; native `go test` does not provide `syscall/js`.

Testing and validation:

- [test/README.md](../test/README.md)
- [tools/README.md](../tools/README.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md#wasm-build-failures)

## Add Routing

Use this path when the app needs multiple pages, route params, loaders, redirects, metadata, or nested layouts.

Public API:

- `router.NewHashRouter`, `router.NewRouter`
- `Register`, `Mount`, `router.Outlet()`
- `router.UseNavigate`, `router.UseParams`, `router.UseQuery`, `router.UseSearchParams`, `router.UseRevalidator`

Runnable examples:

- [examples/55-hash-router](../examples/55-hash-router)
- [examples/56-browser-router](../examples/56-browser-router)
- [examples/60-route-loaders](../examples/60-route-loaders)
- [examples/64-nested-layout-routes](../examples/64-nested-layout-routes)
- [examples/65-router-guards](../examples/65-router-guards)

Production caveats:

- Browser-router deployments need server rewrites so deep links resolve to the app shell.
- Keep route registration on canonical leading-slash paths.
- Route factories that return elements should return `ui.CreateElement(...)`; do not call hooks directly in the factory itself.

Testing and validation:

- [router/README.md](../router/README.md)
- [examples/README.md](../examples/README.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md#route-misconfiguration)

## Add SSR And Hydration

Use this path when the app needs request-time HTML, route-aware bootstrapping, or hydration reuse.

Public API:

- `ui.RenderToString(...)`
- `ui.Hydrate(...)`
- `router.HydrateMount(...)`
- bootstrap helpers documented through the SSR examples and migration guide

Runnable examples:

- [examples/70-render-to-string](../examples/70-render-to-string)
- [examples/71-hydrate](../examples/71-hydrate)
- [examples/72-router-hydrate-mount](../examples/72-router-hydrate-mount)
- [examples/73-ssr-bootstrap](../examples/73-ssr-bootstrap)
- [examples/18-ssr-server-routing](../examples/18-ssr-server-routing)

Production caveats:

- Treat hydration as DOM reuse over markup that must structurally match the server output.
- Reuse the same bootstrap payload rules across server render and browser resume.
- Verify route data, IDs, and metadata behavior after hydration, not just after client-side navigation.

Testing and validation:

- [MIGRATIONS.md](MIGRATIONS.md)
- [REFERENCE_MAP.md](REFERENCE_MAP.md#ssr-and-hydration)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md#hydration-mismatch-warnings)

## Test A Component Or App Flow

Use this path when validating runtime behavior, wasm behavior, or browser flows.

Public API and tools:

- native Go tests for framework or app packages
- `go test -exec .\tools\go_js_wasm_exec.bat ...` for Windows `js/wasm` coverage
- Playwright suites under `test/` and `examples/`

Runnable examples and suites:

- [test/README.md](../test/README.md)
- [examples/README.md](../examples/README.md)

Production caveats:

- Browser-only behavior should have at least one browser or `js/wasm` check.
- SSR apps should include hydration checks, not just server-render snapshots.

Testing and validation:

- [tools/README.md](../tools/README.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md#broken-example-serving)

## Ship A Production Wasm Build

Use this path when turning a local app into a deployable wasm bundle.

Public API and tools:

- normal `go build` with `GOOS=js` and `GOARCH=wasm`
- the matching `wasm_exec.js` from `$(go env GOROOT)/lib/wasm/`
- the repo's example server or your own static server for local validation

Runnable examples:

- [examples/01-counter](../examples/01-counter)
- [examples/70-render-to-string](../examples/70-render-to-string)
- [examples/18-ssr-server-routing](../examples/18-ssr-server-routing)

Production caveats:

- Always ship the wasm binary and `wasm_exec.js` from the same Go toolchain.
- Serve `.wasm` with the correct MIME type.
- Validate cache behavior for generated assets and rollout timing for updated binaries.

Testing and validation:

- [tools/README.md](../tools/README.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md#missing-wasm_execjs)

## Debug Hydration Issues

Use this path when the server-rendered app boots but the resumed app falls back, logs mismatches, or behaves differently after the client takes over.

Public API:

- `ui.Hydrate(...)`
- `router.HydrateMount(...)`
- devtools snapshots and diagnostics when the issue needs runtime visibility

Runnable examples:

- [examples/71-hydrate](../examples/71-hydrate)
- [examples/72-router-hydrate-mount](../examples/72-router-hydrate-mount)
- [examples/74-ssr-route-data-reuse](../examples/74-ssr-route-data-reuse)
- [examples/69-devtools-diagnostics](../examples/69-devtools-diagnostics)

Production caveats:

- Check the exact server HTML the client sees before the first hydration pass.
- Verify route data reuse, IDs, query parsing, and conditional branches that depend on browser-only state.

Testing and validation:

- [TROUBLESHOOTING.md](TROUBLESHOOTING.md#hydration-mismatch-warnings)
- [REFERENCE_MAP.md](REFERENCE_MAP.md#diagnostics-and-debugging)

## Related Docs

- [START_HERE.md](START_HERE.md)
- [WALKTHROUGHS.md](WALKTHROUGHS.md)
- [REFERENCE_MAP.md](REFERENCE_MAP.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md)