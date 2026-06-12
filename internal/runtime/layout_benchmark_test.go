package runtime

import (
	"testing"
	"unsafe"
)

var (
	benchFiberBoolSink  bool
	benchFiberIntSink   int
	benchFiberIfaceSink any
)

func BenchmarkRuntimeLayoutBaselines(parseB *testing.B) {
	parseB.ReportMetric(float64(unsafe.Sizeof(Fiber{})), "fiber-bytes")
	parseB.ReportMetric(float64(unsafe.Alignof(Fiber{})), "fiber-align")
	parseB.ReportMetric(float64(unsafe.Sizeof(Hooks{})), "hooks-bytes")
	parseB.ReportMetric(float64(unsafe.Alignof(Hooks{})), "hooks-align")
	parseB.ReportMetric(float64(unsafe.Sizeof(Element{})), "element-bytes")
	parseB.ReportMetric(float64(unsafe.Sizeof(FetchState{})), "fetchstate-bytes")

	for parseI := 0; parseI < parseB.N; parseI++ {
	}
}

func BenchmarkFiberHotFieldScan256(parseB *testing.B) {
	parseFibers := make([]Fiber, 256)
	for parseIndex := range parseFibers {
		parseFibers[parseIndex].dirty = parseIndex%2 == 0
		parseFibers[parseIndex].needsUpdate = parseIndex%3 == 0
		parseFibers[parseIndex].typeOf = "div"
		if parseIndex > 0 {
			parseFibers[parseIndex].parent = &parseFibers[parseIndex-1]
		}
		if parseIndex+1 < len(parseFibers) {
			parseFibers[parseIndex].sibling = &parseFibers[parseIndex+1]
		}
	}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseDirtyCount := 0
		var parseLastType any
		var isAnyNeedsUpdate bool
		for parseIndex2 := range parseFibers {
			parseFiber := &parseFibers[parseIndex2]
			if parseFiber.parent != nil && parseFiber.dirty {
				parseDirtyCount++
			}
			if parseFiber.sibling != nil && parseFiber.needsUpdate {
				isAnyNeedsUpdate = true
			}
			parseLastType = parseFiber.typeOf
		}
		benchFiberIntSink = parseDirtyCount
		benchFiberIfaceSink = parseLastType
		benchFiberBoolSink = isAnyNeedsUpdate
	}
}

func BenchmarkFiberSiblingWalk256(parseB *testing.B) {
	parseFibers := make([]Fiber, 256)
	for parseIndex := range parseFibers {
		parseFibers[parseIndex].typeOf = "div"
		parseFibers[parseIndex].dirty = parseIndex%2 == 0
		if parseIndex+1 < len(parseFibers) {
			parseFibers[parseIndex].sibling = &parseFibers[parseIndex+1]
		}
	}
	parseHead := &parseFibers[0]

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseCount := 0
		var parseLastType any
		for parseFiber := parseHead; parseFiber != nil; parseFiber = parseFiber.sibling {
			if parseFiber.dirty {
				parseCount++
			}
			parseLastType = parseFiber.typeOf
		}
		benchFiberIntSink = parseCount
		benchFiberIfaceSink = parseLastType
	}
}

func BenchmarkHooksIndexResetHotPath(parseB *testing.B) {
	parseHooks := &Hooks{
		states:    make([]any, 16),
		deps:      make([][]any, 8),
		memos:     make([]memoizedValue, 4),
		callbacks: make([]callbackValue, 4),
		refs:      make([]*RefValue, 2),
		ids:       make([]string, 2),
		fetches:   make([]fetchValue, 2),
		funcs:     make([]funcHandlerValue, 2),
		cleanups:  make([]func(), 2),
		atoms:     make([]string, 2),
	}

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseHooks.index = 7
		parseHooks.stateIndex = 3
		parseHooks.depIndex = 2
		parseHooks.memoIndex = 1
		parseHooks.callbackIndex = 1
		parseHooks.refIndex = 1
		parseHooks.idIndex = 1
		parseHooks.fetchIndex = 1
		parseHooks.funcIndex = 1
		parseHooks.atomIndex = 1
		parseHooks.cleanupIndex = 1

		resetHookRenderState(&Fiber{hooks: parseHooks})
		benchFiberIntSink = parseHooks.index + parseHooks.stateIndex + parseHooks.depIndex + parseHooks.memoIndex
	}
}
