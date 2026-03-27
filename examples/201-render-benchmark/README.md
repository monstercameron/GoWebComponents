# Example 201 Browser Benchmark

This example provides a browser-side benchmark runner that compares:

- React 18
- runtime1, the current GoWebComponents runtime
- runtime2 with one Go WASM worker preparing core and content chunks before the local runtime2 shell commits DOM updates
- runtime2 with four Go WASM workers preparing the same chunks in parallel before the local runtime2 shell commits DOM updates

## What It Measures

The runner drives the same browser-visible scenarios across all four subjects:

- core list render
- core list update
- core view refresh
- content-card render
- content-card update
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
- seeded scenario order and framework order for reproducibility

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

## Important Boundary

The `runtime2` subjects here still keep DOM ownership on the main thread. What moved off-thread in this example is the core and content chunk-preparation step, not final DOM commit, hydration, or hook execution.
