package runtime2

import "testing"

// buildSnapshotDispatchFastHashLegacyBytes preserves the previous FNV-1a fast-hash implementation for benchmark comparison.
func buildSnapshotDispatchFastHashLegacyBytes(parsePayload []byte) uint64 {
	const (
		getDispatchFastHashOffset uint64 = 14695981039346656037
		getDispatchFastHashPrime  uint64 = 1099511628211
	)
	buildDispatchFastHash := getDispatchFastHashOffset
	for _, getPayloadByte := range parsePayload {
		buildDispatchFastHash ^= uint64(getPayloadByte)
		buildDispatchFastHash *= getDispatchFastHashPrime
	}
	if buildDispatchFastHash == 0 {
		return 1
	}
	return buildDispatchFastHash
}

// buildSnapshotDispatchFastHashLegacy preserves the previous buffered fast-hash path for benchmark comparison.
func buildSnapshotDispatchFastHashLegacy(parseEnvelope SnapshotEnvelope, parseSourceIDs []string, parseScratch []byte) (uint64, []byte, error) {
	if parseScratch == nil {
		parseScratch = make([]byte, 0, 256)
	}
	parseScratch = parseScratch[:0]
	parsePayload, parsePayloadErr := appendSnapshotDispatchEnvelopeWithSourceIDs(parseScratch, parseEnvelope, parseSourceIDs)
	if parsePayloadErr != nil {
		return 0, parseScratch, parsePayloadErr
	}
	return buildSnapshotDispatchFastHashLegacyBytes(parsePayload), parsePayload[:0], nil
}

// BenchmarkBuildSnapshotDispatchFastHashCurrentVsLegacy compares the streamed fast-hash path against the previous buffered payload path.
func BenchmarkBuildSnapshotDispatchFastHashCurrentVsLegacy(parseB *testing.B) {
	getEnvelope := buildPerfSnapshotDispatchHashEnvelope()
	getSourceIDs := []string{"filters", "stats"}
	parseB.Run("current_streamed_maphash", func(parseB *testing.B) {
		var getScratch []byte
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			var parseErr error
			_, getScratch, parseErr = buildSnapshotDispatchFastHashIntoWithSourceIDs(getEnvelope, getSourceIDs, getScratch)
			if parseErr != nil {
				parseB.Fatalf("buildSnapshotDispatchFastHashIntoWithSourceIDs returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("legacy_buffered_fnv1a", func(parseB *testing.B) {
		var getScratch []byte
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			var parseErr error
			_, getScratch, parseErr = buildSnapshotDispatchFastHashLegacy(getEnvelope, getSourceIDs, getScratch)
			if parseErr != nil {
				parseB.Fatalf("buildSnapshotDispatchFastHashLegacy returned error: %v", parseErr)
			}
		}
	})
}
