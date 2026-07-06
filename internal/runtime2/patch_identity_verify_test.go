package runtime2

import "testing"

// TestVerifyPatchStreamIdentityAcceptsCanonicalRoundTrip pins that a canonical
// patch stream (identity computed from its parts) survives a build -> binary
// encode -> decode round trip and still verifies — the recompute-from-decoded-parts
// deterministically reproduces the carried identity, so the defense-in-depth check
// does not false-reject a valid patch.
func TestVerifyPatchStreamIdentityAcceptsCanonicalRoundTrip(parseT *testing.T) {
	parsePatchStream := parseBuildPatchStreamForBinaryPayloadTest(parseT)
	if parsePatchStream.GetPatchIdentity == "" {
		parseT.Fatal("expected a canonical patch stream to carry a computed identity")
	}
	parsePayload, parseErr := BuildBinaryPatchPayload(parsePatchStream)
	if parseErr != nil {
		parseT.Fatalf("BuildBinaryPatchPayload: %v", parseErr)
	}
	parseDecoded, parseDecodeErr := ParseBinaryPatchPayload(parsePayload)
	if parseDecodeErr != nil {
		parseT.Fatalf("ParseBinaryPatchPayload: %v", parseDecodeErr)
	}
	if parseVerifyErr := VerifyPatchStreamIdentity(parseDecoded); parseVerifyErr != nil {
		parseT.Fatalf("canonical decoded patch must verify, got: %v", parseVerifyErr)
	}
}

// TestVerifyPatchStreamIdentityRejectsTamperedBody pins that a body mutated after
// the identity was computed (carried identity left intact) is caught — the exact
// corruption/tamper scenario the check defends against across the worker bridge.
func TestVerifyPatchStreamIdentityRejectsTamperedBody(parseT *testing.T) {
	parseDecoded := parseBuildPatchStreamForBinaryPayloadTest(parseT)
	if len(parseDecoded.GetStringTable) == 0 && len(parseDecoded.GetOps) == 0 {
		parseT.Skip("need a stream with body content to tamper")
	}
	// Tamper the string table (or ops) while leaving the carried identity intact.
	if len(parseDecoded.GetStringTable) > 0 {
		parseDecoded.GetStringTable = append(append([]string(nil), parseDecoded.GetStringTable...), "\x00tampered\x00")
	} else {
		parseDecoded.GetOps = append(append([]PatchStreamOpRaw(nil), parseDecoded.GetOps...), parseDecoded.GetOps[0])
	}
	if parseVerifyErr := VerifyPatchStreamIdentity(parseDecoded); parseVerifyErr == nil {
		parseT.Fatal("expected a tampered patch body to fail identity verification")
	}
}

// TestVerifyPatchStreamIdentityAcceptsEmptyIdentity pins that a stream carrying no
// identity is accepted (nothing to verify against) rather than rejected.
func TestVerifyPatchStreamIdentityAcceptsEmptyIdentity(parseT *testing.T) {
	if parseErr := VerifyPatchStreamIdentity(PatchStreamRaw{}); parseErr != nil {
		parseT.Fatalf("a stream with no carried identity must be accepted, got: %v", parseErr)
	}
}
