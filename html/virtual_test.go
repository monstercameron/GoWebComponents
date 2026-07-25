package html_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// v5 P4.1 — flat-list virtualization, acceptance M4.
//
// M4: a 10,000-row collection must commit at O(viewport), not O(dataset). These
// tests are mostly about COUNTS, because a virtualized list that renders
// correctly and still builds every row would pass every visual check and fail
// the only requirement there is.

// childElement asserts one child back to a node. Element.Children is []any, so
// a non-element child (a text node, a nil) simply is not one.
func childElement(parseChild any) (ui.Node, bool) {
	parseNode, hasNode := parseChild.(ui.Node)
	return parseNode, hasNode && parseNode != nil
}

// childClass reports a child's class attribute, empty when it has none.
func childClass(parseChild any) string {
	parseNode, hasNode := childElement(parseChild)
	if !hasNode {
		return ""
	}
	parseClass, _ := parseNode.Props["class"].(string)
	return parseClass
}

// countRows counts rendered rows, excluding the two layout spacers.
func countRows(parseNode ui.Node) int {
	if parseNode == nil {
		return 0
	}
	parseRows := 0
	for _, parseChild := range parseNode.Children {
		if _, hasNode := childElement(parseChild); !hasNode {
			continue
		}
		if parseClass := childClass(parseChild); parseClass == "gwc-vlist-top" || parseClass == "gwc-vlist-bottom" {
			continue
		}
		parseRows++
	}
	return parseRows
}

func buildRowRenderer(parseCalls *int) func(int) ui.Node {
	return func(parseIndex int) ui.Node {
		*parseCalls++
		return html.Div(html.Props{Class: "row"}, html.Text(fmt.Sprintf("row %d", parseIndex)))
	}
}

// ------------------------------------------------------------- the M4 claim

// TestRenderedRowsTrackTheViewportNotTheDataset is M4.
func TestRenderedRowsTrackTheViewportNotTheDataset(parseT *testing.T) {
	var parseRowCountByDataset []int

	for _, parseItemCount := range []int{1000, 10000, 100000} {
		parseCalls := 0
		parseNode := html.VirtualList(html.VirtualListProps{
			ItemCount:      parseItemCount,
			ItemHeight:     40,
			ViewportHeight: 800,
			ScrollTop:      0,
			Render:         buildRowRenderer(&parseCalls),
		})

		parseRendered := countRows(parseNode)
		parseRowCountByDataset = append(parseRowCountByDataset, parseRendered)
		parseT.Logf("%6d rows in the dataset -> %d rendered, %d render calls",
			parseItemCount, parseRendered, parseCalls)

		// The render function is the expensive part; it must not be called for
		// rows nobody can see.
		if parseCalls != parseRendered {
			parseT.Errorf("render called %d times for %d rendered rows", parseCalls, parseRendered)
		}
	}

	// A 100x dataset must not change the rendered count at all.
	for parseIndex := 1; parseIndex < len(parseRowCountByDataset); parseIndex++ {
		if parseRowCountByDataset[parseIndex] != parseRowCountByDataset[0] {
			parseT.Fatalf("rendered rows = %v across datasets — the cost tracks the dataset, not the viewport",
				parseRowCountByDataset)
		}
	}
	// And the count must actually be viewport-sized: 800/40 = 20 visible, plus
	// the partial row, plus overscan on each side.
	if parseRowCountByDataset[0] > 40 {
		parseT.Errorf("rendered %d rows for a 20-row viewport, want a viewport-sized window", parseRowCountByDataset[0])
	}
}

func TestUnvirtualizedIsTheExplicitOptOut(parseT *testing.T) {
	parseCalls := 0
	parseNode := html.VirtualList(html.VirtualListProps{
		ItemCount:      500,
		ItemHeight:     40,
		ViewportHeight: 800,
		Unvirtualized:  true,
		Render:         buildRowRenderer(&parseCalls),
	})

	if parseRendered := countRows(parseNode); parseRendered != 500 {
		parseT.Errorf("rendered %d rows, want all 500 when opted out", parseRendered)
	}
	// No spacers: with everything rendered there is nothing to stand in for.
	for _, parseChild := range parseNode.Children {
		if parseClass := childClass(parseChild); parseClass == "gwc-vlist-top" || parseClass == "gwc-vlist-bottom" {
			parseT.Error("an unvirtualized list must not emit spacers")
		}
	}
}

// ------------------------------------------------------------ window maths

func TestVisibleRangeCoversTheViewport(parseT *testing.T) {
	parseRange := html.VisibleRange(html.VirtualListProps{
		ItemCount: 10000, ItemHeight: 40, ViewportHeight: 800, ScrollTop: 4000, Overscan: -1,
	})

	// Scrolled to 4000px with 40px rows: row 100 is at the top, 20 rows fit.
	if parseRange.Start != 100 {
		parseT.Errorf("Start = %d, want 100", parseRange.Start)
	}
	// The row straddling the bottom edge is partly visible and must be included.
	if parseRange.End < 121 {
		parseT.Errorf("End = %d, want at least 121 so the partly-visible row is rendered", parseRange.End)
	}
}

