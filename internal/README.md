# Internal Package

Location: `internal/`

This directory contains the framework implementation. It is not the user-facing API surface.

## Layout

### `internal/runtime/`

Core runtime implementation:

- `types.go`: core runtime data structures
- `reconciler.go`: tree diffing, child reconciliation, commit preparation
- `scheduler.go`: update scheduling and work loop entrypoints
- `hooks.go`: local state, effects, memoization, refs, IDs, callback wrappers
- `state.go`: shared atom registry and subscriptions
- `runtime.go`: runtime bootstrap and global runtime wiring
- `events.go`: Go event wrapper behavior
- `html.go`: helper element constructors used internally
- `interfaces.go`: runtime/platform contracts
- `shim.go`: wasm-facing helper wrappers
- `hooks_fetch.go`: wasm fetch hook path
- `hooks_fetch_stub.go`: native stub for fetch hook compatibility

### `internal/platform/jsdom/`

Browser adapters for `js/wasm`.

### `internal/platform/mockdom/`

Mock adapters used by native tests.

## Recommended Debugging Order

If you are debugging runtime behavior, start here:

1. `internal/runtime/types.go`
2. `internal/runtime/reconciler.go`
3. `internal/runtime/scheduler.go`
4. `internal/runtime/hooks.go`
5. `internal/runtime/runtime.go`

## Current Validation

As of 2026-03-14:

- `go test ./internal/runtime` passes
- native `internal/runtime` statement coverage is `100%`
- runtime microbenchmarks exist across reconciler, hooks, scheduler, state, runtime, html, shim, and types
- separate `js/wasm` tests and benchmarks exist for wasm-only runtime paths and browser adapters

## Contributor Notes

- Runtime changes should come with behavior-focused tests first, not only coverage-oriented tests.
- Performance changes should be benchmarked before and after.
- Browser-bound code in `internal/platform/jsdom/` should be profiled separately from native runtime code.
