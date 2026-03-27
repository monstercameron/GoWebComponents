package runtime2

import (
	"encoding/json"
	"fmt"
)

// SharedSnapshotPage stores one mutable shared-memory-like page buffer and publication generation state.
type SharedSnapshotPage struct {
	storeSharedSnapshotPageBuffer     []byte
	storeSharedSnapshotPageGeneration uint64
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
	parseHeaderBytes, parseErr := BuildSharedSnapshotPageHeader(SharedSnapshotPageKindSnapshot, parseGeneration, parsePayloadLength, SharedSnapshotPageStatusWriting)
	if parseErr != nil {
		return SharedSnapshotPageHeader{}, parseErr
	}
	copy(parseSharedSnapshotPage.storeSharedSnapshotPageBuffer[:sharedSnapshotPageHeaderSize], parseHeaderBytes)
	parseSharedSnapshotPage.storeSharedSnapshotPageGeneration = parseGeneration
	return ParseSharedSnapshotPageHeader(parseSharedSnapshotPage.storeSharedSnapshotPageBuffer[:sharedSnapshotPageHeaderSize])
}

// HandleSharedSnapshotPublishComplete finalizes one started publish generation by flipping page status to complete.
func (parseSharedSnapshotPage *SharedSnapshotPage) HandleSharedSnapshotPublishComplete(parseGeneration uint64) (SharedSnapshotPageHeader, error) {
	if parseSharedSnapshotPage == nil {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot page is nil")
	}
	parseHeaderCurrent, parseErr := ParseSharedSnapshotPageHeader(parseSharedSnapshotPage.storeSharedSnapshotPageBuffer[:sharedSnapshotPageHeaderSize])
	if parseErr != nil {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot publish complete requires a begun publish: %w", parseErr)
	}
	if parseHeaderCurrent.Status != SharedSnapshotPageStatusWriting {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot publish complete requires writing status")
	}
	if parseHeaderCurrent.Generation != parseGeneration {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot publish generation mismatch expected=%d actual=%d", parseHeaderCurrent.Generation, parseGeneration)
	}
	parseHeaderBytes, parseErr := BuildSharedSnapshotPageHeader(parseHeaderCurrent.Kind, parseHeaderCurrent.Generation, parseHeaderCurrent.PayloadLength, SharedSnapshotPageStatusComplete)
	if parseErr != nil {
		return SharedSnapshotPageHeader{}, parseErr
	}
	copy(parseSharedSnapshotPage.storeSharedSnapshotPageBuffer[:sharedSnapshotPageHeaderSize], parseHeaderBytes)
	return ParseSharedSnapshotPageHeader(parseSharedSnapshotPage.storeSharedSnapshotPageBuffer[:sharedSnapshotPageHeaderSize])
}

// HandleSharedSnapshotReadHeader reads and validates the current page header for reader-side consumption.
func (parseSharedSnapshotPage *SharedSnapshotPage) HandleSharedSnapshotReadHeader() (SharedSnapshotPageHeader, error) {
	if parseSharedSnapshotPage == nil {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared snapshot page is nil")
	}
	parseHeader, parseErr := ParseSharedSnapshotPageHeader(parseSharedSnapshotPage.storeSharedSnapshotPageBuffer[:sharedSnapshotPageHeaderSize])
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
