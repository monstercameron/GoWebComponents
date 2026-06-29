package runtime2

import "testing"

// TestParseBinaryPayloadSpanValidatesBoundsAndReturnsSpan verifies payload span parsing rejects invalid offsets and lengths before returning one bounded slice.
func TestParseBinaryPayloadSpanValidatesBoundsAndReturnsSpan(parseT *testing.T) {
	parsePayload := []byte("abcdef")
	parseSpan, parseNextOffset, parseErr := parseBinaryPayloadSpan(parsePayload, 2, 3, "payload", "field")
	if parseErr != nil {
		parseT.Fatalf("parseBinaryPayloadSpan(valid) returned error: %v", parseErr)
	}
	if string(parseSpan) != "cde" || parseNextOffset != 5 {
		parseT.Fatalf("parseBinaryPayloadSpan(valid) = (%q, %d), want (%q, %d)", string(parseSpan), parseNextOffset, "cde", 5)
	}

	if _, _, parseErr := parseBinaryPayloadSpan(parsePayload, -1, 1, "payload", "field"); parseErr == nil {
		parseT.Fatal("expected negative offset to fail")
	}
	if _, _, parseErr := parseBinaryPayloadSpan(parsePayload, 0, -1, "payload", "field"); parseErr == nil {
		parseT.Fatal("expected negative length to fail")
	}
	if _, _, parseErr := parseBinaryPayloadSpan(parsePayload, 5, 2, "payload", "field"); parseErr == nil {
		parseT.Fatal("expected overflowing payload span to fail")
	}
}

// TestParseBinaryScalarAndStringHelpersDecodeAndRejectTruncation verifies scalar and string offset helpers decode sequential fields and reject truncated payloads.
func TestParseBinaryScalarAndStringHelpersDecodeAndRejectTruncation(parseT *testing.T) {
	parsePayload := appendBinaryUint16(nil, 0x1234)
	parsePayload = appendBinaryUint32(parsePayload, 0x05060708)
	parsePayload = appendBinaryUint64(parsePayload, 0x090a0b0c0d0e0f10)
	parsePayload = appendBinaryUint16(parsePayload, uint16(len("go")))
	parsePayload = append(parsePayload, []byte("go")...)

	parseUint16Value, parseOffset, parseErr := parseBinaryUint16(parsePayload, 0, "payload", "uint16")
	if parseErr != nil {
		parseT.Fatalf("parseBinaryUint16(valid) returned error: %v", parseErr)
	}
	if parseUint16Value != 0x1234 || parseOffset != 2 {
		parseT.Fatalf("parseBinaryUint16(valid) = (%#x, %d), want (%#x, %d)", parseUint16Value, parseOffset, uint16(0x1234), 2)
	}

	parseUint32Value, parseOffset, parseErr := parseBinaryUint32(parsePayload, parseOffset, "payload", "uint32")
	if parseErr != nil {
		parseT.Fatalf("parseBinaryUint32(valid) returned error: %v", parseErr)
	}
	if parseUint32Value != 0x05060708 || parseOffset != 6 {
		parseT.Fatalf("parseBinaryUint32(valid) = (%#x, %d), want (%#x, %d)", parseUint32Value, parseOffset, uint32(0x05060708), 6)
	}

	parseUint64Value, parseOffset, parseErr := parseBinaryUint64(parsePayload, parseOffset, "payload", "uint64")
	if parseErr != nil {
		parseT.Fatalf("parseBinaryUint64(valid) returned error: %v", parseErr)
	}
	if parseUint64Value != 0x090a0b0c0d0e0f10 || parseOffset != 14 {
		parseT.Fatalf("parseBinaryUint64(valid) = (%#x, %d), want (%#x, %d)", parseUint64Value, parseOffset, uint64(0x090a0b0c0d0e0f10), 14)
	}

	parseStringValue, parseOffset, parseErr := parseBinaryString(parsePayload, parseOffset, "payload", "text")
	if parseErr != nil {
		parseT.Fatalf("parseBinaryString(valid) returned error: %v", parseErr)
	}
	if parseStringValue != "go" || parseOffset != len(parsePayload) {
		parseT.Fatalf("parseBinaryString(valid) = (%q, %d), want (%q, %d)", parseStringValue, parseOffset, "go", len(parsePayload))
	}

	if _, _, parseErr := parseBinaryUint16([]byte{1}, 0, "payload", "uint16"); parseErr == nil {
		parseT.Fatal("expected truncated uint16 payload to fail")
	}
	if _, _, parseErr := parseBinaryUint32([]byte{1, 2, 3}, 0, "payload", "uint32"); parseErr == nil {
		parseT.Fatal("expected truncated uint32 payload to fail")
	}
	if _, _, parseErr := parseBinaryUint64([]byte{1, 2, 3, 4, 5, 6, 7}, 0, "payload", "uint64"); parseErr == nil {
		parseT.Fatal("expected truncated uint64 payload to fail")
	}
	if _, _, parseErr := parseBinaryLengthPrefixedSpan([]byte{1}, 0, "payload", "span"); parseErr == nil {
		parseT.Fatal("expected truncated length-prefixed span to fail")
	}
	if _, _, parseErr := parseBinaryString([]byte{3, 0, 'g', 'o'}, 0, "payload", "text"); parseErr == nil {
		parseT.Fatal("expected truncated length-prefixed string to fail")
	}
}
