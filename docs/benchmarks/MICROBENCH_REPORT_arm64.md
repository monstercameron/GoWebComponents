# GoWebComponents — Whole-Program Microbenchmark Report

- **Machine / toolchain:** `go1.26.3` `windows/arm64` (Snapdragon X2, 18 logical CPUs)
- **Generated:** 2026-06-13
- **Harness:** `gwc bench --lane all` (`go test -run ^$ -bench . -benchmem`), native + js/wasm lanes
- **Coverage:** 500 benchmarks across 34 package×lane targets — **391 native**, **109 wasm**
- **Raw data:** `docs/benchmarks/run-arm64.json` · profiles in `bin/prof/` · latency samples in `bin/prof/latency.txt`
- **Note:** the committed `docs/benchmarks/latest.json` (amd64, go1.26.0) was **not** overwritten; this run went to `run-arm64.json`.

## 1. Throughput — core hot paths (native)

| Path | Benchmark | ns/op | B/op | allocs/op |
| --- | --- | ---: | ---: | ---: |
| Diff, stable list (16) | `ReconcileChildrenStableList16` | 3,303 | **0** | **0** |
| Diff, with fragments | `ReconcileChildrenWithFragments16` | 3,553 | 24 | 1 |
| Diff, keyed | `ReconcileChildrenKeyedStableList16` | 4,028 | 48 | 2 |
| Component unit-of-work | `PerformUnitOfWorkFunctionComponentLeaf` | 7,762 | 776 | 6 |
| Targeted reactive text update | `FineGrainedKeyedDashboardReactiveTextUpdate16` | 3,112 | 184 | 8 |
| Full 16-component re-render | `FineGrainedKeyedDashboardComponentUpdate16` | 46,080 | 10,140 | 150 |
| SSR render (public surface) | `RenderToStringPublicSSRSurface` | 2,830 | — | — |
| SSR render, 100 rows | `SSRRenderToString100Rows` | 304,700 | — | — |
| Bootstrap marshal (binary) | `MarshalSSRBootstrapBinary` | 1,810 | — | — |
| No-op patch stream | `BuildPatchStreamIdentity` | **~10** | — | — |

**Reads:**
- The steady-state child diff is **allocation-free** (0 B/op, 0 allocs/op) — the reconciler reuses fiber/prop structures on stable lists.
- **Fine-grained reactivity earns its keep:** a targeted reactive-text update (3.1µs / 8 allocs) is **~15× cheaper** than re-rendering the 16-component subtree (46µs / 150 allocs).
- The identity/no-op patch path short-circuits to ~10ns — unchanged regions cost essentially nothing.

## 2. `CurrentVsLegacy` A/B suite — current engine vs deprecated paths

The codebase carries paired benchmarks proving the current engine beats the legacy implementations it replaced. These dominate the "slowest" list precisely because the *legacy* arm is the baseline being beaten:

| Benchmark | current (native) | legacy (native) | speedup |
| --- | ---: | ---: | ---: |
| `ApplyCommittedChildOrder` | 162.8µs | 8.16ms | **~50×** |
| `BuildCanonicalRenderIR` | 985µs | 1.11ms | ~1.13× + fewer allocs |
| `CommitRegionPatchTransaction/append-256` | partial-snapshot | 855µs / **1.63 MB**/op (reclone) | reclone is the alloc hog |

The legacy `reclone` snapshot path allocates **1.63 MB/op**; the current partial-snapshot path is the optimization. Keep the legacy arms as regression guards only.

## 3. CPU & allocation hotspots (pprof)

### Reconcile engine (`internal/runtime`, pure diff/commit)
Time concentrates in fiber cloning:
- `(*Runtime).buildUpdatedFiber` — 17.5% cum
- `(*Runtime).cloneChildFibers` — 3.5% cum
- `(*Runtime).commitWork` — 2.9% cum

→ The main lever for further reconcile speedups is reducing fiber-clone work on update.

### ⚠️ Inspector instrumentation dominates the "representative scenarios"
`BenchmarkProfilingRepresentativeScenarios` (large-list / nested-routes / async-dashboard / portal-overlays, ~7µs each) spends **>50% CPU in `(*Runtime).Inspect`** and **90.6% of all allocations (45.5 GB cumulative) in `collectFlamegraphFrames`**, plus `inspectFiberTreeWithPath` / `collectHotBranches`. This benchmark measures the **devtools inspector/flamegraph machinery**, not pure rendering. Worth confirming the inspector is meant to be live on this path — if it leaks into production renders it's a massive allocation source.

