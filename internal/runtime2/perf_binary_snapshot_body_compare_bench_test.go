package runtime2

import (
	"hash/crc32"
	"testing"
)

// buildBinarySnapshotBodyCompareBenchEnvelope returns one stable snapshot envelope used for current-vs-legacy body path benchmarks.
func buildBinarySnapshotBodyCompareBenchEnvelope(parseB *testing.B) SnapshotEnvelope {
	parseB.Helper()
	return SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-1"),
		Epoch:            7,
		InputVersion:     19,
		SourceVersion:    11,
		Props: map[string]any{
			"title": "Orders",
			"count": 42,
			"meta": map[string]any{
				"priority": "high",
				"visible":  true,
			},
		},
		Sources: map[string]any{
			"status": "healthy",
			"items": []any{
				map[string]any{
					"id":    "a",
					"score": 9,
				},
				map[string]any{
					"id":    "b",
					"score": 7,
				},
			},
		},
	}
}

// buildBinarySnapshotEnvelopeLegacyBodyCapacityBenchmark preserves the legacy static snapshot-body capacity hint path.
func buildBinarySnapshotEnvelopeLegacyBodyCapacityBenchmark(parseEnvelope SnapshotEnvelope) ([]byte, error) {
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return nil, parseErr
	}
	parseRegionIDLength := 2 + len(string(parseEnvelope.RegionInstanceID))
	parseCap := binaryEnvelopeHeaderSize + parseRegionIDLength + 24 + 4 + 64 + 4 + 4 + (len(parseEnvelope.Sources)+1)*16
	parsePayload := make([]byte, binaryEnvelopeHeaderSize, parseCap)
	var parseErr error
	parsePayload, parseErr = appendBinarySnapshotBody(parsePayload, parseEnvelope)
	if parseErr != nil {
		return nil, parseErr
	}
	parseBodyLength := uint32(len(parsePayload) - binaryEnvelopeHeaderSize)
	parseChecksum := crc32.ChecksumIEEE(parsePayload[binaryEnvelopeHeaderSize:])
	writeBinaryEnvelopeHeaderAt(parsePayload, BinaryEnvelopeKindSnapshot, parseBodyLength, parseChecksum)
	return parsePayload, nil
}

// parseBinarySnapshotEnvelopeLegacyPropsValidationBenchmark preserves the legacy parse behavior that revalidated decoded props.
func parseBinarySnapshotEnvelopeLegacyPropsValidationBenchmark(parsePayload []byte) (SnapshotEnvelope, error) {
	parseEnvelope, parseErr := ParseBinarySnapshotEnvelope(parsePayload)
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	if parseValidateErr := ValidateSnapshotEnvelope(parseEnvelope); parseValidateErr != nil {
		return SnapshotEnvelope{}, parseValidateErr
	}
	return parseEnvelope, nil
}

// BenchmarkBuildAndParseBinarySnapshotEnvelopeCurrentVsLegacy compares current body sizing and parse validation paths against legacy behavior.
func BenchmarkBuildAndParseBinarySnapshotEnvelopeCurrentVsLegacy(parseB *testing.B) {
	parseEnvelope := buildBinarySnapshotBodyCompareBenchEnvelope(parseB)
	parsePayload, parsePayloadErr := BuildBinarySnapshotEnvelope(parseEnvelope)
	if parsePayloadErr != nil {
		parseB.Fatalf("BuildBinarySnapshotEnvelope(seed) returned error: %v", parsePayloadErr)
	}
	parseB.Run("build/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := buildBinarySnapshotEnvelopeLegacyBodyCapacityBenchmark(parseEnvelope); parseErr != nil {
				parseB.Fatalf("buildBinarySnapshotEnvelopeLegacyBodyCapacityBenchmark returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("build/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := BuildBinarySnapshotEnvelope(parseEnvelope); parseErr != nil {
				parseB.Fatalf("BuildBinarySnapshotEnvelope returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("parse/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := parseBinarySnapshotEnvelopeLegacyPropsValidationBenchmark(parsePayload); parseErr != nil {
				parseB.Fatalf("parseBinarySnapshotEnvelopeLegacyPropsValidationBenchmark returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("parse/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := ParseBinarySnapshotEnvelope(parsePayload); parseErr != nil {
				parseB.Fatalf("ParseBinarySnapshotEnvelope returned error: %v", parseErr)
			}
		}
	})
}
