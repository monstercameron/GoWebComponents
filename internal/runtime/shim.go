//go:build js && wasm
// +build js,wasm

package runtime

import "reflect"

// Global wrapper functions that provide simplified API for public packages
// These match the old fiber package API

// GoUseStateGlobal wraps GoUseState with global fiber context
func GoUseStateGlobal[T any](parseInitialValue T) (func() T, func(interface{})) {
	parseRt := GetGlobalRuntime()
	return GoUseState(parseRt, parseInitialValue)
}

// GoUseEffectGlobal wraps GoUseEffect
func GoUseEffectGlobal(parseEffect func() func(), parseDeps ...interface{}) {
	// GoUseEffect doesn't need Runtime, it works with current fiber
	GoUseEffect(parseEffect, parseDeps...)
}

// GoUseMemoGlobal wraps GoUseMemo
func GoUseMemoGlobal(parseCompute func() interface{}, parseDeps ...interface{}) interface{} {
	// GoUseMemo doesn't need Runtime, it works with current fiber
	return GoUseMemo(parseCompute, parseDeps...)
}

// GoUseMemoGlobalTyped wraps GoUseMemo with an expected target type for
// hot-reload restoration.
func GoUseMemoGlobalTyped(parseCompute func() interface{}, parseTargetType reflect.Type, parseDeps ...interface{}) interface{} {
	return GoUseMemoTyped(parseCompute, parseTargetType, parseDeps...)
}

// GoUseCallbackGlobal wraps GoUseCallback
func GoUseCallbackGlobal(parseFn interface{}, parseDeps ...interface{}) interface{} {
	// GoUseCallback doesn't need Runtime, it works with current fiber
	return GoUseCallback(parseFn, parseDeps...)
}

// GoUseRefGlobal wraps GoUseRef
func GoUseRefGlobal(parseInitialValue interface{}) *RefValue {
	// GoUseRef doesn't need Runtime, it works with current fiber
	return GoUseRef(parseInitialValue)
}

// GoUseIdGlobal wraps GoUseId
func GoUseIdGlobal() string {
	// GoUseId doesn't need Runtime, it works with current fiber
	return GoUseId()
}

// GoUseFetchGlobal wraps GoUseFetch with global fiber context
func GoUseFetchGlobal(parseUrl string, parseOptions ...interface{}) (func() FetchState, func()) {
	// GoUseFetch doesn't need Runtime, it works with current fiber
	return GoUseFetch(parseUrl, parseOptions...)
}

// GoUseFuncGlobal wraps GoUseFunc with WASM event handler wrapping
func GoUseFuncGlobal(parseFn interface{}) interface{} {
	// The core GoUseFunc now handles wrapping via DOMAdapter
	return GoUseFunc(parseFn)
}

// GoUseAtomGlobal wraps GoUseAtom with global runtime
func GoUseAtomGlobal[T any](parseId string, parseInitialValue T) (func() T, func(T)) {
	parseRt := GetGlobalRuntime()
	get, set := GoUseAtom(parseRt, parseId, parseInitialValue)
	// Convert internal setter func(interface{}) to typed setter func(T)
	parseTypedSet := func(parseV T) {
		set(parseV)
	}
	return get, parseTypedSet
}

// GoUseTransitionPendingGlobal exposes the shared transition pending flag as a subscribed atom.
func GoUseTransitionPendingGlobal() (func() bool, func(bool)) {
	return GoUseAtomGlobal(transitionPendingAtomID, false)
}

// StartTransitionGlobal runs fn in a non-urgent transition context.
func StartTransitionGlobal(parseFn func()) {
	GetGlobalRuntime().StartTransition(parseFn)
}

// Text creates a text node
func Text(parseContent string) *Element {
	return &Element{
		Type:        "TEXT_ELEMENT",
		TextContent: parseContent,
		Children:    emptyChildren,
	}
}
