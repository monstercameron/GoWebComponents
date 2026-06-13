# 07 Data Loading And Mutations

Use this chapter when you are deciding who owns async reads, shared cached data, explicit mutations, optimistic updates, or offline replay.

It is the right chapter for:

- choosing between `fetch.UseFetch`, `fetch.UseResource[T](...)`, `fetch.UseCachedResource[T](...)`, and `fetch.Fetch(...)`
- deciding when route loaders should own the first read instead of a component
- keeping optimistic updates and offline replay explicit instead of magical
- separating client-owned UI state from server-owned read models

Use another chapter instead when:

- you need local UI state ownership first: go to [06 State And Reactivity](06-state-and-reactivity.md)
- you need route params, route loaders, redirects, or route revalidation in depth: go to [08 Routing](08-routing.md)
- you need SSR bootstrap reuse and hydration details for cached data: go to [09 SSR And Hydration](09-ssr-and-hydration.md)
- you need forms, secure form posts, or accessibility-first form behavior in depth: go to [11 Forms Accessibility And I18n](11-forms-accessibility-and-i18n.md)

## Overview

The first decision is not which fetch helper looks nicest. The first decision is who owns the read.

Use the smallest owner that matches the screen:

- `fetch.UseFetch(...)`: raw browser-style fetch state around one URL
- `fetch.UseResource[T](...)`: typed component-owned async value with cancellation and reload
- route loaders: route-owned first-load data, including SSR-aware entry data
- `fetch.UseCachedResource[T](...)`: shared cached async data reused across components
- `fetch.LoadCached[T](...)`: non-hook shared-cache reuse from route loaders or imperative code
- `fetch.Fetch(...)`: imperative request path from handlers, goroutines, or utilities
- `fetch.Upload(...)`: imperative multipart upload path when progress or cancellation matters
- `fetch.OpenMutationQueue(...)`: durable replay queue for writes that must survive offline or reloads

Keep one boundary clear:

- read models belong in loaders, resources, or shared cache
- write intent belongs in explicit submit handlers, imperative fetch calls, or the offline mutation queue

## Stability Note

The core read surface is `Stable`:

- `fetch.UseFetch(...)`
- `fetch.UseResource[T](...)`
- `fetch.Fetch(...)`

Important advanced surfaces:

- `fetch.UseCachedResource[T](...)` is a shipped public shared-cache tool, but advanced cached-resource lifecycle details such as persistence, bootstrap resume policy, and aggressive cache orchestration should still be treated carefully because the repo policy explicitly labels advanced cached-resource flows as `Experimental`
- `fetch.LoadCached[T](...)`, `fetch.RestoreCacheBootstrap(...)`, and persistent shared-cache configuration belong to that same advanced cache boundary
- `fetch.OpenMutationQueue(...)` is a shipped public queue surface, but replay policy, conflict resolution, and authoritative server behavior remain application-owned by design

## Minimal Example

Use `fetch.UseFetch(...)` when the point is the raw request state itself.

```go
package main

import (
	"fmt"

	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

// renderStatusProbe keeps one raw fetch request local to the current component.
func renderStatusProbe() ui.Node {
	storeURL := ui.UseState("/api/health")
	storeResource := fetch.UseFetch(storeURL.Get())
	getState := storeResource.Get()

	handleUserRefresh := ui.UseEvent(func() {
		storeResource.Refetch()
	})

	getStatus := "Idle"
	if getState.Loading {
		getStatus = "Loading"
	} else if getState.Error != "" {
		getStatus = "Error"
	} else if getState.Data != nil {
		getStatus = "Ready"
	}

	return h.Main(
		h.Class("mx-auto max-w-xl space-y-4 p-6"),
		h.H1("Raw fetch state"),
		h.P(h.Textf("Status: %s", getStatus)),
		h.Pre(fmt.Sprint(getState.Data)),
		h.Button(h.Type("button"), h.OnClick(handleUserRefresh), "Refetch"),
	)
}

// main mounts the raw-fetch example into the browser DOM.
func main() {
	ui.Render(ui.CreateElement(renderStatusProbe, nil), "#app")
	utils.WaitForever()
}
```

Why this is the right smallest path:

- the component owns one request
- you can read loading, error, and raw payload directly
- there is no typed loader or shared-cache policy until the app actually needs it

## Production-Shaped Example

For most real component-owned reads, prefer `fetch.UseResource[T](...)` so the loader stays typed and cancellable.

