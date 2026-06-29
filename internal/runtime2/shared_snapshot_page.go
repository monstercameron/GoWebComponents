package runtime2

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

// SharedSnapshotPage stores one mutable shared-memory-like page buffer and publication generation state.
type SharedSnapshotPage struct {
	storeSharedSnapshotPageBuffer     []byte
	storeSharedSnapshotPageGeneration uint64
}

// setSnapshotPageHeader writes one validated shared snapshot page header into page storage and returns the same header values.
func (parseSharedSnapshotPage *SharedSnapshotPage) setSnapshotPageHeader(parseKind SharedSnapshotPageKind, parseGeneration uint64, parsePayloadLength uint32, parseStatus SharedSnapshotPageStatus) SharedSnapshotPageHeader {
	parseHeader := parseSharedSnapshotPage.storeSharedSnapshotPageBuffer[:sharedSnapshotPageHeaderSize]
	parseHeader[0] = sharedSnapshotPageHeaderMagic[0]
	parseHeader[1] = sharedSnapshotPageHeaderMagic[1]
	parseHeader[2] = sharedSnapshotPageHeaderMagic[2]
	parseHeader[3] = sharedSnapshotPageHeaderMagic[3]
	binary.LittleEndian.PutUint16(parseHeader[4:6], sharedSnapshotPageHeaderVersion)
	binary.LittleEndian.PutUint16(parseHeader[6:8], uint16(parseKind))
	binary.LittleEndian.PutUint64(parseHeader[8:16], parseGeneration)
	binary.LittleEndian.PutUint32(parseHeader[16:20], parsePayloadLength)
	binary.LittleEndian.PutUint32(parseHeader[20:24], uint32(parseStatus))
	return SharedSnapshotPageHeader{
		Kind:          parseKind,
		Generation:    parseGeneration,
		PayloadLength: parsePayloadLength,
		Status:        parseStatus,
	}
}

