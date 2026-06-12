package runtime

import (
	"errors"
	"testing"
)

// buildAgentWriteFixtureTree builds ROOT -> ul -> [Item(key:a) -> button(onclick)]
// and returns the runtime plus the button fiber with an onclick handler already
// installed in its props. This mirrors the buildAgentRefFixtureTree pattern but
// adds a handler so EmitAgentEvent has something to invoke.
func buildAgentWriteFixtureTree(parseOnClick func()) (*Runtime, *Fiber) {
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseList := &Fiber{typeOf: "ul", parent: parseRoot}
	parseRoot.child = parseList

	parseItemType := &ComponentType{Name: "Item", QualifiedName: "example.com/app.Item"}
	parseItemA := &Fiber{typeOf: parseItemType, parent: parseList, props: map[string]any{"key": "a"}}
	parseList.child = parseItemA

	parseButton := &Fiber{
		typeOf: "button",
		parent: parseItemA,
		props:  map[string]any{"onclick": parseOnClick},
	}
	parseItemA.child = parseButton

	parseRt := &Runtime{currentRoot: parseRoot}
	return parseRt, parseButton
}

// TestEmitAgentEventInvokesHandler pins that EmitAgentEvent calls the handler
// stored in fiber props and that the call is observable.
func TestEmitAgentEventInvokesHandler(t *testing.T) {
	parseCalled := false
	parseRt, parseButton := buildAgentWriteFixtureTree(func() {
		parseCalled = true
	})

	parseRef := AgentRefForFiber(parseButton)
	if parseRef == "" {
		t.Fatalf("expected non-empty ref for button fiber")
	}

	if parseErr := EmitAgentEvent(parseRt, parseRef, "onclick", nil); parseErr != nil {
		t.Fatalf("EmitAgentEvent returned unexpected error: %v", parseErr)
	}
	if !parseCalled {
		t.Fatal("handler was not invoked by EmitAgentEvent")
	}
}

// TestEmitAgentEventNormalisesEventName pins that "click" and "onclick" both
// resolve to the same prop key.
func TestEmitAgentEventNormalisesEventName(t *testing.T) {
	parseClickCount := 0
	parseRt, parseButton := buildAgentWriteFixtureTree(func() {
		parseClickCount++
	})
	parseRef := AgentRefForFiber(parseButton)

	for _, parseEventName := range []string{"click", "onclick"} {
		if parseErr := EmitAgentEvent(parseRt, parseRef, parseEventName, nil); parseErr != nil {
			t.Fatalf("EmitAgentEvent(%q) returned unexpected error: %v", parseEventName, parseErr)
		}
	}
	if parseClickCount != 2 {
		t.Fatalf("expected 2 invocations, got %d", parseClickCount)
	}
}

// TestEmitAgentEventStaleRef pins that a stale ref returns ErrAgentRefStale.
func TestEmitAgentEventStaleRef(t *testing.T) {
	parseRt, parseButton := buildAgentWriteFixtureTree(func() {})
	parseRef := AgentRefForFiber(parseButton)

	// Detach the subtree so the ref becomes stale.
	parseRt.currentRoot.child.child = nil

	parseErr := EmitAgentEvent(parseRt, parseRef, "onclick", nil)
	if !errors.Is(parseErr, ErrAgentRefStale) {
		t.Fatalf("expected ErrAgentRefStale, got %v", parseErr)
	}
}

// TestEmitAgentEventNoHandler pins that a fiber with no matching prop returns
// ErrAgentNoHandler.
func TestEmitAgentEventNoHandler(t *testing.T) {
	parseRt, parseButton := buildAgentWriteFixtureTree(func() {})
	parseRef := AgentRefForFiber(parseButton)

	parseErr := EmitAgentEvent(parseRt, parseRef, "onkeydown", nil)
	if !errors.Is(parseErr, ErrAgentNoHandler) {
		t.Fatalf("expected ErrAgentNoHandler, got %v", parseErr)
	}
}

