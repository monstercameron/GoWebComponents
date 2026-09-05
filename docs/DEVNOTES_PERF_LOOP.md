# Perf-loop dev notes

Running log of the benchmark→refine→regress→keep/revert loop against the
Example 201 browser benchmark (React 19 side-by-side). Rules: one refinement
per iteration, native + browser regression suites must stay green, a change is
kept only when a same-thermal-window A/B shows it positive (end-to-end report
runs swing ±25–30% per scenario on this fanless machine — the phase probe's
back-to-back toggle comparison is the trusted instrument). Goal: beat React
where we can with near-native execution.

Context (pre-loop state): same-run geomean vs React ≈ 0.79–0.83 band, 4–5
scenario wins per run. See CHANGELOG v4.1.0 for the campaign that got here.

## v5.0.5 closure — 2026-09-05: production renderer fast paths (KEPT SET)

The refreshed Example 201 harness exposed a different starting point after it
was corrected to compare a production GWC wasm build with the vendored React
19.2.4 production bundle. Using the explicit GWC/React DOM-ready ratio, an
untouched v5.0.4 worktree measured 2.614 across the 19-scenario geometric mean.

The retained set moves common typed hosts, event props, direct text, typed
component props, hooks, DOM mutations, subtree commits, serialized-tree binding,
and repeated plain-host allocation off their generic paths. The best observed
full report was 1.211 (6/19 scenario wins); the final normal-GC release-policy
run was 1.465. The original sub-1.0 research goal was not reached.

Two late experiments were rejected. Checking a browser deadline every 16 fibers
reduced clock calls but worsened DOM-ready latency by overshooting frame budgets.
Forcing a Go collection from an idle callback after large unmounts suppressed a
hook outlier but moved the pause into subsequent mount interactions, producing
repeatable 3–5x mount regressions. The release retains GOGC=300 and normal Go
collection scheduling.

## Iteration 1 — 2026-07-05: sibling-run serialized mounts (KEPT)

**Hypothesis.** Flat-list mounts (core/primitive/content render, the append
scenarios) still paid one `CreatePreparedElement` template-innerHTML parse per
row: each row is a one-host subtree, below `tryCommitSerializedSubtree`'s
3-host threshold, so the whole-subtree serializer never fired for them.

**Change.** `prepareSerializedSiblingRuns` (reconciler_commit_subtree.go):
when a parent is about to batch-append ≥2 placements (`isParseBatching` gate —
update-only passes pay nothing), maximal runs of consecutive eligible
placement siblings are serialized into ONE HTML string, parsed with one
`CreateHTMLFragment` bridge call (template content), and bound member-by-member
via the existing zip walk. Members keep `effectTagPlacement` with dom pre-set,
so the normal commit recursion appends them in sibling order through the
existing multi-arg append batch. Ineligible members close the run and fall
back per-node. mockdom implements the fragment capability via x/net/html, so
the native suite exercises the path (`TestSerializedSiblingRunMountFires` pins
it: flat-list mounts must produce parsed text nodes, never per-row
setTextContent).

**Debuggability.** Same fallback ladder as the single-subtree path — remove
the capability (or return early) and everything reverts to per-node prepared
elements; the run preparer has no side effects until a run flushes.

**Measured (same-window phase-probe A/B, ON vs OFF, phaseSum per 9 runs):**
core-render 17.0→18.0 OFF, core-append 15.8→17.2 (domReady 12.4 vs 16.1,
−23%), content-render 8.7→10.0, primitive-render 9.4→12.3 (commit 5.8 vs 8.6,
−33%), hooks-render 14.7→15.8; deep-render flat (nested cards already take the
single-subtree serializer). ON wins 5/6. The scored report run's core-append
outlier (9.1ms) contradicted the probe and reproduced neither direction —
treated as noise per the ±30% rule.

**Regression:** full native suite + unit lane green; full browser suite
(runtime1 + runtime2 workers) green; both serialization pinning tests green.

## Iteration 2 — 2026-07-05: typed ui.UseMemo pass-through (KEPT)

