package runtime

import "testing"

func BenchmarkDivWithTextChildren(parseB *testing.B) {
	parseProps := map[string]interface{}{"class": "card"}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = Div(parseProps, "alpha", "beta", "gamma")
	}
}

func BenchmarkDivWithComponents4(parseB *testing.B) {
	parseProps := map[string]interface{}{"class": "card"}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = DivWithComponents(parseProps, testComponentRef, testComponentRef, testComponentRef, testComponentRef)
	}
}

func BenchmarkWithComponentsGeneric4(parseB *testing.B) {
	parseProps := map[string]interface{}{"id": "host"}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = WithComponents("section", parseProps, testComponentRef, testComponentRef, testComponentRef, testComponentRef)
	}
}
