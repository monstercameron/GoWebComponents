package runtime2

import (
	"fmt"
	"hash/crc32"
)

// BuildBinarySnapshotEnvelope encodes one validated snapshot envelope using the runtime2 binary body and header.
func BuildBinarySnapshotEnvelope(parseEnvelope SnapshotEnvelope) ([]byte, error) {
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return nil, parseErr
	}
	// Pre-reserve header space; body is appended directly after to avoid a second allocation.
	parseRegionIDLen := 2 + len(string(parseEnvelope.RegionInstanceID))
	parseCap := binaryEnvelopeHeaderSize + parseRegionIDLen + 24 + 4 + 64 + 4 + 4 + (len(parseEnvelope.Sources)+1)*16
	parsePayload := make([]byte, binaryEnvelopeHeaderSize, parseCap)
	var parseErr error
	parsePayload, parseErr = appendBinarySnapshotBody(parsePayload, parseEnvelope)
	if parseErr != nil {
		return nil, parseErr
	}
	parseBodyLen := uint32(len(parsePayload) - binaryEnvelopeHeaderSize)
	parseChecksum := crc32.ChecksumIEEE(parsePayload[binaryEnvelopeHeaderSize:])
	writeBinaryEnvelopeHeaderAt(parsePayload, BinaryEnvelopeKindSnapshot, parseBodyLen, parseChecksum)
	return parsePayload, nil
}

// ParseBinarySnapshotEnvelope decodes and validates one binary snapshot envelope payload.
func ParseBinarySnapshotEnvelope(parsePayload []byte) (SnapshotEnvelope, error) {
	parseHeader, parseBody, parseErr := parseBinaryEnvelopePayload(parsePayload, BinaryEnvelopeKindSnapshot)
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseChecksum := crc32.ChecksumIEEE(parseBody)
	if parseChecksum != parseHeader.Checksum {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: binary snapshot checksum mismatch expected=%d actual=%d", parseHeader.Checksum, parseChecksum)
	}
	return ParseBinarySnapshotBody(parseBody)
}

// parseBinaryEnvelopePayload decodes one binary envelope header and returns the validated body span for the expected kind.
func parseBinaryEnvelopePayload(parsePayload []byte, parseExpectedKind BinaryEnvelopeKind) (BinaryEnvelopeHeader, []byte, error) {
	parseHeader, parseErr := ParseBinaryEnvelopeHeader(parsePayload)
	if parseErr != nil {
		return BinaryEnvelopeHeader{}, nil, parseErr
	}
	if parseHeader.Kind != parseExpectedKind {
		return BinaryEnvelopeHeader{}, nil, fmt.Errorf("runtime2: binary envelope kind %d does not match expected kind %d", parseHeader.Kind, parseExpectedKind)
	}
	parseBodyStart := binaryEnvelopeHeaderSize
	parseBodyEnd := parseBodyStart + int(parseHeader.PayloadLength)
	if parseBodyEnd > len(parsePayload) {
		return BinaryEnvelopeHeader{}, nil, fmt.Errorf("runtime2: binary envelope payload length %d exceeds available bytes", parseHeader.PayloadLength)
	}
	return parseHeader, parsePayload[parseBodyStart:parseBodyEnd], nil
}
