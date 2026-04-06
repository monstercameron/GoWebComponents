//go:build !js || !wasm
// +build !js !wasm

package runtime

import "reflect"

// GoUseStateGlobal wraps GoUseState with the global runtime on non-browser targets.
func GoUseStateGlobal[T any](parseStateInitialValue T) (func() T, func(interface{})) {
	parseRuntime := GetGlobalRuntime()
	return GoUseState(parseRuntime, parseStateInitialValue)
}

// GoUseEffectGlobal wraps GoUseEffect on non-browser targets.
func GoUseEffectGlobal(parseEffectFn func() func(), parseEffectDeps ...interface{}) {
	GoUseEffect(parseEffectFn, parseEffectDeps...)
}

// GoUseMemoGlobal wraps GoUseMemo on non-browser targets.
func GoUseMemoGlobal(parseMemoCompute func() interface{}, parseMemoDeps ...interface{}) interface{} {
	return GoUseMemo(parseMemoCompute, parseMemoDeps...)
}

// GoUseMemoGlobalTyped wraps GoUseMemoTyped on non-browser targets.
func GoUseMemoGlobalTyped(parseMemoCompute func() interface{}, parseMemoTargetType reflect.Type, parseMemoDeps ...interface{}) interface{} {
	return GoUseMemoTyped(parseMemoCompute, parseMemoTargetType, parseMemoDeps...)
}

// GoUseIdGlobal wraps GoUseId on non-browser targets.
func GoUseIdGlobal() string {
	return GoUseId()
}

// GoUseAtomGlobal wraps GoUseAtom with the global runtime on non-browser targets.
func GoUseAtomGlobal[T any](parseAtomID string, parseAtomInitialValue T) (func() T, func(T)) {
	parseRuntime := GetGlobalRuntime()
	parseGetter, parseSetter := GoUseAtom(parseRuntime, parseAtomID, parseAtomInitialValue)
	return parseGetter, func(parseValue T) {
		parseSetter(parseValue)
	}
}
