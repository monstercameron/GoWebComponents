# Desktop frontend bridge

Experimental optional bridge for GWC Wasm frontends hosted by a native desktop
application. The root module uses `/v6`. No native Wails package is imported by
this package; web and desktop UIs share the portable contract.

## Portable file dialogs

Application code no longer needs RPC names or generated Wails types for file pickers:

```go
parseSelection, parseErr := desktop.OpenFile(parseContext, desktop.FileDialogOptions{
    Title: "Choose a document",
    Filters: []desktop.FileFilter{{Name: "Text", Pattern: "*.txt"}},
})
if parseErr != nil { /* unavailable, invalid, remote, cancelled context, etc. */ }
if parseSelection.Cancelled { /* operator dismissed the native picker */ }
```

`OpenFiles`, `OpenDirectory` and `SaveFile` use the same options/result contract.
**SaveFile only selects a destination path; it never writes a file.** Paths are
not permission tokens and must be validated by any later file-access service.
Normal operator dismissal is `Cancelled: true` with no paths and no error;
context cancellation/deadline is an error. Calls use a bounded five-minute wait,
respect earlier caller deadlines, and cannot forcibly dismiss an OS dialog.

The package functions discover the current bridge. The same methods on `Client`
support injected transports for testing. `Client.Supports(FileDialogs)` is a UI
predicate; `Client.Require(FileDialogs)` returns a detailed preflight error.
No arguments to Require means require a compatible live bridge. Unknown feature
names are invalid. Ordinary web/native SSR calls return CodeUnavailable, without
importing Wails. Run async operations in owned tasks, not during component render.

The portable native workflow surface also includes typed clipboard, message,
window, and screen APIs. `Clipboard`, `MessageDialogs`, `WindowControls`, and
`Screens` are independently policy-controlled; `NativeMenus`,
`PersistentStorage`, and `ReportExport` reserve names for host capabilities.
Parse immutable host policy with `ParseFeaturePolicy("all")`, `"none"`, or a
comma-separated allowlist. The effective host set is the intersection of that
policy and adapter capabilities.

Additional optional capabilities cover runtime application/context menus,
caller-owned child windows from host-approved templates, system tray icons,
global shortcuts, window events, screen-coordinate transforms, system inspection,
external HTTP(S) URLs, local file-manager reveal, explicit autostart and printing.
Each has its own policy gate; `NativeBackend` remains source-compatible through
optional extension interfaces. Menu/shortcut callbacks carry typed data, not
frontend-supplied native code. See the [Windows coverage matrix](../docs/windows-api-coverage.md).

For example, shared application code changes its own native title at runtime:

```go
parseClient, parseErr := desktop.Connect()
if parseErr == nil && parseClient.Supports(desktop.WindowControls) {
    _, parseErr = parseClient.ControlWindow(parseContext, desktop.WindowRequest{
        Action: "set-title", Title: "My document — saved",
    })
}
```

Printing, opening external applications and modifying autostart are explicit
operator actions; they are never performed during component render or startup.
Web mode remains available without any Wails dependency.

```go
parsePolicy, _ := desktop.ParseFeaturePolicy("clipboard,message-dialogs")
parseHost := desktop.NewNativeHost(parseBackend, parsePolicy)
parseText, parseErr := parseHost.ClipboardRead(parseContext)
parseReply, parseErr := parseHost.ShowMessage(parseContext,
    desktop.MessageRequest{Kind: "question", Title: "Confirm", Message: "Continue?"})
```

For generated Wails bindings, expose `NativeHost.Execute`, which accepts a
versioned `NativeRequest` and returns a `NativeReply` carrying structured
`interop.ErrorCode` failures. This prevents native errors from becoming an
unclassified remote exception.

## Replaceable native backend

The separately versionable [Wails adapter module](wails/README.md) implements
`FileDialogBackend`. Host wiring is explicit:

```go
parseFiles := desktop.NewFileDialogHost(wailsadapter.NewFileDialogs(), true)
// Register parseFiles as a native service and map its generated SelectPaths
// binding to desktop.FileDialogMethod in the desktop bootstrap.
```

The host's zero value or `false` opt-in denies calls before backend work. It
advertises only enabled support through GetMethods, validates inputs, and returns
portable error envelopes so generated-binding errors don't erase classification.
Custom/fake backends implement one context-aware method; no Wails types cross this
boundary. Native adapters resolve the invoking window from context, never focus.

This service's checks still apply to direct generated-binding calls that bypass
the frontend SDK. They are application policy, not an OS sandbox; unrelated
native code using Wails directly is outside that policy. Keep optional native
imports and generated bindings in the host integration, not shared UI source.

## Host setup

Serve `desktop.BootstrapSource` as a local external ES module (`desktop.js`).
Before starting Wasm, import it and install an explicit generated-binding map:

```js
import { createDesktopTransport } from "/desktop.js";
globalThis.__gwcDesktop = createDesktopTransport({
  methods: { "counter.increment": bindings.CounterService.Increment },
  topics: ["counter.progress"],
  events: runtime.Events,
  platform: "windows",
  hostVersion: "wails/v3.0.0-beta.17"
});
window.addEventListener("pagehide", () => globalThis.__gwcDesktop.close(), { once: true });
```

