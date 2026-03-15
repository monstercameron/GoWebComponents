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
	subscriberPool sync.Pool
}

// NewAtomRegistry creates a new atom registry
func NewAtomRegistry() *AtomRegistry {
	registry := &AtomRegistry{
		atoms:         make(map[string]interface{}),
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

	ar.mu.Lock()
	ar.atoms[id] = value

	subs, ok := ar.subscriptions[id]
	if !ok || len(subs) == 0 {
		ar.mu.Unlock()
		return
	}
	if len(subs) == 1 {
		var onlyFiber *Fiber
		for fiber := range subs {
			onlyFiber = fiber
			break
		}
		ar.mu.Unlock()
		notify(onlyFiber)
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
			currentValue := get()

			var newValue T
			if fn, ok := newValueOrUpdater.(func(T) T); ok {
				newValue = fn(currentValue)
			} else if directValue, ok := newValueOrUpdater.(T); ok {
				newValue = directValue
			} else if newValueOrUpdater == nil && nilableState {
				var zero T
				newValue = zero
			} else {
				return
			}

			if fastEqual(currentValue, newValue) {
				return
			}

			rt.atomRegistry.setAtomAndNotify(id, newValue, rt.ScheduleUpdateForFiber)
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
