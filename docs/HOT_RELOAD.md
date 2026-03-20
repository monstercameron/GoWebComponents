# Hot Reload Audit

This note records the current hot reload model in GoWebComponents.

The repo now supports in-page WASM module replacement with a state-preserving reload bridge. It preserves shared atom state and compatible serializable component-local hook state while keeping the existing DOM in place during a hot reload, runs a pre-reload cleanup bridge for old effect resources, and falls back to a remount only when the runtime cannot safely migrate the preserved state.

## Current Model

Hot reload now targets in-page module replacement rather than full page reload.

The intended development model is:

- rebuild the wasm bundle after edits
- keep the browser-side dev server and reload transport external to the app
- export shared and serializable local state before the old module is replaced
- run cleanup for stale effect-owned resources before the new module boots
- restore shared and migratable local state after reload when the runtime can map the saved slots safely
- remount incompatible sections instead of preserving ambiguous local state incorrectly

That means the framework is still rebuilding and re-instantiating the WASM bundle, but it no longer depends on a full page reload when the hot reload path succeeds.

Reasons for that boundary:

- the runtime still rebuilds the bundle as a single WASM module rather than patching one component function body in isolation
- router registrations and other global registries still need explicit restart or reset behavior
- preserving correctness during development is more important than keeping every ambiguous local slot alive
- the current cleanup-plus-restore bridge covers the high-value developer workflow of keeping DOM, shared state, and migratable local state across edits

If the framework revisits subtree-level code patching later, that should be treated as an additional layer on top of the current module-replacement workflow rather than as a prerequisite for useful hot reload.

## Current Blocking Assumptions

### Component identity is still tied to the rendered fiber tree

The runtime treats a component as the function or element type attached to a fiber. When that identity changes, the renderer does not have a remapping layer that can safely preserve the previous fiber subtree.

Relevant code:

- `internal/runtime/reconciler.go`
- `internal/runtime/scheduler.go`
- `internal/runtime/types.go`

Implication:

- edited component code can be reloaded, but the runtime still expects a fresh render pass rather than hot-swapping one component implementation into an already mounted fiber tree

### Hook state is ordered, not named

Hook storage uses call order and per-hook indexes such as `index`, `stateIndex`, `memoIndex`, `refIndex`, `idIndex`, `atomIndex`, `depIndex`, and `cleanupIndex`.

Relevant code:

- `internal/runtime/hooks.go`
- `internal/runtime/state.go`
- `internal/runtime/transition.go`

Implication:

- inserting, removing, or reordering hooks changes the slot layout
- hot replacement can only preserve local hook state when the signature and hook-order check succeeds

### Shared state lives in global registries

Atoms, derived atoms, and router state are stored in process-global registries.

Relevant code:

- `internal/runtime/state.go`
- `router/router.go`
- `router/router_test.go`

Implication:

- hot replacement must either reuse the existing registry entries or perform explicit cleanup before re-registering
- naive module reload can double-register routes, overwrite atom defaults, or leave stale closures subscribed to old runtime objects

### Route registration is mutable and closure-backed

Router registration mutates route maps and stores component factories directly.

Relevant code:

- `router/router.go`
- `router/browser_test_helpers_wasm_test.go`

Implication:

- route updates need a replace-in-place story or a full router remount
- if the hot reload path re-registers routes without cleanup, closures from the old module can remain reachable

### Effect cleanup currently follows unmount and dependency changes

Effects run cleanup when dependencies change or when the fiber is deleted.

Relevant code:

- `internal/runtime/hooks.go`
- `internal/runtime/reconciler.go`
- `internal/runtime/reconciler_test.go`

Implication:

- a hot update needs an explicit cleanup pass for stale listeners, timers, observers, and JS handles before the new code mounts
- otherwise, a reload can preserve visible state but leak old side effects

### JS bridges and event handlers are disposable

Browser-side callbacks are frequently backed by `js.Func` values and other closure-owned handles.