```go
package deployview

import (
	"context"
	"time"

	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/ui"
)

type deployPreview struct {
	Environment string
	NodeCount   int
	Window      string
}

// renderDeployPreview keeps one typed async loader local to the current panel.
func renderDeployPreview() ui.Node {
	storeEnvironment := ui.UseState("staging")
	storeResource := fetch.UseResource(func(getCtx context.Context) (deployPreview, error) {
		select {
		case <-time.After(900 * time.Millisecond):
		case <-getCtx.Done():
			return deployPreview{}, getCtx.Err()
		}

		if storeEnvironment.Get() == "production" {
			return deployPreview{Environment: "production", NodeCount: 12, Window: "02:00 UTC"}, nil
		}
		return deployPreview{Environment: "staging", NodeCount: 4, Window: "now"}, nil
	}, storeEnvironment.Get())

	handleUserStaging := ui.UseEvent(func() {
		storeEnvironment.Set("staging")
	})
	handleUserProduction := ui.UseEvent(func() {
		storeEnvironment.Set("production")
	})
	handleUserReload := ui.UseEvent(func() {
		storeResource.Reload()
	})
	handleUserCancel := ui.UseEvent(func() {
		storeResource.Cancel()
	})

	getState := storeResource.Get()

	return h.Section(
		h.Class("space-y-4 rounded-2xl border border-slate-200 bg-white p-5"),
		h.H2("Typed resource"),
		h.Div(
			h.Class("flex flex-wrap gap-3"),
			h.Button(h.Type("button"), h.OnClick(handleUserStaging), "Staging"),
			h.Button(h.Type("button"), h.OnClick(handleUserProduction), "Production"),
			h.Button(h.Type("button"), h.OnClick(handleUserReload), "Reload"),
			h.Button(h.Type("button"), h.OnClick(handleUserCancel), "Cancel"),
		),
		h.IfElse(getState.Loading,
			h.P("Loading preview..."),
			h.IfElse(getState.Ready,
				h.P(h.Textf("%s nodes=%d window=%s", getState.Value.Environment, getState.Value.NodeCount, getState.Value.Window)),
				h.P("No preview yet"),
			),
		),
	)
}
```

Why this is the better default for real screens:

- the loader returns a typed value instead of raw `interface{}`
- cancellation is built into the ownership model
- dependency changes and manual reload stay explicit

## Scale-Up Example

In a larger app, let route loaders and later components share one normalized cache key, then keep optimistic mutation behavior explicit on top of that cache entry.

```go
package workspaceroute

import (
	"context"
	"time"

	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/router"
)

type workspaceSummary struct {
	Subject  string
	Revision int
	Projects int
}

// buildWorkspaceSummaryKey normalizes the shared cache identity for both loaders and components.
func buildWorkspaceSummaryKey(getSubject string) string {
	return "workspace:summary:" + getSubject
}

// loadWorkspaceSummaryAttrs seeds the shared cache from the route loader instead of forcing a second read later.
func loadWorkspaceSummaryAttrs(getCtx context.Context, getRouteCtx router.RouteContext) (router.Attrs, error) {
	getSubject := getRouteCtx.Params.Get("subject")
	getCacheKey := buildWorkspaceSummaryKey(getSubject)

	getSummary, getErr := fetch.LoadCached(getCtx, getCacheKey, func(getLoadCtx context.Context) (workspaceSummary, error) {
		_ = getLoadCtx
		return workspaceSummary{Subject: getSubject, Revision: 4, Projects: 7}, nil
	}, fetch.CacheOptions{
		StaleAfter:   20 * time.Second,
		MaxAge:       90 * time.Second,
		DisposeAfter: 3 * time.Minute,
		Persist:      true,
	})
	if getErr != nil {
		return nil, getErr
	}

	return router.Attrs{
		"cacheKey": getCacheKey,
		"summary":  getSummary,
	}, nil
}
```

```go
package workspaceview

import (
	"context"

	h "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

type workspaceSummary struct {
	Subject  string
	Revision int
	Projects int
}

// renderWorkspaceSummary reads the same shared cache key the route loader already seeded.
func renderWorkspaceSummary(getProps router.Attrs) ui.Node {
	getCacheKey, _ := getProps["cacheKey"].(string)
	storeSummary := fetch.UseCachedResource(getCacheKey, func(getCtx context.Context) (workspaceSummary, error) {
		_ = getCtx
		return workspaceSummary{Subject: "atlas-admin", Revision: 5, Projects: 8}, nil
	}, fetch.CacheOptions{
		StaleAfter:   20 * time.Second,
		MaxAge:       90 * time.Second,
		DisposeAfter: 3 * time.Minute,
		Persist:      true,
	})

	handleUserOptimisticProject := ui.UseEvent(func() {
		storeSummary.Update(func(getPrevious workspaceSummary) workspaceSummary {
			getPrevious.Projects++
			return getPrevious
		})
	})
	handleUserRevalidate := ui.UseEvent(func() {
		storeSummary.Invalidate()
	})

	getState := storeSummary.Get()

	return h.Section(
		h.Class("space-y-4 rounded-2xl border border-slate-200 bg-white p-5"),
		h.H2("Shared cached summary"),
		h.Div(
			h.Class("flex flex-wrap gap-3"),
			h.Button(h.Type("button"), h.OnClick(handleUserOptimisticProject), "Optimistic +1"),
			h.Button(h.Type("button"), h.OnClick(handleUserRevalidate), "Invalidate"),
		),
		h.P(h.Textf("ready=%t stale=%t projects=%d revision=%d", getState.Ready, getState.Stale, getState.Value.Projects, getState.Value.Revision)),
	)
}
```

