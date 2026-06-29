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
	return appendBinaryEnvelopeHeader(make([]byte, 0, binaryEnvelopeHeaderSize), parseKind, parsePayloadLength, parseChecksum)
}

// writeBinaryEnvelopeHeaderAt writes one binary envelope header into dst starting at offset 0.
// dst must have len >= binaryEnvelopeHeaderSize.
func writeBinaryEnvelopeHeaderAt(dst []byte, parseKind BinaryEnvelopeKind, parsePayloadLength uint32, parseChecksum uint32) {
	dst[0] = binaryEnvelopeHeaderMagic[0]
	dst[1] = binaryEnvelopeHeaderMagic[1]
	dst[2] = binaryEnvelopeHeaderMagic[2]
	dst[3] = binaryEnvelopeHeaderMagic[3]
	binary.LittleEndian.PutUint16(dst[4:6], binaryEnvelopeHeaderVersion)
	dst[6] = byte(parseKind)
	dst[7] = byte(binaryEnvelopeSectionCount)
	binary.LittleEndian.PutUint32(dst[8:12], parsePayloadLength)
	binary.LittleEndian.PutUint32(dst[12:16], parseChecksum)
}

// appendBinaryEnvelopeHeader appends one encoded binary envelope header into dst and returns the extended slice.
func appendBinaryEnvelopeHeader(dst []byte, parseKind BinaryEnvelopeKind, parsePayloadLength uint32, parseChecksum uint32) ([]byte, error) {
	if parseKind == 0 {
		return nil, fmt.Errorf("runtime2: binary envelope kind is required")
	}
	dst = append(dst, binaryEnvelopeHeaderMagic[0], binaryEnvelopeHeaderMagic[1], binaryEnvelopeHeaderMagic[2], binaryEnvelopeHeaderMagic[3])
	dst = binary.LittleEndian.AppendUint16(dst, binaryEnvelopeHeaderVersion)
	dst = append(dst, byte(parseKind), byte(binaryEnvelopeSectionCount))
	dst = binary.LittleEndian.AppendUint32(dst, parsePayloadLength)
	dst = binary.LittleEndian.AppendUint32(dst, parseChecksum)
	return dst, nil
}

// ParseBinaryEnvelopeHeader decodes and validates a binary envelope header.
func ParseBinaryEnvelopeHeader(parsePayload []byte) (BinaryEnvelopeHeader, error) {
	if len(parsePayload) < binaryEnvelopeHeaderSize {
		return BinaryEnvelopeHeader{}, fmt.Errorf("runtime2: binary envelope header is truncated")
	}
	if parsePayload[0] != binaryEnvelopeHeaderMagic[0] ||
		parsePayload[1] != binaryEnvelopeHeaderMagic[1] ||
		parsePayload[2] != binaryEnvelopeHeaderMagic[2] ||
		parsePayload[3] != binaryEnvelopeHeaderMagic[3] {
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
