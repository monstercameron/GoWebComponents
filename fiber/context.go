//go:build js && wasm
// +build js,wasm

package fiber

import "sync/atomic"

var nextContextID int64

// Context stores a value that can be shared across a component subtree.
type Context struct {
	id           int64
	defaultValue interface{}
	Provider     *ContextProvider
	Consumer     *ContextConsumer
}

// ContextProvider identifies provider elements in the fiber tree.
type ContextProvider struct {
	context *Context
}

// ContextConsumer identifies consumer elements in the fiber tree.
type ContextConsumer struct {
	context *Context
}

// CreateContext creates a new Context with a default value.
func CreateContext[T any](defaultValue T) *Context {
	context := &Context{
		id:           atomic.AddInt64(&nextContextID, 1),
		defaultValue: defaultValue,
	}

	context.Provider = &ContextProvider{context: context}
	context.Consumer = &ContextConsumer{context: context}

	return context
}

// GoUseContext reads the nearest provider value for a Context.
func GoUseContext[T any](context *Context) T {
	if context == nil {
		panic("GoUseContext called with nil context")
	}

	debugf("HOOKS", "🎯 GoUseContext called for context %d\n", context.id)

	currentFiber := getCurrentFiber()
	if currentFiber == nil {
		debugf("HOOKS", "🚨 GoUseContext: currentFiber is nil!\n")
		panic("GoUseContext called outside component context")
	}

	if currentFiber.hooks == nil {
		debugf("HOOKS", "🔧 GoUseContext: initializing hooks container\n")
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	if err := validateHookOrder(currentFiber.hooks, HookTypeContext, position); err != nil {
		debugf("HOOKS", "🚨 GoUseContext: %v\n", err)
	}

	value := resolveContextValue(currentFiber, context)
	if typedValue, ok := value.(T); ok {
		return typedValue
	}

	var zero T
	debugf("HOOKS", "🚨 GoUseContext: type assertion failed for context %d, expected %T, got %T\n",
		context.id, zero, value)
	return zero
}

func resolveContextValue(fiber *Fiber, context *Context) interface{} {
	if context == nil {
		return nil
	}

	if fiber != nil && fiber.contextValues != nil {
		if value, ok := fiber.contextValues[context.id]; ok {
			return value
		}
	}

	return context.defaultValue
}

func deriveContextValues(parentValues map[int64]interface{}, contextID int64, value interface{}) map[int64]interface{} {
	derived := make(map[int64]interface{}, len(parentValues)+1)
	for key, existingValue := range parentValues {
		derived[key] = existingValue
	}
	derived[contextID] = value
	return derived
}
