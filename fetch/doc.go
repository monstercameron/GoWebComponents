// Package fetch provides data loading utilities for GoWebComponents.
//
// The package exposes two public hook styles:
//   - UseFetch for low-level fetch state around a URL and browser-style refetching
//   - UseResource for typed, context-aware async loading in non-trivial components
//   - UseCachedResource for shared cached async state with deduplication and invalidation
//
// UseResource is the preferred choice when callers want typed results,
// cancellation, dependency-driven reloads, or loader logic that does more than
// a single raw fetch call. UseFetch remains useful when callers explicitly want
// the raw fetch state and response payload without introducing a typed loader.
//
// Basic usage:
//
//	import "github.com/monstercameron/GoWebComponents/fetch"
//
//	func UserList() ui.Node {
//	    resource := fetch.UseFetch("/api/users")
//	    state := resource.Get()
//
//	    if state.Loading {
//	        return html.Div(html.Props{}, html.P(html.Props{}, html.Text("Loading...")))
//	    }
//
//	    if state.Error != "" {
//	        return html.Div(html.Props{}, html.P(html.Props{}, html.Text("Error: "+state.Error)))
//	    }
//
//	    users := state.Data.(string) // Parse JSON or use as needed
//	    refresh := ui.UseEvent(func() {
//	        resource.Refetch()
//	    })
//	    return html.Div(html.Props{},
//	        html.H1(html.Props{}, html.Text("Users")),
//	        html.Pre(html.Props{}, html.Text(users)),
//	        html.Button(html.Props{OnClick: refresh}, html.Text("Refresh")),
//	    )
//	}
//
// Available functions:
//
//   - UseFetch: Hook for manual raw fetch state management via a handle
//   - UseResource: Typed async resource hook for context-aware loaders
//   - UseCachedResource: Shared typed cache hook with stale-while-revalidate behavior
//   - Fetch: Low-level fetch function returning a channel for manual control
//
// The UseFetch hook:
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
