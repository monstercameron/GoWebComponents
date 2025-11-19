// Package fetch provides declarative data fetching utilities for GoWebComponents.
//
// This package simplifies asynchronous HTTP requests in WASM applications with
// React Query-style hooks that automatically manage loading, error, and data states.
//
// Basic usage:
//
//	import "github.com/monstercameron/GoWebComponents/fetch"
//
//	func UserList(props dom.Attrs) *fiber.Element {
//	    getState, refetch := fetch.UseFetch("/api/users")
//	    state := getState()
//
//	    if state.Loading {
//	        return dom.Div(nil, dom.P(nil, "Loading..."))
//	    }
//
//	    if state.Error != "" {
//	        return dom.Div(nil, dom.P(nil, "Error: "+state.Error))
//	    }
//
//	    users := state.Data.(string) // Parse JSON or use as needed
//	    return dom.Div(nil,
//	        dom.H1(nil, "Users"),
//	        dom.Pre(nil, users),
//	        dom.Button(map[string]interface{}{
//	            "onclick": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	                refetch() // Manually trigger a refetch
//	                return nil
//	            }),
//	        }, "Refresh"),
//	    )
//	}
//
// Available functions:
//
//   - UseFetch: Hook for automatic data fetching with state management
//   - Fetch: Low-level fetch function returning a channel for manual control
//
// The UseFetch hook automatically:
//   - Manages loading, error, and data states
//   - Fetches data when the component mounts
//   - Re-fetches when the URL changes
//   - Provides a refetch function for manual updates
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