**Hypothesis.** The wasm `ui.UseMemo[T]` wrapper allocated a `func() any`
adapter closure and ran `reflect.TypeOf((*T)(nil)).Elem()` on EVERY call —
800×/render in hooks-render — only to feed a hot-reload coercion target that
matters solely on the restore path.

**Change.** `runtime.GoUseMemoFor[T]` (hooks.go): typed compute passes through
unwrapped; the memo prologue is factored into `ensureMemoSlot` (shared with
the untyped entry); `reflect.TypeFor[T]()` runs only inside the hot-reload
restore branch. `ui.UseMemo` is now a one-line delegate. Untyped
`GoUseMemo`/`GoUseMemoGlobalTyped` remain for map-props/legacy callers.

**Debuggability.** Same memo slots, same signature ("memo"), same hot-reload
snapshot format — devtools/agent introspection unchanged.

**Measured (same-window probe A/B, hooks-render):** typed 8.97ms domReady /
4.21ms render vs old 10.26 / 4.58 — **−12.5% domReady, −8% render phase**.

**Regression:** native suites + unit lane green both arms; hot-reload memo
restore covered by existing runtime tests.

## Iteration 3 — 2026-07-05: typed hooks in the subject + wasm-node instrument (KEPT)

**New instrument.** The hooks mirror benchmark now compiles for wasm (build
tag removed) and runs under node via `tools/go_js_wasm_exec.bat` — attributable
wasm-side numbers without a browser. First reading: the untyped hook walk
costs **1.48ms/pass in wasm vs 0.57ms with GoUseMemoOf/GoUseEffectOf (2.6x)**
— wasm amplifies the closure/deps allocations ~14x over native.

**Fairness finding.** The GWC subject's hook cell ran `ui.UseEffect(fn)` with
NO deps — re-running 800 effects every render — while React's cell uses
`useEffect(fn, [])`, which runs ONCE. The GWC subject was doing strictly more
work than the React subject it is compared against.

**Change.** `renderBenchmarkManyHooks` now uses `ui.UseEffectOf(static,
index)` (run-once, matching React's `[]`) and `ui.UseMemoOf(static, index)`
(matching `[parseIndex]`), with static top-level hook bodies — no closure
captures.

**Measured.** wasm-node walk: 2.6x (above). Browser probe: flat
(8.89ms vs 8.97ms domReady — the browser scenario is no longer hook-walk
bound; residual is the UseState global-runtime path, component dispatch, and
GC). hooks-render absolute across scored runs: 8.3 → 7.4ms. Scored geomean
0.784, inside the established 0.75–0.83 noise band.

**Kept because:** subject semantics now mirror React's cell exactly (the
comparison was biased against GWC), the typed-hook walk win is decisively
measured in the wasm instrument, and no scenario regressed outside noise.
Full unit lane + full browser suite green.

## Iteration 4 — 2026-07-05: text children in serialized mounts (KEPT)

**Hypothesis.** deep-render (0.53) never took the serialized-mount path: each
deep-tree layer mixes a text label with a nested element, and the serializer
rejected any TEXT_ELEMENT child — so the 61-level chain paid 61 per-node
creations + 61 text-node bridges every mount.

**Change.** `serializeMountSubtree` inlines plain text children (escaped),
with two zip-alignment guards: empty text (parses to no node) and adjacent
text runs (merge into one node) fall back to per-node mounting. TEXT_ELEMENT
fibers now defer render-phase DOM creation like fast-lane hosts (any that end
up outside a serialized subtree are created by the commit placement branch —
same bridge call, later). Added `Runtime.SerializedMountRoots()` — an
observable counter for whether the fast mount paths fire, used by the new
`TestSerializedMountHandlesMixedTextChildren` pin (mount + post-mount text
update through the bound text-node fibers).

**Measured.** Phase probe: deep-render domReady 7.8–9.4ms → **2.81ms**
(commit 3.3–4.3 → 1.46ms); content-render 4.9ms. Scored run: deep-render
ratio 0.53 → 0.86, deep-update flipped to a **1.10 win**, geomean 0.788 with
4 wins; no scenario regressed beyond the noise band.

