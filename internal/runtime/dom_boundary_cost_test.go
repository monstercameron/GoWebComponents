package runtime

import (
	"fmt"
	"testing"
)

// DOM boundary cost — where the initial-render deficit against React actually is.
//
// Every DOMAdapter call is one wasm<->JS crossing in a browser build:
// CreateElement, SetAttribute, AppendChild are separate crossings, each
// marshalling its arguments through the syscall/js shim. React does the same DOM
// work with ZERO crossings, because it is already in JS.
//
// Measured on the 201 render benchmark, the content-render scenario:
//
//	GWC    13.471 ms   1 mutation record,  12 nodes added
//	React   1.843 ms  12 mutation records, 12 nodes added
//
// GWC is 7.3x slower while issuing FEWER DOM mutations — it builds detached and
// attaches once. So the deficit is not DOM operation count. This test counts what
// it actually is: 374 boundary crossings for that scenario, which against 13.471
// ms is roughly 36 us apiece.
//
// For scale, the reconciler builds 600 fibers in ~25 us (BenchmarkMountFromEmpty200).
// Reconciliation is not the cost; the boundary is. Slab-allocating fibers cut ~8 us
// from this scenario — 0.06% — and correctly showed no browser movement.
//
// The lever this points at is batching the commit into ONE crossing rather than
// making each crossing faster.

// countingDOMAdapter wraps the test adapter and counts boundary operations.
//
// Each of these is one syscall/js crossing in a real browser build.
type countingDOMAdapter struct {
	*testDOMAdapter
	creates    int
	attrs      int
	appends    int
	properties int
	texts      int
}

func (parseA *countingDOMAdapter) CreateElement(parseTag string) DOMNode {
	parseA.creates++
	return parseA.testDOMAdapter.CreateElement(parseTag)
}
func (parseA *countingDOMAdapter) CreateTextNode(parseText string) DOMNode {
	parseA.texts++
	return parseA.testDOMAdapter.CreateTextNode(parseText)
}
func (parseA *countingDOMAdapter) SetAttribute(parseNode DOMNode, parseName, parseValue string) {
	parseA.attrs++
	parseA.testDOMAdapter.SetAttribute(parseNode, parseName, parseValue)
}
func (parseA *countingDOMAdapter) SetProperty(parseNode DOMNode, parseName string, parseValue any) {
	parseA.properties++
	parseA.testDOMAdapter.SetProperty(parseNode, parseName, parseValue)
}
func (parseA *countingDOMAdapter) AppendChild(parseParent, parseChild DOMNode) {
	parseA.appends++
	parseA.testDOMAdapter.AppendChild(parseParent, parseChild)
}

// buildContentCardTree approximates one render-benchmark content card: an
// article wrapping a header row, two paragraphs, and a tag strip.
func buildContentCardTree(parseCards int, parseTags int) []any {
	parseOut := make([]any, 0, parseCards)
	for parseCard := range parseCards {
		parseTagNodes := make([]any, 0, parseTags)
		for parseTag := range parseTags {
			parseTagNodes = append(parseTagNodes, &Element{Type: "span", Props: map[string]any{
				"class":    "benchmark-content-tag rounded-full border border-white/10 px-2 py-1 text-[11px] uppercase tracking-[0.14em] text-slate-300",
				"children": []any{fmt.Sprintf("tag-%d", parseTag)},
			}})
		}
		parseOut = append(parseOut, &Element{Type: "article", Props: map[string]any{
			"class": "benchmark-content-card rounded-2xl border border-white/10 bg-white/[0.04] p-4 shadow-lg shadow-black/20",
			"key":   fmt.Sprintf("card-%d", parseCard),
			"children": []any{
				&Element{Type: "div", Props: map[string]any{"class": "flex items-center justify-between gap-3", "children": []any{
					&Element{Type: "h2", Props: map[string]any{"class": "benchmark-content-title text-base font-semibold text-white", "children": []any{"title"}}},
					&Element{Type: "span", Props: map[string]any{"class": "benchmark-content-status rounded-full border border-cyan-400/20 bg-cyan-400/10 px-2 py-1", "children": []any{"status"}}},
				}}},
				&Element{Type: "p", Props: map[string]any{"class": "mt-3 text-sm leading-6 text-slate-300", "children": []any{"summary"}}},
				&Element{Type: "p", Props: map[string]any{"class": "benchmark-content-meta mt-3 text-xs uppercase", "children": []any{"meta"}}},
				&Element{Type: "div", Props: map[string]any{"class": "mt-4 flex flex-wrap gap-2", "children": parseTagNodes}},
			},
		}})
	}
	return parseOut
}

// TestDOMBoundaryCrossingsAreTheInitialRenderCost quantifies the crossings and
// guards against them growing.
func TestDOMBoundaryCrossingsAreTheInitialRenderCost(parseT *testing.T) {
	parseAdapter := &countingDOMAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})

	parseContainer := parseAdapter.CreateElement("section")
	parseAdapter.creates = 0
	parseRoot := &Element{Type: "div", Props: map[string]any{"children": buildContentCardTree(12, 3)}}
	parseRt.Render(parseRoot, parseContainer)

	parseTotal := parseAdapter.creates + parseAdapter.texts + parseAdapter.attrs +
		parseAdapter.properties + parseAdapter.appends
	parseT.Logf("content-render (12 cards x 3 tags): creates=%d texts=%d attrs=%d props=%d appends=%d  TOTAL=%d",
		parseAdapter.creates, parseAdapter.texts, parseAdapter.attrs,
		parseAdapter.properties, parseAdapter.appends, parseTotal)

	// A guard, not a target. If a change makes the commit chattier this catches
	// it; the real win would be a large DROP, which should lower this bound.
	if parseTotal > 500 {
		parseT.Errorf("content-render now costs %d boundary crossings; the commit got chattier", parseTotal)
	}
}
