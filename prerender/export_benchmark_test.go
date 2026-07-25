package prerender

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func BenchmarkNormalizeRoutePath(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		parsePath, parseErr := normalizeRoutePath("/docs/getting-started/")
		if parseErr != nil {
			parseB.Fatalf("normalizeRoutePath: %v", parseErr)
		}
		if parsePath == "" {
			parseB.Fatal("normalizeRoutePath returned empty path")
		}
	}
}

func BenchmarkBuildTargetJSONBootstrap(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		parseTarget := buildTarget("/docs/getting-started", ui.SSRBootstrapFormatJSON)
		if parseTarget.HTMLFile == "" || parseTarget.BootstrapFile == "" {
			parseB.Fatal("buildTarget returned incomplete target")
		}
	}
}
