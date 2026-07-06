package runtime

import (
	"fmt"
	"maps"
	"slices"
	"sync"
)

// AtomRegistry manages global state atoms with fine-grained reactivity.
// Each atom has a unique ID and tracks which fibers are subscribed to it.
type AtomRegistry struct {
	mu            sync.RWMutex
	atoms         map[string]any
	subscriptions map[string]map[*Fiber]bool // atomID -> set of subscribed fibers
	derived       map[string]derivedAtom
	dependents    map[string]map[string]bool // source atom id -> derived ids
}

type derivedAtom struct {
	deps       []string
	compute    func() any
	active     bool
	generation uint64 // incremented each time the entry is replaced by RegisterDerivedAtom
}

// NewAtomRegistry creates a new atom registry.
func NewAtomRegistry() *AtomRegistry {
	return &AtomRegistry{
		atoms:         make(map[string]any),
		subscriptions: make(map[string]map[*Fiber]bool),
		derived:       make(map[string]derivedAtom),
		dependents:    make(map[string]map[string]bool),
	}
}

// RegisterDerivedAtom registers or replaces a derived atom and computes its current value.
func (parseAr *AtomRegistry) RegisterDerivedAtom(parseId string, parseDeps []string, parseCompute func() any) error {
	if parseAr == nil {
		return fmt.Errorf("atom registry not initialized")
	}
	if parseCompute == nil {
		return fmt.Errorf("derived atom %s compute function cannot be nil", parseId)
	}
	if slices.Contains(parseDeps, parseId) {
		return fmt.Errorf("derived atom %s cannot depend on itself", parseId)
	}

	parseAr.mu.Lock()
	parseNextGen := uint64(0)
	if parseExisting, parseOk := parseAr.derived[parseId]; parseOk {
		for _, parseDep2 := range parseExisting.deps {
			if parseDependents := parseAr.dependents[parseDep2]; parseDependents != nil {
				delete(parseDependents, parseId)
				if len(parseDependents) == 0 {
					delete(parseAr.dependents, parseDep2)
				}
			}
		}
		parseNextGen = parseExisting.generation + 1
	}
	for _, parseDep3 := range parseDeps {
		if parseAr.hasDerivedDependencyPathLocked(parseDep3, parseId, map[string]bool{}) {
			parseAr.mu.Unlock()
			parseErr := fmt.Errorf("derived atom cycle detected involving %s", parseId)
			ReportDiagnostic("state", DiagnosticWarning, parseErr.Error())
			return parseErr
		}
	}
	parseCloneDeps := append([]string(nil), parseDeps...)
	parseAr.derived[parseId] = derivedAtom{deps: parseCloneDeps, compute: parseCompute, active: true, generation: parseNextGen}
	for _, parseDep4 := range parseCloneDeps {
		if parseAr.dependents[parseDep4] == nil {
			parseAr.dependents[parseDep4] = make(map[string]bool)
		}
		parseAr.dependents[parseDep4][parseId] = true
	}
	parseAr.mu.Unlock()

	_, parseErr2 := parseAr.recomputeDerived(parseId, nil)
	if parseErr2 != nil {
		ReportDiagnostic("state", DiagnosticWarning, parseErr2.Error())
	}
	return parseErr2
}

// hasDerivedDependencyPathLocked is a core package helper.
func (parseAr *AtomRegistry) hasDerivedDependencyPathLocked(parseStart string, parseTarget string, parseSeen map[string]bool) bool {
	if parseStart == parseTarget {
		return true
	}
	if parseSeen[parseStart] {
		return false
	}
	parseSeen[parseStart] = true
	parseDerived, parseOk := parseAr.derived[parseStart]
	if !parseOk || !parseDerived.active {
		return false
	}
	for _, parseDep := range parseDerived.deps {
		if parseAr.hasDerivedDependencyPathLocked(parseDep, parseTarget, parseSeen) {
			return true
		}
	}
	return false
}

// GetAtom retrieves an atom's current value.
func (parseAr *AtomRegistry) GetAtom(parseId string) (any, bool) {
	parseAr.mu.RLock()
	parseAtom, parseOk := parseAr.atoms[parseId]
	parseAr.mu.RUnlock()
	if !parseOk {
		return nil, false
	}
	return parseAtom, true
}

