package runtime

import "testing"

var parseHostPropsShapeSink *Element

// BenchmarkCreateElementHostPropsShapes measures the attribute-view cost for
// props that can and cannot use the compact host path.
func BenchmarkCreateElementHostPropsShapes(parseB *testing.B) {
	parseHandler := func() {}
	parseCases := []struct {
		getName  string
		getProps map[string]any
	}{
		{getName: "event-only", getProps: map[string]any{"onclick": parseHandler}},
		{getName: "special-only", getProps: map[string]any{"value": "draft"}},
		{getName: "mixed", getProps: map[string]any{"class": "row", "onclick": parseHandler}},
		{getName: "strings-only", getProps: map[string]any{"class": "row", "id": "item"}},
	}
	for _, parseCase := range parseCases {
		parseB.Run(parseCase.getName, func(parseInner *testing.B) {
			parseInner.ReportAllocs()
			for parseInner.Loop() {
				parseHostPropsShapeSink = CreateElementOwned("input", parseCase.getProps)
			}
		})
	}
}
