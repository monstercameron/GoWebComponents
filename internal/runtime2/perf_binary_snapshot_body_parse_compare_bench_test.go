package runtime2

import (
	"fmt"
	"testing"
)

var storeBinarySnapshotBodyParseBenchmarkEnvelopeSink SnapshotEnvelope

// parseBinarySnapshotBodyLegacyBenchmark parses one snapshot body using the previous helper-driven decode path.
func parseBinarySnapshotBodyLegacyBenchmark(parsePayload []byte) (SnapshotEnvelope, error) {
	parseRegionInstanceIDText, parseOffset, parseErr := parseBinaryString(parsePayload, 0, "snapshot-body", "region_instance_id")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseEpoch, parseOffset, parseErr := parseBinaryUint64(parsePayload, parseOffset, "snapshot-body", "epoch")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseInputVersion, parseOffset, parseErr := parseBinaryUint64(parsePayload, parseOffset, "snapshot-body", "input_version")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseSourceVersion, parseOffset, parseErr := parseBinaryUint64(parsePayload, parseOffset, "snapshot-body", "source_version")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parsePropsLength, parseOffset, parseErr := parseBinaryUint32(parsePayload, parseOffset, "snapshot-body", "props length")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parsePropsSpan, parseOffset, parseErr := parseBinaryPayloadSpan(parsePayload, parseOffset, int(parsePropsLength), "snapshot-body", "props payload")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseProps, parseErr := ParseBinaryPropsValue(parsePropsSpan)
	if parseErr != nil {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot props: %w", parseErr)
	}
	parseSourceIDTableLength, parseOffset, parseErr := parseBinaryUint32(parsePayload, parseOffset, "snapshot-body", "source-id-table length")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseSourceIDTableSpan, parseOffset, parseErr := parseBinaryPayloadSpan(parsePayload, parseOffset, int(parseSourceIDTableLength), "snapshot-body", "source-id-table payload")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseSourceValuesLength, parseOffset, parseErr := parseBinaryUint32(parsePayload, parseOffset, "snapshot-body", "source-values length")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseSourceValuesSpan, parseOffset, parseErr := parseBinaryPayloadSpan(parsePayload, parseOffset, int(parseSourceValuesLength), "snapshot-body", "source-values payload")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseSourceIDsCache := storeBinarySourceIDsPool.Get().(*buildBinarySourceIDsCache)
	parseSourceIDsCache.getSourceIDs = parseSourceIDsCache.getSourceIDs[:0]
	parseSourceIDs, parseSourceIDsErr := parseBinarySourceIDTableInto(parseSourceIDsCache.getSourceIDs, parseSourceIDTableSpan)
	if parseSourceIDsErr != nil {
		releaseSnapshotBodySourceIDsCache(parseSourceIDsCache)
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot source-id table: %w", parseSourceIDsErr)
	}
	parseSourceIDsCache.getSourceIDs = parseSourceIDs
	parseSources, parseSourcesErr := parseBinarySourceValuesSection(parseSourceIDsCache.getSourceIDs, parseSourceValuesSpan)
	if parseSourcesErr != nil {
		releaseSnapshotBodySourceIDsCache(parseSourceIDsCache)
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot source-values: %w", parseSourcesErr)
	}
	releaseSnapshotBodySourceIDsCache(parseSourceIDsCache)
	if parseOffset != len(parsePayload) {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: snapshot-body has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	parseRegionInstanceID, parseErr := ParseRegionInstanceID(parseRegionInstanceIDText)
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	if parseErr := validateBinarySnapshotBodyRequiredVersions(parseEpoch, parseInputVersion); parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	return SnapshotEnvelope{
		RegionInstanceID: parseRegionInstanceID,
		Epoch:            parseEpoch,
		InputVersion:     parseInputVersion,
		SourceVersion:    parseSourceVersion,
		Props:            parseProps,
		Sources:          parseSources,
	}, nil
}

// BenchmarkParseBinarySnapshotBodyCurrentVsLegacy compares the direct parser against the previous helper-based parser.
func BenchmarkParseBinarySnapshotBodyCurrentVsLegacy(parseB *testing.B) {
	parseEnvelope := SnapshotEnvelope{
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
	parsePayload, parsePayloadErr := BuildBinarySnapshotBody(parseEnvelope)
	if parsePayloadErr != nil {
		parseB.Fatalf("BuildBinarySnapshotBody returned error: %v", parsePayloadErr)
	}
	parseB.SetBytes(int64(len(parsePayload)))

	parseB.Run("legacy_helper_parse", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseEnvelope, parseErr := parseBinarySnapshotBodyLegacyBenchmark(parsePayload)
			if parseErr != nil {
				parseB.Fatalf("parseBinarySnapshotBodyLegacyBenchmark returned error: %v", parseErr)
			}
			storeBinarySnapshotBodyParseBenchmarkEnvelopeSink = parseEnvelope
		}
	})

	parseB.Run("current_direct_parse", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseEnvelope, parseErr := ParseBinarySnapshotBody(parsePayload)
			if parseErr != nil {
				parseB.Fatalf("ParseBinarySnapshotBody returned error: %v", parseErr)
			}
			storeBinarySnapshotBodyParseBenchmarkEnvelopeSink = parseEnvelope
		}
	})
}
