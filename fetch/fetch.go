//go:build js && wasm
// +build js,wasm

package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
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

// MultipartFile describes one browser file to append to a multipart form-data body.
type MultipartFile struct {
	FieldName string
	File      ui.File
	Filename  string
}

// MultipartBody describes a multipart form-data payload with text fields and browser files.
type MultipartBody struct {
	Fields map[string]string
	Files  []MultipartFile
}

// State represents the state of a fetch operation managed by UseFetch.
type State = runtime.FetchState

// Result represents the result of a manual Fetch operation.
type Result struct {
	Data    interface{}
	Status  int
	Headers map[string]string
	Err     error
}

// UploadUpdate reports upload progress and the final upload result.
type UploadUpdate struct {
	Loaded           int64
	Total            int64
	LengthComputable bool
	Done             bool
	Result           Result
}

// HTTPError reports a non-success HTTP status while preserving the response metadata on Result.
type HTTPError struct {
	Status     int
	StatusText string
	Body       string
	Headers    map[string]string
}

func (e HTTPError) Error() string {
	if e.Status <= 0 {
		return "request failed"
	}
	if statusText := strings.TrimSpace(e.StatusText); statusText != "" {
		return fmt.Sprintf("request failed with status %d %s", e.Status, statusText)
	}
	return fmt.Sprintf("request failed with status %d", e.Status)
}

// Text returns the response body when it is string-backed.
func (r Result) Text() string {
	if text, ok := r.Data.(string); ok {
		return text
	}
	return fmt.Sprint(r.Data)
}

// DecodeJSON decodes a string-backed response body into target.
func (r Result) DecodeJSON(target interface{}) error {
	if target == nil {
		return errors.New("decode target is nil")
	}
	body := strings.TrimSpace(r.Text())
	if body == "" {
		return errors.New("response body is empty")
	}
	if err := json.Unmarshal([]byte(body), target); err != nil {
		return fmt.Errorf("decode response json: %w", err)
	}
	return nil
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

		bodyValue, isFormData, err := bodyToJSValue(options.Body)
		if err != nil {
			ch <- Result{Err: err}
			return
		}

		if options.Headers != nil {
			headers := js.Global().Get("Object").New()
			for k, v := range options.Headers {
				if isFormData && strings.EqualFold(k, "Content-Type") {
					continue
				}
				headers.Set(k, fmt.Sprint(v))
			}
			requestOptions.Set("headers", headers)
		}

		if !bodyValue.IsUndefined() && !bodyValue.IsNull() {
			requestOptions.Set("body", bodyValue)
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
			status := resp.Get("status").Int()
			statusText := resp.Get("statusText").String()
			headers := responseHeadersToMap(resp.Get("headers"))
			textPromise := resp.Call("text")

			bodyThen = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				defer bodyThen.Release()
				defer bodyCatch.Release()
				result := Result{
					Status:  status,
					Headers: headers,
				}
				if len(args) > 0 {
					result.Data = args[0].String()
					if status < 200 || status >= 300 {
						result.Err = HTTPError{Status: status, StatusText: statusText, Body: result.Text(), Headers: headers}
					}
					ch <- result
					return nil
				}
				result.Err = errors.New("empty response from fetch")
				ch <- result
				return nil
			})

			bodyCatch = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				defer bodyThen.Release()
				defer bodyCatch.Release()
				ch <- Result{
					Status:  status,
					Headers: headers,
					Err:     fmt.Errorf("failed to read body: %v", args),
				}
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

