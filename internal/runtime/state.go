package runtime

import (
	"fmt"
	"sync"
)

// AtomRegistry manages global state atoms with fine-grained reactivity.
// Each atom has a unique ID and tracks which fibers are subscribed to it.
type AtomRegistry struct {
	mu            sync.RWMutex
	atoms         map[string]*atomState
	subscriptions map[string]map[*Fiber]bool // atomID -> set of subscribed fibers
}

// atomState holds the actual value for an atom
type atomState struct {
	value interface{}
}

// NewAtomRegistry creates a new atom registry
func NewAtomRegistry() *AtomRegistry {
	return &AtomRegistry{
		atoms:         make(map[string]*atomState),
		subscriptions: make(map[string]map[*Fiber]bool),
	}
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
	subscribers := make([]*Fiber, 0)
	if subs, ok := ar.subscriptions[id]; ok {
		for fiber := range subs {
			subscribers = append(subscribers, fiber)
		}
	}

	return subscribers
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

	// Validate hook order
	if err := validateHookOrder(fiber.hooks, HookTypeAtom, fiber.hooks.index); err != nil {
		panic(err)
	}
	fiber.hooks.index++

	// Initialize atom if it doesn't exist
	rt.atomRegistry.InitAtom(id, initialValue)

	// Subscribe this fiber to the atom
	rt.atomRegistry.Subscribe(id, fiber)

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
		} else {
			return
		}

		// Skip update if value hasn't changed
		if fastEqual(currentValue, newValue) {
			return
		}

		// Update atom and get subscribers
		subscribers := rt.atomRegistry.SetAtom(id, newValue)

		// Schedule updates for all subscribed fibers
		// TODO: dedupe subscribers and skip nil/dead fibers before scheduling updates
		for _, subFiber := range subscribers {
			rt.ScheduleUpdateForFiber(subFiber)
		}
	}

	return get, set
}

// atomHookState stores the atom ID for cleanup
type atomHookState struct {
	atomID string
}

// CleanupAtomSubscriptions removes all atom subscriptions for a fiber
// This should be called when a fiber is being removed from the tree
func (rt *Runtime) CleanupAtomSubscriptions(fiber *Fiber) {
	if rt.atomRegistry == nil {
		return
	}

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

	subscribers := rt.atomRegistry.SetAtom(id, value)

	// Schedule updates for all subscribers
	for _, fiber := range subscribers {
		rt.ScheduleUpdateForFiber(fiber)
	}

	return nil
}
