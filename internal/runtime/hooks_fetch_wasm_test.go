//go:build js && wasm
// +build js,wasm

package runtime

import (
	"syscall/js"
	"testing"
	"time"
)

func setGlobalJSValue(parseName string, parseValue interface{}) func() {
	parseGlobal := js.Global()
	parsePrev := parseGlobal.Get(parseName)
	parseGlobal.Set(parseName, parseValue)
	return func() {
		parseGlobal.Set(parseName, parsePrev)
	}
}

func makeResolvedPromise(parseValue js.Value) js.Value {
	return js.Global().Get("Promise").Call("resolve", parseValue)
}

func waitForFetchState(parseT *testing.T, parseGetter func() FetchState, parseCond func(FetchState) bool) FetchState {
	parseT.Helper()
	parseDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) {
		parseState := parseGetter()
		if parseCond(parseState) {
			return parseState
		}
		time.Sleep(10 * time.Millisecond)
	}
	parseState2 := parseGetter()
	parseT.Fatalf("timed out waiting for fetch state, last state: %+v", parseState2)
	return FetchState{}
}

func initWasmFetchTestRuntime() (*Runtime, *Fiber) {
	resetGlobalRuntimeForTest()
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseRt.currentRoot = &Fiber{typeOf: "ROOT"}
	InitGlobalRuntime(Config{DOMAdapter: parseRt.domAdapter, Scheduler: parseRt.scheduler})
	parseFiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(parseFiber)
	return parseRt, parseFiber
}

func TestGoUseFetch_InitialStateAndLoading(parseT *testing.T) {
	_, _ = initWasmFetchTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	parseRestoreFetch := setGlobalJSValue("fetch", js.Null())
	defer parseRestoreFetch()

	parseGetter, parseRefetch := GoUseFetch("/api/test")

	parseInitial := parseGetter()
	if parseInitial.Loading || parseInitial.Error != "" || parseInitial.Data != nil {
		parseT.Fatalf("unexpected initial fetch state: %+v", parseInitial)
	}

	parseRefetch()

	parseLoading := parseGetter()
	if !parseLoading.Loading {
		parseT.Fatalf("expected loading state immediately after refetch, got %+v", parseLoading)
	}

	_ = waitForFetchState(parseT, parseGetter, func(parseState FetchState) bool { return !parseState.Loading })
}

func TestGoUseFetch_Success(parseT *testing.T) {
	_, _ = initWasmFetchTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	parseTextFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return makeResolvedPromise(js.ValueOf("payload"))
	})
	defer parseTextFn.Release()

	parseResponse := js.Global().Get("Object").New()
	parseResponse.Set("ok", true)
	parseResponse.Set("text", parseTextFn)

	parseFetchFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		return makeResolvedPromise(parseResponse)
	})
	defer parseFetchFn.Release()

	parseRestoreFetch := setGlobalJSValue("fetch", parseFetchFn)
	defer parseRestoreFetch()

	parseGetter, parseRefetch := GoUseFetch("/api/test")
	parseRefetch()

	parseState := waitForFetchState(parseT, parseGetter, func(parseState2 FetchState) bool { return !parseState2.Loading })
	if parseState.Error != "" {
		parseT.Fatalf("expected empty error on success, got %q", parseState.Error)
	}
	if parseData, parseOk := parseState.Data.(string); !parseOk || parseData != "payload" {
		parseT.Fatalf("expected payload data, got %#v", parseState.Data)
	}
}

func TestGoUseFetch_NonOKResponse(parseT *testing.T) {
	_, _ = initWasmFetchTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	parseResponse := js.Global().Get("Object").New()
	parseResponse.Set("ok", false)
	parseResponse.Set("status", 500)
	parseResponse.Set("statusText", "Server Error")

	parseFetchFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return makeResolvedPromise(parseResponse)
	})
	defer parseFetchFn.Release()

	parseRestoreFetch := setGlobalJSValue("fetch", parseFetchFn)
	defer parseRestoreFetch()

	parseGetter, parseRefetch := GoUseFetch("/api/test")
	parseRefetch()

	parseState := waitForFetchState(parseT, parseGetter, func(parseState2 FetchState) bool { return !parseState2.Loading })
	if parseState.Error != "Fetch failed: 500 Server Error" {
		parseT.Fatalf("expected formatted non-ok error, got %q", parseState.Error)
	}
	if parseState.Data != nil {
		parseT.Fatalf("expected nil data on HTTP error, got %#v", parseState.Data)
	}
}

func TestGoUseFetch_FetchUnavailable(parseT *testing.T) {
	_, _ = initWasmFetchTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	parseRestoreFetch := setGlobalJSValue("fetch", js.Null())
	defer parseRestoreFetch()

	parseGetter, parseRefetch := GoUseFetch("/api/test")
	parseRefetch()

	parseState := waitForFetchState(parseT, parseGetter, func(parseState2 FetchState) bool { return !parseState2.Loading })
	if parseState.Error != "fetch API unavailable" {
		parseT.Fatalf("expected unavailable fetch error, got %q", parseState.Error)
	}
}

func TestGoUseFetch_RejectedPromise(parseT *testing.T) {
	_, _ = initWasmFetchTestRuntime()
	defer func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	}()

	parsePromise := js.Global().Get("Object").New()
	parseThenFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return parsePromise
	})
	defer parseThenFn.Release()
	parseCatchFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseHandler := parseArgs2[0]
		var parseTimeoutFn js.Func
		parseTimeoutFn = js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
			defer parseTimeoutFn.Release()
			parseHandler.Invoke(js.ValueOf("boom"))
			return nil
		})
		js.Global().Call("setTimeout", parseTimeoutFn, 0)
		return parsePromise
	})
	defer parseCatchFn.Release()
	parsePromise.Set("then", parseThenFn)
	parsePromise.Set("catch", parseCatchFn)

	parseFetchFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		return parsePromise
	})
	defer parseFetchFn.Release()

	parseRestoreFetch := setGlobalJSValue("fetch", parseFetchFn)
	defer parseRestoreFetch()

	parseGetter, parseRefetch := GoUseFetch("/api/test")
	parseRefetch()

	parseState := waitForFetchState(parseT, parseGetter, func(parseState2 FetchState) bool { return !parseState2.Loading })
	if parseState.Error != "Fetch failed" {
		parseT.Fatalf("expected rejected promise error, got %q", parseState.Error)
	}
}
