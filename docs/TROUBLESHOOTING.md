# Troubleshooting

This page collects the most common setup and runtime failures currently visible in the repo's documented workflows.

## Wasm Build Failures

Symptoms:

- `go build` fails when targeting `js/wasm`
- the output `.wasm` file is missing
- browser tests fail because the expected wasm artifact was not rebuilt

Checks:

- confirm `GOOS=js` and `GOARCH=wasm` are set for the build command
- confirm the output directory exists and the target package builds on its own
- on Windows, use the repo's wrapper commands or PowerShell environment assignments consistently in the same shell session

Useful references:

- [tools/README.md](../tools/README.md)
- [examples/README.md](../examples/README.md)

## Missing `wasm_exec.js`

Symptoms:

- the page loads but Go never starts
- the browser console reports `Go is not defined`
- the wasm binary exists but the runtime bootstrap script is missing or mismatched

Checks:

- copy `wasm_exec.js` from the same Go toolchain that built the wasm binary
- confirm the HTML page points at the served script path that actually exists
- avoid mixing a newly built wasm binary with an older `wasm_exec.js`

Useful references:

- [WORKFLOWS.md](WORKFLOWS.md#ship-a-production-wasm-build)
- [BROWSER_SUPPORT.md](BROWSER_SUPPORT.md)
- [README.md](../README.md)

## Broken Example Serving

Symptoms:

- `/examples` does not load
- example HTML loads but shared assets or wasm binaries 404
- Playwright example runs hit the wrong server or stale routes

Checks:

- start the documented Express server with `npm --prefix tools/devtools run dev:examples`
- confirm the expected port is free before starting another dev server
- check that the requested example path exists under `examples/`
- verify generated wasm output is present under `bin/examples/` when the page expects it

Useful references:

- [examples/README.md](../examples/README.md)
- [tools/README.md](../tools/README.md)
- [test/README.md](../test/README.md)

## Hydration Mismatch Warnings

Symptoms:

- the SSR page renders, but hydration falls back or logs mismatch diagnostics
- the server HTML looks correct, but the resumed app diverges after startup

Checks:

- compare the server HTML against the first client render inputs, especially route data, generated IDs, and environment-dependent conditionals
- verify bootstrap payload reuse matches the route and component tree being hydrated
- inspect metadata, params, and query-derived branches that may differ between server and browser phases
- reduce the page to the smallest SSR example that still reproduces the mismatch

Useful references:

- [WORKFLOWS.md](WORKFLOWS.md#debug-hydration-issues)
- [WALKTHROUGHS.md](WALKTHROUGHS.md#server-rendered-app)
- [examples/71-hydrate](../examples/71-hydrate)
- [examples/18-ssr-server-routing](../examples/18-ssr-server-routing)

## Route Misconfiguration

Symptoms:

- direct links fail under browser routing
- registered routes never match
- layout or guard behavior works in one path but not another

Checks:

- register canonical leading-slash paths
- confirm the chosen router matches the deployment mode: hash router for static-only hosts, browser router when rewrites are available
- keep hooks inside component functions, not route-factory helpers that only return elements
- verify redirects, loaders, and metadata are attached to the intended route entry

Useful references:

- [router/README.md](../router/README.md)
- [examples/55-hash-router](../examples/55-hash-router)
- [examples/56-browser-router](../examples/56-browser-router)

## Interop Mistakes

Symptoms:

- browser-only code crashes in native tests
- direct `syscall/js` usage leaks into app code without cleanup or guards
- third-party browser APIs work during local experimentation but break during SSR or hydration

Checks:

- keep browser interop behind browser execution paths and test it under `js/wasm`
- isolate direct DOM or browser API usage behind component effects or explicit event flows
- avoid assuming browser globals exist during native SSR execution

Useful references:

- [WORKFLOWS.md](WORKFLOWS.md#test-a-component-or-app-flow)
- [BROWSER_SUPPORT.md](BROWSER_SUPPORT.md)
- [MIGRATIONS.md](MIGRATIONS.md)

## Related Docs

- [ACTIONABLE_ERRORS.md](ACTIONABLE_ERRORS.md)
- [START_HERE.md](START_HERE.md)
- [WORKFLOWS.md](WORKFLOWS.md)
- [WALKTHROUGHS.md](WALKTHROUGHS.md)
- [REFERENCE_MAP.md](REFERENCE_MAP.md)
