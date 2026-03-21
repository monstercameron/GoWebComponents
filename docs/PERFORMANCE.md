# Performance Notes

Last updated: 2026-03-15

This file tracks the current performance state of the repo. It intentionally avoids old speculative claims and stale file references.

## Current Focus Areas

The runtime hot paths are:

- `internal/runtime/reconciler.go`
- `internal/runtime/scheduler.go`
- `internal/runtime/hooks.go`
- `internal/runtime/state.go`
- `internal/platform/jsdom/adapters.go`

## Latest Validation Pass

Validated on 2026-03-15 after the router wasm test-harness fix.

Passing test lanes:

- Native tests: `go test ./internal/runtime`
- Wasm package tests: `go test -exec .\\tools\\go_js_wasm_exec.bat ./fetch ./html ./state ./ui ./devtools ./router`
- Wasm jsdom tests: `go test -exec .\\tools\\go_js_wasm_exec.bat ./internal/platform/jsdom`
- Example browser specs:
  - `examples/tests/02-text-input.spec.ts`
  - `examples/tests/07-goroutines.spec.ts`
  - `examples/tests/10-advanced-form.spec.ts`
  - `examples/tests/16-devtools.spec.ts`

The router wasm suite originally hung in the browser-history metadata tests because the mock browser helpers called nested JS string methods from inside `js.FuncOf(...)` callbacks. Replacing those with Go-side `strings.Index(...)` parsing in the test helper restored stable router wasm test execution.

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
- Direct-value atom registry storage with reduced `GoUseAtom(...)` init churn.
- Cached `GoUseAtom(...)` getter/setter accessors per hook slot.
- Single-subscriber fast path in atom notifications.
- Sibling placement batching in `commitWork(...)` when the DOM adapter exposes batch hooks.
- Specialized `WASMDOMAdapter.WrapFunction` wrappers chosen once at wrap time.
- Build-specific `GoUseFunc(...)` validation fast path for common signatures.
- Work-in-progress fiber reuse through alternates in reconciler and scheduler.
- Comparable-key fast path in keyed reconciliation.
- Pooled keyed reconciliation scratch maps/slices and destructive consumption of matched keyed children.
- Cached jsdom document query methods and direct indexed access for DOM collections.

### Optimization work that was reverted

The repo also tried more aggressive reconciler micro-optimizations around props-map creation and DOM update batching. Those changes regressed the benchmark set and were removed.

This pass also tried a hot/cold `Fiber` split and a dedicated non-batching initial `updateDomProperties(...)` path. Both regressed the benchmark set and were removed.

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

Current measurements from the latest Windows amd64 pass:

- `BenchmarkIsNilableTypeCachedPointer`: `27.58 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkFastEqualInt`: `5.264 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkFastEqualString`: `7.177 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkAreDepsEqual3Primitives`: `20.79 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkGoUseStateIntDirectUpdate`: `365.7 ns/op`, `248 B/op`, `5 allocs/op`
- `BenchmarkGoUseStatePointerNilReset`: `290.6 ns/op`, `212 B/op`, `1 allocs/op`
- `BenchmarkGoUseMemoSameDeps`: `59.81 ns/op`, `16 B/op`, `1 allocs/op`
- `BenchmarkGoUseCallbackSameDeps`: `50.36 ns/op`, `16 B/op`, `1 allocs/op`
- `BenchmarkGoUseRefStable`: `7.771 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkGoUseIdStable`: `8.613 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkGoUseFuncWrap`: `65.56 ns/op`, `8 B/op`, `1 allocs/op`
- `BenchmarkGoUseEffectSameDeps`: `53.59 ns/op`, `16 B/op`, `1 allocs/op`
- `BenchmarkDivWithComponents4`: `731.2 ns/op`, `744 B/op`, `9 allocs/op`
- `BenchmarkWithComponentsGeneric4`: `793.4 ns/op`, `760 B/op`, `10 allocs/op`
- `BenchmarkCreateElementHostWithTextChildren`: `749.3 ns/op`, `664 B/op`, `8 allocs/op`
- `BenchmarkPropsEqualChildrenDifferentPointer`: `110.6 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkReconcileChildrenStableList16`: `6921 ns/op`, `3329 B/op`, `16 allocs/op`
- `BenchmarkReconcileChildrenKeyedStableList16`: `10065 ns/op`, `3382 B/op`, `18 allocs/op`
- `BenchmarkUpdateDomPropertiesInitialRender`: `1393 ns/op`, `1120 B/op`, `7 allocs/op`
- `BenchmarkUpdateDomPropertiesSteadyState`: `1002 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkCommitWorkPlacementChain16`: `678.2 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkRunEffectsChain16`: `511.0 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkScheduleUpdate`: `282.0 ns/op`, `208 B/op`, `1 allocs/op`
- `BenchmarkRenderSteadyState`: `619.5 ns/op`, `584 B/op`, `5 allocs/op`
- `BenchmarkAtomRegistrySetAtom32Subscribers`: `921.0 ns/op`, `264 B/op`, `1 allocs/op`
- `BenchmarkGoUseAtomIntUpdate`: `804.4 ns/op`, `248 B/op`, `5 allocs/op`
- `BenchmarkGoUseAtomGetter`: `29.17 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkGoUseAtomStableRerender`: `71.88 ns/op`, `0 B/op`, `0 allocs/op`

