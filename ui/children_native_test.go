//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// renderChildrenText renders a normalized child slice inside a <div> and returns
// the host text, so we can assert what survived normalization.
func renderChildrenText(t *testing.T, parseNodes []ui.Node) string {
	t.Helper()
	parseOut, parseErr := ui.RenderToString(html.Div(html.Props{}, parseNodes...))
	if parseErr != nil {
		t.Fatalf("render error: %v", parseErr)
	}
	return parseOut
}

// Ported from React's ReactChildren-test (the toArray/flatten/filter spirit).
// GWC's html.Children normalizes a variadic of mixed inputs into []ui.Node:
// nested slices flatten recursively, nil is filtered out, and strings/Stringers
// become text nodes. (Architectural note: GWC has no Children.map/only/forEach
// API — normalization is the analog; and a bare bool becomes its text form
// rather than being dropped like React, since GWC children are typed ui.Node in
// normal use.)
func TestChildrenFlattenAndFilter(t *testing.T) {
	// Flattening: deeply nested slices collapse to a flat node list.
	parseNested := html.Children(
		ui.Text("a"),
		[]ui.Node{html.Span(html.Props{}, ui.Text("b")), html.Span(html.Props{}, ui.Text("c"))},
		[]any{ui.Text("d"), []ui.Node{ui.Text("e")}},
	)
	if parseGot := len(parseNested); parseGot != 5 {
		t.Errorf("nested flatten: got %d nodes, want 5", parseGot)
	}

	// nil is filtered out entirely.
	parseWithNil := html.Children(ui.Text("a"), nil, ui.Text("b"), nil)
	if parseGot := len(parseWithNil); parseGot != 2 {
		t.Errorf("nil filtering: got %d nodes, want 2", parseGot)
	}

	// Empty / all-nil normalizes to a nil slice.
	if parseEmpty := html.Children(); parseEmpty != nil {
		t.Errorf("empty Children: got %v, want nil", parseEmpty)
	}
	if parseAllNil := html.Children(nil, nil); parseAllNil != nil {
		t.Errorf("all-nil Children: got %v, want nil", parseAllNil)
	}

	// Strings are converted to text nodes and rendered (with escaping).
	parseStrings := html.Children("x", []string{"y", "z"})
	if parseGot := len(parseStrings); parseGot != 3 {
		t.Fatalf("string normalization: got %d nodes, want 3", parseGot)
	}
	if parseOut := renderChildrenText(t, parseStrings); parseOut != "<div>xyz</div>" {
		t.Errorf("string children rendered = %q, want <div>xyz</div>", parseOut)
	}
}

// TestChildrenPreservesOrder confirms normalization keeps document order across
// the flatten (a mis-ordered flatten would scramble rendered output).
func TestChildrenPreservesOrder(t *testing.T) {
	parseNodes := html.Children(
		ui.Text("1"),
		[]ui.Node{ui.Text("2"), ui.Text("3")},
		ui.Text("4"),
	)
	if parseOut := renderChildrenText(t, parseNodes); parseOut != "<div>1234</div>" {
		t.Errorf("order after flatten = %q, want <div>1234</div>", parseOut)
	}
}
