package shorthand

import "testing"

func BenchmarkTagWithMixedArgs(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		node := Tag("section",
			Class("rounded-xl border border-slate-300 p-4"),
			Data("bench", "true"),
			Aria("label", "benchmark section"),
			Text("alpha"),
			Text("beta"),
		)
		if node == nil {
			b.Fatal("Tag returned nil")
		}
	}
}

func BenchmarkClassNames(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		value := ClassNames(
			"grid gap-4",
			When(true, "items-center"),
			When(false, "hidden"),
			"px-4 py-2",
		)
		if value == "" {
			b.Fatal("ClassNames returned empty class list")
		}
	}
}
