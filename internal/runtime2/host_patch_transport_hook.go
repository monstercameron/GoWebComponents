package runtime2

import (
	"encoding/json"
	"fmt"
)

// HostPatchConsumeResult reports one decoded patch payload and host-side commit outcome.
type HostPatchConsumeResult struct {
	GetTransportTier TransportTier
	GetPatchStream   PatchStreamRaw
	GetCommitResult  HostRegionWorkerOutputResult
}

// ParseStructuredClonePatchPayloadJSON decodes one structured-clone patch payload into a validated patch stream.
func ParseStructuredClonePatchPayloadJSON(parsePayload []byte) (PatchStreamRaw, error) {
	if len(parsePayload) == 0 {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: structured-clone patch payload is required")
	}
	var parsePatchStream PatchStreamRaw
	if parseDecodeErr := json.Unmarshal(parsePayload, &parsePatchStream); parseDecodeErr != nil {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: decode structured-clone patch payload: %w", parseDecodeErr)
	}
	if _, parseHeaderErr := ParsePatchStreamHeader(parsePatchStream.GetHeader, parsePatchStream.GetHeader.RegionID); parseHeaderErr != nil {
		return PatchStreamRaw{}, parseHeaderErr
	}
	return parsePatchStream, nil
}

// HandleHostRegionPatchConsume decodes one transport-tier patch payload and commits it through host patch orchestration.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionPatchConsume(
	parseTransportTier TransportTier,
	parsePayload []byte,
	parseSharedPatchPage *SharedPatchPage,
	parseDOMCommitter *DOMCommitter,
) (HostPatchConsumeResult, error) {
	if parseHostRegionAdapter == nil {
		return HostPatchConsumeResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	parseTier, parseTierErr := ParseTransportTier(string(parseTransportTier))
	if parseTierErr != nil {
		return HostPatchConsumeResult{}, parseTierErr
	}
	var parsePatchStream PatchStreamRaw
	switch parseTier {
	case TransportTierStructuredClone:
		parsePatchStream, parseTierErr = ParseStructuredClonePatchPayloadJSON(parsePayload)
	case TransportTierBinary:
		parsePatchStream, parseTierErr = ParseBinaryPatchPayload(parsePayload)
	case TransportTierSharedBuffer:
		parseSharedEnvelope, parseSharedErr := ParseSharedPatchPayloadFromPage(parseSharedPatchPage)
		if parseSharedErr != nil {
			return HostPatchConsumeResult{}, parseSharedErr
		}
		parsePatchStream, parseTierErr = ParseStructuredClonePatchPayloadJSON(parseSharedEnvelope.PatchPayload)
	default:
		return HostPatchConsumeResult{}, fmt.Errorf("runtime2: transport tier %q is unsupported for patch consume", parseTier)
	}
	if parseTierErr != nil {
		return HostPatchConsumeResult{}, parseTierErr
	}
	return parseHostRegionAdapter.handleHostRegionPatchConsumeKnownPatchStream(parseTier, parsePatchStream, parseDOMCommitter)
}

// handleHostRegionPatchConsumeKnownPatchStream commits one already-decoded patch stream through host patch orchestration.
func (parseHostRegionAdapter *HostRegionAdapter) handleHostRegionPatchConsumeKnownPatchStream(
	parseTransportTier TransportTier,
	parsePatchStream PatchStreamRaw,
	parseDOMCommitter *DOMCommitter,
) (HostPatchConsumeResult, error) {
	if parseHostRegionAdapter == nil {
		return HostPatchConsumeResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	parseTier, parseTierErr := ParseTransportTier(string(parseTransportTier))
	if parseTierErr != nil {
		return HostPatchConsumeResult{}, parseTierErr
	}
	parseCommitResult, parseCommitErr := parseHostRegionAdapter.HandleHostRegionPatchCommit(parsePatchStream, parseDOMCommitter)
	if parseCommitErr != nil {
		return HostPatchConsumeResult{}, parseCommitErr
	}
	return HostPatchConsumeResult{
		GetTransportTier: parseTier,
		GetPatchStream:   parsePatchStream,
		GetCommitResult:  parseCommitResult,
	}, nil
}