// agentAtomSubscribers returns the current subscriber fibers for one atom and
// whether the atom value exists. It is used by the agent bridge delete guard.
func (parseAr *AtomRegistry) agentAtomSubscribers(parseId string) ([]*Fiber, bool) {
	if parseAr == nil {
		return nil, false
	}
	parseAr.mu.RLock()
	_, parseExists := parseAr.atoms[parseId]
	parseSubs := parseAr.subscriptions[parseId]
	parseSubscribers := make([]*Fiber, 0, len(parseSubs))
	for parseFiber := range parseSubs {
		parseSubscribers = append(parseSubscribers, parseFiber)
	}
	parseAr.mu.RUnlock()
	return parseSubscribers, parseExists
}

// deleteAtom removes one atom value and its registry bookkeeping.
func (parseAr *AtomRegistry) deleteAtom(parseId string) {
	if parseAr == nil {
		return
	}
	parseAr.mu.Lock()
	delete(parseAr.atoms, parseId)
	delete(parseAr.subscriptions, parseId)
	if parseDerived, parseOk := parseAr.derived[parseId]; parseOk {
		for _, parseDep := range parseDerived.deps {
			if parseDependents := parseAr.dependents[parseDep]; parseDependents != nil {
				delete(parseDependents, parseId)
				if len(parseDependents) == 0 {
					delete(parseAr.dependents, parseDep)
				}
			}
		}
		delete(parseAr.derived, parseId)
	}
	delete(parseAr.dependents, parseId)
	parseAr.mu.Unlock()
}

// SetAtom updates an atom's value and returns subscribed fibers.
func (parseAr *AtomRegistry) SetAtom(parseId string, parseValue any) []*Fiber {
	parseAr.mu.Lock()

	// Update or create atom
	parseAr.atoms[parseId] = parseValue

	// Get all subscribed fibers
	if parseSubs, parseOk := parseAr.subscriptions[parseId]; parseOk {
		parseSubscribers := make([]*Fiber, 0, len(parseSubs))
		for parseFiber := range parseSubs {
			parseSubscribers = append(parseSubscribers, parseFiber)
		}
		parseAr.mu.Unlock()
		return parseSubscribers
	}
	parseAr.mu.Unlock()

	return nil
}

// setAtomAndNotify is a core package helper.
func (parseAr *AtomRegistry) setAtomAndNotify(parseId string, parseValue any, parseNotify func(*Fiber)) {
	if parseNotify == nil {
		_ = parseAr.SetAtom(parseId, parseValue)
		return
	}

	parseFibers := parseAr.setValueAndCollectSubscribers(parseId, parseValue)
	for _, parseDerivedID := range parseAr.listDependents(parseId) {
		parseDerivedFibers, parseErr := parseAr.recomputeDerived(parseDerivedID, map[string]bool{parseId: true})
		if parseErr != nil {
			ReportDiagnostic("state", DiagnosticWarning, parseErr.Error())
			continue
		}
		parseFibers = append(parseFibers, parseDerivedFibers...)
	}
	notifyFibersUnique(parseFibers, parseNotify)
}

