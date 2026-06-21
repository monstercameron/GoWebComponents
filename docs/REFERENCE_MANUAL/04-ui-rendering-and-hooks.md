# 04 UI Rendering And Hooks

Use this chapter when you are working in the `ui` package surface: component composition, rendering, hooks, async helpers, overlays, transitions, worker tasks, and the narrow-update path.

It is the right chapter for:

- understanding the normal component model
- choosing the right hook family for local feature state
- separating normal rerendering from explicit narrow-update or worker-oriented paths
- deciding when async boundaries, transitions, overlays, or worker tasks belong in the page

Use another chapter instead when:

- you need typed DOM authoring details: go to [05 HTML Authoring](05-html-authoring.md)
- you need shared state, selectors, or snapshots: go to [06 State And Reactivity](06-state-and-reactivity.md)
- you need routing or route-owned loaders: go to [08 Routing](08-routing.md)
- you need SSR and bootstrap ownership in depth: go to [09 SSR And Hydration](09-ssr-and-hydration.md)

## Overview

`ui` is the primary public runtime surface for GoWebComponents.

Start here when you need:

- `ui.Render` or `ui.CreateElement` to mount a component tree
- local hooks such as `UseState`, `UseEffect`, `UseReducer`, `UseRef`, and `UsePrevious`
- typed event wiring with `UseEvent`
- async helpers such as `AsyncBoundary`, `Lazy`, `UseTask`, and `UseWorkerTask`
- overlays, focus helpers, composite navigation, and form ownership
- explicit narrow-update or parallel-region authoring

The default model is still:

- component functions return `ui.Node`
- hooks drive ordinary rerendering
- the runtime reconciles the tree

Everything else in this chapter layers on top of that default.

## Stability Note

The core rendering and hook surface is mostly `Stable`:

- `ui.Render`
- `ui.CreateElement`
- `ui.Fragment`
- `ui.UseState`
- `ui.UseReducer`
- `ui.UseEffect`
- `ui.UseRef`
- `ui.UsePrevious`
- `ui.UseId`
- `ui.UseEvent`
- `ui.CreateContext`
- `ui.UseContext`

Important surfaces with extra caution:

- `ui.StartTransition`, `ui.UseTransition`, and `ui.UseDeferredValue` are `Experimental`
- `ui.AsyncBoundary` and `ui.Lazy` are `Experimental`
- `ui.ReactiveRegion` is an explicit optimization tool, not the default authoring path
- `ui.ParallelRegion` is a public shell for the worker-backed path and should be treated as a specialized advanced surface

## Minimal Example

The normal `ui` shape is one function component, one local state owner, and one typed event path.

```go
package main

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

// renderCounterApp renders one small interactive component using the default ui model.
func renderCounterApp() ui.Node {
	storeCount := ui.UseState(0)
	handleUserIncrement := ui.UseEvent(func() {
		storeCount.Update(func(getPreviousCount int) int {
			return getPreviousCount + 1
		})
	})

	return Main(
		Class("mx-auto max-w-xl space-y-4 p-6"),
		H1("ui basics"),
		P(Textf("Count: %d", storeCount.Get())),
		Button(Type("button"), OnClick(handleUserIncrement), "Increment"),
	)
}

// main mounts the root component into the browser DOM.
func main() {
	ui.Render(ui.CreateElement(renderCounterApp, nil), "#app")
	utils.WaitForever()
}
```

Why this is still the baseline:

- `ui.CreateElement` keeps component composition explicit
- `ui.UseState` and `ui.UseEvent` cover most early interactive work
- the component rerenders normally without any extra performance or transport machinery

## Production-Shaped Example

Once one feature owns several fields and named transitions, keep the render tree readable and move the workflow into a reducer-backed hook.

```go
package dashboard

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

type dashboardFilterState struct {
	Query        string
	ShowAssigned bool
}

type dashboardFilterAction struct {
	Kind  string
	Value string
}

// buildDashboardFilterState applies one semantic transition to the filter workflow.
func buildDashboardFilterState(getState dashboardFilterState, getAction dashboardFilterAction) dashboardFilterState {
	switch getAction.Kind {
	case "set-query":
		getState.Query = getAction.Value
	case "toggle-assigned":
		getState.ShowAssigned = !getState.ShowAssigned
	}
	return getState
}

// useDashboardFilterWorkflow keeps related local state transitions behind one feature hook.
func useDashboardFilterWorkflow() (dashboardFilterState, ui.Handler, ui.Handler) {
	storeFilter := ui.UseReducer(buildDashboardFilterState, dashboardFilterState{})

	handleUserQuery := ui.UseEvent(func(getEvent ui.InputEvent) {
		storeFilter.Dispatch(dashboardFilterAction{
			Kind:  "set-query",
			Value: getEvent.GetValue(),
		})
	})
	handleUserAssignedToggle := ui.UseEvent(func() {
		storeFilter.Dispatch(dashboardFilterAction{Kind: "toggle-assigned"})
	})

	return storeFilter.Get(), handleUserQuery, handleUserAssignedToggle
}

// renderDashboardToolbar renders a production-shaped local workflow without spreading dispatch logic through the whole tree.
func renderDashboardToolbar() ui.Node {
	getFilter, handleUserQuery, handleUserAssignedToggle := useDashboardFilterWorkflow()

	return Section(
		Class("space-y-4 rounded-2xl border border-slate-200 bg-white p-5"),
		H2("Dashboard filters"),
		Input(
			Type("text"),
			Value(getFilter.Query),
			OnInput(handleUserQuery),
			Placeholder("Search queue"),
		),
		Label(
			Class("flex items-center gap-3"),
			Input(
				Type("checkbox"),
				Checked(getFilter.ShowAssigned),
				OnChange(handleUserAssignedToggle),
			),
			Span("Show assigned only"),
		),
	)
}
```

