// Package fetch provides declarative data fetching utilities for GoWebComponents.
//
// This package simplifies asynchronous HTTP requests in WASM applications with
// handle-based hooks that manage loading, error, and data states.
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
//   - UseFetch: Hook for manual fetch state management via a typed handle
//   - Fetch: Low-level fetch function returning a channel for manual control
//
// The UseFetch hook:
//   - Manages loading, error, and data states
//   - Returns a handle with Get and Refetch methods
//   - Leaves fetch timing under caller control
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
