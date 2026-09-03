# Changelog

## v5.0.3 - 2026-09-03

This release completes the v5 promotion path after the earlier tagged builds
stopped in CI before GitHub could publish them.

- The docs-site release-note mirror now tracks the latest changelog section.
- Release CI builds the generated Atlas and load-harness wasm assets before the
  clean-checkout unit-test gate.
- `google.golang.org/grpc` is upgraded to v1.82.1 and
  `github.com/yuin/goldmark` to v1.7.17, resolving the reachable
  GO-2026-6061 and GO-2026-5320 advisories reported by `govulncheck`.
- Playwright-Go moves to `github.com/mxschmitt/playwright-go` v0.6201.1, whose
  npm-based driver installer replaces the retired Azure CDN used by v0.5700.1.

## v5.0.2 - 2026-07-28

**No framework behaviour changed.** Examples, documentation, measurement and CI
only; the sole edit under `internal/` is a gofmt blank line in
`inspect_reporting.go`. The substance is that v5 now has a real application
exercising it, and that measuring it invalidated several numbers the plan was
relying on.

### The measurements were wrong, and the instruments were the reason

`PRODUCTION_READINESS.md` carries a banner over every gate number recorded before
2026-07-26, because they were taken with a probe that generated no input:

- `driveTyping` in the P0.2 harness is a `setTimeout` that types nothing, and
  always had been. Event Timing only records **trusted** events, so page script
  cannot produce input at all — it has to come from the automation driver.
- **M1's recorded pass was an artifact.** Equivalence between an idle arm and a
  "loaded" arm whose probe also did nothing is trivially true no matter what the
  runtime does.
- **M2 and M7 were reporting zeros from dead instruments.** Six deliberately
  injected 180 ms main-thread blocks went uncounted; LoAF does not fire in
  headless Chromium, and M7 sampled no collection at all.
- **The measurement machine was contended.** 247 livereload dev servers leaked by
  `tools/gwc`'s dev-loop tests were still running, the oldest two days old. Fixed
  at the source with `killListenersOnPort` in `tools/gwc/start_test.go`.

Re-measured headed, with real trusted keystrokes on a quiet machine (four runs,
~650 interaction samples each, all three §1.2 workloads confirmed running).

### What the honest numbers showed about M2

Long frames are **coalesced update batches, not slow rendering**. Per-keystroke
Go phase totals:

| | commits | work units | time |
|---|---:|---:|---:|
| typical keystroke | 2.2 | 89 | **8.2 ms** |
| worst keystrokes | 4–5 | 153–154 | **82–131 ms** |

2.3× the commits and 1.7× the work produce **16× the time**; that superlinearity
is the finding. When keystrokes arrive faster than the loop drains them, several
commits execute inside one frame. Four hypotheses were tested and refuted, each
with a control: worker chatter (2 messages/second), GC (`gc=0` on the worst
frames), style/layout (`styleAndLayoutDuration = 0`), and environment preemption
(1200 idle frames in 20 s, zero long). Load is not the variable — input arrival
rate is. Enabling `FrameBudgetMs` made it measurably worse, which is two
independent scheduling attempts reaching the same answer. `internal/runtime` is
0.43% of the native render path; there was never a constant factor there worth
16×. The fix that followed is the drain-between-frames change already on this
branch, not a faster reconciler.

### Atlas Commerce OS is now the reference v5 application

`PRODUCTION_READINESS.md` said it plainly — *"no real application exercises v5"* —
and the flagship server example (~29k lines, 90 Go files) now does, with new
packages written to be read rather than merely to work:

- **`shared/design`** — the app's design system authored entirely in Go on the
  typed-CSS package. No Tailwind step, no `.css` file, no CDN: every rule is a Go
  value folded into a hashed class and emitted through the `Sink`, so the native
  SSR lane and the wasm lane share one authoring surface.
- **`shared/bootfallback`** — owns every byte of JavaScript the document ships
  (`wasm_exec.js` plus a ~20-line generated snippet) and the markup that snippet
  reveals when the client cannot start. The fallback copy, styling, `<noscript>`
  content and failure-message selection are all Go.
- **`shared/api`** — adopts `//gwc:server` server functions for the typed
  client/server boundary, with the `!js || !wasm` constraint doing load-bearing
  work: it keeps the real implementations, the driver and the query layer out of
  `app.wasm`, which is what the M5 size budget and the two-artifact split exist
  for.
- **`client/bootsurface.go`** and a large `public_sections.go` rework.

### CI gained the gates that would have caught this

- **`atlas-boot.yml`** — Atlas was in **zero** CI gates (`examples-build.yml`
  covers `./examples/public/...`; Atlas lives under `examples/server/`) and was
  consequently broken at boot from 2026-04-08 while every collected signal read
  green: it compiled for both targets and its 7 test packages passed.
- **`server-functions.yml`** — guards the `//gwc:server` codegen contract. Editing
  a signature without re-running `gwc server gen` leaves a stub that still
  compiles and still calls the old contract; the compiler cannot catch it because
  both sides are individually valid, so it fails at runtime in a browser.
- **`catalog-smoke.yml`** — now derives the module path *and* the require version
  from `go.mod` instead of hardcoding `github.com/monstercameron/GoWebComponents`,
  which stopped resolving at the `/v5` move and silently sent the gate to the
  network for the local checkout.

### Also

- The livereload development server now falls back to the configured app shell for
  missing HTML navigation routes, so history-router reloads work while missing
  static assets continue to return 404.
- `docs/ATLAS_PERF_BASELINE.md` — the measurement pass, every number carrying the
  command that reproduces it, hot spots that were *expected and did not appear*
  reported as such (harness share of the native allocation profile: 0.015%).
- `docs/V5_DOMAIN_WORKER_ASSESSMENT.md`, plus `docs/plans/v5-plan.md` updates
  recording the M2 investigation and its correction.
- `examples/testing/v5-load-harness` — `probe.go` and a `gate.test.mjs` so the
  harness's own instruments are tested; `examples/testing/atlas-perf` fixtures;
  `examples/v5-two-artifact` app/services rework.
- Additional `test/playwrightgo` example coverage.

## v5.0.1 - 2026-07-28

**Fix: `css/u` no longer collides with `html/shorthand` under dot-import.**

`css/u/exports.go` re-exported seven names that `html/shorthand` already
declares, which broke the exact pattern that file exists to enable — dot-import
`u` alongside `shorthand`. Any file doing both failed to compile with
`X redeclared in this block`:

| Name | Clashing `html/shorthand` declaration |
|---|---|
| `Track` | `<track>` element |
| `Repeat` | node repetition helper |
| `LinearGradient` | `<linearGradient>` SVG element |
| `RadialGradient` | `<radialGradient>` SVG element |
| `Stop` | `<stop>` SVG element |
| `Circle` | `<circle>` SVG element |
| `Ellipse` | `<ellipse>` SVG element |

Those seven are no longer re-exported under `u`; reach them as `css.Track`,
`css.Repeat`, `css.LinearGradient`, `css.RadialGradient`, `css.Stop`,
`css.Circle`, `css.Ellipse`. This follows the rule the file already applied to
`Gap`, `Bg`, `Rounded`, `Border`, and friends — a name that would shadow stays
qualified. Unambiguous neighbours (`LinearGradientTo`, `RepeatingLinearGradient`,
`RepeatingRadialGradient`, `RepeatFit`, `RepeatFill`, `StopAt`, `StopSpan`,
`CircleAt`, `EllipseAt`, `CircleSizedAt`) are untouched and stay bare.

Source-compatible for any code that was already compiling, since code that
dot-imported both could not build at all, and nothing in this repository
referenced the removed aliases.

## v5.0.0 - 2026-07-25

**Major release: the module path is now `github.com/monstercameron/GoWebComponents/v5`.**
Update your imports; `go get -u` will not move you here on its own, which is the
point of a major version.

### Why this is a major

`ui.ParallelRegion`, `ui.RegisterParallelRegion`, and the surrounding
parallel-region API are **removed**, and `ui.Hydrate`/`ui.HydrateInto` lose their
parallel-region bridge parameter. Rendering across workers was measured at ~11%
— a rounding error against its cost in complexity — so runtime2 was retired as a
renderer. That is a breaking change to exported API, which `VERSIONING.md`
requires a major bump for. The branch previously carried it under `/v4`; this
release corrects that.

### The v5 thesis, measured

Heavier background work should take longer to COMPLETE, never longer to PAINT.
With the three §1.2 workloads relocated to a domain worker, the harness reports a
loaded p95 frame of **16.70 ms against a 16.70 ms idle frame** — equivalent,
against **1633.30 ms** for the same workloads on the render thread in v4.

### Added

- **`ui.PostAsync`** — the supported way for a worker reply, gRPC callback, or
  goroutine to change rendered state. Work is queued and applied at one defined
  point per frame, so an async write cannot land mid-render, and writes posted
  together produce one render rather than N.
- **`Config.AsyncIngress`** (off by default) — routes state setters called
  outside the frame loop through that same inbox, so existing async code becomes
  safe without being rewritten.
- **`domain.Runtime.SetYield` / `BulkCommand.YieldEvery`** — a bulk command can
  now return to its worker's message loop, which is what makes it cancellable at
  all. Previously the cancel sat in a queue behind the run it was meant to stop.
- **Two-artifact packaging** — `app.wasm` renders, `services.wasm` owns the
  engine. The example carries a working transport, and a test fails the build if
  `app.wasm` ever links SQLite, checked on the dependency graph.
- `domain`, `delta`, and `escalate` promoted out of `internal/`, so adopters can
  actually import what the migration guide tells them to.

### Fixed

- **Lane deferral discarded the update it deferred.** The pass that declined a
  fiber cleared the same lane bit it had just set, so no follow-up pass was
  scheduled and nothing stayed dirty. Reachable only with `LaneQueues` on.
- **`WorkerClient.Deliver` could hang the JS event loop.** It read a pending
  reply slot without claiming it, so a duplicate reply — or `Deliver` racing
  `WorkerDied` — sent twice into a one-slot channel and blocked forever, inside
  the message callback.
- **Transaction rollback ran on the context that had just died.** Cleanup reused
  the caller's context, which is usually cancelled precisely when rollback is
  needed, leaving the transaction and its write lock open. A failed commit did no
  cleanup at all.
- **The serialized mount skipped most real markup.** It required both compact
  attributes and no props map; the second condition is about how an element was
  constructed, not whether it can be serialized. Elements built through
  `runtime.CreateElement` with a string props map now mount in one bridge call —
  measured 195 crossings to 2 on a 12-card tree.

### Known open

Honest accounting rather than a clean bill:

- **M2 (zero long tasks) and M7 (3 ms pause budget) are missed**, not met.
- **P3.4 exactly-once is incomplete.** A crash between an effect succeeding and
  the ledger commit still duplicates the effect; closing it needs effects and
  ledger to commit together (a transactional `CheckpointStore`).
- **P6.2 route splitting and P6.3 streaming instantiate are not implemented.**
- **Initial Render still loses to React.** A rare ~20-30 ms outlier in
  content-render inflates its scored mean roughly 3x; several causes have been
  eliminated (see `example201_spike_probe_test.go`) and it is not yet named.

## v4.3.0 - 2026-07-06

Enterprise-hardening release: a file-by-file security, correctness, and
observability audit across the framework, plus two new CI gates that close
long-standing blind spots. Every fix was pinned by a test that was
**negative-verified** (confirmed to fail without the fix) and gated on both the
native and the (now CI-enforced) wasm suites. This is a **minor** release: the only
exported-surface change is add-only (two new `serverfn` setters), so no consumer is
forced to change code to compile — but several **security defaults changed
behavior**; see *Behavior changes* below for opt-outs.

### Behavior changes (security defaults — review before upgrading)

- **serverfn CSRF protection is now on by default.** `serverfn.Handle` requires
  `Content-Type: application/json` and rejects browser-flagged cross-site requests
  (`Sec-Fetch-Site: cross-site`). The generated client always sends
  `application/json`, so generated code is unaffected; only a hand-rolled caller that
  posts a non-JSON body breaks. **Opt out** with `serverfn.SetCSRFProtection(false)`
  if you must accept raw callers and have another CSRF defense.
- **serverfn no longer returns internal error text to clients.** A plain `error`
  from a server function now maps to a generic `500` ("internal server error")
  instead of echoing its message (which could leak a DSN/SQL/path). **Migration:** to
  send a deliberate message, return a `*serverfn.StatusError` (e.g. `serverfn.Conflict("...")`
  or `serverfn.NewStatusError(500, "...")`); the real error is delivered to the new
  `serverfn.SetErrorLogger` sink (default: stderr) so operators keep full diagnostics.
- **Plugins are held to their declared capabilities.** During `plugin.Host` `Setup`,
  an `Add*` call for a capability the plugin did not list in `Manifest.Requires` now
  fails registration (previously only the host-wide capability was checked).
  **Migration:** declare every capability your plugin uses in `Manifest.Requires`.
- **Plugins can only resolve services they declared.** `pluginruntime`'s per-plugin
  `Context.ResolveService` now returns unavailable for a service the plugin did not
  list in `RequiredServices`/`OptionalServices` (the kernel owner's resolution is
  unchanged). **Migration:** declare resolved services in the manifest (Optional if
  absence is tolerated).

### Added

- `serverfn.SetCSRFProtection(bool)` — toggle the CSRF content-type + cross-site
  defenses (default on).
- `serverfn.SetErrorLogger(func(name string, err error))` — route the real
  server-side error behind a 5xx to your logger (default: stderr; `nil` silences).

### Security & correctness

- **serverfn**: CSRF/Content-Type hardening + error-disclosure policy (above);
  read/decode errors and recovered panics are genericized to the client and logged
  server-side.
- **plugin / pluginruntime**: per-plugin capability enforcement and per-plugin
  service allowlist (least privilege at the plugin trust boundary); the cache-key
  decorator output is capped (8 KiB) to stop a runaway decorator from ballooning the
  fetch cache key (reject-not-truncate to avoid key collisions).
- **db/sqlite**: the passphrase encryptor's salt→derived-key cache is now bounded
  (FIFO, 128 entries) — decrypting data sealed by many replicas no longer retains
  derived key material without bound.
- **telemetryredaction**: a fail-visible dev guard (`GWC-TELEMETRY-NONJSON`) fires
  when a non-JSON value (a struct/pointer) reaches `Value` and cannot be walked, so a
  future caller passing a raw struct with secret fields is caught in dev/test instead
  of leaking silently. Scalar hot path stays reflection-free.
- **runtime2**: advisory patch-identity verification at the host/worker decode
  boundary — a patch whose carried identity does not match a recompute from its
  decoded parts is surfaced (`GWC-PATCH-IDENTITY-MISMATCH`) rather than silently
  applied. Advisory (not a hard reject) to avoid false-rejecting legitimately
  non-canonical identities.

### Router (all wasm-verified in a real DOM harness)

- **`BeforeLeave` is honored on browser back/forward.** popstate fires after the URL
  already changed, so a blocking `BeforeLeave` was silently bypassed; the router now
  evaluates the leave guard on popstate and restores the URL (via `pushState`) when it
  blocks. Synchronous guards only (documented follow-up for async on popstate).
- **History listeners are freed when a router is replaced** (hot-reload / remount /
  hash↔history switch) instead of being leaked and left attached to `window`.
- **popstate + hashchange no longer double-render** a single hash navigation
  (deduped by location signature; never blocks a genuine re-navigation).
- **A default-route fallback keeps its layout stack** (an unmatched path falling back
  to the default route was rendered bare, unlike direct navigation).

### Parallel-region worker bridge

- **Worker-bridge failures are surfaced**, not swallowed: a failed mount/update/click
  is reported on `ParallelRegionStatus.GetFallbackReason` instead of only logged (a
  failed region previously showed `WorkerAttached` and silently never updated).
- **Any function-valued prop is stripped** from the display-only worker props
  (previously only an exact-name `on*` allowlist), so an arbitrarily-named callback no
  longer fails the first JSON patch encode and silently degrades the region.

### CI & tests

- **New wasm-test merge gate** (`.github/workflows/wasm-tests.yml`): every package
  shipping a `*_wasm_test.go` now runs under `GOOS=js GOARCH=wasm` in CI — the many
  wasm suites were never executed by the native `go test` lanes. Turning the gate on
  immediately surfaced and fixed a hidden `virtualization` wasm-test failure (its mock
  document lacked `dispatchEvent`).
- **New hydration browser e2e** (`test/playwrightgo`, `#37`): server-renders a probe,
  hydrates it in headless Chromium, and asserts the framework-reported hydration
  metrics show zero fallbacks/discards (`FallbackCount==0`) — proving in a real
  browser that `ui.Hydrate` adopts server DOM rather than silently client-rendering,
  a gap that had no test since inception.

### Fixed

- **virtualization**: hidden wasm-test failure (missing `dispatchEvent` in the mock
  document) exposed by the new wasm CI gate.

## v4.2.0 - 2026-07-05

Performance release, round two: an overnight benchmark-refine-regress loop
against the Example 201 harness. GWC's production build now runs a
**0.85-0.95 same-run geomean band against React 19's production bundle, with
8-10 outright scenario wins per run** (from ~0.73-0.83 at v4.1.0). Every
change was kept only after a same-thermal-window A/B; the full per-iteration
log (including honestly reverted and gate-rejected attempts) is in
`docs/DEVNOTES_PERF_LOOP.md`.

### Performance

- **Interactive GC pacing as a browser default.** Go's GOGC=100 collects
  every time the heap doubles over an interactive app's tiny live set,
  landing stop-the-world pauses (measured up to ~9ms) inside interaction
  windows. GWC wasm apps now initialize with GOGC=300 plus a 512MB
  `debug.SetMemoryLimit` backstop; override or disable per origin via
  `localStorage["gwc:gogc"]` ("off" or a custom percent), with the applied
  policy reported as an info diagnostic. Measured: hooks scenario 13.0 to
  7.4ms mean, in-window collections 5 to 1; the GC-pause-victim scenarios
  (core-update, the refresh trio) now flip to outright wins.
- **Benchmark fairness: production vs production.** The scored browser
  benchmark previously compared a GWC development build (commit timers and
  the hook threading guard active) against React's production bundle. The
  report and boot probe now build with `-profile benchmark`
  (`-tags production`, `-s -w`, trimpath), as does the runtime2 worker; the
  phase/GC probes keep an explicit dev build for their instrumentation.
- **Production builds strip more dev-only cost.** Two raw
  `time.Now`/`time.Since` pairs on the hottest dispatch paths
  (`performUnitOfWork`, `renderFunctionComponent`) now route through the
  production-gated timing helpers, and the missing-key warning's ungated
  O(N) child scan per `reconcileChildren` call is production-gated.
  `go test -tags production` is green and part of the regression set.
- **Serialized mounts extended.** Sibling-run serialization mounts flat
  lists from ONE fragment parse (a 200-row list was one template parse per
  row); plain text children serialize inline, so mixed text+element chains
  (the deep-tree shape) mount as a single parse — deep-render dropped from
  ~8-9ms to ~2.8ms in the phase probe and deep scenarios now win outright.
  Observable via the new `Runtime.SerializedMountRoots()` counter.
- **Typed pass-through hooks and memo.** `ui.UseMemo` no longer allocates a
  `func() any` adapter closure nor runs per-call reflection
  (`runtime.GoUseMemoFor`); `ui.UseMemoOf`/`ui.UseEffectOf` take a static
  compute function plus one comparable dependency for zero steady-state
  allocations (2.6x faster hook walk in wasm).
- **Zero-allocation element construction fields.** `html.Props.DataAttr`
  sets one data-* attribute without a Data map allocation, and
  `html.Props.Text` sets text content without the throwaway child Element
  the `html.Text(...)` form builds. Cumulative native mirror allocations:
  core-update 340 to 180 per pass, core-refresh 344 to 142.
- **Keyed trailing appends skip the map path.** Growing a keyed list matches
  the prefix in order and mounts the tail as placements — no key boxing, no
  keyed-map build (list growth previously boxed every key into `any`).
- **Deletion teardown cost.** Removed a registry-wide atom-subscription scan
  that ran for every deleted fiber that had never subscribed (the
  subscribe-records-on-fiber invariant is now pinned by test), and merged
  two of the three recursive teardown walks per deleted subtree.
- **Cross-node attribute batching.** Commit-phase attribute writes buffer
  into one string-encoded payload applied by a single bridge call into a
  JS-side loop (WeakRef node registry; CSP-safe fallback to direct writes).
  primitive-attribute-update commit time -40%.

### Added

- `ui.Typed[P](fn)` — registers a props-taking component once and returns a
  constructor with a statically dispatched renderer (no `reflect.Value.Call`
  per render; wasm reflection was most of the hooks scenario's remaining
  dispatch cost). Mixing with `ui.CreateElement` preserves component
  identity.
- Benchmark/diagnostic instruments: the hooks/append native mirrors, a
  browser boot/TTI probe, GC counters and refresh/remove/attribute scenarios
  in the phase probe, a GC-pacing A/B probe, and `test/render` now compiles
  for wasm so mirrors run under node without a browser.

### Fixed

- mockdom fidelity: `textContent` concatenates descendant text,
  `ReplaceChildren` reparents like the browser, and the adapter can parse
  HTML fragments (`CreateHTMLSubtree`/`CreateHTMLFragment` via x/net/html),
  so the native suite exercises the serialized-mount and hydration paths.
- The benchmark subject's hook cell now mirrors React's semantics exactly
  (`useEffect(fn, [])` runs once; the previous no-deps form re-ran 800
  effects per render, biasing the comparison against GWC).


## v4.1.1 - 2026-07-04

### Fixed

- **Release gate follow-ups for v4.1.0.** The docs-site latest-release-notes
  mirror now surfaces the v4.1.x entry (the blocking
  `TestLatestEntryIsSurfacedInDocsSiteMirror` gate); the `fetch` test helper
  `waitFetchTestCondition` floors its ceiling at 5s so the realtime heartbeat
  test no longer flakes on loaded CI runners. No library code changes beyond
  v4.1.0 — see that entry for the performance release itself.

## v4.1.0 - 2026-07-04

Performance release: a day-long optimization campaign against the Example 201
browser benchmark (React 19 side-by-side). Same-run geomean vs React improved
from ~0.27x at the start of the campaign to **0.71x**, with two scenarios now
beating React outright (core-stress-update 1.10x, content-update 1.02x). All
lanes green throughout; no public API breaks.

### Performance

- **Synchronous discrete-event flush.** Component event handlers now commit
  their render pass inside the event's own task (`Runtime.FlushScheduledDiscreteWork`,
  called by the wasm event bridge after the handler returns) instead of paying a
  `setTimeout(0)` macrotask hop per interaction — the fixed ~1ms+ tax React's
  sync discrete flushing never paid. Goroutine-spawned state writes still
  coalesce exactly as before (the flush runs after all synchronous handler
  work); a guarded `workLoopDepth` prevents reentrant flushes from handlers
  fired synchronously by commit-phase DOM writes. 15/19 benchmark scenarios
  improved.
- **Hook-slot machinery (hooks-render 2x).** Three fixes measured by the new
  native hooks mirror benchmark (1.25ms → 0.25ms per 2400-hook pass, allocs −47%):
  the dev-mode hook threading guard re-samples its `runtime.Stack` traceback
  every 4096th call instead of every 64th (the traceback was 65% of total CPU
  in hook-dense renders); `GoUseState` caches its getter/setter closure pair
  per slot across renders (closure churn was 47% of allocations), retargeting
  through `hooks.owner` so cached setters never hold stale fibers; and
  `ui.CreateElement` no longer rebuilds and re-stores a component's renderer on
  every call when the implementation is the identical function value
  (`ComponentType.ImplementationMatches`).
- **Typed no-map element fast lane.** html-built compact host elements carry a
  deterministic `[]HostAttr` slice and `Key` field instead of a `map[string]any`
  Props map; the reconciler diffs fast-lane pairs positionally, commit touches
  only changed attributes, and SSR/hydration serialize byte-identically. Core
  native update: 155µs → 55µs. Post-creation mutators (`WithKey`, `Show`,
  `WithChildren`) go through the Ensure → mutate → Refresh seam.
- **LIS-hybrid child-order repair.** Keyed reorders keep the longest in-order
  run and move only displaced nodes (one displaced row = one `insertBefore`,
  previously a wholesale `replaceChildren`); heavy permutations still collapse
  to one replace.
- **Serialized subtree mounts.** Eligible all-fast-lane placement subtrees
  mount from one `template.innerHTML` parse + binding walk instead of one
  bridge call per node (a 200-row list mounts in one call instead of ~400);
  per-node prepared+batched creation remains the automatic fallback.
- **Bailout and render-loop pruning.** Subtree bailouts stop descending at
  clean nodes, skip clock reads, and the render pass avoids redundant key
  probes and double event scans.
- **Reconciler/runtime pass (earlier in campaign).** Geomean −33% across the
  native benchmark suite; hooks −97%, SSR string/stream −85%.

### Fixed

- **Diagnostics ring quadratic under report storms.** Once the bounded ring
  filled, every further report rebuilt the entire slice and dedup index
  (measured: 478ms for a 2000-warning hydration mismatch storm). The ring now
  drops to half capacity on overflow, making reports O(1) amortized; the same
  storm costs 6.4ms.
- **mockdom `textContent` fidelity.** The mock adapter now concatenates
  descendant text like a browser, so hydration over parsed markup no longer
  fabricates per-node text mismatches in native tests.

### Added

- **Native benchmark + probe suite.** Native mirrors of the browser benchmark's
  core and hooks scenarios (`test/render/*_benchmark_test.go`), a hydration
  walk benchmark with adoption pinning, a browser boot/TTI probe
  (`TestExample201BootProbe`: wasm fetch / instantiate / subject-ready split),
  and GC visibility in the phase probe (`numGC`/`gcPauseNs` per scenario).
- **mockdom `CreateHTMLSubtree`.** The mock adapter parses HTML subtrees via
  `x/net/html`, so the native suite exercises the serialized-mount and
  hydration-over-markup paths the browser adapter uses.

### Measured (no code change)

- Boot: subject wasm is 2.37MB gzipped (vs ~45KB for the React equivalent);
  localhost TTI 437ms cold / ~114ms warm vs 34ms. The wire size is the
  dominant real-world gap.
- runtime2 workers: a flat ~2x per-update round-trip overhead vs runtime1 at
  2, 4, and 8 workers (protocol-bound, not contention); worst on full-dataset
  re-serialization scenarios (core-append 9–11x).
- GC tails: max/median 1.74 vs React's 1.20; 1–5 collections per scenario
  window with single pauses up to ~9ms — the allocation cuts above are the
  mitigation path.

## v4.0.1 - 2026-06-29

### Fixed

- **`/v4` module path (semantic import versioning).** v4.0.0 was not `go get`-able:
  the module path lacked the major-version suffix Go requires for v2+, so the proxy
  rejected every v2+ tag. The module is now `github.com/monstercameron/GoWebComponents/v4`
  and all internal import paths were updated accordingly. Consumers import
  `github.com/monstercameron/GoWebComponents/v4/...`.
- **Release pipeline.** Pinned `playwright-go` in the browser-test scaffold go.mod
  (offline `go mod tidy` resolution); made the API-baseline comparison
  line-ending-agnostic (CRLF vs LF on Windows checkouts); restored `bin/README.md`
  so the SBOM step's output directory exists; the go-get release smoke now targets
  the `/v4` path and runs `go mod tidy` to populate the transitive `go.sum` closure.

## v4.0.0 - 2026-06-28

### Added

- **`agentui.Registry.Catalog` — allow-list introspection for MCP tools (V4, F4).** Exposes
  the component allow-list as sorted, JSON-serializable `ComponentInfo` (name + permitted
  props) so an agent — or an MCP tool serving the registry over the existing `agentbridge` —
  learns up front exactly what it may emit, turning the allow-list into guidance rather than
  an after-the-fact rejection.
- **`hotreload.SchemaChanged` — state-schema-change detection (V4, C1).** Fingerprints a
  state snapshot's shape (keys + value types) so a state-preserving hot reload can show a
  visible "state reset" when the shape changed instead of silently restoring a persisted
  snapshot into a mismatched type — the ghost-bug fix the plan calls out.
- **`ui/erroroverlay.ErrorOverlay` — in-page Elm-grade error overlay (V4, FB4).** A dogfooded
  component that renders a development error as a dismissible, accessible modal (title,
  message, actionable "Try: …" hint, optional stack), so failures are legible in the page,
  not just the console. `FromError` builds props from a Go error.
- **`gwc buildreport` — "what rebuilt & why" (V4, C2 platform-honest).** Derives a per-build
  report from `go build -debug-actiongraph`: which packages were rebuilt (the `NeedBuild`
  cache-miss signal) vs served from cache, ranked by compile time, with the warm/cold wall
  time. The achievable platform-honest C2 surface; the true 10-rung (incremental wasm
  linking) is an upstream Go ask the plan tracks as non-blocking.
- **VS Code extension surfacing `gwc lint` diagnostics (V4, tooling).** A minimal real
  extension (`tools/vscode-gwc`) that runs `gwc lint --json` on save and shows issues inline
  via a `DiagnosticCollection`. Its mapping core (severity, 1-based→0-based positions, source
  tagging) is host-independent and unit-tested under plain Node; it consumes the stable CLI
  contract owned in Go, so the editor integration stays a thin language-agnostic consumer.
- **Edge/WASI portability of the server stack (V4, FC4).** The server-side V4 packages —
  `serverfn`, `wholestack`, `localfirst`, `agentui`, `query`, `validate`, `timetravel` — are
  written platform-clean and verified to compile to `GOOS=wasip1` (WASI), so the whole-stack
  handler and the sync/server-function/generative-UI engines run in edge WASI runtimes, not
  only on a conventional server. "Ride the wasm platform leap" — the same Go, server-side at
  the edge.
- **`wholestack` — one-binary, whole-stack, edge-portable deployment (V4, FC5).** `Handler`
  composes a single `http.Handler` that serves the embedded wasm bundle (index + `.wasm` +
  `wasm_exec.js`) AND the app's `//gwc:server` functions, with SPA fallback so client-routed
  paths deep-link to the shell. One `go build` is the entire app — no Node, no separate
  static host, no reverse proxy — and it runs anywhere `net/http` runs, including edge
  runtimes. `ListenAndServe` is the one-line entry point. Verified end-to-end: the app shell,
  a static asset, and a server function all served from one handler; SPA fallback; opt-out.
- **In-app devtools/collaboration panels — dogfooded as GWC components (V4).** The plan's
  "panels" and "galleries" are built as testable framework components (the framework
  rendering its own tooling), each verified headlessly through the real reconciler + mock
  DOM: `timetravel/devpanel.Panel` (time-travel scrubber timeline + step controls, C4/FB6),
  `workbench/gallery.Gallery` (component gallery over the same stories `RunStories` tests,
  FB5), `query/devtools.CachePanel` (cache observability table over the new `Cache.Inspect`,
  D2), and `localfirst/facepile.Facepile` (presence "who's here" surface, FC6).
- **`query.MutateAsync` — fire-and-forget optimistic actions (V4, FB2).** Applies the
  optimistic value immediately and reconciles in the background (commit on success, rollback
  on error, `onSettled` callback) — the async mutation that makes Next/Remix-style server
  Actions a one-liner. Verified end-to-end as an optimistic action over a real `//gwc:server`
  function (optimistic value on screen immediately; server's authoritative result committed
  on settle).
- **`timetravel` — snapshot step-back replay engine (V4, C4/FB6).** The pure engine behind
  time-travel devtools (C4) and undo/redo (FB6): a bounded, navigable `History[T]` of
  immutable snapshots with `Record`/`Undo`/`Redo`/`ScrubTo`, standard redo-branch
  truncation, and ring eviction. Owns no clock/DOM/runtime; the devtools panel is a thin
  view over `Labels()`/`ScrubTo`.
- **`ui.UseInspect` — Svelte-style `$inspect` (V4, FB7).** Logs a labeled value's initial
  state and every subsequent change (`old -> new`, by structural equality) through a
  swappable sink (`SetInspectSink` → devtools, a test buffer, or silenced in production).
- **`shorthand` named slots/snippets (V4, FB7).** Explicit, typed named slots (Vue named
  slots / React render-children-by-name): `Slot`/`NewSlots` (last-wins) + `Has`/`Render`/
  `Or` (overridable default content).
- **Public-API baseline tests for V4 packages (V4).** `internal/apidump` extracts a
  package's exported surface (types with their shape, funcs, methods, consts, vars) as
  sorted signature lines and checks it against a committed golden, so any change to the
  public API of `serverfn`/`query`/`localfirst`/`agentui`/`validate`/`anim` fails a test
  with a diff and can only land by regenerating the golden (`UPDATE_API_BASELINE=1`) with
  intent. This is the concrete "graduated to Stable, API-baseline pinned" mechanism.
- **`agentui` — agent-native generative-UI runtime (V4, FC3).** A typed renderable schema
  an agent emits (server-side behind a `//gwc:server` function), validated against a
  component allow-list, then rendered to real UI. The safety property is structural: a
  `Node` carries no code, no event handlers, no raw HTML — only an allow-listed component
  `Type`, string `Props` the component permits, plain (escaped) `Text`, and `Children`. An
  agent composes allow-listed components but can never inject behavior or markup —
  the "allow-listed components, not raw code" guarantee, expressed natively by GWC's typed
  component model. `Registry.Validate` recursively rejects any non-allow-listed type or
  disallowed prop; `Render`/`RenderJSON` gate on validation. `DefaultRegistry` ships safe
  presentational components. Verified end-to-end: agent JSON → validate → render → mounted
  DOM shows the content. Hardened against untrusted-input DoS with safe-by-default size
  bounds (`Limits`/`DefaultLimits`: max depth 32, max 10k nodes; `ValidateWithLimits` to
  tune).
- **`localfirst.PresenceSet` — collaboration presence/awareness (V4, FC6).** Real-time
  collaboration is the converging document store (FC1) plus ephemeral presence; presence is
  the only new piece, so FC6 comes nearly for free. `PresenceSet` tracks live peers (cursor/
  name/status as an opaque payload) with heartbeat expiry and no wall clock — the app
  advances time via `Tick`, so a crashed tab leaves the session deterministically. Distinct
  from synced `Record`s by design: "who is here now" has no history.
- **`localfirst` — built-in local-first sync engine (V4, FC1).** The convergence engine
  behind local-first sync (Zero/Electric/TanStack DB), pure Go on both sides: a
  last-write-wins register per key (LWW-Register CRDT) with a logical `Clock` (higher
  counter wins, replica id breaks ties → every replica resolves a conflict identically). A
  `Replica` gives optimistic local writes, a durable offline pending log, and a `Merge`
  that converges toward authoritative records while acknowledging confirmed writes and
  preserving Lamport monotonicity; an `Authority` is the server-side store. `Mutation`/
  `Record` are JSON-serializable so they ride the `//gwc:server` transport unchanged.
  Marquee guarantee verified: two replicas edit the same key offline, reconnect, and every
  replica + the authority converge to the one value — proven both in-process and
  **end-to-end over the real `serverfn` HTTP transport** (FC1 × FB1). The offline queue is
  durable: `Replica.Export` / `RestoreReplica` round-trip the full replica (records +
  pending log + counter) through JSON, so unsynced writes survive a page reload and still
  converge on reconnect (E3 — offline replay is the same engine).
- **`gwc add` — headless component registry (V4).** The shadcn "own the code" model:
  `gwc add <name>` copies an a11y-correct, self-contained component into your repo (with
  your package name and a provenance header) — you own and restyle it, no runtime
  dependency. Catalog: `disclosure` (WAI-ARIA disclosure) and `tabs` (WAI-ARIA
  tablist/tabpanel with a roving tab stop), each verified native + wasm. Templates live as
  a real compiled package, so the catalog can never ship a component that doesn't build
  (FA3).
- **`gwc vuln` — known-vulnerability reachability scan (V4).** Runs `govulncheck -json`
  and classifies each finding as REACHABLE (a call trace names a vulnerable function) or
  imported-only. Reachable vulnerabilities fail the build; imported-only pass unless
  `-strict`. Complements `gwc supplychain` on the security dimension (D5). Verified against
  the live Go vuln database.
- **`//gwc:server` server functions — the FB1 keystone (V4).** A server function is a
  plain, type-safe `func(context.Context, Req) (Resp, error)` marked `//gwc:server` that
  runs only on the server. The new `serverfn` runtime exposes `Handle` (register it as a
  JSON HTTP endpoint) and `Call` (invoke it, returning the typed response or a typed
  `*ServerError`); both use `net/http`+`encoding/json` with no build tags, so the server
  uses real sockets and the browser uses Fetch-backed `net/http` — the same code path,
  fully testable with `httptest`. `gwc server gen` scans for `//gwc:server` functions,
  enforces the contract, and generates `serverfn_gen_client.go` (browser stubs calling
  `serverfn.Call`) + `serverfn_gen_server.go` (`RegisterServerFunctions(mux)` wiring each
  via `serverfn.Handle`). One Go signature, called from the browser with full compile-time
  type safety and no hand-written fetch/JSON glue — verified end-to-end: native server +
  wasm client both compile, and a function round-trips over HTTP through the generated
  registration. (Contract: shared Req/Resp types in a build-tag-free file, the server
  function in a `//go:build !js || !wasm` file; the codegen rejects an unconstrained file
  to prevent client/server name collisions. `gwc server check` is the CI staleness gate.)
