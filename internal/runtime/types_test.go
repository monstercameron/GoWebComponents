package runtime

import "testing"

func TestFiberCreation(t *testing.T) {
	fiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dirty:  true,
	}
	
	if fiber.typeOf != "div" {
		t.Errorf("Expected typeOf to be 'div', got %v", fiber.typeOf)
	}
	
	if fiber.props == nil {
		t.Error("Expected props map to be initialized")
	}
	
	if !fiber.dirty {
		t.Error("Expected fiber to be marked dirty initially")
	}
}

func TestFiberTreeStructure(t *testing.T) {
	parent := &Fiber{typeOf: "parent"}
	child := &Fiber{typeOf: "child", parent: parent}
	sibling := &Fiber{typeOf: "sibling", parent: parent}
	
	parent.child = child
	child.sibling = sibling
	
	if parent.child != child {
		t.Error("Expected parent.child to point to child fiber")
	}
	
	if child.parent != parent {
		t.Error("Expected child.parent to point to parent fiber")
	}
	
	if child.sibling != sibling {
		t.Error("Expected child.sibling to point to sibling fiber")
	}
	
	if sibling.parent != parent {
		t.Error("Expected sibling.parent to point to parent fiber")
	}
}

func TestHooksInitialization(t *testing.T) {
	hooks := &Hooks{
		state:     make([]interface{}, 0),
		deps:      make([][]interface{}, 0),
		memos:     make([]memoizedValue, 0),
		callOrder: make([]HookCall, 0),
		prevOrder: make([]HookCall, 0),
		index:     0,
	}
	
	if hooks.index != 0 {
		t.Errorf("Expected hooks.index to be 0, got %d", hooks.index)
	}
	
	if hooks.state == nil {
		t.Error("Expected state slice to be initialized")
	}
	
	if hooks.deps == nil {
		t.Error("Expected deps slice to be initialized")
	}
}

func TestElementCreation(t *testing.T) {
	props := map[string]interface{}{
		"id":    "test-element",
		"class": "container",
	}
	
	children := []interface{}{
		"Hello World",
	}
	
	element := &Element{
		Type:     "div",
		Props:    props,
		Children: children,
	}
	
	if element.Type != "div" {
		t.Errorf("Expected Type to be 'div', got %v", element.Type)
	}
	
	if element.Props["id"] != "test-element" {
		t.Errorf("Expected id prop to be 'test-element', got %v", element.Props["id"])
	}
	
	if len(element.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(element.Children))
	}
}

func TestHookTypeConstants(t *testing.T) {
	if HookTypeState != 0 {
		t.Errorf("Expected HookTypeState to be 0, got %d", HookTypeState)
	}
	
	if HookTypeEffect != 1 {
		t.Errorf("Expected HookTypeEffect to be 1, got %d", HookTypeEffect)
	}
	
	if HookTypeMemo != 2 {
		t.Errorf("Expected HookTypeMemo to be 2, got %d", HookTypeMemo)
	}
}
