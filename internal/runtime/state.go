package runtime

import (
	"fmt"
	"sync"
)

// AtomRegistry manages global state atoms with fine-grained reactivity.
// Each atom has a unique ID and tracks which fibers are subscribed to it.
type AtomRegistry struct {
	mu             sync.RWMutex
	atoms          map[string]interface{}
	subscriptions  map[string]map[*Fiber]bool // atomID -> set of subscribed fibers
	derived        map[string]derivedAtom
	dependents     map[string]map[string]bool // source atom id -> derived ids
	subscriberPool sync.Pool
}

type derivedAtom struct {
	deps    []string
	compute func() interface{}
	active  bool
}

// NewAtomRegistry creates a new atom registry
func NewAtomRegistry() *AtomRegistry {
	registry := &AtomRegistry{
		atoms:         make(map[string]interface{}),
		subscriptions: make(map[string]map[*Fiber]bool),
		derived:       make(map[string]derivedAtom),
		dependents:    make(map[string]map[string]bool),
	}
	registry.subscriberPool.New = func() interface{} {
		return make([]*Fiber, 0, 16)
	}
	return registry
}

func (ar *AtomRegistry) RegisterDerivedAtom(id string, deps []string, compute func() interface{}) error {
	if ar == nil {
		return fmt.Errorf("atom registry not initialized")
	}
	if compute == nil {
		return fmt.Errorf("derived atom %s compute function cannot be nil", id)
	}
	for _, dep := range deps {
		if dep == id {
			return fmt.Errorf("derived atom %s cannot depend on itself", id)
		}
	}

	ar.mu.Lock()
	if existing, ok := ar.derived[id]; ok {
		for _, dep := range existing.deps {
			if dependents := ar.dependents[dep]; dependents != nil {
				delete(dependents, id)
				if len(dependents) == 0 {
					delete(ar.dependents, dep)
				}
			}
		}
	}
	cloneDeps := append([]string(nil), deps...)
	ar.derived[id] = derivedAtom{deps: cloneDeps, compute: compute, active: true}
	for _, dep := range cloneDeps {
		if ar.dependents[dep] == nil {
			ar.dependents[dep] = make(map[string]bool)
		}
		ar.dependents[dep][id] = true
	}
	ar.mu.Unlock()

	_, err := ar.recomputeDerived(id, nil)
	return err
}

// GetAtom retrieves an atom's current value
func (ar *AtomRegistry) GetAtom(id string) (interface{}, bool) {
	ar.mu.RLock()
	atom, ok := ar.atoms[id]
	ar.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return atom, true
}

// SetAtom updates an atom's value and returns subscribed fibers
func (ar *AtomRegistry) SetAtom(id string, value interface{}) []*Fiber {
	ar.mu.Lock()

	// Update or create atom
	ar.atoms[id] = value

	// Get all subscribed fibers
	if subs, ok := ar.subscriptions[id]; ok {
		subscribers := make([]*Fiber, 0, len(subs))
		for fiber := range subs {
			subscribers = append(subscribers, fiber)
		}
		ar.mu.Unlock()
		return subscribers
	}
	ar.mu.Unlock()

	return nil
}

func (ar *AtomRegistry) setAtomAndNotify(id string, value interface{}, notify func(*Fiber)) {
	if notify == nil {
		_ = ar.SetAtom(id, value)
		return
	}

	fibers := ar.setValueAndCollectSubscribers(id, value)
	for _, derivedID := range ar.listDependents(id) {
		derivedFibers, err := ar.recomputeDerived(derivedID, map[string]bool{id: true})
		if err != nil {
			ReportDiagnostic("state", DiagnosticWarning, err.Error())
			continue
		}
		fibers = append(fibers, derivedFibers...)
	}
	notifyFibersUnique(fibers, notify)
}

func (ar *AtomRegistry) setValueAndCollectSubscribers(id string, value interface{}) []*Fiber {
	ar.mu.Lock()
	ar.atoms[id] = value
	fibers := ar.collectSubscribersLocked(id)
	ar.mu.Unlock()
	return fibers
}

