# Logging

This page defines the intended first-class logging surface for GoWebComponents applications and framework integrations.

Use it when you need one coherent logging model for runtime, router, fetch, hydration, SSR, interop, worker, and form activity instead of ad hoc `fmt.Println(...)` output.

## Logger Interface And Domain Model

The intended public logger contract is a structured sink with:

- `level`: `debug`, `info`, `warn`, or `error`
- `domain`: stable subsystem name such as `runtime`, `router`, `fetch`, `state`, `hydration`, `ssr`, `interop`, `worker`, or `forms`
- `message`: short human-readable summary
- `timestamp`: event time
- `correlation_id`: optional request or navigation id
- `fields`: structured safe key/value attributes

The intended domain rules are:

- `runtime`: render, commit, boundary recovery, and scheduler-owned events
- `hydration`: mismatch, fallback, resume, and strict-mode failures
- `router`: navigation, redirect, loader, and guard events
- `fetch`: resource load, cache, revalidate, upload, and mutation-queue events
- `state`: atom or derived-state events that are framework-owned and worth surfacing
- `ssr`: request render, bootstrap, and response lifecycle events
- `interop`: browser-wrapper failures and lifecycle warnings
- `worker`: worker startup, timeout, restart, progress, and completion events
- `forms`: validation, submission, redirect, and server-error shaping events

The framework should prefer stable domains and structured fields over embedding operational detail in free-form message text.

## Development Versus Production Outputs

The intended output modes are:

- development output that is human-readable and optimized for terminal or console scanning
- production output that is machine-readable and structured for aggregation or transport

Development output should favor:

- concise line-oriented formatting
- visible level and domain prefixes
- compact correlation-id display when present
- readable summaries of common fields such as route, status, duration, cache key, or component path

Production output should favor:

- one structured record per log event
- stable field names across releases
- correlation ids and domain fields preserved without parsing message text
- compatibility with later forwarding into tracing, metrics, or centralized logging systems

Applications should be able to select a formatter or sink without forking framework behavior. The logging event model should stay consistent even when the output format changes.

## Redaction And Secret Boundaries

These rules are intended to be non-optional for framework-owned logs:

- never log bootstrap payload bodies, cookies, bearer tokens, CSRF secrets, or raw session values
- never log raw request or response headers by default
- never log full form submissions or mutation bodies unless an application explicitly opts into its own redacted application logger
- prefer normalized route ids, mutation kinds, status codes, counts, durations, cache keys, and size buckets over raw payload dumps
- treat correlation ids as opaque request or navigation identifiers, not user identifiers

If a field is potentially sensitive and the framework cannot prove it is safe, the framework should omit it rather than logging and hoping downstream sinks redact it later.

## Route, Loader, And Mutation Integration

The intended framework-owned logging points are:

- router navigation start, blocked navigation, and redirect decisions
- route-loader start, resolve, error, and cancellation
- cache invalidation and revalidation
- queued mutation persistence, retry scheduling, success, and dead-letter transitions
- form submission lifecycle only for framework-owned state transitions, not raw form data payloads

This keeps the logs focused on framework lifecycle and transport state rather than duplicating full application business logs.

## Warning Classification And Escalation

Framework warnings should be classed so they stay actionable:

- `correctness`: invariant failures, invalid hook usage, broken contracts, or failed mutations
- `performance`: slow commits, effects, cleanups, or other measured cost warnings
- `recovered`: framework-owned cases that degraded safely and continued
- `unsupported_recovered`: unsupported or suspicious conditions that were ignored or fell back without crashing
- `informational`: routine lifecycle notices

Escalation rules:

- correctness failures should surface as `error` logs or diagnostics
- performance issues should stay warnings unless the runtime cannot safely continue
- recovered and unsupported-but-recovered cases should stay warnings with enough context to explain the fallback
- informational notices should be suppressible in production-oriented sinks

## Relationship To Diagnostics And Observability

- diagnostics answer "what degraded or failed?"
- logs answer "what happened in sequence?"
- traces answer "what happened for this request or interaction graph?"

One event may surface in more than one channel, but the same stable domain and field names should apply across logs, diagnostics, and future traces whenever possible.

See [SECURITY.md](SECURITY.md) for the broader server-only data and redaction policy that surrounds framework-owned logging and diagnostics.

## What Is Still Open

These items remain separate backlog work:

- a full project-wide redaction policy shared by logs, diagnostics, and traces
- richer sink integrations beyond the in-memory and docs-defined model
