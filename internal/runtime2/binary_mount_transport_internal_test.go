package runtime2

import (
	"encoding/binary"
	"hash/crc32"
	"testing"
)

// buildBinaryMountPayloadForMutationTest builds one valid binary mount payload used by corruption tests.
func buildBinaryMountPayloadForMutationTest(parseT *testing.T) []byte {
	parseT.Helper()
	parsePayload, parseErr := BuildBinaryMountEnvelope(BinaryMountEnvelope{
		RegionInstanceID: RegionInstanceID("region-1"),
		RendererID:       RendererID("dashboard.hot-panel"),
		SourceIDs:        []string{"status"},
		Snapshot: SnapshotEnvelope{
			RegionInstanceID: RegionInstanceID("region-1"),
			Epoch:            1,
			InputVersion:     1,
			Props:            map[string]any{"title": "Orders"},
			Sources:          map[string]any{"status": "ok"},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("BuildBinaryMountEnvelope returned error: %v", parseErr)
	}
	return parsePayload
}

// buildBinaryUpdatePayloadForMutationTest builds one valid binary update payload used by corruption tests.
func buildBinaryUpdatePayloadForMutationTest(parseT *testing.T) []byte {
	parseT.Helper()
	parsePayload, parseErr := BuildBinaryUpdateEnvelope(BinaryUpdateEnvelope{
		RegionInstanceID: RegionInstanceID("region-1"),
		InputVersion:     2,
		Snapshot: SnapshotEnvelope{
			RegionInstanceID: RegionInstanceID("region-1"),
			Epoch:            1,
			InputVersion:     2,
			Props:            map[string]any{"title": "Orders"},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("BuildBinaryUpdateEnvelope returned error: %v", parseErr)
	}
	return parsePayload
}

// buildBinaryMountMutationOffsets returns the outer region and renderer field start offsets and lengths for one valid binary mount payload.
func buildBinaryMountMutationOffsets(parsePayload []byte) (int, int, int, int, error) {
	_, parseBody, parseErr := parseBinaryEnvelopePayload(parsePayload, BinaryEnvelopeKindMount)
	if parseErr != nil {
		return 0, 0, 0, 0, parseErr
	}
	parseRegionLength, parseRegionValueOffset, parseErr := parseBinaryUint16(parseBody, 0, "mount-body", "region_instance_id")
	if parseErr != nil {
		return 0, 0, 0, 0, parseErr
	}
	parseRendererLengthOffset := parseRegionValueOffset + int(parseRegionLength)
	parseRendererLength, parseRendererValueOffset, parseErr := parseBinaryUint16(parseBody, parseRendererLengthOffset, "mount-body", "renderer_id")
	if parseErr != nil {
		return 0, 0, 0, 0, parseErr
	}
	return binaryEnvelopeHeaderSize + parseRegionValueOffset, int(parseRegionLength), binaryEnvelopeHeaderSize + parseRendererValueOffset, int(parseRendererLength), nil
}

// buildBinaryUpdateMutationOffsets returns the outer region field and input-version offsets for one valid binary update payload.
func buildBinaryUpdateMutationOffsets(parsePayload []byte) (int, int, int, error) {
	_, parseBody, parseErr := parseBinaryEnvelopePayload(parsePayload, BinaryEnvelopeKindUpdate)
	if parseErr != nil {
		return 0, 0, 0, parseErr
	}
	parseRegionLength, parseRegionValueOffset, parseErr := parseBinaryUint16(parseBody, 0, "update-body", "region_instance_id")
	if parseErr != nil {
		return 0, 0, 0, parseErr
	}
	parseInputVersionOffset := binaryEnvelopeHeaderSize + parseRegionValueOffset + int(parseRegionLength)
	return binaryEnvelopeHeaderSize + parseRegionValueOffset, int(parseRegionLength), parseInputVersionOffset, nil
}

// storeBinaryEnvelopeHeaderForMutation refreshes one mutated binary envelope header length and checksum.
func storeBinaryEnvelopeHeaderForMutation(parsePayload []byte) {
	setBinaryUint32At(parsePayload, 8, uint32(len(parsePayload)-binaryEnvelopeHeaderSize))
	setBinaryUint32At(parsePayload, 12, crc32.ChecksumIEEE(parsePayload[binaryEnvelopeHeaderSize:]))
}

// TestParseBinaryMountEnvelopeRejectsCorruptedChecksumTrailingBytesAndOuterFieldMismatches verifies binary mount decode rejects checksum corruption, trailing bytes, outer region mismatches, and invalid renderer IDs.
func TestParseBinaryMountEnvelopeRejectsCorruptedChecksumTrailingBytesAndOuterFieldMismatches(parseT *testing.T) {
	parseMountPayload := buildBinaryMountPayloadForMutationTest(parseT)
	parseRegionOffset, parseRegionLength, parseRendererOffset, parseRendererLength, parseOffsetErr := buildBinaryMountMutationOffsets(parseMountPayload)
	if parseOffsetErr != nil {
		parseT.Fatalf("buildBinaryMountMutationOffsets returned error: %v", parseOffsetErr)
	}

	parseChecksumPayload := append([]byte(nil), parseMountPayload...)
	parseChecksumPayload[12] ^= 0xFF
	if _, parseErr := ParseBinaryMountEnvelope(parseChecksumPayload); parseErr == nil {
		parseT.Fatal("expected binary mount checksum mismatch to fail")
	}

	parseTrailingPayload := append(append([]byte(nil), parseMountPayload...), byte('x'))
	storeBinaryEnvelopeHeaderForMutation(parseTrailingPayload)
	if _, parseErr := ParseBinaryMountEnvelope(parseTrailingPayload); parseErr == nil {
		parseT.Fatal("expected binary mount trailing payload bytes to fail")
	}

	parseRegionMismatchPayload := append([]byte(nil), parseMountPayload...)
	copy(parseRegionMismatchPayload[parseRegionOffset:parseRegionOffset+parseRegionLength], []byte("region-2"))
	storeBinaryEnvelopeHeaderForMutation(parseRegionMismatchPayload)
	if _, parseErr := ParseBinaryMountEnvelope(parseRegionMismatchPayload); parseErr == nil {
		parseT.Fatal("expected binary mount outer region mismatch to fail")
	}

	parseRendererPayload := append([]byte(nil), parseMountPayload...)
	for parseIndex := range parseRendererLength {
		parseRendererPayload[parseRendererOffset+parseIndex] = ' '
	}
	storeBinaryEnvelopeHeaderForMutation(parseRendererPayload)
	if _, parseErr := ParseBinaryMountEnvelope(parseRendererPayload); parseErr == nil {
		parseT.Fatal("expected binary mount invalid renderer ID to fail")
	}
}

// TestParseBinaryUpdateEnvelopeRejectsCorruptedChecksumTrailingBytesAndOuterFieldMismatches verifies binary update decode rejects checksum corruption, trailing bytes, zero input versions, and outer metadata mismatches.
func TestParseBinaryUpdateEnvelopeRejectsCorruptedChecksumTrailingBytesAndOuterFieldMismatches(parseT *testing.T) {
	parseUpdatePayload := buildBinaryUpdatePayloadForMutationTest(parseT)
	parseRegionOffset, parseRegionLength, parseInputVersionOffset, parseOffsetErr := buildBinaryUpdateMutationOffsets(parseUpdatePayload)
	if parseOffsetErr != nil {
		parseT.Fatalf("buildBinaryUpdateMutationOffsets returned error: %v", parseOffsetErr)
	}

	parseChecksumPayload := append([]byte(nil), parseUpdatePayload...)
	parseChecksumPayload[12] ^= 0xFF
	if _, parseErr := ParseBinaryUpdateEnvelope(parseChecksumPayload); parseErr == nil {
		parseT.Fatal("expected binary update checksum mismatch to fail")
	}

	parseTrailingPayload := append(append([]byte(nil), parseUpdatePayload...), byte('x'))
	storeBinaryEnvelopeHeaderForMutation(parseTrailingPayload)
	if _, parseErr := ParseBinaryUpdateEnvelope(parseTrailingPayload); parseErr == nil {
		parseT.Fatal("expected binary update trailing payload bytes to fail")
	}

	parseZeroVersionPayload := append([]byte(nil), parseUpdatePayload...)
	binary.LittleEndian.PutUint64(parseZeroVersionPayload[parseInputVersionOffset:parseInputVersionOffset+8], 0)
	storeBinaryEnvelopeHeaderForMutation(parseZeroVersionPayload)
	if _, parseErr := ParseBinaryUpdateEnvelope(parseZeroVersionPayload); parseErr == nil {
		parseT.Fatal("expected binary update zero input version to fail")
	}

	parseRegionMismatchPayload := append([]byte(nil), parseUpdatePayload...)
	copy(parseRegionMismatchPayload[parseRegionOffset:parseRegionOffset+parseRegionLength], []byte("region-2"))
	storeBinaryEnvelopeHeaderForMutation(parseRegionMismatchPayload)
	if _, parseErr := ParseBinaryUpdateEnvelope(parseRegionMismatchPayload); parseErr == nil {
		parseT.Fatal("expected binary update outer region mismatch to fail")
	}

	parseVersionMismatchPayload := append([]byte(nil), parseUpdatePayload...)
	binary.LittleEndian.PutUint64(parseVersionMismatchPayload[parseInputVersionOffset:parseInputVersionOffset+8], 3)
	storeBinaryEnvelopeHeaderForMutation(parseVersionMismatchPayload)
	if _, parseErr := ParseBinaryUpdateEnvelope(parseVersionMismatchPayload); parseErr == nil {
		parseT.Fatal("expected binary update outer input version mismatch to fail")
	}
}
