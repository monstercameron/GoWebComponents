# GWC v5 — Plan Of Record

**Goal: main-thread frame time is invariant to workload size** — without giving
up the single-threaded constant factor.

Date: 2026-07-25 · **Revision: r13**

## Implementation status

Branch `v5`. Phases 1 and 2 complete; Phase A measured through PA.2.

| Item | State | Notes |
|---|---|---|
| P1.1 passive after paint | ✅ | flag `PassiveEffectsAfterPaint` |
| P1.2 real frame budget | ✅ | flag `FrameBudgetMs`, gated on P2.5 in code |
| P1.3 idle dispatch | ✅ | idle + SetTimeout backstop, first wins |
| P1.4 ordering contract | ✅ | 6 tests |
| P2.1 async inbox | ✅ | N posts → 1 pass, measured |
| P2.2 lane queues | ✅ | flag `LaneQueues`; allocation-free |
| P2.3 per-runtime state | ✅ | scoped to first-commit signal; conformance suite |
| P2.4 EnqueueUI removal | ✅ | replaced by the inbox |
| P2.5 interrupt-safe restart | ✅ | T12 closed |
| P2.6 runtime-scoped atoms | ✅ | `ResolveRuntime` |
| PA.1 baseline | ✅ | candidate (a) found spent |
| PA.2 prototypes | ✅ | (b) unsound, (c) wins 2–3x |
| PA.3 ship (c) | ⛔ blocked | see blockers under PA.3 |
| P0.2 subject app | ✅ | 5k-row table, 3 workloads, real-input probe |
| P0.3 v4 baseline | ✅ | **captured** — see below |
| Phase 3 · 4 · 5 · 6 | ⛔ | Phase 3 gated on P0.3 and §11 |

### v4 baseline (P0.3, headless Chromium, windows/arm64)

Every §1.1 row that read "unmeasured" now has a number. Every target misses,
which is the point: this gap is what v5 exists to close.

| Metric | v4 measured | v5 target |
|---|---|---|
| M1 idle p95 frame | 16.80 ms | — (control) |
| M1 **loaded** p95 frame | **1633.30 ms** | equivalent to idle |
| M2 long frames | 245 (worst 1649 ms) | 0 |
| M3 interaction p95 / max | 1128 ms / 1400 ms | <50 / <120 ms |
| M7 max GC pause | 6.60 ms | <3 ms |
| harness wasm, uncompressed | 12.7 MB | — (M5 tracks app.wasm gz) |

Under load, v4's p95 frame is **97x** its idle frame. The plan's thesis is no
longer an argument; it is a measurement.

Two harness bugs surfaced on the first real run and are fixed:

- **Drift gating on the loaded arm.** The loaded arm legitimately trends — a
  50k-row import gets slower as the table grows — so gating on it made the
  harness reject exactly the runs it exists to capture. Drift is now gated on
  the **idle** arm, which is the control; loaded drift is reported only.
- **Synthetic input recorded nothing.** Event Timing only accepts *trusted*
  events, so the app dispatching its own `input` events produced no
  `interactionId` and M3 came back n=0 while frames visibly janked. Real
  keystrokes now come from the Playwright driver.

Open harness item: the probe yields ~56 interactions per run, below the 200 the
tail-reliability guard needs for p95. The guard correctly refuses to quote it.
Raise the input rate for a tighter number — the verdict is unaffected at this
magnitude.

### Phase 1+2 flags measured against the baseline — they REGRESS this scenario

R2 says a flag flips once its acceptance test passes. It did not pass. Both arms
ran back to back on one machine, v5 second so thermal drift worked against it,
and both reported `valid=true` with their configuration verified as applied.

| Metric | v4 | v5 (all flags on) | Δ |
|---|---|---|---|
| loaded p95 frame | 1704.98 ms | **3171.60 ms** | **+86%** |
| worst long frame | 1704.1 ms | **3623.8 ms** | **+113%** |
| interaction p95 | 1368.0 ms | 1440.0 ms | +5% |
| long frames | 233 | 235 | — |
| idle p95 frame | 16.80 ms | 16.70 ms | — (no cost when idle) |

**This is the plan's own thesis confirming itself, not refuting itself — but it
does refute an assumption the plan made.**

Phases 1 and 2 bound the cost of *rendering*. In this scenario frame time is not
dominated by rendering: it is dominated by **domain work on the render thread** —
SQLite executing as wazero-interpreted wasm, and a 2MB JSON decode — which
monopolizes the single wasm thread for hundreds of milliseconds at a stretch.

