package runtime2

import "fmt"

// SharedSnapshotDowngradeReason identifies why snapshot transport downgraded from shared memory.
type SharedSnapshotDowngradeReason string

const (
	// SharedSnapshotDowngradeReasonSharedMemoryUnavailable reports shared memory was not usable for this dispatch.
	SharedSnapshotDowngradeReasonSharedMemoryUnavailable SharedSnapshotDowngradeReason = "shared-memory-unavailable"
	// SharedSnapshotDowngradeReasonInvalidSharedPage reports shared-page publication failed and transport downgraded.
	SharedSnapshotDowngradeReasonInvalidSharedPage SharedSnapshotDowngradeReason = "invalid-shared-page"
)

// ParseSharedSnapshotDowngradeReason validates one downgrade-reason value.
func ParseSharedSnapshotDowngradeReason(parseRaw string) (SharedSnapshotDowngradeReason, error) {
	parseReason := SharedSnapshotDowngradeReason(parseRaw)
	switch parseReason {
	case "", SharedSnapshotDowngradeReasonSharedMemoryUnavailable, SharedSnapshotDowngradeReasonInvalidSharedPage:
		return parseReason, nil
	default:
		return "", fmt.Errorf("runtime2: shared snapshot downgrade reason %q is unsupported", parseRaw)
	}
}

// SharedSnapshotTransportResult stores one selected snapshot transport tier and payload metadata.
type SharedSnapshotTransportResult struct {
	GetRegionInstanceID RegionInstanceID
	GetEpoch            uint64
	GetInputVersion     uint64
	GetTransportTier    TransportTier
	GetMessagePayload   []byte
	GetSharedGeneration uint64
	GetDowngradeReason  SharedSnapshotDowngradeReason
}

// GetSharedSnapshotTransportTier selects the preferred snapshot transport tier from the capability contract and shared-page availability.
func GetSharedSnapshotTransportTier(parseCapabilityReport CapabilityReport, parseHasSharedSnapshotPage bool) (TransportTier, error) {
	if parseErr := ValidateCapabilityReport(parseCapabilityReport); parseErr != nil {
		return "", parseErr
	}
	if parseCapabilityReport.HasSharedMemoryTransportSupport && parseHasSharedSnapshotPage {
		return TransportTierSharedBuffer, nil
	}
	return TransportTierStructuredClone, nil
}

// BuildSharedSnapshotTransportResult selects a transport tier for one snapshot envelope and downgrades to message transport when shared memory is unavailable.
func BuildSharedSnapshotTransportResult(parseEnvelope SnapshotEnvelope, parseCapabilityReport CapabilityReport, parseSharedSnapshotPage *SharedSnapshotPage) (SharedSnapshotTransportResult, error) {
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return SharedSnapshotTransportResult{}, parseErr
	}
	parsePreferredTransportTier, parseErr := GetSharedSnapshotTransportTier(parseCapabilityReport, parseSharedSnapshotPage != nil)
	if parseErr != nil {
		return SharedSnapshotTransportResult{}, parseErr
	}
	parsePayload, parseErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseEnvelope)
	if parseErr != nil {
		return SharedSnapshotTransportResult{}, fmt.Errorf("runtime2: build snapshot message payload: %w", parseErr)
	}
	parseResult := SharedSnapshotTransportResult{
		GetRegionInstanceID: parseEnvelope.RegionInstanceID,
		GetEpoch:            parseEnvelope.Epoch,
		GetInputVersion:     parseEnvelope.InputVersion,
	}
	if parsePreferredTransportTier == TransportTierSharedBuffer {
		parseHeader, parseSharedErr := parseSharedSnapshotPage.HandleSharedSnapshotPublishPayload(parsePayload)
		if parseSharedErr == nil {
			parseResult.GetTransportTier = TransportTierSharedBuffer
			parseResult.GetSharedGeneration = parseHeader.Generation
			return parseResult, nil
		}
		parseResult.GetTransportTier = TransportTierStructuredClone
		parseResult.GetMessagePayload = parsePayload
		parseResult.GetDowngradeReason = SharedSnapshotDowngradeReasonInvalidSharedPage
		return parseResult, nil
	}
	parseResult.GetTransportTier = TransportTierStructuredClone
	parseResult.GetMessagePayload = parsePayload
	parseResult.GetDowngradeReason = SharedSnapshotDowngradeReasonSharedMemoryUnavailable
	return parseResult, nil
}
