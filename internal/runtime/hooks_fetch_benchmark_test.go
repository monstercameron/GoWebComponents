//go:build js && wasm
// +build js,wasm

package runtime

import (
	"syscall/js"
	"testing"
	"time"
)

func BenchmarkGoUseFetchInit(b *testing.B) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	rt.currentRoot = &Fiber{typeOf: "ROOT"}
	InitGlobalRuntime(Config{DOMAdapter: rt.domAdapter, Scheduler: rt.scheduler})
	fiber := &Fiber{typeOf: "bench", props: make(map[string]interface{})}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fiber.hooks = nil
		_, _ = GoUseFetch("/api/test")
	}
}

func BenchmarkGoUseFetchRefetchUnavailable(b *testing.B) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	rt.currentRoot = &Fiber{typeOf: "ROOT"}
	InitGlobalRuntime(Config{DOMAdapter: rt.domAdapter, Scheduler: rt.scheduler})
	fiber := &Fiber{typeOf: "bench", props: make(map[string]interface{})}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	restoreFetch := setGlobalJSValue("fetch", js.Null())
	defer restoreFetch()

	getter, refetch := GoUseFetch("/api/test")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.updateScheduled = false
		refetch()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if !getter().Loading {
				break
			}
			time.Sleep(time.Millisecond)
		}
	}
}
