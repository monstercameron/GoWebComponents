package html

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// Flat-list virtualization — plan item P4.1, acceptance M4.
//
// M4 is the metric that says a 10,000-row collection must commit at O(viewport)
// rather than O(dataset). Map renders every item, which is correct for the small
// lists it was written for and quadratically wrong for a data grid: 10,000 rows
// means 10,000 elements built, 10,000 fibers reconciled, and 10,000 DOM nodes
// the browser must lay out — for the forty a person can see.
//
// VirtualList renders the visible window and represents the rest as height.
// Virtualization is the DEFAULT: a caller opts OUT with Unvirtualized, rather
// than opting in. That direction is the point of "default-on" — the expensive
// behaviour should be the one you have to ask for.

// DefaultOverscan is how many rows are rendered beyond the viewport on each side.
//
// Zero would be correct and would flicker: a scroll event arrives after the
// pixels have already moved, so the row entering view must already exist. Six
// rows is roughly one frame of fast scrolling at typical row heights, which is
// the amount that has to be pre-built to avoid a visible gap.
const DefaultOverscan = 6

// VirtualListProps configure a virtualized list.
type VirtualListProps struct {
	// ItemCount is the total number of rows, however few are rendered.
	ItemCount int
	// ItemHeight is the fixed height of one row in CSS pixels.
	//
	// Fixed rather than measured, and that is a real limitation rather than an
	// oversight: variable heights need a measurement pass per row, which is the
	// O(dataset) cost M4 exists to remove. A list of genuinely variable rows
	// should use Unvirtualized and stay small.
	ItemHeight float64
	// ViewportHeight is the visible height of the scroll container.
	ViewportHeight float64
	// ScrollTop is the current scroll offset.
	ScrollTop float64
	// Overscan is how many extra rows to render on each side. Zero uses
	// DefaultOverscan; negative means none.
	Overscan int
	// Render builds one row. It is called only for rows in the rendered window.
	Render func(parseIndex int) ui.Node
	// Key returns a stable identity for a row, so scrolling reuses fibers
	// instead of rebuilding them. Optional but strongly advised: without it the
	// reconciler keys on position, and scrolling by one row re-renders the
	// entire window.
	Key func(parseIndex int) any
	// Unvirtualized renders every row.
	//
	// The explicit opt-out. It exists for lists that are genuinely small, for
	// print or export views where everything must be in the DOM, and for rows
	// whose height cannot be known in advance.
	Unvirtualized bool
	// Props are applied to the scroll container.
	Props Props
}

// RowRange is the half-open span of rows a virtualized list will render.
type RowRange struct {
	Start int
	End   int
}

// Len reports how many rows the range covers.
func (parseRange RowRange) Len() int {
	if parseRange.End <= parseRange.Start {
		return 0
	}
	return parseRange.End - parseRange.Start
}

// VisibleRange computes which rows to render.
//
// Pure and exported because it is the whole of M4: if this returns a span
// proportional to the dataset, nothing downstream can make the commit
// O(viewport). Testing it directly is cheaper and sharper than inferring it from
// a rendered tree.
func VisibleRange(parseProps VirtualListProps) RowRange {
	if parseProps.ItemCount <= 0 {
		return RowRange{}
	}
	if parseProps.Unvirtualized {
		return RowRange{Start: 0, End: parseProps.ItemCount}
	}
	// A degenerate row height would divide by zero, and a viewport of zero
	// height happens routinely before first layout. Falling back to rendering
	// everything is the safe direction: too many rows is slow, too few is wrong.
	if parseProps.ItemHeight <= 0 || parseProps.ViewportHeight <= 0 {
		return RowRange{Start: 0, End: parseProps.ItemCount}
	}

	parseOverscan := parseProps.Overscan
	if parseOverscan == 0 {
		parseOverscan = DefaultOverscan
	}
	if parseOverscan < 0 {
		parseOverscan = 0
	}

	parseScrollTop := max(parseProps.ScrollTop, 0)

	parseFirstVisible := int(parseScrollTop / parseProps.ItemHeight)
	// Ceiling, plus one: a viewport rarely divides evenly into rows, so the row
	// straddling the bottom edge is partly visible and must be rendered.
	parseVisibleCount := int(parseProps.ViewportHeight/parseProps.ItemHeight) + 2

	parseStart := max(parseFirstVisible-parseOverscan, 0)
	parseEnd := min(parseFirstVisible+parseVisibleCount+parseOverscan, parseProps.ItemCount)
	if parseStart > parseEnd {
		// Scrolled past the end, which happens transiently when a list shrinks
		// underneath a scroll position.
		parseStart = parseEnd
	}
	return RowRange{Start: parseStart, End: parseEnd}
}

// VirtualList renders a scrollable list, building only the visible rows.
//
// The rows sit between two spacer elements whose heights stand in for the rows
// that were not built, so the scrollbar reflects the whole dataset and the
// scroll position means what it looks like it means. Absolute positioning would
// work too and would take the rows out of normal flow, which breaks tables and
// anything relying on sibling layout.
func VirtualList(parseProps VirtualListProps) ui.Node {
	if parseProps.Render == nil {
		return Div(parseProps.Props)
	}

	parseRange := VisibleRange(parseProps)
	parseChildren := make([]ui.Node, 0, parseRange.Len()+2)

	parseVirtualized := !parseProps.Unvirtualized && parseProps.ItemHeight > 0 && parseProps.ViewportHeight > 0

	if parseVirtualized && parseRange.Start > 0 {
		parseChildren = append(parseChildren, spacer("gwc-vlist-top", float64(parseRange.Start)*parseProps.ItemHeight))
	}

	for parseIndex := parseRange.Start; parseIndex < parseRange.End; parseIndex++ {
		parseRow := parseProps.Render(parseIndex)
		if parseRow == nil {
			continue
		}
		if parseProps.Key != nil {
			// Keying by identity rather than position is what makes scrolling
			// cheap: without it the reconciler treats row N of the window as a
			// different row after every scroll and rebuilds the whole window.
			parseRow = WithKey(parseRow, parseProps.Key(parseIndex))
		}
		parseChildren = append(parseChildren, parseRow)
	}

	if parseVirtualized {
		parseTrailing := parseProps.ItemCount - parseRange.End
		if parseTrailing > 0 {
			parseChildren = append(parseChildren, spacer("gwc-vlist-bottom", float64(parseTrailing)*parseProps.ItemHeight))
		}
	}

	return Div(parseProps.Props, parseChildren...)
}

// spacer builds a zero-content element that occupies the height of the rows it
// stands in for.
func spacer(parseClass string, parseHeight float64) ui.Node {
	return Div(Props{
		Class: parseClass,
		Style: map[string]string{
			"height": fmt.Sprintf("%.0fpx", parseHeight),
			// Spacers must not shrink under a flex parent, or the scroll height
			// collapses and the scrollbar stops matching the dataset.
			"flex": "0 0 auto",
		},
		// Spacers are layout, not content. Announcing them would put two
		// meaningless entries into every screen reader rendering of the list.
		Aria: map[string]string{"hidden": "true"},
	})
}
