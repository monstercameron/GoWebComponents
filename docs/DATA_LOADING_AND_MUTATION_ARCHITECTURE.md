# Data Loading And Mutation Architecture

This guide defines the current recommended split for async reads, shared cached data, route loaders, explicit mutations, optimistic UI, offline replay, and revalidation.

Use it when an application is growing beyond one-off `fetch.Fetch(...)` calls and needs a consistent answer for:

- where async reads should live
- when to use route loaders versus component hooks
- when shared cache is worth it
- how mutations should update UI
- how optimistic updates and offline replay fit

## The Short Version

Use the smallest data-loading or mutation tool that matches the screen:

- `fetch.UseFetch(...)`: raw browser-style fetch state when you want the response shape directly
- `fetch.UseResource[T](...)`: typed component-owned async data
- route loaders: route-owned first-load data, params-aware reads, and SSR-aware route entry data
- `fetch.UseCachedResource[T](...)`: shared async data that several consumers or routes reuse
- `fetch.LoadCached[T](...)`: non-hook cache reuse from route loaders or imperative code
- `ui.UseForm[T]` plus explicit submit handlers: authoritative mutation path for most forms
- optimistic `CachedResource.Set(...)` or `Update(...)`: local UX improvement around a real mutation
- `fetch.OpenMutationQueue(...)`: durable offline replay for writes that must survive reloads

Do not invent a second hidden query layer on top of these primitives unless the app has a concrete need the shipped surfaces cannot cover.

## 1. Choose The Read Owner First

The first decision is not “which hook is coolest.” It is “who owns this data read.”

Choose the owner by screen shape:

- route owns the first read: use a route loader
- one component owns the read: use `fetch.UseResource[T](...)`
- several components need the same read: use `fetch.UseCachedResource[T](...)`
- you only want raw response state around one URL: use `fetch.UseFetch(...)`

That ownership decision usually matters more than the transport details.

## 2. Route Loaders

Use route loaders when the data is part of entering or re-entering a route.

Good loader cases:

- route params or query values determine the read
- the route should not render its real content until the read is resolved
- SSR or hydration should reuse the same route-owned first-load contract
- revalidation should be tied to the route lifecycle

Use `router.UseRevalidator()` when the current route should explicitly rerun its loader after a successful mutation or external refresh.

Avoid putting all async reads into route loaders. If the data is optional, deeply local, or only needed after one user interaction, a component-owned resource is often cleaner.

## 3. Component-Owned Async Reads

Use `fetch.UseResource[T](...)` when one component or feature subtree owns the async value.

Good `UseResource[T]` cases:

- a typed value belongs to one panel
- cancellation and dependency-driven reloads matter
- the loader does more than one direct fetch call
- the data is not reused broadly enough to justify a shared cache key

Use `fetch.UseFetch(...)` only when the lower-level raw fetch state is the point.

Good `UseFetch(...)` cases:

- a small raw response viewer
- ad hoc low-level request state
- one screen that really wants the plain fetch payload and manual decoding

For most app data, prefer `UseResource[T]` over `UseFetch(...)`.

## 4. Shared Cache

Use `fetch.UseCachedResource[T](...)` when the same logical query should be reused across multiple components or between route loaders and later component reads.

Good shared-cache cases:

- list plus detail surfaces need the same entity snapshot
- route loader and hydrated component should reuse one normalized key
- stale-while-revalidate behavior is desirable
- optimistic local patching is useful after mutation
- persistence across reloads is useful for reconstructible read data

Use `fetch.LoadCached[T](...)` from route loaders or other non-hook code when they should share the same cache entry as `UseCachedResource[T](...)`.

Do not put arbitrary one-off reads into the shared cache. If the data has only one owner and no reuse story, the cache adds lifecycle cost without helping much.

## 5. Revalidation

Use the revalidation tool that matches the owner:

- route-owned data: `router.UseRevalidator()`
- shared cached data: `CachedResource.Reload()` or `Invalidate()`
- local component-owned data: change the `UseResource[T]` deps or reload token

