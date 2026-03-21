# Cross-Tab Synchronization

This page documents the current first-class model for cross-tab synchronization in GoWebComponents.

Use it when browser tabs need to exchange small state or invalidation signals without hand-rolling `BroadcastChannel` or `storage` event plumbing.

## Current Shipped Surface

The public surface today is:

- `interop.OpenCrossTabChannel(options)`
- `interop.CrossTabChannel.Publish(payload)`
- `interop.CrossTabChannel.Subscribe(handler)`
- `interop.CrossTabChannel.Close()`
- `interop.CrossTabChannel.Transport()`
- `interop.DecodeCrossTabEnvelope[T](message)`
- `interop.SubscribeDecodedCrossTab[T](channel, handler)`
- `interop.CrossTabChannelOptions`
- `interop.CrossTabEnvelope`

Current shipped behavior includes:

- named, opt-in channels rather than one global synchronization bus
- `BroadcastChannel` as the preferred browser transport when available
- automatic `localStorage` plus `storage` event fallback when `BroadcastChannel` is unavailable
- typed decode helpers for JSON-shaped payloads
- per-message metadata through `Name`, `Source`, `Sequence`, and `SentAt`

## Scope Of First-Party Support

The current first-party scope is intentionally narrow:

- cross-tab state hints
- logout or session-expiry notifications
- cache invalidation signals
- draft-state or preference updates
- single-owner offline replay coordination built on app-owned messages and leadership policy

The current scope does not yet include:

- automatic synchronization of every atom or cache key
- background conflict reconciliation
- multi-window popup orchestration
- service-worker coordination
- SSR bootstrap conflict resolution beyond the documented rules below

This boundary is deliberate. The framework now owns the transport and typed envelope shape, while applications still choose what to synchronize and how to merge it.

## Named Channels And Opt-In Scoping

Cross-tab synchronization is opt-in by channel name.

Recommended pattern:

```go
channel, err := interop.OpenCrossTabChannel(interop.CrossTabChannelOptions{
    Name: "theme",
})
```

Rules:

- use one channel per logical concern such as `theme`, `auth`, `draft:checkout`, or `cache:workspace`
- do not publish unrelated concerns into one global channel unless the payloads truly share lifecycle and merge rules
- use a custom `StorageKey` only when multiple apps or environments might otherwise collide in shared browser storage
- prefer small invalidation or intent messages over full app-state replication

This channel naming model is the current answer to scoping and filtering: synchronization is explicit per topic instead of implicit across all client state.

## Transport Model

`interop.OpenCrossTabChannel(...)` resolves transport in this order:

1. `BroadcastChannel`
2. `localStorage` write plus `storage` event fallback

Use `channel.Transport()` for diagnostics or testing if the application needs to know which transport the browser resolved.

Current transport expectations:

- message payloads should stay JSON-shaped
- the sender updates its own local state directly; cross-tab transport is for the other tabs
- `BroadcastChannel` is preferred because it avoids storage churn and intent-log cleanup
- the storage fallback exists so older or constrained environments still have a first-party synchronization path

## Message Shape

Published payloads are wrapped in a `CrossTabEnvelope`:

```go
type CrossTabEnvelope struct {
    Name     string
    Payload  any
    Source   string
    Sequence int64
    SentAt   time.Time
}
```

Applications normally publish just the payload:

```go
_ = channel.Publish(map[string]any{
    "theme": "dark",
})
```

The framework adds:

- `Name`: the logical channel name
- `Source`: a per-tab sender identifier
- `Sequence`: a per-sender monotonic sequence number
- `SentAt`: the sender timestamp

These fields exist to support diagnostics and application-level merge rules.

## Conflict Resolution And Merge Semantics

The transport does not attempt to merge application data automatically.

Current contract:

- cross-tab helpers deliver ordered messages from one sender on one named channel
- messages from different tabs are not treated as a globally serialized transaction log
- if two tabs update the same logical entity concurrently, the application must decide which revision wins
- logout, session-expiry, and cache-invalidation messages should usually be applied immediately because they intentionally narrow or clear state
- optimistic entity edits should carry application revision metadata so receivers can reject stale writes instead of blindly overwriting newer local state

Recommended merge rules:

- use last-writer-wins only for low-risk preference state such as theme or sidebar openness
- include a server revision, updated-at timestamp, or explicit version in payloads for drafts and cache-backed entity updates
- prefer invalidation messages over full object replacement when conflict risk is high

## Bootstrap And Resume Interaction

Cross-tab synchronization should not blindly overwrite fresher state restored from SSR bootstrap or persisted snapshots.

Recommended rules:

- restore SSR bootstrap or persisted snapshots first
- open cross-tab channels after the owning state is initialized
- ignore incoming messages that are older than the locally restored revision or timestamp
- for cache-backed data, prefer cross-tab invalidation messages that trigger revalidation instead of shipping full bootstrap payloads across tabs
- for auth hints, treat a narrowing change such as logout as authoritative immediately, but re-check expansive changes such as regained privileges against the app's real auth source

This remains an application merge decision; the framework transport simply preserves the envelope metadata needed to make that decision.

## Recommended Usage Today

- publish small topic-specific messages
- keep payloads JSON-shaped
- choose stable channel names with one concern per channel
- use `SubscribeDecodedCrossTab[T](...)` instead of hand-parsing untyped maps in app code
- use invalidation messages for caches and richer revisioned payloads for drafts or form-like state
- when coordinating offline replay, elect one replay owner per queue or trust boundary and have non-owner tabs observe results plus invalidate affected cache keys instead of replaying the same mutation stream twice

## Diagnostics And Example

See `examples/94-cross-tab-sync` for a concrete manual test surface covering:

- theme synchronization
- logout propagation
- cache invalidation broadcasting
- draft-state sharing
- transport and message diagnostics

## Still Open Work

The next backlog items after this shipped slice are:

- dedicated helpers for syncing atoms or caches by convention instead of only by raw channel name
- bootstrap and resume APIs that integrate more directly with higher-level framework state
