//go:build !js || !wasm

package render

import (
	"strconv"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// These benchmarks measure the client-side hydration walk (claimHydrationNode
// + boundary tracking) natively: the SSR markup is parsed into mock DOM via
// the adapter's CreateHTMLSubtree, then Hydrate adopts it. SSR *generation*
// was optimized earlier; this is the first measurement of the adoption side.

func buildHydrationList(parseRows int) ui.Node {
	getItems := make([]ui.Node, 0, parseRows)
	for parseIndex := 0; parseIndex < parseRows; parseIndex++ {
		getItems = append(getItems, html.Div(
			html.Props{
				Key:   strconv.Itoa(parseIndex),
				Class: "hydrate-row rounded-xl border px-3 py-2",
				Data:  map[string]string{"row-id": strconv.Itoa(parseIndex)},
			},
			html.Text("Row "+strconv.Itoa(parseIndex)),
		))
	}
	return html.Div(html.Props{ID: "hydrate-container", Class: "grid gap-2"}, getItems...)
}

func benchmarkHydration(parseB *testing.B, parseRows int) {
	getMarkup, parseErr := ui.RenderToString(buildHydrationList(parseRows))
	if parseErr != nil {
		parseB.Fatalf("RenderToString: %v", parseErr)
	}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseB.StopTimer()
		getAdapter := mockdom.NewMockDOMAdapter()
		getScheduler := mockdom.NewMockScheduler(true)
		getRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: getAdapter, Scheduler: getScheduler, Reset: true})
		getContainer := getAdapter.CreateElement("div").(*mockdom.MockDOMNode)
		getSSRRoot := getAdapter.CreateHTMLSubtree(getMarkup)
		if runtime.IsDOMNodeNull(getSSRRoot) {
			parseB.Fatal("failed to parse SSR markup into mock DOM")
		}
		getAdapter.AppendChild(getContainer, getSSRRoot)
		getElement := buildHydrationList(parseRows)
		parseB.StartTimer()

		getRuntime.Hydrate(getElement, getContainer)
		getScheduler.FlushAll()

		parseB.StopTimer()
		// Adoption sanity: hydration must reuse the SSR nodes, not replace them.
		if len(getContainer.Children) != 1 || len(getContainer.Children[0].Children) != parseRows {
			parseB.Fatalf("hydrated tree shape wrong: %d top, want 1 with %d rows",
				len(getContainer.Children), parseRows)
		}
		parseB.StartTimer()
	}
}

func BenchmarkHydrate200Rows(parseB *testing.B)  { benchmarkHydration(parseB, 200) }
func BenchmarkHydrate2000Rows(parseB *testing.B) { benchmarkHydration(parseB, 2000) }

// TestHydrationAdoptsServerDOM pins that the mock-parsed SSR tree is adopted
// node-for-node (no fallback re-create): after hydration the container's
// original parsed nodes are still the live children.
func TestHydrationAdoptsServerDOM(parseT *testing.T) {
	getMarkup, parseErr := ui.RenderToString(buildHydrationList(5))
	if parseErr != nil {
		parseT.Fatalf("RenderToString: %v", parseErr)
	}
	if !strings.Contains(getMarkup, "hydrate-container") {
		parseT.Fatalf("unexpected SSR markup: %s", getMarkup)
	}
	getAdapter := mockdom.NewMockDOMAdapter()
	getScheduler := mockdom.NewMockScheduler(true)
	getRuntime := runtime.NewRuntime(runtime.Config{DOMAdapter: getAdapter, Scheduler: getScheduler, Reset: true})
	getContainer := getAdapter.CreateElement("div").(*mockdom.MockDOMNode)
	getSSRRoot := getAdapter.CreateHTMLSubtree(getMarkup).(*mockdom.MockDOMNode)
	getAdapter.AppendChild(getContainer, getSSRRoot)
	getBeforeIDs := make([]int, 0, 5)
	for _, getRow := range getSSRRoot.Children {
		getBeforeIDs = append(getBeforeIDs, getRow.ID)
	}

	getRuntime.Hydrate(buildHydrationList(5), getContainer)
	getScheduler.FlushAll()

	if len(getContainer.Children) != 1 || getContainer.Children[0] != getSSRRoot {
		parseT.Fatalf("hydration replaced the SSR root instead of adopting it")
	}
	for parseIndex, getRow := range getSSRRoot.Children {
		if getRow.ID != getBeforeIDs[parseIndex] {
			parseT.Fatalf("row %d was recreated (id %d -> %d)", parseIndex, getBeforeIDs[parseIndex], getRow.ID)
		}
	}
}
