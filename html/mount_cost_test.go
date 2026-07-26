package html_test

import (
	"strconv"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// Mount cost per construction lane.
//
// html.Tag has two lanes. toRuntimeCompactProps routes typed Props that hold
// only string attributes (plus Data/Aria maps and a key) into the typed fast
// lane, which carries no props map. Everything else — a Style map, Raw props,
// Value, any numeric prop, any event handler — falls back to the map lane.
//
// The serialized mount (one HTML parse per subtree instead of one bridge call
// per node) was originally gated on the fast lane alone. The question these
// tests answer is which real markup lands where, because a wasm<->JS crossing
// is the dominant cost of an initial render and the lane decides how many there
// are.

// countingAdapter counts bridge operations. Embedding the concrete mock keeps
// every optional capability promoted, so the runtime picks the same commit
// strategy it would in a browser.
type countingAdapter struct {
	*mockdom.MockDOMAdapter
	creates    int
	texts      int
	attrs      int
	appends    int
	subtrees   int
	fragments  int
	traversals int
}

// Traversal crosses the boundary too. bindSerializedSubtree walks the parsed
// tree one GetFirstChild/GetNextSibling per node, so a count that ignores it
// reports a serialized mount as costing two crossings when it costs a walk.
func (parseA *countingAdapter) GetFirstChild(parseNode runtime.DOMNode) runtime.DOMNode {
	parseA.traversals++
	return parseA.MockDOMAdapter.GetFirstChild(parseNode)
}

func (parseA *countingAdapter) GetNextSibling(parseNode runtime.DOMNode) runtime.DOMNode {
	parseA.traversals++
	return parseA.MockDOMAdapter.GetNextSibling(parseNode)
}

func (parseA *countingAdapter) CreateElement(parseTag string) runtime.DOMNode {
	parseA.creates++
	return parseA.MockDOMAdapter.CreateElement(parseTag)
}

func (parseA *countingAdapter) CreateTextNode(parseText string) runtime.DOMNode {
	parseA.texts++
	return parseA.MockDOMAdapter.CreateTextNode(parseText)
}

func (parseA *countingAdapter) SetAttribute(parseNode runtime.DOMNode, parseName, parseValue string) {
	parseA.attrs++
	parseA.MockDOMAdapter.SetAttribute(parseNode, parseName, parseValue)
}

func (parseA *countingAdapter) AppendChild(parseParent, parseChild runtime.DOMNode) {
	parseA.appends++
	parseA.MockDOMAdapter.AppendChild(parseParent, parseChild)
}

func (parseA *countingAdapter) CreateHTMLSubtree(parseHTML string) runtime.DOMNode {
	parseA.subtrees++
	return parseA.MockDOMAdapter.CreateHTMLSubtree(parseHTML)
}

func (parseA *countingAdapter) CreateHTMLFragment(parseHTML string) runtime.DOMNode {
	parseA.fragments++
	return parseA.MockDOMAdapter.CreateHTMLFragment(parseHTML)
}

func (parseA *countingAdapter) total() int {
	return parseA.creates + parseA.texts + parseA.attrs + parseA.appends +
		parseA.subtrees + parseA.fragments + parseA.traversals
}

// benchmarkContentCard reproduces renderBenchmarkContentCard from the Example
// 201 render benchmark verbatim in shape: Class plus a Data map, which the
// compact lane accepts.
func benchmarkContentCard(parseIndex int, parseExtra html.Props) ui.Node {
	getTagNodes := make([]ui.Node, 0, 3)
	for parseTag := range 3 {
		getTagNodes = append(getTagNodes, html.Span(
			html.Props{Class: "benchmark-content-tag rounded-full border border-white/10 px-2 py-1 text-[11px] uppercase tracking-[0.14em] text-slate-300"},
			html.Text("tag-"+strconv.Itoa(parseTag)),
		))
	}
	parseProps := parseExtra
	parseProps.Class = "benchmark-content-card rounded-2xl border border-white/10 bg-white/[0.04] p-4 shadow-lg shadow-black/20"
	parseProps.Data = map[string]string{
		"refresh-token": "1",
		"card-id":       strconv.Itoa(parseIndex),
	}
	return html.Article(parseProps,
		html.Div(html.Props{Class: "flex items-center justify-between gap-3"},
			html.H2(html.Props{Class: "benchmark-content-title text-base font-semibold text-white"}, html.Text("title")),
			html.Span(html.Props{Class: "benchmark-content-status rounded-full border border-cyan-400/20 bg-cyan-400/10 px-2 py-1"}, html.Text("status")),
		),
		html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text("summary")),
		html.P(html.Props{Class: "benchmark-content-meta mt-3 text-xs uppercase"}, html.Text("meta")),
		html.Div(html.Props{Class: "mt-4 flex flex-wrap gap-2"}, getTagNodes...),
	)
}

