package runtime2

import (
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// buildBinarySourceIDsCache pools the sorted source-ID slice backing array to amortize per-call allocations.
type buildBinarySourceIDsCache struct {
	getSourceIDs []string
}

var storeBinarySourceIDsPool = sync.Pool{
	New: func() any {
		return &buildBinarySourceIDsCache{getSourceIDs: make([]string, 0, 8)}
	},
}

// BuildBinarySnapshotBody encodes one validated snapshot envelope into the runtime2 binary body format.
func BuildBinarySnapshotBody(parseEnvelope SnapshotEnvelope) ([]byte, error) {
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return nil, parseErr
	}
	parseRegionIDLength := 2 + len(string(parseEnvelope.RegionInstanceID))
	// Capacity estimate: region + 3×uint64(24) + props-len(4) + props-est(64) + source-table-len(4) + source-values-len(4) + per-source estimate.
	parseCap := parseRegionIDLength + 24 + 4 + 64 + 4 + 4 + (len(parseEnvelope.Sources)+1)*16
	return appendBinarySnapshotBody(make([]byte, 0, parseCap), parseEnvelope)
}

// releaseSnapshotBodySourceIDsCache clears string refs and returns one source-ID cache to the pool.
func releaseSnapshotBodySourceIDsCache(parseCache *buildBinarySourceIDsCache) {
	for parseIDIdx := range parseCache.getSourceIDs {
		parseCache.getSourceIDs[parseIDIdx] = ""
	}
	storeBinarySourceIDsPool.Put(parseCache)
}

// appendBinarySnapshotBody appends one encoded snapshot body into dst and returns the extended slice.
// The caller must have already validated parseEnvelope with ValidateSnapshotEnvelope.
func appendBinarySnapshotBody(dst []byte, parseEnvelope SnapshotEnvelope) ([]byte, error) {
	var parseErr error
	dst, parseErr = appendBinaryLengthPrefixedString(dst, string(parseEnvelope.RegionInstanceID))
	if parseErr != nil {
		return nil, parseErr
	}
	dst = appendBinaryUint64(dst, parseEnvelope.Epoch)
	dst = appendBinaryUint64(dst, parseEnvelope.InputVersion)
	dst = appendBinaryUint64(dst, parseEnvelope.SourceVersion)
	// Props: write-back length prefix so no temp allocation is needed.
	parsePropsLenOff := len(dst)
	dst = append(dst, 0, 0, 0, 0)
	parsePropsStart := len(dst)
	dst, parseErr = buildBinarySourceValueInto(dst, parseEnvelope.Props)
	if parseErr != nil {
		return nil, parseErr
	}
	setBinaryUint32At(dst, parsePropsLenOff, uint32(len(dst)-parsePropsStart))
	// Source IDs: pool the temporary sorted-key slice; clear string refs before returning to pool.
	parseSourceIDsCache := storeBinarySourceIDsPool.Get().(*buildBinarySourceIDsCache)
	parseSourceIDsCache.getSourceIDs = parseSourceIDsCache.getSourceIDs[:0]
	defer releaseSnapshotBodySourceIDsCache(parseSourceIDsCache)
	parseSourceIDsCache.getSourceIDs, parseErr = buildBinaryCanonicalSourceIDsInto(parseSourceIDsCache.getSourceIDs, parseEnvelope.Sources)
	if parseErr != nil {
		return nil, parseErr
	}
	parseNormalizedSourceIDs := parseSourceIDsCache.getSourceIDs
	// Source ID table: write-back length prefix.
	parseSourceIDTableLengthOffset := len(dst)
	dst = appendBinaryUint32(dst, 0)
	parseSourceIDTableStart := len(dst)
	dst, parseErr = appendBinarySourceIDTableFromNormalized(dst, parseNormalizedSourceIDs)
	if parseErr != nil {
		return nil, parseErr
	}
	setBinaryUint32At(dst, parseSourceIDTableLengthOffset, uint32(len(dst)-parseSourceIDTableStart))
	// Source values section: write-back length prefix.
	parseSourceValuesLengthOffset := len(dst)
	dst = appendBinaryUint32(dst, 0)
	parseSourceValuesStart := len(dst)
	dst, parseErr = appendBinarySourceValuesSection(dst, parseNormalizedSourceIDs, parseEnvelope.Sources)
	if parseErr != nil {
		return nil, parseErr
	}
	setBinaryUint32At(dst, parseSourceValuesLengthOffset, uint32(len(dst)-parseSourceValuesStart))
	return dst, nil
}

