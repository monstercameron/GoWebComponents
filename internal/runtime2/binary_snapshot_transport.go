package runtime2

import (
	"fmt"
	"hash/crc32"
)

// BuildBinarySnapshotEnvelope encodes one validated snapshot envelope using the runtime2 binary body and header.
func BuildBinarySnapshotEnvelope(parseEnvelope SnapshotEnvelope) ([]byte, error) {
	parseBody, parseErr := BuildBinarySnapshotBody(parseEnvelope)
	if parseErr != nil {
		return nil, parseErr
	}
	parseChecksum := crc32.ChecksumIEEE(parseBody)
	parseHeader, parseErr := BuildBinaryEnvelopeHeader(BinaryEnvelopeKindSnapshot, uint32(len(parseBody)), parseChecksum)
	if parseErr != nil {
		return nil, parseErr
	}
	parsePayload := make([]byte, 0, len(parseHeader)+len(parseBody))
	parsePayload = append(parsePayload, parseHeader...)
	parsePayload = append(parsePayload, parseBody...)
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
