# Windows integration verification — 2026-09-08

Final Astra review of the continued D2–D4 implementation and the implemented D5
tooling. This supplements the earlier D1 and adapter verification reports; their
earlier capability-handshake/package limitations are historical. This is an
experimental Windows-first integration, not completion of every release gate.

## Scope and environment

- Windows 11 Home build 28000, WebView2 152.0.4191.66, Go 1.26.3, Node 26.2.0.
- Go's installed tool reports windows/arm64; the tested helper and packaged host
  explicitly build windows/amd64, CGO_ENABLED=0. Wasm is js/wasm. No persisted Go
  environment changes were made. Other native platforms are unsupported/deferred.
- Wails beta.17 source revision `5bce785eb1efbd121ef2e1cb2588eb50b9c59068` is clean.
  Root go.mod/go.sum remain unchanged. Native and Wasm desktop dependency lists
  contain no Wails or WebView2 package; only the nested host imports Wails.
- No commits, pushes, root module migration, or changes to unrelated uicodegen.

## Review findings fixed

- CLI prerequisite checking now compares the actual Wails revision, clean source
  checkout, module version, and local replacement directory. Fresh doctor initially
  failed on unrelated lazy module graph entries (`agenthub`/GoGRPCBridge) from
  `go list -m all`; it now resolves the actual service dependency closure before
  generated assets exist and retains the underlying command diagnostics.
- Build/package paths reject traversal, Windows drive/stream aliases, overlapping
  inputs, and linked roots/output trees. Init reserves a new root exclusively;
  packaging preflights and exclusively creates both archive and manifest. Partial
  owned archive output is cleaned on failure; existing output is never truncated.
- Child commands have caller cancellation, per-command timeouts, process-tree
  termination and bounded trailing diagnostics. Smoke acceptance requires the
  correct exit status **and** a complete structured report, not incidental log text.
- Development now observes host exit, calls Wait exactly once per child, stops the
  current restarted host (not a captured initial pointer), cancels builds, preserves
  read errors and releases only its own lock. All source classes use the same full
  Wasm/binding/native rebuild and restart; there is no hot-state preservation claim.
- Native storage previously accepted a decoded 13 MiB store whose JSON/base64 file
  exceeded its own 16 MiB reopen limit. A regression test reproduced the failure.
  Encoded size is now checked before replacement, reads are bounded, and base64
  payload size is checked before allocation. The commit-side context gate prevents
  a queued, cancelled write from committing after the ownership mutex is released.
- Malformed storage records now return structured decode errors (missing record,
  wrong key, invalid version/time/base64); tombstones retain their version.
- kvstate previously admitted stale records when the resolver returned the local
  version, and a tombstone bypassed reconciliation. Shared monotonic acceptance now
  protects both hook and atom invalidation. BindAtom hydrates tombstone versions,
  applies newer deletion state, and ignores writes after cancellation. Late watcher
  installation is released and empty external notification hubs are removed.
- The native capability handshake validates the declared protocol/platform/version
  and installed method map before Wasm. Exact local Host/Origin/Referer middleware
  precedes Wails runtime handlers. CSP, nosniff and same-origin referrer headers are
  applied; a real no-referrer runtime fetch must return 403. These are scoped
  defenses, not authentication against hostile code in the trusted local document.
- The older standalone Node tests used non-namespaced topics and failed the new
  contract. Their fixtures now use `test.tick`; the lifecycle assertions still run.

## Commands and observed results

Commands below ran from the repository root unless another working directory is
stated. Every passing entry means the process exited zero, not merely a package
printing PASS. No root-wide test matrix was substituted for nested-module checks.