- **`router.DecodeQuery` / `EncodeQuery` — typed, validated search params (V4).** Decode
  URL query strings into a typed struct via `query:"name"` tags
  (string/bool/int/uint/float/[]string), then validate it with the `validate` package
  using `validate:"..."` tags on the same struct — one typed read instead of scattered
  `q.Get` + `strconv` + bounds checks. Malformed values error with the offending param;
  validation failures return the `validate.Result`. `EncodeQuery` is the inverse for
  building links (zero fields omitted). Meshes typed routes (B7), compile-safe data (B3),
  and shared validation (B8).
- **`validate` now supports `omitempty` (V4).** An optional field whose value is the zero
  value skips its remaining rules, and is validated normally when present.
- **`anim` keyed-list FLIP + enter/exit transitions (V4).** The FA5 orchestration layer
  over the existing FLIP/spring math: `DiffKeyedRects` classifies a keyed layout change
  into entering/exiting/moving items and computes each survivor's FLIP invert transform
  (`MovedKeys()` filters to items that actually moved); `Transition` is a pure, clock-free
  enter/exit state machine (`Entering→Entered`, `BeginExit→Exiting→Exited`) with
  `Progress`/`IsAnimating`/`IsRemovable`; `StaggerDelay` gives per-index cascade timing.
  Owns no DOM and no clock — fully deterministic and unit-testable. Honors
  `prefers-reduced-motion` via an explicit `MotionPreference` (`Animates()`,
  `EffectiveDuration()`, `NewTransitionPref()`): under reduced motion, transitions snap
  and FLIP moves are skipped (FA5 × D4).
- **`gwc i18n gen` — typed, compile-checked message accessors (V4).** Reads a
  base-locale bundle (`{"namespace":{"key":"text {param}"}}`) and generates
  `i18n_keys_gen.go`: one typed accessor per message wrapping `i18n.Runtime.T`, taking a
  string argument for each `{param}`. A typo in a namespace, key, or interpolation
  parameter — or a forgotten/extra param — becomes a compile error rather than a silent
  runtime miss. `gwc i18n check` is the CI staleness gate (regen + diff). The codegen
  pair to `gwc routes gen`, closing the last stringly-typed surface (F6/B3).
- **`query` package — stable query/data layer (V4).** GoWebComponents' Go-native
  answer to TanStack Query / SWR: a keyed, concurrency-safe cache with request
  de-duplication (N concurrent `Fetch`es for one key coalesce into a single fetcher
  call), stale-while-revalidate (`SWR` returns stale data synchronously and refreshes
  in the background), one-call optimistic mutations with automatic rollback on error
  (`Mutate`), and invalidate-by-key / by-prefix. Kept deliberately **explicit** — the
  key and fetcher are passed at every call site, nothing is auto-tracked — and **pure
  Go**, so the whole layer compiles to wasm and native and is fully unit-testable
  (`WithClock` injects deterministic time). Implements FA2 / graduates B6 to Stable.
  Consumed in components via the new `ui.UseQuery` / `ui.UseMutation` hooks
  (`query.Snapshot[T]` for the render path), covered by two wasm e2e tests through the
  real render path — cached data painting to the DOM, and the full optimistic-mutation
  loop (commit updates the DOM; failure rolls it back).
- **`ui.Form.ValidateStruct()` — turnkey shared validation (V4).** Validates a form's
  value against its `validate:"..."` struct tags with no hand-written client
  validator, so the same struct validates client (wasm) and server (native) from one
  source and the two cannot drift. Capstone of B8.
- **`state.Signal[T]` / `state.NewSignal` / `state.NewComputed` — fine-grained
  reactivity as a first-class primitive (V4).** A terse, ergonomic handle over the
  shared atom registry that updates exactly the DOM nodes bound to it
  (`signal.Text(...)`) without re-rendering the owning component. Needs no
  caller-managed id (one is minted; `NewKeyedSignal` shares an explicit id), and is
  created outside the hook lifecycle so it can live in a package var, event handler,
  or goroutine. `NewComputed` derives from **explicitly declared** sources — GWC
  keeps reactivity predictable and auditable rather than discovering dependencies
  through a hidden runtime graph. Signals compose with `ui.ReactiveRegion` and
  `state.UseSelector` (same source contract as atoms/derived). Covered by native
  unit tests + wasm fine-grained-update tests.
- **`gwc routes gen` — typed, compile-checked route links (V4).** Scans a package
  for `router.MustDefineRoute("/users/:id")` contracts and generates `routes_gen.go`
  with typed `Link*` constructors (`LinkUser(id string) string`), so a missing or
  misnamed path parameter becomes a **compile error** instead of a runtime one —
  closing the last stringly-typed gap that the runtime `RouteContract` can't.
  `gwc routes check` is the CI staleness gate. Output is `go/format`-clean.
- **`validate` package — dependency-free shared struct-tag validation (V4).** One
  Go struct with `validate:"required,email,min=3,oneof=..."` tags validates
  **identically on the server and the client** — `validate.Struct(v)` runs in an
  HTTP handler and, because it uses only reflect/regexp/strconv (no syscall/net/file),
  in the browser via wasm in `ui.UseForm`, so client/server validation can never
  drift. `Result.Fields()` is `map[string]string` (the `ui.FieldErrors` shape) and
  `Result` implements `error` for handlers. Rules: required/min/max/len/email/url/
  oneof/eq/ne/gt/gte/lt/lte/alpha/alphanum/numeric, with nested-struct dotted paths.
- **`gwc llms` — AI-native docs generation (V4).** Generates `llms.txt` (the
  llms.txt-standard index: project header + a linked, summarized table of every
  reference-manual chapter) and `llms-full.txt` (all chapters concatenated for
  full-context ingestion) from `docs/REFERENCE_MANUAL`. Deterministic output with a
  `-check` CI staleness gate (regenerate + diff). The first step toward "the
  framework your AI assistant gets right the first time."
- **`gwc supplychain` — supply-chain / zero-npm audit command (V4).** Proves a
  project has **no npm dependency surface** (scans for `package.json`/lockfiles/
  `node_modules`, excluding research/examples/vendor/testdata), counts direct vs.
  transitive Go modules, confirms `go.sum` checksums are present, and enforces an
  optional `-budget N` on direct external dependencies. Locally-`replace`d modules
  are excluded from the remote count. Emits the `gwc.agentic.v1` JSON envelope or a
  human summary; non-zero exit on failure. The framework itself audits clean
  (0 npm, checksum-verified). Adds **zero** dependencies — parses go.mod/go.sum by hand.
- **`html.BindTo` / `html.BindFunc` (+ shorthand re-exports) — two-way binding for
  any handle.** Complements `html.Bind` (which binds a `ui.State[string]`) with a
  structural `html.Binding` interface (`Get() string` / `Set(string)`), so the new
  `state.Signal[string]`, atom handles, or an explicit getter/setter all bind a
  controlled input in one option — no `value=`+`oninput=parse` pair.
- **`ui.Run(selector, component, props...)` — a one-line browser entrypoint.** It
  wraps `ui.CreateElement` + `ui.Render` + the keep-alive block so a typical
  `main()` is a single call (`ui.Run("#app", renderApp)`) instead of the
  `Render(CreateElement(fn, nil), "#app")` + `utils.WaitForever()` pair. The
  explicit primitives are unchanged and remain the right choice for SSR, native,
  tests, or when `main` must do work after mounting. On the native/SSR slice
  `ui.Run` panics with the unified unsupported-on-server contract, mirroring
  `ui.Render`. README and getting-started examples now lead with `ui.Run`.
- **`interop.KeepAlive()` — the shared keep-alive primitive.** `utils.WaitForever`
  and `ui.Run` both delegate to it. It lives in the leaf `interop` package so
  `ui` can reuse it without the `ui → utils → hotreload → state → ui` import
  cycle. `utils.WaitForever` is unchanged for callers (same name, same behavior).

## v3.5.4 - 2026-06-25

### Fixed

- **Effects were dropped when a component used render-phase updates (regression in
  v3.5.3).** The render-phase convergence loop cleared the fiber's per-iteration
  effect list before each re-run; combined with the deps-equality check (a
  stable-deps effect does not re-register when its deps are unchanged), the effect
  ended up registered in no iteration and never ran. The loop no longer clears the
  effect list — effects accumulate across convergence iterations and are then
  de-duplicated by cleanup-index (keeping the last, i.e. the final render's effect
  per slot). Each effect now runs exactly once with the converged state, for both
  stable and changing deps, and multiple effects run once each in declaration
  order. Covered by `TestRenderPhaseEffectRunsOnceStableDeps` /
  `...ChangingDeps` / `TestRenderPhaseMultipleEffectsEachOnce`.

## v3.5.3 - 2026-06-25

### Fixed

- **Render-phase state updates left the DOM inconsistent with the render.** When a
  component updated its own state during render (`if x { setState(...) }` — the
  derived-state-during-render pattern), the setter stored the new value and
  scheduled a commit, but the component was never re-run, so the committed DOM
  showed a value the component had not actually rendered (e.g. it rendered `0` but
  the DOM showed `1`, and it never converged). `renderFunctionComponent` now
  detects a render-phase update (the setter's owner is the fiber actively
  rendering, tracked by a new `activeRenderFiber` flag — distinct from the ambient
  current fiber so it never trips on the SSR path or hook unit tests) and re-runs
  the component to convergence, bounded at 25 iterations with a diagnostic to
  guard against an unconditional-setState infinite loop. Matches React's
  "keeps restarting until there are no more new updates." Covered by
  `TestRenderPhaseUpdateConverges` / `TestRenderPhaseDerivedStateOneStep` /
  `TestRenderPhaseUnconditionalDoesNotHang`.

## v3.5.2 - 2026-06-25

### Fixed

- **`defaultValue` / `defaultChecked` rendered as inert attributes in SSR.** An
  uncontrolled form field set via `defaultValue` (or `defaultChecked`) serialized
  the prop literally — `<input defaultValue="x">` — which the browser ignores, so
  the field rendered empty/unchecked server-side. The SSR serializer now maps
  `defaultValue` → the element's controlled value and `defaultChecked` → `checked`
  (only when the controlled prop is not already set, so an explicit `value` wins),
  in both the buffered and streaming renderers. Because the mapping happens before
  the controlled-value logic, it flows through every form element: `<input>` gets
  a `value` attribute, `<textarea>` gets text content, and `<select>` marks the
  matching `<option selected>`. Matches React's
  `ReactDOMServerIntegrationInput`/`Textarea`/`Select` defaultValue behavior and
  completes the controlled-input SSR work (v3.4.9 textarea, v3.4.10 select).
  Covered by `TestSSRDefaultValueMapsToControlledValue`.

## v3.5.1 - 2026-06-24

### Added

- **Streaming SSR (`RenderToStream`) now threads context to hooks.** v3.5.0 ran
  hooks in the streaming path but passed no context, so a streamed component's
  `GoUseContextValue` fell back to the descriptor default. The streaming state now
  carries the inherited context map (derived at each `ContextProvider` boundary and
  restored after its children, and captured per pending async boundary so deferred
  Suspense content resolves the same context), so streamed `useContext` resolves
  to the nearest provider — including nested-provider override. Completes the SSR
  hooks work begun in v3.5.0. Covered by `TestStreamRendersHooksAndContext`.

## v3.5.0 - 2026-06-24

### Added

- **Hooks now run during server rendering (`ui.RenderToString`).** Previously the
  string serializer was hook-less: a component calling any hook errored with
  "called outside component context", so only hook-free trees could be
  server-rendered. `RenderToString` now installs a transient hook fiber per
  component, so on the server:
  - `GoUseState` returns its initial value (the setter is a no-op);
  - `GoUseRef` and `GoUseMemo` compute normally;
  - `GoUseContextValue` resolves to the nearest provider's value — context flows
    down through host elements and nested providers override correctly;
  - `GoUseEffect` is queued but never committed, so effects do not run on the
    server (matching React).

  This makes GWC able to server-render real (hook-using) components, matching
  React's `ReactDOMServerIntegrationHooks`. Implemented by threading the inherited
  context map through the SSR renderer and deriving it at each `ContextProvider`
  boundary. Covered by `ui/ssr_hooks_test.go`.

  Notes: SSR-with-hooks uses the package-global current-fiber like the client
  reconciler, so a single process renders hook components one tree at a time
  (concurrent goroutine SSR of hook components needs external serialization — a
  pre-existing hook-architecture constraint). The streaming SSR path runs hooks
  but does not yet thread context (a documented follow-up); the buffered
  `RenderToString` path is fully supported.

## v3.4.10 - 2026-06-24

### Fixed

- **Controlled `<select value="…">` did not mark the selected option in SSR.**
  The serializer put a (browser-ignored) `value` attribute on the `<select>`, so
  a server-rendered controlled select showed no initial selection and mismatched
  on hydration. The SSR serializer now drops the `value` attribute from the
  `<select>` and renders the matching `<option>` with `selected` — matching by the
  option's `value`, or by its text content when it has no value (React's
  fallback), and recursing into `<optgroup>`. An option that already declares
  `selected` is left unchanged. Fixed in both the buffered and streaming SSR
  renderers. Completes the controlled-input SSR work begun in v3.4.9 (textarea);
  found while porting React's `ReactDOMServerIntegrationSelect`. Covered by
  `TestSSRSelect*`.

## v3.4.9 - 2026-06-24

### Fixed

- **Controlled `<textarea value="…">` rendered empty in SSR.** HTML ignores a
  `value` attribute on `<textarea>` — the displayed value is the element's text
  content — but the SSR serializer emitted `<textarea value="x"></textarea>`, so a
  controlled textarea rendered blank server-side and mismatched on hydration (the
  client path sets `.value` as a DOM property, which masked the bug outside SSR).
  The serializer now renders a textarea's string `value` as escaped text content
  (`<textarea>x</textarea>`), matching React's `ReactDOMServerIntegrationTextarea`
  behavior; other attributes are preserved and an explicit child is used when no
  `value` is set. Fixed in both the buffered and streaming SSR renderers. Found
  while porting React's controlled-input SSR tests. Covered by
  `TestSSRTextareaValueRendersAsContent`. (Note: `<select value>` still renders
  the value as an attribute rather than marking the matching `<option selected>`;
  tracked as a follow-up.)

## v3.4.8 - 2026-06-24

### Fixed

- **Duplicate keys in a list leaked host nodes and corrupted later renders.**
  The reconciler's old-fiber lookup is a one-fiber-per-key map, so two siblings
  sharing a `key` collided: the second overwrote the first, and the overwritten
  fiber was never tagged for deletion. The list's host child count then grew past
  the element count (e.g. `[a,a,b]` → `[a,b]` left **3** DOM children instead of
  2), and the orphaned nodes bled into subsequent — even structurally different —
  renders. Duplicate-keyed old fibers are now routed to the positionally-matched
  fallback list, so every old fiber is tracked in exactly one structure and is
  cleaned up; a new duplicate-keyed element also reuses a fallback match. Unique
  keys are unaffected. Found via the native reconciler harness; covered by
  `TestReconcileDuplicateKeysDoNotLeak` / `TestReconcileUniqueKeysUnaffected`.

## v3.4.7 - 2026-06-24

### Security

- **`javascript:` URL injection through the normal element API.** The render
  path emitted URL-bearing attributes (`href`, `src`, `action`, `formaction`,
  `poster`, object `data`, `xlink:href`, …) verbatim, so a user-controlled URL
  passed through the ordinary `html.A`/`html.Img`/`html.Tag` API — e.g.
  `html.A(html.Props{Href: userURL})` — could ship a clickable
  `<a href="javascript:alert(1)">`, an XSS vector. Only markdown and `RawHTML`
  sanitized URLs before; the element API did not. Script-executing schemes
  (`javascript:`, `vbscript:`) are now neutralized to the inert `about:blank`
  sentinel at both the SSR serializer and the browser DOM adapter, tolerating
  the usual obfuscations (leading/embedded whitespace and control characters,
  intermediate CR/LF, mixed casing). Safe schemes (`http(s)`, `data:`, `mailto`,
  `tel`), relative paths, and fragments are untouched, and a URL-looking value in
  a non-URL attribute is left alone. Vectors ported from React's
  `ReactDOMServerIntegrationUntrustedURL` suite; covered by
  `TestURLSanitization*` and the `FuzzSanitizeURLAttributeValue` property fuzz
  (7M+ executions clean).

## v3.4.6 - 2026-06-24

### Security

- **`hardenCSS` NUL-byte breakout bypass.** The CSS `<style>`/comment-close
  hardening skipped NUL bytes inline, *after* its lookahead guards, so an input
  like `*\x00/` or `<\x00/style>` slipped past the guards and then had the NUL
  removed — silently reconstituting `*/` or `</style>` in the emitted CSS (a
  style-element / CSS-comment breakout reachable via `css.Inject`, `css.Global`,
  `css.Raw` values, etc.). NULs are now stripped in a separate first pass so the
  guards run on the final byte stream. Found by fuzzing; regression seed +
  `FuzzInjectHardening`/`FuzzCSSEscape` fuzz tests added.

## v3.4.5 - 2026-06-24

### Fixed

- The `diagnostics` and `internal/diagnostics` test binaries failed to compile
  under `GOOS=js GOARCH=wasm` because build-neutral test files referenced the
  native-only (`//go:build !js`) `WriteHTTPError`. The HTTP-dependent test files
  are now tagged `//go:build !js` to match (and the HTTP benchmark split into a
  native-only file). Same class as the v3.4.4 `state` fix.

## v3.4.4 - 2026-06-24

### Fixed

- The `state` package test binary failed to compile under `GOOS=js GOARCH=wasm`
  because `ExampleUseAtom` was declared in both a wasm-only and a build-neutral
  example file ("redeclared"). Removed the redundant wasm-only duplicate; the
  documented build-neutral example remains.

### Tests

- `MapKeyedComponent` renders through SSR (`RenderToString`, hydration-ready);
  `css.Global`/`Root` rules participate in SSR seed-suppression (not re-injected
  on hydration).

## v3.4.3 - 2026-06-24

### Fixed

- `ui.UseDocumentEvent`/`UseWindowEvent`/`UseGlobalKey` (and `UseNetworkStatus`)
  now degrade to a no-op when the global target lacks `addEventListener` instead
  of throwing, matching the no-DOM guards used elsewhere. No change in real
  browsers/Web Workers (which always have it); defense-in-depth for exotic hosts.

### Tests

- `css.Inject` `<style>`/comment breakout hardening; multi-hook composition
  (UseMount + UseMediaQuery + UseTheme + UseLayoutEffect) coexistence.

## v3.4.2 - 2026-06-24

### Fixed