// updateAtomAndNotify atomically applies parseFn to the atom's current value
// (or parseDefault when the atom has no value yet) and stores the result,
// holding the registry lock across the whole read-compute-write. This closes
// the get-then-set race in functional updates: two goroutines that each read
// the same old value, compute, and write can no longer clobber one another —
// with the lock held, the second updater always observes the first's write.
//
// parseFn MUST be a pure transform of the previous value. Because the registry
// lock is held while it runs, re-entering the registry from within it (reading
// or writing ANY atom, e.g. via Atom.Get/Set) deadlocks. This matches the
// atomic-compute contract of sync.Map-style APIs. When the result equals the
// current value the update is a no-op and no subscriber is notified, matching
// the Set no-op-on-equal behavior.
func (parseAr *AtomRegistry) updateAtomAndNotify(parseId string, parseDefault any, parseFn func(any) any, parseNotify func(*Fiber)) {
	parseAr.mu.Lock()
	parseCurrent, parseOk := parseAr.atoms[parseId]
	if !parseOk {
		parseCurrent = parseDefault
	}
	parseNext := parseFn(parseCurrent)
	if fastEqual(parseCurrent, parseNext) {
		parseAr.mu.Unlock()
		return
	}
	parseAr.atoms[parseId] = parseNext
	parseFibers := parseAr.collectSubscribersLocked(parseId)
	parseAr.mu.Unlock()

	if parseNotify == nil {
		return
	}
	for _, parseDerivedID := range parseAr.listDependents(parseId) {
		parseDerivedFibers, parseErr := parseAr.recomputeDerived(parseDerivedID, map[string]bool{parseId: true})
		if parseErr != nil {
			ReportDiagnostic("state", DiagnosticWarning, parseErr.Error())
			continue
		}
		parseFibers = append(parseFibers, parseDerivedFibers...)
	}
	notifyFibersUnique(parseFibers, parseNotify)
}

// setValueAndCollectSubscribers is a core package helper.
func (parseAr *AtomRegistry) setValueAndCollectSubscribers(parseId string, parseValue any) []*Fiber {
	parseAr.mu.Lock()
	parseAr.atoms[parseId] = parseValue
	parseFibers := parseAr.collectSubscribersLocked(parseId)
	parseAr.mu.Unlock()
	return parseFibers
}

// collectSubscribersLocked is a core package helper.
func (parseAr *AtomRegistry) collectSubscribersLocked(parseId string) []*Fiber {
	parseSubs, parseOk := parseAr.subscriptions[parseId]
	if !parseOk || len(parseSubs) == 0 {
		return nil
	}
	parseFibers := make([]*Fiber, 0, len(parseSubs))
	for parseFiber := range parseSubs {
		parseFibers = append(parseFibers, parseFiber)
	}
	return parseFibers
}

// listDependents is a core package helper.
func (parseAr *AtomRegistry) listDependents(parseId string) []string {
	parseAr.mu.RLock()
	parseDependents := parseAr.dependents[parseId]
	if len(parseDependents) == 0 {
		parseAr.mu.RUnlock()
		return nil
	}
	parseIds := make([]string, 0, len(parseDependents))
	for parseDerivedID := range parseDependents {
		parseIds = append(parseIds, parseDerivedID)
	}
	parseAr.mu.RUnlock()
	return parseIds
}

// recomputeDerived is a core package helper.
func (parseAr *AtomRegistry) recomputeDerived(parseId string, parseTrail map[string]bool) ([]*Fiber, error) {
	if parseTrail == nil {
		parseTrail = map[string]bool{}
	}
	if parseTrail[parseId] {
		return nil, fmt.Errorf("derived atom cycle detected involving %s", parseId)
	}
	parseTrail[parseId] = true
	defer delete(parseTrail, parseId)

	parseAr.mu.RLock()
	parseDerived, parseOk := parseAr.derived[parseId]
	parseAr.mu.RUnlock()
	if !parseOk || !parseDerived.active {
		return nil, nil
	}

	// Capture the generation counter before releasing the lock.  A concurrent
	// RegisterDerivedAtom may replace derived[parseId] (bumping generation) while
	// compute() runs; we must not overwrite the newer value it already stored.
	parseExpectedGen := parseDerived.generation
	parseValue := parseDerived.compute()
	parseFibers, parseChanged := parseAr.setDerivedValueIfGenerationMatches(parseId, parseExpectedGen, parseValue)
	if !parseChanged {
		return nil, nil
	}
	for _, parseDependentID := range parseAr.listDependents(parseId) {
		parseNested, parseErr := parseAr.recomputeDerived(parseDependentID, parseTrail)
		if parseErr != nil {
			return parseFibers, parseErr
		}
		parseFibers = append(parseFibers, parseNested...)
	}
	return parseFibers, nil
}

