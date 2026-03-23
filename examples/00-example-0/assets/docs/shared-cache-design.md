# Shared Cache Design

This page documents the current contract for `fetch.UseCachedResource[T]` and related shared-cache behavior.

Use it when a route or component needs shared async data, optimistic updates, or reuse across multiple subscribers.

## Current Status

- The shared fetch-cache surface is already shipped, including key-based reuse, route-loader interoperability, SSR bootstrap restore, optimistic local updates, and opt-in durable persistence.
- The current durable path is IndexedDB-first through `interop.OpenPersistentStore(...)` with explicit fallback storage behavior.
- This page defines the current cache contract; open work remains mostly around policy extensions such as automatic sweeping or broader eviction heuristics.

## Current Shipped Surface

The public shared-cache surface today is:

- `fetch.UseCachedResource[T](key, loader, options...)`
- `fetch.LoadCached[T](ctx, key, loader, options...)`
- `fetch.CacheOptions{StaleAfter: ..., MaxAge: ..., DisposeAfter: ..., Persist: ...}`
- `fetch.ConfigurePersistentCache(fetch.PersistentCacheOptions{...})`
- `fetch.RestoreCacheBootstrap(ui.SSRBootstrap)`
- `fetch.InspectCachedResources()`
- `CachedResource.Get()`
- `CachedResource.Reload()`
- `CachedResource.Cancel()`
- `CachedResource.Invalidate()`
- `CachedResource.Dispose()`
- `CachedResource.Set(...)`
- `CachedResource.Update(...)`
- `fetch.InvalidateResource(key)`
- `fetch.DisposeResource(key)`
- `fetch.SweepCachedResources()`

Current shipped behavior already includes:

- shared cache state by key
- stale-aware reloads
- imperative cache reuse from route loaders or other non-hook code through `LoadCached`
- optimistic replacement and optimistic update helpers
- in-flight request deduplication for the same key
- hard expiry through `MaxAge`
- idle disposal through `DisposeAfter`
- opt-in durable restore and write-through persistence through `CacheOptions.Persist`
- explicit disposal through `Dispose()` or `DisposeResource(...)`
- SSR bootstrap seeding through `ui.SSRBootstrap.Data["fetchCache"]` plus `fetch.RestoreCacheBootstrap(...)`
- cache inspection for devtools and diagnostics through `fetch.InspectCachedResources()`

## Current Boundary

- Shipped: shared cache state by normalized key, stale-aware reload, durable opt-in persistence, SSR bootstrap restore, optimistic local writes, and explicit inspection or disposal helpers.
- Not shipped: automatic global eviction loops, rollback transactions for optimistic mutations, or hidden persistence of arbitrary browser objects.
- Cache persistence is for reconstructible JSON-shaped client data, not for secrets or server-only policy decisions.

## Cache Key Normalization Rules

Shared-cache identity should be deterministic and application-owned.

Rules:

- keys must be stable strings
- the key should include every input that changes the logical payload
- when the payload depends on path params, query values, locale, auth scope, or user-selected filters, those inputs belong in the key
- do not rely on map iteration order when constructing keys
- normalize empty or default query values so semantically identical queries produce the same key
- use one key for list data and a separate key for detail data unless both truly share the same payload shape
- if an app already has a route-loader identity, reuse that same normalized identity instead of inventing a second cache namespace

Recommended pattern:

```go
key := "products:list?locale=" + locale + "&q=" + url.QueryEscape(strings.TrimSpace(query))
resource := fetch.UseCachedResource(key, loader)
```

## Freshness, Eviction, And Disposal

Current shipped freshness and lifecycle policy is:

- `CacheOptions.StaleAfter`
- `CacheOptions.MaxAge`
- `CacheOptions.DisposeAfter`
- `CacheOptions.Persist`

Current behavior:

- when `StaleAfter` is unset, the cache stays fresh until explicit invalidation or reload
- when `StaleAfter` expires, the next subscriber or reload path may revalidate the key
- stale ready data remains visible during background reload
- when `MaxAge` expires, the ready snapshot is cleared and the next reader reloads cold
- when `DisposeAfter` expires, `SweepCachedResources()` or the next access clears the entry
- when `Persist` is enabled, ready values are written through to durable browser storage and a later session restores that same key before a cold load begins
- applications may explicitly clear one entry with `DisposeResource(key)` or `resource.Dispose()`

Still open work:

- max-entry or max-memory policies
- automatic background sweep scheduling for long-lived apps

Today, lifecycle cleanup is explicit and deterministic rather than hidden behind an always-on eviction loop.

## Durable Persistence

Shared cached resources may now persist across reloads on an opt-in basis.

Current shipped model:

- add `Persist: true` to `CacheOptions` when a key should survive reloads
- persisted values restore before the first cold load for that key
- IndexedDB is the first durable backend through `interop.OpenPersistentStore(...)`
- the default fallback backend is `localStorage` when IndexedDB is unavailable
- `DisposeResource(key)` removes both the in-memory entry and the durable stored value
- `MaxAge` remains a hard expiry boundary for persisted entries as well as in-memory entries

Global configuration is available through:

```go
fetch.ConfigurePersistentCache(fetch.PersistentCacheOptions{
  DatabaseName: "atlas-cache",
  StoreName:    "shared-fetch",
})
```

Recommended rules:

