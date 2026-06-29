package runtime2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	buildStructuredCloneSnapshotEnvelopeTokenRegionInstanceID = `{"region_instance_id":`
	buildStructuredCloneSnapshotEnvelopeTokenEpoch            = `,"epoch":`
	buildStructuredCloneSnapshotEnvelopeTokenInputVersion     = `,"input_version":`
	buildStructuredCloneSnapshotEnvelopeTokenSourceVersion    = `,"source_version":`
	buildStructuredCloneSnapshotEnvelopeTokenProps            = `,"props":`
	buildStructuredCloneSnapshotEnvelopeTokenSources          = `,"sources":`
	parseStructuredCloneSnapshotExtendedSectionProbeThreshold = 96
)

var (
	parseStructuredCloneSnapshotEnvelopeTokenPropsBytes   = []byte(buildStructuredCloneSnapshotEnvelopeTokenProps)
	parseStructuredCloneSnapshotEnvelopeTokenSourcesBytes = []byte(buildStructuredCloneSnapshotEnvelopeTokenSources)
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
	if parseEnvelope.Props == nil && len(parseEnvelope.Sources) == 0 {
		buildPayload := make([]byte, 0, getStructuredCloneSnapshotEnvelopeJSONLength(parseEnvelope, 0, 0))
		buildPayload = appendStructuredCloneSnapshotEnvelopeJSON(buildPayload, parseEnvelope, nil, nil)
		return buildPayload, nil
	}
	parsePayload, parseErr := json.Marshal(parseEnvelope)
	if parseErr != nil {
		return nil, fmt.Errorf("runtime2: encode structured-clone snapshot envelope: %w", parseErr)
	}
	return parsePayload, nil
}

// ParseStructuredCloneSnapshotEnvelopeJSON decodes and validates one structured-clone snapshot envelope payload.
func ParseStructuredCloneSnapshotEnvelopeJSON(parsePayload []byte) (SnapshotEnvelope, error) {
	if len(parsePayload) <= parseStructuredCloneSnapshotExtendedSectionProbeThreshold ||
		!hasParseStructuredCloneSnapshotEnvelopeExtendedSections(parsePayload) {
		if parseEnvelopeFast, hasParseEnvelopeFast := parseStructuredCloneSnapshotEnvelopeHeaderOnly(parsePayload); hasParseEnvelopeFast {
			if parseErr := ValidateSnapshotEnvelope(parseEnvelopeFast); parseErr != nil {
				return SnapshotEnvelope{}, parseErr
			}
			return parseEnvelopeFast, nil
		}
	}
	var parseEnvelope SnapshotEnvelope
	if parseErr := json.Unmarshal(parsePayload, &parseEnvelope); parseErr != nil {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode structured-clone snapshot envelope: %w", parseErr)
	}
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	return parseEnvelope, nil
}

// hasParseStructuredCloneSnapshotEnvelopeExtendedSections reports whether one payload includes props or sources sections.
func hasParseStructuredCloneSnapshotEnvelopeExtendedSections(parsePayload []byte) bool {
	return bytes.Contains(parsePayload, parseStructuredCloneSnapshotEnvelopeTokenPropsBytes) ||
		bytes.Contains(parsePayload, parseStructuredCloneSnapshotEnvelopeTokenSourcesBytes)
}

// getStructuredCloneSnapshotEnvelopeJSONLength returns one capacity hint for a structured-clone snapshot envelope JSON payload.
func getStructuredCloneSnapshotEnvelopeJSONLength(parseEnvelope SnapshotEnvelope, parsePropsLength int, parseSourcesLength int) int {
	buildLength := len(buildStructuredCloneSnapshotEnvelopeTokenRegionInstanceID) +
		2 + len(string(parseEnvelope.RegionInstanceID)) +
		len(buildStructuredCloneSnapshotEnvelopeTokenEpoch) + 20 +
		len(buildStructuredCloneSnapshotEnvelopeTokenInputVersion) + 20
	if parseEnvelope.SourceVersion > 0 {
		buildLength += len(buildStructuredCloneSnapshotEnvelopeTokenSourceVersion) + 20
	}
	if parsePropsLength > 0 {
		buildLength += len(buildStructuredCloneSnapshotEnvelopeTokenProps) + parsePropsLength
	}
	if parseSourcesLength > 0 {
		buildLength += len(buildStructuredCloneSnapshotEnvelopeTokenSources) + parseSourcesLength
	}
	return buildLength + 1
}

