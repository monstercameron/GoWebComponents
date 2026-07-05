package render

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func TestHydrationDiagnosticStorm(t *testing.T) {
	getMarkup, _ := ui.RenderToString(buildHydrationList(20))
	getAdapter := mockdom.NewMockDOMAdapter()
	getScheduler := mockdom.NewMockScheduler(true)
	getRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: getAdapter, Scheduler: getScheduler, Reset: true})
	getContainer := getAdapter.CreateElement("div").(*mockdom.MockDOMNode)
	getAdapter.AppendChild(getContainer, getAdapter.CreateHTMLSubtree(getMarkup).(*mockdom.MockDOMNode))
	getBefore := len(runtime.GetDiagnostics())
	getRuntime.Hydrate(buildHydrationList(20), getContainer)
	getScheduler.FlushAll()
	getDiags := runtime.GetDiagnostics()
	t.Logf("diagnostics before=%d after=%d", getBefore, len(getDiags))
	// Pin: hydrating markup parsed back through CreateHTMLSubtree must not
	// emit per-node mismatch warnings (a storm here previously turned the
	// diagnostics ring quadratic -- 478ms for a 2000-row hydration).
	for _, getD := range getDiags {
		if getD.Severity != runtime.DiagnosticInfo {
			t.Fatalf("unexpected %s diagnostic during clean hydration: %s", getD.Severity, getD.Message)
		}
	}
	if len(getDiags)-getBefore > 2 {
		t.Fatalf("clean 20-row hydration emitted %d diagnostics, want <=2", len(getDiags)-getBefore)
	}
}
