package mockdom

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
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
// attaches once. So the deficit is not DOM operation count; it is the crossings.
//
// These tests live in mockdom rather than runtime deliberately. MockDOMAdapter
// implements CreateHTMLSubtree and CreateHTMLFragment, which the browser adapter
// also implements and runtime's own testDOMAdapter does NOT. Counting crossings
// against an adapter that lacks those capabilities measures the fallback path —
// a configuration that never ships — and overstates the cost.
//
// For scale, the reconciler builds 600 fibers in ~25 us (BenchmarkMountFromEmpty200).
// Reconciliation is not the cost; the boundary is. Slab-allocating fibers cut ~8 us
// from this scenario — 0.06% — and correctly showed no browser movement. The lever
// is batching the commit into few crossings, not making each crossing faster.

// countingDOMAdapter wraps the mock adapter and counts boundary operations.
//
// Embedding the CONCRETE mock rather than the interface keeps every optional
// capability the runtime probes for (serialized subtree, fragment, prepared
// host, batched attrs) promoted and intact — a wrapper that hid one would
// silently change which commit strategy the runtime picks, which is the thing
// being measured.
type countingDOMAdapter struct {
	*MockDOMAdapter
	creates      int
	attrs        int
	appends      int
	properties   int
	texts        int
	subtrees     int
	fragments    int
	traversals   int
	subtreeBytes int
}

// Traversal is a bridge crossing too, and counting only creation hid it.
//
// bindSerializedSubtree zips the fiber tree against the parsed DOM by walking
// it — one GetFirstChild or GetNextSibling per node. Those are Get calls across
// the same boundary as CreateElement, so a serialized mount does not cost the
// one parse plus one append an earlier count reported: it costs that plus a
// walk proportional to the node count.
func (parseA *countingDOMAdapter) GetFirstChild(parseNode runtime.DOMNode) runtime.DOMNode {
	parseA.traversals++
	return parseA.MockDOMAdapter.GetFirstChild(parseNode)
}

func (parseA *countingDOMAdapter) GetNextSibling(parseNode runtime.DOMNode) runtime.DOMNode {
	parseA.traversals++
	return parseA.MockDOMAdapter.GetNextSibling(parseNode)
}

func (parseA *countingDOMAdapter) GetProperty(parseNode runtime.DOMNode, parseName string) any {
	parseA.traversals++
	return parseA.MockDOMAdapter.GetProperty(parseNode, parseName)
}

func (parseA *countingDOMAdapter) CreateElement(parseTag string) runtime.DOMNode {
	parseA.creates++
	return parseA.MockDOMAdapter.CreateElement(parseTag)
}

func (parseA *countingDOMAdapter) CreateTextNode(parseText string) runtime.DOMNode {
	parseA.texts++
	return parseA.MockDOMAdapter.CreateTextNode(parseText)
}

func (parseA *countingDOMAdapter) SetAttribute(parseNode runtime.DOMNode, parseName, parseValue string) {
	parseA.attrs++
	parseA.MockDOMAdapter.SetAttribute(parseNode, parseName, parseValue)
}

func (parseA *countingDOMAdapter) SetProperty(parseNode runtime.DOMNode, parseName string, parseValue any) {
	parseA.properties++
	parseA.MockDOMAdapter.SetProperty(parseNode, parseName, parseValue)
}

func (parseA *countingDOMAdapter) AppendChild(parseParent, parseChild runtime.DOMNode) {
	parseA.appends++
	parseA.MockDOMAdapter.AppendChild(parseParent, parseChild)
}

func (parseA *countingDOMAdapter) CreateHTMLSubtree(parseHTML string) runtime.DOMNode {
	parseA.subtrees++
	parseA.subtreeBytes += len(parseHTML)
	return parseA.MockDOMAdapter.CreateHTMLSubtree(parseHTML)
}

func (parseA *countingDOMAdapter) CreateHTMLFragment(parseHTML string) runtime.DOMNode {
	parseA.fragments++
	parseA.subtreeBytes += len(parseHTML)
	return parseA.MockDOMAdapter.CreateHTMLFragment(parseHTML)
}

