package runtime2

import (
	"encoding/json"
	"fmt"
	"strings"
)

// StructuredCloneMountEnvelope stores the mount payload encoded over structured-clone transport.
type StructuredCloneMountEnvelope struct {
	RegionInstanceID RegionInstanceID `json:"region_instance_id"`
	RendererID       RendererID       `json:"renderer_id"`
	SourceIDs        []string         `json:"source_ids,omitempty"`
	Snapshot         SnapshotEnvelope `json:"snapshot"`
}

// StructuredCloneUpdateEnvelope stores the update payload encoded over structured-clone transport.
type StructuredCloneUpdateEnvelope struct {
	RegionInstanceID RegionInstanceID `json:"region_instance_id"`
	InputVersion     uint64           `json:"input_version"`
	Snapshot         SnapshotEnvelope `json:"snapshot"`
}

// BuildStructuredCloneSnapshotEnvelopeJSON encodes one validated snapshot envelope for structured-clone transport.
func BuildStructuredCloneSnapshotEnvelopeJSON(parseEnvelope SnapshotEnvelope) ([]byte, error) {
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return nil, parseErr
	}
	parsePayload, parseErr := json.Marshal(parseEnvelope)
	if parseErr != nil {
		return nil, fmt.Errorf("runtime2: encode structured-clone snapshot envelope: %w", parseErr)
	}
	return parsePayload, nil
}

// ParseStructuredCloneSnapshotEnvelopeJSON decodes and validates one structured-clone snapshot envelope payload.
func ParseStructuredCloneSnapshotEnvelopeJSON(parsePayload []byte) (SnapshotEnvelope, error) {
	var parseEnvelope SnapshotEnvelope
	if parseErr := json.Unmarshal(parsePayload, &parseEnvelope); parseErr != nil {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode structured-clone snapshot envelope: %w", parseErr)
	}
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}

// BuildStructuredCloneMountEnvelopeJSON encodes one validated mount envelope for structured-clone transport.
func BuildStructuredCloneMountEnvelopeJSON(parseEnvelope StructuredCloneMountEnvelope) ([]byte, error) {
	if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
		return nil, parseErr
	}
	if _, parseErr := ParseRendererID(string(parseEnvelope.RendererID)); parseErr != nil {
		return nil, parseErr
	}
	parseSourceIDs, parseErr := NormalizeSourceIDs(parseEnvelope.SourceIDs)
	if parseErr != nil {
		return nil, parseErr
	}
	parseEnvelope.SourceIDs = parseSourceIDs
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope.Snapshot); parseErr != nil {
		return nil, parseErr
	}
	if parseEnvelope.Snapshot.RegionInstanceID != parseEnvelope.RegionInstanceID {
		return nil, fmt.Errorf("runtime2: mount snapshot region instance ID mismatch")
	}
	parsePayload, parseErr := json.Marshal(parseEnvelope)
	if parseErr != nil {
		return nil, fmt.Errorf("runtime2: encode structured-clone mount envelope: %w", parseErr)
	}
	return parsePayload, nil
}

// ParseStructuredCloneMountEnvelopeJSON decodes and validates one structured-clone mount envelope payload.
func ParseStructuredCloneMountEnvelopeJSON(parsePayload []byte) (StructuredCloneMountEnvelope, error) {
	parseEnvelopeRaw, parseErr := parseStructuredCloneObject(parsePayload)
	if parseErr != nil {
		return StructuredCloneMountEnvelope{}, parseErr
	}
	if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "region_instance_id", structuredCloneFieldString, false); parseErr != nil {
		return StructuredCloneMountEnvelope{}, parseErr
	}
	if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "renderer_id", structuredCloneFieldString, false); parseErr != nil {
		return StructuredCloneMountEnvelope{}, parseErr
	}
	if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "source_ids", structuredCloneFieldList, true); parseErr != nil {
		return StructuredCloneMountEnvelope{}, parseErr
	}
	if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "snapshot", structuredCloneFieldObject, false); parseErr != nil {
		return StructuredCloneMountEnvelope{}, parseErr
	}
	var parseEnvelope StructuredCloneMountEnvelope
	if parseErr := json.Unmarshal(parsePayload, &parseEnvelope); parseErr != nil {
		return StructuredCloneMountEnvelope{}, fmt.Errorf("runtime2: decode structured-clone mount envelope: %w", parseErr)
	}
	if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
		return StructuredCloneMountEnvelope{}, parseErr
	}
	if _, parseErr := ParseRendererID(string(parseEnvelope.RendererID)); parseErr != nil {
		return StructuredCloneMountEnvelope{}, parseErr
	}
	parseSourceIDs, parseErr := NormalizeSourceIDs(parseEnvelope.SourceIDs)
	if parseErr != nil {
		return StructuredCloneMountEnvelope{}, parseErr
	}
	parseEnvelope.SourceIDs = parseSourceIDs
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope.Snapshot); parseErr != nil {
		return StructuredCloneMountEnvelope{}, parseErr
	}
	if parseEnvelope.Snapshot.RegionInstanceID != parseEnvelope.RegionInstanceID {
		return StructuredCloneMountEnvelope{}, fmt.Errorf("runtime2: mount snapshot region instance ID mismatch")
	}
	return parseEnvelope, nil
}

