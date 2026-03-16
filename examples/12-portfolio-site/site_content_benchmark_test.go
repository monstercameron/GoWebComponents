package main

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func BenchmarkPortfolioProjectLookup(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		project, ok := findPortfolioProject("GoWebComponents")
		if !ok || project.Title == "" {
			b.Fatal("expected GoWebComponents project lookup")
		}
	}
}

func BenchmarkPortfolioTechnologyCounting(b *testing.B) {
	projects := portfolioProjects()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if total := totalPortfolioTechnologies(projects); total != 10 {
			b.Fatalf("expected 10 technology tags, got %d", total)
		}
	}
}

func BenchmarkPortfolioSnapshotRenderToString(b *testing.B) {
	node := renderPortfolioSnapshot()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		markup, err := ui.RenderToString(node)
		if err != nil {
			b.Fatal(err)
		}
		if len(markup) == 0 {
			b.Fatal("expected rendered portfolio snapshot")
		}
	}
}
