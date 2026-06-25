//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// Regression test for v3.5.4: render-phase convergence (v3.5.3) must commit
// exactly the final render's effects — running each effect once, with the
// converged state — for both stable and changing deps. The v3.5.3 convergence
// loop cleared the per-iteration effect list, which (combined with the
// deps-equality check) dropped a stable-deps effect entirely.
func TestRenderPhaseEffectRunsOnceStableDeps(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	parseEffectRuns := 0
	var parseSaw []int
	rt.RenderInto(root, runtime.CreateElement(func() *runtime.Element {
		parseGet, parseSet := runtime.GoUseState(rt, 0)
		if parseGet() < 3 {
			parseSet(parseGet() + 1)
		}
		runtime.GoUseEffect(func() func() {
			parseEffectRuns++
			parseSaw = append(parseSaw, parseGet())
			return nil
		}, 0) // stable deps
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet()))
	}, map[string]any{}))

	if parseEffectRuns != 1 {
		t.Errorf("effect runs = %d, want 1 (only the final converged render), saw=%v", parseEffectRuns, parseSaw)
	}
	if len(parseSaw) != 1 || parseSaw[0] != 3 {
		t.Errorf("effect observed %v, want [3] (the converged state)", parseSaw)
	}
}

func TestRenderPhaseEffectRunsOnceChangingDeps(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	parseEffectRuns := 0
	var parseSaw []int
	rt.RenderInto(root, runtime.CreateElement(func() *runtime.Element {
		parseGet, parseSet := runtime.GoUseState(rt, 0)
		if parseGet() < 3 {
			parseSet(parseGet() + 1)
		}
		runtime.GoUseEffect(func() func() {
			parseEffectRuns++
			parseSaw = append(parseSaw, parseGet())
			return nil
		}, parseGet()) // deps change as state converges
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet()))
	}, map[string]any{}))

	if parseEffectRuns != 1 {
		t.Errorf("changing-deps effect runs = %d, want 1 (deduped to the final render), saw=%v", parseEffectRuns, parseSaw)
	}
	if len(parseSaw) != 1 || parseSaw[0] != 3 {
		t.Errorf("changing-deps effect observed %v, want [3]", parseSaw)
	}
}

// Two effects in a render-phase-converging component: each runs once, in order,
// with the converged state.
func TestRenderPhaseMultipleEffectsEachOnce(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	var parseLog []string
	rt.RenderInto(root, runtime.CreateElement(func() *runtime.Element {
		parseGet, parseSet := runtime.GoUseState(rt, 0)
		if parseGet() < 2 {
			parseSet(parseGet() + 1)
		}
		runtime.GoUseEffect(func() func() {
			parseLog = append(parseLog, fmt.Sprintf("a:%d", parseGet()))
			return nil
		}, 0)
		runtime.GoUseEffect(func() func() {
			parseLog = append(parseLog, fmt.Sprintf("b:%d", parseGet()))
			return nil
		}, 0)
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet()))
	}, map[string]any{}))

	if len(parseLog) != 2 || parseLog[0] != "a:2" || parseLog[1] != "b:2" {
		t.Errorf("two effects log = %v, want [a:2 b:2] (each once, converged, in order)", parseLog)
	}
}
