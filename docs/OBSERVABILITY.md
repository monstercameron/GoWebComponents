# Observability

This page defines the current observability model for GoWebComponents applications.

Use it when you need to instrument SSR requests, hydration, routed client activity, or framework lifecycle events without inventing a different event shape for each subsystem.

## At A Glance

- a real first-class observability slice already ships for SSR render, bootstrap serialization, and hydration
- the public entry points are `ui.ObserveSSR(...)`, `ui.RenderToStringObserved(...)`, `ui.MarshalSSRBootstrapObserved(...)`, `ui.MarshalSSRBootstrapBinaryObserved(...)`, `ui.RenderBootstrapScriptObserved(...)`, and hydration options with `SSRObservabilityOptions`
- `devtools` and runtime diagnostics remain the current in-process inspection surface for broader client debugging
- broader router, fetch, worker, and long-lived client lifecycle instrumentation is still planned rather than fully emitted today

## Quick API Chooser

- use `ui.RenderToStringObserved(...)` when an SSR render should emit one structured render observation
- use `ui.MarshalSSRBootstrapObserved(...)`, `ui.MarshalSSRBootstrapBinaryObserved(...)`, or `ui.RenderBootstrapScriptObserved(...)` when bootstrap payload size needs to be measured alongside render work
- use `ui.ObserveSSR(...)` when the application wants a shared subscription for SSR and hydration observations
- use `ui.Hydrate(...)` or `ui.HydrateInto(...)` with `HydrationOptions{Observability: ...}` when correlation ids and hydration metrics should flow into the same event stream
- use `devtools.SnapshotNow()` or `devtools.UseSnapshot(...)` when the goal is interactive inspection of diagnostics, cache state, logs, or multi-client state rather than SSR event collection

## Current Shipped Slice

Today the framework-owned observability contract covers:

- SSR render timing
- bootstrap payload and inline-script size metrics for JSON and CBOR flows
- hydration duration, mismatch counts, fallback counts, discarded-node counts, and existing DOM counts
- request-scoped or render-scoped correlation ids provided by the application
- process-local subscription through `ui.ObserveSSR(...)` plus per-call `OnEvent` callbacks

Today it does not yet claim:

- a unified emitted event stream for every router, fetch, worker, or cache lifecycle
- production-ready sampling and backpressure controls across all subsystems
- vendor-specific trace or metrics exporters
- a single framework-owned client observability bus for every runtime event

## Example Shape

This is the current intended usage pattern for the shipped slice.

```go
package server

import (
	"log"

	"github.com/atdiar/particleui/ui"
)

func renderPage(root ui.Node) (string, error) {
	unsubscribe := ui.ObserveSSR(func(event ui.SSRObservation) {
		log.Printf("name=%s phase=%s correlation=%s", event.Name, event.Phase, event.CorrelationID)
	})
	defer unsubscribe()

	markup, err := ui.RenderToStringObserved(root, ui.SSRObservabilityOptions{
		CorrelationID: "req-42",
		OnEvent: func(event ui.SSRObservation) {
			if event.Render != nil {
				log.Printf("render_ns=%d", event.Render.DurationNs)
			}
		},
	})
	if err != nil {
		return "", err
	}

	bootstrap := ui.SSRBootstrap{}
	if _, err := ui.RenderBootstrapScriptObserved(bootstrap, "", ui.SSRObservabilityOptions{CorrelationID: "req-42"}); err != nil {
		return "", err
	}

	return markup, nil
}
```

## End-To-End Observability Walkthroughs

The current end-to-end reference examples are:

- `examples/18-ssr-server-routing` for one request-time SSR request plus hydration resume
- `examples/17-ssr-routing` for one routed shell with nested async UI that future streaming and broader client lifecycle instrumentation will extend

### 1. One Server-Rendered Request

Use one request-scoped correlation id and keep the same id across:

- SSR render
- bootstrap serialization
- bootstrap reference rendering
- hydration resume

Recommended server shape:

```go
correlationID := requestID()

unsubscribe := ui.ObserveSSR(func(event ui.SSRObservation) {
    appSink.Record(event)
})
defer unsubscribe()

markup, err := ui.RenderToStringObserved(root, ui.SSRObservabilityOptions{
    CorrelationID: correlationID,
})
if err != nil {
    return err
}

bootstrapBytes, err := ui.MarshalSSRBootstrapObserved(payload, ui.SSRObservabilityOptions{
    CorrelationID: correlationID,
})
if err != nil {
    return err
}

_ = bootstrapBytes
```

Recommended client resume shape:

```go
_, _ = ui.Hydrate(root, "#app", ui.HydrationOptions{
    Bootstrap: payload,
    Observability: ui.SSRObservabilityOptions{
        CorrelationID: correlationID,
        OnEvent: func(event ui.SSRObservation) {
            appSink.Record(event)
        },
    },
})
```

What this gives you today:

- SSR render timing through `ui.RenderToStringObserved(...)`
- bootstrap payload and script size through the observed bootstrap helpers
- hydration duration, mismatch counts, fallback counts, and discarded-node counts through `HydrationOptions{Observability: ...}`

This is the current end-to-end shipped observability path.

### 2. One Routed Page Load

For a routed SSR page such as `examples/18-ssr-server-routing/docs/ssr`:

1. create one correlation id at HTTP request entry
2. attach it to observed SSR render and bootstrap work
3. transfer it through the bootstrap payload or another stable page-owned value
4. reuse that same id during the first hydration pass

The routed example already gives you the structure needed for this pattern:

- route-specific SSR view resolution on the server
- route-specific bootstrap payloads
- browser hydration using the same bootstrap payload

The current observer slice does not yet emit a full router navigation event family after hydration. Today the public end-to-end route-load story is:

- framework-owned SSR and hydration observations through `ui`
- app-owned route/load correlation continuity across server and browser
- optional local diagnostics and log capture through `devtools` and `logging`

### 3. One Async Resource Flow

The broader framework-owned async resource event stream is still backlog work. Today the practical recipe is:

1. use the shipped SSR or hydration correlation id as the parent operation id
2. instrument the application-owned async work with that same id
3. keep framework-owned SSR or hydration events and app-owned async records in the same sink

Recommended current split:

- framework-owned timing and hydration correctness come from `ui.ObserveSSR(...)` and `HydrationOptions{Observability: ...}`
- application-owned async resource timing comes from app code, optionally mirrored into `logging`, `devtools`, or browser performance marks

For example, `examples/17-ssr-routing` already includes nested async UI through `ui.Lazy`. That route family is the current example baseline for future broader async-boundary instrumentation, even though the framework does not yet emit dedicated public lazy-boundary events.

## Current Example References

Use these when wiring the shipped slice end to end:

- `examples/18-ssr-server-routing` for request-time SSR, bootstrap transfer, and hydration reuse
- `examples/17-ssr-routing` for routed-shell SSR and nested async UI used as the future streaming/async instrumentation baseline

These references are intentionally concrete so teams can standardize one observability envelope before the wider router/fetch/worker event family is fully emitted by the framework.

## Structured Runtime Event Model

The long-term event model is one structured envelope shared across server and browser instrumentation.

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

This remains the direction for the broader observability surface, not a claim that every subsystem already emits these events today.

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

## Current Event Shape

The shipped SSR and hydration events already use a stable public shape through `ui.SSRObservation`.

Current event fields:

- `Name`
- `Domain`
- `Phase`
- `Timestamp`
- `CorrelationID`
- `Render` for SSR render timing
- `Bootstrap` for payload and script size metrics
- `Hydration` for browser resume metrics

That means applications can start standardizing correlation ids and local sinks now, even before the wider client-lifecycle story is complete.

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

For multi-client coordination, the intended event family also includes peer discovery, lease expiry, query and result timing, transport resolution, reconnect, and unsupported-encoding failures as documented in `docs/MULTI_CLIENTS.md`.

## Client-Side Lifecycle Hooks

The broader public client instrumentation surface should eventually expose stable hooks or callback registration for:

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

## Current Inspection Surface Beyond SSR

Outside the shipped SSR and hydration observer slice, the practical observability stack today is:

- `devtools.Panel` for in-app inspection
- `devtools.SnapshotNow()` and `devtools.UseSnapshot(...)` for programmatic snapshots
- structured runtime diagnostics surfaced through devtools snapshots
- `logging` for scoped structured logs and optional browser-console lifecycle capture

