package runtime2

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// WorkerRenderSourceEntry stores one deterministic declared-source entry for worker renderer input.
type WorkerRenderSourceEntry struct {
	GetSourceID    string
	GetSourceValue any
}

// WorkerRenderInput stores deterministic worker renderer input derived from one snapshot envelope.
type WorkerRenderInput struct {
	GetProps         any
	GetSourceVersion uint64
	GetSourceEntries []WorkerRenderSourceEntry
}

// BuildWorkerRenderInput builds deterministic worker renderer input from one validated snapshot envelope.
func BuildWorkerRenderInput(parseSnapshot SnapshotEnvelope) (WorkerRenderInput, error) {
	getRenderInput, _, parseErr := buildWorkerRenderInputWithSourceOrder(parseSnapshot, nil)
	if parseErr != nil {
		return WorkerRenderInput{}, parseErr
	}
	return getRenderInput, nil
}

// buildWorkerRenderInputWithSourceOrder builds deterministic worker renderer input and reuses one cached normalized source-ID order when the source keyset is unchanged.
func buildWorkerRenderInputWithSourceOrder(parseSnapshot SnapshotEnvelope, parseCachedSourceIDs []string) (WorkerRenderInput, []string, error) {
	return buildWorkerRenderInputWithSourceOrderChecked(parseSnapshot, parseCachedSourceIDs, false)
}

// buildWorkerRenderInputWithSourceOrderFromValidatedSnapshot builds worker renderer input from a snapshot envelope that has already passed validation.
func buildWorkerRenderInputWithSourceOrderFromValidatedSnapshot(parseSnapshot SnapshotEnvelope, parseCachedSourceIDs []string) (WorkerRenderInput, []string, error) {
	return buildWorkerRenderInputWithSourceOrderChecked(parseSnapshot, parseCachedSourceIDs, true)
}

// buildWorkerRenderInputWithSourceOrderChecked builds deterministic worker input and conditionally validates the source snapshot before extraction.
func buildWorkerRenderInputWithSourceOrderChecked(
	parseSnapshot SnapshotEnvelope,
	parseCachedSourceIDs []string,
	parseHasSnapshotValidated bool,
) (WorkerRenderInput, []string, error) {
	if parseSnapshot.RegionInstanceID == "" {
		return WorkerRenderInput{}, nil, nil
	}
	if !parseHasSnapshotValidated {
		if parseSnapshotErr := ValidateSnapshotEnvelope(parseSnapshot); parseSnapshotErr != nil {
			return WorkerRenderInput{}, nil, parseSnapshotErr
		}
	}
	hasSourceOrderMatch := hasWorkerRenderSnapshotSourceOrderMatch(parseSnapshot.Sources, parseCachedSourceIDs)
	parseSourceIDs := parseCachedSourceIDs
	if !hasSourceOrderMatch {
		getNormalizedSourceIDs, parseSourceIDsErr := buildWorkerRenderNormalizedSourceIDsFromSnapshot(parseSnapshot.Sources)
		if parseSourceIDsErr != nil {
			return WorkerRenderInput{}, nil, parseSourceIDsErr
		}
		parseSourceIDs = getNormalizedSourceIDs
	}
	if len(parseSourceIDs) == 0 {
		return WorkerRenderInput{
			GetProps:         parseSnapshot.Props,
			GetSourceVersion: parseSnapshot.SourceVersion,
		}, parseSourceIDs, nil
	}
	buildSourceEntries := make([]WorkerRenderSourceEntry, len(parseSourceIDs))
	for parseSourceIndex, parseSourceID := range parseSourceIDs {
		buildSourceEntries[parseSourceIndex] = WorkerRenderSourceEntry{
			GetSourceID:    parseSourceID,
			GetSourceValue: parseSnapshot.Sources[parseSourceID],
		}
	}
	return WorkerRenderInput{
		GetProps:         parseSnapshot.Props,
		GetSourceVersion: parseSnapshot.SourceVersion,
		GetSourceEntries: buildSourceEntries,
	}, parseSourceIDs, nil
}

// hasWorkerRenderSnapshotSourceOrderMatch reports whether one cached source-ID order exactly matches the current snapshot source keyset.
func hasWorkerRenderSnapshotSourceOrderMatch(parseSnapshotSources map[string]any, parseCachedSourceIDs []string) bool {
	if len(parseSnapshotSources) != len(parseCachedSourceIDs) {
		return false
	}
	for _, parseSourceID := range parseCachedSourceIDs {
		if _, hasSourceID := parseSnapshotSources[parseSourceID]; !hasSourceID {
			return false
		}
	}
	return true
}

// buildWorkerRenderNormalizedSourceIDsFromSnapshot validates and sorts one snapshot source-key set into normalized source ID order.
func buildWorkerRenderNormalizedSourceIDsFromSnapshot(parseSnapshotSources map[string]any) ([]string, error) {
	if len(parseSnapshotSources) == 0 {
		return nil, nil
	}
	buildSourceIDs := make([]string, 0, len(parseSnapshotSources))
	for parseSourceID := range parseSnapshotSources {
		if parseSourceIDErr := validateWorkerRenderSourceID(parseSourceID); parseSourceIDErr != nil {
			return nil, parseSourceIDErr
		}
		buildSourceIDs = append(buildSourceIDs, parseSourceID)
	}
	sort.Strings(buildSourceIDs)
	return buildSourceIDs, nil
}

// validateWorkerRenderSourceID verifies one source ID uses the supported canonical source-key format.
func validateWorkerRenderSourceID(parseSourceID string) error {
	if parseSourceID == "" {
		return fmt.Errorf("runtime2: source ID is required")
	}
	parseTrimmedSourceID := strings.TrimSpace(parseSourceID)
	if parseTrimmedSourceID != parseSourceID {
		return fmt.Errorf("runtime2: source ID %q must not contain surrounding whitespace", parseSourceID)
	}
	for parseIndex := 0; parseIndex < len(parseSourceID); {
		parseByte := parseSourceID[parseIndex]
		if parseByte < utf8.RuneSelf {
			if isWorkerRenderSourceIDByteValid(parseByte) {
				parseIndex++
				continue
			}
			return fmt.Errorf("runtime2: source ID %q contains unsupported character %q", parseSourceID, string(parseByte))
		}
		parseRune, parseRuneSize := utf8.DecodeRuneInString(parseSourceID[parseIndex:])
		if parseRune == utf8.RuneError && parseRuneSize == 1 {
			return fmt.Errorf("runtime2: source ID %q contains unsupported character %q", parseSourceID, string(parseSourceID[parseIndex]))
		}
		if parseRune > utf8.RuneSelf-1 || !isWorkerRenderSourceIDByteValid(byte(parseRune)) {
			return fmt.Errorf("runtime2: source ID %q contains unsupported character %q", parseSourceID, string(parseRune))
		}
		parseIndex += parseRuneSize
	}
	return nil
}

// isWorkerRenderSourceIDByteValid reports whether one ASCII byte is valid in source IDs.
func isWorkerRenderSourceIDByteValid(parseByte byte) bool {
	if parseByte == '.' || parseByte == '-' || parseByte == '_' || parseByte == ':' {
		return true
	}
	if parseByte >= 'a' && parseByte <= 'z' {
		return true
	}
	if parseByte >= 'A' && parseByte <= 'Z' {
		return true
	}
	return parseByte >= '0' && parseByte <= '9'
}