Relevant code:

- `ui/*_wasm.go`
- `examples/86-atlas-commerce-os/shared/atlas/interaction_hooks_wasm.go`
- `tools/livereload/scripts/livereload-client.js`

Implication:

- hot replacement needs a cleanup/rebind strategy for browser-owned callbacks
- preserving state alone is not enough if stale handlers remain wired to the previous module instance

## Practical Conclusion

The current implementation is best described as:

- shared atom state can survive an in-page module reload
- serializable local hook state can survive when the component identity and restorable hook layout remain safely mappable
- effect-owned resources are cleaned up before reload and rebound from the new bundle afterward
- router and other global registries still need explicit restart-or-reset behavior when they own long-lived process state

## Current Signature Model

The runtime now records a development-only component signature for inspected function components.

It captures:

- the component name and qualified function identity
- the optional `key` prop used for list identity
- the ordered hook-kind sequence observed during render

That signature is surfaced through runtime inspection and devtools so development-time tooling can decide whether a change is shape-compatible or should fall back to a remount.

## State Migration Policy

Shape-changing component edits now get a guarded local-state migration layer for serializable hooks.

The current rule is:

- if the component identity and `key` still match, shared atom state may restore through the snapshot bridge
- local `state`, `memo`, `ref`, `id`, and `fetch` slots may restore when their filtered serializable hook layout still matches or only grows or shrinks at the tail
- non-serializable hook changes such as adding or removing `effect`, `callback`, or wrapped function hooks no longer force a remount by themselves
- the runtime still remounts when the serializable hook layout changes in a way it cannot map safely

This means the framework now performs best-effort migration for the serializable subset, but it still refuses to guess when the preserved slots would become ambiguous.

Reasons for that policy:

- serializable hook state is already captured by kind-specific slot arrays, so a large class of shape changes can restore safely without a full remount
- preserving non-serializable hook changes like `effect` and `callback` as cleanup-plus-rebind operations avoids dropping unrelated local state unnecessarily
- explicit remount behavior is still the safer fallback when the runtime cannot prove that a local state restore is still correct

If the framework widens migration further later, it should continue to do so by adding explicit mapping rules instead of restoring ambiguous serializable slots by guesswork alone.

## Async And Router Restart Policy

Async work and router-owned runtime state are restart-first on hot reload.

The current rule is:

- atom snapshots and signature-compatible component-local hook values may restore
- pending async work from the old module instance is not treated as resumable unless a hook explicitly restores a safe serializable view of it
- router registrations, navigation listeners, route factory closures, guards, loaders, and metadata ownership should be rebuilt from the new bundle instead of preserved from the old one

In practice that means a hot reload should prefer restarting these surfaces from clean code rather than trying to continue in-flight work that still points at old closures, goroutines, or browser callbacks.

Why this boundary exists:

- fetch hooks can restore a serializable snapshot of visible fetch state, but the in-flight goroutine and JS promise chain still belong to the old module instance
- router state is stored in mutable registries and listener wiring, so preserving it across code changes risks double-registration and stale closure capture
- async guards, loaders, timers, and event subscriptions are correctness-sensitive; restarting them is safer than pretending they can transparently survive a bundle swap

Operationally, the preserve-versus-restart rule is:

- preserve user-visible snapshot data when it is already modeled as atom state or a compatible serializable hook value
- restart work that depends on active goroutines, callbacks, router listener wiring, or route-owned closure state
- treat restart as the default unless a state class gains an explicit restore contract with cleanup and replay rules

## Fallback Diagnostics

When hot reload drops component-local state and remounts instead, the runtime now emits a development warning instead of failing silently.

That warning is reported through the normal runtime diagnostics surface and includes:

- the component path and component stack when the runtime can attribute the remount to a specific fiber
- whether the fallback happened because component identity changed, the `key` changed, or the hook signature changed
- a hook-order summary when the remount was caused by a shape mismatch

