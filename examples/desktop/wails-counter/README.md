# Wails Counter

Experimental Windows desktop example in an isolated Go module. The existing
GoWebComponents renderer runs as js/wasm inside native Wails WebView2 windows.
Use Go 1.26.3 or a compatible toolchain and an installed WebView2 runtime. This
checkout uses Wails beta.17 from `../../../third_party/wails/v3` and GWC's current
`/v5` module from `../../..`. Initialize the pinned submodule from the repository
root first:

```powershell
git -c core.longpaths=true submodule update --init third_party/wails
```

From this directory, the tested Windows amd64 build is:

```powershell
# These settings affect this shell only; do not use go env -w.
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
$env:CGO_ENABLED = '0'
go run ./tools/build
.\bin\wails-counter.exe
# Optional: inspect two independent windows with the same saved counter.
.\bin\wails-counter.exe --two-windows
```

The helper explicitly builds native children for Windows and its own running
architecture, with CGO disabled, and frontend children for js/wasm. Selecting
the helper's target explicitly also handles machines with persisted cross-target
Go defaults. ARM64 and other OS targets have not been verified.

The build copies local HTML/CSS/bootstrap into assets/dist before the embedding
package can be inspected, compiles Wasm, runs the pinned Wails source CLI to
generate JavaScript bindings (`-b` uses the served built-in runtime), copies
lib/wasm/wasm_exec.js from the same Go toolchain, then embeds everything in
bin/wails-counter.exe. It works with no pre-existing dist directory and fails
on subprocess errors. Go dependency downloads may require network on the first
build; the frontend has no npm/Vite build dependency or remote asset dependency.
The CLI's source build may print v3.0.0-dev; the module and submodule revision
identify the actual beta.17 pin.

`task build` is equivalent if Task is installed. Generated dist and bin outputs
are ignored. The checked-in .gitkeep lets native tests compile the embed package
before building; launching the host still requires a complete build.

```powershell
go test ./...
go vet ./...
.\bin\wails-counter.exe --smoke-test
.\bin\wails-counter.exe --smoke-test --smoke-fault streaming
```

The smoke flag creates two hidden **actual native WebView2 windows** and exits with a JSON
report within 45 seconds. It clicks rendered controls to exercise Go local state,
typed native response decoding, native service errors and native progress reaching
Go-rendered DOM. Three counter/about route cycles exercise GWC routing and
subscription cleanup. It checks Wasm MIME and the CSP's Function-constructor
restriction. The streaming fault deliberately tests buffered instantiation.
It also verifies a durable native save reaches the second Go-rendered window
without sharing local UI counter state. Smoke storage is isolated in a temporary
directory; automated runs never touch your saved application data.

These negative checks must each exit 1 and report visible-boot-error and
native-controls-unavailable:

```powershell
.\bin\wails-counter.exe --smoke-test --smoke-fault missing-binding
.\bin\wails-counter.exe --smoke-test --smoke-fault missing-wasm
```

Normal launches do not register the SmokeService. The counter service includes
an explicit error demonstration. Native controls become available only after
generated exports and runtime readiness have passed. The startup status remains
outside the renderer mount so errors stay visible.
A native GetCapabilities handshake verifies protocol, platform, declared Wails
version and method map before Wasm starts. This is compatibility checking, not
authentication of hostile frontend code.

Service work uses ui.UseTask and the optional desktop adapter. Context cancellation
forwards to the generated Wails request's cancel method and releases the adapter
registry entry. The smoke checks the backend's active/cancelled work counters
after explicit cancellation and route unmount, independently of frontend state.
Cancellation is cooperative and cannot undo completed writes or force an OS picker
to close. Native event subscriptions belong to the counter route and coalesce
progress bursts; route unmount unsubscribes and pagehide closes the whole adapter.
Calls default to a 30-second deadline; use a shorter context deadline when needed.
Wide numeric identifiers must use strings (unsafe numeric magnitudes are rejected).

The remaining manual acceptance steps are to launch visibly, select a file,
cancel the picker and confirm "Dialog cancelled", use keyboard/focus controls,
resize, close with the title bar, and start with the network disconnected.
Automated smoke does not substitute for these OS interactions. WebView2 on the
tested machine logs an unregister-class warning during shutdown even when the
host exits successfully; see the evidence record for exact results and limits.

[D1 verification evidence](../../../docs/plans/v6-wails-verification.md) records the
tested environment, commands, remaining gaps and WAILS-001 through WAILS-007
acceptance. Native Wails dependencies remain isolated from the root module.
[D2 adapter verification](../../../docs/plans/v6-wails-adapter-verification.md)
records the cancellation, transport, wire and event tests added afterward.

## Windows-first integration

The saved counter uses an explicit desktop-backed kvstate binding. Normal runs
persist to `%AppData%/GWCWailsCounter/storage.json`; the host owns this bounded
atomic JSON store and a Windows handle lock. Values survive restart. Stale/equal
versions fail instead of silently overwriting a newer window's write. Closing
or crashing a process releases the kernel lock. Two windows in one host share
the store; two separate hosts cannot concurrently own it.

`ExternalInvalidation: true` keeps native bindings off BroadcastChannel. Each
window subscribes to native `storage.changed` commits and reloads mounted keys;
UI-only state remains local. See [desktop API](../../../desktop/README.md).

From the repository root:

```powershell
go run ./tools/gwc desktop doctor -root ./examples/desktop/wails-counter -json
go run ./tools/gwc desktop build -root ./examples/desktop/wails-counter -json
go run ./tools/gwc test -lane desktop -root ./examples/desktop/wails-counter -json
go run ./tools/gwc desktop dev -root ./examples/desktop/wails-counter
go run ./tools/gwc desktop package -root ./examples/desktop/wails-counter -json
```

`desktop init -root <new-directory>` creates a contributor-linked source template
with absolute local module replacements. It requires this checkout and its pinned
Wails source; it is not an independently published framework release. `dev` owns
and restarts its native process on source edits; it currently rebuilds both sides
and resets window-local state rather than preserving state through hot reload.
`package` emits an explicitly unsigned ZIP and hashes after native smoke passes.
Code signing and installer distribution remain separate release work.
# Windows API Lab extension

Normal launch now opens the comprehensive [Windows API tester](WINDOWS_API_TESTER.md).
The original counter remains available through its navigation link and `#/counter`.
Automated smoke continues to cover the counter, then verifies the tester route.
