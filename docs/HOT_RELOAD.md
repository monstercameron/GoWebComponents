# Hot Reload

GoWebComponents now treats state-preserving hot reload as a first-class development feature for standalone `js/wasm` apps.

## At A Glance

- Enable hot reload from the app with `hotreload.Enable()` or `hotreload.Configure(...)`.
- Run the development loop through `go run ./tools/gwc dev -app ...` unless you specifically need the lower-level shell wrappers.
- Expect state preservation for shared atoms and compatible local hook state, not arbitrary in-place code patching.
- Use `ResetKey` or `ui.HotReloadBoundary(...)` when an edit should intentionally remount or discard preserved state.
- Treat `gwc dev` as the default supported entrypoint: it enables hot reload by default on successful rebuilds, preserves state when the app and edit are compatible, and falls back to a remount or full reload when they are not.

## Quick Development Path

Use this path by default:

- add `hotreload.Enable()` to a standalone wasm app during development
- switch to `hotreload.Configure(...)` when you need an atom allowlist or explicit reset behavior
- start the loop with `go run ./tools/gwc dev -app .\path\to\main.go`
- treat `tools/dev.ps1`, `tools/dev.sh`, and `go run ./tools/livereload ...` as compatibility or lower-level entrypoints rather than the primary documented workflow

The public surface has two parts:

- the `hotreload` package for app-side enablement and snapshot control
- the `gwc dev` launcher command, with `tools/dev.ps1`, `tools/dev.sh`, and `tools/livereload` remaining available as lower-level wrappers

The repo supports in-page WASM module replacement with a state-preserving reload bridge. It preserves shared atom state and compatible serializable component-local hook state while keeping the existing DOM in place during a hot reload, runs a pre-reload cleanup bridge for old effect resources, and falls back to a remount only when the runtime cannot safely migrate the preserved state.

`gwc dev` is the canonical supported workflow for this behavior. The launcher forwards `-hot=true` by default, so compatible rebuilds attempt preserve-state hot reload first; incompatible rebuilds still succeed through the existing remount or full-page reload fallback path instead of asking teams to switch to a different dev command.

## Quick Start

### 1. Enable hot reload in your app

```go
package main

import (
	"github.com/monstercameron/GoWebComponents/hotreload"
	"github.com/monstercameron/GoWebComponents/ui"
)

func main() {
	hotreload.Enable()
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}
```

If you want to limit exported atom state to specific ids:

```go
hotreload.Configure(hotreload.Config{
	AtomIDs: []string{"session", "draft", "filters"},
})
```

If an edit should intentionally invalidate preserved state, bump `ResetKey`:

```go
hotreload.Configure(hotreload.Config{
	ResetKey: "cart-schema-v2",
})
```

### 2. Run the hot reload dev server

Preferred repo workflow:

```powershell
go run ./tools/gwc dev -app .\examples\98-hot-reload\main.go
```

Lower-level compatibility entrypoints still work when you need direct control over the livereload server.

Windows:

```powershell
.\tools\dev.ps1 -App .\examples\98-hot-reload\main.go -Root .\examples\98-hot-reload -Html .\examples\98-hot-reload\hot-reload.html -Port 8099
```

Unix-like systems:

```bash
./tools/dev.sh ./examples/98-hot-reload/main.go ./examples/98-hot-reload ./examples/98-hot-reload/hot-reload.html
```

Then open the served page and edit Go files. On successful rebuilds, the client will try an in-page module swap before falling back to a full reload.

## Public API

The first-class app API lives in `github.com/monstercameron/GoWebComponents/hotreload`.

### `hotreload.Enable()`

Installs the default bridge and attempts to restore any pending snapshot persisted by the dev client.

### `hotreload.Configure(hotreload.Config{...})`

Installs or reconfigures the bridge with explicit options.

Current config fields:

- `AtomIDs`: optional allowlist of atom ids to export through the snapshot bridge
- `ResetKey`: optional explicit snapshot version. When it changes, older snapshots are discarded instead of restored

### `hotreload.Disable()`

Removes the browser bridge from the page.

### `hotreload.Enabled()`

Reports whether the bridge is currently enabled in the running app.

### `hotreload.ExportSnapshot()`

Returns the current JSON payload used by the live-reload client.

### `hotreload.ImportSnapshot(payload)`

Restores a previously exported payload.

The browser bridge now also exposes restore outcomes and hot-reload diagnostics so the live-reload UI can show whether a rebuild restored state, intentionally reset it because `ResetKey` changed, or fell back to remounting part of the tree.

The same bridge now exposes recent route and async restart activity, including router loader and guard logs plus explicit pending-fetch restart notices recorded during hot reload prepare.

