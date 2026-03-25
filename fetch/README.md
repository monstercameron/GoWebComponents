# GWC | Fetch Library

# GoWebComponents (GWC)

## High-Level Overview

The `fetch` library handles async data loading, typed resources, shared cache behavior, uploads, and offline mutation queue flows in GWC applications.

## Public APIs

### `github.com/monstercameron/GoWebComponents/fetch` (`package fetch`)
- Functions: `AsMutationConflictError`, `Cancel`, `Clear`, `Close`, `ConfigurePersistentCache`, `DecodeJSON`, `Dispose`, `DisposeResource`, `Done`, `Enqueue`, `Error`, `Fetch`, `Get`, `GetMutationConflict`, `InspectCachedResources`, `Invalidate`, `InvalidateResource`, `IsMutationConflict`, `List`, `LoadCached`, `NewMutationConflict`, `OpenMutationQueue`, `Refetch`, `Reload`, `Remove`, `Replay`, `ReplayWithOptions`, `RestoreCacheBootstrap`, `ReturnChannel`, `Set`, `SweepCachedResources`, `Text`, `Unwrap`, `Update`, `Upload`, `UseCachedResource`, `UseFetch`, `UseResource`
- Types: `AsyncResource`, `CacheBootstrap`, `CacheBootstrapEntry`, `CacheOptions`, `CacheResumePolicy`, `CachedResource`, `CachedResourceInspection`, `CachedResourceState`, `HTTPError`, `MultipartBody`, `MultipartFile`, `MutationConflict`, `MutationConflictError`, `MutationConflictHandler`, `MutationConflictResolution`, `MutationDraft`, `MutationExecutor`, `MutationQueue`, `MutationQueueOptions`, `MutationReplayOptions`, `MutationReplayReport`, `MutationResolutionAction`, `MutationState`, `Options`, `PersistentCacheOptions`, `QueuedMutation`, `Resource`, `ResourceState`, `Result`, `State`, `UploadUpdate`
- Variables: _none_
- Constants: `CacheBootstrapDataKey`, `CacheResumeAlwaysRefetch`, `CacheResumeStaleWhileRevalidate`, `CacheResumeTrustOnce`, `MutationDead`, `MutationQueued`, `MutationResolutionDead`, `MutationResolutionRemove`, `MutationResolutionReplace`, `MutationResolutionRetry`, `MutationRetrying`

## Subfiles And Purpose

- `cache.go` - Core implementation for cache
- `doc.go` - Package-level Go documentation
- `example_test.go` - Tests for example behavior
- `fetch.go` - Core implementation for fetch
- `fetch_wasm_test.go` - Tests for fetch_wasm behavior
- `mutation_queue.go` - Core implementation for mutation_queue
- `mutation_queue_wasm_test.go` - Tests for mutation_queue_wasm behavior
- `README.md` - Folder-level documentation

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
\-- README.md
```



