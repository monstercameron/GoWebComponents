# Scheduling

This page defines the current runtime scheduling contract for `ui.StartTransition`, `ui.UseTransition`, and `ui.UseDeferredValue`.

It documents the scheduler that ships today, not a speculative concurrent renderer.

## Current Priority Classes

The runtime currently supports two scheduling classes:

- urgent work: ordinary `UseState` and `UseAtom` updates outside a transition
- transition work: non-urgent updates scheduled inside `ui.StartTransition(...)`

There is no richer public ladder today for idle, background, animation, or render-deadline-aware work.

That is intentional. The current scheduler is explicit about one distinction only: urgent updates should land first, and transition updates may defer behind them.

## Pending-State API

The current pending-state answer is already public:

- `ui.UseTransition()` returns a `Transition`
- `Transition.Start(fn)` schedules non-urgent updates
- `Transition.Pending()` reports whether transition work is currently pending

There is no second generic scheduler-pending hook today because the shipped transition handle already covers the intended caller need: pair a non-urgent update with a typed pending flag in the same component.

## Interruptibility Today

The current scheduler does not split one long render into smaller interruptible chunks.

What it does today:

- transition work is deferred through the scheduler timeout lane
- urgent updates can commit before that deferred transition callback runs
- once a given render or commit pass begins, the runtime still runs that pass to completion

This means the current model is deferred, but not truly time-sliced.

The current benchmark coverage now includes transition-heavy list refreshes in `internal/runtime/scheduler_benchmark_test.go`. That benchmark is the current measurement hook for deciding whether the runtime needs real work splitting rather than only a deferred lane.

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

## Production Build Parity

The runtime should preserve the same scheduling semantics under `-tags production` builds.

The current parity expectation is:

- production-tag builds keep the same urgent-versus-transition behavior
- `Transition.Pending()` remains available and accurate
- debug helpers may disappear, but scheduler-visible behavior must not

The repo now validates this through production-tag wasm compile checks alongside the normal runtime tests.
