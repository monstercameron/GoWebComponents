package runtime

import (
	"fmt"
	"sync"
)

// AtomRegistry manages global state atoms with fine-grained reactivity.
// Each atom has a unique ID and tracks which fibers are subscribed to it.
type AtomRegistry struct {
	mu             sync.RWMutex
	atoms          map[string]*atomState
	subscriptions  map[string]map[*Fiber]bool // atomID -> set of subscribed fibers
	subscriberPool sync.Pool
}

// atomState holds the actual value for an atom
type atomState struct {
	value interface{}
}

// NewAtomRegistry creates a new atom registry
func NewAtomRegistry() *AtomRegistry {
	registry := &AtomRegistry{
		atoms:         make(map[string]*atomState),
		subscriptions: make(map[string]map[*Fiber]bool),
	}
	registry.subscriberPool.New = func() interface{} {
		return make([]*Fiber, 0, 16)
	}
	return registry
}

// GetAtom retrieves an atom's current value
func (ar *AtomRegistry) GetAtom(id string) (interface{}, bool) {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	atom, ok := ar.atoms[id]
	if !ok {
		return nil, false
	}
	return atom.value, true
}

// SetAtom updates an atom's value and returns subscribed fibers
func (ar *AtomRegistry) SetAtom(id string, value interface{}) []*Fiber {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	// Update or create atom
	if _, exists := ar.atoms[id]; !exists {
		ar.atoms[id] = &atomState{}
	}
	ar.atoms[id].value = value

	// Get all subscribed fibers
	if subs, ok := ar.subscriptions[id]; ok {
		subscribers := make([]*Fiber, 0, len(subs))
		for fiber := range subs {
			subscribers = append(subscribers, fiber)
		}
		return subscribers
	}

	return nil
}

func (ar *AtomRegistry) setAtomAndNotify(id string, value interface{}, notify func(*Fiber)) {
	if notify == nil {
		_ = ar.SetAtom(id, value)
		return
	}

	ar.mu.Lock()
	if _, exists := ar.atoms[id]; !exists {
		ar.atoms[id] = &atomState{}
	}
	ar.atoms[id].value = value

	subs, ok := ar.subscriptions[id]
	if !ok || len(subs) == 0 {
		ar.mu.Unlock()
		return
	}

	buffer := ar.subscriberPool.Get().([]*Fiber)
	buffer = buffer[:0]
	if cap(buffer) < len(subs) {
		buffer = make([]*Fiber, 0, len(subs))
	}
	for fiber := range subs {
		buffer = append(buffer, fiber)
	}
	ar.mu.Unlock()

	for _, fiber := range buffer {
		notify(fiber)
	}

	clear(buffer)
	ar.subscriberPool.Put(buffer[:0])
}

// InitAtom initializes an atom if it doesn't exist
func (ar *AtomRegistry) InitAtom(id string, initialValue interface{}) {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	if _, exists := ar.atoms[id]; !exists {
		ar.atoms[id] = &atomState{value: initialValue}
	}
}

// Subscribe adds a fiber to an atom's subscription list
func (ar *AtomRegistry) Subscribe(atomID string, fiber *Fiber) {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	if ar.subscriptions[atomID] == nil {
		ar.subscriptions[atomID] = make(map[*Fiber]bool)
	}
	ar.subscriptions[atomID][fiber] = true
}

// Unsubscribe removes a fiber from an atom's subscription list
func (ar *AtomRegistry) Unsubscribe(atomID string, fiber *Fiber) {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	if subs, ok := ar.subscriptions[atomID]; ok {
		delete(subs, fiber)
		if len(subs) == 0 {
			delete(ar.subscriptions, atomID)
		}
	}
}

// UnsubscribeMany removes a fiber from several atom subscriptions under one lock.
func (ar *AtomRegistry) UnsubscribeMany(atomIDs []string, fiber *Fiber) {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	for _, atomID := range atomIDs {
		if subs, ok := ar.subscriptions[atomID]; ok {
			delete(subs, fiber)
			if len(subs) == 0 {
				delete(ar.subscriptions, atomID)
			}
		}
	}
}

// UnsubscribeFiberFromAll removes a fiber from all atom subscriptions
func (ar *AtomRegistry) UnsubscribeFiberFromAll(fiber *Fiber) {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	for atomID, subs := range ar.subscriptions {
		delete(subs, fiber)
		if len(subs) == 0 {
			delete(ar.subscriptions, atomID)
		}
	}
}

// GetSubscriberCount returns the number of fibers subscribed to an atom
func (ar *AtomRegistry) GetSubscriberCount(atomID string) int {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	if subs, ok := ar.subscriptions[atomID]; ok {
		return len(subs)
	}
	return 0
}

// GetAtomCount returns the total number of atoms
func (ar *AtomRegistry) GetAtomCount() int {
	ar.mu.RLock()
	defer ar.mu.RUnlock()
	return len(ar.atoms)
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

	// Initialize atom if it doesn't exist
	rt.atomRegistry.InitAtom(id, initialValue)

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

	// Getter function
	get := func() T {
		value, ok := rt.atomRegistry.GetAtom(id)
		if !ok {
			return initialValue
		}

		// Type assertion
		if typed, ok := value.(T); ok {
			return typed
		}

		// Fallback to initial value if type mismatch
		return initialValue
	}

	nilableState := isNilableType[T]()

	// Setter function
	set := func(newValueOrUpdater interface{}) {
		// Get current value
		currentValue := get()

		// Determine the new value
		var newValue T
		// Try to treat as functional update (func(T) T)
		if fn, ok := newValueOrUpdater.(func(T) T); ok {
			newValue = fn(currentValue)
		} else if directValue, ok := newValueOrUpdater.(T); ok {
			// Direct value
			newValue = directValue
		} else if newValueOrUpdater == nil && nilableState {
			var zero T
			newValue = zero
		} else {
			return
		}

		// Skip update if value hasn't changed
		if fastEqual(currentValue, newValue) {
			return
		}

		// Update atom and schedule all subscribed fibers without per-update slice churn.
		rt.atomRegistry.setAtomAndNotify(id, newValue, rt.ScheduleUpdateForFiber)
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
