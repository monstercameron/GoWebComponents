# Troubleshooting

This page collects the most common setup and runtime failures currently visible in the repo's documented workflows.

## Current Status

Shipped today:

- structured runtime diagnostics with stable metadata fields such as code, docs, remediation, and recoverable classification
- actionable-errors guidance that maps several common runtime failures to canonical remediation anchors
- launcher checks through `go run ./tools/gwc doctor` and lane-oriented validation through `go run ./tools/gwc test -lane ...`
- hydration, SSR, and browser troubleshooting guidance spread across the SSR, security, logging, observability, and workflows docs

Not shipped today:

- one universal auto-fix workflow for every runtime or browser failure
- complete stable error-code coverage across every framework misuse surface

Use this page as the fast triage entrypoint. When a failure already has a stable code or docs anchor, follow that structured path first instead of debugging from raw symptoms alone.

## Fast Triage Order

Use this order before changing code blindly:

1. read the actual error, diagnostic, or browser-console output and keep the original message intact
2. check whether the failure already carries a `code:`, `docs:`, `next:`, or remediation hint
3. reduce the problem to the smallest relevant path: native Go test, js/wasm package test, hydration smoke check, or focused Playwright spec
4. confirm whether the failure is build-time, SSR-time, hydration-time, router-time, or browser-only
5. only then widen into larger workflows such as full example runs or aggregated browser suites

Useful first commands:

- `go run ./tools/gwc doctor`
- `go run ./tools/gwc test -lane unit -lane wasm`
- `go run ./tools/gwc test -lane hydration`
- `go run ./tools/gwc test -lane browser`

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

If the build problem appears only in launcher-driven or mixed workflows, also verify no stale `GOOS` or `GOARCH` values leaked into the current shell before running native tools again.

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

- [WORKFLOWS.md](common-workflows.md#ship-a-production-wasm-build)
- [BROWSER_SUPPORT.md](browser-support.md)
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
- check whether the warning already carries a stable code, docs anchor, component stack, or fiber path
- run a focused hydration-oriented test lane before widening to whole-app browser automation

Useful references:

- [WORKFLOWS.md](common-workflows.md#debug-hydration-issues)
- [WALKTHROUGHS.md](end-to-end-walkthroughs.md#server-rendered-app)
- [ACTIONABLE_ERRORS.md](actionable-errors-and-diagnostics.md)
- [examples/71-hydrate](../examples/71-hydrate)
- [examples/18-ssr-server-routing](../examples/18-ssr-server-routing)

## Structured Diagnostic Output

Many current failures now provide more than a plain message.

When available, prioritize these fields:

- `code:` stable identifier for search, CI logs, or repeated failure matching
- `where:` subsystem, route, component, or handler location
- `path:` fiber or component ancestry when the runtime can attribute the failing subtree
- `next:` first remediation step rather than a generic warning
- `docs:` canonical documentation anchor for the failure family

If the failure already contains those fields, prefer that path over ad hoc repo-wide searching.

Useful references:

- [ACTIONABLE_ERRORS.md](actionable-errors-and-diagnostics.md)
- [LOGGING.md](logging.md)
- [OBSERVABILITY.md](observability.md)

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

- [WORKFLOWS.md](common-workflows.md#test-a-component-or-app-flow)
- [BROWSER_SUPPORT.md](browser-support.md)
- [MIGRATIONS.md](migration-guide.md)

## Browser Test And Dev-Server Confusion

Symptoms:

- Playwright hits the wrong server, wrong port, or stale assets
- example browser tests pass locally once, then fail against older wasm output
- a focused test works, but the aggregated browser suite fails to find the expected route or fixture

Checks:

- confirm which workspace owns the failing browser suite: `test/` or `examples/`
- confirm the expected dev server or static server is the one actually running on the documented port
- rebuild the expected wasm artifact when the browser suite depends on generated output under `bin/`
- prefer the documented focused Playwright command before rerunning the entire aggregated browser matrix

Useful references:

- [test/README.md](../test/README.md)
- [examples/README.md](../examples/README.md)
- [tools/README.md](../tools/README.md)

## Related Docs

- [ACTIONABLE_ERRORS.md](actionable-errors-and-diagnostics.md)
- [START_HERE.md](start-here.md)
- [WORKFLOWS.md](common-workflows.md)
- [WALKTHROUGHS.md](end-to-end-walkthroughs.md)
- [REFERENCE_MAP.md](reference-map.md)
