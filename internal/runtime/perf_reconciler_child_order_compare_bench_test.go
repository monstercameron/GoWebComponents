package runtime

import "testing"

var benchmarkKeptChildren []bool
var benchmarkKeptCount int

// buildChildOrderBenchmarkFixture creates one reversed keyed child order so the child-order repair benchmark measures move-heavy work.
func buildChildOrderBenchmarkFixture(getCount int) (*Runtime, DOMNode, []DOMNode) {
	getAdapter := newTestDOMAdapter()
	getRuntime := NewRuntime(Config{DOMAdapter: getAdapter})
	getParent := getAdapter.CreateElement("section")
	getExpected := make([]DOMNode, 0, getCount)
	getObserved := make([]DOMNode, 0, getCount)

	for range getCount {
		getNode := getAdapter.CreateElement("div")
		getExpected = append(getExpected, getNode)
		getObserved = append(getObserved, getNode)
	}
	for getIndex := len(getObserved) - 1; getIndex >= 0; getIndex-- {
		getAdapter.AppendChild(getParent, getObserved[getIndex])
	}

	return getRuntime, getParent, getExpected
}

// applyLegacyCommittedChildOrder replays the previous child-order repair so the benchmark can compare the new synchronized observed-order path against the old rescan-heavy flow.
func applyLegacyCommittedChildOrder(getRuntime *Runtime, getParent DOMNode, getExpected []DOMNode) {
	if getRuntime == nil || getRuntime.domAdapter == nil || IsDOMNodeNull(getParent) || len(getExpected) == 0 {
		return
	}
	for getIndex, getExpectedNode := range getExpected {
		getObserved := getRuntime.buildObservedChildNodes(getParent, 4)
		if getIndex < len(getObserved) && IsSameDOMNode(getObserved[getIndex], getExpectedNode) {
			continue
		}

		getAttached := false
		for _, getObservedNode := range getObserved {
			if IsSameDOMNode(getObservedNode, getExpectedNode) {
				getAttached = true
				break
			}
		}
		if getAttached {
			getRuntime.domAdapter.RemoveChild(getParent, getExpectedNode)
			getObserved = getRuntime.buildObservedChildNodes(getParent, 4)
		}

		if getIndex < len(getObserved) {
			getRuntime.domAdapter.InsertBefore(getParent, getExpectedNode, getObserved[getIndex])
			continue
		}
		getRuntime.domAdapter.AppendChild(getParent, getExpectedNode)
	}
}

func BenchmarkApplyCommittedChildOrderCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("current", func(parseB *testing.B) {
		for parseB.Loop() {
			parseB.StopTimer()
			getRuntime, getParent, getExpected := buildChildOrderBenchmarkFixture(128)
			parseB.StartTimer()

			getRuntime.applyCommittedChildOrder(getParent, getExpected)
		}
	})

	parseB.Run("legacy", func(parseB *testing.B) {
		for parseB.Loop() {
			parseB.StopTimer()
			getRuntime, getParent, getExpected := buildChildOrderBenchmarkFixture(128)
			parseB.StartTimer()

			applyLegacyCommittedChildOrder(getRuntime, getParent, getExpected)
		}
	})
}

// BenchmarkMarkCommittedChildOrderKept128 measures LIS scratch allocation for one reordered 128-child list.
func BenchmarkMarkCommittedChildOrderKept128(parseB *testing.B) {
	parseMatch := make([]int, 128)
	for parseIndex := range parseMatch {
		parseMatch[parseIndex] = len(parseMatch) - parseIndex - 1
	}
	parseB.ReportAllocs()
	for parseB.Loop() {
		benchmarkKeptChildren, benchmarkKeptCount =
			markCommittedChildOrderKept(parseMatch, len(parseMatch))
	}
}

// buildStableChildOrderBenchmarkFixture creates a DOM parent and matching child list.
func buildStableChildOrderBenchmarkFixture(parseCount int) (*Runtime, DOMNode, []DOMNode) {
	parseAdapter := newTestDOMAdapter()
	parseRuntime := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseParent := parseAdapter.CreateElement("section")
	parseExpected := make([]DOMNode, parseCount)
	for parseIndex := range parseExpected {
		parseExpected[parseIndex] = parseAdapter.CreateElement("div")
		parseAdapter.AppendChild(parseParent, parseExpected[parseIndex])
	}
	return parseRuntime, parseParent, parseExpected
}

// applyLegacyStableCommittedChildOrder checks stability by materializing observed nodes.
func applyLegacyStableCommittedChildOrder(parseRuntime *Runtime, parseParent DOMNode, parseExpected []DOMNode) {
	parseObserved := parseRuntime.buildObservedChildNodes(parseParent, len(parseExpected))
	if !isCommittedChildOrderStable(parseExpected, parseObserved) {
		parseRuntime.applyCommittedChildOrder(parseParent, parseExpected)
	}
}

// BenchmarkApplyCommittedChildOrderStable128 compares the stable-list allocation path.
func BenchmarkApplyCommittedChildOrderStable128(parseB *testing.B) {
	parseB.Run("current", func(parseB *testing.B) {
		parseRuntime, parseParent, parseExpected := buildStableChildOrderBenchmarkFixture(128)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseB.Loop() {
			parseRuntime.applyCommittedChildOrder(parseParent, parseExpected)
		}
	})
	parseB.Run("materialized", func(parseB *testing.B) {
		parseRuntime, parseParent, parseExpected := buildStableChildOrderBenchmarkFixture(128)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseB.Loop() {
			applyLegacyStableCommittedChildOrder(parseRuntime, parseParent, parseExpected)
		}
	})
}
