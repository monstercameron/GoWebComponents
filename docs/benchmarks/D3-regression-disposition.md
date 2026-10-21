# D3 — Historical benchmark drift and current release gate

Hand-authored context for the drift flagged in [`DRIFT_NOTE.md`](./DRIFT_NOTE.md)
(that file is generated; do not edit it). The committed report is historical;
the dedicated fair v6/v7 native and Wasm comparisons are the release evidence.

## What the historical report shows

The committed `DRIFT_NOTE.md` reports, on `go1.26.0 windows/amd64`:

- **Overall normalized score: `100.6`** (`100 × geomean(reference_ns / measured_ns)`).
  This is marginally faster in aggregate for that historical measurement pair.
- Improved / regressed / unchanged: **109 / 114 / 524** at a **2.0% per-benchmark**
  tolerance.
- Reference generated `2026-03-25T15:17:41Z`; report generated `2026-03-25T15:30:55Z`
  — about 13 minutes apart. Both report Windows amd64 and Go 1.26.0; the files
  do not record a host identity.

Two reports produced minutes apart with 109 faster and 114 slower
entries and a near-flat aggregate suggest that the 2% per-entry threshold is
sensitive to measurement variance. The `…ReactiveTextUpdate16-20` outlier near
123% needs a same-runner repeat before it can be classified; the aggregate
alone cannot prove that an individual outlier is noise.

Both historical measurements were on Windows amd64, so that pair does not
measure an ARM64/amd64 difference. It also cannot establish v6/v7 parity on
Linux with Go 1.27; the source, toolchain, and runner must be paired directly.

## What gates D3 now

- `.github/workflows/bench-drift.yml` checks out the base and head revisions into separate
  directories on the same `ubuntu-latest` runner. Because the v6 base predates `gwc bench suite`,
  the workflow builds the suite tool from head, normalizes the base benchmark drivers with
  `normalize-v6-benchmarks.py` (dropping unused legacy loop counters), compile-checks
  the base packages, and then runs the bounded
  production suite for eight representative native packages on each revision in sequence
  (`-count 5`, one warm-up, deterministic shuffle seed).
- The gate uses `gwc bench suite compare -max-regress-pct 15 -min-baseline-ns 100`.
  The comparator uses per-entry medians and MAD-derived significance thresholds;
  timings below 100 ns/op remain advisory after two same-runner runs showed sign
  flips at that scale. The committed reports remain historical documentation;
  same-run suite JSON and console logs are uploaded as evidence.

## Residual / honest caveat

Investigate any hard-gate failure against source and raw samples. Review tiny
advisory entries in the full fair native/Wasm reports for release material.
The historical score of 100.6 is not a v7 release verdict.