// setDerivedValueIfGenerationMatches writes parseValue only when derived[parseId] still
// carries parseExpectedGen (meaning RegisterDerivedAtom has not replaced the
// entry since we last read it) and the value has changed.  Guards against a
// stale recomputation overwriting a fresher value from a concurrent registration.
func (parseAr *AtomRegistry) setDerivedValueIfGenerationMatches(
	parseId string,
	parseExpectedGen uint64,
	parseValue any,
) ([]*Fiber, bool) {
	parseAr.mu.Lock()
	parseCurrent, parseOk := parseAr.derived[parseId]
	if !parseOk || !parseCurrent.active || parseCurrent.generation != parseExpectedGen {
		// A newer registration already computed and stored its own value; discard ours.
		parseAr.mu.Unlock()
		return nil, false
	}
	if parsePrev, parsePrevOk := parseAr.atoms[parseId]; parsePrevOk && fastEqual(parsePrev, parseValue) {
		parseAr.mu.Unlock()
		return nil, false
	}
	parseAr.atoms[parseId] = parseValue
	parseFibers := parseAr.collectSubscribersLocked(parseId)
	parseAr.mu.Unlock()
	return parseFibers, true
}

// notifyFibersUnique is a core package helper.
func notifyFibersUnique(parseFibers []*Fiber, parseNotify func(*Fiber)) {
	if parseNotify == nil || len(parseFibers) == 0 {
		return
	}
	// Fast path: skip map allocation for small subscriber counts (covers the common case).
	if len(parseFibers) <= 4 {
		var parseSeen [4]*Fiber
		parseN := 0
		for _, parseFiber := range parseFibers {
			if parseFiber == nil {
				continue
			}
			parseDup := false
			for parseI := 0; parseI < parseN; parseI++ {
				if parseSeen[parseI] == parseFiber {
					parseDup = true
					break
				}
			}
			if !parseDup {
				parseSeen[parseN] = parseFiber
				parseN++
				parseNotify(parseFiber)
			}
		}
		return
	}
	parseSeenMap := make(map[*Fiber]bool, len(parseFibers))
	for _, parseFiber := range parseFibers {
		if parseFiber == nil || parseSeenMap[parseFiber] {
			continue
		}
		parseSeenMap[parseFiber] = true
		parseNotify(parseFiber)
	}
}

// InitAtom initializes an atom if it doesn't exist.
func (parseAr *AtomRegistry) InitAtom(parseId string, parseInitialValue any) {
	parseAr.mu.Lock()
	if _, parseExists := parseAr.atoms[parseId]; !parseExists {
		parseAr.atoms[parseId] = parseInitialValue
	}
	parseAr.mu.Unlock()
}

// Subscribe adds a fiber to an atom's subscription list.
func (parseAr *AtomRegistry) Subscribe(parseAtomID string, parseFiber *Fiber) {
	parseAr.mu.Lock()

	if parseAr.subscriptions[parseAtomID] == nil {
		parseAr.subscriptions[parseAtomID] = make(map[*Fiber]bool)
	}
	parseAr.subscriptions[parseAtomID][parseFiber] = true
	parseAr.mu.Unlock()
}

// Unsubscribe removes a fiber from an atom's subscription list.
func (parseAr *AtomRegistry) Unsubscribe(parseAtomID string, parseFiber *Fiber) {
	parseAr.mu.Lock()

	if parseSubs, parseOk := parseAr.subscriptions[parseAtomID]; parseOk {
		delete(parseSubs, parseFiber)
		if len(parseSubs) == 0 {
			delete(parseAr.subscriptions, parseAtomID)
		}
	}
	parseAr.mu.Unlock()
}

// UnsubscribeMany removes a fiber from several atom subscriptions under one lock.
func (parseAr *AtomRegistry) UnsubscribeMany(parseAtomIDs []string, parseFiber *Fiber) {
	parseAr.mu.Lock()

	for _, parseAtomID := range parseAtomIDs {
		if parseSubs, parseOk := parseAr.subscriptions[parseAtomID]; parseOk {
			delete(parseSubs, parseFiber)
			if len(parseSubs) == 0 {
				delete(parseAr.subscriptions, parseAtomID)
			}
		}
	}
	parseAr.mu.Unlock()
}

