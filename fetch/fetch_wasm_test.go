//go:build js && wasm
// +build js,wasm

package fetch

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync/atomic"
	"syscall/js"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

type noOpScheduler struct{}

func (noOpScheduler) RequestIdleCallback(callback func(runtime.Deadline)) {}

func (noOpScheduler) SetTimeout(callback func(), delay int) {}

func installFetchHookContext(t *testing.T) {
	t.Helper()
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	t.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})
}

func setGlobalJSValue(name string, value interface{}) func() {
	global := js.Global()
	prev := global.Get(name)
	global.Set(name, value)
	return func() {
		global.Set(name, prev)
	}
}

func makeResolvedPromise(value js.Value) js.Value {
	return js.Global().Get("Promise").Call("resolve", value)
}

func waitForResult(t *testing.T, ch <-chan Result) Result {
	t.Helper()
	select {
	case result := <-ch:
		return result
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for fetch result")
		return Result{}
	}
}

func TestUseFetchWrapper(t *testing.T) {
	installFetchHookContext(t)
	restoreFetch := setGlobalJSValue("fetch", js.Null())
	defer restoreFetch()

	resource := UseFetch("/api/demo", Options{Method: "POST"})
	initial := resource.Get()
	if initial.Loading || initial.Error != "" || initial.Data != nil {
		t.Fatalf("unexpected initial state: %+v", initial)
	}

	resource.Refetch()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state := resource.Get()
		if !state.Loading {
			if state.Error != "fetch API unavailable" {
				t.Fatalf("expected unavailable fetch error, got %q", state.Error)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for fetch hook to settle")
}

func TestUseFetchReturnsStableHandleShape(t *testing.T) {
	installFetchHookContext(t)
	resource := UseFetch("/api/demo")
	if state := resource.Get(); state.Loading || state.Error != "" || state.Data != nil {
		t.Fatalf("unexpected initial fetch state: %+v", state)
	}
}

func TestUseResourceReturnsStableHandleShape(t *testing.T) {
	installFetchHookContext(t)

	resource := UseResource(func(ctx context.Context) (string, error) {
		return "ok", nil
	}, "dep")

	state := resource.Get()
	if state.Loading || state.Error != nil || state.Ready {
		t.Fatalf("unexpected initial resource state: %+v", state)
	}
	resource.Reload()
	resource.Cancel()
}

func TestAsyncResourceZeroValue(t *testing.T) {
	var resource AsyncResource[string]
	state := resource.Get()
	if state.Loading || state.Error != nil || state.Ready || state.Value != "" {
		t.Fatalf("expected zero-value resource state, got %+v", state)
	}
	resource.Reload()
	resource.Cancel()
}

func TestCachedResourceZeroValue(t *testing.T) {
	var resource CachedResource[string]
	state := resource.Get()
	if state.Loading || state.Error != nil || state.Ready || state.Stale || state.Value != "" || !state.UpdatedAt.IsZero() {
		t.Fatalf("expected zero-value cached resource state, got %+v", state)
	}
	resource.Reload()
	resource.Cancel()
	resource.Invalidate()
	resource.Set("ignored")
	resource.Update(func(prev string) string { return prev + "x" })
}

func TestUseCachedResourceReturnsStableHandleShape(t *testing.T) {
	installFetchHookContext(t)
	resource := UseCachedResource("users", func(ctx context.Context) (string, error) {
		return "ok", nil
	})

	state := resource.Get()
	if state.Loading || state.Error != nil || state.Ready || state.Stale {
		t.Fatalf("unexpected initial cached resource state: %+v", state)
	}
	resource.Reload()
	resource.Cancel()
	resource.Invalidate()
}

func TestUseCachedResourceDeduplicatesInflightRequests(t *testing.T) {
	installFetchHookContext(t)
	var loads int32
	loader := func(ctx context.Context) (string, error) {
		atomic.AddInt32(&loads, 1)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(25 * time.Millisecond):
			return "shared", nil
		}
	}

	runtime.SetCurrentFiber(&runtime.Fiber{})
	first := UseCachedResource("shared-users", loader)
	runtime.SetCurrentFiber(&runtime.Fiber{})
	second := UseCachedResource("shared-users", loader)

	first.Reload()
	second.Reload()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		firstState := first.Get()
		secondState := second.Get()
		if firstState.Ready && secondState.Ready {
			if firstState.Value != "shared" || secondState.Value != "shared" {
				t.Fatalf("expected both cached resources to share the same value, got %+v and %+v", firstState, secondState)
			}
			if atomic.LoadInt32(&loads) != 1 {
				t.Fatalf("expected exactly one loader call, got %d", atomic.LoadInt32(&loads))
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for cached resource to settle; first=%+v second=%+v", first.Get(), second.Get())
}

func TestInvalidateResourceMarksSnapshotStale(t *testing.T) {
	installFetchHookContext(t)
	runtime.SetCurrentFiber(&runtime.Fiber{})
	resource := UseCachedResource("invalidate-demo", func(ctx context.Context) (string, error) {
		return "fresh", nil
	})

	resource.Set("ready")
	resource.Invalidate()

	state := resource.Get()
	if !state.Ready || !state.Stale || state.Error != nil || state.Value != "ready" {
		t.Fatalf("expected invalidated cached state to preserve ready data and become stale, got %+v", state)
	}
}

func TestCachedResourceUpdateSharesOptimisticValue(t *testing.T) {
	installFetchHookContext(t)
	runtime.SetCurrentFiber(&runtime.Fiber{})
	first := UseCachedResource("optimistic", func(ctx context.Context) (int, error) {
		return 0, nil
	})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	second := UseCachedResource("optimistic", func(ctx context.Context) (int, error) {
		return 0, nil
	})

	first.Set(3)
	second.Update(func(prev int) int {
		return prev + 4
	})

	firstState := first.Get()
	secondState := second.Get()
	if !firstState.Ready || !secondState.Ready || firstState.Value != 7 || secondState.Value != 7 {
		t.Fatalf("expected optimistic update to be shared, got %+v and %+v", firstState, secondState)
	}
}

func TestCachedResourceReloadKeepsStaleValueOnError(t *testing.T) {
	installFetchHookContext(t)
	var failNext atomic.Bool
	loader := func(ctx context.Context) (string, error) {
		if failNext.Load() {
			return "", errors.New("reload failed")
		}
		return "fresh", nil
	}

	runtime.SetCurrentFiber(&runtime.Fiber{})
	resource := UseCachedResource("reload-error", loader)
	resource.Set("cached")
	failNext.Store(true)
	resource.Reload()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state := resource.Get()
		if !state.Loading {
			if state.Value != "cached" || state.Error == nil || !state.Ready || !state.Stale {
				t.Fatalf("expected stale cached value to survive reload error, got %+v", state)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for cached resource reload error state: %+v", resource.Get())
}

func TestFetchUnavailable(t *testing.T) {
	restoreFetch := setGlobalJSValue("fetch", js.Null())
	defer restoreFetch()

	result := waitForResult(t, Fetch("/api/demo", Options{}))
	if result.Err == nil || !strings.Contains(result.Err.Error(), "fetch API unavailable") {
		t.Fatalf("expected unavailable fetch error, got %#v", result.Err)
	}
}

func TestBuildMultipartFormDataAppendsFieldsAndFiles(t *testing.T) {
	var appendCalls []string
	appendFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 {
			entry := args[0].String() + "="
			if args[1].Type() == js.TypeString {
				entry = args[0].String() + "=" + args[1].String()
			} else {
				entry += args[1].Get("name").String()
			}
			appendCalls = append(appendCalls, entry)
		}
		return nil
	})
	defer appendFn.Release()

	formDataCtor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		object := js.Global().Get("Object").New()
		object.Set("append", appendFn)
		return object
	})
	defer formDataCtor.Release()
	restoreFormData := setGlobalJSValue("FormData", formDataCtor)
	defer restoreFormData()

	rawFile := js.Global().Get("Object").New()
	rawFile.Set("name", "avatar.png")
	form, err := buildMultipartFormData(MultipartBody{
		Fields: map[string]string{"title": "Demo"},
		Files:  []MultipartFile{{FieldName: "asset", File: ui.FileFromJSValue(rawFile)}},
	})
	if err != nil {
		t.Fatalf("expected multipart form-data to build, got %v", err)
	}
	if !form.Truthy() {
		t.Fatal("expected multipart form-data js value")
	}
	if len(appendCalls) != 2 {
		t.Fatalf("expected two multipart append calls, got %v", appendCalls)
	}
	if appendCalls[0] != "title=Demo" || appendCalls[1] != "asset=avatar.png" {
		t.Fatalf("unexpected multipart append calls: %v", appendCalls)
	}
}

func TestUploadUnavailableWithoutXMLHttpRequest(t *testing.T) {
	restoreXHR := setGlobalJSValue("XMLHttpRequest", js.Null())
	defer restoreXHR()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	update := <-Upload(ctx, "/api/upload", Options{Body: MultipartBody{}})
	if !update.Done || update.Result.Err == nil || !strings.Contains(update.Result.Err.Error(), "XMLHttpRequest unavailable") {
		t.Fatalf("expected unavailable upload transport error, got %+v", update)
	}
}

func TestFetchHTTPErrorIncludesStatusHeadersAndDecodesJSON(t *testing.T) {
	textFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return makeResolvedPromise(js.ValueOf(`{"message":"invalid upload"}`))
	})
	defer textFn.Release()

	headers := js.Global().Get("Object").New()
	forEachFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		callback := args[0]
		callback.Invoke("application/json", "Content-Type")
		callback.Invoke("trace-123", "X-Trace")
		return nil
	})
	defer forEachFn.Release()
	headers.Set("forEach", forEachFn)

	response := js.Global().Get("Object").New()
	response.Set("status", 422)
	response.Set("statusText", "Unprocessable Entity")
	response.Set("headers", headers)
	response.Set("text", textFn)

	fetchFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return makeResolvedPromise(response)
	})
	defer fetchFn.Release()

	restoreFetch := setGlobalJSValue("fetch", fetchFn)
	defer restoreFetch()

	result := waitForResult(t, Fetch("/api/demo", Options{}))
	if result.Status != 422 {
		t.Fatalf("expected status 422, got %d", result.Status)
	}
	httpErr, ok := result.Err.(HTTPError)
	if !ok {
		t.Fatalf("expected HTTPError, got %#v", result.Err)
	}
	if httpErr.Status != 422 || httpErr.StatusText != "Unprocessable Entity" {
		t.Fatalf("unexpected http error metadata: %+v", httpErr)
	}
	if result.Headers["Content-Type"] != "application/json" || result.Headers["X-Trace"] != "trace-123" {
		t.Fatalf("expected response headers to be preserved, got %#v", result.Headers)
	}
	var payload map[string]string
	if err := result.DecodeJSON(&payload); err != nil {
		t.Fatalf("expected json response to decode, got %v", err)
	}
	if payload["message"] != "invalid upload" {
		t.Fatalf("unexpected decoded payload: %#v", payload)
	}
}

