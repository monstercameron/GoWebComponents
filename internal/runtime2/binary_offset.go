package runtime2

import (
	"encoding/binary"
	"fmt"
)

// parseBinaryPayloadSpan returns one bounded payload span or a detailed offset error.
func parseBinaryPayloadSpan(parsePayload []byte, parseOffset int, parseLength int, parseSection string, parseField string) ([]byte, int, error) {
	if parseOffset < 0 {
		return nil, 0, fmt.Errorf("runtime2: binary %s %s offset %d is invalid", parseSection, parseField, parseOffset)
	}
	if parseLength < 0 {
		return nil, 0, fmt.Errorf("runtime2: binary %s %s length %d is invalid", parseSection, parseField, parseLength)
	}
	if parseOffset > len(parsePayload) || parseLength > len(parsePayload)-parseOffset {
		return nil, 0, fmt.Errorf(
			"runtime2: binary %s %s offset=%d length=%d exceeds payload size %d",
			parseSection,
			parseField,
			parseOffset,
			parseLength,
			len(parsePayload),
		)
	}
	parseEnd := parseOffset + parseLength
	return parsePayload[parseOffset:parseEnd], parseEnd, nil
}

// parseBinaryUint16 decodes one uint16 field at the given offset.
func parseBinaryUint16(parsePayload []byte, parseOffset int, parseSection string, parseField string) (uint16, int, error) {
	parseSpan, parseNextOffset, parseErr := parseBinaryPayloadSpan(parsePayload, parseOffset, 2, parseSection, parseField)
	if parseErr != nil {
		return 0, 0, parseErr
	}
	return binary.LittleEndian.Uint16(parseSpan), parseNextOffset, nil
}

// parseBinaryUint32 decodes one uint32 field at the given offset.
func parseBinaryUint32(parsePayload []byte, parseOffset int, parseSection string, parseField string) (uint32, int, error) {
	parseSpan, parseNextOffset, parseErr := parseBinaryPayloadSpan(parsePayload, parseOffset, 4, parseSection, parseField)
	if parseErr != nil {
		return 0, 0, parseErr
	}
	return binary.LittleEndian.Uint32(parseSpan), parseNextOffset, nil
}

// parseBinaryUint64 decodes one uint64 field at the given offset.
func parseBinaryUint64(parsePayload []byte, parseOffset int, parseSection string, parseField string) (uint64, int, error) {
	parseSpan, parseNextOffset, parseErr := parseBinaryPayloadSpan(parsePayload, parseOffset, 8, parseSection, parseField)
	if parseErr != nil {
		return 0, 0, parseErr
	}
	return binary.LittleEndian.Uint64(parseSpan), parseNextOffset, nil
}

// parseBinaryLengthPrefixedSpan decodes one uint16-length-prefixed payload span.
func parseBinaryLengthPrefixedSpan(parsePayload []byte, parseOffset int, parseSection string, parseField string) ([]byte, int, error) {
	parseLength, parseNextOffset, parseErr := parseBinaryUint16(parsePayload, parseOffset, parseSection, parseField)
	if parseErr != nil {
		return nil, 0, parseErr
	}
	return parseBinaryPayloadSpan(parsePayload, parseNextOffset, int(parseLength), parseSection, parseField)
}

// parseBinaryString decodes one uint16-length-prefixed string field.
func parseBinaryString(parsePayload []byte, parseOffset int, parseSection string, parseField string) (string, int, error) {
	parseSpan, parseEndOffset, parseErr := parseBinaryLengthPrefixedSpan(parsePayload, parseOffset, parseSection, parseField)
	if parseErr != nil {
		return "", 0, parseErr
	}
	return string(parseSpan), parseEndOffset, nil
}
