package runtime2

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
)

// StructuredClonePatchEnvelope stores one patch payload encoded over structured-clone transport.
type StructuredClonePatchEnvelope struct {
	RegionInstanceID RegionInstanceID `json:"region_instance_id"`
	Epoch            uint64           `json:"epoch"`
	PatchVersion     uint64           `json:"patch_version"`
	PatchPayload     []byte           `json:"patch_payload"`
}

const (
	buildStructuredClonePatchEnvelopeTokenRegionID    = `{"region_instance_id":`
	buildStructuredClonePatchEnvelopeTokenEpoch       = `,"epoch":`
	buildStructuredClonePatchEnvelopeTokenPatchVersion = `,"patch_version":`
	buildStructuredClonePatchEnvelopeTokenPayload     = `,"patch_payload":"`
)

// ValidateStructuredClonePatchEnvelope verifies one structured-clone patch envelope contract.
func ValidateStructuredClonePatchEnvelope(parseEnvelope StructuredClonePatchEnvelope) error {
	if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
		return parseErr
	}
	if parseEnvelope.Epoch == 0 {
		return fmt.Errorf("runtime2: patch envelope epoch is required")
	}
	if parseEnvelope.PatchVersion == 0 {
		return fmt.Errorf("runtime2: patch envelope patch version is required")
	}
	if len(parseEnvelope.PatchPayload) == 0 {
		return fmt.Errorf("runtime2: patch envelope payload is required")
	}
	return nil
}

// BuildStructuredClonePatchEnvelopeJSON encodes one validated structured-clone patch envelope.
func BuildStructuredClonePatchEnvelopeJSON(parseEnvelope StructuredClonePatchEnvelope) ([]byte, error) {
	if parseErr := ValidateStructuredClonePatchEnvelope(parseEnvelope); parseErr != nil {
		return nil, parseErr
	}
	parsePayload := make([]byte, 0, getStructuredClonePatchEnvelopeJSONLength(parseEnvelope))
	parsePayload = append(parsePayload, buildStructuredClonePatchEnvelopeTokenRegionID...)
	parsePayload = strconv.AppendQuote(parsePayload, string(parseEnvelope.RegionInstanceID))
	parsePayload = append(parsePayload, buildStructuredClonePatchEnvelopeTokenEpoch...)
	parsePayload = strconv.AppendUint(parsePayload, parseEnvelope.Epoch, 10)
	parsePayload = append(parsePayload, buildStructuredClonePatchEnvelopeTokenPatchVersion...)
	parsePayload = strconv.AppendUint(parsePayload, parseEnvelope.PatchVersion, 10)
	parsePayload = append(parsePayload, buildStructuredClonePatchEnvelopeTokenPayload...)
	parseEncodedPayloadLength := base64.StdEncoding.EncodedLen(len(parseEnvelope.PatchPayload))
	parseEncodedPayloadStart := len(parsePayload)
	parsePayload = append(parsePayload, make([]byte, parseEncodedPayloadLength)...)
	base64.StdEncoding.Encode(parsePayload[parseEncodedPayloadStart:parseEncodedPayloadStart+parseEncodedPayloadLength], parseEnvelope.PatchPayload)
	parsePayload = append(parsePayload, '"', '}')
	return parsePayload, nil
}

// ParseStructuredClonePatchEnvelopeJSON decodes and validates one structured-clone patch envelope payload.
func ParseStructuredClonePatchEnvelopeJSON(parsePayload []byte) (StructuredClonePatchEnvelope, error) {
	var parseEnvelope StructuredClonePatchEnvelope
	if parseErr := json.Unmarshal(parsePayload, &parseEnvelope); parseErr != nil {
		return StructuredClonePatchEnvelope{}, fmt.Errorf("runtime2: decode structured-clone patch envelope: %w", parseErr)
	}
	if parseErr := ValidateStructuredClonePatchEnvelope(parseEnvelope); parseErr != nil {
		return StructuredClonePatchEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}

// getStructuredClonePatchEnvelopeJSONLength returns an upper-bound capacity for one structured-clone patch-envelope JSON payload.
func getStructuredClonePatchEnvelopeJSONLength(parseEnvelope StructuredClonePatchEnvelope) int {
	return len(buildStructuredClonePatchEnvelopeTokenRegionID) +
		2 + len(string(parseEnvelope.RegionInstanceID)) +
		len(buildStructuredClonePatchEnvelopeTokenEpoch) + 20 +
		len(buildStructuredClonePatchEnvelopeTokenPatchVersion) + 20 +
		len(buildStructuredClonePatchEnvelopeTokenPayload) +
		base64.StdEncoding.EncodedLen(len(parseEnvelope.PatchPayload)) +
		2
}
