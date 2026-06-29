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
- `runtime_controls.go`, `memory_hygiene.go`: priority lanes, strict mode, deterministic replay, bounded state, and long-session memory hygiene
- `panic_*.go`, `profiling.go`, `hot_reload.go`: panic reporting, profiling, and hot-reload support
- `*_test.go` and `*_benchmark_test.go`: contract, regression, and performance coverage

## How To Navigate Changes

- Start with `runtime.go` and `scheduler.go` for render lifecycle issues.
- Start with `runtime_controls.go` for lane scheduling, strict-mode probes, replay capture/replay, and runtime limits.
- Start with the `reconciler*` files for DOM shape, fiber identity, or commit bugs.
- Start with `hooks.go` plus the nearest focused hook file for state or effect behavior.
- Start with `hydration.go` or `ssr.go` for server-rendered flows.

The public entrypoint remains `ui/`; this folder is implementation detail for maintainers.

## Telemetry And CSP Notes

- Panic reports are redacted through `logging.ConfigureTelemetryRedaction` before console emission or `OnReport` callbacks.
- SSR streaming accepts a per-request script nonce and applies it to emitted boundary patch scripts.

## Runtime Controls

- Use `NewRuntime` for each mounted root that needs isolated atom state, scheduling, IDs, and event wrappers.
- Use `Config.StrictMode` for dev-time double-render, setState-during-render, and effect cleanup-contract checks.
- Use `Config.Limits` to bound pending effects, queued update coalescing, replay events, and diagnostics.
- Use `StartReplayRecording` / `StopReplayRecording` and `ReplayUpdates` to capture and replay deterministic scheduling streams.
- Use `InternalStateSnapshot` and `CheckMemoryHygiene` for long-session fiber, atom, subscriber, queue, and heap-pressure diagnostics.
