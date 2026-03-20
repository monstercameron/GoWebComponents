package runtime

import "sync/atomic"

var nextContextID int64

// ContextDescriptor identifies a context value stream within a component tree.
type ContextDescriptor struct {
	ID           int64
	DefaultValue interface{}
}

// ContextProviderType marks provider elements in the runtime tree.
type ContextProviderType struct {
	Descriptor *ContextDescriptor
}

// NewContextDescriptor creates a new context descriptor with a unique runtime ID.
func NewContextDescriptor(defaultValue interface{}) *ContextDescriptor {
	return &ContextDescriptor{
		ID:           atomic.AddInt64(&nextContextID, 1),
		DefaultValue: defaultValue,
	}
}

// NewContextProviderType creates a provider marker for the given descriptor.
func NewContextProviderType(descriptor *ContextDescriptor) *ContextProviderType {
	return &ContextProviderType{Descriptor: descriptor}
}

// GoUseContextValue reads the nearest provider value for a context descriptor.
func GoUseContextValue(descriptor *ContextDescriptor) interface{} {
	if descriptor == nil {
		panic("GoUseContextValue called with nil context descriptor")
	}

	fiber := GetCurrentFiber()
	if fiber == nil {
		ReportDiagnostic("runtime", DiagnosticError, "GoUseContextValue called outside component context")
		panic("GoUseContextValue called outside component context")
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{owner: fiber}
	} else if fiber.hooks.owner == nil {
		fiber.hooks.owner = fiber
	}

	recordHookSignature(fiber.hooks, "context")
	fiber.hooks.index++
	return resolveContextValue(fiber, descriptor)
}

func resolveContextValue(fiber *Fiber, descriptor *ContextDescriptor) interface{} {
	if descriptor == nil {
		return nil
	}

	if fiber != nil && fiber.contextValues != nil {
		if value, ok := fiber.contextValues[descriptor.ID]; ok {
			return value
		}
	}

	return descriptor.DefaultValue
}

func deriveContextValues(parentValues map[int64]interface{}, contextID int64, value interface{}) map[int64]interface{} {
	derived := make(map[int64]interface{}, len(parentValues)+1)
	for key, existingValue := range parentValues {
		derived[key] = existingValue
	}
	derived[contextID] = value
	return derived
}

func markSubtreeNeedsUpdate(fiber *Fiber) {
	for current := fiber; current != nil; current = current.sibling {
		current.needsUpdate = true
		markSubtreeNeedsUpdate(current.child)
	}
}
