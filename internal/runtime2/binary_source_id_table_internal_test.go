package runtime2

import "testing"

// TestAppendBinarySourceIDTableFromNormalizedExpandsExistingPayload verifies appending a normalized source-ID table grows the destination payload when capacity is insufficient and preserves the existing prefix bytes.
func TestAppendBinarySourceIDTableFromNormalizedExpandsExistingPayload(parseT *testing.T) {
	parsePayload := []byte{0xAB}
	parseAppendedPayload, parseErr := appendBinarySourceIDTableFromNormalized(parsePayload, []string{"alpha", "beta"})
	if parseErr != nil {
		parseT.Fatalf("appendBinarySourceIDTableFromNormalized returned error: %v", parseErr)
	}
	if parseAppendedPayload[0] != 0xAB {
		parseT.Fatalf("appendBinarySourceIDTableFromNormalized prefix byte = 0x%X, want 0xAB", parseAppendedPayload[0])
	}
	parseSourceIDs, parseParseErr := ParseBinarySourceIDTable(parseAppendedPayload[1:])
	if parseParseErr != nil {
		parseT.Fatalf("ParseBinarySourceIDTable(appended payload) returned error: %v", parseParseErr)
	}
	if len(parseSourceIDs) != 2 || parseSourceIDs[0] != "alpha" || parseSourceIDs[1] != "beta" {
		parseT.Fatalf("ParseBinarySourceIDTable(appended payload) source IDs = %#v, want [alpha beta]", parseSourceIDs)
	}
}

// TestParseBinarySourceIDTableRejectsTruncatedAndUnsupportedPayload verifies binary source-ID decoding rejects truncated payloads and unsupported characters.
func TestParseBinarySourceIDTableRejectsTruncatedAndUnsupportedPayload(parseT *testing.T) {
	if _, parseErr := ParseBinarySourceIDTable([]byte{1}); parseErr == nil {
		parseT.Fatal("expected truncated source-id-table count to fail")
	}
	if _, parseErr := ParseBinarySourceIDTable([]byte{1, 0, 1, 0, '!'}); parseErr == nil {
		parseT.Fatal("expected unsupported source-id-table character to fail")
	}
}
