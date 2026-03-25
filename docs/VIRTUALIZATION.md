# Virtualization

This page defines the current direction for list and table virtualization in GoWebComponents.

Use it when deciding where virtualization belongs, what problem the first release should solve, and how it relates to the existing `ui` and runtime surface.

## Current Direction

The current ownership decision is:

- virtualization should start as a supported companion package
- it should not land as an unlabeled expansion of core `ui` on the first pass
- examples alone are not enough once the API is declared, but examples-first proving should still happen before the surface is treated as broadly stable

In practical terms:

- the project should provide one maintained virtualization answer
- that answer should be documented and benchmarked
- the answer should live outside the smallest core `ui` surface until its constraints, diagnostics, and workload coverage are proven

## Why This Boundary Fits The Project

Virtualization is important, but it is not a tiny primitive.

It brings along decisions about:

- row identity
- measurement
- overscan
- scroll ownership
- scroll restoration
- accessibility tradeoffs
- SSR and hydration behavior
- diagnostics and performance budgets

That is a substantial surface area. Starting it as a supported companion package keeps the main `ui` API smaller while still giving applications one first-party-supported answer.

This matches the current scope rules:

- core owns broadly reusable rendering and hook primitives
- companion packages may own heavier workflow surfaces that still need first-party guidance
- docs and examples should prove the shape before it quietly becomes part of the smallest default authoring surface

## What This Decision Means

This decision does mean:

- virtualization is a real supported direction, not an application-only hack
- the project should eventually ship one maintained API for windowed rendering
- the API should be documented, benchmarked, and tested like other supported surfaces

This decision does not mean:

- every app should import virtualization by default
- virtualization belongs in `ui` before the first release constraints are settled
- examples alone are enough forever once production apps depend on the feature

## Relationship To Core `ui`

Core `ui` still owns the underlying rendering model and the composition rules virtualization must obey.

That includes:

- reconciliation semantics
- keys and identity rules
- hooks and local state behavior
- hydration and SSR contracts
- diagnostics and devtools integration hooks

The companion virtualization layer should build on those rules rather than inventing a second rendering model.

## Expected Path

The intended path is:

1. define the workload scope and constraints clearly
2. define the API and row-identity model
3. ship the first maintained primitive in a supported companion package
4. prove the behavior with examples, benchmarks, diagnostics, and regression tests
5. only reconsider moving pieces into core `ui` if the primitive proves both small and broadly reusable enough to justify that change

## First Supported Workload Shapes

The first supported scope should be intentionally narrow:

- uniform-height vertical lists
- one-dimensional scrolling
- item collections where each row has one stable identity

That first pass should not claim support for:

- variable-height rows
- semantic tables as part of the same first primitive
- two-dimensional grids
- nested grouped virtualization
- masonry or waterfall layouts

Why this scope:

- uniform-height vertical lists are the simplest shape to reason about for window math, overscan, and scroll restoration
- they still solve a real product problem for feeds, logs, activity lists, inboxes, and result sets
- they avoid dragging measurement invalidation and table semantics into the first release

Practical first-release target:

- long feeds
- event logs
- result lists
- operator queues

Explicit defer list:

- variable-height behavior comes later, if the fixed-height contract proves correct and useful
- table virtualization needs its own follow-up decision because semantic headers, row groups, and cell relationships make it a different contract

## API Shape

The first public virtualization surface should be a component plus a row-render callback, backed by internal viewport state.

That means the first maintained shape should look conceptually like:

- one virtualized list component
- explicit item slice input
- explicit item key function
- explicit row height
- explicit row renderer

It should not start as:

- a totally low-level state bag that makes every consumer rebuild list math themselves
- a hook-only API that leaves scroll container wiring and placeholder layout ambiguous
- a one-size-fits-all table-and-grid abstraction on day one

Why this shape fits the first pass:

- a component makes ownership of the scroll window and placeholder sizing easier to understand
- the row-render callback keeps row markup application-owned
- the internal state can still be exposed later through lower-level primitives if the first implementation proves stable

Practical rule:

- first release: one narrow virtualized-list component surface
- later follow-up: extract lower-level primitives only when real examples prove they are needed