func (ar *AtomRegistry) collectSubscribersLocked(id string) []*Fiber {
	subs, ok := ar.subscriptions[id]
	if !ok || len(subs) == 0 {
		return nil
	}
	fibers := make([]*Fiber, 0, len(subs))
	for fiber := range subs {
		fibers = append(fibers, fiber)
	}
	return fibers
}

func (ar *AtomRegistry) listDependents(id string) []string {
	ar.mu.RLock()
	dependents := ar.dependents[id]
	if len(dependents) == 0 {
		ar.mu.RUnlock()
		return nil
	}
	ids := make([]string, 0, len(dependents))
	for derivedID := range dependents {
		ids = append(ids, derivedID)
	}
	ar.mu.RUnlock()
	return ids
}

func (ar *AtomRegistry) recomputeDerived(id string, trail map[string]bool) ([]*Fiber, error) {
	if trail == nil {
		trail = map[string]bool{}
	}
	if trail[id] {
		return nil, fmt.Errorf("derived atom cycle detected involving %s", id)
	}
	trail[id] = true
	defer delete(trail, id)

	ar.mu.RLock()
	derived, ok := ar.derived[id]
	ar.mu.RUnlock()
	if !ok || !derived.active {
		return nil, nil
	}

	value := derived.compute()
	fibers := ar.setValueAndCollectSubscribers(id, value)
	for _, dependentID := range ar.listDependents(id) {
		nested, err := ar.recomputeDerived(dependentID, trail)
		if err != nil {
			return fibers, err
		}
		fibers = append(fibers, nested...)
	}
	return fibers, nil
}

func notifyFibersUnique(fibers []*Fiber, notify func(*Fiber)) {
	if notify == nil || len(fibers) == 0 {
		return
	}
	seen := make(map[*Fiber]bool, len(fibers))
	for _, fiber := range fibers {
		if fiber == nil || seen[fiber] {
			continue
		}
		seen[fiber] = true
		notify(fiber)
	}
}

// InitAtom initializes an atom if it doesn't exist
func (ar *AtomRegistry) InitAtom(id string, initialValue interface{}) {
	ar.mu.Lock()
	if _, exists := ar.atoms[id]; !exists {
		ar.atoms[id] = initialValue
	}
	ar.mu.Unlock()
}

// Subscribe adds a fiber to an atom's subscription list
func (ar *AtomRegistry) Subscribe(atomID string, fiber *Fiber) {
	ar.mu.Lock()

	if ar.subscriptions[atomID] == nil {
		ar.subscriptions[atomID] = make(map[*Fiber]bool)
	}
	ar.subscriptions[atomID][fiber] = true
	ar.mu.Unlock()
}

// Unsubscribe removes a fiber from an atom's subscription list
func (ar *AtomRegistry) Unsubscribe(atomID string, fiber *Fiber) {
	ar.mu.Lock()

	if subs, ok := ar.subscriptions[atomID]; ok {
		delete(subs, fiber)
		if len(subs) == 0 {
			delete(ar.subscriptions, atomID)
		}
	}
	ar.mu.Unlock()
}

// UnsubscribeMany removes a fiber from several atom subscriptions under one lock.
func (ar *AtomRegistry) UnsubscribeMany(atomIDs []string, fiber *Fiber) {
	ar.mu.Lock()

	for _, atomID := range atomIDs {
		if subs, ok := ar.subscriptions[atomID]; ok {
			delete(subs, fiber)
			if len(subs) == 0 {
				delete(ar.subscriptions, atomID)
			}
		}
	}
	ar.mu.Unlock()
}

// UnsubscribeFiberFromAll removes a fiber from all atom subscriptions
func (ar *AtomRegistry) UnsubscribeFiberFromAll(fiber *Fiber) {
	ar.mu.Lock()

	for atomID, subs := range ar.subscriptions {
		delete(subs, fiber)
		if len(subs) == 0 {
			delete(ar.subscriptions, atomID)
		}
	}
	ar.mu.Unlock()
}

