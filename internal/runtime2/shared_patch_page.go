package runtime2

import "fmt"

// SharedPatchPage stores one mutable shared-memory-like page buffer and publication generation state for patch payloads.
type SharedPatchPage struct {
	storeSharedPatchPageBuffer     []byte
	storeSharedPatchPageGeneration uint64
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
	parseHeaderBytes, parseHeaderErr := BuildSharedSnapshotPageHeader(
		SharedSnapshotPageKindPatch,
		parseGeneration,
		uint32(len(parsePayload)),
		SharedSnapshotPageStatusWriting,
	)
	if parseHeaderErr != nil {
		return SharedSnapshotPageHeader{}, parseHeaderErr
	}
	copy(parseSharedPatchPage.storeSharedPatchPageBuffer[:sharedSnapshotPageHeaderSize], parseHeaderBytes)
	parseBodyStart := sharedSnapshotPageHeaderSize
	parseBodyEnd := parseBodyStart + len(parsePayload)
	if parseBodyEnd > len(parseSharedPatchPage.storeSharedPatchPageBuffer) {
		return SharedSnapshotPageHeader{}, fmt.Errorf("runtime2: shared patch publish payload length %d exceeds page capacity", len(parsePayload))
	}
	copy(parseSharedPatchPage.storeSharedPatchPageBuffer[parseBodyStart:parseBodyEnd], parsePayload)
	parseCompleteHeaderBytes, parseCompleteHeaderErr := BuildSharedSnapshotPageHeader(
		SharedSnapshotPageKindPatch,
		parseGeneration,
		uint32(len(parsePayload)),
		SharedSnapshotPageStatusComplete,
	)
	if parseCompleteHeaderErr != nil {
		return SharedSnapshotPageHeader{}, parseCompleteHeaderErr
	}
	copy(parseSharedPatchPage.storeSharedPatchPageBuffer[:sharedSnapshotPageHeaderSize], parseCompleteHeaderBytes)
	parseSharedPatchPage.storeSharedPatchPageGeneration = parseGeneration
	return ParseSharedSnapshotPageHeader(parseSharedPatchPage.storeSharedPatchPageBuffer[:sharedSnapshotPageHeaderSize])
}

// GetSharedPatchReadPayload reads the current completed shared patch payload bytes.
func (parseSharedPatchPage *SharedPatchPage) GetSharedPatchReadPayload() ([]byte, error) {
	if parseSharedPatchPage == nil {
		return nil, fmt.Errorf("runtime2: shared patch page is nil")
	}
	parseHeader, parseHeaderErr := ParseSharedSnapshotPageHeader(parseSharedPatchPage.storeSharedPatchPageBuffer[:sharedSnapshotPageHeaderSize])
	if parseHeaderErr != nil {
		return nil, parseHeaderErr
	}
	if parseHeader.Kind != SharedSnapshotPageKindPatch {
		return nil, fmt.Errorf("runtime2: shared patch page kind %d is unsupported", parseHeader.Kind)
	}
	if parseHeader.Status != SharedSnapshotPageStatusComplete {
		return nil, fmt.Errorf("runtime2: shared patch read detected torn write generation=%d", parseHeader.Generation)
	}
	parseBodyStart := sharedSnapshotPageHeaderSize
	parseBodyEnd := parseBodyStart + int(parseHeader.PayloadLength)
	if parseBodyEnd > len(parseSharedPatchPage.storeSharedPatchPageBuffer) {
		return nil, fmt.Errorf("runtime2: shared patch payload length %d exceeds page capacity", parseHeader.PayloadLength)
	}
	parsePayload := make([]byte, parseHeader.PayloadLength)
	copy(parsePayload, parseSharedPatchPage.storeSharedPatchPageBuffer[parseBodyStart:parseBodyEnd])
	return parsePayload, nil
}
