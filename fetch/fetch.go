//go:build js && wasm
// +build js,wasm

package fetch

import (
	"github.com/monstercameron/GoWebComponents/fiber"
)

// Options represents configuration for HTTP fetch operations.
type Options = fiber.FetchOptions

// State represents the state of a fetch operation managed by UseFetch.
type State = fiber.FetchState

// Result represents the result of a manual Fetch operation.
type Result = fiber.FetchResult

// UseFetch is a hook that simplifies data fetching within a component.
// It automatically manages loading, error, and data states.
//
// The hook fetches data when:
//   - The component mounts (first render)
//   - The URL changes
//   - The refetch function is called manually
//
// Returns:
//   - A getter function that returns the current State (loading, error, data)
//   - A refetch function to manually trigger a new fetch
//
// Example with automatic fetching:
//
//	func DataDisplay(props dom.Attrs) *fiber.Element {
//	    getState, _ := fetch.UseFetch("https://api.example.com/data")
//	    state := getState()
//
//	    if state.Loading {
//	        return dom.P(nil, "Loading...")
//	    }
//	    if state.Error != "" {
//	        return dom.P(nil, "Error: "+state.Error)
//	    }
//	    return dom.Pre(nil, state.Data.(string))
//	}
//
// Example with manual refetch:
//
//	func RefreshableData(props dom.Attrs) *fiber.Element {
//	    getState, refetch := fetch.UseFetch("/api/data")
//	    state := getState()
//
//	    refreshBtn := dom.Button(map[string]interface{}{
//	        "onclick": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	            refetch()
//	            return nil
//	        }),
//	        "disabled": state.Loading,
//	    }, "Refresh")
//
//	    return dom.Div(nil, dom.Pre(nil, state.Data.(string)), refreshBtn)
//	}
//
// Example with options:
//
//	func PostData(props dom.Attrs) *fiber.Element {
//	    getState, _ := fetch.UseFetch("/api/submit", fetch.Options{
//	        Method: "POST",
//	        Headers: map[string]interface{}{
//	            "Content-Type": "application/json",
//	        },
//	        Body: `{"message": "Hello"}`,
//	    })
//	    // ...
//	}
func UseFetch(url string, options ...Options) (func() State, func()) {
	return fiber.GoUseFetch(url, options...)
}

// Fetch performs an asynchronous HTTP fetch operation and returns a channel for the result.
// This is a lower-level API for imperative fetching that requires manual state management.
//
// The returned channel will receive exactly one Result containing either:
//   - Data: the response body as a string (if successful)
//   - Err: an error (if the request failed)
//
// Important: The channel should be consumed and then returned to the pool using
// ReturnChannel() to prevent memory leaks.
//
// Example:
//
//	func fetchUserData(userID int) {
//	    ch := fetch.Fetch(
//	        fmt.Sprintf("/api/users/%d", userID),
//	        fetch.Options{Method: "GET"},
//	    )
//
//	    result := <-ch
//	    fetch.ReturnChannel(ch) // Return channel to pool
//
//	    if result.Err != nil {
//	        fmt.Printf("Fetch failed: %v\n", result.Err)
//	        return
//	    }
//
//	    fmt.Printf("User data: %s\n", result.Data)
//	}
//
// For most use cases, prefer UseFetch which handles state management automatically.
func Fetch(url string, options Options) <-chan Result {
	return fiber.GoFetch(url, options)
}

// ReturnChannel returns a fetch result channel to the pool for reuse.
// This should be called after consuming the result from Fetch() to prevent memory leaks.
//
// Example:
//
//	ch := fetch.Fetch(url, options)
//	result := <-ch
//	fetch.ReturnChannel(ch) // Important: return to pool
//
// Note: This function is only needed when using the low-level Fetch function.
// UseFetch handles channel management automatically.
func ReturnChannel(ch <-chan Result) {
	fiber.ReturnFetchChannel(ch)
}