Fine-grained prototype comparison from the latest Windows amd64 pass:

- `BenchmarkFineGrainedKeyedDashboardComponentUpdate16`: `13548 ns/op`, `9538 B/op`, `141 allocs/op`
- `BenchmarkFineGrainedKeyedDashboardReactiveTextUpdate16`: `2073 ns/op`, `512 B/op`, `6 allocs/op`

Initial conclusion for the text-only fine-grained prototype:

- a hot value update inside a stable keyed dashboard row is materially cheaper when it stays on the reactive text path instead of rerendering the owning component and reconciling the keyed list again
- the current win is specific to narrow text updates; broader region updates, selector-based reads, and list-filtering scenarios still need separate measurement

Ancestor-rerender comparison for 64 stable children from the latest Windows amd64 pass:

- `BenchmarkFineGrainedAncestorRerenderStaticLeaves64`: `19489 ns/op`, `2901 B/op`, `27 allocs/op`
- `BenchmarkFineGrainedAncestorRerenderReactiveRegions64`: `19418 ns/op`, `2902 B/op`, `27 allocs/op`

Current conclusion for the clean-clone transfer cost:

- for the current 64-region ancestor-rerender benchmark shape, the reactive-region path is now effectively at parity with static leaves on this machine: removing clone-time ownership transfer for unchanged subscriptions and redirecting stale subscribed fibers to their live fine-grained twin eliminated the earlier CPU and allocation gap versus the 30.7us / 16.3KB / 220 alloc baseline

Earlier arm64 numbers are kept below for comparison history, but the list above is the current baseline for this branch.

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

Follow-up hook/atom/keyed reconciliation pass from the latest Windows arm64 session:

- `BenchmarkGoUseFuncWrap`: `69.44 ns/op`, `8 B/op`, `1 allocs/op`
- `BenchmarkGoUseAtomGetter`: `24.22 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkGoUseAtomIntUpdate`: `304.1 ns/op`, `208 B/op`, `4 allocs/op`
- `BenchmarkGoUseAtomStableRerender`: `83.55 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkReconcileChildrenKeyedStableList16`: `8591 ns/op`, `2869 B/op`, `18 allocs/op`

The keyed reconciler benchmark remained meaningfully slower than the non-keyed stable list path, but it improved substantially from its earlier shape in this pass:

- earlier keyed benchmark after first keyed benchmark addition: `13983 ns/op`, `5138 B/op`, `26 allocs/op`
- after comparable-key matching, pooled scratch state, and destructive consumption: `8591 ns/op`, `2869 B/op`, `18 allocs/op`

Commit traversal was audited but not changed further in this pass; the benchmark cost is already low relative to atom fan-out and browser-bound DOM work.

Some reconciler benchmarks remain noisy because cold microbenchmarks do not model steady-state alternate reuse perfectly. That is why repeated runs and comparison tooling are still a worthwhile next step.

### SSR transport microbenchmarks

Run on Windows:

```powershell
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go test ./ui -run ^$ -bench "RenderToStringPublicSSRSurface|MarshalSSRBootstrapJSON|MarshalSSRBootstrapBinary|UnmarshalSSRBootstrapJSON|UnmarshalSSRBootstrapBinary|RenderBootstrapReferenceScript" -benchmem
```

Current measurements from the latest Windows amd64 pass:

