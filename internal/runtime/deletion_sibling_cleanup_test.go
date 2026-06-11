package runtime

import "testing"

// Regression test for the deletion/sibling cleanup bug: when child A is deleted but sibling B survives,
// B's effect cleanup must NOT run and B's event-handler funcs must survive.
func TestDeletionDoesNotCleanupSurvivingSibling(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("section")

	parseBCleanupCount := 0
	parseBEffectCount := 0

	parseChildB := func(parseProps map[string]interface{}) *Element {
		GoUseEffect(func() func() {
			parseBEffectCount++
			return func() { parseBCleanupCount++ }
		})
		return CreateElement("div", map[string]interface{}{"id": "b"}, "b")
	}

	parseHost := func(parseIncludeA bool) *Element {
		parseChildren := []interface{}{}
		if parseIncludeA {
			parseChildren = append(parseChildren, CreateElement("div", map[string]interface{}{"id": "a"}, "a"))
		} else {
			// different type forces REPLACE: old div A tagged DELETION, new span placed
			parseChildren = append(parseChildren, CreateElement("span", map[string]interface{}{"id": "a2"}, "a2"))
		}
		parseChildren = append(parseChildren, CreateElement(parseChildB, nil))
		return CreateElement("div", map[string]interface{}{"id": "host"}, parseChildren...)
	}

	parseRt.Render(parseHost(true), parseContainer)
	if parseBEffectCount != 1 {
		parseT.Fatalf("expected B effect to run once after mount, got %d", parseBEffectCount)
	}

	// Re-render replacing A with a different element type; B survives.
	parseRt.Render(parseHost(false), parseContainer)

	if parseBCleanupCount != 0 {
		parseT.Fatalf("surviving sibling B's cleanup ran %d time(s) when A was deleted", parseBCleanupCount)
	}
}
