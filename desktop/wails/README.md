# Wails adapter for GWC desktop workflows

This is a separate Go module, not a root-SDK dependency. It implements the
portable `desktop.FileDialogBackend` and `desktop.NativeBackend` contracts using Wails v3.0.0-beta.17.
Only the native application host imports this package. Wasm/shared application
code imports `github.com/monstercameron/GoWebComponents/v5/desktop` instead.

```go
import (
    "github.com/monstercameron/GoWebComponents/v5/desktop"
    wailsadapter "github.com/monstercameron/GoWebComponents/v5/desktop/wails"
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
advertise support. NativeBackend covers explicit clipboard read/write, caller-
owned `Ok`/`Yes`/`No` message dialogs, window info/control operations, and typed
screen enumeration. Windows calls require Wails' caller-window context. The adapter owns the pinned-version mapping
and its narrow private-cancellation-sentinel shim; neither leaks into SDK users.

The current module is **unreleased**. Checked-in local replace directives and the
example's v0.0.0 placeholder support this contributor workspace. A consuming main
module must explicitly select compatible SDK/adapter versions and any development
replacements: dependency modules' replace directives are not inherited. Fresh
`gwc desktop init` scaffolds receive the needed absolute local replacements.

For release, publish a separately tagged adapter version, depend on the tested
Wails module release normally, and keep the source submodule optional for upstream
inspection/patch development. Do not require downstream users to clone it.
Wails upgrades require adapter contract tests plus real native WebView checks.
No v4 adapter or alternate backend is promised before there is a real use case.

Tests: `go test ./...` and `go vet ./...` from this module with the explicit
Windows native target. Root `go test ./desktop` does not traverse this module.
