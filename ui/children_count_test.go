//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// Deepens the ReactChildren port toward equivalency for the GWC-applicable subset
// (count / flatten / filter — GWC has html.Children normalization, not the
// Children.map/forEach/count API). Mirrors React's count/flatten cases.
func TestChildrenCountAcrossShapes(t *testing.T) {
	span := func(parseS string) ui.Node { return html.Span(html.Props{}, ui.Text(parseS)) }

	cases := []struct {
		name string
		in   []any
		want int
	}{
		{"nil-only -> 0", []any{nil}, 0},
		{"all-nil -> 0", []any{nil, nil}, 0},
		{"empty -> 0", []any{}, 0},
		{"single arrayless -> 1", []any{span("a")}, 1},
		{"single in slice -> 1", []any{[]ui.Node{span("a")}}, 1},
		{"flat -> 3", []any{span("a"), span("b"), span("c")}, 3},
		{"nested 2 levels", []any{span("a"), []ui.Node{span("b"), span("c")}}, 3},
		{"nested 3 levels", []any{[]any{[]ui.Node{span("a"), span("b")}, span("c")}}, 3},
		{"mixed kinds", []any{"text", span("a"), []any{"more", span("b")}}, 4},
		{"nil interspersed", []any{span("a"), nil, span("b"), nil, span("c")}, 3},
	}
	for _, parseCase := range cases {
		if parseGot := len(html.Children(parseCase.in...)); parseGot != parseCase.want {
			t.Errorf("%s: count = %d, want %d", parseCase.name, parseGot, parseCase.want)
		}
	}
}

// Deeply nested children flatten in document order (a mis-ordered or partial
// flatten would scramble the rendered output).
func TestChildrenDeepFlattenOrder(t *testing.T) {
	parseNodes := html.Children(
		ui.Text("1"),
		[]any{ui.Text("2"), []ui.Node{ui.Text("3"), ui.Text("4")}},
		ui.Text("5"),
	)
	parseOut, parseErr := ui.RenderToString(html.Div(html.Props{}, parseNodes...))
	if parseErr != nil {
		t.Fatalf("render error: %v", parseErr)
	}
	if parseOut != "<div>12345</div>" {
		t.Errorf("deep flatten order = %q, want <div>12345</div>", parseOut)
	}
}
