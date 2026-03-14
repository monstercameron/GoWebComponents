package runtime

import (
	"fmt"
	"strings"
	"testing"
)

func drainScheduledTimeouts(t *testing.T, scheduler *testScheduler, maxCallbacks int) int {
	t.Helper()

	processed := 0
	for len(scheduler.timeouts) > 0 {
		callbacks := append([]func(){}, scheduler.timeouts...)
		scheduler.timeouts = scheduler.timeouts[:0]

		for _, callback := range callbacks {
			if processed >= maxCallbacks {
				t.Fatalf("scheduled work did not settle after %d callbacks", maxCallbacks)
			}
			processed++
			callback()
		}
	}

	return processed
}

func findNodeByID(node DOMNode, id string) *testDOMNode {
	testNode, ok := node.(*testDOMNode)
	if !ok || testNode == nil {
		return nil
	}

	if testNode.attributes["id"] == id {
		return testNode
	}

	for _, child := range testNode.children {
		if found := findNodeByID(child, id); found != nil {
			return found
		}
	}

	return nil
}

func collectNodesByClass(node DOMNode, className string, out *[]*testDOMNode) {
	testNode, ok := node.(*testDOMNode)
	if !ok || testNode == nil {
		return
	}

	for _, token := range strings.Fields(testNode.attributes["class"]) {
		if token == className {
			*out = append(*out, testNode)
			break
		}
	}

	for _, child := range testNode.children {
		collectNodesByClass(child, className, out)
	}
}

func nodeTextContent(node DOMNode) string {
	testNode, ok := node.(*testDOMNode)
	if !ok || testNode == nil {
		return ""
	}

	if testNode.nodeType == "text" {
		return testNode.text
	}

	var builder strings.Builder
	for _, child := range testNode.children {
		builder.WriteString(nodeTextContent(child))
	}

	return builder.String()
}

func invokeClick(t *testing.T, node *testDOMNode) {
	t.Helper()

	handler, ok := node.properties["onclick"].(func())
	if !ok {
		t.Fatalf("expected onclick handler with func() signature, got %T", node.properties["onclick"])
	}

	handler()
}

func TestBenchmarkStyleListUpdateSettlesAndUpdatesDOM(t *testing.T) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	InitGlobalRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	rt := GetGlobalRuntime()

	const listSize = 10

	benchmarkComponent := func(props map[string]interface{}) *Element {
		items, setItems := GoUseState(rt, []string{})
		view, setView := GoUseState(rt, "list")
		renderTicks, setRenderTicks := GoUseState(rt, 0)

		// Mirror the benchmark path by scheduling one extra state update
		// whenever list/view state changes.
		GoUseEffect(func() func() {
			setRenderTicks(func(prev int) int { return prev + 1 })
			return nil
		}, items(), view())

		renderList := GoUseFunc(func() {
			setView("list")

			nextItems := make([]string, listSize)
			for i := 0; i < listSize; i++ {
				nextItems[i] = fmt.Sprintf("Item %d", i)
			}
			setItems(nextItems)
		})

		updateList := GoUseFunc(func() {
			currentItems := items()
			nextItems := make([]string, len(currentItems))
			for i, item := range currentItems {
				nextItems[i] = item + " (Updated)"
			}
			setItems(nextItems)
		})

		children := make([]interface{}, 0, len(items()))
		for _, item := range items() {
			children = append(children, Div(map[string]interface{}{"class": "list-item"}, item))
		}

		return Div(map[string]interface{}{"id": "app"},
			Div(map[string]interface{}{"id": "controls"},
				Button(map[string]interface{}{"id": "btn-render", "onclick": renderList}, "Render Items"),
				Button(map[string]interface{}{"id": "btn-update", "onclick": updateList}, "Update Items"),
			),
			P(map[string]interface{}{"id": "item-count"}, fmt.Sprintf("Count: %d", len(items()))),
			P(map[string]interface{}{"id": "render-ticks"}, fmt.Sprintf("Ticks: %d", renderTicks())),
			Div(map[string]interface{}{"id": "container"}, children...),
		)
	}

	container := adapter.CreateElement("div")
	rt.Render(CreateElement(benchmarkComponent, nil), container)
	drainScheduledTimeouts(t, scheduler, 50)

	renderButton := findNodeByID(container, "btn-render")
	if renderButton == nil {
		t.Fatal("expected render button in committed DOM")
	}

	invokeClick(t, renderButton)
	drainScheduledTimeouts(t, scheduler, 50)

	itemCountNode := findNodeByID(container, "item-count")
	if itemCountNode == nil {
		t.Fatal("expected item count node after render")
	}
	if got := nodeTextContent(itemCountNode); got != "Count: 10" {
		t.Fatalf("expected rendered item count to be 10, got %q", got)
	}

	updateButton := findNodeByID(container, "btn-update")
	if updateButton == nil {
		t.Fatal("expected update button after render")
	}

	invokeClick(t, updateButton)
	drainScheduledTimeouts(t, scheduler, 50)

	var listItems []*testDOMNode
	collectNodesByClass(container, "list-item", &listItems)
	if len(listItems) != listSize {
		t.Fatalf("expected %d list items after update, got %d", listSize, len(listItems))
	}

	for i, itemNode := range listItems {
		expected := fmt.Sprintf("Item %d (Updated)", i)
		if got := nodeTextContent(itemNode); got != expected {
			t.Fatalf("expected updated text %q at index %d, got %q", expected, i, got)
		}
	}

	if len(scheduler.timeouts) != 0 {
		t.Fatalf("expected all scheduled work to settle, found %d pending callbacks", len(scheduler.timeouts))
	}
}