// TestEmitAgentEventPanicContained pins that a handler that panics does not
// propagate the panic out of EmitAgentEvent; it is converted to an error.
func TestEmitAgentEventPanicContained(t *testing.T) {
	parseRt, _ := buildAgentWriteFixtureTree(func() {})

	// Build a tree with a panicking handler directly, skipping the helper.
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseButton := &Fiber{
		typeOf: "button",
		parent: parseRoot,
		props: map[string]any{
			"onclick": func() { panic("agent test panic") },
		},
	}
	parseRoot.child = parseButton
	parseRt.currentRoot = parseRoot

	parseRef := AgentRefForFiber(parseButton)
	parseErr := EmitAgentEvent(parseRt, parseRef, "onclick", nil)
	if parseErr == nil {
		t.Fatal("expected non-nil error from panicking handler, got nil")
	}
}

// TestEmitAgentEventFuncStringHandler pins that a func(string) handler receives
// the empty string (zero value for the string type in the agent context).
func TestEmitAgentEventFuncStringHandler(t *testing.T) {
	parseGotValue := "UNSET"
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseButton := &Fiber{
		typeOf: "button",
		parent: parseRoot,
		props: map[string]any{
			"oninput": func(parseV string) { parseGotValue = parseV },
		},
	}
	parseRoot.child = parseButton
	parseRt := &Runtime{currentRoot: parseRoot}

	parseRef := AgentRefForFiber(parseButton)
	if parseErr := EmitAgentEvent(parseRt, parseRef, "oninput", nil); parseErr != nil {
		t.Fatalf("EmitAgentEvent with func(string) handler returned error: %v", parseErr)
	}
	if parseGotValue != "" {
		t.Fatalf("expected empty string, got %q", parseGotValue)
	}
}

// TestEmitAgentEventFuncErrorHandlerPropagates pins that a func() error handler
// that returns an error is wrapped and returned.
func TestEmitAgentEventFuncErrorHandlerPropagates(t *testing.T) {
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseButton := &Fiber{
		typeOf: "button",
		parent: parseRoot,
		props: map[string]any{
			"onclick": func() error { return errors.New("handler error") },
		},
	}
	parseRoot.child = parseButton
	parseRt := &Runtime{currentRoot: parseRoot}

	parseRef := AgentRefForFiber(parseButton)
	parseErr := EmitAgentEvent(parseRt, parseRef, "onclick", nil)
	if parseErr == nil {
		t.Fatal("expected error from func() error handler, got nil")
	}
}

// TestEmitAgentEventNilRuntime pins that EmitAgentEvent is safe when passed a
// nil runtime.
func TestEmitAgentEventNilRuntime(t *testing.T) {
	parseErr := EmitAgentEvent(nil, "someref", "onclick", nil)
	if parseErr == nil {
		t.Fatal("expected error with nil runtime, got nil")
	}
}

// TestSetAgentStateUpdatesOnlyAddressedStateSlot verifies bridge.set-state's
// runtime primitive maps a hook slot to the matching state storage pair and
// leaves other state hooks untouched.
func TestSetAgentStateUpdatesOnlyAddressedStateSlot(t *testing.T) {
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseComponent := &Fiber{
		typeOf: NewComponentType("example/Probe", "Probe", "example/Probe", nil, nil),
		parent: parseRoot,
		hooks: &Hooks{
			signature: []string{"state", "memo", "state"},
			states:    []any{1, 1, "old", "old"},
		},
	}
	parseComponent.hooks.owner = parseComponent
	parseRoot.child = parseComponent
	parseRt := &Runtime{
		currentRoot:     parseRoot,
		atomRegistry:    NewAtomRegistry(),
		updateScheduled: true,
	}

	parseRef := AgentRefForFiber(parseComponent)
	if parseErr := SetAgentState(parseRt, parseRef, 2, "new"); parseErr != nil {
		t.Fatalf("SetAgentState returned error: %v", parseErr)
	}
	if parseComponent.hooks.states[0] != 1 || parseComponent.hooks.states[1] != 1 {
		t.Fatalf("first state hook changed unexpectedly: %#v", parseComponent.hooks.states[:2])
	}
	if parseComponent.hooks.states[2] != "new" || parseComponent.hooks.states[3] != "new" {
		t.Fatalf("second state hook was not updated in current+pending slots: %#v", parseComponent.hooks.states[2:4])
	}
	if !parseComponent.dirty || !parseComponent.needsUpdate {
		t.Fatalf("component was not marked dirty for agent state update")
	}
}

