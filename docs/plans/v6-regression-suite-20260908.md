# Full regression check — 2026-09-08

Requested after the integration handoff. This is a new run, not a reuse of the
previous acceptance results. No implementation or test fixes were made.

## Completed results

| Check | Result |
| --- | --- |
| Uncached root suite, Windows amd64, CGO off, serial packages | Failed: `docs/doclint` and `fetch`; other root packages passed |
| Nested `desktop/wails` native suite | Passed, 0.211s |
| Nested desktop example native suite | Passed |
| Nested livereload and agenthub suites | Passed, 1.748s / 2.541s |
| Full discovered Wasm lane | Failed: two CSS tests and API baseline filesystem access |
| Hydration lane, separately run after Wasm failure | Passed |
| Desktop Wasm suite with desktop mode tag | Passed, 0.923s |
| Adapter unsupported-platform Wasm suite | Passed, 0.205s |
| Desktop frontend Wasm suite | Passed, 0.243s |
| Desktop real dev/idle integration | Passed, 17.936s |
| Main Playwright-Go browser package | Passed, 72.648s |
| Full GWC CLI suite with browser tag | Passed, 182.386s |
| Extended browser examples | Failed assertions; package exceeded 20-minute limit, 1200.374s |
| Kernel-plugin devtools browser example | Failed: configured example source path missing |
| JavaScript transport, smoke-wait, extension and load-harness tests | Failed only in load-harness integration drift assertion |
| Race suite | Blocked: no working Windows CGO compiler configuration |

Root command: `go test -p 1 ./... -count=1`, with process-local Windows/amd64,
CGO disabled and empty GOFLAGS. The CLI package passed in 86.665s.
Browser command: `go test -tags playwrightgo -p 1 ./test/playwrightgo/... -count=1 -timeout 20m`.
CLI browser command: `go test -tags playwrightgo ./tools/gwc -count=1 -timeout 15m`.
The browser command exited 1. The examples package timed out as
`TestExample201LongFrameParity` started compiling; this is incomplete coverage,
not proof that that individual test requires 20 minutes. Later tests in the
package did not run. The next browser package still ran and failed independently.

## Reproduced failures

1. `TestDocsGwcFlagsExist`: the documentation scanner treats the valid Go test
   command in `v6-astra-final-verification.md:216` as a GWC CLI invocation and
   rejects Go's `-run`, `-tags`, and `-v`. Its substring match on `tools/gwc`
   does not distinguish `go test` from `go run`.
2. `TestRealtimeControllerHeartbeatTimeoutClosesAndReconnects`: timed out waiting
   for exactly one transport close. Three isolated repeats produced one failure
   followed by two passes. The original full-suite failure remains a failure.
3. Wasm `TestFastFoldRespectsConflictOrdering` and `TestFastFoldIsClearedByReset`:
   expected emitted CSS but received an empty style block. Both reproduce in
   isolation. The Node runner has no DOM; the Wasm sink explicitly emits nothing
   and Harvest returns empty without a document. Native CSS tests passed.
4. Wasm `TestPublicAPIBaseline`: `syscall.Open: O_DIRECTORY is not supported on
   Windows`, reproduced in isolation. Native API-baseline tests passed.
5. Load-harness `SEAM: buildReport marks a drifted run invalid and gate fails it`:
   consistently gets valid=true. The fixture drifts the loaded arm, whereas
   `buildReport` explicitly gates only on idle-control drift. This is a
   test/implementation contract mismatch, not evidence of a desktop API failure.

Race testing first rejected CGO=0. Retrying with CGO=1 and the installed clang
failed because its MSVC target rejects `-mthreads`. The test-owned race process
was stopped after this repeated prerequisite error; no race pass is claimed.

Passing automated tests do not constitute native visual/manual acceptance.

## Extended browser findings

Full captured package output: [browser log](../../bin/test-results/v6-full-regression-browser-20260908-1330.log).

- Atlas: unhandled `Transition was skipped`, missing saved-view import worker
  returning HTTP 404, settings persistence/expected dashboard content mismatches,
  and recovery-route destination timeouts.
- Example 100: multiple admin, auth, settings, chat/sidebar and startup cases
  failed, including missing UI state, RPC failures and timeouts. These have not
  been individually diagnosed or attributed to the desktop integration.
- Example 200 runtime execution and Example 201 dispatch comparison failed;
  the latter had zero worker metric samples for both compared modes.
- Kernel-plugin devtools could not build because
  `examples/testing/kernel-plugin-devtools/main.go` does not exist.

The example suite is much broader than desktop integration. Failures here are
observations against the current working tree and available built fixtures, not
a claim that all were introduced by Wails. This run did not rebuild every legacy
example fixture or execute upstream Wails' own test suite.

After the timeout, a fresh process-tree query found no remaining descendants of
the test-owned examples process. Existing unrelated development servers were not
terminated. The Atlas tracked example database is modified after the tests and
was preserved rather than blindly reset. No application/test fixes were made.