```go
package mutationflow

import (
	"context"

	"github.com/monstercameron/GoWebComponents/fetch"
)

// buildMutationQueue opens the durable browser-backed replay queue for writes that must survive reloads.
func buildMutationQueue() (fetch.MutationQueue, error) {
	return fetch.OpenMutationQueue(fetch.MutationQueueOptions{
		StorageKey:  "workspace:offline-writes",
		MaxAttempts: 4,
	})
}

// handleUserReplayMutations replays queued writes through one authoritative application-owned executor.
func handleUserReplayMutations(getCtx context.Context, getQueue fetch.MutationQueue) error {
	_, getErr := getQueue.Replay(getCtx, func(getReplayCtx context.Context, getMutation fetch.QueuedMutation) error {
		getResultChan := fetch.Fetch(getMutation.URL, fetch.Options{
			Method: getMutation.Method,
			Body:   getMutation.Body,
		})
		getResult := <-getResultChan
		return getResult.Err
	})
	return getErr
}
```

Why this scales:

- one normalized key unifies route-owned first load and component-owned later reads
- optimistic cache updates stay local and reversible instead of pretending the server already agreed
- offline replay remains explicit because the app still owns auth, conflict handling, and revalidation after success

## UseFetch Versus UseResource Versus UseCachedResource

Use `fetch.UseFetch(...)` when:

- you really want raw loading, error, and response state
- one URL drives the component directly
- the parsing and orchestration are simple and local

Use `fetch.UseResource[T](...)` when:

- the loader is typed
- cancellation matters
- the screen owns one async value that does not deserve a shared key

Use `fetch.UseCachedResource[T](...)` when:

- multiple components or route loaders should reuse the same logical query
- stale-aware refresh and optimistic local patching are valuable
- you are willing to own one normalized cache identity deliberately

Practical rule:

- `UseFetch` is the raw hook
- `UseResource` is the preferred typed local read helper
- `UseCachedResource` is the shared-cache tool for reused read models

## Imperative Fetch And Upload

Use `fetch.Fetch(...)` when the request belongs in an event handler, helper, or goroutine instead of a component-scoped hook.

```go
getResultChan := fetch.Fetch("/api/reindex", fetch.Options{Method: "POST"})
go func() {
	getResult := <-getResultChan
	_ = getResult
}()
```

Use `fetch.Upload(...)` when multipart uploads need progress or explicit cancellation:

```go
for getUpdate := range fetch.Upload(getCtx, "/api/upload", fetch.Options{
	Body: fetch.MultipartBody{},
}) {
	_ = getUpdate
}
```

If you do not need upload progress, plain `fetch.Fetch(...)` or a normal HTML form post is often simpler.

## Server Actions Functions And RPC Boundary

Use the smallest server-owned mutation surface that matches the workflow:

- server actions when the user is already in a form or submit flow and progressive HTML fallback matters
- server functions when the app needs one typed imperative server-owned capability that is not naturally a route loader or form post
- explicit RPC envelopes only when several imperative operations share one transport contract and that contract is worth owning deliberately

Keep the ownership split explicit:

- reads still belong in route loaders, `UseResource`, or `UseCachedResource` by default
- writes stay authoritative on the server even when the client shows optimistic UI
- auth, CSRF, timeout, correlation-id, and retry policy belong to the app transport boundary, not to the fetch cache itself
- same-origin contracts are the easiest place to keep forms, loaders, cache invalidation, and auth/session behavior consistent

## Optimistic UI And Offline Replay Rules

Optimistic UI is a local UX improvement, not proof of mutation success.

Preferred pattern:

1. apply `Set(...)` or `Update(...)` on the cached value
2. perform the real mutation
3. invalidate, reload, or route-revalidate the stale read models
4. roll back or repair the optimistic state if the mutation fails

Use `fetch.OpenMutationQueue(...)` when:

- writes must survive reloads
- the network may disappear for long enough that retry-later is a product requirement
- replay reports, dead letters, or conflict handling need to be visible operationally

Do not use the queue when a normal online failure is acceptable and easier to explain.

