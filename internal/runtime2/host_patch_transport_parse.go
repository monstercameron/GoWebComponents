package runtime2

import (
	"encoding/json"
	"fmt"
)

// ParseHostPatchPayloadWithFallback decodes one patch payload based on one patch-ready control envelope with fallback across shared-buffer, binary, and structured-clone tiers.
func ParseHostPatchPayloadWithFallback(
	parsePatchReadyEnvelope ControlEnvelope,
	parseMessagePayload []byte,
	parseSharedPatchPage *SharedPatchPage,
) (TransportTier, PatchStreamRaw, error) {
	if parsePatchReadyEnvelope.Kind != ControlKindPatchReady {
		return "", PatchStreamRaw{}, fmt.Errorf("runtime2: patch-ready control envelope is required")
	}
	if parseEnvelopeErr := ValidateControlEnvelope(parsePatchReadyEnvelope); parseEnvelopeErr != nil {
		return "", PatchStreamRaw{}, parseEnvelopeErr
	}
	parseExpectedRegionID := string(parsePatchReadyEnvelope.RegionInstanceID)
	parseExpectedPatchVersion := parsePatchReadyEnvelope.PatchVersion
	parseValidatePatchStream := func(parseTier TransportTier, parsePatchStream PatchStreamRaw) (TransportTier, PatchStreamRaw, error) {
		parseHeader, parseHeaderErr := ParsePatchStreamHeader(parsePatchStream.GetHeader, parsePatchStream.GetHeader.RegionID)
		if parseHeaderErr != nil {
			return "", PatchStreamRaw{}, parseHeaderErr
		}
		if parseHeader.RegionID != parseExpectedRegionID {
			return "", PatchStreamRaw{}, fmt.Errorf("runtime2: patch payload region mismatch expected=%q actual=%q", parseExpectedRegionID, parseHeader.RegionID)
		}
		if parseHeader.PatchVersion != parseExpectedPatchVersion {
			return "", PatchStreamRaw{}, fmt.Errorf("runtime2: patch payload version mismatch expected=%d actual=%d", parseExpectedPatchVersion, parseHeader.PatchVersion)
		}
		return parseTier, parsePatchStream, nil
	}
	switch parsePatchReadyEnvelope.TransportTier {
	case TransportTierSharedBuffer:
		if parseSharedPatchPage != nil {
			parseSharedEnvelope, parseSharedEnvelopeErr := ParseSharedPatchPayloadFromPage(parseSharedPatchPage)
			if parseSharedEnvelopeErr == nil {
				parseStructuredPatchStream, parseStructuredPatchStreamErr := parsePatchStreamFromStructuredClonePayload(parseSharedEnvelope.PatchPayload)
				if parseStructuredPatchStreamErr == nil {
					return parseValidatePatchStream(TransportTierSharedBuffer, parseStructuredPatchStream)
				}
			}
		}
		parseBinaryPatchStream, parseBinaryPatchStreamErr := ParseBinaryPatchPayload(parseMessagePayload)
		if parseBinaryPatchStreamErr == nil {
			return parseValidatePatchStream(TransportTierBinary, parseBinaryPatchStream)
		}
		parseStructuredPatchStream, parseStructuredPatchStreamErr := parsePatchStreamFromStructuredClonePayload(parseMessagePayload)
		if parseStructuredPatchStreamErr != nil {
			return "", PatchStreamRaw{}, fmt.Errorf("runtime2: shared patch decode failed and fallback decode failed: %v; %w", parseBinaryPatchStreamErr, parseStructuredPatchStreamErr)
		}
		return parseValidatePatchStream(TransportTierStructuredClone, parseStructuredPatchStream)
	case TransportTierBinary:
		parseBinaryPatchStream, parseBinaryPatchStreamErr := ParseBinaryPatchPayload(parseMessagePayload)
		if parseBinaryPatchStreamErr == nil {
			return parseValidatePatchStream(TransportTierBinary, parseBinaryPatchStream)
		}
		parseStructuredPatchStream, parseStructuredPatchStreamErr := parsePatchStreamFromStructuredClonePayload(parseMessagePayload)
		if parseStructuredPatchStreamErr != nil {
			return "", PatchStreamRaw{}, fmt.Errorf("runtime2: binary patch decode failed and structured fallback failed: %v; %w", parseBinaryPatchStreamErr, parseStructuredPatchStreamErr)
		}
		return parseValidatePatchStream(TransportTierStructuredClone, parseStructuredPatchStream)
	case TransportTierStructuredClone:
		parseStructuredPatchStream, parseStructuredPatchStreamErr := parsePatchStreamFromStructuredClonePayload(parseMessagePayload)
		if parseStructuredPatchStreamErr != nil {
			return "", PatchStreamRaw{}, parseStructuredPatchStreamErr
		}
		return parseValidatePatchStream(TransportTierStructuredClone, parseStructuredPatchStream)
	default:
		return "", PatchStreamRaw{}, fmt.Errorf("runtime2: patch-ready transport tier %q is unsupported for host patch parse", parsePatchReadyEnvelope.TransportTier)
	}
}

// parsePatchStreamFromStructuredClonePayload decodes one structured-clone patch payload into one patch stream.
func parsePatchStreamFromStructuredClonePayload(parsePayload []byte) (PatchStreamRaw, error) {
	parseStructuredEnvelope, parseStructuredEnvelopeErr := ParseStructuredClonePatchEnvelopeJSON(parsePayload)
	if parseStructuredEnvelopeErr == nil {
		var parsePatchStream PatchStreamRaw
		if parsePatchStreamErr := json.Unmarshal(parseStructuredEnvelope.PatchPayload, &parsePatchStream); parsePatchStreamErr != nil {
			return PatchStreamRaw{}, fmt.Errorf("runtime2: decode structured-clone patch envelope payload: %w", parsePatchStreamErr)
		}
		if _, parseHeaderErr := ParsePatchStreamHeader(parsePatchStream.GetHeader, parsePatchStream.GetHeader.RegionID); parseHeaderErr != nil {
			return PatchStreamRaw{}, parseHeaderErr
		}
		if parsePatchStream.GetHeader.RegionID != string(parseStructuredEnvelope.RegionInstanceID) {
			return PatchStreamRaw{}, fmt.Errorf("runtime2: structured-clone patch envelope region mismatch")
		}
		if parsePatchStream.GetHeader.PatchVersion != parseStructuredEnvelope.PatchVersion {
			return PatchStreamRaw{}, fmt.Errorf("runtime2: structured-clone patch envelope patch version mismatch")
		}
		return parsePatchStream, nil
	}
	var parsePatchStream PatchStreamRaw
	if parsePatchStreamErr := json.Unmarshal(parsePayload, &parsePatchStream); parsePatchStreamErr != nil {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: decode structured-clone patch payload: %w", parsePatchStreamErr)
	}
	if _, parseHeaderErr := ParsePatchStreamHeader(parsePatchStream.GetHeader, parsePatchStream.GetHeader.RegionID); parseHeaderErr != nil {
		return PatchStreamRaw{}, parseHeaderErr
	}
	return parsePatchStream, nil
}
