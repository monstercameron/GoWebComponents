package runtime

import (
	"errors"
	"testing"
)

func buildAgentStateReadFixture() (*Runtime, string) {
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseComponent := &Fiber{
		typeOf: &ComponentType{Name: "Counter", QualifiedName: "example.Counter"},
		parent: parseRoot,
		hooks: &Hooks{
			signature: []string{"memo", "state", "effect", "state"},
			states:    []any{"first", nil, 42, nil},
		},
	}
	parseRoot.child = parseComponent
	return &Runtime{currentRoot: parseRoot}, AgentRefForFiber(parseComponent)
}

func TestGetAgentStateReadsStateSlotWithoutMutating(parseT *testing.T) {
	parseRt, parseRef := buildAgentStateReadFixture()
	parseValue, parseErr := GetAgentState(parseRt, parseRef, 1)
	if parseErr != nil {
		parseT.Fatalf("GetAgentState first state: %v", parseErr)
	}
	if parseValue != "first" {
		parseT.Fatalf("first state = %#v, want first", parseValue)
	}
	parseValue2, parseErr := GetAgentState(parseRt, parseRef, 3)
	if parseErr != nil {
		parseT.Fatalf("GetAgentState second state: %v", parseErr)
	}
	if parseValue2 != 42 {
		parseT.Fatalf("second state = %#v, want 42", parseValue2)
	}
	if parseRt.currentRoot.child.hooks.states[0] != "first" || parseRt.currentRoot.child.hooks.states[2] != 42 {
		parseT.Fatalf("GetAgentState mutated hook storage: %#v", parseRt.currentRoot.child.hooks.states)
	}
}

func TestGetAgentStateRejectsInvalidInputs(parseT *testing.T) {
	if _, parseErr := GetAgentState(nil, "ref", 0); !errors.Is(parseErr, ErrAgentRefStale) {
		parseT.Fatalf("nil runtime error = %v, want ErrAgentRefStale", parseErr)
	}
	parseRt, parseRef := buildAgentStateReadFixture()
	if _, parseErr := GetAgentState(parseRt, parseRef, -1); !errors.Is(parseErr, ErrAgentStateSlotInvalid) {
		parseT.Fatalf("negative slot error = %v, want ErrAgentStateSlotInvalid", parseErr)
	}
	if _, parseErr := GetAgentState(parseRt, parseRef, 0); !errors.Is(parseErr, ErrAgentStateSlotInvalid) {
		parseT.Fatalf("non-state slot error = %v, want ErrAgentStateSlotInvalid", parseErr)
	}
	if _, parseErr := GetAgentState(parseRt, parseRef, 8); !errors.Is(parseErr, ErrAgentStateSlotInvalid) {
		parseT.Fatalf("out-of-range slot error = %v, want ErrAgentStateSlotInvalid", parseErr)
	}
}

func TestGetAgentStateRejectsMissingStateStorage(parseT *testing.T) {
	parseRt, parseRef := buildAgentStateReadFixture()
	parseRt.currentRoot.child.hooks.states = []any{"only-one-state"}
	if _, parseErr := GetAgentState(parseRt, parseRef, 3); !errors.Is(parseErr, ErrAgentStateSlotInvalid) {
		parseT.Fatalf("missing storage error = %v, want ErrAgentStateSlotInvalid", parseErr)
	}
}
