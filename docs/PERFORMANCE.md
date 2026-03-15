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
- Comparable-type fast path and extra primitive coverage in `fastEqual`.
- Cached nil-able type detection shared by `GoUseState` and `GoUseAtom`.
- Cached scheduler continuation callback.
- Batched atom unsubscription with `UnsubscribeMany(...)`.
- Direct component helper child creation in `html.go`.
- Lower-overhead `CreateElement(...)` prop map sizing and empty-children reuse.
- Lower-overhead `propsEqual(...)` fallback for non-standard children slices.
- Lower-churn initial `updateDomProperties(...)` batching path.
- Expanded microbenchmark coverage for callback/ref/id/func hooks, atom-registry operations, and commit/effect traversal.
- Pooled atom subscriber notification path for `GoUseAtom(...)` updates.
- Sibling placement batching in `commitWork(...)` when the DOM adapter exposes batch hooks.
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

Current measurements from the latest Windows arm64 pass:

- `BenchmarkFastEqualInt`: `3.824 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkFastEqualString`: `5.775 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkAreDepsEqual3Primitives`: `17.11 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkDivWithComponents4`: `598.8 ns/op`, `744 B/op`, `9 allocs/op`
- `BenchmarkWithComponentsGeneric4`: `548.6 ns/op`, `760 B/op`, `10 allocs/op`
- `BenchmarkCreateElementHostWithTextChildren`: `563.5 ns/op`, `664 B/op`, `8 allocs/op`
- `BenchmarkPropsEqualChildrenDifferentPointer`: `91.56 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkReconcileChildrenStableList16`: `3625 ns/op`, `2817 B/op`, `16 allocs/op`
- `BenchmarkUpdateDomPropertiesInitialRender`: `1110 ns/op`, `1120 B/op`, `7 allocs/op`

Additional microbenchmarks from the same Windows arm64 pass:

- `BenchmarkIsNilableTypeCachedPointer`: `16.13 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkGoUseCallbackSameDeps`: `32.68 ns/op`, `16 B/op`, `1 allocs/op`
- `BenchmarkGoUseRefStable`: `14.30 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkGoUseIdStable`: `12.90 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkGoUseFuncWrap`: `93.76 ns/op`, `8 B/op`, `1 allocs/op`
- `BenchmarkCommitWorkPlacementChain16`: `565.4 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkRunEffectsChain16`: `157.2 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkAtomRegistryInitAtomExisting`: `117.4 ns/op`, `7 B/op`, `0 allocs/op`
- `BenchmarkAtomRegistryGetAtom`: `50.41 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkAtomRegistrySubscribeUnsubscribe`: `659.4 ns/op`, `192 B/op`, `2 allocs/op`
- `BenchmarkGoUseAtomGetter`: `54.35 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkAtomRegistryUnsubscribeMany8`: `4453 ns/op`, `1536 B/op`, `16 allocs/op`

Follow-up native runtime pass after atom fan-out audit:

- `BenchmarkGoUseAtomIntUpdate`: `614.5 ns/op`, `232 B/op`, `5 allocs/op`
- `BenchmarkGoUseAtomStableRerender`: `146.3 ns/op`, `112 B/op`, `2 allocs/op`
- `BenchmarkCommitWorkPlacementChain16`: `307.3 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkRunEffectsChain16`: `96.48 ns/op`, `0 B/op`, `0 allocs/op`

Follow-up commit batching pass:

- `BenchmarkCommitWorkPlacementChain16`: `311.5 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkRunEffectsChain16`: `140.7 ns/op`, `0 B/op`, `0 allocs/op`

The native benchmark change is effectively flat; the reason to keep this commit batching path is to enable lower browser-side append churn for adapters like `internal/platform/jsdom`, not to chase native microbenchmark wins.

Commit traversal was audited but not changed further in this pass; the benchmark cost is already low relative to atom fan-out and browser-bound DOM work.

