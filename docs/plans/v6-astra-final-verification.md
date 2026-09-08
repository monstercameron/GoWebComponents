# Final independent desktop review — 2026-09-08

Scope: Windows-first v6 working-tree integration, with the public root module still
`/v5`. This review covers portable desktop contracts, the isolated `desktop/wails`
adapter, CLI build/dev/test/package boundaries, and the Windows API Lab. It does
not turn compilation, mocked native backends, or headless browser tests into
physical Windows UI acceptance. No commits, pushes, signing operations, root
module dependency edits, or upstream Wails source edits were made.

## Findings resolved during final review

- Disabled persistent storage previously opened/locked the durable store before
  policy-controlled registration. Construction/cleanup is now gated as well.
- Effective capabilities require the complete registered method set. Native menus
  use explicit non-RPC `Capabilities.Features` advertisement, not an invented
  callable menu-install method; JS capabilities preserve a bounded snapshot.
- Report export advertisement accounts for both export policy and the available
  file-dialog backend. Native and legacy entry points enforce their own policy,
  rather than trusting frontend controls.
- Native request errors and message-result kind mismatches are classified and
  rejected. Clipboard results are bounded on both sides. An independent regression
  reproduced invalid client inputs reaching transport; typed clients now reject
  invalid UTF-8, NUL text, unknown message kinds and invalid window sizes before
  JSON can silently repair input or native work begins.
- Native-menu F8 and window-control Escape now have independent feature gates.
  A window-controls-only host retains its fullscreen escape shortcut. Unit tests
  cover all, none, menus-only, controls-only, clipboard-only and nil service maps.
- The unused pre-adapter message-dialog helper was removed after final native lint
  identified it. The current path uses the typed host/adapter implementation.
- Desktop `dev -dry-run` and first-class `dev -json` now return a read-only plan;
  neither builds, locks nor starts a host. Unsupported app/server/output/profile
  flags are rejected rather than silently ignored. Web dev rejects desktop-only
  feature configuration.
- A desktop test target always resolves the desktop lane, including `all` and
  direct watch configuration. Feature ceilings propagate through the native lane.
- A claimed web build rejects a `gwc_desktop` profile tag and explicitly supplies
  even empty build tags. Actual regression builds first reproduced leaked desktop
  source selection from both process GOFLAGS and a temporary persisted GOENV file,
  then passed using the corrected generated command. No user Go environment file
  was changed.
- The direct build helper canonicalizes comma-separated feature names before
  linker flags, including valid whitespace-separated input. The dual-tag fixture
  now actually compiles each selected source set instead of merely listing files.
- First-class web build/dev recognizes desktop scaffold metadata, produces and
  serves the separate `assets/web` bundle, and preserves native `assets/dist`.
  Unsupported canonical-target overrides are rejected, dry-run validates metadata,
  and every existing web output destination is checked for symlinks before writes.
- Ordinary, TUI and agent web-dev launchers now share target-scoped child environment
  sanitation. A real agent-mode test then exposed an existing WebSocket idle panic:
  retrying a Gorilla read after its 500ms timeout poisoned the connection. The reader
  now stays idle without a deadline, closes through context cancellation, uses
  cancellable dialing and backs off real reconnect failures by 300ms. No panic
  recovery or increased idle timeout hides the defect.

## Independent automated evidence

Native commands used process-local `GOOS=windows`, `GOARCH=amd64`, `CGO_ENABLED=0`.
Wasm commands used process-local `GOOS=js`, `GOARCH=wasm`, `CGO_ENABLED=0`.

Root package checks:

```powershell
go test ./desktop ./tools/gwc -count=1
go vet ./desktop ./tools/gwc
node --test desktop/desktop.test.mjs
go test -exec C:/Users/mreca/Desktop/GoWebComponents/tools/go_js_wasm_exec.bat ./desktop -count=1
go test -tags gwc_desktop -exec C:/Users/mreca/Desktop/GoWebComponents/tools/go_js_wasm_exec.bat ./desktop -count=1
golangci-lint run --timeout 60s ./desktop
```

The earlier complete CLI package suite exited zero in 106.799 seconds. After all
final metadata-dispatch, argument/output guards, environment sanitation and idle
WebSocket fixes, the complete CLI package suite was repeated and passed in
100.528 seconds, followed by a clean vet and the real-process integration rerun
listed below. The earlier green command is not substituted for the final one.
Actual Wasm desktop suites passed both ordinary and `gwc_desktop` modes; JavaScript
transport tests passed all six cases. Tests exercise real promises, deadlines,
cancellation, closed transports, typed envelopes and non-RPC menu discovery.
The js/wasm dependency list for `./desktop` contains no Wails dependency.

