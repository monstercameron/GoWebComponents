package html

import (
	"fmt"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

// TestMapIndexedThreadsIndex verifies MapIndexed passes the element index and
// preserves order, and that an empty slice renders nothing.
func TestMapIndexedThreadsIndex(parseT *testing.T) {
	parseNodes := MapIndexed([]string{"a", "b", "c"}, func(parseIndex int, parseItem string) ui.Node {
		return Li(Props{}, Text(fmt.Sprintf("%d:%s", parseIndex, parseItem)))
	})

	parseMarkup, parseErr := ui.RenderToString(Ul(Props{}, parseNodes...))
	if parseErr != nil {
		parseT.Fatalf("expected MapIndexed SSR render, got %v", parseErr)
	}
	if parseMarkup != "<ul><li>0:a</li><li>1:b</li><li>2:c</li></ul>" {
		parseT.Fatalf("expected indexed order, got %q", parseMarkup)
	}

	if MapIndexed([]string{}, func(parseIndex int, parseItem string) ui.Node { return Text(parseItem) }) != nil {
		parseT.Fatal("expected empty MapIndexed result to be nil")
	}
}

// TestMapKeyedIndexedThreadsIndexAndKey verifies the index reaches both the key
// builder and the render callback.
func TestMapKeyedIndexedThreadsIndexAndKey(parseT *testing.T) {
	parseNodes := MapKeyedIndexed([]string{"alpha", "beta"},
		func(parseIndex int, parseItem string) any { return fmt.Sprintf("k%d:%s", parseIndex, parseItem) },
		func(parseIndex int, parseItem string) ui.Node { return Li(Props{}, Textf("%d-%s", parseIndex, parseItem)) },
	)
	if len(parseNodes) != 2 {
		parseT.Fatalf("expected 2 keyed nodes, got %d", len(parseNodes))
	}
	if parseNodes[0].Props["key"] != "k0:alpha" || parseNodes[1].Props["key"] != "k1:beta" {
		parseT.Fatalf("expected indexed keys, got %#v / %#v", parseNodes[0].Props["key"], parseNodes[1].Props["key"])
	}
	parseMarkup, parseRenderErr := ui.RenderToString(Ul(Props{}, parseNodes...))
	if parseRenderErr != nil {
		parseT.Fatalf("expected MapKeyedIndexed render, got %v", parseRenderErr)
	}
	if !strings.Contains(parseMarkup, "<li>0-alpha</li><li>1-beta</li>") {
		parseT.Fatalf("expected indexed render, got %q", parseMarkup)
	}

	if MapKeyedIndexed([]int{}, func(parseIndex int, parseItem int) any { return parseIndex }, func(parseIndex int, parseItem int) ui.Node { return Textf("%d", parseItem) }) != nil {
		parseT.Fatal("expected empty MapKeyedIndexed result to be nil")
	}
}

// TestMapOrRendersListOrFallback verifies MapOr renders the mapped list when
// items exist and exactly the fallback node when the slice is empty or nil.
func TestMapOrRendersListOrFallback(parseT *testing.T) {
	parseFilled := MapOr([]int{1, 2}, func(parseValue int) ui.Node {
		return Li(Props{}, Textf("n%d", parseValue))
	}, P(Props{}, Text("empty")))
	parseFilledMarkup, parseErr := ui.RenderToString(Ul(Props{}, parseFilled))
	if parseErr != nil {
		parseT.Fatalf("expected MapOr filled render, got %v", parseErr)
	}
	if !strings.Contains(parseFilledMarkup, "<li>n1</li><li>n2</li>") || strings.Contains(parseFilledMarkup, "empty") {
		parseT.Fatalf("expected mapped list and no fallback, got %q", parseFilledMarkup)
	}

	for _, parseEmpty := range [][]int{{}, nil} {
		parseFallback := MapOr(parseEmpty, func(parseValue int) ui.Node { return Textf("n%d", parseValue) }, P(Props{}, Text("empty")))
		parseMarkup, parseErr2 := ui.RenderToString(Div(Props{}, parseFallback))
		if parseErr2 != nil {
			parseT.Fatalf("expected MapOr fallback render, got %v", parseErr2)
		}
		if parseMarkup != "<div><p>empty</p></div>" {
			parseT.Fatalf("expected exactly the fallback for empty/nil, got %q", parseMarkup)
		}
	}
}

// TestRangeAndRepeat verifies count-based rendering and non-positive guards.
func TestRangeAndRepeat(parseT *testing.T) {
	parseRanged := Range(3, func(parseIndex int) ui.Node { return Li(Props{}, Textf("i%d", parseIndex)) })
	parseMarkup, parseErr := ui.RenderToString(Ul(Props{}, parseRanged...))
	if parseErr != nil {
		parseT.Fatalf("expected Range render, got %v", parseErr)
	}
	if parseMarkup != "<ul><li>i0</li><li>i1</li><li>i2</li></ul>" {
		parseT.Fatalf("expected ranged indices, got %q", parseMarkup)
	}
	if Range(0, func(parseIndex int) ui.Node { return Text("x") }) != nil {
		parseT.Fatal("expected Range(0) to be nil")
	}
	if Range(-2, func(parseIndex int) ui.Node { return Text("x") }) != nil {
		parseT.Fatal("expected Range(negative) to be nil")
	}

	parseRepeated := Repeat(2, Li(Props{}, Text("dot")))
	parseRepMarkup, parseErr2 := ui.RenderToString(Ul(Props{}, parseRepeated...))
	if parseErr2 != nil {
		parseT.Fatalf("expected Repeat render, got %v", parseErr2)
	}
	if parseRepMarkup != "<ul><li>dot</li><li>dot</li></ul>" {
		parseT.Fatalf("expected node repeated twice, got %q", parseRepMarkup)
	}
	if Repeat(0, Text("x")) != nil || Repeat(-1, Text("x")) != nil {
		parseT.Fatal("expected non-positive Repeat to be nil")
	}
}

// TestMaybeOrRendersValueOrFallback verifies MaybeOr dereferences a present
// pointer and otherwise renders the fallback without a nil deref.
func TestMaybeOrRendersValueOrFallback(parseT *testing.T) {
	parseValue := "present"
	parsePresent := MaybeOr(&parseValue, func(parseInner string) ui.Node { return Text(parseInner) }, Text("fallback"))
	parsePresentMarkup, parseErr := ui.RenderToString(Div(Props{}, parsePresent))
	if parseErr != nil {
		parseT.Fatalf("expected MaybeOr present render, got %v", parseErr)
	}
	if parsePresentMarkup != "<div>present</div>" {
		parseT.Fatalf("expected dereferenced value, got %q", parsePresentMarkup)
	}

	parseAbsent := MaybeOr[string](nil, func(parseInner string) ui.Node { return Text(parseInner) }, Text("fallback"))
	parseAbsentMarkup, parseErr2 := ui.RenderToString(Div(Props{}, parseAbsent))
	if parseErr2 != nil {
		parseT.Fatalf("expected MaybeOr absent render, got %v", parseErr2)
	}
	if parseAbsentMarkup != "<div>fallback</div>" {
		parseT.Fatalf("expected fallback for nil pointer, got %q", parseAbsentMarkup)
	}
}

// TestCondFirstTrueWins verifies Cond selects the first true branch, takes
// Otherwise only when no condition matches, and returns nil with neither.
func TestCondFirstTrueWins(parseT *testing.T) {
	// First true branch wins even though a later branch is also true.
	parseFirst := Cond(
		Match(false, Text("a")),
		Match(true, Text("b")),
		Match(true, Text("c")),
		Otherwise(Text("default")),
	)
	if parseFirst.TextContent != "b" {
		parseT.Fatalf("expected first true branch (b), got %#v", parseFirst)
	}

	// No condition matches -> Otherwise.
	parseDefault := Cond(
		Match(false, Text("a")),
		Match(false, Text("b")),
		Otherwise(Text("default")),
	)
	if parseDefault.TextContent != "default" {
		parseT.Fatalf("expected Otherwise branch, got %#v", parseDefault)
	}

	// No match and no Otherwise -> nil.
	if Cond(Match(false, Text("a")), Match(false, Text("b"))) != nil {
		parseT.Fatal("expected nil when no branch matches and no Otherwise is given")
	}

	// A matching condition wins over Otherwise regardless of branch order.
	parseOrder := Cond(
		Otherwise(Text("default")),
		Match(true, Text("hit")),
	)
	if parseOrder.TextContent != "hit" {
		parseT.Fatalf("expected matched branch over Otherwise, got %#v", parseOrder)
	}
}
