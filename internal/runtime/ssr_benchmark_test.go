package runtime

import (
	"fmt"
	"testing"
)

// buildSSRBenchmarkTree builds a representative page: a component-wrapped
// list of rows with attributes, nested hosts, and text children.
func buildSSRBenchmarkTree(parseRows int) *Element {
	parseRowComponent := func(parseProps map[string]interface{}) *Element {
		parseIdx, _ := parseProps["idx"].(int)
		return CreateElement("li", map[string]interface{}{"class": "row", "data-idx": fmt.Sprintf("%d", parseIdx)},
			CreateElement("span", map[string]interface{}{"class": "label"}, "Row label"),
			CreateElement("span", map[string]interface{}{"class": "value"}, fmt.Sprintf("value-%d", parseIdx)),
		)
	}
	parseChildren := make([]interface{}, 0, parseRows)
	for parseIdx := 0; parseIdx < parseRows; parseIdx++ {
		parseChildren = append(parseChildren, CreateElement(parseRowComponent, map[string]interface{}{"idx": parseIdx}))
	}
	return CreateElement("div", map[string]interface{}{"class": "page"},
		CreateElement("header", map[string]interface{}{"class": "head"}, "SSR Benchmark"),
		CreateElement("ul", map[string]interface{}{"class": "list"}, parseChildren...),
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
