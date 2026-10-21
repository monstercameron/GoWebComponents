package runtime

import "testing"

// TestFragmentSiblingKeepsNilSlotForStatefulNeighbour pins that a conditional
// child (nil when off) keeps its slot even when a sibling Fragment forces the
// children to be flattened, so toggling it does not remount the neighbours.
func TestFragmentSiblingKeepsNilSlotForStatefulNeighbour(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, FrameBudgetMs: -1})

	parseMounts := 0
	parseCounter := func() *Element {
		GoUseEffect(func() func() {
			parseMounts++
			return nil
		}, 0)
		return CreateElement("i", nil, "counter")
	}
	parseBuild := func(isParseShown bool) *Element {
		var parseMaybe any
		if isParseShown {
			parseMaybe = CreateElement("span", nil, "extra")
		}
		return CreateElement("div", nil,
			parseMaybe,
			CreateElement(parseCounter, nil),
			CreateElement("FRAGMENT", nil, CreateElement("p", nil, "a"), CreateElement("p", nil, "b")),
		)
	}

	parseContainer := parseAdapter.CreateElement("div")
	parseRt.Render(parseBuild(true), parseContainer)
	pumpContextReviewWork(parseScheduler)
	parseRt.Render(parseBuild(false), parseContainer)
	pumpContextReviewWork(parseScheduler)
	parseRt.Render(parseBuild(true), parseContainer)
	pumpContextReviewWork(parseScheduler)

	if parseMounts != 1 {
		parseT.Fatalf("toggling a nil sibling remounted the neighbour: mounts=%d, want 1", parseMounts)
	}
}
