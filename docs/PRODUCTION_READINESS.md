# v5 production readiness

> ## 2026-07-26 — the gate numbers below this banner were measured with a probe that generated no input
>
> `driveTyping` in the P0.2 harness is a `setTimeout` that types nothing, and always
> has been. Event Timing only records TRUSTED events, so page script cannot produce
> input at all — it must come from the automation driver. Consequences:
>
> - **M1's recorded pass is an artifact.** Equivalence between an idle arm and a
>   "loaded" arm in which the probe also did nothing is trivially true regardless of
>   what the runtime does.
> - **M2 and M7 were reporting zeros from dead instruments.** Six deliberately
>   injected 180 ms main-thread blocks went uncounted; LoAF does not fire in headless
>   Chromium. M7 reported 0 ms because no collection was ever sampled.
> - **A second contaminant:** 247 livereload dev servers leaked by `tools/gwc`'s
>   dev-loop tests were running on the measurement machine, the oldest two days old.
>   Any benchmark taken alongside them is contended. Fixed — see
>   `killListenersOnPort` in `tools/gwc/start_test.go`.
>
> ### Re-measured properly: headed browser, real trusted keystrokes, quiet machine
>
> Four runs, ~650 interaction samples each, all three §1.2 workloads confirmed running:
>
> | gate | target | measured | verdict |
> |---|---|---|---|
> | M1 frame equivalence | equivalent | equivalent in 4/4 | **PASS** |
> | M2 long frames | 0 | 22, 28, 31, 36 | **FAIL** |
> | M3 interaction p95 | < 50 ms | 34, 48, 48, 56 | **MARGINAL** |
> | M3 interaction max | < 120 ms | 72, 88, 96, 136 | **MARGINAL** |
> | M7 max GC pause | < 3 ms | 9.3, 9.8, 9.9, 11.0 | **FAIL** |
>
> **M1 genuinely passes.** The thesis holds: with real typing and real background
> work, loaded frame time is statistically equivalent to idle. That is the claim v5
> exists to make, and it is now supported by evidence rather than by an idle page.
>
> **M2 and M7 genuinely fail.** LoAF attribution for the long frames:
> `scriptedMs=1345, styleAndLayoutMs=0`, top invoker `INPUT#filter.oninput=825ms`.
> Zero style/layout means this is not a rendering-pipeline problem; it is Go work in
> the input path. A ~10 ms GC pause cannot explain a 100 ms frame, so M2 and M7 have
> separate causes.
>
> **v5's scheduling flags make it worse, confirmed twice.** With `?v5=1`
> (PassiveEffectsAfterPaint + LaneQueues + frame budget) the same workload produced 36
> long frames instead of 28, a 119 ms worst frame instead of 82 ms, and 457 ms of
> blocking instead of 70 ms. This independently reproduces the plan's own finding
> that the frame-budget default had to be reverted.
>
> ### A stale served artifact invalidated most of 2026-07-26's browser numbers
>
> The harness page is served from a COPY of `v5harness.wasm` in a scratchpad
> directory, not from the repository. That copy went stale early in the day, so
> every browser measurement taken afterwards ran the same old binary regardless
> of what was rebuilt. Three conclusions drawn from those runs are withdrawn:
>
> - **The paint-yield cascade guard was never tested.** It was recorded as
>   "refuted"; it had in fact never executed in a browser. See the
>   `sched 2026-07-26` note in `internal/runtime/scheduler.go`.
> - **The dev-build vs `-tags production` comparison (21.7 vs 17.3 long frames)
>   was noise.** Both arms ran the same binary.
> - **The arena pre-grow experiment's first result was meaningless.** Re-run
>   correctly, it did work in isolation but did nothing here — see below.
>
> Anyone measuring this harness must check the served artifact's checksum against
> the freshly built one before trusting a number. This is the third
> instrument-level defect found in this gate suite in two days, after the probe
> that generated no input and the 247 leaked dev servers.
>
> ### M2 re-measured on a verified-fresh production binary
>
> With the served artifact confirmed by checksum, three runs: **16, 15, 17 long
> frames** (worst 98-101ms), M1 equivalent in all three, M3 p95 40-48ms.
>
> That is materially better than the 22-36 recorded all day from the stale copy,
> and much more stable. M2 still fails against a budget of 0, and M3 still misses
> its 120ms ceiling on the worst interaction, but the real gap is smaller than
> this document previously claimed.
>
> ### M7: the pointer-density hypothesis is refuted; the cause is still open
>
> This document previously attributed M7 to "mark-termination cost over a
> pointer-dense fiber tree." A standalone probe outside this repo refutes that.
> Same byte count, one pointer-free heap and one pointer-dense heap shaped like a
> fiber tree:
>
> | heap | bytes-only | pointer-dense |
> |---|---|---|
> | 1MB | 8.80ms | 0.70ms |
> | 2MB | 0.50ms | 0.70ms |
> | 4MB | 0.30ms | 0.30ms |
>
> Pointer density is irrelevant. In that probe the expensive pause tracked heap
> GROWTH, and pre-growing the wasm arena at boot fixed it decisively — fresh
> pages, three runs each, 1MB workload: **8.70/8.40/8.60ms without, 1.30/1.70/1.40ms
> with**.
>
> **That fix does not transfer to the harness.** Pre-growing 32MB at boot (cost:
> 53-79ms, charged to M10) left M7 at 9.5-12.5ms, unchanged. So the harness's
> pauses are not arena growth either, and the pre-grow was reverted rather than
> kept as a boot cost that buys nothing.
>
> Two further candidates were then tested and eliminated:
>
> - **`syscall/js` reference table.** Pause is flat at 8.0-8.9ms across 0, 500,
>   2000 and 8000 live `js.Func` callbacks. Not handler count.
> - **Collector warmup ordinal.** Recording EVERY cycle rather than the max shows
>   the cost sits at a fixed ordinal in a fresh instance — the SECOND collection:
>   `perCycle=[0.0  8.1  0.8 1.0 0.4 0.5 0.6 ...]`, identical at 0.07MB and 1MB
>   heaps and with any handler count. Real, and the harness boot now warms three
>   collections past it — but it did NOT move M7 (9.2-10.8ms with, 9.5-12.5ms
>   without). Not what the harness is hitting.
>
> ### The premise behind the rejected GC knobs was wrong
>
> This document and the harness both asserted the heap is "under a megabyte", and
> used that to explain why GOGC pacing cannot help. Measured, it peaks at
> **11.02MB** (per-window: 5.35 6.03 4.54 1.63 1.14 1.73 11.02 9.44 6.20 6.13
> 6.65 6.48). So the knob was dismissed on a false premise and had to be re-tested.
>
> Re-tested at GOGC 20 and 10: collections rise from 6 to **66-69** per run and
> the heap is held at ~1.0-1.7MB instead of peaking at 11MB. The max pause does
> **not** move — 9.5, 18.0, 13.5, 14.9ms — and M2 gets worse (26-29 long frames vs
> 12-18).
>
> ### Where that leaves M7
>
> The max pause is invariant to heap size, collection frequency, pointer density,
> arena growth, handler count and collector warmup. Eight mitigations across two
> sessions have moved it by nothing. Meanwhile a quiescent probe on the same
> machine, at the same heap sizes, pauses 0.3-0.7ms.
>
> The remaining structural difference is that the harness's collections happen
> inside a running app — JS/wasm interop, DOM commits, a live Web Worker — where
> the probe's happen on an idle heap. Go's wasm runtime is single-threaded and
> cooperatively scheduled on the JS event loop, so a wall-clock `PauseNs` spanning
> a return to that event loop can include browser work that is not collection
> cost. That would explain the invariance, and it would mean **`PauseNs` in
> js/wasm is not the pure stop-the-world measure M7 assumes.**
>
> ### M7 RESOLVED (cause, not fix): collections that land mid-render
>
> That hypothesis was tested and is also wrong — `PauseNs` UNDERSTATES rather than
> inflates. Forced collections bracketed with `performance.now()` cost 1.9-6.2ms
> of wall time while Go attributes 0.1-0.4ms of it to "pause", because Go's pause
> excludes concurrent mark, which in single-threaded wasm still blocks the main
> thread. So M7's number is conservative, not contaminated.
>
> The cause came out of tracing EVERY cycle instead of the max
> (`window.__gwcV5PauseTrace`, added in probe.go). One representative run:
>
> | phase | cycles |
> |---|---|
> | boot / warmup | `1:9.1  2:0.8  3:1.4  4:0.9  5:2.0  6:0.6  7:0.5  8:0.9  9:1.1` |
> | **during the measured run** | `10:12.2 11:1.8 12:7.3 13:6.9 14:8.1 15:8.6 16:0.7 17:9.3 18:7.3 19:0.7 20:9.0 21:7.3` |
> | forced, seconds later | `22:0.2 23:0.1 24:0.1 25:0.3` |
>
> Collections cost 7-12ms REPEATEDLY while the renderer is working, and 0.1-0.3ms
> on a comparable heap moments after it stops. Forced collections report 0.1-1.3ms
> idle, under background load, and after load — the earlier "under load" audit
> showed nothing because it ran the workloads WITHOUT typing, so no render passes
> were in flight. Rendering is the variable, not load.
>
> The mechanism follows from the reconciler's own design: mid-render, BOTH fiber
> trees are live at once (`alternate` double-buffering) plus the freshly built
> element graph, so a collection landing there traces roughly twice the tree at
> its largest. This also explains the invariance to every knob tried — heap size,
> collection frequency, pointer density, arena growth, handler count, warmup
> ordinal. None of them change what is live during a render pass.
>
> **M2 and M7 therefore share one lever, and this document's claim that they have
> "separate causes" was wrong.** Both are bounded by render-path allocation: it
> sets how often a collection is triggered mid-pass and how much is live when one
> is. The remaining targets are named by the allocation profile —
> `buildElementWithHostProps` (25%), `cloneElementProps` + `maps.Copy` (24%) — and
> the prediction to test is that cutting them moves M2 and M7 TOGETHER. The
> `reportDuplicateKeys` fix below is the first increment of exactly that work.
>
> The 0.3-0.7ms quiescent floor remains the reason to think 3ms is reachable.
>
> M2 remains where the plan placed it: the intrinsic cost of one
> render-and-commit pass, closing only by making reconciliation and commit
> faster. Note that of the "three scheduling attempts" this document previously
> cited, only two were validly measured.
>
> **The instruments now refuse to report a number they did not measure.** M1 fails
> without recorded interactions, M2 fails when the frame timeline saw a long frame the
> observer missed, M7 fails when a zero pause came from a probe that never answered.
> Each guard has a paired test proving it fails when blind and passes on a real result.


