# Production Correctness

This page defines the current minimum production-correctness bar for the core runtime.

The goal is not "some examples seem to work." The goal is deterministic, test-backed behavior for the flows that make real applications brittle when the runtime is underspecified.

## Minimum Bar

The runtime should not claim serious production readiness until these behaviors are stable and specified:

- render and rerender flows commit deterministic DOM updates for the same virtual tree and state history
- state, atom, and transition updates converge to the latest intended result without leaving stale pending work behind
- portals preserve ownership and cleanup rules when targets change or subtrees unmount
- error boundaries recover scoped failures without corrupting surrounding UI state and surface diagnostics with path context
- hydration either reuses matching DOM safely or falls back per subtree with explicit diagnostics
- mount or unmount churn does not leak subscribers, deferred work, or stale cleanup state across long-lived sessions

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

## Deliberate Non-Claims

This bar is currently about the runtime core.

It does not yet claim that every higher-level package has equivalent production coverage for:

- router loaders and auth races
- offline replay
- worker orchestration
- multi-surface coordination
- production build-tag parity

Those remain separate backlog items and should not be implied by this document alone.