Against a thread-monopolizing workload, the Phase 1/2 changes actively hurt,
and the mechanism is clear: they all trade a longer wall-clock path for a more
interruptible one. P1.1 adds a scheduler hop per commit, P1.2 makes the work
loop yield far more often, P2.2 adds deferral passes. Every one of those hops
queues *behind* the blocking domain work. More yielding is only a win when the
thing you yield to is short.

Consequences, recorded rather than smoothed over:

1. **The flags stay off.** All three. This is R2 working exactly as intended —
   the discipline caught a regression that "it should be faster" reasoning would
   have shipped.
2. **Phase 3 is not one phase among six; it is the load-bearing one.** No amount
   of scheduler work fixes a scenario whose cost is domain work on the render
   thread. Only moving that work off-thread does.
3. **§1.2's scenario is Phase-3-shaped.** It was designed to prove the v5 thesis
   end to end, which means it cannot validate Phase 1/2 in isolation. Those need
   a render-bound scenario — a deep tree with heavy component bodies and no
   blocking domain work — which does not exist yet.
4. **The all-on arm conflates three changes.** Before any of them is judged
   individually, each needs its own arm.

Open work created by this result: per-flag arms, and a render-bound companion
scenario that can actually exercise what Phases 1 and 2 improve.

> **r10 is a structural rewrite, not a content change.** r9 had grown to 1,147
> lines, **44% of it blockquoted implementation mechanism** — checkpoint
> schemas, WAL pragmas, `RETURNING` patterns, publish ordering — accreted by an
> adversarial review loop that kept asking "what's the mechanism?" and kept
> getting answered at implementation depth.
>
> Those decisions are not the plan's to make. They belong to whoever implements
> each item, with measurements that do not exist yet. Freezing them now means
> either they are wrong and get overridden, or right by luck and constrained the
> implementer for nothing.
>
> Every mechanism has been demoted to one of two things: an **acceptance
> criterion** (what "done" means, no *how*), or a **named open design question**
> owned by its item (§11). Nothing was silently dropped; §11 is the receipt.
> Review history is in `v5-review-log.md`.

---

## 1. Objective

> Heavier work takes longer to **complete**. It never takes longer to **paint**.

Not a target: heavy work also finishing instantly. Chasing that builds the wrong
architecture.

### 1.1 Success criteria

**Gate** — 🔴 blocking (v5 does not ship without it) · ⚪ advisory (directional;
ship short of it with a recorded reason).

| ID | Metric | Baseline | Target | Gate | Owner |
|---|---|---|---|---|---|
| M1 | Frame time under sustained domain load vs idle | unmeasured | statistically equivalent | 🔴 | P0.3 |
| M2 | Long frames (>50ms) during background work | present | zero | 🔴 | P0.3 |
| M3 | Input-to-paint **p95** under load (+ max ceiling) | unmeasured | <50ms p95, <120ms max | 🔴 | P0.3 |
| M4 | 10k-row collection commit cost | O(dataset) | O(viewport) | 🔴 | P0.3 |
| M5 | `app.wasm` gzipped | 2.37MB | <1.6MB | ⚪ | P0.3 |
| M6 | runtime1 geomean vs React | 0.71x | no regression | 🔴 | P0.3 |
| M7 | GC max pause, render thread | ~9ms | <3ms | ⚪ | P0.3 |
| M8a/b | `cloneElementProps` alloc share / absolute allocs-per-op | 53% / — | <20% / −40% | ⚪ | PA.1 |
| M9 | Fine-grained update + list cost vs Solid, Svelte 5, one Rust peer, on the **js-framework-benchmark** protocol | never measured | within 2x | ⚪ | P0.5 |
| M10 | `services.wasm` size + time-to-first-command | never measured | <400ms | ⚪ | P3.10 |
| M11 | **Worker** resident memory: indexes + sqlite heap @20k rows | never measured | index <2x payload | ⚪ | P3.15a |
| M12 | **Render-thread** resident projection memory, and the GC pause it costs | never measured | sets `Resident()` default | 🔴 | P3.7 |

**M1–M4 + M6 + M12 are the ship gate.** M1 is the acceptance test for the
*thesis*; M6 is the guarantee we did not pay for it with single-threaded speed;
M12 bounds the mechanism that could quietly undo M1. The advisory metrics are
real targets whose miss is a recorded trade-off, not a release veto — M5 and M9
in particular measure against a floor (Go wasm size) and a moving competitor,
neither of which should be able to block a version that meets its own thesis.