// MoveSubscriptions transfers a fiber's ownership across several atom subscriptions under one lock.
func (parseAr *AtomRegistry) MoveSubscriptions(parseAtomIDs []string, parseFrom *Fiber, parseTo *Fiber) {
	if parseAr == nil || len(parseAtomIDs) == 0 || parseFrom == parseTo {
		return
	}

	parseAr.mu.Lock()
	for _, parseAtomID := range parseAtomIDs {
		if parseAtomID == "" {
			continue
		}
		parseSubs := parseAr.subscriptions[parseAtomID]
		if parseSubs == nil {
			if parseTo == nil {
				continue
			}
			parseSubs = make(map[*Fiber]bool)
			parseAr.subscriptions[parseAtomID] = parseSubs
		}
		if parseFrom != nil {
			delete(parseSubs, parseFrom)
		}
		if parseTo != nil {
			parseSubs[parseTo] = true
		}
		if len(parseSubs) == 0 {
			delete(parseAr.subscriptions, parseAtomID)
		}
	}
	parseAr.mu.Unlock()
}

// MoveSubscription transfers a fiber's ownership for a single atom subscription.
func (parseAr *AtomRegistry) MoveSubscription(parseAtomID string, parseFrom *Fiber, parseTo *Fiber) {
	if parseAr == nil || parseAtomID == "" || parseFrom == parseTo {
		return
	}
	parseAr.mu.Lock()
	parseSubs := parseAr.subscriptions[parseAtomID]
	if parseSubs == nil {
		if parseTo == nil {
			parseAr.mu.Unlock()
			return
		}
		parseSubs = make(map[*Fiber]bool)
		parseAr.subscriptions[parseAtomID] = parseSubs
	}
	if parseFrom != nil {
		delete(parseSubs, parseFrom)
	}
	if parseTo != nil {
		parseSubs[parseTo] = true
	}
	if len(parseSubs) == 0 {
		delete(parseAr.subscriptions, parseAtomID)
	}
	parseAr.mu.Unlock()
}

// UnsubscribeFiberFromAll removes a fiber from all atom subscriptions.
func (parseAr *AtomRegistry) UnsubscribeFiberFromAll(parseFiber *Fiber) {
	parseAr.mu.Lock()

	for parseAtomID, parseSubs := range parseAr.subscriptions {
		delete(parseSubs, parseFiber)
		if len(parseSubs) == 0 {
			delete(parseAr.subscriptions, parseAtomID)
		}
	}
	parseAr.mu.Unlock()
}

// GetSubscriberCount returns the number of fibers subscribed to an atom.
func (parseAr *AtomRegistry) GetSubscriberCount(parseAtomID string) int {
	parseAr.mu.RLock()
	if parseSubs, parseOk := parseAr.subscriptions[parseAtomID]; parseOk {
		parseCount := len(parseSubs)
		parseAr.mu.RUnlock()
		return parseCount
	}
	parseAr.mu.RUnlock()
	return 0
}

// GetAtomCount returns the total number of atoms.
func (parseAr *AtomRegistry) GetAtomCount() int {
	parseAr.mu.RLock()
	parseCount := len(parseAr.atoms)
	parseAr.mu.RUnlock()
	return parseCount
}

// GetSubscriberTotal returns the total number of atom subscriptions across all atoms.
func (parseAr *AtomRegistry) GetSubscriberTotal() int {
	if parseAr == nil {
		return 0
	}
	parseAr.mu.RLock()
	defer parseAr.mu.RUnlock()
	parseTotal := 0
	for _, parseSubs := range parseAr.subscriptions {
		parseTotal += len(parseSubs)
	}
	return parseTotal
}

// Snapshot returns a shallow copy of all atom values currently stored.
func (parseAr *AtomRegistry) Snapshot() map[string]any {
	if parseAr == nil {
		return nil
	}

	parseAr.mu.RLock()
	defer parseAr.mu.RUnlock()
	if len(parseAr.atoms) == 0 {
		return map[string]any{}
	}

	parseSnapshot := make(map[string]any, len(parseAr.atoms))
	maps.Copy(parseSnapshot, parseAr.atoms)
	return parseSnapshot
}

