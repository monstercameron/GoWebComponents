package runtime

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
)

// ErrAgentNoHandler reports that the named event (e.g. "onclick") has no
// handler registered on the resolved fiber. It is distinct from
// ErrAgentRefStale so callers can distinguish "node is gone" from
// "node exists but has no such handler".
var ErrAgentNoHandler = errors.New("fiber has no handler for the requested event")

// ErrAgentStateSlotInvalid reports that a bridge.set-state command addressed a
// hook slot that does not exist or is not a state hook.
var ErrAgentStateSlotInvalid = errors.New("agent state slot is invalid")

// ErrAgentAtomHasSubscribers reports that a bridge.delete-atom command tried
// to delete an atom that still has live subscribers without force=true.
var ErrAgentAtomHasSubscribers = errors.New("agent atom has live subscribers")

// EmitAgentEvent resolves parseRef to its live fiber under schedulerMu,
// locates the handler for parseEventName (e.g. "click" or "onclick") in the
// fiber's props map, and invokes it with a synthesised zero-value event. It
// mirrors exactly the type-switch dispatch testkit/render uses so the same
// handler signatures that work in tests work here. The scheduler lock is
// acquired via resolveAgentRefLocked (not ResolveAgentRef) so the caller must
// NOT already hold schedulerMu. Panics inside the handler are caught and
// returned as errors (crash containment — an agent command must never kill
// the runtime). Returns ErrAgentRefStale when the ref no longer resolves,
// ErrAgentRefInvalid when the ref string is malformed, and ErrAgentNoHandler
// when the fiber carries no handler under parseEventName.
func EmitAgentEvent(parseRt *Runtime, parseRef string, parseEventName string, parsePayload map[string]any) error {
	if parseRt == nil {
		return fmt.Errorf("EmitAgentEvent: nil runtime: %w", ErrAgentRefStale)
	}

	// Normalise the event name: accept both "click" and "onclick".
	parseNormName := writeNormaliseEventName(parseEventName)

	schedulerMu.Lock()
	parseFiber, parseErr := parseRt.resolveAgentRefLocked(parseRef)
	schedulerMu.Unlock()
	if parseErr != nil {
		return parseErr
	}

	// Retrieve the handler from fiber props. Props are set during reconcile and
	// are the same map the testkit dispatch reads from MockDOMNode.Props.
	if parseFiber.props == nil {
		return fmt.Errorf("event %q on ref %q: %w", parseNormName, parseRef, ErrAgentNoHandler)
	}
	parseHandler, parseOk := parseFiber.props[parseNormName]
	if !parseOk || parseHandler == nil {
		return fmt.Errorf("event %q on ref %q: %w", parseNormName, parseRef, ErrAgentNoHandler)
	}

	// Invoke the handler, catching any panic for crash containment.
	return writeInvokeHandler(parseHandler, parseNormName, parseRef)
}

// AgentDeleteAtomResult describes the outcome of DeleteAgentAtom.
type AgentDeleteAtomResult struct {
	Deleted     bool
	Subscribers []string
}

// SetAgentState resolves parseRef, validates parseHookSlot against the fiber's
// recorded hook signature, writes the matching state hook's current and pending
// slots, and schedules the same owned-fiber update path used by GoUseState.
func SetAgentState(parseRt *Runtime, parseRef string, parseHookSlot int, parseValue any) error {
	if parseRt == nil {
		return fmt.Errorf("SetAgentState: nil runtime: %w", ErrAgentRefStale)
	}
	if parseHookSlot < 0 {
		return fmt.Errorf("state slot %d: %w", parseHookSlot, ErrAgentStateSlotInvalid)
	}
	if parseErr := parseRt.waitForAgentWriteReady(); parseErr != nil {
		return parseErr
	}

	schedulerMu.Lock()
	parseFiber, parseErr := parseRt.resolveAgentRefLocked(parseRef)
	if parseErr != nil {
		schedulerMu.Unlock()
		return parseErr
	}
	parseHooks := parseFiber.hooks
	if parseHooks == nil || parseHookSlot >= len(parseHooks.signature) || parseHooks.signature[parseHookSlot] != "state" {
		schedulerMu.Unlock()
		return fmt.Errorf("state slot %d on ref %q: %w", parseHookSlot, parseRef, ErrAgentStateSlotInvalid)
	}
	parseStateIndex := agentStateIndexForHookSlot(parseHooks.signature, parseHookSlot)
	parseStateSlot := parseStateIndex * 2
	parsePendingSlot := parseStateSlot + 1
	if parseStateSlot < 0 || parsePendingSlot >= len(parseHooks.states) {
		schedulerMu.Unlock()
		return fmt.Errorf("state slot %d on ref %q has no state storage: %w", parseHookSlot, parseRef, ErrAgentStateSlotInvalid)
	}
	parseCurrentValue := parseHooks.states[parseStateSlot]
	parseCoercedValue, parseCoerceOK := coerceHotReloadValue(parseValue, reflect.TypeOf(parseCurrentValue))
	if !parseCoerceOK {
		schedulerMu.Unlock()
		return fmt.Errorf("state slot %d on ref %q cannot accept value type %T for current type %T: %w", parseHookSlot, parseRef, parseValue, parseCurrentValue, ErrAgentStateSlotInvalid)
	}
	if fastEqual(parseCurrentValue, parseCoercedValue) {
		schedulerMu.Unlock()
		return nil
	}
	parseHooks.states[parsePendingSlot] = parseCoercedValue
	parseHooks.states[parseStateSlot] = parseCoercedValue
	parseTargetFiber := parseHooks.owner
	if parseTargetFiber == nil {
		parseTargetFiber = parseFiber
	}
	schedulerMu.Unlock()

	parseRt.ScheduleOwnedFiberUpdateWithOrigin(parseTargetFiber, "agent.set-state")
	return nil
}

