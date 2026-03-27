package runtime2

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"sync"
)

var storeSnapshotFingerprintHasherPool = sync.Pool{
	New: func() any {
		return sha256.New()
	},
}

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
	return buildSnapshotSourceValues(parseNormalizedSourceIDs, parseSourceValues)
}

// buildSnapshotSourceValues selects declared source values from one normalized source ID list.
func buildSnapshotSourceValues(parseNormalizedSourceIDs []string, parseSourceValues map[string]any) (map[string]any, error) {
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
	return validateSnapshotSourceVersionConsistency(parseNormalizedSourceIDs, parseSourceVersions)
}

// validateSnapshotSourceVersionConsistency verifies all declared normalized source versions describe one coherent snapshot.
func validateSnapshotSourceVersionConsistency(parseNormalizedSourceIDs []string, parseSourceVersions map[string]uint64) (uint64, error) {
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
	parseNormalizedSourceIDs, parseNormalizeErr := NormalizeSourceIDs(parseSourceIDs)
	if parseNormalizeErr != nil {
		return SnapshotEnvelope{}, parseNormalizeErr
	}
	return buildSnapshotEnvelopeFromNormalizedSourceIDs(
		parseRegionInstanceID,
		parseEpoch,
		parseInputVersion,
		parseProps,
		parseNormalizedSourceIDs,
		parseSourceValues,
		parseSourceVersions,
	)
}

// buildSnapshotEnvelopeFromNormalizedSourceIDs validates and builds one coherent snapshot envelope from normalized source IDs.
func buildSnapshotEnvelopeFromNormalizedSourceIDs(
	parseRegionInstanceID RegionInstanceID,
	parseEpoch uint64,
	parseInputVersion uint64,
	parseProps any,
	parseNormalizedSourceIDs []string,
	parseSourceValues map[string]any,
	parseSourceVersions map[string]uint64,
) (SnapshotEnvelope, error) {
	parseEnvelope, parseEnvelopeErr := buildSnapshotEnvelopeFromNormalizedSourceIDsWithoutValidation(
		parseRegionInstanceID,
		parseEpoch,
		parseInputVersion,
		parseProps,
		parseNormalizedSourceIDs,
		parseSourceValues,
		parseSourceVersions,
	)
	if parseEnvelopeErr != nil {
		return SnapshotEnvelope{}, parseEnvelopeErr
	}
	if parseValidateErr := ValidateSnapshotEnvelope(parseEnvelope); parseValidateErr != nil {
		return SnapshotEnvelope{}, parseValidateErr
	}
	return parseEnvelope, nil
}

// buildSnapshotEnvelopeFromNormalizedSourceIDsWithoutValidation builds one coherent snapshot envelope from normalized source IDs without envelope-level validation.
func buildSnapshotEnvelopeFromNormalizedSourceIDsWithoutValidation(
	parseRegionInstanceID RegionInstanceID,
	parseEpoch uint64,
	parseInputVersion uint64,
	parseProps any,
	parseNormalizedSourceIDs []string,
	parseSourceValues map[string]any,
	parseSourceVersions map[string]uint64,
) (SnapshotEnvelope, error) {
	parseSources, parseSourcesErr := buildSnapshotSourceValues(parseNormalizedSourceIDs, parseSourceValues)
	if parseSourcesErr != nil {
		return SnapshotEnvelope{}, parseSourcesErr
	}
	return buildSnapshotEnvelopeFromNormalizedSourceSnapshotWithoutValidation(
		parseRegionInstanceID,
		parseEpoch,
		parseInputVersion,
		parseProps,
		parseNormalizedSourceIDs,
		parseSources,
		parseSourceVersions,
	)
}

// buildSnapshotEnvelopeFromNormalizedSourceSnapshotWithoutValidation builds one coherent snapshot envelope from normalized source IDs plus a prevalidated source snapshot.
func buildSnapshotEnvelopeFromNormalizedSourceSnapshotWithoutValidation(
	parseRegionInstanceID RegionInstanceID,
	parseEpoch uint64,
	parseInputVersion uint64,
	parseProps any,
	parseNormalizedSourceIDs []string,
	parseSourceValues map[string]any,
	parseSourceVersions map[string]uint64,
) (SnapshotEnvelope, error) {
	parseSourceVersion, parseSourceVersionErr := validateSnapshotSourceVersionConsistency(parseNormalizedSourceIDs, parseSourceVersions)
	if parseSourceVersionErr != nil {
		return SnapshotEnvelope{}, parseSourceVersionErr
	}
	return SnapshotEnvelope{
		RegionInstanceID: parseRegionInstanceID,
		Epoch:            parseEpoch,
		InputVersion:     parseInputVersion,
		SourceVersion:    parseSourceVersion,
		Props:            parseProps,
		Sources:          parseSourceValues,
	}, nil
}

// GetSnapshotFingerprint returns a stable fingerprint for a snapshot envelope.
func GetSnapshotFingerprint(parseEnvelope SnapshotEnvelope) (string, error) {
	parseFingerprintHash, parseFingerprintErr := GetSnapshotFingerprintHash(parseEnvelope)
	if parseFingerprintErr != nil {
		return "", parseFingerprintErr
	}
	return hex.EncodeToString(parseFingerprintHash[:]), nil
}

// GetSnapshotFingerprintHash returns a stable SHA-256 digest for a snapshot envelope.
func GetSnapshotFingerprintHash(parseEnvelope SnapshotEnvelope) ([sha256.Size]byte, error) {
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return [sha256.Size]byte{}, parseErr
	}
	return getSnapshotFingerprintHashWithoutValidation(parseEnvelope)
}

// getSnapshotFingerprintHashWithoutValidation computes one snapshot SHA-256 digest without envelope-level validation.
func getSnapshotFingerprintHashWithoutValidation(parseEnvelope SnapshotEnvelope) ([sha256.Size]byte, error) {
	parseHasher := buildSnapshotFingerprintHasher()
	if parseHasher == nil {
		return [sha256.Size]byte{}, fmt.Errorf("runtime2: snapshot fingerprint hasher is nil")
	}
	parseHasher.Reset()
	defer func() {
		parseHasher.Reset()
		storeSnapshotFingerprintHasherPool.Put(parseHasher)
	}()
	if parseErr := buildJSONHashDigest(parseHasher, parseEnvelope); parseErr != nil {
		return [sha256.Size]byte{}, fmt.Errorf("runtime2: encode snapshot fingerprint payload: %w", parseErr)
	}
	buildHash := [sha256.Size]byte{}
	buildHashBytes := parseHasher.Sum(buildHash[:0])
	copy(buildHash[:], buildHashBytes)
	return buildHash, nil
}

// buildSnapshotFingerprintHasher acquires one reusable SHA-256 hasher from pool state.
func buildSnapshotFingerprintHasher() hash.Hash {
	parseHasher, hasHasher := storeSnapshotFingerprintHasherPool.Get().(hash.Hash)
	if hasHasher && parseHasher != nil {
		return parseHasher
	}
	return sha256.New()
}