## Cache Bootstrap And Persistence

The shared-cache surface also supports:

- `fetch.ConfigurePersistentCache(...)` for durable cache-store configuration
- `CacheOptions{Persist: true}` for opt-in durable read-cache persistence
- `fetch.RestoreCacheBootstrap(...)` for SSR-seeded shared cache restore before hydration

These are powerful, but they are still part of the advanced shared-cache boundary.

Keep these rules in place:

- cache keys must be normalized and deterministic
- cached payloads should stay JSON-shaped if they may be persisted or bootstrapped
- persisted cache entries are reconstructible copies, not authority
- logout, tenant switch, or privilege narrowing should explicitly dispose or purge affected cache keys

## API Family Reference

Use this table before you widen a read or mutation flow.

| API family | Representative APIs | Stability | Use it when | Prefer something else when |
| --- | --- | --- | --- | --- |
| Raw hook state | `UseFetch`, `Resource.Get`, `Resource.Refetch` | `Stable` | one component wants raw request state around a URL | the loader deserves typed values and cancellation |
| Typed component resource | `UseResource`, `AsyncResource.Get`, `Reload`, `Cancel` | `Stable` | one component or panel owns a typed async value | several readers should reuse the same query |
| Imperative request path | `Fetch`, `Upload`, `ReturnChannel` | `Stable` low-level surface | the request starts from an event handler, helper, or goroutine | a component-owned hook is clearer |
| Shared cached resource | `UseCachedResource`, `CachedResource.Get`, `Reload`, `Invalidate`, `Dispose`, `Set`, `Update` | advanced public cache surface | several readers share one logical query and stale-aware reuse matters | the data only has one owner |
| Route-loader cache reuse | `LoadCached`, `InvalidateResource`, `DisposeResource`, `SweepCachedResources` | advanced public cache surface | route loaders and later components should share one cache key | the screen does not need shared cache state |
| Cache persistence and bootstrap | `ConfigurePersistentCache`, `RestoreCacheBootstrap`, `CacheOptions{Persist, StaleAfter, MaxAge, DisposeAfter}` | advanced public cache surface, partly `Experimental` | the app intentionally persists or bootstraps reconstructible shared read models | the value is sensitive, huge, or not JSON-safe |
| Offline replay | `OpenMutationQueue`, `Enqueue`, `Replay`, `ReplayWithOptions`, `MutationReplayReport` | shipped public queue surface | durable write intent and replay policy are product requirements | normal online mutation failure is enough |
| Conflict handling | `NewMutationConflict`, `IsMutationConflict`, `GetMutationConflict`, `MutationConflictResolution` | shipped public queue surface | replay needs explicit conflict branching and repair | the app does not need durable replay or conflict-aware requeue |

## Design Notes And Boundaries

Keep these rules in mind when choosing a data-loading shape:

- choose the read owner first, then choose the helper
- route loaders own route entry data; components own local reads
- shared cache is for reused read models, not for every request in the app
- mutations remain explicit and application-owned even when optimistic UI or offline replay exists
- cached values and bootstrapped payloads should stay reconstructible, JSON-shaped, and non-sensitive
- atoms are not a substitute for async data ownership

## Common Failure Modes

- putting every async read into route loaders even when it is deeply local
- using `UseFetch(...)` everywhere instead of typed resources
- inventing multiple cache keys for the same logical query
- applying optimistic updates without a revalidation or rollback plan
- persisting cache payloads that are too large, sensitive, or not JSON-safe
- treating offline replay as if it removes the need for server-side conflict handling
- mixing UI coordination state and authoritative read models in the same store

## Validation

Use the smallest examples that prove the ownership layer you are adopting.

Raw fetch and typed local resources:

```powershell
go run ./tools/gwc dev -app .\examples\public\use-fetch\main.go
go run ./tools/gwc dev -app .\examples\public\use-resource\main.go
go run ./tools/gwc dev -app .\examples\public\fetch-imperative\main.go
```

Shared cache and route-loader reuse:

```powershell
go run ./tools/gwc dev -app .\examples\public\use-cached-resource\main.go
go run ./tools/gwc dev -app .\examples\public\protected-routes\main.go
go run ./tools/gwc dev -app .\examples\public\server-side-rendering-cache-bootstrap\main.go
```

Offline replay boundary:

```powershell
go run ./tools/gwc dev -app .\examples\public\progressive-web-app-offline-cache\main.go
```

## Topic Pagination
Topic 7 of 16. Use previous and next to move through the ordered manual chapters; the first and last topics wrap.
- Previous topic: [06 State And Reactivity](06-state-and-reactivity.md)
- Topic index: [Reference Manual](README.md)
- Next topic: [08 Routing](08-routing.md)