What was checked, what it said, and what is left. Written from measurements taken
on 2026-07-26 against HEAD; every number here is reproducible with the command
next to it.

## Release gates

This repository defines production-readiness in CI, not in prose: 17 workflows,
several of them merge gates. Their local status:

| Gate | Command | Status |
|---|---|---|
| vet, native | `go vet ./...` | pass |
| vet, js/wasm | `GOOS=js GOARCH=wasm go vet ./ui/ ./html/ …` | pass |
| unit tests | `go test ./...` | pass |
| wasm unit tests | 27 packages via `go_js_wasm_exec` | pass |
| public-API baseline | `go test ./serverfn/ ./query/ … -run APIBaseline` | pass |
| doc samples compile | `go test -tags doccompile ./docs/doclint` | pass, **was broken** |
| doclint commands | `go test -tags doclintcmd ./docs/doclint` | pass |
| doctor parity | `go test ./tools/gwc/ -run "Doctor\|…"` | pass |
| starter templates | `go test ./tools/gwc -run TestDefaultStarterTemplates…` | pass |
| examples build | `go build ./examples/public/...` | pass |
| changelog release gate | `go test ./tools/changelogcheck/` | pass |
| **race** | `go test -race ./...` | **CANNOT RUN — see below** |
| **bench drift** | `gwc bench -lane native -fail-on-regression` | **CANNOT RUN — see below** |
| supply chain / vuln | `gwc supplychain`, `gwc vuln` | needs network |

