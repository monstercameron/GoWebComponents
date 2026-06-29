package interop

import "testing"

// TestEnvelopeRoundTrip verifies that encodeEnvelope and decodeEnvelope are
// exact inverses for a range of IV and ciphertext lengths.
func TestEnvelopeRoundTrip(parseT *testing.T) {
	parseCases := []struct {
		parseIV []byte
		parseCT []byte
	}{
		{parseIV: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}, parseCT: []byte("hello encrypted world")},
		{parseIV: make([]byte, 12), parseCT: []byte{0, 255, 128, 64}},
		{parseIV: []byte{0xde, 0xad, 0xbe, 0xef, 0xca, 0xfe, 0xba, 0xbe, 0x01, 0x23, 0x45, 0x67}, parseCT: []byte{}},
	}
	for _, parseCase := range parseCases {
		parseEnv := encodeEnvelope(parseCase.parseIV, parseCase.parseCT)
		parseGotIV, parseGotCT, parseErr := decodeEnvelope(parseEnv)
		if parseErr != nil {
			parseT.Fatalf("decodeEnvelope(%q) returned unexpected error: %v", parseEnv, parseErr)
		}
		if len(parseGotIV) != len(parseCase.parseIV) {
			parseT.Fatalf("IV length mismatch: got %d want %d", len(parseGotIV), len(parseCase.parseIV))
		}
		for parseI, parseByte := range parseCase.parseIV {
			if parseGotIV[parseI] != parseByte {
				parseT.Fatalf("IV byte %d mismatch: got %d want %d", parseI, parseGotIV[parseI], parseByte)
			}
		}
		if len(parseGotCT) != len(parseCase.parseCT) {
			parseT.Fatalf("ciphertext length mismatch: got %d want %d", len(parseGotCT), len(parseCase.parseCT))
		}
		for parseI, parseByte := range parseCase.parseCT {
			if parseGotCT[parseI] != parseByte {
				parseT.Fatalf("ciphertext byte %d mismatch: got %d want %d", parseI, parseGotCT[parseI], parseByte)
			}
		}
	}
}

// TestDecodeEnvelopeMissingSeparatorErrors verifies that a corrupted envelope
// with no colon separator returns a structured decode error, not silent garbage.
func TestDecodeEnvelopeMissingSeparatorErrors(parseT *testing.T) {
	_, _, parseErr := decodeEnvelope("nodivider")
	if parseErr == nil {
		parseT.Fatal("expected error for envelope missing ':' separator, got nil")
	}
	if !IsCode(parseErr, CodeDecode) {
		parseT.Fatalf("expected CodeDecode error for missing separator, got %v", parseErr)
	}
}

// TestDecodeEnvelopeCorruptedBase64Errors verifies that an envelope with
// invalid base64 in either segment returns a structured decode error.
func TestDecodeEnvelopeCorruptedBase64Errors(parseT *testing.T) {
	parseCases := []string{
		"!!!invalid!!!:AAAA",
		"AAAA:!!!invalid!!!",
	}
	for _, parseEnv := range parseCases {
		_, _, parseErr := decodeEnvelope(parseEnv)
		if parseErr == nil {
			parseT.Fatalf("expected decode error for corrupted envelope %q, got nil", parseEnv)
		}
		if !IsCode(parseErr, CodeDecode) {
			parseT.Fatalf("expected CodeDecode for corrupted envelope %q, got %v", parseEnv, parseErr)
		}
	}
}

// TestDecodeEnvelopeEmptySegmentsAreValid verifies that an envelope with empty
// IV or empty ciphertext decodes without error (both are base64-representable).
func TestDecodeEnvelopeEmptySegmentsAreValid(parseT *testing.T) {
	parseEnv := encodeEnvelope([]byte{}, []byte{})
	parseIV, parseCT, parseErr := decodeEnvelope(parseEnv)
	if parseErr != nil {
		parseT.Fatalf("empty segment envelope returned unexpected error: %v", parseErr)
	}
	if len(parseIV) != 0 || len(parseCT) != 0 {
		parseT.Fatalf("expected empty IV and CT, got iv=%v ct=%v", parseIV, parseCT)
	}
}
