# Astra framework-only verification — 2026-09-08

This checkpoint covers framework source, tools, and adapter libraries only. No
example application was edited, built, or run during this verification phase.
Passing tests are not a claim of 100% code coverage, physical desktop UI
acceptance, or support for historical retired renderers.

## Independent findings and changes

- Doclint's tightened command detector missed valid `./gwc`, `gwc.exe`, and
  `go run -tags ... ./tools/gwc` forms, while accepting `tools/gwc-extra`.
  Added reproducing positive/negative command tests, then used shell-field
  parsing and an exact launcher/package match. Go compiler flags before the
  package are not mistaken for launcher flags. Actual unknown launcher flags
  still fail the audit.
- Reviewed embedded API extraction against filesystem extraction. Added parity,
  platform-union, ignored test-file, malformed-source, and signature-drift
  regressions. Restored the native `UPDATE_API_BASELINE=1` regeneration path;
  Wasm explicitly directs regeneration to native Go. The baseline was not
  regenerated or relaxed. A transient two-argument `Check` compile error during
  editing was corrected to the actual three-argument API before final runs.
- CSS's Wasm test sink uses the public sink interface and retains emitted-CSS
  assertions. Strengthened repeat stability to require the initial color to
  actually be emitted, so an empty sink cannot produce a false pass.
- Desktop's expired-context test raced the delivery of a one-nanosecond timer.
  It now waits for `ctx.Done()` with a one-second failure bound before asserting
  the deadline error. Production deadlines and assertions are unchanged.
- Runtime race evidence exposed a real native SSR defect: concurrently resolving
  stream boundaries mutated the process-global current fiber and hook owner.
  Native SSR now installs scoped, goroutine-specific transient fibers, including
  nested-render restoration and panic/suspension cleanup. Runtime and atom-scope
  lookups use that context. Browser ambient-fiber behavior is unchanged. A forced
  overlap test verifies two request identities, the same atom key with different
  values, and exact per-request HTML; a second test checks nested/panic cleanup.
  This uses the existing runtime-stack identity technique, now also needed for
  native production correctness; no performance-budget claim is made.
- Two runtime test harness races were distinct from that production bug:
  asynchronous boundary retries appended to an unprotected test scheduler queue,
  and the nonce test polled `bytes.Buffer` while rendering wrote it. Queue
  publication/draining is synchronized; the nonce test waits on actual emitted
  chunks and reads the buffer only after completion. Assertions are preserved.
- Reviewed native `ui.State`'s separate mutation and value locks: updater `Get`
  is safe, concurrent mutations serialize, and panic cleanup releases the lock.
  Documented that recursive mutation of the same state from its updater is not
  supported.
- Final realtime review found that atomics alone did not prevent a constructor
  from publishing after stop, a completing heartbeat from reinstalling timers,
  or stale timer generations from operating on a restarted connection. Lifecycle
  transitions now serialize; external constructor/send/close calls release that
  lock to allow synchronous callbacks, then validate their generation before
  committing results. Deterministic blocked-constructor and blocked-heartbeat
  tests join late completions before asserting closed state and no timers. Old
  transport callbacks and the exact timer entry methods are exercised after a
  restart. A constructor synchronously invokes its open callback in the fixture
  to catch lock reentrancy failures.

## Commands directly executed

Unless stated otherwise, root commands use `GOOS=windows GOARCH=amd64
CGO_ENABLED=0`; Wasm commands use `GOOS=js GOARCH=wasm CGO_ENABLED=0` and
`-exec C:/Users/mreca/Desktop/GoWebComponents/tools/go_js_wasm_exec.bat`.
All PASS rows below had exit code 0.

| Command | Result |
| --- | --- |
| `go test ./internal/apidump ./validate ./css ./docs/doclint -count=1` | PASS; 0.829 / 0.864 / 1.214 / 1.272 seconds |
| `go test ./desktop -count=1` | PASS; 0.791 seconds after deterministic deadline precondition fix |
| `go vet ./desktop ./internal/apidump ./validate ./css ./docs/doclint` | PASS |
| `node --test desktop/desktop.test.mjs` | PASS; 6 tests, zero skips |
| Wasm `go test -exec <runner> ./desktop ./css ./validate -count=1` | PASS; 1.158 / 0.462 / 0.437 seconds |
| Wasm `go test -tags gwc_desktop -exec <runner> ./desktop -count=1` | PASS; 1.157 seconds |
| In isolated `desktop/wails`: native `go test ./... -count=1` and `go vet ./...` | PASS; tests 0.290 seconds |
| In isolated `desktop/wails`: Wasm `go test -exec <runner> ./... -count=1` | PASS; 0.293 seconds; unsupported native backend contract, not OS dialog execution |
| Wasm `go test -tags production -exec <runner> ./hotreload ./utils -count=1 -v` | PASS; 0.187 / 0.193 seconds; covers complementary production-only cases |
| Wasm `go test -exec <runner> ./internal/runtime -count=1` | PASS; 3.532 seconds after SSR changes |
| `go test -tags production ./internal/runtime -run SSR -count=1` | PASS; 0.210 seconds |
| `go vet ./internal/runtime` | PASS |
| ARM64 native `go test ./fetch -count=1 -timeout=60s` followed by `go vet ./fetch` | PASS; tests 5.857 seconds, final command exit 0 |
| Wasm `go test -exec <runner> ./fetch -count=1 -timeout=60s` | PASS; 1.168 seconds |
| `git diff --check -- internal/runtime docs/doclint css validate internal/apidump desktop` | PASS |

