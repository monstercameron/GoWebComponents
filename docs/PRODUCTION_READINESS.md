# v5 production readiness

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
| M9, M12 | Still unmeasured: no cross-framework comparison against Solid/Svelte/Rust, no projection-induced pause measurement. |
| P6.2, P6.3 | Route splitting and streaming instantiation unimplemented. |
| Commit inside the frame budget | Reconciliation is sliced; deletion, DOM commit, order repair, and layout effects are not. This is the structural reason M2 resists scheduling fixes. |
