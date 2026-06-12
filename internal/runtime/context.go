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
func NewContextDescriptor(parseDefaultValue interface{}) *ContextDescriptor {
	return &ContextDescriptor{
		ID:           atomic.AddInt64(&nextContextID, 1),
		DefaultValue: parseDefaultValue,
	}
}

// NewContextProviderType creates a provider marker for the given descriptor.
func NewContextProviderType(parseDescriptor *ContextDescriptor) *ContextProviderType {
	return &ContextProviderType{Descriptor: parseDescriptor}
}

// GoUseContextValue reads the nearest provider value for a context descriptor.
func GoUseContextValue(parseDescriptor *ContextDescriptor) interface{} {
	if parseDescriptor == nil {
		panic(actionableContextDescriptorNilPanic("GoUseContextValue"))
	}

	parseFiber := requireCurrentHookFiber("GoUseContextValue")

	if parseFiber.hooks == nil {
		parseFiber.hooks = &Hooks{owner: parseFiber}
	} else if parseFiber.hooks.owner == nil {
		parseFiber.hooks.owner = parseFiber
	}

	recordHookSignature(parseFiber.hooks, "context")
	parseFiber.hooks.index++
	return resolveContextValue(parseFiber, parseDescriptor)
}

// resolveContextValue is a core package helper.
func resolveContextValue(parseFiber *Fiber, parseDescriptor *ContextDescriptor) interface{} {
	if parseDescriptor == nil {
		return nil
	}

	if parseFiber != nil && parseFiber.contextValues != nil {
		if parseValue, parseOk := parseFiber.contextValues[parseDescriptor.ID]; parseOk {
			return parseValue
		}
	}

	return parseDescriptor.DefaultValue
}

// deriveContextValues is a core package helper.
func deriveContextValues(parseParentValues map[int64]interface{}, parseContextID int64, parseValue interface{}) map[int64]interface{} {
	parseDerived := make(map[int64]interface{}, len(parseParentValues)+1)
	for parseKey, parseExistingValue := range parseParentValues {
		parseDerived[parseKey] = parseExistingValue
	}
	parseDerived[parseContextID] = parseValue
	return parseDerived
}

// markSubtreeNeedsUpdate is a core package helper.
func markSubtreeNeedsUpdate(parseFiber *Fiber, parseOrigin string) {
	for parseCurrent := parseFiber; parseCurrent != nil; parseCurrent = parseCurrent.sibling {
		parseCurrent.needsUpdate = true
		if parseCurrent.updateOrigin == "" {
			parseCurrent.updateOrigin = parseOrigin
		}
		markSubtreeNeedsUpdate(parseCurrent.child, parseOrigin)
	}
}