### Race-instrumented runtime execution

Used `GOOS=windows GOARCH=amd64 CGO_ENABLED=1 GOFLAGS=` and compiler
`bin/toolchains/llvm-mingw-20260826-ucrt-x86_64/bin/x86_64-w64-mingw32-clang.exe`.

1. `go test -c -race -o ./bin/framework-runtime-race.test.exe ./internal/runtime`
   compiled successfully.
2. From `internal/runtime`, ran the absolute binary path with quoted arguments
   `'-test.count=1' '-test.timeout=90s'`. All tests passed, exit 0, approximately
   5.9 seconds wall time; no race reports. Full output is retained in
   `bin/test-results/v6-framework-runtime-race-binary-final.log`.

The final fetch revision was also compiled with the same compiler/environment:
`go test -c -race -o ./bin/framework-fetch-race.test.exe ./fetch` (exit 0), then
executed from `fetch` using its absolute path with `'-test.count=1'
'-test.timeout=60s'`. The entire package passed, exit 0, approximately 7.88 seconds
wall time, including deterministic overlap regressions. No race reports;
`bin/test-results/v6-framework-fetch-race-binary-final.log` retains the output.

The normal full native and race `go test` attempts printed package success
(2.607 and 6.117 seconds respectively) but exited **1** when Go could not unlink
its temporary test executable: Windows `Access is denied`. These are not listed
as successful commands. Logs: `v6-framework-runtime-native-review.log` and
`v6-framework-runtime-race-review.log` in `bin/test-results`. The cause of that
Windows cleanup failure is not established. Explicit build/run avoids deleting
the just-executed test file and preserves an auditable artifact.

Other unsuccessful invocation attempts were corrected and not counted: generic
MSVC-target clang rejected `-mthreads`; an unquoted PowerShell dotted test flag
was split into `-test`; a subsequent relative binary invocation from the package
directory did not locate the binary. The final absolute-path invocation above
actually executed the tests and returned 0. A Wasm command containing regex
alternation was misinterpreted by the Windows batch runner; the full production
package rerun without regex metacharacters passed.

## Skip audit of parent-owned JSON matrix

At review time, `v6-framework-native-final.jsonl` had 93 passing packages and one
package-level no-test entry (`docs/REFERENCE_MANUAL`), not a failed test.
`v6-framework-wasm-final.jsonl` had 28 passing packages. These matrix runs preceded
the final runtime correction; root owns the consolidated final reruns.

- Three ordinary Wasm skips are production-only hotreload/utils tests. The
  complementary production-tag package run above executes and passes them.
- Four Wasm skips are two HTML event-option hook tests and two UI preference
  hook tests that use native-only outside-component fixtures. Their native
  counterparts pass; this does not itself prove browser preference interaction.
- Two CLI symlink-root/output tests skip because Windows symlink creation is not
  available to the test process. Their security assertions were not removed.
  These remain a platform-prerequisite coverage gap unless a capable lane runs.
- `projection/TestM12GCPauseIsMeasuredButNativeOnly` reported zero observable GC
  pause. Review found a circular-buffer index one behind the newly observed
  collections; root took ownership of that measurement fix and final rerun.
  Do not count an unusable timing observation as a passed performance budget.

The earlier CLI browser JSON run subsequently finished with failures in
`TestGeneratedScaffoldServesOverDevServer`,
`TestGeneratedScaffoldLauncherBuildTestVerifyReleaseRoundTrip`, and
`TestRunVerifyJSONRunsTestsAndCIBuild` (package elapsed 404.713 seconds). Root
identified shared scaffold directories between concurrent test processes and
changed fixtures to `t.TempDir`; Windows executable-cleanup failures were also
present. These failures remain evidence, not retroactive passes. Root owns the
fresh ARM64 native/browser matrix and retained-binary race reruns.