// parseSnapshotPageHeader decodes and validates one shared snapshot page header from page storage.
func (parseSharedSnapshotPage *SharedSnapshotPage) parseSnapshotPageHeader() (SharedSnapshotPageHeader, error) {
	parseHeader := parseSharedSnapshotPage.storeSharedSnapshotPageBuffer[:sharedSnapshotPageHeaderSize]
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
	if parseKind != SharedSnapshotPageKindSnapshot && parseKind != SharedSnapshotPageKindPatch {
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

// BuildSharedSnapshotPage creates one page buffer used for shared snapshot publication flows.
func BuildSharedSnapshotPage(parseCapacity int) (*SharedSnapshotPage, error) {
	if parseCapacity < sharedSnapshotPageHeaderSize {
		return nil, fmt.Errorf("runtime2: shared snapshot page capacity %d is smaller than header size %d", parseCapacity, sharedSnapshotPageHeaderSize)
	}
	return &SharedSnapshotPage{
		storeSharedSnapshotPageBuffer: make([]byte, parseCapacity),
	}, nil
}

// HandleSharedSnapshotPublishBegin starts a new publish generation and writes a writing-status page header.
func (parseSharedSnapshotPage *SharedSnapshotPage) HandleSharedSnapshotPublishBegin(parsePayloadLength uint32) (SharedSnapshotPageHeader, error) {
	if parseSharedSnapshotPage == nil {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot page is nil")
	}
	parseGeneration := parseSharedSnapshotPage.storeSharedSnapshotPageGeneration + 1
	parseHeader := parseSharedSnapshotPage.setSnapshotPageHeader(
		SharedSnapshotPageKindSnapshot,
		parseGeneration,
		parsePayloadLength,
		SharedSnapshotPageStatusWriting,
	)
	parseSharedSnapshotPage.storeSharedSnapshotPageGeneration = parseGeneration
	return parseHeader, nil
}

// HandleSharedSnapshotPublishComplete finalizes one started publish generation by flipping page status to complete.
func (parseSharedSnapshotPage *SharedSnapshotPage) HandleSharedSnapshotPublishComplete(parseGeneration uint64) (SharedSnapshotPageHeader, error) {
	if parseSharedSnapshotPage == nil {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot page is nil")
	}
	parseHeaderCurrent, parseErr := parseSharedSnapshotPage.parseSnapshotPageHeader()
	if parseErr != nil {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot publish complete requires a begun publish: %w", parseErr)
	}
	if parseHeaderCurrent.Status != SharedSnapshotPageStatusWriting {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot publish complete requires writing status")
	}
	if parseHeaderCurrent.Generation != parseGeneration {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot publish generation mismatch expected=%d actual=%d", parseHeaderCurrent.Generation, parseGeneration)
	}
	return parseSharedSnapshotPage.setSnapshotPageHeader(
		parseHeaderCurrent.Kind,
		parseHeaderCurrent.Generation,
		parseHeaderCurrent.PayloadLength,
		SharedSnapshotPageStatusComplete,
	), nil
}

// HandleSharedSnapshotReadHeader reads and validates the current page header for reader-side consumption.
func (parseSharedSnapshotPage *SharedSnapshotPage) HandleSharedSnapshotReadHeader() (SharedSnapshotPageHeader, error) {
	if parseSharedSnapshotPage == nil {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot page is nil")
	}
	parseHeader, parseErr := parseSharedSnapshotPage.parseSnapshotPageHeader()
	if parseErr != nil {
		return SharedSnapshotPageHeader{}, parseErr
	}
	if parseHeader.Status == SharedSnapshotPageStatusWriting {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot read detected torn write generation=%d", parseHeader.Generation)
	}
	return parseHeader, nil
}

// HandleSharedSnapshotReadHeaderAtGeneration reads one completed page header and enforces expected generation matching.
func (parseSharedSnapshotPage *SharedSnapshotPage) HandleSharedSnapshotReadHeaderAtGeneration(parseExpectedGeneration uint64) (SharedSnapshotPageHeader, error) {
	if parseExpectedGeneration == 0 {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: expected shared snapshot generation is required")
	}
	parseHeader, parseErr := parseSharedSnapshotPage.HandleSharedSnapshotReadHeader()
	if parseErr != nil {
		return SharedSnapshotPageHeader{}, parseErr
	}
	if parseHeader.Generation != parseExpectedGeneration {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot stale generation expected=%d actual=%d", parseExpectedGeneration, parseHeader.Generation)
	}
	return parseHeader, nil
}

// HandleSharedSnapshotPublishPayload writes one payload into the page body and completes the publish generation.
func (parseSharedSnapshotPage *SharedSnapshotPage) HandleSharedSnapshotPublishPayload(parsePayload []byte) (SharedSnapshotPageHeader, error) {
	parseHeaderBegin, parseErr := parseSharedSnapshotPage.HandleSharedSnapshotPublishBegin(uint32(len(parsePayload)))
	if parseErr != nil {
		return SharedSnapshotPageHeader{}, parseErr
	}
	parseBodyStart := sharedSnapshotPageHeaderSize
	parseBodyEnd := parseBodyStart + len(parsePayload)
	if parseBodyEnd > len(parseSharedSnapshotPage.storeSharedSnapshotPageBuffer) {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot publish payload length %d exceeds page capacity", len(parsePayload))
	}
	copy(parseSharedSnapshotPage.storeSharedSnapshotPageBuffer[parseBodyStart:parseBodyEnd], parsePayload)
	return parseSharedSnapshotPage.HandleSharedSnapshotPublishComplete(parseHeaderBegin.Generation)
}

// GetSharedSnapshotReadPayload reads the current completed page payload bytes.
func (parseSharedSnapshotPage *SharedSnapshotPage) GetSharedSnapshotReadPayload() ([]byte, error) {
	parseHeader, parseErr := parseSharedSnapshotPage.HandleSharedSnapshotReadHeader()
	if parseErr != nil {
		return nil, parseErr
	}
	parseBodyStart := sharedSnapshotPageHeaderSize
	parseBodyEnd := parseBodyStart + int(parseHeader.PayloadLength)
	if parseBodyEnd > len(parseSharedSnapshotPage.storeSharedSnapshotPageBuffer) {
		return nil, fmt.Errorf("runtime2: shared snapshot payload length %d exceeds page capacity", parseHeader.PayloadLength)
	}
	parsePayload := make([]byte, parseHeader.PayloadLength)
	copy(parsePayload, parseSharedSnapshotPage.storeSharedSnapshotPageBuffer[parseBodyStart:parseBodyEnd])
	return parsePayload, nil
}

// ParseSharedSnapshotEnvelope decodes the current completed shared-page payload back into a validated SnapshotEnvelope.
func (parseSharedSnapshotPage *SharedSnapshotPage) ParseSharedSnapshotEnvelope() (SnapshotEnvelope, error) {
	parsePayload, parseErr := parseSharedSnapshotPage.GetSharedSnapshotReadPayload()
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	var parseEnvelope SnapshotEnvelope
	if parseErr := json.Unmarshal(parsePayload, &parseEnvelope); parseErr != nil {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode shared snapshot envelope: %w", parseErr)
	}
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}
