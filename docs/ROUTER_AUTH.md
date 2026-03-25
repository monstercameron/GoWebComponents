# Router Async Guards And Auth Design

This page describes the current router guard surface, what is shipped today for protected-route flows, and which auth-aware pieces are still application-owned.

## Current Status

Shipped today:

- synchronous guards through `BeforeEnter` and `BeforeLeave`
- async guards through `BeforeEnterAsync` and `BeforeLeaveAsync`
- route-level guard fallback rendering through `GuardPending`, `Unauthorized`, and `Authorizing`
- route redirects, route loaders, route loading UI, route error UI, and revalidation
- bounded return-to helpers through `router.ReturnToParam`, `router.PreserveReturnTo(...)`, and `router.ReadReturnTo(...)`
- protected-route examples for redirect recovery, manual authorizing UI, manual unauthorized UI, and shared cache reuse
- async guard race and cancellation coverage in router browser tests

Not shipped today:

- a router-owned auth store such as `AuthProvider(...)` or `UseAuth()`
- router policy sugar such as `RequireAuthenticated()` or `RequireClaim(...)`
- a public `UseGuardNavigation()` hook

The router already supports async guard execution. The missing layer is framework-owned auth state and framework-owned auth policy helpers.

## Shipped Guard Surface

The current route options already include both sync and async guard entry points:

```go
type Options struct {
    BeforeEnter      GuardFunc
    BeforeLeave      LeaveGuardFunc
    BeforeEnterAsync AsyncGuardFunc
    BeforeLeaveAsync AsyncLeaveGuardFunc
    GuardPending     interface{}
    Unauthorized     interface{}
    Authorizing      interface{}
    Loader           LoaderFunc
    Loading          interface{}
    Error            interface{}
}
```

The current function types are:

```go
type GuardFunc func(RouteContext) GuardResult
type LeaveGuardFunc func(current RouteContext, next RouteContext) GuardResult

type AsyncGuardFunc func(context.Context, RouteContext) GuardDecision
type AsyncLeaveGuardFunc func(context.Context, RouteContext, RouteContext) GuardDecision

type GuardResult struct {
    Redirect string
    Blocked  bool
    Reason   string
}

type GuardDecision struct {
    Redirect  string
    Blocked   bool
    Reason    string
    Retryable bool
    Denied    bool
}
```

Helper functions exist for the synchronous path:

```go
func AllowNavigation() GuardResult
func BlockNavigation(reason string) GuardResult
func RedirectNavigation(path string) GuardResult
```

Typical synchronous protected-route flow:

```go
r.Register("/editor", editorPage, router.Options{
    BeforeEnter: func(ctx router.RouteContext) router.GuardResult {
        if !signedIn {
            return router.RedirectNavigation("/login")
        }
        return router.AllowNavigation()
    },
    BeforeLeave: func(current router.RouteContext, next router.RouteContext) router.GuardResult {
        if dirty {
            return router.BlockNavigation("Unsaved changes are blocking navigation")
        }
        return router.AllowNavigation()
    },
})
```

See `examples/65-router-guards` for the complete redirect-and-dirty-form flow.

## Async Guard Semantics

Async guards are no longer just design notes. The router runs them today with cancellable attempt tracking.

The evaluation order is:

1. current-route `BeforeLeave`
2. current-route `BeforeLeaveAsync`
3. target-route `BeforeEnter`
4. target-route `BeforeEnterAsync`

That preserves the synchronous fast path and only starts async work when a synchronous guard has not already redirected or blocked.

Current semantics from the router implementation and browser tests:

- each navigation attempt gets its own cancellable `context.Context`
- starting a new navigation cancels the previous async guard attempt
- late completions from stale attempts are ignored
- browser back and forward navigation create fresh attempts too
- redirects returned by guards restart navigation with a new normalized target
- route loaders do not start until target-route guards allow the navigation
- a blocked or denied async guard keeps the current route active

The current async tests cover:

- dropping stale results during double navigation
- creating fresh attempts for back and forward navigation
- delaying route loaders until async guards resolve
- cleaning up pending async guard work when a replacement route wins

## Meaning Of `GuardDecision`

Use the zero value to allow navigation.

Interpret the fields this way:

- `Redirect`: navigate to another internal target instead of completing the requested route
- `Blocked`: stop the navigation and keep the current route
- `Reason`: optional explanation for diagnostics or app-owned UI
- `Denied`: mark the decision as an authorization-style denial instead of a generic block
- `Retryable`: identify a transient failure so application UI can offer retry behavior

Important current limitation:

- the core router uses `Blocked` and `Denied` to stop navigation, but it does not yet render specialized framework-owned unauthorized or retry UI from those flags

That means `Denied` and `Retryable` are useful for application policy code and future framework evolution, but applications still own the visual treatment today.