func TestUploadSuccessPreservesResponseMetadataAndIgnoresLateCancel(t *testing.T) {
	var uploadProgressListener js.Value
	var loadListener js.Value
	var setHeaders []string
	var abortCalls int

	uploadAddEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 && args[0].String() == "progress" {
			uploadProgressListener = args[1]
		}
		return nil
	})
	defer uploadAddEventListenerFn.Release()

	addEventListenerFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 && args[0].String() == "load" {
			loadListener = args[1]
		}
		return nil
	})
	defer addEventListenerFn.Release()

	setRequestHeaderFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) >= 2 {
			setHeaders = append(setHeaders, args[0].String()+"="+args[1].String())
		}
		return nil
	})
	defer setRequestHeaderFn.Release()

	abortFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		abortCalls++
		return nil
	})
	defer abortFn.Release()

	getAllResponseHeadersFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return "Content-Type: application/json\r\nX-Upload-ID: up-42\r\n"
	})
	defer getAllResponseHeadersFn.Release()

	sendFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		progressEvent := js.Global().Get("Object").New()
		progressEvent.Set("loaded", 3)
		progressEvent.Set("total", 9)
		progressEvent.Set("lengthComputable", true)
		if uploadProgressListener.Truthy() {
			uploadProgressListener.Invoke(progressEvent)
		}
		loadEvent := js.Global().Get("Object").New()
		loadEvent.Set("loaded", 9)
		loadEvent.Set("total", 9)
		loadEvent.Set("lengthComputable", true)
		if loadListener.Truthy() {
			loadListener.Invoke(loadEvent)
		}
		return nil
	})
	defer sendFn.Release()

	openFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer openFn.Release()

	xhrUpload := js.Global().Get("Object").New()
	xhrUpload.Set("addEventListener", uploadAddEventListenerFn)

	xhr := js.Global().Get("Object").New()
	xhr.Set("status", 201)
	xhr.Set("statusText", "Created")
	xhr.Set("responseText", `{"asset_id":"asset-1"}`)
	xhr.Set("open", openFn)
	xhr.Set("setRequestHeader", setRequestHeaderFn)
	xhr.Set("addEventListener", addEventListenerFn)
	xhr.Set("send", sendFn)
	xhr.Set("abort", abortFn)
	xhr.Set("getAllResponseHeaders", getAllResponseHeadersFn)
	xhr.Set("upload", xhrUpload)

	xhrCtor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return xhr
	})
	defer xhrCtor.Release()
	restoreXHR := setGlobalJSValue("XMLHttpRequest", xhrCtor)
	defer restoreXHR()

	formDataCtor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		object := js.Global().Get("Object").New()
		object.Set("append", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
		return object
	})
	defer formDataCtor.Release()
	restoreFormData := setGlobalJSValue("FormData", formDataCtor)
	defer restoreFormData()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var updates []UploadUpdate
	for update := range Upload(ctx, "/api/upload", Options{
		Method: "POST",
		Headers: map[string]interface{}{
			"X-Test":       "ok",
			"Content-Type": "multipart/form-data",
		},
		Body: MultipartBody{Fields: map[string]string{"title": "Demo"}},
	}) {
		updates = append(updates, update)
	}
	cancel()
	time.Sleep(20 * time.Millisecond)

	if abortCalls != 0 {
		t.Fatalf("expected completed upload to ignore late context cancellation, got %d abort calls", abortCalls)
	}
	if len(updates) != 2 {
		t.Fatalf("expected progress plus final upload update, got %d updates: %+v", len(updates), updates)
	}
	if len(setHeaders) != 1 || setHeaders[0] != "X-Test=ok" {
		t.Fatalf("expected multipart upload to keep custom headers and drop manual content-type, got %v", setHeaders)
	}
	final := updates[len(updates)-1]
	if !final.Done || final.Result.Err != nil {
		t.Fatalf("expected successful final update, got %+v", final)
	}
	if final.Result.Status != 201 || final.Result.Headers["X-Upload-ID"] != "up-42" {
		t.Fatalf("expected upload result metadata, got %+v", final.Result)
	}
	if final.Loaded != 9 || final.Total != 9 || !final.LengthComputable {
		t.Fatalf("expected final upload progress to reflect load event, got %+v", final)
	}
	var payload map[string]string
	if err := final.Result.DecodeJSON(&payload); err != nil {
		t.Fatalf("expected upload json response to decode, got %v", err)
	}
	if payload["asset_id"] != "asset-1" {
		t.Fatalf("unexpected upload payload: %#v", payload)
	}
}

