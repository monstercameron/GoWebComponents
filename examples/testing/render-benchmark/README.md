# Example 201 Browser Benchmark

This example provides a browser-side benchmark runner that compares:

- React 19.2.4
- runtime1, the current GoWebComponents runtime
- runtime2 at a worker-scaling matrix of 1, 2, 4, and 8 Go WASM workers preparing the same chunks before the local runtime2 shell commits DOM updates

## What It Measures

The runner drives the same browser-visible scenarios across all benchmark subjects:

- core list render
- core list update
- core stress update
- core view refresh
- core append
- core filter
- content-card render
- content-card update
- content-card refresh
- primitive host-node render
- primitive text update
- primitive attribute update
- primitive append
- primitive remove
- deep-tree render
- deep-tree update
- deep-tree refresh
- enterprise workspace subtree update
- hook-grid render

The comparison is intentionally browser-facing. It measures what the page sees after the subject is already loaded, not only isolated Go or JS microbenchmarks.

The harness now records two finish lines for every scenario:

- `DOM Ready`: the correctness contract for the scenario became true
- `Paint Proxy`: one `requestAnimationFrame` boundary after the DOM-ready checkpoint

`DOM Ready` is the primary timing and is now detected with task-level polling rather than frame-by-frame `requestAnimationFrame` waits, so the measured work no longer collapses into one-frame buckets during Playwright runs.

It also records browser-visible diagnostics per scenario:

- child-list, attribute, and text-mutation counts
- nodes added and removed
- long-task counts and duration when the browser reports them
- worker batch count, last-batch duration, and prepared-item count for the runtime2 subjects
- seeded scenario order plus per-scenario rotated framework order for reproducibility without one framework keeping the same run slot all day

The report also includes an RT2-only worker-scaling section that compares:

- worker batch mean time
- speedup relative to the smallest worker count in the same run
- prepared item count
- DOM-ready and paint-proxy timings for context

The report avoids one mixed overall multiplier across all scenarios. It groups results by category, keeps `DOM Ready` as the uncapped lead timing, and still shows same-run `DOM vs React` deltas as a diagnostic.
Benchmark JSON now also records `DOM Ready` standard deviation, coefficient of variation, spread ratio, and a bimodality-aware `DOM Ready Rep.` value. The representative equals the mean for unimodal samples; if a churn run splits into two stable timing clusters, the report flags the gap and uses the dominant cluster, with equal cluster ties choosing the slower cluster. The original mean, median, and sorted samples remain in the JSON and Markdown report for auditability.
`DOM Score` is now a fixed-reference integer score from `examples/testing/render-benchmark/score-reference.json`, where `100` equals the checked-in Example 201 reference profile rather than whichever framework happened to run in the current report.
The report also separates the worker-relevant overall headline from the full-surface overall score so deep-tree, primitive, hook-grid, and enterprise-workspace subtree scenarios do not distort the runtime2 worker story.

## Run The Example

Build the shared wasm subject:

```powershell
go run ./tools/gwc build -app .\examples\testing\render-benchmark\main.go -root .\examples\testing\render-benchmark -out .\bin\examples\render-benchmark.wasm
```

Build the worker wasm used by the runtime2 subjects:

```powershell
$env:GOOS='js'
$env:GOARCH='wasm'
go build -o .\bin\examples\render-benchmark-worker.wasm .\examples\testing\render-benchmark\backgroundworker
```

Serve the examples catalog:

```powershell
go run ./tools/gwc examples
```

Then open:

- `http://127.0.0.1:8090/examples/testing/render-benchmark/`
- `http://127.0.0.1:8090/examples/testing/render-benchmark/?runtime2WorkerCounts=1,2,4,8` for the default scaling matrix explicitly
- `http://127.0.0.1:8090/examples/testing/render-benchmark/?runtime2WorkerCounts=2,8` to compare only the two-worker and eight-worker runtime2 subjects
- `http://127.0.0.1:8090/examples/testing/render-benchmark/?runtime2Workers=8` as the backward-compatible shorthand for `1` and `8`
- `http://127.0.0.1:8090/examples/testing/render-benchmark/?runtime2WorkerCounts=1,2,4,8&runtime2WorkScale=12` for an RT2-only stress run that amplifies worker-preparation differences
- `http://127.0.0.1:8090/examples/testing/render-benchmark/?subjectSet=runtime2-scaling&runtime2WorkerCounts=1,2,4,8&runtime2WorkScale=12` for the RT2-only stress route without React or runtime1 subjects
- `http://127.0.0.1:8090/examples/testing/render-benchmark/?subjectSet=runtime2-scaling&runtime2WorkerCounts=1,2,4,8&runtime2WorkScale=12&runtime2Dispatch=batch` for the optimized lane-batched worker dispatch path (default)
- `http://127.0.0.1:8090/examples/testing/render-benchmark/?subjectSet=runtime2-scaling&runtime2WorkerCounts=1,2,4,8&runtime2WorkScale=12&runtime2Dispatch=chunk` for the legacy per-chunk request path (useful as a before-baseline)
- `http://127.0.0.1:8090/examples/testing/render-benchmark/?runtime2WorkerCounts=1,2,4,8&runtime2CoreFastPath=off` to force worker RPC on small core batches (disables the default local core fast path for A/B checks)

## Run The Automated Benchmark And Report

```powershell
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample201BrowserBenchmarkReport -v
```

That test:

- builds the wasm subject through `gwc build`
- serves the example through `gwc examples`
- runs the browser benchmark with Playwright Chromium
- uses a fixed seed so scenario order and rotated framework order are explicit in the report
- writes JSON and Markdown reports under `bin/test-results/example-201-browser-benchmark/`

Run the dispatch A/B benchmark route:

```powershell
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample201BrowserBenchmarkDispatchCompare -v
```

## Important Boundary

The `runtime2` subjects here still keep DOM ownership on the main thread. What moved off-thread in this example is the core and content chunk-preparation step, not final DOM commit, hydration, or hook execution.

The default structural-churn matrix currently focuses on append and filter. Prepend, reverse, and sort are not part of the active comparison route yet because the current public runtime2 shell does not preserve those ordering semantics reliably enough for a fair browser benchmark.

The primitive render/update/churn scenarios are included to isolate flat host-operation cost across frameworks. They are main-thread-owned in runtime2 today, so they stay outside the worker-relevant headline score.

The deep-tree update/refresh and enterprise workspace subtree-update scenarios are included to cover enterprise-style nested DOM change patterns, but they are still main-thread-owned in runtime2 today and therefore stay outside the worker-relevant headline score.

`runtime2WorkScale` is useful for RT2-only stress comparisons, but it is not a fair cross-framework default because the extra digest work is specific to the RT2 worker-preparation path.

`runtime2CoreFastPath` defaults to enabled and lets small core batches (`<= 64` items) bypass worker RPC through a local cache-backed preparation path to cut update latency; set `runtime2CoreFastPath=off` when you want raw worker-dispatch timing.