// appendStructuredCloneSnapshotEnvelopeJSON appends one structured-clone snapshot envelope JSON payload to parseDst.
func appendStructuredCloneSnapshotEnvelopeJSON(
	parseDst []byte,
	parseEnvelope SnapshotEnvelope,
	parsePropsPayload []byte,
	parseSourcesPayload []byte,
) []byte {
	parseDst = append(parseDst, buildStructuredCloneSnapshotEnvelopeTokenRegionInstanceID...)
	parseDst = strconv.AppendQuote(parseDst, string(parseEnvelope.RegionInstanceID))
	parseDst = append(parseDst, buildStructuredCloneSnapshotEnvelopeTokenEpoch...)
	parseDst = strconv.AppendUint(parseDst, parseEnvelope.Epoch, 10)
	parseDst = append(parseDst, buildStructuredCloneSnapshotEnvelopeTokenInputVersion...)
	parseDst = strconv.AppendUint(parseDst, parseEnvelope.InputVersion, 10)
	if parseEnvelope.SourceVersion > 0 {
		parseDst = append(parseDst, buildStructuredCloneSnapshotEnvelopeTokenSourceVersion...)
		parseDst = strconv.AppendUint(parseDst, parseEnvelope.SourceVersion, 10)
	}
	if len(parsePropsPayload) > 0 {
		parseDst = append(parseDst, buildStructuredCloneSnapshotEnvelopeTokenProps...)
		parseDst = append(parseDst, parsePropsPayload...)
	}
	if len(parseSourcesPayload) > 0 {
		parseDst = append(parseDst, buildStructuredCloneSnapshotEnvelopeTokenSources...)
		parseDst = append(parseDst, parseSourcesPayload...)
	}
	parseDst = append(parseDst, '}')
	return parseDst
}

// parseStructuredCloneSnapshotEnvelopeHeaderOnly decodes one header-only structured-clone snapshot envelope produced by appendStructuredCloneSnapshotEnvelopeJSON.
func parseStructuredCloneSnapshotEnvelopeHeaderOnly(parsePayload []byte) (SnapshotEnvelope, bool) {
	parseOffset, hasParseLiteral := hasStructuredCloneSnapshotLiteralAt(parsePayload, 0, buildStructuredCloneSnapshotEnvelopeTokenRegionInstanceID)
	if !hasParseLiteral {
		return SnapshotEnvelope{}, false
	}
	parseRegionStart, parseRegionEnd, parseOffset, hasParseRegionEscape, hasParseQuotedText := parseStructuredCloneSnapshotQuotedTextSpan(parsePayload, parseOffset)
	if !hasParseQuotedText {
		return SnapshotEnvelope{}, false
	}
	parseOffset, hasParseLiteral = hasStructuredCloneSnapshotLiteralAt(parsePayload, parseOffset, buildStructuredCloneSnapshotEnvelopeTokenEpoch)
	if !hasParseLiteral {
		return SnapshotEnvelope{}, false
	}
	parseEpoch, parseOffset, hasParseUint := parseStructuredCloneSnapshotUint(parsePayload, parseOffset)
	if !hasParseUint {
		return SnapshotEnvelope{}, false
	}
	parseOffset, hasParseLiteral = hasStructuredCloneSnapshotLiteralAt(parsePayload, parseOffset, buildStructuredCloneSnapshotEnvelopeTokenInputVersion)
	if !hasParseLiteral {
		return SnapshotEnvelope{}, false
	}
	parseInputVersion, parseOffset, hasParseUint := parseStructuredCloneSnapshotUint(parsePayload, parseOffset)
	if !hasParseUint {
		return SnapshotEnvelope{}, false
	}
	var parseSourceVersion uint64
	parseSourceOffset, hasParseSourceVersionLiteral := hasStructuredCloneSnapshotLiteralAt(parsePayload, parseOffset, buildStructuredCloneSnapshotEnvelopeTokenSourceVersion)
	if hasParseSourceVersionLiteral {
		parseSourceVersionValue, parseSourceValueOffset, hasParseSourceVersion := parseStructuredCloneSnapshotUint(parsePayload, parseSourceOffset)
		if !hasParseSourceVersion {
			return SnapshotEnvelope{}, false
		}
		parseSourceVersion = parseSourceVersionValue
		parseOffset = parseSourceValueOffset
	}
	if parseOffset < 0 || parseOffset >= len(parsePayload) || parsePayload[parseOffset] != '}' || parseOffset != len(parsePayload)-1 {
		return SnapshotEnvelope{}, false
	}
	var parseRegionInstanceID string
	if hasParseRegionEscape {
		parseDecodedRegion, parseDecodeErr := strconv.Unquote(string(parsePayload[parseRegionStart-1 : parseRegionEnd+1]))
		if parseDecodeErr != nil {
			return SnapshotEnvelope{}, false
		}
		parseRegionInstanceID = parseDecodedRegion
	} else {
		parseRegionInstanceID = string(parsePayload[parseRegionStart:parseRegionEnd])
	}
	return SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID(parseRegionInstanceID),
		Epoch:            parseEpoch,
		InputVersion:     parseInputVersion,
		SourceVersion:    parseSourceVersion,
	}, true
}

