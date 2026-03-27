package runtime2

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// SnapshotEnvelope stores one versioned region snapshot for worker dispatch.
type SnapshotEnvelope struct {
	RegionInstanceID RegionInstanceID `json:"region_instance_id"`
	Epoch            uint64           `json:"epoch"`
	InputVersion     uint64           `json:"input_version"`
	SourceVersion    uint64           `json:"source_version,omitempty"`
	Props            any              `json:"props,omitempty"`
	Sources          map[string]any   `json:"sources,omitempty"`
}

// ValidateMonotonicInputVersion verifies input versions do not move backward.
func ValidateMonotonicInputVersion(parsePrevious uint64, parseCurrent uint64) error {
	if parseCurrent < parsePrevious {
		return fmt.Errorf("runtime2: input version moved backward previous=%d current=%d", parsePrevious, parseCurrent)
	}
	return nil
}

// BuildSourceSnapshot selects declared source values from the current source map.
func BuildSourceSnapshot(parseSourceIDs []string, parseSourceValues map[string]any) (map[string]any, error) {
	parseNormalizedSourceIDs, parseErr := NormalizeSourceIDs(parseSourceIDs)
	if parseErr != nil {
		return nil, parseErr
	}
	if len(parseNormalizedSourceIDs) == 0 {
		return nil, nil
	}
	parseSnapshot := make(map[string]any, len(parseNormalizedSourceIDs))
	for _, parseSourceID := range parseNormalizedSourceIDs {
		parseValue, parseHasValue := parseSourceValues[parseSourceID]
		if !parseHasValue {
			return nil, fmt.Errorf("runtime2: missing declared source value for %q", parseSourceID)
		}
		if parseErr := ValidateSerializableProps(parseValue); parseErr != nil {
			return nil, fmt.Errorf("runtime2: source %q is not serializable: %w", parseSourceID, parseErr)
		}
		parseSnapshot[parseSourceID] = parseValue
	}
	return parseSnapshot, nil
}

// ValidateSourceSnapshotConsistency verifies all declared source versions describe one coherent snapshot.
func ValidateSourceSnapshotConsistency(parseSourceIDs []string, parseSourceVersions map[string]uint64) (uint64, error) {
	parseNormalizedSourceIDs, parseErr := NormalizeSourceIDs(parseSourceIDs)
	if parseErr != nil {
		return 0, parseErr
	}
	if len(parseNormalizedSourceIDs) == 0 {
		return 0, nil
	}
	var parseCommonVersion uint64
	for parseIndex, parseSourceID := range parseNormalizedSourceIDs {
		parseSourceVersion, parseHasVersion := parseSourceVersions[parseSourceID]
		if !parseHasVersion {
			return 0, fmt.Errorf("runtime2: missing declared source version for %q", parseSourceID)
		}
		if parseIndex == 0 {
			parseCommonVersion = parseSourceVersion
			continue
		}
		if parseSourceVersion != parseCommonVersion {
			return 0, fmt.Errorf("runtime2: source version mismatch for %q expected=%d actual=%d", parseSourceID, parseCommonVersion, parseSourceVersion)
		}
	}
	return parseCommonVersion, nil
}

// ValidateSnapshotEnvelope verifies a snapshot envelope is complete enough to dispatch.
func ValidateSnapshotEnvelope(parseEnvelope SnapshotEnvelope) error {
	if _, parseErr := ParseRegionInstanceID(string(parseEnvelope.RegionInstanceID)); parseErr != nil {
		return parseErr
	}
	if parseEnvelope.Epoch == 0 {
		return fmt.Errorf("runtime2: snapshot epoch is required")
	}
	if parseEnvelope.InputVersion == 0 {
		return fmt.Errorf("runtime2: snapshot input version is required")
	}
	if parseErr := ValidateSerializableProps(parseEnvelope.Props); parseErr != nil {
		return parseErr
	}
	return nil
}

// BuildSnapshotEnvelope validates and builds a coherent snapshot envelope.
func BuildSnapshotEnvelope(parseRegionInstanceID RegionInstanceID, parseEpoch uint64, parseInputVersion uint64, parseProps any, parseSourceIDs []string, parseSourceValues map[string]any, parseSourceVersions map[string]uint64) (SnapshotEnvelope, error) {
	parseSources, parseErr := BuildSourceSnapshot(parseSourceIDs, parseSourceValues)
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseSourceVersion, parseErr := ValidateSourceSnapshotConsistency(parseSourceIDs, parseSourceVersions)
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseEnvelope := SnapshotEnvelope{
		RegionInstanceID: parseRegionInstanceID,
		Epoch:            parseEpoch,
		InputVersion:     parseInputVersion,
		SourceVersion:    parseSourceVersion,
		Props:            parseProps,
		Sources:          parseSources,
	}
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}

// GetSnapshotFingerprint returns a stable fingerprint for a snapshot envelope.
func GetSnapshotFingerprint(parseEnvelope SnapshotEnvelope) (string, error) {
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return "", parseErr
	}
	parsePayload, parseErr := json.Marshal(parseEnvelope)
	if parseErr != nil {
		return "", fmt.Errorf("runtime2: encode snapshot fingerprint payload: %w", parseErr)
	}
	parseHash := sha256.Sum256(parsePayload)
	return hex.EncodeToString(parseHash[:]), nil
}