// buildBinaryCanonicalSourceIDsInto validates and sorts source IDs from one source-value map into parseDst and returns the extended slice.
func buildBinaryCanonicalSourceIDsInto(parseDst []string, parseSourceValues map[string]any) ([]string, error) {
	for parseSourceID := range parseSourceValues {
		if parseSourceID == "" {
			return parseDst, fmt.Errorf("runtime2: source ID is required")
		}
		if strings.TrimSpace(parseSourceID) != parseSourceID {
			return parseDst, fmt.Errorf("runtime2: source ID %q must not contain surrounding whitespace", parseSourceID)
		}
		parseUnsupportedRune, hasUnsupportedRune := getBinarySourceIDUnsupportedRune(parseSourceID)
		if hasUnsupportedRune {
			return parseDst, fmt.Errorf("runtime2: source ID %q contains unsupported character %q", parseSourceID, string(parseUnsupportedRune))
		}
		parseDst = append(parseDst, parseSourceID)
	}
	sort.Strings(parseDst)
	return parseDst, nil
}

// buildBinaryCanonicalSourceIDs validates and sorts source IDs from one source-value map without duplicate-map checks.
func buildBinaryCanonicalSourceIDs(parseSourceValues map[string]any) ([]string, error) {
	if len(parseSourceValues) == 0 {
		return nil, nil
	}
	return buildBinaryCanonicalSourceIDsInto(make([]string, 0, len(parseSourceValues)), parseSourceValues)
}

