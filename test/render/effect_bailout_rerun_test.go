//go:build !js || !wasm

package render

import (
	"strconv"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestMountEffectDoesNotRerunAfterBailoutReuse pins that queued effects are
// consumed when they run. Fibers below a bailout boundary are REUSED (same
// Fiber object) across commits; a pass that queues no effects takes the
// full-tree effect scan, which previously re-executed every stale queued
// effect on reused fibers — re-firing mount effects on unrelated updates and
// overwriting their cleanups without calling them. The effect here sits one
// component level below the bailout clone line, where fibers are relinked
// rather than cloned.
func TestMountEffectDoesNotRerunAfterBailoutReuse(t *testing.T) {
	getAdapter := mockdom.NewMockDOMAdapter()
	getScheduler := mockdom.NewMockScheduler(false)
	getRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: getAdapter, Scheduler: getScheduler, Reset: true})
	getRoot := getAdapter.CreateElement("div").(*mockdom.MockDOMNode)

	getEffectRuns := 0
	getInnerWithEffect := func() ui.Node {
		runtime.GoUseEffect(func() func() {
			getEffectRuns++
			return nil
		}, 1)
		return html.Span(html.Props{Text: "inner"})
	}
	getEffectHolder := func() ui.Node {
		return html.Div(html.Props{}, ui.CreateElement(getInnerWithEffect))
	}

	var storeBump func(any)
	getCounter := func() ui.Node {
		getCount, parseSet := runtime.GoUseState(getRuntime, 0)
		storeBump = parseSet
		return html.Div(html.Props{}, html.Span(html.Props{Text: strconv.Itoa(getCount())}))
	}

	getApp := func() ui.Node {
		return html.Div(html.Props{},
			ui.CreateElement(getCounter),
			ui.CreateElement(getEffectHolder),
		)
	}

	if parseErr := getRuntime.RenderInto(getRoot, ui.CreateElement(getApp)); parseErr != nil {
		t.Fatalf("RenderInto: %v", parseErr)
	}
	getScheduler.FlushAll()
	if getEffectRuns != 1 {
		t.Fatalf("effect runs after mount = %d, want 1", getEffectRuns)
	}

	// Unrelated updates: the counter re-renders with no effects of its own, so
	// the commit takes the full-tree effect scan over the reused subtree.
	storeBump(1)
	getScheduler.FlushAll()
	storeBump(2)
	getScheduler.FlushAll()

	if getEffectRuns != 1 {
		t.Fatalf("mount effect re-ran on unrelated updates: runs = %d, want 1", getEffectRuns)
	}
}
