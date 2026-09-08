# 07 Data Loading And Mutations

Use this chapter when you are deciding who owns async reads, shared cached data, explicit mutations, optimistic updates, or offline replay.

It is the right chapter for:

- choosing between `fetch.UseResource[T](...)`, `fetch.UseCachedResource[T](...)`, and `fetch.Fetch(...)` (and the deprecated `fetch.UseFetch`)
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

- `fetch.UseResource[T](...)`: typed component-owned async value with cancellation and reload — the default choice
- `fetch.UseFetch(...)`: *(deprecated — prefer `UseResource`)* raw browser-style fetch state around one URL
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

- `fetch.UseResource[T](...)`
- `fetch.Fetch(...)`
- `fetch.UseFetch(...)` *(deprecated — prefer `UseResource`; kept stable for legacy callers)*

Important advanced surfaces:

- `fetch.UseCachedResource[T](...)` is a shipped public shared-cache tool, but advanced cached-resource lifecycle details such as persistence, bootstrap resume policy, and aggressive cache orchestration should still be treated carefully because the repo policy explicitly labels advanced cached-resource flows as `Experimental`
- `fetch.LoadCached[T](...)`, `fetch.RestoreCacheBootstrap(...)`, and persistent shared-cache configuration belong to that same advanced cache boundary
- `fetch.OpenMutationQueue(...)` is a shipped public queue surface, but replay policy, conflict resolution, and authoritative server behavior remain application-owned by design

## Minimal Example

Start with the smallest typed read: `fetch.UseResource[T](...)` owns one async value local to a
component, with built-in cancellation and reload, and returns a typed result instead of raw
`interface{}`.

```go
package main

import (
	"context"
	"time"

	h "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/fetch"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// renderStatusProbe owns one typed async read local to the current component.
func renderStatusProbe() ui.Node {
	storeResource := fetch.UseResource(func(getCtx context.Context) (string, error) {
		select {
		case <-time.After(400 * time.Millisecond):
		case <-getCtx.Done():
			return "", getCtx.Err()
		}
		return "healthy", nil
	})
	getState := storeResource.Get()

	handleUserRefresh := ui.UseEvent(func() {
		storeResource.Reload()
	})

	getStatus := "Idle"
	if getState.Loading {
		getStatus = "Loading"
	} else if getState.Error != nil {
		getStatus = "Error"
	} else if getState.Ready {
		getStatus = getState.Value
	}

	return h.Main(
		h.Class("mx-auto max-w-xl space-y-4 p-6"),
		h.H1("Typed resource state"),
		h.P(h.Textf("Status: %s", getStatus)),
		h.Button(h.Type("button"), h.OnClick(handleUserRefresh), "Reload"),
	)
}

// main mounts the example into the browser DOM.
func main() {
	ui.Run("#app", renderStatusProbe)
}
```

Why this is the right smallest path:

- the component owns one typed async value
- loading, error, and the typed result are read directly
- cancellation and reload are built in, with no shared-cache policy until the app needs it

> **Legacy:** `fetch.UseFetch(url)` returns raw loading/error/`interface{}` state for one URL. It is
> **deprecated** — prefer `UseResource` (or `ui.UseQuery` for cached, tag-invalidated data). It
> remains only for existing callers; see the API family table below.

## Production-Shaped Example

For most real component-owned reads, prefer `fetch.UseResource[T](...)` so the loader stays typed and cancellable.

```go
package deployview

import (
	"context"
	"time"

	h "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/fetch"
	"github.com/monstercameron/GoWebComponents/v6/ui"
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

```go gwc:build
package workspaceroute

