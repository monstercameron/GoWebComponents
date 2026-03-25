package runtime

import "testing"

func TestFiberCreation(parseT *testing.T) {
	parseFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dirty:  true,
	}

	if parseFiber.typeOf != "div" {
		parseT.Errorf("Expected typeOf to be 'div', got %v", parseFiber.typeOf)
	}

	if parseFiber.props == nil {
		parseT.Error("Expected props map to be initialized")
	}

	if !parseFiber.dirty {
		parseT.Error("Expected fiber to be marked dirty initially")
	}
}

func TestFiberTreeStructure(parseT *testing.T) {
	parseParent := &Fiber{typeOf: "parent"}
	parseChild := &Fiber{typeOf: "child", parent: parseParent}
	parseSibling := &Fiber{typeOf: "sibling", parent: parseParent}

	parseParent.child = parseChild
	parseChild.sibling = parseSibling

	if parseParent.child != parseChild {
		parseT.Error("Expected parent.child to point to child fiber")
	}

	if parseChild.parent != parseParent {
		parseT.Error("Expected child.parent to point to parent fiber")
	}

	if parseChild.sibling != parseSibling {
		parseT.Error("Expected child.sibling to point to sibling fiber")
	}

	if parseSibling.parent != parseParent {
		parseT.Error("Expected sibling.parent to point to parent fiber")
	}
}

func TestHooksInitialization(parseT *testing.T) {
	parseHooks := &Hooks{
		states: make([]interface{}, 0),
		deps:   make([][]interface{}, 0),
		index:  0,
	}

	if parseHooks.index != 0 {
		parseT.Errorf("Expected hooks.index to be 0, got %d", parseHooks.index)
	}

	if parseHooks.states == nil {
		parseT.Error("Expected state slice to be initialized")
	}

	if parseHooks.deps == nil {
		parseT.Error("Expected deps slice to be initialized")
	}
}

func TestElementCreation(parseT *testing.T) {
	parseProps := map[string]interface{}{
		"id":    "test-element",
		"class": "container",
	}

	parseChildren := []interface{}{
		"Hello World",
	}

	parseElement := &Element{
		Type:     "div",
		Props:    parseProps,
		Children: parseChildren,
	}

	if parseElement.Type != "div" {
		parseT.Errorf("Expected Type to be 'div', got %v", parseElement.Type)
	}

	if parseElement.Props["id"] != "test-element" {
		parseT.Errorf("Expected id prop to be 'test-element', got %v", parseElement.Props["id"])
	}

	if len(parseElement.Children) != 1 {
		parseT.Errorf("Expected 1 child, got %d", len(parseElement.Children))
	}
}
