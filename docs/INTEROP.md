# Interop Guidance

This page documents the supported usage rules for the public `interop` package.

Use it when an app needs browser APIs, DOM measurement, observers, or dynamic module access without dropping back to raw `syscall/js`.

## Scope

The public `interop` package is the supported bridge for:

- storage, history, location, clipboard, timers, media queries, cross-tab channel helpers, and popup or secondary-window coordination
- document lookup and element handles
- browser event listeners and custom events
- dedicated browser worker lifecycles and typed worker message envelopes
- resize and intersection observers
- dynamic module import and exported function calls

If a use case is already covered here, prefer `interop` over ad hoc `syscall/js`.

## Lifetime And Cleanup Rules

Keep interop handles scoped to the component or function that owns them.

- Cancel every `Subscription` returned by `Listen(...)`, `Subscribe(...)`, `ObserveResize(...)`, `ObserveIntersection(...)`, or media-query subscriptions during cleanup.
- Dispose every imported `Module` once the calling component, route, or feature no longer owns it.
- Do not keep long-lived global references to `Element` handles from short-lived route content. Re-resolve them from `CurrentDocument()` when the owning UI is mounted again.
- Treat event payloads and DOM measurements as snapshots. Read what you need, convert it into Go values, and avoid storing raw browser handles in shared state.
- When route changes or conditional rendering can replace a node, reacquire the `Element` handle after the new subtree is committed instead of assuming the old handle still points at a live host element.

## SSR Guardrails

`interop` is browser-only by design.

On non-`js/wasm` builds:

- constructors such as `LocalStorage()`, `CurrentDocument()`, `WindowEvents()`, and `ImportModule(...)` return `interop.Error` with code `unavailable`
- zero-value wrappers also fail clearly through the same error shape instead of silently succeeding

Recommended pattern:

1. Keep browser-only interop inside client entrypoints, event handlers, or effects.
2. Let SSR paths branch before they depend on browser APIs.
3. When a shared helper can run on both server and client, handle `CodeUnavailable` explicitly instead of assuming the browser exists.

## Structured Errors

`interop` failures are reported through `interop.Error`.

Use:

- `interop.IsCode(err, interop.CodeUnavailable)`
- `interop.AsError(err)`
- `interop.CodeOf(err)`

Current codes cover unavailable browser APIs, rejected promises, disposed handles, encode and decode failures, missing exports, non-function exports, and invalid input.

Treat these codes as the stable branch point for app-level fallback behavior and logging.

## Serialization Guidance

Interop values cross the Go and browser boundary best when they stay JSON-shaped.

- strings, booleans, and numbers cross directly
- structs, slices, maps, and arrays are encoded as JSON-shaped payloads
- use `interop.Decode(...)` to project object or array payloads into typed Go structs
- prefer plain structs or maps for custom-event detail values and module return payloads
- avoid passing functions, cyclic structures, or opaque host objects through `Decode(...)`
- keep large payloads coarse-grained; repeated tiny conversions create more boundary traffic than one typed payload

When you need a real browser object such as a DOM node or file handle, keep it in the dedicated wrapper type (`Element`, `ui.File`, module handle) instead of trying to JSON-round-trip it.

## Performance Guidance

Interop overhead is usually dominated by repeated boundary crossings, not single calls.

- read once, then decode into a Go struct instead of calling `Get(...)` repeatedly from app code
- prefer `storage.GetMany(...)` and `document.ElementsByID(...)` when one feature needs a grouped read instead of many scattered single-key or single-node lookups
- prefer one subscription with typed fan-out in Go over many duplicated listeners on the same target
- cancel observers and listeners as soon as their owner unmounts
- avoid polling for layout when `ObserveResize(...)` or `ObserveIntersection(...)` is enough
- reuse one imported `Module` handle while a feature is active instead of repeatedly importing the same module
- batch application logic on the Go side after the browser callback fires; keep the callback itself narrow

## Preferred Examples

The clearest current interop examples are:

- `interop/interop_wasm_test.go`
- `examples/87-ssr-secure-forms`
- `examples/90-browser-interop`
- `examples/91-worker-text-index`
- `examples/88-web-components`
- `examples/94-cross-tab-sync`
- `docs/CROSS_TAB.md`
- `docs/MULTI_SURFACE.md`
- `docs/WORKERS.md`
- `docs/CUSTOM_ELEMENTS.md`
