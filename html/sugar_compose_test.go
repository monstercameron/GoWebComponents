package html

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

// TestConditionalPropOptions verifies AttrIf/ClassIf/StyleIf apply only when the
// condition holds and are no-ops (skipped) otherwise.
func TestConditionalPropOptions(parseT *testing.T) {
	parseOn, parseErr := ui.RenderToString(Div(PropsOf(
		AttrIf(true, "data-on", "1"),
		ClassIf(true, "active"),
		StyleIf(true, map[string]string{"color": "red"}),
	), Text("x")))
	if parseErr != nil {
		parseT.Fatalf("expected on render, got %v", parseErr)
	}
	for _, parseWant := range []string{`data-on="1"`, `class="active"`, "color:red"} {
		if !strings.Contains(parseOn, parseWant) {
			parseT.Fatalf("expected %q in %q", parseWant, parseOn)
		}
	}

	parseOff, parseErr2 := ui.RenderToString(Div(PropsOf(
		AttrIf(false, "data-on", "1"),
		ClassIf(false, "active"),
		StyleIf(false, map[string]string{"color": "red"}),
	), Text("x")))
	if parseErr2 != nil {
		parseT.Fatalf("expected off render, got %v", parseErr2)
	}
	if parseOff != "<div>x</div>" {
		parseT.Fatalf("expected no conditional attributes, got %q", parseOff)
	}
}