The goal is not to make remounts disappear. The goal is to make them explain themselves clearly enough that developers can tell whether a local state drop was expected.

## Callback And Effect Resource Policy

Callback values and effect-owned resources now use cleanup plus rebind semantics during hot reload.

That includes:

- `ui.UseCallback` closures that are rebuilt from the new bundle after reload
- effect-owned listeners, timers, observers, and JS handles managed through `ui.UseEffect`
- wrapped event-handler functions that must release old browser-owned wrappers before the new runtime attaches replacements

The current model is not to serialize those closures. Instead, the old runtime cleans them up before reload and the new runtime recreates them from fresh code.

Reasons for that boundary:

- callback identity alone is not enough to prove that the closure still points at valid runtime state after a bundle reload
- effect resources need ordered teardown and re-registration, not raw value restoration
- explicit cleanup before reload avoids leaking stale listeners, timers, and wrapped JS handlers from the previous module instance

So the current policy is:

- preserve serializable state
- run effect cleanup before the old module is replaced
- rerun effect setup from fresh code after the new module boots
- rebuild callback closures from the new bundle instead of trying to transport them across reloads

If the framework widens this area later, it should do so by adding more explicit cleanup, replay, and ownership contracts rather than a generic callback snapshot feature.

## Effect Refresh

`Runtime.RefreshEffectsForFiber(...)` is the current development-time hook for stale side effects.

It:

- runs any registered effect cleanups for the target subtree
- clears the cleanup slots so the old teardown is not replayed
- bumps an internal effect-generation counter

On the next render, `ui.UseEffect` sees the newer generation and reruns the effect even when the dependency list is unchanged.

## Failure Recovery

The browser reload client treats a hot-update exception as a recoverable failure.

Its current policy is:

- save the current exported app snapshot before attempting hot reload
- if the WASM hot-update path throws or cannot complete, record the failure and force a full reload
- preserve the stored snapshot so the post-reload restore step can rebuild the previous state

That means the development session falls back to a known-good full reload instead of leaving the page half-updated.

## Snapshot Transport

Hot reloads now use the live-reload server as the snapshot courier.

The current handshake is:

- the server asks connected clients to export their current app snapshot before a hot build completes
- the client replies with a `state_snapshot` payload
- the server carries that payload through the `build_complete` message when the build succeeds
- the browser reload client prefers the forwarded snapshot, but still falls back to its local session-storage copy if needed

This keeps the transport optional while still giving the server a concrete role in state restoration.

## Preservation Boundary

This is the explicit state boundary the current reload bridge supports.

### Preserved today

- shared atom values restored through `state.ExportSnapshot()` and `state.ImportSnapshot()`
- compatible component-local hook state for `ui.UseState`, `ui.UseRef`, and `ui.UseId`
- compatible `ui.UseMemo` values when the dependency list still matches
- DOM identity and browser-managed element state for reused host nodes
- JSON-compatible reload payloads persisted by the live-reload client through `sessionStorage`

### Not preserved today

- `ui.UseCallback` and other callback closures that cannot be serialized safely across a module reload
- `ui.UseEffect` and other effect-owned side effects unless the component remounts cleanly
- router internal state, navigation listeners, and route factory closures when the signature changes
- pending async work owned by the old module instance unless it is explicitly cancelled and re-created
- DOM state for nodes that fail compatibility checks and are remounted

### Intentionally reset on reload

- event listeners and timers owned by the previous JS module instance
- stale route/component closures from the old bundle
- any transient debug or diagnostic panel state that is not part of the atom snapshot

### Boundary Rule

If a value is not serializable through the snapshot bridge or is not owned by `state` atoms, treat it as remount-only for now.

That makes the current behavior predictable:

- shared data can survive development reloads
- local behavior can survive when the new bundle is shape-compatible
- future work can widen the boundary one state class at a time instead of promising resumability everywhere at once
