package shorthand

import "testing"

func BenchmarkTagWithMixedArgs(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		parseNode := Tag("section",
			ClassStr("rounded-xl border border-slate-300 p-4"),
			Data("bench", "true"),
			Aria("label", "benchmark section"),
			Text("alpha"),
			Text("beta"),
		)
		if parseNode == nil {
			parseB.Fatal("Tag returned nil")
		}
	}
}

func BenchmarkClassNames(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		parseValue := ClassNames(
			"grid gap-4",
			When(true, "items-center"),
			When(false, "hidden"),
			"px-4 py-2",
		)
		if parseValue == "" {
			parseB.Fatal("ClassNames returned empty class list")
		}
	}
}
