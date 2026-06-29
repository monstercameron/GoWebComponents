package runtime2

import (
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
	"unicode"
	"unicode/utf8"
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
	// Capacity estimate: region + 3×uint64(24) + props-len(4) + props-est(64) + source-table-len(4) + source-values-len(4) + per-source estimate.
	parseCap := getBinarySnapshotBodyCapacityHint(parseEnvelope)
	return appendBinarySnapshotBody(make([]byte, 0, parseCap), parseEnvelope)
}

// getBinarySnapshotBodyCapacityHint returns one coarse binary snapshot-body capacity hint that reduces grow-and-copy churn.
func getBinarySnapshotBodyCapacityHint(parseEnvelope SnapshotEnvelope) int {
	parseRegionIDLength := 2 + len(string(parseEnvelope.RegionInstanceID))
	parsePropsLengthHint := getBinarySnapshotBodyValueCapacityHint(parseEnvelope.Props)
	parseSourceIDTableLengthHint := 2
	parseSourceValuesLengthHint := 0
	for parseSourceID, parseSourceValue := range parseEnvelope.Sources {
		parseSourceIDTableLengthHint += 2 + len(parseSourceID)
		parseSourceValuesLengthHint += 4 + getBinarySnapshotBodyValueCapacityHint(parseSourceValue)
	}
	return parseRegionIDLength +
		24 + // epoch + input_version + source_version
		4 + parsePropsLengthHint +
		4 + parseSourceIDTableLengthHint +
		4 + parseSourceValuesLengthHint
}

// getBinarySnapshotBodyValueCapacityHint returns one low-cost encoded-size hint for one binary source value.
func getBinarySnapshotBodyValueCapacityHint(parseValue any) int {
	switch getValue := parseValue.(type) {
	case nil, bool:
		return 1
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64:
		return 9
	case string:
		return 5 + len(getValue)
	case []any:
		return 16 + len(getValue)*32
	case map[string]any:
		parseHint := 24 + len(getValue)*40
		for parseKey := range getValue {
			parseHint += len(parseKey)
		}
		return parseHint
	default:
		return 48
	}
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
	parseSourceIDsCache.getSourceIDs, parseErr = buildBinaryCanonicalSourceIDsInto(parseSourceIDsCache.getSourceIDs, parseEnvelope.Sources)
	if parseErr != nil {
		releaseSnapshotBodySourceIDsCache(parseSourceIDsCache)
		return nil, parseErr
	}
	parseNormalizedSourceIDs := parseSourceIDsCache.getSourceIDs
	// Source ID table: write-back length prefix.
	parseSourceIDTableLengthOffset := len(dst)
	dst = appendBinaryUint32(dst, 0)
	parseSourceIDTableStart := len(dst)
	dst, parseErr = appendBinarySourceIDTableFromNormalized(dst, parseNormalizedSourceIDs)
	if parseErr != nil {
		releaseSnapshotBodySourceIDsCache(parseSourceIDsCache)
		return nil, parseErr
	}
	setBinaryUint32At(dst, parseSourceIDTableLengthOffset, uint32(len(dst)-parseSourceIDTableStart))
	// Source values section: write-back length prefix.
	parseSourceValuesLengthOffset := len(dst)
	dst = appendBinaryUint32(dst, 0)
	parseSourceValuesStart := len(dst)
	dst, parseErr = appendBinarySourceValuesSectionTrusted(dst, parseNormalizedSourceIDs, parseEnvelope.Sources)
	if parseErr != nil {
		releaseSnapshotBodySourceIDsCache(parseSourceIDsCache)
		return nil, parseErr
	}
	setBinaryUint32At(dst, parseSourceValuesLengthOffset, uint32(len(dst)-parseSourceValuesStart))
	releaseSnapshotBodySourceIDsCache(parseSourceIDsCache)
	return dst, nil
}

