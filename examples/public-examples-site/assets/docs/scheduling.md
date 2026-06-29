# Scheduling

This page defines the current runtime scheduling contract for `ui.StartTransition`, `ui.UseTransition`, and `ui.UseDeferredValue`.

It documents the scheduler that ships today, not a speculative concurrent renderer.

## Current Status

Shipped today:

- transition scheduling through `ui.StartTransition(...)`
- component-local transition control through `ui.UseTransition()`
- lagging derived values through `ui.UseDeferredValue(...)`
- transition pending state through `Transition.Pending()`
- browser and runtime tests that prove urgent updates can win ahead of deferred transition work
- example coverage in `examples/public/transition-hooks` and `examples/public/use-deferred-value`

Not shipped today:

- a multi-priority public scheduler ladder for idle, background, animation, or deadline-aware work
- time-sliced rendering that pauses and resumes one in-flight render pass
- a second generic scheduler-pending hook beyond `UseTransition()`

The current public model is narrow on purpose: one urgent lane, one transition lane, and one lagging-value helper.

## Current Priority Classes

The runtime currently supports two scheduling classes:

- urgent work: ordinary `UseState` and `UseAtom` updates outside a transition
- transition work: non-urgent updates scheduled inside `ui.StartTransition(...)`

There is no richer public ladder today for idle, background, animation, or render-deadline-aware work.

That is intentional. The current scheduler is explicit about one distinction only: urgent updates should land first, and transition updates may defer behind them.

## Public API Shape

The current hook-facing API is:

```go
func StartTransition(fn func())
func UseTransition() Transition
func UseDeferredValue[T any](value T) T

type Transition struct {
	Pending() bool
	Start(fn func())
}
```

Use `ui.StartTransition(...)` when the scheduling decision belongs outside the component-local `Transition` handle. Use `transition.Start(...)` when the same component also needs to expose or read `transition.Pending()`.

## Pending-State API

The current pending-state answer is already public:

- `ui.UseTransition()` returns a `Transition`
- `Transition.Start(fn)` schedules non-urgent updates
- `Transition.Pending()` reports whether transition work is currently pending

There is no second generic scheduler-pending hook today because the shipped transition handle already covers the intended caller need: pair a non-urgent update with a typed pending flag in the same component.

The current wasm tests explicitly verify both halves of that contract:

- transition-wrapped public state updates stay deferred before the queued scheduler callback runs
- `Transition.Pending()` reports `true` before flush and `false` after the transition settles

## Interruptibility Today

The current scheduler does not split one long render into smaller interruptible chunks.

What it does today:

- transition work is deferred through the scheduler timeout lane
- urgent updates can commit before that deferred transition callback runs
- once a given render or commit pass begins, the runtime still runs that pass to completion

This means the current model is deferred, but not truly time-sliced.

The current benchmark coverage now includes transition-heavy list refreshes in `internal/runtime/scheduler_benchmark_test.go`. That benchmark is the current measurement hook for deciding whether the runtime needs real work splitting rather than only a deferred lane.

Related runtime coverage also proves:

- `UseState` updates inside a transition defer until the timeout lane runs
- `UseAtom` updates inside a transition defer through that same lane
- when no scheduler is installed, transition callbacks run immediately instead of stalling forever
- overlapping urgent and transition updates settle coherently under production-correctness coverage

## Deferred Values

`ui.UseDeferredValue(...)` is the current answer when the caller wants to keep showing a previous committed value while a transition catches up.

Its behavior today is:

- it returns the current committed value immediately on the first render
- when the input changes, it keeps returning the previous committed value
- it schedules the replacement through `StartTransition(...)`
- once the transition settles, the deferred value catches up to the latest input

Use it for derived lists, expensive summaries, or preview panes where typing or route intent should stay responsive while a heavier derived view lags briefly behind.

## Route Loaders And Async Boundaries

Transition scheduling currently affects local state and atom updates only.

Current interaction rules:

- a transition does not suppress route-loader pending state
- a transition does not override `router` loading or revalidation indicators
- a transition does not suppress `ui.AsyncBoundary` fallback behavior when the async state itself is pending or errored
- `ui.UseDeferredValue(...)` is the intended tool when the caller wants to keep rendering a previous value while a transition catches up

In practice:

- use route-loader pending UI for route and data navigation
- use async boundaries for async subtree fallback decisions
- use transitions for non-urgent local refreshes that should not block urgent interaction

The current examples model that separation directly:

- `examples/public/transition-hooks` uses transitions for typeahead filtering, dashboard tab swaps, and route-style section transitions
- `examples/public/use-deferred-value` demonstrates lagging derived values separately from pending async work
- `examples/public/text-input` demonstrates `UseDebounced(...)` and `UseThrottled(...)`, which are adjacent timing helpers but not scheduler priority lanes

## What Scheduling Is Not

The repo also ships timing-oriented hooks such as `ui.UseDebounced(...)` and `ui.UseThrottled(...)`.

They are useful, but they are not scheduler priority controls.

Use this distinction:

- use transitions when work is semantically non-urgent and should yield to urgent interaction
- use deferred values when a derived view should intentionally lag behind a newer source value
- use debounced or throttled helpers when you need rate limiting or delayed propagation of changing inputs over time

Those helpers can expose their own `Pending()` state, but that pending state means timer-based delay, not transition-lane scheduling.

## Production Build Parity

The runtime should preserve the same scheduling semantics under `-tags production` builds.

The current parity expectation is:

- production-tag builds keep the same urgent-versus-transition behavior
- `Transition.Pending()` remains available and accurate
- debug helpers may disappear, but scheduler-visible behavior must not

The repo now validates this through production-tag wasm compile checks alongside the normal runtime tests.

## Current Boundary

This document describes the current public scheduling slice only.

It does not claim:

- render-deadline budgeting
- cancellation of an already-running render pass
- first-party animation scheduling primitives
- automatic coordination between transitions and route-loader pending surfaces

Those would be separate additions on top of the current two-lane model, not hidden behavior in the existing API.
