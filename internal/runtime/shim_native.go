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

// GoUseLayoutEffectGlobal wraps GoUseLayoutEffect on non-browser targets (G36).
func GoUseLayoutEffectGlobal(parseEffectFn func() func(), parseEffectDeps ...any) {
	GoUseLayoutEffect(parseEffectFn, parseEffectDeps...)
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

// GoUseAtomGlobal wraps GoUseAtom on non-browser targets, resolving the atom
// registry to the CURRENT SERVER RENDER's scope when there is one and to the
// process-global runtime otherwise.
//
// The "otherwise" is the browser-equivalent case (native reconciler renders, unit
// tests, out-of-render access) and must stay on the global registry: an atom is
// supposed to be shared and to outlive renders.
//
// The scoped case is not an optimization, it is the fix for a cross-request state
// leak. Resolving straight to GetGlobalRuntime() here meant every server render in
// the process shared one registry, and AtomRegistry.InitAtom is init-if-absent —
// so request 1 seeded the atom and request 2's initial value was ignored, putting
// request 1's data in request 2's HTML. A server render also never unmounts, so
// its subscriptions were never released and the registry grew forever.
//
// If you are tempted to put GetGlobalRuntime() back because "atoms are global":
// they are, per PAGE. On a server, per REQUEST is the equivalent boundary.
// ssr_atom_scope.go states the whole contract.
func GoUseAtomGlobal[T any](parseAtomID string, parseAtomInitialValue T) (func() T, func(T)) {
	parseRuntime := ssrRequestAtomScope()
	if parseRuntime == nil {
		parseRuntime = GetGlobalRuntime()
	}
	parseGetter, parseSetter := GoUseAtom(parseRuntime, parseAtomID, parseAtomInitialValue)
	return parseGetter, func(parseValue T) {
		parseSetter(parseValue)
	}
}