- enable persistence only for keys whose payloads are intentionally JSON-shaped and reasonably small
- prefer one durable key per normalized cache identity instead of ad hoc mixed payload buckets
- use `MaxAge` when persisted results must not survive indefinitely across sessions
- use explicit disposal on logout or trust-boundary changes when cached data should be purged immediately
- treat durable read-cache entries as reconstructible copies, not as the only source of truth; if a logout, user switch, or privilege narrowing event occurs, purge them immediately instead of trying to salvage stale visibility
- keep secrets, raw authorization context, and server-only policy results out of persisted cache payloads even when the same data is already cached in memory transiently

## Request Deduplication

Concurrent subscribers for the same key should share one in-flight load.

Current shipped behavior:

- if one `UseCachedResource` call starts a load for key `K`, another subscriber for `K` reuses that in-flight work
- repeated `Reload()` calls while the same key is already pending do not stampede the loader
- the shared snapshot updates once the authoritative load completes

This behavior is already covered by `fetch/cache.go` and wasm tests.

## Mutation And Optimistic Update API

Current optimistic-write surface is:

- `Set(value)` for full optimistic replacement
- `Update(func(prev T) T)` for optimistic transforms
- `Invalidate()` when the optimistic value should be revalidated against authoritative data

Intended usage model:

1. apply a local optimistic change with `Set` or `Update`
2. perform the real mutation through `Fetch`, `Upload`, form submission, or another app-owned write path
3. invalidate or reload the affected keys after the authoritative result arrives

Current contract:

- optimistic cache writes are local UX state, not proof that the server accepted a mutation
- list and detail keys must both be updated or invalidated when they represent the same entity in different shapes
- mutation rollback is application-owned today; the framework does not yet provide a first-class mutation transaction or rollback primitive

## Route Loader Interoperability

Current shipped model:

- route loaders may call `fetch.LoadCached[T](ctx, key, loader, options...)`
- component-level readers may use the same normalized key through `fetch.UseCachedResource[T](key, loader, options...)`
- invalidation or disposal by key affects both paths because they share one registry entry and one atom-backed snapshot

Recommended pattern:

```go
func userLoader(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
    key := "user:" + routeCtx.Params.Get("id")
    user, err := fetch.LoadCached(ctx, key, func(ctx context.Context) (User, error) {
        return loadUser(ctx, routeCtx.Params.Get("id"))
    }, fetch.CacheOptions{StaleAfter: 30 * time.Second})
    if err != nil {
        return nil, err
    }
    return router.Attrs{"cacheKey": key, "user": user}, nil
}
```

Then a child component can read the same key through `UseCachedResource` without forcing a second fetch path.

## SSR Bootstrap And Resume

Current shipped model:

- servers may embed cache entries under `ui.SSRBootstrap.Data["fetchCache"]`
- the browser restores those entries with `fetch.RestoreCacheBootstrap(payload)`
- `UseCachedResource[T]` then starts from the seeded value instead of forcing a cold first read

Bootstrap shape:

```json
{
  "fetchCache": {
    "entries": [
      {
        "key": "products:list",
        "value": {"items": ["SSR item"]},
        "updatedAt": "2026-03-18T19:00:00Z",
        "resumePolicy": "trust-once",
        "staleAfter": 60000000000
      }
    ]
  }
}
```

`staleAfter` is serialized as a Go duration in nanoseconds, matching JSON round-trips for `time.Duration`.

## Revalidation On Resume Policies

Bootstrap entries now support explicit first-client resume behavior:

- `trust-once`
  The seeded value is treated as ready and fresh until normal invalidation or `StaleAfter` says otherwise.
- `stale-while-revalidate`
  The seeded value remains visible, and the first client read marks it stale only when the embedded timestamp is already older than the bootstrap entry's `staleAfter` window or when that timestamp is absent.
- `always-refetch`
  The seeded value remains visible for first paint, but the first client subscriber still forces an authoritative load regardless of the embedded timestamp.

These policies are explicit because startup speed and freshness are different product choices.

## Serialization Safety

Cached values should stay JSON-shaped when they are candidates for route-loader reuse or SSR bootstrap.

Recommended rules:

- strings, numbers, booleans, arrays, maps, and plain structs are safe defaults
- avoid storing opaque browser handles, callbacks, or environment-specific objects in shared cached values
- keep secrets and server-only authorization context out of any value that might later become bootstrap data
- keep payloads intentionally small so the first HTML response does not bloat for one oversized cache entry
- use one bootstrap entry per normalized cache key instead of embedding ad hoc mixed payloads
- prefer values that round-trip through standard JSON without custom codecs unless the server and client both deliberately own a stronger transport

## Devtools Visibility

Current shipped inspection surface is `fetch.InspectCachedResources()`.

The devtools panel now surfaces:

- cache key
- ready, stale, and pending state
- subscriber count
- resume policy
- last successful update time
- last error

## Recommended Usage Today

- use `UseCachedResource[T]` when the same query should be reused across components
- keep keys explicit and deterministic
- use `StaleAfter` for freshness expectations
- add `MaxAge` when stale-but-resident entries should eventually expire hard
- add `DisposeAfter` when idle entries should not live forever
- use `Set` and `Update` only for deliberate optimistic UI
- revalidate after authoritative writes
- reuse the same normalized key from route loaders and component readers when the payload shape truly matches
- call `DisposeResource` or `SweepCachedResources` in long-lived surfaces that need deterministic cleanup
- when SSR is already producing first-route data, seed the same normalized cache key into `ui.SSRBootstrap.Data["fetchCache"]` and restore it before hydration
- enable `Persist` only for keys whose durable replay across reloads is desirable and safe