func mountCards(parseT *testing.T, parseExtra html.Props) *countingAdapter {
	parseT.Helper()
	parseAdapter := &countingAdapter{MockDOMAdapter: mockdom.NewMockDOMAdapter()}
	parseContainer := parseAdapter.MockDOMAdapter.CreateElement("section")

	parseChildren := make([]ui.Node, 0, 12)
	for parseIndex := range 12 {
		parseChildren = append(parseChildren, benchmarkContentCard(parseIndex, parseExtra))
	}
	parseRoot := html.Div(html.Props{Class: "benchmark-content-grid"}, parseChildren...)

	runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter}).
		Render(parseRoot, parseContainer)
	return parseAdapter
}

// TestCompactLaneMarkupAlreadyMountedInOneCall records the state of the world
// BEFORE the serialized mount was widened, and pins it so it cannot regress.
//
// The Example 201 content card is pure string attributes plus a Data map, so it
// takes the typed fast lane and was always eligible. This is why widening the
// gate to the map lane produced no movement in that benchmark: those elements
// were never the ones paying per-node cost.
func TestCompactLaneMarkupAlreadyMountedInOneCall(parseT *testing.T) {
	parseAdapter := mountCards(parseT, html.Props{})

	parseT.Logf("compact lane, 12 benchmark content cards: creates=%d texts=%d attrs=%d appends=%d subtrees=%d fragments=%d TOTAL=%d",
		parseAdapter.creates, parseAdapter.texts, parseAdapter.attrs, parseAdapter.appends,
		parseAdapter.subtrees, parseAdapter.fragments, parseAdapter.total())

	if parseAdapter.subtrees+parseAdapter.fragments == 0 {
		parseT.Error("compact-lane markup did not take the serialized mount at all")
	}
	// About 159: one parse, one append, and a bind walk of roughly two crossings
	// per node. NOT "a handful" — an earlier version of this bound said 10,
	// because the counter ignored the GetFirstChild/GetNextSibling traversal
	// that bindSerializedSubtree performs on every node.
	if parseAdapter.total() > 200 {
		parseT.Errorf("compact-lane markup cost %d bridge crossings; the serialized mount stopped firing",
			parseAdapter.total())
	}
}

// TestMapLaneMarkupAlsoMountsInOneCall is what widening the gate bought.
//
// A Raw prop pushes the card onto the map lane while every value stays a
// string, so the fiber is compact — its attributes are fully captured in
// getHostAttrs — but it carries a props map. The old gate tested for BOTH
// conditions, so the props map alone disqualified the whole subtree and it
// reverted to one bridge call per node.
//
// Measured on this exact tree by toggling the gate, with traversal counted:
// 279 crossings before, 159 after — a 43% reduction, not the 97x an earlier
// creation-only count implied. The bind walk costs about two crossings per node
// either way, so the saving is the per-node creation, not the whole cost.
//
// The same applies to any element built through runtime.CreateElement with a
// string props map, which is how the ui package and generated code construct
// markup.
func TestMapLaneMarkupAlsoMountsInOneCall(parseT *testing.T) {
	parseAdapter := mountCards(parseT, html.Props{Raw: map[string]any{"data-flag": "on"}})

	parseT.Logf("map lane (Raw props), 12 benchmark content cards: creates=%d texts=%d attrs=%d appends=%d subtrees=%d fragments=%d TOTAL=%d",
		parseAdapter.creates, parseAdapter.texts, parseAdapter.attrs, parseAdapter.appends,
		parseAdapter.subtrees, parseAdapter.fragments, parseAdapter.total())

	if parseAdapter.subtrees+parseAdapter.fragments == 0 {
		parseT.Error("no subtree was serialized on the map lane")
	}
	if parseAdapter.total() > 200 {
		parseT.Errorf("map-lane markup cost %d bridge crossings, against ~159 when the widening is working and ~279 without it; the serialized mount stopped reaching map-lane subtrees",
			parseAdapter.total())
	}
}

// TestStyleMapMarkupStillMountsPerNode records a REMAINING cost, so it is not
// mistaken for covered ground.
//
// A Style map is a map value, so buildElementHostProps clears
// isCompactHostProps and getHostAttrs is discarded — there is no attribute
// slice to serialize from, and the widening cannot help. Style-bearing elements
// therefore still mount one bridge call per node. Serializing them would mean
// flattening the map to a style="..." string, which is a change to how
// compactness itself is computed and would affect the UPDATE differ, not just
// mounting.
//
// The assertion is deliberately loose: it pins the shape as unimproved rather
// than pinning a number nobody should optimize against directly.
func TestStyleMapMarkupStillMountsPerNode(parseT *testing.T) {
	parseAdapter := mountCards(parseT, html.Props{Style: map[string]string{"opacity": "1"}})

	parseT.Logf("style map (still per-node), 12 benchmark content cards: creates=%d texts=%d attrs=%d appends=%d subtrees=%d fragments=%d TOTAL=%d",
		parseAdapter.creates, parseAdapter.texts, parseAdapter.attrs, parseAdapter.appends,
		parseAdapter.subtrees, parseAdapter.fragments, parseAdapter.total())

	if parseAdapter.total() <= 200 {
		parseT.Errorf("style-map markup now costs only %d crossings — if it was made serializable, this test documents stale limits and should be rewritten",
			parseAdapter.total())
	}
}
