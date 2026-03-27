package runtime2

import (
	"encoding/json"
	"fmt"
)

// WorkerPatchTransportResult reports one worker update result plus selected patch transport metadata.
type WorkerPatchTransportResult struct {
	GetUpdateResult       WorkerRegionUpdateResult
	HasPatchPayload       bool
	GetPatchPayload       []byte
	GetTransportTier      TransportTier
	HasPatchReadyEnvelope bool
	GetPatchReadyEnvelope ControlEnvelope
}

// SelectPatchTransportTier selects the preferred patch transport tier from one capability report.
func SelectPatchTransportTier(parseCapabilityReport CapabilityReport, parseHasSharedPatchPage bool) (TransportTier, error) {
	if parseErr := ValidateCapabilityReport(parseCapabilityReport); parseErr != nil {
		return "", parseErr
	}
	if parseCapabilityReport.HasSharedMemoryTransportSupport && parseHasSharedPatchPage {
		return TransportTierSharedBuffer, nil
	}
	if parseCapabilityReport.HasBinaryTransportSupport {
		return TransportTierBinary, nil
	}
	if parseCapabilityReport.HasStructuredCloneSupport {
		return TransportTierStructuredClone, nil
	}
	return "", fmt.Errorf("runtime2: no supported patch transport tier is available")
}

// BuildStructuredClonePatchPayloadJSON encodes one patch payload for structured-clone transport.
func BuildStructuredClonePatchPayloadJSON(parsePatchPayload any) ([]byte, error) {
	if parsePatchPayload == nil {
		return nil, fmt.Errorf("runtime2: patch payload is required")
	}
	parsePayload, parseErr := json.Marshal(parsePatchPayload)
	if parseErr != nil {
		return nil, fmt.Errorf("runtime2: encode structured-clone patch payload: %w", parseErr)
	}
	return parsePayload, nil
}

// HandleWorkerRegionUpdateWithPatchTransport runs one worker update and selects patch transport for patch-ready results.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) HandleWorkerRegionUpdateWithPatchTransport(
	parseUpdate WorkerRegionUpdateSpec,
	parseCapabilityReport CapabilityReport,
) (WorkerPatchTransportResult, error) {
	return parseWorkerRegionRuntime.handleWorkerRegionUpdateWithPatchTransportEnvelopeOption(
		parseUpdate,
		parseCapabilityReport,
		true,
	)
}

// handleWorkerRegionUpdateWithPatchTransportEnvelopeOption runs one worker update, selects patch transport, and optionally builds a patch-ready control envelope.
func (parseWorkerRegionRuntime *WorkerRegionRuntime) handleWorkerRegionUpdateWithPatchTransportEnvelopeOption(
	parseUpdate WorkerRegionUpdateSpec,
	parseCapabilityReport CapabilityReport,
	parseHasPatchReadyEnvelope bool,
) (WorkerPatchTransportResult, error) {
	if parseWorkerRegionRuntime == nil {
		return WorkerPatchTransportResult{}, fmt.Errorf("runtime2: worker region runtime is nil")
	}
	parseUpdateResult, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(parseUpdate)
	if parseUpdateErr != nil {
		return WorkerPatchTransportResult{}, parseUpdateErr
	}
	parseResult := WorkerPatchTransportResult{
		GetUpdateResult: parseUpdateResult,
	}
	if !parseUpdateResult.HasPatchReady {
		return parseResult, nil
	}
	parseTransportTier, parseTierErr := SelectPatchTransportTier(parseCapabilityReport, false)
	if parseTierErr != nil {
		return WorkerPatchTransportResult{}, parseTierErr
	}
	var parsePatchPayload []byte
	if parseTransportTier == TransportTierBinary {
		parseBinaryPayload, parseBinaryPayloadErr := BuildBinaryPatchPayload(parseUpdateResult.PatchIR)
		if parseBinaryPayloadErr != nil {
			return WorkerPatchTransportResult{}, parseBinaryPayloadErr
		}
		parsePatchPayload = parseBinaryPayload
	}
	if parseTransportTier == TransportTierStructuredClone {
		parseStructuredClonePayload, parseStructuredClonePayloadErr := BuildStructuredClonePatchPayloadJSON(parseUpdateResult.PatchIR)
		if parseStructuredClonePayloadErr != nil {
			return WorkerPatchTransportResult{}, parseStructuredClonePayloadErr
		}
		parsePatchPayload = parseStructuredClonePayload
	}
	parseResult.HasPatchPayload = true
	parseResult.GetPatchPayload = parsePatchPayload
	parseResult.GetTransportTier = parseTransportTier
	if parseHasPatchReadyEnvelope {
		parsePatchReadyEnvelope, parsePatchReadyErr := BuildControlPatchReadyEnvelope(
			RegionInstanceID(parseUpdate.RegionID),
			parseUpdateResult.PatchIR.GetHeader.PatchVersion,
			parseUpdate.InputVersion,
			parseTransportTier,
		)
		if parsePatchReadyErr != nil {
			return WorkerPatchTransportResult{}, parsePatchReadyErr
		}
		parseResult.HasPatchReadyEnvelope = true
		parseResult.GetPatchReadyEnvelope = parsePatchReadyEnvelope
	}
	return parseResult, nil
}