### Sanitize is parser-bound, not GWC-bound
`SanitizeLargeDocument` (1.39ms, 679 KB, 7050 allocs): **66% of allocations are inside `golang.org/x/net/html.ParseFragmentWithOptions`** (`addElement` 32%, `addText` 26%, `strings.Builder` 26%). GWC's only hot function is `stripControlChars` (9% CPU). Throughput here is bounded by the upstream HTML parser; GWC-side optimization headroom is small.

## 4. Latency distribution (`-count=10`, native)

| Benchmark | min | median | max | CV% |
| --- | ---: | ---: | ---: | ---: |
| `ReconcileChildrenStableList16` | 3.29µs | 3.35µs | 3.47µs | 1.6 |
| `FineGrainedKeyedDashboardReactiveTextUpdate16` | 3.00µs | 3.11µs | 3.56µs | 5.0 |
| `FineGrainedKeyedDashboardComponentUpdate16` | 39.1µs | 41.0µs | 43.9µs | 3.3 |
| `RenderToStringPublicSSRSurface` | 3.48µs | 3.82µs | 4.22µs | 6.2 |
| `MarshalSSRBootstrapBinary` | 1.73µs | 1.81µs | 1.86µs | 2.0 |
| `BuildPatchStreamIdentity/*` | 10ns | 10ns | 11ns | ~3.5 |
| `SanitizeLargeDocument` | 1.10ms | 1.29ms | **2.06ms** | **20.0** |

Core engine paths are very stable (CV ≤ 6%). The outlier is **sanitize (CV 20%, p-max 2.06ms vs median 1.29ms)** — its tail is GC-driven, a direct consequence of the 679 KB/op parser allocation.

## 5. Native vs WASM

Go/wasm runs the same code **~4.5–6.5× slower** than native:

| Benchmark | native | wasm | ratio |
| --- | ---: | ---: | ---: |
| `ReconcileChildrenKeyedStableList16` | 4.03µs | 25.82µs | 6.4× |
| `SSRRenderToString100Rows` | 304.7µs | 1.38ms | 4.5× |
| `ApplyCommittedChildOrder/current` | 162.8µs | 816µs | 5.0× |

Allocation counts are identical across lanes (same code); only wall-time scales. Budget client-side (wasm) render work at ~5× the native number.

## 6. Caveats — 3 wasm-lane targets failed (pre-existing, not benchmark regressions)

- **`state` (wasm):** build failure — `ExampleUseAtom` redeclared in `example_test.go` and `useatom_example_test.go`. Genuine test-file bug; blocks the package's wasm benchmarks.
- **`interop` (wasm):** `TestGetCookieNativeReturnsUnavailable` panics under the node wasm host (`syscall/js: Value.Get on undefined` in `readRawCookies`).
- **`internal/platform/jsdom` (wasm):** test failure under the node host.

None are caused by this run, and none affect the 500 measured benchmarks. The `state` redeclaration is a quick, worthwhile fix.

## 7. SLA / budget analysis

There is **no absolute latency SLA defined in-repo** — the only codified budget is the harness's **2% regression tolerance** plus a normalized score (100 = parity with the committed `reference.json`). The runtime also tracks `RouteStartupBudget` telemetry (hydration / startup-commit / first-interaction durations) but with no threshold constants. So this section judges against two lenses: the **60fps frame budget (16.67ms)** — the natural SLA for a render framework — and the framework's own **relative score**.

### Frame-budget SLA (16.67ms = one 60fps frame)

**No production path violates the frame budget, in either lane.** The only path over 16.67ms is a deliberately-retained *legacy* A/B baseline.

| Lane | Over 16.67ms (frame) | 1–16.67ms (sub-frame, watch) |
| --- | --- | --- |
| native | 0 (legacy A/B arm 8.16ms only) | sanitize 1.24ms; legacy IR build 1.11ms |
| **wasm** (browser runtime) | 0 production (`ApplyCommittedChildOrder/legacy` 36.35ms is the A/B baseline) | see below |

