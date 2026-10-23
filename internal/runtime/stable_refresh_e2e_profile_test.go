package runtime

import (
	"fmt"
	"testing"
)

// buildStableRefreshProfileFixture mounts a static nested tree for refresh checks.
func buildStableRefreshProfileFixture(parseT testing.TB) (*Runtime, *testScheduler, DOMNode) {
	parseT.Helper()
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRuntime := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseRows := make([]any, 0, 128)
	for parseRow := range 128 {
		parseLeaf := CreateElement("span", map[string]any{"id": fmt.Sprintf("leaf-%d", parseRow)}, fmt.Sprintf("row-%d", parseRow))
		parseNested := parseLeaf
		for parseDepth := 0; parseDepth < 8; parseDepth++ {
			parseNested = CreateElement("div", nil, parseNested)
		}
		parseRows = append(parseRows, CreateElement("section", map[string]any{"key": parseRow}, parseNested))
	}
	parseRuntime.Render(CreateElement("main", nil, parseRows...), parseContainer)
	for len(parseScheduler.timeouts) > 0 {
		parseCallbacks := append([]func(){}, parseScheduler.timeouts...)
		parseScheduler.timeouts = parseScheduler.timeouts[:0]
		for _, parseCallback := range parseCallbacks {
			parseCallback()
		}
	}
	return parseRuntime, parseScheduler, parseContainer
}

func BenchmarkStableRefresh128Nested(parseB *testing.B) {
	parseRuntime, parseScheduler, _ := buildStableRefreshProfileFixture(parseB)
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseB.Loop() {
		parseRuntime.ScheduleUpdate()
		for len(parseScheduler.timeouts) > 0 {
			parseCallbacks := append([]func(){}, parseScheduler.timeouts...)
			parseScheduler.timeouts = parseScheduler.timeouts[:0]
			for _, parseCallback := range parseCallbacks {
				parseCallback()
			}
		}
	}
}

func TestStableRefresh128NestedPreservesDOM(parseT *testing.T) {
	parseRuntime, parseScheduler, parseContainer := buildStableRefreshProfileFixture(parseT)
	parseBefore := textFromNode(parseContainer)
	parseBeforeNodes := buildStableRefreshDOMNodes(parseContainer)
	parseRuntime.ScheduleUpdate()
	for len(parseScheduler.timeouts) > 0 {
		parseCallbacks := append([]func(){}, parseScheduler.timeouts...)
		parseScheduler.timeouts = parseScheduler.timeouts[:0]
		for _, parseCallback := range parseCallbacks {
			parseCallback()
		}
	}
	if parseAfter := textFromNode(parseContainer); parseAfter != parseBefore {
		parseT.Fatalf("no-op refresh changed DOM text: got %q, want %q", parseAfter, parseBefore)
	}
	parseAfterNodes := buildStableRefreshDOMNodes(parseContainer)
	if len(parseAfterNodes) != len(parseBeforeNodes) {
		parseT.Fatalf("no-op refresh changed DOM node count: got %d, want %d", len(parseAfterNodes), len(parseBeforeNodes))
	}
	for parseIndex := range parseBeforeNodes {
		if parseAfterNodes[parseIndex] != parseBeforeNodes[parseIndex] {
			parseT.Fatalf("no-op refresh replaced DOM node %d", parseIndex)
		}
	}
}

// buildStableRefreshDOMNodes captures DOM identity in tree order.
func buildStableRefreshDOMNodes(parseNode DOMNode) []*testDOMNode {
	parseTyped, parseOk := parseNode.(*testDOMNode)
	if !parseOk || parseTyped == nil {
		return nil
	}
	parseNodes := []*testDOMNode{parseTyped}
	for _, parseChild := range parseTyped.children {
		parseNodes = append(parseNodes, buildStableRefreshDOMNodes(parseChild)...)
	}
	return parseNodes
}
