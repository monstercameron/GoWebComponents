package runtime2

import (
	"encoding/json"
	"fmt"

	"github.com/monstercameron/GoWebComponents/v4/diagnostics"
)

// reportPatchIdentityMismatch surfaces a decoded-patch identity mismatch detected
// at the host/worker trust boundary. A test seam; the default emits a fail-visible
// diagnostic. Advisory only — see verifyDecodedPatchIdentity.
var reportPatchIdentityMismatch = func(parseRegionID string, parsePatchVersion uint64, parseErr error) {
	diagnostics.Emit(diagnostics.NewReport(diagnostics.Options{
		Code:     "GWC-PATCH-IDENTITY-MISMATCH",
		Headline: "decoded patch failed its identity check",
		Summary:  fmt.Sprintf("region %q patch version %d: %v — the patch crossed the worker transport boundary and its carried identity does not match a recompute from its parts, indicating a corrupted or tampered body", parseRegionID, parsePatchVersion, parseErr),
		Next:     "investigate the worker->host patch transport for corruption or tampering (the patch was still applied: this is an advisory integrity check, not a gate)",
	}))
}

// verifyDecodedPatchIdentity runs the defense-in-depth identity check on a patch
// decoded at the host/worker boundary and REPORTS a mismatch WITHOUT rejecting the
// patch. Rejecting is deliberately avoided here: buildPatchStreamIdentityFromParts
// distinguishes nil from empty-but-non-nil ops, and a stream may legitimately carry
// a hand-set (non-canonical) identity, so a hard reject could false-reject a valid
// patch and break rendering. Advisory detection removes the silent-application gap
// with zero false-reject risk; promoting it to a hard reject is gated on the
// worker-bridge harness (#83) proving every patch that reaches this boundary
// round-trips its identity exactly.
func verifyDecodedPatchIdentity(parsePatchStream PatchStreamRaw) {
	if parseErr := VerifyPatchStreamIdentity(parsePatchStream); parseErr != nil {
		reportPatchIdentityMismatch(parsePatchStream.GetHeader.RegionID, parsePatchStream.GetHeader.PatchVersion, parseErr)
	}
}

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
		if parsePatchStream.GetHeader.RegionID != parseExpectedRegionID {
			return "", PatchStreamRaw{}, fmt.Errorf("runtime2: patch payload region mismatch expected=%q actual=%q", parseExpectedRegionID, parsePatchStream.GetHeader.RegionID)
		}
		if parsePatchStream.GetHeader.PatchVersion != parseExpectedPatchVersion {
			return "", PatchStreamRaw{}, fmt.Errorf("runtime2: patch payload version mismatch expected=%d actual=%d", parseExpectedPatchVersion, parsePatchStream.GetHeader.PatchVersion)
		}
		// Defense-in-depth: the patch has crossed the worker serialization boundary,
		// so verify its carried identity against a recompute from its decoded parts.
		// Advisory (surfaces a mismatch, does not reject) — see verifyDecodedPatchIdentity.
		verifyDecodedPatchIdentity(parsePatchStream)
		return parseTier, parsePatchStream, nil
	}
	hasParseMessagePayloadBinaryPrefix := hasParseBinaryPatchPayloadPrefix(parseMessagePayload)
	switch parsePatchReadyEnvelope.TransportTier {
	case TransportTierSharedBuffer:
		if parseSharedPatchPage != nil {
			parseSharedEnvelope, parseSharedEnvelopeErr := ParseSharedPatchPayloadFromPage(parseSharedPatchPage)
			if parseSharedEnvelopeErr == nil {
				parseStructuredPatchStream, parseStructuredPatchStreamErr := ParseStructuredClonePatchPayloadJSON(parseSharedEnvelope.PatchPayload)
				if parseStructuredPatchStreamErr == nil {
					return parseValidatePatchStream(TransportTierSharedBuffer, parseStructuredPatchStream)
				}
			}
		}
		parseBinaryFallbackDetail := "binary decode skipped due non-binary payload prefix"
		if hasParseMessagePayloadBinaryPrefix {
			parseBinaryPatchStream, parseBinaryPatchStreamErr := ParseBinaryPatchPayload(parseMessagePayload)
			if parseBinaryPatchStreamErr == nil {
				return parseValidatePatchStream(TransportTierBinary, parseBinaryPatchStream)
			}
			parseBinaryFallbackDetail = parseBinaryPatchStreamErr.Error()
		}
		parseStructuredPatchStream, parseStructuredPatchStreamErr := parsePatchStreamFromStructuredClonePayload(parseMessagePayload)
		if parseStructuredPatchStreamErr != nil {
			return "", PatchStreamRaw{}, fmt.Errorf("runtime2: shared patch decode failed and fallback decode failed: %s; %w", parseBinaryFallbackDetail, parseStructuredPatchStreamErr)
		}
		return parseValidatePatchStream(TransportTierStructuredClone, parseStructuredPatchStream)
	case TransportTierBinary:
		parseBinaryFallbackDetail := "binary decode skipped due non-binary payload prefix"
		if hasParseMessagePayloadBinaryPrefix {
			parseBinaryPatchStream, parseBinaryPatchStreamErr := ParseBinaryPatchPayload(parseMessagePayload)
			if parseBinaryPatchStreamErr == nil {
				return parseValidatePatchStream(TransportTierBinary, parseBinaryPatchStream)
			}
			parseBinaryFallbackDetail = parseBinaryPatchStreamErr.Error()
		}
		parseStructuredPatchStream, parseStructuredPatchStreamErr := parsePatchStreamFromStructuredClonePayload(parseMessagePayload)
		if parseStructuredPatchStreamErr != nil {
			return "", PatchStreamRaw{}, fmt.Errorf("runtime2: binary patch decode failed and structured fallback failed: %s; %w", parseBinaryFallbackDetail, parseStructuredPatchStreamErr)
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

// hasParseBinaryPatchPayloadPrefix reports whether one payload begins with runtime2 binary patch transport framing bytes.
func hasParseBinaryPatchPayloadPrefix(parsePayload []byte) bool {
	if len(parsePayload) < binaryPatchTransportHeaderLength {
		return false
	}
	return parsePayload[0] == binaryPatchTransportMagic[0] &&
		parsePayload[1] == binaryPatchTransportMagic[1] &&
		parsePayload[2] == binaryPatchTransportMagic[2] &&
		parsePayload[3] == binaryPatchTransportMagic[3]
}

// parsePatchStreamFromStructuredClonePayload decodes one structured-clone patch payload into one patch stream.
func parsePatchStreamFromStructuredClonePayload(parsePayload []byte) (PatchStreamRaw, error) {
	if hasParseStructuredClonePatchEnvelopePrefix(parsePayload) {
		return parsePatchStreamFromStructuredCloneEnvelopePayload(parsePayload)
	}
	parsePatchStream, parsePatchStreamErr := parsePatchStreamFromStructuredCloneRawPayload(parsePayload)
	if parsePatchStreamErr == nil {
		return parsePatchStream, nil
	}
	parsePatchStreamFallback, parsePatchStreamFallbackErr := parsePatchStreamFromStructuredCloneEnvelopePayload(parsePayload)
	if parsePatchStreamFallbackErr == nil {
		return parsePatchStreamFallback, nil
	}
	return PatchStreamRaw{}, parsePatchStreamErr
}

// parsePatchStreamFromStructuredCloneEnvelopePayload decodes one structured-clone patch envelope JSON payload into one validated patch stream.
func parsePatchStreamFromStructuredCloneEnvelopePayload(parsePayload []byte) (PatchStreamRaw, error) {
	parseStructuredEnvelope, parseStructuredEnvelopeErr := ParseStructuredClonePatchEnvelopeJSON(parsePayload)
	if parseStructuredEnvelopeErr != nil {
		return PatchStreamRaw{}, parseStructuredEnvelopeErr
	}
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

// parsePatchStreamFromStructuredCloneRawPayload decodes one raw patch stream JSON payload into one validated patch stream.
func parsePatchStreamFromStructuredCloneRawPayload(parsePayload []byte) (PatchStreamRaw, error) {
	var parsePatchStream PatchStreamRaw
	if parsePatchStreamErr := json.Unmarshal(parsePayload, &parsePatchStream); parsePatchStreamErr != nil {
		return PatchStreamRaw{}, fmt.Errorf("runtime2: decode structured-clone patch payload: %w", parsePatchStreamErr)
	}
	if _, parseHeaderErr := ParsePatchStreamHeader(parsePatchStream.GetHeader, parsePatchStream.GetHeader.RegionID); parseHeaderErr != nil {
		return PatchStreamRaw{}, parseHeaderErr
	}
	return parsePatchStream, nil
}

// hasParseStructuredClonePatchEnvelopePrefix reports whether one payload begins with canonical structured-clone patch envelope JSON tokens.
func hasParseStructuredClonePatchEnvelopePrefix(parsePayload []byte) bool {
	parseStart := 0
	for parseStart < len(parsePayload) {
		parseByte := parsePayload[parseStart]
		if parseByte != ' ' && parseByte != '\n' && parseByte != '\r' && parseByte != '\t' {
			break
		}
		parseStart++
	}
	if len(parsePayload)-parseStart < len(buildStructuredClonePatchEnvelopeTokenRegionID) {
		return false
	}
	for parseIndex := range len(buildStructuredClonePatchEnvelopeTokenRegionID) {
		if parsePayload[parseStart+parseIndex] != buildStructuredClonePatchEnvelopeTokenRegionID[parseIndex] {
			return false
		}
	}
	return true
}
