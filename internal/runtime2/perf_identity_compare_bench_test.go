package runtime2

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"testing"
)

var (
	storeRuntime2PatchIdentityBenchmarkSink string
)

// buildRuntime2LegacyPatchStreamIdentity computes patch identity using the pre-optimization JSON marshal path.
func buildRuntime2LegacyPatchStreamIdentity(parseRaw PatchStreamRaw) (string, error) {
	parsePayload, parsePayloadErr := json.Marshal(struct {
		GetHeader      PatchStreamHeaderRaw
		GetStringTable []string
		GetOps         []PatchStreamOpRaw
	}{
		GetHeader:      parseRaw.GetHeader,
		GetStringTable: parseRaw.GetStringTable,
		GetOps:         parseRaw.GetOps,
	})
	if parsePayloadErr != nil {
		return "", fmt.Errorf("runtime2: encode patch identity payload: %w", parsePayloadErr)
	}
	parseHasher := fnv.New64a()
	_, _ = parseHasher.Write(parsePayload)
	return fmt.Sprintf("%x", parseHasher.Sum64()), nil
}

// BenchmarkBuildPatchStreamIdentityCurrentVsLegacy compares current patch identity hashing against the legacy marshal-based implementation.
func BenchmarkBuildPatchStreamIdentityCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("small/current", func(parseB *testing.B) {
		parsePatchStream := buildPerfHotspotPatchStream(
			parseB,
			parseBuildAgent3BenchRenderOutput("before", "active"),
			parseBuildAgent3BenchRenderOutput("after", "idle"),
		)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseIdentity, parseIdentityErr := BuildPatchStreamIdentity(parsePatchStream)
			if parseIdentityErr != nil {
				parseB.Fatalf("BuildPatchStreamIdentity returned error: %v", parseIdentityErr)
			}
			storeRuntime2PatchIdentityBenchmarkSink = parseIdentity
		}
	})
	parseB.Run("small/legacy", func(parseB *testing.B) {
		parsePatchStream := buildPerfHotspotPatchStream(
			parseB,
			parseBuildAgent3BenchRenderOutput("before", "active"),
			parseBuildAgent3BenchRenderOutput("after", "idle"),
		)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseIdentity, parseIdentityErr := buildRuntime2LegacyPatchStreamIdentity(parsePatchStream)
			if parseIdentityErr != nil {
				parseB.Fatalf("buildRuntime2LegacyPatchStreamIdentity returned error: %v", parseIdentityErr)
			}
			storeRuntime2PatchIdentityBenchmarkSink = parseIdentity
		}
	})
	parseB.Run("large/current", func(parseB *testing.B) {
		parsePatchStream := buildPerfHotspotPatchStream(
			parseB,
			buildPerfHotspotKeyedListRenderOutput(64, 0, "active"),
			buildPerfHotspotKeyedListRenderOutput(64, 1, "idle"),
		)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseIdentity, parseIdentityErr := BuildPatchStreamIdentity(parsePatchStream)
			if parseIdentityErr != nil {
				parseB.Fatalf("BuildPatchStreamIdentity returned error: %v", parseIdentityErr)
			}
			storeRuntime2PatchIdentityBenchmarkSink = parseIdentity
		}
	})
	parseB.Run("large/legacy", func(parseB *testing.B) {
		parsePatchStream := buildPerfHotspotPatchStream(
			parseB,
			buildPerfHotspotKeyedListRenderOutput(64, 0, "active"),
			buildPerfHotspotKeyedListRenderOutput(64, 1, "idle"),
		)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseIdentity, parseIdentityErr := buildRuntime2LegacyPatchStreamIdentity(parsePatchStream)
			if parseIdentityErr != nil {
				parseB.Fatalf("buildRuntime2LegacyPatchStreamIdentity returned error: %v", parseIdentityErr)
			}
			storeRuntime2PatchIdentityBenchmarkSink = parseIdentity
		}
	})
}