func (parseA *countingDOMAdapter) total() int {
	return parseA.creates + parseA.texts + parseA.attrs + parseA.properties +
		parseA.appends + parseA.subtrees + parseA.fragments + parseA.traversals
}

// perNodeDOMAdapter forces the per-node commit path.
//
// It embeds the INTERFACE rather than the concrete mock, so only the 19
// DOMAdapter methods are promoted; CreateHTMLSubtree and CreateHTMLFragment are
// not part of that interface and therefore do not exist on this type. The
// runtime's capability assertions fail and it mounts node by node — which makes
// this the reference implementation to compare a batched mount against.
type perNodeDOMAdapter struct {
	runtime.DOMAdapter
}

// mockTreeSignature serializes a mock subtree to a stable string.
//
// Two normalizations, both about the mock rather than the DOM:
//
// Attributes are sorted because the map lane builds its attribute slice by
// ranging a Go map, so source order is randomized per run; the DOM it produces
// is what must match, not the order the attributes were written in.
//
// Text is flattened. The mock stores a lone text child two different ways: the
// per-node path sets TextContent on the element (the runtime's direct-text
// optimization), while a parsed subtree gets a real "#text" child node. In a
// browser both are one text node, so comparing the raw shapes would report a
// difference that does not exist in the DOM being measured.
func mockTreeSignature(parseNode *MockDOMNode, parseBuilder *strings.Builder) {
	if parseNode == nil {
		parseBuilder.WriteString("<nil>")
		return
	}
	if parseNode.Tag == "#text" {
		parseBuilder.WriteString(parseNode.TextContent)
		return
	}
	parseBuilder.WriteByte('<')
	parseBuilder.WriteString(parseNode.Tag)
	parseNames := make([]string, 0, len(parseNode.Attrs))
	for parseName := range parseNode.Attrs {
		parseNames = append(parseNames, parseName)
	}
	sort.Strings(parseNames)
	for _, parseName := range parseNames {
		fmt.Fprintf(parseBuilder, " %s=%q", parseName, parseNode.Attrs[parseName])
	}
	parseBuilder.WriteByte('>')
	if parseNode.TextContent != "" {
		parseBuilder.WriteString(parseNode.TextContent)
	}
	for _, parseChild := range parseNode.Children {
		mockTreeSignature(parseChild, parseBuilder)
	}
	parseBuilder.WriteString("</")
	parseBuilder.WriteString(parseNode.Tag)
	parseBuilder.WriteByte('>')
}

// describeSignatureDiff reports the first divergence with a window of context.
//
// These signatures run to tens of kilobytes; printing both in full buries the
// one byte that differs under a screen of identical markup.
func describeSignatureDiff(parseLeft, parseRight string) string {
	parseIndex := 0
	for parseIndex < len(parseLeft) && parseIndex < len(parseRight) && parseLeft[parseIndex] == parseRight[parseIndex] {
		parseIndex++
	}
	parseWindow := func(parseText string) string {
		parseStart := max(parseIndex-40, 0)
		parseEnd := min(parseIndex+80, len(parseText))
		return parseText[parseStart:parseEnd]
	}
	return fmt.Sprintf("first difference at byte %d of %d/%d\n  batched:  ...%s...\n  per-node: ...%s...",
		parseIndex, len(parseLeft), len(parseRight), parseWindow(parseLeft), parseWindow(parseRight))
}

func signatureOf(parseNode runtime.DOMNode) string {
	var parseBuilder strings.Builder
	parseMock, _ := parseNode.(*MockDOMNode)
	mockTreeSignature(parseMock, &parseBuilder)
	return parseBuilder.String()
}

