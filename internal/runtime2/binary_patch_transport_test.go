package runtime2

import "testing"

// parseBuildPatchStreamForBinaryPayloadTest builds one non-no-op patch stream used by binary payload transport tests.
func parseBuildPatchStreamForBinaryPayloadTest(parseT *testing.T) PatchStreamRaw {
	parseT.Helper()
	parsePreviousIR, parsePreviousIRErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "text",
		"text": "before",
	})
	if parsePreviousIRErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR(previous) returned error: %v", parsePreviousIRErr)
	}
	parseNextIR, parseNextIRErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "text",
		"text": "after",
	})
	if parseNextIRErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR(next) returned error: %v", parseNextIRErr)
	}
	parsePatchStream, parseHasNoOp, parsePatchErr := BuildCanonicalPatchStream("region-1", 1, 2, 2, parsePreviousIR, parseNextIR)
	if parsePatchErr != nil {
		parseT.Fatalf("BuildCanonicalPatchStream returned error: %v", parsePatchErr)
	}
	if parseHasNoOp {
		parseT.Fatal("expected non-no-op patch stream for binary payload test")
	}
	return parsePatchStream
}

// TestBuildBinaryPatchPayloadRoundTrips verifies binary patch payload encoding and decoding round-trip one patch stream.
func TestBuildBinaryPatchPayloadRoundTrips(parseT *testing.T) {
	parsePatchStream := parseBuildPatchStreamForBinaryPayloadTest(parseT)
	parsePayload, parsePayloadErr := BuildBinaryPatchPayload(parsePatchStream)
	if parsePayloadErr != nil {
		parseT.Fatalf("BuildBinaryPatchPayload returned error: %v", parsePayloadErr)
	}
	parseDecodedPatchStream, parseDecodedPatchStreamErr := ParseBinaryPatchPayload(parsePayload)
	if parseDecodedPatchStreamErr != nil {
		parseT.Fatalf("ParseBinaryPatchPayload returned error: %v", parseDecodedPatchStreamErr)
	}
	if parseDecodedPatchStream.GetHeader.RegionID != parsePatchStream.GetHeader.RegionID {
		parseT.Fatalf("expected decoded region ID %q, got %q", parsePatchStream.GetHeader.RegionID, parseDecodedPatchStream.GetHeader.RegionID)
	}
	if parseDecodedPatchStream.GetHeader.PatchVersion != parsePatchStream.GetHeader.PatchVersion {
		parseT.Fatalf("expected decoded patch version %d, got %d", parsePatchStream.GetHeader.PatchVersion, parseDecodedPatchStream.GetHeader.PatchVersion)
	}
}

// TestParseBinaryPatchPayloadRejectsMalformedFrame verifies malformed binary patch framing is rejected.
func TestParseBinaryPatchPayloadRejectsMalformedFrame(parseT *testing.T) {
	parsePayload := []byte("BAD!")
	if _, parseErr := ParseBinaryPatchPayload(parsePayload); parseErr == nil {
		parseT.Fatal("expected malformed binary patch frame to fail")
	}
}
