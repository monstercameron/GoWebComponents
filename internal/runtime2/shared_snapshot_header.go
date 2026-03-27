package runtime2

import (
	"encoding/binary"
	"fmt"
)

const (
	sharedSnapshotPageHeaderSize    = 24
	sharedSnapshotPageHeaderMagic   = "GSP1"
	sharedSnapshotPageHeaderVersion = uint16(1)
)

// SharedSnapshotPageKind identifies the payload kind stored in one shared snapshot page.
type SharedSnapshotPageKind uint16

const (
	// SharedSnapshotPageKindSnapshot identifies snapshot payload pages.
	SharedSnapshotPageKindSnapshot SharedSnapshotPageKind = 1
)

// SharedSnapshotPageStatus identifies publication status for one shared snapshot page.
type SharedSnapshotPageStatus uint32

const (
	// SharedSnapshotPageStatusWriting marks an in-progress publish.
	SharedSnapshotPageStatusWriting SharedSnapshotPageStatus = 0
	// SharedSnapshotPageStatusComplete marks a fully published payload.
	SharedSnapshotPageStatusComplete SharedSnapshotPageStatus = 1
)

// SharedSnapshotPageHeader stores decoded shared snapshot page header fields.
type SharedSnapshotPageHeader struct {
	Kind          SharedSnapshotPageKind
	Generation    uint64
	PayloadLength uint32
	Status        SharedSnapshotPageStatus
}

// BuildSharedSnapshotPageHeader encodes one shared snapshot page header.
func BuildSharedSnapshotPageHeader(parseKind SharedSnapshotPageKind, parseGeneration uint64, parsePayloadLength uint32, parseStatus SharedSnapshotPageStatus) ([]byte, error) {
	if parseKind == 0 {
		return nil, fmt.Errorf("runtime2: shared snapshot page kind is required")
	}
	parseHeader := make([]byte, sharedSnapshotPageHeaderSize)
	copy(parseHeader[0:4], []byte(sharedSnapshotPageHeaderMagic))
	binary.LittleEndian.PutUint16(parseHeader[4:6], sharedSnapshotPageHeaderVersion)
	binary.LittleEndian.PutUint16(parseHeader[6:8], uint16(parseKind))
	binary.LittleEndian.PutUint64(parseHeader[8:16], parseGeneration)
	binary.LittleEndian.PutUint32(parseHeader[16:20], parsePayloadLength)
	binary.LittleEndian.PutUint32(parseHeader[20:24], uint32(parseStatus))
	return parseHeader, nil
}

// ParseSharedSnapshotPageHeader decodes and validates one shared snapshot page header.
func ParseSharedSnapshotPageHeader(parseHeader []byte) (SharedSnapshotPageHeader, error) {
	if len(parseHeader) < sharedSnapshotPageHeaderSize {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot page header is truncated")
	}
	if string(parseHeader[0:4]) != sharedSnapshotPageHeaderMagic {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot page magic %q is invalid", string(parseHeader[0:4]))
	}
	parseVersion := binary.LittleEndian.Uint16(parseHeader[4:6])
	if parseVersion != sharedSnapshotPageHeaderVersion {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot page version %d is unsupported", parseVersion)
	}
	parseKind := SharedSnapshotPageKind(binary.LittleEndian.Uint16(parseHeader[6:8]))
	if parseKind != SharedSnapshotPageKindSnapshot {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot page kind %d is unsupported", parseKind)
	}
	parseStatus := SharedSnapshotPageStatus(binary.LittleEndian.Uint32(parseHeader[20:24]))
	if parseStatus != SharedSnapshotPageStatusWriting && parseStatus != SharedSnapshotPageStatusComplete {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot page status %d is unsupported", parseStatus)
	}
	return SharedSnapshotPageHeader{
		Kind:          parseKind,
		Generation:    binary.LittleEndian.Uint64(parseHeader[8:16]),
		PayloadLength: binary.LittleEndian.Uint32(parseHeader[16:20]),
		Status:        parseStatus,
	}, nil
}