// ParseBinarySnapshotBody decodes one binary snapshot body into a validated snapshot envelope.
func ParseBinarySnapshotBody(parsePayload []byte) (SnapshotEnvelope, error) {
	parseRegionInstanceIDText, parseOffset, parseErr := parseBinaryString(parsePayload, 0, "snapshot-body", "region_instance_id")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseEpoch, parseOffset, parseErr := parseBinaryUint64(parsePayload, parseOffset, "snapshot-body", "epoch")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseInputVersion, parseOffset, parseErr := parseBinaryUint64(parsePayload, parseOffset, "snapshot-body", "input_version")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseSourceVersion, parseOffset, parseErr := parseBinaryUint64(parsePayload, parseOffset, "snapshot-body", "source_version")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parsePropsLength, parseOffset, parseErr := parseBinaryUint32(parsePayload, parseOffset, "snapshot-body", "props length")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parsePropsSpan, parseOffset, parseErr := parseBinaryPayloadSpan(parsePayload, parseOffset, int(parsePropsLength), "snapshot-body", "props payload")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseProps, parseErr := ParseBinaryPropsValue(parsePropsSpan)
	if parseErr != nil {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot props: %w", parseErr)
	}
	parseSourceIDTableLength, parseOffset, parseErr := parseBinaryUint32(parsePayload, parseOffset, "snapshot-body", "source-id-table length")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseSourceIDTableSpan, parseOffset, parseErr := parseBinaryPayloadSpan(parsePayload, parseOffset, int(parseSourceIDTableLength), "snapshot-body", "source-id-table payload")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseSourceValuesLength, parseOffset, parseErr := parseBinaryUint32(parsePayload, parseOffset, "snapshot-body", "source-values length")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseSourceValuesSpan, parseOffset, parseErr := parseBinaryPayloadSpan(parsePayload, parseOffset, int(parseSourceValuesLength), "snapshot-body", "source-values payload")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	// Pool the source-ID slice for the lifetime of this parse; source strings become map keys and live on.
	parseSourceIDsCache := storeBinarySourceIDsPool.Get().(*buildBinarySourceIDsCache)
	parseSourceIDsCache.getSourceIDs = parseSourceIDsCache.getSourceIDs[:0]
	var parseIDSectionErr error
	parseSourceIDsCache.getSourceIDs, parseIDSectionErr = parseBinarySourceIDTableInto(parseSourceIDsCache.getSourceIDs, parseSourceIDTableSpan)
	var parseSources map[string]any
	var parseSourceSectionErr error
	if parseIDSectionErr == nil {
		parseSources, parseSourceSectionErr = parseBinarySourceValuesSection(parseSourceIDsCache.getSourceIDs, parseSourceValuesSpan)
	}
	for parseIDIdx := range parseSourceIDsCache.getSourceIDs {
		parseSourceIDsCache.getSourceIDs[parseIDIdx] = ""
	}
	storeBinarySourceIDsPool.Put(parseSourceIDsCache)
	if parseIDSectionErr != nil {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot source-id table: %w", parseIDSectionErr)
	}
	if parseSourceSectionErr != nil {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot source-values: %w", parseSourceSectionErr)
	}
	if parseOffset != len(parsePayload) {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: snapshot-body has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	parseRegionInstanceID, parseErr := ParseRegionInstanceID(parseRegionInstanceIDText)
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

// getBinaryLengthPrefixedStringLength returns the encoded size for one uint16-length-prefixed string.
func getBinaryLengthPrefixedStringLength(parseValue string) (int, error) {
	if len(parseValue) > 0xFFFF {
		return 0, fmt.Errorf("runtime2: binary string %q is too large", parseValue)
	}
	return 2 + len(parseValue), nil
}

// appendBinaryLengthPrefixedString appends one uint16-length-prefixed string to the provided payload buffer.
func appendBinaryLengthPrefixedString(parsePayload []byte, parseValue string) ([]byte, error) {
	if len(parseValue) > 0xFFFF {
		return nil, fmt.Errorf("runtime2: binary string %q is too large", parseValue)
	}
	parsePayload = appendBinaryUint16(parsePayload, uint16(len(parseValue)))
	parsePayload = append(parsePayload, parseValue...)
	return parsePayload, nil
}

// buildBinaryLengthPrefixedString encodes one uint16-length-prefixed string.
func buildBinaryLengthPrefixedString(parseValue string) ([]byte, error) {
	return appendBinaryLengthPrefixedString(make([]byte, 0, 2+len(parseValue)), parseValue)
}

// buildBinarySourceValuesSection encodes source values in the canonical order of the source-ID table.
func buildBinarySourceValuesSection(parseSourceIDs []string, parseSourceValues map[string]any) ([]byte, error) {
	parsePayload := make([]byte, 0, len(parseSourceIDs)*8)
	return appendBinarySourceValuesSection(parsePayload, parseSourceIDs, parseSourceValues)
}

// appendBinarySourceValuesSection appends source values in canonical source-ID order to the provided payload buffer.
func appendBinarySourceValuesSection(parsePayload []byte, parseSourceIDs []string, parseSourceValues map[string]any) ([]byte, error) {
	for _, parseSourceID := range parseSourceIDs {
		parseSourceValue, parseHasSourceValue := parseSourceValues[parseSourceID]
		if !parseHasSourceValue {
			return nil, fmt.Errorf("runtime2: missing source value for %q", parseSourceID)
		}
		// Write-back length: reserve placeholder, encode directly into parsePayload, fix up length.
		lenOff := len(parsePayload)
		parsePayload = append(parsePayload, 0, 0, 0, 0)
		itemStart := len(parsePayload)
		var parseErr error
		parsePayload, parseErr = buildBinarySourceValueInto(parsePayload, parseSourceValue)
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: encode source %q: %w", parseSourceID, parseErr)
		}
		binary.LittleEndian.PutUint32(parsePayload[lenOff:], uint32(len(parsePayload)-itemStart))
	}
	return parsePayload, nil
}

// parseBinarySourceValuesSection decodes source values in the canonical order of the source-ID table.
func parseBinarySourceValuesSection(parseSourceIDs []string, parsePayload []byte) (map[string]any, error) {
	if len(parseSourceIDs) == 0 {
		if len(parsePayload) != 0 {
			return nil, fmt.Errorf("runtime2: source-values payload has %d bytes with no source IDs", len(parsePayload))
		}
		return nil, nil
	}
	parseSources := make(map[string]any, len(parseSourceIDs))
	parseOffset := 0
	for parseIndex, parseSourceID := range parseSourceIDs {
		if parseOffset+4 > len(parsePayload) {
			return nil, fmt.Errorf("runtime2: decode source value[%d] length: payload is truncated", parseIndex)
		}
		parseValueLength := int(binary.LittleEndian.Uint32(parsePayload[parseOffset : parseOffset+4]))
		parseOffset += 4
		if parseValueLength < 0 || parseValueLength > len(parsePayload)-parseOffset {
			return nil, fmt.Errorf(
				"runtime2: decode source value[%d] payload: length %d exceeds payload size %d",
				parseIndex,
				parseValueLength,
				len(parsePayload)-parseOffset,
			)
		}
		parseValue, parseErr := ParseBinarySourceValue(parsePayload[parseOffset : parseOffset+parseValueLength])
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode source %q: %w", parseSourceID, parseErr)
		}
		parseSources[parseSourceID] = parseValue
		parseOffset += parseValueLength
	}
	if parseOffset != len(parsePayload) {
		return nil, fmt.Errorf("runtime2: source-values payload has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	return parseSources, nil
}
