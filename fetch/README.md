# GWC | Fetch Library

# GoWebComponents (GWC)

## High-Level Overview

The `fetch` library handles async data loading, typed resources, shared cache behavior, uploads, bounded realtime browser streams, and offline mutation queue flows in GWC applications.

## Public APIs

### `github.com/monstercameron/GoWebComponents/fetch` (`package fetch`)
- Functions: `AsMutationConflictError`, `Cancel`, `Clear`, `Close`, `ConfigurePersistentCache`, `DecodeJSON`, `Dispose`, `DisposeResource`, `Done`, `Enqueue`, `Error`, `Fetch`, `Get`, `GetMutationConflict`, `InspectCachedResources`, `Invalidate`, `InvalidateResource`, `IsMutationConflict`, `List`, `LoadCached`, `NewMutationConflict`, `Open`, `OpenMutationQueue`, `Refetch`, `Reload`, `Remove`, `Replay`, `ReplayWithOptions`, `RestoreCacheBootstrap`, `ReturnChannel`, `Send`, `Set`, `SweepCachedResources`, `Text`, `Unwrap`, `Update`, `Upload`, `UseCachedResource`, `UseEventSource`, `UseFetch`, `UseResource`, `UseWebSocket`
- Types: `AsyncResource`, `CacheBootstrap`, `CacheBootstrapEntry`, `CacheOptions`, `CacheResumePolicy`, `CachedResource`, `CachedResourceInspection`, `CachedResourceState`, `EventSource`, `EventSourceOptions`, `HTTPError`, `MultipartBody`, `MultipartFile`, `MutationConflict`, `MutationConflictError`, `MutationConflictHandler`, `MutationConflictResolution`, `MutationDraft`, `MutationExecutor`, `MutationQueue`, `MutationQueueOptions`, `MutationReplayOptions`, `MutationReplayReport`, `MutationResolutionAction`, `MutationState`, `Options`, `PersistentCacheOptions`, `QueuedMutation`, `RealtimeError`, `RealtimeMessage`, `RealtimeState`, `RealtimeStatus`, `Resource`, `ResourceState`, `Result`, `State`, `UploadUpdate`, `WebSocket`, `WebSocketOptions`
- Variables: _none_
- Constants: `CacheBootstrapDataKey`, `CacheResumeAlwaysRefetch`, `CacheResumeStaleWhileRevalidate`, `CacheResumeTrustOnce`, `MutationDead`, `MutationQueued`, `MutationResolutionDead`, `MutationResolutionRemove`, `MutationResolutionReplace`, `MutationResolutionRetry`, `MutationRetrying`, `RealtimeClosed`, `RealtimeConnecting`, `RealtimeIdle`, `RealtimeOpen`, `RealtimeReconnecting`, `RealtimeUnsupported`

## Subfiles And Purpose

- `cache.go` - Core implementation for cache
- `doc.go` - Package-level Go documentation
- `example_test.go` - Tests for example behavior
- `fetch.go` - Core implementation for fetch
- `fetch_wasm_test.go` - Tests for fetch_wasm behavior
- `mutation_queue.go` - Core implementation for mutation_queue
- `mutation_queue_wasm_test.go` - Tests for mutation_queue_wasm behavior
- `realtime.go` - Shared bounded WebSocket and EventSource hook state
- `realtime_native.go` - Native unsupported realtime stubs
- `realtime_wasm.go` - Browser WebSocket and EventSource transports
- `README.md` - Folder-level documentation

## Realtime Hooks

`UseWebSocket` and `UseEventSource` are browser hooks for small realtime surfaces that need bounded in-memory state. Both hooks keep only the newest `MaxMessages` and `MaxErrors`, reconnect with capped exponential backoff, and expose a `RealtimeState` snapshot through `Get()`.

On non-browser builds the hooks compile and return `RealtimeUnsupported` with a descriptive error. This keeps SSR, native tests, and tooling builds safe without pretending a browser transport exists.

```go
socket := fetch.UseWebSocket("wss://example.test/live", fetch.WebSocketOptions{
    MaxMessages:       16,
    MaxReconnects:     3,
    HeartbeatInterval: 30 * time.Second,
})

state := socket.Get()
if state.Open {
    _ = socket.Send("ping")
}
```

## File Map

```text
fetch/
|-- cache.go
|-- doc.go
|-- example_test.go
|-- fetch.go
|-- fetch_wasm_test.go
|-- mutation_queue.go
|-- mutation_queue_wasm_test.go
|-- realtime.go
|-- realtime_native.go
|-- realtime_wasm.go
\-- README.md
```