### The doc-sample gate was broken by the /v5 move

The harness compiled each marked sample in a scratch module whose go.mod
required `github.com/monstercameron/GoWebComponents` as a literal string. Moving
the library to `/v5` made every sample fail with "does not contain package
.../v5/ui" — fifteen of them, all of which were fine. Fixed by reading the module
path from the repository's own go.mod, with the require version derived from the
major suffix (Go rejects a require of `.../v5` at `v0.0.0`, so that could not be
a constant either).

### Race detection is unavailable on this machine, and it matters here

`-race` requires cgo, and cgo on windows/arm64 has no usable C toolchain: the
installed clang defaults to an MSVC target that rejects the `-mthreads` flag Go
passes, and no mingw runtime is present. `GOARCH=amd64` does not help — the same
cgo requirement applies.

This is the most valuable gate that is not running, because the concurrency
surface changed materially: frame-loop ownership is now tracked per goroutine
across `atomic.Int32`/`atomic.Uint64`, the async inbox drains entries that may
run on a producing goroutine under hard overflow, and several of those fields
were converted to atomics only after being introduced as plain ints. CI must
cover this; local green means less than usual.

### The bench-drift baseline is stale and architecture-mismatched

`docs/benchmarks/latest.json` was generated **2026-03-25 on amd64** and holds 249
benchmarks; a current run on this machine produces 197 on arm64. Comparing across
architectures is meaningless — a local run reports "61 regressed" that says
nothing about the code. The baseline was deliberately NOT regenerated here:
writing an arm64 baseline would corrupt the gate for the amd64 runner that
actually enforces it. Whether the gate passes on the v5 branch is **unknown**.