// buildContentCardTree approximates one render-benchmark content card: an
// article wrapping a header row, two paragraphs, and a tag strip.
//
// Built through CreateElement with a props MAP, not as raw Element literals.
// That distinction is the whole point: a raw literal never runs
// buildElementHostProps, so it is not compact, and measuring one would report
// the per-node path no matter what the commit strategy does. This is the shape
// html.Article(html.Props{Class: ...}, ...) actually produces.
func buildContentCardTree(parseCards int, parseTags int) []any {
	parseOut := make([]any, 0, parseCards)
	for parseCard := range parseCards {
		parseTagNodes := make([]any, 0, parseTags)
		for parseTag := range parseTags {
			parseTagNodes = append(parseTagNodes, runtime.CreateElement("span", map[string]any{
				"class": "benchmark-content-tag rounded-full border border-white/10 px-2 py-1 text-[11px] uppercase tracking-[0.14em] text-slate-300",
			}, fmt.Sprintf("tag-%d", parseTag)))
		}
		parseOut = append(parseOut, runtime.CreateElement("article", map[string]any{
			"class": "benchmark-content-card rounded-2xl border border-white/10 bg-white/[0.04] p-4 shadow-lg shadow-black/20",
			"key":   fmt.Sprintf("card-%d", parseCard),
		},
			runtime.CreateElement("div", map[string]any{"class": "flex items-center justify-between gap-3"},
				runtime.CreateElement("h2", map[string]any{"class": "benchmark-content-title text-base font-semibold text-white"}, "title"),
				runtime.CreateElement("span", map[string]any{"class": "benchmark-content-status rounded-full border border-cyan-400/20 bg-cyan-400/10 px-2 py-1"}, "status"),
			),
			runtime.CreateElement("p", map[string]any{"class": "mt-3 text-sm leading-6 text-slate-300"}, "summary"),
			runtime.CreateElement("p", map[string]any{"class": "benchmark-content-meta mt-3 text-xs uppercase"}, "meta"),
			runtime.CreateElement("div", map[string]any{"class": "mt-4 flex flex-wrap gap-2"}, parseTagNodes...),
		))
	}
	return parseOut
}

// TestDOMBoundaryCrossingsAreTheInitialRenderCost quantifies the crossings and
// guards against them growing.
func TestDOMBoundaryCrossingsAreTheInitialRenderCost(parseT *testing.T) {
	parseAdapter := &countingDOMAdapter{MockDOMAdapter: NewMockDOMAdapter()}
	parseRT := runtime.NewRuntime(runtime.Config{DOMAdapter: parseAdapter})

	parseContainer := parseAdapter.MockDOMAdapter.CreateElement("section")
	parseRT.Render(runtime.CreateElement("div", nil, buildContentCardTree(12, 3)...), parseContainer)

	parseT.Logf("content-render (12 cards x 3 tags): creates=%d texts=%d attrs=%d props=%d appends=%d subtrees=%d fragments=%d traversals=%d (%d serialized bytes)  TOTAL=%d",
		parseAdapter.creates, parseAdapter.texts, parseAdapter.attrs,
		parseAdapter.properties, parseAdapter.appends, parseAdapter.subtrees,
		parseAdapter.fragments, parseAdapter.traversals, parseAdapter.subtreeBytes,
		parseAdapter.total())

	// A guard, not a target.
	//
	// The number moved twice, and both moves were corrections rather than
	// changes to the code being measured. It read 374 while the count was taken
	// against runtime's testDOMAdapter, which has no CreateHTMLSubtree and so
	// measured the fallback path. It then read 2 while the count ignored
	// traversal — but bindSerializedSubtree walks the parsed tree with
	// GetFirstChild and GetNextSibling, and those cross the same boundary
	// CreateElement does.
	//
	// Counting traversal, a serialized mount of this scenario costs one parse,
	// one append, and a walk proportional to the node count: about 159. The
	// per-node path costs roughly three crossings per node. The advantage is
	// real but far smaller than a creation-only count suggests.
	if parseAdapter.total() > 200 {
		parseT.Errorf("content-render now costs %d boundary crossings; the commit fell back to per-node mounting",
			parseAdapter.total())
	}
	if parseAdapter.subtrees+parseAdapter.fragments == 0 {
		parseT.Error("no subtree was serialized; the batched mount did not fire at all")
	}
}

