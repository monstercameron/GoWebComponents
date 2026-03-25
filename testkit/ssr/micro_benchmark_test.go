package ssr

import "testing"

func BenchmarkNormalizeStaticRoutePathMicro(b *testing.B) {
	paths := []string{
		"/",
		"/docs",
		"/docs/getting-started",
		"/docs/reference/",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := normalizeStaticRoutePath(paths[i%len(paths)])
		if err != nil {
			b.Fatal(err)
		}
	}
}
