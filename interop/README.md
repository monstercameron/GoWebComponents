# Interop Package

The `interop` package is the public browser and module bridge for GoWebComponents applications that need common browser APIs without spreading raw `syscall/js` access through app code.

## Current Surface

- `LocalStorage()` and `SessionStorage()` for typed storage access
- `OpenPersistentStore(...)` for IndexedDB-first durable key/value persistence with explicit fallback storage
- `storage.GetMany(...)` for grouped storage lookups
- `WindowLocation()` and `WindowHistory()` for browser URL and history state work
- `NavigatorClipboard()` for promise-backed clipboard reads and writes
- `SetTimeout(...)` and `SetInterval(...)` for cancelable timer handles
- `WindowEvents()` and `DocumentEvents()` for browser event listeners plus custom-event dispatch
- `MatchMedia(...)` for media-query state and change subscriptions
- `CurrentDocument()` for DOM lookup helpers
- `document.ElementsByID(...)` for grouped host lookups
- `Element` handles for focus, blur, click, scroll, bounding-rect reads, event listeners, resize observers, and intersection observers
- `OpenCrossTabChannel(...)` for named cross-tab messaging with `BroadcastChannel` plus storage-event fallback
- `DecodeCrossTabEnvelope[T](...)` and `SubscribeDecodedCrossTab[T](...)` for typed cross-tab payload handling
- `OpenSecondaryWindowChannel(...)` and `WindowOpenerChannel(...)` for typed popup or secondary-window coordination through `postMessage`
- `DecodeWindowEnvelope[T](...)` and `SubscribeDecodedWindow[T](...)` for typed opener or popup payload handling
- `SurfaceSignal`, `SubscribeSurfaceSignals(...)`, `PublishSessionExpired(...)`, `PublishRouteFocus(...)`, `PublishSelection(...)`, and `PublishIntent(...)` for common multi-surface opener or popup workflows
- `NewWorker(...)` for dedicated browser worker lifecycles
- `worker.Request(...)`, `RequestWorkerDecoded(...)`, and `SubscribeDecodedWorker[T](...)` for typed worker message flows
- `ImportModule(...)` for dynamic module loading, export calls, value reads, and explicit disposal
- `GlobalThis()` plus `Value.Present()`, `Value.Get(...)`, `Value.Set(...)`, `Value.Delete(...)`, `Value.Call(...)`, `Value.Invoke(...)`, `Value.ToGo()`, and `Value.SetFunction(...)` for generic `globalThis` access and temporary JS bridge wiring
- `Decode(...)` for mapping JSON-shaped interop payloads back into typed Go structs
- `DecodeCustomEvent[T](...)` and `SubscribeDecoded[T](...)` for typed custom-event detail handling
- `AsError(...)`, `CodeOf(...)`, and `IsCode(...)` for structured interop error handling

## SSR Behavior

These helpers are browser-only.

On non-`js/wasm` builds they return structured `interop.Error` values with code `unavailable` so SSR and native tests fail clearly instead of silently returning ambiguous zero values.

See [`docs/INTEROP.md`](../docs/INTEROP.md) for lifecycle, SSR, serialization, and performance guidance, [`docs/CROSS_TAB.md`](../docs/CROSS_TAB.md) for the current synchronization contract, [`docs/MULTI_SURFACE.md`](../docs/MULTI_SURFACE.md) for popup and secondary-window coordination, [`docs/WORKERS.md`](../docs/WORKERS.md) for the current worker contract, and [`docs/ERROR_BOUNDARIES.md`](../docs/ERROR_BOUNDARIES.md) for the current route-composition and SSR boundary rules.

## Module Helpers

`ImportModule(...)` resolves a module namespace through dynamic import and returns a `Module` handle.

Use:

- `module.Call(ctx, "namedExport", args...)`
- `module.CallDefault(ctx, args...)`
- `module.Value(ctx, "namedValue")`
- `module.Dispose()`

Returned values are converted into JSON-shaped Go values. Use `Decode(...)` when you want to map an object or array payload into a typed Go target.

## DOM and Event Helpers

Use `CurrentDocument()` when you need to locate a rendered DOM node without dropping into raw `syscall/js`.

- `document.ElementByID("hero")`
- `document.QuerySelector("[data-chart-host]")`

From an `Element` handle you can:

- call `Focus()`, `Blur()`, `Click()`, and `ScrollIntoView(...)`
- read `BoundingClientRect()`
- attach listeners with `Listen(...)`, raw custom-event handlers with `Subscribe(...)`, or typed custom-event handlers with `SubscribeDecoded[T](...)`
- watch layout or visibility changes with `ObserveResize(...)` and `ObserveIntersection(...)`

This is the intended bridge for imperative measurements and third-party widget attachment points. Keep the JS lifetime local to the component that owns the element and cancel listener or observer subscriptions during cleanup.

For browser-defined custom elements, pair these DOM helpers with `html.CustomElement(...)` and the rules in [`docs/CUSTOM_ELEMENTS.md`](../docs/CUSTOM_ELEMENTS.md).