// DeleteAgentAtom deletes one atom from the registry. When live subscribers
// exist and parseForce is false, it fails closed and reports their refs.
func DeleteAgentAtom(parseRt *Runtime, parseID string, parseForce bool) (AgentDeleteAtomResult, error) {
	if parseRt == nil || parseRt.atomRegistry == nil {
		return AgentDeleteAtomResult{}, fmt.Errorf("atom registry not initialized")
	}
	parseTrimmedID := strings.TrimSpace(parseID)
	if parseTrimmedID == "" {
		return AgentDeleteAtomResult{}, fmt.Errorf("delete atom: id is required")
	}

	parseSubscribers, parseExists := parseRt.atomRegistry.agentAtomSubscribers(parseTrimmedID)
	if !parseExists {
		return AgentDeleteAtomResult{}, fmt.Errorf("delete atom: atom %q is not registered", parseTrimmedID)
	}
	parseRefs := parseRt.agentSubscriberRefs(parseSubscribers)
	if len(parseRefs) > 0 && !parseForce {
		return AgentDeleteAtomResult{Subscribers: parseRefs}, fmt.Errorf("delete atom %q: %d live subscribers: %w", parseTrimmedID, len(parseRefs), ErrAgentAtomHasSubscribers)
	}
	parseRt.atomRegistry.deleteAtom(parseTrimmedID)
	parseRt.AdvanceAgentStateVersion()
	return AgentDeleteAtomResult{Deleted: true, Subscribers: parseRefs}, nil
}

func agentStateIndexForHookSlot(parseSignature []string, parseHookSlot int) int {
	parseStateIndex := 0
	for parseIndex := 0; parseIndex < parseHookSlot; parseIndex++ {
		if parseSignature[parseIndex] == "state" {
			parseStateIndex++
		}
	}
	return parseStateIndex
}

func (parseRt *Runtime) waitForAgentWriteReady() error {
	const parseTimeout = 2 * time.Second
	parseDeadline := time.Now().Add(parseTimeout)
	for {
		schedulerMu.Lock()
		isInFlight := parseRt != nil && parseRt.updateScheduled && parseRt.wipRoot != nil
		schedulerMu.Unlock()
		if !isInFlight {
			return nil
		}
		if time.Now().After(parseDeadline) {
			return fmt.Errorf("agent write timed out waiting for in-flight render to finish")
		}
		time.Sleep(time.Millisecond)
	}
}

func (parseRt *Runtime) agentSubscriberRefs(parseSubscribers []*Fiber) []string {
	if len(parseSubscribers) == 0 {
		return nil
	}
	parseRefs := make([]string, 0, len(parseSubscribers))
	parseSeen := make(map[string]struct{}, len(parseSubscribers))
	for _, parseFiber := range parseSubscribers {
		if parseFiber == nil {
			continue
		}
		parseRef := AgentRefForFiber(parseFiber)
		if strings.TrimSpace(parseRef) == "" {
			continue
		}
		if _, parseExists := parseSeen[parseRef]; parseExists {
			continue
		}
		parseSeen[parseRef] = struct{}{}
		parseRefs = append(parseRefs, parseRef)
	}
	sort.Strings(parseRefs)
	return parseRefs
}

// writeNormaliseEventName normalises an event name so that both "click" and
// "onclick" resolve to the prop key the reconciler stores ("onclick").
func writeNormaliseEventName(parseName string) string {
	if len(parseName) == 0 {
		return parseName
	}
	// Already prefixed: pass through as-is.
	if len(parseName) > 2 && parseName[:2] == "on" {
		return parseName
	}
	return "on" + parseName
}

// writeInvokeHandler calls parseHandler using the same type switch testkit
// dispatch uses. A synthetic zero-value event is supplied for handler
// signatures that expect one. Panics inside the handler are converted to a
// returned error (crash containment). Platform-specific handler types
// (func(js.Value), func(GoEvent)) are handled by writeInvokeHandlerPlatform
// which is provided by agent_write_wasm.go / agent_write_native.go.
func writeInvokeHandler(parseHandler any, parseEventName string, parseRef string) (parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseErr = fmt.Errorf("handler %q on ref %q panicked: %v", parseEventName, parseRef, parseRecovered)
		}
	}()

	switch parseTyped := parseHandler.(type) {
	case func():
		parseTyped()
		return nil
	case func() error:
		if parseCallErr := parseTyped(); parseCallErr != nil {
			return fmt.Errorf("handler %q on ref %q returned error: %w", parseEventName, parseRef, parseCallErr)
		}
		return nil
	case func(string):
		// String handlers receive the empty string (no DOM value in agent context).
		parseTyped("")
		return nil
	default:
		// Delegate to the platform-specific implementation for func(js.Value),
		// func(GoEvent), etc. Returns a non-nil error when the type is unknown.
		return writeInvokeHandlerPlatform(parseTyped, parseEventName, parseRef)
	}
}