// TestBatchedMountMatchesPerNodeMount is the correctness half, and the reason
// the crossing count above is trustworthy rather than merely low.
//
// A batched mount that produced different DOM would be faster and wrong, and no
// crossing count would notice. So the same tree is rendered twice — once
// through an adapter with the serialized capabilities, once through one without
// — and the resulting DOM must be identical.
func TestBatchedMountMatchesPerNodeMount(parseT *testing.T) {
	parseBatchedAdapter := &countingDOMAdapter{MockDOMAdapter: NewMockDOMAdapter()}
	parseBatchedContainer := parseBatchedAdapter.MockDOMAdapter.CreateElement("section")
	runtime.NewRuntime(runtime.Config{DOMAdapter: parseBatchedAdapter}).
		Render(runtime.CreateElement("div", nil, buildContentCardTree(12, 3)...), parseBatchedContainer)

	parsePerNodeMock := NewMockDOMAdapter()
	parsePerNodeAdapter := &perNodeDOMAdapter{DOMAdapter: parsePerNodeMock}
	parsePerNodeContainer := parsePerNodeMock.CreateElement("section")
	runtime.NewRuntime(runtime.Config{DOMAdapter: parsePerNodeAdapter}).
		Render(runtime.CreateElement("div", nil, buildContentCardTree(12, 3)...), parsePerNodeContainer)

	if parseBatchedAdapter.subtrees+parseBatchedAdapter.fragments == 0 {
		parseT.Fatal("the batched render did not serialize anything, so this compares two identical per-node mounts and proves nothing")
	}

	parseBatched := signatureOf(parseBatchedContainer)
	parsePerNode := signatureOf(parsePerNodeContainer)
	if parseBatched != parsePerNode {
		parseT.Errorf("batched and per-node mounts produced different DOM\n%s",
			describeSignatureDiff(parseBatched, parsePerNode))
	}
}

// TestUnsafeAttributesFallBackToPerNodeMount pins the divergence guard.
//
// The serializer writes attributes through the SSR writer, which DROPS names it
// rejects and REWRITES url-bearing values. Both are correct for SSR and wrong
// for a mount: the per-node path calls SetAttribute with the raw name and
// value, so a subtree containing either case must not take the serialized path,
// or identical markup would land in the DOM differently depending on host count
// and sibling grouping. Map-lane attribute names come from user maps, so this
// is reachable input.
func TestUnsafeAttributesFallBackToPerNodeMount(parseT *testing.T) {
	parseCases := []struct {
		name  string
		props map[string]any
	}{
		{"rejected attribute name", map[string]any{"class": "a", "bad name": "x"}},
		{"rewritten url value", map[string]any{"class": "a", "href": "javascript:alert(1)"}},
	}

	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			parseBatchedAdapter := &countingDOMAdapter{MockDOMAdapter: NewMockDOMAdapter()}
			parseBatchedContainer := parseBatchedAdapter.MockDOMAdapter.CreateElement("section")
			runtime.NewRuntime(runtime.Config{DOMAdapter: parseBatchedAdapter}).Render(
				runtime.CreateElement("div", nil,
					runtime.CreateElement("a", parseCase.props,
						runtime.CreateElement("span", map[string]any{"class": "b"}, "one"),
						runtime.CreateElement("span", map[string]any{"class": "c"}, "two"),
					)), parseBatchedContainer)

			parsePerNodeMock := NewMockDOMAdapter()
			parsePerNodeContainer := parsePerNodeMock.CreateElement("section")
			runtime.NewRuntime(runtime.Config{DOMAdapter: &perNodeDOMAdapter{DOMAdapter: parsePerNodeMock}}).Render(
				runtime.CreateElement("div", nil,
					runtime.CreateElement("a", parseCase.props,
						runtime.CreateElement("span", map[string]any{"class": "b"}, "one"),
						runtime.CreateElement("span", map[string]any{"class": "c"}, "two"),
					)), parsePerNodeContainer)

			if parseBatched, parsePerNode := signatureOf(parseBatchedContainer), signatureOf(parsePerNodeContainer); parseBatched != parsePerNode {
				parseT.Errorf("serialized mount changed the DOM for %s\n%s",
					parseCase.name, describeSignatureDiff(parseBatched, parsePerNode))
			}
		})
	}
}

