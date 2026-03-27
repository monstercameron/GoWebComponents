package runtime2

import (
	"encoding/binary"
	"fmt"
)

const (
	binaryEnvelopeHeaderSize    = 16
	binaryEnvelopeHeaderMagic   = "GWB1"
	binaryEnvelopeHeaderVersion = uint16(1)
	binaryEnvelopeSectionCount  = uint8(1)
)

// BinaryEnvelopeKind identifies the payload kind carried by one binary envelope.
type BinaryEnvelopeKind uint8

const (
	// BinaryEnvelopeKindSnapshot identifies snapshot envelope payloads.
	BinaryEnvelopeKindSnapshot BinaryEnvelopeKind = 1
	// BinaryEnvelopeKindMount identifies mount envelope payloads.
	BinaryEnvelopeKindMount BinaryEnvelopeKind = 2
	// BinaryEnvelopeKindUpdate identifies update envelope payloads.
	BinaryEnvelopeKindUpdate BinaryEnvelopeKind = 3
)

// BinaryEnvelopeHeader stores decoded binary envelope header fields.
type BinaryEnvelopeHeader struct {
	Kind          BinaryEnvelopeKind
	SectionCount  uint8
	PayloadLength uint32
	Checksum      uint32
}

// BuildBinaryEnvelopeHeader encodes a binary envelope header with magic, version, kind, payload length, and checksum.
func BuildBinaryEnvelopeHeader(parseKind BinaryEnvelopeKind, parsePayloadLength uint32, parseChecksum uint32) ([]byte, error) {
	if parseKind == 0 {
		return nil, fmt.Errorf("runtime2: binary envelope kind is required")
	}
	parseHeader := make([]byte, binaryEnvelopeHeaderSize)
	copy(parseHeader[0:4], []byte(binaryEnvelopeHeaderMagic))
	binary.LittleEndian.PutUint16(parseHeader[4:6], binaryEnvelopeHeaderVersion)
	parseHeader[6] = byte(parseKind)
	parseHeader[7] = byte(binaryEnvelopeSectionCount)
	binary.LittleEndian.PutUint32(parseHeader[8:12], parsePayloadLength)
	binary.LittleEndian.PutUint32(parseHeader[12:16], parseChecksum)
	return parseHeader, nil
}

// ParseBinaryEnvelopeHeader decodes and validates a binary envelope header.
func ParseBinaryEnvelopeHeader(parsePayload []byte) (BinaryEnvelopeHeader, error) {
	if len(parsePayload) < binaryEnvelopeHeaderSize {
		return BinaryEnvelopeHeader{}, fmt.Errorf("runtime2: binary envelope header is truncated")
	}
	if string(parsePayload[0:4]) != binaryEnvelopeHeaderMagic {
		return BinaryEnvelopeHeader{}, fmt.Errorf("runtime2: binary envelope magic %q is invalid", string(parsePayload[0:4]))
	}
	parseVersion := binary.LittleEndian.Uint16(parsePayload[4:6])
	if parseVersion != binaryEnvelopeHeaderVersion {
		return BinaryEnvelopeHeader{}, fmt.Errorf("runtime2: binary envelope version %d is unsupported", parseVersion)
	}
	parseKind := BinaryEnvelopeKind(parsePayload[6])
	if parseKind != BinaryEnvelopeKindSnapshot && parseKind != BinaryEnvelopeKindMount && parseKind != BinaryEnvelopeKindUpdate {
		return BinaryEnvelopeHeader{}, fmt.Errorf("runtime2: binary envelope kind %d is unsupported", parseKind)
	}
	parseSectionCount := parsePayload[7]
	if parseSectionCount != binaryEnvelopeSectionCount {
		return BinaryEnvelopeHeader{}, fmt.Errorf("runtime2: binary envelope section count %d is unsupported", parseSectionCount)
	}
	return BinaryEnvelopeHeader{
		Kind:          parseKind,
		SectionCount:  parseSectionCount,
		PayloadLength: binary.LittleEndian.Uint32(parsePayload[8:12]),
		Checksum:      binary.LittleEndian.Uint32(parsePayload[12:16]),
	}, nil
}
