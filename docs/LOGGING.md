# Logging

This page defines the current first-class logging surface for GoWebComponents applications and framework integrations.

Use it when you need one coherent logging model for runtime, router, fetch, hydration, SSR, interop, worker, and form activity instead of ad hoc `fmt.Println(...)` output.

## At A Glance

- The public `logging` package already ships a small structured logger built around stable scopes, levels, messages, and field maps.
- `logging.New(scope)` is the normal application entrypoint for emitting scoped `debug`, `info`, `warn`, and `error` events.
- `logging.AttachBrowserConsole(...)` can wire common browser lifecycle and interaction events into that same structured surface on `js/wasm` builds.
- Broader redaction, diagnostics, and observability rules still matter because logs are only one part of the framework feedback model.

## Quick API Chooser

Use this rule of thumb:

- choose `logging.New(scope)` when one feature, route, or example needs a reusable scoped logger
- choose package-level `logging.Log(...)` when you already have a level and scope and do not need a retained logger value
- choose `AttachBrowserConsole(...)` in browser entrypoints when you want document, navigation, form, resize, rejection, and mount events logged through the same package
- choose diagnostics or observability hooks alongside logs when the problem is framework degradation, mismatch analysis, or correlation across larger workflows rather than just event sequencing

## Example Shape

```go
log := logging.New("checkout")

cleanup := logging.AttachBrowserConsole(logging.BrowserConsoleOptions{Scope: "checkout"})
defer cleanup()

log.Info("checkout mounted", logging.Fields{"route": "/checkout"})
log.Warn("payment widget slow", logging.Fields{"duration_ms": 420})
log.Error("submit failed", logging.Fields{"status": 502, "retryable": true})
```

This is the current intended application shape: one stable scope per feature area, concise messages, structured safe fields, and optional browser-console wiring in `js/wasm` entrypoints.

## Current Public Surface

The current public package is `github.com/monstercameron/GoWebComponents/logging`.

Current entrypoints:

- `logging.New(scope)`
- `Logger.Scope()`
- `Logger.Log(...)`
- `Logger.Debug(...)`
- `Logger.Info(...)`
- `Logger.Warn(...)`
- `Logger.Error(...)`
- `logging.Log(level, scope, message, fields)`
- `logging.AttachBrowserConsole(logging.BrowserConsoleOptions{...})`

## Logger Interface And Domain Model

The public logger contract is a structured sink with:

- `level`: `debug`, `info`, `warn`, or `error`
- `domain`: stable subsystem name such as `runtime`, `router`, `fetch`, `state`, `hydration`, `ssr`, `interop`, `worker`, or `forms`
- `message`: short human-readable summary
- `timestamp`: event time
- `correlation_id`: optional request or navigation id
- `fields`: structured safe key/value attributes

Recommended domain rules are:

- `runtime`: render, commit, boundary recovery, and scheduler-owned events
- `hydration`: mismatch, fallback, resume, and strict-mode failures
- `router`: navigation, redirect, loader, and guard events
- `fetch`: resource load, cache, revalidate, upload, and mutation-queue events
- `state`: atom or derived-state events that are framework-owned and worth surfacing
- `ssr`: request render, bootstrap, and response lifecycle events
- `interop`: browser-wrapper failures and lifecycle warnings
- `worker`: worker startup, timeout, restart, progress, and completion events
- `forms`: validation, submission, redirect, and server-error shaping events

The framework should prefer stable scopes or domains and structured fields over embedding operational detail in free-form message text.

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
- multi-client peer discovery, timeout, transport, reconnect, and authorization failures when the experimental multi-client surface is in use

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

See [MULTI_CLIENTS.md](MULTI_CLIENTS.md) for the intended peer-id, topic, payload-size, and binary-redaction rules specific to sovereign browser-client coordination.

## What Is Still Open

These items remain separate backlog work:

- a full project-wide redaction policy shared by logs, diagnostics, and traces
- richer sink integrations beyond the in-memory and docs-defined model

## Review Checklist

- does application code use a stable scope through `logging.New(...)` instead of scattering unrelated free-form console output
- are field maps structured and safe instead of dumping raw payloads, secrets, or whole request bodies
- is `AttachBrowserConsole(...)` only enabled where browser lifecycle and interaction logging is actually useful
- are logs, diagnostics, and observability concerns kept complementary rather than forcing one channel to explain everything
- do production-facing logs preserve stable field names that downstream tooling can consume without parsing message text