func TestFetchMarshalError(t *testing.T) {
	result := waitForResult(t, Fetch("/api/demo", Options{
		Body: map[string]float64{"bad": math.NaN()},
	}))
	if result.Err == nil || !strings.Contains(result.Err.Error(), "failed to encode body") {
		t.Fatalf("expected marshal error, got %#v", result.Err)
	}
}

func TestFetchSuccess(t *testing.T) {
	var capturedURL string
	var capturedMethod string
	var capturedHeader string
	var capturedBody string

	textFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return makeResolvedPromise(js.ValueOf("response body"))
	})
	defer textFn.Release()

	response := js.Global().Get("Object").New()
	response.Set("status", 200)
	response.Set("statusText", "OK")
	headers := js.Global().Get("Object").New()
	headers.Set("forEach", js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil }))
	response.Set("headers", headers)
	response.Set("text", textFn)

	fetchFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		capturedURL = args[0].String()
		opts := args[1]
		capturedMethod = opts.Get("method").String()
		capturedHeader = opts.Get("headers").Get("X-Test").String()
		capturedBody = opts.Get("body").String()
		return makeResolvedPromise(response)
	})
	defer fetchFn.Release()

	restoreFetch := setGlobalJSValue("fetch", fetchFn)
	defer restoreFetch()

	result := waitForResult(t, Fetch("/api/demo", Options{
		Method:  "POST",
		Headers: map[string]interface{}{"X-Test": "ok"},
		Body:    "body",
	}))

	if result.Err != nil {
		t.Fatalf("expected successful fetch result, got error %v", result.Err)
	}
	if result.Data != "response body" {
		t.Fatalf("expected response body, got %#v", result.Data)
	}
	if capturedURL != "/api/demo" || capturedMethod != "POST" || capturedHeader != "ok" || capturedBody != "body" {
		t.Fatalf("unexpected fetch invocation: url=%q method=%q header=%q body=%q", capturedURL, capturedMethod, capturedHeader, capturedBody)
	}
}

func TestReturnChannelIsNoOp(t *testing.T) {
	ch := make(chan Result, 1)
	ReturnChannel(ch)
	ch <- Result{Data: "ok"}
	if result := <-ch; result.Data != "ok" {
		t.Fatalf("expected channel contents to remain untouched, got %#v", result)
	}
}

func BenchmarkBuildMultipartFormData(b *testing.B) {
	appendFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
	defer appendFn.Release()

	formDataCtor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		object := js.Global().Get("Object").New()
		object.Set("append", appendFn)
		return object
	})
	defer formDataCtor.Release()
	restoreFormData := setGlobalJSValue("FormData", formDataCtor)
	defer restoreFormData()

	rawFile := js.Global().Get("Object").New()
	rawFile.Set("name", "hero.png")
	body := MultipartBody{
		Fields: map[string]string{
			"title":    "Demo",
			"audience": "buyers",
		},
		Files: []MultipartFile{{FieldName: "asset", File: ui.FileFromJSValue(rawFile)}},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		form, err := buildMultipartFormData(body)
		if err != nil {
			b.Fatal(err)
		}
		if !form.Truthy() {
			b.Fatal("expected form data result")
		}
	}
}
