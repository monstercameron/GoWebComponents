# Server Integration

This page defines the current production-shaped Go HTTP integration story for GoWebComponents SSR apps.

Use it when you need to know how request handlers, middleware, SSR rendering, bootstrap payloads, asset serving, and API endpoints fit together without inventing a new server shape for each app.

## Canonical App Shape

The current first-party story is a normal Go `net/http` server with five responsibilities:

- serve immutable static assets such as `.wasm`, `wasm_exec.js`, CSS, images, and any worker scripts
- serve one browser entry document route or a set of SSR page routes
- render request-time HTML with `ui.RenderToString(...)`
- emit one bootstrap payload that the browser passes back into `ui.Hydrate(...)`
- expose same-origin API and form-post handlers alongside the SSR routes

This is the intended baseline shape for production apps today. Use the existing SSR examples as the reference pieces:

- `examples/18-ssr-server-routing` for route-aware server rendering
- `examples/87-ssr-secure-forms` for request-time forms, CSRF checks, redirects, and multipart posts
- `examples/73-ssr-bootstrap` and `examples/74-ssr-route-data-reuse` for bootstrap transfer and hydration reuse

## Request Pipeline

For a normal SSR page request, the canonical flow is:

1. top-level HTTP middleware attaches request IDs, logging context, recovery, timeout, session state, and any auth or CSRF context
2. the route handler resolves the current route, request-scoped loader data, and any request-owned view model
3. the handler renders HTML with `ui.RenderToString(...)`
4. the handler embeds or references the bootstrap payload needed for hydration reuse
5. the browser loads the wasm bundle and resumes with `ui.Hydrate(...)` or `router.HydrateMount(...)`

Treat SSR rendering as request-scoped. Do not cache mutable request state in package globals.

## Assets And Entry Documents

- Serve `.wasm` and `wasm_exec.js` as static assets under stable URLs.
- Use cache-busted asset names or explicit immutable-cache headers for versioned bundles.
- Keep the SSR document template responsible for linking CSS, `wasm_exec.js`, the wasm bootstrap loader, and the bootstrap payload script.
- Keep API handlers and asset handlers on the same origin by default unless you intentionally need cross-origin browser traffic.

The simplest supported production setup today is one Go binary that serves both the SSR pages and the static bundle directory.

## Bootstrap Payload Rules

- Serialize only request-scoped data that the browser needs to resume without refetching immediately.
- Keep secrets, raw session material, and internal-only server fields out of bootstrap payloads.
- Treat `ui.SSRBootstrap` as the canonical transfer channel for seeded IDs, atom snapshots, and other hydration-owned resume data.
- Use route-specific bootstrap keys for loader payloads instead of one unstructured global blob.
- Prefer bootstrap references for large payloads when inline scripts would become too large for the HTML response.

See `docs/STATE_TRANSFER.md` for the current classification and ownership rules.
See `docs/SECURITY.md` for the broader threat model and secure serialization boundary.

## Middleware Guidance

The current recommended middleware order for SSR apps is:

1. request ID or correlation ID
2. access logging and timing start
3. panic recovery
4. real client IP or proxy normalization when needed
5. compression
6. cache policy headers
7. session or auth context
8. CSRF protection for unsafe methods
9. request timeout or cancellation context
10. the final app mux with SSR pages, APIs, and static assets

The important rules are:

- Recovery should wrap the SSR handler so panics during render become controlled `500` responses and diagnostics instead of broken connections.
- Auth and session middleware should run before SSR so route handlers and form endpoints see the same user context.
- CSRF validation should run before unsafe form handlers, not inside component render logic.
- Compression should wrap final responses, but be aware that aggressive buffering can interfere with any future streamed SSR transport.
- Request timeout middleware should propagate `context.Context` into loaders, API clients, and external fetches so canceled page requests stop downstream work.

## Internal Versus External API Patterns

The current supported patterns are:

- internal same-origin Go handlers for app-owned data and mutations
- external upstream APIs behind server-owned clients when credentials, retries, rate limits, or schema normalization should stay off the browser
- direct browser fetches only for public, browser-safe endpoints that do not need request-time SSR ownership

Use these rules:

- Route loaders and SSR handlers should prefer internal Go services or internal handlers so one request context can own auth, timeout, and tracing.
- `fetch.UseResource[T](...)` in the browser should call same-origin handlers when the same data also participates in SSR or hydration reuse.
- Form submissions should usually post to same-origin Go handlers that can validate, mutate, redirect, and re-render with shared server-side auth context.
- When a browser flow must call an external API directly, make the CORS, auth-token, and failure-mode tradeoff explicit in app code and docs.

## Timeout And Auth Propagation

- Derive downstream HTTP clients from the incoming request context for SSR loaders and API fan-out.
- Propagate request deadlines to internal service calls so abandoned browser requests do not keep backend work running.
- Resolve the authenticated user once in middleware, then pass that identity through request context or typed handler dependencies.
- Do not serialize bearer tokens, session secrets, or CSRF secrets into bootstrap data just to make browser fetches easier.
- If client-side follow-up requests need auth, prefer same-origin cookie or session flows over exposing raw upstream credentials to wasm code.

## Forms, APIs, And Route Data Together

The intended production shape is:

- SSR GET requests render HTML and include bootstrap state for the current route
- same-origin JSON endpoints power route loaders and browser-side `fetch.UseResource(...)`
- same-origin POST handlers own mutation validation, redirects, and server error shaping
- hydration resumes into the same route tree and reuses bootstrap data where available

This keeps one authoritative server boundary for auth, validation, redirects, and data shaping while still letting the browser do later client-side navigation and resource refreshes.

## First-Party Reference Server

The current first-party reference server is `examples/86-atlas-commerce-os/server`.

Use it when you need one integrated example that combines:

- request-time SSR pages
- browser hydration over shared UI code
- same-origin API and mutation handlers
- static asset serving
- bootstrap payload generation
- CSRF-aware form and mutation handling
- route-aware metadata and session-aware server behavior

`examples/87-ssr-secure-forms` remains the focused form-flow example. Atlas is the production-shaped integrated example.

## Deployment Guidance

The current supported deployment modes are:

- static hosting for client-only wasm bundles
- one Go server that serves SSR pages, APIs, and static assets together
- one Go SSR or API server behind a reverse proxy or CDN
- split SSR/API deployments when both surfaces stay same-origin to the browser through the front door

Guidance by mode:

- Static hosting: use this only for client-only apps or prerendered HTML without request-time SSR ownership.
- Single Go server: this is the default supported production shape for SSR apps because routing, auth, forms, assets, and bootstrap emission stay in one process boundary.
- Reverse proxy in front of Go: preserve request ids, client IP forwarding, cache headers, and compression behavior, and verify that proxy buffering does not conflict with any future streaming SSR experiments.
- Split SSR/API topology: keep browser traffic same-origin through the frontend domain when possible, and make timeouts, auth propagation, and cache headers consistent across both server tiers.

Operational rules:

- Serve versioned static assets with immutable caching when possible.
- Keep HTML responses and bootstrap payloads on shorter cache lifetimes unless the route is explicitly cacheable.
- Validate `.wasm` MIME type, `wasm_exec.js` version match, and asset path rewrites in staging before release.
- When using a CDN or reverse proxy, verify route rewrites for browser-router deep links and SSR route pass-through separately.

## What Is Still Open

These items are still deliberately separate backlog work:

- observability hooks for request-level SSR timing and hydration counters