**WASM sub-frame watch list** (wasm is the real client runtime, so these matter most):

| ns→ms | Path | Note |
| ---: | --- | --- |
| **13.08ms** | `GoUseFetchRefetchUnavailable` | 🔴 anomalous — ~¾ of a frame for a fetch-unavailable path; likely an error/retry branch. Investigate. |
| 3.81ms | `HydrateMediumTreeFirstUpdate` | 🟡 one-time hydration, ~7000 allocs |
| 3.68ms | `HydrateMediumTreeReuse` | 🟡 hydration reuse path |
| 2.24ms | `CurrentNestedLayoutRoute` | nested route resolution |
| 1.63ms | `TransitionListRefresh250` | 250-item list refresh |
| 1.38ms | `SSRRenderToString100Rows` | SSR (normally native, not wasm) |
| 1.02ms | `ProfilingRepresentativeScenarios/portal-overlays` | inspector-laden (see §3) |

### Relative SLA (normalized score vs `reference.json`)

Overall **67/100** — but the reference is amd64 / go1.26.0 and this run is **arm64 Snapdragon X2**, so the absolute level is mostly an architecture artifact. The diagnostic value is the **bucket spread**:

| Bucket | Score | Benches |
| --- | ---: | ---: |
| Compute | 82.8 | 6 |
| Memory | 80.3 | 2 |
| End-to-End | 79.7 | 7 |
| Sync/Concurrency | 73.0 | 11 |
| **Alloc/Runtime** | **35.1** | **71** |

Compute/Memory/E2E/Sync cluster at ~73–83 (consistent with this core being ~20–25% slower than the reference box). **Alloc/Runtime is the outlier at 35** — ~2.3× worse than every other bucket. That is the real "below SLA" signal: **allocation-heavy runtime paths underperform disproportionately on this hardware.** The CPU profile confirms it — `runtime.mallocgc` is **26.7%** of CPU, with GC scan/`memclrNoHeapPointers`/`sysUnusedOS` dominating the rest. **GC/allocation, not compute, is the systemic bottleneck**, and on a fanless ARM laptop that also costs battery/thermal.

**Allocation hogs feeding that bucket:** inspector `collectFlamegraphFrames` (45.5 GB cumulative — likely should not be live on render), legacy `reclone` snapshot (1.63 MB/op), hydration (~7000 allocs), sanitize (679 KB, parser-bound).

## 8. Optimizations applied

### ✅ Inspector flamegraph collection — `collectFlamegraphFrames` (`internal/runtime/inspect.go`)

The single largest allocation site in the whole sweep. Root causes:
1. `make([]FlamegraphFrameSnapshot, 0, 256)` preallocated a ~30 KB backing array on **every** call, while real snapshots emit only a handful of frames.
2. `append(append([]string(nil), path...), name)` copied the entire ancestor path **at every node** (O(N·D)).
3. Frames were built for the whole tree, then truncated to 256.

Fix: lazy-grown result slice (cap 16, grows as needed), a single reused push/pop path stack, and early-exit once the limit is hit. Behavior-preserving (DFS order, `Path` format, frame[0] timings all unchanged; `inspect`/`plugininterposer` tests pass).

| `BenchmarkProfilingRepresentativeScenarios` | Before | After | Δ |
| --- | ---: | ---: | ---: |
| B/op | 36,150 | **5,348** | **−85%** |
| allocs/op | 33 | 30 | −9% |
| ns/op | ~6,900 | **~3,600** | **~1.9× faster** |

Cumulative allocation of `collectFlamegraphFrames` over a 2s run fell from **45.5 GB → 1.4 GB**; total `Inspect()` allocation 50 GB → 3.7 GB. This speeds up every devtools/agent snapshot query (the path is *not* on the production render loop, confirmed via `plugininterposer.go`).

### ✅ SSR attribute serialization — `writeSSRProps` / `serializeProps` / `serializeStyleMap` (`internal/runtime/ssr.go`)

Each function allocated a `[]string` key buffer per element (runtime-sized → heap) and passed it to `sort.Strings`, which boxes into `sort.Interface` and forces the backing array to escape. Fix: a stack-allocated 16-element small buffer + `slices.Sort` (generic, no boxing) → allocation-free key sorting for the common case (≤16 attributes), identical lexical output.

