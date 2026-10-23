# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

## v7.0.8 — production benchmark correction

This patch includes the v7.0.7 rendering and correctness improvements below
and corrects how the Example 201 browser benchmark measures production Go/Wasm.
The public API, module path, and Go 1.27 requirement are unchanged.

Production Example 201 Go/Wasm benchmark builds exclude the development
example logger; development builds retain it. The logger ran inside scored
click actions in the historical Go browser comparisons while the React
fixture had no matching logger. A same-run, four-block Hook Grid Render A/B
measured logger-on medians of 4.6 and 5.0 ms and logger-off medians of 3.0
and 3.2 ms: **35.4% less measured DOM-ready time** by the mean of block
medians, with 40 samples and 40 rendered hook nodes per block. This measures
benchmark instrumentation overhead, **not a core-library speedup or an
overall React-relative improvement**. The published v7.0.7 results remain
historical measurements with that instrumentation present. See
[#90](https://github.com/monstercameron/GoWebComponents/pull/90), the
[same-run A/B](https://github.com/monstercameron/GoWebComponents/actions/runs/39956875326),
and [benchmark evidence](docs/benchmarks/V7-RELEASE-EVIDENCE.md).

## v7.0.7 — runtime performance and correctness patch

This patch reduces work in common rendering paths. Stable child-order commits avoid materializing an observed-node slice; exact child-order prefixes append only the remaining nodes, and a confirmed adjacent swap is repaired with one `InsertBefore`. Additional fast paths cover canonical scheduler origins, key-only element props, hook-owner lookup, retained DOM props, clean reused fibers, typed-prop comparability, and tracked atom revision writes. Derived atoms reject stale recomputations when a newer dependency write occurs, and deleting then re-registering an intermediate derived atom preserves its downstream consumers. The router query-revalidation Wasm test now waits for the loaded value to render before fixture cleanup, avoiding a test-lifecycle race. The native benchmark-drift gate balances base and candidate samples in ABBA order and aggregates all four reports without changing its 15% per-entry threshold. The public API and Go 1.27 requirement are unchanged. See [#73](https://github.com/monstercameron/GoWebComponents/pull/73), [#74](https://github.com/monstercameron/GoWebComponents/pull/74), [#75](https://github.com/monstercameron/GoWebComponents/pull/75), [#76](https://github.com/monstercameron/GoWebComponents/pull/76), [#77](https://github.com/monstercameron/GoWebComponents/pull/77), [#78](https://github.com/monstercameron/GoWebComponents/pull/78), [#79](https://github.com/monstercameron/GoWebComponents/pull/79), [#80](https://github.com/monstercameron/GoWebComponents/pull/80), [#81](https://github.com/monstercameron/GoWebComponents/pull/81), [#82](https://github.com/monstercameron/GoWebComponents/pull/82), [#83](https://github.com/monstercameron/GoWebComponents/pull/83), [#84](https://github.com/monstercameron/GoWebComponents/pull/84), [#85](https://github.com/monstercameron/GoWebComponents/pull/85), [#86](https://github.com/monstercameron/GoWebComponents/pull/86), and [#87](https://github.com/monstercameron/GoWebComponents/pull/87).

Representative before/after measurements for the PRs in this release are below. Percentages are **reductions in operation time**, not additive speedups. Most workloads are microbenchmarks or use a mock DOM; they do not measure whole-app or React-relative performance.

| PR | Measured time reduction | Workload |
| --- | ---: | --- |
| [#73](https://github.com/monstercameron/GoWebComponents/pull/73) | No reliable time gain | Stable child order; one allocation removed. |
| [#74](https://github.com/monstercameron/GoWebComponents/pull/74) | 90.6–96.3% | Exact canonical scheduler-origin lookup, native microbenchmark. |
| [#75](https://github.com/monstercameron/GoWebComponents/pull/75) | About 35–40% | Key-only props copy, isolated microbenchmark. |
| [#76](https://github.com/monstercameron/GoWebComponents/pull/76) | 35.5% | 40-fiber × 20-hook-group render under Node/Wasm; resolver alone fell 84.3%. |
| [#77](https://github.com/monstercameron/GoWebComponents/pull/77) | 19.7% native; 19.5% Wasm | Five-property DOM update using a mock DOM. |
| [#78](https://github.com/monstercameron/GoWebComponents/pull/78) | About 27% Wasm | Whole-runtime no-op refresh of a 128-row tree using a mock DOM; native result was noisy. |
| [#79](https://github.com/monstercameron/GoWebComponents/pull/79) | 50.0% native; 49.8% Wasm | 1,000-row exact-order suffix append using a mock DOM. |
| [#80](https://github.com/monstercameron/GoWebComponents/pull/80) | 19.6% Wasm | Whole-runtime 12-card content refresh using a mock DOM; native timing was inconclusive. |
| [#81](https://github.com/monstercameron/GoWebComponents/pull/81) | 72–75% | Direct adjacent-swap repair at 256–1,000 children using a mock DOM. |
| [#82](https://github.com/monstercameron/GoWebComponents/pull/82), [#83](https://github.com/monstercameron/GoWebComponents/pull/83) | Not measured | CI reporting and benchmark-gate changes. |
| [#84](https://github.com/monstercameron/GoWebComponents/pull/84), [#85](https://github.com/monstercameron/GoWebComponents/pull/85) | Not measured | Derived-atom correctness fixes. |
| [#86](https://github.com/monstercameron/GoWebComponents/pull/86) | Not measured | Wasm test fix. |
| [#87](https://github.com/monstercameron/GoWebComponents/pull/87) | 15.6% tracked `SetAtom`; 1.80% native-suite geometric mean | Node/Wasm microbenchmark: 153.2→129.3 ns/op, allocations unchanged. Balanced native CI: 176 entries, 22 improved, 2 regressed below the 15% limit, 152 within noise. |

The final Go 1.27.3 Node/Wasm A/B for #87 used the exact #85 source as its baseline; the balanced native CI compared #87 against main after #86. Neither result measures browser or React-relative performance.

A fair React 19.2.4 comparison on `main` at `40dd11ac` found GoWebComponents faster in 6 of 19 scenarios. The four reported overall React/GWC ratios were 0.863, 0.915, 0.897, and 0.922 (all below 1). In the paired browser comparison, v7's geometric mean was 3.51% slower than v6; the maintainer accepted this measured cost. These results describe the measured runs and do not establish parity with React. See the [fair browser comparison workflow](.github/workflows/fair-example201.yml) and [report visibility change in #82](https://github.com/monstercameron/GoWebComponents/pull/82).

## v7.0.6 — performance patch

`ComputedSignal.Text` no longer copies its private source-ID slice before
reactive text construction filters and joins the IDs. A focused two-source
benchmark fell from 9 allocations and 648 B to 8 allocations and 616 B per
operation. The public defensive-copy behavior of
`ReactiveRegionSourceIDs`, rendered text, API, and Go 1.27 requirement are
unchanged.

The [native drift rerun](https://github.com/monstercameron/GoWebComponents/actions/runs/39668432879)
passed across 164 shared benchmarks with a 0.13% slower geometric mean; its
previously flagged, unchanged hydration entry was 0.77% slower, within noise.
The [same-runner native comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39667994389)
passed across 185 shared entries with a 0.68% slower geometric mean.
The [direct v7.0.5 browser rerun](https://github.com/monstercameron/GoWebComponents/actions/runs/39668858776)
passed all 19 scenarios with a 0.41% faster geometric mean. An earlier
[browser capture](https://github.com/monstercameron/GoWebComponents/actions/runs/39668036316)
was 1.21% slower in aggregate without any scenario beyond its noise limit.
The [same-runner Wasm comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39667999519)
was 0.03% slower across 154 shared entries. Its strict per-entry gate flagged
an unchanged legacy child-placement benchmark at +18.26% against a 15% limit;
that benchmark does not call the changed state code. A repeat measurement was
scheduled to investigate the outlier.

## v7.0.5 — performance patch

Child-order repair now sizes its observed-node scratch slice from the expected
child count, with a minimum capacity of four. A direct 128-child reorder
benchmark against v7.0.4 fell from 12 to 7 allocations and from 11,072 B to
8,960 B per operation. The slice is temporary; a sparse DOM after external
mutation can reserve more space than the number of nodes observed. Child order,
the public API, and the Go 1.27 requirement are unchanged.

The [direct v7.0.4 browser comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39659570762)
passed all 19 scenarios with a 0.85% faster geometric mean. The
[same-runner native comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39660285250)
was 0.18% faster across 185 shared entries; the changed child-order path was
1.86% faster. The [same-runner Wasm comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39660285216)
passed its 15% per-entry gate across 154 shared entries with a 0.69% slower
geometric mean. Its changed child-order path was 0.34% slower, within noise.
One [native drift run](https://github.com/monstercameron/GoWebComponents/actions/runs/39660492852)
flagged the unchanged `FiberSiblingWalk256` benchmark at +20.5%; the dedicated
native comparison measured that entry at +0.81%.

## v7.0.4 — performance patch

Committed child-order repair now keeps each longest-increasing-subsequence tail
value beside its source index in one scratch slice. A direct 128-child reorder
benchmark fell from 4 to 3 allocations per call, with bytes unchanged at
3,200 B. Child movement, the public API, and the Go 1.27 requirement are
unchanged.

Native `gwc bench` runs now use `-trimpath` so checkout path length does not
change stack-capturing benchmark allocations. In a same-source Windows control,
two checkout paths reported 4,046 B and 3,806 B per operation before this
change; both reported 3,286 B with `-trimpath`.

The [Example 201 browser comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39645462626)
passed all 19 scenarios with a 0.95% faster geometric mean. The
[same-runner native comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39645466871)
completed 184 shared entries with a 1.02% slower geometric mean; the changed
child-order path was 1.47% slower, within its noise threshold. The
[same-runner Wasm comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39645471633)
passed its 15% per-entry gate across 153 shared entries with a 0.23% slower
geometric mean.

## v7.0.3 — performance patch

Nested fragments now flatten into one pooled scratch slice instead of borrowing
one slice per nesting level. In a focused eight-level, 16-child benchmark, this
reduced allocation from 192 B and 8 allocations per reconciliation to 24 B
and 1 allocation. The public API, child order, nil slots, and Go 1.27
requirement are unchanged.

Diagnostics stack parsing now removes frame offsets without an intermediate
split slice. Five local runs of `BenchmarkNewReport` consistently moved from
44 to 39 allocations and from 6,783 B to 6,623 B per report. Frame locations
and line numbers are covered by a focused test.

The [combined Example 201 browser comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39631334597)
passed all 19 scenarios with a 1.68% faster geometric mean. The
[same-runner native comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39631225924)
was 2.35% faster across 183 shared entries. Its stack-capturing diagnostics
entries were about 44% slower than the v7.0.2 baseline; a
[fragment-only control](https://github.com/monstercameron/GoWebComponents/actions/runs/39628425661)
reproduced this with unchanged diagnostics source. The
[same-runner Wasm comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39631967281)
passed its 15% per-entry gate across 152 shared entries, with a 0.46% slower
geometric mean.

## v7.0.2 — performance patch

Host elements with event-only or special-property props now avoid allocating
an unused compact attribute view. A focused native benchmark fell from 216 B
and 3 allocations per element to 152 B and 2 allocations; string-only props
were unchanged. The public `CreateElement` props-copy contract, exported API,
and Go 1.27 requirement are unchanged.

The [same-runner Wasm suite](https://github.com/monstercameron/GoWebComponents/actions/runs/39613566908)
passed with 148 matched entries and a 0.21% faster geometric mean. The
[Example 201 browser comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39618939912)
passed all 19 scenarios with a 6.36% faster geometric mean. The
[native drift suite](https://github.com/monstercameron/GoWebComponents/actions/runs/39618937798)
passed all per-entry gates across 158 matches; its overall geometric mean was
2.70% slower in that run, including slower unrelated projection and
virtualization entries. CI also balances browser capture order and gives one
asynchronous router Wasm test a longer scheduling deadline.
