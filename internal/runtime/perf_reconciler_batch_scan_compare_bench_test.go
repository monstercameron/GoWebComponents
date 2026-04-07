package runtime

import "testing"

// buildPlacementBatchBenchmarkTree creates one domless-heavy child subtree so the batch-eligibility benchmark measures threshold scanning instead of direct host-only siblings.
func buildPlacementBatchBenchmarkTree(parseAdapter DOMAdapter, parseLeafCount int) *Fiber {
	parseWrapperA := &Fiber{typeOf: func() *Element { return nil }}
	parseWrapperB := &Fiber{typeOf: func() *Element { return nil }, parent: parseWrapperA}
	parseWrapperA.child = parseWrapperB
	var parsePrev *Fiber
	for parseIndex := 0; parseIndex < parseLeafCount; parseIndex++ {
		parseLeaf := &Fiber{
			typeOf:    "div",
			dom:       parseAdapter.CreateElement("div"),
			effectTag: "PLACEMENT",
			parent:    parseWrapperB,
		}
		if parseWrapperB.child == nil {
			parseWrapperB.child = parseLeaf
		} else {
			parsePrev.sibling = parseLeaf
		}
		parsePrev = parseLeaf
	}
	return parseWrapperA
}

// countCommittedPlacementChildrenLegacy counts every DOM-bearing placement in one subtree without the current threshold short-circuit.
func countCommittedPlacementChildrenLegacy(parseRuntime *Runtime, parseFiber *Fiber) int {
	parseCount := 0
	for parseFiber != nil {
		if parseRuntime.isPortalFiber(parseFiber) {
			parseFiber = parseFiber.sibling
			continue
		}
		if !IsDOMNodeNull(parseFiber.dom) {
			if parseFiber.effectTag == "PLACEMENT" {
				parseCount++
			}
			parseFiber = parseFiber.sibling
			continue
		}
		if parseFiber.child != nil {
			parseCount += countCommittedPlacementChildrenLegacy(parseRuntime, parseFiber.child)
		}
		parseFiber = parseFiber.sibling
	}
	return parseCount
}

func BenchmarkHasCommittedPlacementChildrenAtLeastCurrentVsLegacy(parseB *testing.B) {
	parseRuntime := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseRoot := buildPlacementBatchBenchmarkTree(parseRuntime.domAdapter, 512)

	parseB.Run("current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if !parseRuntime.hasCommittedPlacementChildrenAtLeast(parseRoot.child, 2) {
				parseB.Fatal("expected placement threshold to succeed")
			}
		}
	})

	parseB.Run("legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if countCommittedPlacementChildrenLegacy(parseRuntime, parseRoot.child) < 2 {
				parseB.Fatal("expected placement threshold to succeed")
			}
		}
	})
}