## Row Identity And Keys

Virtualized rows must use stable item identity supplied by the caller.

The first contract should require:

- an explicit item-key function or equivalent stable key input
- keys that remain stable across scrolling, filtering, and reordering
- row reuse based on item identity, not visible index alone

The first contract should reject or strongly warn against:

- using the visible row index as the durable identity
- keys that change when the window shifts
- keys that depend on transient sort position instead of the underlying item

Why this rule matters:

- virtualization reuses row shells aggressively
- unstable identity will corrupt row-local UI state, focus, and expanded or selected behavior
- the same key expectations that matter in normal reconciliation become stricter once rows unmount and remount outside the visible window

Practical rule:

- if the item does not already have a stable identity, the caller should derive one before it reaches the virtualized list

## Viewport And Scroll Ownership

The first virtualized-list primitive should own its own scroll container.

That means the first release should assume:

- the virtualization component renders the scrollable viewport element
- it reads scroll offset from that owned container
- it computes the visible range from that owned container
- it owns the spacer or placeholder geometry that makes the full list height coherent

The first release should not claim support for an arbitrary external scrolling parent.

Why this boundary fits the first pass:

- it keeps scroll metrics, visible-range math, and restoration behavior tied to one known element
- it avoids ambiguous ownership between app-shell scroll containers and the virtualized list
- it makes diagnostics and regression tests much easier to interpret

Follow-up direction:

- external scroll-parent support may be added later if real examples prove it is needed
- if that happens, it should be an explicit second mode with its own constraints rather than an implicit first-release promise

## Overscan Policy

The first release should use item-count-based overscan, not pixel-based overscan.

Recommended default:

- render the visible rows plus a small fixed number of extra rows before and after the window
- default overscan should be symmetrical
- callers may override the count explicitly, but the first contract should keep one simple item-count knob

Why this fits the first pass:

- uniform-height rows already make item-count overscan easy to reason about
- it is simpler to explain and test than pixel-budget overscan
- it gives the first release a clear jank-versus-work tradeoff without inventing a more dynamic prediction model

The first contract should not promise:

- velocity-aware overscan
- pixel-budget overscan
- separate before/after heuristics tuned by scroll direction

Rule of thumb:

- start with a conservative fixed item-count overscan
- expand only if benchmarks or real examples prove the simpler model is not sufficient

## Measurement Model

The first release should assume one fixed row height for the whole virtualized list.

That means the first public contract should require:

- one explicit row-height input for the list
- visible-range math derived from that fixed height
- spacer geometry derived from `item_count * row_height`

The first release should not claim support for:

- per-row measured heights
- estimated height plus later correction
- automatic DOM measurement during scrolling

Why this boundary fits the first pass:

- it keeps the window math simple enough to reason about and test
- it matches the intentionally narrow first workload of uniform-height vertical lists
- it avoids promising scroll correction behavior before anchor and invalidation rules are settled

Practical rule:

- if the rows are not actually uniform height, the caller should not use the first virtualization primitive
- variable-height support should be treated as a later second phase with its own measurement, invalidation, and correction contract

## Variable-Height Invalidation

Variable-height invalidation rules are intentionally out of scope for the first release because the first release does not support variable-height rows.

That means the first contract should make no promise about:

- remeasuring a row after content changes
- remeasuring after width changes or responsive wrapping
- remeasuring after font loading
- remeasuring after async image or media load
- correcting scroll offset after a measured-height mismatch

If a later second-phase variable-height mode is added, it must define all of those invalidation and correction rules explicitly before the mode is treated as supported.

Practical rule:

- first release: fixed-height lists only, so no row remeasurement lifecycle exists
- later variable-height release: must ship with explicit invalidation triggers, scroll-correction behavior, diagnostics, and regression coverage

## Scroll Restoration And Anchors

The first release should use a simple hybrid restoration model:

- preserve the owned scroll container's pixel offset for ordinary rerenders of the same list
- also record the first visible item's stable key as the restoration anchor

Restore behavior should be:

- if the same keyed item still exists after a rerender, route return, or data refresh, restore relative to that anchor item first
- if the anchor item no longer exists, fall back to the stored pixel offset
- always clamp the final scroll position to the current list bounds