The projection ring index was corrected by root; a fresh ARM64 attempt still
reported zero observable pause, so the timing limitation remains genuine.

All independently owned final focused commands above pass. A consolidated
all-green framework verdict still requires the final root matrix and explicit
prerequisite skip disposition; it is not inferred from those focused passes.

## Final-gate follow-up: cleanup and runner lifecycle

The subsequent full native clean-environment log
`bin/test-results/v6-framework-native-cleanenv-final.jsonl` contains 93 passing
packages and the one package with no test files (94 framework packages total).
Its two symlink cases were independently executed under a capable Windows
process: `v6-elevated-symlink-tests.jsonl` has a passing package and zero skips.
Root added a real GC-pause histogram upper-bound fallback for the zero-resolution
MemStats case, with synthetic rejection tests and the original threshold intact;
review found no weakened budget or fabricated timing reading.

The full retained race matrix had one failure: hookcheck `TestCmdEndToEnd`
assertions completed, but `t.TempDir` could not remove its child executable.
Inspection found both child runs use `CombinedOutput`, which waits and reaps the
process; no missing `Wait` was found. Go's own `TempDir` implementation already
retries Windows access-denied removals for two seconds. No arbitrary additional
sleep, cleanup omission, or relaxed assertion was introduced.

Fresh `GOENV=off GOFLAGS= GOOS=windows GOARCH=arm64 CGO_ENABLED=0`
`go test -work ./tools/hookcheck/cmd/hookcheck -count=5 -timeout=90s` passed,
exit 0, 15.936 seconds. The package was then compiled with the established AMD64
race compiler into `bin/framework-hookcheck-race.test.exe`; executed from its
package directory with ARM64 child-Go environment and `'-test.count=10'
'-test.timeout=90s' '-test.v'`, all ten repetitions passed, cleanup included,
exit 0. Output: `bin/test-results/v6-hookcheck-race-repeat.log`. The original
sporadic Windows deletion failure remains unexplained, not retroactively passed.

The frozen Wasm matrix's validate process printed `PASS` but failed to exit and
was killed after eleven minutes. No validate test installs JS timers or custom
exit handlers; Go's official runner wires `go.exit` to Node's actual process
exit. The repository batch wrapper adds a `cmd.exe` intermediary; after the
outer timeout, its original Node child was still visible in the process inventory.
That is a runner process-tree limitation, not evidence that the tests completed
successfully. It does not establish why the original Node process hung.

With `GOENV=off GOFLAGS= GOOS=js GOARCH=wasm CGO_ENABLED=0`, validate
`-count=10 -timeout=20s` passed through the batch wrapper (1.195 seconds) and
through the direct official runner (0.240 seconds), both exit 0. The latter uses
`-exec '"C:/Program Files/nodejs/node.exe" "C:/Program Files/Go/lib/wasm/wasm_exec_node.js"'`.
Root selected this direct runner for the final 28-package rerun, so Go owns the
actual Node child rather than an intermediate shell. No success-text detection
or forced successful exit is used. Final full matrix reconciliation remains
read-only once these executions finish.

## Acceptance event reconciliation

Independently parsed the final JSON events, counting test/subtest events rather
than claiming these are unique functions or coverage percentages:

| Log in `bin/test-results` | Completed result |
| --- | --- |
| `v6-framework-native-acceptance.jsonl` | 93 tested packages pass + 1 package with no tests; 4,517 test/subtest pass events, zero failures |
| `v6-framework-wasm-acceptance.jsonl` | 28 packages pass; 2,169 test/subtest pass events, zero failures |
| `v6-framework-production-acceptance.jsonl` | 2 packages, 10 pass events, zero failures/skips |
| `v6-elevated-symlink-tests.jsonl` | Both named symlink tests pass, zero failures/skips |

The native lane's two symlink skips match those two elevated pass events.
The seven Wasm skips are all explicitly reconciled:

- `hotreload/TestEnableIsDisabledInProduction`,
  `hotreload/TestProductionCompatibilityAPIsRemainNoOps`, and
  `utils/TestEnableHotReloadIsDisabledInProduction` pass in the production log.
- `html/TestEventOptionHelpersWrapHandlers` and
  `html/TestExpandedEventOptionHelpersEmitNativePropsAndPassive` pass in the
  final native log.
- `ui/TestPreferenceHooksUnavailableDefaults` and
  `ui/TestPreferenceHooksReadCurrentMatches` pass in the final native log.

The corrected `projection/TestM12GCPauseIsMeasuredButNativeOnly` passes in the
final native log without skipping. This remains a native advisory measurement,
not the distinct browser performance-budget acceptance.

