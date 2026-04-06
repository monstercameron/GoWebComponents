package runtime

import "testing"

// TestCommitRootReordersMixedKeyedAndUnkeyedChildren verifies committed DOM order follows the new fiber order even when keyed reconciliation reuses one unkeyed sibling.
func TestCommitRootReordersMixedKeyedAndUnkeyedChildren(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("section")

	parseRenderHost := func(parseSpacerHeight string, parseIncludeRowB bool) *Element {
		parseChildren := []interface{}{
			CreateElement("div", map[string]interface{}{"id": "row-a", "key": "a"}, "a"),
			CreateElement("div", map[string]interface{}{
				"id":    "pad",
				"style": map[string]string{"height": parseSpacerHeight},
			}),
		}
		if parseIncludeRowB {
			parseChildren = []interface{}{
				CreateElement("div", map[string]interface{}{
					"id":    "pad",
					"style": map[string]string{"height": parseSpacerHeight},
				}),
				CreateElement("div", map[string]interface{}{"id": "row-a", "key": "a"}, "a"),
				CreateElement("div", map[string]interface{}{"id": "row-b", "key": "b"}, "b"),
			}
		}
		return CreateElement("div", map[string]interface{}{"id": "host"}, parseChildren...)
	}

	parseAssertChildOrder := func(parseExpectedIDs []string, parseExpectedSpacerHeight string) {
		parseT.Helper()
		parseRootChildren := parseAdapter.GetChildren(parseContainer)
		if len(parseRootChildren) != 1 {
			parseT.Fatalf("expected one host root child, got %d", len(parseRootChildren))
		}
		parseHost, parseOk := parseRootChildren[0].(*testDOMNode)
		if !parseOk {
			parseT.Fatalf("expected host node type *testDOMNode, got %T", parseRootChildren[0])
		}
		if len(parseHost.children) != len(parseExpectedIDs) {
			parseT.Fatalf("expected %d host children, got %d", len(parseExpectedIDs), len(parseHost.children))
		}
		for parseIndex, parseExpectedID := range parseExpectedIDs {
			parseChild, parseOk2 := parseHost.children[parseIndex].(*testDOMNode)
			if !parseOk2 {
				parseT.Fatalf("expected child %d to be *testDOMNode, got %T", parseIndex, parseHost.children[parseIndex])
			}
			if parseChild.attributes["id"] != parseExpectedID {
				parseT.Fatalf("expected child order %v, got id %q at index %d", parseExpectedIDs, parseChild.attributes["id"], parseIndex)
			}
		}
		parseSpacer, parseOk3 := parseHost.children[0].(*testDOMNode)
		if !parseOk3 {
			parseT.Fatalf("expected spacer node type *testDOMNode, got %T", parseHost.children[0])
		}
		if parseSpacer.attributes["id"] == "pad" && parseSpacer.styles["height"] != parseExpectedSpacerHeight {
			parseT.Fatalf("expected spacer height %q, got %q", parseExpectedSpacerHeight, parseSpacer.styles["height"])
		}
	}

	parseRt.Render(parseRenderHost("20px", false), parseContainer)
	parseAssertChildOrder([]string{"row-a", "pad"}, "20px")

	parseRt.Render(parseRenderHost("60px", true), parseContainer)
	parseAssertChildOrder([]string{"pad", "row-a", "row-b"}, "60px")
}
