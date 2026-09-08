# Independent regression repair — 2026-09-08

Scope: load-harness JavaScript seams, the historical Example 200 and kernel-plugin
browser suites, and Example 201's active benchmark and retired worker options.
No Example 100, Atlas, CSS, fetch or doclint files were changed in this lane.

## Root causes and corrections

- The load-harness integration fixture put drift in the loaded arm but expected
  invalidity. Production intentionally gates validity on the idle control only.
  The corrected seams test both cases: loaded drift is a valid measurement that
  fails M1, whereas idle drift fails validity. Metric assertions were retained.
- Example 200 and kernel-plugin-devtools were not moved. Commit
  `eeda966e6abd04249e7b96c8330c08fb110a3658` deliberately retired runtime2 and
  removed both examples in v5 P5.2. Their old browser execution assertions targeted
  deleted renderer behavior. With coordinator approval, these are now explicitly
  named migration tests asserting absent retired sources and actual browser 404
  responses. This is not evidence that the deleted renderer executes successfully.
- Commit `37b21540d3fad75f1b67160d50d07e4e25884e6b` restored the React/runtime1
  comparison without runtime2 arms. A legacy runtime2-scaling request nevertheless
  produced a successful report with zero frameworks, scenarios and samples.
  The runner now rejects retired subject/worker options visibly, rejects the
  promise, and clears any stale success report. Five real-browser cases cover
  chunk, batch, worker counts 1/4 and 2/8, and the legacy single-count option.
- The current React/runtime1 benchmark remains real measured coverage. Every
  reported framework must contain every scenario, nonempty timing samples matching
  the iteration count, and no invented runtime2 worker metrics. Generated report
  prose distinguishes current subjects from historical worker fields.
- The slicing diagnostic's missing-development-instrumentation branch now fails
  instead of skipping. Missing required runtime evidence cannot make the suite green.

## Completed focused validation

Native browser-test processes used process-local `GOOS=windows`, `GOARCH=amd64`,
`CGO_ENABLED=0`; their helpers build actual js/wasm artifacts as needed.

```powershell
node examples/testing/v5-load-harness/integration.test.mjs
node examples/testing/v5-load-harness/stats.test.mjs
node examples/testing/v5-load-harness/gate.test.mjs
go test -tags playwrightgo ./test/playwrightgo/examples ./test/playwrightgo/kernelplugindevtools -run 'Test(Example200|Example201Retired|KernelPlugin)' -count=1 -v -timeout 3m
go test -tags playwrightgo ./test/playwrightgo/examples -run 'Test(Example201BrowserBenchmarkReport$|Example201ScoreReference|BuildExample201|FormatExample201)' -count=1 -v -timeout 5m
```

All commands exited zero. JavaScript: 11 integration, 19 statistics, 19 gate
checks. Actual browser retirement checks: Example 200 9.54s; kernel plugin 9.18s;
five unsupported benchmark configurations 2.82s. Actual supported benchmark plus
reference/statistics/formatting checks: package 27.546s, browser measurement 27.22s.
Generated benchmark JSON/Markdown are under
`bin/test-results/example-201-browser-benchmark/browser-benchmark-report.*`.

The complete Example 201 rerun then exited zero with no skips:

```powershell
go test -tags playwrightgo ./test/playwrightgo/examples -run '^TestExample201' -count=1 -v -timeout 10m
go vet -tags playwrightgo ./test/playwrightgo/examples ./test/playwrightgo/kernelplugindevtools
```

All 11 Example 201 tests passed; package time 194.114s. This includes current
benchmark reporting, retirement guards, boot, GC tuning, long-frame parity, phase
totals, CPU profiling, slicing and both spike diagnostics. Vet exited zero with
no findings. The slicing diagnostic actually observed all 40 items after progress,
rather than skipping missing instrumentation.

Performance observations remain observations: the long-frame parity diagnostic
recorded long frames in both subjects, and the production spike diagnostic recorded
a 225ms outlier. These diagnostic tests establish successful measurement/execution,
not compliance with a zero-long-frame or performance release budget. No tests were
skipped or budgets relaxed by this repair. Historical renderer execution assertions
were replaced only where deliberate retirement made them inapplicable, with explicit
coordinator approval and the migration contracts described above.

This lane is green. It does not claim that the entire repository suite passed;
that final broader matrix is owned by the coordinator.
