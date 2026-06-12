package runtime

import (
	"fmt"
	"testing"
)

// buildSSRBenchmarkTree builds a representative page: a component-wrapped
// list of rows with attributes, nested hosts, and text children.
func buildSSRBenchmarkTree(parseRows int) *Element {
	parseRowComponent := func(parseProps map[string]any) *Element {
		parseIdx, _ := parseProps["idx"].(int)
		return CreateElement("li", map[string]any{"class": "row", "data-idx": fmt.Sprintf("%d", parseIdx)},
			CreateElement("span", map[string]any{"class": "label"}, "Row label"),
			CreateElement("span", map[string]any{"class": "value"}, fmt.Sprintf("value-%d", parseIdx)),
		)
	}
	parseChildren := make([]any, 0, parseRows)
	for parseIdx := range parseRows {
		parseChildren = append(parseChildren, CreateElement(parseRowComponent, map[string]any{"idx": parseIdx}))
	}
	return CreateElement("div", map[string]any{"class": "page"},
		CreateElement("header", map[string]any{"class": "head"}, "SSR Benchmark"),
		CreateElement("ul", map[string]any{"class": "list"}, parseChildren...),
	)
}

// BenchmarkSSRRenderToString100Rows measures server-side HTML rendering of a
// 100-row component page — the per-request hot path for SSR deployments.
func BenchmarkSSRRenderToString100Rows(parseB *testing.B) {
	parseTree := buildSSRBenchmarkTree(100)
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseIdx := 0; parseIdx < parseB.N; parseIdx++ {
		parseMarkup, parseErr := RenderToString(parseTree)
		if parseErr != nil || len(parseMarkup) == 0 {
			parseB.Fatalf("render failed: %v", parseErr)
		}
	}
}