// buildBinaryCanonicalSourceIDsInto validates and sorts source IDs from one source-value map into parseDst and returns the extended slice.
func buildBinaryCanonicalSourceIDsInto(parseDst []string, parseSourceValues map[string]any) ([]string, error) {
	for parseSourceID := range parseSourceValues {
		if parseSourceID == "" {
			return parseDst, fmt.Errorf("runtime2: source ID is required")
		}
		if hasBinarySourceIDSurroundingWhitespace(parseSourceID) {
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

// hasBinarySourceIDSurroundingWhitespace reports whether one source ID begins or ends with Unicode whitespace.
func hasBinarySourceIDSurroundingWhitespace(parseSourceID string) bool {
	if len(parseSourceID) == 0 {
		return false
	}
	parseFirstByte := parseSourceID[0]
	if parseFirstByte < utf8.RuneSelf {
		if hasBinarySourceIDASCIIWhitespace(parseFirstByte) {
			return true
		}
	} else {
		parseFirstRune, _ := utf8.DecodeRuneInString(parseSourceID)
		if unicode.IsSpace(parseFirstRune) {
			return true
		}
	}
	parseLastByte := parseSourceID[len(parseSourceID)-1]
	if parseLastByte < utf8.RuneSelf {
		return hasBinarySourceIDASCIIWhitespace(parseLastByte)
	}
	parseLastRune, _ := utf8.DecodeLastRuneInString(parseSourceID)
	return unicode.IsSpace(parseLastRune)
}

// hasBinarySourceIDASCIIWhitespace reports whether one ASCII byte is a whitespace code point used by Unicode space classes.
func hasBinarySourceIDASCIIWhitespace(parseByte byte) bool {
	return parseByte == ' ' || (parseByte >= '\t' && parseByte <= '\r')
}

// ParseBinarySnapshotBody decodes one binary snapshot body into a validated snapshot envelope.
func ParseBinarySnapshotBody(parsePayload []byte) (SnapshotEnvelope, error) {
	parseOffset := 0
	if parseOffset+2 > len(parsePayload) {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot-body region_instance_id: length is truncated")
	}
	parseRegionInstanceIDLength := int(binary.LittleEndian.Uint16(parsePayload[parseOffset : parseOffset+2]))
	parseOffset += 2
	if parseRegionInstanceIDLength > len(parsePayload)-parseOffset {
		return SnapshotEnvelope{}, fmt.Errorf(
			"runtime2: decode snapshot-body region_instance_id: length %d exceeds payload size %d",
			parseRegionInstanceIDLength,
			len(parsePayload)-parseOffset,
		)
	}
	parseRegionInstanceIDText := string(parsePayload[parseOffset : parseOffset+parseRegionInstanceIDLength])
	parseOffset += parseRegionInstanceIDLength
	parseRegionInstanceID, parseRegionInstanceIDErr := ParseRegionInstanceID(parseRegionInstanceIDText)
	if parseRegionInstanceIDErr != nil {
		return SnapshotEnvelope{}, parseRegionInstanceIDErr
	}
	if parseOffset+8 > len(parsePayload) {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot-body epoch: payload is truncated")
	}
	parseEpoch := binary.LittleEndian.Uint64(parsePayload[parseOffset : parseOffset+8])
	parseOffset += 8
	if parseOffset+8 > len(parsePayload) {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot-body input_version: payload is truncated")
	}
	parseInputVersion := binary.LittleEndian.Uint64(parsePayload[parseOffset : parseOffset+8])
	parseOffset += 8
	if parseOffset+8 > len(parsePayload) {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot-body source_version: payload is truncated")
	}
	parseSourceVersion := binary.LittleEndian.Uint64(parsePayload[parseOffset : parseOffset+8])
	parseOffset += 8
	if parseOffset+4 > len(parsePayload) {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot-body props length: payload is truncated")
	}
	parsePropsLength := int(binary.LittleEndian.Uint32(parsePayload[parseOffset : parseOffset+4]))
	parseOffset += 4
	if parsePropsLength > len(parsePayload)-parseOffset {
		return SnapshotEnvelope{}, fmt.Errorf(
			"runtime2: decode snapshot-body props payload: length %d exceeds payload size %d",
			parsePropsLength,
			len(parsePayload)-parseOffset,
		)
	}
	parsePropsSpan := parsePayload[parseOffset : parseOffset+parsePropsLength]
	parseOffset += parsePropsLength
	parseProps, parsePropsErr := ParseBinaryPropsValue(parsePropsSpan)
	if parsePropsErr != nil {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot props: %w", parsePropsErr)
	}
	if parsePropsValidateErr := ValidateSerializableProps(parseProps); parsePropsValidateErr != nil {
		return SnapshotEnvelope{}, parsePropsValidateErr
	}
	if parseOffset+4 > len(parsePayload) {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot-body source-id-table length: payload is truncated")
	}
	parseSourceIDTableLength := int(binary.LittleEndian.Uint32(parsePayload[parseOffset : parseOffset+4]))
	parseOffset += 4
	if parseSourceIDTableLength > len(parsePayload)-parseOffset {
		return SnapshotEnvelope{}, fmt.Errorf(
			"runtime2: decode snapshot-body source-id-table payload: length %d exceeds payload size %d",
			parseSourceIDTableLength,
			len(parsePayload)-parseOffset,
		)
	}
	parseSourceIDTableSpan := parsePayload[parseOffset : parseOffset+parseSourceIDTableLength]
	parseOffset += parseSourceIDTableLength
	if parseOffset+4 > len(parsePayload) {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot-body source-values length: payload is truncated")
	}
	parseSourceValuesLength := int(binary.LittleEndian.Uint32(parsePayload[parseOffset : parseOffset+4]))
	parseOffset += 4
	if parseSourceValuesLength > len(parsePayload)-parseOffset {
		return SnapshotEnvelope{}, fmt.Errorf(
			"runtime2: decode snapshot-body source-values payload: length %d exceeds payload size %d",
			parseSourceValuesLength,
			len(parsePayload)-parseOffset,
		)
	}
	parseSourceValuesSpan := parsePayload[parseOffset : parseOffset+parseSourceValuesLength]
	parseOffset += parseSourceValuesLength
	// Pool the source-ID slice for the lifetime of this parse; source strings become map keys and live on.
	parseSourceIDsCache := storeBinarySourceIDsPool.Get().(*buildBinarySourceIDsCache)
	parseSourceIDsCache.getSourceIDs = parseSourceIDsCache.getSourceIDs[:0]
	parseSourceIDs, parseSourceIDsErr := parseBinarySourceIDTableInto(parseSourceIDsCache.getSourceIDs, parseSourceIDTableSpan)
	if parseSourceIDsErr != nil {
		releaseSnapshotBodySourceIDsCache(parseSourceIDsCache)
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot source-id table: %w", parseSourceIDsErr)
	}
	parseSourceIDsCache.getSourceIDs = parseSourceIDs
	parseSources, parseSourcesErr := parseBinarySourceValuesSection(parseSourceIDsCache.getSourceIDs, parseSourceValuesSpan)
	if parseSourcesErr != nil {
		releaseSnapshotBodySourceIDsCache(parseSourceIDsCache)
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot source-values: %w", parseSourcesErr)
	}
	releaseSnapshotBodySourceIDsCache(parseSourceIDsCache)
	if parseOffset != len(parsePayload) {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: snapshot-body has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	if parseVersionsErr := validateBinarySnapshotBodyRequiredVersions(parseEpoch, parseInputVersion); parseVersionsErr != nil {
		return SnapshotEnvelope{}, parseVersionsErr
	}
	return SnapshotEnvelope{
		RegionInstanceID: parseRegionInstanceID,
		Epoch:            parseEpoch,
		InputVersion:     parseInputVersion,
		SourceVersion:    parseSourceVersion,
		Props:            parseProps,
		Sources:          parseSources,
	}, nil
}

// validateBinarySnapshotBodyRequiredVersions validates required snapshot version fields already decoded from the body.
func validateBinarySnapshotBodyRequiredVersions(parseEpoch uint64, parseInputVersion uint64) error {
	if parseEpoch == 0 {
		return fmt.Errorf("runtime2: snapshot epoch is required")
	}
	if parseInputVersion == 0 {
		return fmt.Errorf("runtime2: snapshot input version is required")
	}
	return nil
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

// appendBinarySourceValuesSectionTrusted appends source values in canonical source-ID order without missing-key checks.
func appendBinarySourceValuesSectionTrusted(parsePayload []byte, parseSourceIDs []string, parseSourceValues map[string]any) ([]byte, error) {
	for _, parseSourceID := range parseSourceIDs {
		lenOff := len(parsePayload)
		parsePayload = append(parsePayload, 0, 0, 0, 0)
		itemStart := len(parsePayload)
		var parseErr error
		parsePayload, parseErr = buildBinarySourceValueInto(parsePayload, parseSourceValues[parseSourceID])
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
		parseRemaining := len(parsePayload) - parseOffset
		if parseRemaining < 4 {
			return nil, fmt.Errorf("runtime2: decode source value[%d] length: payload is truncated", parseIndex)
		}
		parseValueLength := int(binary.LittleEndian.Uint32(parsePayload[parseOffset:]))
		parseOffset += 4
		parseRemaining -= 4
		if parseValueLength > parseRemaining {
			return nil, fmt.Errorf(
				"runtime2: decode source value[%d] payload: length %d exceeds payload size %d",
				parseIndex,
				parseValueLength,
				parseRemaining,
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