// TestSerializedMountSurvivesParserNormalization is the correctness limit of
// mounting from an HTML string.
//
// bindSerializedSubtree zips fiber children against parsed DOM children by
// POSITION, on the assumption that the parser returns exactly the structure the
// serializer emitted. An HTML parser does not promise that. It promises to
// produce a conforming tree, and it will insert, move, or drop nodes to get
// one — <tr> outside a section element gets an implicit <tbody> wrapped around
// it, content before a valid <table> child gets hoisted out, and so on.
//
// When that happens the zip pairs each fiber with the wrong node: a <tr> fiber
// binds to the <tbody> the parser invented, its children bind to that tbody's
// children, and every later update writes to the wrong element. Nothing errors —
// the DOM is simply not the one the component described.
//
// So this renders shapes a parser is known to rewrite and requires the batched
// mount to agree with the per-node mount, which is definitionally correct
// because it builds the tree node by node and never consults a parser.
func TestSerializedMountSurvivesParserNormalization(parseT *testing.T) {
	parseCases := []struct {
		name string
		tree func() *runtime.Element
	}{
		{
			// <tr> as a direct child of <table>: the parser inserts <tbody>.
			name: "table rows without an explicit tbody",
			tree: func() *runtime.Element {
				return runtime.CreateElement("table", map[string]any{"class": "t"},
					runtime.CreateElement("tr", map[string]any{"class": "r1"},
						runtime.CreateElement("td", map[string]any{"class": "c1"}, "a"),
						runtime.CreateElement("td", map[string]any{"class": "c2"}, "b"),
					),
					runtime.CreateElement("tr", map[string]any{"class": "r2"},
						runtime.CreateElement("td", map[string]any{"class": "c3"}, "c"),
						runtime.CreateElement("td", map[string]any{"class": "c4"}, "d"),
					),
				)
			},
		},
		{
			// A <p> cannot contain a <div>; the parser closes the <p> early and
			// promotes the div to a sibling.
			name: "block element inside a paragraph",
			tree: func() *runtime.Element {
				return runtime.CreateElement("section", map[string]any{"class": "s"},
					runtime.CreateElement("p", map[string]any{"class": "p1"},
						runtime.CreateElement("div", map[string]any{"class": "d1"}, "inner"),
						runtime.CreateElement("span", map[string]any{"class": "sp"}, "tail"),
					),
				)
			},
		},
		{
			// <li> outside a list, and nested lists, are both commonly reshaped.
			name: "list items with a nested list",
			tree: func() *runtime.Element {
				return runtime.CreateElement("ul", map[string]any{"class": "u"},
					runtime.CreateElement("li", map[string]any{"class": "l1"}, "one"),
					runtime.CreateElement("li", map[string]any{"class": "l2"},
						runtime.CreateElement("ul", map[string]any{"class": "u2"},
							runtime.CreateElement("li", map[string]any{"class": "l3"}, "deep"),
						),
					),
				)
			},
		},
		{
			// <select> may only contain option/optgroup; anything else is dropped
			// or hoisted, which loses a node the fiber tree still expects.
			name: "select with a stray child",
			tree: func() *runtime.Element {
				return runtime.CreateElement("select", map[string]any{"class": "sel"},
					runtime.CreateElement("option", map[string]any{"class": "o1"}, "one"),
					runtime.CreateElement("div", map[string]any{"class": "stray"}, "nope"),
					runtime.CreateElement("option", map[string]any{"class": "o2"}, "two"),
				)
			},
		},
	}

	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			parseBatchedAdapter := &countingDOMAdapter{MockDOMAdapter: NewMockDOMAdapter()}
			parseBatchedContainer := parseBatchedAdapter.MockDOMAdapter.CreateElement("main")
			runtime.NewRuntime(runtime.Config{DOMAdapter: parseBatchedAdapter}).
				Render(parseCase.tree(), parseBatchedContainer)

			parsePerNodeMock := NewMockDOMAdapter()
			parsePerNodeContainer := parsePerNodeMock.CreateElement("main")
			runtime.NewRuntime(runtime.Config{DOMAdapter: &perNodeDOMAdapter{DOMAdapter: parsePerNodeMock}}).
				Render(parseCase.tree(), parsePerNodeContainer)

			parseBatched := signatureOf(parseBatchedContainer)
			parsePerNode := signatureOf(parsePerNodeContainer)
			if parseBatched != parsePerNode {
				parseT.Errorf("serialized mount produced different DOM than per-node mount (serialized %d subtree(s))\n%s",
					parseBatchedAdapter.subtrees+parseBatchedAdapter.fragments,
					describeSignatureDiff(parseBatched, parsePerNode))
			}
		})
	}
}

