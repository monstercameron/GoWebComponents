# Auth And Session Integration

This guide defines the current recommended auth and session shape for GoWebComponents applications.

Use it when you need a practical answer for:

- cookie versus bearer-token flows
- SSR session resolution
- client-side auth hints after hydration
- route protection and return-to navigation
- logout handling
- same-origin API and mutation calls

## The Short Version

Current recommended defaults:

- resolve the real session on the server
- prefer same-origin cookie-backed sessions for SSR apps and normal same-origin APIs
- treat client-side auth state as a hint for UX, not as the real authority
- use route guards for navigation behavior, not for actual security
- keep API authorization, form authorization, and mutation authorization server-owned
- use bounded return-to helpers for login recovery

Bearer tokens remain an explicit application-owned path, not the default framework recommendation.

## 1. Start With The Authority Boundary

The first auth decision is simple:

- server is authoritative
- browser is advisory

That means:

- protected SSR routes must be enforced on the server
- protected API and form handlers must be enforced on the server
- browser route guards improve UX, but they do not replace server checks

If a direct URL request hits the server first, the server must still decide whether the user is allowed to see the page.

## 2. Recommended Session Model

For most serious apps, prefer same-origin cookie-backed sessions.

Why this is the default:

- SSR handlers can resolve the current user before render
- same-origin form posts and API calls can reuse the same session boundary
- the browser does not need raw upstream bearer tokens just to make ordinary app requests
- logout and expiry can be handled once in the server session layer instead of copied across multiple client transports

Recommended shape:

1. middleware resolves the current session from cookies or equivalent server-owned session state
2. SSR handlers, API handlers, and form posts all read the same request-owned auth context
3. the server may emit a minimal client-safe auth hint into bootstrap for hydrated route behavior
4. later browser-side reads and writes continue to use the same-origin session

## 3. Bearer Token Flows

Bearer tokens are still valid when the application truly needs them, but they are an explicit escape hatch.

Use them when:

- the browser must call an external API directly
- the app is intentionally client-only and not using same-origin server ownership
- an existing platform contract already requires browser-managed token refresh

If you use bearer tokens:

- keep the refresh flow application-owned
- keep the storage decision explicit and reviewable
- do not serialize raw upstream tokens into SSR bootstrap casually
- prefer short-lived access tokens plus server-owned refresh handling when possible

The framework does not currently ship a token-refresh manager or identity-provider integration layer.

## 4. SSR And Bootstrap Auth Hints

SSR apps may transfer a minimal auth hint to the browser after the server resolves the request.

Good bootstrap auth hint content:

- authenticated versus unauthenticated status
- display name already visible in the page
- role or claim summaries that are safe for the client to know
- auth version or revision useful for route revalidation

Do not put these in bootstrap:

- session secrets
- bearer tokens
- raw cookies
- CSRF secrets
- private authorization metadata the browser does not need

Use bootstrap auth hints to avoid a jarring first client redirect or needless “unknown session” flash after hydration. Do not treat them as the final source of truth.

## 5. Route Protection

Use router guards for client navigation behavior.

Current practical pattern:

- synchronous redirect when the app already knows the user is unauthenticated
- manual pending or authorizing UI while client-side session knowledge is still settling
- manual unauthorized UI for denied subsections when the app wants to stay on the same shell
- bounded return-to recovery after successful sign-in

This is the right job for:

- `BeforeEnter`
- `BeforeEnterAsync`
- `router.PreserveReturnTo(...)`
- `router.ReadReturnTo(...)`

This is not the right job for:

- final authorization
- server mutation protection
- data secrecy

See [ROUTER_AUTH.md](ROUTER_AUTH.md) and `examples/92-protected-routes` for the current guarded-route pattern.

## 6. Roles And Claims

Keep role and claim evaluation explicit and application-owned.

Recommended split:

- server evaluates the real authorization policy
- client may read safe role or claim summaries for navigation and presentation
- route guards may use those summaries to avoid bad UX paths
- API handlers and form posts must still re-evaluate the real policy on the server

Good client-visible role usage:

- show or hide optional navigation
- label the current workspace or role mode
- choose which protected route to attempt

Bad client-visible role usage:

- trusting the client role value as proof for API access
- assuming a client-visible claim means server access is already granted

## 7. Logout, Expiry, And Session Changes

Treat logout and session expiry as first-class transitions.

Recommended behavior:

- clear or invalidate persisted client-visible auth-adjacent data on logout
- revalidate route data or shared cache entries that depended on the old user
- force reconnect or re-handshake for long-lived transports when the session changes
- redirect or downgrade protected UI intentionally instead of leaving stale privileged state on screen

Good cleanup candidates on logout:

- shared cached data tied to the old principal
- offline mutation queues that should not replay across users
- bootstrap-resumed auth hints
- any persisted snapshots that would leak across principals on the same device

Sign-out should be treated as one atomic state-reset operation, not as a slow
series of unrelated cleanups.

Recommended order:

1. mark the shell as signing out so protected mutations or reconnect attempts do
   not start new work mid-transition
2. clear auth-derived atoms, auth hints, and any in-memory identity summaries
3. cancel in-flight RPC streams and reject new authenticated transport work
   until a new session exists
4. invalidate or purge user-scoped cache entries, offline queues, and other
   persisted session-derived state
5. remove session storage or browser-held credentials that should not survive
   logout
6. replace the current protected route with the sign-in or public landing route

