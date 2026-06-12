//go:build js && wasm

package runtime

import "reflect"

// Global wrapper functions that provide simplified API for public packages
// These match the old fiber package API

// GoUseStateGlobal wraps GoUseState with global fiber context
func GoUseStateGlobal[T any](parseStateInitialValue T) (func() T, func(interface{})) {
	parseRuntime := GetGlobalRuntime()
	return GoUseState(parseRuntime, parseStateInitialValue)
}

// GoUseEffectGlobal wraps GoUseEffect
func GoUseEffectGlobal(parseEffectFn func() func(), parseEffectDeps ...interface{}) {
	// GoUseEffect doesn't need Runtime, it works with current fiber
	GoUseEffect(parseEffectFn, parseEffectDeps...)
}

// GoUseMemoGlobal wraps GoUseMemo
func GoUseMemoGlobal(parseMemoCompute func() interface{}, parseMemoDeps ...interface{}) interface{} {
	// GoUseMemo doesn't need Runtime, it works with current fiber
	return GoUseMemo(parseMemoCompute, parseMemoDeps...)
}

// GoUseMemoGlobalTyped wraps GoUseMemo with an expected target type for
// hot-reload restoration.
func GoUseMemoGlobalTyped(parseMemoCompute func() interface{}, parseMemoTargetType reflect.Type, parseMemoDeps ...interface{}) interface{} {
	return GoUseMemoTyped(parseMemoCompute, parseMemoTargetType, parseMemoDeps...)
}

// GoUseCallbackGlobal wraps GoUseCallback
func GoUseCallbackGlobal(parseCallbackFn interface{}, parseCallbackDeps ...interface{}) interface{} {
	// GoUseCallback doesn't need Runtime, it works with current fiber
	return GoUseCallback(parseCallbackFn, parseCallbackDeps...)
}

// GoUseRefGlobal wraps GoUseRef
func GoUseRefGlobal(parseRefInitialValue interface{}) *RefValue {
	// GoUseRef doesn't need Runtime, it works with current fiber
	return GoUseRef(parseRefInitialValue)
}

// GoUseIdGlobal wraps GoUseId
func GoUseIdGlobal() string {
	// GoUseId doesn't need Runtime, it works with current fiber
	return GoUseId()
}

// GoUseFetchGlobal wraps GoUseFetch with global fiber context
func GoUseFetchGlobal(parseFetchURL string, parseFetchOptions ...interface{}) (func() FetchState, func()) {
	// GoUseFetch doesn't need Runtime, it works with current fiber
	return GoUseFetch(parseFetchURL, parseFetchOptions...)
}

// GoUseFuncGlobal wraps GoUseFunc with WASM event handler wrapping
func GoUseFuncGlobal(parseHandlerFn interface{}) interface{} {
	// The core GoUseFunc now handles wrapping via DOMAdapter
	return GoUseFunc(parseHandlerFn)
}

// BuildDOMWrappedFunctionGlobal wraps one plain Go callback with the active DOM adapter without requiring a hook call.
func BuildDOMWrappedFunctionGlobal(parseHandlerFn interface{}) interface{} {
	if parseHandlerFn == nil {
		return nil
	}
	parseRuntime := GetGlobalRuntime()
	if parseRuntime.domAdapter == nil {
		panic(actionableRuntimeDOMAdapterPanic("BuildDOMWrappedFunctionGlobal"))
	}
	return parseRuntime.domAdapter.WrapFunction(parseHandlerFn)
}

// BuildDOMWrappedFunctionIfReadyGlobal wraps one plain Go callback when the global runtime already has a DOM adapter.
func BuildDOMWrappedFunctionIfReadyGlobal(parseHandlerFn interface{}) (interface{}, bool) {
	if parseHandlerFn == nil {
		return nil, false
	}
	parseRuntime := GetGlobalRuntime()
	if parseRuntime.domAdapter == nil {
		return nil, false
	}
	return parseRuntime.domAdapter.WrapFunction(parseHandlerFn), true
}

// GoUseAtomGlobal wraps GoUseAtom with global runtime
func GoUseAtomGlobal[T any](parseAtomID string, parseAtomInitialValue T) (func() T, func(T)) {
	parseRuntime := GetGlobalRuntime()
	get, set := GoUseAtom(parseRuntime, parseAtomID, parseAtomInitialValue)
	// Convert internal setter func(interface{}) to typed setter func(T)
	parseAtomSetTyped := func(parseAtomValue T) {
		set(parseAtomValue)
	}
	return get, parseAtomSetTyped
}

// GoUseTransitionPendingGlobal exposes the shared transition pending flag as a subscribed atom.
func GoUseTransitionPendingGlobal() (func() bool, func(bool)) {
	return GoUseAtomGlobal(transitionPendingAtomID, false)
}

// StartTransitionGlobal runs fn in a non-urgent transition context.
func StartTransitionGlobal(parseTransitionFn func()) {
	GetGlobalRuntime().StartTransition(parseTransitionFn)
}

// Text creates a text node
func Text(parseTextContent string) *Element {
	return &Element{
		Type:        "TEXT_ELEMENT",
		TextContent: parseTextContent,
		Children:    emptyChildren,
	}
}
