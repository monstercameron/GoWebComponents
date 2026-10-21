# Latest Release Notes

This docs-site page mirrors the first section returned by
`tools/changelogcheck.LatestEntry(CHANGELOG.md)`.

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
