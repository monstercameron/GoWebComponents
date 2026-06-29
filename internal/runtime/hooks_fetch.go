//go:build js && wasm

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
func GoUseFetch(parseUrl string, parseOptions ...interface{}) (func() FetchState, func()) {
	parseFiber := requireCurrentHookFiber("GoUseFetch")

	if parseFiber.hooks == nil {
		parseFiber.hooks = &Hooks{
			owner:     parseFiber,
			states:    make([]interface{}, 0),
			deps:      make([][]interface{}, 0),
			memos:     make([]memoizedValue, 0),
			callbacks: make([]callbackValue, 0),
			refs:      make([]*RefValue, 0),
			ids:       make([]string, 0),
			fetches:   make([]fetchValue, 0),
			cleanups:  make([]func(), 0),
		}
	} else if parseFiber.hooks.owner == nil {
		parseFiber.hooks.owner = parseFiber
	}

	recordHookSignature(parseFiber.hooks, "fetch")
	parseFiber.hooks.index++

	parseFetchIdx := parseFiber.hooks.fetchIndex
	parseFiber.hooks.fetchIndex++

	// Initialize fetch state if needed
	if len(parseFiber.hooks.fetches) <= parseFetchIdx {
		parseNewFetches := make([]fetchValue, parseFetchIdx+1, (parseFetchIdx+1)*2)
		copy(parseNewFetches, parseFiber.hooks.fetches)
		parseState := FetchState{Data: nil, Error: "", Loading: false}
		if parseRestoredState, parseOk := parseFiber.hooks.restoreFetchValue(parseFetchIdx, parseUrl); parseOk {
			parseState = parseRestoredState
		}
		parseNewFetches[parseFetchIdx] = fetchValue{
			state: parseState,
			url:   parseUrl,
			fiber: parseFiber,
		}
		parseFiber.hooks.fetches = parseNewFetches
	} else {
		// Update fiber reference on every render to ensure it's current
		parseFiber.hooks.fetches[parseFetchIdx].fiber = parseFiber
	}

	parseHooks := parseFiber.hooks
	parseIdx := parseFetchIdx

	// Getter returns current fetch state
	parseGetter := func() FetchState {
		if parseIdx < len(parseHooks.fetches) {
			return parseHooks.fetches[parseIdx].state
		}
		return FetchState{Data: nil, Error: "", Loading: false}
	}

	// Refetch function to manually trigger a fetch
	parseRefetch := func() {
		if parseIdx >= len(parseHooks.fetches) {
			return
		}

		// Mark as loading
		parseHooks.fetches[parseIdx].state = FetchState{Data: nil, Error: "", Loading: true}

		// Trigger component re-render
		parseRt := GetGlobalRuntime()
		if parseRt != nil {
			parseRt.ScheduleUpdateForFiberWithOrigin(parseFiber, "async-resource")
		}

		// Start fetch in a goroutine
		go func() {
			defer containAsyncPanic("runtime", "UseFetch request")
			// Use syscall/js to call fetch API
			parseFetch := js.Global().Get("fetch")
			if !parseFetch.Truthy() {
				parseHooks.fetches[parseIdx].state = FetchState{
					Data:    nil,
					Error:   "fetch API unavailable",
					Loading: false,
				}
				if parseRt != nil {
					parseRt.ScheduleUpdateForFiberWithOrigin(parseHooks.fetches[parseIdx].fiber, "async-resource")
				}
				return
			}

			// Create promise
			parsePromise := parseFetch.Invoke(parseUrl)

			// Handle promise
			var parseThen, parseCatch js.Func

			parseThen = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				defer containAsyncPanic("runtime", "GoUseFetch callback")
				defer parseThen.Release()
				defer parseCatch.Release()

				parseResp := parseArgs[0]
				if !parseResp.Get("ok").Bool() {
					parseStatusText := parseResp.Get("statusText").String()
					parseStatus := parseResp.Get("status").Int()
					parseErrorMsg := fmt.Sprintf("Fetch failed: %d %s", parseStatus, parseStatusText)

					parseHooks.fetches[parseIdx].state = FetchState{
						Data:    nil,
						Error:   parseErrorMsg,
						Loading: false,
					}
					if parseRt != nil {
						parseRt.ScheduleUpdateForFiberWithOrigin(parseHooks.fetches[parseIdx].fiber, "async-resource")
					}
					return nil
				}

				// Get text
				parseTextPromise := parseResp.Call("text")

				var parseTextThen, parseTextCatch js.Func
				parseTextThen = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
					defer containAsyncPanic("runtime", "GoUseFetch callback")
					defer parseTextThen.Release()
					defer parseTextCatch.Release()

					parseData := parseArgs2[0].String()
					parseHooks.fetches[parseIdx].state = FetchState{
						Data:    parseData,
						Error:   "",
						Loading: false,
					}
					if parseRt != nil {
						parseRt.ScheduleUpdateForFiberWithOrigin(parseHooks.fetches[parseIdx].fiber, "async-resource")
					}
					return nil
				})

				parseTextCatch = js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
					defer containAsyncPanic("runtime", "GoUseFetch callback")
					defer parseTextThen.Release()
					defer parseTextCatch.Release()

					parseHooks.fetches[parseIdx].state = FetchState{
						Data:    nil,
						Error:   "Failed to read response body",
						Loading: false,
					}
					if parseRt != nil {
						parseRt.ScheduleUpdateForFiberWithOrigin(parseHooks.fetches[parseIdx].fiber, "async-resource")
					}
					return nil
				})

				parseTextPromise.Call("then", parseTextThen)
				parseTextPromise.Call("catch", parseTextCatch)
				return nil
			})

			parseCatch = js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
				defer containAsyncPanic("runtime", "GoUseFetch callback")
				defer parseThen.Release()
				defer parseCatch.Release()

				parseHooks.fetches[parseIdx].state = FetchState{
					Data:    nil,
					Error:   "Fetch failed", // Matches test expectation
					Loading: false,
				}
				if parseRt != nil {
					parseRt.ScheduleUpdateForFiberWithOrigin(parseHooks.fetches[parseIdx].fiber, "async-resource")
				}
				return nil
			})

			parsePromise.Call("then", parseThen)
			parsePromise.Call("catch", parseCatch)
		}()
	}

	return parseGetter, parseRefetch
}
