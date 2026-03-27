# Worker Guidance

This page documents the current worker model for the public `interop` package.

Use it when an app has browser-only CPU-heavy work that should move off the main UI thread without dropping into raw worker wiring from application code.

## Current Status

- The repo ships a real first-party worker surface through `interop.OpenWorker(...)`, typed request and subscription helpers, first-class `MessageChannel` or `MessagePort` helpers for worker subchannels, and the component-facing `ui.UseWorkerTask[...]` bridge.
- The supported shape today is dedicated browser `Worker` usage for explicit app-owned background compute, not a hidden framework scheduler.
- The worker contract is exercised by focused wasm tests in `interop/interop_wasm_test.go` and `ui/ui_wasm_test.go`, and by the runnable `examples/91-worker-text-index` example.

## Scope

The first-party worker surface is currently limited to dedicated browser `Worker` instances plus browser `MessageChannel` or `MessagePort` primitives that pair naturally with those workers.

- supported: dedicated Web Workers created through `interop.OpenWorker(...)`
- service workers now have a first-class companion surface in `pwa.RegisterServiceWorker(...)` for explicit registration and update-lifecycle coordination
- not yet supported as first-class APIs in `interop`: `SharedWorker`, background sync, or service-worker-owned cross-tab coordination
- intended package: `interop`, because workers are browser runtime interop rather than a replacement for normal Go goroutines

Normal Go goroutines still remain the right tool for in-process concurrency inside the wasm runtime. Reach for workers only when the browser thread itself would otherwise be blocked by parsing, indexing, formatting, or other CPU-heavy work.

## Current Boundary

- Shipped: dedicated worker lifecycle management, typed request or progress or result envelopes, explicit cancellation and timeout handling, explicit `MessageChannel` or `MessagePort` subchannels, and one component-level worker task helper.
- Not shipped: `SharedWorker` support, worker pools, framework-owned job scheduling, general transferable-object ownership helpers beyond `MessagePort` handoff, or service-worker coordination hidden behind the same API.
- Service workers are documented separately under `pwa.RegisterServiceWorker(...)`; this page is about dedicated compute workers created by application code.

## Public Surface

The current public API is:

- `interop.OpenWorker(ctx, interop.WorkerOptions{...})`
- `interop.OpenMessageChannel()`
- `channel.Port1()`
- `channel.Port2()`
- `port.Post(...)`
- `port.PostPorts(...)`
- `port.Subscribe(...)`
- `interop.SubscribeDecodedMessagePort[T](port, handler)`
- `worker.Post(...)`
- `worker.PostPorts(...)`
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

When a worker flow needs a dedicated duplex side channel instead of the shared worker event stream:

- create a linked pair through `interop.OpenMessageChannel()`
- pass one endpoint into the worker with `worker.PostPorts(...)`
- use `MessagePort.Post(...)`, `MessagePort.PostPorts(...)`, and `SubscribeDecodedMessagePort[T](...)` on the handed-off port for the custom stream

## Lifecycle

Worker lifecycle is explicit.

- `OpenWorker(...)` creates the dedicated worker and optionally waits for `phase: "ready"`
- `OpenMessageChannel(...)` creates two entangled `MessagePort` endpoints that can stay on one side or be handed off across a worker boundary
- `Terminate()` stops the current worker instance and leaves the handle inactive until `Restart(...)`
- `Restart(...)` terminates any current instance, creates a fresh worker, and reapplies the startup handshake rules
- `Subscribe(...)` and `SubscribeDecodedWorker(...)` are tied to the currently active worker instance; recreate long-lived subscriptions after a restart
- `MessagePort` ownership is explicit; the owner that receives or creates a port should close it when the custom side channel is no longer needed

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
- first-class today: `MessagePort` ownership transfer through `worker.PostPorts(...)`, `scope.PostPorts(...)`, and `port.PostPorts(...)`
- not yet first-class in the public Go API: `ArrayBuffer` transfer helpers, `ReadableStream`, or other browser-native ownership-transfer types
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

The task state surface is intentionally small and explicit: `Running`, `Ready`, `Cancelled`, `Started`, `Error`, `ProgressReady`, `Progress`, and `Value`. The current contract is that starting a new run cancels the previous in-flight request, progress is optional, and component cleanup terminates the owned worker instance.

## Preferred Examples

- `examples/91-worker-text-index`
- `interop/interop_wasm_test.go`
- `ui/ui_wasm_test.go`
