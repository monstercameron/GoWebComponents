# Observability

This page defines the current intended observability model for GoWebComponents applications.

Use it when you need to instrument SSR requests, hydration, routed client activity, or framework lifecycle events without inventing a different event shape for each subsystem.

## Structured Runtime Event Model

The intended event model is one structured envelope shared across server and browser instrumentation.

Every event should carry:

- `name`: stable event name such as `ssr.request.start`, `runtime.hydration.finish`, or `router.loader.resolve`
- `domain`: subsystem such as `ssr`, `runtime`, `router`, `fetch`, `state`, `interop`, `worker`, or `forms`
- `phase`: one of `start`, `progress`, `finish`, `error`, or `cancel`
- `timestamp`: wall-clock event time
- `correlation_id`: one end-to-end page-load or interaction id
- `span_id`: the local event or operation id
- `parent_span_id`: optional parent operation id
- `attributes`: structured scalar fields safe for logs, traces, and metrics export

The current intended domain coverage is:

- SSR request lifecycle
- route matching and loader activity
- bootstrap serialization and restore
- hydration start, mismatch, fallback, and completion
- async-boundary and lazy-node resolution
- cache hit, miss, invalidate, and revalidate activity
- worker request, progress, completion, and timeout flows
- browser interop events where framework-owned lifecycle matters

This is a contract-definition milestone, not a claim that every subsystem already emits these events today.

## Correlation IDs Across SSR, Bootstrap, And Hydration

The intended end-to-end model is one correlation id per page request or browser-owned navigation.

For SSR pages:

1. the server creates a request correlation id at the start of the HTTP request
2. SSR render, loader work, bootstrap serialization, and response write timing all attach to that id
3. the server transfers that same id through bootstrap metadata or a stable HTML attribute
4. client hydration adopts that id for hydration diagnostics, route reuse, and first client-side lifecycle events

For client-only navigations after hydration:

- the router creates a new navigation correlation id
- nested loader work, cache activity, and async-boundary events attach to that navigation id
- background work triggered by the navigation uses child span ids under the same correlation id

Correlation ids should be opaque strings. They must not encode user ids, session ids, auth tokens, or sensitive request metadata.

## Request-Scoped Tracing For SSR Apps

The intended SSR trace for one request is:

1. request accepted
2. route match resolved
3. auth or session context resolved
4. loaders started and completed
5. render started and finished
6. bootstrap serialized
7. response headers written
8. response body finished

Recommended trace attributes today:

- request method, path template, and normalized route id
- SSR surface or screen name when the app has multiple shells
- loader count, loader duration totals, and slowest loader id
- render duration
- bootstrap byte size
- response status
- cache-control mode
- whether the request rendered from a public or authenticated session context

If one request fans out to internal services or external APIs, those calls should inherit the same request context and correlation id rather than generating unrelated trace roots.

## Client Lifecycle Expectations

Once the browser resumes, the intended client event set should cover:

- hydration start and finish
- hydration mismatch and subtree fallback counts
- route navigation start and finish
- loader pending time and resolve time
- async-boundary fallback visibility duration
- cache hit or miss and revalidation activity
- worker job lifecycle and cancellation

The client side should reuse the transferred correlation id for the first hydration pass, then create new correlation ids for later navigations or long-lived user interactions.

## Client-Side Lifecycle Hooks

The intended public client instrumentation surface should expose stable hooks or callback registration for:

- navigation start and completion
- loader pending, resolve, error, and cancel
- hydration start, finish, mismatch, and subtree fallback
- async-boundary fallback visible, resolved, timed out, and errored
- cache hit, miss, invalidate, and revalidate
- worker job start, progress, complete, timeout, and cancel
- render and commit cost summaries at framework-owned boundaries

The intended contract shape is callback-based and event-oriented rather than one callback per subsystem-specific API. Applications should be able to subscribe once and filter by `domain`, `name`, `phase`, or attribute fields instead of wiring separate instrumentation for router, fetch, worker, and hydration paths.

This is still a definition milestone. It does not claim that the runtime already emits all of these lifecycle hooks today.

## Metrics And Diagnostics Boundary

Use this boundary when deciding where data belongs:

- traces answer "what happened for this one request or navigation?"
- metrics answer "how often or how long does this happen over time?"
- diagnostics answer "what went wrong or degraded?"

Examples:

- hydration fallback count belongs in metrics and may also produce one diagnostic entry
- one loader timing sequence belongs in a trace
- one tag mismatch warning belongs in diagnostics with correlation metadata

## Export Formats

The intended export targets are:

- in-memory devtools views for local debugging
- newline-delimited JSON event export for local capture and replay-friendly inspection
- structured trace export shaped so a future bridge to OpenTelemetry-style spans remains possible
- metrics snapshots or counters that can be scraped, serialized, or forwarded without requiring one specific metrics backend

Export rules:

- one canonical internal event model should map outward to different transports
- local file export should preserve correlation ids, timestamps, domains, and safe attributes
- devtools should be able to read the same internal event stream without inventing a parallel diagnostics-only format
- export formats should stay append-only and versioned once public

The project should not lock itself into one tracing or metrics vendor as part of the first public instrumentation surface.

## Sampling And Noise Control

Instrumentation has to stay usable under churn. The intended rules are:

- errors and mismatch or fallback conditions are unsampled by default
- low-cost high-frequency success events should support rate limiting or probabilistic sampling
- repeated identical events should be coalesced where a counter or aggregate summary is more useful than one record per occurrence
- development defaults may keep more detail than production defaults, but the field model should stay the same
- long-lived panels and local file capture should support bounded ring buffers instead of unbounded event growth

Recommended first sampling boundaries:

- per-navigation or per-request traces may be fully retained when they contain an error
- steady-state cache hits, successful renders, and frequent worker progress ticks should be sampled or aggregated
- route transitions and hydration should remain easy to inspect without flooding output during repeated polling or rapid rerenders

## Safe Attribute Rules

- Keep secrets, raw headers, tokens, cookies, CSRF values, and full bootstrap payloads out of attributes.
- Prefer route ids, surface names, counts, durations, booleans, and size buckets over free-form payload dumps.
- Use explicit redaction at subsystem boundaries rather than assuming exporters will clean values later.

## What Is Still Open

These items remain separate backlog work:

- actual public instrumentation hooks for SSR and browser lifecycle emission
- client-side lifecycle APIs and sampling controls
- export formats for traces and metrics
- an end-to-end observability example
