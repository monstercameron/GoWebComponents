# Performance Notes

Last updated: 2026-03-14

This file tracks the current performance state of the repo. It intentionally avoids old speculative claims and stale file references.

## Current Focus Areas

The runtime hot paths are:

- `internal/runtime/reconciler.go`
- `internal/runtime/scheduler.go`
- `internal/runtime/hooks.go`
- `internal/runtime/state.go`
- `internal/platform/jsdom/adapters.go`

## Completed Work In The Current Pass

### Correctness fixes that affected performance-sensitive paths

- Fixed function components that return `nil` so they correctly delete previous children.
- Fixed DOM-less subtree deletion so all descendant sibling branches are removed.
- Fixed stale hook ownership on reused/skipped function components.
- Fixed nil resets for `GoUseState` and `GoUseAtom` on nil-able types.
- Fixed scheduler ancestor propagation for already-dirty leaves.
- Fixed lazy global runtime upgrade behavior.

### Optimization work that was kept

- Reconciler `children` comparison fast path.
- Reduced overhead in single-child function-component reconciliation.
- Primitive-first `fastEqual` path in hooks.
- Cached nil-able type detection in state hooks.
- Cached scheduler continuation callback.
- Batched atom unsubscription with `UnsubscribeMany(...)`.
- Direct component helper child creation in `html.go`.
- Specialized `WASMDOMAdapter.WrapFunction` wrappers chosen once at wrap time.
- Work-in-progress fiber reuse through alternates in reconciler and scheduler.

### Optimization work that was reverted

The repo also tried more aggressive reconciler micro-optimizations around props-map creation and DOM update batching. Those changes regressed the benchmark set and were removed.

## Benchmarks

### Native runtime benchmarks

Run:

```bash
go test ./internal/runtime -run ^$ -bench . -benchmem
```

Measured wins kept in the repo:

- `BenchmarkDivWithComponents4`
  - before: `1654 ns/op`, `2088 B/op`, `17 allocs/op`
  - after: `736.9 ns/op`, `744 B/op`, `9 allocs/op`
- `BenchmarkWithComponentsGeneric4`
  - before: `1820 ns/op`, `2104 B/op`, `18 allocs/op`
  - after: `829.0 ns/op`, `760 B/op`, `10 allocs/op`
- `BenchmarkCleanupAtomSubscriptions8`
  - before: `4477 ns/op`
  - after: `3629 ns/op`
- `BenchmarkPropsEqualChildrenDifferentPointer`
  - before: `259.8 ns/op`
  - after: `85.82 ns/op`
- `BenchmarkReconcileChildrenStableList16`
  - earlier pass improvement: `13825 ns/op` -> `4042 ns/op`
- `BenchmarkPerformUnitOfWorkFunctionComponentLeaf`
  - earlier pass improvement: `1991 ns/op` -> `469.9 ns/op`

Some reconciler benchmarks remain noisy because cold microbenchmarks do not model steady-state alternate reuse perfectly. That is why repeated runs and comparison tooling are still a worthwhile next step.

### Wasm adapter benchmarks

Run on Windows:

```powershell
$env:GOOS = "js"
$env:GOARCH = "wasm"
go test -exec .\tools\go_js_wasm_exec.bat ./internal/platform/jsdom -run ^$ -bench . -benchmem
```

Current wasm adapter measurements:

- `BenchmarkWASMDOMAdapterCreateElement`: `103665 ns/op`, `248 B/op`, `24 allocs/op`
- `BenchmarkWASMDOMAdapterSetAttribute`: `51720 ns/op`, `136 B/op`, `13 allocs/op`
- `BenchmarkWASMDOMAdapterAppendChild`: `119989 ns/op`, `344 B/op`, `33 allocs/op`
- `BenchmarkWASMDOMAdapterWrapFunctionNoArgsInvoke`: `14414 ns/op`, `32 B/op`, `3 allocs/op`
- `BenchmarkLegacyWrapFunctionNoArgsInvoke`: `14621 ns/op`, `32 B/op`, `3 allocs/op`
- `BenchmarkWASMDOMAdapterWrapFunctionStringInvoke`: `28284 ns/op`, `80 B/op`, `8 allocs/op`
- `BenchmarkLegacyWrapFunctionStringInvoke`: `29076 ns/op`, `80 B/op`, `8 allocs/op`

The current wrapper specialization is modestly faster than the legacy generic wrapper.

## What To Optimize Next

1. Add repeated benchmark runs and `benchstat`-style comparison to reduce noise.
2. Keep pushing steady-state reconciliation/fiber reuse benchmarks instead of only cold-path microbenchmarks.
3. Profile browser-bound paths separately from native runtime paths.
4. Treat correctness regressions as blockers; performance changes in this repo have repeatedly shown that low-level wins are only worth keeping if the benchmark set and behavior tests both stay green.
