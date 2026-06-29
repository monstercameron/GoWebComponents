package runtime

import "fmt"

// GetAgentState resolves parseRef, validates parseHookSlot is a state hook, and
// returns the current value stored in that state slot WITHOUT mutating it. It is
// the read counterpart of SetAgentState, used to capture a prior value so an
// agent set-state can be recorded as a reversible mutation.
func GetAgentState(parseRt *Runtime, parseRef string, parseHookSlot int) (any, error) {
	if parseRt == nil {
		return nil, fmt.Errorf("GetAgentState: nil runtime: %w", ErrAgentRefStale)
	}
	if parseHookSlot < 0 {
		return nil, fmt.Errorf("state slot %d: %w", parseHookSlot, ErrAgentStateSlotInvalid)
	}

	schedulerMu.Lock()
	defer schedulerMu.Unlock()

	parseFiber, parseErr := parseRt.resolveAgentRefLocked(parseRef)
	if parseErr != nil {
		return nil, parseErr
	}
	parseHooks := parseFiber.hooks
	if parseHooks == nil || parseHookSlot >= len(parseHooks.signature) || parseHooks.signature[parseHookSlot] != "state" {
		return nil, fmt.Errorf("state slot %d on ref %q: %w", parseHookSlot, parseRef, ErrAgentStateSlotInvalid)
	}
	parseStateIndex := agentStateIndexForHookSlot(parseHooks.signature, parseHookSlot)
	parseStateSlot := parseStateIndex * 2
	if parseStateSlot < 0 || parseStateSlot >= len(parseHooks.states) {
		return nil, fmt.Errorf("state slot %d on ref %q has no state storage: %w", parseHookSlot, parseRef, ErrAgentStateSlotInvalid)
	}
	return parseHooks.states[parseStateSlot], nil
}