M11 and M12 sit on opposite sides of the boundary and answer different
questions. Do not derive one from the other.

### 1.2 Reference scenario

`examples/testing/v5-load-harness` — instrument **built and tested** (42 tests
green); subject app is P0.2. Runs concurrently: 50k-row SQLite import ·
full-text re-index · 2MB JSON fetch+decode every 3s · 5k-row virtualized table
under scroll and filter · continuous typing (the latency probe).

Every claim of "flawless" is a claim about this harness.

---

## 2. Thesis

**The reconciler's diff is at the floor. The allocation path is not.**

`MICROBENCH_REPORT_arm64.md` says both, and early revisions quoted only the
first:

- steady-state child diff: **3.3µs, 0 allocs** — nothing left to win
- **`cloneElementProps` is the framework-wide allocation bottleneck** — 46–56%
  across four benchmarks, the report's own *"single highest-leverage
  optimization,"* deferred because the fix is architectural

Two tracks:

- **Track 1 (Phases 0–4)** — bound the main thread's work. Architecture.
- **Track A** — remove the allocation bottleneck. Constant factor.

**Not independent:** PA.2/PA.3 and P2.3 both rewrite
`internal/runtime/reconciler_elements.go`. PA.3 lands first.

**Invariants.** A proposal serving none of these is out of scope.

- **A** — the main thread does work proportional only to what is *visible*
- **B** — nothing outside the frame loop can perturb a frame
- **C** — no hot-path allocation a caller contract does not require

---

## 3. Threat model

| # | Threat | Evidence | Inv | Phase |
|---|---|---|---|---|
| T1 | Passive effects run before paint | `reconciler_commit.go:475-479` | B | P1 |
| T2 | Time-based yielding never fires | `continueWorkLoop` passes `globalInfiniteDeadline`; `TimeRemaining()` returns constant 1000 | A | P1 |
| T3 | Background work promoted, never deferred | `coalesceScheduledUpdateLocked:149` takes `min(lane)` | B | P2 |
| T4 | Async writes race the in-flight tree | `hooks.go:191-192` writes both state slots in place | B | P2 |
| T5 | No batching across tasks | no `Batch` API | B | P2 |
| T6 | Domain work on the render thread | sqlite = wazero-interpreted wasm inside app wasm | A | P3 |
| T7 | GC pauses on the render thread | max/median 1.74 vs React 1.20; ~9ms peaks | A | P3+A |
| T8 | Unbounded commit on large mounts | nothing caps per-pass size | A | P4 |
| T9 | Silent degradation cliffs | effect queue >1024 → full-tree scan; coalesce >4096 → dropped | A | P4 |
| T10 | Cold start | 2.37MB gz, 437ms TTI | A | P3/P6 |
| T11 | Props cloning dominates allocations | `MICROBENCH_REPORT_arm64.md:174-190` | C | PA |
| T12 | Interrupt restart can commit a childless root | `runtime_controls.go:143-156` → `reconciler.go:156` | B | P2 |

T3/T4/T5 share one root cause; P2.1 retires all three.

---

## 4. Engineering rules

- **R1 — no runtime1 regression.** M6 is a gate, not a report.
- **R2 — feature-flagged.** Each phase defaults to current behavior until its
  acceptance test passes. **Compound criteria (a/b/c) must pass in full**;
  partial passage does not authorize a flip.
- **R3 — measure before and after.** No measurement, not done.
- **R4 — no silent fallback.** Every degradation path emits a diagnostic.
- **R5 — v5 is additive.** No forced public API break. Two conditional
  exceptions, both declared: P1.3's `Scheduler` contract, and PA.3's
  `CreateElement` props contract *if* tie-break rule 2 is invoked.
- **R6 — diagnostics carry an allocation budget.** Precedent:
  `collectFlamegraphFrames` was 90.6% of all allocations.
- **R7 — post-flip rollback.** Every default-on flag ships with a kill switch.
- **R8 — the plan states *what* and *why*; items own *how*.** A mechanism
  belongs in an acceptance criterion or §11, never in the plan body. *(New in
  r10 — the rule whose absence produced the r6–r9 bloat.)*

---

## 5. Work plan

Effort: **S** ≤2d · **M** ≤1w · **L** ≤3w · **XL** >3w

