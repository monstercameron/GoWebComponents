# Interop Guidance

This page documents the supported usage rules for the public `interop` package.

Use it when an app needs browser APIs, DOM measurement, observers, or dynamic module access without dropping back to raw `syscall/js`.

## At A Glance

- Prefer `interop` over raw `syscall/js` whenever the public package already covers the browser capability you need.
- Keep interop ownership local: components or features that create subscriptions, modules, workers, or element handles should also clean them up.
- Treat `interop` values as typed boundary objects or JSON-shaped payloads, not as long-lived shared runtime state.
- Handle browser unavailability explicitly on SSR or native paths through structured `interop.Error` codes rather than assuming browser globals exist.

## Quick API Chooser

Use this rule of thumb:

- choose `GetDocument()` and `Element` helpers when a component needs measurement, focus, observers, or imperative widget attachment
- choose `GetWindowEvents()` or `GetDocumentEvents()` when browser events or custom events should stay inside the supported event bridge
- choose `ImportModule(...)` when a feature needs dynamic module loading and explicit disposal
- choose `OpenCrossTabChannel(...)`, `OpenSecondaryWindowChannel(...)`, `WindowOpenerChannel(...)`, `OpenWorker(...)`, or `OpenMessageChannel(...)` when coordination crosses tabs, windows, or workers instead of only one DOM tree

## Scope

The public `interop` package is the supported bridge for:

- storage, history, location, clipboard, timers, media queries, cross-tab channel helpers, and popup or secondary-window coordination
- document lookup and element handles
- browser event listeners and custom events
- dedicated browser worker lifecycles and typed worker message envelopes
- resize and intersection observers
- dynamic module import and exported function calls

If a use case is already covered here, prefer `interop` over ad hoc `syscall/js`.

## Example Shape

```go
document, err := interop.GetDocument()
if err != nil {
	return nil
}

host, ok, err := document.QuerySelector("[data-chart-host]")
if err != nil || !ok {
	return nil
}

rect, err := host.BoundingClientRect()
if err != nil {
	return nil
}

module, err := interop.ImportModule(context.Background(), "/static/chart.mjs")
if err != nil {
	return nil
}
defer module.Dispose()

_, err = module.CallDefault(context.Background(), map[string]any{
	"width":  rect.Width,
	"height": rect.Height,
})
return err
```

This is the intended shape: resolve the browser handle through `interop`, convert what you need into Go values, keep ownership local, and dispose or unsubscribe when the owner is done.

## Lifetime And Cleanup Rules

Keep interop handles scoped to the component or function that owns them.

- Cancel every `Subscription` returned by `Listen(...)`, `Subscribe(...)`, `ObserveResize(...)`, `ObserveIntersection(...)`, or media-query subscriptions during cleanup.
- Dispose every imported `Module` once the calling component, route, or feature no longer owns it.
- Do not keep long-lived global references to `Element` handles from short-lived route content. Re-resolve them from `GetDocument()` when the owning UI is mounted again.
- Treat event payloads and DOM measurements as snapshots. Read what you need, convert it into Go values, and avoid storing raw browser handles in shared state.
- When route changes or conditional rendering can replace a node, reacquire the `Element` handle after the new subtree is committed instead of assuming the old handle still points at a live host element.

## SSR Guardrails

`interop` is browser-only by design.

On non-`js/wasm` builds:

- constructors such as `GetLocalStorage()`, `GetDocument()`, `GetWindowEvents()`, and `ImportModule(...)` return `interop.Error` with code `unavailable`
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
- `examples/90-browser-interop`
- `examples/91-worker-text-index`
- `examples/95-multi-window-console`
- `examples/97-multi-client-presence`
- `examples/97-multi-client-binary`
- `examples/88-web-components`
- `examples/94-cross-tab-sync`
- `docs/CROSS_TAB.md`
- `docs/MULTI_SURFACE.md`
- `docs/WORKERS.md`
- `docs/CUSTOM_ELEMENTS.md`

## Review Checklist

- does the code use the supported `interop` surface before reaching for raw `syscall/js`
- are subscriptions, observers, module handles, and worker handles cancelled or disposed by the same feature that created them
- are SSR and native paths handling `CodeUnavailable` explicitly instead of assuming browser APIs exist
- are payloads crossing the boundary kept JSON-shaped and decoded once instead of scattered repeated property reads
- is the code reacquiring DOM handles after remounts or route changes instead of caching stale element references