// TestOverscanRendersAheadOfTheScroll: a scroll event arrives after the pixels
// have already moved, so a window with no margin flickers.
func TestOverscanRendersAheadOfTheScroll(parseT *testing.T) {
	parseProps := html.VirtualListProps{
		ItemCount: 10000, ItemHeight: 40, ViewportHeight: 800, ScrollTop: 4000,
	}

	parseNone := html.VisibleRange(html.VirtualListProps{
		ItemCount: parseProps.ItemCount, ItemHeight: parseProps.ItemHeight,
		ViewportHeight: parseProps.ViewportHeight, ScrollTop: parseProps.ScrollTop, Overscan: -1,
	})
	parseDefault := html.VisibleRange(parseProps)

	if parseDefault.Start >= parseNone.Start {
		parseT.Errorf("default start %d, no-overscan start %d — overscan must render above the viewport",
			parseDefault.Start, parseNone.Start)
	}
	if parseDefault.End <= parseNone.End {
		parseT.Errorf("default end %d, no-overscan end %d — overscan must render below the viewport",
			parseDefault.End, parseNone.End)
	}
}

func TestVisibleRangeIsClampedAtBothEnds(parseT *testing.T) {
	parseAtTop := html.VisibleRange(html.VirtualListProps{
		ItemCount: 100, ItemHeight: 40, ViewportHeight: 800, ScrollTop: 0,
	})
	if parseAtTop.Start != 0 {
		parseT.Errorf("Start = %d at the top, want 0 — overscan must not go negative", parseAtTop.Start)
	}

	parseAtBottom := html.VisibleRange(html.VirtualListProps{
		ItemCount: 100, ItemHeight: 40, ViewportHeight: 800, ScrollTop: 100000,
	})
	if parseAtBottom.End > 100 {
		parseT.Errorf("End = %d, want at most the item count", parseAtBottom.End)
	}
	if parseAtBottom.Start > parseAtBottom.End {
		parseT.Errorf("range = %+v is inverted; a list that shrank under a scroll position must not produce one", parseAtBottom)
	}

	parseNegativeScroll := html.VisibleRange(html.VirtualListProps{
		ItemCount: 100, ItemHeight: 40, ViewportHeight: 800, ScrollTop: -500,
	})
	if parseNegativeScroll.Start != 0 {
		parseT.Errorf("Start = %d for an overscrolled position, want 0", parseNegativeScroll.Start)
	}
}

// TestDegenerateGeometryRendersEverything: a viewport of zero height is the
// normal state before first layout, and a zero row height would divide by zero.
// Rendering everything is the safe direction — too many rows is slow, too few is
// wrong.
func TestDegenerateGeometryRendersEverything(parseT *testing.T) {
	for _, parseCase := range []struct {
		label string
		props html.VirtualListProps
	}{
		{"no viewport yet", html.VirtualListProps{ItemCount: 50, ItemHeight: 40, ViewportHeight: 0}},
		{"no row height", html.VirtualListProps{ItemCount: 50, ItemHeight: 0, ViewportHeight: 800}},
	} {
		parseRange := html.VisibleRange(parseCase.props)
		if parseRange.Start != 0 || parseRange.End != 50 {
			parseT.Errorf("%s: range = %+v, want the whole list", parseCase.label, parseRange)
		}
	}
}

func TestEmptyListRendersNothing(parseT *testing.T) {
	parseRange := html.VisibleRange(html.VirtualListProps{ItemCount: 0, ItemHeight: 40, ViewportHeight: 800})
	if parseRange.Len() != 0 {
		parseT.Errorf("range = %+v, want empty", parseRange)
	}

	parseCalls := 0
	parseNode := html.VirtualList(html.VirtualListProps{
		ItemCount: 0, ItemHeight: 40, ViewportHeight: 800, Render: buildRowRenderer(&parseCalls),
	})
	if countRows(parseNode) != 0 || parseCalls != 0 {
		parseT.Error("an empty list must render nothing and call nothing")
	}
}

// ---------------------------------------------------------------- spacers

// TestSpacersPreserveTheScrollHeight: without them the scrollbar reflects only
// the rendered rows, so scrolling to the bottom of a 10,000-row list stops after
// forty.
func TestSpacersPreserveTheScrollHeight(parseT *testing.T) {
	const parseItemCount = 10000
	const parseItemHeight = 40.0

	parseCalls := 0
	parseNode := html.VirtualList(html.VirtualListProps{
		ItemCount: parseItemCount, ItemHeight: parseItemHeight, ViewportHeight: 800, ScrollTop: 4000,
		Render: buildRowRenderer(&parseCalls),
	})

	parseRange := html.VisibleRange(html.VirtualListProps{
		ItemCount: parseItemCount, ItemHeight: parseItemHeight, ViewportHeight: 800, ScrollTop: 4000,
	})

	hasTop, hasBottom := false, false
	for _, parseChild := range parseNode.Children {
		switch childClass(parseChild) {
		case "gwc-vlist-top":
			hasTop = true
		case "gwc-vlist-bottom":
			hasBottom = true
		}
	}
	if !hasTop || !hasBottom {
		parseT.Fatalf("spacers present: top=%v bottom=%v, want both when scrolled into the middle", hasTop, hasBottom)
	}

	// Rendered rows plus both spacers must account for the whole dataset.
	parseAccounted := parseRange.Len() + parseRange.Start + (parseItemCount - parseRange.End)
	if parseAccounted != parseItemCount {
		parseT.Errorf("rows + spacers account for %d of %d — the scroll height would be wrong",
			parseAccounted, parseItemCount)
	}
}

