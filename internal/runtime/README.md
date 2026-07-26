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

## Known Scope Limits

Recorded here rather than left as an unstated gap between what the plan claims
and what the code does.

- **`schedulerMu` is package-level, not per-runtime.** P2.3 scoped runtime state
  to `internal/runtime` and moved the ready signal, atom registry, and diagnostic
  limits onto the `Runtime`. The scheduler mutex did not move, so two runtimes in
  one process serialize every `Schedule*` call against each other. This is
  invisible today: §7 of the v5 plan declares multi-runtime support a non-goal
  across the public surface, and a green P2.3 suite means "two renderers with
  independent state and atoms", not "N runtimes rendering concurrently".

  What it would take: `schedulerMu` is taken by every scheduling entry point,
  by `commitRoot`'s state transition, and by the profiling recorder, so moving it
  onto the `Runtime` is mechanical but wide. It should be done with a measurement
  attached — an uncontended mutex is nearly free, so the change buys nothing
  until there is a second runtime actually rendering, and that is the thing §7
  says is out of scope. Do not do it as cleanup; do it when a two-runtime
  workload exists to justify it.

- **Inline closure props defeat memoization by design.** `sameFunctionIdentity`
  compares the code pointer AND the funcval data pointer, so two allocations of
  the same closure literal are unequal and the owning fiber is dirty every
  render. That is correct — the alternative is stale closures — but it means a
  component passing `func() { ... }` inline as a prop re-renders and re-writes
  that handler to the DOM on every commit, and `SetProperty` is not covered by
  the attribute batch. Use `ui.UseEvent`, whose wrapper is stable across renders
  and therefore compares equal and is skipped. See the note on `ui.WrapHandler`.

## Telemetry And CSP Notes

- Panic reports are redacted through `logging.ConfigureTelemetryRedaction` before console emission or `OnReport` callbacks.
- SSR streaming accepts a per-request script nonce and applies it to emitted boundary patch scripts.

## Runtime Controls

- Use `NewRuntime` for each mounted root that needs isolated atom state, scheduling, IDs, and event wrappers.
- Use `Config.StrictMode` for dev-time double-render, setState-during-render, and effect cleanup-contract checks.
- Use `Config.Limits` to bound pending effects, queued update coalescing, replay events, and diagnostics.
- Use `StartReplayRecording` / `StopReplayRecording` and `ReplayUpdates` to capture and replay deterministic scheduling streams.
- Use `InternalStateSnapshot` and `CheckMemoryHygiene` for long-session fiber, atom,
  subscriber, queue, goroutine, and heap-pressure diagnostics. **Neither is free, and
  neither belongs in a per-frame poll.** `InternalStateSnapshot` walks the whole
  committed fiber tree while holding `schedulerMu`, so it blocks scheduling for
  O(tree) per sample, and `CheckMemoryHygiene` additionally calls
  `runtime.ReadMemStats`, which stops the world. Sampling either on a timer
  manufactures the pauses M7 exists to bound. Call them on demand, or on a slow
  interval when investigating.
- Set `MemoryHygieneOptions.MaxGoroutines` when investigating a leak: the heap and
  fiber thresholds cannot see a goroutine parked on a channel, which is how the
  suspended-boundary watcher leak (fixed in `ee39ac99`) stayed invisible.