| Command | Result |
| --- | --- |
| `go test ./desktop ./kvstate -count=1` | Pass. Includes payload validation, native unavailability, lifecycle, external invalidation and tombstone/version regression tests. |
| `go test ./tools/gwc -run 'TestDesktop\|TestValidateScaffoldDesktop\|TestNormalizeScaffoldMetadataPreservesWeb\|TestNormalizeTestLanesDesktop\|TestAcquireDesktopDev' -count=1` | Pass; symlink-creation test explicitly skipped because Windows lacks the required privilege. |
| `go test ./tools/gwc -run TestDesktop -count=1 -v` | Pass. Actual controlled child processes prove restart/cancel/rebuild-failure reaping, normal host exit observation, and bounded process deadline. |
| `go vet ./desktop ./kvstate ./tools/gwc` | Pass. |
| `go test ./tools/gwc -run 'TestNormalizeScaffoldMetadata\|TestLoadScaffoldMetadata\|TestScaffoldMetadata\|TestResolveBuildConfig\|TestResolveDevConfig\|TestResolveReleaseConfig\|TestRunBuild\|TestRunDevAgentDryRun' -count=1` | Pass; targeted existing web metadata/build/dev/release regression tests. |
| `node --test desktop/desktop.test.mjs` | All 5 tests pass after fixture correction. |
| `GOOS=js GOARCH=wasm CGO_ENABLED=0 go test -exec C:/Users/mreca/Desktop/GoWebComponents/tools/go_js_wasm_exec.bat ./desktop ./kvstate -count=1` | Pass in the real Wasm runtime. Executor must be absolute. Env notation here denotes process-scoped PowerShell variables. |
| Same Wasm environment: `go vet ./desktop ./kvstate` | Pass. |
| Same Wasm environment: `golangci-lint run --timeout 60s ./desktop` | Pass. |
| Same Wasm environment: `golangci-lint run --timeout 60s ./desktop ./kvstate` | Fails on 7 existing kvstate findings: unchecked Save in old tests (3), unload Listen (1), browser Subscribe/Publish (2), unused resetEnginesForTest (1). Not claimed clean or silently fixed. |
| Nested example: `go test ./... -count=1`, `go vet ./...`, `golangci-lint run --timeout 60s ./...` | All pass after fixing new shutdown/test cleanup diagnostics. Storage tests include crash-process lock release, restart, stale/concurrent versions, corruption, encoded quota and cancelled commit. |
| Nested example: `go run ./tools/build` then `./bin/wails-counter.exe --smoke-test` | Final build and actual two-window smoke pass all 17 checks. |
| `./bin/gwc-desktop-verify.exe test -lane desktop -root C:/Users/mreca/Desktop/GoWebComponents/examples/desktop/wails-counter -json` | Pass: build, nested tests, normal 17-check native smoke, missing-binding and missing-wasm exit 1 with required error checks, and streaming fallback with the additional fallback check. Desktop remains opt-in, absent from `all`. |
| `go test -tags playwrightgo ./test/playwrightgo/examples -run '^TestCrossBrowserConformance$/(chromium\|firefox\|webkit)/rendering_state/counter$' -count=1 -v` | All three pass; total test 17.72 s. This protects ordinary browser counter behavior, not the default kvstate BroadcastChannel persistence path. |
| `git diff --check` | Pass (Windows line-ending warnings only). |

The 17 native checks are dom, native-handshake, origin-guard, local-counter,
native-call, native-error, native-event, durable-native-state,
two-window-invalidation, window-local-state, backend-cancellation,
unmount-cancellation, adapter-cleanup, routing, route-remount, wasm-mime and csp.
Generated rejection and context-cancel log messages are expected assertions.
WebView2 still emits unregister-class warning 1412 at shutdown despite the correct
exit code; automated process inspection found no remaining owned host/supervisor.

## Fresh scaffold and distribution evidence

Built the verifier with `go build -o ./bin/gwc-desktop-verify.exe ./tools/gwc`.
`desktop init`, `desktop doctor`, `desktop build` and `desktop package` succeeded
against a new `bin/desktop-integration-verify-20260908` directory with no prior
dist. A second fresh `bin/desktop-final-verify-20260908` scaffold includes the
security middleware and passed doctor plus package's full 17-check native smoke.
Re-running init/package against that existing directory returned exit 1 and
truthful JSON errors; the original artifacts and manifest hashes were preserved.

For the second scaffold the independently recomputed hashes matched its manifest:

- Executable SHA256: `ae27ee6b447f3f8219810b53996a18dafbf61f4172e945cb3906126bc8d63fb3`.
- ZIP SHA256: `f24ce1a62c94874a7a72ab23446fc419298679868b7fa8d950df816545cfc6bc`.