// BuildStructuredCloneUpdateEnvelopeJSON encodes one validated update envelope for structured-clone transport.
func BuildStructuredCloneUpdateEnvelopeJSON(parseEnvelope StructuredCloneUpdateEnvelope) ([]byte, error) {
	if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
		return nil, parseErr
	}
	if parseEnvelope.InputVersion == 0 {
		return nil, fmt.Errorf("runtime2: update input version is required")
	}
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope.Snapshot); parseErr != nil {
		return nil, parseErr
	}
	if parseEnvelope.Snapshot.RegionInstanceID != parseEnvelope.RegionInstanceID {
		return nil, fmt.Errorf("runtime2: update snapshot region instance ID mismatch")
	}
	if parseEnvelope.Snapshot.InputVersion != parseEnvelope.InputVersion {
		return nil, fmt.Errorf("runtime2: update snapshot input version mismatch")
	}
	parsePayload, parseErr := json.Marshal(parseEnvelope)
	if parseErr != nil {
		return nil, fmt.Errorf("runtime2: encode structured-clone update envelope: %w", parseErr)
	}
	return parsePayload, nil
}

// ParseStructuredCloneUpdateEnvelopeJSON decodes and validates one structured-clone update envelope payload.
func ParseStructuredCloneUpdateEnvelopeJSON(parsePayload []byte) (StructuredCloneUpdateEnvelope, error) {
	parseEnvelopeRaw, parseErr := parseStructuredCloneObject(parsePayload)
	if parseErr != nil {
		return StructuredCloneUpdateEnvelope{}, parseErr
	}
	if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "region_instance_id", structuredCloneFieldString, false); parseErr != nil {
		return StructuredCloneUpdateEnvelope{}, parseErr
	}
	if parseErr := validateStructuredCloneFieldType(parseEnvelopeRaw, "snapshot", structuredCloneFieldObject, false); parseErr != nil {
		return StructuredCloneUpdateEnvelope{}, parseErr
	}
	var parseEnvelope StructuredCloneUpdateEnvelope
	if parseErr := json.Unmarshal(parsePayload, &parseEnvelope); parseErr != nil {
		return StructuredCloneUpdateEnvelope{}, fmt.Errorf("runtime2: decode structured-clone update envelope: %w", parseErr)
	}
	if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
		return StructuredCloneUpdateEnvelope{}, parseErr
	}
	if parseEnvelope.InputVersion == 0 {
		return StructuredCloneUpdateEnvelope{}, fmt.Errorf("runtime2: update input version is required")
	}
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope.Snapshot); parseErr != nil {
		return StructuredCloneUpdateEnvelope{}, parseErr
	}
	if parseEnvelope.Snapshot.RegionInstanceID != parseEnvelope.RegionInstanceID {
		return StructuredCloneUpdateEnvelope{}, fmt.Errorf("runtime2: update snapshot region instance ID mismatch")
	}
	if parseEnvelope.Snapshot.InputVersion != parseEnvelope.InputVersion {
		return StructuredCloneUpdateEnvelope{}, fmt.Errorf("runtime2: update snapshot input version mismatch")
	}
	return parseEnvelope, nil
}

type structuredCloneFieldType uint8

const (
	structuredCloneFieldString structuredCloneFieldType = iota + 1
	structuredCloneFieldList
	structuredCloneFieldObject
)

func parseStructuredCloneObject(parsePayload []byte) (map[string]json.RawMessage, error) {
	var parseEnvelopeRaw map[string]json.RawMessage
	if parseErr := json.Unmarshal(parsePayload, &parseEnvelopeRaw); parseErr != nil {
		return nil, fmt.Errorf("runtime2: decode structured-clone envelope: %w", parseErr)
	}
	return parseEnvelopeRaw, nil
}

func validateStructuredCloneFieldType(parseEnvelopeRaw map[string]json.RawMessage, parseField string, parseFieldType structuredCloneFieldType, parseOptional bool) error {
	parseRawValue, parseHasValue := parseEnvelopeRaw[parseField]
	if !parseHasValue {
		if parseOptional {
			return nil
		}
		return fmt.Errorf("runtime2: structured-clone field %q is required", parseField)
	}
	parseRawText := strings.TrimSpace(string(parseRawValue))
	switch parseFieldType {
	case structuredCloneFieldString:
		if !strings.HasPrefix(parseRawText, "\"") {
			return fmt.Errorf("runtime2: structured-clone field %q must be a string", parseField)
		}
	case structuredCloneFieldList:
		if !strings.HasPrefix(parseRawText, "[") {
			return fmt.Errorf("runtime2: structured-clone field %q must be a list", parseField)
		}
	case structuredCloneFieldObject:
		if !strings.HasPrefix(parseRawText, "{") {
			return fmt.Errorf("runtime2: structured-clone field %q must be an object", parseField)
		}
	default:
		return fmt.Errorf("runtime2: structured-clone field %q has unsupported type guard", parseField)
	}
	return nil
}
