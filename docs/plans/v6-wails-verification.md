# V6 Wails D0/D1 verification

Recorded 2026-09-08 by the final Astra verification pass, after the bounded
Luna implementation work. Scope is the isolated Windows counter and its
source pin; this is not acceptance of the later desktop framework backlog.

## Verdict

| Item | Evidence / remaining acceptance |
| --- | --- |
| WAILS-001 | Met: beta.17 source/API record, exact submodule pin, direct generated ES modules and clean asset build verified. The source CLI prints `v3.0.0-dev`; the git revision/module pin, not that display string, identifies the version. |
| WAILS-002 | Met for this Windows environment: isolated module resolution and native build/tests pass; root Go module and persisted Go settings unchanged. |
| WAILS-003 | Met: browser artifact baseline unchanged; fresh counter interactions pass in Chromium, Firefox and WebKit. |
| WAILS-004 | Met: a build with no dist directory produces one native WebView host; real GWC counter, typed CSS and GWC hash router execute there. |
| WAILS-005 | Partial: typed native response, native error, native progress reaching the Go-rendered UI, concurrent local input and route subscription cleanup verified. Real OS file selection/cancellation remains manual. |
| WAILS-006 | Technical acceptance met: generated bindings and runtime validated before Wasm/control readiness; external ImportModule helper works under the selected CSP; missing bindings produce persistent visible error DOM. Overall dependency chain still has the WAILS-005 manual gap. |
| WAILS-007 | Partial: embedded startup, matching runtime asset, Wasm MIME, buffered fallback, routing and host process termination verified. Physical keyboard/focus, interactive resizing, title-bar close and disconnected-network launch remain manual. |

## Defects found and fixed

The initial harness called generated JavaScript directly and accepted a hash
string as routing proof. It could pass while the Go UI's native event path was
broken. The initial frontend listened for DOM CustomEvents, but no bridge
forwarded Wails events to that DOM target. Readiness imported a wrapper module
without proving that its generated service bindings existed, and the renderer
could remove the startup status node because it was inside the mount.

The example now uses `router.NewHashRouter`, distinct counter/about route
components, `ui.UseTask` for import/service work, and typed `CounterState`
decoding. Native progress is explicitly forwarded to one page-owned DOM event
path; the native listener unsubscribes on pagehide and the Go route cancels its
DOM subscription on unmount. Three route unmount/remount cycles exercise fresh
local state, bridge readiness and progress delivery. Task cancellation suppresses
stale UI completion; forwarding cancellation to backend requests is **not**
implemented or claimed by this D1 example.

The stronger test exposed another real integration issue: generic interop
error summarization omitted non-enumerable JavaScript Error.message, displaying
only `{"name":"RuntimeError","cause":{}}`. The example's promise wrapper
preserves name/message in the rejection payload; the native failure text now
reaches the Go task and rendered status. Root interop was not changed.

Bootstrap now checks required generated exports and Wails Events.On before
starting Wasm. The startup status is outside #app. A local CSP permits same-origin
scripts and Wasm compilation (`wasm-unsafe-eval`), but blocks the Function
constructor. Inline styles remain permitted for generated typed CSS/style props.
This is a tested Windows policy, not a cross-platform security certification.

The build helper explicitly sets native GOOS=windows, GOARCH to its own running
architecture and CGO_ENABLED=0; the Wasm child explicitly gets js/wasm. Merely
removing process variables would have allowed persisted Go target defaults to
leak back in. Environment name filtering is case-insensitive for Windows.
`all:dist` includes the checked-in .gitkeep so native tests can compile the embed
package before frontend assets are built; it is not a substitute for building.

## Reproduction and results

Unless marked repository-root, commands run in `examples/desktop/wails-counter`.
Observed host: Go 1.26.3 windows/arm64 installation; effective helper/host target
windows/amd64, CGO_ENABLED=0; WebView2 152.0.4191.66 on Windows 11 ARM hardware.
No global Go environment changes, npm install, Vite, C compiler installation,
root Wails dependency, upstream submodule edits, commits or pushes were made.

