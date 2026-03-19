# Worker Guidance

This page documents the current worker model for the public `interop` package.

Use it when an app has browser-only CPU-heavy work that should move off the main UI thread without dropping into raw worker wiring from application code.

## Scope

The first-party worker surface is currently limited to dedicated browser `Worker` instances.

- supported: dedicated Web Workers created through `interop.NewWorker(...)`
- not yet supported as first-class APIs: `SharedWorker`, `ServiceWorker`, background sync, or cross-tab coordination
- intended package: `interop`, because workers are browser runtime interop rather than a replacement for normal Go goroutines

Normal Go goroutines still remain the right tool for in-process concurrency inside the wasm runtime. Reach for workers only when the browser thread itself would otherwise be blocked by parsing, indexing, formatting, or other CPU-heavy work.

## Public Surface

The current public API is:

- `interop.NewWorker(ctx, interop.WorkerOptions{...})`
- `worker.Post(...)`
- `worker.Subscribe(...)`
- `interop.SubscribeDecodedWorker[T](worker, handler)`
- `worker.Request(ctx, name, payload, onProgress)`
- `interop.RequestWorkerDecoded[Req, Progress, Result](...)`
- `ui.UseWorkerTask[Req, Progress, Result](...)`
- `worker.Terminate()`
- `worker.Restart(ctx)`

`WorkerOptions` currently supports:

- `URL`: worker entrypoint URL
- `Name`: optional worker name
- `Type`: `classic` or `module`
- `Ready`: wait for a startup handshake before returning
- `ReadyTimeout`: optional timeout for the ready handshake

## Message Envelope

The typed request helpers assume a JSON-shaped envelope:

```json
{
  "id": "worker-1",
  "phase": "progress",
  "name": "build-index",
  "payload": { "percent": 40 },
  "error": ""
}
```

Field meanings:

- `id`: request correlation id; omit it for one-way events
- `phase`: `ready`, `request`, `progress`, `result`, `error`, or `message`
- `name`: logical operation or event name
- `payload`: JSON-shaped request, progress, or result body
- `error`: remote error text for `phase == "error"`

Rules:

- `worker.Request(...)` posts `phase: "request"` with an auto-generated `id`
- progress updates should echo the same `id` with `phase: "progress"`
- final success should echo the same `id` with `phase: "result"`
- final failure should echo the same `id` with `phase: "error"`
- startup handshake should post `phase: "ready"` when `WorkerOptions.Ready` is enabled

Untyped event streams can still use `worker.Post(...)` and `worker.Subscribe(...)`, but the envelope above is the supported contract for typed request or progress flows.

## Lifecycle

Worker lifecycle is explicit.

- `NewWorker(...)` creates the dedicated worker and optionally waits for `phase: "ready"`
- `Terminate()` stops the current worker instance and leaves the handle inactive until `Restart(...)`
- `Restart(...)` terminates any current instance, creates a fresh worker, and reapplies the startup handshake rules
- `Subscribe(...)` and `SubscribeDecodedWorker(...)` are tied to the currently active worker instance; recreate long-lived subscriptions after a restart

For UI ownership, keep the worker handle scoped to the route, component, or service that owns the background job. Cleanup should terminate the worker or at least cancel any in-flight request contexts when the owner unmounts.

## Cancellation, Timeouts, And Backpressure

`worker.Request(...)` is context-driven.

- `context.WithCancel(...)` cancels the wait path and returns `interop` error code `cancelled`
- `context.WithTimeout(...)` or deadlines return code `timeout`
- late worker responses are ignored because the request listeners are removed once the request completes, times out, or is cancelled

Backpressure is currently defined, but not yet automated by a queueing API:

- each `Request(...)` is independent
- the framework does not yet provide a built-in scheduler, request coalescer, or worker pool
- apps should debounce noisy UI input, cap concurrent requests, or serialize work explicitly when the worker can be overwhelmed
- if a worker protocol only supports one active job at a time, enforce that rule in app code or in the worker script until first-class coordination helpers land

## Serialization Boundaries

Worker payloads should stay structured-clone friendly and, for typed decoding, JSON-shaped.

- preferred: strings, booleans, numbers, arrays, slices, maps, and structs
- acceptable with care: byte slices or large arrays, as long as the worker script expects cloneable data
- not yet first-class in the public Go API: transferable objects, `MessagePort`, `ReadableStream`, or other browser-native ownership-transfer types
- unsupported for typed helpers: functions, DOM nodes, cyclic objects, and opaque host objects

If a payload cannot be serialized through the current `interop` conversion layer, the call fails with structured encode or decode errors instead of silently dropping values.

## UI Composition

`ui.UseWorkerTask(...)` is now the first public component-facing bridge for worker jobs.

Use it when a component should:

- reuse one worker instance across repeated runs
- launch a typed request from a UI event
- surface pending, progress, ready, cancellation, and failure state through one hook handle
- terminate the worker automatically when the owning component unmounts

The lower-level `interop` worker APIs remain the right choice for service-style ownership or custom multi-request orchestration outside one component.

## Preferred Examples

- `examples/91-worker-text-index`
- `interop/interop_wasm_test.go`
