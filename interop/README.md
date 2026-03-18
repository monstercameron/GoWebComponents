# Interop Package

The `interop` package is the public browser and module bridge for GoWebComponents applications that need common browser APIs without spreading raw `syscall/js` access through app code.

## Current Surface

- `LocalStorage()` and `SessionStorage()` for typed storage access
- `WindowLocation()` and `WindowHistory()` for browser URL and history state work
- `NavigatorClipboard()` for promise-backed clipboard reads and writes
- `SetTimeout(...)` and `SetInterval(...)` for cancelable timer handles
- `WindowEvents()` and `DocumentEvents()` for custom-event dispatch and subscription
- `MatchMedia(...)` for media-query state and change subscriptions
- `ImportModule(...)` for dynamic module loading, export calls, value reads, and explicit disposal
- `Decode(...)` for mapping JSON-shaped interop payloads back into typed Go structs

## SSR Behavior

These helpers are browser-only.

On non-`js/wasm` builds they return structured `interop.Error` values with code `unavailable` so SSR and native tests fail clearly instead of silently returning ambiguous zero values.

## Module Helpers

`ImportModule(...)` resolves a module namespace through dynamic import and returns a `Module` handle.

Use:

- `module.Call(ctx, "namedExport", args...)`
- `module.CallDefault(ctx, args...)`
- `module.Value(ctx, "namedValue")`
- `module.Dispose()`

Returned values are converted into JSON-shaped Go values. Use `Decode(...)` when you want to map an object or array payload into a typed Go target.