**Serial chains — the real critical path:**

| Chain | Items | Minimum |
|---|---|---|
| Services | P3.3→P3.4→P3.5→P3.15a→P3.7 | ~18 weeks |
| Allocation→globals | PA.1→PA.2→PA.3→P2.3 | ~10 weeks |
| Projection prerequisite | P2.6→P3.7 | folds into Services |
| Windowing gate | P4.1 → Phase 3 exit | blocks Phase 3 close |

**Staffing assumption — the number these durations depend on.**
The chains above are *serial minimums*, and they are only parallel if separate
people run them. This plan assumes **two engineers**: one on Services
(Phase 3), one on Track A → Track 1 (Phase A, Phases 1–2), with Phases 0, 4,
and 6 absorbed by whoever is free.

**One engineer context-switching between chains roughly doubles the calendar
time**, because the chains are serial internally and the switch cost lands on
the longest one. At one engineer this is a 4–6 quarter program, and that should
change whether it is attempted, not just when it is expected.

**This is a 2–3 quarter program at minimum, at two engineers.**

### Phase 0 — Instrumentation

| ID | Item | Accept | Effort |
|---|---|---|---|
| P0.1a | Load-harness instrument | ✅ **DONE** — 42 tests green | — |
| P0.1b | Per-frame phase bucketing | one run emits per-frame reconcile/commit/effect/GC | S |
| P0.2 | Harness subject app + Playwright driver | emits M1–M4 on Windows CI | M |
| P0.3 | Capture v4 baseline → `v5-baseline.json` | every §1.1 metric has a number | M |
| P0.4 | Reconcile:commit ratio per scenario | documented ratio — input to P5 | S |
| P0.5 | Cross-framework benchmark beyond React | M9 has a number | L |

### Phase A — Allocation architecture

| ID | Item | Accept | Effort |
|---|---|---|---|
| PA.1 | Instrument `cloneElementProps` cost | M8 baseline recorded | S |
| PA.2 | Prototype 3 designs behind flags: Owned-migration · copy-on-write · slice-backed small-map | comparison table on all four affected benchmarks, **plus a selection by the §6 tie-break** | L |
| PA.3 | Ship candidate (c) — slice-backed small props | M8a <20%, M8b −40%, M6 holds, M7 improves; if the props contract breaks, a `gwc vet` lint and PA.3-owned migration note ship with it | L |

> **PA.3 blockers, found during PA.2 (r11).** The microbench report called this
> "architectural, not a micro-edit"; these are the specifics.
>
> 1. **136 direct `Element.Props` readers** — 66 inside `internal/runtime`, 70
>    outside. A slice-backed store needs lazy map materialization behind a view
>    helper (the pattern `EnsureElementProps` / `fastLanePropsView` already use
>    for the compact host lane) before any of them can be left alone.
> 2. **The runtime WRITES to the props map.** `buildElementWithHostProps` sets
>    `props["children"]`, and allocates a map purely to hold children when the
>    caller passed none. So props is not read-only caller data, and a storage
>    swap has to carry that write path.
> 3. **`props["children"]` is load-bearing, not vestigial.** `getFiberChildren`
>    falls back to it when `fiber.children` is empty — which is exactly the
>    direct-text case, where `Children` is deliberately `emptyChildren`. And
>    `propsEqualIgnoringChildren` (`reconciler.go:1004-1007`) special-cases the
>    key during diffing, so the diff path has to agree with whatever the storage
>    does.
>
> Sequence for PA.3: introduce the view helper and migrate the 66 in-package
> readers first, behind a flag, with the compact-host lane as the reference
> implementation. Only then swap the storage. The 70 external readers are the
> reason `EnsureElementProps` is already public — they keep working through it.
| PA.4 | Compile-time hoisting spike — **conditional on PA.3 missing M8b** | ceiling measured on 201 scenarios | L |

### Phase 1 — Frame integrity *(no API change)*

| ID | Item | Accept | Effort |
|---|---|---|---|
| P1.1 | Passive effects after paint | a 20ms passive effect no longer delays paint | M |
| P1.2 | Real frame deadline | no slice exceeds budget + one unit; **and** the deadline path cannot activate while P2.5 is absent — enforced in code, not process | S |
| P1.3 | Idle continuation via `RequestIdleCallback` | background-lane work runs in idle time; a no-op `Scheduler` impl is detected and falls back with a diagnostic | S |
| P1.4 | Effect-ordering regression suite | existing ordering tests pass unmodified | S |

