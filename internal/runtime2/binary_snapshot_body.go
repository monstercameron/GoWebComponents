package runtime2

import (
	"encoding/binary"
	"fmt"
)

// BuildBinarySnapshotBody encodes one validated snapshot envelope into the runtime2 binary body format.
func BuildBinarySnapshotBody(parseEnvelope SnapshotEnvelope) ([]byte, error) {
	if parseErr := ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return nil, parseErr
	}
	parseRegionIDPayload, parseErr := buildBinaryLengthPrefixedString(string(parseEnvelope.RegionInstanceID))
	if parseErr != nil {
		return nil, parseErr
	}
	parsePropsPayload, parseErr := BuildBinaryPropsValue(parseEnvelope.Props)
	if parseErr != nil {
		return nil, parseErr
	}
	parseSourceIDs := make([]string, 0, len(parseEnvelope.Sources))
	for parseSourceID := range parseEnvelope.Sources {
		parseSourceIDs = append(parseSourceIDs, parseSourceID)
	}
	parseSourceIDTablePayload, parseErr := BuildBinarySourceIDTable(parseSourceIDs)
	if parseErr != nil {
		return nil, parseErr
	}
	parseNormalizedSourceIDs, parseErr := ParseBinarySourceIDTable(parseSourceIDTablePayload)
	if parseErr != nil {
		return nil, parseErr
	}
	parseSourceValuesPayload, parseErr := buildBinarySourceValuesSection(parseNormalizedSourceIDs, parseEnvelope.Sources)
	if parseErr != nil {
		return nil, parseErr
	}
	parsePayload := make([]byte, 0, len(parseRegionIDPayload)+32+len(parsePropsPayload)+len(parseSourceIDTablePayload)+len(parseSourceValuesPayload))
	parsePayload = append(parsePayload, parseRegionIDPayload...)
	parsePayload = append(parsePayload, buildBinaryUint64(parseEnvelope.Epoch)...)
	parsePayload = append(parsePayload, buildBinaryUint64(parseEnvelope.InputVersion)...)
	parsePayload = append(parsePayload, buildBinaryUint64(parseEnvelope.SourceVersion)...)
	parsePayload = append(parsePayload, buildBinaryUint32(uint32(len(parsePropsPayload)))...)
	parsePayload = append(parsePayload, parsePropsPayload...)
	parsePayload = append(parsePayload, buildBinaryUint32(uint32(len(parseSourceIDTablePayload)))...)
	parsePayload = append(parsePayload, parseSourceIDTablePayload...)
	parsePayload = append(parsePayload, buildBinaryUint32(uint32(len(parseSourceValuesPayload)))...)
	parsePayload = append(parsePayload, parseSourceValuesPayload...)
	return parsePayload, nil
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
	parseSourceIDs, parseErr := ParseBinarySourceIDTable(parseSourceIDTableSpan)
	if parseErr != nil {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot source-id table: %w", parseErr)
	}
	parseSourceValuesLength, parseOffset, parseErr := parseBinaryUint32(parsePayload, parseOffset, "snapshot-body", "source-values length")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseSourceValuesSpan, parseOffset, parseErr := parseBinaryPayloadSpan(parsePayload, parseOffset, int(parseSourceValuesLength), "snapshot-body", "source-values payload")
	if parseErr != nil {
		return SnapshotEnvelope{}, parseErr
	}
	parseSources, parseErr := parseBinarySourceValuesSection(parseSourceIDs, parseSourceValuesSpan)
	if parseErr != nil {
		return SnapshotEnvelope{}, fmt.Errorf("runtime2: decode snapshot source-values: %w", parseErr)
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

// buildBinaryLengthPrefixedString encodes one uint16-length-prefixed string.
func buildBinaryLengthPrefixedString(parseValue string) ([]byte, error) {
	if len(parseValue) > 0xFFFF {
		return nil, fmt.Errorf("runtime2: binary string %q is too large", parseValue)
	}
	parsePayload := make([]byte, 2+len(parseValue))
	binary.LittleEndian.PutUint16(parsePayload[0:2], uint16(len(parseValue)))
	copy(parsePayload[2:], []byte(parseValue))
	return parsePayload, nil
}

// buildBinaryUint32 encodes one uint32 value.
func buildBinaryUint32(parseValue uint32) []byte {
	parsePayload := make([]byte, 4)
	binary.LittleEndian.PutUint32(parsePayload, parseValue)
	return parsePayload
}

// buildBinaryUint64 encodes one uint64 value.
func buildBinaryUint64(parseValue uint64) []byte {
	parsePayload := make([]byte, 8)
	binary.LittleEndian.PutUint64(parsePayload, parseValue)
	return parsePayload
}

// buildBinarySourceValuesSection encodes source values in the canonical order of the source-ID table.
func buildBinarySourceValuesSection(parseSourceIDs []string, parseSourceValues map[string]any) ([]byte, error) {
	parsePayload := make([]byte, 0, len(parseSourceIDs)*8)
	for _, parseSourceID := range parseSourceIDs {
		parseSourceValue, parseHasSourceValue := parseSourceValues[parseSourceID]
		if !parseHasSourceValue {
			return nil, fmt.Errorf("runtime2: missing source value for %q", parseSourceID)
		}
		parseValuePayload, parseErr := BuildBinarySourceValue(parseSourceValue)
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: encode source %q: %w", parseSourceID, parseErr)
		}
		parsePayload = append(parsePayload, buildBinaryUint32(uint32(len(parseValuePayload)))...)
		parsePayload = append(parsePayload, parseValuePayload...)
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
		parseValueLength, parseNextOffset, parseErr := parseBinaryUint32(parsePayload, parseOffset, "source-values", fmt.Sprintf("value[%d] length", parseIndex))
		if parseErr != nil {
			return nil, parseErr
		}
		parseValueSpan, parseValueEnd, parseErr := parseBinaryPayloadSpan(parsePayload, parseNextOffset, int(parseValueLength), "source-values", fmt.Sprintf("value[%d] payload", parseIndex))
		if parseErr != nil {
			return nil, parseErr
		}
		parseValue, parseErr := ParseBinarySourceValue(parseValueSpan)
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode source %q: %w", parseSourceID, parseErr)
		}
		parseSources[parseSourceID] = parseValue
		parseOffset = parseValueEnd
	}
	if parseOffset != len(parsePayload) {
		return nil, fmt.Errorf("runtime2: source-values payload has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	return parseSources, nil
}