Why this fits the first pass:

- pixel offset is the cheapest and most natural restoration path when list shape is unchanged
- stable item identity gives the list a durable anchor when inserts or deletes happen ahead of the visible window
- uniform-height rows make anchor-to-offset reconstruction straightforward without introducing correction churn

The first contract should not promise:

- restoration across reordered data that discards stable identity
- exact preservation through variable-height estimation mismatch
- cross-list restoration between unrelated datasets that merely share similar row markup

Practical rule:

- first-release callers should expect good restoration only when item keys stay stable and the list still represents the same underlying collection

## SSR And Hydration

The first release should use a fixed initial-render budget for SSR and hydration.

That means the first contract should:

- render only the first small window of rows on the server
- use the same initial row range during hydration
- delay switching to measured viewport-driven window math until after the browser has mounted the owned scroll container

Recommended first-pass shape:

- server render: first `N` rows plus the correct overall spacer geometry derived from `item_count * row_height`
- hydration: start with that same `N`-row window so markup stays stable
- post-mount: compute the real visible range from the owned scroll container and then adjust the rendered window

Why this fits the first pass:

- the server does not know the browser viewport size accurately enough to promise a true visible window
- matching the SSR and hydration window avoids accidental markup drift during boot
- fixed-height rows make spacer geometry predictable without needing browser measurement on the server

The first contract should not promise:

- server-side prediction of the real viewport height
- immediate viewport-perfect row counts before hydration completes
- variable-height placeholder estimation during SSR

Practical rule:

- first release should favor a predictable SSR-to-browser handoff over aggressive server-side viewport guessing

## Accessibility Expectations

The first release should treat virtualization as a rendering optimization, not as permission to weaken collection semantics.

That means the first contract should require:

- honest container semantics chosen by the caller for the underlying collection shape
- stable row identity so focus and selection can be restored meaningfully
- enough metadata for callers to expose total counts, labels, and active-item context when the UX needs it

The first release should assume:

- only rendered rows exist in the DOM and accessibility tree at a given moment
- offscreen rows are not represented as hidden duplicate accessibility nodes
- focus must be preserved by item identity when a focused row leaves and later re-enters the rendered window

Practical first-pass guidance:

- callers should keep keyboard focus on a stable interactive descendant or restore it by item key when the row remounts
- callers should expose collection labels and counts at the container level when the total size matters to screen-reader users
- virtualization should not invent a roving-tabindex model by default; row-level keyboard behavior remains application-owned

The first contract should not promise:

- screen-reader access to rows that are not currently rendered
- automatic accessibility metadata for every possible collection pattern
- built-in keyboard-navigation schemes that assume every virtualized list behaves like the same widget

## Table Behavior

Semantic tables are out of scope for the first virtualization primitive.

That means the first release should:

- support list-shaped collections only
- not claim that the same primitive honestly virtualizes `<table>` semantics
- require a dedicated follow-up table virtualization surface if semantic rows, headers, and cells need first-party support

Why this boundary fits the first pass:

- table headers, row groups, cell relationships, and keyboard expectations are a different contract from list rows
- pretending a list primitive automatically solves semantic table virtualization would overpromise accessibility and layout behavior
- keeping tables out of scope avoids collapsing feeds, logs, and data grids into one ambiguous API

Practical rule:

- first release: use the virtualized-list primitive for feeds, logs, queues, and result lists
- later table support: define a separate primitive or clearly separate mode with explicit table semantics

## Sticky And Grouped Interactions

Sticky headers, sticky columns, and grouped sections are out of scope for the first virtualization primitive.

That means the first release should not promise:

- built-in sticky header behavior inside the virtualized viewport
- sticky columns or frozen panes
- grouped-section virtualization with pinned group headers

Why this boundary fits the first pass:

- sticky and grouped layouts add another layer of range math, z-order rules, and scroll-coupled behavior
- sticky columns mainly belong to a table or grid contract, which is already deferred
- grouped sections need their own identity, restoration, and accessibility rules rather than being smuggled into the first list primitive

Practical rule:

- first release: one plain scrolling virtualized list
- later follow-up: add sticky or grouped behavior only with explicit diagnostics and examples that prove the math stays understandable

