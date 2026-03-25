package prerender

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func BenchmarkNormalizeRoutePath(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		path, err := normalizeRoutePath("/docs/getting-started/")
		if err != nil {
			b.Fatalf("normalizeRoutePath: %v", err)
		}
		if path == "" {
			b.Fatal("normalizeRoutePath returned empty path")
		}
	}
}

func BenchmarkBuildTargetJSONBootstrap(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		target := buildTarget("/docs/getting-started", ui.SSRBootstrapFormatJSON)
		if target.HTMLFile == "" || target.BootstrapFile == "" {
			b.Fatal("buildTarget returned incomplete target")
		}
	}
}