// GetSubscriberCount returns the number of fibers subscribed to an atom
func (ar *AtomRegistry) GetSubscriberCount(atomID string) int {
	ar.mu.RLock()
	if subs, ok := ar.subscriptions[atomID]; ok {
		count := len(subs)
		ar.mu.RUnlock()
		return count
	}
	ar.mu.RUnlock()
	return 0
}

// GetAtomCount returns the total number of atoms
func (ar *AtomRegistry) GetAtomCount() int {
	ar.mu.RLock()
	count := len(ar.atoms)
	ar.mu.RUnlock()
	return count
}

// Snapshot returns a shallow copy of all atom values currently stored.
func (ar *AtomRegistry) Snapshot() map[string]interface{} {
	if ar == nil {
		return nil
	}

	ar.mu.RLock()
	defer ar.mu.RUnlock()
	if len(ar.atoms) == 0 {
		return map[string]interface{}{}
	}

	snapshot := make(map[string]interface{}, len(ar.atoms))
	for id, value := range ar.atoms {
		snapshot[id] = value
	}
	return snapshot
}

// RestoreSnapshot merges atom values from snapshot and returns subscribed fibers
// that should be notified about the updates.
func (ar *AtomRegistry) RestoreSnapshot(snapshot map[string]interface{}) []*Fiber {
	if ar == nil || len(snapshot) == 0 {
		return nil
	}

	unique := make(map[*Fiber]bool)
	ar.mu.Lock()
	for id, value := range snapshot {
		ar.atoms[id] = value
		if subs, ok := ar.subscriptions[id]; ok {
			for fiber := range subs {
				unique[fiber] = true
			}
		}
	}
	ar.mu.Unlock()

	if len(unique) == 0 {
		return nil
	}

	fibers := make([]*Fiber, 0, len(unique))
	for fiber := range unique {
		fibers = append(fibers, fiber)
	}
	return fibers
}

// GoUseAtom provides access to global state with fine-grained reactivity.
// Unlike useState which is local to a component, atoms are shared across components.
// When an atom updates, only components that use that specific atom re-render.
func GoUseAtom[T any](rt *Runtime, id string, initialValue T) (func() T, func(interface{})) {
	if rt.atomRegistry == nil {
		panic("Runtime atom registry not initialized")
	}

	fiber := GetCurrentFiber()
	if fiber == nil {
		panic("GoUseAtom must be called within a component")
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{owner: fiber}
	} else if fiber.hooks.owner == nil {
		fiber.hooks.owner = fiber
	}

	fiber.hooks.index++

	atomIdx := fiber.hooks.atomIndex
	fiber.hooks.atomIndex++
	hooks := fiber.hooks
	trackedAtomID := ""
	hasTrackedAtom := len(hooks.atoms) > atomIdx
	if hasTrackedAtom {
		trackedAtomID = hooks.atoms[atomIdx]
	}
	hasAccessor := len(hooks.atomFuncs) > atomIdx

	// Stable rerenders of the same atom do not need to re-initialize the registry entry.
	if !hasTrackedAtom || trackedAtomID != id {
		rt.atomRegistry.InitAtom(id, initialValue)
	}

	// Track subscription in fiber for efficient cleanup
	if !hasTrackedAtom {
		needed := atomIdx + 1
		if needed <= cap(hooks.atoms) {
			hooks.atoms = hooks.atoms[:needed]
		} else {
			newAtoms := make([]string, needed, needed*2)
			copy(newAtoms, hooks.atoms)
			hooks.atoms = newAtoms
		}
		hooks.atoms[atomIdx] = id
		rt.atomRegistry.Subscribe(id, fiber)
	} else if trackedAtomID != id {
		if trackedAtomID != "" {
			rt.atomRegistry.Unsubscribe(trackedAtomID, fiber)
		}
		hooks.atoms[atomIdx] = id
		rt.atomRegistry.Subscribe(id, fiber)
	}

	nilableState := isNilableType[T]()
	accessorNeedsRefresh := !hasAccessor || trackedAtomID != id
	if accessorNeedsRefresh {
		needed := atomIdx + 1
		if needed <= cap(hooks.atomFuncs) {
			hooks.atomFuncs = hooks.atomFuncs[:needed]
		} else {
			newAccessors := make([]atomAccessorValue, needed, needed*2)
			copy(newAccessors, hooks.atomFuncs)
			hooks.atomFuncs = newAccessors
		}

		get := func() T {
			value, ok := rt.atomRegistry.GetAtom(id)
			if !ok {
				return initialValue
			}

			if typed, ok := value.(T); ok {
				return typed
			}

			return initialValue
		}

		set := func(newValueOrUpdater interface{}) {
			apply := func() {
				currentValue := get()
				newValue, ok := resolveStateUpdateValue(currentValue, newValueOrUpdater, nilableState)
				if !ok {
					return
				}

				if fastEqual(currentValue, newValue) {
					return
				}

				rt.atomRegistry.setAtomAndNotify(id, newValue, rt.ScheduleUpdateForFiber)
			}

			if rt.ShouldDeferStateUpdates() {
				rt.ScheduleTransition(apply)
				return
			}

			apply()
		}

		hooks.atomFuncs[atomIdx] = atomAccessorValue{getter: get, setter: set}
	}

	get, _ := hooks.atomFuncs[atomIdx].getter.(func() T)
	set, _ := hooks.atomFuncs[atomIdx].setter.(func(interface{}))
	if get == nil || set == nil {
		panic("GoUseAtom accessor cache type mismatch")
	}

	return get, set
}

