# Router Async Guards And Auth Design

This page defines the planned public model for async navigation guards and auth-aware routing.

Current shipped status:

- synchronous `BeforeEnter` and `BeforeLeave` guards exist today
- route loaders, redirects, and revalidation exist today
- `router.ReturnToParam`, `router.PreserveReturnTo(...)`, and `router.ReadReturnTo(...)` ship today
- protected-route examples now exist for redirect, authorizing-shell, and manual unauthorized-fallback flows
- async guards and auth-aware route primitives are not implemented yet

This document is the contract the next implementation pass should follow.

## Goals

The router should support:

- async route entry and leave checks that can redirect, block, or fail
- cancellable guard work that cannot commit stale results
- explicit pending UI while a guard is resolving
- a lightweight auth signal for route gating without owning a full identity stack
- route-group policy inheritance so protected sections do not duplicate the same checks

The router should not become an identity-provider framework. Authentication remains application-owned; the router only consumes a compact auth snapshot and route policy result.

## Planned Async Guard Surface

Keep the existing synchronous guards and add parallel async hooks.

Planned route options:

```go
type Options struct {
    BeforeEnter      GuardFunc
    BeforeLeave      LeaveGuardFunc
    BeforeEnterAsync AsyncGuardFunc
    BeforeLeaveAsync AsyncLeaveGuardFunc
    GuardPending     interface{}
    Unauthorized     interface{}
    Authorizing      interface{}
}
```

Planned function shapes:

```go
type AsyncGuardFunc func(context.Context, RouteContext) GuardDecision
type AsyncLeaveGuardFunc func(context.Context, RouteContext, RouteContext) GuardDecision

type GuardDecision struct {
    Redirect  string
    Blocked   bool
    Reason    string
    Retryable bool
    Denied    bool
}
```

Interpretation rules:

- zero-value decision means allow navigation
- `Redirect` wins over other fields
- `Denied` means the route is known to be forbidden or unauthorized and should use route-level unauthorized UI when provided
- `Blocked` means stay on the current route
- `Retryable` means the failure came from a transient condition and the router may expose retry UI

Synchronous and async guards should compose in this order:

1. `BeforeLeave`
2. `BeforeLeaveAsync`
3. `BeforeEnter`
4. `BeforeEnterAsync`

That preserves today's fast-path synchronous behavior and keeps async work out of the path when a synchronous guard already blocks or redirects.

## Cancellation And Race Semantics

Every navigation attempt should carry its own monotonically increasing attempt id and cancellable context.

Rules:

- when a new navigation starts, all in-flight async guard work from older attempts is cancelled
- late completions from cancelled attempts must be ignored
- browser back and forward navigation creates a fresh attempt id just like click-driven navigation
- a redirect emitted by an async guard starts a new navigation attempt, not a mutation of the old one
- route loaders must not start for a target route until the enter guards for that same attempt resolve to allow
- pending leave guards keep the current route mounted until the attempt completes or is cancelled

This matches the loader model already used by the router: current work is authoritative, stale work is discarded.

## Pending Navigation UX

Async guards need explicit pending UI rather than silent dead time.

Planned router state:

```go
type GuardNavigationState struct {
    Pending     bool
    Leaving     bool
    Entering    bool
    CurrentPath string
    TargetPath  string
    Reason      string
    Retryable   bool
}
```

Planned hook:

```go
func UseGuardNavigation() GuardNavigationState
```

Intended behavior:

- while a guard is pending, repeated navigations to the same target should be coalesced
- routes may disable repeated buttons or links while `Pending` is true
- `GuardPending` route content is preferred when the pending state belongs to that route boundary
- layout routes may render their own pending shell for child-route checks

This state is for UX only. It must not be treated as authorization proof.

## Lightweight Auth Context Model

The router should consume a minimal auth snapshot, not a full auth SDK.

Planned shape:

```go
type AuthStatus string

const (
    AuthUnknown         AuthStatus = "unknown"
    AuthAuthorizing     AuthStatus = "authorizing"
    AuthAuthenticated   AuthStatus = "authenticated"
    AuthUnauthenticated AuthStatus = "unauthenticated"
)

type AuthState struct {
    Status  AuthStatus
    Subject string
    Claims  map[string]any
    Version int64
}
```

Planned access:

- `router.AuthProvider(state router.AuthState, child ui.Node)`
- `router.UseAuth() router.AuthState`

Purpose:

- enough information for route gating and conditional UI
- enough versioning to invalidate stale guard results when session state changes
- no assumptions about token refresh protocol, storage location, or identity vendor

