//go:build js && wasm
// +build js,wasm

package runtime

import (
	"syscall/js"
	"testing"
	"time"
)

func BenchmarkGoUseFetchInit(parseB *testing.B) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseRt.currentRoot = &Fiber{typeOf: "ROOT"}
	InitGlobalRuntime(Config{DOMAdapter: parseRt.domAdapter, Scheduler: parseRt.scheduler})
	parseFiber := &Fiber{typeOf: "bench", props: make(map[string]interface{})}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseFiber.hooks = nil
		_, _ = GoUseFetch("/api/test")
	}
}

func BenchmarkGoUseFetchRefetchUnavailable(parseB *testing.B) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseRt.currentRoot = &Fiber{typeOf: "ROOT"}
	InitGlobalRuntime(Config{DOMAdapter: parseRt.domAdapter, Scheduler: parseRt.scheduler})
	parseFiber := &Fiber{typeOf: "bench", props: make(map[string]interface{})}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseRestoreFetch := setGlobalJSValue("fetch", js.Null())
	defer parseRestoreFetch()

	parseGetter, parseRefetch := GoUseFetch("/api/test")

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseRt.updateScheduled = false
		parseRefetch()
		parseDeadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(parseDeadline) {
			if !parseGetter().Loading {
				break
			}
			time.Sleep(time.Millisecond)
		}
	}
}
