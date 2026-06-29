# Fine-Grained Reactivity

Last updated: 2026-03-20

This document defines the first shipped boundary for fine-grained reactivity in GoWebComponents.

The goal is narrower updates for high-frequency UI paths so atom or signal-style changes do not always force full component or page-level rerenders.

## At A Glance

- Fine-grained reactivity is an explicit performance tool, not a replacement for the normal component model.
- The default remains hook-driven component rerender and normal reconciliation.
- The current shipped narrow path is `ui.ReactiveRegion(...)` backed by explicit `state.Atom[...]`, `state.Derived[...]`, or `state.UseSelector(...)` sources.
- The narrow path is for hot display regions where structure is stable and the owning component does not need to rerun.
- If props, context, hook dependencies, routing state, hydration recovery, or boundary state are involved, the runtime should fall back to normal reconciliation.

## Quick Decision Guide

Use normal component rerender when:

- the update changes layout, branching, or which hooks run
- the value comes from local hook state that the component body needs to read again
- the subtree participates in routing, async boundaries, hydration recovery, or error recovery
- the region is not anchored to a stable local DOM area

Use a fine-grained region when:

- one hot shared value changes much more often than the surrounding layout
- the owning component should stay stable while a small leaf or host-only subtree updates
- the data already lives in a shared atom, derived source, or a selector projection
- you want a measurable reduction in rerender and allocation work on dashboards, inspectors, editors, or similar high-frequency surfaces

Rule of thumb: if you would describe the optimization as "update this one anchored display region without rerunning the whole owner", `ui.ReactiveRegion(...)` is the right direction.

## Example Shape

The current public mental model is: keep structure and events in the component, keep shared hot values in `state`, and isolate the hot display path behind an explicit subscribed region.

```go
package dashboard

import (
	"strconv"

	. "github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/state"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

type dashboardModel struct {
	Hot   int
	Title string
}

func HotPanel() ui.Node {
	model := state.UseAtom("dashboard-model", dashboardModel{Hot: 0, Title: "Orders"})
	hot := state.UseSelector("dashboard-model-hot", model, func(value dashboardModel) int {
		return value.Hot
	})

	return Div(
		Class("panel"),
		H2(Text(model.Get().Title)),
		ui.ReactiveRegion(func() ui.Node {
			return Span(
				Class("panel-value"),
				Text(strconv.Itoa(hot.Get())),
			)
		}, hot),
	)
}
```

Why this shape matters:

- the component still owns layout and event wiring
- the selector narrows a larger shared model to the hot value that actually drives the region
- the reactive region gives the runtime a clear subscribed boundary instead of relying on hidden dependency capture
- the surrounding component can stay stable while the hot value updates

## Goal

The first pass should reduce broad rerender work in cases such as:

- dashboard counters and status panels
- large filtered or sorted lists
- spreadsheet-style cells and inline editors
- live search and typeahead result regions
- inspector panes and developer tooling overlays

The target is not to replace the existing component model. The target is to add an explicit narrow-update path where a smaller subscribed region can update without rerunning unrelated component code.

## Current Runtime Baseline

Today the runtime behaves at component-fiber granularity:

- state and atom updates call `ScheduleUpdateForFiber(...)`
- the target fiber and its ancestors are marked dirty
- work is scheduled from the root
- normal reconciliation decides the final DOM changes

That model is still the default. The first fine-grained pass adds a narrower option rather than replacing it.

## First Shipped Boundary

The first shipped boundary is: explicit subscribed render regions inside an existing component tree.

That means:

- a component may create a narrow reactive region whose output is tied to one or more explicit state subscriptions
- updates to those subscriptions may rerender only that region instead of rerunning the whole owning component
- the region still commits through the runtime's DOM adapter and commit machinery rather than hand-written direct DOM mutation at call sites
- the owning component remains the lifecycle and cleanup boundary for the region

## What May Update Narrowly

The first pass may bypass a full component rerender only for stable, explicit regions whose structure is locally owned and easy to replace safely.

Allowed targets:

- text content updates
- host attribute or property updates on stable existing DOM nodes
- small host-only child lists under a stable anchor
- atom or signal-backed computed output rendered inside an explicit subscribed region
- repeated high-frequency reads that do not require rerunning surrounding hooks

In practical terms, the first fine-grained path should prefer regions that behave like "replace this anchored leaf or small host subtree".

## What Still Forces Component Reconciliation