// hasStructuredCloneSnapshotLiteralAt reports whether one literal token appears at the given byte offset.
func hasStructuredCloneSnapshotLiteralAt(parsePayload []byte, parseOffset int, parseLiteral string) (int, bool) {
	if parseOffset < 0 || len(parsePayload)-parseOffset < len(parseLiteral) {
		return 0, false
	}
	for parseIndex := 0; parseIndex < len(parseLiteral); parseIndex++ {
		if parsePayload[parseOffset+parseIndex] != parseLiteral[parseIndex] {
			return 0, false
		}
	}
	return parseOffset + len(parseLiteral), true
}

// parseStructuredCloneSnapshotQuotedTextSpan parses one JSON quoted string and returns raw byte span indexes plus parse state.
func parseStructuredCloneSnapshotQuotedTextSpan(parsePayload []byte, parseOffset int) (int, int, int, bool, bool) {
	if parseOffset < 0 || parseOffset >= len(parsePayload) || parsePayload[parseOffset] != '"' {
		return 0, 0, 0, false, false
	}
	parseStart := parseOffset + 1
	hasParseEscape := false
	for parseCursor := parseStart; parseCursor < len(parsePayload); parseCursor++ {
		parseByte := parsePayload[parseCursor]
		if parseByte < 0x20 {
			return 0, 0, 0, false, false
		}
		if parseByte == '\\' {
			hasParseEscape = true
			continue
		}
		if parseByte != '"' {
			continue
		}
		parseBackslashCount := 0
		for parseBack := parseCursor - 1; parseBack >= parseStart && parsePayload[parseBack] == '\\'; parseBack-- {
			parseBackslashCount++
		}
		if parseBackslashCount%2 == 1 {
			continue
		}
		return parseStart, parseCursor, parseCursor + 1, hasParseEscape, true
	}
	return 0, 0, 0, false, false
}

// parseStructuredCloneSnapshotUint parses one JSON uint64 value at the given byte offset.
func parseStructuredCloneSnapshotUint(parsePayload []byte, parseOffset int) (uint64, int, bool) {
	if parseOffset < 0 || parseOffset >= len(parsePayload) || parsePayload[parseOffset] < '0' || parsePayload[parseOffset] > '9' {
		return 0, 0, false
	}
	var parseValue uint64
	for parseOffset < len(parsePayload) {
		parseByte := parsePayload[parseOffset]
		if parseByte < '0' || parseByte > '9' {
			break
		}
		parseDigit := uint64(parseByte - '0')
		if parseValue > (^uint64(0)-parseDigit)/10 {
			return 0, 0, false
		}
		parseValue = parseValue*10 + parseDigit
		parseOffset++
	}
	return parseValue, parseOffset, true
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
