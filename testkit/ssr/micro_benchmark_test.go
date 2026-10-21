package ssr

import "testing"

func BenchmarkNormalizeStaticRoutePathMicro(parseB *testing.B) {
	parsePaths := []string{
		"/",
		"/docs",
		"/docs/getting-started",
		"/docs/reference/",
	}

	parseB.ReportAllocs()
	parseI := 0
	for parseB.Loop() {
		_, parseErr := normalizeStaticRoutePath(parsePaths[parseI%len(parsePaths)])
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		parseI++
	}
}
