package runtime

import "testing"

// TestDOMRefPublishesOnHydratedNode pins that a ref on a server-rendered
// element receives the reused node, not only nodes created by placement.
func TestDOMRefPublishesOnHydratedNode(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, FrameBudgetMs: -1})
	parseContainer := parseAdapter.CreateElement("div")
	parseServerNode := parseAdapter.CreateElement("input")
	parseAdapter.SetAttribute(parseServerNode, "id", "target")
	parseAdapter.AppendChild(parseContainer, parseServerNode)

	parseSink := &captureSink{}
	parseRt.Hydrate(CreateElement("input", map[string]any{"id": "target", DOMRefKey: parseSink}), parseContainer)
	runHydrationWork(parseT, parseScheduler)

	if IsDOMNodeNull(parseSink.node) || !parseSink.node.Equals(parseServerNode) {
		parseT.Fatalf("hydrated element never reached its ref: %#v", parseSink.node)
	}
}

// TestDOMRefMovesToNewSinkOnSurvivingElement pins that swapping the ref sink on
// an element that stays mounted detaches the old sink and attaches the new one.
func TestDOMRefMovesToNewSinkOnSurvivingElement(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("div")
	parseFirst, parseSecond := &captureSink{}, &captureSink{}

	parseRt.Render(renderHostWith(parseFirst, true, ""), parseContainer)
	parseRt.Render(renderHostWith(parseSecond, true, ""), parseContainer)

	if IsDOMNodeNull(parseSecond.node) {
		parseT.Fatal("new ref sink never received the surviving node")
	}
	if !IsDOMNodeNull(parseFirst.node) {
		parseT.Fatal("old ref sink still points at the node after the ref moved")
	}
}