## A recommendation that measurement does not support

Two independent reviews named `cloneElementProps` as the largest framework-wide
allocation source, at 46–56%, and made finishing that optimization a ship
requirement. It does not reproduce.

Allocation profile of the render-path benchmarks
(`go test ./internal/runtime/ -bench "Mount|Reconcile|Render" -memprofile`):

| site | flat | cum |
|---|---:|---:|
| `buildRenderShareTree` (benchmark scaffolding, not framework) | 55.05% | — |
| `buildElementWithHostProps` | 13.34% | — |
| `maps.Copy` (called by cloneElementProps) | 4.62% | — |
| `buildElementHostProps` | 4.31% | — |
| **`cloneElementProps`** | **3.14%** | **7.76%** |

It has exactly one caller, `CreateElement`. The `html` package builds through
`CreateElementOwned` and the typed compact lane, so the clone never fires for
markup written with `html.Props` — which is most application markup. Removing it
means moving children off the props map and changing a retained-props contract on
a core path, for under 8% of allocations that are not what either remaining
budget is bound by. Not worth the risk on this evidence.

## What measuring M10 found

The example had been "verified" by a packaging test that checks which symbols
land in which binary. It had never been RUN. Four defects, each of which fails by
hanging rather than by erroring:

1. `ui.Render(..., "root")` passed a bare id where a CSS selector is expected.
   The runtime reported it clearly; nothing was reading the console.
2. A worker-level error carries no request id, so it was delivered against id 0,
   matched no waiter, and was dropped. A services.wasm that never loaded
   presented as a command that never returned.
3. The app posted its first command without waiting for the worker's ready
   signal. A message sent before `onmessage` exists is dropped silently — which
   the services binary's own comment predicts, and which the app ignored.
4. `js.Value.String()` on an absent field returns the literal `"<undefined>"`,
   a non-empty string, so every SUCCESSFUL reply was read as a rejection.

The pattern is worth more than the individual bugs: every one of them fails by
producing nothing, and the test only became diagnosable after it captured console
output, page errors, and failed requests. A browser test that can only say
"waited 60 seconds" names no cause.

## What is still missing

| | Why it is not done |
|---|---|
| M2 — 8 long frames against a budget of 0 | Render performance. React reaches 0 on the same nine scenarios, so the budget is achievable and this is the same gap M6 reports as 0.70x. Both scheduling directions have been tested; the remaining path is making reconciliation and commit faster. |
| M7 — ~6.4 ms GC pause against 3 ms | Six mitigations tested and rejected (GOGC 40/20/10, boot warming, periodic collection, memory limit, pause-window off-by-one, frame-budget sweep). Forced collections cost 0.2–0.5 ms warm and 6.3 ms cold; what remains is mark-termination cost over a pointer-dense fiber tree. |
| CashFlux still on v4.2.0 | Different repository. No real application exercises v5, so nothing has validated it outside two synthetic harnesses. |
| Real GoGRPCBridge stress | Never run. The console trace that motivated v5 is dominated by application work — `hydrate`, `pull`, `flush` at 5–33 s — which no reconciler change touches. |
| ~~P3.4 exactly-once~~ | **CLOSED.** `TransactionalCheckpointStore` lets a store host the effect inside its own transaction, so the effect and its applied-record commit together; `Execute` uses it when present. `Runtime.ExactlyOnce()` reports which guarantee is in force, so an application does not have to infer it. A store that cannot do this keeps the previous behaviour, and a test pins that weaker bound so the difference stays visible. What remains is an implementation over real storage — the SQLite store writing the record inside the effect's own transaction. |
| M10 | **MEASURED: 1084 ms against a 400 ms target — missed.** Covers page load, app instantiate, worker spawn, services instantiate, SQLite open, command round trip, and the reply applied through the inbox. Closing the measurement found four defects in the two-artifact example that a packaging test could not see, listed below. |
| ~~M12~~ | **MET.** Residency costs +0.20 ms; worst pause with 20,000 rows resident is 0.90 ms against a 3 ms ceiling. A ship-gate item, now closed. |
| M9 | Still unmeasured: no cross-framework comparison against Solid, Svelte 5, or a Rust framework. |
| P6.2, P6.3 | Route splitting and streaming instantiation unimplemented. |
| Commit inside the frame budget | Reconciliation is sliced; deletion, DOM commit, order repair, and layout effects are not. This is the structural reason M2 resists scheduling fixes. |
