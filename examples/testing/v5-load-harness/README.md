# V5 Load Harness

The responsiveness instrument for GWC v5. Implements **P0.1–P0.3** of
`docs/plans/v5-plan.md` and produces the metrics the version is judged on.

## Why this is not Example 201

Example 201 measures **throughput**: how long does scenario X take to complete,
in isolation, versus React. That is the right instrument for the single-threaded
optimization work that took the geomean from 0.27x to 0.71x, and it stays the
gate for M6 (no runtime1 regression).

It cannot measure what v5 is about. v5's claim is:

> Heavier work takes longer to **complete**. It never takes longer to **paint**.

That is a statement about the **frame timeline while several workloads run at
once**, which 201 never produces — every scenario there runs alone, and its
finish line is a single `requestAnimationFrame` boundary rather than a
distribution.

Concretely, 201 cannot answer:

| Need | Why 201 can't |
|---|---|
| M1 frame time under load vs idle | no background load generator exists |
| M2 zero long frames | long tasks are counted as a diagnostic, never gated |
| M3 input-to-paint p99 | no Event Timing / INP measurement |
| M7 max GC pause | probe reports `PauseTotalNs`, a cumulative sum — one 9ms pause and thirty 0.3ms pauses look identical |

The two harnesses are complementary. 201 guards the constant factor; this one
guards the distribution.

## Design

**Load is never measured. The probe is.** Background workloads exist only to
perturb a foreground interaction probe. The entire v5 thesis is that they must
fail to. Workload throughput is reported alongside, so a "pass" achieved by
doing no background work is visible rather than silent.

**Interleaved arms.** `DEVNOTES_PERF_LOOP` records ±25–30% per-scenario swings
on this fanless machine, and a load run heats it. Running idle×N then loaded×N
assigns all thermal drift to the loaded arm. The harness alternates
IDLE, LOADED, IDLE, LOADED… and then `detectDrift()` proves whether the
interleaving actually held. **A drifted run fails the gate regardless of its
numbers** — an invalid measurement is never reported as a pass.

**Distributions, never means.** A mean frame time of 12ms with a p99 of 180ms
is a broken app that reports as healthy. Everything is percentile-based, spread
is reported as median absolute deviation, and bimodal frame distributions are
surfaced rather than smoothed — a clean/jank split *is* the bug v5 exists to
fix.

**Equivalence, not absence of difference.** M1 claims two samples are the
*same*. A non-significant difference test does not show that; it shows the
harness failed to detect one, which noise guarantees. `stats.equivalent()` uses
a bootstrap CI of the p95 difference and requires the whole interval to fall
inside a pre-declared margin.

**Deterministic statistics.** Bootstrap uses a seeded PRNG so CI gates do not
flake for reasons unrelated to the code under test.

## Files

| File | Role |
|---|---|
| `stats.js` | percentiles, bootstrap CI, equivalence test, drift detection, cluster split. Pure, node-testable |
| `metrics.js` | the instrument: rAF distribution, Long Animation Frames, Event Timing, Go probe diffing |
| `harness.js` | interleaved A/B orchestration, report building, budget gating |
| `probe.go` | Go-side probe — phase totals plus **windowed max GC pause** |
| `budgets.json` | gate thresholds and the v4 baseline slot |

## Instrumentation notes

**Long Animation Frames over longtask.** LoAF reports the whole animation
frame, attributes blocking time to specific scripts, and separates
style/layout cost — including `forcedStyleAndLayoutDuration`, the classic
cause of a long frame that phase totals alone cannot explain. Chrome 123+.
The longtask fallback keeps the harness portable but loses attribution, so the
report records which instrument produced a run; a run measured with the weaker
one is never silently compared against a stronger one.

**Event Timing granularity.** Chrome rounds `event.duration` to 8ms for
privacy. Fine for a 50ms gate, useless for chasing a 2ms win — use the Go
phase totals for that. Entries with `interactionId === 0` are excluded so
scroll noise cannot dilute the interaction percentile.

**GC max pause.** `MemStats.PauseNs` is a 256-entry circular buffer indexed by
`(NumGC+255)%256`. `probe.go` scans only the cycles between two reads to report
a *windowed* maximum. If more than 256 collections occurred between reads the
true maximum is unknowable, and the probe reports `pauseSampleTruncated` rather
than lying.

**Refresh rate is detected, not assumed.** A hardcoded 16.7ms threshold
under-reports dropped frames by 2x on a 120Hz panel.

**The probe is never read inside a measured window.** `ReadMemStats` stops the
world and would perturb the frames being measured.

## Status

Implemented: `stats.js`, `metrics.js`, `harness.js`, `probe.go`, `budgets.json`,
plus the subject app (`main.go`, `index.html`) — **P0.2**.

42 unit tests green across three suites:

```
node stats.test.mjs && node gate.test.mjs && node integration.test.mjs
```

### Running the harness

```powershell
$env:GOOS='js'; $env:GOARCH='wasm'
go build -tags production -o ./examples/testing/v5-load-harness/v5harness.wasm ./examples/testing/v5-load-harness
$env:GOOS=''; $env:GOARCH=''
go run ./tools/gwc examples     # serves the catalog
```

**`-tags production` is required, not optional.** Without it the binary carries
the development guards, and the gate then reports numbers for a binary nobody
ships. The dev build calls `runtime.Stack` once per render session to recover a
goroutine id for the hook-ownership check (`internal/runtime/hook_threading.go`
documents it as costing microseconds) and runs the duplicate-sibling-key scan on
every keyed reconcile. Every gate number recorded before 2026-07-26 was measured
without the tag.

A second trap sits behind the first: **the page is served from a COPY of
`v5harness.wasm`, not from this directory.** Rebuilding here does not change what
the browser runs. Verify the served artifact before trusting any number:

```powershell
curl -s http://127.0.0.1:<port>/v5harness.wasm -o served.wasm
Get-FileHash served.wasm, ./examples/testing/v5-load-harness/v5harness.wasm
```

A stale copy silently invalidated a full day of gate measurements on 2026-07-26,
including an A/B whose two arms turned out to be the same binary.

Then open the harness page and press **Run harness**. It writes
`window.__gwcV5Report` and `window.__gwcV5Verdict`, which is what a Playwright
driver reads.

### Subject app

`main.go` exposes three globals:

| Global | Shape |
|---|---|
| `__gwcV5Workloads` | `import`, `reindex`, `decode` — each `start`/`stop`/`stats` |
| `__gwcV5Probes` | `typing(durationMs)` → Promise |
| `__gwcV5Probe` | phase totals + windowed max GC pause (`probe.go`) |

Two design points worth keeping if this is rewritten:

**The probe drives real DOM events**, not state setters. Event Timing only
records entries with a genuine `interactionId`, so a probe that mutated state
behind the DOM's back would produce no interaction records at all and M3 would
silently measure nothing.

**The decode workload builds its 2MB payload locally** instead of fetching one.
`fetch()` already runs off the main thread, so the main-thread cost this
scenario needs to model is the *decode*, not the transfer — and building it
locally keeps the harness deterministic and offline.

**Workload throughput is reported alongside the metrics** (`completed` counts),
so a "pass" achieved by doing no background work is visible rather than silent.
