//go:build js && wasm
// +build js,wasm

package runtime

import (
	"fmt"
	"syscall/js"
)

// GoUseFetch is a manual-trigger fetch hook that manages async data fetching
// It uses goroutines and channels internally to fetch data without blocking
// Unlike GoUseEffect-based auto-fetching, this requires explicit refetch() calls
//
// Returns:
//   - A getter function for the current FetchState (data, error, loading)
//   - A refetch function to manually trigger the fetch
//
// The FetchState contains:
//   - Data: the fetched response body (as string)
//   - Error: error message if fetch failed (empty string if successful)
//   - Loading: whether currently fetching
func GoUseFetch(url string, options ...interface{}) (func() FetchState, func()) {
	fiber := GetCurrentFiber()
	if fiber == nil {
		panic(actionableHookUsagePanic("GoUseFetch"))
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{
			owner:     fiber,
			states:    make([]interface{}, 0),
			deps:      make([][]interface{}, 0),
			memos:     make([]memoizedValue, 0),
			callbacks: make([]callbackValue, 0),
			refs:      make([]*RefValue, 0),
			ids:       make([]string, 0),
			fetches:   make([]fetchValue, 0),
			cleanups:  make([]func(), 0),
		}
	} else if fiber.hooks.owner == nil {
		fiber.hooks.owner = fiber
	}

	recordHookSignature(fiber.hooks, "fetch")
	fiber.hooks.index++

	fetchIdx := fiber.hooks.fetchIndex
	fiber.hooks.fetchIndex++

	// Initialize fetch state if needed
	if len(fiber.hooks.fetches) <= fetchIdx {
		newFetches := make([]fetchValue, fetchIdx+1, (fetchIdx+1)*2)
		copy(newFetches, fiber.hooks.fetches)
		state := FetchState{Data: nil, Error: "", Loading: false}
		if restoredState, ok := fiber.hooks.restoreFetchValue(fetchIdx, url); ok {
			state = restoredState
		}
		newFetches[fetchIdx] = fetchValue{
			state: state,
			url:   url,
			fiber: fiber,
		}
		fiber.hooks.fetches = newFetches
	} else {
		// Update fiber reference on every render to ensure it's current
		fiber.hooks.fetches[fetchIdx].fiber = fiber
	}

	hooks := fiber.hooks
	idx := fetchIdx

	// Getter returns current fetch state
	getter := func() FetchState {
		if idx < len(hooks.fetches) {
			return hooks.fetches[idx].state
		}
		return FetchState{Data: nil, Error: "", Loading: false}
	}

	// Refetch function to manually trigger a fetch
	refetch := func() {
		if idx >= len(hooks.fetches) {
			return
		}

		// Mark as loading
		hooks.fetches[idx].state = FetchState{Data: nil, Error: "", Loading: true}

		// Trigger component re-render
		rt := GetGlobalRuntime()
		if rt != nil {
			rt.ScheduleUpdateForFiberWithOrigin(fiber, "async-resource")
		}

		// Start fetch in a goroutine
		go func() {
			// Use syscall/js to call fetch API
			fetch := js.Global().Get("fetch")
			if !fetch.Truthy() {
				hooks.fetches[idx].state = FetchState{
					Data:    nil,
					Error:   "fetch API unavailable",
					Loading: false,
				}
				if rt != nil {
					rt.ScheduleUpdateForFiberWithOrigin(hooks.fetches[idx].fiber, "async-resource")
				}
				return
			}

			// Create promise
			promise := fetch.Invoke(url)

			// Handle promise
			var then, catch js.Func

			then = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				defer then.Release()
				defer catch.Release()

				resp := args[0]
				if !resp.Get("ok").Bool() {
					statusText := resp.Get("statusText").String()
					status := resp.Get("status").Int()
					errorMsg := fmt.Sprintf("Fetch failed: %d %s", status, statusText)

					hooks.fetches[idx].state = FetchState{
						Data:    nil,
						Error:   errorMsg,
						Loading: false,
					}
					if rt != nil {
						rt.ScheduleUpdateForFiberWithOrigin(hooks.fetches[idx].fiber, "async-resource")
					}
					return nil
				}

				// Get text
				textPromise := resp.Call("text")

				var textThen, textCatch js.Func
				textThen = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					defer textThen.Release()
					defer textCatch.Release()

					data := args[0].String()
					hooks.fetches[idx].state = FetchState{
						Data:    data,
						Error:   "",
						Loading: false,
					}
					if rt != nil {
						rt.ScheduleUpdateForFiberWithOrigin(hooks.fetches[idx].fiber, "async-resource")
					}
					return nil
				})

				textCatch = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					defer textThen.Release()
					defer textCatch.Release()

					hooks.fetches[idx].state = FetchState{
						Data:    nil,
						Error:   "Failed to read response body",
						Loading: false,
					}
					if rt != nil {
						rt.ScheduleUpdateForFiberWithOrigin(hooks.fetches[idx].fiber, "async-resource")
					}
					return nil
				})

				textPromise.Call("then", textThen)
				textPromise.Call("catch", textCatch)
				return nil
			})

			catch = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				defer then.Release()
				defer catch.Release()

				hooks.fetches[idx].state = FetchState{
					Data:    nil,
					Error:   "Fetch failed", // Matches test expectation
					Loading: false,
				}
				if rt != nil {
					rt.ScheduleUpdateForFiberWithOrigin(hooks.fetches[idx].fiber, "async-resource")
				}
				return nil
			})

			promise.Call("then", then)
			promise.Call("catch", catch)
		}()
	}

	return getter, refetch
}
