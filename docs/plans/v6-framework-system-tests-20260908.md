# V6 framework/system test acceptance — 2026-09-08

Status: **PASS for the framework/system matrix below.** Every conditional test
skipped in one lane has a named passing execution in its applicable lane.
This is a test-result statement, not a claim of 100% code coverage or desktop
manual/release acceptance. Example-application repairs remained paused.

## Scope and final evidence

The root scope is 94 framework, internal, docs, test/testkit and developer-tool
packages. One documentation package has no test files. Application packages
under `examples/` and `research/`, upstream Wails tests, and temporary diagnostic
files under `bin/` are outside this scope. Nested first-party modules are tested
separately. Counts below include passing tests and subtests; lanes overlap.

| Lane | Final result | Evidence under `bin/test-results/` |
| --- | --- | --- |
| Native framework | 93 tested packages, 4,517 passes, zero failures; one no-test package | `v6-framework-native-acceptance.jsonl` |
| Race framework | Same 93 tested packages and 4,517 passes in the final per-package union | `v6-framework-race-acceptance.jsonl`, corrected package runs below |
| Wasm framework | 28 packages, 2,169 passes, zero failures | `v6-framework-wasm-acceptance.jsonl` |
| Browser framework + CLI | 2 packages, 1,005 passes, zero failures | `v6-framework-browser-acceptance.jsonl` |
| Production Wasm complements | 2 packages, 10 passes, zero failures/skips | `v6-framework-production-acceptance.jsonl` |
| Elevated Windows symlink checks | 2 passes, zero failures/skips | `v6-elevated-symlink-tests.jsonl` |
| Agent hub native / race | 25 / 25 passes, zero failures/skips | `v6-framework-agenthub-fixed-acceptance.jsonl`, `v6-framework-agenthub-race-acceptance.jsonl` |
| Livereload native / race | 74 / 74 passes, zero failures/skips | `v6-framework-livereload-acceptance.jsonl`, `v6-framework-livereload-race-acceptance.jsonl` |
| Isolated Wails adapter native / Wasm | 3 / 2 passes, zero failures/skips | `v6-framework-adapter-native-acceptance.jsonl`, `v6-framework-adapter-wasm-acceptance.jsonl` |
| Desktop SDK + developer-tool JavaScript | 16 passes, zero failures/skips | `node --test desktop/desktop.test.mjs tools/vscode-gwc/test/diagnostics.test.mjs tools/devtools-extension/test/bridge.test.mjs` |

The race union is explicit: the broad run retained failures in `tools/gwc` and
`tools/hookcheck/cmd/hookcheck` from Windows executable cleanup after assertions.
Those failed attempts are **not** retroactively marked successful. Their final
race-instrumented package reruns passed:

- `v6-framework-cli-race-final.jsonl`: 975 passes, zero failures, 181.990 seconds.
- `v6-framework-hookcheck-race-final.jsonl`: one pass, zero failures, 2.362 seconds;
  the same package also passed ten earlier race repetitions with cleanup intact.

Together with the other 91 passing tested packages, these complete the 93-package
race test union. The broad log also contains a no-test temporary GC probe package;
the probe was removed after diagnosis and is not counted as framework coverage.

## Conditional tests were executed, not waived

- The two symlink safety tests skip in ordinary Windows processes. Both executed
  and passed in the user-approved administrator PowerShell run. Neither Windows
  security settings nor application privilege defaults were changed.
- The three production-only hotreload/utils tests passed in the production Wasm
  lane. Four native-only HTML/UI hook tests passed in the native lane. These are
  the seven intentional conditional skips in the ordinary Wasm log.
- The GC measurement no longer skips on a zero-resolution clock. It uses actual
  runtime pause-histogram samples and conservatively sums their upper bounds;
  missing/unusable measurements fail. The original budget is unchanged. Repeated
  focused tests exercised both the ordinary measurement and fallback paths.

## Corrections made

- Isolated native SSR rendering fibers and atom scopes across concurrent requests.
- Serialized realtime lifecycle transitions, rejected late transport publication,
  and made timer callbacks generation-aware; added deterministic overlap tests.
- Allowed native state updater callbacks to read state without deadlocking while
  preserving serialized updates and panic-safe lock release.
- Corrected docs command invocation parsing, filesystem-independent Wasm API
  checks, CSS test sinks, and asynchronous test counter/scheduler/buffer access.
- Isolated scaffold fixtures per test, normalized native child-test environments,
  and terminated owned dev-server listeners before cancelling launchers.
- Serialized agent-hub test WebSocket writes and captured keepalive timing per
  session, preventing active goroutines from reading mutable test defaults.

No assertions, safety checks, cleanup checks, or performance thresholds were
removed to obtain these results. Fifty-four confirmed obsolete scaffold-test
processes and the timed-out validation Node process were stopped; no user files
were removed in that cleanup.

## Verification environment

Native tests used Windows ARM64, `GOENV=off`, empty `GOFLAGS`, and `CGO_ENABLED=0`.
These were process-local overrides; saved Go settings were not modified.

Race binaries were built for Windows AMD64 with CGO and the verified portable
LLVM-MinGW compiler under `bin/toolchains/`. Retained CLI/hookcheck race binaries
were executed with ARM64 settings for child Go commands. Their already-compiled
race instrumentation remains enabled; no race detection was suppressed.

Wasm tests used Go's official Node runner directly:
`-exec '"C:/Program Files/nodejs/node.exe" "C:/Program Files/Go/lib/wasm/wasm_exec_node.js"'`.
This avoids a batch intermediary that could orphan Node after a timeout. Test
success requires a real successful process exit, never just matching `PASS` text.

Independent review and detailed failure history:
[Astra framework verification](v6-astra-framework-verification.md).