The first pass should fall back to normal component rerender and reconciliation for anything that changes ownership, hook order, or broader tree structure.

Required fallback cases:

- `UseState`, `UseReducer`, props, or context changes on a component
- hook dependency changes that affect `UseEffect`, `UseMemo`, `UseTransition`, or deferred scheduling behavior
- keyed list reconciliation above the subscribed region boundary
- conditional branches that add or remove component children outside the narrow region anchor
- portal structure changes
- router transitions, loader updates, guard results, metadata changes, and error-boundary state
- hydration mismatch recovery
- suspense, lazy, async-boundary, or error-boundary transitions
- any update where the runtime cannot prove the target region is still structurally safe to replace in isolation

If there is doubt, the runtime should rerender the normal component path.

## Mixed-Model Rule

The first model is intentionally mixed:

- hooks still own component lifecycle, effects, and structural rendering
- fine-grained subscriptions own only explicit narrow render regions
- a fine-grained update must not implicitly rerun surrounding hook logic
- a hook-driven rerender may recreate, move, or delete fine-grained regions as part of normal reconciliation

This keeps the existing mental model intact while allowing a smaller update target inside it.

## Mixed-Model Boundary

The runtime should treat hooks and fine-grained subscriptions as two different update authorities with a strict boundary between them.

### Hooks Still Own These Cases

The runtime must use normal component rerender and reconciliation when the update changes any value that the component's hook execution depends on.

That includes:

- `UseState`, `UseReducer`, and component-local derived values
- incoming props and prop-driven conditionals
- context reads through `UseContext`
- effect dependencies and cleanup scheduling through `UseEffect`
- memo and callback dependencies through `UseMemo`, `UseCallback`, and `UseEvent`
- transition, deferred-value, async-boundary, lazy, and error-boundary behavior
- any render branch that changes which hooks execute or how many hooks execute

Rule: if the component body would need to run again to keep hooks correct, the runtime must rerender the component.

### Fine-Grained Subscriptions May Own These Cases

Fine-grained subscriptions may bypass a full component rerender only when the update is fully contained inside an explicit subscribed region and the surrounding component logic does not need to re-execute.

That currently means:

- atom-backed or derived-value-backed reactive text output
- future selector-backed reads whose output only updates a stable anchored region
- future small host subtree replacements whose structure is owned entirely by the subscribed region

Rule: if the update can be expressed as "rerender this explicit subscribed region using already-declared dependencies" and no surrounding hooks need new values, the narrow path is allowed.

### Ownership Rules

The subscribed region and the owning component each have a distinct responsibility.

- the owning component creates the region, places it in the tree, and owns its lifetime
- the subscribed region owns only its narrow render output
- deleting or remounting the owning component automatically deletes or remounts the subscribed region
- a subscribed region must not outlive the fiber subtree that created it

This prevents fine-grained subscriptions from becoming detached mini-applications hidden inside a component tree.

### Read Rules

The first pass should keep read behavior explicit.

- hooks may read atoms, derived values, and selectors during normal component render
- subscribed regions may read only the explicit sources they declare for the narrow update path
- a subscribed region must not implicitly capture arbitrary hook state from the owning component body
- if a value comes from local hook state rather than an explicit reactive source, the region must fall back to normal component rerender semantics

Rule: explicit source reads are allowed in subscribed regions; implicit hook-state capture is not.

### Fallback Rules

The runtime should prefer correctness over narrowness.

It must fall back to full component rerender when:

- the region boundary moves or disappears because of a component rerender
- the runtime cannot prove the subscribed region still maps to a stable DOM anchor
- a selector or projection changes shape in a way that affects surrounding structure
- hook-driven state and fine-grained state both change in the same logical update and ordering would otherwise be ambiguous
- an error, hydration mismatch, or boundary reset invalidates the local narrow-update assumption

Rule: when mixed ownership becomes ambiguous, fall back to normal reconciliation.

### Current Practical Guidance

For the current text-only prototype, the safe mental model is simple:

- use hooks for structure, events, effects, async behavior, and local state
- use fine-grained subscriptions only for explicit hot-value display paths
- if a UI change affects layout, branching, or hook behavior, keep it on the normal component path

That gives the runtime a narrow but useful win without splitting the framework into two competing authoring models.

## Scheduling And Batching Semantics

The first fine-grained pass should not introduce a second independent scheduler. Narrow updates should reuse the current runtime scheduling model and commit boundary.

### Single Scheduler Rule

