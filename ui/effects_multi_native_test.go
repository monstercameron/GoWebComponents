//go:build !js || !wasm

package ui_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// Deepens the effects port toward React equivalency: several effects in one
// component run in declaration order on mount; on a re-render a dep-driven effect
// re-runs (cleanup then setup) only when its deps change, a static-deps effect
// does not re-run, and a no-deps effect re-runs every render; cleanups run on
// unmount.
func TestMultipleEffectsOrderingAndDeps(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	var parseLog []string
	var parseSetDep func(any)

	parseComponent := func() *runtime.Element {
		parseGetDep, parseSetter := runtime.GoUseState(rt, 0)
		parseSetDep = parseSetter
		runtime.GoUseEffect(func() func() {
			parseLog = append(parseLog, "setup-static")
			return func() { parseLog = append(parseLog, "cleanup-static") }
		}, 0)
		runtime.GoUseEffect(func() func() {
			parseLog = append(parseLog, "setup-dep")
			return func() { parseLog = append(parseLog, "cleanup-dep") }
		}, parseGetDep())
		runtime.GoUseEffect(func() func() {
			parseLog = append(parseLog, "setup-nodeps")
			return nil
		})
		return runtime.CreateElement("span", map[string]any{}, "x")
	}

	rt.RenderInto(root, runtime.CreateElement("section", map[string]any{},
		runtime.CreateElement(parseComponent, map[string]any{})))
	if parseGot := strings.Join(parseLog, ","); parseGot != "setup-static,setup-dep,setup-nodeps" {
		t.Fatalf("mount effect order = %q", parseGot)
	}

	// Dep change: the dep-driven effect re-runs (cleanup then setup); the no-deps
	// effect re-runs; the static-deps effect does not.
	parseLog = nil
	parseSetDep(1)
	if parseGot := strings.Join(parseLog, ","); parseGot != "cleanup-dep,setup-dep,setup-nodeps" {
		t.Errorf("dep-change effect order = %q, want cleanup-dep,setup-dep,setup-nodeps", parseGot)
	}

	// Unmount: cleanups run (the no-deps effect returned nil, so no cleanup-nodeps).
	parseLog = nil
	rt.RenderInto(root, runtime.CreateElement("section", map[string]any{}))
	if parseGot := strings.Join(parseLog, ","); parseGot != "cleanup-static,cleanup-dep" {
		t.Errorf("unmount cleanup order = %q, want cleanup-static,cleanup-dep", parseGot)
	}
}
