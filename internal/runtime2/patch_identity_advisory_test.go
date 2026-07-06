package runtime2

import "testing"

// withCapturedPatchIdentityMismatch swaps the advisory reporter for one that counts
// mismatches, restoring the default on cleanup.
func withCapturedPatchIdentityMismatch(parseT *testing.T) *int {
	parseT.Helper()
	parseCount := 0
	parseOld := reportPatchIdentityMismatch
	reportPatchIdentityMismatch = func(string, uint64, error) { parseCount++ }
	parseT.Cleanup(func() { reportPatchIdentityMismatch = parseOld })
	return &parseCount
}

// TestVerifyDecodedPatchIdentityReportsMismatchWithoutRejecting pins that the
// advisory check surfaces a corrupted/tampered patch (carried identity intact, body
// changed) but never rejects it — a canonical patch is silent.
func TestVerifyDecodedPatchIdentityReportsMismatchWithoutRejecting(parseT *testing.T) {
	parseCount := withCapturedPatchIdentityMismatch(parseT)

	parseCanonical := parseBuildPatchStreamForBinaryPayloadTest(parseT)
	if parseCanonical.GetPatchIdentity == "" {
		parseT.Fatal("expected a canonical (identity-bearing) patch for this test")
	}

	// Canonical patch: no advisory.
	verifyDecodedPatchIdentity(parseCanonical)
	if *parseCount != 0 {
		parseT.Fatalf("canonical patch must not trigger an advisory, got %d", *parseCount)
	}

	// Tamper the body while leaving the carried identity intact — the recompute
	// diverges and the advisory fires (but the patch is still returned upstream).
	parseTampered := parseCanonical
	parseTampered.GetStringTable = append(append([]string(nil), parseCanonical.GetStringTable...), "smuggled-string")
	verifyDecodedPatchIdentity(parseTampered)
	if *parseCount != 1 {
		parseT.Fatalf("tampered patch must trigger exactly one advisory, got %d", *parseCount)
	}
}

// TestHostBoundaryDoesNotFalseAdviseOnCanonicalPatch is the safety proof for #45:
// a canonical patch that round-trips through the real host/worker decode boundary
// (binary AND structured-clone tiers) must NOT produce a spurious identity advisory.
// This is the property that made hard-gating unsafe; the advisory must be quiet on
// legitimate traffic.
func TestHostBoundaryDoesNotFalseAdviseOnCanonicalPatch(parseT *testing.T) {
	// Binary tier.
	func() {
		parseCount := withCapturedPatchIdentityMismatch(parseT)
		parsePatchStream := parseBuildPatchStreamForBinaryPayloadTest(parseT)
		parsePayload, parseErr := BuildBinaryPatchPayload(parsePatchStream)
		if parseErr != nil {
			parseT.Fatalf("BuildBinaryPatchPayload: %v", parseErr)
		}
		parseEnvelope, parseEnvErr := BuildControlPatchReadyEnvelope(
			RegionInstanceID(parsePatchStream.GetHeader.RegionID),
			parsePatchStream.GetHeader.PatchVersion,
			parsePatchStream.GetHeader.InputVersion,
			TransportTierBinary,
		)
		if parseEnvErr != nil {
			parseT.Fatalf("BuildControlPatchReadyEnvelope: %v", parseEnvErr)
		}
		if _, _, parseDecErr := ParseHostPatchPayloadWithFallback(parseEnvelope, parsePayload, nil); parseDecErr != nil {
			parseT.Fatalf("ParseHostPatchPayloadWithFallback(binary): %v", parseDecErr)
		}
		if *parseCount != 0 {
			parseT.Fatalf("binary round-trip of a canonical patch must not false-advise, got %d", *parseCount)
		}
	}()

	// Structured-clone tier.
	func() {
		parseCount := withCapturedPatchIdentityMismatch(parseT)
		parsePatchStream, _, parseEnvelopePayload := buildStructuredClonePatchEnvelopeForHostParseTest(parseT)
		parseEnvelope, parseEnvErr := BuildControlPatchReadyEnvelope(
			RegionInstanceID(parsePatchStream.GetHeader.RegionID),
			parsePatchStream.GetHeader.PatchVersion,
			parsePatchStream.GetHeader.InputVersion,
			TransportTierStructuredClone,
		)
		if parseEnvErr != nil {
			parseT.Fatalf("BuildControlPatchReadyEnvelope: %v", parseEnvErr)
		}
		if _, _, parseDecErr := ParseHostPatchPayloadWithFallback(parseEnvelope, parseEnvelopePayload, nil); parseDecErr != nil {
			parseT.Fatalf("ParseHostPatchPayloadWithFallback(structured): %v", parseDecErr)
		}
		if *parseCount != 0 {
			parseT.Fatalf("structured-clone round-trip of a canonical patch must not false-advise, got %d", *parseCount)
		}
	}()
}

// TestVerifyDecodedPatchIdentitySilentOnEmptyIdentity pins that a patch carrying no
// identity (e.g. a fast-path header patch) is not flagged.
func TestVerifyDecodedPatchIdentitySilentOnEmptyIdentity(parseT *testing.T) {
	parseCount := withCapturedPatchIdentityMismatch(parseT)
	verifyDecodedPatchIdentity(PatchStreamRaw{})
	if *parseCount != 0 {
		parseT.Fatalf("empty-identity patch must not trigger an advisory, got %d", *parseCount)
	}
	// Sanity: the underlying verifier also treats empty identity as acceptable.
	if parseErr := VerifyPatchStreamIdentity(PatchStreamRaw{}); parseErr != nil {
		parseT.Fatalf("empty-identity verify should be nil, got %v", parseErr)
	}
}