- `state.GlobalAtom.Set` and `ui.SetTheme` no longer re-render subscribers on a
  no-op write — setting a value equal to the current one is now skipped (matching
  `UseState`'s dedup). Equality is a recover-guarded `==`, so non-comparable
  slice/map values still always write. `SetTheme` still applies the
  `data-theme` attribute either way.

### Tests

- Dedup render-count test (no-op `Set` does not re-render, changed `Set` does);
  `GlobalAtom` composite value types (slice/struct/map); `MapKeyedComponent`
  nil-rendering rows; `OnNavigate` initial-mount contract.

## v3.4.1 - 2026-06-24

### Fixed

- `router.FragmentHref` dropped the current query string — an in-page anchor on a
  URL like `/list?page=2` produced `/list#main`, so clicking it navigated away
  from the query state. It now preserves the query (`/list?page=2#main`).

### Tests

- Edge coverage for `css.Global`/`Layer` + variant compositions (`@layer`+`Hover`,
  `Global`+`@media`, `LayerGlobal`+`DataTheme`), partial/empty `Theme.RootRules`,
  `interop.Await` rejection with non-Error payloads, and the FragmentHref query
  regression.

## v3.4.0 - 2026-06-24

### Added

- **Reactive theming — `ui.UseTheme` / `ui.SetTheme` / `ui.CurrentTheme`** — a
  hook that subscribes a component to the active theme name and a setter that
  switches it everywhere (re-rendering all subscribers and applying
  `<html data-theme="…">` so `[data-theme]` rules and `:root` token overrides
  take effect). `SetTheme`/`CurrentTheme` switch/read from outside a render
  (global hotkeys, OS theme listeners). The reactive capstone over the typed-CSS
  token system — style with tokens (below), switch with UseTheme.
- **`css.Theme.RootRules` / `css.EmitThemeTokens`** — emit a typed `Theme`'s
  scales as a `:root` custom-property palette (`--color-*`, `--space-*`,
  `--text-*`, `--radius-*`), bridging the typed theme to a live CSS-variable
  palette. Compose with `Root`, `LayerGlobal`, or `DataTheme` and reference via
  `Var`; a runtime `setProperty` then reskins without regenerating classes
  (closes the CSS2 token story end-to-end).

### Fixed

- `interop` cookie read/write (`readRawCookies`/`writeRawCookie`) panicked
  (`Value.Get on undefined`) in no-DOM wasm contexts (Web Worker, no-DOM SSR
  side, the node test runner); they now return a structured `CodeUnavailable`
  error, matching the native stubs.
- `router.FragmentHref` no longer emits a malformed double-hash
  (`#/path#fragment`) under a hash router; it returns `<path>#<fragment>` and is
  documented as a history-router helper (hash routers keep the route in the URL
  fragment, so in-page fragment anchors there need programmatic scrolling).

### Tests

- CSS design-token + `var()` round-trip and `Theme.RootRules` coverage (CSS2).
- Hardened the G1 `MapKeyedComponent` coverage with a reorder +
  variable-length-removal case proving per-row hook state follows the key when
  the list is reordered and shrinks.
- Authoritative `CSS.escape` cross-checks for `ui.CSSEscape`.

## v3.3.0 - 2026-06-24

### Added

- **Hooks/handlers in loops — `MapKeyedComponent`** (G1): renders each item as its
  own keyed component (own fiber), so the render func may use `UseState`/`UseEvent`
  and `On*` handlers directly inside the loop — no hand-extracted row component,
  no "hooks in a variable-length loop" footgun. Per-row state is isolated and
  persists across renders by key.
- **SVG chart primitives** (G8) — `shorthand.Ellipse`, `TSpan`, `LinearGradient`,
  `RadialGradient`, `GradientStop`, `ClipPath`, `Mask`, `SvgPattern`, `SvgImage`,
  `ForeignObject`, `Symbol`, `Marker` (the runtime already namespaced these;
  helpers were missing), enabling native-Go charting.
- **CSS base layer & extraction** — `css.Preflight` / `css.PreflightInLayer` (an
  opt-in modern reset, CSS5) and `css.CriticalCSS` (the extract half of the
  author-in-Go / ship-inline SSR pipeline, paired with `SeedFromDocument`, CSS6).
- **Base-href-safe links** (G7) — `router.Href` (mode-aware href: `#path` for hash
  routers, bare path for history) and `router.FragmentHref` (in-page anchors that
  embed the live path so they survive `<base href>`).
- **Lifecycle & environment hooks** — `ui.UseMount` (run-once-on-mount with
  cleanup, G38); `ui.UseMediaQuery` (reactive `matchMedia`, G20);
  `ui.UseNetworkStatus` (reactive `navigator.onLine`, G32); `ui.ViewTransition`
  (View Transitions API with graceful fallback, G27).
- **Element-ref hooks** — `ui.UseElementGeometry` (measured box, ResizeObserver,
  G26); `ui.UseIntersection` (IntersectionObserver visibility, G27);
  `ui.UseAnimationRestart` (double-rAF keyframe replay, G27);
  `ui.UsePointerEvents` / `ui.UseWheel` (managed element pointer/wheel listeners,
  G35). Plus `interop.WrapElement` to bridge a DOM node to the interop Element
  surface. All have native no-op stubs.
- **Router navigation lifecycle** — `router.OnNavigate(fn)` fires on each real
  navigation with the new Location (scroll-reset / analytics / title seam, G28).
- **CSS cascade layers** — `css.Layer`, `css.LayerGlobal`, `css.DeclareLayers`
  for `@layer`-based override precedence instead of order/`!important` accidents
  (CSS4).

- **Reactive routing — `router.UseRoute` / `router.UseLocation`** (G6): a hook
  that subscribes the calling component to navigation and returns the active
  `router.Location` (path, query, params). Memoized chrome (active-nav highlight,
  breadcrumb) now stays in sync on navigation without threading the path down as
  a prop. The router publishes the live location into a well-known atom on every
  render (deduped on path+query).
- **Layout effects — `ui.UseLayoutEffect`** (G36): runs synchronously after the
  commit mutates the DOM, before paint, and before the same component's passive
  `UseEffect` callbacks. Removes the `setTimeout`/`rAF` guesswork for post-render
  focus/scroll/measure work. Backed by an `Effect.Layout` lane in the reconciler.
- **Global / token CSS — `css.Global`, `css.Root`, `css.Inject`, `css.Within`,
  `css.DataTheme`** (CSS1/CSS2/CSS3/G30): author un-prefixed top-level rules
  (element/`:root`/semantic-class selectors), a `:root` custom-property palette,
  ancestor-attribute (`[data-theme="…"] &`) variants, and runtime `<style>`
  injection keyed by id — all hardened against `<style>` breakout and flowing
  through the same SSR/Harvest sink as `New`.
- **Non-hook shared state — `state.GlobalAtom[T]`** (G39): read/write a shared
  atom from ANY context, including outside render (global key handlers, undo/redo,
  background goroutines). Targets the same registry as `UseAtom`, so writes
  re-render subscribers; pre-render writes persist instead of silently dropping.
  Backed by a no-notify `runtime.InitAtomValue` seeding primitive.
- **Promise→Go bridge — `interop.Value.Await` / `AwaitCall`** (G27/G31/G33/G34):
  resolve a JS Promise into Go with context cancellation and managed `js.Func`
  lifetime, collapsing the manual `then`/`catch`/`Release` chain. Plus a
  **Notifications wrapper** (`interop.RequestNotificationPermission`,
  `PostNotification`, `NotificationPermissionState`).
- **CSS-safe ids — `ui.CSSEscape` / `ui.SelectorID`**: WHATWG-compliant identifier
  escaping for building selectors from arbitrary ids.

### Fixed

- **CSS-unsafe generated ids** (G29): `UseId` now emits `gwc-N-N` (hyphen
  separator) instead of `gwc:N:N`. The old colon form threw a `SyntaxError` in
  `querySelector("#"+id)` and could panic a wasm callback; generated ids are now
  valid CSS identifiers needing no escaping.

## v3.2.0 - 2026-06-20

### Added

- **Typed, type-safe CSS (`css` + `css/u`)** — a raw-CSS layer (typed values,
  properties, variants, SCSS-style selector composition, a hashed/deduped
  registry behind a `Sink`, runtime `<style>` injection + SSR buffer) with a
  Tailwind-shaped utility layer on top. `css.Class(...any)` mixes literal strings,
  typed rules, variant slices, and pre-folded sheets (clsx-style). Emitted CSS is
  hardened against `</style>` breakout; `Hex`/`Var` validate by construction.
- **DOM & lifecycle primitives** — `ui.UseDOMRef` (real element ref via
  commit-phase capture) + `html.Ref`/`shorthand.Ref`; `ui.UseAutoFocus`
  (focus-on-mount); `html.RawHTML`/`RawHTMLUnsafe` (markup parsed to real nodes —
  no innerHTML sink — via `x/net/html` + the `sanitize` allowlist);
  `ui.UseDocumentEvent`/`UseWindowEvent`/`UseGlobalKey` (managed global listeners
  with effect-scoped `js.Func` lifetime); `ui.OnReady` + a `gwc:ready` DOM event
  (deterministic first-render signal); `ui.UseForceUpdate`;
  `ui.UseTimeout`/`UseInterval`; `ui.Download`/`ui.PickFile`; `html.Bind`
  (two-way input binding); `a11y.RadioGroup` and `a11y.AlertDialog` (headless
  WAI-ARIA); the `textutil` package (`Humanize`/`TitleCase`).
- **`tools/hookcheck`** — a static `go/ast` "rules of hooks" analyzer (library +
  CLI) that flags `Use*`/`On*`-handler hooks called inside loops, with FuncLit
  awareness and a `//hookcheck:ignore` directive. No `x/tools` dependency.
- **Client-side SQLite (`db/sqlite`)** — pure-Go SQLite in the browser
  (in-memory / IndexedDB-snapshot, no cgo) with `Exec`/`Query`/`Tx`/`Flush`, plus
  **encryption at rest**: `Options.Encryptor` + `NewPassphraseEncryptor`
  (PBKDF2-HMAC-SHA256 → AES-256-GCM, key never stored, tamper-evident). See
  `encryption.go` for the threat model.
- **Durable reactive state (`kvstate`)** — `UsePersistedState`/`BindAtom` over a
  pluggable `PersistenceBackend` (default SQLite), with JSON/CBOR codecs,
  Immediate/Debounced/OnUnload write strategies, LastWriteWins/Versioned conflict
  resolvers, BroadcastChannel cross-tab sync, a named registry, and
  `Export`/`Import` — the ingress/egress surface for cross-app/domain/device sync
  over any backend.

### Changed

- `css` selector helper `Ref` renamed to `SheetRef` (`u.SheetRef`) so the
  universal DOM-ref `Ref` is unambiguous when dot-importing.
- `shorthand.Class` (the string class setter) renamed to `shorthand.ClassStr`;
  call sites migrated. The typed `css.Class(...any)` subsumes the string form.

### Fixed

- SSR no longer leaks the internal DOM-ref key as a bogus attribute.
- `a11y.RadioGroup` emits the roving `tabindex="0"` via `Raw` (the `TabIndex==0`
  serializer omission would otherwise erase the single tab stop).

## v3.1.0 - 2026-06-13

### Added

- **Agentic detached rendering** â€” `bridge.render-tree` can inject allowlisted
  structured element trees into live selectors through an isolated detached
  runtime; snapshots now include detached roots and full text content, and
  `agenthub` falls forward from stale sessions to the latest active session.
- **Atlas Commerce OS coverage matrix and screenshot baselines** â€” the Atlas
  browser-flow manifest now includes public/internal screenshot baseline
  targets, PNG validation, a Playwright capture harness, and a route/feature
  coverage matrix that verifies test/story references stay real.
- **Agent bridge demo app** â€” added a small public wasm demo whose visible
  state is atom-owned so the live agent bridge can change copy, count, colors,
  and shape state in the browser.

- **Agent bridge** — a first-class surface for agents to inspect and drive a
  live app. Runtime `agent_read`/`agent_write` primitives resolve stable node
  refs and apply state writes (with `agent_state_version` guarding stale
  writes); the `agentbridge` package wraps them in a versioned read/write/control
  command protocol (native + wasm clients); the standalone `tools/agenthub`
  module hosts an out-of-process session hub; and `gwc` gains `agentic` live-
  bridge, rebuild, snapshot-diff, and export-test commands. Dogfooded in the
  `ai-chat-wizard` showcase, covered by an `example100` Playwright-Go dogfood
  e2e and a headless CI workflow, with a threat model under `security/`.
- **Agent bridge — self-description, safety, and replay** — `bridge.describe`
  now returns a complete control manifest (atoms with JSON schemas, event
  topics with subscriber counts, navigable routes, mountable components, and
  typed-publishable topics) with a stable shape. Added an audit trail
  (`bridge.audit`) covering every mutating command, reversible mutations with
  `bridge.undo`, a `dryRun` preview for `set-atom`, deterministic capture/replay
  (`bridge.replay`), and a single-writer lease. `events.RegisterTopic[T]` +
  `events.PublishJSON` let a JSON publish reach concretely-typed subscribers;
  `events.Topics`/`SubscriberCount` and `router.RegisteredRoutes` expose the
  live vocabulary. All bridge verbs are exposed as `gwc` CLI subcommands and
  MCP tools. Compiled in only under the `gwcagent` build tag + `?gwc-dev=agent`
  and excluded from release builds; see `docs/PRODUCTION_READINESS.md`.
- **`gwc browser` + `gwc screenshot` — DevTools/CDP browser-proxy backend** —
  `gwc mcp` now has a second backend alongside the in-wasm semantic bridge: a
  Playwright/CDP browser proxy for the pixels and real windows the structural
  bridge can't reach. `gwc browser` opens a **headed (visible) Chromium** window
  (`--no-sandbox`) on a dev URL and keeps it open with a CDP debugging port, so
  the engineer watches the app live during copilot dev while the agent drives
  it. `gwc screenshot` captures a pixel-perfect PNG — full page or a single CSS
  selector — either by launching its own headless Chromium, or with `-cdp` by
  **attaching over CDP to the engineer's open `gwc browser` window** to capture
  exactly what is on screen. Both are exposed as CLI subcommands and
  auto-generated MCP tools (`gwc_browser`, `gwc_screenshot`), emitting the
  standard agentic envelope. Compiled only under the `playwrightgo` build tag to
  keep the browser toolchain out of the lean default binary; covered by
  `playwrightgo` tests that open a real headed window and assert valid,
  non-trivial PNG output (the cross-process attach flow is verified end-to-end).
- **Browser-proxy verbs — input, capture, read, visual diff** — the CDP proxy
  now gives an agent hands, ears, and sight beyond the GWC fiber tree, not just
  eyes. **Input:** `gwc click` (selector or x/y), `type` (Fill or `-append`
  keystrokes), `press` (keys/chords), `hover`, `scroll` — real DOM input over
  CDP, reaching detached/third-party/raw-selector elements that the in-wasm
  `emit` (which only calls registered fiber handlers) cannot. **Capture:**
  `gwc console` (console messages + uncaught JS errors, with `-reload` for
  boot-time output) and `gwc network` (requests/responses/failures, non-2xx
  flagged) read the browser's CDP event stream that the wasm `logs` command
  can't see. **Read:** `gwc dom` (real rendered text/HTML/attributes for a
  selector) and `gwc eval` (read-oriented JS evaluation; dual-use, gated like
  the rest of the proxy). **Visual diff:** `gwc screenshot-diff` compares two
  PNGs (changed-pixel count/%, per-channel `-threshold`, highlighted diff
  image, `ok=false` over `-fail-over`) — the pixel counterpart to
  `snapshot-diff`, and the only proxy verb that ships in the **default** build
  (pure `image/png`, no browser). Every verb supports `-cdp` (attach to the
  engineer's window) or `-url` (launch its own headless Chromium), emits the
  standard envelope, and is an auto-generated MCP tool. Covered by unit tests
  (diff math) and `playwrightgo` tests that drive real Chromium (input mutates
  the DOM; console/network/dom/eval assert exact captured values); demonstrated
  live end-to-end (click→eval→screenshot-diff on a running window).
- **Browser-proxy verbs — verify, diagnose, input completeness** — second round
  of CDP-proxy verbs closing the verify and diagnose loops. **Verify:**
  `gwc expect` (assert a DOM/page condition — selector/visible/text/count/eval —
  and return `ok` = whether it holds, so an agent can gate a step) and `gwc wait`
  (block until a DOM condition holds; the browser counterpart to bridge
  `wait-for`). **Diagnose:** `gwc trace` (record a replayable Playwright
  `trace.zip` — DOM snapshots + screenshots + network) and `gwc a11y` (dump the
  accessibility tree's roles + accessible names via a raw CDP
  `Accessibility.getFullAXTree` session). **Input completeness:** `gwc select`
  (`<select>` by value/label), `gwc upload` (file inputs), `gwc drag` (drag one
  selector onto another) — finishing the set begun with click/type/press/hover/
  scroll. **Network write side:** `gwc mock` intercepts a `-route` glob and
  stubs (`-status`/`-body`) or `-abort`s it to exercise error paths. All `-cdp`/
  `-url`, auto-generated MCP tools, covered by `playwrightgo` tests against real
  Chromium. Also fixed the `dev_loop_browser_e2e` harness (it now rewrites every
  relative `replace` directive — including `agenthub` — to an absolute path when
  copying the livereload module, so that lane passes again).
- **`events` introspection** — `events/introspect` exposes the live topic/
  subscriber graph for tooling and the agent bridge.
- **Gesture primitives** (`anim`) — pure pan/drag and pinch primitives tracking
  delta, velocity, center, scale, end-state, and zero-distance guards.
- **`gwc test -lane i18n`** — runs the message-extraction / locale-completeness
  checks as a CI-addressable lane (included in `all`).

- **Selective / progressive hydration (islands)** — `ui.HydrationIsland` marks
  independently resumable SSR islands; `ui.HydrateIsland` schedules per-island
  browser hydration on visible / interaction / idle / immediate triggers (with
  optional timeout); `ui.ConfigureHydrationIslandBudget` caps concurrent island
  hydration and `ui.InspectHydrationIslandBudget` validates startup/deferred
  island plans. Attacks the wasm-startup gap directly.
- **Route-level code splitting** — router routes declare a `RouteChunk` via
  `RegisterLazy` / `RegisterLazyRoute` or `Options.Chunk`; the router gates
  rendering on the chunk, runs it before the route factory/loader, cancels stale
  chunk loads on navigation, supports a custom `Loader` or script URLs, and
  routes pending/error states through chunk- or route-local fallbacks.
- **`a11y` package** — headless menu, combobox, listbox, datepicker-grid, and
  table builders over the overlay/focus/composite primitives, with semantic
  HTML/ARIA contracts.
- **`servercomponents` package** — server-rendered component boundaries
  (native + wasm split) for server-driven UI.
- **`telemetry` package** + `internal/telemetryredaction` — opt-in telemetry
  with structured redaction of sensitive fields.
- **`scheduler` package** — a standalone cooperative task scheduler.
- **`agentbridge` package** — an agent-facing bridge surface over the framework.
- **Browser devtools extension** — `devtools` now generates a Chrome/Firefox
  Manifest V3 and emits the stable `gwc.devtools.extension.v1` panel payload
  (component tree, props/state inspection, extension sections / atom-graph
  contributions, diagnostics, logs, commit profiling).
- **Agentic toolchain (`tools/gwc`)** — `agentic_toolchain` consolidates the
  agent-facing tool surface.
- **Runtime controls & memory hygiene** (`internal/runtime`) — first-class
  runtime controls, passive-event support, and memory-hygiene passes.

- **`sanitize` package** — a DOMPurify-equivalent HTML sanitizer built on
  `golang.org/x/net/html`: allowlisted tags/attributes/URL-schemes, drops
  script/style/iframe/svg and `on*`/`style` attributes, unwraps unknown tags.
- **`events` package** — `UseTopic` typed in-app pub/sub fan-out bus with
  lifecycle-tied subscriptions, replay-last opt-in, and panic-contained delivery.
- **`anim` package** — spring physics (semi-implicit Euler + presets), standard
  easings with `Interpolate`, and `ComputeFLIP`.
- **`deprecation` package** — `Warn(api, replacement)` emits a one-time
  `GWC-DEPRECATION` structured diagnostic.
- **`ui` hooks** — `UsePrefersReducedMotion`, `UsePrefersColorScheme`, and
  `UsePersistedState[T]` (write-through + cross-tab sync + corrupt/quota-safe).
- **`fetch`** — declarative resilience (`RetryPolicy`, `CircuitBreaker`,
  `ExecuteWithPolicy`) and `ScopeCacheKey` for the global cache namespace.
- **`flags`** — `RemoteProvider` for remote config (poll, fail-safe last-known-
  good, staleness, kill-switch, contained payloads).
- **`i18n`** — `FormatRelativeTime` and `FormatList` (en/fr/ja/ar); new
  `i18n/extract` message-extraction + locale-completeness tooling.
- **`interop`** — typed cookie helper (`GetCookie`/`SetCookie`/`ExpireCookie`),
  `RequestPersistentStorage`/`IsStoragePersisted`, a WebCrypto bridge
  (`GenerateAESKey`/`Encrypt`/`Decrypt` + `EncryptedStore`), and an opt-in
  Browser Intl bridge (`IntlFormatNumber`/`IntlFormatDate`).
- **Streaming SSR** (`ui.RenderToStream`/`RenderToStreamObserved`) and
  render-thread hook-threading enforcement (`GWC-RUNTIME-HOOK-THREADING`).
- **Versioned state-snapshot migration** in `hotreload` (app-owned
  `snapshotVersion` + ordered migrations).
- **Crash containment** as the runtime default: panics in render/event/effect/
  cleanup/async are contained and reported as structured agent-readable
  diagnostics instead of killing the page.
- **`gwc dev` auto-doctor** — environment diagnosis on dev-server failure, with
  a `-no-doctor` opt-out.
- **Tooling & docs guards** — `docs/doclint` doc-drift guard (repo paths +
  `gwc` flag existence), generated `docs/errorcodes` reference and
  `docs/capabilities` matrix, and `tools/changelogcheck`.
- **Docs** — CONTRIBUTING.md, CONVENTIONS.md, SECURITY.md,
  PRODUCTION_READINESS.md, ACCESSIBILITY.md, BENCHMARKS.md, the generated
  error-code and capability-matrix pages, and committed `.vscode/` editor config.
- **Docs site** — generated favicon and OG/Twitter social-preview assets.

### Changed

- **Example 100 admin operations** â€” persisted user-list query state, surfaced
  active/blocked status badges, added reasoned disable/restore confirmation,
  wired billing access and quota override actions, and pinned admin wasm parser
  coverage.
- **Atlas Commerce OS TODO closeout** â€” completed the remaining Atlas public,
  internal, performance, accessibility, localization, preference, SEO,
  recovery, and testing backlog markers with shipped code, docs, tests, and
  screenshot evidence.
- **Starter feature matrices** â€” generated starter `FEATURE_MATRIX.md` files
  now use explicit `selected` / `available` labels instead of unchecked
  checklist boxes, keeping TODO scans focused on real backlog.

- **ai-chat-wizard "Aurora" redesign** — ground-up design-token CSS system
  (violet accent, Geist/Space Grotesk/Geist Mono type), open-canvas assistant
  replies in a 46rem reading column, floating composer, and a framework-API
  showcase: `anim` springs baked into CSS keyframes, `ui.UsePersistedState`
  density toggle, `ui.UsePrefersReducedMotion`, `ui.UseAnnouncer` stream
  completion, `i18n.FormatRelativeTime` sidebar timestamps, browser-ICU cost
  formatting via `interop.IntlFormatNumber`, and the new shorthand helpers
  (`MapKeyed`, `MapKeyedIndexed`, `Repeat`).
- **Go 1.26 toolchain** — root and `tools/livereload` go.mod bumped to
  `go 1.26.0` (Green Tea GC by default, lower cgo overhead, better slice
  stack-allocation), and the `go fix` modernizer suite applied module-wide
  (~550 files): `interface{}` → `any`, `for i := range n`, built-in
  `min`/`max`, `maps.`/`slices.` helpers, `strings.CutPrefix`/`SplitSeq`,
  Go 1.26 `new(expr)`, and `wg.Go()`. Mechanical rewrites only; native and
  js/wasm builds, vet, and package tests verified.
- **Lint-clean framework** — `gwc lint` (golangci-lint + gwc-hooks) now
  reports zero issues across all 47 non-example packages (was 41:
  errcheck/gosimple/ineffassign/staticcheck/unused/gwc-hooks). Dead helpers
  removed, hook calls hoisted to named top-level component functions
  (`virtualization` row body, `html.toHandler`, example tests), wasm-only
  false positives annotated with reasons.
- README leads with a 30-second golden-path quickstart; capability summary and
  Public Packages refreshed (i18n/interop/pwa/virtualization).
- Legacy `gwc` flag aliases (`-main`/`-index`/`-output`) are labeled deprecated
  in help.
- Bumped `golang.org/x/net` to v0.55.0.

### Performance

- **Runtime inspector flamegraph collection** — `collectFlamegraphFrames` no
  longer preallocates a fixed 256-frame backing array per call or copies the
  full ancestor path at every node; it grows the result lazily, reuses one
  push/pop path stack, and short-circuits at the frame limit. Cuts the
  representative-scenarios inspector snapshot from ~36 KB to ~5.3 KB per op
  (−85%) and ~1.9× faster, speeding up every devtools/agent snapshot query.
- **SSR attribute serialization** — `writeSSRProps`/`serializeProps`/
  `serializeStyleMap` collect attribute keys in a stack-allocated buffer and
  sort with `slices.Sort` (no `sort.Interface` boxing), removing one heap slice
  per host element during server-side rendering.
- **Telemetry redaction** — `redactValue` builds array element paths by string
  concatenation instead of `fmt.Sprintf`, cutting ~26% of allocations and ~29%
  of time on the redaction path that runs for every redacted telemetry record;
  added the package's first benchmark (`BenchmarkRedactValueNestedArrays`).

### Fixed

- **GitHub Actions GoGRPCBridge gates** -- workflows now use the latest Go
  `1.26.x` patch toolchain, check the canonical `GoGRPCBridge` module/path,
  delegate bridge WASM example builds to the bridge runner, and install the
  Playwright version used by the root module.

- **Agent bridge hardening** — 18 defects found and fixed across six adversarial
  review rounds, each regression-tested: a `SendCommand` socket-death bug
  returning a zero-value ack with no error; the wasm client never stripping the
  leading `?` so the bridge never activated from a real URL; mount/unmount
  check-then-render TOCTOU races and a stuck-reservation leak; `bridge.undo`/
  `bridge.replay` bypassing the hub write lease; a data race on the runtime
  replay buffer; `set-atom` silently accepting a wrong-typed value for a
  concrete slice/map atom; a non-constant-time token compare; an unbounded undo
  stack; `replay` silently resetting on double-start; an unbounded
  predecessor-chain recursion; and missing audit-trail coverage on several
  mutating commands. Real-browser dogfood verified after the fixes.
- `logging` (wasm) — console output now leads with the message string before
  the structured record: the record object's console preview shows only a few
  properties in nondeterministic Go-map order, which made messages unreadable
  in devtools and browser-test log matching flaky.
- ai-chat-wizard — stub providers stream word-by-word with configurable pacing
  (`CHAT_STUB_CHUNK_DELAY_MS`; `0` restores single-shot), fixing the
  happy-path browser test that raced the instant single-delta stream; the
  server also accepts the legacy `CHAT_STUB_PROVIDERS` spelling.
- `logging` — `slog.Attr` values passed as log args were collapsed to their
  string form because the `fmt.Stringer` case matched first; Attrs now
  normalize to key/value maps as intended.
- ai-chat-wizard example — admin redaction helpers shallow-copied generated
  protobuf messages (copying their internal mutex, a `go vet` copylocks
  violation); they now use `proto.Clone`.
- Repointed ~136 stale example paths across README, AGENTS.md, the reference
  manual, and runbooks (the `examples/01-counter` → `examples/public/counter`
  relocation and the renamed SSR/PWA examples), un-breaking the front-door
  copy-paste quickstart commands.

### Security

- **Markdown URL-scheme XSS** — `html.RenderMarkdown` now allowlists
  http/https/mailto/relative/# and drops `javascript:`/`data:`/`vbscript:`
  (incl. obfuscated) in links, images, and autolinks.
- **Live-reload CSWSH** — the dev-server WebSocket validates the request Origin
  against the dev host:port (opt-out `-allow-any-origin`, with a warning).
- **Raw-HTML sink removed** — `SetInnerHTML` is gone from the DOM adapter
  interface and the wasm DOM, with a reflection guard preventing reintroduction.
- **Dependency CVEs** — the `golang.org/x/net` v0.55.0 bump clears 5
  govulncheck findings in the HTML parser; `govulncheck` now runs in CI
  (informational) and SECURITY.md documents the disclosure policy.

## 2026-04-08

### grouped examples layout and public site refresh

- Reorganized the runnable examples into grouped `examples/public`, `examples/server`, and `examples/testing` trees, removed checked-in generated host HTML entrypoints, and updated repo docs, launcher paths, livereload expectations, and Playwright fixtures to match the new layout.
- Added `gwc examples build-public-site`, staged preview hosts plus mirrored source under `examples/public-examples-site/assets/`, and updated the Pages workflow to publish the renamed public examples site with embedded example wasm binaries.
- Introduced `examples/internal/exampleboot` plus selector-aware example logging so embedded and standalone public examples can share mount, hydrate, and keepalive behavior without duplicating host glue.

### plugin runtime and devtools hardening follow-up

- Fixed plugin host and internal plugin kernel cleanup and reboot behavior so closed kernels reject reuse, builtin services rebind on reboot, repeated host extension registration reference-counts correctly, and closed hosts drop stale contributions.
- Repaired runtime effect scheduling and early UI event capture so first-render effects still run after commit and kernel-backed UI event services can observe interactions before the first snapshot read.
- Replaced goroutine-based devtools snapshot polling on `js/wasm` with browser timers, broadened shorthand prop handling for the public examples site rendering surface, and tightened focused regression coverage around the repaired paths.

### public examples site performance and cache lifetime fixes

- Reduced the public examples browser hot path by deferring preview and source rendering until requested, turning debug event logging off by default, and falling back to plain source rendering for large files instead of always running the expensive tokenizer path.
- Added bounded Cache Storage handling for preview wasm binaries so the shell keeps only the current and most recent example builds, expires stale entries, removes superseded versions, and clears the older preview cache namespace to avoid steady RAM and storage growth.
- Regenerated the checked-in public examples shell plus preview hosts so the staged site reflects the deferred loading, bounded preview cache lifetime, and lighter browser-side rendering behavior.

### public example presentation normalization

- Restyled the shared example shell and the remaining bespoke public demos to match the compact counter example more closely, removing docs-like framing, trimming duplicate purpose copy, and emphasizing the live control surface.
- Added compact routed wrappers for the portfolio docs and not-found flows, then refreshed the mirrored `examples/public-examples-site/assets/code/` source tree so the public catalog reflects the clearer purpose-first example layouts.
- Applied the same compact text treatment to the generated preview host so the public examples browser de-emphasizes long explanatory paragraphs and oversized bullet lists across the staged catalog.

## 2026-04-07

### core plugin kernel and first-pass devtools plugin

- Added an internal plugin kernel plus typed interposer-backed services for runtime, router, fetch, UI, assets, security, and devtools so first-party plugins can reach deeply into framework state without binding to unstable runtime implementation details.
- Reworked `devtools` to compose app-owned sections and actions with kernel-owned plugin contributions, added the first-pass kernel bridge and capture service, and documented the new companion-host versus kernel split across the package docs and READMEs.
- Added the kernel-plugin devtools example and focused Playwright coverage proving the pluginized devtools flow renders live framework sections through the new kernel path.
- Added the implementation plan and granular implementation backlog docs for the plugin framework so the shipped kernel surface, rollout, and remaining follow-on work stay explicit.

### wasm and browser regression hardening for the plugin pass

- Fixed `ui.WrapHandler(...)` and parallel-region event bridging so late-installed bridged handlers are wrapped through the active DOM adapter instead of panicking or leaving stale host props cached in the runtime.
- Hardened `internal/platform/jsdom` append behavior for browser test shims, stabilized browser-compiler generator assertions across Windows line endings, restored the embedded livereload client asset, and taught `gwc tailwind` about Windows `arm64` binaries.
- Added focused native and `js/wasm` regression coverage around the repaired handler bridge, jsdom append fallback, and parallel-region worker event paths while also validating the broader non-example native, wasm, browser, and Playwright suites.

### plugin-host devtools extension seam

- Extended the `plugin` companion host with capability-gated devtools section and overlay-action providers so consumer-owned plugins can contribute richer diagnostics UI without widening runtime internals or relying on hidden discovery.
- Added `devtools.ApplyHostExtensions(...)` to map host-owned devtools contributions into the existing extension-section and error-overlay state, with cleanup that restores the prior app-owned devtools wiring.
- Covered the new seam with focused `plugin` and `devtools` tests for capability enforcement, cloned reads, rollback safety after failed setup, and host-to-devtools action-context mapping.

### waitforever starter and example wiring cleanup

- Added the production `utils.WaitForever()` stub so release-tag `js/wasm` programs can block on one named helper instead of open-coded `select {}` loops.
- Replaced raw `select {}` keeps-alive paths in Example 21, catalog implementation copy, launcher import output, launcher start output, and the starter golden files with `utils.WaitForever()` so the generated guidance and emitted app skeletons stay aligned.

### example 100 customer-error fixture seeding cleanup

- Simplified the Example 100 customer-error Playwright setup to reuse the deterministic happy-path database seeding helper instead of copying a prebuilt runtime fixture database from disk.

### runtime2 renderer and benchmark performance pass

- Reduced `runtime2` host and commit overhead with append-only and remove-only patch-transaction fast paths, narrower rollback snapshots, cheaper insert-only host patch cache handling, and focused compare benchmarks for the new commit paths.
- Cut `js/wasm` DOM bridge cost in `internal/platform/jsdom` and `internal/runtime` by caching prepared host mounts, reusing variadic batch buffers, keeping append batching active through child-order repair, and broadening reconciler optimization coverage and micro-benchmarks.
- Improved Example 201 worker-backed benchmark paths by adding worker-native parallel-region rendering hooks, more adaptive worker batching and cache reuse, and clearer worker timing diagnostics for the React 19 comparison runs.

## 2026-04-06

### runtime1 render-path and example 201 benchmark pass

- Reduced `runtime1` host-render overhead across reconciliation and DOM commit by separating host-only props, adding direct-text host storage, tightening compact host mount paths, caching component render metadata, and broadening targeted runtime coverage and benchmarks.
- Improved Example 201’s benchmark surface and harness by trimming unnecessary primitive/deep-tree benchmark work, updating the local React 19 benchmark bundle path, and fixing benchmark report gaps around worker metrics and score inputs.
- Validated the shipped `runtime1` slice with focused package tests plus the scoped Example 201 Playwright benchmark run instead of the unrelated broader browser suites.

### core logging overhaul and wasm panic console capture

- Reworked the core `logging` package into a shared structured record pipeline with native JSON-line output, browser `console.*` object output, low-ceremony key/value logging, and context-backed correlation plus trace enrichment.
- Added focused native and `js/wasm` logging coverage for variadic field normalization, context metadata propagation, stable slog-like record fields, and browser-console level mapping.
- Emitted framework-owned `js/wasm` panic reports as structured `console.error` records with slog-like level metadata, documented the panic-reporting contract in the reference manual, and added focused browser tests for panic capture plus hidden raw rethrow suppression.

### reference manual consolidation and api browser

- Replaced the fragmented top-level `docs/*.md` app-authoring set with a consolidated `docs/REFERENCE_MANUAL/` chapter set and removed the legacy duplicate docs from the repository.
- Added ordered topic pagination across the manual plus a dense `16-api-browser.md` chapter that catalogs the public packages, high-level usage, key parameter objects, returned handles, and source anchors.
- Rewired root docs, package READMEs, example READMEs, agent guidance, and devtools doc pointers to point at the consolidated manual, and removed the obsolete docs-ingestion planning and `tools/doc_ingest/` assets.

### browser compatibility workflow checkout

- Updated `.github/workflows/browser-compatibility.yml` to fetch submodules recursively so CI jobs that depend on the pinned `third_party/GoGRPCBridge` checkout can resolve modules and compile the browser suite.

### core runtime state and virtualization coverage cleanup

- Fixed a public `ui.ParallelRegion(...)` panic path by guarding `reflect.Value.IsNil()` behind nilable-kind checks.
- Fixed mixed keyed and unkeyed child reordering in `internal/runtime/` so reused DOM nodes are moved into their committed sibling order instead of being left in stale positions.
- Split browser-only virtualization helpers into focused native and wasm files, normalized persisted restoration JSON keys to `scrollTop` and `anchorKey`, and expanded `state`, `virtualization`, and `mockdom` coverage around the repaired paths.

### runtime2 coverage and patch helper hardening

- Fixed `internal/runtime2` style canonicalization so map-backed styles normalize the same way as equivalent string styles, including trimmed scalar values.
- Added broad regression coverage across runtime2 patch-building, snapshot hashing, host-region, recovery, and transport helpers so the remaining low files in `internal/runtime2/` clear the per-file coverage floor.

### fetch native coverage and cache correctness

- Split browser-only fetch and upload transport code out of `fetch/fetch.go` into focused wasm and native files so shared cache and mutation logic is testable off-wasm.
- Fixed stale cached-resource handle mutations after disposal or replacement, canceled in-flight loads before optimistic cache updates, retried persistent cache-store opens after failures, honored `DeleteOnCorruption`, and treated nil response bodies as empty text instead of `"<nil>"`.

### browser support coverage and interop guards

- Removed unused browser-storage batching code from `interop/interop_wasm.go` and aligned the wasm tests with native JavaScript throw behavior instead of Go callback panics.
- Hardened hot-reload bridge registration cleanup, preserved structured console envelope keys under collisions, clarified browser console window-error field names, and expanded `devtools`, `router`, `logging`, `hotreload`, `i18n`, `interop`, and `utils` wasm coverage.

### pwa promise guards and wasm coverage

- Fixed `pwa` service worker, cache storage, and installability await helpers so primitive browser return values no longer panic when inspected for promise methods under `syscall/js`.
- Added focused GoDocs and intent comments to the repaired `pwa` browser helpers and expanded wasm coverage for service worker lifecycle, cache diagnostics, and installability error handling.

### repository navigation refresh

- Added a human-first repository map in `docs/REPO_MAP.md` and linked it from the root README and docs index.
- Added or tightened local entry docs for `bin/`, `log/`, `third_party/`, `internal/`, `ui/`, and `interop/` so contributors can find the right starting points faster.
- Updated ignore rules so local runtime outputs stay out of source control while the new folder READMEs remain tracked.

### core runtime hot-path and wasm storage cleanup

- Replaced several formatting and normalization hot paths with cheaper builders across `internal/runtime/` and `internal/runtime2/`, including hook IDs, renderer feature-flag lookup, render-style scalar formatting, remove-attr dedupe keys, and scheduler shard IDs.
- Added focused benchmark coverage for those core fast paths so the new implementations are compared directly against the legacy paths.
- Hardened `interop/interop_wasm.go` storage access so non-callable browser storage methods return structured errors instead of panicking.

### chat-wizard managed start and seeding alignment

- Aligned Example 100 seeding and server runtime to the same default `chat_history.db` path.
- Taught `gwc` managed chat-wizard start flows to auto-seed missing or empty local databases and to build the client and background-worker wasm artifacts plus Brotli sidecars before launch.
- Updated the managed-start tests and operator smoke docs to match the repaired launch and seed flow.

### example 100 runtime fallback and copy refresh

- Made the composer `runtime2` display region fail open to inline rendering instead of panicking the app when region registration fails.
- Fixed nil-billing handling in the account cost view so missing billing responses no longer imply a configured platform fee.
- Corrected Example 100 README, in-app dev-tool pointers, and fallback catalog copy so the documented capabilities match the shipped example more closely.

### example 100 worker render performance pass

- Reduced redundant client worker work by gating refreshes on signatures, pruning request payloads, and removing extra clone staging before async worker dispatch.
- Replaced large concatenated signature strings with compact rolling-hash signatures in the app and worker paths.
- Reworked metadata extraction and conversion in the background worker to avoid full-content string copies, regex-heavy fenced-block parsing, and repeated label or fence normalization overhead, with focused wasm benchmark coverage for each step.

### runtime2 dispatch hashing and patch fallback

- Added typed scalar slice and typed `map[string]scalar` fast paths to the buffered and streamed runtime2 dispatch-hash encoders, along with regression coverage and compare benches.
- Taught canonical patch diffing to fall back to `replace-subtree` for root-ID structural deltas and added canonical DOM snapshot seeding for rebuilding region DOM state from canonical IR.

### public parallel-region click bridge

- Added runtime2 semantic event control envelopes, worker event dispatch handling, metadata-only registry lookup and mutation helpers, and metadata-aware worker renderer registration.
- Added `html.OnClickParallel(...)` and a bounded browser click bridge so local-first `ui.ParallelRegion(...)` nodes preserve their local click handlers while forwarding one semantic click event into runtime2 and committing the resulting worker patch.
- Seeded worker region state from already-rendered public nodes for post-mount patching and refreshed the runtime2 authoring and backlog docs to match the shipped behavior.

### GoGRPCBridge metadata forwarding

- Updated the pinned `third_party/GoGRPCBridge` submodule so grpctunnel forwards request ID, correlation ID, `traceparent`, and `tracestate` bridge headers into backend gRPC metadata.
- Added focused grpctunnel coverage proving the forwarded metadata reaches backend unary handlers.

## 2026-04-05

### runtime2 event-slot metadata

- Replaced the non-operative placeholder event-slot metadata slice with a normalized `v1` schema in `internal/runtime2/`.
- Added validation and transport coverage for declared slot and event pairs, legacy placeholder-version normalization, and duplicate or malformed slot rejection.
- Wired event-slot metadata through the runtime2 renderer registry so renderer capability metadata now carries validated slot declarations.
- Kept the legacy placeholder JSON helper names as compatibility wrappers over the normalized event-slot transport path and updated the runtime design notes to drop stale placeholder wording.

### plugin support-tier promotion

- Promoted `plugin` from experimental to supported companion in the package docs, root README, API policy, ecosystem docs, and comparison notes.
- Updated the first-party `examples/99-plugin-host` copy so the example matches the supported-companion classification.

### core runtime and platform package refactors

- Split several oversized core files into smaller focused units across `devtools/`, `fetch/`, `internal/runtime/`, `internal/runtime2/`, `interop/`, `router/`, `testkit/render/`, and `ui/` without changing the public package layout.
- Hardened core runtime behavior and coverage:
  - tightened `internal/runtime` adapter and scheduler contract coverage
  - fixed `utils` goroutine-monitor lifecycle handling for `js/wasm`
  - clarified `DOMNode` contract handling and expanded `mockdom` and `jsdom` validation
  - restored runtime2 shard and host-region invariants with targeted regression and benchmark coverage

### gwc tooling and livereload refactors

- Split large `tools/gwc` command files into focused command helpers for benching, imports, release, startup rendering, test lanes, wasm variants, verification, and doctor flows.
- Split `tools/livereload` into HTTP, path, and support helpers and kept `gwc dev` aligned with the full livereload source set.
- Updated starter scaffold output and the matching goldens, and added the `fsnotify` module requirement needed by the current tooling path.

### example app refactors and fixes

- Split several oversized example files in `examples/00-example-0`, `examples/14-omi`, and `examples/server/ai-chat-wizard` into smaller focused files.
- Refreshed example-backed docs and router snippets to match the shipped public router surface.
- Landed focused example fixes around the portfolio contact form, seeded test expectations, readonly ops reporting, billing normalization, logging redaction, and SSO SQL paths.

### docs backlog and reference refresh

- Reduced `docs/TODO.md` to the active backlog, removed completed execution history from the root todo file, and refreshed the remaining maintainability notes.
- Updated `docs/MULTITHREADED_RUNTIME_TODO.md` to reflect the runtime2 work that has already landed.
- Refreshed related reference docs and editor snippets in `docs/ACCESSIBILITY.md`, `docs/API_POLICY.md`, `docs/ERROR_BOUNDARIES.md`, `docs/IDE_INTEGRATION.md`, `docs/MIGRATIONS.md`, `docs/VIRTUALIZATION.md`, `docs/WORKFLOWS.md`, and `docs/examples/gwc-vscode.code-snippets.json`.

### docs ingestion planning assets

- Added `docs/DOC_INGESTION_PLAN.md` and `docs/DOC_INGESTION_TODOS.md` to capture the repository docs-ingestion design and approval flow.
- Added `docs/docs.sql` and `docs/seed_docs.ps1` to define and seed the docs-ingestion schema.

### repository layout maps

- Added a repo-layout section to `README.md` so contributors can quickly locate the public packages, internals, tests, tools, docs, examples, and local helper folders.
- Added local navigation READMEs for `agents/`, `scripts/`, `internal/platform/`, `internal/runtime/`, `internal/runtime2/`, and `tools/runnerconfig/`.
- Tightened `docs/README.md`, `internal/README.md`, and related folder maps so the larger implementation areas are easier to navigate without scanning the tree by hand.

### docs ingest tooling cleanup

- Moved the executable docs-ingestion assets out of `docs/` and into `tools/doc_ingest/` so the docs folder stays prose-first.
- Replaced the giant checked-in docs-ingestion SQL snapshot with a compact `tools/doc_ingest/schema.sql` schema file and a runnable `tools/doc_ingest/seed.ps1` local seed flow.
- Updated the docs-ingestion plan, todo notes, and tools docs to point generated outputs at `bin/doc_ingest/` instead of writing databases and temp files back into `docs/`.

### readme presentation refresh

- Replaced the README hero image with the new ChatGPT-generated artwork and moved the asset into `docs/assets/chatgpt.png` so the top-level repo no longer carries the long original image filename.

## 2026-03-28

### example 100: admin operator drilldowns, mutations, and rollups

- Expanded `examples/server/ai-chat-wizard` admin/operator backend coverage:
  - added typed business queue/detail, customer timeline, chats anomalies, providers drilldown, provider mutation, ops drilldown, and ops mutation RPC contracts in `proto/chat.proto`
  - added matching server handlers and focused tests under `server/app/` for the new dashboard/admin surfaces
  - added typed query/store wiring for customer timeline, billing dunning, failed payments, chat anomalies, provider routing/guardrails, and provider usage rollups
- Added supporting persistence and UI followups:
  - introduced provider usage daily rollup schema/migration support plus admin SQL files under `examples/server/ai-chat-wizard/sql/store/`
  - regenerated `proto/chat.pb.go` and `proto/chat_grpc.pb.go`
  - wired the business dashboard slice to reuse the workspace panel in `client/app/dashboard.go` and `client/app/dashboard_shell.go`

### docs and interop: worker validation fixes and demo backlog shaping

- Tightened browser interop contracts in `interop/interop_wasm.go`:
  - validate unsupported worker types before browser-surface discovery
  - resolve Go-worker URLs without assuming `document` exists
  - report closed popup peers as disposed for `WindowChannel.Focus` and `WindowChannel.Close`
- Fixed and expanded js/wasm validation in `interop/interop_wasm_test.go` for worker surfaces, cross-surface malformed-envelope handling, and structured-clone boundary failures
- Updated backlog/changelog tracking in `docs/TODO.md`, `examples/server/ai-chat-wizard/TODO.md`, and `examples/server/ai-chat-wizard/CHANGELOG.md`

### example 100: provider drilldown contracts, workspace followups, and worker interop validation

- Added follow-up provider drilldown scaffolding in `examples/server/ai-chat-wizard`:
  - introduced typed `GetAdminProvidersDrilldown` request/response contracts and `AdminProvidersDrilldownSummary` in `proto/chat.proto`
  - added `WorkspaceModelRoutingPolicyEntry` to the shared admin/control-plane proto surface
  - added store-side workspace model-routing policy row loading in `server/app/store_superuser.go`
- Added supporting workspace/admin wiring:
  - introduced `sql/store/ops/list_workspace_model_routing_policies.sql` and query-loader wiring in `server/app/queries.go`
  - applied small workspace-admin shell/table followups in `client/app/admin_workspaces.go`, `client/app/app.go`, and `client/app/app_shell.go`
- Expanded browser-lane worker interop validation in `interop/interop_wasm_test.go`:
  - direct coverage for empty worker URLs, unsupported worker types, empty request names, nil subscribe handlers, and Go-wasm worker URL/bootstrap descriptor failures
- Updated follow-up backlog tracking in `docs/TODO.md`

### example 100: cache core, admin operations, billing surfaces, and interop regression coverage

- Expanded `examples/server/ai-chat-wizard` with a reusable client cache subsystem and policy layer:
  - added `client/cachecore/` with shared scoped-record, SWR, invalidation, retry, storage, snapshot, worker-contract, and reconciliation primitives
  - added `client/cachepolicy/` with example-owned policies for dashboard, billing, thread, locale, catalog, and outbox resources
  - extended the background worker with maintenance-plan support and runtime wiring in `client/backgroundworker/`
- Broadened Example 100 product and admin surfaces across client, server, proto, and SQL:
  - updated client app state and views in `client/app/` for admin workspaces, account costs, profile/runtime data, and related shell constants
  - added or expanded server-side admin, customer-billing, pricing-page, conversation-metadata, public-auth, and superuser diagnostics handlers in `server/app/`
  - updated billing/store logic, formula guards, and plan/invoice persistence in `server/app/` plus `sql/store/`
  - regenerated `proto/chat.pb.go` and `proto/chat_grpc.pb.go` after `proto/chat.proto` changes
- Expanded regression coverage:
  - deeper browser interop wasm coverage in `interop/interop_wasm_test.go`
  - broader Example 100 Playwright coverage for admin journeys, authenticated flows, customer-safe error surfaces, and the full-stack demo sweep
- Updated backlog and project notes in `docs/TODO.md` and `examples/server/ai-chat-wizard/TODO.md`, and refreshed supporting example shell CSS in `examples/static/css/tailwind.css`

### example 100: admin dashboard data layer + five slice UIs

- Added live admin data fetching and full slice-level UI across the admin dashboard in `examples/server/ai-chat-wizard`:
  - Created `client/app/admin_data.go` with:
    - `adminDashboardData` render-only snapshot type and flat row types (`adminSummarySnapshot`, `adminUserRow`, `adminConvRow`, `adminProviderRow`, `adminDailyRow`)
    - `parseUseAdminDashboard` hook — async gRPC call to `GetAdminDashboard` with 30-day window, sequence-tracked cancellation, and loading / denied / error state transitions
    - `parseMarshalAdminDashboardResp` converter from proto response to flat snapshot
  - Extended `appViewState` in `client/app/app_shell.go` with `AdminDashboardData adminDashboardData` field; added `parseAdminDashboard` as the 10th parameter of `parseDeriveAppViewState`
  - Wired the hook in `client/app/app.go` after `parseUseAccountCostSummary`; updated `parseDeriveAppViewState` call accordingly
  - Registered all six dashboard routes explicitly in `ParseRun()` (`/app/dashboard`, `/app/dashboard/business`, `/app/dashboard/customers`, `/app/dashboard/chats`, `/app/dashboard/providers`, `/app/dashboard/ops`)
  - Rewrote `client/app/dashboard.go` with:
    - Path-based dispatcher `renderDashboardBody` routing to five slice renderers from a shared `renderDashboardSliceWrap` scroll container
    - Context-aware top bar (Back to Dashboard vs Back to workspace)
    - Slice tiles upgraded to real `<a>` links (`IsReady: true` on all five)
    - **Business** slice: platform-total KPI grid, 30-day window KPI grid, top-users-by-spend table, daily usage trend table
    - **Customers** slice: total-user count pill, recent-users table (display name, email, convs, messages, spend, joined, last seen)
    - **Chats** slice: conversation KPI cards, recent-conversations table with truncated preview, message count, cost, and last-activity date
    - **Providers** slice: provider health status table (label, available/not-configured status, auth, latency, request count, last error)
    - **Ops** slice: open-incidents, support-tickets, active-experiments, failed-events KPI grid + daily usage table
    - Shared primitives: `renderDashboardLoadingState`, `renderDashboardEmptyState`, `renderDashboardDeniedBanner`, `renderDashboardErrorBanner`, `renderDashboardTable`, `renderDashboardKPICard`, `renderDashboardSectionHeader`, `renderDashboardSliceHeader`
    - Formatting helpers: `formatDashboardInt64`, `parseDashboardUserLabel`, `parseDashboardShortAt`, `parseDashboardBool`
- Marked eight TODO items as complete in `examples/server/ai-chat-wizard/TODO.md`
- Validation:
  - `$env:GOOS="js" ; $env:GOARCH="wasm" ; go build ./examples/server/ai-chat-wizard/client/app/...`
  - `$env:GOOS="js" ; $env:GOARCH="wasm" ; go build ./examples/server/ai-chat-wizard/client/...`

## 2026-03-27 (continued)

### example 100 + router: history fragments, shell-route coverage, and landing boot cleanup

- Fixed several route and first-paint regressions across `examples/server/ai-chat-wizard` and the shared router:
  - client-side landing/login navigation no longer treats `/home -> /` as a no-op
  - extensionless `/app/...` routes like `/app/settings?panel=settings-profile` now serve the client shell instead of 404ing
  - pricing-page fragment routes now work on initial load, click, and browser back/forward for `/pricing#plans` and `/pricing#faq`
  - history-router core now preserves `#fragment` targets and rerenders on `hashchange`
- Fixed public-route loading flashes in example 100:
  - landing routes now bypass the auth-loading shell and render marketing content immediately after hydration
  - the bootstrap overlay uses route-aware copy for marketing pages
  - the bootstrap overlay is removed from the DOM after mount so stale boot text cannot reflow after hydration
- Added focused regression coverage in:
  - `router/browser_router_test.go`
  - `examples/server/ai-chat-wizard/client/app/route_sync_test.go`
  - `examples/server/ai-chat-wizard/client/app/marketing_fragment_test.go`
  - `examples/server/ai-chat-wizard/client/app/app_shell_test.go`
  - `examples/server/ai-chat-wizard/server/app/runtime_helpers_additional_test.go`
- Validation:
  - `go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./router -run TestBrowserRouterNavigateHistoryFragmentPreservesPath`
  - `go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./router -run TestBrowserRouterNavigateHistoryTargetPreservesFragment`
  - `go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./router -run TestBrowserRouterNavigateReplaceHistoryFragmentPreservesPath`
  - `go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./router -run TestBrowserRouterHistoryHashchangeRerendersCurrentRoute`
  - `go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./examples/server/ai-chat-wizard/client/app -run TestShouldNavigateLandingRoute`
  - `go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./examples/server/ai-chat-wizard/client/app -run TestParseResolvePricingFragmentScroll`
  - `go test -exec "C:\Users\Cam\Desktop\GoWebComponents\tools\go_js_wasm_exec.bat" ./examples/server/ai-chat-wizard/client/app -run TestShouldRenderLandingShellEarly`
  - `go test ./examples/server/ai-chat-wizard/server/app -run TestChatBootstrapLoaderTracksDownloadPhase`
  - `go run ./examples/server/ai-chat-wizard/cmd/build-client`
  - `go build ./examples/server/ai-chat-wizard/cmd/server`
  - live browser smoke against `http://127.0.0.1:8095/`, `/pricing#faq`, and `/app/settings?panel=settings-profile`

### example 100: operational schema coverage + table inventory

- Expanded `examples/server/ai-chat-wizard` schema/migration coverage for operational SaaS workflows with new tables for:
  - auth/session lifecycle: `auth_sessions`, `auth_token_versions`, `password_reset_tokens`, `email_verification_tokens`
  - workspace operations: `workspace_invitations`
  - webhook delivery history: `webhook_deliveries`
  - support threading: `support_ticket_messages`
  - incident timelines: `incident_updates`
  - async operations: `notification_outbox`, `background_jobs`
- Added a schema reference document at `examples/server/ai-chat-wizard/SCHEMA_TABLES.md` that lists each table, its columns, and its purpose.
- Validation:
  - `sqlite3 :memory: ".read examples/server/ai-chat-wizard/sql/store/schema.sql"`
  - replay-safe migration validation against a temp SQLite database
  - `go test ./examples/server/ai-chat-wizard/proto`

### example 100: authenticated token usage traceability for billing

- Added immutable usage-event persistence for `examples/server/ai-chat-wizard` so each logged-in completion can be audited for billing:
  - schema/migration updates in:
    - `examples/server/ai-chat-wizard/sql/store/schema.sql`
    - `examples/server/ai-chat-wizard/sql/store/migrations.sql`
  - new SQL queries:
    - `examples/server/ai-chat-wizard/sql/store/save_usage_event.sql`
    - `examples/server/ai-chat-wizard/sql/store/list_usage_events.sql`
    - `examples/server/ai-chat-wizard/sql/store/get_model_pricing.sql`
- Extended provider and transport usage metadata:
  - `provider.ChatResult` now carries `usage_source` and `provider_request_id` with provider implementations updated in:
    - `examples/server/ai-chat-wizard/server/provider/openai_provider.go`
    - `examples/server/ai-chat-wizard/server/provider/anthropic_provider.go`
    - `examples/server/ai-chat-wizard/server/provider/cerebras_provider.go`
    - `examples/server/ai-chat-wizard/server/provider/stub_provider.go`
  - final `ChatChunk` now includes:
    - `usage_event_id`
    - `provider_id`
    - `total_cost_usd`
    - `usage_source`
    - `usage_persisted`
  - proto changes in:
    - `examples/server/ai-chat-wizard/proto/chat.proto`
    - regenerated `chat.pb.go` and `chat_grpc.pb.go`
- `Send` server flow now snapshots event-time pricing and persists both:
  - successful completion usage events (`status=completed`)
  - failed-stream usage events (`status=failed`) for reconciliation parity
  - in `examples/server/ai-chat-wizard/server/app/server.go` and store plumbing in `store.go`/`queries.go`
- Added plan-aware auto model routing for authenticated sends:
  - revised billing SQL/table definitions with `billing_plan_model_access`
  - new effective-plan model query `list_billing_effective_model_access_by_user.sql`
  - `Send` now auto-selects the plan default model when request model is blank
  - model preference RPCs now apply effective plan model policy and persistence repair
- Added focused tests and micro-benches:
  - store usage lifecycle/scoping/pricing tests in `examples/server/ai-chat-wizard/server/app/store_test.go`
  - success + error stream usage assertions in:
    - `examples/server/ai-chat-wizard/server/app/rpc_test.go`
    - `examples/server/ai-chat-wizard/server/app/rpc_additional_test.go`
  - provider usage metadata assertions in:
    - `examples/server/ai-chat-wizard/server/provider/stub_provider_test.go`
    - `examples/server/ai-chat-wizard/server/provider/provider_streaming_additional_test.go`
  - usage micro-benches in `examples/server/ai-chat-wizard/server/app/benchmark_test.go`
- Validation:
  - `go test ./examples/server/ai-chat-wizard/server/app`
  - `go test ./examples/server/ai-chat-wizard/server/provider ./examples/server/ai-chat-wizard/proto`
  - `go test ./examples/server/ai-chat-wizard/server/app -run ^$ -bench "BenchmarkStoreCorePaths/(save_usage_event|list_usage_events)$" -benchtime=1x`
  - `go run ./tools/gwc test -lane unit -app .\examples\server\ai-chat-wizard\cmd\server\main.go -root .\examples\server\ai-chat-wizard`

### examples and tooling cleanup: example 203 + three-digit catalog support

- Added the new `examples/203-use-state-rerender-trace` state example, including:
  - `examples/203-use-state-rerender-trace/main.go`
  - `examples/203-use-state-rerender-trace/use-state-rerender-trace.html`
  - docs updates in `examples/README.md` and `examples/MANUAL_TESTING.md`
- Updated example discovery in `tools/gwc/examples.go` from `^\d{2}-` to `^\d+-` so three-digit example folders are included, and added focused coverage in `tools/gwc/examples_test.go`.
- Tightened `gwc` livereload client-script resolution wiring in `tools/gwc/dev.go` and `tools/gwc/main.go`, with supporting test updates.
- Added non-js/wasm build constraints for static export helpers in `examples/102-static-export-site/site.go` and `examples/102-static-export-site/export_test.go`.
- Included cleanup and dead-code removals across examples, interop, testkit, and wasm/native helper packages to keep package builds warning-free and reduce unused wrappers.
- Validation:
  - `go test ./tools/gwc -run "Test(BuildExamplesListingIncludesThreeDigitExamples|ResolveLauncherLivereloadClientScriptUsesOverride|ResolveLauncherLivereloadHelpersCoverDefaultsAndErrors)" -count=1`
  - `go test ./examples/102-static-export-site -run Test -count=1`

### runtime2 dispatch pressure pass: fast-hash streaming + coordinator read collapse

- Applied one focused dispatch hot-path optimization set in:
  - `internal/runtime2/snapshot_dispatch_hash.go`
  - `internal/runtime2/host_region_adapter.go`
  - `internal/runtime2/coordinator.go`
- Runtime changes:
  - `buildSnapshotDispatchFastHashIntoWithSourceAndPropsKeys(...)` now streams canonical envelope/props/source encoding directly into one pooled `maphash.Hash` (`writeSnapshotDispatchFastHashEnvelopeWithSourceAndPropsKeys(...)`) instead of staging a single combined payload byte slice before hashing.
  - Added `Coordinator.GetEntrySnapshotAndDispatchFields(...)` so snapshot capture can read epoch/renderer/source IDs plus fallback/monotonic dispatch fields in one `RLock` round.
  - `handleHostRegionUpdateSnapshot(...)` now returns those dispatch validation fields, and `handleHostRegionUpdateDispatchWithKnownPriority(...)` reuses them instead of issuing a second coordinator read.
- Validation:
  - `go test ./internal/runtime2 -run "Test(BuildSnapshotDispatchFastHashIntoWithSourceIDsMatchesBuffered|BuildSnapshotDispatchFastHashIntoWithSourceAndPropsKeysMatchesBuffered|HandleHostRegionDispatchHash.*|HandleHostRegionUpdateDispatch(ShortCircuitsNoChange|NoChangeDoesNotAdvanceDispatchedVersion))$" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "Benchmark(BuildSnapshotDispatchFastHashCurrentVsLegacy|HandleHostRegionDispatchHashCurrentVsLegacy|HandleHostRegionSnapshotHashPrefilterCurrentVsLegacy|HandleHostRegionManyHotRegionsBoundedWorkers)$" -benchmem -count=3`
  - `go test -c -o ./bin/runtime2.test.exe ./internal/runtime2`
  - `./bin/runtime2.test.exe --% -test.run=^$ -test.bench=BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$ -test.benchtime=10s -test.cpuprofile=./bin/runtime2.hot.cpu.pprof`
  - `go tool pprof -top ./bin/runtime2.hot.cpu.pprof`
- Benchmark/profile snapshot (Windows/amd64, i7-12700):
  - `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers`: `~12.3-12.6 us/op`, `383-384 B/op`, `47 allocs/op`
  - latest pprof top no longer shows `buildSnapshotDispatchFastHash` or `maps.(*Iter).Next` as top nodes; `runtime.duffcopy` dropped to about `3.2% flat` in the latest 10s capture.

### runtime2 dispatch pressure pass: adapter coordinator cache + fast-hash write batching

- Applied one additional dispatch hot-path optimization set in:
  - `internal/runtime2/host_region_adapter.go`
  - `internal/runtime2/snapshot_dispatch_hash.go`
- Runtime changes:
  - Added adapter-local cached coordinator snapshot or dispatch fields (epoch, renderer, source IDs, last snapshot version, last dispatched version) and used them on the dispatch-driven snapshot-capture path (`handleHostRegionUpdateSnapshot(..., parseShouldStoreSnapshotVersion=false)`) so urgent dispatch avoids a separate coordinator `RLock` read on every update.
  - Wired cache updates through mount/dispose/remount and snapshot/dispatch write paths so monotonic snapshot and dispatch state stays in sync with coordinator writes.
  - Reduced fast-hash overhead by batching marker+length writes (`writeSnapshotDispatchFastHashByteAndUint64(...)`, `writeSnapshotDispatchFastHashByteAndByte(...)`) and using `maphash.Hash.WriteString(...)` for string payload writes.
- Validation:
  - `go test ./internal/runtime2 -run "Test(HandleHostRegionUpdateDispatch.*|HandleHostRegionUpdateSnapshot.*|HandleHostRegionWorkerDeath.*|HandleHostRegionRepairRemount.*|HandleHostRegionStructuralRemount.*)$" -count=1`
  - `go test ./internal/runtime2 -run "Test(BuildSnapshotDispatchFastHashIntoWithSourceIDsMatchesBuffered|BuildSnapshotDispatchFastHashIntoWithSourceAndPropsKeysMatchesBuffered|HandleHostRegionDispatchHash.*|HandleHostRegionUpdateDispatch.*)$" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "Benchmark(HandleHostRegionManyHotRegionsBoundedWorkers|HandleHostRegionDispatchHashCurrentVsLegacy|BuildSnapshotDispatchFastHashCurrentVsLegacy)$" -benchmem -count=3`
  - `go test -c -o ./bin/runtime2.test.exe ./internal/runtime2`
  - `./bin/runtime2.test.exe --% -test.run=^$ -test.bench=BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$ -test.benchtime=10s -test.cpuprofile=./bin/runtime2.hot.cpu.pprof`
  - `go tool pprof -top ./bin/runtime2.hot.cpu.pprof`
- Benchmark/profile snapshot (Windows/amd64, i7-12700):
  - `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers` improved from pre-pass `~17.9-21.2 us/op` to post-pass `~11.6-15.4 us/op` in the same compare command (`383 B/op`, `47 allocs/op` unchanged).
  - `BenchmarkHandleHostRegionDispatchHashCurrentVsLegacy/stable_payload_current_digest_guard` improved from roughly `~812-905 ns/op` to `~694-746 ns/op`.
  - `BenchmarkBuildSnapshotDispatchFastHashCurrentVsLegacy/current_streamed_maphash` stayed in-band (`~1.34-1.65 us/op` before vs `~1.44-1.52 us/op` after; no alloc change).
  - Recent pprof captures show lock/hash-pressure reductions on the hot path (`sync/atomic.(*Int32).Add` around `~2.7-3.0% flat`, `hash/maphash.(*Hash).Write` around `~4.2-5.3% flat`).
  - Note: `go test ./internal/runtime2` in this worktree is currently red due unrelated existing failures (`TestBuildSchedulerCanonicalizesShardIDsForLookup`, `TestDuplicateKeySiblingCanonicalBuildFails`), so this pass was validated with focused suites and microbenches.

### runtime2 binary source decode and source-id validation tightening

- Reviewed recent runtime2 commits and added missing changelog detail for:
  - `1b7efb5` (`tighten runtime2 binary source decode paths`)
  - `272fbd0` (`tighten runtime2 binary source id validation`)
- Runtime changes:
  - `internal/runtime2/binary_source_value.go`
    - split `ParseBinarySourceValue(...)` into scalar-first and composite decode paths (`parseBinarySourceScalarValue(...)` and `parseBinarySourceCompositeValue(...)`),
    - updated list/map decode loops to use scalar fast handling before recursive composite decode.
  - `internal/runtime2/binary_source_id_table.go`
    - tightened append/parse length handling with trusted append path and explicit capacity reservation,
    - tightened source-ID validation with an ASCII fast path plus UTF-8 fallback (`isBinarySourceIDByteValid(...)` + `utf8.DecodeRuneInString(...)`) so invalid bytes/runes fail deterministically.
- Validation:
  - `go test ./internal/runtime2 -run "Test(BuildBinarySourceIDTableRoundTripsCanonicalIDs|ParseBinarySourceIDTableRejectsNonCanonicalOrder|BuildBinarySourceValueRoundTripsScalars|BuildBinarySourceValueRoundTripsSmallList|BuildBinarySourceValueRoundTripsNestedMap|BuildBinarySourceValueRejectsUnsupportedKind)$" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "Benchmark(BinarySourceIDTableCurrentVsLegacy|ParseBinarySourceValueNestedCurrentVsLegacy)$" -benchmem -count=3`
- Benchmark snapshot (Windows/amd64, i7-12700):
  - source-id append: `legacy ~94-107 ns/op, 240 B/op, 4 allocs/op` -> `current ~60-65 ns/op, 128 B/op, 2 allocs/op`
  - source-id parse: parity (`legacy ~269-289 ns/op` vs `current ~275-288 ns/op`, both `240 B/op, 9 allocs/op`)
  - nested source-value parse: slight improvement (`legacy ~1641-1691 ns/op` vs `current ~1564-1634 ns/op`, both `2880 B/op, 58 allocs/op`)

### runtime2 binary source-id table capacity pass

- Completed the next runtime2 perf todo in `internal/runtime2/binary_source_id_table.go`:
  - `appendBinarySourceIDTableFromNormalized(...)` now computes exact encoded length (`getBinarySourceIDTableLengthFromNormalized(...)`) and reserves destination capacity once (`getBinarySourceIDTablePayloadWithCapacity(...)`) before writing.
  - Kept parse validation behavior intact while refreshing compare-bench coverage in `internal/runtime2/perf_binary_source_id_table_compare_bench_test.go`.
- Validation:
  - `go test ./internal/runtime2 -run "Test(BuildBinarySourceIDTableRoundTripsCanonicalIDs|ParseBinarySourceIDTableRejectsNonCanonicalOrder|BuildBinarySnapshotEnvelopeRoundTrips|ParseBinarySnapshotEnvelopeRejectsTruncatedPayload|ParseBinarySnapshotEnvelopeRejectsIncorrectLength)$" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "Benchmark(ParseBinarySourceIDTableCanonical|BinarySourceIDTableCurrentVsLegacy)$" -benchmem -count=3`
- Benchmark snapshot (Windows/amd64, i7-12700):
  - append: `legacy ~92.7-110.9 ns/op, 240 B/op, 4 allocs/op` -> `current ~62.5-75.5 ns/op, 128 B/op, 2 allocs/op`
  - parse: parity (`legacy ~271-307 ns/op` vs `current ~269-279 ns/op`, both `240 B/op, 9 allocs/op`)

### runtime2 dispatch fast-hash pass: maphash upgrade for no-change gating

- Replaced the byte-loop FNV implementation in `buildSnapshotDispatchFastHash(...)` with a process-seeded `maphash.Bytes(...)` path in `internal/runtime2/snapshot_dispatch_hash.go`.
- Kept the existing non-zero sentinel behavior (`0 -> 1`) and all call-site contracts intact.
- Updated benchmark/docs wording from `fnv` to generic `fast` prefilter terminology where the implementation now uses `maphash`.
- Updated `internal/runtime2/perf_snapshot_dispatch_fast_hash_compare_bench_test.go` so the legacy sub-bench preserves the previous buffered FNV path (`legacy_buffered_fnv1a`) while current uses streamed maphash (`current_streamed_maphash`).

Validation:
- `go test ./internal/runtime2 -run "TestHandleHostRegionDispatchHash" -count=1`
- `go test ./internal/runtime2 -run ^$ -bench "Benchmark(BuildSnapshotDispatchFastHashCurrentVsLegacy|HandleHostRegionDispatchHashCurrentVsLegacy|HandleHostRegionSnapshotHashPrefilterCurrentVsLegacy)$" -benchmem -count=3`
- `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostWorkerRegionUpdateOrchestrationCurrentVsLegacy$" -benchmem -count=3`

Benchmark snapshot (Windows/amd64, i7-12700):
- `BenchmarkBuildSnapshotDispatchFastHashCurrentVsLegacy/current_streamed_maphash`: `~933-1046 ns/op`, `104 B/op`, `3 allocs/op`
- `BenchmarkBuildSnapshotDispatchFastHashCurrentVsLegacy/legacy_buffered_fnv1a`: `~1422-1681 ns/op`, `104 B/op`, `3 allocs/op`
- `BenchmarkHandleHostRegionDispatchHashCurrentVsLegacy/stable_payload_current_digest_guard`: `~381-392 ns/op`, `0 B/op`, `0 allocs/op`
- `BenchmarkHandleHostRegionSnapshotHashPrefilterCurrentVsLegacy/no_change_current_fast_prefilter`: `~245-257 ns/op`, `0 B/op`, `0 allocs/op`

### render-benchmark example 201 MT worker latency improvements

Five latency improvements implemented across `examples/testing/render-benchmark/workers.go`, `examples/testing/render-benchmark/backgroundworker/main.go`, and `examples/testing/render-benchmark/shared/shared.go`.

- **Progressive per-lane chunk delivery** — replaced the all-or-nothing `WaitGroup` barrier with per-lane partial delivery. Added `onPartialChunks func([]ChunkResult)` callback parameter to all six fan-out functions (`requestBenchmarkWorkerCoreChunks`, `requestBenchmarkWorkerContentChunks`, and both `ByBatch`/`ByChunk` variants). Each lane goroutine emits a `make+copy` snapshot and calls the callback immediately on completion. `handleBenchmarkWorkerPrepareEffect` wires `onPartialCoreLaneDone` / `onPartialContentLaneDone` closures that call `parseCoreChunkState.Set(...)` / `parseContentChunkState.Set(...)` per-lane under a stale-generation guard.

- **Per-item generation early abort in background worker** — added two package-level `atomic.Uint64` counters (`getBenchmarkWorkerCoreLatestGeneration`, `getBenchmarkWorkerContentLatestGeneration`) to `backgroundworker/main.go`. Counters are written in `shouldBenchmarkWorkerDropStaleRequest` when a newer generation arrives and read every 8 items in `buildBenchmarkWorkerCoreChunkWithCache` / `buildBenchmarkWorkerContentChunkWithCache`. Also checks generation between chunks inside `buildBenchmarkWorkerCoreBatchWithCache` / `buildBenchmarkWorkerContentBatchWithCache`. Returns `HasStale: true` on abort; batch builders propagate stale early exits.

- **Data-driven adaptive chunk count** — `buildBenchmarkWorkerAutoChunkCount` now reads `parseWorkerState.GetLastBatchMS` instead of ignoring it. Chunk target scales to `workerCount×1` when last batch < 3 ms (reduce IPC overhead), `workerCount×2` at 3–8 ms (original heuristic), and `workerCount×4` when > 8 ms (maximise progressive fanout against slow workers).

- **Stronger dependency hash** — both `buildBenchmarkWorkerCoreItemsDependency` and `buildBenchmarkWorkerContentItemsDependency` replaced 4-point sample loops with a full pass over every item, reusing the existing `buildBenchmarkWorkerCoreItemDependency` / `buildBenchmarkWorkerContentItemDependency` per-item mixers. Both `hasBenchmarkWorkerCoreChunkCacheMatch` / `hasBenchmarkWorkerContentChunkCacheMatch` O(n) item-scan guards simplified to item-count-only checks; string comparisons on the main JS-thread hot path eliminated.

- **Dirty-set delta sends** — added `GetDirtyItemIndexes []int` field (omitempty) to `BenchmarkWorkerCoreBatchChunkRequest` and `BenchmarkWorkerContentBatchChunkRequest` in `shared/shared.go`. Three new helpers: `buildBenchmarkWorkerCoreItemDirtyIndexes`, `buildBenchmarkWorkerContentItemDirtyIndexes`, `buildBenchmarkWorkerChunkLocalDirtyIndexes`. Client computes dirty indexes by comparing previous/current equal-length item slices and passes chunk-local slices into batch requests. Worker adds per-chunk result caches (`getBenchmarkWorkerCoreChunkResultByIndex`, `getBenchmarkWorkerContentChunkResultByIndex`) and a dirty-set fast path: when a previous chunk result and dirty indexes are both available, only dirty items are recomputed and XOR-merged into the previous result; clean items are copied without any cache-key allocation.

- Benchmarks (Windows/amd64, i7-12700, `examples/testing/render-benchmark/shared`):
  - `SnapshotCopyNilCallback`: `0.12 ns/op, 0 allocs` — nil-guard is zero-cost
  - `SnapshotCopy4Chunks`: `~64 ns/op, 1 alloc (320 B)`
  - `SnapshotCopy8Chunks`: `~118–137 ns/op, 1 alloc (704 B)`
  - `SnapshotCopy16Chunks`: `~311–334 ns/op, 1 alloc (1408 B)`
  - `BuildCoreChunkResult40Items`: `~1.25 ms/op` — snapshot overhead <0.01% of lane work
  - `BuildCoreChunkResult240Items`: `~8 ms/op` — snapshot overhead <0.005%

### runtime2 structured-clone snapshot transport header-only fast paths

- Completed the next runtime2 transport todo in `internal/runtime2/structured_clone_snapshot_transport.go`:
  - `BuildStructuredCloneSnapshotEnvelopeJSON(...)` now takes a direct header-only append path when both `props` and `sources` are absent.
  - `ParseStructuredCloneSnapshotEnvelopeJSON(...)` now takes a header-only fast parser when payloads omit `props`/`sources`, while preserving full `json.Unmarshal` fallback for rich payloads.
- Added compare benchmark coverage:
  - `internal/runtime2/perf_structured_clone_snapshot_transport_compare_bench_test.go`
  - `BenchmarkStructuredCloneSnapshotEnvelopeCurrentVsLegacy` (build-rich, build-header-only, parse-rich, parse-header-only).
- Validation:
  - `go test ./internal/runtime2 -run "Test(BuildStructuredCloneSnapshotEnvelopeJSONRoundTrips|ParseStructuredCloneSnapshotEnvelopeJSONFailsMalformedEnvelope|ParseStructuredCloneSnapshotEnvelopeJSONDefaultsOptionalFields|ParseStructuredCloneSnapshotEnvelopeJSONRejectsMalformedJSON|ParseStructuredCloneSnapshotEnvelopeJSONRejectsNonObjectPayload|SelectSnapshotTransportTierPrefersBinary|BuildSnapshotTransportPayloadWithFallbackDowngradesOnBinaryEncodeFailure|ParseSnapshotTransportPayloadWithFallbackDowngradesOnBinaryDecodeFailure)" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildStructuredCloneSnapshotEnvelopeJSON$" -benchmem -count=5`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkStructuredCloneSnapshotEnvelopeCurrentVsLegacy$" -benchmem -count=3`
- Benchmark snapshot (Windows/amd64, i7-12700):
  - header-only build: `legacy ~182-208 ns/op, 128 B/op, 2 allocs/op` -> `current ~86-96 ns/op, 112 B/op, 1 alloc/op`
  - header-only parse: `legacy ~669-944 ns/op, 288 B/op, 6 allocs/op` -> `current ~82-89 ns/op, 8 B/op, 1 alloc/op`
  - rich paths stayed in-band:
    - build: both around `~1.0-1.15 us/op`, `624 B/op`, `14 allocs/op`
    - parse: current around `~2.40-2.55 us/op` with matching `1152 B/op`, `28 allocs/op`.

### runtime2 host patch fallback parse benchmark coverage extension

- Extended fallback parse benchmark coverage in `internal/runtime2/perf_host_patch_transport_parse_compare_bench_test.go` by adding `shared_buffer_fallback_structured` current-vs-legacy sub-bench (shared tier selected, no shared page, structured message payload fallback path).
- Validation:
  - `go test ./internal/runtime2 -run "TestParseHostPatchPayloadWithFallback" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseHostPatchPayloadWithFallbackCurrentVsLegacy$" -benchmem -count=5`
  - `go test ./internal/runtime2 -run ^$ -bench "Benchmark(ParseBinaryPatchPayload|GetSharedPatchReadPayload)$" -benchmem -count=5`
- Benchmark snapshot:
  - `shared_buffer_fallback_structured` kept lower current-path allocation pressure (`legacy 952 B/op, 19 allocs/op` vs `current 888 B/op, 17 allocs/op`) with noisy ns/op on this host.

### runtime2 worker render-input source normalization and cache reuse follow-up

- Extended the `BuildWorkerRenderInput(...)` optimization pass in `internal/runtime2/worker_render_input_adapter.go`:
  - replaced the mismatch normalization chain (`map keys -> NormalizeSourceIDs(...) -> clone`) with a single validated source-key extraction+sort pass (`buildWorkerRenderNormalizedSourceIDsFromSnapshot(...)`),
  - removed the redundant post-normalization cached-source-ID clone on mismatch and now return normalized IDs directly as the next cache order.
- Added focused behavior coverage:
  - `internal/runtime2/worker_render_input_adapter_test.go` with `TestBuildWorkerRenderInputRejectsInvalidSourceID`.
- Revised source-order compare benchmark to pin old/new behavior explicitly:
  - `internal/runtime2/perf_worker_render_input_source_order_compare_bench_test.go`.
- Validation:
  - `go test ./internal/runtime2 -run "Test(BuildWorkerRenderInputWithSourceOrderReusesCachedSourceIDs|BuildWorkerRenderInputWithSourceOrderRebuildsOnSourceSetChange|BuildWorkerRenderInputRejectsInvalidSourceID|HandleWorkerRegion|BuildWorkerRenderInput|BuildSnapshotEnvelope)" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildWorkerRenderInputSourceOrderCurrentVsLegacy$" -benchmem -count=5`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleWorkerRegionUpdateSnapshotDrivenVsMetadataOnly$" -benchmem -count=5`
- Benchmark snapshot (Windows/amd64, i7-12700):
  - source-order rebuild path: `legacy ~522-639 ns/op, 640 B/op, 4 allocs/op` -> `current ~438-497 ns/op, 384 B/op, 2 allocs/op`,
  - source-order cached path: current/legacy both remain in-band at `~242-300 ns/op, 256 B/op, 1 alloc/op`,
  - worker update benchmark on this host moved from pre-change `snapshot-driven ~3947-6056 ns/op, 5846-5847 B/op, 41 allocs/op` and `metadata-only ~4912-6430 ns/op, 5486 B/op, 37 allocs/op` to post-change `snapshot-driven ~2337-3310 ns/op, 3015 B/op, 25 allocs/op` and `metadata-only ~2319-2972 ns/op, 2654-2655 B/op, 21 allocs/op`.

### runtime2 snapshot props shape-fingerprint cache in host update capture

- Completed the next runtime2 perf todo by adding a second props-cache tier in `HandleHostRegionUpdateSnapshot(...)`:
  - tier 1: existing immutable scalar token cache,
  - tier 2: new flat `map[string]any` shape cache using key-count plus sorted-key hash and per-key scalar type markers.
- Runtime changes:
  - `internal/runtime2/host_region_adapter.go`
    - stores cached sorted keys and type markers for flat props,
    - hot-path cache hit checks use O(n) map lookups against cached sorted keys/type markers (no per-hit key sorting),
    - preserves full `ValidateSerializableProps(...)` fallback for non-flat or nested props.
  - `internal/runtime2/spec.go`
    - added shape-fingerprint builders and matcher helpers for flat scalar `map[string]any` props.
- Coverage additions:
  - `internal/runtime2/spec_shape_fingerprint_test.go`
  - `internal/runtime2/host_region_snapshot_capture_test.go` (`TestHandleHostRegionUpdateSnapshotPropsCacheInvalidatesOnTypeFlip`)
  - `internal/runtime2/perf_host_region_props_shape_fingerprint_compare_bench_test.go`
- Validation:
  - `go test internal/runtime2/registry.go internal/runtime2/spec.go internal/runtime2/spec_shape_fingerprint_test.go -run "TestBuildSerializablePropsFlatShapeFingerprint" -count=1`
  - `go test internal/runtime2/host_region_snapshot_capture_test.go -run "TestHandleHostRegionUpdateSnapshot(PropsCacheInvalidatesOnInPlaceMutation|PropsCacheInvalidatesOnTypeFlip|CapturesPropsAndSources|ReusesSourceMapForStableVersionTuple|InvalidatesSourceMapReuseOnVersionChange|InvalidatesSourceMapReuseOnRemountEpoch)" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionUpdateSnapshot(CurrentVsLegacy|PropsShapeFingerprintCurrentVsLegacy)$" -benchmem -count=5`
  - `go test internal/runtime2/host_region_pressure_bench_test.go -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
- Benchmark snapshot (Windows/amd64, i7-12700):
  - wide flat stable props:
    - current: `~525.9-750.1 ns/op`, `592 B/op`, `4 allocs/op`
    - legacy: `~1345-1550 ns/op`, `1008 B/op`, `11 allocs/op`
  - wide flat changed props (same key/type shape):
    - current: `~574.5-689.5 ns/op`, `600 B/op`, `4 allocs/op`
    - legacy: `~1258-1893 ns/op`, `1016 B/op`, `11 allocs/op`
  - pressure benchmark remained in broad historical range with unchanged allocation profile:
    - `~16.2-24.6 us/op`, `382-383 B/op`, `47 allocs/op`.

### runtime2 source map ordered-key dispatch hashing todo closure

- Closed the next runtime2 todo after verifying the source key-order optimization path is already active:
  - `handleHostRegionDispatchHash(...)` passes canonical declared source IDs into `appendSnapshotDispatchEnvelopeWithSourceIDs(...)`,
  - source hashing uses `appendSnapshotDispatchSourceMap(...)` + `appendSnapshotDispatchAnyMapWithOrderedKeys(...)`,
  - when ordered source IDs fully cover the source map, dispatch hashing skips per-update generic map key extraction/sorting.
- Validation:
  - `go test internal/runtime2/host_region_update_dispatch_test.go -run "TestHandleHostRegionUpdateDispatch" -count=1`
  - `go test internal/runtime2/host_region_pressure_bench_test.go -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
- Result:
  - ordered source-ID hash path confirmed active in code and tests,
  - pressure benchmark remained in expected band with unchanged allocation profile (`~16.2-24.6 us/op`, `382-383 B/op`, `47 allocs/op`).

### runtime2 dispatch coordinator-write batching todo closure

- Closed the coordinator-write batching todo after verifying dispatch branches already use single coordinator write transactions:
  - urgent changed dispatch path uses `StoreRegionSnapshotDispatchState(...)` for snapshot+dispatch version persistence in one lock-held write,
  - no-change and deferred paths each use `SetRegionSnapshotState(...)` as one transaction write.
- Validation:
  - `go test internal/runtime2/host_region_update_dispatch_test.go -run "TestHandleHostRegionUpdateDispatch" -count=1`
  - `go test internal/runtime2/host_region_pressure_bench_test.go -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
- Result:
  - no split snapshot/dispatched write helper pattern remains in dispatch branches,
  - pressure benchmark allocation profile remained unchanged (`382-383 B/op`, `47 allocs/op`).

### runtime2 allocation-reporting benchmark todo closure

- Closed the `b.ReportAllocs()` benchmark todo after verifying both primary pressure benches already report allocations:
  - `internal/runtime2/host_region_pressure_bench_test.go` (`BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers`)
  - `internal/runtime2/agent3_bench_test.go` (`BenchmarkHandleHostRegionUpdateDispatchLoop`)
- Validation:
  - `rg -n "ReportAllocs\\(" internal/runtime2/host_region_pressure_bench_test.go internal/runtime2/agent3_bench_test.go`
  - `go test internal/runtime2/host_region_pressure_bench_test.go -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
- Result:
  - allocation reporting is active and current pressure runs keep reporting `383 B/op`, `47 allocs/op`.

### runtime2 capability report lock-free reads and one-time detection memoization

- Completed the capabilities memoization todo in `internal/runtime2/capabilities.go`:
  - replaced lock-based capability reads with lock-free atomic reads (`atomic.Bool` + `atomic.Value`),
  - memoized runtime capability detection once per process via `sync.Once` in `InitCapabilityReportFromRuntime()`.
- Added focused microbenchmark:
  - `internal/runtime2/capabilities_bench_test.go` with `BenchmarkGetCapabilityReport`.
- Validation:
  - `go test internal/runtime2/capabilities_test.go -run "Test(GetCapabilityReportDefaultsWithoutWorkerSupport|InitCapabilityReportStoresDetectedCapabilities|InitCapabilityReportFromRuntimeUsesDetectedSource)" -count=1`
  - `go test internal/runtime2/capabilities_bench_test.go -run ^$ -bench "BenchmarkGetCapabilityReport$" -benchmem -count=5`
- Benchmark snapshot (Windows/amd64, i7-12700):
  - `initialized-lock-free-read`: `~2.063-2.152 ns/op`, `0 B/op`, `0 allocs/op`
  - `uninitialized-default-read`: `~1.644-1.987 ns/op`, `0 B/op`, `0 allocs/op`.

### runtime2 coordinator trusted-ID mutation path for mounted host regions

- Completed the next runtime2 perf todo by splitting coordinator mutable-entry reads into validated and trusted variants.
- Added trusted coordinator mutation helpers in `internal/runtime2/coordinator.go` for hot mounted-region operations:
  - `handleCoordinatorCommitRegionTrusted(...)`
  - `handleCoordinatorFallbackRegionTrusted(...)`
  - `handleCoordinatorRestartRegionTrusted(...)`
  - `handleCoordinatorSetRegionAttachedTrusted(...)`
  - trusted stale counter increment helpers
- Updated host hot paths to use trusted helpers when the region ID is already mount-validated:
  - `internal/runtime2/host_region_adapter.go`
  - `internal/runtime2/host_control_dispatcher.go`
- Kept external coordinator API behavior unchanged by preserving `ParseRegionInstanceID(...)` checks in exported methods.
- Added focused compare benches in `internal/runtime2/perf_coordinator_get_update_compare_bench_test.go`:
  - `BenchmarkCoordinatorGetMutableEntryValidatedVsTrusted`
  - `BenchmarkCoordinatorCommitRegionValidatedVsTrusted`
- Validation:
  - `go test ./internal/runtime2 -run "Test(SetRegionSnapshotState|SetRegionAttachedStoresAttachedState|HandleHostRegionMount|HandleHostRegionUpdateSnapshot)" -count=1`
  - `go test ./internal/runtime2 -run "Test(CommitRegion|FallbackRegion|RestartRegion|SetRegionAttached|IncrementRegionDroppedStalePatchCount|IncrementRegionIgnoredStaleDiagnosticCount|IncrementRegionRepairRemountCount|HandleHostControlEnvelopeDispatchesPatchReady|HandleHostControlEnvelopeDispatchesDiagnostic|HandleHostControlEnvelopeDispatchesRestart|HandleHostRegionPatchReadyStaleBeforeRepairFloorIncrementsDroppedCounter|HandleHostRegionWorkerOutputOlderVersionIncrementsDroppedCounter)" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkCoordinator(GetMutableEntryValidatedVsTrusted|CommitRegionValidatedVsTrusted)$" -benchmem -count=5`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
- Benchmark snapshot (Windows/amd64, i7-12700):
  - mutable entry read: validated `~33.8-40.3 ns/op` vs trusted `~25.7-29.2 ns/op` (`0 allocs` both),
  - commit write: validated `~77.1-95.3 ns/op` vs trusted `~69.9-82.1 ns/op` (`0 allocs` both),
  - pressure bench remained in-band at `~15.9-17.6 us/op`, `383 B/op`, `47 allocs/op`.

### runtime2 props layout evaluation: map iteration vs sorted key/value slice scan

- Closed the next runtime2 todo as an explicit evaluation pass before changing snapshot envelope semantics.
- Added `internal/runtime2/perf_snapshot_dispatch_prop_layout_compare_bench_test.go` with:
  - `BenchmarkSnapshotDispatchPropLayoutCurrentVsEntries`
  - current path: `appendSnapshotDispatchAnyMap(...)` (map iterate + sort each call)
  - candidate path: pre-sorted `[]buildSnapshotDispatchMapEntry` sequential scan (`appendSnapshotDispatchAnyMapEntries(...)`)
- Validation:
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkSnapshotDispatchPropLayoutCurrentVsEntries$" -benchmem -count=5`
- Benchmark snapshot (Windows/amd64, i7-12700):
  - current map path: `~679.6-699.4 ns/op`, `104 B/op`, `3 allocs/op`
  - candidate slice scan: `~49.8-63.8 ns/op`, `0 B/op`, `0 allocs/op`
- Outcome:
  - sequential key/value slices show clear hot-path potential for steady prop shapes,
  - runtime integration is deferred to a follow-up safe-cache pass because existing decode and test paths still intentionally treat `SnapshotEnvelope.Props` as `map[string]any`.

### runtime2 snapshot hash FNV-64a prefilter, dispatch hash FNV-64a, and priority-skip refactor

Three hot-path optimizations applied to `internal/runtime2/host_region_adapter.go`.

**1. Snapshot hash FNV-64a prefilter (`handleHostRegionSnapshotHash`)**
- Added two new struct fields: `storeHostRegionSnapshotFastHash uint64` and `storeHostRegionSnapshotHashScratch []byte`.
- On each call, `appendSnapshotDispatchEnvelope` now produces canonical bytes (InputVersion normalized to 1), then `buildSnapshotDispatchFastHash` (FNV-64a) runs over the result.
- If FNV-64a matches the stored fast hash, SHA-256 is skipped entirely and the stored digest is returned directly.
- All four state-reset paths (mount, dispose, structural remount, repair remount) now also clear the new fast-hash fields.

**2. Dispatch hash FNV-64a (`handleHostRegionDispatchHash`)**
- Replaced the `sha256.Sum256` in the payload-hash path with `buildSnapshotDispatchFastHash` (FNV-64a).
- SHA-256 struct fields (`storeHostRegionDispatchHash`, `storeHostRegionDispatchBytes`) are now cleared on every hash computation, removing stale state.
- The version-vector gate still exits early (no hashing) on source-version mismatch; the FNV-64a path runs only when the version vector matches and a payload check is needed.

**3. Dispatch priority skip (`HandleHostRegionUpdateDispatch`)**
- Added unexported `handleHostRegionUpdateDispatchWithKnownPriority` containing the full dispatch body.
- `HandleHostRegionUpdateDispatch` now routes directly to the inner function with `HostRegionDispatchPriorityUrgent`, skipping the `ParseHostRegionDispatchPriority` call on every urgent invocation.
- `HandleHostRegionUpdateDispatchWithTransition` also routes directly.
- `HandleHostRegionUpdateDispatchWithPriority` still validates the externally-supplied priority before delegating.

**New microbenchmark**
- Added `BenchmarkHandleHostRegionSnapshotHashPrefilterCurrentVsLegacy` in
  `internal/runtime2/perf_host_region_snapshot_hash_prefilter_compare_bench_test.go`.
- Covers no-change and changed sub-cases against a self-contained legacy state struct that always runs SHA-256.

**Validation**
- `go test ./internal/runtime2 -run "TestHandleHostRegionSnapshotFingerprint|TestHandleHostRegionUpdateDispatch|TestHandleHostRegionDispatchHash|TestHandleHostRegionUpdateSnapshot" -count=1`
- `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionSnapshotHashPrefilterCurrentVsLegacy|BenchmarkHandleHostRegionSnapshotFingerprint|BenchmarkHandleHostRegionDispatchHashCurrentVsLegacy|BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred|BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers" -benchmem -count=5`

**Benchmark snapshot (Windows/amd64, i7-12700)**

Snapshot fingerprint no-change hot path (primary target):
- `BenchmarkHandleHostRegionSnapshotFingerprint/no-change`: `~746-1000 ns/op`, `456 B/op`, `8 allocs/op` → `~189-258 ns/op`, `0 B/op`, `0 allocs/op` **(~3-4x speedup, zero allocs)**

Snapshot hash prefilter compare (new benchmark):
- `no_change_legacy_sha_always`: `~1602-1983 ns/op`, `1906 B/op`, `20 allocs/op`
- `no_change_current_fnv_prefilter`: `~349-409 ns/op`, `0 B/op`, `0 allocs/op` **(~4-5x speedup, zero allocs)**
- `changed_legacy_sha_always`: `~2491-3691 ns/op`, `3597 B/op`, `31 allocs/op`
- `changed_current_fnv_prefilter`: `~2727-4058 ns/op`, `2588 B/op`, `25 allocs/op` (comparable latency, 6 fewer allocs)

Deferred dispatch no-change path:
- `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/no-change`: `~371-430 ns/op`, `0 allocs` → `~278-320 ns/op`, `0 allocs` **(~14-25% speedup)**
- Follow-up re-run confirms: `no-change ~275-297 ns/op`, `changed ~472-632 ns/op`, `changed-reused-spec ~379-431 ns/op`, all `0 allocs/op` on no-change and reused-spec paths.

Dispatch hash compare:
- `changed_payloads_current_version_vector_gate`: `~12-16 ns/op`, `0 allocs` (version-vector source-version mismatch exits fast; unchanged)
- `stable_payload_current_digest_guard`: `~897-1061 ns/op`, `104 B/op`, `3 allocs` (FNV-64a digest comparison — previous 5 ns was from an InputVersion fast-exit overridden by correctness test)

Pressure benchmark:
- `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers`: `~16724-20561 ns/op`, `382 B/op`, `47 allocs/op` (within prior noise band)

### runtime2 coordinator pointer-entry todo closure

- Closed the coordinator-map layout todo after verifying pointer-entry storage is already active in `internal/runtime2/coordinator.go` (`storeEntries map[RegionInstanceID]*CoordinatorEntry`).
- Validation:
  - `go test ./internal/runtime2 -run "Test(SetRegionSnapshotState|UpdateRegionAndGetEntry|StoreRegionSnapshotAndDispatchedVersion|HandleHostRegionUpdateSnapshot)" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkCoordinatorDispatchTransactionCurrentVsLegacy$" -benchmem -count=5`
- Microbench snapshot:
  - `current_single_snapshot_and_dispatch`: `~40.96-44.42 ns/op`
  - `legacy_split_snapshot_and_dispatch`: `~64.34-81.14 ns/op`
  - both paths remained `0 B/op`, `0 allocs/op`.

### runtime2 snapshot hash prefilter todo closure

- Closed the pending snapshot-hash prefilter todo after verifying the FNV-64a + SHA path is already implemented in `handleHostRegionSnapshotHash(...)`.
- Validation run:
  - `go test ./internal/runtime2 -run "TestHandleHostRegionUpdateSnapshot" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegion(UpdateSnapshot|ManyHotRegionsBoundedWorkers)$" -benchmem -count=5`
- Current benchmark baseline from this validation:
  - `BenchmarkHandleHostRegionUpdateSnapshot`: `~316-428 ns/op`, `592 B/op`, `4 allocs/op`
  - `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers`: `~16.8-21.6 us/op` (with emitted p50/p95/p99 metrics).

### runtime2 serializable-props fast path: typed scalar containers

- Completed the next runtime2 perf todo in `internal/runtime2/spec.go` by keeping common typed scalar containers on the non-reflective path:
  - added direct fast exits for typed scalar slices like `[]int` and `[]string`,
  - added typed `map[string]scalar` validation with the existing ref and DOM-marker key guards,
  - skipped redundant recursive checks for primitive leaves inside `[]any` and `map[string]any`.
- Added focused coverage in:
  - `internal/runtime2/spec_test.go`
  - `internal/runtime2/perf_serializable_props_compare_bench_test.go`
- Validation:
  - `go test ./internal/runtime2 -run "Test(HandleHostRegionUpdateSnapshotPropsCacheInvalidatesOnInPlaceMutation|ValidateSerializablePropsAcceptsTypedScalarContainers|ValidateSerializablePropsRejectsTypedScalarMapRefMarker)" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkValidateSerializablePropsCurrentVsLegacy$" -benchmem -count=3`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=5`
- Benchmark snapshot:
  - typed string map: `~269.5-313.3 ns/op`, `128 B/op`, `8 allocs/op` legacy -> `~69.7-74.0 ns/op`, `0 B/op`, `0 allocs/op` current,
  - typed int slice: `~58.5-88.7 ns/op`, `24 B/op`, `1 alloc/op` legacy -> `~18.5-24.0 ns/op`, `24 B/op`, `1 alloc/op` current,
  - widened host pressure benchmark stayed in the expected `~17.5-21.0 us/op`, `382 B/op`, `47 allocs/op` band.

### runtime2 dispatch no-change: enforced two-tier vector + fast-hash check

- Completed the next runtime2 perf todo for dispatch no-change checks by enforcing two tiers in `handleHostRegionDispatchHash(...)`:
  - tier 1: version-vector mismatch still short-circuits as changed,
  - tier 2: exact version-vector matches now run canonical payload fast-hash comparison before returning no-change.
- Added focused regression coverage in `internal/runtime2/snapshot_dispatch_hash_test.go` proving payload mutations with unchanged version-vector fields are treated as changed.
- Validation:
  - `go test ./internal/runtime2 -run "TestHandleHostRegionUpdateDispatch" -count=1`
  - `go test ./internal/runtime2 -run "TestHandleHostRegionDispatchHash.*" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "Benchmark(HandleHostRegionDispatchHashCurrentVsLegacy|HandleHostRegionManyHotRegionsBoundedWorkers)$" -benchmem -count=5`
- Benchmark snapshot:
  - changed-payload current path stayed low-latency (`~11.0-12.9 ns/op`, `0 allocs/op`),
  - exact-vector stable repeats now pay fast-hash verification (`~1.0-1.3 us/op`, `104 B/op`, `3 allocs/op`) versus prior direct vector short-circuit (`~4.7 ns/op`),
  - pressure benchmark stayed in broad prior range (`~17.2-26.4 us/op`).

### runtime2 pressure profiling: unified cpu/mutex/block/mem capture

- Completed the runtime2 profiling-workflow TODO for pressure benchmarks by standardizing one run that captures all key profile artifacts together.
- Validation flow used:
  - `go test -c -o ./bin/runtime2.test.exe ./internal/runtime2`
  - `./bin/runtime2.test.exe -test.run=^$ -test.bench=BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$ -test.benchtime=10s -test.cpuprofile=./bin/runtime2.hot.cpu.pprof -test.mutexprofile=./bin/runtime2.hot.mutex.pprof -test.blockprofile=./bin/runtime2.hot.block.pprof -test.memprofile=./bin/runtime2.hot.mem.pprof`
- Artifacts confirmed from one invocation:
  - `bin/runtime2.hot.cpu.pprof`
  - `bin/runtime2.hot.mutex.pprof`
  - `bin/runtime2.hot.block.pprof`
  - `bin/runtime2.hot.mem.pprof`
- Note: profile-enabled runs intentionally distort benchmark means (`ns/op`, `B/op`, `allocs/op`) and are diagnostics-only, not regression baselines.

### runtime2 benchmark telemetry: p50/p95/p99 tail metrics

- Completed the runtime2 benchmarking TODO for tail-latency sampling in pressure and update-dispatch loops:
  - added p50/p95/p99 sample reporting to `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers` (`internal/runtime2/host_region_pressure_bench_test.go`),
  - added a focused update-dispatch loop benchmark with p50/p95/p99 metrics in `agent3_bench_test.go` (`BenchmarkHandleHostRegionUpdateDispatchLoop`).
- Implementation notes:
  - benchmarks now collect span-based latency samples during hot loops and emit percentile metrics via `b.ReportMetric(...)`,
  - span sampling avoids Windows timer-resolution collapse on very short per-iteration timings.
- Validation commands:
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers$" -benchmem -count=20`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionUpdateDispatchLoop$" -benchmem -count=5`
- Benchmark snapshots (Windows/amd64, i7-12700):
  - pressure benchmark now emits `dispatch-batch-p50/p95/p99` around `~23.5-23.8 us`, `~31.2-47.4 us`, `~39.3-48.3 us` respectively, while mean stayed near prior range (`~19.6-25.7 us/op` before instrumentation pass vs `~18.8-25.0 us/op` after),
  - new agent3 update-dispatch loop baseline: `p50 ~740-749 ns`, `p95 ~1109-1207 ns`, `p99 ~1126-1379 ns` with mean `~788-874 ns/op`.

### runtime2 dispatch hash: version-vector no-change gate

- Completed the runtime2 dispatch-hash TODO for a pre-hash version-vector gate in `handleHostRegionDispatchHash(...)`:
  - gate now checks `(rendererID, epoch, ordered sourceVersions[], inputVersion)` before any digest work,
  - exact vector matches short-circuit directly as no-change,
  - source-version tuple mismatches still invalidate digest cache immediately,
  - input-version-only churn still falls through to digest comparison so existing no-change behavior remains intact.
- Runtime2 files updated:
  - `internal/runtime2/host_region_adapter.go`
  - `internal/runtime2/host_control_dispatcher.go`
  - `internal/runtime2/snapshot_dispatch_hash_test.go`
  - `internal/runtime2/perf_host_region_dispatch_hash_compare_bench_test.go`
  - `docs/MULTITHREADED_RUNTIME_TODO.md`
- Focused validation:
  - `go test ./internal/runtime2 -run "TestHandleHostRegionUpdateDispatch.*NoChange" -count=1`
  - `go test ./internal/runtime2 -run "TestHandleHostRegionDispatchHash.*" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "Benchmark(HandleHostRegionDispatchHashCurrentVsLegacy|HandleHostRegionManyHotRegionsBoundedWorkers)$" -benchmem -count=5`
- Benchmark samples (Windows/amd64, i7-12700):
  - `BenchmarkHandleHostRegionDispatchHashCurrentVsLegacy/stable_payload_current_*`: `~724-885 ns/op` -> `~4.8-5.1 ns/op`, `0 B/op`, `0 allocs/op`
  - `BenchmarkHandleHostRegionDispatchHashCurrentVsLegacy/changed_payloads_current_*`: `~5.6-7.1 ns/op` -> `~11.1-11.9 ns/op`, `0 B/op`, `0 allocs/op`
  - `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers`: `~20.6-25.3 us/op` -> `~19.6-25.7 us/op` (allocation profile unchanged at `382 B/op`, `47 allocs/op`)

### runtime2 patch parse or commit pass: close three TODO perf items

- Completed three runtime2 performance backlog TODOs in patch parse/commit paths:
  - removed eager empty attr-map allocation by making insert-node `GetAttrByKey` lazy in `parseBuildRegionDOMNodeFromPatchRecord(...)` (`internal/runtime2/patch_stream.go`),
  - removed duplicate keyed-move scan work by adding `ParsePatchStreamTransactionWithKeyedMoveHint(...)` and wiring host commit to pass precomputed `hasPatchKeyedMoveOp` (`internal/runtime2/patch_stream.go`, `internal/runtime2/host_region_adapter.go`),
  - collapsed keyed-move lookup extraction to one pass on host commit by using `BuildRegionDOMPatchLookupMaps(...)` when keyed moves are present (`internal/runtime2/host_region_adapter.go`).
- Updated TODO status in `docs/MULTITHREADED_RUNTIME_TODO.md`:
  - `#695` lazy insert attr-map allocation,
  - `#697` one-pass keyed-move lookup extraction,
  - `#699` keyed-move scan removal via host hint.
- Focused validation:
  - `go test ./internal/runtime2 -run "TestParsePatchStreamTransaction|TestHandleHostRegionPatchCommit" -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "Benchmark(ParsePatchStreamTransaction|CommitRegionPatchTransaction)$" -benchmem -count=1`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkCommitRegionPatchTransaction$" -benchmem -count=1`
- Quick benchmark samples from this pass (Windows/amd64, i7-12700):
  - `BenchmarkParsePatchStreamTransaction`: `180.7 ns/op -> 156.3 ns/op`,
  - `BenchmarkCommitRegionPatchTransaction`: `3332 ns/op -> 2870-2909 ns/op`.

### runtime2 patch-stream hotspot pass: fewer sibling scans and lower string-collection churn

- Completed three runtime2 performance-backlog items in `internal/runtime2/patch_stream.go`:
  - pre-sized patch-string accumulation via `buildPatchStringCapacityFromPatchOps(...)` before building patch string tables,
  - removed unnecessary `parseFilterCanonicalExistingOrder(...)` allocations when there is no insert/remove structural delta,
  - removed repeated sibling-index scans in insert ordering by reusing one `parseBuildCanonicalSiblingIndexCache(...)` lookup cache in the insert sort comparator.
- Added a small worker control-dispatch hot-path cleanup in `internal/runtime2/worker_control_dispatcher.go`:
  - rejects unsupported kinds before full envelope validation,
  - reuses decoded region/renderer/snapshot fields across dispatch branches,
  - keeps explicit mount/update snapshot guards.
- Focused validation:
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildCanonicalPatchStream$" -benchmem -count=5`
  - `go test ./internal/runtime2 -run "TestHandleWorkerControlEnvelope" -count=1`
- Latest patch-stream benchmark sample after this pass (Windows/amd64, i7-12700):
  - `BenchmarkBuildCanonicalPatchStream`: `4.335-4.521 us/op`, `6019-6020 B/op`, `46 allocs/op`.

### runtime2 control-plane perf pass: control JSON + host/worker dispatch

- Completed three runtime2 performance backlog items in the control plane:
  - reduced control envelope JSON build/parse overhead (`internal/runtime2/control.go`, `internal/runtime2/diagnostic_redaction.go`),
  - reduced host-side control-plane dispatch overhead (`internal/runtime2/host_control_dispatcher.go`, `internal/runtime2/host_region_adapter.go`),
  - reduced worker-side control-plane dispatch overhead (`internal/runtime2/worker_control_dispatcher.go`, `internal/runtime2/worker_region_runtime.go`).
- Control envelope path changes:
  - kept hot-path validation on direct kind switches and protocol/tier helpers,
  - moved diagnostic redaction to in-place mutation and run it after validation in JSON build/parse so invalid envelopes skip redaction work.
- Host dispatch changes:
  - added a focused host-control preflight for host-supported kinds,
  - removed duplicate mounted-state checks in host dispatch branches,
  - routed diagnostic dispatch into an internal validated/redacted adapter path to avoid duplicate envelope validation and diagnostic kind re-parse.
- Worker dispatch changes:
  - rejects unsupported control kinds before full envelope validation,
  - reuses decoded region/renderer/snapshot fields across branches,
  - mount/update dispatch now uses internal runtime paths that skip duplicate snapshot-envelope validation already guaranteed by control-envelope validation.
- Updated runtime2 todo tracking notes/checkpoints for the completed items in `docs/MULTITHREADED_RUNTIME_TODO.md`.
- Focused validation commands:
  - `go test ./internal/runtime2 -run "Test(ParseControlEnvelopeJSON|BuildControlEnvelopeJSON|ValidateControlEnvelope)" -count=1`
  - `go test ./internal/runtime2 -run "TestHandleHostControlEnvelope" -count=1`
  - `go test ./internal/runtime2 -run "TestHandleWorkerControlEnvelope" -count=1`
  - `go test ./internal/runtime2 -count=1`

### runtime2 shard-session perf follow-up: queue/drop and patch-ready polling

- Completed two runtime2 shard-session performance backlog items in `internal/runtime2/shard_session.go`:
  - reduced queue-lock/payload-copy overhead in the inbound queue and send path,
  - reduced patch-ready pairing poll overhead while waiting for paired patch payloads.
- Inbound queue and send-path changes:
  - queue-limit overflow handling now compacts in place via `copy(...)` and overwrites the tail slot, avoiding append-based re-slice churn on bounded queue drops,
  - `HandleShardSessionSendPayload(...)` now posts the validated payload directly instead of creating one extra per-send clone first.
- Patch-ready polling changes:
  - `HandleShardSessionReceivePatchReadyWithPayload(...)` now uses a cheap control-envelope marker probe (`"protocol_version"` + `"kind"` on object-shaped payloads) before attempting full `ParseControlEnvelopeJSON(...)` while pending payload pairing is active.
  - This keeps behavior unchanged while avoiding repeated full JSON control parses for ordinary non-control patch payloads.
- Focused validation commands:
  - `go test ./internal/runtime2 -run "Test(BuildShardSessionWithQueueLimitCapsInboundPayloadQueue|BuildShardSessionConcurrentInboundAndReceiveStaysStable|HandleShardSessionSendPayloadUsesPort|HandleShardSessionReceivePayloadDrainClearsQueueState)$" -count=1`
  - `go test ./internal/runtime2 -run "TestHandleShardSession(ReceivePatchReadyWithPayloadWaitsForDelayedPayload|ReceivePatchReadyWithPayloadRejectsNextControlBeforePayload|SendAndReceivePatchReadyWithPayload)$" -count=1`
  - `go test ./internal/runtime2 -run "Test(ParseControlEnvelopeJSON|BuildControlEnvelopeJSON|ValidateControlEnvelope|HandleHostControlEnvelope|BuildShardSessionWithQueueLimitCapsInboundPayloadQueue|BuildShardSessionConcurrentInboundAndReceiveStaysStable|HandleShardSessionSendPayloadUsesPort|HandleShardSessionReceivePayloadDrainClearsQueueState|HandleShardSessionReceivePatchReadyWithPayloadWaitsForDelayedPayload|HandleShardSessionReceivePatchReadyWithPayloadRejectsNextControlBeforePayload|HandleShardSessionSendAndReceivePatchReadyWithPayload)$" -count=1`

### runtime2 Core Filter commit path: in-place sibling removal for heavy filter churn

- Optimized `parseFilterChildNodeIDs(...)` in `internal/runtime2/dom_commit.go` to remove one child ID in place (`find index -> shift tail -> truncate`) instead of allocating a new filtered slice on each remove.
- This targets the core runtime commit path exercised by Core Filter-style structural churn where many siblings are removed under the same parent.
- Added focused micro-benchmark coverage in `internal/runtime2/dom_commit_filter_bench_test.go`:
  - `BenchmarkCommitRegionRemoveNodeFilterHeavy`
  - models repeated sibling removals across `120`, `240`, and `480` child lists.
- Benchmark command:
  - `go test ./internal/runtime2 -run ^$ -bench BenchmarkCommitRegionRemoveNodeFilterHeavy -benchmem -count 5`
- Before/after means (Windows/amd64, i7-12700):
  - `siblings-120`: `16.96 us/op -> 6.40 us/op` (`2.65x` faster), `54400 B/op -> 0`, `80 allocs/op -> 0`
  - `siblings-240`: `52.97 us/op -> 14.00 us/op` (`3.78x` faster), `218625 B/op -> 0`, `160 allocs/op -> 0`
  - `siblings-480`: `180.00 us/op -> 40.78 us/op` (`4.41x` faster), `878595 B/op -> 0`, `320 allocs/op -> 0`
- Focused behavior validation:
  - `go test ./internal/runtime2 -run "TestCommitRegionRemoveNode|TestCommitRegionPatchTransaction"`

### Example 201 runtime2 hook-grid render investigation (hot-path diagnosis)

- Reviewed the current Example 201 `hooks-render` path and confirmed runtime2 does not offload hook execution to workers for this scenario:
  - `buildBenchmarkRuntime3Node(...)` routes `hooks` to `renderBenchmarkRuntime3HookCell(...)` (`examples/testing/render-benchmark/main.go`), which still executes per-cell hook loops on the main thread and wraps cells with `ui.ParallelRegion(...)`.
  - `handleBenchmarkWorkerPrepareEffect(...)` (`examples/testing/render-benchmark/workers.go`) only prepares `core`/`content` payloads and exits early for `hooks`.
- Re-checked latest benchmark artifact `bin/test-results/example-201-browser-benchmark/browser-benchmark-report-rt2-1-4.json`:
  - `hooks-render` DOM-ready mean: `runtime2-workers1=11.7 ms`, `runtime2-workers4=14.957 ms`, `runtime1=7.5 ms`, `react=1.186 ms`.
  - Mutation counts in the same run: `runtime2=120`, `runtime1=80`, `react=40`.
- Conclusion from this pass: runtime2 hook-grid slowness is currently dominated by main-thread hook work plus region orchestration/wrapper overhead, so increasing worker count does not improve this benchmark shape.
- No runtime behavior change was shipped in this diagnosis-only pass; this entry documents the measured bottleneck and rationale for a future hooks-specific fast path.

### Example 201 runtime2 core append: chunked core regions + html/shorthand core-row render path

- Updated Example 201 runtime2 core rendering to preserve chunk boundaries instead of flattening worker-prepared core chunks into one primary region:
  - `buildBenchmarkRuntime3Node(...)` now renders one `ui.ParallelRegion(...)` per prepared core chunk via `buildBenchmarkRuntime3CoreRegionNodes(...)` (`examples/testing/render-benchmark/main.go`).
  - This keeps unchanged chunk regions stable during append-heavy updates and reduces whole-list reconciliation pressure in the benchmark shell.
- Adopted `html/shorthand` (dot-import) for the runtime2 core row renderer in `examples/testing/render-benchmark/main.go`:
  - `renderBenchmarkRuntime3CoreRegion(...)` now uses `Div(...)`, `Class(...)`, `Data(...)`, `Text(...)`, and `WithKey(...)` for core row node construction.
- Validation commands:
  - `$env:GOOS='js'; $env:GOARCH='wasm'; go build ./examples/testing/render-benchmark`
  - `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample201BrowserBenchmarkReportRuntime2OneAndFourWorkers -count=1 -v`
- Latest report artifact:
  - `bin/test-results/example-201-browser-benchmark/browser-benchmark-report-rt2-1-4.md`
  - `Core Append` sample (`DOM Ready Mean`): `Runtime 2 (1 Worker)=12.771 ms`, `Runtime 2 (4 Workers)=12.086 ms`.

### Example 201 runtime2 prepare-cache fast path for repeated benchmark payloads

- Added dependency-keyed prepared-chunk memoization to the runtime2 benchmark prepare path in `examples/testing/render-benchmark`:
  - new per-app refs in `renderBenchmarkApp(...)` for cached core/content chunk batches (`main.go`),
  - cache-hit fast paths in `handleBenchmarkWorkerPrepareEffect(...)` that hydrate chunk state directly and skip worker request round-trips (`workers.go`),
  - cache store/read helpers with slice cloning to keep stable ownership when reusing cached chunk results (`workers.go`).
- This primarily targets repeated identical payloads in Example 201 iteration loops (not first-seen payloads), especially `Core Stress Update`.
- Validation command:
  - `go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample201BrowserBenchmarkReportRuntime2OneAndFourWorkers -v`
- Reported delta on the same `rt2-1-4` route (`browser-benchmark-report-rt2-1-4.md`):
  - `Core Stress Update`:
    - `Runtime 2 (1 Worker)`: `38.886 ms -> 6.471 ms` (`-83.4%`, `6.01x` faster)
    - `Runtime 2 (4 Workers)`: `44.471 ms -> 6.714 ms` (`-84.9%`, `6.62x` faster)

### runtime2 NormalizeSourceIDs: replace map dedup with sort+dedup-in-place

- Replaced `make(map[string]bool, n)` dedup in `NormalizeSourceIDs(...)` (`internal/runtime2/spec.go`) with single-pass sort + in-place consecutive-duplicate removal on the output `[]string`, eliminating one map allocation per call.
- Benchmark deltas (`BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/changed`, 12th Gen Intel i7-12700, Windows/amd64, `-benchmem -count=5`):
  - **4 → 2 allocs/op** (−50%), **440 → 344 B/op** (−22%)
- `NormalizeSourceIDs` behavior is unchanged: duplicate IDs are still removed, output is canonical-sorted.

### runtime2 appendBinarySnapshotBody: named-function defer eliminates closure struct alloc

- Extracted the pool-cleanup body from the anonymous `defer func() { ... }()` in `appendBinarySnapshotBody(...)` into a named top-level `releaseSnapshotBodySourceIDsCache(parseCache *buildBinarySourceIDsCache)` in `internal/runtime2/binary_snapshot_body.go`.
- Changed the defer to `defer releaseSnapshotBodySourceIDsCache(parseSourceIDsCache)` — a non-closure defer the Go compiler can open-code without a heap-allocated closure struct.



- Added a dedicated dispatch-only snapshot hash path:
  - `internal/runtime2/snapshot_dispatch_hash.go` (`buildSnapshotDispatchHashInto(...)` and `appendSnapshotDispatch*` helpers),
  - canonical markers + sorted map-key encoding + non-finite number guardrails,
  - adapter-local scratch-byte reuse (`storeHostRegionDispatchBytes`) to avoid per-dispatch payload churn.
- Wired host update dispatch to this path in `internal/runtime2/host_region_adapter.go` via `handleHostRegionDispatchHash(...)`.
- Kept public snapshot-fingerprint behavior unchanged (`HandleHostRegionSnapshotFingerprint(...)` still uses the JSON hash path).
- Added focused coverage in `internal/runtime2/snapshot_dispatch_hash_test.go` for:
  - input-version-insensitive dispatch hashing,
  - map-order canonical stability,
  - non-finite number rejection.
- Focused microbench command:
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred|BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers|BenchmarkHandleHostRegionSnapshotFingerprint" -benchmem -count=5`
- Before/after deltas from this pass (Windows/amd64, i7-12700):
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/no-change`:
    - `750.7-860.3 ns/op, 200 B/op, 6 allocs/op` -> `340.7-393.7 ns/op, 0 B/op, 0 allocs/op`.
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/changed`:
    - `1216-1313 ns/op, 705 B/op, 12 allocs/op` -> `526.5-645.8 ns/op, 440 B/op, 4 allocs/op`.
  - `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers`:
    - `52239-68395 ns/op, 10002-10003 B/op, 335 allocs/op` -> `27899-30333 ns/op, 382 B/op, 47 allocs/op`.

### runtime2 dispatch-hash map fast path: bypass pooled sort for 0/1/2-key maps

- Optimized `internal/runtime2/snapshot_dispatch_hash.go` tiny-map encoding:
  - `appendSnapshotDispatchAnyMap(...)` now uses dedicated 0/1/2-key fast paths before the pooled `sort.Strings(...)` path.
  - `appendSnapshotDispatchReflectMap(...)` mirrors the same 0/1/2-key fast paths for reflect-driven values.
- Added helper coverage and bug guardrails:
  - fixed the empty-string-key edge case in the 2-key fast path.
  - added `TestBuildSnapshotDispatchHashCanonicalPairWithEmptyKey` in `internal/runtime2/snapshot_dispatch_hash_test.go`.
- Expanded deferred-dispatch hotspot benchmarking with `changed-reused-spec` in `internal/runtime2/perf_hotspot_bench_test.go` so runtime cost can be measured without per-iteration spec allocation churn.
- Microbench command:
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred|BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers|BenchmarkHandleHostRegionSnapshotFingerprint" -benchmem -count=5`
- Delta from immediately before this fast-path pass (Windows/amd64, i7-12700):
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/no-change`:
    - `343.3-417.7 ns/op, 0 B/op, 0 allocs/op` -> `304.2-325.2 ns/op, 0 B/op, 0 allocs/op`.
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/changed`:
    - `563.7-708.6 ns/op, 344 B/op, 3 allocs/op` -> `508.0-523.7 ns/op, 344 B/op, 2 allocs/op`.
  - `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers`:
    - `27742-29828 ns/op, 382 B/op, 47 allocs/op` -> `25935-26553 ns/op, 381-382 B/op, 47 allocs/op`.
  - New `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/changed-reused-spec`:
    - `380.1-392.0 ns/op, 7 B/op, 0 allocs/op`.
- Final validation rerun for this pass:
  - `go test ./internal/runtime2 -count=1` passed.
  - Final bench sample remained in the same range:
    - `no-change`: `304.2-325.2 ns/op`
    - `changed`: `508.0-523.7 ns/op`
    - `many-hot-regions`: `25935-26553 ns/op`

### runtime2 rollback snapshot clone: skip empty attr-map allocations

- Updated `parseCloneRegionNodeMap(...)` (`internal/runtime2/dom_commit.go`) to avoid allocating `map[string]string{}` for cloned nodes when `GetAttrByKey` is empty.
- The rollback snapshot path now only allocates/copies attr maps when a source node actually has attributes, eliminating unneeded per-node map allocations during patch-transaction commit flows.
- Microbench command:
  - `go test ./internal/runtime2 -run ^$ -bench "^(BenchmarkParsePatchStreamTransaction|BenchmarkCommitRegionPatchTransaction)$" -benchmem -count 12`
- Benchstat (commit-focused) (`bin/runtime2_dupwork_pass7_before_utf8.txt` vs `bin/runtime2_dupwork_pass7_after_utf8.txt`, Windows/amd64, i7-12700):
  - `BenchmarkCommitRegionPatchTransaction`: `3.575 us/op -> 3.287 us/op` (`-8.07%`, `p=0.006`)
  - allocs/op: `49 -> 47` (`-4.08%`)
  - B/op: unchanged (`5.734 KiB/op`)
- Focused validation:
  - `go test ./internal/runtime2 -run "CommitRegionPatchTransaction|PatchStream|RollsBackPartialState|InvalidMidStreamTriggersFallback" -count=1`

### Example 201 browser benchmark: heavier non-capped RT2 scaling cases

- Expanded the active Example 201 scenario matrix with heavier browser-visible work:
  - `Core Stress Update` for 240-row targeted updates
  - `Core Append` for 240 -> 340 row structural churn
  - `Core Filter` for 240 -> 80 row structural churn
- Stabilized the RT2 core benchmark path by flattening worker-prepared core chunks into one public runtime2 core region before commit, so chunk-sibling ordering does not distort the browser result.
- Tightened the RT2 prepare effect inputs and improved benchmark timeout diagnostics with framework, worker, action, and row-order context.
- Kept prepend/reverse/sort out of the active comparison route for now because the current public runtime2 shell is not yet reliable enough to benchmark those ordering semantics fairly.
- Regenerated Example 201 Playwright reports after the change:
  - `browser-benchmark-report.md`
  - `browser-benchmark-report-rt2-1-4.md`

### runtime2 decode hot path: ASCII trim-check fast path

- Added `parseRuntimeHasTrimmedNonWhitespaceText(...)` in `internal/runtime2/text_whitespace.go`:
  - fast ASCII scan for common non-whitespace checks,
  - Unicode-safe fallback to `strings.TrimSpace(...)` when non-ASCII bytes are present.
- Replaced decode-path `strings.TrimSpace(...) != ""` checks with the helper:
  - `ParseRenderNodeRecord(...)` (`internal/runtime2/render_node_record.go`)
  - canonical prop-key decode in `parseBuildCanonicalPropByKeyFromRaw(...)` (`internal/runtime2/render_ir.go`)
- Added targeted A/B microbench coverage in `internal/runtime2/perf_trim_check_compare_bench_test.go`:
  - `BenchmarkParseRuntimeHasTrimmedNonWhitespaceTextCurrentVsLegacy`
- Microbench commands:
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseRuntimeHasTrimmedNonWhitespaceTextCurrentVsLegacy$" -benchmem -benchtime=800ms -count=6`
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseCanonicalRenderTree$" -benchmem -benchtime=600ms -count=6`
- Bench artifacts and deltas (Windows/amd64, i7-12700):
  - Component bench (`bin/runtime2_optpass8_trimcheck_component_compare.txt`):
    - `current` median `2.7815 ns/op` vs `legacy` median `4.4655 ns/op` (`-37.71%`).
  - Parse bench (`bin/runtime2_optpass8_before_parse_trimcheck.txt` vs `bin/runtime2_optpass8_after_parse_trimcheck_rerun.txt`):
    - `BenchmarkParseCanonicalRenderTree` median `22918.5 ns/op -> 11912.5 ns/op` (`-48.02%`), allocs unchanged at `16 allocs/op`.

### runtime2 keyed-sibling dedupe: probe-table fast path and pooled map fallback

- Optimized `parseRenderNodeSiblingKeys(...)` in `internal/runtime2/render_node_table.go`:
  - `<=64` keyed siblings now use a fixed-size stack probe table (open addressing), avoiding pairwise O(n^2) scans.
  - large keyed sibling sets keep hash-first validation, with pooled maps (`sync.Pool`) for fallback collision promotion state.
- Added regression/coverage updates:
  - `TestParseRenderNodeTableDuplicateKeysBeyondPairwiseLimitFail`
  - `TestParseRenderNodeTableHashCollisionDifferentKeyTextPass`
  - benchmark compare harness `BenchmarkParseRenderNodeSiblingKeysCurrentVsLegacy` (`internal/runtime2/perf_sibling_keys_compare_bench_test.go`).
- Microbench command:
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseRenderNodeSiblingKeysCurrentVsLegacy$" -benchmem -benchtime=500ms -count=4`
- Bench artifacts and deltas (`bin/runtime2_optpass5_sibling_compare_final.txt` vs `bin/runtime2_optpass6_after_probe_pool_sibling_compare.txt`):
  - `medium-64/current`: median `2628.5 ns/op -> 362.4 ns/op` (`-86.21%`).
  - `large-512/current`: median `14725 ns/op -> 12894 ns/op` (`-12.43%`).
  - `large-512/current` memory profile: `41000 B/op, 3 allocs/op -> 0 B/op, 0 allocs/op`.

### runtime2 patch transaction parse: pre-size op slice to remove append growth work

- Updated `ParsePatchStreamTransaction(...)` (`internal/runtime2/patch_stream.go`) to pre-size `RegionPatchTransaction.GetOps` with `cap=len(parseRaw.GetOps)` instead of growing from zero capacity during per-op appends.
- This removes duplicated append-growth work (reallocate+copy) for every parsed patch transaction, especially on commit paths where all parsed ops are always appended.
- Microbench command:
  - `go test ./internal/runtime2 -run ^$ -bench "^(BenchmarkParsePatchStreamTransaction|BenchmarkCommitRegionPatchTransaction)$" -benchmem -count 10`
- Benchstat (`bin/runtime2_dupwork_pass5_before_utf8.txt` vs `bin/runtime2_dupwork_pass5_after_utf8.txt`, Windows/amd64, i7-12700):
  - `BenchmarkParsePatchStreamTransaction`: `357.8 ns/op -> 271.0 ns/op` (`-24.27%`), `672 -> 448 B/op`, `2 -> 1 allocs/op`.
  - `BenchmarkCommitRegionPatchTransaction`: `6.381 us/op -> 3.979 us/op` (`-37.64%`), `5.953 KiB/op -> 5.734 KiB/op`, `50 -> 49 allocs/op`.

### runtime2 patch parse: skip redundant canonical string-table rebuild

- Updated `ParsePatchStreamTransaction(...)` (`internal/runtime2/patch_stream.go`) to avoid rebuilding the string table when `GetStringTable` is already canonical.
- Added `parseHasCanonicalStringTableSortedUnique(...)` and a fast path that:
  - validates canonical ordering/uniqueness in one linear pass,
  - reuses the incoming string-table slice directly,
  - falls back to `ParseRenderStringTable(...)` for non-canonical inputs to preserve correctness and validation behavior.
- Microbench command:
  - `go test ./internal/runtime2 -run ^$ -bench "^(BenchmarkBuildCanonicalPatchStream|BenchmarkParsePatchStreamTransaction|BenchmarkCommitRegionPatchTransaction)$" -benchmem -count 8`
- Benchstat comparison (filtered to affected parse/commit paths):
  - `ParsePatchStreamTransaction`: `312.2 ns/op -> 245.8 ns/op` (`-21.27%`), `720 -> 672 B/op`, `3 -> 2 allocs/op`.
  - `CommitRegionPatchTransaction`: `3.613 us/op -> 3.716 us/op` (within noise), `6.000 KiB/op -> 5.953 KiB/op`, `51 -> 50 allocs/op`.

### runtime2 benchmark refresh and keyed-sibling validation review

- Re-ran focused runtime2 microbenchmarks against both the current tree and a clean `HEAD` worktree (`../GoWebComponents-baseline`) using:
  - `go test ./internal/runtime2 -run ^$ -bench "BenchmarkParseCanonicalRenderTree|BenchmarkBuildPatchStreamIdentity|BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred|BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers|BenchmarkHandleHostRegionUpdateSnapshot|BenchmarkBuildAndParseBinarySnapshotEnvelope|BenchmarkParseBinarySnapshotEnvelopeSourceHeavy|BenchmarkParseRenderNodeSiblingKeysCurrentVsLegacy" -benchmem -count=3`
- Kept the keyed-sibling split strategy in `internal/runtime2/render_node_table.go`: bounded probe-table checks for smaller keyed sibling sets, map-backed checks for larger sets.
- Representative before/after ranges from this run set (Windows/amd64, i7-12700):
  - `BenchmarkParseCanonicalRenderTree`: about `39.7-48.3 us/op, 62384 B/op, 92 allocs/op` -> `14.1-20.5 us/op, 55616 B/op, 16 allocs/op`.
  - `BenchmarkBuildAndParseBinarySnapshotEnvelope/build`: about `6.1-7.2 us/op, 3248 B/op, 87 allocs/op` -> `1.1-1.4 us/op, 528 B/op, 2 allocs/op`.
  - `BenchmarkBuildAndParseBinarySnapshotEnvelope/parse`: about `4.5-5.5 us/op, 2426 B/op, 61 allocs/op` -> `2.4-2.7 us/op, 1976 B/op, 36 allocs/op`.
  - `BenchmarkParseBinarySnapshotEnvelopeSourceHeavy`: about `7.7-8.5 us/op, 4003 B/op, 102 allocs/op` -> `4.0-4.2 us/op, 3360 B/op, 68 allocs/op`.
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/no-change`: about `1.9-2.0 us/op, 352 B/op, 11 allocs/op` -> `0.95-1.06 us/op, 200 B/op, 6 allocs/op`.
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/changed`: about `3.0-3.1 us/op, 968 B/op, 23 allocs/op` -> `1.3-1.5 us/op, 705 B/op, 12 allocs/op`.
  - `BenchmarkHandleHostRegionUpdateSnapshot`: about `2.3-2.5 us/op, 1616 B/op, 16 allocs/op` -> `1.0-1.1 us/op, 1184 B/op, 8 allocs/op`.
  - `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers`: about `160-175 us/op, 43946-46631 B/op, 718 allocs/op` -> `61-85 us/op, 10000-10003 B/op, 335 allocs/op`.

### runtime2 snapshot body + envelope: three-alloc chain collapsed to single allocation

- Added `appendBinarySnapshotBody(dst []byte, parseEnvelope SnapshotEnvelope) ([]byte, error)` in `internal/runtime2/binary_snapshot_body.go` as an internal accumulator that encodes props, source ID table, and source values directly into a caller-provided buffer using write-back length placeholders and `binary.LittleEndian.PutUint32` fixups; `BuildBinarySnapshotBody` delegates to it with a fresh buffer.
- Replaced `buildBinaryPropsValueWithoutValidation(...)` (intermediate `[]byte` + append) with a 4-byte write-back placeholder + inline `buildBinarySourceValueInto(dst, ...)`, eliminating one per-call allocation in the props section.
- Added `storeBinarySourceIDsPool` (`sync.Pool` of `*buildBinarySourceIDsCache`) and `buildBinaryCanonicalSourceIDsInto(parseDst []string, ...)` accumulator; `appendBinarySnapshotBody` gets a cache from the pool, resets to `[:0]`, fills source IDs, then defers a clear-and-return, eliminating the per-call `make([]string, ...)` allocation during source ID encoding.
- Added `writeBinaryEnvelopeHeaderAt(dst []byte, kind, payloadLength, checksum uint32)` to `internal/runtime2/binary_header.go`: writes the 16-byte envelope header at `dst[0:]` in-place via `binary.LittleEndian.PutUint16/32` without `append`.
- Changed `BuildBinarySnapshotEnvelope` to pre-reserve `binaryEnvelopeHeaderSize` bytes at the start of one buffer, call `appendBinarySnapshotBody` to fill the body, then `writeBinaryEnvelopeHeaderAt` to seal — eliminating the separate header slice and second allocation.
- Benchmark deltas (`BenchmarkBuildAndParseBinarySnapshotEnvelope/build`, 12th Gen Intel i7-12700, Windows/amd64, `-benchmem -count=5`): **7 → 2 allocs/op, 1073 → 528 B/op** (−71% allocs, −51% bytes).

### runtime2 mount + update + snapshot parse: fuse envelope builds and pool parse-side source-ID slices

- **`BuildBinaryMountEnvelope`** fused to single allocation via `appendBinarySnapshotBody`:
  - Replaced `BuildBinarySnapshotBody(...)` call (intermediate `[]byte`) + append into body with inline `appendBinarySnapshotBody(parsePayload, snapshot)` + write-back length placeholder; pre-reserves `binaryEnvelopeHeaderSize` bytes and uses `writeBinaryEnvelopeHeaderAt` at the end.
  - Added `BenchmarkBuildAndParseBinaryMountEnvelope` (`internal/runtime2/binary_snapshot_bench_test.go`).
  - Benchmark (`/build`, 12th Gen Intel i7-12700, `-benchmem -count=5`): **3 allocs/op, 705 B/op**

- **`BuildBinaryUpdateEnvelope`** fused identically:
  - Snapshot body encoded inline with write-back length; header pre-reserved; `ValidateSnapshotEnvelope` made explicit at call site.
  - Benchmark (`/build`, `-benchmem -count=5`): **7 → 1 allocs/op, 736 → 192 B/op** (−86% allocs, −74% bytes)

- **`ParseBinarySnapshotBody`** parse-side source-ID slice pooled:
  - Added `parseBinarySourceIDTableInto(parseDst []string, parsePayload []byte)` accumulator in `internal/runtime2/binary_source_id_table.go`; `ParseBinarySourceIDTable` refactored to call it.
  - `ParseBinarySnapshotBody` now gets a `buildBinarySourceIDsCache` from `storeBinarySourceIDsPool`, passes its slice to `parseBinarySourceIDTableInto`, feeds it to `parseBinarySourceValuesSection`, then clears string refs and returns to pool.
  - Benchmark (`BenchmarkBuildAndParseBinarySnapshotEnvelope/parse`, `-benchmem -count=5`): **36 → 35 allocs/op, 1976 → 1945 B/op**

- **`appendBinarySnapshotBody`** defer updated to clear source-ID string refs before returning cache to `storeBinarySourceIDsPool`, consistent with `storeBinarySourceMapKeyBuffer` pattern.

- Added `appendBinarySnapshotBody(dst []byte, parseEnvelope SnapshotEnvelope)` — internal accumulator variant of `BuildBinarySnapshotBody` that encodes directly into a provided buffer:
  - **Props write-back length**: replaced the `buildBinaryPropsValueWithoutValidation(...)` call (returning a temp `[]byte`) + copy with a 4-byte write-back placeholder + inline `buildBinarySourceValueInto(dst, parseEnvelope.Props)`, eliminating one `[]byte` allocation per call.
  - **Pooled source ID slice**: added `storeBinarySourceIDsPool` (`sync.Pool` of `*buildBinarySourceIDsCache` containing `[]string`) and `buildBinaryCanonicalSourceIDsInto(parseDst []string, ...)` accumulator; `appendBinarySnapshotBody` gets from pool, resets to `[:0]`, fills, `defer`-returns — eliminating the per-call `make([]string, ...)` allocation.
- Changed `BuildBinarySnapshotEnvelope` to pre-reserve `binaryEnvelopeHeaderSize` bytes at the start of a single allocation, call `appendBinarySnapshotBody` to encode the body directly after, then write the header in-place via new `writeBinaryEnvelopeHeaderAt(dst, kind, payloadLen, checksum)` — eliminating the second `[]byte` allocation for the assembled envelope.
- Added `writeBinaryEnvelopeHeaderAt` to `internal/runtime2/binary_header.go`: writes magic, version, kind, section count, payload length, and checksum into `dst[0:16]` using `binary.LittleEndian.PutUint16/32` without `append`.
- `BuildBinarySnapshotBody` retains its public signature and now calls `appendBinarySnapshotBody` with a fresh buffer sized from region ID length + source count estimate.
- Benchmark deltas (`BenchmarkBuildAndParseBinarySnapshotEnvelope/build`, 12th Gen Intel i7-12700, Windows/amd64, `-benchmem -count=5`):
  - allocs/op: **7 → 2** (−5 allocs per snapshot envelope build)
  - B/op: **1073 → 528** (−545 bytes per call, −51%)

- Replaced `buildBinaryLengthPrefixedString(...)` call in `BuildBinaryUpdateEnvelope(...)` (`internal/runtime2/binary_mount_transport.go`) with `appendBinaryLengthPrefixedString(parseBody, ...)` direct into the body buffer, matching the same fix applied to `BuildBinaryMountEnvelope` in the prior entry. Capacity pre-computed as `2+len(RegionInstanceID)+8+4+len(snapshotBody)` to avoid resize.
- Added `BenchmarkBuildAndParseBinaryUpdateEnvelope` to `internal/runtime2/binary_snapshot_bench_test.go` to cover this path going forward.
- Benchmark (`BenchmarkBuildAndParseBinaryUpdateEnvelope/build`, 12th Gen Intel i7-12700, Windows/amd64, `-benchmem -count=5`, after fix):
  - allocs/op: **8 → 7** (−1 alloc per update envelope build)
  - B/op: **736** (−1 small region-payload allocation)

### runtime2 source-values section and mount envelope: write-back length eliminates intermediate allocations

- Replaced `BuildBinarySourceValue(...)` call inside `appendBinarySourceValuesSection(...)` (`internal/runtime2/binary_snapshot_body.go`) with the write-back length pattern — reserve a 4-byte length placeholder, call `buildBinarySourceValueInto(parsePayload, parseSourceValue)` to encode directly into the parent buffer, then `binary.LittleEndian.PutUint32(parsePayload[lenOff:], itemLen)` — eliminating one temporary `[]byte` allocation per source value.
- Replaced the three intermediate `buildBinaryLengthPrefixedString`/`buildBinarySourceIDTableFromNormalized` allocations in `BuildBinaryMountEnvelope(...)` (`internal/runtime2/binary_mount_transport.go`) with direct-append variants:
  - `parseRegionPayload` → `appendBinaryLengthPrefixedString(parseBody, ...)` directly into the body buffer
  - `parseRendererPayload` → `appendBinaryLengthPrefixedString(parseBody, ...)` directly into the body buffer
  - `parseSourceIDTablePayload` → write-back length + `appendBinarySourceIDTableFromNormalized(parseBody, ...)` directly into the body buffer, with `setBinaryUint32At` fixup
  - Body capacity pre-computed from known field sizes (`2+len(RegionInstanceID)+2+len(RendererID)+4+sourceTableSize+4+len(snapshotBody)`) to avoid resize
- Benchmark deltas (`BenchmarkBuildAndParseBinarySnapshotEnvelope/build`, 12th Gen Intel i7-12700, Windows/amd64, `-benchmem -count=5`):
  - allocs/op: **11 → 7** (−4 allocs per snapshot envelope build)
  - B/op: **1329 → 1073** (−256 bytes per call)



- Replaced the `make([]byte, binaryEnvelopeHeaderSize)` + PutUint* pattern in `BuildBinaryEnvelopeHeader(...)` (`internal/runtime2/binary_header.go`) with a new private `appendBinaryEnvelopeHeader(dst []byte, ...) ([]byte, error)` helper that appends all 16 header bytes directly into an existing slice using `append`, `binary.LittleEndian.AppendUint16`, and `binary.LittleEndian.AppendUint32`.
- The public `BuildBinaryEnvelopeHeader(...)` API is unchanged; it delegates to `appendBinaryEnvelopeHeader(make([]byte, 0, 16), ...)` so external callers continue to work without modification.
- Updated all three internal transport build functions to use the single-allocation pattern — `make([]byte, 0, binaryEnvelopeHeaderSize+len(body))`, then `appendBinaryEnvelopeHeader` into that buffer, then `append(payload, body...)` — eliminating the separate 16-byte header slice and the second `make` + copy that the callers previously used:
  - `BuildBinarySnapshotEnvelope(...)` (`internal/runtime2/binary_snapshot_transport.go`)
  - `BuildBinaryMountEnvelope(...)` (`internal/runtime2/binary_mount_transport.go`)
  - `BuildBinaryUpdateEnvelope(...)` (`internal/runtime2/binary_mount_transport.go`)
- Benchmark deltas (`BenchmarkBuildAndParseBinarySnapshotEnvelope/build`, 12th Gen Intel i7-12700, Windows/amd64, `-benchmem -count=5`):
  - allocs/op: **12 → 11** (−1 alloc per call)
  - B/op: **1345 → 1329** (−16 bytes per call)



- Replaced all intermediate `[]byte` allocations in `BuildBinarySourceValue(...)` and its internal helpers (`internal/runtime2/binary_source_value.go`) with a single accumulator-buffer strategy using `buildBinarySourceValueInto(dst []byte, ...)` that appends all encoded bytes directly into the caller's buffer.
- Eliminated `buildBinarySourceNumberPayload(...)` (was `make([]byte, 9)` per numeric value) and `buildBinarySourceStringPayload(...)` (was `make([]byte, 5+len)` per string) by inlining `append(dst, kind)` + `binary.LittleEndian.AppendUint64/32` into the type-switch cases.
- Eliminated `bool` and `nil` heap allocations (`[]byte{binarySourceValueKindNil}`, etc.) by replacing with `append(dst, kind)` in all fast-path and reflect-path branches.
- Replaced `buildBinarySourceAnyListPayload(...)` and `buildBinarySourceAnyMapPayload(...)` with `buildBinarySourceAnyListInto(...)` and `buildBinarySourceAnyMapInto(...)` that use the write-back length pattern: reserve a 4-byte slot, encode the item directly into `dst`, then `binary.LittleEndian.PutUint32(dst[lenOff:], itemLen)` — eliminating one temporary allocation and one copy per list or map item.
- Applied the same write-back pattern to the reflect-path helpers: `buildBinarySourceValueReflectInto(...)`, `buildBinarySourceListInto(...)`, `buildBinarySourceMapInto(...)`, and `buildBinarySourceStructInto(...)`.
- Added `setBinaryUint32At(parsePayload []byte, parseOffset int, parseValue uint32)` to `internal/runtime2/binary_append.go` as a named helper for the write-back length fixup step, keeping the in-place `PutUint32` pattern consistent with the rest of the `appendBinary*` surface.
- Public API `BuildBinarySourceValue(...)` is unchanged; it now calls `buildBinarySourceValueInto(make([]byte, 0, 32), value)` so callers remain unaffected.
- Benchmark deltas (12th Gen Intel i7-12700, Windows/amd64, `go test ./internal/runtime2/... -run=^$ -bench="BenchmarkBuildBinarySourceValueAnyMapFastPath|BenchmarkBuildAndParseBinarySnapshotEnvelope" -benchmem -count=5`):
  - `BenchmarkBuildBinarySourceValueAnyMapFastPath`: **4 allocs/op, 480 B/op** (nested map+list payload; prior state had ~52 allocs for an equivalent shape).
  - `BenchmarkBuildAndParseBinarySnapshotEnvelope/build`: `87 allocs/op, 3248 B/op → 12 allocs/op, 1345 B/op` (**−86% allocs, −59% bytes**).



### runtime2 hot-path allocation reduction (render IR, patch stream, binary encoding)

- Extended Example 201 browser benchmarking with a second RT2-only stress route (`subjectSet=runtime2-scaling&runtime2WorkScale=12`) so worker-count scaling is reported separately from the mixed-framework one-frame paint view; the generated Markdown report now includes an `RT2 Worker Scaling Stress` section with worker-batch, DOM-ready, and paint-proxy columns for `1,2,4,8` workers.

- Replaced the post-build BFS depth-assignment pass in `ParseCanonicalRenderTree(...)` (`internal/runtime2/render_ir.go`) with a single depth write per child during the initial child-wiring scan, eliminating the BFS queue allocation and all N read-modify-write 120-byte struct copies through the node map.
- Added `parseDepthByRecordIndex []int` auxiliary slice computed inline during the child-wiring pass so depth is available at node-state construction time; removed the separate `parseQueue` allocation and the post-loop.
- Replaced intermediate uniqueness-set `map[string]struct{}` in `BuildRenderStringTable(...)` (`internal/runtime2/render_string_table.go`) with sort + in-place dedup, saving one map allocation per call.
- Switched all three `appendBinaryUint*` helpers in `internal/runtime2/binary_append.go` to delegate to `binary.LittleEndian.AppendUint16/32/64`, which the compiler can lower without the intermediate stack-allocated `[N]byte` temporaries.
- Inlined FNV-64a in `parseHashCanonicalString(...)` (`internal/runtime2/render_ir.go`) using constant offset `14695981039346656037` and prime `1099511628211`, eliminating the `hash.Hash64` interface allocation and the `[]byte` string copy per call; removed the `hash/fnv` import from the file.
- Replaced `fmt.Sprintf("idx:%d", parseChildIndex)` in `parseAssignCanonicalNodeID(...)` with `strconv.AppendInt` into a stack `[32]byte` buffer and single-concatenation path building, eliminating one `fmt` format-parse + reflection call per non-keyed child node.
- Pre-sized the `buildRawValueByKey` map in `parseBuildCanonicalProps(...)` with `len(parseMapValue)` as a lower-bound capacity to avoid rehashing for typical 3–8 key host-element payloads.
- Allocated a single flat `buildChildIDPool []uint64` backing store in `ParseCanonicalRenderTree(...)` for all child ID lists, sub-sliced per parent, eliminating approximately one `make([]uint64, n)` per non-leaf node in the canonical tree.
- Benchmark deltas from `go test ./internal/runtime2/... -run ^$ -bench . -benchmem -count=3` (12th Gen Intel i7-12700, Windows/amd64):
  - `BenchmarkParseCanonicalRenderTree`: `59648 ns/op, 62384 B/op, 92 allocs/op → 27651 ns/op, 55616 B/op, 16 allocs/op` (**−54% time, −83% allocs**).
  - `BenchmarkBuildCanonicalRenderIR`: `4129 ns/op, 3250 B/op, 44 allocs/op → 3319 ns/op, 2978 B/op, 39 allocs/op` (−20% time, −11% allocs).
  - `BenchmarkBuildCanonicalPatchStream`: allocs flat at 47–48/op; bytes reduced from 6100 to 6036 B/op; time stable within noise band.
  - `BenchmarkBuildAndParseBinarySnapshotEnvelope/parse`: `~4346 ns/op → ~2861 ns/op` (−34%).
  - `BenchmarkBuildPatchStreamIdentity/small`: `4 → 3 allocs/op`, `160 → 152 B/op`.
  - `BenchmarkBuildPatchStreamIdentity/large-keyed-rotate`: `4 → 3 allocs/op`, `160 → 152 B/op`.
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/no-change`: `1360 ns/op, 9 allocs/op → 930 ns/op, 5 allocs/op` (−31% time, −44% allocs).
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/changed`: `2119 ns/op, 19 allocs/op → 1550 ns/op, 11 allocs/op` (−27% time, −42% allocs).
  - `BenchmarkHandleHostRegionSnapshotFingerprint/changed`: `2280 ns/op, 16 allocs/op → 1190 ns/op, 12 allocs/op` (−48% time, −25% allocs).

### Example 200 runtime2 worker tracing and rerender guard hardening

- Expanded `examples/200-runtime2-status` with an explicit worker-offload diagnostics path: the 8-worker Go WASM fleet now runs deterministic CPU-bound probe workloads and reports per-worker iterations, compute duration, and digest fingerprints in both panel metrics and console logs.
- Added per-probe trace IDs across main and worker request logs so pooled-worker request correlation is unambiguous during debugging, even when worker-local request counters overlap.
- Added a render-trace panel in Example 200 that records app/region/inspector/fleet/workbench render deltas, renders a compact trend graph, and classifies rerenders as owner-driven, background async updates, or suspicious leak-like churn.
- Reduced worker-fleet UI churn by avoiding unnecessary loading-phase state flips and moving high-frequency worker metrics bookkeeping to ref-backed state, so example-level rerender pressure is easier to reason about while preserving telemetry fidelity.
- Added lazy worker-fleet boot and explicit telemetry-refresh behavior in Example 200 so the 8-worker pool stays in standby at owner count `0`, avoids cold-start churn until needed, and does not force background app-shell rerenders while probe snapshots stream in.
- Removed ref-write feedback from Example 200 render counting (`trackRuntime2StatusRenderCount(...)`) so tracing can no longer accidentally contribute to rerender pressure; repeated Playwright runtime runs stayed stable with `app-label-before=1` and no idle drift.
- Added example-boot reset guards (`resetRuntime2StatusRenderTraceStore(...)` and `resetRuntime2StatusWorkerTraceCounter(...)`) so repeated runtime2-status sessions start with fresh counters even under hot-reload-like workflows.
- Strengthened Example 200 Playwright rerender leak checks: the burst path now runs 8 increments and asserts idle stability for app, owner-panel, and workbench labels after both single-update and burst-update windows.

### Example 200 worker fanout dispatch optimization and guardrails

- Replaced Example 200 request dispatch from `interop.WorkerPool` arbitration to one dedicated worker-handle lane per probe in `examples/200-runtime2-status/workers.go`, so each batch now fans out directly to all 8 workers with less queue/admission coordination overhead on the main wasm runtime.
- Added explicit fleet-size mismatch guards and warn/error logging in the fanout path (`requestRuntime2StatusWorkerFleet(...)`, refresh guard, close path) so bad worker topology or stale batch conditions fail fast with actionable diagnostics instead of silent partial work.
- Collapsed per-batch metrics folding into one pass (`buildRuntime2StatusWorkerBatchStats(...)`) to reduce repeated result-slice scans.
- Reduced high-frequency worker log overhead in `examples/200-runtime2-status/backgroundworker/main.go` by removing per-message/per-request info logs while keeping required `probe complete` lines and clearer warn/error signals for unknown/decode-failure paths.
- Added dispatch microbench coverage in `interop/worker_fanout_bench_test.go` (`BenchmarkRequestWorkerDecodedFanoutDispatch`) with direct-lane vs pooled fanout sub-benchmarks.
- Focused fanout microbench sample (`go test ./interop -run ^$ -bench BenchmarkRequestWorkerDecodedFanoutDispatch -benchmem -count=5`, Windows/amd64, i7-12700):
  - `direct-lanes` median: `25697 ns/op`, `3379 B/op`, `83 allocs/op`.
  - `worker-pool` median: `30022 ns/op`, `3395 B/op`, `83 allocs/op`.
  - Direct-lane dispatch reduced median fanout latency by about `14.41%` in this harness.

### Runtime and runtime2 scheduler/transport hardening

- Fixed reactive subscription movement during fiber cloning and keyed/non-keyed reconciliation so region-scoped atom subscribers move from stale fibers to live fibers without duplicate or leaked registrations.
- Hardened subscriber scheduling to prefer granular updates under fine-grained ancestors, map stale subscribers back to live alternates when possible, and ignore detached stale subscribers once a mounted tree exists.
- Added bounded guardrails and coverage for runtime2 patch-stream and DOM-commit operation volume, plus keyed-move sibling-count limits to fail fast before runaway diff/commit work.
- Added shard-session concurrency and ordering hardening: mutex-protected queue access, stale-port inbound rejection, queue-overflow warning throttling, delayed patch-ready payload handling, and contextual payload-send errors.
- Reworked `interop.WorkerPool` replacement flow to use slot-based worker ownership, asynchronous repair, close-time repair cancellation, and surfaced replacement failure context on later requests.
- Extended `ui.ParallelRegionSpec` with `SchedulerShardIDs` so apps can declare custom shard pools and remount regions when shard topology changes.
- Optimized runtime2 prop-serializability validation with a fast success path (`isSerializableValueFast(...)`) and `MapRange` iteration so normal serializable props avoid expensive path-building while unsupported shapes still produce detailed failure errors.
- Added wasm regression coverage for refresh-only `RenderInto(...)` updates with sibling `ui.ParallelRegion(...)` shells to ensure shell DOM nodes are reused in place and no child-list churn operations are emitted.
- Added runtime2 hotspot benchmarks for canonical tree parse, patch identity hashing, and deferred host-region dispatch paths, plus render-node sibling-key validation and snapshot fingerprint hash helper tuning.
- Revised `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers` to precompute region IDs/specs and reuse per-region props maps so benchmark output reflects runtime2 dispatch behavior rather than harness-only `fmt.Sprintf` and transient map allocations.
- Added focused runtime2 benchmark coverage in `internal/runtime2/snapshot_hash_bench_test.go` for snapshot identity (`GetSnapshotFingerprintHash(...)` vs `GetSnapshotFingerprint(...)`) and serializable-props validation fast-path behavior (valid fast path vs invalid fallback path).
- Optimized runtime2 host snapshot/update dispatch hot paths by removing redundant envelope validation work on already-normalized internal paths and by using trusted snapshot-hash helpers inside adapter-internal no-change checks.
- Added scheduler update-job coalescing for queued same-region updates at the same cancel generation, plus `TestHandleSchedulerUpdateCoalescesQueuedRegionUpdates` coverage to lock in the queue-depth contract.
- Re-validated snapshot fingerprint hashing under runtime2 hot-path pressure and kept the simpler `json.Marshal(...)` + `sha256.Sum256(...)` path in `internal/runtime2/snapshot.go` after an A/B check showed lower allocation counts than the streaming-hasher variant for the dispatch-heavy benches.
- Reduced runtime2 binary snapshot transport overhead in `internal/runtime2/binary_snapshot_body.go`, `internal/runtime2/binary_source_value.go`, `internal/runtime2/binary_source_id_table.go`, and `internal/runtime2/binary_offset.go` by removing duplicate props validation during body encode, adding non-reflect `[]any`/`map[string]any` source-value fast paths, replacing parse-time source-ID re-normalization with direct canonical validation, removing per-parse string-concat field labels, replacing value-typed `sync.Pool` key-buffer entries with pointer-backed cache entries to avoid interface boxing allocations, and appending source-id/source-values sections directly into the final snapshot buffer with in-place length backfill.
- Runtime2 binary snapshot microbench deltas from `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildAndParseBinarySnapshotEnvelope" -benchmem`:
  - `BenchmarkBuildAndParseBinarySnapshotEnvelope/build`: `2928 B/op -> 1625 B/op`, `67 -> 29 allocs/op`.
  - `BenchmarkBuildAndParseBinarySnapshotEnvelope/parse`: `2266 B/op -> 1976 B/op`, `51 -> 36 allocs/op`.
- Added focused guard benchmarks for the new binary transport hot paths in `internal/runtime2/binary_source_value_bench_test.go`:
  - `BenchmarkBuildBinarySourceValueAnyMapFastPath`
  - `BenchmarkParseBinarySourceIDTableCanonical`
- Runtime2 hash-path A/B microbench deltas from `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers|BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred|BenchmarkHandleHostRegionSnapshotFingerprint" -benchmem`:
  - `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers`: about `335 -> 287 allocs/op` (`~10.0 KB/op -> ~11.9 KB/op`).
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/no-change`: `6 -> 5 allocs/op` (`200 B/op -> 256 B/op`).
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/changed`: `12 -> 11 allocs/op` (`705 B/op -> 776 B/op`).
- Runtime2 focused bench deltas after these changes (same host machine, `go test ./internal/runtime2 -run ^$ -bench ... -benchmem`):
  - `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers`: about `671 -> 575 allocs/op` and `~34 KB/op -> ~19 KB/op`.
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/no-change`: `13 -> 11 allocs/op` and `424 B/op -> 392 B/op`.
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/changed`: `25 -> 21 allocs/op` and `1024 B/op -> 960 B/op`.
- Reduced canonical tree parse overhead in `internal/runtime2/render_node_table.go` by pre-sizing node-record storage, switching child-span overlap ownership tracking from a map to a slice, and avoiding keyed-sibling hash-map setup when sibling sets contain zero or one keyed child.
- Added no-validation internal helpers for snapshot envelope/hash hot paths in `internal/runtime2/snapshot.go` and updated host no-change dispatch fingerprint handling in `internal/runtime2/host_region_adapter.go` to reuse cached digest state on unchanged payloads while preserving public validation contracts.
- Tightened runtime2 host snapshot capture again by reusing prevalidated source snapshots when building update envelopes and by skipping coordinator source-ID rewrites when the mounted source list is unchanged.
- Focused runtime2 snapshot-capture deltas from `go test ./internal/runtime2 -run ^$ -bench "BenchmarkHandleHostRegionUpdateSnapshot|BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred|BenchmarkHandleHostRegionSnapshotFingerprint" -benchmem`:
  - `BenchmarkHandleHostRegionUpdateSnapshot`: `1021 ns/op, 1552 B/op, 12 allocs/op -> 749 ns/op, 1184 B/op, 8 allocs/op`.
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/no-change`: `985 ns/op, 240 B/op, 22 allocs/op -> 789 ns/op, 256 B/op, 5 allocs/op`.
  - `BenchmarkHandleHostRegionSnapshotFingerprint/no-change`: `630 ns/op, 240 B/op, 22 allocs/op -> 562 ns/op, 256 B/op, 5 allocs/op`.
- Cleared host snapshot hash cache state alongside fingerprint-string resets during control restart handling in `internal/runtime2/host_control_dispatcher.go` so restart/remount flows cannot reuse stale no-change detection state.
- Reduced transient allocations in runtime2 identity and binary framing paths by switching patch identity formatting to `strconv.FormatUint(...)` and replacing string-based magic checks with byte-wise comparisons in `internal/runtime2/binary_patch_transport.go`, `internal/runtime2/binary_header.go`, and `internal/runtime2/shared_snapshot_header.go`.
- Added direct `BenchmarkHandleHostRegionSnapshotFingerprint` no-change/changed coverage in `internal/runtime2/perf_hotspot_bench_test.go` to track adapter-level fingerprint fast-path behavior separately from dispatch-level benchmarks.
- Optimized runtime2 patch identity hashing in `BuildPatchStreamIdentity(...)` by streaming canonical JSON directly into the FNV hasher (`buildJSONHashDigest(...)`) instead of allocating a full marshaled payload buffer.
- Added `BenchmarkBuildPatchStreamIdentityCurrentVsLegacy` in `internal/runtime2/perf_identity_compare_bench_test.go` so patch identity hashing can be measured against the legacy marshal-based implementation in one benchmark run.
- Runtime2 patch-identity benchmark deltas from `go test ./internal/runtime2 -run ^$ -bench BenchmarkBuildPatchStreamIdentity -benchmem`:
  - `BenchmarkBuildPatchStreamIdentity/small`: median `1566 ns/op -> 1060 ns/op` and `521 B/op -> 160 B/op`.
  - `BenchmarkBuildPatchStreamIdentity/large-keyed-rotate`: median `28533 ns/op -> 21252 ns/op` and `8365 B/op -> 160 B/op`.
- Reduced runtime2 patch-identity overhead again by pooling FNV hashers in `BuildPatchStreamIdentity(...)` via `sync.Pool` (`internal/runtime2/patch_stream.go`), removing one hot-path allocation per identity computation.
- Runtime2 patch-identity follow-up benchmark deltas from `go test ./internal/runtime2 -run ^$ -bench "BenchmarkBuildPatchStreamIdentity/(small|large-keyed-rotate)$" -benchmem -benchtime=700ms -count=5`:
  - `BenchmarkBuildPatchStreamIdentity/small`: median `1300 ns/op -> 1273 ns/op`, `160 B/op -> 152 B/op`, `4 -> 3 allocs/op`.
  - `BenchmarkBuildPatchStreamIdentity/large-keyed-rotate`: median `23618 ns/op -> 20409 ns/op`, `160 B/op -> 152 B/op`, `4 -> 3 allocs/op`.
- Optimized runtime2 host snapshot fingerprinting in `internal/runtime2/snapshot.go` and `internal/runtime2/host_region_adapter.go` by routing hot internal paths through the no-validation hash helper while preserving the existing marshal-based digest output contract, with compatibility coverage in `internal/runtime2/snapshot_hash_compare_test.go`.
- Added focused snapshot-hash compare benchmark coverage in `internal/runtime2/perf_snapshot_hash_compare_bench_test.go` (current path vs legacy marshal path) to keep this hot path measurable as runtime2 evolves.
- Runtime2 host-dispatch allocation deltas from `benchstat` (`pre-opt-pass2` vs `post-opt-pass2-final`):
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/no-change`: `22 -> 5 allocs/op` (`-77.27%`).
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/changed`: `29 -> 11 allocs/op` (`-62.07%`).
  - `BenchmarkHandleHostRegionSnapshotFingerprint/no-change`: `22 -> 5 allocs/op` (`-77.27%`).
  - `BenchmarkHandleHostRegionSnapshotFingerprint/changed`: `30 -> 12 allocs/op` (`-60.00%`).
  - `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers`: `1055 -> 287 allocs/op` (`-72.80%`) and `12.00 KiB/op -> 11.63 KiB/op` (`-3.00%`).
- Saved this pass's reproducible benchmark artifacts under `bin/test-results/runtime2-bench/` (`pre-opt-pass2*.txt`, `post-opt-pass2*.txt`, and host/snapshot focused compare runs).
- Added canonical decode quick paths in `internal/runtime2/render_ir.go` and `internal/runtime2/render_node_table.go`: flat child-ID pooling, precomputed parent/child spans, direct raw-prop map decode (skip sort/copy path), and keyed-sibling early exits when duplicate checks are unnecessary.
- Runtime2 hot-path microbench deltas (`go test ./internal/runtime2 -run ^$ -bench "^(BenchmarkBuildCanonicalPatchStream|BenchmarkParsePatchStreamTransaction|BenchmarkCommitRegionPatchTransaction|BenchmarkParseCanonicalRenderTree|BenchmarkBuildPatchStreamIdentity|BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred)$" -benchmem -count 5`):
  - Geomean: `-29.81% sec/op`, `-48.91% B/op`, `-27.75% allocs/op`.
  - `BenchmarkParseCanonicalRenderTree`: `38.70us -> 21.69us`, `60.92KiB -> 59.10KiB`, `92 -> 19 allocs/op`.
  - `BenchmarkCommitRegionPatchTransaction`: `5.799us -> 4.185us`.
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/no-change`: `1.449us -> 1.050us`, `11 -> 7 allocs/op`.
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/changed`: `2.315us -> 1.387us`, `23 -> 13 allocs/op`.

- Added another runtime2 perf pass with conservative hot-path optimizations in `spec.go`, `snapshot.go`, `host_region_adapter.go`, and `render_node_table.go`:
  - Added `isSerializableAnyFast(...)` to short-circuit common `map[string]any` / `[]any` payload validation before reflective traversal.
  - Added `buildSnapshotEnvelopeFromNormalizedSourceIDsWithoutValidation(...)` and used it on already-normalized host update paths to avoid duplicate envelope validation work.
  - Added sequential-node-ID and small-keyed-sibling duplicate-check fast paths in `ParseRenderNodeTable(...)` to cut map allocation pressure in canonical decode.
- Runtime2 full-bench deltas from this pass (`go test ./internal/runtime2 -run ^$ -bench . -benchmem -benchtime=200ms`):
  - `BenchmarkParseCanonicalRenderTree`: `28178 ns/op, 65424 B/op, 22 allocs/op -> 23939 ns/op, 55616 B/op, 16 allocs/op`.
  - `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers`: `67132 ns/op, 14969 B/op, 476 allocs/op -> 55362 ns/op, 11894 B/op, 285 allocs/op`.
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/no-change`: `1077 ns/op, 320 B/op, 9 allocs/op -> 913.9 ns/op, 256 B/op, 5 allocs/op`.
  - `BenchmarkHandleHostRegionUpdateDispatchWithPriorityDeferred/changed`: `1608 ns/op, 904 B/op, 19 allocs/op -> 1249 ns/op, 776 B/op, 10 allocs/op`.
  - `BenchmarkHandleHostRegionSnapshotFingerprint/no-change`: `795.2 ns/op, 288 B/op, 7 allocs/op -> 653.1 ns/op, 256 B/op, 5 allocs/op`.
  - `BenchmarkHandleHostRegionSnapshotFingerprint/changed`: `1218 ns/op, 872 B/op, 16 allocs/op -> 1120 ns/op, 808 B/op, 12 allocs/op`.
  - `BenchmarkHandleHostRegionUpdateSnapshot`: `1408 ns/op, 1616 B/op, 16 allocs/op -> 1267 ns/op, 1552 B/op, 12 allocs/op`.
- Removed duplicated sibling-scan work in canonical patch planning by caching per-parent canonical sibling indexes and switching canonical move simulation to in-place slice updates, so hot-path anchor lookup and move planning avoid repeated linear rescans.
- Focused runtime2 microbench deltas for this pass (`go test ./internal/runtime2 -run ^$ -bench "^(BenchmarkBuildCanonicalPatchStream|BenchmarkCommitRegionPatchTransaction|BenchmarkParsePatchStreamTransaction)$" -benchmem -count 5`):
  - Geomean sec/op: about `-1.59%`; allocations unchanged across the measured benches.
  - `BenchmarkBuildCanonicalPatchStream`: `5.069 us -> 4.979 us`.
  - `BenchmarkCommitRegionPatchTransaction`: `3.636 us -> 3.604 us`.
  - `BenchmarkParsePatchStreamTransaction`: `522.0 ns -> 511.0 ns`.
- Removed duplicated patch-transaction parse work in `ParsePatchStreamTransaction(...)` by dropping the redundant pre-pass patch-order walk and by lazily cloning sibling/known-node/removal tracking maps only for op kinds that mutate or require those structures.
- Runtime2 microbench deltas for this parse-lazy-allocation pass (`go test ./internal/runtime2 -run ^$ -bench "^(BenchmarkParsePatchStreamTransaction|BenchmarkCommitRegionPatchTransaction|BenchmarkBuildCanonicalPatchStream)$" -benchmem -count 8`):
  - `BenchmarkParsePatchStreamTransaction`: `768.6 ns -> 530.1 ns`, `1040 B/op -> 976 B/op`, `6 -> 5 allocs/op`.
  - `BenchmarkCommitRegionPatchTransaction`: `5.549 us -> 4.458 us`, `6864 B/op -> 6800 B/op`, `53 -> 52 allocs/op`.
  - `BenchmarkBuildCanonicalPatchStream` was also sampled in the same run; this pass did not directly modify canonical diff generation.
- Removed unneeded sibling-bound lookup work on host patch commit paths that do not include keyed-move ops by reusing `BuildKnownNodeIDsForRegionDOMIndex(...)` and only building sibling counts when `parseHasPatchKeyedMoveOp(...)` is true.
- Runtime2 keyed-move-guard microbench deltas (`go test ./internal/runtime2 -run ^$ -bench "^(BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers|BenchmarkParsePatchStreamTransaction|BenchmarkCommitRegionPatchTransaction)$" -benchmem -count 5`):
  - `BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers`: `56.97 us -> 50.72 us` (`-10.97% sec/op`).
  - `BenchmarkParsePatchStreamTransaction` and `BenchmarkCommitRegionPatchTransaction`: no statistically significant regression in this sample set.
- Removed duplicated/unneeded keyed-move prep work in runtime2 patch paths by deferring keyed-move-op scans until sibling bounds are actually needed in `ParsePatchStreamTransaction(...)`, and by only storing sibling-count entries for parent nodes that currently have at least one child in `BuildRegionDOMPatchLookupMaps(...)` and `BuildSiblingCountByParentForRegionDOMIndex(...)`.
- Runtime2 sibling-count map guard microbench deltas (`go test ./internal/runtime2 -run ^$ -bench "^(BenchmarkParsePatchStreamTransaction|BenchmarkCommitRegionPatchTransaction)$" -benchmem -count 8`):
  - `BenchmarkParsePatchStreamTransaction`: `566.1 ns -> 442.4 ns` (`-21.84% sec/op`).
  - `BenchmarkCommitRegionPatchTransaction`: `3.881 us -> 3.613 us` (directional improvement, not statistically significant in this sample set).
- Removed unneeded reverse string-ref map allocations on runtime2 patch-parse paths by making `ParseRenderStringTable(...)` validate + copy entries without building `storeRenderStringRefByString`, and by adding a fallback linear lookup in `GetRenderStringRef(...)` only for parsed-table call sites that actually require reverse lookup.
- Runtime2 parsed-string-table guard microbench deltas (`go test ./internal/runtime2 -run ^$ -bench "^(BenchmarkParsePatchStreamTransaction|BenchmarkCommitRegionPatchTransaction|BenchmarkBuildCanonicalPatchStream)$" -benchmem -count 8`):
  - Geomean: `-23.63% sec/op`, `-10.86% B/op`, `-16.73% allocs/op`.
  - `BenchmarkParsePatchStreamTransaction`: `490.1 ns -> 301.6 ns`, `976 B/op -> 720 B/op`, `5 -> 3 allocs/op`.
  - `BenchmarkCommitRegionPatchTransaction`: `4.572 us -> 4.047 us`, `6400 B/op -> 6144 B/op`, `53 -> 51 allocs/op`.
  - `BenchmarkBuildCanonicalPatchStream` was sampled in the same run with no code-path changes in canonical diff generation from this pass.

### Example 201 browser benchmark harness

- Added `examples/testing/render-benchmark`, including runtime1, runtime2 (single-worker), runtime2 (4-worker), and React 18 subjects with shared scenario contracts and local vendored browser assets.
- Added Playwright-Go benchmark automation (`TestExample201BrowserBenchmarkReport`) that builds wasm artifacts, executes seeded browser scenarios, and emits structured JSON + Markdown reports under `bin/test-results/example-201-browser-benchmark/`.
- Added a `content-card refresh` scenario so the browser suite now measures refresh-only behavior for both core-list and content-card views.
- Revised the browser harness to wait for DOM-settle frames after scenario preparation, reducing prepare-phase mutation spillover into the measured RT2 window and making mutation counts and timing comparisons fairer.
- Removed RT2-only benchmark DOM metadata from the rendered core/content shells so the benchmark no longer charges runtime2 for debug-only attribute churn that React and runtime1 were not paying.
- Extended the browser report schema and rendered tables with worker-preparation diagnostics for the runtime2 subjects, including batch-count deltas, last-batch duration, and prepared-item counts for the measured run window.
- Optimized the single-worker runtime2 benchmark path by matching chunk fan-out to the configured worker count, so the one-worker mode no longer pays four-request and four-region overhead for one-worker preparation batches.
- Added configurable `Runtime 2 (N Workers)` benchmark support through the Example 201 runner, subject route, and worker-count query parsing, with focused Playwright coverage for custom worker counts.
- Documented the Example 201 benchmark workflow in `examples/README.md` and `docs/PERFORMANCE.md`.
- Replaced runtime2 worker-preparation dispatch in `examples/testing/render-benchmark/workers.go` from `interop.WorkerPool` arbitration to a dedicated worker-fleet lane model (`[]interop.Worker`) with direct chunk-to-worker requests, removing queue/admission overhead on each prep batch.
- Optimized worker batch fanout by reusing one resolved `runtime2WorkScale` value per batch and passing chunk sub-slices directly (instead of per-chunk copy allocations) for core/content worker requests.
- Kept worker-mode behavior and metrics contracts stable (`metric-worker-count`, `metric-worker-batch-count`, `metric-worker-items`, `metric-worker-batch-ms`) while adding empty-fleet guard errors for clearer failure diagnostics.
- Added `docs/WORKER_POOLS_VS_LANES.md` and linked it from docs indexes so worker-heavy features have one explicit decision guide for `interop.OpenWorkerPool(...)` versus direct worker-lane fanout.
- Focused dispatch microbench sample (`go test ./interop -run ^$ -bench BenchmarkRequestWorkerDecodedFanoutDispatch -benchmem -count=3`, Windows/amd64, i7-12700) still shows direct-lane fanout faster than pooled dispatch:
  - `direct-lanes`: about `18.4-21.5 us/op`, `~3380 B/op`, `83 allocs/op`.
  - `worker-pool`: about `28.6-30.1 us/op`, `~3395 B/op`, `83 allocs/op`.
- Revalidated Example 201 integration after this dispatch change:
  - `go test ./test/playwrightgo/examples -tags playwrightgo -run TestExample201BrowserBenchmarkHonorsConfiguredWorkerCounts -count=1`
  - `go test ./test/playwrightgo/examples -tags playwrightgo -run TestExample201BrowserBenchmarkReport -count=1`
- Added explicit worker-dispatch strategy support for Example 201 runtime2 subjects via `runtime2Dispatch`:
  - `runtime2Dispatch=batch` (default): one multi-chunk request per worker lane.
  - `runtime2Dispatch=chunk`: legacy one-request-per-chunk dispatch.
- Updated `examples/testing/render-benchmark/benchmark-runner.js` to forward `runtime2WorkScale` and `runtime2Dispatch` query params into each subject iframe route, so stress and dispatch A/B routes now propagate as documented.
- Hardened Example 201 worker-prepare effect dependency tracking in `examples/testing/render-benchmark/workers.go` with comparable content-hash dependency tokens for core/content payloads; this removed stale-payload scheduling races seen during structural-churn scenarios (`core-reverse` under multi-worker runs).
- Added richer benchmark timeout diagnostics in `examples/testing/render-benchmark/benchmark-subject.js` (framework, worker status, last action, and row-order break window) so flaky scenario failures surface actionable context in CI logs.
- Added focused dispatch A/B benchmark automation in `TestExample201BrowserBenchmarkDispatchCompare` and report outputs:
  - `bin/test-results/example-201-browser-benchmark/browser-benchmark-report-rt2-dispatch-chunk.{json,md}`
  - `bin/test-results/example-201-browser-benchmark/browser-benchmark-report-rt2-dispatch-batch.{json,md}`
- Dispatch compare sample (`go test ./test/playwrightgo/examples -tags playwrightgo -run TestExample201BrowserBenchmarkDispatchCompare -count=1 -v`, Windows/amd64, i7-12700):
  - Worker-batch mean across worker-metric scenarios: `12.973 ms -> 10.277 ms` (`+20.78%` faster in `batch` mode).
  - DOM-ready mean across worker-metric scenarios: `27.828 ms -> 25.101 ms` (`+9.80%` faster in this sample).
- Added a runtime2 local core fast path in `examples/testing/render-benchmark/workers.go` for small core batches (`<= 64` items): when enabled, this path bypasses worker RPC and uses one local cache-backed prepared-item map to reduce request/listener/structured-clone overhead on update-heavy small-list scenarios.
- Added a public query toggle `runtime2CoreFastPath` (`on` by default, `off` to force worker RPC) and forwarded it through `examples/testing/render-benchmark/benchmark-runner.js` so route-level A/B checks stay reproducible.
- Added an inline runtime2 core-update fast path in `examples/testing/render-benchmark/main.go` so small core updates that qualify for `runtime2CoreFastPath` prepare and store core chunks in the click-event path (with cache + generation updates) before the effect fallback runs; this removes extra effect-cycle latency on `Core Update`.
- Hardened `examples/testing/render-benchmark/benchmark-runner.js` iframe subject loading/inspection paths with `SecurityError` guards around cross-window property reads, preventing benchmark aborts on transient cross-origin window access during Playwright runs.
- Focused Core Update sample on the mixed-framework report route (`go test ./test/playwrightgo/examples -tags playwrightgo -run TestExample201BrowserBenchmarkReport -count=1`):
  - `RT2x1`: `8.500 ms -> 4.271 ms` (`-49.75%` DOM-ready).
  - `RT2x2`: `9.143 ms -> 3.714 ms` (`-59.38%` DOM-ready).
  - `RT2x4`: `10.429 ms -> 3.714 ms` (`-64.39%` DOM-ready).
  - `RT2x8`: `10.500 ms -> 3.500 ms` (`-66.67%` DOM-ready).
- Repeated direct runner A/B sample (`/examples/testing/render-benchmark/?iterations=11&warmups=2&seed=20101&runtime2WorkerCounts=1,2,4,8`, `runtime2CoreFastPath=on` vs `off`) confirms additional `Core Update` wins with the inline path:
  - `RT2x1`: `5.764 ms -> 4.173 ms` (`-27.60%` vs fast-path off).
  - `RT2x2`: `4.873 ms -> 3.945 ms` (`-19.04%` vs fast-path off).
  - `RT2x4`: `4.182 ms -> 2.809 ms` (`-32.83%` vs fast-path off).
  - `RT2x8`: `5.127 ms -> 3.936 ms` (`-23.23%` vs fast-path off).

### Parallel-region runtime status surface and diagnostics docs

- Added a public read-only `ui.GetParallelRegionRuntimeStatus(...)` helper and `ui.ParallelRegionStatus` shape so apps and tooling can inspect one tracked region's ownership mode, shard assignment, epoch, hydration flags, snapshot and dispatch and commit versions, transport tier, stale counters, and fallback reason without touching mutable runtime2 internals.
- Added focused regression coverage in `ui/parallel_region_test.go` and `ui/ui_wasm_test.go` for public status validation, missing-region behavior, dispatch-version reporting, and hydrated attach-state reporting.
- Added `examples/200-runtime2-status` plus `examples/README.md` indexing so adopters can inspect the public runtime-status contract in a dedicated tooling-style panel.
- Updated `examples/testing/parallel-region-basic`, `examples/109-parallel-region-grid`, and `examples/110-parallel-region-diagnostics` copy and diagnostics surfacing to reflect current local-first shell ownership with active runtime2 dispatch and operator metrics.
- Updated parallel-region docs in `docs/PARALLEL_REGION_AUTHORING.md`, `docs/PARALLEL_REGION_TROUBLESHOOTING.md`, and `docs/REFERENCE_MAP.md` with transition semantics and operator-facing runtime-status field guidance, and marked completed multithreaded-runtime follow-ups in `docs/MULTITHREADED_RUNTIME_TODO.md`.

### GoGRPCBridge submodule hardening and release alignment

- Advanced `third_party/GoGRPCBridge` to `1b39bbc` with grouped security hardening, abuse controls, observability hooks, release workflow guardrails, generated docs-page sync, and a checked-in docs catalog manifest.
- Updated repo-level GoGRPCBridge governance and rollout docs (`docs/GOGRPCBRIDGE_DEV_TO_PROD_ROADMAP.md`, `docs/GOGRPCBRIDGE_REQUIRED_CHECKS.md`, `docs/GOGRPCBRIDGE_SUBMODULE_LIFECYCLE.md`, `docs/GOGRPCBRIDGE_INTEGRATION_MATRIX.md`, `docs/REFERENCE_MAP.md`, `docs/README.md`, and `docs/TODO.md`) to reflect current required checks and lifecycle expectations.
- Hardened CI and release workflow wiring for the GoGRPCBridge integration path in `.github/workflows/gogrpcbridge-ci.yml` and `.github/workflows/release.yml`.

### Example 100 and coverage expansion

- Expanded `examples/server/ai-chat-wizard` build, seed, server, and tunnel surfaces with follow-up fixes and targeted regression coverage.
- Added focused additional coverage suites across examples, runtime, UI, browser/render helpers, and `tools/gwc` command paths to close branch and helper gaps.
- Added `.github/CODEOWNERS` plus shared bootstrap/test artifacts in `scripts/bootstrap-gogrpcbridge.ps1` and `third_party/_shared/data/todos.json`.

### Runtime helper consistency updates

- Added multithreaded runtime follow-up documentation in `docs/MULTITHREADED_RUNTIME.md` and `docs/MULTITHREADED_RUNTIME_TODO.md`.
- Applied helper-path consistency updates in `interop`, `i18n`, and UI form native/non-native paths, with corresponding module dependency updates in `go.mod` and `go.sum`.

## 2026-03-26

### Worker multithreading primitives and lifecycle hardening

- Added first-class worker coordination primitives in `interop`, including `MessageChannel`/`MessagePort` support, transferred-port posting helpers, `SharedBuffer` shared-memory APIs, `Atomics` wait/notify wrappers, and nested structured transport for shared buffers and binary payloads.
- Added `OpenWorkerPool(...)` with bounded queueing, graceful drain/close behavior, typed request routing through the existing decoded helpers, and worker replacement or fail-fast shutdown when pooled workers are unexpectedly disposed.
- Hardened worker lifecycle handling across `interop` and `ui`, including bad-message failure paths, duplicate worker-start protection, stale cancel cleanup, and explicit worker subscription teardown in the Example 100 chat runtime.
- Expanded native and js/wasm worker coverage with focused regression tests for worker-to-worker port communication, shared-memory coordination, pool scheduling and repair, nested payload transport, and `UseWorkerTask(...)` terminal-state handling.

### Launcher lint workflow documentation

- Documented the repo-local `gwc lint` / `gwc review` workflow in `docs/TESTING.md`, including saved-report commands, auto-install behavior for the default `golangci-lint` binary, and the non-zero exit contract when findings are present.

### Lint cleanup and runtime pool guardrails

- Cleared outstanding lint findings across runtime, interop, launcher, i18n, and UI test surfaces, including staticcheck, gosimple, govet, ineffassign, and unused diagnostics reported through `gwc lint`.
- Fixed runtime scratch-slice pool usage patterns that triggered `SA6002` by switching reconciler scratch pools to pointer-backed typed wrappers with dedicated `get`/`clear` helpers.
- Updated focused runtime lint coverage paths so reconciler branch tests and scratch-pool call sites now use the guarded helper APIs instead of raw `sync.Pool.Put` with slice values.

### Example 100 API migration and lint recovery

- Migrated `examples/server/ai-chat-wizard` to the current launcher/runtime APIs, including updated client state and helper field names, background worker interop calls, gRPC client/test method names, and markdown/sql helper callsites.
- Restored Example 100 build and verification flow so both client wasm and server packages compile and `gwc verify` passes for the example root.
- Cleared launcher lint findings for Example 100 by fixing unused-path drift, staticcheck callback assertions, errcheck cleanup in store/test helpers, and removing dead legacy helpers.

## 2026-03-25

### Launcher lint runbook and Playwright deprecation follow-up

- Documented the launcher-owned `gwc lint` / `gwc review` workflow in `docs/TESTING.md`, including saved-report examples, auto-install behavior for the default `golangci-lint` binary, and the expected non-zero exit contract when findings are present.
- Updated the release startup Playwright probe in `tools/gwc/main.go` to use locator-based click interaction instead of the deprecated page-level `Click(...)` API.

### Starter scaffold output refresh

- Refreshed `gwc start` scaffold generation and the checked-in starter goldens, including updated generated `main.go` and starter test fixtures for standalone, contributor-linked, SSR, and browser-test starter variants.

### Naming-sweep compile and launcher follow-up

- Restored missed helper and field references across `router`, `tools/gwc`, and selected runtime tests after the naming and GoDoc sweep so the router, UI, and launcher packages compile again under the repo-local test cache.
- Fixed managed example-server runtime state and log artifacts to honor `gwc-runner.json` `paths.artifactRoot` directly instead of inserting the workspace artifact namespace.

### Adoption maturity, shell-routing, and enterprise guidance

- Added `docs/ADOPTION_MATURITY.md` to define feature-area readiness tiers, starter support tiers and cadence, ecosystem maintenance signals, enterprise evaluation packet expectations, and guided learning/community-growth lanes.
- Added `docs/CLIENT_SHELL_ROUTING.md` to document the recommended single-runtime route-tree pattern for mixed marketing, auth, workspace, and canvas flows inside one GoWebComponents client shell.
- Added `docs/TEAM_CONVENTIONS.md` and `docs/ENTERPRISE_PILOT.md` to capture team-scale architecture conventions, review and migration checklists, component-library workflow guidance, enterprise pilot gates, incident runbook expectations, and deployment validation checklists.
- Updated `docs/README.md` and `docs/TODO.md` to index the new guidance and mark the related adoption, shell-routing, team-conventions, and enterprise-pilot roadmap items complete.

### Example 100 marketing heading consolidation

- Shared the Example 100 landing and pricing top-bar shell, brand block, and CTA actions through common marketing-page helpers so those page headers no longer drift visually.
- Shared the Example 100 landing and pricing hero heading treatment so the eyebrow, headline, body copy, and CTA row stay aligned across both pages.

### Launcher lifecycle, profiling, and testkit expansion follow-up

- Added first-class launcher lifecycle and delivery commands: `gwc init`, `gwc inspect`, `gwc upgrade`, `gwc migrate`, `gwc prerender`/`gwc export`, and `gwc deploy`, with associated tests and runner-config/dev-runtime integration updates.
- Aligned generated/imported wasm output paths to `bin/main.wasm` across scaffold templates, CI golden fixtures, and import-generated HTML startup scripts.
- Expanded public test surfaces with richer `testkit/render`, `testkit/router`, and `testkit/ssr` helpers plus a new `test/browser` coordination harness, including render-count and warning assertion utilities for wasm fixtures.
- Added runtime startup-cost attribution and route startup budget reporting (runtime + devtools), with new regression fixtures for small, routed mid-sized, and production-shaped profiling budgets.
- Added `examples/107-server-interactive-poc`, refined `examples/server/ai-chat-wizard` shell and bootloader phase handling, and updated related regression tests.
- Extended GoDoc/API-audit coverage across diagnostics, html/i18n/interop/pwa/state/ui helper surfaces, and checked in function inventory/audit reports used by the naming review workflow.

### Grouped cleanup of legacy scripts and fixtures

- Removed deprecated example shell and PowerShell helper scripts from `examples/` and Atlas docs script scaffolding paths.
- Removed legacy Node and JSX helper artifacts, including runner path scripts, dev-server Node entrypoints, test-build script wrappers, import fixtures, and stale docs/example placeholders.

### Agent guidance refresh for gwc usage

- Simplified `AGENTS.md` into a terse operator format and added a compact `gwc` command map with effective feature coverage, key flags, primary docs links, and practical command examples.

### GWC launcher expansion, browser-suite migration, and Example 100 UI refresh

- Added launcher-owned wasm experiment helpers under `gwc wasm` (`measure`, `compare`, `compare-compression`, `compare-cache`, `compare-toolchain`) plus `gwc bench` compare/capture follow-ups and updated launcher docs around the expanded benchmark and wasm workflow.
- Added `gwc env` so operator-facing environment variables can be inspected in text or JSON output with secret-shaped values redacted by default.
- Documented `gwc env` in both `docs/GWC.md` and `tools/README.md`, including JSON output and redaction behavior guidance.
- Migrated remaining browser-suite and workspace wiring from npm/TypeScript Playwright configs to launcher-owned Playwright-Go flows, removed deprecated npm test workspaces and legacy spec/config files, and updated docs/workflow references to match.
- Refreshed `examples/server/ai-chat-wizard` landing/auth/pricing surfaces, updated chat-shell bootstrap copy and related server tests, and rebuilt shared Tailwind output through the launcher-owned CSS pipeline.
- Fixed Example 100 landing-header navigation actions to use app-route-aware links for home/pricing/login/chat entry behavior.
- Removed deprecated `tools/*.ps1` and `tools/*.sh` wrapper scripts now superseded by the launcher-owned `gwc` command surface.
- Removed legacy `test/browser/index.ts` and `test/browser/index.test.ts` bridge files that were tied to the older browser-harness path.
- Removed checked-in generated browser helper/service-worker/static-script artifacts from examples static/script paths that are now treated as generated runtime assets.

### Tailwind CLI first-class launcher workflow

- Added a first-class `gwc tailwind` command in `tools/gwc` that regenerates the shared Tailwind manifest and stylesheet without npm, with machine-readable `-json` output support.
- Added standalone Tailwind CLI download and cache management under `third_party/tailwindcss/bin/<version>/`, including checksum verification against the upstream `sha256sums.txt` release artifact when available.
- Updated `examples/server/ai-chat-wizard/cmd/build-client` to run `go run ./tools/gwc tailwind` before wasm compilation so Example 100 no longer depends on npm for CSS rebuilds.

### GWC benchmark scoring and launcher decomposition follow-up

- Extended `gwc bench` with package-level parallelism, checked-in `docs/benchmarks/reference.json` baselines, and reference-normalized geometric bucket scores so benchmark runs keep their raw `ns/op`, `B/op`, and `allocs/op` data while also emitting a stable summary score for regression tracking.
- Split the oversized `tools/gwc/main.go` launcher file into command-specific modules such as `dev.go`, `doctor.go`, `examples.go`, and `start.go`, and added `tools/gwc/docs/README.md` so future launcher work has a clearer ownership map.
- Aligned the launcher browser lane with the checked-in Playwright-Go package layout so browser test workspace discovery and the focused `tools/gwc` tests continue to match the repo-owned runner contract.

### Model picker persistence hardening in Example 100

- Fixed `examples/server/ai-chat-wizard` model selection persistence so new chats no longer land in a blank provider/model state after reconnects or stale cached preferences.
- Added client-side recovery for invalid or blank persisted model selections, with deterministic fallback to the first available catalog model and persisted repair back to the server.
- Added server-side `GetSelectedModel` fallback repair and persistence so normalized model preferences stay stable across sessions.
- Added focused regression coverage for selection-repair behavior in both client wasm tests and server RPC tests.

### Example 100 speech synthesis fallback and opt-in flow

- Added an OpenAI TTS-only fallback path in `examples/server/ai-chat-wizard` so speech playback can be enabled without switching the active chat provider or selected model for normal text generation.
- Added client-side helper coverage for selecting a speech-capable OpenAI synthesis model, plus UI copy and modal-flow updates that describe the new opt-in behavior explicitly.
- Removed the older model-preferences speech-provider switch path and centralized speech-model resolution inside the TTS controller instead of mutating the user's saved chat provider preference.

### Playwright-Go rollout and Example 100 runtime hardening

- Replaced the repo and examples browser npm entrypoints, release smoke steps, and compatibility workflow with Go-based `playwrightgo` suites, and updated the browser-support and runner-config docs to match the new browser-workspace contract.
- Hardened `examples/server/ai-chat-wizard` memory extraction by sending a strict OpenAI JSON-schema response request, adding lifecycle and save-failure logging around extraction, and tightening provider HTTP coverage for the structured payload.
- Exposed a configurable usage-premium percentage in the Example 100 chat bootstrap script so client-side surfaces can read the server-defined premium multiplier during startup.
- Added account-level cost aggregation in the Example 100 client so the composer can show premium-adjusted account totals with coverage-gap handling on top of the existing per-thread cost summary.

### Tooling, devtools, and testing expansion wave

- Expanded the `gwc` launcher with dashboard/start TUI work, broader release/build/import/start/verify coverage, extracted `tools/runnerconfig` docs and tests, and new starter golden fixtures for standalone, contributor-linked, browser-test, and SSR app templates.
- Expanded `devtools` with bug-capture, support-bundle, trace-capture, serialization-boundary, extension-section, and richer runtime snapshot plumbing, backed by substantial wasm/native test coverage.
- Broadened Example 100, Atlas Commerce OS, SSR, interop, plugin, PWA, router, UI, and runtime coverage with a large set of focused regression tests, plus new public `test/browser` helpers and additional `testkit` consumer fixtures.
- Added and documented new example coverage for static islands, staged rollout config, SSR route helpers, virtualization, and the repo examples site while refreshing ecosystem, IDE, testing, and runner-config docs.
- Removed checked-in generated coverage/build artifacts from the tree and tightened ignore rules so local coverage output no longer pollutes status or release-oriented workflows.

### API naming and launcher workflow consolidation

- Aligned public API naming across `ui`, `utils`, `fetch`, `hotreload`, `logging`, and `devtools`, including test and example call-site updates and refreshed `docs/PUBLIC_API_CONVENTIONS.md` guidance.
- Extended launcher-owned tooling with stronger `runnerconfig` coverage and a new `gwc files` surface for filtered project file inventory output with repeatable extension and directory filters.

### Bench coverage and benchmark reporting

- Added micro-benchmark coverage across core and companion packages (including `devtools`, `diagnostics`, `fetch`, `head`, `hotreload`, `html`, `i18n`, `interop`, `logging`, `plugin`, `prerender`, `pwa`, `router`, `state`, `ui`, `utils`, `virtualization`, `testkit`, and `tools/runnerconfig`) to keep benchmark regressions visible in routine development.
- Added `gwc bench` to discover benchmark-bearing packages, run both native and `js/wasm` lanes, and write machine-readable benchmark snapshots to `docs/benchmarks/latest.json` with baseline-to-current delta comparison support.
- Documented the launcher benchmark workflow in `docs/PERFORMANCE.md` and `tools/README.md`, including lane selection and JSON output behavior for automation and CI.

### Example Playwright and chat-wizard follow-ups

- Relocated example-specific Playwright configs under `test/playwright/examples`, updated `examples/package.json` scripts, and refreshed docs/manual-smoke references so example test entrypoints resolve from one centralized config location.
- Refined Example 100 chat-wizard desktop UI spacing and control styling (composer controls, panel density, sidebar row emphasis, and message-meta contrast) for cleaner day-to-day operator ergonomics.

### Optional Playwright-Go smoke lane

- Added a build-tagged `test/playwrightgo` Chromium smoke test that validates the `playwright-go` driver/browser lifecycle and a minimal page interaction path.
- Added the required Playwright-Go module dependencies so the smoke lane can be enabled explicitly without changing default repo test behavior.

## 2026-03-24

### Example 100 AI chat wizard and supporting tooling

- Added a new `examples/server/ai-chat-wizard` full-stack showcase with a Go `js/wasm` client, Go server, gRPC-over-WebSocket bridge, SQLite-backed auth and conversation persistence, per-user settings and memories, streamed assistant replies, worker-backed markdown rendering, and focused Playwright/manual test support.
- Added richer chat-product behavior across the example, including reconnect and idle-resume handling for the gRPC bridge, collapsible assistant thinking sections, per-message and per-thread token-cost display, Mermaid and KaTeX rendering in message bubbles, TTS playback wiring, smoother streaming scroll behavior, and a stronger wasm/bootstrap loading shell.
- Added a canvas-style code workspace path for previewable assistant artifacts with split chat/canvas layout, canvas session state, focus-region editing, patch history, console capture, overlay mode, and a dedicated canvas-only route.
- Moved the example build output and runtime data into `examples/server/ai-chat-wizard/bin/`, added dedicated build/run scripts, runtime SQL-file loading, local Tailwind build support for the example shell, and follow-up cleanup so legacy source-tree wasm outputs are removed instead of accumulating under `client/`.

### Browser interop worker and scroll primitives

- Expanded `interop` with safer JavaScript exception wrapping for `Value.Invoke(...)` and `Value.Call(...)`, preserving thrown browser errors as structured remote failures instead of surfacing vague panics or illegal-invocation behavior.
- Added explicit element scroll helpers through `Element.SetScrollTop(...)` and `Element.ScrollMetrics(...)` so apps can drive scroll containers directly without ad hoc `syscall/js` reads and writes.
- Added `NewGoWASMWorker(...)`, `CurrentWorkerScope()`, and `WorkerScope` helpers so Go-authored wasm workers can be bootstrapped from `wasm_exec.js` plus a wasm module URL while still using the existing worker message envelope and request/progress/result pattern.
- Added native/wasm coverage around worker bootstrapping, JS exception wrapping, receiver-bound method calls, and browser storage method invocation so the new interop helpers stay consistent across runtime and tests.

### Smaller framework and docs follow-ups

- Added `OnMouseUp(...)` support to the public `html` props surface and the shorthand/sugar helpers, with tests proving the prop is emitted and wrapped consistently with the other event helpers.
- Fixed wrapped panic unwrapping for uncomparable payloads so reported panic wrappers preserve original map-like payloads instead of losing the underlying deferred/runtime failure context during repeated wrapping.
- Added the `28-scaling-local-state-with-use-reducer` example and supporting docs so reducer-based local-state composition has a smaller focused teaching surface alongside the larger examples and workflow docs.

## 2026-03-22

### HTML authoring sugar, shorthand ergonomics, and browser validation

- Added the first additive `html` authoring sugar layer, including `Children(...)`, `Textf(...)`, `TextIf(...)`, `When(...)`, `ClassNames(...)`, `If(...)`, `IfElse(...)`, `Unless(...)`, `Map(...)`, `MapKeyed(...)`, `FlatMap(...)`, `FilterMap(...)`, `Join(...)`, `Maybe(...)`, `OrElse(...)`, `Coalesce(...)`, `Switch(...)`, `Case(...)`, `Default(...)`, `PropsOf(...)`, `WithProps(...)`, first-pass prop helpers, explicit event-wrapper helpers, and closure-scoped `Debounce(...)` and `Throttle(...)` helpers so primitive DOM composition can be lighter without replacing the stable typed builder surface.
- Added the companion `html/shorthand` package with mixed-argument host-tag wrappers such as `Div(...)`, `Button(...)`, and `Input(...)`, plus `FromProps(...)`, while keeping the existing typed `html.Div(...)` builder family compatibility-stable for existing `[]ui.Node` call sites.
- Extended SSR to render reactive text nodes correctly, broadened native, shorthand, and exact-markup test coverage for the html surface, and pushed html package coverage beyond the targeted threshold with parity checks against explicit builders.
- Updated package docs, API policy, roadmap tracking, and several examples to demonstrate both `PropsOf(...)`-style additive sugar and the companion shorthand path, including form-heavy, toggle, todo, and html-form example flows.
- Hardened the browser smoke-test harness by making the Playwright web-server port configurable, then validated the mounted html path with focused and broader Playwright suites plus a direct manual smoke pass covering input, form submit, effect mount or cleanup, todo creation, and fetch-driven rendering.

### Static HTML and JSX converter workflow

- Added a new `gwc import` launcher command that converts static `.html`, `.htm`, `.jsx`, and `.tsx` sources into a single inspectable `main.go` built from the public `html` builder API instead of generating an opaque scaffold or hidden transform output.
- Added focused parser and generation coverage in `tools/gwc/import_test.go`, together with persisted converter fixtures under `test/gwc-import-fixtures/` that exercise nested structures, custom tags, boolean and raw attributes, `data-*`, `aria-*`, style maps, and explicit rejection of dynamic JSX child expressions.
- Moved launcher-owned temporary output from the OS temp directory into `bin/tmp/` so release-test and converter-adjacent temporary artifacts stay rooted under the project tree.
- Added ignored repo-local converter preview outputs under `bin/converter/` and validated the catalog fixture visually by running both the original HTML and converted wasm app, capturing screenshots for each, and confirming the rendered output matched at the browser level.

### GWC launcher coverage and examples runtime

- Added broad launcher coverage for `tools/gwc`, including focused tests for `build`, `release`, `test`, `verify`, `examples`, project detection and config precedence, the `start` Bubble Tea scaffold flow, runner-config overrides, and launcher output helpers so the newer CLI surface is exercised as a maintained contract instead of only by ad hoc manual checks.
- Promoted the repo examples catalog toward a launcher-owned runtime by adding a dedicated `examples/gwc-examples-site` `js/wasm` application, generating `examples/static/catalog.json`, and replacing the older mostly static examples shell with a richer catalog experience that can load, filter, and persist catalog state from the served metadata.
- Updated the repo tooling docs to describe `gwc` as the canonical launcher for build, release, examples, dev, doctor, test, verify, and start workflows, including the current `gwc-runner.json` override points for enterprise-oriented path configuration.
- Hardened the browser-facing PWA and interop support layer with explicit wasm browser-global helpers plus expanded wasm and native coverage around persistence, installability, service-worker, and diagnostics paths so browser-only helpers behave consistently in test and runtime environments.

## 2026-03-21

### PWA helpers and durable offline persistence

- Added a new public `pwa` companion package with explicit web-app manifest helpers, installability observation, service-worker registration and update-lifecycle helpers, wasm-release-manifest-driven asset planning, Cache Storage coordination, and structured diagnostics so PWA behavior stays app-owned instead of hidden in the core rendering runtime.
- Added a first-class IndexedDB-first durable persistence seam through `interop.OpenPersistentStore(...)`, including typed JSON helpers, blocked-upgrade and quota-aware interop error codes, opt-in corruption recovery, and focused native and `js/wasm` coverage for IndexedDB and fallback-storage behavior.
- Extended `fetch` with opt-in durable shared-cache persistence through `CacheOptions.Persist` and `fetch.ConfigurePersistentCache(...)`, and moved the offline mutation queue onto the same IndexedDB-first persistence boundary while preserving explicit fallback storage control.
- Extended `state` with `SavePersistentSnapshot(...)`, `LoadPersistentSnapshot(...)`, and `RestorePersistentSnapshot(...)` so atom snapshots can survive reloads through the same durable browser-storage boundary used by cache and queue helpers.
- Added runnable PWA examples for installability and offline cache or replay behavior, plus focused Playwright coverage for installability state, service-worker update flow, offline shell warmup, stale cache cleanup, queued write replay, offline navigation fallback, and cross-tab coordination.
- Extended the offline PWA slice with app-owned Background Sync registration through `pwa.ServiceWorkerRegistration.RegisterSync(...)`, conflict-aware offline replay via `fetch.ReplayWithOptions(...)` plus `fetch.NewMutationConflict(...)`, and example coverage for graceful Background Sync fallback plus conflict requeue resolution.
- Expanded the platform docs around PWA scope, Cache Storage policy, offline mutation retention, cross-tab replay ownership, release-manifest reuse, security posture, manual testing, and backlog status so the new offline behavior is documented as an explicit public contract.

### Unified test runner and harness stabilization

- Added a canonical repo-root `npm test` entrypoint that runs the main native Go suite, the `js/wasm` Go lane, the nested `tools/livereload` Go tests, the primary Playwright workspace under `test/`, and the aggregated Playwright suites under `examples/` from one orchestrated runner.
- Added `scripts/run-main-tests.mjs` plus focused root scripts for each lane so contributors can run the full matrix or isolate one lane without manually hopping across workspaces.
- Hardened the Windows-oriented harness flow by scrubbing leaked `GOOS` and `GOARCH` values for native tools, invoking nested npm workflows through the active Node/npm runtime, and making Playwright worker counts configurable from the environment for stability.
- Folded more example coverage into the main path by aggregating dedicated example suites such as SSR server-routing, Atlas SSR, and startup experiments under the examples workspace `test:all` script.
- Stabilized the selective hot-reload Playwright coverage by using ephemeral ports, draining dev-server output, detecting early startup exits, clearing stale wasm artifacts before startup, and waiting for a single remounted changed subtree before asserting preservation behavior.
- Simplified one brittle browser stress assertion so the test continues to validate user-visible state correctness without depending on incidental render-count behavior.

### Wrapped panic contract and actionable runtime diagnostics

- Added a unified wrapped-panic reporting path across runtime render, event, effect, cleanup, loader, hydration, startup, deferred, and SSR failure phases, keeping the original panic payload while attaching stable framework codes, `where:` context, component or route `path:`, plain-language runtime consequences, remediation guidance, and grouped app/framework/platform stack sections.
- Added structured panic reporting hooks and browser-console emission support so wrapped fatal panics can be mirrored consistently into runtime diagnostics, buffered logs, devtools snapshots, and browser-observed Playwright assertions instead of appearing as raw low-context Go panic output.
- Expanded actionable framework misuse messages for hooks, atom access, `ui.CreateElement(...)`, server-only UI APIs, router registration, and nil-context misuse so invalid public API usage now follows the same stable diagnostic contract instead of ad hoc panic strings.
- Added broad runtime coverage for wrapped panic formatting, suppression behavior, recovery-versus-rethrow policy, SSR panic handling, hydration and deferred panic reporting, and propagation into devtools snapshots and browser-visible logging.
- Updated docs with a formal fatal panic log contract, new actionable error codes for server and tool failures, and explicit recovery-versus-fatal rules so future diagnostic changes stay aligned across runtime, docs, and tooling.

### First public testing companion surface

- Added `docs/TESTING.md` to define the intended first-party testing surface as a companion module made of focused helper packages instead of one monolithic core testing API.
- Added the first public `testkit/render` fixture for `js/wasm` tests with controlled mock-DOM rendering, rerendering, accessibility-first role and name queries, lower-level id/text/tag queries, synthetic event dispatch helpers, and deterministic scheduler settlement helpers.
- Added `testkit/hooks` with a lightweight `RenderHook(...)` harness so hook-driven state flows can be exercised without writing a bespoke host component for every wasm-side test.
- Added `testkit/router` with hash and history fixtures that support route registration, initial path setup, navigation, params and query inspection, and delegated rendered-route assertions from public APIs.
- Added `testkit/ssr` with snapshot helpers, typed bootstrap-payload assertions, and a lightweight hydration smoke harness so SSR and hydration paths can be tested from public entrypoints instead of repo-local fixtures.
- Added consumer-oriented example tests and direct coverage for each of the new testkit packages so the shipped surface is documented by working test patterns rather than only by API signatures.

### Example and tool error reporting

- Added a small public `diagnostics` package over the internal reporting helpers so example servers and tools can emit the same structured actionable reports used by the runtime.
- Updated several SSR and server-integrated examples to return structured request and startup failures with stable codes, runtime consequences, remediation guidance, and docs anchors instead of plain `panic(...)` or raw `http.Error(...)` responses.
- Updated the live-reload tool to emit structured diagnostics for watcher failures, websocket delivery problems, client-script injection errors, manifest generation issues, rebuild failures, and startup problems so hot-reload breakage is easier to interpret from the terminal and browser.
- Extended devtools snapshot structures and UI summaries to retain wrapped panic metadata such as top frame and runtime consequence, keeping in-app inspection aligned with the new fatal panic contract.

## 2026-03-20

### SSR observability hooks

- Added a focused SSR observability surface in `ui` through `ObserveSSR(...)` plus observed render and bootstrap helpers so applications can capture request-level render timing, bootstrap payload sizes, and inline bootstrap script sizes without adopting a framework-specific logger backend.
- Extended browser hydration instrumentation to report per-hydration duration, existing DOM counts, mismatch counts, fallback counts, and discarded-node counts, with optional correlation ids threaded from `ui.Hydrate(...)` options into the emitted observation.
- Added native and runtime coverage that verifies render observations, bootstrap size metrics, and hydration fallback summaries are emitted from the shipped public surface.

### State transfer contracts and update envelopes

- Extended `ui.SSRBootstrap` and `ui.SSRBootstrapReference` with explicit schema versions and decode-time validation so newer payload schemas are rejected instead of being silently misread by older clients.
- Added typed state-transfer helpers in `ui` for route data, form defaults, cache seeds, session hints, scoped payload inspection, and non-JSON-friendly payload encoding for bytes, time values, text-marshaled ids, and explicit CBOR payloads.
- Added `ui.SSRStateUpdate` plus JSON and CBOR encode/decode helpers so post-hydration server-to-client payload refreshes can use versioned text or binary update envelopes instead of replacing the full bootstrap payload.
- Added `ui.AnalyzeSSRBootstrapSize(...)` so applications can measure inline JSON, inline script, and CBOR payload sizes and choose between inline, sidecar JSON, or sidecar CBOR transport with explicit warning bands.

### Wasm build experiment tooling and release-size comparison

- Added repeatable wasm build experiment helpers for phase-attributed timing, cache-topology comparison, CI-friendly manifest comparison, and explicit Go toolchain comparison so build-speed and artifact-size decisions can be made from saved JSON results instead of ad hoc shell timings.
- Added a stable representative target set for small, routed mid-sized, and large wasm applications, together with updated build experiment docs and completed backlog tracking for the current measurement surface.
- Extended wasm compression and release tooling with a Node-based Brotli fallback plus `npx --package binaryen wasm-opt` support so gzip, brotli, and optimized-wasm variants can be measured on standard contributor machines even when the host PowerShell runtime lacks Brotli support or `wasm-opt` is not installed globally.
- Recorded the current accepted and rejected build-optimization outcomes in the docs, including the stripped release baseline, gzip release sidecars, cache-behavior guidance, and the still-not-default status of Brotli and `wasm-opt` until the saved comparison results justify promoting them.

### Fine-grained reactivity and subscribed-region updates

- Added explicit fine-grained subscribed-region support on top of the existing fiber runtime through reactive text nodes, general reactive regions, selector-backed shared projections, and transition-aware shared-state update paths.
- Narrow subscribed updates can now update stable text nodes, host properties, and small anchored host-only subtrees without rerendering the owning component, while hook-driven rerenders still take precedence when mixed ownership would otherwise be ambiguous.
- Expanded runtime inspection and devtools snapshots with fine-grained fiber counts, granular mark and commit counters, and per-node update-origin or reactive-source metadata so narrow updates are visible in diagnostics instead of appearing as generic dirty work.
- Added deeper runtime, wasm, SSR, benchmark, and Playwright coverage for subscribed-region behavior, selector stability, transition deferral, hydration reuse, ancestor-rerender costs, and browser-level non-rerender guarantees.
- Added `docs/FINE_GRAINED_REACTIVITY.md` plus related roadmap and performance updates so the current mixed-model contract, measurements, and limits are documented as part of the repo history.
- Followed up with targeted rerender-path optimization passes that removed the measured ancestor-rerender overhead for the current stable-region benchmark shape by reusing unchanged reactive-source bookkeeping and redirecting stale subscribed fibers to their live fine-grained twin instead of transferring ownership during every clean clone.
- Expanded profiling and devtools visibility further with descendant host and descendant text commit counters inside fine-grained regions, plus regression coverage that proves stale subscribed twins still resolve to the live current region after ancestor rerenders.

## 2026-03-18

### Browser interop and multipart workflows

- Added a new public `interop` package with typed wrappers for browser storage, history, location, clipboard, timers, custom and generic event listeners, document lookup, element handles, resize observers, intersection observers, media queries, and dynamic module import.
- Added structured interop error handling through `interop.Error`, `IsCode(...)`, `AsError(...)`, and `CodeOf(...)`, together with native and `js/wasm` test coverage for browser-only failure paths and wrapper behavior.
- Added multipart request support to `fetch` through `MultipartBody`, `MultipartFile`, and `Upload(...)`, including progress updates, cancellation, preserved response status and headers, and structured HTTP error reporting for non-success responses.
- Added public browser file helpers in `ui` so file inputs and upload flows can work through typed `ui.File` wrappers instead of raw `syscall/js` values.

### SSR forms and server-integrated application flows

- Added the `87-ssr-secure-forms` example to demonstrate request-time SSR form rendering, CSRF-aware form posting, multipart upload validation, server round-trips that preserve user input, and post-submit `303 See Other` redirects.
- Added stronger `ui` form support for server-driven workflows, including server-error shaping, request-targeted submit modes, CSRF naming helpers, and typed file helpers on both browser and native targets so SSR form code no longer has to hand-roll those pieces.
- Added focused tests and microbenchmarks for secure SSR form rendering and multipart upload handling, plus wasm-side fetch and interop benchmarks for multipart form-data creation and browser interop calls.
- Added the first Atlas Commerce OS server-integrated workflow pass with request-time rendering, mock session routing, CSRF protection, SQLite-backed read and mutation flows, bootstrap payloads, startup migrations, and dedicated SSR browser coverage.

### Documentation and adoption guides

- Added `docs/START_HERE.md`, `docs/FORMS.md`, `docs/WORKFLOWS.md`, `docs/WALKTHROUGHS.md`, `docs/TROUBLESHOOTING.md`, and `docs/REFERENCE_MAP.md` to give new adopters a task-oriented route through the current public package surface.
- Expanded the docs index and migration guidance so the current public package surface, SSR and form workflows, troubleshooting path, and reference-app adoption path are linked from one consistent set of entry documents.
- Added dedicated Atlas Commerce OS documentation, layout maps, screenshots, and server notes so the reference application is documented as a real server-integrated workflow instead of only as runnable code.
- Added `interop/README.md` and refreshed `fetch` and docs index guidance so multipart upload, SSR form, and browser interop workflows are documented from the public API perspective instead of requiring commit archaeology.

### Browser platform integration and runnable examples

- Expanded `interop` with first-class worker helpers, typed worker request/progress/result envelopes, cross-tab channels with `BroadcastChannel` and `storage` fallback, popup/opener window channels, multi-surface signals, and additional typed browser wrappers so common browser coordination flows no longer require ad hoc `syscall/js`.
- Added `html.CustomElement(...)` and related custom-element guidance so browser-defined web components can be consumed with explicit attribute, presence-attribute, and property mapping instead of raw prop spreading.
- Added explicit mount-target APIs through `ui.RenderInto(...)` and `ui.HydrateInto(...)`, plus the `ui.UseWorkerTask[...]` hook for binding worker-backed browser jobs into normal component state.
- Added `examples/public/web-components`, `89-exported-custom-element`, `90-browser-interop`, `91-worker-text-index`, `94-cross-tab-sync`, and `95-multi-window-console` to demonstrate third-party custom elements, export-side web-component prototypes, typed browser interop, worker-backed CPU-heavy UI, cross-tab sync, and multi-window coordination.

### Cache reuse, auth-routing, and offline workflows

- Expanded `fetch.UseCachedResource[T]` with cache freshness and disposal policies, imperative `LoadCached(...)`, cache inspection, SSR bootstrap restore, resume-policy handling, and route-loader interoperability so shared data reuse works across component, loader, and SSR resume flows.
- Added a durable browser-backed offline mutation queue through `fetch.OpenMutationQueue(...)` with replay, deduplication, retry scheduling, dead-letter state, and wasm-side tests for replay and cancellation behavior.
- Added safe post-auth redirect helpers in `router` for preserving and validating internal `return_to` targets, and added the `92-protected-routes` and `93-ssr-cache-bootstrap` examples to demonstrate protected navigation, shared cache reuse, and SSR-seeded cache hydration.
- Added cache, offline mutation, auth-routing, observability, and server-integration documentation so the current routed-app and shared-data model is described as a public contract instead of only through tests and examples.

### Hydration diagnostics, runtime hardening, and devtools

- Hardened hydration and recovery with component-stack-aware mismatch diagnostics, opt-in strict hydration mode, preserved browser-owned form control state during the initial reuse pass, and explicit docs for hydration behavior, state transfer, streaming SSR boundaries, and mismatch recovery.
- Expanded runtime diagnostics to record structured classifications, buffered framework logs, and component path/stack context for recovered boundary errors, and surfaced those logs and diagnostics through the devtools snapshot and panel.
- Added broader runtime correctness and benchmark coverage, including production-correctness scenarios, hydration reuse and fallback benchmarks, and transition scheduling benchmarks, while also keeping the `production` wasm utils surface aligned with development exports.
- Refreshed `examples/public/transition-hooks` into a more realistic transition-style UX example and aligned the backlog/docs around the current runtime, scheduling, error-boundary, and hydration behavior.

### Tooling, release engineering, and platform policy docs

- Added `tools/build-wasm-release.ps1` and a sample wasm size-budget file so release-style builds can emit stripped artifacts, gzip sidecars, hashes, manifests, and optional size-budget enforcement from one helper.
- Added docs covering cache behavior, custom elements, interop, workers, cross-tab sync, multi-surface coordination, error boundaries, production correctness, scheduling, hydration, streaming SSR, server integration, observability, logging, prerender, assets, browser support, PWA boundaries, security, configuration, wasm releases, build experiments, and onboarding.
- Updated example indexes, manual-testing guidance, and docs entrypoints so the newer example set and system-level docs are discoverable from the top-level navigation.
- Ignored the repo-local `.gotmp/` Go temp/build directory so local release-build and wasm test artifacts stop appearing as worktree noise.

## 2026-03-16

### Internationalization and localization

- Added a new public `i18n` package with `UseLocale(...)`, `Provider(...)`, `UseI18n()`, deterministic bundle registration, missing-message fallback behavior, interpolation, pluralization, select-style branching, and locale-aware number or date formatting helpers.
- Added route-oriented helpers `PrefixPath(...)` and `ResolvePath(...)` so locale prefixes can stay application-owned without pushing locale policy into `router` itself.
- Extended `ui.SSRBootstrap` with typed `I18n` payload support and bundle conversion helpers so server-rendered pages can transfer active locale, fallback locale, direction, and the initial message subset used during hydration.
- Added `docs/I18N.md` to define the current i18n scope, SSR transfer model, locale-aware routing guidance, and RTL or directionality expectations.
- Added `examples/public/locale-switcher`, `examples/public/server-side-rendering-internationalization-bootstrap`, and `examples/public/locale-routing` together with focused Playwright coverage for runtime locale switching, bootstrap-driven locale hydration, and locale-prefixed loader-driven routing.
- Added native `i18n` tests and a translation microbenchmark covering fallback lookup, pluralization, formatting, SSR bootstrap round-tripping, and locale-aware path helpers, plus `ui` bootstrap tests that assert the new `I18n` payload is serialized and initialized correctly.

### Portal layering and overlay management

- Added `ui.Overlay(...)` and `ui.UseOverlayStack(...)` for shared overlay layering, stack-derived z-order, nested escape and outside-click routing, and coordinated focus ownership across portal-backed surfaces.
- Updated `ui.AccessibleOverlay(...)` to compose on top of the shared overlay stack instead of wiring instance-local modal behavior independently.
- Updated overlay side effects to support nested scroll-lock counting and nested background inert ownership by app-root selector.
- Added `docs/OVERLAYS.md` to document the current overlay layering model, dismissal routing, and anchored-position guidance.
- Added `examples/public/overlay-stack` and `examples/82-overlay-anchor` together with focused Playwright specs covering nested dialogs, dialog-plus-popover routing, tooltip-over-menu layering, and portal retargeting.
- Added stack-manager coverage in `ui` tests so topmost escape handling, outside-dismiss routing, focus-trap ownership, and derived z-index behavior are validated directly.

### Accessibility guidance baseline

- Added `ui.UseFocusManager()`, `ui.UseFocusTrap(...)`, `ui.UseCompositeNavigation(...)`, `ui.UseAnnouncer()`, and `ui.AccessibleOverlay(...)` as the first public accessibility-focused primitives.
- Expanded `docs/ACCESSIBILITY.md` to document the shipped accessibility model for focus restoration, modal overlays, composite keyboard navigation, live-region announcements, forms, route changes, and async UI.
- Added baseline and API-level tests covering accessible prop preservation, `ui.UseId()` behavior, composite navigation keyboard flow, and live-region rendering.
- Added focused accessibility examples for modal overlays, composite widgets, form validation announcements, and routed page announcements, together with Playwright browser coverage.

### Head management and SEO guidance

- Added `docs/HEAD_MANAGEMENT.md` to define the current head-management model, including the boundary between router-managed metadata and application-owned explicit SEO markup.
- Documented canonical URL policy, structured-data guidance, resource-hint guidance, social metadata examples, and sitemap or robots integration expectations for the current SSR surface.
- Added tests covering managed head-tag deduplication after hydrated startup and SSR output checks that enforce a single managed title, description, and canonical tag set.

### Metadata model and composition policy

- Added `router.MetadataNode(...)` so route-managed title, description, and canonical tags can be rendered during SSR through `ui.RenderToString(...)`.
- Updated client router metadata reconciliation to treat SSR and client navigation as one managed metadata flow, reusing and cleaning up only router-owned head tags.
- Updated the SSR server-routing example to render managed metadata tags through the shared router helper instead of hand-built escaped strings.
- Documented the current composition decision that slots are out of scope in favor of ordinary children, explicit props, context, portals, and layout routes.

### API policy and migration guidance

- Added `docs/API_POLICY.md` to define public API stability tiers, semver expectations, deprecation timing, breaking-change rules, and current latest-major-only support policy.
- Added `docs/MIGRATIONS.md` as the release-to-release migration index, starting with guidance for moving into the current `v3.x` public package layout.
- Linked the new policy and migration docs from the root README and docs index.
- Updated the documentation backlog to mark the API stability and support policy work complete.

### Example catalog and backlog alignment

- Expanded the feature example catalog so the examples index and shared catalog copy map the shipped examples more directly to the current public APIs and learning goals.
- Refreshed the root README and docs backlog structure around the current public package surface, numbered roadmap sections, and completed-work tracking instead of leaving newer example and docs work scattered across older notes.
- Added the first Atlas Commerce OS planning backlog so the larger reference-application effort has an explicit timeline in the repo instead of only appearing through implementation commits later.

## 2026-03-15

### Runtime optimization work

- Reduced `GoUseFunc(...)` validation overhead with build-specific fast paths for common function signatures.
- Reduced `GoUseAtom(...)` overhead by storing atom values directly, adding a single-subscriber notification fast path, and caching getter/setter accessors per hook slot.
- Added keyed-reconciliation fast paths, pooled keyed scratch state, and lower-overhead keyed-child consumption to reduce keyed list churn.
- Cached DOM prop metadata in `updateDomProperties(...)` and added a steady-state DOM property benchmark path.
- Cached additional jsdom document query bindings and switched DOM collection traversal to indexed access instead of `item(...)` calls.
- Added runtime layout and keyed reconciliation benchmark coverage used to validate and reject broader cache-layout experiments.
- Reverted broader experiments that regressed the benchmark suite, including the hot/cold `Fiber` split and a dedicated non-batching initial DOM-props fast path.

### Runtime correctness fixes

- Fixed function-component deletion so removing one DOM-less component subtree does not incorrectly remove sibling component DOM.

### SSR and hydration groundwork

- Added the first internal SSR render-to-string path for host elements, text nodes, fragments, and simple function components.
- Added public server-side rendering support through `ui.RenderToString(...)` on non-browser targets.
- Added a dedicated `Hydrate(...)` and runtime `HydrateTo(...)` entrypoint so client resume now has a real API path instead of reusing plain render calls.
- Upgraded hydration from whole-container fallback to real DOM reuse for matching host and text nodes, with subtree-level fallback when structure matching fails.
- Added hydration mismatch diagnostics for text, tag/structure, trailing-node, and critical attribute differences, with clear warn-versus-replace behavior.
- Added bootstrap-time atom snapshot restore and ID-seed application so hydration resumes shared state and `UseId` generation from the transferred SSR payload.
- Deferred hydration-time atom subscriptions and follow-up update notifications until the hydration commit completes, then ran effects against the committed tree.
- Added safe SSR bootstrap helpers for inline JSON payloads, including script-tag rendering and browser-side bootstrap-script reading.
- Added optional CBOR-based binary bootstrap encoding/decoding for SSR payload transport.
- Added sidecar bootstrap reference support so hydration can load external JSON or CBOR bootstrap payloads instead of only inline script JSON.
- Added SSR/bootstrap and hydration tests covering DOM reuse, subtree fallback, text mismatch recovery, trailing-node cleanup, bootstrap atom restore, and ID-seed resume behavior.
- Added SSR transport microbenchmarks showing CBOR bootstrap encode/decode is materially cheaper than JSON for larger sidecar-style payloads in the current implementation.

### Public package and wasm tests

- Added wasm-facing public package tests for `html`, `ui`, `state`, and `fetch` wrappers.
- Expanded HTML builder coverage to verify public prop preservation, wrapper tag selection, and text/fragment helper behavior.
- Added public wrapper coverage for `state.UseComputed`, `fetch.UseResource`, `ui.UsePrevious`, `ui.UseChannel`, and `ui.UseTask`.
- Added public wrapper coverage for `ui.UseReducer`, `ui.UseForm`, `state.UseDerived`, and state snapshot persistence helpers.

### Public hooks and state ergonomics

- Added `state.UseComputed[T]` as a typed derived-state helper on top of the memoization runtime.
- Added `state.UseDerived[T]` as a read-only shared derived-atom helper with explicit source-atom dependencies.
- Added state snapshot export/import and browser storage persistence helpers for same-process restore and JSON-compatible saved state.
- Added `fetch.UseResource[T]` for typed async loading with cancellation, reloads, and typed ready/error state.
- Added `fetch.UseCachedResource[T]` with shared keyed cache state, in-flight request deduplication, stale-while-revalidate refreshes, invalidation, and optimistic update helpers.
- Added `ui.UsePrevious[T]`, `ui.UseChannel[T]`, and `ui.UseTask[T]` to cover render-time comparisons, channel-driven UI streams, and explicit cancellable background jobs.
- Added `ui.UseReducer`, `ui.UseForm`, `ui.UseDebounced`, and `ui.UseThrottled` to cover reducer-style local state, structured form lifecycle handling, and delayed/rate-limited derived UI values.
- Added `ui.StartTransition`, `ui.UseTransition`, and `ui.UseDeferredValue` as the first concurrent-style scheduling primitives for non-urgent tree refreshes and lagging derived values.
- Added `ui.AsyncBoundary`, `ui.UseLazyNode`, and `ui.Lazy` as the first explicit async UI primitives for fallback rendering, deferred subtree resolution, and timeout-aware loading boundaries.
- Added `ui.ErrorBoundary` as a runtime-recognized subtree recovery wrapper with fallback rendering, explicit reset support, and server/client parity for render-time panic recovery.

### Router fixes and tests

- Normalized router default routes, registration paths, and navigation targets so hash/history routing behaves consistently across trailing-slash and hash-prefixed inputs.
- Hardened browser-router setup against missing browser globals in wasm test environments.
- Added wasm browser-environment test helpers and broader router wasm test coverage for navigation, registration, and normalized default-route behavior.
- Added route patterns, typed `UseParams`, `UseQuery`, and `UseNavigate` helpers, plus query-aware path normalization for hash and history routing.
- Added route-level loaders with route-scoped loading/error renderers, cancellation on navigation changes, and route+query keyed revalidation.
- Added `UseRouteData`, `UseRevalidator`, and manual current-route revalidation support with router tests for params, query parsing, loaders, cancellation, and explicit reruns.
- Added route option support for titles, declarative redirects, synchronous `BeforeEnter`/`BeforeLeave` guards, and route-managed description/canonical metadata with cleanup on route changes.
- Added nested layout routes through `router.Options{Layout: true}` and explicit child rendering through `router.Outlet()`.
- Added nested route-stack resolution so parent layout routes render from shallowest matching prefix to the final leaf route, with parent params scoped per level and leaf routes receiving the final merged param set.
- Extended nested routing support across hash and history routers, including layout-aware guard evaluation and per-route loader-data scoping during nested rendering.
- Expanded router edge-case coverage for decoded params, unsupported optional-segment syntax, exact-vs-pattern precedence, blocked navigation, redirected navigation, and metadata replacement/cleanup semantics.
- Added nested layout route tests covering outlet rendering, per-level params, per-level route data, leaf-metadata precedence, nested redirects, `InspectCurrentRoute()`, and history-router nested guard behavior.
- Added router microbenchmarks for nested route-stack resolution, nested `Current()` rendering, and nested navigation evaluation baselines in the js/wasm test lane.

### Devtools and inspection

- Added a new public `devtools` package with runtime snapshot capture and an embeddable in-browser inspector panel.
- Added committed component tree inspection, hook summaries, route inspection, and runtime totals to the new devtools surface.
- Added structured diagnostics for invalid hook usage, missing render targets, duplicate route registration, and invalid route component configuration.
- Embedded the devtools panel into the `14-omi` showcase example.
- Added baseline profiling counters for renders, scheduled updates, work-loop passes, processed units, commits, and last render/commit timings.
- Added structured missing-key diagnostics for mixed keyed/unkeyed sibling lists.
- Added runtime error-boundary recovery for render, effect, cleanup, and event-handler panics together with diagnostics and follow-up rerender scheduling.
- Added a smaller standalone `16-devtools` example focused on the public devtools package.

### Example upgrades

- Reworked the goroutine example around `ui.UseTask` and `ui.UseChannel`, replacing manual task bookkeeping and adding a streamed channel-driven UI panel.
- Reworked the fetch example around `fetch.UseResource[T]` with typed list/detail loading, explicit retry controls, and cancellation.
- Reworked the fetch example further around `ui.AsyncBoundary` and `ui.Lazy`, replacing manual loading-branch plumbing with explicit async subtree boundaries and a deferred auxiliary panel.
- Reworked the fetch example again around `fetch.UseCachedResource[T]`, shared query reuse, key-based invalidation, and optimistic detail/list updates.
- Updated the atoms example to demonstrate `state.UseComputed` for derived labels and summaries.
- Updated the text-input example to demonstrate `ui.UseDebounced` and `ui.UseThrottled` with visible delayed preview and throttled count feedback.
- Reworked the advanced-form example around `ui.UseForm`, async validation, retryable submission, and router-driven success flow.
- Expanded the OMI example with nested route navigation, loader-backed detail/search/protected routes, and manual route revalidation controls.
- Added the `19-nested-routes` example to demonstrate dashboard and docs layout shells, nested settings layouts, explicit outlet composition, and nested param-driven leaf routes.

### Documentation refresh

- Aligned the `fetch` package docs and examples around `UseFetch` as the low-level raw hook and `UseResource[T]` as the preferred typed async resource API.
- Extended the `fetch` docs to cover `UseCachedResource[T]`, shared cached queries, optimistic updates, and async-boundary integration.
- Rewrote the `router` package docs to match the current public API, including params, query helpers, loaders, and manual route revalidation.
- Expanded router docs to cover layout-route registration, `router.Outlet()`, nested route matching rules, and per-layout route-data behavior.
- Refreshed portfolio and showcase copy to use the current `UseState`/`UseEffect`/`UseMemo` and `UseFetch`/`UseResource` naming instead of stale legacy names.
- Expanded `state` and `ui` docs to cover typed computed state, previous-value tracking, channel subscriptions, and cancellable tasks.
- Documented the transition/deferred-value scheduling model and the current decision to avoid a dedicated layout-effect hook until concrete DOM-read-before-paint cases appear.
- Expanded `ui` docs further to cover the new `ErrorBoundary` contract, fallback callbacks, and explicit reset behavior.
- Added docs for shared derived state, snapshot persistence, route guards, route-managed metadata, reducer/form helpers, and debounced/throttled UI hooks.
- Updated the backlog and examples index to mark nested routes and layout routes complete and list the new multi-level example.
- Expanded the root README with a project-wide feature inventory, clarified the stable public package surface, and brought the shipped examples list up to the current `16` through `19` demo set.
- Refreshed the SSR demo and README copy so the examples describe the shipped hydration behavior accurately: bootstrap restore, matching DOM reuse, and subtree fallback on structural mismatch.

### Browser tests and benchmarks

- Expanded Playwright integration coverage for filtered todo flows, effect cleanup under broader app activity, and mixed local/shared-state burst interactions.
- Added Playwright coverage for debounced/throttled text input behavior, advanced-form validation/retry/success flow, and the standalone devtools example.
- Added jsdom wasm benchmarks for `GetElementById(...)` and `QuerySelectorAll(...)` and refreshed the React-vs-Go browser benchmark run.

## 2026-03-14

### Runtime fixes

- Fixed function-component reconciliation so components that return `nil` correctly delete previously rendered children.
- Fixed DOM-less subtree deletion so all descendant sibling DOM branches are removed during unmount.
- Fixed stale hook ownership on reused/skipped function components so later stateful updates target the correct fiber.
- Fixed `GoUseState` so nil-able state types can be reset with `set(nil)`.
- Fixed `GoUseAtom` so nil-able atom state types can be reset with `set(nil)`.
- Fixed `GoUseAtom` initialization on fresh component fibers so global atom hooks work on first use.
- Fixed scheduler update propagation so clean ancestors are marked even when a leaf is already dirty.
- Fixed global runtime initialization so `InitGlobalRuntime()` can upgrade a lazily created global runtime.
- Fixed HTML component helper wrappers so component refs are wrapped as component elements instead of being passed as raw children.
- Removed the artificial `50ms` delay from the wasm fetch hook.
- Removed runtime debug logging from the wasm fetch hook.
- Completed the runtime event wrapper so `GoEvent` exposes target and key-code access and safely handles missing event methods.
- Optimized `Text()` shim output to use the text-element layout directly.
- Normalized the native fetch stub error and removed per-call closure allocation from the stub implementation.

### Runtime optimization work

- Added work-in-progress fiber reuse through alternates in the reconciler and scheduler.
- Added batched atom unsubscription with `UnsubscribeMany(...)` to reduce lock churn during cleanup.
- Optimized component helper child creation in `html.go` to avoid unnecessary wrapper overhead.
- Specialized `WASMDOMAdapter.WrapFunction` so callback dispatch type selection happens once at wrap time instead of on every invocation.
- Kept only measured low-risk optimizations and reverted regressions found during benchmark runs.

### Runtime tests

- Added contract-focused reconciler tests.
- Added extra hooks contract and edge-case tests.
- Added scheduler contract and edge-case tests.
- Added atom state contract and cleanup tests.
- Added runtime global-init and render contract tests.
- Added HTML helper contract tests.
- Added wasm fetch hook tests.
- Added wasm event adapter tests.
- Added shim tests for global wrapper behavior.
- Added interfaces contract tests and compile-time adapter assertions.
- Added exhaustive native runtime coverage tests and raised native `internal/runtime` statement coverage to `100%`.

### Browser component and integration tests

- Added Playwright component tests for local state isolation, rerender stability, effect cleanup, and `UseId` behavior.
- Added Playwright integration tests for todo flows, shared atom propagation, theme updates, fetch flows, and form handling.
- Added deep state stress tests covering `5+`, `25+`, `100+`, repeated burst updates, mixed local/shared state, and rerender persistence.
- Added test app components to exercise stress and mixed-state scenarios.
- Added mock API and test server endpoints needed by the browser suites.

### Benchmarks

- Added runtime microbenchmarks for reconciler hot paths.
- Added hooks microbenchmarks.
- Added scheduler microbenchmarks.
- Added atom state microbenchmarks.
- Added runtime wiring microbenchmarks.
- Added HTML helper microbenchmarks.
- Added shim microbenchmarks.
- Added native fetch stub microbenchmarks.
- Added `types.go` microbenchmarks.
- Added wasm adapter boundary benchmarks for DOM operations and callback wrapping.

### Benchmark tooling and reporting

- Fixed the browser benchmark harness and added a runtime update regression test to catch render/update performance regressions.
- Added benchmark build and inspection scripts for the browser benchmark suite.
- Documented the current browser benchmark results in the project documentation.

### Public API and examples

- Introduced the typed `ui` and `html` public APIs and started moving examples onto the new surface.
- Added the `14-omi` showcase example.
- Added the `15-calculator` interactive calculator example with engine, UI, and presets.
- Updated the examples index and static landing page wiring for the new showcases.

### Documentation refresh

- Refreshed the root README to reflect the current runtime architecture, test status, public package guidance, and install/build instructions.
- Updated browser compiler and project docs to match the current repo layout and tooling.

### Tooling and developer workflow

- Rebuilt the development server around Node/Express and saved it in `tools/dev-server/`.
- Added dynamic `/examples` listing based on the filesystem instead of hard-coded routes.
- Updated server scripts and tooling docs to use the new development server.
- Added a wasm test execution helper in `tools/go_js_wasm_exec.bat`.
- Added new test scripts for component, integration, and state browser test layers.
- Added a GitHub Pages deployment workflow for the examples site and a generated redirect page for the published showcase.
- Upgraded CI workflows to current `checkout`, `setup-go`, and `setup-node` action majors.
- Upgraded CI to Go `1.25.4` and Node `24`.
- Stabilized flaky Playwright hook/browser tests by replacing transient timing assertions with stable result checks and stronger selectors.
- Tracked the Playwright test lockfile, added npm caching for the test package, and split CI dependency install, browser install, and test execution into separate steps.
- Fixed automated release version calculation so semantic releases continue from the latest `vX.Y.Z` tag instead of resetting to `v0.0.x`.

### Validation status

- `go test ./internal/runtime` passes.
- Native `internal/runtime` statement coverage is `100%`.
- Browser component, integration, and deep state stress suites pass.
- Wasm runtime and adapter tests pass in the separate `js/wasm` test lane.
- The examples site deploys through the GitHub Pages workflow.
- The `CI + Release` workflow passes on `master`.
- Automated semantic release tagging resumed and published `v3.0.3`.

## 2025-11-26

### Browser compiler experiment

- Moved the in-browser compiler experiment into its own more self-contained example flow under `examples/public/browser-compiler`.
- Added the supporting browser-compiler scripts, package index generation flow, and bundled `js/wasm` standard-library assets needed to run that experiment from the repo.

## 2025-11-19

### Runtime architecture and performance

- Reworked the runtime architecture and aligned the examples with the updated rendering and hook model.
- Optimized reconciliation and rendering hot paths across the runtime and jsdom adapter layers.
- Fixed the goroutines example so background-driven UI updates render correctly.

### Benchmarks and test structure

- Added a browser performance benchmark app and Playwright benchmark coverage, and restructured the broader test layout around that workflow.
- Expanded runtime and reconciliation coverage while tightening test expectations around hook/fiber behavior.
- Fixed advanced hash-router test behavior around navigation history and visibility-driven cases.

### CI and release workflow

- Added the initial CI/CD workflow and refreshed the project README with workflow badges and updated project assets.
- Iterated on CI reliability by excluding unsuitable benchmark/unit-test paths, switching browser-test dependency install behavior, and wiring E2E web-server startup on port `8081`.
- Scoped CI browser coverage away from portfolio-site cases that required a separate examples server.

### Examples, docs, and site polish

- Refined the portfolio-site presentation, including the 3D showcase card rendering issues and updated hero-image assets.
- Added license and example coverage for pkg.go.dev across the public packages.

## 2025-11-18

### WASM runtime and fetch groundwork

- Added early WASM runtime improvements centered on a new fetch hook path and lower-level function-wrapping support.
- Updated the jsdom/mockdom adapter and runtime shim layers to support that browser-focused hook flow.
- Added early browser test coverage for events, hooks, and reconciliation rerender behavior using the test app fixtures.

### Documentation cleanup

- Cleaned up the root README formatting and readability before the larger runtime and tooling changes that followed.


