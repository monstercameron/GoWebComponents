# Internal Runtime

Location: `internal/runtime/`

This is the default single-threaded renderer and hook runtime behind the public `ui` package.

## File Layout

- `runtime.go`: runtime construction, render entrypoints, and adapter wiring
- `scheduler.go`: update scheduling and transition behavior
- `reconciler.go`, `reconciler_elements.go`, `reconciler_commit.go`: fiber reconciliation and DOM commit flow
- `hooks.go`, `hooks_fetch.go`, `state.go`, `transition.go`: hook behavior and reactive state plumbing
- `events.go`: event adaptation and typed event helpers
- `hydration.go`, `ssr.go`: hydration and server-rendering paths
- `inspect.go`, `inspect_reporting.go`, `diagnostic_metadata.go`, `strict_diagnostics.go`: inspection and diagnostics support
- `panic_*.go`, `profiling.go`, `hot_reload.go`: panic reporting, profiling, and hot-reload support
- `*_test.go` and `*_benchmark_test.go`: contract, regression, and performance coverage

## How To Navigate Changes

- Start with `runtime.go` and `scheduler.go` for render lifecycle issues.
- Start with the `reconciler*` files for DOM shape, fiber identity, or commit bugs.
- Start with `hooks.go` plus the nearest focused hook file for state or effect behavior.
- Start with `hydration.go` or `ssr.go` for server-rendered flows.

The public entrypoint remains `ui/`; this folder is implementation detail for maintainers.
