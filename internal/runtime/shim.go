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

// GoUseAtomGlobal wraps GoUseAtom with global runtime
func GoUseAtomGlobal[T any](id string, initialValue T) (func() T, func(interface{})) {
	rt := GetGlobalRuntime()
	return GoUseAtom(rt, id, initialValue)
}

// Text creates a text node
func Text(content string) *Element {
	return &Element{
		Type:     "TEXT_ELEMENT",
		Props:    map[string]interface{}{"nodeValue": content},
		Children: []interface{}{},
	}
}
