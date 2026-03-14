package runtime

import "testing"

func BenchmarkDivWithTextChildren(b *testing.B) {
	props := map[string]interface{}{"class": "card"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Div(props, "alpha", "beta", "gamma")
	}
}

func BenchmarkDivWithComponents4(b *testing.B) {
	props := map[string]interface{}{"class": "card"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = DivWithComponents(props, testComponentRef, testComponentRef, testComponentRef, testComponentRef)
	}
}

func BenchmarkWithComponentsGeneric4(b *testing.B) {
	props := map[string]interface{}{"id": "host"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = WithComponents("section", props, testComponentRef, testComponentRef, testComponentRef, testComponentRef)
	}
}
