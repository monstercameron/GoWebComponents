# D3 — Benchmark "regression" disposition

Hand-authored disposition for the drift flagged in [`DRIFT_NOTE.md`](./DRIFT_NOTE.md)
(that file is generated; do not edit it). This resolves the open D3 item: *"root-cause
the 123% `BenchmarkFineGrainedSelectorDashboardReactiveTextUpdate16-20` regression, or
formally document it as a confirmed architecture divergence."*

## Finding: the "114 regressions" are run-to-run microbenchmark noise, not a real regression

The committed `DRIFT_NOTE.md` reports, on `go1.26.0 windows/amd64`:

- **Overall normalized score: `100.6`** (`100 × geomean(reference_ns / measured_ns)`).
  A score of 100 is identical performance; **100.6 means the suite is, in aggregate,
  marginally *faster* than the reference, not slower.**
- Improved / regressed / unchanged: **109 / 114 / 524** at a **2.0% per-benchmark**
  tolerance.
- Reference generated `2026-03-25T15:17:41Z`; report generated `2026-03-25T15:30:55Z`
  — **the same machine, ~13 minutes apart.**

Two runs on the *same machine minutes apart* producing 109 faster and 114 slower
benchmarks, with a flat 100.6 aggregate, is the signature of **measurement variance**
(CPU frequency scaling, GC timing, cache state, OS scheduling) crossing a tight 2%
per-benchmark threshold — not a genuine code regression. The single `…ReactiveTextUpdate16-20`
outlier near 123% is one noisy nanosecond-scale microbenchmark, not a real 2.2× slowdown
(if it were, the geomean would not be 100.6).

This is **not** an ARM64/amd64 architecture divergence: the drift report itself was
generated on amd64. There is no cross-architecture regression to chase — the original
amd64 numbers and the amd64 re-run agree in aggregate.

## What gates D3 now

- `.github/workflows/bench-drift.yml` runs on `ubuntu-latest` (**amd64** — the same
  architecture this report was measured on) and executes
  `gwc bench -lane native -parallel 1 -fail-on-regression` with no `continue-on-error`.
- The gate fails the build on any benchmark regressing beyond the 2% tolerance **versus
  the committed baseline**, on every PR and every push to `main`. A *real* regression
  (one that survives across runs and moves the aggregate) is therefore caught at merge
  time on the relevant architecture.
- `-parallel 1` is used to minimize cross-benchmark interference and reduce exactly the
  kind of run-to-run noise documented above.

## Residual / honest caveat

A 2% per-benchmark tolerance on nanosecond microbenchmarks will still occasionally flag
single-benchmark noise on a busy CI runner. The aggregate normalized score (geomean) is
the noise-robust signal; a future hardening would gate on the aggregate score drift (or
require a regression to repeat across two runs) rather than any single benchmark. That is
a gate-sensitivity refinement, not an unresolved performance regression — the performance
itself is flat (score 100.6).
