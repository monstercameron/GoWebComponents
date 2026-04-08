package runtime

import (
	"fmt"
	"strings"
	"testing"
)

func drainScheduledTimeouts(parseT *testing.T, parseScheduler *testScheduler, parseMaxCallbacks int) int {
	parseT.Helper()

	parseProcessed := 0
	for len(parseScheduler.timeouts) > 0 {
		parseCallbacks := append([]func(){}, parseScheduler.timeouts...)
		parseScheduler.timeouts = parseScheduler.timeouts[:0]

		for _, parseCallback := range parseCallbacks {
			if parseProcessed >= parseMaxCallbacks {
				parseT.Fatalf("scheduled work did not settle after %d callbacks", parseMaxCallbacks)
			}
			parseProcessed++
			parseCallback()
		}
	}

	return parseProcessed
}

func findNodeByID(parseNode DOMNode, parseId string) *testDOMNode {
	parseTestNode, parseOk := parseNode.(*testDOMNode)
	if !parseOk || parseTestNode == nil {
		return nil
	}

	if parseTestNode.attributes["id"] == parseId {
		return parseTestNode
	}

	for _, parseChild := range parseTestNode.children {
		if parseFound := findNodeByID(parseChild, parseId); parseFound != nil {
			return parseFound
		}
	}

	return nil
}

func collectNodesByClass(parseNode DOMNode, parseClassName string, parseOut *[]*testDOMNode) {
	parseTestNode, parseOk := parseNode.(*testDOMNode)
	if !parseOk || parseTestNode == nil {
		return
	}

	for _, parseToken := range strings.Fields(parseTestNode.attributes["class"]) {
		if parseToken == parseClassName {
			*parseOut = append(*parseOut, parseTestNode)
			break
		}
	}

	for _, parseChild := range parseTestNode.children {
		collectNodesByClass(parseChild, parseClassName, parseOut)
	}
}

func nodeTextContent(parseNode DOMNode) string {
	parseTestNode, parseOk := parseNode.(*testDOMNode)
	if !parseOk || parseTestNode == nil {
		return ""
	}

	if parseTestNode.nodeType == "text" {
		return parseTestNode.text
	}

	var parseBuilder strings.Builder
	for _, parseChild := range parseTestNode.children {
		parseBuilder.WriteString(nodeTextContent(parseChild))
	}

	return parseBuilder.String()
}

func invokeClick(parseT *testing.T, parseNode *testDOMNode) {
	parseT.Helper()

	parseHandler, parseOk := parseNode.properties["onclick"].(func())
	if !parseOk {
		parseT.Fatalf("expected onclick handler with func() signature, got %T", parseNode.properties["onclick"])
	}

	parseHandler()
}

func TestBenchmarkStyleListUpdateSettlesAndUpdatesDOM(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	InitGlobalRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseRt := GetGlobalRuntime()

	const listSize = 10

	parseBenchmarkComponent := func(parseProps map[string]interface{}) *Element {
		parseItems, setItems := GoUseState(parseRt, []string{})
		parseView, setView := GoUseState(parseRt, "list")
		renderTicks, setRenderTicks := GoUseState(parseRt, 0)

		// Mirror the benchmark path by scheduling one extra state update
		// whenever list/view state changes.
		GoUseEffect(func() func() {
			setRenderTicks(func(parsePrev int) int { return parsePrev + 1 })
			return nil
		}, parseItems(), parseView())

		renderList := GoUseFunc(func() {
			setView("list")

			parseNextItems := make([]string, listSize)
			for parseI := 0; parseI < listSize; parseI++ {
				parseNextItems[parseI] = fmt.Sprintf("Item %d", parseI)
			}
			setItems(parseNextItems)
		})

		parseUpdateList := GoUseFunc(func() {
			parseCurrentItems := parseItems()
			parseNextItems2 := make([]string, len(parseCurrentItems))
			for parseI2, parseItem := range parseCurrentItems {
				parseNextItems2[parseI2] = parseItem + " (Updated)"
			}
			setItems(parseNextItems2)
		})

		parseChildren := make([]interface{}, 0, len(parseItems()))
		for _, parseItem2 := range parseItems() {
			parseChildren = append(parseChildren, Div(map[string]interface{}{"class": "list-item"}, parseItem2))
		}

		return Div(map[string]interface{}{"id": "app"},
			Div(map[string]interface{}{"id": "controls"},
				Button(map[string]interface{}{"id": "btn-render", "onclick": renderList}, "Render Items"),
				Button(map[string]interface{}{"id": "btn-update", "onclick": parseUpdateList}, "Update Items"),
			),
			P(map[string]interface{}{"id": "item-count"}, fmt.Sprintf("Count: %d", len(parseItems()))),
			P(map[string]interface{}{"id": "render-ticks"}, fmt.Sprintf("Ticks: %d", renderTicks())),
			Div(map[string]interface{}{"id": "container"}, parseChildren...),
		)
	}

	parseContainer := parseAdapter.CreateElement("div")
	parseRt.Render(CreateElement(parseBenchmarkComponent, nil), parseContainer)
	drainScheduledTimeouts(parseT, parseScheduler, 50)

	renderButton := findNodeByID(parseContainer, "btn-render")
	if renderButton == nil {
		parseT.Fatal("expected render button in committed DOM")
	}

	invokeClick(parseT, renderButton)
	drainScheduledTimeouts(parseT, parseScheduler, 50)

	parseItemCountNode := findNodeByID(parseContainer, "item-count")
	if parseItemCountNode == nil {
		parseT.Fatal("expected item count node after render")
	}
	if parseGot := nodeTextContent(parseItemCountNode); parseGot != "Count: 10" {
		parseT.Fatalf("expected rendered item count to be 10, got %q", parseGot)
	}

	parseUpdateButton := findNodeByID(parseContainer, "btn-update")
	if parseUpdateButton == nil {
		parseT.Fatal("expected update button after render")
	}

	invokeClick(parseT, parseUpdateButton)
	drainScheduledTimeouts(parseT, parseScheduler, 50)

	var parseListItems []*testDOMNode
	collectNodesByClass(parseContainer, "list-item", &parseListItems)
	if len(parseListItems) != listSize {
		parseT.Fatalf("expected %d list items after update, got %d", listSize, len(parseListItems))
	}

	for parseI3, parseItemNode := range parseListItems {
		parseExpected := fmt.Sprintf("Item %d (Updated)", parseI3)
		if parseGot2 := nodeTextContent(parseItemNode); parseGot2 != parseExpected {
			parseT.Fatalf("expected updated text %q at index %d, got %q", parseExpected, parseI3, parseGot2)
		}
	}

	if len(parseScheduler.timeouts) != 0 {
		parseT.Fatalf("expected all scheduled work to settle, found %d pending callbacks", len(parseScheduler.timeouts))
	}
}

func TestRenderRunsQueuedEffectsAfterCommit(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	isParseExecuted := false
	parseComponent := func() *Element {
		GoUseEffect(func() func() {
			isParseExecuted = true
			return nil
		})
		return CreateElement("div", nil)
	}

	parseContainer := parseAdapter.CreateElement("div")
	parseRt.Render(CreateElement(parseComponent, nil), parseContainer)
	drainScheduledTimeouts(parseT, parseScheduler, 10)

	if !isParseExecuted {
		parseT.Fatal("expected initial render commit to run queued effects")
	}
}