During rebuilds, the live-reload UI now also shows a pre-reload compatibility plan before the new bundle swaps in. That plan reports whether the incoming edit is expected to preserve compatible local state, remount changed subtrees, restart router or async work, or force a full reload so the next reload step is explicit instead of heuristic.

For subtree-scoped resets, wrap a section in `ui.HotReloadBoundary(...)` and change its `ResetKeys` when that part of the tree should intentionally remount on the next hot reload without forcing an app-wide `ResetKey` bump.

### `hotreload.Prepare()`

Runs cleanup needed before the old runtime instance is replaced.

## Compatibility Wrapper

`utils.EnableHotReload(true)` and `utils.InstallHotReloadBridge(...)` still work, but they are now compatibility wrappers over the `hotreload` package.

New app code should prefer `hotreload.Enable()` or `hotreload.Configure(...)` directly.

On non-browser builds, the public `hotreload` functions degrade to no-ops or empty results rather than pretending hot reload is available outside `js/wasm`.

## Dev Server API

The supported dev-server entrypoints are:

- `go run ./tools/gwc dev -app ...`
- `tools/dev.ps1`
- `tools/dev.sh`
- `go run ./tools/livereload -app ...`

Use `gwc dev` as the default documented entrypoint. The other commands remain useful when you need to debug or script the underlying livereload server directly.

The preferred flag names are now:

- `-App` / `-app`: app entrypoint or app directory
- `-Root` / `-root`: served root
- `-Html` / `-html`: HTML shell path
- `-Wasm` / `-wasm`: wasm output path

Legacy aliases still work for compatibility:

- `-Main` / `-main`
- `-Index` / `-index`
- `-Output` / `-output`

## How It Works

The hot reload loop is:

1. the dev server watches Go files and rebuilds the wasm bundle
2. before the old bundle is replaced, the browser exports a snapshot payload
3. the app runtime runs `hotreload.Prepare()` to clean up stale effect-owned resources
4. the client swaps in the rebuilt wasm module
5. the app restores the snapshot through `hotreload.ImportSnapshot(...)`
6. if the swap fails, the client falls back to a full page reload and reuses the stored snapshot on the next load

`ResetKey` provides the first explicit reset control. This is the opt-in answer for edits where preserved state would be misleading, such as changing a state initializer from `ui.UseState(4)` to `ui.UseState(5)` and wanting the next rebuild to start fresh.

## Example Surfaces

Use these repo examples as the reference flows:

- `examples/98-hot-reload`: smallest end-to-end hot reload sandbox
- `test/testapp/main.go`: regression app that exercises state restore, effect cleanup, and failure recovery
- `examples/12-portfolio-site`: larger routed app that enables the public `hotreload` package
- `examples/92-protected-routes`: routed loader plus shared-cache surface for guarded navigation and route-owned async data
- `examples/93-ssr-cache-bootstrap`: SSR-seeded shared-cache surface for bootstrap-backed startup and client revalidation

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

Broader validation should not stop at the isolated sandbox. The current manual hot-reload matrix is:

- `examples/12-portfolio-site` for routed shell continuity
- `examples/92-protected-routes` for guarded navigation, route loaders, and shared cache reuse
- `examples/93-ssr-cache-bootstrap` for SSR bootstrap seeds and cached resource resume behavior

Reasons for that boundary:

- the runtime still rebuilds the bundle as a single WASM module rather than patching one component function body in isolation
- router registrations and other global registries still need explicit restart or reset behavior
- preserving correctness during development is more important than keeping every ambiguous local slot alive
- the current cleanup-plus-restore bridge covers the high-value developer workflow of keeping DOM, shared state, and migratable local state across edits

If the framework revisits subtree-level code patching later, that should be treated as an additional layer on top of the current module-replacement workflow rather than as a prerequisite for useful hot reload.

The current HMR-v2 workflow now goes one step further than plain full-snapshot restore: `ui.CreateElement(...)` produces stable runtime-recognized component handles instead of using the shared `ui.renderComponent` function as the only component identity, those handles now own the current implementation binding instead of smuggling it through element props, the dev build loop emits a changed-component manifest alongside successful builds, and the restore path now uses that manifest to preserve unchanged compatible component state while remounting changed subtrees. This is still module replacement plus selective restore, not literal in-place WASM code patching.

## Review Checklist

- does the app enable hot reload explicitly instead of assuming the dev server can preserve state by itself
- does the documented workflow use `gwc dev` first and reserve lower-level wrappers for special cases
- are `ResetKey` and `ui.HotReloadBoundary(...)` used when preserved state would be misleading after an edit
- do teams understand that incompatible hook-order or component-shape edits can still force remount behavior
- are production expectations kept separate from development-only hot reload behavior

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
- the embedded client asset under `tools/livereload/scripts/livereload-client.txt`

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
