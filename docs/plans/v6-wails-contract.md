# Wails v3 beta.17 contract for a GWC Wasm desktop host

Status: source inspection only (WAILS-001), 2026-09-08. The vendored Wails
checkout is `third_party/wails` at tag `v3.0.0-beta.17`, commit
`5bce785eb1efbd121ef2e1cb2588eb50b9c59068`. This document records observed
APIs; it is not a promise that the beta is a supported production dependency.

## What the pinned source actually provides

The native module is `github.com/wailsapp/wails/v3` (`third_party/wails/v3/go.mod`,
`go 1.25.0`). A host imports `github.com/wailsapp/wails/v3/pkg/application`.
Keep the embed declaration in an assets package (a `cmd/desktop` package cannot
embed a sibling or parent tree):

```go
// assets/assets.go
package assets

import "embed"

//go:embed dist
var FS embed.FS
```

The native entrypoint then uses that package:

```go
// cmd/desktop/main.go
import "example.com/wails-counter/assets"

app := application.New(application.Options{
    Services: []application.Service{application.NewService(&CounterService{})},
    Assets: application.AssetOptions{
        Handler: application.BundledAssetFileServer(assets.FS),
    },
})
app.Window.NewWithOptions(application.WebviewWindowOptions{URL: "/"})
if err := app.Run(); err != nil { log.Fatal(err) }
```

`AssetOptions.Handler` and `BundledAssetFileServer` are in
`v3/pkg/application/application_options.go`. `BundledAssetFileServer` wraps
the static FS handler and has a `/wails/runtime.js` fallback
(`v3/internal/assetserver/bundled_assetserver.go`). In a normal `application.New`
host, the outer AssetServer middleware in `v3/pkg/application/application.go`
intercepts `/wails/runtime.js` first and serves the runtime (with its transport
prelude), then delegates ordinary paths to the configured handler. Thus the
embedded handler owns app assets; the outer server owns the runtime and Wails
transport routes. `WebviewWindowOptions.URL` is the initial asset URL.
`application.New` and `App.Run` are in `v3/pkg/application/application.go`. The normal examples
use `//go:embed assets/*` (`v3/examples/binding/main.go` and
`v3/examples/cancel-async/main.go`), but a narrower `assets/dist` subtree is
preferable when the package owns only assembled frontend output.

The default Wails templates are not a no-npm frontend: the base template's
`v3/internal/templates/base/frontend/package.json` uses Vite and
`@wailsio/runtime`. That tooling is optional for GWC. `generate bindings` has
the explicit `-b/--bundled-runtime` flag (`v3/internal/flags/bindings.go`),
which makes generated modules import `/wails/runtime.js` instead of the npm
package. The checked-in generated example confirms this exact import:
`v3/examples/binding/assets/bindings/.../greetservice.js`.

## Binding generation and generated ES modules

The command is registered as `wails3 generate bindings [flags] [patterns...]`
(`v3/cmd/wails3/main.go`, `v3/cmd/wails3/README.md`). Important flags are:

```text
-b                 import /wails/runtime.js (no npm runtime package)
-d <dir>           output directory (default frontend/bindings)
-ts                TypeScript instead of JavaScript
-i                 TypeScript interfaces
-names             call methods by name rather than numeric IDs
-noindex           omit generated package index files
-f <flags>         extra Go build flags used while scanning
```

The generated service shape is a stable module-level function that calls
`$Call.ByID(<numeric-id>, args...)` and returns `$CancellablePromise<T>`; see
`v3/examples/cancel-async/assets/bindings/.../service.js` and
`v3/examples/binding/assets/bindings/.../greetservice.js`. Index/model files are
generated beside the service module. Do not hand-maintain numeric IDs: rerun
the generator when the Go service changes.

Generation scans Go packages and writes output (`v3/internal/commands/bindings.go`).
It defaults to cleaning the output directory, using a temporary sibling and
syncing files into place to avoid watcher rename loops. This creates an embed
ordering constraint: generated files cannot be emitted into a directory that
must already exist for a `go:embed` pattern while the generator first loads the
native package. A clean checkout must therefore use this order:

1. Create the isolated module and native service. The build helper must create
   `assets/dist` and copy real source assets there before loading the embedding
   package. A dotfile placeholder alone is not sufficient for `//go:embed dist`.
2. Build GWC Wasm and assemble local HTML/CSS/bootstrap assets in `assets/dist`.
3. Run `go run github.com/wailsapp/wails/v3/cmd/wails3 generate bindings -b -d assets/dist/bindings ./cmd/desktop`
   (or a narrower native package pattern) from the example module.
4. Copy the matching `wasm_exec.js` and compile/run the native package, whose
   embed now includes the complete assembled tree. The example's Go build
   helper implements these steps; run it before tests on a fresh checkout.

Do not put a Wails import in the GWC frontend package. Native `cmd/desktop`
and frontend `frontend` must be separate packages in a nested module, so the
root `go.mod` remains Wails-free. A clean-build test must remove `assets/dist`
and repeat the stages; a pre-populated developer directory is not evidence.