Why this is a better `ui` pattern for real features:

- the reducer owns named workflow transitions
- the render function stays readable
- the feature can later widen into route params, shared state, or typed resources without rewriting the whole UI model

## Scale-Up Example

In a larger app, scale `ui` by separating global composition, feature hooks, and specialized surfaces such as overlays or async boundaries.

```go
package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"

	"my-app/internal/feature/search"
	"my-app/internal/feature/workspace"
)

// RenderRootApp keeps providers and shell composition at the app layer.
func RenderRootApp() ui.Node {
	return Main(
		Class("min-h-screen bg-slate-50"),
		ui.CreateElement(search.RenderSearchToolbar, nil),
		ui.CreateElement(workspace.RenderWorkspacePanel, nil),
	)
}
```

```go
package search

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// RenderSearchToolbar keeps the feature surface local until the app has a real reason to widen ownership.
func RenderSearchToolbar() ui.Node {
	storeQuery := ui.UseState("")
	getDeferredQuery := ui.UseDeferredValue(storeQuery.Get())
	getTransition := ui.UseTransition()

	handleUserQuery := ui.UseEvent(func(getEvent ui.InputEvent) {
		storeQuery.Set(getEvent.GetValue())
	})

	return Section(
		Class("space-y-3 border-b bg-white p-4"),
		Input(Type("text"), Value(storeQuery.Get()), OnInput(handleUserQuery), Placeholder("Search")),
		P(Textf("Deferred preview: %s", getDeferredQuery)),
		If(getTransition.Pending(), P("Refreshing derived results...")),
	)
}
```

Why this scales:

- `RenderRootApp` owns shell composition
- feature packages own local `ui` workflows first
- transitions or async helpers stay feature-local until there is a product-level reason to coordinate them elsewhere

## API Family Reference

Use this table to choose the right `ui` family before you add more machinery than the workload needs.

| API family | Representative APIs | Stability | Use it when | Prefer something else when |
| --- | --- | --- | --- | --- |
| Mount and composition | `Render`, `CreateElement`, `Fragment`, `Portal` | `Stable` | you need to mount or compose the tree | never; these are core building blocks |
| Local hooks | `UseState`, `UseReducer`, `UseEffect`, `UseRef`, `UsePrevious`, `UseId` | `Stable` | one component or subtree owns the workflow | unrelated branches need shared ownership |
| Typed events | `UseEvent`, `WrapHandler` | `Stable` | you want typed event handlers without raw DOM plumbing | the event should remain in plain HTML props only |
| Context | `CreateContext`, `UseContext` | `Stable` | one subtree needs shared dependency injection | the value really belongs in shared state across unrelated branches |
| Async helpers | `AsyncBoundary`, `Lazy`, `UseTask`, `UseLazyNode` | mostly `Experimental` or advanced | one subtree has async ownership or explicit lazy boundaries | the work is simple enough for ordinary render and effect flow |
| Scheduling | `StartTransition`, `UseTransition`, `UseDeferredValue` | `Experimental` | the update is semantically non-urgent | the work is urgent or simple enough for normal rerendering |
| Form and accessibility | `UseForm`, `AccessibleOverlay`, `UseAnnouncer`, `UseFocusTrap`, `UseCompositeNavigation` | core form surface `Stable`, some helpers more specialized | the feature owns dialog, keyboard, announcement, or form workflows | the UI is still simple semantic markup with no richer interaction model |
| Worker helpers | `UseWorkerTask`, `UseTask` | advanced public surface | the feature has CPU-heavy or backgroundable work | ordinary local state and render flow are enough |
| Narrow-update path | `ReactiveRegion` | explicit optimization surface | one anchored hot region changes much more often than its owner | the component body still needs to rerun for correctness |
| Worker-backed render shell | `RegisterParallelRegion`, `ParallelRegion`, `BuildParallelRegionSourceIDs` | advanced public shell | you are authoring a worker-safe display region intentionally | you only need narrow subscribed rerendering today |
| SSR helpers | `RenderToString`, `RenderToStream`, `Hydrate`, bootstrap helpers | `Stable` entrypoints with deeper operational rules; streaming is advanced | the app owns request-time HTML, streaming async-boundary shells, or hydration | the app is client-only |

## Error Boundaries And Parallel-Region Recovery

Use `ui.ErrorBoundary` when one subtree should recover locally from render, event, or effect panics without treating the whole app shell as failed.

