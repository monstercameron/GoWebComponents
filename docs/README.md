# GoWebComponents Documentation

This directory contains the project-level documentation that is still useful after the runtime and tooling cleanup on 2026-03-14.

## Files

### `README.md`
High-level documentation index and pointers to the current runtime, examples, tests, and tools.

### `PERFORMANCE.md`
Current performance notes, benchmark entrypoints, and the results that were actually measured in the current codebase.

### `TODO.md`
Current backlog and near-term work. This is a live backlog, not a historical archive of every idea the project has ever had.

## Where The Core Lives

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
- [../examples/README.md](../examples/README.md)
- [../test/README.md](../test/README.md)
- [../tools/README.md](../tools/README.md)