**Exit:** M2 zero for effect-driven scenarios · M6 green · P1.2 gated on P2.5.

### Phase 2 — Frame isolation *(keystone)*

| ID | Item | Accept | Effort |
|---|---|---|---|
| P2.5 | Fix T12 interrupt restart | no commit of a childless root under mid-`performUnitOfWork` interruption. **Unblocks P1.2's flip** | M |
| P2.1 | Async inbox — one drain per frame | a goroutine writing state during a multi-slice pass renders exactly once, next frame; **empty-queue drain <100ns/frame** | M |
| P2.2 | Lane queues + expiration | background work never runs in an input-lane pass; a transition under sustained input completes within expiry; **allocation-free with no regression on `ReconcileChildrenStableList16`**; expiry resolution must not itself produce a long frame (M2) | L |
| P2.3 | Per-runtime state — **scoped to `internal/runtime`** | a behavioral two-runtime conformance suite: each fires its own ready hooks, resolves `RenderDetached` to its own container, honors its own strict-mode and diagnostic limits. `-race` clean is necessary, **not sufficient** — every global involved is mutex-guarded | L |
| P2.6 | Runtime-scope the atom registry | two runtimes hold independent registries; a projection in one is invisible to the other. **Hard dependency of P3.7** | L |
| P2.4 | Delete `EnqueueUI`/`ProcessUIQueue` | — | S |

**Exit:** T3/T4/T5/T12 closed with tests · `-race` clean · M6 green.

### Phase 3 — Off-thread domain *(additive — see D4)*

| ID | Item | Accept | Effort |
|---|---|---|---|
| P3.1 | ArrayBuffer transferables | binary payloads transfer, not structured-clone | S |
| P3.2 | Reduce mixed-ops rollback snapshot alloc | **benchmark the mixed-ops path first**; then report its measured contribution to browser `core-append` as a fraction | M |
| P3.3 | Extract service substrate from runtime2 | hello-world round-trips on all three transports **with a measurable definition of degradation** (SAB unavailable → falls back to push-on-change, diagnostic emitted, no error surfaced to the app); no regression on runtime2 compare benches. **Carries a contingency — see below** | XL |
| P3.4 | `domain.wasm` runtime | (a) atomic commands **replay** with no duplicate effects; (b) bulk commands **resume** — crash at row 30k of 50k finishes at exactly 50k; (c) cancelled commands do neither | L |
| P3.5 | SQLite off-thread behind `OpenOffThread` | (a) an app not using it runs byte-identically to v4; (b) one using it has no wazero symbol in `app.wasm`; (c) **a mixed app works with both models side by side** | L |
| P3.6 | OPFS backend, single-writer | incremental durability by kill-and-reopen; a second tab reaches a read-only state without throwing and surfaces it to the app; when OPFS is unavailable the IndexedDB tier is used with a diagnostic and no app-visible error | M |
| P3.15a | Delta publication engine — **precedes P3.7** | M11; delta ops are O(change) | L |
| P3.7 | Projection API | (a) incremental publish time flat 1k→5k→20k; (b) full path measured and documented O(N); (c) **the §1.2 filter keystroke performs zero domain round-trips**, asserted by message count; (d) a misspelled command does not compile; (e) M12 | L |
| P3.8a | Command failure — blocking subset | worker death mid-command and rejection surfacing both tested; a domain panic yields recoverable UI state | M |
| P3.8b | Timeout · retry · optimistic rollback | each has a test and a documented UI pattern | M |
| P3.10 | Two-artifact packaging | M5, M10; direct `Query` on an off-thread handle is a compile error | M |
| P3.11 | SSR bootstrap carries initial projections | a server-rendered route is `Ready()` on first render | M |
| P3.12 | Compute pool service | *(added r10 — this item had no criterion for nine revisions)* a CPU-bound job runs off-thread with bounded queueing and backpressure, and its worker restarts without losing queued jobs. **Blocked on §11-Q4** | M |
| P3.13 | Migration guide + CashFlux port | CashFlux runs on v5 with its e2e suite green | L |
| P3.14 | Domain hot-reload | editing a `domain.Handle` body reloads without dropping published projections | M |
| P3.15b | Escalation assistant | flags every `Query`/`Exec` on an escalated handle, classifies read/write, proposes a signature. **Not a codemod** — bodies are manual | M |