Some reconciler benchmarks remain noisy because cold microbenchmarks do not model steady-state alternate reuse perfectly. That is why repeated runs and comparison tooling are still a worthwhile next step.

### Wasm adapter benchmarks

Run on Windows:

```powershell
$env:GOOS = "js"
$env:GOARCH = "wasm"
go test -exec .\tools\go_js_wasm_exec.bat ./internal/platform/jsdom -run ^$ -bench . -benchmem
```

For lower-noise repeated runs and saved outputs:

```powershell
.\tools\bench-runtime.ps1 -Package ./internal/platform/jsdom -Count 5 -Exec .\tools\go_js_wasm_exec.bat
```

Compare two saved runs when `benchstat` is installed:

```powershell
.\tools\bench-compare.ps1 -Baseline .\tools\bench-before.txt -Candidate .\tools\bench-after.txt
```

Current wasm adapter measurements:

- `BenchmarkWASMDOMAdapterCreateElement`: `103665 ns/op`, `248 B/op`, `24 allocs/op`
- `BenchmarkWASMDOMAdapterSetAttribute`: `51720 ns/op`, `136 B/op`, `13 allocs/op`
- `BenchmarkWASMDOMAdapterAppendChild`: `119989 ns/op`, `344 B/op`, `33 allocs/op`
- `BenchmarkWASMDOMAdapterWrapFunctionNoArgsInvoke`: `14414 ns/op`, `32 B/op`, `3 allocs/op`
- `BenchmarkLegacyWrapFunctionNoArgsInvoke`: `14621 ns/op`, `32 B/op`, `3 allocs/op`
- `BenchmarkWASMDOMAdapterWrapFunctionStringInvoke`: `28284 ns/op`, `80 B/op`, `8 allocs/op`
- `BenchmarkLegacyWrapFunctionStringInvoke`: `29076 ns/op`, `80 B/op`, `8 allocs/op`

Latest profiling-oriented wasm adapter pass added these boundary-focused benchmarks:

- `BenchmarkWASMDOMAdapterQuerySelector`: `227259 ns/op`, `240 B/op`, `23 allocs/op`
- `BenchmarkWASMDOMAdapterSetPropertyString`: `14081 ns/op`, `8 B/op`, `1 allocs/op`
- `BenchmarkWASMDOMAdapterBatchAppend16`: `2290805 ns/op`, `1720 B/op`, `161 allocs/op`
- `BenchmarkWASMEventAdapterAddRemoveListener`: `197265 ns/op`, `192 B/op`, `16 allocs/op`

The current wasm profile says the expensive browser crossings are still DOM tree mutation, selector queries, and event listener registration/removal. Property sets are comparatively cheap.

Follow-up jsdom fragment reuse pass:

- `BenchmarkWASMDOMAdapterBatchAppend16`: `1697474 ns/op`, `1640 B/op`, `153 allocs/op`
- `BenchmarkWASMDOMAdapterAppendChild`: `337387 ns/op`, `344 B/op`, `33 allocs/op`
- `BenchmarkWASMDOMAdapterQuerySelector`: `211698 ns/op`, `240 B/op`, `23 allocs/op`
- `BenchmarkWASMEventAdapterAddRemoveListener`: `229812 ns/op`, `192 B/op`, `16 allocs/op`

Reusing the same `DocumentFragment` across commit batches is a real win for the batched append path. Event listener churn still dominates its own path and needs a different optimization strategy.

The current wrapper specialization is modestly faster than the legacy generic wrapper.

## What To Optimize Next

1. Add repeated benchmark runs and `benchstat`-style comparison to reduce noise.
2. Keep pushing steady-state reconciliation/fiber reuse benchmarks instead of only cold-path microbenchmarks.
3. Profile browser-bound paths separately from native runtime paths.
4. Treat correctness regressions as blockers; performance changes in this repo have repeatedly shown that low-level wins are only worth keeping if the benchmark set and behavior tests both stay green.
