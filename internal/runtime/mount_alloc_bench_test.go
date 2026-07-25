package runtime

import (
	"fmt"
	"testing"
	"unsafe"
)

// Mount-path allocation guard.
//
// Double-buffering makes UPDATES allocation-free — acquireWorkInProgress reuses
// the previous generation's fiber. A freshly MOUNTED node has no alternate to
// reuse, so mount was the path that allocated, and profiling a 200-row mount put
// fiber allocation at 98.5% of allocated bytes and ~36% of CPU in allocation
// and GC.
//
// This benchmark exists to keep that fixed. It builds the element tree ONCE,
// outside the timed loop, because an earlier version built it inside and spent
// 98% of its profile in tree construction — measuring the benchmark rather than
// the reconciler.

func buildMountBenchTree(parseRows int) []any {
	parseChildren := make([]any, 0, parseRows)
	for parseIndex := range parseRows {
		parseChildren = append(parseChildren, &Element{
			Type: "div",
			Props: map[string]any{
				"class": "row",
				"key":   fmt.Sprintf("k%d", parseIndex),
				"children": []any{
					&Element{Type: "span", Props: map[string]any{"class": "label", "children": []any{"name"}}},
					&Element{Type: "span", Props: map[string]any{"class": "value", "children": []any{"42"}}},
				},
			},
		})
	}
	return parseChildren
}

// BenchmarkMountFromEmpty200 reconciles a 600-node tree from nothing.
//
// With slab allocation this reports a handful of allocations rather than one per
// mounted node. A regression shows up here as the allocation count tracking the
// node count again.
func BenchmarkMountFromEmpty200(parseB *testing.B) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseElements := buildMountBenchTree(200)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseB.Loop() {
		parseParent := &Fiber{alternate: &Fiber{}}
		parseRt.deletions = parseRt.deletions[:0]
		parseRt.reconcileChildren(parseParent, parseElements)
	}
}

// TestMountAllocationsDoNotTrackNodeCount pins the property the slab exists for.
//
// A per-node allocation strategy makes this scale with the tree; slab allocation
// keeps it near the slab count. The bound is deliberately loose — it is checking
// an order of magnitude, not a constant.
func TestMountAllocationsDoNotTrackNodeCount(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseElements := buildMountBenchTree(200)

	parseAllocs := testing.AllocsPerRun(20, func() {
		parseParent := &Fiber{alternate: &Fiber{}}
		parseRt.deletions = parseRt.deletions[:0]
		parseRt.reconcileChildren(parseParent, parseElements)
	})

	parseT.Logf("mounting 600 nodes allocated %.0f times (fiber = %d bytes)",
		parseAllocs, int(unsafe.Sizeof(Fiber{})))

	// 600 nodes per mount. Per-node allocation would put this in the hundreds.
	if parseAllocs > 60 {
		parseT.Errorf("mounting 600 nodes allocated %.0f times; allocation is tracking node count again", parseAllocs)
	}
}