**Exit: M1 on the P0.2 harness.** The acceptance test for v5.
**Requires P4.1** — §1.2's 5k-row table is unbounded without windowing.

**P3.3 contingency.** P3.3 is the single largest item in the plan and everything
in Phase 3 sits behind it, yet it had no stated fallback — while Phase 5, a far
smaller bet, has a good one. Symmetric treatment:

- **Decision point:** after the transport extraction, before scheduler/recovery.
- **If the compare benches regress materially:** stop extracting. Ship Phase 3
  on a **purpose-built minimal transport** (structured-clone only, no shared
  memory, no idempotency ledger) and leave runtime2 intact and unreferenced.
  This costs performance headroom, not correctness — D2 already establishes the
  payload, not the transport, dominates.
- **If extraction proves cheaper than expected:** continue and pull P3.15a
  forward.
- Either way the decision is **recorded with its measurement**, not taken
  silently.

**P3.3 contingency — decision taken 2026-07-24: continue extracting.**

| Measurement | Value |
|---|---|
| `BenchmarkCompareLocalVsWorkerBackedRendering/local` | 2197 ns/op · 1472 B/op · 18 allocs/op |
| `BenchmarkCompareLocalVsWorkerBackedRendering/worker-backed` | 6485 ns/op · 7749 B/op · 43 allocs/op |

No regression: the transport extraction created a new package and modified no
runtime2 code, and the worker-backed ratio (2.95×) sits where D2 already places
it — protocol-bound, payload-dominated. The clause that would have stopped the
extraction did not fire, so the scheduler, recovery coordinator, and idempotency
ledger proceed.

One finding from the extraction changes what "extract" means for the remaining
items and is recorded here because it applies to all of them: **runtime2's
services delegate ordering to the patch parse layer above them.** The ledger
resets on any epoch mismatch, and the recovery coordinator accepts version
arguments it never reads — both safe there only because a stale epoch is
rejected before either is reached. A payload-agnostic service has no such layer,
so each extraction must carry its own ordering guard. These are deliberate
behavioral differences from the source, not ports, and each is pinned by a test
that names it.

### Phase 4 — Bounded commit

| ID | Item | Accept | Effort |
|---|---|---|---|
| P4.1 | Flat-list virtualization default-on | M4 | M |
| P4.2 | Workload budgets | every T9 cliff has a **defined** behavior (not merely a diagnostic) and a devtools signal. **Subject to R6** | M |
| P4.3 | Two-runtime devtools timeline | *(added r10)* a single timeline correlates render-thread and domain-worker activity by correlation ID, and a command's full lifecycle is traceable across the boundary | M |
| P4.4 | GC pacing pass | M7 | M |

### Phase 5 — Resolve runtime2

P5.1 re-measure component render share (method: P0.4).
**If <35%** → P5.2 retire runtime2 as a renderer, delete `dom_commit.go`,
`patch_*.go` (**L**). **If ≥35%** → P5.3 element-shipping hybrid (**XL**).
*Expected outcome: retire.* Stated as a falsifiable prediction.

### Phase 6 — Cold start *(parallel)*

> Phase 6 received the least scrutiny of any phase — nine review rounds
> concentrated on Phase 3. Four of its five items had no acceptance criteria at
> all until r11. Treat these as first-draft bars.

| ID | Item | Accept | Effort |
|---|---|---|---|
| P6.1 | `wasm-opt -Oz` + brotli in `gwc build` | measured size delta recorded per step; no runtime behavior change across the full test suite | S |
| P6.2 | Route-level code splitting | a route not visited never downloads its chunk, proven by a network assertion; initial payload shrinks measurably | M |
| P6.3 | Streaming instantiate + SSR shell | first contentful paint precedes wasm-ready, asserted on the timeline | M |
| P6.4 | Streaming HTML + out-of-order Suspense | a slow boundary does not delay delivery of the rest of the shell | L |
| P6.5 | Honest size story published | docs state the Go wasm floor, current numbers, and that React-bundle parity is not a goal — reviewed by someone outside the project for whether it reads as honest rather than defensive | S |

---

## 6. Decisions of record

**D1 — split by state ownership, not technical layer.**
UI instance (render, commit, view state) ↔ commands/projections ↔ `domain.wasm`
(sqlite + the logic owning it, single writer) ↔ compute pool (stateless).
Scale by adding **bounded contexts**, not layers. The business↔db seam is the
thickest data path in any app; cutting it serializes every query and spans
transactions across an async boundary.