// RestoreSnapshot merges atom values from snapshot and returns subscribed fibers
// that should be notified about the updates.
func (parseAr *AtomRegistry) RestoreSnapshot(parseSnapshot map[string]any) []*Fiber {
	if parseAr == nil || len(parseSnapshot) == 0 {
		return nil
	}

	parseUnique := make(map[*Fiber]bool)
	parseAr.mu.Lock()
	for parseId, parseValue := range parseSnapshot {
		parseAr.atoms[parseId] = parseValue
		if parseSubs, parseOk := parseAr.subscriptions[parseId]; parseOk {
			for parseFiber := range parseSubs {
				parseUnique[parseFiber] = true
			}
		}
	}
	parseAr.mu.Unlock()

	if len(parseUnique) == 0 {
		return nil
	}

	parseFibers := make([]*Fiber, 0, len(parseUnique))
	for parseFiber2 := range parseUnique {
		parseFibers = append(parseFibers, parseFiber2)
	}
	return parseFibers
}

// GoUseAtom provides access to global state with fine-grained reactivity.
// Unlike useState which is local to a component, atoms are shared across components.
// When an atom updates, only components that use that specific atom re-render.
func GoUseAtom[T any](parseRt *Runtime, parseId string, parseInitialValue T) (func() T, func(any)) {
	if parseRt.atomRegistry == nil {
		panic(actionableGoUseAtomRegistryPanic())
	}

	parseFiber := requireCurrentHookFiber("GoUseAtom")

	if parseFiber.hooks == nil {
		parseFiber.hooks = &Hooks{owner: parseFiber}
	} else if parseFiber.hooks.owner == nil {
		parseFiber.hooks.owner = parseFiber
	}

	recordHookSignature(parseFiber.hooks, "atom")
	parseFiber.hooks.index++

	parseAtomIdx := parseFiber.hooks.atomIndex
	parseFiber.hooks.atomIndex++
	parseHooks := parseFiber.hooks
	parseTrackedAtomID := ""
	hasTrackedAtom := len(parseHooks.atoms) > parseAtomIdx
	if hasTrackedAtom {
		parseTrackedAtomID = parseHooks.atoms[parseAtomIdx]
	}
	hasAccessor := len(parseHooks.atomFuncs) > parseAtomIdx

	// Stable rerenders of the same atom do not need to re-initialize the registry entry.
	if !hasTrackedAtom || parseTrackedAtomID != parseId {
		parseRt.atomRegistry.InitAtom(parseId, parseInitialValue)
	}

	// Track subscription in fiber for efficient cleanup
	if !hasTrackedAtom {
		parseNeeded := parseAtomIdx + 1
		if parseNeeded <= cap(parseHooks.atoms) {
			parseHooks.atoms = parseHooks.atoms[:parseNeeded]
		} else {
			parseNewAtoms := make([]string, parseNeeded, parseNeeded*2)
			copy(parseNewAtoms, parseHooks.atoms)
			parseHooks.atoms = parseNewAtoms
		}
		parseHooks.atoms[parseAtomIdx] = parseId
		if parseRt.hydrating {
			parseRt.queueHydrationSubscription(parseId, parseFiber, true)
		} else {
			parseRt.atomRegistry.Subscribe(parseId, parseFiber)
		}
	} else if parseTrackedAtomID != parseId {
		if parseTrackedAtomID != "" {
			if parseRt.hydrating {
				parseRt.queueHydrationSubscription(parseTrackedAtomID, parseFiber, false)
			} else {
				parseRt.atomRegistry.Unsubscribe(parseTrackedAtomID, parseFiber)
			}
		}
		parseHooks.atoms[parseAtomIdx] = parseId
		if parseRt.hydrating {
			parseRt.queueHydrationSubscription(parseId, parseFiber, true)
		} else {
			parseRt.atomRegistry.Subscribe(parseId, parseFiber)
		}
	}

	parseNilableState := isNilableType[T]()
	isParseAccessorNeedsRefresh := !hasAccessor || parseTrackedAtomID != parseId
	if isParseAccessorNeedsRefresh {
		parseNeeded2 := parseAtomIdx + 1
		if parseNeeded2 <= cap(parseHooks.atomFuncs) {
			parseHooks.atomFuncs = parseHooks.atomFuncs[:parseNeeded2]
		} else {
			parseNewAccessors := make([]atomAccessorValue, parseNeeded2, parseNeeded2*2)
			copy(parseNewAccessors, parseHooks.atomFuncs)
			parseHooks.atomFuncs = parseNewAccessors
		}

		get := func() T {
			parseValue, parseOk := parseRt.atomRegistry.GetAtom(parseId)
			if !parseOk {
				return parseInitialValue
			}

			if parseTyped, parseOk2 := parseValue.(T); parseOk2 {
				return parseTyped
			}

			return parseInitialValue
		}

		set := func(parseNewValueOrUpdater any) {
			apply := func(parseUpdateOrigin string) {
				parseCurrentValue := get()
				parseNewValue, parseOk3 := resolveStateUpdateValue(parseCurrentValue, parseNewValueOrUpdater, parseNilableState)
				if !parseOk3 {
					return
				}

				if fastEqual(parseCurrentValue, parseNewValue) {
					return
				}

				parseRt.atomRegistry.setAtomAndNotify(parseId, parseNewValue, func(parseFiber2 *Fiber) {
					parseRt.ScheduleSubscribedFiberUpdateWithOrigin(parseFiber2, parseUpdateOrigin)
				})
			}

			if parseRt.ShouldDeferStateUpdates() {
				parseRt.ScheduleTransition(func() {
					apply("transition")
				})
				return
			}

			apply("atom")
		}

		parseHooks.atomFuncs[parseAtomIdx] = atomAccessorValue{getter: get, setter: set}
	}

	get, _ := parseHooks.atomFuncs[parseAtomIdx].getter.(func() T)
	set, _ := parseHooks.atomFuncs[parseAtomIdx].setter.(func(any))
	if get == nil || set == nil {
		panic(actionableGoUseAtomAccessorPanic())
	}

	return get, set
}