Nested adapter module (`desktop/wails`):

```powershell
go test ./... -count=1
go vet ./...
golangci-lint run --timeout 60s ./...
# With the Wasm target: verifies unsupported-backend behavior, not Windows UI.
go test -exec C:/Users/mreca/Desktop/GoWebComponents/tools/go_js_wasm_exec.bat ./... -count=1
```

All exited zero. Native tests explicitly reject missing caller-window contexts.

Nested example module:

```powershell
go test ./internal/services ./cmd/desktop -count=1
go vet ./internal/services ./cmd/desktop
golangci-lint run --timeout 60s ./internal/services ./cmd/desktop
# With the Wasm target:
go test -tags gwc_desktop -exec C:/Users/mreca/Desktop/GoWebComponents/tools/go_js_wasm_exec.bat ./frontend -count=1
go vet ./frontend
```

All passed after the dead-helper cleanup and independent shortcut-policy fix.
The existing symlink export test remains explicitly skipped where Windows denies
symlink creation privilege; alias handling is not represented as a successful
real symlink test. API report no-overwrite, cancellation-before-create, Unicode
bounds, snapshot isolation and disabled direct-call tests remain covered.

Actual CLI checks:

```powershell
go run ./tools/gwc dev -target desktop -root examples/desktop/wails-counter -features 'screens, clipboard' -dry-run -json
go run ./tools/gwc build -target desktop -root examples/desktop/wails-counter -profile tinygo -json
```

The first exited zero with `dryRun:true` and canonical `clipboard,screens`, without
starting a host. The second returned the expected exit 1 and structured JSON error
for unsupported desktop profile flags; this negative test is not a failed build
attempt at a supported configuration.

The final independent real-browser command was:

```powershell
go test -tags playwrightgo ./test/playwrightgo -run '^TestDesktopLabWebE2E$' -count=1 -v
```

It exited zero (50.779 seconds total, test 49.68 seconds) using real Chromium.
It built the separate web output, navigated tester/counter routes, verified disabled
native controls, filled/read Unicode text, clicked the local counter, rejected
native-module requests and page errors, and confirmed no desktop transport was
installed. This proves browser behavior, not native Windows input.

A simultaneous final focused native desktop/CLI run printed both packages passing
but the Go command then exited 1 with `unlinkat ... desktop.test.exe: Access is
denied` while cleaning its temporary build directory. That attempt is an
environment cleanup failure, not a clean passing command; earlier successful
commands above remain distinct. A bounded isolated rerun is recorded below when
complete. Native smoke timer failures under concurrent load likewise require an
isolated rerun rather than a changed timeout or a fabricated success.

## Boundaries and final artifacts

Final native all/none/selected/fallback/fault/scaffold/package runs are recorded
in the parent integration evidence with their exact artifact identities. Do not
substitute older API Lab hashes or the earlier 17/20-check reports for that matrix.
The older integration report deliberately preserves failed attempts and corrections.

At the final coordination checkpoint, the root verifier reported successful
packaging and native runtime all/none/selected/file-dialogs-disabled runs, plus
compiled-none/runtime-all and compiled-selected/runtime-all runs proving that
runtime flags cannot broaden the compiled ceiling. Missing-binding and missing-Wasm
faults returned the expected exit 1. The newest streaming attempt timed out at its
unchanged 45-second bound while broad Go tests and other smoke processes competed
for resources; its isolated rerun is a separate required result, not an assumed
pass based on an older executable. The isolated rerun subsequently passed all
23 checks with exit 0 on the final artifact. Thus the latest full native matrix
passes; the earlier load-sensitive timeout remains recorded.

Final unsigned package identity, independently read from the package manifest:

- Directory: `bin/v6-release-20260908`.
- Executable SHA256: `2c7db74f8c686debd3c5f2afa83055904f62a64aedcd0ef4f368ab84b8ed5255`.
- ZIP SHA256: `f4a798f47cb45b72bfd23e7f1d3bfc8877b129db68d732c6c80a4a589c48cbad`.
- Manifest ceiling: `all`; `signed:false`; pinned Wails revision
  `5bce785eb1efbd121ef2e1cb2588eb50b9c59068`.

The root verifier also checked archive-entry identity. The compiler installation
reports windows/arm64, while explicit native build targeting is windows/amd64 with
CGO disabled; the manifest's `go version` alone is not the artifact architecture.

