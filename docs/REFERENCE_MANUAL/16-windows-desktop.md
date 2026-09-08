# Windows desktop integration

Status: Windows-first, experimental contributor workflow. GWC's Wasm renderer
runs in Wails WebView2; native services run in a separate Go runtime. The browser
renderer is unchanged. Native Wails dependencies stay in the isolated host module.

## Start from the checked-in example

From the GWC repository root, initialize the exact pinned submodule, then inspect,
build and test the desktop target:

```powershell
git -c core.longpaths=true submodule update --init third_party/wails
go run ./tools/gwc desktop doctor -root ./examples/desktop/wails-counter -json
go run ./tools/gwc desktop build -root ./examples/desktop/wails-counter -json
go run ./tools/gwc test -lane desktop -root ./examples/desktop/wails-counter -json
.\examples\desktop\wails-counter\bin\wails-counter.exe --two-windows
```

Requirements: Windows, Go 1.26.3 (the verified toolchain), installed WebView2,
and pinned Wails beta.17 source. The CLI targets Windows amd64 explicitly. No
global Go environment changes or global Wails CLI installation are needed.
First-time Go dependency resolution may use the network; packaged UI assets
are local. ARM64 binaries are not part of the tested initial target.

## Create and develop a project

```powershell
go run ./tools/gwc desktop init -root ./bin/my-desktop-app -json
go run ./tools/gwc desktop dev -root ./bin/my-desktop-app
```

Init refuses an existing target directory and copies allowlisted source only,
not stale generated bindings or binaries. This first template is contributor-
linked: its module replacements point to this checkout and pinned Wails source.
It preserves `example.com/gwc-wails-counter` as the example module identity.
Independent published templates and arbitrary target layouts are not yet supported.

The optional `desktop` field in schema-v1 `gwc-start.json` describes native entry,
frontend entry, asset directory, output path and Wails pin. Absent desktop metadata
preserves web defaults. Invalid paths and unsupported layouts fail explicitly.

Dev owns its host process and rebuilds/restarts it on relevant source changes.
Generated asset and binary outputs do not trigger rebuild loops. Window-local
state resets on restart; saved state remains in the native store. This first
supervisor does not provide frontend-only hot reload or watch the entire linked
GWC checkout. Its per-project lock prevents simultaneous ownership; it never
takes over another development process.

## Native calls, state and security

### Build mode and capability gates

`gwc build/dev/test -target desktop` selects the native workflow; their default
target remains web. Desktop `-features all|none|<comma-separated names>` sets the
compiled host allowance. Runtime `--features` can only narrow that allowance.
For example, `-features file-dialogs,window-controls` does not grant clipboard,
persistence, native menus or report export. Unknown names are errors.

Inside the example, `go run ./tools/build -target web` builds standalone assets
under `assets/web`, without Wails imports. Desktop assets remain in `assets/dist`.
The optional task commands are `build:web`, `build:desktop`, and `test:desktop`.
`gwc dev -target desktop -dry-run` and `-json` return a read-only development
plan. To run the supervisor, omit those flags; the explicit `gwc desktop dev`
command also supports its existing JSON completion report. Canonical desktop
builds use the module's configured entry/output and reject web-only app/profile/
output overrides instead of silently ignoring them.

Use `//go:build js && wasm && gwc_desktop` for application files that must be
excluded from web output; provide a `!gwc_desktop` companion when shared callers
need the same signature. `desktop.IsDesktopBuild()` reports the compiled mode.
For runtime availability use `Client.Supports`/`Require`: build mode alone grants
no permission. The portable SDK itself does not require a desktop build tag.
See the [two-gate decision](../plans/v6-desktop-build-gates.md).

Use the optional [desktop package](../../desktop/README.md) from Go frontend
components, with `ui.UseTask` for asynchronous calls and effect-owned event
cancellation. Native service methods must validate input and cooperate with
context cancellation. The two runtimes share serialized DTOs, not memory.

The example includes local/native counters, a file picker, progress, native
errors, cancellable work and saved state shared between independent windows.
Storage uses an app-owned bounded atomic JSON file with version conflict checks
and a Windows handle lock. It is not the root SQLite temp-directory adapter.
The desktop-backed kvstate binding explicitly opts into native invalidation.

Treat all code loaded into the WebView as privileged. Serve trusted bundled
assets, keep external content out of that view, validate native inputs, and do
not enable production dev/agent endpoints. The JS method map is not authorization.
The tested CSP disallows the Function-constructor import fallback; browser code
must load modules through the external bootstrap. Full hostile-navigation and
accessibility/IME certification remains a release gate, not an inferred property.

## Test and package

The `desktop` lane is explicit and is not silently added to existing default or
`all` web lanes. It tests the nested host module, builds embedded assets, launches
actual native windows, exercises buffered Wasm fallback, and checks visible
failure behavior for missing bindings/Wasm. Unsupported native platforms fail
explicitly rather than counting as successful desktop tests.

```powershell
go run ./tools/gwc desktop package -root ./examples/desktop/wails-counter -json
```

Packaging produces an **unsigned** ZIP and an artifact manifest with hashes and
tool versions after native smoke validation. It is not a signed installer.
Signing credentials, an installer/update policy and manual Windows acceptance
are still needed before a production distribution claim. macOS/Linux are deferred
by the Windows-first scope decision.

See [plan](../plans/v6-wails-desktop.md), [backlog](../../todos.md),
[Windows compatibility and manual checks](../plans/v6-wails-windows-compatibility.md),
[D1 evidence](../plans/v6-wails-verification.md) and
[D2 evidence](../plans/v6-wails-adapter-verification.md) and
[final integration verification](../plans/v6-wails-integration-verification.md). A workflow definition is
not evidence of a completed hosted CI run.