The third fresh `bin/desktop-release-verify-20260908` scaffold exposed a polling
boundary failure: `smoke wait timed out: remount native event` while the same
report already contained `Native progress: 100%`. The old helper checked expiry
before sampling the state after a hidden-WebView throttled timer wake. It now
samples first and still rejects absent results after expiry. Both cases pass
`node --test assets/smoke_wait_test.mjs` in the nested example. The failed package
properly returned exit 1 with full diagnostics and created no archive/manifest.
A new `bin/desktop-smoke-boundary-verify-20260908` scaffold tests the corrected
helper: init and package exited zero with all 17 native checks. Independently
recomputed SHA256 values matched its successful unsigned manifest:

- Executable: `0444ec3ccbf27dc5c176467d96a9f22e5a79108528d90619de4047f0d8e65463`.
- ZIP: `cbc568c674942f63352b0ad986ffffbd32e96b1e41e8e42bac16b92d99a8139e`.

Final fault reruns invoked that exact executable with `--smoke-test --smoke-fault`
and each of `missing-binding`, `missing-wasm`, and `streaming`: the first two
exited 1 with visible-boot-error/native-controls-unavailable; streaming exited 0
with all 17 required checks plus wasm-fallback. Thus the corrected bootstrap was
verified in both normal package smoke and buffered-instantiation native smoke.

These ignored directories and earlier preserved dist output remain as evidence,
not source artifacts. Scaffolds are explicitly contributor-linked to this checkout,
not independent templates with portable published dependency replacement paths.

## Remaining gates and verdict

- WAILS-012/013/014: technical acceptance passes, including fresh source scaffold,
  metadata safety, pin checking and honest JSON failure propagation.
- WAILS-015: supervisor lifecycle/rebuild tests pass with actual controlled child
  processes. This does not certify physical window close or a manual editor-driven
  hot-reload experience; implementation deliberately uses full restart for edits.
- WAILS-016/017: native persistence and typed backend technical tests pass.
- WAILS-018: native two-window invalidation and cleanup pass. Real browser default
  BroadcastChannel persistence regression remains a separate outstanding check.
- WAILS-019: actual windows prove shared durable state and independent local state;
  competing edits and reopen/resynchronization are unit/service-covered where noted,
  not a complete native-window scenario suite. Keep full acceptance open.
- WAILS-020: local desktop lane passes; workflow source/long-path checkout changes
  were reviewed, but no hosted Windows WebView or Linux portable-race CI run is
  claimed. Linux portable tests do not mean Linux native support.
- WAILS-005/007 and 021–024 remain partial/open for the documented native manual,
  security, CI, distribution and performance gates. WAILS-001–004/006 and D2
  technical evidence remains as recorded in the earlier reports.
- Picker selection/cancel, physical keyboard/IME, resizing/DPI/accessibility,
  clipboard/external navigation, disconnected launch and title-bar close still
  require explicit manual/native testing. A no-referrer fetch 403 is not proof of
  every remote navigation/webview message boundary.
- Packaging is an unsigned Windows ZIP, not a signed installer, updater or
  production release. No signing identity or certificate was requested/used.
- Final fresh scaffold executable is 36,476,416 bytes, ZIP 16,004,239 bytes and
  Wasm 13,676,041 bytes.
  Importing kvstate retains SQLite defaults even when the selected backend is the
  native JSON service. This overhead is recorded, not optimized away. No cold-start,
  per-window memory, bridge-latency or responsiveness budget is established.
- Owned adapter registries/queues and Go callbacks are bounded, not total heap for
  arbitrarily externally retained never-settling JS promises. Cancellation cannot
  undo completed durable writes or force a native OS picker closed.

## API tester automated addendum — 2026-09-08

This independent pass covers the reviewed API tester source and the frozen native
build after picker-cancellation and interactive-timeout corrections. Manual API
observations are recorded separately; automated tests did not open visible dialogs,
read/write the clipboard, or manipulate the user's existing tester window.

