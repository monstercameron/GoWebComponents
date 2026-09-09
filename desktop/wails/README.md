# Wails adapter for GWC desktop workflows

This is a separate Go module, not a root-SDK dependency. It implements the
portable `desktop.FileDialogBackend`, `desktop.NativeBackend`, and optional
capability contracts using Wails v3.0.0-beta.17 with a pinned Windows menu patch
in the [GWC-maintained fork](https://github.com/monstercameron/wails).
Only the native application host imports this package. Wasm/shared application
code imports `github.com/monstercameron/GoWebComponents/v6/desktop` instead.

```go
import (
    "github.com/monstercameron/GoWebComponents/v6/desktop"
    wailsadapter "github.com/monstercameron/GoWebComponents/v6/desktop/wails"
    "github.com/wailsapp/wails/v3/pkg/application"
)

// In host startup; false or a zero-value host denies file-dialog operations.
parseFiles := desktop.NewFileDialogHost(wailsadapter.NewFileDialogs(), true)
parseService := application.NewService(parseFiles)
```

Register the service with the app. Advertise `parseFiles.GetMethods()` in the
existing capability handshake, then install its generated SelectPaths binding
under `desktop.FileDialogMethod`. The example bootstrap demonstrates this mapping;
application UI calls OpenFile/OpenFiles/OpenDirectory/SaveFile without binding names.

Windows is the only implemented native platform. `NewFileDialogs` and
`NewNativeBackend` return nil on other platforms, so the host does not falsely
advertise support. The Windows backend covers clipboard, message and file dialogs,
caller-owned window controls, printing, screens and coordinate conversion,
runtime menus, tray, global shortcuts, environment, external URLs, file-manager
reveal, and autostart. Child windows require host-registered templates; window
events require host lifecycle hooks, and file drops require explicit opt-in.
Each family has a separate capability gate. See the
[Windows coverage matrix](../../docs/windows-api-coverage.md) for restrictions
and verification status. Windows calls require Wails' caller-window context. The adapter owns the pinned-version mapping
and its narrow private-cancellation-sentinel shim; neither leaks into SDK users.

The current module is **unreleased**. Checked-in local replace directives and the
example's v0.0.0 placeholder support this contributor workspace. A consuming main
module must explicitly select compatible SDK/adapter versions and any development
replacements: dependency modules' replace directives are not inherited. Fresh
`gwc desktop init` scaffolds receive the needed absolute local replacements.

The contributor build currently requires the exact clean fork revision pinned
by the root submodule and build tool. It is not an unmodified upstream release.
A standalone adapter publication must provide a tested dependency resolution
for that patch: Go does not inherit dependency modules' replace directives.
Wails upgrades require adapter contract tests plus real native WebView checks.
No v4 adapter or alternate backend is promised before there is a real use case.

Tests: `go test ./...` and `go vet ./...` from this module with the explicit
Windows native target. Root `go test ./desktop` does not traverse this module.
