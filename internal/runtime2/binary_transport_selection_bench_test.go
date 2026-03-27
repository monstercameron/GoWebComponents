package runtime2

import (
	"strings"
	"testing"
)

// buildSnapshotTransportBenchmarkEnvelope builds one representative snapshot envelope for transport fallback benchmarks.
func buildSnapshotTransportBenchmarkEnvelope() SnapshotEnvelope {
	return SnapshotEnvelope{
		RegionInstanceID: "region-1",
		Epoch:            1,
		InputVersion:     2,
		Props: map[string]any{
			"title": "Orders",
			"count": 42,
		},
		Sources: map[string]any{
			"status": "healthy",
		},
	}
}

// buildSnapshotTransportBenchmarkLongRegionEnvelope builds one snapshot envelope that forces binary encode fallback.
func buildSnapshotTransportBenchmarkLongRegionEnvelope() SnapshotEnvelope {
	parseRegionID := RegionInstanceID(strings.Repeat("r", 70_000))
	return SnapshotEnvelope{
		RegionInstanceID: parseRegionID,
		Epoch:            1,
		InputVersion:     2,
	}
}

// buildSnapshotTransportBenchmarkBinaryAndStructuredCapabilities builds one capability report with binary and structured-clone support.
func buildSnapshotTransportBenchmarkBinaryAndStructuredCapabilities() CapabilityReport {
	return BuildCapabilityReport(CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasBinaryTransportSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
}

// BenchmarkBuildSnapshotTransportPayloadWithFallback measures snapshot transport build fallback paths.
func BenchmarkBuildSnapshotTransportPayloadWithFallback(parseB *testing.B) {
	parseCapabilities := buildSnapshotTransportBenchmarkBinaryAndStructuredCapabilities()
	parseEnvelope := buildSnapshotTransportBenchmarkEnvelope()
	parseFallbackEnvelope := buildSnapshotTransportBenchmarkLongRegionEnvelope()
	parseB.Run("binary-success", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, _, parseBuildErr := BuildSnapshotTransportPayloadWithFallback(parseEnvelope, parseCapabilities); parseBuildErr != nil {
				parseB.Fatalf("BuildSnapshotTransportPayloadWithFallback(binary-success) returned error: %v", parseBuildErr)
			}
		}
	})
	parseB.Run("binary-fallback-structured", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, _, parseBuildErr := BuildSnapshotTransportPayloadWithFallback(parseFallbackEnvelope, parseCapabilities); parseBuildErr != nil {
				parseB.Fatalf("BuildSnapshotTransportPayloadWithFallback(binary-fallback) returned error: %v", parseBuildErr)
			}
		}
	})
}

// BenchmarkParseSnapshotTransportPayloadWithFallback measures snapshot transport parse fallback paths.
func BenchmarkParseSnapshotTransportPayloadWithFallback(parseB *testing.B) {
	parseEnvelope := buildSnapshotTransportBenchmarkEnvelope()
	parseBinaryPayload, parseBinaryPayloadErr := BuildBinarySnapshotEnvelope(parseEnvelope)
	if parseBinaryPayloadErr != nil {
		parseB.Fatalf("BuildBinarySnapshotEnvelope returned error: %v", parseBinaryPayloadErr)
	}
	parseStructuredPayload, parseStructuredPayloadErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseEnvelope)
	if parseStructuredPayloadErr != nil {
		parseB.Fatalf("BuildStructuredCloneSnapshotEnvelopeJSON returned error: %v", parseStructuredPayloadErr)
	}
	parseB.Run("binary-tier-binary-payload", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, _, parseParseErr := ParseSnapshotTransportPayloadWithFallback(TransportTierBinary, parseBinaryPayload); parseParseErr != nil {
				parseB.Fatalf("ParseSnapshotTransportPayloadWithFallback(binary-tier-binary-payload) returned error: %v", parseParseErr)
			}
		}
	})
	parseB.Run("binary-tier-structured-payload", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, _, parseParseErr := ParseSnapshotTransportPayloadWithFallback(TransportTierBinary, parseStructuredPayload); parseParseErr != nil {
				parseB.Fatalf("ParseSnapshotTransportPayloadWithFallback(binary-tier-structured-payload) returned error: %v", parseParseErr)
			}
		}
	})
}