Use those surfaces when the question is interactive debugging or runtime inspection, not only SSR event timing.

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

## First-Party Integration Recipes

The current first-party guidance is to treat `ui.SSRObservation` as the canonical framework-owned event payload, then adapt that payload into whichever sink the application already uses.

Use one adapter layer per process instead of scattering sink-specific wiring across handlers, route loaders, and hydration entry points.

### OpenTelemetry-Style Traces

Use the framework correlation id as the trace root or request correlation attribute, then map each observation into one span or one span event depending on the sink shape.

Recommended current mapping:

- `CorrelationID` becomes the trace or request correlation key
- `Name` becomes the span name or span event name
- `Domain` and `Phase` become stable attributes
- `Render`, `Bootstrap`, and `Hydration` metrics become typed attributes on the emitted span or event

Example adapter shape:

```go
unsubscribe := ui.ObserveSSR(func(event ui.SSRObservation) {
    ctx, span := tracer.Start(context.Background(), event.Name)
    defer span.End()

    span.SetAttributes(
        attribute.String("gwc.domain", event.Domain),
        attribute.String("gwc.phase", event.Phase),
        attribute.String("gwc.correlation_id", event.CorrelationID),
    )
    if event.Render != nil {
        span.SetAttributes(attribute.Int64("gwc.render.duration_ns", event.Render.DurationNs))
    }
    if event.Bootstrap != nil {
        span.SetAttributes(
            attribute.String("gwc.bootstrap.format", event.Bootstrap.Format),
            attribute.Int("gwc.bootstrap.payload_bytes", event.Bootstrap.PayloadBytes),
            attribute.Int("gwc.bootstrap.script_bytes", event.Bootstrap.ScriptBytes),
        )
    }
    if event.Hydration != nil {
        span.SetAttributes(
            attribute.Int64("gwc.hydration.duration_ns", event.Hydration.DurationNs),
            attribute.Int("gwc.hydration.mismatch_count", event.Hydration.MismatchCount),
            attribute.Int("gwc.hydration.fallback_count", event.Hydration.FallbackCount),
        )
    }

    _ = ctx
})
defer unsubscribe()
```

Keep the adapter thin. The framework event already carries the stable fields you need. The application still owns trace parentage for route loaders, auth resolution, and external service calls.

### Structured Metrics

Metrics work best when the adapter aggregates repeated observation fields into counters and histograms rather than forwarding one raw event as one metric sample.

Recommended current mapping:

- histogram:
  - render duration
  - hydration duration
  - bootstrap payload bytes
  - bootstrap script bytes
- counter:
  - hydration mismatches
  - hydration fallbacks
  - hydration failures
  - SSR render errors

Example adapter shape:

```go
unsubscribe := ui.ObserveSSR(func(event ui.SSRObservation) {
    switch {
    case event.Render != nil:
        metrics.RecordHistogram("gwc_ssr_render_duration_ns", event.Render.DurationNs, event.Domain, event.Phase)
    case event.Bootstrap != nil:
        metrics.RecordHistogram("gwc_ssr_bootstrap_payload_bytes", int64(event.Bootstrap.PayloadBytes), event.Domain, event.Phase)
        metrics.RecordHistogram("gwc_ssr_bootstrap_script_bytes", int64(event.Bootstrap.ScriptBytes), event.Domain, event.Phase)
    case event.Hydration != nil:
        metrics.RecordHistogram("gwc_runtime_hydration_duration_ns", event.Hydration.DurationNs, event.Domain, event.Phase)
        metrics.AddCounter("gwc_runtime_hydration_mismatch_total", int64(event.Hydration.MismatchCount), event.Domain, event.Phase)
        metrics.AddCounter("gwc_runtime_hydration_fallback_total", int64(event.Hydration.FallbackCount), event.Domain, event.Phase)
        if event.Hydration.Failed {
            metrics.AddCounter("gwc_runtime_hydration_failure_total", 1, event.Domain, event.Phase)
        }
    }
})
defer unsubscribe()
```

Keep metric names low-cardinality. Prefer route ids, screen names, and bounded status labels over free-form paths or payload-derived strings.

### Browser Performance Marks

Browser performance marks are the easiest way to line up framework timing with page-level user timing without inventing a second client event model.

