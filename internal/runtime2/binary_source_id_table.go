package runtime2

import (
	"encoding/binary"
	"fmt"
	"strings"
)

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
	parseTableLength, parseTableLengthErr := getBinarySourceIDTableLengthFromNormalized(parseNormalizedSourceIDs)
	if parseTableLengthErr != nil {
		return nil, parseTableLengthErr
	}
	parsePayload := make([]byte, 0, parseTableLength)
	return appendBinarySourceIDTableFromNormalizedTrusted(parsePayload, parseNormalizedSourceIDs), nil
}

// getBinarySourceIDTableLengthFromNormalized returns the exact encoded length for one normalized source-id table.
func getBinarySourceIDTableLengthFromNormalized(parseNormalizedSourceIDs []string) (int, error) {
	if len(parseNormalizedSourceIDs) > 0xFFFF {
		return 0, fmt.Errorf("runtime2: source ID table length %d exceeds uint16", len(parseNormalizedSourceIDs))
	}
	parseLength := 2
	for _, parseSourceID := range parseNormalizedSourceIDs {
		if len(parseSourceID) > 0xFFFF {
			return 0, fmt.Errorf("runtime2: source ID %q is too large for binary table", parseSourceID)
		}
		parseLength += 2 + len(parseSourceID)
	}
	return parseLength, nil
}

// appendBinarySourceIDTableFromNormalized appends one already-normalized source-ID list into the binary table format.
func appendBinarySourceIDTableFromNormalized(parsePayload []byte, parseNormalizedSourceIDs []string) ([]byte, error) {
	parseTableLength, parseTableLengthErr := getBinarySourceIDTableLengthFromNormalized(parseNormalizedSourceIDs)
	if parseTableLengthErr != nil {
		return nil, parseTableLengthErr
	}
	parsePayload = getBinarySourceIDTablePayloadWithCapacity(parsePayload, parseTableLength)
	return appendBinarySourceIDTableFromNormalizedTrusted(parsePayload, parseNormalizedSourceIDs), nil
}

// appendBinarySourceIDTableFromNormalizedTrusted appends one validated canonical source-ID table into the binary payload.
func appendBinarySourceIDTableFromNormalizedTrusted(parsePayload []byte, parseNormalizedSourceIDs []string) []byte {
	parsePayload = appendBinaryUint16(parsePayload, uint16(len(parseNormalizedSourceIDs)))
	for _, parseSourceID := range parseNormalizedSourceIDs {
		parsePayload = appendBinaryUint16(parsePayload, uint16(len(parseSourceID)))
		parsePayload = append(parsePayload, parseSourceID...)
	}
	return parsePayload
}

// getBinarySourceIDTablePayloadWithCapacity ensures one payload has room for the additional source-ID table bytes.
func getBinarySourceIDTablePayloadWithCapacity(parsePayload []byte, parseAdditionalLength int) []byte {
	parseRequiredCapacity := len(parsePayload) + parseAdditionalLength
	if cap(parsePayload) >= parseRequiredCapacity {
		return parsePayload
	}
	parseExpanded := make([]byte, len(parsePayload), parseRequiredCapacity)
	copy(parseExpanded, parsePayload)
	return parseExpanded
}

// ParseBinarySourceIDTable decodes and validates one canonical source-ID table.
func ParseBinarySourceIDTable(parsePayload []byte) ([]string, error) {
	if len(parsePayload) < 2 {
		return nil, fmt.Errorf("runtime2: binary source-id-table count is truncated")
	}
	parseCount := int(binary.LittleEndian.Uint16(parsePayload[0:2]))
	return parseBinarySourceIDTableInto(make([]string, 0, parseCount), parsePayload)
}

// parseBinarySourceIDTableInto decodes one canonical source-ID table appending results into parseDst and returns the extended slice.
func parseBinarySourceIDTableInto(parseDst []string, parsePayload []byte) ([]string, error) {
	if len(parsePayload) < 2 {
		return parseDst, fmt.Errorf("runtime2: binary source-id-table count is truncated")
	}
	parseCount := int(binary.LittleEndian.Uint16(parsePayload[0:2]))
	parseOffset := 2
	parsePreviousSourceID := ""
	parseBaseLength := len(parseDst)
	parseRequiredLength := parseBaseLength + parseCount
	if cap(parseDst) < parseRequiredLength {
		parseExpanded := make([]string, parseRequiredLength)
		copy(parseExpanded, parseDst)
		parseDst = parseExpanded
	} else {
		parseDst = parseDst[:parseRequiredLength]
	}
	parseDecodedCount := 0
	for parseIndex := 0; parseIndex < parseCount; parseIndex++ {
		if parseOffset+2 > len(parsePayload) {
			return parseDst[:parseBaseLength+parseDecodedCount], fmt.Errorf("runtime2: decode source_id[%d]: length is truncated", parseIndex)
		}
		parseSourceIDLength := int(binary.LittleEndian.Uint16(parsePayload[parseOffset : parseOffset+2]))
		parseOffset += 2
		if parseSourceIDLength > len(parsePayload)-parseOffset {
			return parseDst[:parseBaseLength+parseDecodedCount], fmt.Errorf(
				"runtime2: decode source_id[%d]: length %d exceeds payload size %d",
				parseIndex,
				parseSourceIDLength,
				len(parsePayload)-parseOffset,
			)
		}
		parseSourceID := string(parsePayload[parseOffset : parseOffset+parseSourceIDLength])
		parseOffset += parseSourceIDLength
		if parseSourceID == "" {
			return parseDst[:parseBaseLength+parseDecodedCount], fmt.Errorf("runtime2: source ID is required")
		}
		if hasBinarySourceIDSurroundingWhitespace(parseSourceID) {
			return parseDst[:parseBaseLength+parseDecodedCount], fmt.Errorf("runtime2: source ID %q must not contain surrounding whitespace", parseSourceID)
		}
		parseUnsupportedRune, hasUnsupportedRune := getBinarySourceIDUnsupportedRune(parseSourceID)
		if hasUnsupportedRune {
			return parseDst[:parseBaseLength+parseDecodedCount], fmt.Errorf("runtime2: source ID %q contains unsupported character %q", parseSourceID, string(parseUnsupportedRune))
		}
		if parseIndex > 0 && parseSourceID <= parsePreviousSourceID {
			if parseSourceID == parsePreviousSourceID {
				return parseDst[:parseBaseLength+parseDecodedCount], fmt.Errorf("runtime2: source-id-table contains duplicates")
			}
			return parseDst[:parseBaseLength+parseDecodedCount], fmt.Errorf("runtime2: source-id-table is not canonical at index %d", parseIndex)
		}
		parseDst[parseBaseLength+parseIndex] = parseSourceID
		parsePreviousSourceID = parseSourceID
		parseDecodedCount++
	}
	if parseOffset != len(parsePayload) {
		return parseDst[:parseBaseLength+parseDecodedCount], fmt.Errorf("runtime2: source-id-table has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	return parseDst, nil
}

// getBinarySourceIDUnsupportedRune returns the first source-ID rune outside the supported runtime2 identifier contract.
func getBinarySourceIDUnsupportedRune(parseSourceID string) (rune, bool) {
	for _, parseRune := range parseSourceID {
		parseAllowed := parseRune == '.' || parseRune == '-' || parseRune == '_' || parseRune == ':'
		if parseAllowed || (parseRune >= 'a' && parseRune <= 'z') || (parseRune >= 'A' && parseRune <= 'Z') || (parseRune >= '0' && parseRune <= '9') {
			continue
		}
		return parseRune, true
	}
	return 0, false
}
