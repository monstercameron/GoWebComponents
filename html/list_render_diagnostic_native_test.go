//go:build !js

package html_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// countListHookDiagnostics mounts parseComponent on a mock DOM runtime and
// returns how many GWC-RUNTIME-HOOK-IN-LIST diagnostics it produced.
func countListHookDiagnostics(parseT *testing.T, parseComponent func() *runtime.Element) int {
	parseT.Helper()
	runtime.ClearDiagnostics()
	defer runtime.ClearDiagnostics()
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseContainer := parseAdapter.CreateElement("section")
	runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter}).
		Render(runtime.CreateElement(parseComponent, nil), parseContainer)
	parseCount := 0
	for _, parseDiagnostic := range runtime.GetDiagnostics() {
		if parseDiagnostic.Code == "GWC-RUNTIME-HOOK-IN-LIST" {
			parseCount++
		}
	}
	return parseCount
}

func TestMapAndMapKeyedReportHandlerHooksInLists(parseT *testing.T) {
	parseItems := []int{1, 2, 3}
	parseRow := func(parseItem int) ui.Node {
		// The handler hook is what the inline OnClick(func) allocates on wasm.
		runtime.GoUseFunc(func() {})
		return html.Li(html.Props{})
	}
	parseKey := func(parseItem int) any { return parseItem }
	if parseCount := countListHookDiagnostics(parseT, func() *runtime.Element {
		return html.Ul(html.Props{}, html.Map(parseItems, parseRow)...)
	}); parseCount != 1 {
		parseT.Errorf("Map: want exactly one diagnostic, got %d", parseCount)
	}
	if parseCount := countListHookDiagnostics(parseT, func() *runtime.Element {
		return html.Ul(html.Props{}, html.MapKeyed(parseItems, parseKey, parseRow)...)
	}); parseCount != 1 {
		parseT.Errorf("MapKeyed: want exactly one diagnostic, got %d", parseCount)
	}
}

func TestMapKeyedComponentRowsReportNothing(parseT *testing.T) {
	parseItems := []int{1, 2, 3}
	parseCount := countListHookDiagnostics(parseT, func() *runtime.Element {
		return html.Ul(html.Props{}, html.MapKeyedComponent(parseItems,
			func(parseItem int) any { return parseItem },
			func(parseItem int) ui.Node {
				runtime.GoUseFunc(func() {})
				return html.Li(html.Props{})
			})...)
	})
	if parseCount != 0 {
		parseT.Errorf("MapKeyedComponent rows own their hooks; got %d diagnostics", parseCount)
	}
}
