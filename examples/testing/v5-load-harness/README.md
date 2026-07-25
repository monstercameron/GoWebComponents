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

Implemented: `stats.js`, `metrics.js`, `harness.js`, `probe.go`, `budgets.json`.

Not yet built (**P0.2**): the subject app — `main.go` with the five concurrent
workloads from `v5-plan.md` §1.2 (50k-row SQLite import, full-text re-index,
2MB fetch+decode loop, 5k-row virtualized table, continuous typing probe) —
and the Playwright driver under `test/playwrightgo/examples`.

The subject app must expose:

```js
window.__gwcV5Workloads = { import: Workload, reindex: Workload, fetch: Workload, ... }
window.__gwcV5Probes    = { typing: Probe, scroll: Probe, filter: Probe }
window.__gwcV5Probe     = () => string  // registered by probe.go
```

Until P0.2 lands, `stats.js` is independently testable under node and is the
piece worth reviewing first — it is where a wrong result would be least
obvious.
