//go:build js && wasm
// +build js,wasm

package fetch

import (
	"encoding/json"
	"errors"
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// Options represents configuration for HTTP fetch operations.
type Options struct {
	Method  string
	Headers map[string]interface{}
	Body    interface{}
}

// State represents the state of a fetch operation managed by UseFetch.
type State = runtime.FetchState

// Result represents the result of a manual Fetch operation.
type Result struct {
	Data interface{}
	Err  error
}

// UseFetch is a hook that simplifies data fetching within a component.
// It uses the new hooks.UseFetch entry point internally.
func UseFetch(url string, options ...Options) (func() State, func()) {
	args := make([]interface{}, len(options))
	for i, opt := range options {
		args[i] = opt
	}
	return hooks.UseFetch(url, args...)
}

// Fetch performs an asynchronous HTTP fetch operation and returns a channel for the result.
// This uses the browser Fetch API from WASM; it currently returns the response body as text.
func Fetch(url string, options Options) <-chan Result {
	ch := make(chan Result, 1)

	go func() {
		fetch := js.Global().Get("fetch")
		if !fetch.Truthy() {
			ch <- Result{Err: errors.New("fetch API unavailable in this environment")}
			return
		}

		opts := js.Global().Get("Object").New()

		method := options.Method
		if method == "" {
			method = "GET"
		}
		opts.Set("method", method)

		if options.Headers != nil {
			headers := js.Global().Get("Object").New()
			for k, v := range options.Headers {
				headers.Set(k, fmt.Sprint(v))
			}
			opts.Set("headers", headers)
		}

		if options.Body != nil {
			switch body := options.Body.(type) {
			case string:
				opts.Set("body", body)
			default:
				if encoded, err := json.Marshal(body); err == nil {
					opts.Set("body", string(encoded))
				} else {
					ch <- Result{Err: fmt.Errorf("failed to encode body: %w", err)}
					return
				}
			}
		}

		promise := fetch.Invoke(url, opts)

		var bodyThen js.Func
		var bodyCatch js.Func
		var resolve js.Func
		var reject js.Func

		resolve = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			defer resolve.Release()
			defer reject.Release()

			resp := args[0]
			textPromise := resp.Call("text")

			bodyThen = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				defer bodyThen.Release()
				defer bodyCatch.Release()
				if len(args) > 0 {
					ch <- Result{Data: args[0].String()}
					return nil
				}
				ch <- Result{Err: errors.New("empty response from fetch")}
				return nil
			})

			bodyCatch = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				defer bodyThen.Release()
				defer bodyCatch.Release()
				ch <- Result{Err: fmt.Errorf("failed to read body: %v", args)}
				return nil
			})

			textPromise.Call("then", bodyThen)
			textPromise.Call("catch", bodyCatch)
			return nil
		})

		reject = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			defer resolve.Release()
			defer reject.Release()
			ch <- Result{Err: fmt.Errorf("fetch failed: %v", args)}
			return nil
		})

		promise.Call("then", resolve)
		promise.Call("catch", reject)
	}()

	return ch
}

// ReturnChannel returns a fetch result channel to the pool for reuse.
// With the new implementation channels are one-shot, so this is a no-op kept for API compatibility.
func ReturnChannel(ch <-chan Result) {
	_ = ch
}