**Regression:** full native suite + unit lane green; full browser suite
(runtime1 + runtime2 workers) green; all three serialization pins green.

## Iteration 5 — 2026-07-05: bulk sibling deletions via ReplaceChildren (REVERTED)

**Hypothesis.** primitive-remove pays one removeChild bridge hop per deleted
row (100 hops); the survivors are known, so one ReplaceChildren(survivors)
call should beat the per-node walk.

**Built.** Grouped same-parent deletions (threshold 8), collected survivor
top-level DOM nodes in old-tree order, one ReplaceChildren per parent, with a
self-clearing `bulkDetached` fiber flag so commitDeletion/deleteFiberSubtree
still ran the full unmount lifecycle (verified by a pin counting effect
cleanups) while skipping the DOM hop. Mechanism worked — the pin showed one
replaceChildren, zero removeChild, lifecycle intact.

**Measured (same-window probe A/B).** primitive-remove: ON 3.04ms domReady /
1.83ms commit vs OFF 2.57 / 2.04 — commit −10% but domReady +18%, both inside
noise. Chrome's removeChild is evidently cheap for trailing-row removal;
there was no bridge-hop bottleneck to save.

**Reverted per the only-keep-positive rule.** Kept from the attempt:
mockdom.ReplaceChildren (browser-fidelity move semantics — also gives the
existing wholesale child-order path native coverage) and primitive-remove in
the phase-probe scenario list. Lesson: deletion is not a bridge-bound path;
don't revisit without new evidence.

## Iteration 6 — 2026-07-05: fastEqual interface-identity fast path (REVERTED)

**Hypothesis.** Stable UseEvent handler wrappers (js.Func, an incomparable
struct) survive props equality only through reflect.DeepEqual — paying it per
handler per element per pass across the ~50-button app shell in refresh
scenarios.

**Built.** A two-pointer-load identity check at the top of fastEqual (same
type descriptor + same boxed data pointer => equal; Object.is semantics for
same-boxed NaN), plus a semantics pin (`TestFastEqualIdentityFastPath`).

**Measured (same-window probe A/B).** Refresh trio and update scenarios FLAT
in both directions. The hypothesis missed how the scheduler works: granular
dirty marking means clean shell components never re-render at all during
refresh scenarios, so handler compares never run on the hot path.

**Reverted with a NOTE at the fastEqual site;** the semantics pin stays (it
covers the DeepEqual fallback for incomparable types either way). Probe
scenario list now includes core-refresh and content-refresh permanently.
Observation for the next iteration: core-update's probe reading was dominated
by a 5.0–5.6ms GC pause, not phase work — remaining update-path gains live in
allocation reduction or GC pacing, not algorithm changes.

## Iteration 7 — 2026-07-05: gated mutex idea; in-order trailing appends (KEPT)

**Gate that saved a build.** wasm-node micro-benchmark of GetGlobalRuntime
(the per-hook mutex on the ui.UseState path): **26ns/op** — 800 calls/render
= 21µs, irrelevant. Lock-free-global idea rejected on evidence before any
code. Benchmark kept as the gate record.

**Real target from the new append mirror** (`BenchmarkMirrorCoreAppend`,
200→300 keyed rows): 22% of the pass's allocations were KEY BOXING —
`fiberReconcileKey`/`elementReconcileKey`/`fiberComparableKey` box every
string key into `any` because trailing appends failed the in-order trial
(old chain exhausts → bail to the map[any]*Fiber path, ~600 boxes/pass).

**Change.** `tryReconcileKeyedChildrenInOrder` now accepts elements remaining
after the old chain is exhausted: validated prefix updates in place, the tail
mounts as placements — list growth never touches the boxing map path.
Deletions/reorders still fall back exactly as before. Pinned by
`TestKeyedTrailingAppendKeepsPrefixIdentity` (prefix DOM identity preserved,
tail in order, no removals).

