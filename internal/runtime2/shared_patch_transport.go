package runtime2

import "fmt"

// SharedPatchDowngradeReason identifies why patch transport downgraded from shared buffer to message transport.
type SharedPatchDowngradeReason string

const (
	// SharedPatchDowngradeReasonSharedMemoryUnavailable reports shared memory was not usable for this patch dispatch.
	SharedPatchDowngradeReasonSharedMemoryUnavailable SharedPatchDowngradeReason = "shared-memory-unavailable"
	// SharedPatchDowngradeReasonInvalidSharedPage reports shared patch page publication failed and transport downgraded.
	SharedPatchDowngradeReasonInvalidSharedPage SharedPatchDowngradeReason = "invalid-shared-page"
)

// SharedPatchTransportResult stores one selected patch transport tier and payload metadata.
type SharedPatchTransportResult struct {
	GetRegionInstanceID RegionInstanceID
	GetEpoch            uint64
	GetPatchVersion     uint64
	GetTransportTier    TransportTier
	GetMessagePayload   []byte
	GetSharedGeneration uint64
	GetDowngradeReason  SharedPatchDowngradeReason
}

// BuildSharedPatchTransportResult selects one transport tier for one patch envelope and downgrades when shared-buffer publication is unavailable.
func BuildSharedPatchTransportResult(
	parseEnvelope StructuredClonePatchEnvelope,
	parseCapabilityReport CapabilityReport,
	parseSharedPatchPage *SharedPatchPage,
) (SharedPatchTransportResult, error) {
	if parseErr := ValidateStructuredClonePatchEnvelope(parseEnvelope); parseErr != nil {
		return SharedPatchTransportResult{}, parseErr
	}
	if parseErr := ValidateCapabilityReport(parseCapabilityReport); parseErr != nil {
		return SharedPatchTransportResult{}, parseErr
	}
	parseMessagePayload, parseMessagePayloadErr := BuildStructuredClonePatchEnvelopeJSON(parseEnvelope)
	if parseMessagePayloadErr != nil {
		return SharedPatchTransportResult{}, parseMessagePayloadErr
	}
	parseResult := SharedPatchTransportResult{
		GetRegionInstanceID: parseEnvelope.RegionInstanceID,
		GetEpoch:            parseEnvelope.Epoch,
		GetPatchVersion:     parseEnvelope.PatchVersion,
	}
	if parseCapabilityReport.HasSharedMemoryTransportSupport && parseSharedPatchPage != nil {
		parseHeader, parseSharedPublishErr := parseSharedPatchPage.HandleSharedPatchPublishPayload(parseMessagePayload)
		if parseSharedPublishErr == nil {
			parseResult.GetTransportTier = TransportTierSharedBuffer
			parseResult.GetSharedGeneration = parseHeader.Generation
			return parseResult, nil
		}
		parseResult.GetTransportTier = TransportTierStructuredClone
		parseResult.GetMessagePayload = parseMessagePayload
		parseResult.GetDowngradeReason = SharedPatchDowngradeReasonInvalidSharedPage
		return parseResult, nil
	}
	parseResult.GetTransportTier = TransportTierStructuredClone
	parseResult.GetMessagePayload = parseMessagePayload
	parseResult.GetDowngradeReason = SharedPatchDowngradeReasonSharedMemoryUnavailable
	return parseResult, nil
}

// ParseSharedPatchPayloadFromPage reads and decodes one structured-clone patch envelope from a shared patch page.
func ParseSharedPatchPayloadFromPage(parseSharedPatchPage *SharedPatchPage) (StructuredClonePatchEnvelope, error) {
	if parseSharedPatchPage == nil {
		return StructuredClonePatchEnvelope{}, fmt.Errorf("runtime2: shared patch page is nil")
	}
	parsePayload, parsePayloadErr := parseSharedPatchPage.GetSharedPatchReadPayload()
	if parsePayloadErr != nil {
		return StructuredClonePatchEnvelope{}, parsePayloadErr
	}
	return ParseStructuredClonePatchEnvelopeJSON(parsePayload)
}
