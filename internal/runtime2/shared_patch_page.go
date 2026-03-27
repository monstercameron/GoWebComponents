package runtime2

import (
	"encoding/binary"
	"fmt"
)

// SharedPatchPage stores one mutable shared-memory-like page buffer and publication generation state for patch payloads.
type SharedPatchPage struct {
	storeSharedPatchPageBuffer     []byte
	storeSharedPatchPageGeneration uint64
}

// setPatchPageHeader writes one validated shared patch page header into page storage and returns the same header values.
func (parseSharedPatchPage *SharedPatchPage) setPatchPageHeader(parseGeneration uint64, parsePayloadLength uint32, parseStatus SharedSnapshotPageStatus) SharedSnapshotPageHeader {
	parseHeader := parseSharedPatchPage.storeSharedPatchPageBuffer[:sharedSnapshotPageHeaderSize]
	parseHeader[0] = sharedSnapshotPageHeaderMagic[0]
	parseHeader[1] = sharedSnapshotPageHeaderMagic[1]
	parseHeader[2] = sharedSnapshotPageHeaderMagic[2]
	parseHeader[3] = sharedSnapshotPageHeaderMagic[3]
	binary.LittleEndian.PutUint16(parseHeader[4:6], sharedSnapshotPageHeaderVersion)
	binary.LittleEndian.PutUint16(parseHeader[6:8], uint16(SharedSnapshotPageKindPatch))
	binary.LittleEndian.PutUint64(parseHeader[8:16], parseGeneration)
	binary.LittleEndian.PutUint32(parseHeader[16:20], parsePayloadLength)
	binary.LittleEndian.PutUint32(parseHeader[20:24], uint32(parseStatus))
	return SharedSnapshotPageHeader{
		Kind:          SharedSnapshotPageKindPatch,
		Generation:    parseGeneration,
		PayloadLength: parsePayloadLength,
		Status:        parseStatus,
	}
}

// parsePatchPageHeader decodes and validates one shared patch page header from page storage.
func (parseSharedPatchPage *SharedPatchPage) parsePatchPageHeader() (SharedSnapshotPageHeader, error) {
	parseHeader := parseSharedPatchPage.storeSharedPatchPageBuffer[:sharedSnapshotPageHeaderSize]
	if parseHeader[0] != sharedSnapshotPageHeaderMagic[0] ||
		parseHeader[1] != sharedSnapshotPageHeaderMagic[1] ||
		parseHeader[2] != sharedSnapshotPageHeaderMagic[2] ||
		parseHeader[3] != sharedSnapshotPageHeaderMagic[3] {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot page magic %q is invalid", string(parseHeader[0:4]))
	}
	parseVersion := binary.LittleEndian.Uint16(parseHeader[4:6])
	if parseVersion != sharedSnapshotPageHeaderVersion {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot page version %d is unsupported", parseVersion)
	}
	parseKind := SharedSnapshotPageKind(binary.LittleEndian.Uint16(parseHeader[6:8]))
	if parseKind != SharedSnapshotPageKindPatch {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared patch page kind %d is unsupported", parseKind)
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

// BuildSharedPatchPage creates one page buffer used for shared patch payload publication flows.
func BuildSharedPatchPage(parseCapacity int) (*SharedPatchPage, error) {
	if parseCapacity < sharedSnapshotPageHeaderSize {
		return nil, fmt.Errorf("runtime2: shared patch page capacity %d is smaller than header size %d", parseCapacity, sharedSnapshotPageHeaderSize)
	}
	return &SharedPatchPage{
		storeSharedPatchPageBuffer: make([]byte, parseCapacity),
	}, nil
}

// HandleSharedPatchPublishPayload writes one patch payload into the page body and completes one publish generation.
func (parseSharedPatchPage *SharedPatchPage) HandleSharedPatchPublishPayload(parsePayload []byte) (SharedSnapshotPageHeader, error) {
	if parseSharedPatchPage == nil {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared patch page is nil")
	}
	parseGeneration := parseSharedPatchPage.storeSharedPatchPageGeneration + 1
	parseSharedPatchPage.setPatchPageHeader(
		parseGeneration,
		uint32(len(parsePayload)),
		SharedSnapshotPageStatusWriting,
	)
	parseBodyStart := sharedSnapshotPageHeaderSize
	parseBodyEnd := parseBodyStart + len(parsePayload)
	if parseBodyEnd > len(parseSharedPatchPage.storeSharedPatchPageBuffer) {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared patch publish payload length %d exceeds page capacity", len(parsePayload))
	}
	copy(parseSharedPatchPage.storeSharedPatchPageBuffer[parseBodyStart:parseBodyEnd], parsePayload)
	parseHeader := parseSharedPatchPage.setPatchPageHeader(
		parseGeneration,
		uint32(len(parsePayload)),
		SharedSnapshotPageStatusComplete,
	)
	parseSharedPatchPage.storeSharedPatchPageGeneration = parseGeneration
	return parseHeader, nil
}

// getPatchPageReadPayloadSpan returns one validated shared patch payload span backed by page storage.
func (parseSharedPatchPage *SharedPatchPage) getPatchPageReadPayloadSpan() ([]byte, error) {
	if parseSharedPatchPage == nil {
		return nil, fmt.Errorf("runtime2: shared patch page is nil")
	}
	parseHeader, parseHeaderErr := parseSharedPatchPage.parsePatchPageHeader()
	if parseHeaderErr != nil {
		return nil, parseHeaderErr
	}
	if parseHeader.Status != SharedSnapshotPageStatusComplete {
		return nil, fmt.Errorf("runtime2: shared patch read detected torn write generation=%d", parseHeader.Generation)
	}
	parseBodyStart := sharedSnapshotPageHeaderSize
	parseBodyEnd := parseBodyStart + int(parseHeader.PayloadLength)
	if parseBodyEnd > len(parseSharedPatchPage.storeSharedPatchPageBuffer) {
		return nil, fmt.Errorf("runtime2: shared patch payload length %d exceeds page capacity", parseHeader.PayloadLength)
	}
	return parseSharedPatchPage.storeSharedPatchPageBuffer[parseBodyStart:parseBodyEnd], nil
}

// GetSharedPatchReadPayload reads the current completed shared patch payload bytes.
func (parseSharedPatchPage *SharedPatchPage) GetSharedPatchReadPayload() ([]byte, error) {
	parsePayloadSpan, parsePayloadErr := parseSharedPatchPage.getPatchPageReadPayloadSpan()
	if parsePayloadErr != nil {
		return nil, parsePayloadErr
	}
	parsePayload := make([]byte, len(parsePayloadSpan))
	copy(parsePayload, parsePayloadSpan)
	return parsePayload, nil
}
