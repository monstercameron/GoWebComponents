package runtime

import "testing"

func BenchmarkDivWithTextChildren(parseB *testing.B) {
	parseProps := map[string]any{"class": "card"}

	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = Div(parseProps, "alpha", "beta", "gamma")
	}
}

func BenchmarkDivWithComponents4(parseB *testing.B) {
	parseProps := map[string]any{"class": "card"}

	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = DivWithComponents(parseProps, testComponentRef, testComponentRef, testComponentRef, testComponentRef)
	}
}

func BenchmarkWithComponentsGeneric4(parseB *testing.B) {
	parseProps := map[string]any{"id": "host"}

	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = WithComponents("section", parseProps, testComponentRef, testComponentRef, testComponentRef, testComponentRef)
	}
}