func TestSetAgentStateRejectsOutOfRangeAndNonStateSlots(t *testing.T) {
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseComponent := &Fiber{
		typeOf: NewComponentType("example/Probe", "Probe", "example/Probe", nil, nil),
		parent: parseRoot,
		hooks: &Hooks{
			signature: []string{"state", "memo"},
			states:    []any{1, 1},
		},
	}
	parseComponent.hooks.owner = parseComponent
	parseRoot.child = parseComponent
	parseRt := &Runtime{currentRoot: parseRoot, atomRegistry: NewAtomRegistry(), updateScheduled: true}
	parseRef := AgentRefForFiber(parseComponent)

	if parseErr := SetAgentState(parseRt, parseRef, 4, 2); !errors.Is(parseErr, ErrAgentStateSlotInvalid) {
		t.Fatalf("expected ErrAgentStateSlotInvalid for out-of-range slot, got %v", parseErr)
	}
	if parseErr := SetAgentState(parseRt, parseRef, 1, 2); !errors.Is(parseErr, ErrAgentStateSlotInvalid) {
		t.Fatalf("expected ErrAgentStateSlotInvalid for memo slot, got %v", parseErr)
	}
	if parseComponent.hooks.states[0] != 1 || parseComponent.hooks.states[1] != 1 {
		t.Fatalf("invalid writes changed state storage: %#v", parseComponent.hooks.states)
	}
}

func TestDeleteAgentAtomRequiresForceWithSubscribers(t *testing.T) {
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseComponent := &Fiber{
		typeOf: NewComponentType("example/Reader", "Reader", "example/Reader", nil, nil),
		parent: parseRoot,
	}
	parseRoot.child = parseComponent
	parseRegistry := NewAtomRegistry()
	parseRegistry.InitAtom("agent.delete.count", 1)
	parseRegistry.Subscribe("agent.delete.count", parseComponent)
	parseRt := &Runtime{currentRoot: parseRoot, atomRegistry: parseRegistry}

	parseResult, parseErr := DeleteAgentAtom(parseRt, "agent.delete.count", false)
	if !errors.Is(parseErr, ErrAgentAtomHasSubscribers) {
		t.Fatalf("expected ErrAgentAtomHasSubscribers, got %v", parseErr)
	}
	if len(parseResult.Subscribers) != 1 || parseResult.Subscribers[0] == "" {
		t.Fatalf("expected one subscriber ref, got %#v", parseResult.Subscribers)
	}
	if _, parseExists := parseRt.GetAtomValue("agent.delete.count"); !parseExists {
		t.Fatal("atom was deleted without force")
	}

	parseForced, parseForceErr := DeleteAgentAtom(parseRt, "agent.delete.count", true)
	if parseForceErr != nil {
		t.Fatalf("forced DeleteAgentAtom returned error: %v", parseForceErr)
	}
	if !parseForced.Deleted {
		t.Fatal("forced delete did not report Deleted=true")
	}
	if _, parseExists := parseRt.GetAtomValue("agent.delete.count"); parseExists {
		t.Fatal("atom still exists after forced delete")
	}
	if parseRegistry.GetSubscriberCount("agent.delete.count") != 0 {
		t.Fatalf("expected forced delete to clear subscribers, got %d", parseRegistry.GetSubscriberCount("agent.delete.count"))
	}
}
