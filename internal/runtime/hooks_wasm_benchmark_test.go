//go:build js && wasm && production

package runtime

import "testing"

// BenchmarkHookGridRerender models the production browser benchmark's 40
// components with 20 state/effect/memo hooks each across steady re-renders.
func BenchmarkHookGridRerender(parseB *testing.B) {
	const parseComponentCount = 40
	const parseHooksPerComponent = 20

	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()
	parseWasMultiRootActive := multiRootActive.Load()
	multiRootActive.Store(false)
	defer multiRootActive.Store(parseWasMultiRootActive)
	parseRt := GetGlobalRuntime()
	parseFibers := make([]*Fiber, parseComponentCount)
	parseEffect := func() func() { return nil }
	parseMemo := func(parseDep int) int { return parseDep * 2 }
	for parseComponent := range parseFibers {
		parseFiber := &Fiber{typeOf: "hook-grid-cell", ownerRuntime: parseRt}
		parseFibers[parseComponent] = parseFiber
		SetCurrentFiber(parseFiber)
		for parseHook := 0; parseHook < parseHooksPerComponent; parseHook++ {
			_ = GoUseStateSlotGlobal(parseHook + parseComponent)
			GoUseEffectOf(parseEffect, parseHook)
			_ = GoUseMemoOf(parseMemo, parseHook)
		}
	}
	SetCurrentFiber(nil)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseB.Loop() {
		for _, parseFiber := range parseFibers {
			resetHookRenderState(parseFiber)
			SetCurrentFiber(parseFiber)
			for parseHook := 0; parseHook < parseHooksPerComponent; parseHook++ {
				_ = GoUseStateSlotGlobal(parseHook)
				GoUseEffectOf(parseEffect, parseHook)
				_ = GoUseMemoOf(parseMemo, parseHook)
			}
		}
	}
	SetCurrentFiber(nil)
}