Do not try to solve every refresh with full-page reloads or unrelated atom updates.

Preferred pattern:

1. perform the authoritative mutation
2. revalidate the route or cache entries that actually became stale
3. keep surrounding UI state intact unless the product flow truly needs a full navigation reset

## 6. Mutation Ownership

Mutations remain explicit and application-owned.

Today the practical default is:

- local form state through `ui.UseForm[T]`
- explicit submit handler
- authoritative server validation
- explicit UI update or revalidation after success

Use normal `fetch.Fetch(...)`, `fetch.Upload(...)`, HTML form posts, or route-aware server handlers based on the endpoint contract.

Do not hide mutation transport behind magical framework-owned actions if the app still needs to decide:

- auth headers
- redirect behavior
- rollback policy
- which caches or routes become stale

## 7. Optimistic UI

Optimistic updates are useful, but they are not the source of truth.

Current recommended pattern:

1. update the visible cached value locally with `Set(...)` or `Update(...)`
2. perform the real mutation
3. invalidate or reload the affected cache key or route after the server response
4. roll back or repair the optimistic state if the mutation fails

Use optimistic UI when:

- the user clearly benefits from immediate feedback
- the rollback story is understandable
- the app can identify which read models became stale

Avoid optimistic updates when:

- the server result is likely to differ significantly
- conflicts are common
- the screen cannot explain rollback cleanly

## 8. Offline Replay

Use `fetch.OpenMutationQueue(...)` when writes must survive reloads or offline periods.

Good queue cases:

- draft submissions from unreliable networks
- field operations that must replay later
- deliberate offline-capable workflows with durable local intent

Not every mutation needs a queue. Most apps should still submit online and fail clearly when the network or server is unavailable.

Use the queue when the product explicitly needs:

- durable retry
- deduplication
- replay reporting
- dead-letter visibility
- conflict handling after reconnect

## 9. Persistence And SSR Resume

Keep the ownership split clear:

- SSR bootstrap and route loaders own first paint data
- shared cache bootstrap is for reconstructible read state
- offline mutation queue is for durable write intent
- atoms and snapshots are for client-owned UI state, not generic server data transport

Do not mix these layers casually.

Practical rule:

- read models belong in loaders, resources, and shared cache
- write intent belongs in forms, explicit mutation handlers, and the mutation queue
- UI coordination belongs in local state, reducers, context, or atoms

## 10. Recommended Defaults For Real Apps

For a typical non-trivial app:

- route entry data: route loaders
- panel-local async reads: `fetch.UseResource[T](...)`
- shared reusable read models: `fetch.UseCachedResource[T](...)`
- form state: `ui.UseForm[T]`
- post-success refresh: route revalidation or targeted cache invalidation
- optimistic list or detail patching: cache `Set(...)` or `Update(...)`
- offline writes only where the product needs them: mutation queue

## 11. Common Failure Modes

Watch for these mistakes:

- using atoms as a shadow async cache
- putting every read into route loaders even when it is deeply local
- using `UseFetch(...)` everywhere instead of typed resources
- inventing inconsistent cache keys across route and component readers
- optimistic updates without a revalidation or rollback path
- durable offline queueing for writes that should just fail fast

## 12. Example Starting Points

- typed async component reads: `examples/43-use-resource`
- shared cached reads: `examples/44-use-cached-resource`
- route loaders and revalidation: `examples/60-route-loaders`, `examples/61-use-revalidator`
- secure server-backed forms: `examples/87-ssr-secure-forms`
- route-loader plus shared-cache reuse: `examples/92-protected-routes`
- SSR-seeded cache bootstrap: `examples/93-ssr-cache-bootstrap`
- offline mutation queueing: `examples/97-pwa-offline-cache`

## Review Checklist

- is each async read owned by the smallest believable screen or route boundary
- are shared reads actually using a stable shared key
- are route-owned and component-owned reads clearly separated
- does each mutation have an explicit success, error, and stale-data refresh path
- are optimistic or offline paths only used where the product genuinely needs them
