package prerender

import "testing"

func BenchmarkNormalizeRoutePathMicro(b *testing.B) {
	paths := []string{
		"/",
		"/docs",
		"/docs/getting-started",
		"/docs/reference",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := normalizeRoutePath(paths[i%len(paths)])
		if err != nil {
			b.Fatal(err)
		}
	}
}
