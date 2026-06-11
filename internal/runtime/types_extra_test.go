package runtime

import "testing"

func TestFetchState_ZeroValue(parseT *testing.T) {
	var parseState FetchState

	if parseState.Data != nil {
		parseT.Fatal("expected zero-value fetch state data to be nil")
	}
	if parseState.Error != "" {
		parseT.Fatal("expected zero-value fetch state error to be empty")
	}
	if parseState.Loading {
		parseT.Fatal("expected zero-value fetch state to not be loading")
	}
}

func TestRefValue_IsMutableContainer(parseT *testing.T) {
	parseRef := &RefValue{Current: "initial"}
	parseRef.Current = "updated"

	if parseRef.Current != "updated" {
		parseT.Fatal("expected ref current value to be mutable")
	}
}

func TestHooks_ZeroValueIsUsable(parseT *testing.T) {
	var parseHooks Hooks

	if parseHooks.index != 0 || parseHooks.stateIndex != 0 || parseHooks.cleanupIndex != 0 {
		parseT.Fatal("expected zero-value hooks counters to start at zero")
	}
	if parseHooks.states != nil || parseHooks.deps != nil || parseHooks.cleanups != nil {
		parseT.Fatal("expected zero-value hooks slices to be nil until initialized")
	}
}

func TestFiber_HoldsRuntimeStatePointers(parseT *testing.T) {
	parseHooks := &Hooks{}
	parseParent := &Fiber{typeOf: "parent"}
	parseChild := &Fiber{
		typeOf:      "child",
		parent:      parseParent,
		hooks:       parseHooks,
		dirty:       true,
		needsUpdate: true,
		effectTag:   effectTagUpdate,
	}

	if parseChild.parent != parseParent {
		parseT.Fatal("expected fiber to retain parent pointer")
	}
	if parseChild.hooks != parseHooks {
		parseT.Fatal("expected fiber to retain hooks pointer")
	}
	if !parseChild.dirty || !parseChild.needsUpdate {
		parseT.Fatal("expected fiber flags to remain set")
	}
	if parseChild.effectTag != effectTagUpdate {
		parseT.Fatal("expected fiber effect tag to remain set")
	}
}

func TestElement_CanRepresentTextOptimization(parseT *testing.T) {
	parseElem := &Element{
		Type:        "TEXT_ELEMENT",
		TextContent: "hello",
		Children:    emptyChildren,
	}

	if parseElem.Type != "TEXT_ELEMENT" {
		parseT.Fatal("expected text element type")
	}
	if parseElem.TextContent != "hello" {
		parseT.Fatal("expected text content to be stored directly")
	}
	if len(parseElem.Children) != 0 {
		parseT.Fatal("expected text element to have no children")
	}
}

func TestAttrs_BehavesLikePropsMap(parseT *testing.T) {
	parseAttrs := Attrs{"id": "app", "class": "shell"}

	if parseAttrs["id"] != "app" {
		parseT.Fatal("expected attrs to behave like a props map")
	}
	parseAttrs["role"] = "main"
	if parseAttrs["role"] != "main" {
		parseT.Fatal("expected attrs to allow mutation like a map")
	}
}