**Measured.** Native append mirror: 4720→3920 allocs (−17%, exactly the
boxing), 320→~280µs (−12%). Browser probe A/B: flat in a cool window
(7.56 vs 7.76ms domReady; earlier hot-window readings were 12–16ms).
Kept: native-positive, browser non-negative, strictly less work, and appends
are the dominant list-growth shape in real apps.

## Iteration 8 — 2026-07-05: re-baseline and loop conclusion

Final scored run: geomean 0.733 (React drew unusually fast sub-ms readings
this window). Band across the loop's five scored runs: **0.73–0.83**, with
3–5 outright wins per run and the winning scenarios ROTATING between runs
(enterprise-subtree 1.35 / deep-refresh 1.30 / primitive-attribute 1.01 this
time; deep-render 1.57 / content-update 1.34 the run before; core-stress 1.61
before that) — every scenario except hooks-render, core-update and
content-refresh has reached ≥0.85 in at least one recent run.

**Loop verdict: noise floor reached.** The last three refinement attempts
were built soundly and measured honestly; only one survived the
keep-if-positive rule. Per-scenario run variance (±30%) now exceeds the
expected effect of any remaining single-pass refinement. Stopping here per
the loop's own discipline.

**Loop totals:** 5 kept (sibling-run serialized mounts, typed ui.UseMemo
pass-through, typed subject hooks + fairness fix, text children in serialized
mounts, in-order trailing appends), 2 built-and-reverted (bulk deletions,
fastEqual identity), 2 gate-rejected before building (microtask kick was
pre-loop; GetGlobalRuntime mutex at 26ns/op). New permanent instruments:
wasm-node benching of test/render, SerializedMountRoots counter, GC counters
+ refresh/remove/attribute scenarios in the phase probe, append/hooks native
mirrors.

**Remaining known gaps are project-scale, not loop-scale:** hooks-render
(wasm hook-walk factor + GC), core-update (GC pause dominated), boot wire
size (brotli −28% is a server-config win), runtime2 worker protocol
(transferables/deltas). See gwc-eight-area-research memory / CHANGELOG.

# Four-edges loop (restarted 2026-07-05)

## Edge loop iteration 1: interactive GC pacing as a framework default (KEPT — biggest single lever of the campaign)

**Research.** Edges 1 and 3 (hooks-render, GC-pause victims) share one root:
Go's GOGC=100 collects every time the heap doubles over a tiny live set, so
an interactive wasm app takes a stop-the-world collection every few MB of
render churn — pauses up to ~9ms landing inside 1–3ms interaction windows.
V8 never pays this; its nursery collects tiny garbage incrementally.

**Change.** `ui/gc_pacing.go` (js&&wasm): every GWC browser app now inits
with GOGC=300 + a 512MB `debug.SetMemoryLimit` backstop, applied in
`ensureInitialized` before the runtime boots. Debuggable: the applied policy
is reported as an info diagnostic; `localStorage["gwc:gogc"]` overrides
("off" = Go defaults, integer = custom percent). The benchmark subject's
private knob was removed in favor of the framework override; the GC probe
test now A/Bs `off` vs the framework default.

**Measured (same-window A/B, framework default vs Go default):**
hooks-render 13.0→7.4ms mean / 26.0→17.3 max / 5→1 collections;
core-update 4.5→2.4ms (max 7.9→3.4); core-stress-update 6.6→4.5ms, 4→0
collections. Scored run after: **geomean 0.936** (previous band 0.73–0.83),
**8 outright wins** (enterprise-subtree 1.49, deep-refresh 1.39, deep-update
1.32, primitive-attribute 1.28, core-update 1.10, content-update 1.07,
primitive-text 1.02, core-stress 1.01) + 3 near-parity; hooks-render best
absolute ever (5.3ms, ratio 0.41). Full unit lane + full browser suite green.

**Trade-off accepted:** heap grows to ~4x live before collecting (bounded at
512MB). For a browser tab with a few-MB live set this is single-digit-MB
headroom in practice; apps that care can override or disable per origin.