## Row State, Forms, And Focus

Virtualization should assume that offscreen rows may unmount and later remount.

That means the first integration guidance should be:

- treat row-local hook state as disposable unless it is safe to lose during scrolling
- externalize meaningful user work by stable item key when that state must survive row unmounts
- restore focus by stable item key when the focused row remounts instead of assuming the DOM node stayed alive

Practical examples:

- expanded or collapsed row state that matters across scrolling should live in parent or shared state keyed by item id
- form draft values that the user would expect to keep should not live only inside a row component that may unmount
- ephemeral hover state or purely decorative animation state may stay row-local

Why this rule matters:

- virtualization reuses and discards row shells as the window moves
- relying on row-local lifetime for important state will produce surprising data loss
- stable item identity already exists in the contract, so it should also be the anchor for persisted row state and focus restoration

Practical rule:

- if losing the state would surprise the user, store it outside the row by item key

## Fine-Grained Reactivity Integration

Virtualization and fine-grained reactivity solve different problems and should not compete for ownership of the same update.

The first integration rule should be:

- virtualization owns viewport range, scroll-driven row mount or unmount, and spacer geometry
- `state.Select(...)` and `ui.ReactiveRegion(...)` may optimize updates inside rows that are already rendered
- fine-grained subscriptions should not be treated as a replacement for virtualization when the real problem is too many mounted rows

Why this boundary fits:

- virtualization reduces the number of mounted rows
- fine-grained reactivity reduces the amount of rerender work inside the mounted set
- mixing those responsibilities makes performance behavior harder to reason about and harder to debug

Practical rule:

- use virtualization first when the bottleneck is collection size
- use selectors or reactive regions inside visible rows only when a smaller hotspot remains after windowing
- do not expect fine-grained row internals to preserve row-local state across virtualization unmounts

## Current Shipped Primitive

The repo now includes one low-level viewport helper in the `virtualization` companion package:

- `ComputeViewportState(...)` for fixed-height visible and rendered range math
- `ObserveOwnedViewport(...)` for owned-element scroll and resize observation in browser builds

- `List(...)` for the first owned-scroll, fixed-height virtualized list component

The current shipped surface is intentionally still narrow:

- one fixed-height vertical list
- one owned scroll container
- one typed row-render callback plus item-key function
- one typed viewport-diagnostics snapshot for visible and rendered range inspection
- fixed-height measurement and scroll-correction diagnostics that explicitly report zero churn today
- row mount and unmount churn counters for scroll-window inspection
- no variable-height rows, semantic tables, sticky regions, or grouped sections yet

## Performance Budgets

The first fixed-height release should have simple budgets that match the diagnostics already exposed today.

Recommended first-pass ceilings:

- rendered row count: stay comfortably below `visible_count + (overscan * 2)` and treat double-rendering beyond that as a bug
- overscan: default should stay small and symmetrical; callers should justify values much larger than the default because row work scales directly with overscan
- measurement churn: `MeasurementCount == 0` and `InvalidationCount == 0` for the fixed-height primitive
- scroll correction: `ScrollCorrectionCount == 0` for the fixed-height primitive
- row churn: steady scrolling should not cause unbounded row mount growth relative to rows entering the window; `RowMountCount` and `RowUnmountCount` should track scroll-window movement rather than entire-list rerenders

Practical review rule:

- if diagnostics show the rendered row count drifting above the expected visible-plus-overscan window, treat that as a correctness bug
- if a fixed-height list reports non-zero measurement or scroll-correction churn, treat that as a regression
- if mount and unmount churn spikes far beyond the number of rows actually entering and leaving the window, inspect key stability and unnecessary full-list rerenders first

These are intentionally narrow budgets for the current fixed-height surface.

Later follow-up work may add:

- dropped-frame or RAF-based timing budgets
- variable-height measurement budgets
- table-specific row and cell budgets

## Current Rule

Rule of thumb:

- if the problem is basic rendering or hooks composition, it belongs in core `ui`
- if the problem is full windowed-list ownership with measurement, overscan, scrolling, and restoration policy, it should start in the supported virtualization companion layer