The host must serve trusted local assets and explicitly register native services.
Non-RPC capabilities, currently installed native menus, can be advertised using
`features: ["native-menus"]`. `Client.Supports(NativeMenus)` checks this explicit
advertisement; it does not invent a menu-install RPC. RPC-backed features require
their complete method set regardless of any feature advertisement.
This JavaScript allowlist and capability marker are feature discovery, **not a
security boundary** against code already executing in the privileged WebView.
`hostVersion` describes application configuration, not an authenticated handshake.

## Frontend calls and events

```go
parseClient, parseErr := desktop.Connect()
parseValue, parseErr := desktop.Call[CounterState](parseContext, parseClient, "counter.increment")
parseStop, parseErr := desktop.Subscribe[Progress](parseContext, parseClient,
    "counter.progress", func(parseValue Progress, parseErr error) { /* update state */ })
```

Run calls in `ui.UseTask`/`UseTaskCtx`; return subscription cancellation from
`ui.UseEffect`. Native/SSR and ordinary browsers without the bootstrap return
`interop.CodeUnavailable`. An injected `Transport` supports native unit testing.
Errors use the existing `interop.Error` codes (missing export, encode/decode,
remote, cancelled, timeout, disposed). A closed window maps to `CodeDisposed`.

`Call` retains a 30-second ceiling, including when given context.Background.
For interactive native dialogs, explicitly opt one call into a longer bounded wait:

```go
parseResult, parseErr := desktop.CallWithTimeout[DialogResult](parseContext,
    parseClient, 5*time.Minute, "dialog.open")
```

The timeout must be positive and no greater than `MaximumRequestTimeout` (ten
minutes); invalid values return `interop.CodeInvalid` before starting native work.
Earlier caller deadlines always win. The Windows tester uses five minutes only
for pickers, message dialogs, and report export; routine calls retain 30 seconds.
Context cancellation forwards `.cancel()` to the original
generated Wails request; native services must cooperate by observing ctx.Done().
Cancellation cannot reverse a completed write or necessarily dismiss an OS dialog;
the user may still need to close that dialog. Closing the native Wails window
also cancels its active native calls via the pinned host runtime.

The default registry allows 256 concurrent requests and 256 subscriptions.
Go polls synchronous JSON envelopes rather than giving js.Func callbacks to
promises. Forgotten/late results cannot invoke released Go callbacks. This bounds
adapter-owned resources, not JavaScript heap retained by arbitrary third-party
never-settling promises. Event bursts keep the **latest** value per subscription
and deliver at most once per 16ms poll; do not use this channel as an audit log.

Payloads are JSON-shaped. Use strings for identifiers beyond JavaScript's exact
integer range; unsafe numeric integers are rejected. Use base64 for bytes and
RFC3339 strings for times. Nil is JSON null. Absent fields follow Go's JSON tags;
arbitrary pointers, functions, cycles, NaN and Infinity are not bridge contracts.

## Durable state

`NewStorageBackend(client)` implements `kvstate.PersistenceBackend` using the
explicit `storage.load/save/delete/keys` methods. Its wire records carry decimal
string versions/timestamps and base64 bytes. The example native service owns a
bounded atomic JSON store at `%AppData%/GWCWailsCounter/storage.json`, guarded by
a Windows OS handle lock. This is a small-app KV implementation, not SQLite or
a distributed database. Versions must increase; stale/equal writes fail, and
deletion preserves a tombstone version for safe recreation.

Select both the backend and its notification path explicitly:

```go
parseOptions := kvstate.Options{
    Name: "my-desktop-app",
    Backend: desktop.NewStorageBackend(parseClient),
    Strategy: kvstate.Immediate{},
    ExternalInvalidation: true,
}
parseStop, parseErr := desktop.SubscribeStorage(parseContext, parseClient,
    parseOptions.Name, parseOnError)
```

The native service emits `storage.changed` only after a successful commit.
SubscribeStorage reloads all mounted keys under the logical name, so coalescing
several commits cannot lose a different key's invalidation. Reloads do not write
or rebroadcast. `ExternalInvalidation` avoids calling the asynchronous native
backend from the browser's synchronous BroadcastChannel callback. Browser
bindings retain their existing default transport.

## Evidence and limits

Run native contracts with `go test ./desktop ./kvstate`, JS contracts with
`node --test desktop/desktop.test.mjs`, and Wasm contracts with the repository's
`tools/go_js_wasm_exec.bat` runner and process-local js/wasm Go target variables.
The isolated example requires its own build and actual native smoke tests;
root tests do not cover the nested module.

See [example](../examples/desktop/wails-counter/README.md),
[adapter verification](../docs/plans/v6-wails-adapter-verification.md), and
[authoritative backlog](../todos.md). Windows amd64/WebView2 is the current tested
target. This is not yet a signed, cross-platform production support commitment.
