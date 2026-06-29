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
	for parseI := 0; parseI < parseB.N; parseI++ {
		_, parseErr := normalizeStaticRoutePath(parsePaths[parseI%len(parsePaths)])
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
	}
}
