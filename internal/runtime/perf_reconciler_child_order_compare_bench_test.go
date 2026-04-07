package runtime

import "testing"

// buildChildOrderBenchmarkFixture creates one reversed keyed child order so the child-order repair benchmark measures move-heavy work.
func buildChildOrderBenchmarkFixture(getCount int) (*Runtime, DOMNode, []DOMNode) {
	getAdapter := newTestDOMAdapter()
	getRuntime := NewRuntime(Config{DOMAdapter: getAdapter})
	getParent := getAdapter.CreateElement("section")
	getExpected := make([]DOMNode, 0, getCount)
	getObserved := make([]DOMNode, 0, getCount)

	for getIndex := 0; getIndex < getCount; getIndex++ {
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
		getObserved := getRuntime.buildObservedChildNodes(getParent)
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
			getObserved = getRuntime.buildObservedChildNodes(getParent)
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
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseB.StopTimer()
			getRuntime, getParent, getExpected := buildChildOrderBenchmarkFixture(128)
			parseB.StartTimer()

			getRuntime.applyCommittedChildOrder(getParent, getExpected)
		}
	})

	parseB.Run("legacy", func(parseB *testing.B) {
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseB.StopTimer()
			getRuntime, getParent, getExpected := buildChildOrderBenchmarkFixture(128)
			parseB.StartTimer()

			applyLegacyCommittedChildOrder(getRuntime, getParent, getExpected)
		}
	})
}
