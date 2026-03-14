package runtime

import "testing"

func TestFetchState_ZeroValue(t *testing.T) {
	var state FetchState

	if state.Data != nil {
		t.Fatal("expected zero-value fetch state data to be nil")
	}
	if state.Error != "" {
		t.Fatal("expected zero-value fetch state error to be empty")
	}
	if state.Loading {
		t.Fatal("expected zero-value fetch state to not be loading")
	}
}

func TestRefValue_IsMutableContainer(t *testing.T) {
	ref := &RefValue{Current: "initial"}
	ref.Current = "updated"

	if ref.Current != "updated" {
		t.Fatal("expected ref current value to be mutable")
	}
}

func TestHooks_ZeroValueIsUsable(t *testing.T) {
	var hooks Hooks

	if hooks.index != 0 || hooks.stateIndex != 0 || hooks.cleanupIndex != 0 {
		t.Fatal("expected zero-value hooks counters to start at zero")
	}
	if hooks.states != nil || hooks.deps != nil || hooks.cleanups != nil {
		t.Fatal("expected zero-value hooks slices to be nil until initialized")
	}
}

func TestFiber_HoldsRuntimeStatePointers(t *testing.T) {
	hooks := &Hooks{}
	parent := &Fiber{typeOf: "parent"}
	child := &Fiber{
		typeOf:      "child",
		parent:      parent,
		hooks:       hooks,
		dirty:       true,
		needsUpdate: true,
		effectTag:   "UPDATE",
	}

	if child.parent != parent {
		t.Fatal("expected fiber to retain parent pointer")
	}
	if child.hooks != hooks {
		t.Fatal("expected fiber to retain hooks pointer")
	}
	if !child.dirty || !child.needsUpdate {
		t.Fatal("expected fiber flags to remain set")
	}
	if child.effectTag != "UPDATE" {
		t.Fatal("expected fiber effect tag to remain set")
	}
}

func TestElement_CanRepresentTextOptimization(t *testing.T) {
	elem := &Element{
		Type:        "TEXT_ELEMENT",
		TextContent: "hello",
		Children:    emptyChildren,
	}

	if elem.Type != "TEXT_ELEMENT" {
		t.Fatal("expected text element type")
	}
	if elem.TextContent != "hello" {
		t.Fatal("expected text content to be stored directly")
	}
	if len(elem.Children) != 0 {
		t.Fatal("expected text element to have no children")
	}
}

func TestAttrs_BehavesLikePropsMap(t *testing.T) {
	attrs := Attrs{"id": "app", "class": "shell"}

	if attrs["id"] != "app" {
		t.Fatal("expected attrs to behave like a props map")
	}
	attrs["role"] = "main"
	if attrs["role"] != "main" {
		t.Fatal("expected attrs to allow mutation like a map")
	}
}
