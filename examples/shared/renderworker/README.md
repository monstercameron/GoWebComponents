# Generic Render Worker Methodology

This package provides a small, explicit API for CPU-heavy UI work that should run off the main thread.

## 1) Open a fixed worker pool

Use `BuildRenderWorkerPool` from the app/runtime side with explicit size and queue settings.

## 2) Register typed handlers in the worker

Use `SetRenderWorkerDecodedHandler` from the worker side:

- request name
- typed request/response function
- `DecodedHandlerOptions` guardrails

Guardrails:

- `ShouldRecoverPanic` isolates crashes from user code.
- `GetRequestTimeout` bounds long-running user logic.

## 3) Keep custom experiments isolated

Put local or product-specific handlers in one function (for example `setBackgroundWorkerCustomHandlers`) so experimentation does not leak into core worker wiring.

## 4) Dispatch from the app by request name

The app uses `interop.RequestWorkerDecoded` against either one worker or `WorkerPool`.
Start with one request shape, then add request names as new tasks are introduced.

## 5) Multi-worker lane optimization primitives

This package includes reusable lane-dispatch optimizations ported from Example 201:

- `BuildRenderWorkerAdaptiveChunkBounds` for weighted contiguous chunk partitioning
- `BuildRenderWorkerChunkPlans` for concrete chunk plans with estimated work weights
- `BuildRenderWorkerLanePlans` for weighted greedy worker-lane balancing
- `RequestRenderWorkerChunkBatches` for one-request-per-lane batched fanout with chunk-index aggregation
- `BuildRenderWorkerChunkLocalIndexes` for dirty-index localization inside one chunk payload
- `RenderWorkerGenerationTracker` for request-generation stale-drop guards

## 6) Fail soft

If pooled worker requests fail, prefer a documented fallback path (for example sync render on main thread) so the UI remains usable.
