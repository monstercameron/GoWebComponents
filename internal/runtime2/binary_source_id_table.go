package runtime2

import "fmt"

// BuildBinarySourceIDTable encodes one canonical source-ID table for binary snapshot transport.
func BuildBinarySourceIDTable(parseSourceIDs []string) ([]byte, error) {
	parseNormalizedSourceIDs, parseErr := NormalizeSourceIDs(parseSourceIDs)
	if parseErr != nil {
		return nil, parseErr
	}
	return buildBinarySourceIDTableFromNormalized(parseNormalizedSourceIDs)
}

// buildBinarySourceIDTableFromNormalized encodes one already-normalized source-ID list into the binary table format.
func buildBinarySourceIDTableFromNormalized(parseNormalizedSourceIDs []string) ([]byte, error) {
	if len(parseNormalizedSourceIDs) > 0xFFFF {
		return nil, fmt.Errorf("runtime2: source ID table length %d exceeds uint16", len(parseNormalizedSourceIDs))
	}
	parsePayload := make([]byte, 0, 2+len(parseNormalizedSourceIDs)*4)
	parsePayload = appendBinaryUint16(parsePayload, uint16(len(parseNormalizedSourceIDs)))
	for _, parseSourceID := range parseNormalizedSourceIDs {
		if len(parseSourceID) > 0xFFFF {
			return nil, fmt.Errorf("runtime2: source ID %q is too large for binary table", parseSourceID)
		}
		parsePayload = appendBinaryUint16(parsePayload, uint16(len(parseSourceID)))
		parsePayload = append(parsePayload, parseSourceID...)
	}
	return parsePayload, nil
}

// ParseBinarySourceIDTable decodes and validates one canonical source-ID table.
func ParseBinarySourceIDTable(parsePayload []byte) ([]string, error) {
	parseCount, parseOffset, parseErr := parseBinaryUint16(parsePayload, 0, "source-id-table", "count")
	if parseErr != nil {
		return nil, parseErr
	}
	parseSourceIDs := make([]string, 0, int(parseCount))
	for parseIndex := 0; parseIndex < int(parseCount); parseIndex++ {
		parseSourceID, parseNextOffset, parseErr := parseBinaryString(parsePayload, parseOffset, "source-id-table", fmt.Sprintf("source_id[%d]", parseIndex))
		if parseErr != nil {
			return nil, parseErr
		}
		parseSourceIDs = append(parseSourceIDs, parseSourceID)
		parseOffset = parseNextOffset
	}
	if parseOffset != len(parsePayload) {
		return nil, fmt.Errorf("runtime2: source-id-table has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	parseNormalizedSourceIDs, parseErr := NormalizeSourceIDs(parseSourceIDs)
	if parseErr != nil {
		return nil, parseErr
	}
	if len(parseNormalizedSourceIDs) != len(parseSourceIDs) {
		return nil, fmt.Errorf("runtime2: source-id-table contains duplicates")
	}
	for parseIndex, parseSourceID := range parseSourceIDs {
		if parseNormalizedSourceIDs[parseIndex] != parseSourceID {
			return nil, fmt.Errorf("runtime2: source-id-table is not canonical at index %d", parseIndex)
		}
	}
	return parseSourceIDs, nil
}
