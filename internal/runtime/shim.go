//go:build js && wasm
// +build js,wasm

package runtime

// Global wrapper functions that provide simplified API for public packages
// These match the old fiber package API

// GoUseStateGlobal wraps GoUseState with global fiber context
func GoUseStateGlobal[T any](initialValue T) (func() T, func(interface{})) {
	rt := GetGlobalRuntime()
	return GoUseState(rt, initialValue)
}

// GoUseEffectGlobal wraps GoUseEffect
func GoUseEffectGlobal(effect func() func(), deps ...interface{}) {
	// GoUseEffect doesn't need Runtime, it works with current fiber
	GoUseEffect(effect, deps...)
}

// GoUseMemoGlobal wraps GoUseMemo
func GoUseMemoGlobal(compute func() interface{}, deps ...interface{}) interface{} {
	// GoUseMemo doesn't need Runtime, it works with current fiber
	return GoUseMemo(compute, deps...)
}

// GoUseCallbackGlobal wraps GoUseCallback
func GoUseCallbackGlobal(fn interface{}, deps ...interface{}) interface{} {
	// GoUseCallback doesn't need Runtime, it works with current fiber
	return GoUseCallback(fn, deps...)
}

// GoUseRefGlobal wraps GoUseRef
func GoUseRefGlobal(initialValue interface{}) *RefValue {
	// GoUseRef doesn't need Runtime, it works with current fiber
	return GoUseRef(initialValue)
}

// GoUseIdGlobal wraps GoUseId
func GoUseIdGlobal() string {
	// GoUseId doesn't need Runtime, it works with current fiber
	return GoUseId()
}

// GoUseFetchGlobal wraps GoUseFetch with global fiber context
func GoUseFetchGlobal(url string, options ...interface{}) (func() FetchState, func()) {
	// GoUseFetch doesn't need Runtime, it works with current fiber
	return GoUseFetch(url, options...)
}

// GoUseFuncGlobal wraps GoUseFunc with WASM event handler wrapping
func GoUseFuncGlobal(fn interface{}) interface{} {
	// The core GoUseFunc now handles wrapping via DOMAdapter
	return GoUseFunc(fn)
}

// GoUseAtomGlobal wraps GoUseAtom with global runtime
func GoUseAtomGlobal[T any](id string, initialValue T) (func() T, func(T)) {
	rt := GetGlobalRuntime()
	get, set := GoUseAtom(rt, id, initialValue)
	// Convert internal setter func(interface{}) to typed setter func(T)
	typedSet := func(v T) {
		set(v)
	}
	return get, typedSet
}

// GoUseTransitionPendingGlobal exposes the shared transition pending flag as a subscribed atom.
func GoUseTransitionPendingGlobal() (func() bool, func(bool)) {
	return GoUseAtomGlobal(transitionPendingAtomID, false)
}

// StartTransitionGlobal runs fn in a non-urgent transition context.
func StartTransitionGlobal(fn func()) {
	GetGlobalRuntime().StartTransition(fn)
}

// Text creates a text node
func Text(content string) *Element {
	return &Element{
		Type:        "TEXT_ELEMENT",
		TextContent: content,
		Children:    emptyChildren,
	}
}
