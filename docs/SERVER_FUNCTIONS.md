# Server Functions

This document defines the first-class server function model beyond forms in GoWebComponents.

Use it when the application needs a typed server-owned call for mutations or queries that are not best expressed as a browser-native HTML form post.

## At A Glance

- a server function is an application-owned typed call into same-origin server code
- it is broader than a form action because it may represent non-form mutations or explicit query-style commands
- it is narrower than a transport framework because it does not turn core GWC into an RPC stack
- it should layer on existing public surfaces such as `fetch`, `router`, `ui`, SSR bootstrap, and same-origin auth handling

## Decision

GoWebComponents should support the idea of server functions as a documented application pattern, but not as a magic core runtime primitive today.

That means:

- server functions are first-class at the architecture level
- they are still declared by ordinary application code and ordinary server routes
- they should use typed request and result models
- they should not bypass existing fetch, auth, CSRF, SSR, or routing boundaries

## What Counts As A Server Function

Treat a call as a server function when all of these are true:

- the browser is intentionally invoking server-owned logic, not just loading a document
- the call has one typed request and one typed result contract
- the server remains the authority for validation, auth, side effects, and canonical success
- the function should compose with the existing route, cache, and SSR model rather than replace it

Typical cases:

- save workspace preferences
- run an authenticated mutation that is not naturally expressed as a form post
- trigger a server-owned workflow step
- request a typed query that depends on session or server-only context and is not a route-loader first-paint concern

## Declaration Model

The current recommended declaration model is explicit and application-owned:

1. define request and result Go structs in application code
2. expose a same-origin HTTP handler or endpoint that owns the server function
3. call it through `fetch.Fetch(...)`, `fetch.UseResource[T](...)`, or an application-specific helper
4. map the typed result back into route state, form state, or shared cache intentionally

This keeps declaration visible in ordinary Go code instead of hiding it behind compiler-owned or framework-owned magic.

## Relationship To Server Actions

Server actions are the form-oriented subset of server functions.

Use a server action when:

- the browser-native HTML form contract matters
- progressive enhancement matters
- redirect-after-submit and form validation are central

Use a broader server function when:

- the interaction is not form-shaped
- the caller is a command button, background retry, settings save, or typed query helper
- no progressive HTML form path is required

## Relationship To Fetch Helpers

Server functions do not replace `fetch`.

Instead:

- `fetch` remains the transport and lifecycle tool
- a server function is the application-level contract sitting above that transport

If the call does not need a named server-owned contract, plain `fetch.Fetch(...)` is enough.

## Relationship To RPC

Server functions also do not imply a framework-owned RPC stack.

Differences from transport-specific RPC clients:

- server functions may be implemented over ordinary same-origin HTTP routes
- they do not require WebSocket tunnels, protobuf, or generated stubs
- they fit the current default framework posture better than transport-specific RPC for many apps

If an application needs high-frequency streaming, generated clients, or transport-managed multiplexing, that belongs in the companion-package RPC direction documented in `docs/RPC_TRANSPORT.md`, not in core server-function semantics.

## Composition With Loaders, Revalidation, Auth, And SSR

Server functions should compose with existing framework ownership rules instead of creating a second hidden data plane.

### Route Loaders And Revalidation

Use this rule:

- route loaders own route-entry reads and SSR first paint
- server functions own explicit follow-up commands or typed non-form queries
- successful server functions should revalidate only the routes or shared cache entries that actually became stale

Recommended follow-up after a successful server function:

1. keep the server function authoritative for the write result
2. invalidate or reload the affected `fetch` cache keys
3. call `router.UseRevalidator()` only when the current route's loader-owned data is now stale

Do not hide loader invalidation inside a magical transport layer.

### Auth And CSRF

Server functions must obey the same auth and CSRF boundary as the rest of the app:

- same-origin session and permission checks stay server-owned
- form-shaped server functions should still use the server-action CSRF rules
- non-form server functions should use explicit same-origin protection and any app-required CSRF or replay defense instead of assuming that fetch alone is a boundary
- auth failure should resolve into one typed result or HTTP status the caller can branch on intentionally

### SSR And Hydration

Server functions are not an SSR replacement.

Use this split:

- SSR and route loaders own first paint
- bootstrap payloads may seed the first hydrated read when the route already decided that data belongs on first paint
- server functions attach after hydration for later commands, retries, refresh buttons, settings saves, and similar explicit user actions

If JavaScript or hydration is unavailable:

- form-shaped server functions should degrade to the documented server-action or HTML form path when progressive enhancement is promised
- non-form server functions should have an explicit fallback story, such as a normal document navigation, a disabled control, or no enhancement at all

### Cache And Shared State

Server functions may update UI through:

- targeted `fetch` cache invalidation or reload
- explicit `ui.UseForm[T]` projection for form-shaped validation failures
- local state updates for optimistic or completion messaging

They should not silently mutate unrelated route or shared state without one explicit invalidation or update path.
