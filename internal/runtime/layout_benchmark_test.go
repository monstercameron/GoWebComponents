package runtime

import (
	"testing"
	"unsafe"
)

var (
	benchFiberBoolSink  bool
	benchFiberIntSink   int
	benchFiberIfaceSink interface{}
)

func BenchmarkRuntimeLayoutBaselines(b *testing.B) {
	b.ReportMetric(float64(unsafe.Sizeof(Fiber{})), "fiber-bytes")
	b.ReportMetric(float64(unsafe.Alignof(Fiber{})), "fiber-align")
	b.ReportMetric(float64(unsafe.Sizeof(Hooks{})), "hooks-bytes")
	b.ReportMetric(float64(unsafe.Alignof(Hooks{})), "hooks-align")
	b.ReportMetric(float64(unsafe.Sizeof(Element{})), "element-bytes")
	b.ReportMetric(float64(unsafe.Sizeof(FetchState{})), "fetchstate-bytes")

	for i := 0; i < b.N; i++ {
	}
}

func BenchmarkFiberHotFieldScan256(b *testing.B) {
	fibers := make([]Fiber, 256)
	for index := range fibers {
		fibers[index].dirty = index%2 == 0
		fibers[index].needsUpdate = index%3 == 0
		fibers[index].typeOf = "div"
		if index > 0 {
			fibers[index].parent = &fibers[index-1]
		}
		if index+1 < len(fibers) {
			fibers[index].sibling = &fibers[index+1]
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dirtyCount := 0
		var lastType interface{}
		var anyNeedsUpdate bool
		for index := range fibers {
			fiber := &fibers[index]
			if fiber.parent != nil && fiber.dirty {
				dirtyCount++
			}
			if fiber.sibling != nil && fiber.needsUpdate {
				anyNeedsUpdate = true
			}
			lastType = fiber.typeOf
		}
		benchFiberIntSink = dirtyCount
		benchFiberIfaceSink = lastType
		benchFiberBoolSink = anyNeedsUpdate
	}
}

func BenchmarkFiberSiblingWalk256(b *testing.B) {
	fibers := make([]Fiber, 256)
	for index := range fibers {
		fibers[index].typeOf = "div"
		fibers[index].dirty = index%2 == 0
		if index+1 < len(fibers) {
			fibers[index].sibling = &fibers[index+1]
		}
	}
	head := &fibers[0]

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		var lastType interface{}
		for fiber := head; fiber != nil; fiber = fiber.sibling {
			if fiber.dirty {
				count++
			}
			lastType = fiber.typeOf
		}
		benchFiberIntSink = count
		benchFiberIfaceSink = lastType
	}
}

func BenchmarkHooksIndexResetHotPath(b *testing.B) {
	hooks := &Hooks{
		states:    make([]interface{}, 16),
		deps:      make([][]interface{}, 8),
		memos:     make([]memoizedValue, 4),
		callbacks: make([]callbackValue, 4),
		refs:      make([]*RefValue, 2),
		ids:       make([]string, 2),
		fetches:   make([]fetchValue, 2),
		funcs:     make([]funcHandlerValue, 2),
		cleanups:  make([]func(), 2),
		atoms:     make([]string, 2),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hooks.index = 7
		hooks.stateIndex = 3
		hooks.depIndex = 2
		hooks.memoIndex = 1
		hooks.callbackIndex = 1
		hooks.refIndex = 1
		hooks.idIndex = 1
		hooks.fetchIndex = 1
		hooks.funcIndex = 1
		hooks.atomIndex = 1
		hooks.cleanupIndex = 1

		resetHookRenderState(&Fiber{hooks: hooks})
		benchFiberIntSink = hooks.index + hooks.stateIndex + hooks.depIndex + hooks.memoIndex
	}
}