## Runtime calls and cancellation

The bundled runtime source is `v3/internal/runtime/desktop/@wailsio/runtime/src`.
`calls.ts` exports `Call`, `ByID`, and `ByName`; generated bindings use `ByID`.
The call allocates a request ID, invokes runtime object `Call` (object 0), and
returns `CancellablePromise`. Calling `.cancel(cause?)` invokes runtime object
`CancelCall` (object 10) with the same `call-id`; the Go method receives the
request context and can observe `ctx.Done()`. The canonical proof is
`v3/examples/cancel-async/service.go`, whose `LongRunning(ctx, milliseconds)`
selects between a timer and `ctx.Done()`.

Cancellation is cooperative. The JS promise can be cancelled while the Go
operation may already have performed side effects; UI cancellation is not
proof that a native mutation was undone. Cancellation and timeout should be
mapped distinctly by a GWC adapter, and every service should accept
`context.Context` when interruption matters. Promise chains retain the
cancellable type, but cleanup still needs an explicit bounded lifecycle test.

## Services, startup, windows, dialogs, and events

`application.NewService[T]` and `NewServiceWithOptions` are defined in
`v3/pkg/application/services.go`. Optional `ServiceStartup(ctx, options)` gets
an app-lifetime context cancelled before shutdown; optional `ServiceShutdown()`
is called in reverse registration order. `application.Options.Services` controls
registration order. These are native host lifecycle hooks, not frontend hooks.

Native window setup is `app.Window.NewWithOptions(application.WebviewWindowOptions{...})`.
The returned `application.Window` supports `Show`, `Hide`, `Close`,
`EmitEvent`, `OnWindowEvent`, and `RegisterHook`; concrete options and methods
are in `v3/pkg/application/webview_window.go` and the window examples. Window
closing/focus events use `github.com/wailsapp/wails/v3/pkg/events` (see
`v3/examples/events/main.go`). A first app should use one named window and
hash routing; OS deep-link translation is separate work.

Native-to-UI custom events are `app.Event.Emit(name, data...)` or
`window.EmitEvent(name, data...)`. Frontend subscriptions are
`Events.On(name, callback)`, `Events.Once`, or `Events.OnMultiple`; each returns
an unsubscribe function (`v3/internal/runtime/desktop/@wailsio/runtime/src/events.ts`).
Frontend emits are `Events.Emit(name, data)`. Event names should use an explicit
desktop namespace; Wails events are not the local GWC topic registry and should
not be implicitly forwarded to it.

Dialogs are native and synchronous from the Go callback's point of view. The
open-file flow is `app.Dialog.OpenFile().CanChooseFiles(true).AttachToWindow(win).PromptForSingleSelection()`;
the result and error are separate (`v3/examples/dialogs-basic/main.go`). A
user cancel is a normal empty result, not automatically a service failure; the
adapter must preserve that distinction.

## Minimal reproducible isolated example

Use `examples/desktop/wails-counter/` with its own `go.mod`:

```text
go.mod                         # requires GWC module + Wails beta.17
cmd/desktop/main.go             # native package; application.New
frontend/main.go                # //go:build js && wasm; ui.Render + select{}
contracts/                       # portable DTOs only
internal/services/               # native-only service(s), context-aware
assets/assets.go                 # owns //go:embed dist
assets/dist/                     # assembled local files + generated bindings
```

Build the frontend with `GOOS=js GOARCH=wasm go build -o assets/dist/app.wasm
./frontend`; copy the matching Go `lib/wasm/wasm_exec.js` (from the active
`go env GOROOT`; current Go installations use this path), place a local
`index.html` that loads it and
the GWC Wasm module, then run the binding command above with `-b`. The native
main imports the embedding assets package, registers `application.NewService(&CounterService{})`,
creates `WebviewWindowOptions{URL:"/"}`, and runs `app.Run()`.

The target validation contract is: cold start with no server/network; GWC counter
renders; one generated service call round-trips; cancel a long call and verify
the service sees `ctx.Done()`; emit one progress event and unsubscribe it;
open-file dialog cancel remains distinguishable; closing the window releases
frontend listeners. Cancellation propagation belongs to D2; actual dialog
selection/cancel needs its own OS evidence. Do not infer either from D1's build
or basic smoke success. Run this in the nested module explicitly—root `go test
./...` does not cover it.

## Version and scope conclusions

The pin is usable for a no-npm runtime path because generated bindings can
import the host-served `/wails/runtime.js`. It still has a Go 1.25 requirement,
and the stock templates use npm/Vite, so “no npm” applies only to the GWC
frontend assembly path and must be proved by a clean isolated build. Keep the
Wails dependency out of the root module until the spike proves the contract.
Wails beta.17's generated modules, cancellation handle, event unsubscribe, and
asset/runtime routes are sufficient evidence for D0/WAILS-001; they do not yet
establish cross-platform WebView behavior, durable cancellation of mutations,
or production support.
