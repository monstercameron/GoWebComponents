# Production Correctness

This page defines the current minimum production-correctness bar for the core runtime.

The goal is not "some examples seem to work." The goal is deterministic, test-backed behavior for the flows that make real applications brittle when the runtime is underspecified.

## At A Glance

- Production correctness in this repo starts with deterministic runtime behavior, not with broad marketing claims.
- The current bar is centered on render determinism, state convergence, hydration fallback, portal ownership, cleanup discipline, and scoped boundary recovery.
- Recovery must stay observable. When the runtime contains a failure, it should also surface structured diagnostics, logs, or HTTP error reports that make the failure actionable.
- Use [actionable-errors-and-diagnostics.md](actionable-errors-and-diagnostics.md) for the diagnostic contract, [hydration.md](hydration.md) for DOM reuse and fallback rules, and [observability.md](observability.md) for the currently shipped inspection surfaces.

## Current Shipped Slice

What is already real in the repo today:

- composed runtime correctness coverage in `internal/runtime/production_correctness_test.go`
- hydration reuse and subtree fallback coverage in `internal/runtime/hydration_test.go`
- scoped recovery through runtime-owned error boundaries with component-stack-aware diagnostics
- structured runtime diagnostics and log classifications surfaced through runtime, router, fetch, devtools, and the public `diagnostics` package
- server-side HTTP error helpers through `diagnostics.Build(...)` and `diagnostics.WriteHTTPError(...)`

What this page still does not claim:

- that every higher-level package already has the same production bar as the runtime core
- that every transport, worker, or offline path has equivalent race coverage
- that build-tag, browser, and deployment permutations are fully closed out

## Minimum Bar

The runtime should not claim serious production readiness until these behaviors are stable and specified:

- render and rerender flows commit deterministic DOM updates for the same virtual tree and state history
- state, atom, and transition updates converge to the latest intended result without leaving stale pending work behind
- portals preserve ownership and cleanup rules when targets change or subtrees unmount
- error boundaries recover scoped failures without corrupting surrounding UI state and surface diagnostics with path context
- hydration either reuses matching DOM safely or falls back per subtree with explicit diagnostics
- mount or unmount churn does not leak subscribers, deferred work, or stale cleanup state across long-lived sessions

## Operational Safety Surface

Correctness is not only about converging state. It is also about how failures are contained and reported.

The current operational surface includes:

- error boundaries that recover the failing subtree while preserving surrounding shell state when recovery is allowed
- hydration diagnostics that distinguish reusable markup from mismatch fallback
- router and fetch logs or diagnostics that classify informational, recovered, correctness, and performance-significant events
- embeddable devtools inspection for current diagnostics, tree shape, and recent runtime behavior
- HTTP error report helpers for native server paths that need structured user-facing or operator-facing responses

That is the current standard: if a failure is recoverable, the runtime should degrade in place and record why. If a failure is not recoverable, the emitted report still needs enough structure for support, debugging, and testing.

## Current Coverage

The current core-runtime correctness pass now includes:

- composed-flow coverage in `internal/runtime/production_correctness_test.go` for hydration, shared atoms, portals, and boundary recovery in one scenario
- churn coverage in the same file for repeated mount or unmount cycles with atom subscription release and cleanup accounting
- overlapping urgent plus transition update coverage in the same file so repeated bursts settle to one coherent final state
- boundary recovery diagnostics with component-stack context in `internal/runtime/error_boundary_test.go`

Existing adjacent coverage already complements this bar:

- `internal/runtime/hydration_test.go`
- `internal/runtime/portal_test.go`
- `internal/runtime/transition_test.go`
- `internal/runtime/update_regression_test.go`

## Current Reference Scenario

The clearest current proof point is the composed production-correctness scenario in `internal/runtime/production_correctness_test.go`.

That test exercises one realistic sequence instead of isolated primitives:

- hydrate a server-rendered shell
- open a portal-backed overlay
- update shared atom state across shell and overlay
- trigger a component failure inside an error boundary
- verify fallback recovery without corrupting surrounding UI
- reset the boundary and confirm the tree settles with no leftover scheduled work

That is the right shape for this page: correctness claims should come from converged end-to-end runtime sequences, not only from tiny isolated unit assertions.

## Deliberate Non-Claims

This bar is currently about the runtime core.

It does not yet claim that every higher-level package has equivalent production coverage for:

- router loaders and auth races
- offline replay
- worker orchestration
- multi-surface coordination
- production build-tag parity

Those remain separate backlog items and should not be implied by this document alone.

## Review Checklist

- does the production claim point to a deterministic runtime behavior, not just anecdotally working examples
- does every recovery claim also mention the diagnostic or reporting path that makes the recovery observable
- are runtime-core guarantees clearly separated from router, offline, worker, or deployment guarantees that remain incomplete
- do failure examples preserve surrounding UI state and settle scheduled work instead of only hiding the symptom