A new repeated agenthub run exposed concurrent writes by the test client. The
final reviewed change synchronizes both diagnostic and acknowledgement data
frames with a test-local mutex; all report/redaction/command assertions remain.
Connection timing is also snapshotted before hello and stored on each Session,
so long-lived ping/read/pong callbacks never reread timing globals modified by
subsequent tests. The sole production `runFrameLoop` caller initializes all
three durations. No skips or relaxed assertions were introduced by that repair.

Final browser, agenthub, and race acceptance results are still being captured by
root at this checkpoint. The race log includes an additional no-test
`bin/test-results` diagnostic package discovered while a now-removed measurement
probe existed. It is excluded from the 94 framework package scope, explicitly,
not silently counted as a passing framework test.

## Final independent verdict

**PASS for the enumerated framework verification matrix.** The final layered
results below have zero unresolved failing tests. Every conditional test skip in
these logs has a named passing counterpart as documented above. This is not a
claim of 100% code coverage, successful execution of every build-tag combination,
example application acceptance, or physical native UI validation.

| Final scope | Actual passing evidence |
| --- | --- |
| Root native | 93 tested packages + 1 no-test package; 4,517 passing test/subtest events |
| Root race, reconciled per package | Same 93 tested packages + 1 no-test package; 4,517 passing test/subtest events |
| Wasm | 28 packages; 2,169 passing test/subtest events |
| Browser (`v6-framework-browser-acceptance.jsonl`) | Both packages pass; 1,005 passing test/subtest events |
| Production Wasm complement | 2 packages; 10 passing events; zero skips |
| Elevated symlink complement | Both tests pass; zero skips |
| Agenthub final native / race | 25 / 25 passing events; zero skips |
| Livereload native / race | 74 / 74 passing events; zero skips |
| Isolated Wails adapter native / Wasm | 3 / 2 passing events; zero skips |
| JavaScript SDK + developer-tool tests | 16 tests pass; zero skips |

The JavaScript command was independently repeated after source freeze:
`node --test desktop/desktop.test.mjs tools/vscode-gwc/test/diagnostics.test.mjs tools/devtools-extension/test/bridge.test.mjs`.
It exited 0: six SDK tests and ten developer-tool tests, not sixteen SDK tests.

### Exact race reconciliation (not a retroactive successful broad command)

`v6-framework-race-acceptance.jsonl` remains a **failed command**: 91 tested
packages passed; `tools/gwc` and `tools/hookcheck/cmd/hookcheck` failed Windows
temporary-executable cleanup. Two no-test package entries were present:
`docs/REFERENCE_MANUAL` and the transient diagnostic `bin/test-results` package.
The latter is explicitly outside framework scope.

For the final per-package race union, retained the broad log's events for all
packages except those two failed packages and the diagnostic package, then used:

- `v6-framework-cli-race-final.jsonl`: the freshly compiled, retained
  race-instrumented `tools/gwc` binary passed, exit 0, 181.990 seconds;
  975 test/subtest pass events, zero failure events, two symlink skips.
- `v6-framework-hookcheck-race-final.jsonl`: retained race-instrumented
  hookcheck test binary passed, exit 0, 2.362 seconds; its one test passed,
  zero skips.

The retained binaries remain AMD64 race-instrumented. Their environment uses
`GOENV=off GOARCH=arm64 CGO_ENABLED=0` for separately launched child Go commands,
which prevents outer cross-target settings from selecting emulated executables
for native scaffold baselines. This does not change instrumentation compiled
into the parent test binaries. Ordinary `go test -race` does not implicitly
instrument arbitrary child `go build` commands either. Two inconsistent scaffold
test subprocesses now use the existing `buildNativeGoEnv` helper; baseline
assertions, explicit Wasm builds, and temporary-directory cleanup are retained.
`GOENV=off` is material: removing process variables alone does not override a
persisted Go environment.

Independently computed union: **93 package passes, one no-test framework package,
4,517 test/subtest passes, zero failure events, and exactly the two symlink skips**.
Its package identity set exactly matches the final native framework scope.
Both skipped symlink tests pass in the elevated complement. The seven Wasm skips
are matched by the three production and four native named passes; there are no
unexplained skipped test cases left in this audited matrix.

Earlier failures, including the original agenthub concurrent-write panic, the
Wasm process that hung after printing success, and Windows cleanup errors, remain
in their original logs and in this report. They were addressed by reviewed fixes
or explicit successful reruns, not by deleting tests, skipping failures, relaxing
thresholds, suppressing cleanup, or interpreting `PASS` text as process success.