// TestSerializedMountAgreesAcrossTagCombinations sweeps parent/child tag pairs
// rather than trusting a hand-written list of dangerous shapes.
//
// The blocklist in serializedMountReshapedTags is only as good as the cases
// someone thought of, and HTML's tree-construction rules are long. This renders
// every parent/child combination both ways and requires them to agree, so a tag
// whose content model is enforced by the parser and NOT on the list fails here
// rather than in an application.
//
// The per-node mount is the oracle: it builds the tree node by node and never
// consults a parser, so it produces exactly what the component described by
// construction.
func TestSerializedMountAgreesAcrossTagCombinations(parseT *testing.T) {
	parseParents := []string{
		"div", "span", "section", "article", "header", "footer", "main", "aside",
		"nav", "ul", "ol", "li", "dl", "dt", "dd", "p", "blockquote", "figure",
		"figcaption", "table", "thead", "tbody", "tr", "td", "th", "caption",
		"select", "optgroup", "option", "datalist", "form", "fieldset", "legend",
		"label", "button", "a", "h1", "h2", "pre", "code", "em", "strong", "small",
		"details", "summary", "picture", "video", "audio", "canvas", "template",
		"my-widget", "x-panel",
	}
	parseChildren := []string{"div", "span", "li", "tr", "td", "option", "p", "a", "my-widget"}

	parseMismatches := 0
	for _, parseParent := range parseParents {
		for _, parseChild := range parseChildren {
			parseName := parseParent + ">" + parseChild
			parseBuild := func() *runtime.Element {
				return runtime.CreateElement("div", map[string]any{"class": "host"},
					runtime.CreateElement(parseParent, map[string]any{"class": "p"},
						runtime.CreateElement(parseChild, map[string]any{"class": "a"}, "one"),
						runtime.CreateElement(parseChild, map[string]any{"class": "b"}, "two"),
					),
				)
			}

			parseBatchedAdapter := &countingDOMAdapter{MockDOMAdapter: NewMockDOMAdapter()}
			parseBatchedContainer := parseBatchedAdapter.MockDOMAdapter.CreateElement("main")
			runtime.NewRuntime(runtime.Config{DOMAdapter: parseBatchedAdapter}).
				Render(parseBuild(), parseBatchedContainer)

			parsePerNodeMock := NewMockDOMAdapter()
			parsePerNodeContainer := parsePerNodeMock.CreateElement("main")
			runtime.NewRuntime(runtime.Config{DOMAdapter: &perNodeDOMAdapter{DOMAdapter: parsePerNodeMock}}).
				Render(parseBuild(), parsePerNodeContainer)

			parseBatched := signatureOf(parseBatchedContainer)
			parsePerNode := signatureOf(parsePerNodeContainer)
			if parseBatched != parsePerNode {
				parseMismatches++
				parseT.Errorf("%s: serialized mount disagrees with per-node mount\n%s",
					parseName, describeSignatureDiff(parseBatched, parsePerNode))
			}
		}
	}
	parseT.Logf("swept %d parent/child combinations, %d mismatches",
		len(parseParents)*len(parseChildren), parseMismatches)
}
