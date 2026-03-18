# GoWebComponents Documentation

This directory contains the project-level documentation that is still useful after the runtime and tooling cleanup on 2026-03-14.

## Files

### `START_HERE.md`
Recommended entrypoint for new adopters, including the preferred package surface, first example picks, and the modern path through the docs.

### `WORKFLOWS.md`
Task-oriented guidance for common developer jobs such as building a client-only app, adding routing, adding SSR, testing, shipping a wasm build, and debugging hydration.

### `WALKTHROUGHS.md`
End-to-end app-shape walkthroughs that connect the current examples and package surface into realistic starting paths.

### `REFERENCE_MAP.md`
Cross-links for concepts, public APIs, examples, production caveats, and related docs.

### `TROUBLESHOOTING.md`
Common setup and runtime failure guidance for wasm builds, `wasm_exec.js`, example serving, hydration, routing, and browser interop.

### `FORMS.md`
Current supported `ui.UseForm` modes, server-post conventions, redirect-after-submit guidance, and the boundary between shipped form state helpers and application-owned server workflows.

### `README.md`
High-level documentation index and pointers to the current runtime, examples, tests, and tools.

### `PERFORMANCE.md`
Current performance notes, benchmark entrypoints, and the results that were actually measured in the current codebase.

### `API_POLICY.md`
Stability tiers, semver rules, deprecation lifecycle, migration requirements, and latest-major support expectations.

### `MIGRATIONS.md`
Release-to-release upgrade guidance, starting with the transition into the current `v3.x` public package layout.

### `HEAD_MANAGEMENT.md`
Current head-management, SEO, canonical URL, structured-data, and resource-hint guidance, including the boundary between router-managed metadata and app-owned explicit head markup.

### `ACCESSIBILITY.md`
Current accessibility support baseline, including typed semantic markup, `ui.UseId()`, application-owned accessibility responsibilities, and current non-goals.

### `OVERLAYS.md`
Current overlay layering model, stack coordination rules, anchored-position guidance, and the boundary between framework-owned overlay behavior and application-owned placement logic.

### `I18N.md`
Current internationalization scope, locale context model, message catalog helpers, SSR bootstrap transfer, locale-aware routing guidance, and RTL directionality guidance.

### `TODO.md`
Current backlog and near-term work. This is a live backlog, not a historical archive of every idea the project has ever had.

## Where The Core Lives

## Start Here

If you are new to the repo, use this order:

1. [START_HERE.md](START_HERE.md)
2. [WORKFLOWS.md](WORKFLOWS.md)
3. [WALKTHROUGHS.md](WALKTHROUGHS.md)
4. [REFERENCE_MAP.md](REFERENCE_MAP.md)
5. [TROUBLESHOOTING.md](TROUBLESHOOTING.md)

The old docs referred to a `/fiber` directory. That is stale.

The current implementation core is:

- `internal/runtime/`
- `internal/platform/jsdom/`
- `dom/`, `hooks/`, `render/`, `state/`, `fetch/`, `router/`

If you are debugging the framework itself, start here:

1. `internal/runtime/types.go`
2. `internal/runtime/reconciler.go`
3. `internal/runtime/scheduler.go`
4. `internal/runtime/hooks.go`
5. `internal/runtime/runtime.go`

## Current Validation Status

As of 2026-03-14:

- `go test ./internal/runtime` passes
- Native `internal/runtime` statement coverage is `100%`
- Playwright component, integration, and deep state stress suites pass
- Separate `js/wasm` tests and benchmarks exist for wasm-only runtime and adapter code

## Related Docs

- [../README.md](../README.md)
- [../CHANGELOG.md](../CHANGELOG.md)
- [START_HERE.md](START_HERE.md)
- [WORKFLOWS.md](WORKFLOWS.md)
- [WALKTHROUGHS.md](WALKTHROUGHS.md)
- [REFERENCE_MAP.md](REFERENCE_MAP.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- [FORMS.md](FORMS.md)
- [API_POLICY.md](API_POLICY.md)
- [ACCESSIBILITY.md](ACCESSIBILITY.md)
- [OVERLAYS.md](OVERLAYS.md)
- [I18N.md](I18N.md)
- [HEAD_MANAGEMENT.md](HEAD_MANAGEMENT.md)
- [MIGRATIONS.md](MIGRATIONS.md)
- [../examples/README.md](../examples/README.md)
- [../test/README.md](../test/README.md)
- [../tools/README.md](../tools/README.md)