// TestStyleVarSetsCustomProperty verifies StyleVar emits a single style property.
func TestStyleVarSetsCustomProperty(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(Div(PropsOf(StyleVar("--accent", "blue")), Text("x")))
	if parseErr != nil {
		parseT.Fatalf("expected StyleVar render, got %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "--accent:blue") {
		parseT.Fatalf("expected custom property, got %q", parseMarkup)
	}
}

// TestMergePropsOverlaysAndUnionsWithoutMutating verifies override scalars win,
// base zero-fields are retained, the maps union (override per key), and neither
// input is mutated.
func TestMergePropsOverlaysAndUnionsWithoutMutating(parseT *testing.T) {
	parseBase := Props{Class: "base", Title: "keep", Raw: map[string]any{"data-k": "1", "data-base": "b"}}
	parseOverride := Props{Class: "over", Raw: map[string]any{"data-k": "2", "data-over": "o"}}

	parseMerged := MergeProps(parseBase, parseOverride)
	if parseMerged.Class != "over" {
		parseT.Fatalf("expected override Class to win, got %q", parseMerged.Class)
	}
	if parseMerged.Title != "keep" {
		parseT.Fatalf("expected base Title retained when override is zero, got %q", parseMerged.Title)
	}
	if parseMerged.Raw["data-k"] != "2" || parseMerged.Raw["data-base"] != "b" || parseMerged.Raw["data-over"] != "o" {
		parseT.Fatalf("expected unioned Raw with override winning, got %#v", parseMerged.Raw)
	}
	// Inputs must be untouched.
	if parseBase.Class != "base" || parseBase.Raw["data-k"] != "1" || len(parseBase.Raw) != 2 {
		parseT.Fatalf("expected base unmutated, got %#v", parseBase)
	}
	if parseOverride.Class != "over" || len(parseOverride.Raw) != 2 {
		parseT.Fatalf("expected override unmutated, got %#v", parseOverride)
	}
}

// TestDefaultPropsFillsOnlyMissing verifies explicit props win and defaults fill
// only the zero fields.
func TestDefaultPropsFillsOnlyMissing(parseT *testing.T) {
	parseProps := Props{Class: "explicit"}
	parseDefaults := Props{Class: "fallback", Role: "button"}

	parseResult := DefaultProps(parseProps, parseDefaults)
	if parseResult.Class != "explicit" {
		parseT.Fatalf("expected explicit Class to win, got %q", parseResult.Class)
	}
	if parseResult.Role != "button" {
		parseT.Fatalf("expected default Role to fill the gap, got %q", parseResult.Role)
	}
}

// TestMarkdownRendersThroughShorthand verifies Markdown delegates to
// RenderMarkdown and produces semantic HTML.
func TestMarkdownRendersThroughShorthand(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(Div(Props{}, Markdown("# Title\n\nBody text.")...))
	if parseErr != nil {
		parseT.Fatalf("expected Markdown render, got %v", parseErr)
	}
	if !strings.Contains(parseMarkup, "<h1>Title</h1>") || !strings.Contains(parseMarkup, "Body text.") {
		parseT.Fatalf("expected markdown HTML, got %q", parseMarkup)
	}
}

// TestTextLinesSplitsOnNewlines verifies TextLines interleaves <br> and yields a
// single text node when there is no newline.
func TestTextLinesSplitsOnNewlines(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(P(Props{}, TextLines("alpha\nbeta")...))
	if parseErr != nil {
		parseT.Fatalf("expected TextLines render, got %v", parseErr)
	}
	if parseMarkup != "<p>alpha<br>beta</p>" {
		parseT.Fatalf("expected line break between lines, got %q", parseMarkup)
	}

	parseSingle, parseErr2 := ui.RenderToString(P(Props{}, TextLines("solo")...))
	if parseErr2 != nil {
		parseT.Fatalf("expected single-line render, got %v", parseErr2)
	}
	if parseSingle != "<p>solo</p>" {
		parseT.Fatalf("expected single text node, got %q", parseSingle)
	}
}

// TestShowTogglesHiddenWithoutRemoving verifies Show keeps the node mounted but
// hidden when false, and returns it unchanged when true.
func TestShowTogglesHiddenWithoutRemoving(parseT *testing.T) {
	parseHidden, parseErr := ui.RenderToString(Show(false, Div(Props{}, Text("content"))))
	if parseErr != nil {
		parseT.Fatalf("expected hidden render, got %v", parseErr)
	}
	if !strings.Contains(parseHidden, "hidden") || !strings.Contains(parseHidden, "content") {
		parseT.Fatalf("expected node mounted but hidden, got %q", parseHidden)
	}

	parseVisible, parseErr2 := ui.RenderToString(Show(true, Div(Props{}, Text("content"))))
	if parseErr2 != nil {
		parseT.Fatalf("expected visible render, got %v", parseErr2)
	}
	if strings.Contains(parseVisible, "hidden") {
		parseT.Fatalf("expected no hidden attribute when shown, got %q", parseVisible)
	}

	if Show(false, nil) != nil {
		parseT.Fatal("expected Show(false, nil) to stay nil")
	}
}

func TestShowClonesPropsBeforeHiding(parseT *testing.T) {
	parseNode := Div(Props{Raw: map[string]any{"data-state": "shared"}}, Text("content"))
	parseHidden := Show(false, parseNode)
	if parseHidden == parseNode {
		parseT.Fatal("expected Show(false, node) to return a cloned node")
	}
	if _, parseOk := parseNode.Props["hidden"]; parseOk {
		parseT.Fatalf("expected source node props to stay visible, got %#v", parseNode.Props)
	}
	if parseHidden.Props["hidden"] != true {
		parseT.Fatalf("expected hidden clone prop, got %#v", parseHidden.Props)
	}
	parseHidden.Props["data-state"] = "changed"
	if parseNode.Props["data-state"] != "shared" {
		parseT.Fatalf("expected source props map to stay unaliased, got %#v", parseNode.Props)
	}
}

// TestWithChildrenAppends verifies WithChildren appends children and skips nil.
func TestWithChildrenAppends(parseT *testing.T) {
	parseNode := WithChildren(Div(Props{}, Text("a")), Text("b"), nil, Text("c"))
	parseMarkup, parseErr := ui.RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("expected WithChildren render, got %v", parseErr)
	}
	if parseMarkup != "<div>abc</div>" {
		parseT.Fatalf("expected appended children, got %q", parseMarkup)
	}
	if WithChildren(nil, Text("x")) != nil {
		parseT.Fatal("expected WithChildren(nil, ...) to stay nil")
	}
}

var benchmarkMergePropsSink Props

func BenchmarkMergePropsHotPath(parseB *testing.B) {
	parseBase := Props{
		ID:       "card-1",
		Class:    "card quiet",
		Title:    "Base",
		Role:     "region",
		TabIndex: -1,
		Style: map[string]string{
			"display": "grid",
			"gap":     "8px",
			"color":   "black",
		},
		Data: map[string]string{
			"owner": "base",
			"slot":  "body",
		},
		Aria: map[string]string{
			"label": "Base card",
		},
		Raw: map[string]any{
			"data-base": "1",
			"draggable": true,
		},
	}
	parseOverride := Props{
		Class:       "card selected",
		Placeholder: "Search",
		Disabled:    true,
		Style: map[string]string{
			"color":       "blue",
			"font-weight": "600",
		},
		Data: map[string]string{
			"owner": "override",
		},
		Aria: map[string]string{
			"selected": "true",
		},
		Raw: map[string]any{
			"data-override": "1",
		},
	}
	parseB.ReportAllocs()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		benchmarkMergePropsSink = MergeProps(parseBase, parseOverride)
	}
}
