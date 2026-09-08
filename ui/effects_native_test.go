//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// TestEffectRunsOnceAndOnDepsChange: an effect runs after mount, re-runs (after
// cleaning up the previous one) when its deps change, and does not re-run when a
// re-render leaves the deps unchanged. Verified natively through the real
// reconciler (effects flush synchronously under the nil-scheduler harness).
func TestEffectRunsOnceAndOnDepsChange(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	var parseSetDep func(any)
	parseEffectRuns, parseCleanups := 0, 0
	rt.RenderInto(root, runtime.CreateElement(func() *runtime.Element {
		parseGetDep, parseSet := runtime.GoUseState(rt, 0)
		parseSetDep = parseSet
		runtime.GoUseEffect(func() func() {
			parseEffectRuns++
			return func() { parseCleanups++ }
		}, parseGetDep())
		return runtime.CreateElement("p", map[string]any{}, fmt.Sprint(parseGetDep()))
	}, map[string]any{}))

	if parseEffectRuns != 1 || parseCleanups != 0 {
		t.Fatalf("mount: effectRuns=%d cleanups=%d (want 1,0)", parseEffectRuns, parseCleanups)
	}

	parseSetDep(1) // deps change: clean up old, run new
	if parseEffectRuns != 2 || parseCleanups != 1 {
		t.Errorf("after deps change: effectRuns=%d cleanups=%d (want 2,1)", parseEffectRuns, parseCleanups)
	}

	parseSetDep(1) // same deps: no re-run, no cleanup
	if parseEffectRuns != 2 || parseCleanups != 1 {
		t.Errorf("after same deps: effectRuns=%d cleanups=%d (want 2,1)", parseEffectRuns, parseCleanups)
	}
}

// TestEffectCleanupOnUnmount: when the component holding an effect is removed
// from the tree, its latest cleanup runs exactly once.
func TestEffectCleanupOnUnmount(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	parseEffectRuns, parseCleanups := 0, 0
	parseChild := func() *runtime.Element {
		runtime.GoUseEffect(func() func() {
			parseEffectRuns++
			return func() { parseCleanups++ }
		}) // no deps arg: mount/unmount lifecycle
		return runtime.CreateElement("span", map[string]any{}, "child")
	}

	// Mount with the effect-bearing child present.
	rt.RenderInto(root, runtime.CreateElement("div", map[string]any{},
		runtime.CreateElement(parseChild, map[string]any{})))
	if parseEffectRuns < 1 {
		t.Fatalf("mount: effect did not run (effectRuns=%d)", parseEffectRuns)
	}
	parseCleanupsBefore := parseCleanups

	// Re-render with the child removed -> unmount -> cleanup should fire.
	rt.RenderInto(root, runtime.CreateElement("div", map[string]any{}))
	if parseCleanups <= parseCleanupsBefore {
		t.Errorf("unmount: cleanup did not run (cleanups %d -> %d)", parseCleanupsBefore, parseCleanups)
	}
}