**D2 — runtime2 is real multithreading; its overhead is protocol-bound.**
Flat at 2, 4, and 8 workers (`CHANGELOG.md:278`), worst case "core-append 9–11x
on full-dataset re-serialization." Workers rendering UI means the payload *is*
the output. Workers owning data means only a projection crosses.

**D3 — the component API changes by two hooks.** `UseProjection` (resident,
synchronous) and `UseCommand`. Local view state untouched.

**D4 — the off-thread split is opt-in per database.**
`sqlite.Open` stays in-process with `*sql.Rows`, unchanged and default.
`sqlite.OpenOffThread` returns a distinct type with no `Query`/`Exec` — two
constructors, because a variadic option cannot change a function's return type
and therefore cannot carry a compile-time guarantee. Escalating later costs a
constructor change and re-typed call sites; P3.15b exists because of that.

**D5 — COOP/COEP is a product decision.** Push-on-change is the default and only
required tier. Seqlock/SAB was **cut** — CashFlux's portal embedding cannot
enable cross-origin isolation, and a fast path most deployments cannot turn on
did not justify space in the breaking phase.

**PA.2 tie-break**, in order: (1) must hold M6; (2) must preserve the
`CreateElement` retained-props contract unless it is the only option clearing
M8a; (3) maximize M8b; (4) M7; (5) fewest call sites changed.

---

## 7. Non-goals

- Matching React's bundle size. Go wasm has a floor.
- runtime2 reaching runtime1 parity as a renderer — it pays runtime1's full
  commit cost plus encode, two `CopyBytes` crossings, decode, and verification.
- Interruptible commit. Bound it via virtualization instead.
- **Multi-runtime support across the public surface.** `router`, `hotreload`,
  `fetch`, `agentbridge`, `devtools`, `testkit`, and `ui`'s parallel-region
  bridge resolve through `GetGlobalRuntime()` (32 non-test call sites, 9
  packages) and stay single-global-runtime. A green P2.3/P2.6 suite means "two
  renderers with independent state and atoms" — not "N runtimes in a process."

---

## 8. Risks

| Risk | Sev | Mitigation |
|---|---|---|
| Serialization tax eats the win | High | P3.7's incremental publish, gated on wire size *and* publish-compute flatness |
| Second Go runtime memory on mobile | High | M10, M11, M12; one shared services instance |
| Residency re-imports GC pressure onto the render thread | High | M12 owns the budget; `Resident()` defaults from it |
| P2.3 destabilizes the hot path | High | R1 gate, flag, behavioral conformance suite |
| PA.3 breaks the props contract | High | A/B all three first; `gwc vet` lint; loud or not at all |
| P1.2 flipped before P2.5 | High | code-level gate, not process |
| Diagnostics become the bottleneck | Med | R6 |
| OPFS unavailable (Safari/iOS) | Med | verify Safari 17+ early; IndexedDB tier stays |
| WAL + OPFS crash-atomicity unverified | Med | **§11-Q7** — verify before P3.4/P3.6 compose |
| Scope: 2–3 quarter program | High | P3.1–P3.4 is a shippable substrate milestone; Track A ships independently |

---

## 9. Consistency invariants

Any edit must re-check these. Claims appearing in more than one place must agree.

1. Track A ↔ Track 1 parallelism (PA.3 precedes P2.3) — §2, §5, §10
2. Serial chains and program duration — §5, §8
3. M3 gates **p95** + max, not p99 — §1.1, `budgets.json`, `gate()`
4. M8 is two metrics — §1.1, PA.1, PA.3
5. Seqlock is **cut**; push-on-change only — §1.2, D5, §8
6. P2.3's boundary is `internal/runtime`; atoms are P2.6 — §7
7. Every metric has an owning item, in the phase that produces it
8. Every cross-phase prerequisite appears in §5's chains
9. Every risk names its *current* mitigation
10. Every "see guide X" names a guide whose scope covers it
11. A split-out sub-item precedes the parent whose criterion depends on it
12. Every persistent structure a phase introduces has an owning measurement
13. The revision header matches the newest content
14. Every universal principle survives §1.2's reference scenario
15. **No capability is named without either a mechanism or a §11 entry**
16. A metric constrains the side of the boundary it was written for
17. **No implementation mechanism appears in the plan body** *(R8)*
18. **Every work item has an acceptance criterion.** The one gap in this guard
    for ten revisions — it caught stale headers and misused metrics while three
    items carried an effort estimate and no definition of done.
