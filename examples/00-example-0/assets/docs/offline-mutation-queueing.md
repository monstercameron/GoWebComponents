# Offline Mutation Queueing

This page documents the current contract for first-class offline mutation support in `fetch`.

Use it when a browser app needs to queue writes locally, survive reloads, and replay those writes later without hand-rolling storage, deduplication, and retry timing.

## At A Glance

- the queue surface is already shipped and durable today
- queue persistence is IndexedDB-first with `localStorage` fallback
- replay remains application-owned so auth headers, merge policy, and real HTTP execution stay under app control
- queue state is visible through replay reports, structured framework logs, and PWA diagnostics snapshots

## Quick API Chooser

- use `fetch.OpenMutationQueue(...)` when browser-originated writes must survive reloads or offline periods
- use `queue.Replay(...)` when normal retry and dead-letter behavior is enough
- use `queue.ReplayWithOptions(...)` when version or conflict handling needs an explicit `ConflictHandler`
- use `fetch.NewMutationConflict(...)` and `fetch.GetMutationConflict(...)` when the executor must signal or inspect authoritative version conflicts
- use `pwa.MutationQueueDiagnosticsSource(...)` plus `pwa.InspectDiagnostics(...)` when the queue should participate in a broader offline diagnostics view

## Current Shipped Slice

Today the first-class queue contract covers:

- durable enqueue and reopen behavior
- insertion-order replay
- deduplication through `DedupKey`
- exponential backoff with deferred retry scheduling
- terminal dead-letter persistence after retry exhaustion
- explicit conflict handling and resolution paths
- structured queue lifecycle logs and diagnostics-friendly replay reports

Today it does not claim:

- automatic connectivity listeners
- automatic service-worker registration or automatic sync orchestration
- built-in optimistic rollback or merge semantics
- encrypted persistence for sensitive payloads

## Example Shape

This is the current intended lifecycle: queue locally, replay through an authoritative executor, then expose queue state through structured diagnostics.

```go
package app

import (
    "context"
    "fmt"
    "time"

    "github.com/atdiar/particleui/fetch"
    "github.com/atdiar/particleui/pwa"
)

func syncOrders(ctx context.Context) error {
    queue, err := fetch.OpenMutationQueue(fetch.MutationQueueOptions{
        StorageKey:  "orders:offline",
        MaxAttempts: 4,
        BaseDelay:   2 * time.Second,
        MaxDelay:    2 * time.Minute,
    })
    if err != nil {
        return err
    }

    _, err = queue.Enqueue(fetch.MutationDraft{
        Kind:     "order.submit",
        DedupKey: "order:draft:123",
        Method:   "POST",
        URL:      "/api/orders",
        Body:     map[string]any{"items": []string{"sku-1"}},
    })
    if err != nil {
        return err
    }

    report, err := queue.Replay(ctx, func(ctx context.Context, mutation fetch.QueuedMutation) error {
        result := <-fetch.Fetch(mutation.URL, fetch.Options{Method: mutation.Method, Body: mutation.Body})
        return result.Err
    })
    if err != nil {
        return err
    }

    snapshot, err := pwa.InspectDiagnostics(ctx, pwa.DiagnosticsOptions{
        OfflineQueue: pwa.MutationQueueDiagnosticsSource(&queue),
    })
    if err == nil {
        fmt.Println(report.DeadLetters, snapshot.OfflineQueue.DeadLetters)
    }
    return nil
}
```

## Current Shipped Surface

The public surface today is:

- `fetch.OpenMutationQueue(options...)`
- `fetch.MutationQueue.Enqueue(draft)`
- `fetch.MutationQueue.List()`
- `fetch.MutationQueue.Remove(id)`
- `fetch.MutationQueue.Clear()`
- `fetch.MutationQueue.Replay(ctx, executor)`
- `fetch.MutationQueue.ReplayWithOptions(ctx, executor, options...)`
- `fetch.MutationDraft`
- `fetch.QueuedMutation`
- `fetch.MutationQueueOptions`
- `fetch.MutationReplayReport`
- `fetch.MutationConflict`
- `fetch.MutationConflictError`
- `fetch.NewMutationConflict(err, conflict)`
- `fetch.IsMutationConflict(err)`
- `fetch.AsMutationConflictError(err)`
- `fetch.GetMutationConflict(err)`
- `fetch.MutationConflictResolution`
- `fetch.MutationReplayOptions`