The example executable tested here has SHA256
`3297271cd7c26f1636808d43dffc89e020dfa276f0689187036c537f8d4b3932`.
From `examples/desktop/wails-counter`, both commands exited zero:

```powershell
.\bin\wails-counter.exe --smoke-test
.\bin\wails-counter.exe --smoke-test --smoke-fault streaming
```

Normal smoke passed all 20 checks, including `api-tester-ui`, `api-window-info`,
and `api-session-report`; streaming passed those 20 plus `wasm-fallback`. These
use hidden actual WebView2 windows and isolated temporary storage. Intentional
native rejection/cancellation diagnostics appeared as expected. The previously
recorded `Chrome_WidgetWin_0` shutdown warning (1412) also appeared; process exit
was zero. The visible tester remained running, and neither smoke process remained.

With process-local `GOOS=windows`, `GOARCH=amd64`, and `CGO_ENABLED=0`, these
commands exited zero:

```powershell
# Nested example module
go test ./internal/services ./cmd/desktop -count=1
go vet ./internal/services ./cmd/desktop
go test ./internal/services -run 'TestAPI' -count=1 -v
# Repository root
go test ./desktop ./tools/gwc -count=1
go vet ./desktop ./tools/gwc
```

The full tools/gwc package test took 113.074 seconds. API service tests cover
UTF-8-safe bounded reports, returned-result/snapshot isolation, exact pinned
Windows cancellation compatibility, context cancellation before export creation,
exclusive no-overwrite output, fixture/case aliases, and Windows device names.
`TestAPIReportExportRejectsFixtureLink` explicitly **skipped** because this host
cannot create symlinks without additional privilege; no symlink execution proof
is claimed. The cancellation adapter recognizes the pinned CFD private error's
exact leaf message rather than a publicly exported sentinel; genuine and joined
errors remain errors. Native manual Cancel must still verify that compatibility.

With process-local `GOOS=js`, `GOARCH=wasm`, and `CGO_ENABLED=0`, these commands
also exited zero (root or nested module as indicated):

```powershell
# Repository root
go test -exec C:/Users/mreca/Desktop/GoWebComponents/tools/go_js_wasm_exec.bat ./desktop -count=1
go vet ./desktop
golangci-lint run --timeout 60s ./desktop
# Nested example module
go test -exec C:/Users/mreca/Desktop/GoWebComponents/tools/go_js_wasm_exec.bat ./frontend -count=1
go build -o bin/api-tester-verify.wasm ./frontend
go vet ./frontend
golangci-lint run --timeout 60s ./frontend
```

The bridge retains `Call`'s 30-second default. `CallWithTimeout` rejects nonpositive
or greater-than-ten-minute overrides before native work, preserves earlier caller
deadlines, and forwards cancellation/forgets real JS requests on timeout. Common
native/Wasm tests inspect default/extended deadline policy without a 30-second
sleep; real JS promise tests cover earlier-deadline cleanup. Only tester pickers,
message dialogs, and export opt into five minutes. UI tests cover the complete
manual-case list, selected timeout policy, actual outcome rendering, and suppressed
late poll updates after unmount cancellation. This does not certify physical focus,
IME, native menu behavior, or accessibility.

A fresh contributor-linked scaffold was created and built without changing the
frozen example executable. Each root command exited zero with successful JSON:

```powershell
go run ./tools/gwc desktop init -root bin/desktop-api-final-verify-20260908 -json
go run ./tools/gwc desktop doctor -root bin/desktop-api-final-verify-20260908 -json
go run ./tools/gwc desktop build -root bin/desktop-api-final-verify-20260908 -json
```

Running that scaffold's `bin/wails-counter.exe --smoke-test` also exited zero with
all 20 checks. Its executable SHA256 is
`679c8ef53b02774a535d833fafc9a5f2d754726e75b5acf96706e7d42ee572ca`;
executable size is 36,964,864 bytes and Wasm is 14,064,380 bytes. The source
allowlist includes the tester Go/CSS, API/menu service, contracts and tester guide.
This addendum validates fresh build/boot, not a new signed release or installer;
prior ZIP/hash evidence above remains a distinct earlier artifact.