## Pending, Unauthorized, And Authorizing UI

The route option fields already exist:

```go
type Options struct {
    GuardPending interface{}
    Unauthorized interface{}
    Authorizing  interface{}
}
```

What is true today:

- the option fields are part of `router.Options`
- async guards already run and can delay route commitment
- the router can render `GuardPending`, `Unauthorized`, and `Authorizing` fallback content before the route component mounts

What is not true today:

- the router does not expose a public guard-state hook
- the router does not ship auth policy helpers or a framework-owned auth state store

Use route-level fallback content for whole-route authorizing or unauthorized states. Keep manual in-route auth UI when only part of a page is gated by claims, feature flags, or subsection-specific checks.

## Current Protected-Route Pattern

The current recommended protected-route composition is:

1. use a synchronous `BeforeEnter` redirect when the session is already known to be unauthenticated
2. preserve the requested internal target with `PreserveReturnTo(...)`
3. recover the target on the login route with `ReadReturnTo(...)`
4. use `nav.Replace(...)` after sign-in so the transient login route does not linger in history
5. render manual authorizing or unauthorized content inside the protected route when session resolution or subsection claims remain application-owned
6. keep server authorization authoritative for SSR responses, APIs, and mutations

The current example for that flow is `examples/92-protected-routes`.

Its guard uses the bounded return-to helper like this:

```go
func protectedGuard(ctx router.RouteContext) router.GuardResult {
    if liveSession.Status != sessionUnauthenticated {
        return router.AllowNavigation()
    }
    values := url.Values{}
    values.Set(router.ReturnToParam, router.PreserveReturnTo(ctx.Path, ctx.Query.Values()))
    return router.RedirectNavigation("/login?" + values.Encode())
}
```

Its login route resolves the saved internal target like this:

```go
query := router.UseQuery()
returnTo := router.ReadReturnTo(query.Values(), "/workspace")
nav.Replace(returnTo)
```

That example also shows a useful production pattern:

- let the route loader seed shared data with `fetch.LoadCached(...)`
- let the component read the same cache entry with `fetch.UseCachedResource(...)`
- keep manual authorizing and unauthorized states inside the route component until the router gains framework-managed auth UI

## Return-To Rules

`PreserveReturnTo(...)` and `ReadReturnTo(...)` are the safe primitives for post-auth navigation.

They currently enforce these rules:

- internal paths are normalized before use
- original query values are preserved
- absolute or host-qualified URLs are rejected
- oversized values fall back to the supplied internal default
- network-path forms such as `//evil.example` are rejected

Use them for login redirects, re-auth flows, and resumable protected navigation.

## Auth Ownership Boundary

The router is intentionally not an identity-provider framework.

Keep these responsibilities application-owned today:

- fetching session state
- claim evaluation and authorization policy rules
- deciding when auth is unknown, resolving, authenticated, or unauthenticated
- rendering manual authorizing and unauthorized UI
- revalidating loaders after a session transition when protected data must refresh

Keep these responsibilities server-owned in all cases:

- SSR authorization
- API authorization
- mutation authorization
- redirect enforcement for direct protected URL requests that hit the server first

The client router improves navigation behavior and recovery UX. It does not make the browser authoritative.

## What May Come Next

These additions would fit the current API direction, but they are not available yet:

- a public guard-state hook for pending guard UX
- application-neutral auth context helpers
- reusable policy helpers layered on top of `GuardDecision`

If those pieces land later, they should build on the shipped `GuardDecision`, async attempt cancellation, and return-to safety model instead of replacing them.
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

## Silent Refresh And Active Route Consistency

Silent token or session refresh should not look like a brand-new navigation when
the active authenticated principal and route entitlement remain the same.

Recommended rules:

- do not re-fire active route guards purely because a background refresh
  renewed the current session for the same user and tenant boundary
- do not cold-reload already allowed protected-route loaders unless the refresh
  changed data-shaping auth inputs such as principal, tenant, or claims that the
  route actually depends on
- update auth-derived atoms and session hints in one coordinated transition so
  the route tree does not briefly flash unauthorized UI and then recover a
  moment later
- while refresh is still unresolved, prefer an explicit pending or authorizing
  state over eagerly downgrading the current protected route
- if refresh proves the principal or tenant changed, treat that as a real auth
  boundary change: cancel protected loaders, clear stale route-owned data, and
  re-run the guard path intentionally

The practical distinction is "same identity, fresher credential" versus "new
identity or narrower access." Only the second case should behave like a true
route auth transition.

## Security Boundary

Client-side auth-aware routing is a UX and consistency feature.

It is not a substitute for server-side authorization.

Applications still need server-side checks for:

- data access
- mutations
- SSR route responses
- file or API endpoints

The router can improve navigation behavior and pending states, but the server remains authoritative for real access control.