// CleanupAtomSubscriptions removes all atom subscriptions for a fiber
// This should be called when a fiber is being removed from the tree
func (rt *Runtime) CleanupAtomSubscriptions(fiber *Fiber) {
	if rt.atomRegistry == nil {
		return
	}

	// Optimization: Only unsubscribe from atoms this fiber is actually using
	if fiber.hooks != nil && len(fiber.hooks.atoms) > 0 {
		rt.atomRegistry.UnsubscribeMany(fiber.hooks.atoms, fiber)
		return
	}

	// Fallback for fibers without hooks or if atoms list is empty (shouldn't happen if using GoUseAtom)
	rt.atomRegistry.UnsubscribeFiberFromAll(fiber)
}

// GetAtomValue is a helper to get an atom value directly (for debugging/testing)
func (rt *Runtime) GetAtomValue(id string) (interface{}, bool) {
	if rt.atomRegistry == nil {
		return nil, false
	}
	return rt.atomRegistry.GetAtom(id)
}

// SetAtomValue is a helper to set an atom value directly (for debugging/testing)
func (rt *Runtime) SetAtomValue(id string, value interface{}) error {
	if rt.atomRegistry == nil {
		return fmt.Errorf("atom registry not initialized")
	}

	rt.atomRegistry.setAtomAndNotify(id, value, rt.ScheduleUpdateForFiber)

	return nil
}

func (rt *Runtime) RegisterDerivedAtom(id string, deps []string, compute func() interface{}) error {
	if rt == nil || rt.atomRegistry == nil {
		return fmt.Errorf("atom registry not initialized")
	}
	return rt.atomRegistry.RegisterDerivedAtom(id, deps, compute)
}

// SnapshotAtoms returns a copy of all currently registered atoms.
func (rt *Runtime) SnapshotAtoms() map[string]interface{} {
	if rt == nil || rt.atomRegistry == nil {
		return map[string]interface{}{}
	}
	return rt.atomRegistry.Snapshot()
}

// RestoreAtomSnapshot merges atom values from snapshot and schedules updates for
// any subscribed fibers.
func (rt *Runtime) RestoreAtomSnapshot(snapshot map[string]interface{}) error {
	if rt == nil || rt.atomRegistry == nil {
		return fmt.Errorf("atom registry not initialized")
	}

	for _, fiber := range rt.atomRegistry.RestoreSnapshot(snapshot) {
		rt.ScheduleUpdateForFiber(fiber)
	}
	return nil
}