Current shipped behavior includes:

- local persistence through an IndexedDB-first durable store with `localStorage` fallback
- replay in insertion order
- duplicate suppression through `DedupKey`
- exponential backoff for retryable failures
- terminal dead-letter state after the configured retry budget is exhausted
- application-owned replay execution, so the app keeps control over the real HTTP call and auth headers
- explicit conflict signaling plus app-owned conflict resolution policies for dead-letter, retry, removal, or requeue-with-merge
- an explicit service-worker Background Sync scheduling pattern through `pwa.ServiceWorkerRegistration.RegisterSync(...)`, with manual replay fallback when the browser does not expose one-shot sync

## Current Inspection Surface

Beyond `MutationReplayReport`, the queue already feeds two useful operational surfaces:

- structured framework logs such as `mutation queued for replay`, `mutation replay succeeded`, and dead-letter transitions
- `pwa.InspectDiagnostics(...)` snapshots when adapted through `pwa.MutationQueueDiagnosticsSource(...)`

That means queue failures and dead letters do not need to live only in raw console output or ad hoc app logs.

## Scope Of First-Class Support

The first shipped scope is intentionally narrow:

- queue browser-originated writes locally
- persist them across reloads
- replay them later through an application-owned executor
- expose retry and terminal-failure state so the UI can respond deliberately

The current scope does not yet include:

- automatic service-worker registration or automatic Background Sync orchestration
- automatic connectivity listeners
- built-in domain-specific merge logic after reconnect
- built-in optimistic transaction or rollback primitives
- encrypted local storage of sensitive payloads
- browser file or stream persistence inside the queue

This boundary is deliberate. The framework now owns the durable queue mechanics, while applications still own domain-specific replay policy.

## Persistent Queue Model

Open a queue with:

```go
queue, err := fetch.OpenMutationQueue(fetch.MutationQueueOptions{
    StorageKey:  "orders:offline",
    MaxAttempts: 4,
    BaseDelay:   2 * time.Second,
    MaxDelay:    2 * time.Minute,
})
```

Then enqueue writes:

```go
entry, err := queue.Enqueue(fetch.MutationDraft{
    Kind:     "order.submit",
    DedupKey: "order:draft:123",
    Method:   "POST",
    URL:      "/api/orders",
    Headers:  map[string]string{"Content-Type": "application/json"},
    Body:     map[string]any{"items": cartItems},
    Metadata: map[string]string{"schema": "v1"},
})
```

Current queue rules:

- entries are appended in the order they are enqueued
- `Replay(...)` processes entries in that same order
- successful replay removes the entry from storage
- failed replay increments `Attempts`, records `LastError`, and schedules `NextAttemptAt`
- once `Attempts >= MaxAttempts`, the entry becomes `State == "dead"` and remains visible until the app removes it or clears the queue
- default durable storage is `interop.OpenPersistentStore(...)` with IndexedDB first and `localStorage` fallback; callers may override the durable store through `MutationQueueOptions.StoreResolver` or the fallback storage path through `MutationQueueOptions.StorageResolver`

## Replay And Retry Semantics

Replay is explicit and application-driven:

```go
report, err := queue.Replay(ctx, func(ctx context.Context, mutation fetch.QueuedMutation) error {
    result := <-fetch.Fetch(mutation.URL, fetch.Options{
        Method: mutation.Method,
        Headers: map[string]any{
            "Content-Type": mutation.Headers["Content-Type"],
        },
        Body: mutation.Body,
    })
    if result.Err != nil {
        return result.Err
    }
    return nil
})
```

Current shipped retry behavior:

- `DedupKey` suppresses enqueuing a second non-dead entry for the same logical write
- the first failure moves the entry to `retrying`
- retry delay grows exponentially from `BaseDelay` up to `MaxDelay`
- entries whose `NextAttemptAt` is still in the future are deferred, not executed
- terminal failures become `dead` and stay persisted for user-visible remediation
- if `ctx` is canceled during replay, already-processed items stay committed and untouched remaining items stay queued

`MutationReplayReport` summarizes one pass:

- `Succeeded`
- `Deferred`
- `Retried`
- `DeadLetters`
- `Conflicts`
- `Resolved`
- `Remaining`

## Background Sync And Conflict Resolution

The queue still keeps replay explicit, but the shipped first-party pattern now covers the two most common coordination gaps around reconnect:

1. register one-shot Background Sync from an app-owned service worker registration when the browser supports it
2. fall back to a visible manual replay action when it does not
3. let the replay executor signal a conflict through `fetch.NewMutationConflict(...)`
4. resolve that conflict through `ReplayWithOptions(...)` and an application-owned `ConflictHandler`
5. inspect conflict details later through `fetch.GetMutationConflict(err)` when UI or telemetry needs structured branching

Typical flow:

```go
registration, err := pwa.RegisterServiceWorker(ctx, pwa.ServiceWorkerOptions{
    URL:   "/sw.js",
    Scope: "/app/",
})
if err == nil && registration.BackgroundSyncCapabilities().OneShot {
    _ = registration.RegisterSync(ctx, "orders-offline-replay")
}

report, err := queue.ReplayWithOptions(ctx, executor, fetch.MutationReplayOptions{
    ConflictHandler: func(ctx context.Context, mutation fetch.QueuedMutation, conflict fetch.MutationConflict) (fetch.MutationConflictResolution, error) {
        return fetch.MutationConflictResolution{
            Action: fetch.MutationResolutionReplace,
            Draft: fetch.MutationDraft{
                Metadata: map[string]string{"revision": conflict.RemoteVersion, "resolved": conflict.RemoteVersion},
            },
            Message: "rebased onto remote revision",
        }, nil
    },
})
```

This keeps queue persistence, retry timing, and durable visibility inside the framework surface, while still leaving merge semantics and authoritative retry policy under app control.

## Optimistic Updates And Rollback

The queue stores the authoritative mutation payload. It does not automatically mutate local UI state.

Current recommended model:

1. apply optimistic UI locally through atom state or `fetch.UseCachedResource[T]` with `Set`, `Update`, or `Invalidate`
2. enqueue the authoritative write payload in the mutation queue
3. when replay eventually succeeds, invalidate or reload the affected cache keys
4. when replay reaches `dead`, either roll back the optimistic UI or surface a conflict and recovery action to the user

Current contract:

- optimistic UI is application-owned
- rollback is application-owned
- conflict resolution is application-owned
- the queue is responsible for durable storage, ordering, retry timing, and visibility into terminal failure state

## Serialization And Security Boundaries

Queued payloads should stay intentionally simple and safe to persist.

Recommended rules:

- use JSON-shaped request bodies and metadata
- use `Headers` only for replay-relevant headers that are safe to persist
- do not store bearer tokens, session secrets, CSRF secrets, or other sensitive credentials in the queue
- prefer deriving volatile auth headers at replay time inside the executor instead of persisting them
- do not store `js.Value`, `File`, `Blob`, streams, functions, or other browser-native handles in `Body`
- include an application schema or migration hint in `Metadata` when queued payloads may span app upgrades
- keep queue entries small because browser storage is limited and cleartext at rest unless the application adds its own encryption layer
- clear queues on logout, user switch, or trust-boundary changes unless the queued drafts are explicitly designed to survive that account transition
- if queued writes may contain regulated or highly sensitive business data, define an application-owned retention window and purge path instead of letting dead letters survive forever by accident

The queue currently serializes through standard JSON. If a payload cannot round-trip cleanly through JSON, it is outside the supported first-class boundary.

## Recommended Usage Today

- use one queue per logical workflow or trust boundary instead of one giant mixed queue
- choose explicit `StorageKey` values so different apps or environments do not collide
- use `DedupKey` for draft-save or submit-once semantics
- keep replay executors authoritative and idempotent where possible
- invalidate related shared-cache keys after successful replay
- surface `dead` entries in the UI so users can retry, edit, or discard them intentionally

## Review Checklist

- does the app keep authoritative HTTP execution and volatile auth headers inside the replay executor
- are `DedupKey`, retry timing, and dead-letter handling chosen deliberately per workflow
- are queued payloads JSON-shaped and free of tokens, secrets, and browser-native handles
- does the UI surface `dead` or conflict entries instead of silently leaving them persisted forever
- does the diagnostics story use replay reports, framework logs, or `pwa.InspectDiagnostics(...)` instead of relying only on raw console output

## Still Open Work

The queue-specific first-class blockers for this slice are closed.

Broader non-goals remain automatic connectivity listeners, built-in optimistic rollback primitives, and encrypted local storage of sensitive payloads.

For the broader PWA and offline-cache integration boundary around that queue, see [PWA.md](pwa-and-offline-app-support.md).
