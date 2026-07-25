package runtime

import (
	"fmt"
	"testing"
)

// v5 P5.1 — component render share, the measurement that decides Phase 5.
//
// runtime2 exists to ship COMPONENT RENDERING to workers. Whether that can pay
// is an Amdahl question: if component bodies are a small share of a render pass,
// parallelizing them cannot move the total much no matter how many workers are
// thrown at it. The plan sets the bar at 35% and predicts the answer is below
// it, which would retire runtime2 as a renderer.
//
// The share is measured by DIFFERENCE, which is the part worth explaining.
// Instrumenting the runtime to time component invocations would add overhead to
// the thing being measured and would need the instrumentation to be trusted.
// Instead the same tree is rendered twice, with identical structure and
// identical output, differing only in how much work each component body does.
// The delta over the added work isolates the body share without instrumenting
// anything.

// componentBodyWork is a fixed, non-eliminable unit of work for a component
// body to do.
//
// Returned and consumed so the compiler cannot remove it, and arithmetic rather
// than allocation so it measures CPU rather than the allocator — the allocator's
// cost would land in both arms anyway and cancel, but only after adding noise.
//
//go:noinline
func componentBodyWork(parseIterations int) int {
	parseAccumulator := 0
	for parseIndex := range parseIterations {
		parseAccumulator = parseAccumulator*31 + parseIndex
		parseAccumulator ^= parseAccumulator >> 7
	}
	return parseAccumulator
}

// buildRenderShareTree builds a tree of host elements, one per "component",
// each having done parseWorkPerBody units of work to produce itself.
func buildRenderShareTree(parseComponentCount int, parseWorkPerBody int) *Element {
	parseChildren := make([]any, 0, parseComponentCount)
	for parseIndex := range parseComponentCount {
		parseAccumulator := 0
		if parseWorkPerBody > 0 {
			parseAccumulator = componentBodyWork(parseWorkPerBody)
		}
		parseChildren = append(parseChildren, &Element{
			Type: "div",
			Props: map[string]any{
				"class": "row",
				"key":   fmt.Sprintf("k%d", parseIndex),
				// The accumulator is written into the output so it cannot be
				// optimized away, and so both arms produce structurally identical
				// trees differing only in one attribute's value.
				"data-v":   parseAccumulator & 0xff,
				"children": []any{&Element{Type: "span"}},
			},
		})
	}
	return &Element{Type: "div", Props: map[string]any{"children": parseChildren}}
}

// benchmarkRenderShare runs one full reconcile pass over a freshly built tree.
func benchmarkRenderShare(parseB *testing.B, parseComponentCount int, parseWorkPerBody int) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseB.Loop() {
		parseTree := buildRenderShareTree(parseComponentCount, parseWorkPerBody)
		parseParent := &Fiber{alternate: &Fiber{}}
		parseRt.deletions = parseRt.deletions[:0]
		parseRt.reconcileChildren(parseParent, []any{parseTree})
	}
}

// BenchmarkRenderShare_TrivialBodies is the baseline arm: components that do no
// work beyond producing their element.
func BenchmarkRenderShare_TrivialBodies(parseB *testing.B) {
	benchmarkRenderShare(parseB, 200, 0)
}

// BenchmarkRenderShare_LightBodies models a typical component: a little
// formatting, a couple of conditionals.
func BenchmarkRenderShare_LightBodies(parseB *testing.B) {
	benchmarkRenderShare(parseB, 200, 50)
}

// BenchmarkRenderShare_HeavyBodies models an unusually expensive component —
// the case most favourable to runtime2's premise.
func BenchmarkRenderShare_HeavyBodies(parseB *testing.B) {
	benchmarkRenderShare(parseB, 200, 500)
}

// BenchmarkRenderShare_BodyWorkAlone measures the body work with no rendering
// at all, so the share can be computed rather than inferred.
func BenchmarkRenderShare_BodyWorkAlone(parseB *testing.B) {
	parseB.Run("per-body-50", func(parseSub *testing.B) {
		parseSub.ResetTimer()
		for parseSub.Loop() {
			for range 200 {
				if componentBodyWork(50) == -1 {
					parseSub.Fatal("impossible")
				}
			}
		}
	})
	parseB.Run("per-body-500", func(parseSub *testing.B) {
		parseSub.ResetTimer()
		for parseSub.Loop() {
			for range 200 {
				if componentBodyWork(500) == -1 {
					parseSub.Fatal("impossible")
				}
			}
		}
	})
}