## Edge loop iteration 2: adversarial reviewer findings — build-profile fairness + ungated dev costs (KEPT)

The adversarial reviewer (sequential Sonnet) audited the hot paths and found
what the whole campaign missed:

1. **Every scored run to date compared a GWC DEV build against React's
   PRODUCTION bundle.** `buildExample201BenchmarkWasm` never passed
   `-profile`, which resolves to "development" (no `-tags production`) —
   leaving the commit timers (2 clock reads × up to 5 sites per fiber) and
   the hook threading guard active in every benchmark while React ran
   `react.latest.production.js`. FIXED: the scored report and boot probe now
   build with `-profile benchmark` (production tags, -s -w, trimpath); the
   runtime2 worker builds with the same flags; the phase/GC probes keep an
   explicit dev build (their instrumentation is compiled out of production).
2. **Two raw `time.Now()/time.Since()` pairs bypassed the gating helpers** on
   the hottest dispatch paths — `performUnitOfWork` (every non-bailed fiber)
   and `renderFunctionComponent` (every component render). FIXED: routed
   through `commitTimingStart/SinceNs`, so production builds strip them like
   the commit-side timers.
3. **`reportMissingKeys` ran an ungated O(N) full-child scan per
   reconcileChildren call** purely for a dev warning (the keyed-dispatch
   check it duplicates short-circuits on the first key). FIXED: gated behind
   the production split; three dev-diagnostic tests now skip under
   `-tags production` (two of them were pre-existing failures under that tag).

Reviewer findings ranked lower and deferred: single-key Data/Aria scalar
fast path (needs alloc A/B), BatchSetAttributes rename/reroute (nearly dead
on hot paths).

**Measured (first production-vs-production run):** geomean 0.845, 4 wins + 6
near-parity; **hooks-render 0.51 — best ever** (guard + timers gone).
Cross-run comparison to the dev-build 0.936 run is not meaningful — the new
configuration needs its own band; sub-ms scenarios (core-refresh,
content-refresh) still dominate variance. `go test -tags production
./internal/runtime` is now green and part of the loop's regression set.

## Edge loop iteration 3: production-config band confirmation

Second scored run: **geomean 0.955, TEN outright wins** (enterprise 1.65,
content-update 1.49, deep-render 1.39, content-refresh 1.27, core-stress
1.23, core-render 1.21, deep-update 1.19, primitive-attribute 1.04,
content-render 1.03, primitive-append 1.02). Production band after two runs:
**0.845–0.955**; 12 of 19 scenarios have beaten React outright in at least
one run. Edge status: cold mounts effectively closed (core-render won BOTH
runs, 1.21/1.30); GC victims mostly flipped (content-refresh 0.45→1.27
across runs); structural residuals are hooks-render (0.42–0.51) and
primitive-remove (0.50–0.70). Next: the reviewer's deferred single-key
Data-map allocation finding (one map alloc per row per render feeding the GC
edge).

## Edge loop iteration 4: Props.DataAttr zero-alloc single data-* attribute (KEPT)

**Research.** The reviewer's deferred finding: carrying ONE dynamic data-*
attribute through the typed fast lane forced a `map[string]string` allocation
per element per render (bucket alloc + hashing) — every core row pays it.

**Change (additive API).** `html.Props.DataAttr DataAttribute{Name, Value}` —
a value field, zero allocations — consumed by both construction lanes and
sorted into the same deterministic data segment as Data-map entries (names
interned like the map path). The Data/Aria maps remain for multi-attribute
cases. Pinned by `TestDataAttrSingleAttribute` (compact lane, map lane,
coexistence + segment ordering). Adopted in the mirror rows and the
benchmark subject's core rows (primitive rows already avoided per-row maps).

**Measured.** Native mirrors: core-update 340→260 allocs (−24%, bytes −28%),
core-append 3920→3320 (−15%), core-refresh 303→223 (−26%, bytes −45%).
Browser scored run: geomean 0.881 (mid-band; three production runs now
0.845/0.955/0.881), 8 wins including core-append 1.37 and core-stress 1.19;
**hooks-render 0.60 — best ever, third consecutive improvement** as
allocation cuts compound with GC pacing. All suites green including
`-tags production`.