// CleanupAtomSubscriptions removes all atom subscriptions for a fiber
// This should be called when a fiber is being removed from the tree
func (parseRt *Runtime) CleanupAtomSubscriptions(parseFiber *Fiber) {
	if parseRt.atomRegistry == nil {
		return
	}

	// Optimization: Only unsubscribe from atoms this fiber is actually using
	if parseFiber.hooks != nil && len(parseFiber.hooks.atoms) > 0 {
		parseRt.atomRegistry.UnsubscribeMany(parseFiber.hooks.atoms, parseFiber)
		if len(parseFiber.reactiveSourceIDs) > 0 {
			parseRt.atomRegistry.UnsubscribeMany(parseFiber.reactiveSourceIDs, parseFiber)
			parseFiber.reactiveSourceIDs = nil
			parseFiber.reactiveAtomID = ""
		}
		return
	}
	if len(parseFiber.reactiveSourceIDs) > 0 {
		parseRt.atomRegistry.UnsubscribeMany(parseFiber.reactiveSourceIDs, parseFiber)
		parseFiber.reactiveSourceIDs = nil
		parseFiber.reactiveAtomID = ""
		return
	}

	// A fiber with no recorded atoms and no reactive source ids was never
	// subscribed to anything: every Subscribe call site records the id on
	// hooks.atoms or fiber.reactiveSourceIDs at the same site (state.go,
	// syncFineGrainedSubscriptions, hydration-deferred variants). The old
	// registry-wide UnsubscribeFiberFromAll fallback here took the registry
	// mutex once per deleted fiber (hundreds of times per bulk removal) to
	// scan for subscriptions that cannot exist. Pinned by
	// TestPlainFiberDeletionSkipsAtomRegistry.
}

// GetAtomValue is a helper to get an atom value directly (for debugging/testing)
func (parseRt *Runtime) GetAtomValue(parseId string) (any, bool) {
	if parseRt.atomRegistry == nil {
		return nil, false
	}
	return parseRt.atomRegistry.GetAtom(parseId)
}