## Router-Level Unauthorized And Authorizing UI

This item is still implementation work, but the intended model is:

- `Unauthorized` renders when a guard or auth policy says the route is denied
- `Authorizing` renders while auth status is `unknown` or `authorizing` and a route policy depends on it
- these route-level UIs should be available on both leaf routes and layout routes
- `Unauthorized` should support both unauthenticated and forbidden outcomes through route-provided content or a framework fallback

Planned route option shape:

```go
type Options struct {
    Unauthorized interface{}
    Authorizing  interface{}
}
```

Until implemented, applications should continue to redirect or render their own fallback content manually.

## Current Protected-Route Pattern

Today, applications should compose the shipped pieces this way:

- use a synchronous `BeforeEnter` guard when auth is already known and the route should redirect immediately
- preserve the requested internal target with `PreserveReturnTo(...)`
- recover it on the login or re-auth route with `ReadReturnTo(...)`
- render manual authorizing or unauthorized content inside the route tree when auth is still unresolved or when a subsection is forbidden
- keep server authorization authoritative for SSR, APIs, and mutations

See:

- `examples/65-router-guards` for the basic redirect and leave-guard flow
- `examples/92-protected-routes` for return-to recovery, manual authorizing UI, manual unauthorized fallback content, and route-loader/shared-cache reuse

## Route-Group Auth Inheritance

Layout routes should define shared auth policy for their descendants.

Rules:

- layout route policy applies to all nested child routes by default
- child routes may strengthen the requirement but should not silently weaken an inherited requirement
- public child routes inside protected layout trees should be modeled as separate route branches instead of opt-out exceptions
- when multiple layouts contribute policy, the effective route requirement is the most restrictive merged policy

This keeps parent shells authoritative and avoids leaf-route drift.

## Route Policy Hooks

Above raw boolean guards, the router should expose reusable policy helpers.

Planned helpers:

- `router.RequireAuthenticated()`
- `router.RequireClaim(name string, value any)`
- `router.RequirePolicy(name string, fn func(AuthState, RouteContext) GuardDecision)`

These helpers should compile down to the same guard decision model so apps can share policy logic without adopting a framework-owned auth backend.

## Return-To And Post-Auth Navigation Helpers

Post-auth redirects should preserve the route the user originally requested.

Implemented helpers:

```go
const ReturnToParam = "return_to"
func PreserveReturnTo(path string, query url.Values) string
func ReadReturnTo(query url.Values, fallback string) string
```

Rules:

- preserve the normalized target path plus the original query string
- keep the value URL-safe and bounded in size
- reject absolute external URLs and fall back to an internal default route
- use replacement navigation after successful auth so the transient login route does not remain in history
- allow applications to attach one small intent token when they need to resume a specific action after auth, but keep that token application-owned

## Auth Refresh And Invalidation

Auth-aware routing must react to session changes after initial mount.

Rules:

- when `AuthState.Version` changes, in-flight auth-dependent guards become stale
- if the active route no longer satisfies its policy after refresh or logout, the router should re-evaluate the current route policy immediately
- background refresh that moves auth from `unknown` to `authenticated` may resolve a pending route in place without forcing a manual retry
- logout or expiry should cancel protected-route loaders that have not completed yet
- auth refresh should not automatically rerun unrelated public-route loaders

## Server-Hydrated Auth Hint Transfer

SSR should be able to hand the client a minimal auth hint for first-paint consistency.

Planned model:

- the server emits a compact `AuthState` bootstrap payload alongside route data
- the first client-side guard evaluation reads that bootstrap state before any fresh auth API request completes
- the bootstrap state is a hint for initial client consistency, not a replacement for server authorization
- once fresh auth state arrives, `Version` changes trigger normal invalidation and re-evaluation

The bootstrap payload should stay intentionally small:

- status
- subject or anonymous marker
- small claim subset only when routing actually needs it
- version or revision marker

## Loader Ordering With Auth Gates

The intended ordering is:

1. synchronous leave guards
2. async leave guards
3. synchronous enter guards
4. async enter guards
5. route loaders
6. final route render

Protected-route loaders should not start before an auth gate allows the navigation. Public shell loaders may still run earlier only when explicitly attached to an already-allowed parent layout.

## Security Boundary

Client-side auth-aware routing is a UX and consistency feature.

It is not a substitute for server-side authorization.

Applications still need server-side checks for:

- data access
- mutations
- SSR route responses
- file or API endpoints

The router can improve navigation behavior and pending states, but the server remains authoritative for real access control.
