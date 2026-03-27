# Worker Guidance

This page documents the current worker model for the public `interop` package.

Use it when an app has browser-only CPU-heavy work that should move off the main UI thread without dropping into raw worker wiring from application code.

## Current Status

- The repo ships a real first-party worker surface through `interop.OpenWorker(...)`, `interop.OpenGoWASMWorker(...)`, `interop.OpenWorkerPool(...)`, typed request and subscription helpers, first-class `MessageChannel` or `MessagePort` helpers for worker subchannels, optional `SharedArrayBuffer` or `Atomics` helpers for shared-memory coordination, and the component-facing `ui.UseWorkerTask[...]` bridge.
- The supported shape today is dedicated browser `Worker` usage for explicit app-owned background compute, not a hidden framework scheduler.
- The worker contract is exercised by focused wasm tests in `interop/interop_wasm_test.go` and `ui/ui_wasm_test.go`, and by the runnable `examples/91-worker-text-index` example.
- Pool-versus-lane selection guidance for performance-sensitive fanout paths lives in `WORKER_POOLS_VS_LANES.md`.

## Scope

The first-party worker surface is currently limited to dedicated browser `Worker` instances plus browser `MessageChannel` or `MessagePort` primitives that pair naturally with those workers, with an optional shared-memory path through `SharedArrayBuffer` or `Atomics` when the page is cross-origin-isolated.

- supported: dedicated Web Workers created through `interop.OpenWorker(...)`
- service workers now have a first-class companion surface in `pwa.RegisterServiceWorker(...)` for explicit registration and update-lifecycle coordination
- not yet supported as first-class APIs in `interop`: `SharedWorker`, background sync, or service-worker-owned cross-tab coordination
- intended package: `interop`, because workers are browser runtime interop rather than a replacement for normal Go goroutines

Normal Go goroutines still remain the right tool for in-process concurrency inside the wasm runtime. Reach for workers only when the browser thread itself would otherwise be blocked by parsing, indexing, formatting, or other CPU-heavy work.

## Current Boundary

- Shipped: dedicated worker lifecycle management, typed request or progress or result envelopes, fixed-size worker-pool scheduling with bounded queueing, explicit cancellation and timeout handling, explicit `MessageChannel` or `MessagePort` subchannels, optional shared-memory capability inspection plus `SharedBuffer` byte, atomic, and wait or notify helpers, and one component-level worker task helper.
- Not shipped: `SharedWorker` support, framework-owned job scheduling beyond explicit app-owned pools, general transferable-object ownership helpers beyond `MessagePort` handoff and `SharedBuffer`, or service-worker coordination hidden behind the same API.
- Service workers are documented separately under `pwa.RegisterServiceWorker(...)`; this page is about dedicated compute workers created by application code.

## Public Surface

The current public API is:

- `interop.OpenWorker(ctx, interop.WorkerOptions{...})`
- `interop.OpenGoWASMWorker(ctx, interop.GoWASMWorkerOptions{...})`
- `interop.OpenWorkerPool(ctx, interop.WorkerPoolOptions{...})`
- `interop.OpenMessageChannel()`
- `interop.GetSharedMemorySupport()`
- `interop.OpenSharedBuffer(byteLength)`
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
- `pool.Request(ctx, name, payload, onProgress)`
- `pool.Drain(ctx)`
- `pool.Close()`
- `pool.GetSize()`
- `pool.GetQueueLimit()`
- `interop.GetWorkerScope()`
- `scope.PostPorts(...)`
- `sharedBuffer.GetByteLength()`
- `sharedBuffer.ReadBytes(offset, dest)`
- `sharedBuffer.WriteBytes(offset, source)`
- `sharedBuffer.GetInt32Length()`
- `sharedBuffer.LoadInt32(index)`
- `sharedBuffer.StoreInt32(index, value)`
- `sharedBuffer.AddInt32(index, delta)`
- `sharedBuffer.SubInt32(index, delta)`
- `sharedBuffer.AndInt32(index, mask)`
- `sharedBuffer.OrInt32(index, mask)`
- `sharedBuffer.XorInt32(index, mask)`
- `sharedBuffer.ExchangeInt32(index, value)`
- `sharedBuffer.CompareExchangeInt32(index, oldValue, newValue)`
- `sharedBuffer.WaitInt32(index, expected, timeout)`
- `sharedBuffer.NotifyInt32(index, count)`
- `ui.UseWorkerTask[Req, Progress, Result](...)`
- `worker.Terminate()`
- `worker.Restart(ctx)`

## Shared Memory Contract

Shared memory is an optional advanced path for dedicated workers, not the default worker contract.

- gate it through `interop.GetSharedMemorySupport()`
- only create buffers through `interop.OpenSharedBuffer(...)` when `CanUseSharedMemory` is true
- browsers only expose this path when the page is cross-origin-isolated; in practice that usually means the app shell and worker responses keep `Cross-Origin-Opener-Policy: same-origin` plus `Cross-Origin-Embedder-Policy: require-corp` or an equivalent embedder policy in place
- when shared memory is unavailable, keep the feature on the normal message-passing worker path through `worker.Request(...)`, `worker.Post(...)`, or `MessagePort` side channels
- use shared memory for mutable coordination state or larger buffers that benefit from in-place updates, and keep readiness, progress, errors, and user-visible results on the normal worker message path
- the shared-memory surface now includes byte copies, int32 `Atomics` read or write helpers, worker-safe `WaitInt32(...)`, and `NotifyInt32(...)` wakeups on `SharedBuffer`
- `WaitInt32(...)` is intentionally worker-only; calling it on the main-thread path returns unavailable instead of risking a UI-thread stall