The root's uncached full repository test command also exited 1: the reported
failures were Windows temporary-executable cleanup Access denied in the CLI SSR
fixture test and the hookcheck end-to-end test after their underlying oracles ran.
Other packages reported passing. A full command with these cleanup failures is
not called green. The root's subsequent `go test -p 1 ./...` exited zero, including
the previously failing CLI (99.627 seconds) and hookcheck (2.437 seconds) packages.
This serialized pass predates the final first-class web metadata-dispatch correction
described below; that CLI correction requires its own focused and browser reruns.
The cleanup failures are retained discrepancies rather than suppressed diagnostics.

A last workflow review found that the original first-class `gwc build/dev -target
web` still consumed native paths from the desktop project's metadata, despite the
standalone web helper working. The root's fresh-scaffold browser test reproduced
that discrepancy (`dev could not detect app entrypoint`). The metadata-aware CLI
path was corrected. The root subsequently ran the fresh-scaffold first-class CLI
`TestDesktopLabWebE2E` in real Chromium successfully (15.193 seconds) and inspected
its screenshot; its final rerun after the latest CLI edits passed in 9.783 seconds.
This is separate evidence from the earlier helper-backed pass.

The independent tagged real-process web-dev test starts a fresh temporary scaffold,
GETs root/bootstrap/CSS/Go runtime/Wasm, rejects exposure of module files, and edits
only temporary frontend Go source outside the served directory. It deliberately
injects `GOFLAGS=-tags=gwc_desktop` and an undefined desktop-tagged symbol: neither
initial nor watcher builds may select that source. Ordinary and `-agent` modes
are both exercised; the final assertion waits for initial watcher completion,
then requires a changed served Wasm hash containing the unique edited UI string.
Native `assets/dist` must remain absent. This proves serving and rebuilding, not
browser hot-reload DOM preservation or native visual interaction.

The first agent-mode attempts failed with `panic: repeated read on failed websocket
connection` and left two child dev servers behind. Their exact command lines and
PIDs were verified before terminating only those test-owned trees. The corrected
idle regression holds one connection for 1.2 seconds (over twice the former
deadline), then asserts cancellation closes the socket and reaps its reader within
one second. Both live dev modes subsequently passed. Symlink-creation regression
tests remain explicitly skipped on this Windows account because it lacks the
required privilege; source guards are not mislabeled as executed symlink evidence.

Final independent command sequence, after all code changes, with process-local
Windows/amd64 and CGO disabled:

```powershell
go test ./tools/gwc -count=1
go vet ./tools/gwc
go test -tags desktopintegration ./tools/gwc -run '^Test(DesktopWebDevIntegration|AgentDevWebSocketRemainsIdleAndCancels)$' -count=1 -v
```

The entire sequential command exited 0. Full CLI tests passed in 100.528 seconds;
vet emitted no findings. Idle connection/cancellation passed in 1.22 seconds.
Actual ordinary and agent dev checks each passed in 6.90 seconds (13.80 seconds
combined; tagged package 15.299 seconds). The checks wait for initial compilation
to finish before editing and require the edited unique string in the changed Wasm,
so an initial helper-to-dev rebuild cannot masquerade as source-watcher evidence.
All final test-owned dev process trees were stopped and reaped.

Real Windows interaction on the latest adapter/feature-gated build is blocked:
the computer-use surface returned black screenshots and failed activation twice,
and further blind interaction was stopped. Prior manual observations are historical
evidence, not proof that the current executable has passed physical keyboard/IME,
picker, fullscreen, context-menu or accessibility checks. The pinned upstream
accelerator reset issue has an application-level edit-menu mitigation, with unit
evidence; a fresh physical Ctrl+V/Unicode test remains subject to that same external
interactive-desktop blocker.

Windows is the requested native platform. Other native platforms are unsupported,
not failed Windows acceptance. Hosted CI execution, signing/installers, updater
behavior and startup/memory/performance budgets are not inferred from local tests.
The exact-origin middleware and no-referrer runtime probe are bounded evidence,
not a claim that all external navigation and privileged WebView attack paths have
been penetration-tested. Local capability advertisements are not authentication.

## Final verdict

The reviewed Windows-first automated integration is passing: portable contracts,
real Wasm transport, native feature-ceiling/fault/package matrix, fresh-scaffold
Chromium web behavior, and first-class ordinary/agent web-dev serving and rebuilding.
No known release-blocking correctness regression found in this review remains
unfixed. This is not unrestricted release certification: current native physical
UI acceptance is externally blocked, Windows symlink tests need privilege, and
hosted CI, signing/installers and performance budgets remain unverified gates.