There is still one runtime work queue and one root commit path.

- fine-grained updates may mark a narrower fiber dirty
- hook-driven updates may mark a component path dirty
- both kinds of work still flow through the same scheduled root pass and the same commit phase

Rule: fine-grained updates change the dirty target, not the existence of a separate rendering pipeline.

### Default Urgent Update Semantics

Outside transitions, subscribed-region updates are urgent by default.

- an atom or derived value update for a fine-grained region should schedule work immediately through the existing root scheduler
- multiple urgent updates before the scheduled pass should batch into that same pass
- the committed DOM should reflect the latest source values seen by that pass

Rule: urgent fine-grained updates batch the same way urgent component updates already batch today.

### Transition Semantics

Updates issued inside `StartTransition(...)` should follow the existing deferred scheduling path.

- if a fine-grained source update happens inside a transition, it should be deferred through `ScheduleTransition(...)` before it dirties the subscribed region
- transition-pending state remains owned by the existing runtime transition atom and hook-facing transition APIs
- when the deferred callback runs, the fine-grained region may still take the narrow path if the mixed-model rules say it is safe

Rule: transition priority is decided before granularity is decided.

### Hook-Plus-Fine-Grained Collision Rule

When a hook-driven component update and a fine-grained subscription update affect the same owning subtree in the same scheduled window, the hook-driven rerender wins.

- the component rerender may recreate, move, or delete the subscribed region
- the runtime must not try to commit a stale narrow update ahead of that rerender
- after the rerender, any surviving subscribed regions may continue receiving narrow updates normally

Rule: when hook correctness and fine-grained narrowness conflict, hook correctness has priority.

### Coalescing Rule

The scheduler should coalesce repeated writes aggressively.

- repeated writes to the same fine-grained source before a pass should collapse to the latest source value visible when work executes
- repeated dirty marks for the same subscribed region should not enqueue extra root passes
- multiple subscribed regions dirtied before the pass should share one scheduled root pass when possible

Rule: batch by scheduled pass, not by individual subscription callback.

### Commit Rule

Fine-grained updates still commit inside the normal runtime commit phase.

- DOM writes from fine-grained regions should preserve the same ordering guarantees as ordinary commit work
- if a pass includes both broad and narrow updates, the runtime should produce one coherent committed tree state
- browser-visible DOM should not reflect a partially applied mix of old hook-driven structure and new fine-grained values after commit returns

Rule: one scheduled pass yields one coherent committed result.

### Effect Rule

Fine-grained updates do not create a second effect system.

- `UseEffect` cleanup and rerun behavior remains tied to component rerenders, not to narrow subscribed-region-only text updates
- a narrow update that does not rerender the owning component must not rerun surrounding effects implicitly
- if an effect must observe the change through normal hook dependencies, the update belongs on the component path instead

Rule: no implicit effect flushes from narrow text updates.

### Hydration Rule

Hydration keeps its existing safety gate.

- fine-grained subscribed updates discovered during hydration should queue until hydration completes
- after hydration finishes, queued updates may schedule normally and take the narrow path only if the hydrated region is still valid
- if hydration falls back for the subtree, normal reconciliation semantics take over

Rule: hydration safety outranks fine-grained eagerness.

### Error And Boundary Rule

Error recovery, suspense-like behavior, and boundary resets remain component-level concerns.

- if a fine-grained update targets a subtree whose owning component is inside an active recovery path, the runtime should prefer the normal component rerender path
- narrow updates must not bypass error-boundary or async-boundary correctness

Rule: boundary recovery may widen the update scope at any time.

### Current Practical Scheduling Model

For the current text-only prototype, the effective rule set is:

- urgent atom updates schedule one root pass
- transition-wrapped atom updates defer first, then schedule one root pass
- direct runtime atom writes and snapshot restores now also defer through that same transition lane instead of bypassing it
- repeated text-region updates before that pass collapse to the latest value
- if the owning component is also rerendering, the component rerender takes precedence
- no extra effect flush is introduced for text-only narrow updates

This keeps the feature aligned with the runtime's current scheduling architecture while still allowing smaller dirty regions.

## API Direction For The First Pass

The first pass should favor explicitness over magic.

Properties of the initial API surface:

- explicit subscription points, not hidden dependency capture across arbitrary render code
- explicit region boundaries, not whole-template automatic granularity
- compatibility with current `state` atoms and derived values
- a future path for signal-style primitives if they use the same narrow-region contract

