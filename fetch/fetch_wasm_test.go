//go:build js && wasm
// +build js,wasm

package fetch

import (
	"math"
	"strings"
	"syscall/js"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
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

func TestFetchUnavailable(t *testing.T) {
	restoreFetch := setGlobalJSValue("fetch", js.Null())
	defer restoreFetch()

	result := waitForResult(t, Fetch("/api/demo", Options{}))
	if result.Err == nil || !strings.Contains(result.Err.Error(), "fetch API unavailable") {
		t.Fatalf("expected unavailable fetch error, got %#v", result.Err)
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
