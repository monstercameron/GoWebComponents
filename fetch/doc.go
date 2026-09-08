// Package fetch provides data loading utilities for GoWebComponents.
//
// The package exposes these public hook styles:
//   - UseResource for typed, context-aware async loading in non-trivial components
//   - UseCachedResource for shared cached async state with deduplication and invalidation
//   - UseQuery and UseInfiniteQuery for tag-aware cache invalidation,
//     paginated data, and optimistic rollback helpers
//   - UseWebSocket and UseEventSource for bounded browser realtime streams
//   - OpenMutationQueue for durable offline write replay in browser storage
//   - UseFetch (deprecated) for low-level raw fetch state around a URL
//
// UseResource is the preferred choice in nearly all cases: it gives typed results,
// cancellation, dependency-driven reloads, and loader logic beyond a single raw
// fetch call. UseFetch is deprecated — prefer UseResource (or ui.UseQuery for
// cached, tag-invalidated data); it remains only for legacy callers.
//
// Basic usage:
//
//	import (
//	    "context"
//
//	    "github.com/monstercameron/GoWebComponents/v6/fetch"
//	)
//
//	func UserList() ui.Node {
//	    resource := fetch.UseResource(func(ctx context.Context) (string, error) {
//	        return fetchUsers(ctx) // your typed loader
//	    })
//	    state := resource.Get()
//
//	    if state.Loading {
//	        return html.Div(html.Props{}, html.P(html.Props{}, html.Text("Loading...")))
//	    }
//
//	    if state.Err != nil {
//	        return html.Div(html.Props{}, html.P(html.Props{}, html.Text("Error: "+state.Err.Error())))
//	    }
//
//	    refresh := ui.UseEvent(func() {
//	        resource.Reload()
//	    })
//	    return html.Div(html.Props{},
//	        html.H1(html.Props{}, html.Text("Users")),
//	        html.Pre(html.Props{}, html.Text(state.Data)),
//	        html.Button(html.Props{OnClick: refresh}, html.Text("Refresh")),
//	    )
//	}
//
// Available functions:
//
//   - UseResource: Typed async resource hook for context-aware loaders
//   - UseCachedResource: Shared typed cache hook with stale-while-revalidate behavior
//   - UseQuery: Tag-aware cached query hook with optimistic rollback helpers
//   - UseInfiniteQuery: Cached paginated query hook for load-more and infinite-scroll views
//   - InvalidateQueryTag / InvalidateQueryTags: Mark related cached queries stale by tag
//   - UseWebSocket: Bounded WebSocket hook with reconnect, backoff, and optional heartbeat
//   - UseEventSource: Bounded EventSource hook with reconnect, backoff, and heartbeat timeout tracking
//   - OpenMutationQueue: Durable queued write storage plus replay helpers for offline workflows
//   - Fetch: Low-level fetch function returning a channel for manual control
//   - UseFetch (deprecated): Raw fetch state hook — prefer UseResource
//
// The UseFetch hook (deprecated — prefer UseResource):
//   - Manages loading, error, and raw response data states
//   - Returns a handle with Get and Refetch methods
//   - Leaves response parsing and higher-level orchestration under caller control
//
// The UseResource hook:
//   - Accepts a loader of type func(context.Context) (T, error)
//   - Returns a typed handle with Get, Reload, and Cancel methods
//   - Cancels in-flight work when the component unmounts or dependencies change
//
// The UseCachedResource hook:
//   - Accepts a stable cache key plus a loader of type func(context.Context) (T, error)
//   - Returns a typed handle with Get, Reload, Cancel, Invalidate, Set, and Update methods
//   - Reuses cached values across components, deduplicates in-flight reloads, and keeps ready data visible during background refreshes
//
// UseQuery is the higher-level cache API for application data. It keeps the
// same typed state shape as UseCachedResource, adds normalized tags for
// invalidating related queries together, and exposes OptimisticUpdate rollback
// handles for mutation flows. UseInfiniteQuery stores pages under one cache key
// and appends additional pages with LoadNext while preserving the same tag and
// optimistic-update behavior.
//
// The realtime hooks:
//   - Return handles with bounded RealtimeState snapshots
//   - Keep only the newest MaxMessages and MaxErrors entries
//   - Reconnect with capped exponential backoff
//   - Compile on non-browser targets with RealtimeUnsupported state
//
// For more control, use the Fetch function directly:
//
//	ch := fetch.Fetch("/api/data", fetch.Options{
//	    Method: "POST",
//	    Headers: map[string]interface{}{"Content-Type": "application/json"},
//	    Body: `{"key": "value"}`,
//	})
//	result := <-ch
//	if result.Err != nil {
//	    // Handle error
//	}
//	// Use result.Data
package fetch