import (
	"context"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/fetch"
	"github.com/monstercameron/GoWebComponents/v6/router"
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

	h "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v6/fetch"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
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

```go gwc:build
package mutationflow

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v6/fetch"
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

## UseResource Versus UseCachedResource Versus UseQuery

Use `fetch.UseResource[T](...)` when:

- the loader is typed
- cancellation matters
- the screen owns one async value that does not deserve a shared key

Use `fetch.UseCachedResource[T](...)` when:

- multiple components or route loaders should reuse the same logical query
- stale-aware refresh and optimistic local patching are valuable
- you are willing to own one normalized cache identity deliberately

Use `ui.UseQuery(...)` (over the `query` package) when:

- you want tag-aware invalidation across related queries
- optimistic mutations with automatic rollback are part of the flow
- the data layer benefits from request de-duplication and SWR (see the **Query Cache** section below)

Practical rule:

- `UseResource` is the preferred typed local read helper
- `UseCachedResource` is the shared-cache tool for reused read models
- `ui.UseQuery` is the tag-invalidated query/cache layer for application data
- `UseFetch` is the **deprecated** raw hook — prefer `UseResource`

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

### Typed error statuses (`serverfn.StatusError`)

A server function returns an ordinary `error`, which maps to HTTP 500 by default. To return a
client-error status, return a `*serverfn.StatusError` (or wrap one with `fmt.Errorf("...: %w", …)`):

```go
func getUser(ctx context.Context, req Req) (User, error) {
    u, ok := store.Find(req.ID)
    if !ok {
        return User{}, serverfn.NotFound("no such user") // → HTTP 404
    }
    return u, nil
}
```

Constructors cover the common cases: `BadRequest`, `Unauthorized`, `Forbidden`, `NotFound`,
`Conflict`, `UnprocessableEntity`, plus `NewStatusError(status, msg)` for any code. The status
round-trips to the caller as `*serverfn.ServerError`, so the client branches on the real reason:

```go
_, err := serverfn.Call[Req, User](ctx, "GetUser", req)
var se *serverfn.ServerError
if errors.As(err, &se) && se.Status == http.StatusNotFound {
    // show a not-found state instead of a generic failure
}
```

A plain `error` is unchanged (still 500) — adopting `StatusError` is opt-in and non-breaking.

### Server functions (`//gwc:server`)

A server function is a plain, type-safe `func(context.Context, Req) (Resp, error)` that runs only on
the server. Mark it with a `//gwc:server` doc comment, in a file constrained to the server build
(`//go:build !js || !wasm`); keep the shared `Req`/`Resp` types in a build-tag-free file.

```go
//go:build !js || !wasm

package api

import "context"

type GetUserReq struct{ ID string }

// //gwc:server
func GetUser(ctx context.Context, req GetUserReq) (User, error) {
	u, ok := store.Find(req.ID)
	if !ok {
		return User{}, serverfn.NotFound("no such user")
	}
	return u, nil
}
```

`gwc server gen -pkg .` scans the package, validates the `(context.Context, Req) (Resp, error)`
shape, and writes two files:

- `serverfn_gen_client.go` (`//go:build js && wasm`) — browser stubs that call
  `serverfn.Call[Req, Resp](ctx, "GetUser", req)`
- `serverfn_gen_server.go` (`//go:build !js || !wasm`) — `RegisterServerFunctions(mux *http.ServeMux)`
  that wires each function via `serverfn.Handle`

The browser calls the generated stub with full compile-time type safety and no hand-written
fetch/JSON glue; the server uses real sockets and the browser uses Fetch-backed `net/http`, so the
same code path is testable with `httptest`. `gwc server check` is the CI staleness gate. To point the
client at a non-default base URL, call `serverfn.Configure(baseURL)` (the default route prefix is
`serverfn.RoutePrefix`, `/_gwc/fn/`).

#### Whole-stack deployment (`wholestack`)

`wholestack.Handler` / `wholestack.ListenAndServe` compose ONE `http.Handler` that serves the
embedded wasm bundle AND the app's server functions, with SPA fallback so client-routed paths
deep-link to the shell — one `go build` is the entire app (see
[13 Assets, Deployment, And PWA](13-assets-deployment-and-pwa.md)).

```go
//go:embed dist
var assets embed.FS

func main() {
	sub, _ := fs.Sub(assets, "dist")
	log.Fatal(wholestack.ListenAndServe(":8080", wholestack.Options{
		Assets:            sub,
		RegisterServerFns: api.RegisterServerFunctions, // generated
	}))
}
```

## Query Cache And Mutations

For application data that several components share — with tag-aware invalidation, request
de-duplication, stale-while-revalidate, and optimistic mutations with automatic rollback — use the
`query` package through the `ui.UseQuery` / `ui.UseMutation` hooks. It is the Go-native answer to
TanStack Query / SWR, kept deliberately explicit (you pass the key and fetcher at every call site)
and pure Go (compiles to wasm and native, unit-testable with `query.WithClock`).

Create one cache (usually a package var), then read through `ui.UseQuery`:

```go
var appCache = query.New(query.WithStaleTime(30 * time.Second))

func renderUserCard(props userProps) ui.Node {
	res := ui.UseQuery(appCache, "user/"+props.ID, func() (User, error) {
		return api.GetUser(context.Background(), api.GetUserReq{ID: props.ID})
	}, props.ID)

	switch {
	case res.Status == query.StatusLoading:
		return h.P("Loading...")
	case res.Err != nil:
		return h.P(h.Textf("Error: %v", res.Err))
	default:
		return h.P(h.Text(res.Data.Name))
	}
}
```

`query.Result[T]` carries `Data`, `Err`, `Status` (`StatusIdle`/`StatusLoading`/`StatusSuccess`/
`StatusError`), `UpdatedAt`, `Stale`, and `Fetching` so the render path can show stale-while-fetching
states. The last `deps` arguments behave like `UseEffect` deps — the query re-keys when they change.

Mutations apply an optimistic value immediately and reconcile in the background:

```go
mutate := ui.UseMutation[User](appCache, "user/"+id)
handleSave := ui.UseEvent(func() {
	mutate(optimisticUser, func() (User, error) {
		return api.SaveUser(context.Background(), draft) // commit on success, rollback on error
	})
})
```

`query.MutateAsync(cache, key, optimistic, fn, onSettled)` is the fire-and-forget form (it does not
block render and calls `onSettled` with the final `Result[T]`). Invalidate related data with
`cache.Invalidate(key)`, `cache.InvalidatePrefix(prefix)`, or `cache.InvalidateAll()`; inspect cache
state for devtools with `cache.Inspect()` (see the time-travel/devtools chapter). When a component
should suspend until data is ready (rendering an `AsyncBoundary` fallback meanwhile), use
`ui.UseSuspenseQuery`, which returns the value `T` directly and throws to the nearest error boundary
on failure.

## Generative UI (`agentui`)

When an agent (or any untrusted source) should compose UI, `agentui` renders a typed schema against a
component allow-list — the safety property is structural: a `Node` carries no code, no event
handlers, and no raw HTML, only an allow-listed component `Type`, string `Props` the component
permits, escaped `Text`, and `Children`.

```go
reg := agentui.DefaultRegistry()
reg.Register(agentui.ComponentSpec{
	Name:         "callout",
	AllowedProps: []string{"tone"},
	Render: func(props map[string]string, children []ui.Node) ui.Node {
		return html.Div(html.Props{Class: "callout-" + props["tone"]}, children...)
	},
})

node, err := reg.RenderJSON(agentOutput) // validates, then renders to a safe ui.Node
```

`Registry.Validate` (and `ValidateWithLimits`) recursively reject any non-allow-listed type or
disallowed prop; `Render`/`RenderJSON` gate on validation. Untrusted input is bounded by
`agentui.DefaultLimits` (max depth 32, max 10k nodes) — pass explicit `Limits` to
`ValidateWithLimits` to tune. `Registry.Catalog()` returns the allow-list as sorted
`ComponentInfo` (name + permitted props) so an agent — or an MCP tool serving the registry over
`agentbridge` — learns up front exactly what it may emit, turning the allow-list into guidance rather
than an after-the-fact rejection.

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
| Raw hook state | `UseFetch`, `Resource.Get`, `Resource.Refetch` | `Stable` (deprecated — prefer `UseResource`) | a legacy component still wants raw request state around a URL | new code — use the typed `UseResource` instead |
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
