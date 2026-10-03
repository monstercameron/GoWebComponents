package main

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func BenchmarkPortfolioProjectLookup(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		parseProject, parseOk := findPortfolioProject("GoWebComponents")
		if !parseOk || parseProject.Title == "" {
			parseB.Fatal("expected GoWebComponents project lookup")
		}
	}
}

func BenchmarkPortfolioTechnologyCounting(parseB *testing.B) {
	parseProjects := portfolioProjects()
	parseB.ReportAllocs()
	for parseB.Loop() {
		if parseTotal := totalPortfolioTechnologies(parseProjects); parseTotal != 10 {
			parseB.Fatalf("expected 10 technology tags, got %d", parseTotal)
		}
	}
}

func BenchmarkFeaturedPortfolioProjects(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		parseFeatured := featuredPortfolioProjects()
		if len(parseFeatured) != 2 {
			parseB.Fatalf("expected 2 featured projects, got %d", len(parseFeatured))
		}
	}
}

func BenchmarkPortfolioSnapshotRenderToString(parseB *testing.B) {
	parseNode := renderPortfolioSnapshot()
	parseB.ReportAllocs()
	for parseB.Loop() {
		parseMarkup, parseErr := ui.RenderToString(parseNode)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		if len(parseMarkup) == 0 {
			parseB.Fatal("expected rendered portfolio snapshot")
		}
	}
}

func BenchmarkPortfolioProjectsGridRenderToString(parseB *testing.B) {
	parseNode := renderPortfolioProjectsGridSnapshot()
	parseB.ReportAllocs()
	for parseB.Loop() {
		parseMarkup, parseErr := ui.RenderToString(parseNode)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		if len(parseMarkup) == 0 {
			parseB.Fatal("expected rendered project grid snapshot")
		}
	}
}