`WorkerOptions` currently supports:

- `URL`: worker entrypoint URL
- `Name`: optional worker name
- `Type`: `classic` or `module`
- `Ready`: wait for a startup handshake before returning
- `ReadyTimeout`: optional timeout for the ready handshake

`GoWASMWorkerOptions` currently supports:

- `RuntimeURL`: URL for `wasm_exec.js`
- `WASMURL`: URL for the worker-specific Go WASM binary
- `Name`: optional worker name
- `Ready`: wait for a startup handshake before returning
- `ReadyTimeout`: optional timeout for the ready handshake

`WorkerPoolOptions` currently supports:

- `Size`: fixed number of workers kept in the pool
- `QueueLimit`: number of queued requests allowed beyond the running worker count
- `OpenWorker`: callback that constructs one worker handle for the pool

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
- startup handshake should post `phase: "ready"` when `WorkerOptions.Ready` or `GoWASMWorkerOptions.Ready` is enabled

For worker binaries written in Go, call `scope.Ready("bootstrap")` from `interop.GetWorkerScope()` once the worker has installed its handlers.

Untyped event streams can still use `worker.Post(...)` and `worker.Subscribe(...)`, but the envelope above is the supported contract for typed request or progress flows.

When a worker flow needs a dedicated duplex side channel instead of the shared worker event stream:

- create a linked pair through `interop.OpenMessageChannel()`
- pass one endpoint into the worker with `worker.PostPorts(...)` or back to the main thread with `scope.PostPorts(...)`
- use `MessagePort.Post(...)`, `MessagePort.PostPorts(...)`, and `SubscribeDecodedMessagePort[T](...)` on the handed-off port for the custom stream

This is the intended escape hatch for richer topologies such as worker-owned substreams, multiplexed jobs, or one long-lived coordination pipe without turning the default worker envelope into a framework RPC stack.

## Lifecycle

Worker lifecycle is explicit.

- `OpenWorker(...)` and `OpenGoWASMWorker(...)` create the dedicated worker and optionally wait for `phase: "ready"`
- `OpenWorkerPool(...)` creates a fixed worker set, rejects new work after `Drain(...)` or `Close()`, and reuses those worker handles across requests
- if one pooled worker becomes disposed during a request, the pool attempts to open a replacement automatically; if that repair fails, the pool closes and later requests fail fast instead of silently shrinking concurrency
- `OpenMessageChannel(...)` creates two entangled `MessagePort` endpoints that can stay on one side or be handed off across a worker boundary
- `Terminate()` stops the current worker instance and leaves the handle inactive until `Restart(...)`
- `Restart(...)` terminates any current instance, creates a fresh worker, and reapplies the startup handshake rules
- `Subscribe(...)` and `SubscribeDecodedWorker(...)` are tied to the currently active worker instance; recreate long-lived subscriptions after a restart
- `MessagePort` ownership is explicit; the owner that receives or creates a port should close it when the custom side channel is no longer needed
- `pool.Drain(ctx)` waits for already accepted work to finish before terminating the worker set, while `pool.Close()` terminates the pool immediately and causes queued or active requests to fail promptly

For UI ownership, keep the worker handle scoped to the route, component, or service that owns the background job. Cleanup should terminate the worker or at least cancel any in-flight request contexts when the owner unmounts.

## Cancellation, Timeouts, And Backpressure

`worker.Request(...)` is context-driven.

- `context.WithCancel(...)` cancels the wait path and returns `interop` error code `cancelled`
- `context.WithTimeout(...)` or deadlines return code `timeout`
- late worker responses are ignored because the request listeners are removed once the request completes, times out, or is cancelled

Backpressure is now available through explicit pool sizing and queue limits:

- each direct `worker.Request(...)` call is still independent
- `OpenWorkerPool(...)` provides fixed-size scheduling and bounded queueing for request-style worker jobs
- once the pool is full, later requests wait for admission until capacity frees or their request context is cancelled or times out
- apps should still debounce noisy UI input and choose pool sizes or queue limits that match the worker protocol and hardware budget
- if a worker protocol only supports one active job at a time, open a pool with `Size: 1` and a deliberate `QueueLimit` instead of hand-rolling the scheduler

## Serialization Boundaries

Worker payloads should stay structured-clone friendly and, for typed decoding, JSON-shaped.

- preferred: strings, booleans, numbers, arrays, slices, maps, and structs
- acceptable with care: byte slices or large arrays, as long as the worker script expects cloneable data
- first-class today: `MessagePort` ownership transfer through `worker.PostPorts(...)`, `scope.PostPorts(...)`, and `port.PostPorts(...)`, plus `SharedBuffer` and `[]byte` values posted directly, used as `WorkerMessage.Payload`, or nested inside documented string-keyed map, slice, array, and exported struct payload graphs
- current limit: nested `SharedBuffer` or `[]byte` transport is documented for acyclic JSON-shaped payload graphs only; non-string-keyed maps, cyclic graphs, and opaque host objects remain outside the structured-conversion contract
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
