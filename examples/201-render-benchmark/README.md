# Example 201 Browser Benchmark

This example provides a browser-side benchmark runner that compares:

- React 18
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
- deep-tree render
- hook-grid render

The comparison is intentionally browser-facing. It measures what the page sees after the subject is already loaded, not only isolated Go or JS microbenchmarks.

The harness now records two finish lines for every scenario:

- `DOM Ready`: the correctness contract for the scenario became true
- `Paint Proxy`: one `requestAnimationFrame` boundary after the DOM-ready checkpoint

It also records browser-visible diagnostics per scenario:

- child-list, attribute, and text-mutation counts
- nodes added and removed
- long-task counts and duration when the browser reports them
- worker batch count, last-batch duration, and prepared-item count for the runtime2 subjects
- seeded scenario order and framework order for reproducibility

The report also includes an RT2-only worker-scaling section that compares:

- worker batch mean time
- speedup relative to the smallest worker count in the same run
- prepared item count
- DOM-ready and paint-proxy timings for context

The report avoids one mixed overall multiplier across all scenarios. It groups results by category and uses category-local geometric-relative scores for the paint-proxy timing instead.

## Run The Example

Build the shared wasm subject:

```powershell
go run ./tools/gwc build -app .\examples\201-render-benchmark\main.go -root .\examples\201-render-benchmark -out .\bin\examples\render-benchmark.wasm
```

Build the worker wasm used by the runtime2 subjects:

```powershell
$env:GOOS='js'
$env:GOARCH='wasm'
go build -o .\bin\examples\render-benchmark-worker.wasm .\examples\201-render-benchmark\backgroundworker
```

Serve the examples catalog:

```powershell
go run ./tools/gwc examples
```

Then open:

- `http://127.0.0.1:8090/examples/201-render-benchmark/`
- `http://127.0.0.1:8090/examples/201-render-benchmark/?runtime2WorkerCounts=1,2,4,8` for the default scaling matrix explicitly
- `http://127.0.0.1:8090/examples/201-render-benchmark/?runtime2WorkerCounts=2,8` to compare only the two-worker and eight-worker runtime2 subjects
- `http://127.0.0.1:8090/examples/201-render-benchmark/?runtime2Workers=8` as the backward-compatible shorthand for `1` and `8`
- `http://127.0.0.1:8090/examples/201-render-benchmark/?runtime2WorkerCounts=1,2,4,8&runtime2WorkScale=12` for an RT2-only stress run that amplifies worker-preparation differences
- `http://127.0.0.1:8090/examples/201-render-benchmark/?subjectSet=runtime2-scaling&runtime2WorkerCounts=1,2,4,8&runtime2WorkScale=12` for the RT2-only stress route without React or runtime1 subjects
- `http://127.0.0.1:8090/examples/201-render-benchmark/?subjectSet=runtime2-scaling&runtime2WorkerCounts=1,2,4,8&runtime2WorkScale=12&runtime2Dispatch=batch` for the optimized lane-batched worker dispatch path (default)
- `http://127.0.0.1:8090/examples/201-render-benchmark/?subjectSet=runtime2-scaling&runtime2WorkerCounts=1,2,4,8&runtime2WorkScale=12&runtime2Dispatch=chunk` for the legacy per-chunk request path (useful as a before-baseline)

## Run The Automated Benchmark And Report

```powershell
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample201BrowserBenchmarkReport -v
```

That test:

- builds the wasm subject through `gwc build`
- serves the example through `gwc examples`
- runs the browser benchmark with Playwright Chromium
- uses a fixed seed so scenario and framework order are explicit in the report
- writes JSON and Markdown reports under `bin/test-results/example-201-browser-benchmark/`

Run the dispatch A/B benchmark route:

```powershell
go test -tags playwrightgo ./test/playwrightgo/examples -run TestExample201BrowserBenchmarkDispatchCompare -v
```

## Important Boundary

The `runtime2` subjects here still keep DOM ownership on the main thread. What moved off-thread in this example is the core and content chunk-preparation step, not final DOM commit, hydration, or hook execution.

The default structural-churn matrix currently focuses on append and filter. Prepend, reverse, and sort are not part of the active comparison route yet because the current public runtime2 shell does not preserve those ordering semantics reliably enough for a fair browser benchmark.

`runtime2WorkScale` is useful for RT2-only stress comparisons, but it is not a fair cross-framework default because the extra digest work is specific to the RT2 worker-preparation path.