## Edge loop iteration 5: reviewer round 2 — dead registry scan, merged teardown walks, Props.Text (ALL KEPT)

Three findings from the second adversarial pass shipped together:

1. **Dead registry-wide scan per deleted fiber (B1).** A fiber with empty
   `hooks.atoms` and empty `reactiveSourceIDs` has provably never subscribed
   (every Subscribe call site records the id on the fiber — audited), yet
   `CleanupAtomSubscriptions` fell through to `UnsubscribeFiberFromAll`,
   taking the registry mutex and ranging ALL subscription sets — hundreds of
   times per bulk removal. Now short-circuits; the recording invariant is
   pinned by `TestPlainFiberDeletionSkipsAtomRegistry`, and one legacy test
   that subscribed directly without recording was aligned with the real
   API pattern.
2. **Deletion teardown walks merged (B2).** commitDeletion walked every
   deleted subtree three times (cleanups, atom unsubscribe, ref release).
   The two order-independent bookkeeping phases now share one walk
   (`teardownDeletedSubtree`); `runCleanups` stays a separate first pass so
   user cleanup callbacks can still read descendant refs.
3. **`Props.Text` direct text content (C1).** `html.Text(child)` allocates a
   full throwaway Element that Tag discards after extracting the string —
   one of the highest-frequency allocation sites in text-heavy trees. The
   new additive field feeds `CreateElementCompactHostOwnedText` directly
   (map lane falls back to a text child). Pinned by
   `TestPropsTextDirectContent`; adopted in mirror + subject core rows.

**Measured.** Native mirrors (cumulative from pre-iteration-4 baselines):
core-update 340→**180** allocs (−47%), core-append 4720→**2720** (−42%),
core-refresh 344→**142** (−59%, bytes 30k→10.9k). Browser scored run:
**geomean 0.952** — band top — with 8 wins (core-stress 1.86, deep-render
1.27, primitive-append 1.26, enterprise 1.25, content-update 1.20,
deep-update 1.13, core-filter 1.11, primitive-attribute 1.05) and
**hooks-render 0.73 at 4.0ms absolute — both best-ever by a wide margin**.
Production-band runs to date: 0.845 / 0.955 / 0.881 / 0.952. All suites
green incl. `-tags production`. Deferred (gate before building): render-trace
lazy activation (A1), signature byte-enum (A2), compact-props single scan (C2).

## Edge loop iteration 6: deferred-findings gates — all rejected; loop concluded

A1 (render-trace lazy activation) gated on the wasm-node hooks mirror:
recording ON 592–702µs vs OFF 549–663µs — overlapping ranges, ~5% at best,
against a real behavior change to the agent-bridge introspection surface.
Rejected. A2 (signature byte enum) and C2 (compact-props single scan) have
smaller expected wins with more touched sites; rejected without building,
per the reviewer's own caveats.

**Edge-loop conclusion.** Six iterations: 5 substantive keeps (interactive GC
pacing default; production-vs-production benchmark fairness + two ungated
dev-cost fixes; Props.DataAttr; dead-registry-scan removal + merged teardown
walks + Props.Text), 2 adversarial review rounds (both productive — round 1
found the build-profile bug the whole campaign missed), 4 gate-rejections
that cost nothing. **Production-vs-production band: 0.85–0.95 geomean,
8–10 wins per run.** Edge scorecard: cold mounts CLOSED (core/deep-render
at/above parity), GC victims CLOSED (pacing default + alloc cuts; core-update
and the refresh trio flip wins run to run), primitive-remove improved to
0.74–0.89 (registry-scan + teardown fixes), hooks-render 0.27 → **0.73 /
4.0ms absolute** — a 2.7x ratio improvement; the remainder is the wasm-vs-JIT
execution factor on 2,400 hook calls, a toolchain-economics boundary, not a
framework defect. "Flawless" per the loop's own noise floor: reached.