19. **Every metric declares blocking or advisory.** A plan that cannot say what
    ships when a target is missed has not decided anything.
20. **Every XL item carries a contingency**, symmetric with Phase 5's. The
    largest bets are the ones that most need a stated fallback.

---

## 10. Start here

**P0.1b → P0.3 → P1.1**, with **PA.1 → PA.2 → PA.3** in parallel.
P1.1 is self-contained, needs no API change, and converts a guaranteed dropped
frame on every heavy effect into a bounded one.

---

## 11. Open design questions

Owned by their items, resolved at implementation time with measurements. **These
are deliberately unanswered** — answering them now would freeze decisions ahead
of the evidence needed to make them well.

| # | Question | Owner |
|---|---|---|
| Q1 | Does one `domain.wasm` bottleneck apps with several heavy contexts — N instances, or an internal work queue? | P3.4 |
| Q2 | Is a second Go runtime's memory acceptable on low-end mobile? | P0.3 |
| Q4 | Does the compute pool reuse `services.wasm` or get a third minimal binary? **Blocks P3.12's sizing.** | P3.12 |
| Q5 | Does compile-time hoisting justify a compiler pass, or does small-map props suffice? | PA.3 |
| Q6 | Why a bespoke protocol over the WASM Component Model? *(Current position: Go's js/wasm has no Component Model support and wasip2 cannot host a DOM-adjacent worker — **re-check before P3.3 starts**, and shape the transport interface so a CM backend could slot in.)* | P3.3 |
| Q7 | Does WAL-mode crash atomicity hold over OPFS sync access handles in `ncruces/go-sqlite3`? The bulk-resume guarantee depends on it. | P3.6 |
| Q8 | Where does a bulk command's progress record live, and what makes it survive a crash? *(Constraint: SQLite has no atomic commit across separate database files.)* | P3.4 |
| Q9 | How does incremental publish handle a mutation whose changed rows are not known until the statement runs, and projections derived from joins the handler did not touch? | P3.15a |
| Q10 | Is a domain panic worker-local or worker-fatal? Blast radius differs, and P3.4(b)'s resume semantics depend on the answer. | P3.8a |
| Q11 | When a lane's expiry fires, what happens? Forced synchronous completion could itself produce a long frame and threaten M2. | P2.2 |
| Q12 | What do T9's cliffs *do* once observable — reject, drop-oldest, degrade, block? | P4.2 |
| Q13 | Does `DependsOn`-style staleness detection stay signal rather than noise under chunked bulk commits? | P3.15a |
| Q14 | Does P3.3's "SAB tier" still mean what it meant before D5 cut the seqlock tier? | P3.3 |

### 11.1 Resolved

Recorded as their owning items shipped. A question is only moved here when the
implementation actually settled it; the rest stay open above.

| # | Resolution | Settled by |
|---|---|---|
| Q4 | **Reuse `services.wasm`; no third binary.** A third artifact adds a second Go runtime instance — the memory Q2 already flags as the mobile risk — and buys no isolation, because compute jobs are the app's own Go code either way. The dispatcher already routes across N workers over one binary, so the pool scales by worker count, not artifact count. A separate binary would only be justified by a different dependency set, and there is none. | P3.12 |
| Q8 | **In a `CheckpointStore`, and what makes it survive is whichever store the app supplies.** The constraint named in the question — SQLite cannot commit atomically across separate database files — is precisely why exactly-once is not claimed unconditionally: backed by the same transactional store as the effects, progress and effect commit together; backed by anything else, resume re-executes at most one checkpoint interval and `BulkResult.MaxReplayWindow` reports the bound. | P3.4 |
| Q9 | **Two publication paths, chosen by what the producer knows.** `Publish` scans the new state when the changed rows are not knowable in advance (a statement whose effects the handler did not enumerate, a projection derived from a join it never touched) and is O(N) in time while still emitting O(change) ops. `Apply` skips the scan when the producer can enumerate its own changes, and is O(change) in both. Measured: 5.95 ms vs 71 ns at 20k rows. | P3.15a |
| Q10 | **Worker-local.** A domain panic is contained at the command boundary, the command is not recorded as applied so it stays retryable, and the worker survives — which is the only thing that makes P3.8a's "recoverable UI state" mean anything. Blast radius is one command. Classified as a rejection rather than a death, so a retry policy does not treat it as possibly-applied. | P3.4, P3.8a |