The important rule is coordination: components should not briefly render with an
old user identity after transport shutdown, and transports should not keep
running after the router has already presented a signed-out shell.

## 8. Same-Origin APIs And Mutations

Prefer same-origin APIs and form handlers for authenticated app flows.

Why:

- cookies and session middleware stay consistent with SSR
- CSRF handling stays server-owned and predictable
- server logs, redirects, and validation failures remain coherent
- browser code does not need to manage external auth details for ordinary product flows

Recommended pattern:

- SSR GET pages: same-origin page handlers
- JSON reads: same-origin API handlers
- HTML posts or JSON mutations: same-origin handlers with server-owned auth and CSRF checks
- browser-side follow-up reads: same-origin fetches or route loaders

See [SERVER_INTEGRATION.md](SERVER_INTEGRATION.md), [FORMS.md](FORMS.md), and [SECURITY.md](SECURITY.md) for the adjacent transport and CSRF rules.

## 9. CSRF And Auth

CSRF still matters when using cookie-backed sessions for unsafe methods.

Recommended model:

- source the CSRF token from the current server-rendered page or bootstrap
- keep validation server-owned
- send the token through the documented header or form-field conventions
- treat CSRF failure as an authoritative server rejection, not a client retry signal

Auth and CSRF solve different problems:

- auth answers who the user is
- CSRF answers whether the browser mutation request is intentionally initiated by the current app flow

## 10. Client-Only Apps

Client-only apps may still need auth, but the tradeoffs are sharper.

Recommended rules:

- keep the token storage decision explicit
- treat browser persistence as a risk boundary, not a convenience detail
- document token refresh and expiry handling early
- prefer same-origin backend facades when the browser would otherwise need broad upstream credentials

If the app may later add SSR, avoid designing around browser-only token assumptions that would be awkward to unwind.

## 11. Single-Shell Runtime Pattern

Some products want one long-lived WASM shell to own marketing routes, sign-in,
sign-up, and the authenticated workspace without any server-rendered detour page
between auth states.

That pattern is supported, but the ownership boundaries should stay explicit:

- the running client owns route transitions and temporary auth UX
- the server still owns the real session, token minting, CSRF validation, and
  final authorization
- sign-in, refresh, and sign-out should resolve inside the existing shell rather
  than by forcing a full page reload unless the platform itself requires it

Recommended single-shell shape:

1. public routes, auth routes, and protected workspace routes live in one router tree
2. sign-in submits to a same-origin action or API and receives a new server
   session or short-lived browser credential
3. the client updates its auth hint in memory and navigates directly into the
   protected workspace
4. background refresh keeps the session current before expiry while the shell
   stays mounted
5. sign-out or refresh failure downgrades the same running shell back to public
   or sign-in routes

Token and session storage rules for this pattern:

- for same-origin cookie sessions, keep the real credential in the cookie and
  keep only client-safe auth hints in memory
- for browser-managed bearer tokens, prefer in-memory access-token storage by
  default and persist only what the platform contract truly requires
- do not persist raw access tokens in local storage, bootstrap payloads, or
  framework-owned diagnostics just to make startup easier
- if a secure browser-storage handle is unavoidable, make that decision
  application-owned, reviewable, and isolated from generic framework state

Background refresh rules:

- schedule refresh from an explicit server expiry or renewal hint rather than
  ad hoc polling
- centralize refresh in one shell-owned coordinator instead of letting multiple
  routes or RPC callers compete
- refresh early enough to avoid an avoidable expiry edge, with a small safety
  window and jitter when many clients may renew at once
- while refresh is in flight, treat auth as pending rather than flashing
  unauthorized UI immediately

Failure handling rules:

- if silent refresh fails, stop treating the old auth hint as valid
- cancel or fail closed any in-flight protected route loaders that depended on
  the expired credential
- force long-lived RPC transports to reconnect or close with an auth-expired
  result instead of letting stale streams continue indefinitely
- transition the router intentionally to sign-in or a public landing route
  without a full document reload
- apply the same ordered state-reset model on explicit sign-out so auth atoms,
  cache state, persisted session data, and route state cannot drift apart

## 12. Recommended Defaults For Real Apps

For a typical non-trivial SSR or routed app:

- server-owned cookie session
- same-origin APIs and form handlers
- minimal server-hydrated auth hint
- guarded client navigation with return-to recovery
- explicit logout invalidation for caches, queued writes, and session-derived UI
- server-owned authorization on every protected read or write

## 13. Common Failure Modes

Watch for these mistakes:

- treating route guards as the real security boundary
- serializing secrets or raw tokens into bootstrap
- leaving user-specific cached data alive after logout
- mixing cookie session auth and bearer-token auth casually in the same flow
- trusting client-visible roles or claims as proof for server access

## 14. Example Starting Points

- guarded client routes and post-login return-to: `examples/92-protected-routes`
- SSR forms, CSRF, and same-origin mutation handling: `examples/87-ssr-secure-forms`
- integrated SSR plus session-aware server shape: `examples/86-atlas-commerce-os/server`
- gRPC-style session-backed app surface: `examples/100-ai-chat-wizard`

## Review Checklist

- is the real session resolved on the server
- are protected reads and writes still enforced server-side
- is the browser carrying only the minimum client-safe auth hint
- does logout clear or invalidate user-owned cached and persisted state
- are return-to and post-auth navigation paths bounded to internal destinations