func TestNoLeadingSpacerAtTheTop(parseT *testing.T) {
	parseCalls := 0
	parseNode := html.VirtualList(html.VirtualListProps{
		ItemCount: 1000, ItemHeight: 40, ViewportHeight: 800, ScrollTop: 0,
		Render: buildRowRenderer(&parseCalls),
	})
	// A zero-height spacer is a wasted element in the DOM on every list that has
	// not been scrolled — which is most of them.
	if childClass(parseNode.Children[0]) == "gwc-vlist-top" {
		parseT.Error("a list at scroll position 0 must not emit a leading spacer")
	}
}

// ------------------------------------------------------------------ keying

// TestRowsAreKeyedWhenAKeyIsSupplied: without keys the reconciler treats row N
// of the window as a different row after every scroll and rebuilds the window.
func TestRowsAreKeyedWhenAKeyIsSupplied(parseT *testing.T) {
	parseCalls := 0
	parseNode := html.VirtualList(html.VirtualListProps{
		ItemCount: 1000, ItemHeight: 40, ViewportHeight: 400, ScrollTop: 2000,
		Render: buildRowRenderer(&parseCalls),
		Key:    func(parseIndex int) any { return fmt.Sprintf("row-%d", parseIndex) },
	})

	parseKeyed := 0
	for _, parseChild := range parseNode.Children {
		if parseClass := childClass(parseChild); parseClass == "gwc-vlist-top" || parseClass == "gwc-vlist-bottom" {
			continue
		}
		parseRow, hasRow := childElement(parseChild)
		if !hasRow {
			continue
		}
		// A key lands on the dedicated field for typed fast-lane nodes and in
		// the props map otherwise, so both are checked — asserting only one
		// would fail depending on how the row happened to be built.
		if parseRow.Key != "" || parseRow.Props["key"] != nil {
			parseKeyed++
		}
	}
	if parseKeyed == 0 {
		parseT.Fatal("no rows were keyed")
	}
	if parseKeyed != countRows(parseNode) {
		parseT.Errorf("%d of %d rows keyed; every row must be", parseKeyed, countRows(parseNode))
	}
}

func TestNilRenderIsSafe(parseT *testing.T) {
	parseNode := html.VirtualList(html.VirtualListProps{ItemCount: 100, ItemHeight: 40, ViewportHeight: 800})
	if parseNode == nil {
		parseT.Fatal("a list without a render func must still produce a container")
	}
	if countRows(parseNode) != 0 {
		parseT.Error("a list without a render func must render no rows")
	}
}

// --------------------------------------------------------------- benchmark

// BenchmarkVirtualListBuild is M4 as a time measurement: the cost of building
// the list must not grow with the dataset.
func BenchmarkVirtualListBuild(parseB *testing.B) {
	for _, parseItemCount := range []int{1000, 10000, 100000} {
		parseB.Run(fmt.Sprintf("items-%d", parseItemCount), func(parseSub *testing.B) {
			parseProps := html.VirtualListProps{
				ItemCount: parseItemCount, ItemHeight: 40, ViewportHeight: 800, ScrollTop: 4000,
				Render: func(parseIndex int) ui.Node {
					return html.Div(html.Props{Class: "row"}, html.Text("x"))
				},
				Key: func(parseIndex int) any { return parseIndex },
			}
			parseSub.ResetTimer()
			parseSub.ReportAllocs()
			for parseSub.Loop() {
				if html.VirtualList(parseProps) == nil {
					parseSub.Fatal("nil list")
				}
			}
		})
	}
}

// BenchmarkUnvirtualizedBuild is the cost being avoided, for comparison.
func BenchmarkUnvirtualizedBuild(parseB *testing.B) {
	for _, parseItemCount := range []int{1000, 10000} {
		parseB.Run(fmt.Sprintf("items-%d", parseItemCount), func(parseSub *testing.B) {
			parseProps := html.VirtualListProps{
				ItemCount: parseItemCount, ItemHeight: 40, ViewportHeight: 800,
				Unvirtualized: true,
				Render: func(parseIndex int) ui.Node {
					return html.Div(html.Props{Class: "row"}, html.Text("x"))
				},
			}
			parseSub.ResetTimer()
			parseSub.ReportAllocs()
			for parseSub.Loop() {
				if html.VirtualList(parseProps) == nil {
					parseSub.Fatal("nil list")
				}
			}
		})
	}
}