- `BenchmarkRenderToStringPublicSSRSurface`: `5693 ns/op`, `4160 B/op`, `54 allocs/op`
- `BenchmarkMarshalSSRBootstrapJSON`: `6085 ns/op`, `3161 B/op`, `41 allocs/op`
- `BenchmarkMarshalSSRBootstrapBinary`: `2310 ns/op`, `240 B/op`, `2 allocs/op`
- `BenchmarkUnmarshalSSRBootstrapJSON`: `7910 ns/op`, `2200 B/op`, `53 allocs/op`
- `BenchmarkUnmarshalSSRBootstrapBinary`: `5257 ns/op`, `1848 B/op`, `41 allocs/op`
- `BenchmarkRenderBootstrapReferenceScript`: `700.4 ns/op`, `464 B/op`, `6 allocs/op`

Current conclusion:

- Inline JSON remains the simplest default for small bootstrap payloads embedded directly into SSR HTML.
- The CBOR sidecar path is materially cheaper than JSON for bootstrap encode/decode in this repo and is the better transport when payload size starts to matter.
- Base64-inlined binary is still not attractive here; the better binary shape is an external sidecar referenced from the page and decoded on the wasm client.

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

Current wasm adapter measurements from the latest Windows js/wasm pass:

- `BenchmarkWASMDOMAdapterCreateElement`: `79153 ns/op`, `248 B/op`, `24 allocs/op`
- `BenchmarkWASMDOMAdapterSetAttribute`: `53092 ns/op`, `136 B/op`, `13 allocs/op`
- `BenchmarkWASMDOMAdapterAppendChild`: `131177 ns/op`, `344 B/op`, `33 allocs/op`
- `BenchmarkWASMDOMAdapterQuerySelector`: `92169 ns/op`, `240 B/op`, `23 allocs/op`
- `BenchmarkWASMDOMAdapterGetElementById`: `93303 ns/op`, `240 B/op`, `23 allocs/op`
- `BenchmarkWASMDOMAdapterQuerySelectorAll`: `107746 ns/op`, `338 B/op`, `27 allocs/op`
- `BenchmarkWASMDOMAdapterSetPropertyString`: `5819 ns/op`, `8 B/op`, `1 allocs/op`
- `BenchmarkWASMDOMAdapterBatchAppend16`: `763398 ns/op`, `1780 B/op`, `171 allocs/op`
- `BenchmarkWASMEventAdapterAddRemoveListener`: `71854 ns/op`, `192 B/op`, `16 allocs/op`
- `BenchmarkWASMDOMAdapterWrapFunctionNoArgsInvoke`: `16744 ns/op`, `32 B/op`, `3 allocs/op`
- `BenchmarkWASMDOMAdapterWrapFunctionStringInvoke`: `29579 ns/op`, `80 B/op`, `8 allocs/op`
- `BenchmarkLegacyWrapFunctionNoArgsInvoke`: `16003 ns/op`, `32 B/op`, `3 allocs/op`
- `BenchmarkLegacyWrapFunctionStringInvoke`: `29756 ns/op`, `80 B/op`, `8 allocs/op`

Earlier wasm measurements are retained below for trend reference.

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

Follow-up query/collection pass from the latest Windows arm64 wasm session:

- `BenchmarkWASMDOMAdapterQuerySelector`: `117253 ns/op`, `240 B/op`, `23 allocs/op`
- `BenchmarkWASMDOMAdapterGetElementById`: `94245 ns/op`, `240 B/op`, `23 allocs/op`
- `BenchmarkWASMDOMAdapterQuerySelectorAll`: `125510 ns/op`, `335 B/op`, `27 allocs/op`

Caching bound document query methods and using direct indexed collection access materially improved the `QuerySelectorAll` path compared with the earlier benchmark (`145711 ns/op`, `431 B/op`, `34 allocs/op`).

## What To Optimize Next

1. Add repeated benchmark runs and `benchstat`-style comparison to reduce noise.
2. Keep pushing steady-state reconciliation/fiber reuse benchmarks instead of only cold-path microbenchmarks.
3. Profile browser-bound paths separately from native runtime paths.
4. Treat correctness regressions as blockers; performance changes in this repo have repeatedly shown that low-level wins are only worth keeping if the benchmark set and behavior tests both stay green.
5. Treat host prop update churn and DOM/event boundary cost as the next likely levers; broader struct-layout rewrites have not paid off here.

For the broader build-profile, size-budget, and release-artifact policy around those measurements, see [WASM_RELEASES.md](WASM_RELEASES.md) and [BUILD_EXPERIMENTS.md](BUILD_EXPERIMENTS.md).

For the broader build-profile and release-flag policy around those measurements, see [WASM_RELEASES.md](WASM_RELEASES.md).