// Upload performs an XHR-backed upload so callers can observe progress and cancel through context.
func Upload(ctx context.Context, url string, options Options) <-chan UploadUpdate {
	ch := make(chan UploadUpdate, 8)

	go func() {
		xhrCtor := js.Global().Get("XMLHttpRequest")
		if !xhrCtor.Truthy() {
			ch <- UploadUpdate{Done: true, Result: Result{Err: errors.New("XMLHttpRequest unavailable in this environment")}}
			close(ch)
			return
		}

		bodyValue, isFormData, err := bodyToJSValue(options.Body)
		if err != nil {
			ch <- UploadUpdate{Done: true, Result: Result{Err: err}}
			close(ch)
			return
		}

		xhr := xhrCtor.New()
		method := options.Method
		if method == "" {
			method = "POST"
		}
		xhr.Call("open", method, url, true)
		if options.Headers != nil {
			for k, v := range options.Headers {
				if isFormData && strings.EqualFold(k, "Content-Type") {
					continue
				}
				xhr.Call("setRequestHeader", k, fmt.Sprint(v))
			}
		}

		var progressFn js.Func
		var loadFn js.Func
		var errorFn js.Func
		var abortFn js.Func
		var progressRegistered bool
		var loadRegistered bool
		var errorRegistered bool
		var abortRegistered bool
		var once sync.Once
		done := make(chan struct{})
		cleanup := func() {
			if progressRegistered {
				progressFn.Release()
			}
			if loadRegistered {
				loadFn.Release()
			}
			if errorRegistered {
				errorFn.Release()
			}
			if abortRegistered {
				abortFn.Release()
			}
		}
		finalize := func(update UploadUpdate) {
			once.Do(func() {
				close(done)
				ch <- update
				close(ch)
				cleanup()
			})
		}

		progressFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) == 0 {
				return nil
			}
			event := args[0]
			update := UploadUpdate{
				Loaded:           int64(event.Get("loaded").Float()),
				Total:            int64(event.Get("total").Float()),
				LengthComputable: event.Get("lengthComputable").Truthy(),
			}
			select {
			case ch <- update:
			default:
			}
			return nil
		})
		progressRegistered = true

		loadFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			status := xhr.Get("status").Int()
			statusText := xhr.Get("statusText").String()
			responseText := xhr.Get("responseText")
			headers := parseRawHeaders(xhr.Call("getAllResponseHeaders").String())
			result := Result{
				Data:    responseText.String(),
				Status:  status,
				Headers: headers,
			}
			if status < 200 || status >= 300 {
				result.Err = HTTPError{Status: status, StatusText: statusText, Body: result.Text(), Headers: headers}
			}
			loaded, total, computable := progressFromEvent(args)
			finalize(UploadUpdate{
				Loaded:           loaded,
				Total:            total,
				LengthComputable: computable,
				Done:             true,
				Result:           result,
			})
			return nil
		})
		loadRegistered = true

		errorFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			finalize(UploadUpdate{Done: true, Result: Result{Err: errors.New("upload request failed")}})
			return nil
		})
		errorRegistered = true

		abortFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			err := errors.New("upload aborted")
			if ctx != nil && ctx.Err() != nil {
				err = ctx.Err()
			}
			finalize(UploadUpdate{Done: true, Result: Result{Err: err}})
			return nil
		})
		abortRegistered = true

		if upload := xhr.Get("upload"); upload.Truthy() {
			upload.Call("addEventListener", "progress", progressFn)
		}
		xhr.Call("addEventListener", "load", loadFn)
		xhr.Call("addEventListener", "error", errorFn)
		xhr.Call("addEventListener", "abort", abortFn)

		if ctx != nil {
			go func() {
				select {
				case <-ctx.Done():
					xhr.Call("abort")
				case <-done:
				}
			}()
		}

		if !bodyValue.IsUndefined() && !bodyValue.IsNull() {
			xhr.Call("send", bodyValue)
		} else {
			xhr.Call("send")
		}
	}()

	return ch
}

func responseHeadersToMap(headers js.Value) map[string]string {
	if headers.IsUndefined() || headers.IsNull() {
		return nil
	}
	values := map[string]string{}
	callback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 2 {
			return nil
		}
		values[args[1].String()] = args[0].String()
		return nil
	})
	defer callback.Release()
	headers.Call("forEach", callback)
	if len(values) == 0 {
		return nil
	}
	return values
}

func parseRawHeaders(raw string) map[string]string {
	lines := strings.Split(raw, "\n")
	headers := make(map[string]string, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

func progressFromEvent(args []js.Value) (loaded int64, total int64, computable bool) {
	if len(args) == 0 {
		return 0, 0, false
	}
	event := args[0]
	if event.IsUndefined() || event.IsNull() {
		return 0, 0, false
	}
	loaded = int64(event.Get("loaded").Float())
	total = int64(event.Get("total").Float())
	computable = event.Get("lengthComputable").Truthy()
	return loaded, total, computable
}

func bodyToJSValue(body interface{}) (js.Value, bool, error) {
	if body == nil {
		return js.Undefined(), false, nil
	}
	switch typed := body.(type) {
	case string:
		return js.ValueOf(typed), false, nil
	case MultipartBody:
		value, err := buildMultipartFormData(typed)
		return value, true, err
	case *MultipartBody:
		if typed == nil {
			return js.Undefined(), false, nil
		}
		value, err := buildMultipartFormData(*typed)
		return value, true, err
	default:
		encoded, err := json.Marshal(body)
		if err != nil {
			return js.Undefined(), false, fmt.Errorf("failed to encode body: %w", err)
		}
		return js.ValueOf(string(encoded)), false, nil
	}
}

func buildMultipartFormData(body MultipartBody) (js.Value, error) {
	formDataCtor := js.Global().Get("FormData")
	if !formDataCtor.Truthy() {
		return js.Undefined(), errors.New("FormData unavailable in this environment")
	}
	form := formDataCtor.New()
	for key, value := range body.Fields {
		form.Call("append", key, value)
	}
	for _, file := range body.Files {
		if strings.TrimSpace(file.FieldName) == "" {
			continue
		}
		raw := file.File.JSValue()
		if raw.IsUndefined() || raw.IsNull() {
			continue
		}
		if strings.TrimSpace(file.Filename) != "" {
			form.Call("append", file.FieldName, raw, file.Filename)
		} else {
			form.Call("append", file.FieldName, raw)
		}
	}
	return form, nil
}

// ReturnChannel returns a fetch result channel to the pool for reuse.
// With the new implementation channels are one-shot, so this is a no-op kept for API compatibility.
func ReturnChannel(ch <-chan Result) {
	_ = ch
}