Recommended current mapping:

- mark at hydration start when the browser receives the first hydration event
- measure hydration duration using the observation timing
- create named measures for SSR bootstrap size only when the application explicitly needs them for local diagnostics

Example browser-side shape:

```go
_, _ = ui.Hydrate(root, "#app", ui.HydrationOptions{
    Bootstrap: payload,
    Observability: ui.SSRObservabilityOptions{
        CorrelationID: correlationID,
        OnEvent: func(event ui.SSRObservation) {
            if event.Hydration == nil {
                return
            }
            performanceMark("gwc:hydration:start", event.Hydration.StartedAt)
            performanceMark("gwc:hydration:finish", event.Hydration.FinishedAt)
            performanceMeasure("gwc:hydration", "gwc:hydration:start", "gwc:hydration:finish")
        },
    },
})
```

Use this path for local browser tuning and devtools inspection. It is complementary to traces and metrics, not a replacement for them.

### External Error Reporting

Hosted error sinks should receive only the failure-oriented subset of the observation stream plus correlation metadata that lets operators join the error back to traces and logs.

Recommended current forwarding rules:

- forward `Phase == "error"` events
- forward hydration mismatch or fallback events only when they cross app-owned severity thresholds
- attach `CorrelationID`, `Name`, `Domain`, and the typed metric summary
- do not attach raw bootstrap payloads, DOM dumps, tokens, cookies, or full request headers

Example adapter shape:

```go
unsubscribe := ui.ObserveSSR(func(event ui.SSRObservation) {
    if event.Phase != "error" {
        return
    }

    report := errorSink.NewEvent(event.Name)
    report.SetTag("gwc.domain", event.Domain)
    report.SetTag("gwc.phase", event.Phase)
    report.SetTag("gwc.correlation_id", event.CorrelationID)

    if event.Hydration != nil {
        report.SetExtra("duration_ns", event.Hydration.DurationNs)
        report.SetExtra("mismatch_count", event.Hydration.MismatchCount)
        report.SetExtra("fallback_count", event.Hydration.FallbackCount)
        report.SetExtra("failure", event.Hydration.Failure)
    }

    errorSink.Capture(report)
})
defer unsubscribe()
```

This keeps the high-volume success path in metrics and traces while reserving hosted error sinks for actionable failures and degraded hydration cases.

### Recommended Wiring Boundary

Use this split consistently:

- framework emits `ui.SSRObservation`
- application adapter maps that event into traces, metrics, perf marks, logs, or hosted error sinks
- subsystem-specific app work such as loaders, auth, RPC, and external APIs attaches to the same correlation id but remains app-owned instrumentation

That boundary keeps the framework transport-agnostic while still giving teams one documented recipe instead of four unrelated instrumentation styles.

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

The current public SSR slice now ships through `ui.ObserveSSR(...)`, `ui.RenderToStringObserved(...)`, `ui.MarshalSSRBootstrapObserved(...)`, `ui.MarshalSSRBootstrapBinaryObserved(...)`, `ui.RenderBootstrapScriptObserved(...)`, and hydration options that accept `SSRObservabilityOptions`.

That shipped slice covers:

- request-scoped or render-scoped correlation ids supplied by the application
- SSR render timing for `ui.RenderToString(...)` and `ui.RenderToStringObserved(...)`
- bootstrap payload bytes for JSON and CBOR serialization, plus inline script bytes for rendered bootstrap scripts
- hydration duration, existing server DOM counts, mismatch counts, fallback counts, and discarded-node counts when `ui.Hydrate(...)` or `ui.HydrateInto(...)` runs in the browser

These items remain separate backlog work:

- broader client lifecycle APIs beyond hydration
- sampling controls and backpressure policy for high-volume production traffic
- framework-owned exporters for traces and metrics

## Review Checklist

- does the doc distinguish the shipped SSR or hydration observer slice from the broader future instrumentation model
- are correlation ids treated as opaque operational identifiers rather than user or session identifiers
- are diagnostics, logs, traces, and metrics responsibilities kept distinct
- does the instrumentation guidance avoid raw secrets, headers, cookies, and full payload dumps
- do examples use the public `ui`, `devtools`, and `logging` surfaces instead of runtime internals
