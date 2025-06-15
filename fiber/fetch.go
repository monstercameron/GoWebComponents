//go:build js && wasm
// +build js,wasm

package fiber

import (
	"encoding/json"
	"fmt"
	"syscall/js"
)

// GoUseFetch is a hook that simplifies data fetching within a component.
// It manages loading, error, and data states automatically.
// It returns a getter for the current FetchState and a function to trigger a refetch.
func GoUseFetch(url string, options ...FetchOptions) (func() FetchState, func()) {
	getState, setState := GoUseState(FetchState{Loading: true})

	var opts FetchOptions
	if len(options) > 0 {
		opts = options[0]
	}

	fetchData := func() {
		debugf("FETCH", "useFetch: Fetching data from %s\n", url)

		// Set loading state
		setState(FetchState{Loading: true})

		// Create fetch options
		fetchOptions := js.Global().Get("Object").New()
		if opts.Method != "" {
			fetchOptions.Set("method", opts.Method)
		}
		if len(opts.Headers) > 0 {
			headers := js.Global().Get("Object").New()
			for key, value := range opts.Headers {
				headers.Set(key, value)
			}
			fetchOptions.Set("headers", headers)
		}
		if opts.Body != nil {
			switch v := opts.Body.(type) {
			case string:
				fetchOptions.Set("body", v)
			default:
				bodyJSON, err := json.Marshal(v)
				if err != nil {
					setState(FetchState{Error: "Error encoding request body: " + err.Error(), Loading: false})
					return
				}
				fetchOptions.Set("body", string(bodyJSON))
			}
		}

		fetchPromise := js.Global().Call("fetch", url, fetchOptions)
		
		// Create callbacks that will be released after use
		var thenCallback, jsonCallback, catchCallback js.Func
		
		jsonCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			defer jsonCallback.Release() // Clean up immediately after use
			
			data := args[0]
			jsonStr := js.Global().Get("JSON").Call("stringify", data).String()
			var parsedData interface{}
			err := json.Unmarshal([]byte(jsonStr), &parsedData)
			if err != nil {
				debugf("FETCH", "Error parsing data: %v\n", err)
				setState(FetchState{Error: err.Error(), Loading: false})
			} else {
				debugf("FETCH", "useFetch: Successfully fetched data\n")
				setState(FetchState{Data: parsedData, Loading: false})
			}
			return nil
		})
		
		thenCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			defer thenCallback.Release() // Clean up immediately after use
			
			response := args[0]
			if !response.Get("ok").Bool() {
				errorMsg := fmt.Sprintf("HTTP error! status: %s", response.Get("status").String())
				debugf("FETCH", "useFetch: %s\n", errorMsg)
				setState(FetchState{Error: errorMsg, Loading: false})
				return nil
			}
			response.Call("json").Call("then", jsonCallback)
			return nil
		})
		
		catchCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			defer catchCallback.Release() // Clean up immediately after use
			
			err := args[0]
			errorMsg := fmt.Sprintf("Fetch error: %s", err.Get("message").String())
			debugf("FETCH", "%s\n", errorMsg)
			setState(FetchState{Error: errorMsg, Loading: false})
			return nil
		})
		
		fetchPromise.Call("then", thenCallback).Call("catch", catchCallback)
	}

	GoUseEffect(func() {
		fetchData()
	}, []interface{}{url})

	return getState, fetchData
}

// GoFetch performs an asynchronous fetch operation and returns a channel for the result.
// This is a utility function for imperative fetching and requires manual state management.
func GoFetch(url string, options FetchOptions) <-chan FetchResult {
	resultChan := make(chan FetchResult, 1) // Buffered channel to avoid goroutine leak

	// Perform fetch directly without spawning goroutine to avoid goroutine leaks
	fetchOptions := js.Global().Get("Object").New()
	setFetchOptions(fetchOptions, options)
	performFetch(url, fetchOptions, resultChan)

	return resultChan
}

// setFetchOptions configures the fetch options object
func setFetchOptions(fetchOptions js.Value, options FetchOptions) {
	if options.Method != "" {
		fetchOptions.Set("method", options.Method)
	}

	if len(options.Headers) > 0 {
		headers := js.Global().Get("Object").New()
		for key, value := range options.Headers {
			headers.Set(key, value)
		}
		fetchOptions.Set("headers", headers)
	}

	if options.Body != nil {
		switch v := options.Body.(type) {
		case string:
			fetchOptions.Set("body", v)
		default:
			bodyJSON, err := json.Marshal(v)
			if err != nil {
				fetchOptions.Set("body", fmt.Sprintf("error encoding body: %v", err))
			} else {
				fetchOptions.Set("body", string(bodyJSON))
			}
		}
	}
}

// performFetch executes the actual fetch operation
func performFetch(url string, fetchOptions js.Value, resultChan chan<- FetchResult) {
	promise := js.Global().Call("fetch", url, fetchOptions)
	
	// Create callbacks that will be released after use
	var thenCallback, jsonCallback, catchCallback js.Func
	
	jsonCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer jsonCallback.Release() // Clean up immediately after use
		
		data := args[0]
		jsonStr := js.Global().Get("JSON").Call("stringify", data).String()
		var parsedData interface{}
		err := json.Unmarshal([]byte(jsonStr), &parsedData)
		if err != nil {
			resultChan <- FetchResult{Err: fmt.Errorf("error parsing response: %w", err)}
		} else {
			resultChan <- FetchResult{Data: parsedData}
		}
		return nil
	})
	
	thenCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer thenCallback.Release() // Clean up immediately after use
		
		response := args[0]
		if !response.Get("ok").Bool() {
			resultChan <- FetchResult{Err: fmt.Errorf("HTTP error! status: %s", response.Get("status").String())}
			return nil
		}

		response.Call("json").Call("then", jsonCallback)
		return nil
	})
	
	catchCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer catchCallback.Release() // Clean up immediately after use
		
		err := args[0]
		resultChan <- FetchResult{Err: fmt.Errorf("fetch error: %s", err.Get("message").String())}
		return nil
	})
	
	promise.Call("then", thenCallback).Call("catch", catchCallback)
}
