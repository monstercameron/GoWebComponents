//go:build js && wasm
// +build js,wasm

package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
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

// Resource exposes the current low-level fetch state and a refetch helper.
type Resource struct {
	get     func() State
	refetch func()
}

// ResourceState describes the state of a typed async resource.
type ResourceState[T any] struct {
	Value   T
	Loading bool
	Error   error
	Ready   bool
}

// AsyncResource exposes the current typed resource state and lifecycle controls.
type AsyncResource[T any] struct {
	get    func() ResourceState[T]
	reload func()
	cancel func()
}

// UseFetch is a hook that simplifies data fetching within a component.
// It uses the runtime fetch hook directly.
func UseFetch(url string, options ...Options) Resource {
	args := make([]interface{}, len(options))
	for i, opt := range options {
		args[i] = opt
	}
	get, refetch := runtime.GoUseFetch(url, args...)
	return Resource{get: get, refetch: refetch}
}

// Get returns the current low-level fetch state.
func (r Resource) Get() State {
	return r.get()
}

// Refetch restarts the underlying fetch request.
func (r Resource) Refetch() {
	r.refetch()
}

// UseResource provides a typed async resource hook driven by a Go loader.
//
// The loader runs on mount and whenever deps or the reload token change. It
// receives a context that is cancelled when the component unmounts, the
// dependency list changes, or Cancel is called on the returned handle.
func UseResource[T any](loader func(context.Context) (T, error), deps ...interface{}) AsyncResource[T] {
	state := ui.UseState(ResourceState[T]{})
	reloadTick := ui.UseState(0)
	cancelRef := ui.UseRef((context.CancelFunc)(nil))
	requestSeq := ui.UseRef(0)

	startLoad := func() {
		if cancel := cancelRef.Get(); cancel != nil {
			cancel()
		}

		requestSeq.Set(requestSeq.Get() + 1)
		seq := requestSeq.Get()
		ctx, cancel := context.WithCancel(context.Background())
		cancelRef.Set(cancel)

		state.Update(func(prev ResourceState[T]) ResourceState[T] {
			prev.Loading = true
			prev.Error = nil
			return prev
		})

		go func() {
			value, err := loader(ctx)
			if ctx.Err() != nil || requestSeq.Get() != seq {
				return
			}

			state.Set(ResourceState[T]{
				Value:   value,
				Loading: false,
				Error:   err,
				Ready:   err == nil,
			})
		}()
	}

	effectDeps := make([]interface{}, 0, len(deps)+1)
	effectDeps = append(effectDeps, reloadTick.Get())
	effectDeps = append(effectDeps, deps...)

	ui.UseEffect(func() func() {
		startLoad()
		return func() {
			if cancel := cancelRef.Get(); cancel != nil {
				cancel()
				cancelRef.Set(nil)
			}
		}
	}, effectDeps...)

	return AsyncResource[T]{
		get: func() ResourceState[T] { return state.Get() },
		reload: func() {
			reloadTick.Update(func(prev int) int { return prev + 1 })
		},
		cancel: func() {
			if cancel := cancelRef.Get(); cancel != nil {
				cancel()
				cancelRef.Set(nil)
			}
			state.Update(func(prev ResourceState[T]) ResourceState[T] {
				prev.Loading = false
				return prev
			})
		},
	}
}

// Get returns the current typed resource state.
func (r AsyncResource[T]) Get() ResourceState[T] {
	if r.get == nil {
		var zero ResourceState[T]
		return zero
	}

	return r.get()
}

// Reload starts a new resource load.
func (r AsyncResource[T]) Reload() {
	if r.reload != nil {
		r.reload()
	}
}

// Cancel cancels the active resource load, if any.
func (r AsyncResource[T]) Cancel() {
	if r.cancel != nil {
		r.cancel()
	}
}

// Fetch performs an asynchronous HTTP fetch operation and returns a channel for the result.
// This uses the browser Fetch API from WASM; it currently returns the response body as text.
func Fetch(url string, options Options) <-chan Result {
	ch := make(chan Result, 1)

	go func() {
		fetchFunction := js.Global().Get("fetch")
		if !fetchFunction.Truthy() {
			ch <- Result{Err: errors.New("fetch API unavailable in this environment")}
			return
		}

		requestOptions := js.Global().Get("Object").New()

		method := options.Method
		if method == "" {
			method = "GET"
		}
		requestOptions.Set("method", method)

		if options.Headers != nil {
			headers := js.Global().Get("Object").New()
			for k, v := range options.Headers {
				headers.Set(k, fmt.Sprint(v))
			}
			requestOptions.Set("headers", headers)
		}

		if options.Body != nil {
			switch body := options.Body.(type) {
			case string:
				requestOptions.Set("body", body)
			default:
				if encoded, err := json.Marshal(body); err == nil {
					requestOptions.Set("body", string(encoded))
				} else {
					ch <- Result{Err: fmt.Errorf("failed to encode body: %w", err)}
					return
				}
			}
		}

		promise := fetchFunction.Invoke(url, requestOptions)

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