The exact API names remain open, but the first shipped boundary should map to "subscribe here, rerender only this anchored region when these values change".

## Initial Primitive Set

The first in-core primitive set should be atom-backed and region-oriented rather than a full standalone signal runtime.

The recommended first set is:

- existing writable sources through `state.Atom[T]`
- existing shared computed sources through `state.Derived[T]`
- a new read-only subscription source contract that narrow regions can observe explicitly
- a new anchored subscribed-region primitive in `ui` that rerenders only that region when its observed sources change
- optional selector helpers for reading stable slices of larger atom values without forcing unrelated subscribed regions to update

In other words, the first pass should prefer "atom-backed subscribed views and selectors" over "ship signals, computed, and effect as a second reactive framework".

## What The First Pass Should Not Add Yet

The first pass should defer these primitives until the narrow-region path proves itself:

- a general-purpose standalone `Signal[T]` type unrelated to `state.Atom[T]`
- a separate effect system that competes with `ui.UseEffect`
- hidden dependency capture across arbitrary render logic
- automatic template-wide fine-grained compilation or expression lifting

The reason is scope control: the current runtime already has a lifecycle model, effect flushing model, scheduler, and hook contract. Replacing those all at once would expand risk faster than performance evidence.

## Recommended API Shape

The exact exported names remain open, but the first pass should be structurally close to this:

```go
type ReactiveSource[T any] interface {
	Get() T
	SubscribeRegion(region ui.ReactiveRegionHandle)
}

func Select[T any, U any](source state.Atom[T], project func(T) U, equal func(U, U) bool) ReactiveSource[U]

func ReactiveRegion(render func() ui.Node, sources ...any) ui.Node
```

The important part is the shape, not the spelling:

- state remains the source of truth
- narrow updates are opt-in through an explicit region boundary
- selectors can reduce churn for large object or slice atoms
- the region contract is compatible with future signal-like sources if they are added later

## Why This Set Fits The Current Runtime

This primitive set matches the current architecture better than a full signal package because:

- `state.Atom[T]` and `state.Derived[T]` already exist and already model explicit dependencies
- the runtime already knows how to subscribe fibers to atom-backed state
- the main missing piece is a smaller rerender target than the whole component fiber path
- hooks can continue owning effects, transitions, refs, and lifecycle without duplication

That makes the first prototype a narrower scheduling and commit problem, not a full authoring-model rewrite.

## Prototype Status

The first runtime prototype now exists as an explicit subscribed-region narrow-update path.

Current prototype behavior:

- `state.Atom[T]` and `state.Derived[T]` can render an explicit reactive text node through `Text(...)`
- `state.UseSelector(...)` can project a stable shared value from an atom or derived source for the same narrow text path
- `ui.ReactiveRegion(...)` can rerender an explicit subscribed child subtree from one or more explicit shared-state sources
- the runtime subscribes the reactive text fiber or region fiber directly to explicit source IDs
- atom updates can mark only that fine-grained fiber dirty instead of forcing the owning component fiber dirty
- clean ancestor fibers clone through to the dirty descendant so the owning component does not rerun when only the subscribed region changed
- derived selector outputs now suppress subscriber notification when the projected value is unchanged, so projection helpers can reduce update churn on larger source atoms

Current limitation:

- the prototype now supports reactive text, narrow host property updates, small anchored host subtree updates, and hook-scoped selector identities, but broader region ergonomics still remain follow-up work

## Initial Benchmark Signal

The first benchmark compares a stable keyed dashboard row of 16 panels where one hot value changes repeatedly.

Measured on Windows amd64 with `go test ./internal/runtime -run ^$ -bench FineGrainedKeyedDashboard -benchmem`:

- full component rerender path: `13548 ns/op`, `9538 B/op`, `141 allocs/op`
- reactive text path: `2073 ns/op`, `512 B/op`, `6 allocs/op`

Measured on Windows amd64 with `go test ./internal/runtime -run ^$ -bench 'FineGrained(Selector|KeyedDashboard)' -benchmem`:

- keyed dashboard component rerender path: `13380 ns/op`, `9570 B/op`, `141 allocs/op`
- keyed dashboard reactive text path: `2106 ns/op`, `544 B/op`, `6 allocs/op`
- selector-backed dashboard component rerender path: `12584 ns/op`, `9463 B/op`, `125 allocs/op`
- selector-backed dashboard reactive text path: `2386 ns/op`, `600 B/op`, `9 allocs/op`