Keep these rules explicit:

- `AsyncBoundary` is for loading and async fallback ownership
- `ErrorBoundary` is for unexpected failure recovery and reset behavior
- boundary fallbacks should be user-meaningful and feature-local
- reset keys should follow route or workflow identity instead of random rerender noise

For `ParallelRegion(...)` and other narrow worker-backed surfaces:

- keep the output display-oriented and deterministic
- keep input source IDs stable and app-owned
- fall back to the main-thread render path when capability checks, worker startup, or troubleshooting signals say the worker path is unhealthy

## DOM Refs And Lifecycle Hooks

Most components never need a handle to a real DOM element, but interactive UI sometimes does — to move
focus, measure, or attach a managed global listener. These hooks provide that without escaping the
reconciler.

- **`ui.UseDOMRef()` + `html.Ref(ref)` / `shorthand.Ref(ref)`** — capture the live element during the
  commit phase. The ref exposes `.Focus()` and releases its element automatically on unmount, so you
  never hold a stale node. No `getElementById`. See [dom-ref](../../examples/public/dom-ref/).
- **`ui.UseAutoFocus(ref, when)`** — focus a referenced element whenever `when` is true: on first mount
  and each time it is revealed. The declarative replacement for the `autofocus` attribute.
- **`ui.OnReady(fn)`** — run a callback exactly once, right after the first commit to the DOM. The
  framework also dispatches a `gwc:ready` DOM event at the same moment, so host-page scripts and test
  harnesses can wait for a real first frame. See [ready-signal](../../examples/public/ready-signal/).
- **`ui.UseGlobalKey(fn)` / `ui.UseDocumentEvent` / `ui.UseWindowEvent`** — managed document/window
  listeners whose underlying `js.Func` lifetime is scoped to the effect (registered on mount, released
  on unmount), so they cannot leak. See [global-events](../../examples/public/global-events/).
- **`ui.UseTimeout` / `ui.UseInterval`** — timers that are cleared automatically when the component
  unmounts.
- **`ui.Download(...)` / `ui.PickFile(...)`** — browser file egress/ingress for save and open flows.
- **`ui.UseForceUpdate()`** — an escape hatch to schedule a rerender when state lives outside the hook
  system.

All of these obey the rules of hooks: call them while the component is rendering, never inside a loop or
conditional. The `tools/hookcheck` analyzer flags violations statically.

## Design Notes And Boundaries

Keep these rules in mind when working in `ui`:

- ordinary component rerendering is still the default model
- use local hooks first and widen ownership only when the app shape requires it
- call hooks only while the component is rendering on the render goroutine; goroutines, async callbacks, and event handlers may update existing state but must not create new hook slots
- transitions are a narrow, two-lane scheduling tool, not a general concurrent renderer
- `AsyncBoundary` and `Lazy` are explicit async subtree tools, not permission to hide ownership boundaries
- `ReactiveRegion` is an opt-in performance tool, not the default programming model
- `ParallelRegion` is a future-facing public shell for worker-backed rendering, but the default public DOM ownership still stays on the main thread
- overlays and accessibility helpers belong where the interaction model is truly richer than ordinary semantic markup

## Common Failure Modes

- reaching for shared state before local hooks and reducers stop fitting
- calling `UseState`, `UseEffect`, `UseAtom`, or another hook from a `go func`, timer callback, or helper invoked after render instead of from the component body
- using transitions for timer-style delay problems that should instead use debounced or throttled helpers
- treating `UseDeferredValue` as a data-fetching primitive instead of a lagging derived-value helper
- adding `ReactiveRegion` before measuring whether the owner rerender is actually a problem
- authoring `ParallelRegion` as if hooks, refs, or arbitrary event closures already run inside worker-rendered output
- scattering overlay or focus logic through ad hoc DOM plumbing instead of using the explicit helper surfaces
- treating `RenderToString` and `Hydrate` as simple mount helpers without respecting the deeper SSR ownership rules

## Validation

Use the smallest examples that prove the `ui` family you are adopting.

Core render and local hooks:

```powershell
go run ./tools/gwc dev -app .\examples\public\ui-render\main.go
go run ./tools/gwc dev -app .\examples\public\use-reducer\main.go
go run ./tools/gwc lint -root . -path .\examples\public\use-state
```

Transitions and deferred values:

```powershell
go run ./tools/gwc dev -app .\examples\public\transition-hooks\main.go
go run ./tools/gwc dev -app .\examples\public\use-deferred-value\main.go
```

Overlay and accessibility helpers:

```powershell
go run ./tools/gwc dev -app .\examples\public\accessible-overlay\main.go
go run ./tools/gwc dev -app .\examples\public\overlay-stack\main.go
```

Worker helpers and advanced region work:

```powershell
go run ./tools/gwc dev -app .\examples\public\worker-text-index\main.go
go run ./tools/gwc dev -app .\examples\testing\parallel-region-basic\main.go
```

## Topic Pagination
Topic 4 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [03 App Shapes](03-app-shapes.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [05 HTML Authoring](05-html-authoring.md)