| Command / check | Result |
| --- | --- |
| Move assets/dist to validated ignored bin/verify-preserved-dist, then `go run ./tools/build` | Exit 0 with dist confirmed absent before invocation. Copies static assets, builds Wasm, generates bindings via pinned source CLI with `-b`, copies matching wasm_exec.js, then builds host. Restored .gitkeep; previous outputs retained for recovery. |
| `go test ./...` | Exit 0, services pass; includes concurrent increments, canceled caller context, and smoke report completeness/delivery tests. |
| `go vet ./...` | Exit 0. |
| `GOOS=js GOARCH=wasm CGO_ENABLED=0 go vet ./frontend` (process-only PowerShell env) | Exit 0. |
| `GOOS=js GOARCH=wasm CGO_ENABLED=0 go test ./contracts` | Exit 0, no contract test files; compilation only, not Wasm test execution. |
| Repository-root `go run ./tools/gwc verify -app ./examples/desktop/wails-counter/frontend/main.go -root ./examples/desktop/wails-counter -json` | Exit 0: isolated native tests plus CI-profile frontend Wasm build, 7,167,689 bytes. |
| Repository-root `go run ./tools/gwc lint -root ./examples/desktop/wails-counter -path ./internal/services -path ./tools/build -path ./cmd/desktop -timeout 60s -json` | Equivalent prebuilt gwc executable invocation passed, zero issues; golangci-lint 1.64.8 and hook rules enabled. |
| `GOOS=js GOARCH=wasm CGO_ENABLED=0 golangci-lint run --timeout 60s ./frontend` | Exit 0, direct installed linter used because gwc lint forces native env. |
| `bin/wails-counter.exe --smoke-test` | Repeated actual hidden WebView2 launches exit 0. Button-driven Go local increment, typed native increment, native rejection text, native event/render, three route unmount/remount cycles and CSP checked. |
| `bin/wails-counter.exe --smoke-test --smoke-fault streaming` | Exit 0, same checks plus deliberately forced buffered Wasm instantiation. |
| `bin/wails-counter.exe --smoke-test --smoke-fault missing-binding` | Expected exit 1: missing dynamic module; report includes visible-boot-error and native-controls-unavailable. |
| `bin/wails-counter.exe --smoke-test --smoke-fault missing-wasm` | Expected exit 1: Wasm fetch failed (404); same visible error/readiness evidence. |
| wasm_exec.js SHA-256 comparison | Packaged and active toolchain files both `0c949f4996f9a89698e4b5c586de32249c3b69b7baadb64d220073cc04acba14`. |
| Repository-root browser counter build with normal gwc build flags | Exit 0; 6,647,102 bytes; SHA-256 unchanged at `7b312e1724b217a72d75617f844567a26bb086333f3288083330d9b4d4acb627`. |
| Repository-root `go run ./tools/gwc test -lane browser -json` | Exit 0; cached TestMainSuite result retained as cached evidence. |
| Repository-root `go test -tags playwrightgo ./test/playwrightgo/examples -run '^TestCrossBrowserConformance$/(chromium\|firefox\|webkit)/rendering_state/counter$' -count=1 -v` | Fresh exit 0, counter cases Chromium 0.79s, Firefox 1.74s, WebKit 0.74s. Full unrelated matrix was not rerun. |
| Root module / upstream checks | `git diff -- go.mod go.sum` empty; Wails submodule working tree clean. |

The normal packaged Wasm is 7,402,202 bytes; native executable approximately
23.97 MB (unstripped development build). Two full hidden-window smoke scenarios
took 13.10s and 11.96s including multiple route cycles and polling; these are
**not cold-start latency measurements**. The final smoke also explicitly checks
that the served Wasm Content-Type starts with application/wasm.

Final rebuilt normal and forced-fallback runs both exited 0 with explicit
`wasm-mime` evidence. Their success reports contain:

```json
{"ok":true,"checks":["dom","local-counter","native-call","native-error","native-event","routing","route-remount","wasm-mime","csp"],"error":""}
```

The forced-fallback report adds `wasm-fallback`. Final process inspection found
no remaining wails-counter.exe instances. Previous generated assets remain in
the ignored `bin/verify-preserved-dist` directory; they were moved, not deleted.
Final executable SHA-256:
`c9c37287c2836d4a958b0a558429b93a91360b110325ea0742f941cba8f19d0f`.

## Retained failures and limits

- The first strengthened test failed on missing native error text, exposing the
  Error.message mapping defect described above. It passed after the example fix.
- `gwc lint -path ./frontend` exits 1 because its implementation calls the
  linter with buildNativeGoEnv, even if its parent has js/wasm overrides.
  Direct js/wasm lint and Go vet pass. This existing CLI limitation was not
  changed as part of the desktop example.
- WebView2 repeatedly logs `Failed to unregister class Chrome_WidgetWin_0.
  Error = 1412` during shutdown. Passing scenarios still exit 0, failed scenarios
  exit 1, and no wails-counter.exe process remains after the runs. The warning
  is recorded rather than silently interpreted as either a crash or a clean log.
- Earlier work recorded intermittent Windows test-executable unlink access
  failures and an unrelated full-matrix WebKit storage temporary-server failure.
  This verifier's focused native tests and fresh three-engine counter run exited 0.
- Automated clicks and rendered DOM assertions in a hidden native WebView do
  not prove visible layout, physical keyboard/IME, OS dialog selection/cancel,
  user-driven resize, DPI/accessibility, or a title-bar close flow. Those remain
  explicit manual checks. Browser passes are not substituted for native evidence.
- Assets are local and CSP connects only to self; no dev server is used by the
  host. The machine's network was not disabled, so a disconnected launch is
  still a separate manual gate. No macOS/Linux support claim is made.
- General backend cancellation, never-settling callback bounds, persistence,
  multiwindow synchronization, packaging/signing and desktop CLI commands remain
  D2–D5 work. An empty native dialog result is handled distinctly in code, but
  its OS interaction has not been certified by this smoke.

Root-owned Windows checkout workflow changes were reviewed: only step-scoped
GIT_CONFIG_COUNT/KEY_0/VALUE_0 for core.longpaths were added before existing
recursive submodule checkout. No hosted CI run was performed.