// SetAtomValue is a helper to set an atom value directly (for debugging/testing)
func (parseRt *Runtime) SetAtomValue(parseId string, parseValue any) error {
	if parseRt.atomRegistry == nil {
		return fmt.Errorf("atom registry not initialized")
	}
	apply := func(parseUpdateOrigin string) {
		parseRt.atomRegistry.setAtomAndNotify(parseId, parseValue, func(parseFiber *Fiber) {
			parseRt.ScheduleSubscribedFiberUpdateWithOrigin(parseFiber, parseUpdateOrigin)
		})
	}
	if parseRt.ShouldDeferStateUpdates() {
		parseRt.ScheduleTransition(func() {
			apply("transition")
		})
		return nil
	}

	apply("atom")

	return nil
}

// UpdateAtomValue atomically transforms an atom's value with parseFn, holding the
// registry lock across the read-compute-write so concurrent functional updates
// cannot lose each other's writes. parseDefault supplies the previous value when
// the atom has no entry yet. parseFn MUST be pure and MUST NOT re-enter the atom
// registry (that deadlocks under the held lock); see updateAtomAndNotify.
func (parseRt *Runtime) UpdateAtomValue(parseId string, parseDefault any, parseFn func(any) any) error {
	if parseRt.atomRegistry == nil {
		return fmt.Errorf("atom registry not initialized")
	}
	apply := func(parseUpdateOrigin string) {
		parseRt.atomRegistry.updateAtomAndNotify(parseId, parseDefault, parseFn, func(parseFiber *Fiber) {
			parseRt.ScheduleSubscribedFiberUpdateWithOrigin(parseFiber, parseUpdateOrigin)
		})
	}
	if parseRt.ShouldDeferStateUpdates() {
		parseRt.ScheduleTransition(func() {
			apply("transition")
		})
		return nil
	}

	apply("atom")

	return nil
}

// InitAtomValue seeds an atom with value only if it has no value yet, without
// notifying any subscribers. It is the no-side-effect seeding primitive used by
// non-hook global-atom handles (state.GlobalAtom) so constructing a handle never
// clobbers a value written earlier (the G39 pre-render-write-survives guarantee)
// and never schedules a spurious render. Returns true if it seeded.
func (parseRt *Runtime) InitAtomValue(parseId string, parseValue any) bool {
	if parseRt == nil || parseRt.atomRegistry == nil {
		return false
	}
	if _, parseExists := parseRt.atomRegistry.GetAtom(parseId); parseExists {
		return false
	}
	parseRt.atomRegistry.InitAtom(parseId, parseValue)
	return true
}

// RegisterDerivedAtom is a core package helper.
func (parseRt *Runtime) RegisterDerivedAtom(parseId string, parseDeps []string, parseCompute func() any) error {
	if parseRt == nil || parseRt.atomRegistry == nil {
		return fmt.Errorf("atom registry not initialized")
	}
	return parseRt.atomRegistry.RegisterDerivedAtom(parseId, parseDeps, parseCompute)
}

// SnapshotAtoms returns a copy of all currently registered atoms.
func (parseRt *Runtime) SnapshotAtoms() map[string]any {
	if parseRt == nil || parseRt.atomRegistry == nil {
		return map[string]any{}
	}
	return parseRt.atomRegistry.Snapshot()
}

// RestoreAtomSnapshot merges atom values from snapshot and schedules updates for
// any subscribed fibers.
func (parseRt *Runtime) RestoreAtomSnapshot(parseSnapshot map[string]any) error {
	if parseRt == nil || parseRt.atomRegistry == nil {
		return fmt.Errorf("atom registry not initialized")
	}
	apply := func(parseUpdateOrigin string) {
		for _, parseFiber := range parseRt.atomRegistry.RestoreSnapshot(parseSnapshot) {
			parseRt.ScheduleSubscribedFiberUpdateWithOrigin(parseFiber, parseUpdateOrigin)
		}
	}
	if parseRt.ShouldDeferStateUpdates() {
		parseRt.ScheduleTransition(func() {
			apply("transition")
		})
		return nil
	}

	apply("atom")
	return nil
}
