# GWC | Interop Library

# GoWebComponents (GWC)

## High-Level Overview

The `interop` library bridges Go values with JavaScript host capabilities and environment-specific adapters.

## Public APIs

### `github.com/monstercameron/GoWebComponents/interop` (`package interop`)
- Functions: `AsError`, `Assign`, `Back`, `Backend`, `Blur`, `Bool`, `BoundingClientRect`, `Call`, `CallDefault`, `Cancel`, `ClassName`, `Clear`, `Click`, `ClientCanExchange`, `ClientProtocolCompatible`, `ClientSupportsEncoding`, `ClientSupportsTopic`, `Close`, `Closed`, `CodeOf`, `CurrentDocument`, `Decode`, `DecodeClientMessage`, `DecodeCrossTabEnvelope`, `DecodeCustomEvent`, `DecodeJSON`, `DecodeMessagePortMessage`, `DecodeSurfaceSignal`, `DecodeWindowEnvelope`, `DecodeWorkerMessage`, `Delete`, `Dispatch`, `Dispose`, `ElementByID`, `ElementsByID`, `Error`, `Events`, `Float`, `Focus`, `Forward`, `Get`, `GetClipboard`, `GetDocument`, `GetDocumentEvents`, `GetGlobalThis`, `GetItem`, `GetLocalStorage`, `GetMany`, `GetMediaQuery`, `GetSessionStorage`, `GetWindowEnv`, `GetWindowEvents`, `GetWindowHistory`, `GetWindowLocation`, `GetWorkerScope`, `GlobalThis`, `Go`, `Hash`, `Href`, `ID`, `ImportModule`, `Int`, `Invoke`, `IsCode`, `IsNull`, `IsUndefined`, `Key`, `Keys`, `Len`, `Listen`, `LoadPersistentJSON`, `LocalStorage`, `Lookup`, `LookupString`, `Matches`, `Media`, `Message`, `Name`, `NavigatorClipboard`, `NewGoWASMWorker`, `ObserveIntersection`, `ObserveResize`, `OpenCrossTabChannel`, `OpenGoWASMWorker`, `OpenMessageChannel`, `OpenPersistentStore`, `OpenSecondaryWindowChannel`, `OpenWindowOpenerChannel`, `OpenWorker`, `Origin`, `Pathname`, `Port1`, `Port2`, `Post`, `PostPorts`, `Present`, `Progress`, `Publish`, `PublishClientBinaryCrossTab`, `PublishClientBinaryWindow`, `PublishClientEvent`, `PublishClientGoodbye`, `PublishClientGoodbyeWindow`, `PublishClientHello`, `PublishClientHelloWindow`, `PublishClientHelloWindowWithCapabilities`, `PublishClientHelloWithCapabilities`, `PublishClientIntent`, `PublishClientInvalidation`, `PublishClientMessage`, `PublishClientQuery`, `PublishClientResult`, `PublishClientWindowMessage`, `PublishIntent`, `PublishLogout`, `PublishRouteFocus`, `PublishSelection`, `PublishSessionExpired`, `PublishSurfaceSignal`, `PushState`, `QuerySelector`, `ReadText`, `Ready`, `Reload`, `RemoveItem`, `Replace`, `ReplaceState`, `Request`, `RequestWorkerDecoded`, `Restart`, `Result`, `ScheduleInterval`, `ScheduleTimeout`, `ScrollIntoView`, `ScrollMetrics`, `Search`, `SessionStorage`, `Set`, `SetFunction`, `SetInterval`, `SetItem`, `SetJSON`, `SetScrollTop`, `SetTimeout`, `SharedWindowEnv`, `State`, `String`, `Subscribe`, `SubscribeClientMessages`, `SubscribeClientWindowMessages`, `SubscribeDecoded`, `SubscribeDecodedCrossTab`, `SubscribeDecodedMessagePort`, `SubscribeDecodedWindow`, `SubscribeDecodedWorker`, `SubscribeSurfaceSignals`, `TagName`, `TargetOrigin`, `Terminate`, `ToGo`, `Transport`, `Truthy`, `Unwrap`, `Value`, `WriteText`
- Types: `BrowserEvent`, `ClientBinaryPayload`, `ClientCapabilities`, `ClientIdentity`, `ClientMessage`, `ClientMessageKind`, `ClientPayloadEncoding`, `Clipboard`, `CrossTabChannel`, `CrossTabChannelOptions`, `CrossTabEnvelope`, `CustomEvent`, `DecodedCrossTabEnvelope`, `DecodedCustomEvent`, `DecodedMessagePortMessage`, `DecodedWindowEnvelope`, `DecodedWorkerMessage`, `Document`, `Element`, `Error`, `ErrorCode`, `EventTarget`, `GoWASMWorkerOptions`, `History`, `IntersectionEntry`, `IntersectionObserverOptions`, `Location`, `MediaQueryEvent`, `MediaQueryList`, `MessageChannel`, `MessagePort`, `MessagePortMessage`, `Module`, `PersistentStore`, `PersistentStoreBlockedEvent`, `PersistentStoreOptions`, `Rect`, `ResizeEntry`, `ScrollIntoViewOptions`, `Storage`, `Subscription`, `SurfaceIntentAction`, `SurfaceIntentSignal`, `SurfaceRouteSignal`, `SurfaceSelectionSignal`, `SurfaceSessionSignal`, `SurfaceSignal`, `SurfaceSignalKind`, `Timer`, `Value`, `WindowChannel`, `WindowChannelOptions`, `WindowEnv`, `WindowEnvelope`, `Worker`, `WorkerMessage`, `WorkerOptions`, `WorkerScope`
- Variables: _none_
- Constants: `ClientError`, `ClientEvent`, `ClientGoodbye`, `ClientHello`, `ClientIntent`, `ClientInvalidate`, `ClientPayloadBinary`, `ClientPayloadJSON`, `ClientPresenceTopic`, `ClientQuery`, `ClientResult`, `CodeBlocked`, `CodeCancelled`, `CodeDecode`, `CodeDisposed`, `CodeEncode`, `CodeInvalid`, `CodeMissingExport`, `CodeNotFunction`, `CodePromiseRejected`, `CodeQuotaExceeded`, `CodeRemote`, `CodeTimeout`, `CodeUnauthorized`, `CodeUnavailable`, `SurfaceIntentCloseWindow`, `SurfaceIntentFocusPanel`, `SurfaceIntentFocusWindow`, `SurfaceIntentOpenPanel`, `SurfaceSignalIntent`, `SurfaceSignalRoute`, `SurfaceSignalSelection`, `SurfaceSignalSession`

## Subfiles And Purpose

- `compat.go` - Core implementation for compat
- `doc.go` - Package-level Go documentation
- `interop.go` - Core implementation for interop
- `interop_core_zero_branches_test.go` - Tests for interop_core_zero_branches behavior
- `interop_native.go` - Native (non-WASM) implementation for interop
- `interop_native_test.go` - Tests for interop_native behavior
- `interop_wasm.go` - WebAssembly-specific implementation for interop
- `interop_wasm_test.go` - Tests for interop_wasm behavior
- `interop_wrappers_test.go` - Tests for interop_wrappers behavior
- `persistence_wasm.go` - WebAssembly-specific implementation for persistence
- `README.md` - Folder-level documentation
- `value_native.go` - Native (non-WASM) implementation for value
- `value_wasm.go` - WebAssembly-specific implementation for value

## File Map

```text
interop/
|-- compat.go
|-- doc.go
|-- interop.go
|-- interop_core_zero_branches_test.go
|-- interop_native.go
|-- interop_native_test.go
|-- interop_wasm.go
|-- interop_wasm_test.go
|-- interop_wrappers_test.go
|-- persistence_wasm.go
|-- README.md
|-- value_native.go
\-- value_wasm.go
```



