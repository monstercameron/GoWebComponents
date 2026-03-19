# Offline Mutation Queueing

This page documents the current contract for first-class offline mutation support in `fetch`.

Use it when a browser app needs to queue writes locally, survive reloads, and replay those writes later without hand-rolling storage, deduplication, and retry timing.

## Current Shipped Surface

The public surface today is:

- `fetch.OpenMutationQueue(options...)`
- `fetch.MutationQueue.Enqueue(draft)`
- `fetch.MutationQueue.List()`
- `fetch.MutationQueue.Remove(id)`
- `fetch.MutationQueue.Clear()`
- `fetch.MutationQueue.Replay(ctx, executor)`
- `fetch.MutationDraft`
- `fetch.QueuedMutation`
- `fetch.MutationQueueOptions`
- `fetch.MutationReplayReport`

Current shipped behavior includes:

- local persistence through browser storage, defaulting to `localStorage`
- replay in insertion order
- duplicate suppression through `DedupKey`
- exponential backoff for retryable failures
- terminal dead-letter state after the configured retry budget is exhausted
- application-owned replay execution, so the app keeps control over the real HTTP call, auth headers, and conflict handling

## Scope Of First-Class Support

The first shipped scope is intentionally narrow:

- queue browser-originated writes locally
- persist them across reloads
- replay them later through an application-owned executor
- expose retry and terminal-failure state so the UI can respond deliberately

The current scope does not yet include:

- service-worker registration or Background Sync orchestration
- automatic connectivity listeners
- built-in conflict resolution after reconnect
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
- default storage is `localStorage`; callers may override storage creation through `MutationQueueOptions.StorageResolver`

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
- `Remaining`

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

The queue currently serializes through standard JSON. If a payload cannot round-trip cleanly through JSON, it is outside the supported first-class boundary.

## Recommended Usage Today

- use one queue per logical workflow or trust boundary instead of one giant mixed queue
- choose explicit `StorageKey` values so different apps or environments do not collide
- use `DedupKey` for draft-save or submit-once semantics
- keep replay executors authoritative and idempotent where possible
- invalidate related shared-cache keys after successful replay
- surface `dead` entries in the UI so users can retry, edit, or discard them intentionally

## Still Open Work

The next backlog items after this shipped slice are:

- service-worker and Background Sync coordination
- example applications for offline write replay
- first-class conflict resolution helpers

For the broader PWA and offline-cache integration boundary around that queue, see [PWA.md](PWA.md).
