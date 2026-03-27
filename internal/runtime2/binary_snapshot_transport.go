package runtime2

import (
	"encoding/binary"
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
	parseBody, parseChecksumExpected, parseErr := parseBinarySnapshotEnvelopeBody(parsePayload)
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseChecksum := crc32.ChecksumIEEE(parseBody)
	if parseChecksum != parseChecksumExpected {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: binary snapshot checksum mismatch expected=%d actual=%d", parseChecksumExpected, parseChecksum)
	}
	return ParseBinarySnapshotBody(parseBody)
}

// parseBinarySnapshotEnvelopeBody decodes one snapshot-binary envelope header and returns the validated body span plus checksum field.
func parseBinarySnapshotEnvelopeBody(parsePayload []byte) ([]byte, uint32, error) {
	if len(parsePayload) < binaryEnvelopeHeaderSize {
		return nil, 0, fmt.Errorf("runtime2: binary envelope header is truncated")
	}
	if parsePayload[0] != binaryEnvelopeHeaderMagic[0] ||
		parsePayload[1] != binaryEnvelopeHeaderMagic[1] ||
		parsePayload[2] != binaryEnvelopeHeaderMagic[2] ||
		parsePayload[3] != binaryEnvelopeHeaderMagic[3] {
		return nil, 0, fmt.Errorf("runtime2: binary envelope magic %q is invalid", string(parsePayload[0:4]))
	}
	parseVersion := binary.LittleEndian.Uint16(parsePayload[4:6])
	if parseVersion != binaryEnvelopeHeaderVersion {
		return nil, 0, fmt.Errorf("runtime2: binary envelope version %d is unsupported", parseVersion)
	}
	parseKind := BinaryEnvelopeKind(parsePayload[6])
	if parseKind != BinaryEnvelopeKindSnapshot {
		return nil, 0, fmt.Errorf("runtime2: binary envelope kind %d does not match expected kind %d", parseKind, BinaryEnvelopeKindSnapshot)
	}
	parseSectionCount := parsePayload[7]
	if parseSectionCount != binaryEnvelopeSectionCount {
		return nil, 0, fmt.Errorf("runtime2: binary envelope section count %d is unsupported", parseSectionCount)
	}
	parsePayloadLength := binary.LittleEndian.Uint32(parsePayload[8:12])
	parseBodyStart := binaryEnvelopeHeaderSize
	parseBodyEnd := parseBodyStart + int(parsePayloadLength)
	if parseBodyEnd > len(parsePayload) {
		return nil, 0, fmt.Errorf("runtime2: binary envelope payload length %d exceeds available bytes", parsePayloadLength)
	}
	parseChecksum := binary.LittleEndian.Uint32(parsePayload[12:16])
	return parsePayload[parseBodyStart:parseBodyEnd], parseChecksum, nil
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
