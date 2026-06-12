//go:build !js || !wasm

package runtime

import "reflect"

// GoUseStateGlobal wraps GoUseState with the global runtime on non-browser targets.
func GoUseStateGlobal[T any](parseStateInitialValue T) (func() T, func(any)) {
	parseRuntime := GetGlobalRuntime()
	return GoUseState(parseRuntime, parseStateInitialValue)
}

// GoUseEffectGlobal wraps GoUseEffect on non-browser targets.
func GoUseEffectGlobal(parseEffectFn func() func(), parseEffectDeps ...any) {
	GoUseEffect(parseEffectFn, parseEffectDeps...)
}

// GoUseMemoGlobal wraps GoUseMemo on non-browser targets.
func GoUseMemoGlobal(parseMemoCompute func() any, parseMemoDeps ...any) any {
	return GoUseMemo(parseMemoCompute, parseMemoDeps...)
}

// GoUseMemoGlobalTyped wraps GoUseMemoTyped on non-browser targets.
func GoUseMemoGlobalTyped(parseMemoCompute func() any, parseMemoTargetType reflect.Type, parseMemoDeps ...any) any {
	return GoUseMemoTyped(parseMemoCompute, parseMemoTargetType, parseMemoDeps...)
}

// GoUseIdGlobal wraps GoUseId on non-browser targets.
func GoUseIdGlobal() string {
	return GoUseId()
}

// BuildDOMWrappedFunctionIfReadyGlobal reports that DOM callback wrapping is unavailable on non-browser targets.
func BuildDOMWrappedFunctionIfReadyGlobal(parseHandlerFn any) (any, bool) {
	return nil, false
}

// GoUseAtomGlobal wraps GoUseAtom with the global runtime on non-browser targets.
func GoUseAtomGlobal[T any](parseAtomID string, parseAtomInitialValue T) (func() T, func(T)) {
	parseRuntime := GetGlobalRuntime()
	parseGetter, parseSetter := GoUseAtom(parseRuntime, parseAtomID, parseAtomInitialValue)
	return parseGetter, func(parseValue T) {
		parseSetter(parseValue)
	}
}