| `BenchmarkSSRRenderToString100Rows` | Before | After | Δ |
| --- | ---: | ---: | ---: |
| allocs/op | 2,406 | **2,306** | −100 (one heap slice per row) |
| B/op | 219,509 | 214,707 | −4,800 |
| ns/op | ~247k | ~249k | neutral (verified, count=12) |

Escape analysis confirms the buffer stays on the stack. Benefits all SSR/serialization output.

### ✅ Telemetry redaction array path — `redactValue` (`internal/telemetryredaction/redaction.go`)

Per array element, the redactor built an indexed field path with `fmt.Sprintf("%s[%d]", …)` — interface boxing + format parsing on a path that runs on **every redacted telemetry log/trace framework-wide** (via `logging.RedactTelemetryValue`). Replaced with `parsePath + "[" + strconv.Itoa(i) + "]"`. Added the package's first benchmark (`BenchmarkRedactValueNestedArrays`) for regression coverage.

| `BenchmarkRedactValueNestedArrays` | Before | After | Δ |
| --- | ---: | ---: | ---: |
| ns/op | 4,175 | **2,945** | **−29%** |
| allocs/op | 65 | **48** | **−26%** |
| B/op | 2,906 | 2,632 | −9% |

### 🎯 Central finding: `cloneElementProps` is the framework-wide allocation bottleneck

Four separate benchmarks — component update, SSR, list-transition refresh, hydration — are all dominated (46–56% of allocations) by `cloneElementProps` + `buildElementWithHostProps`. `CreateElement(tag, props, …)` defensively clones the props map because arbitrary callers may retain/mutate it. The common idiom `CreateElement("li", map[string]any{…}, …)` passes a **fresh, never-reused literal**, making the clone redundant — but the API can't know that at runtime. An opt-out (`CreateElementOwned`) already exists.

This is the single highest-leverage optimization for the whole framework, but it requires an **architectural decision**, not a micro-edit:
- migrate hot internal/sugar call sites that build fresh maps to `CreateElementOwned`, or
- move to copy-on-write / builder-based props, or a slice-backed small-map for the common ≤4-key case (maps carry high fixed overhead).

Recommend prototyping behind the existing `Owned` fast path and A/B-ing before committing.

### Investigated and intentionally **not** changed (risk/benchmark-artifact)

- **`GoUseFetchRefetchUnavailable` (13 ms wasm)** — the cost is the *benchmark's own* `time.Sleep(1ms)` poll loop (2 s deadline), not framework code. Nothing to optimize; the benchmark measures sleep.
- **`cloneElementProps` (53% of component-update allocs)** — a deliberate safety contract of the public `CreateElement` (callers may retain the props map). An opt-out already exists (`CreateElementOwned`). Removing the clone risks caller-mutation bugs across every app; a real fix is architectural (small-map/slice-backed props), needing design buy-in.
- **Hydration diagnostics (~16% of hydrate allocs)** — driven by the benchmark's mock producing text mismatches; ~32% of the rest is the **test DOM adapter**, not production code. Optimizing would chase a benchmark artifact.
- **`BuildSnapshotTransportPayloadWithFallback/binary-fallback` (897k ns, 369 KB)** — an edge **error path**: it builds the binary buffer, then fails on a pathological 70,000-char region ID (binary uses a uint16 length prefix, max 65,535) and falls back to structured-clone JSON. The common `binary-success` path is already 524 ns / 1 alloc. A complete pre-check would duplicate the entire encode walk; a partial one would only fast-path this benchmark's pathological input.
- **`TransitionListRefresh250` / `CompareSnapshots` export** — the former is dominated by `cloneElementProps` (see central finding); the latter by `ExportSnapshotJSON` building full redacted-JSON trees for both snapshots (inherent to its compare-by-export design). The redaction sub-cost was optimized above.

## How to reproduce

```
go run ./tools/gwc bench --lane all --out docs/benchmarks/run-arm64.json
# focused profiling:
go test ./internal/runtime -run ^$ -bench BenchmarkProfilingRepresentativeScenarios \
  -benchmem -cpuprofile cpu.prof -memprofile mem.prof
go tool pprof -top -sample_index=alloc_space mem.prof
# latency spread:
go test ./sanitize -run ^$ -bench SanitizeLargeDocument -benchmem -count 10
```