Measured on Windows amd64 with `go test ./internal/runtime -run ^$ -bench FineGrainedAncestorRerender -benchmem`:

- ancestor rerender with 64 static leaves: `19489 ns/op`, `2901 B/op`, `27 allocs/op`
- ancestor rerender with 64 stable reactive regions: `19418 ns/op`, `2902 B/op`, `27 allocs/op`

That is the first proof that the narrow text update path avoids a large amount of keyed reconciliation and allocation work for dashboard-style updates. It is not yet proof for every workload, but it is enough to justify continuing with selector and small-region follow-up work.

It is also now a direct measurement of the ancestor-rerender overhead introduced by fine-grained region retention and clean-clone subscription transfer: for the current 64-region benchmark shape, the latest pass removed the practical gap on this machine by keeping unchanged subscriptions on their stale committed twin and redirecting later subscribed updates to the live fine-grained twin, rather than transferring ownership during every clean clone.

## Performance Guardrails

The fine-grained path only makes sense if it avoids adding more overhead than it removes.

Guardrails:

- no extra cost on trees that do not opt in
- explicit subscriptions before any automatic dependency graph discovery
- reuse existing scheduling and commit infrastructure where possible
- keep allocations below the cost of a normal component rerender for the target workloads
- prefer replacing a small anchored subtree over building a second generalized renderer

## Debuggability Requirements

The first pass is not shippable unless developers can tell why a narrow update happened.

The runtime and devtools should eventually expose:

- which region updated
- which atom, signal, or derived value triggered it
- whether the runtime used narrow replacement or fell back to full component reconciliation
- when a region subscription leaked or was cleaned up on unmount

Current devtools support now includes:

- per-node fine-grained flags in the inspected tree
- the reactive source ID currently attached to a fine-grained node
- the last recorded update origin for a node, including `fine-grained` when a subscribed region was the dirty target
- runtime counters for fine-grained fiber count, granular dirty marks, granular commits, and descendant host or text commits that happened inside a fine-grained region subtree

Current failure-mode coverage now includes:

- reactive text source swaps that unsubscribe the old source before subscribing the new one
- cyclic derived dependency updates that report a diagnostic instead of notifying fine-grained subscribers from a partial loop
- deleted subscribed subtrees that release atom subscriptions on unmount so later atom writes do not target detached fibers
- public transition-wrapped shared-state imports through `ui.StartTransition(...)` now prove that supported non-hook snapshot restore paths defer before notifying subscribed readers

Current narrowness coverage now also includes:

- sibling DOM subtrees stay pointer-stable while a fine-grained hot region updates in place
- unchanged projected values skip granular dirty marks and follow-up commits
- selector-backed hot regions are benchmarked against full component rerenders using larger object-shaped source state
- multiple subscribed hot regions can update independently without losing subscription ownership after a sibling region commits
- mixed hook-driven and fine-grained updates in the same scheduled window commit one coherent tree state with the hook rerender taking precedence
- transition-deferred fine-grained writes stay deferred until timeout instead of eagerly mutating the hot region
- post-hydration fine-grained updates reuse hydrated DOM nodes and still avoid rerendering the owning component when only the reactive text changes
- explicit subscribed regions can update host properties in place and can reconcile small anchored host subtrees without rerendering the owning component

## Non-Goals For The First Pass

The first shipped boundary does not attempt to provide:

- compile-time reactivity
- arbitrary DOM mutation outside the runtime commit path
- automatic fine-grained updates for every component expression
- replacement of hooks as the main authoring model
- perfect minimal updates for every list or layout case on day one

## Success Criteria

The first pass is successful if it demonstrates all of the following:

- fewer full component rerenders in targeted high-frequency examples
- lower work counts than current keyed reconciliation on at least some realistic workloads
- no regression to non-opted-in paths
- predictable fallback to normal reconciliation when the narrow path is unsafe
- inspectable update origin in diagnostics or devtools

## Rollout Checklist

Before expanding fine-grained usage in a product surface, verify all of the following:

- the region is anchored to a stable local subtree that can be replaced in isolation
- the hot value is shared through an atom, derived source, or selector rather than implicit hook capture
- the owner component does not need to rerun for correctness when that value changes
- transition, hydration, and boundary behavior still produce the expected fallback to normal reconciliation when required
- the narrow path is measured against the normal component path for the workload you actually care about
- devtools or diagnostics make it clear when the runtime took the fine-grained path