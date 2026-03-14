//go:build js && wasm
// +build js,wasm

package runtime

import (
	"syscall/js"
	"testing"
	"time"
)

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

func makeRejectedPromise(value js.Value) js.Value {
	return js.Global().Get("Promise").Call("reject", value)
}

func makeDeferredRejectedPromise(value js.Value) js.Value {
	var timeoutFunc js.Func
	executor := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		reject := args[1]
		timeoutFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			reject.Invoke(value)
			timeoutFunc.Release()
			return nil
		})
		js.Global().Call("setTimeout", timeoutFunc, 0)
		return nil
	})
	defer executor.Release()
	return js.Global().Get("Promise").New(executor)
}

func waitForFetchState(t *testing.T, getter func() FetchState, cond func(FetchState) bool) FetchState {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state := getter()
		if cond(state) {
			return state
		}
		time.Sleep(10 * time.Millisecond)
	}
	state := getter()
	t.Fatalf("timed out waiting for fetch state, last state: %+v", state)
	return FetchState{}
}

func initWasmFetchTestRuntime() (*Runtime, *Fiber) {
	resetGlobalRuntimeForTest()
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	rt.currentRoot = &Fiber{typeOf: "ROOT"}
	InitGlobalRuntime(Config{DOMAdapter: rt.domAdapter, Scheduler: rt.scheduler})
	fiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(fiber)
	return rt, fiber
}

func TestGoUseFetch_InitialStateAndLoading(t *testing.T) {
	_, _ = initWasmFetchTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	restoreFetch := setGlobalJSValue("fetch", js.Null())
	defer restoreFetch()

	getter, refetch := GoUseFetch("/api/test")

	initial := getter()
	if initial.Loading || initial.Error != "" || initial.Data != nil {
		t.Fatalf("unexpected initial fetch state: %+v", initial)
	}

	refetch()

	loading := getter()
	if !loading.Loading {
		t.Fatalf("expected loading state immediately after refetch, got %+v", loading)
	}

	_ = waitForFetchState(t, getter, func(state FetchState) bool { return !state.Loading })
}

func TestGoUseFetch_Success(t *testing.T) {
	_, _ = initWasmFetchTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	textFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return makeResolvedPromise(js.ValueOf("payload"))
	})
	defer textFn.Release()

	response := js.Global().Get("Object").New()
	response.Set("ok", true)
	response.Set("text", textFn)

	fetchFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return makeResolvedPromise(response)
	})
	defer fetchFn.Release()

	restoreFetch := setGlobalJSValue("fetch", fetchFn)
	defer restoreFetch()

	getter, refetch := GoUseFetch("/api/test")
	refetch()

	state := waitForFetchState(t, getter, func(state FetchState) bool { return !state.Loading })
	if state.Error != "" {
		t.Fatalf("expected empty error on success, got %q", state.Error)
	}
	if data, ok := state.Data.(string); !ok || data != "payload" {
		t.Fatalf("expected payload data, got %#v", state.Data)
	}
}

func TestGoUseFetch_NonOKResponse(t *testing.T) {
	_, _ = initWasmFetchTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	response := js.Global().Get("Object").New()
	response.Set("ok", false)
	response.Set("status", 500)
	response.Set("statusText", "Server Error")

	fetchFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return makeResolvedPromise(response)
	})
	defer fetchFn.Release()

	restoreFetch := setGlobalJSValue("fetch", fetchFn)
	defer restoreFetch()

	getter, refetch := GoUseFetch("/api/test")
	refetch()

	state := waitForFetchState(t, getter, func(state FetchState) bool { return !state.Loading })
	if state.Error != "Fetch failed: 500 Server Error" {
		t.Fatalf("expected formatted non-ok error, got %q", state.Error)
	}
	if state.Data != nil {
		t.Fatalf("expected nil data on HTTP error, got %#v", state.Data)
	}
}

func TestGoUseFetch_FetchUnavailable(t *testing.T) {
	_, _ = initWasmFetchTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	restoreFetch := setGlobalJSValue("fetch", js.Null())
	defer restoreFetch()

	getter, refetch := GoUseFetch("/api/test")
	refetch()

	state := waitForFetchState(t, getter, func(state FetchState) bool { return !state.Loading })
	if state.Error != "fetch API unavailable" {
		t.Fatalf("expected unavailable fetch error, got %q", state.Error)
	}
}

func TestGoUseFetch_RejectedPromise(t *testing.T) {
	_, _ = initWasmFetchTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	promise := js.Global().Get("Object").New()
	thenFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return promise
	})
	defer thenFn.Release()
	catchFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler := args[0]
		var timeoutFn js.Func
		timeoutFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			defer timeoutFn.Release()
			handler.Invoke(js.ValueOf("boom"))
			return nil
		})
		js.Global().Call("setTimeout", timeoutFn, 0)
		return promise
	})
	defer catchFn.Release()
	promise.Set("then", thenFn)
	promise.Set("catch", catchFn)

	fetchFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return promise
	})
	defer fetchFn.Release()

	restoreFetch := setGlobalJSValue("fetch", fetchFn)
	defer restoreFetch()

	getter, refetch := GoUseFetch("/api/test")
	refetch()

	state := waitForFetchState(t, getter, func(state FetchState) bool { return !state.Loading })
	if state.Error != "Fetch failed" {
		t.Fatalf("expected rejected promise error, got %q", state.Error)
	}
}
