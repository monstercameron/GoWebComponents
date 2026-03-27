package runtime2

import "testing"

func buildStructuredClonePatchEnvelopeForHostParseTest(parseT *testing.T) (PatchStreamRaw, StructuredClonePatchEnvelope, []byte) {
	parseT.Helper()
	parsePatchStream := parseBuildPatchStreamForBinaryPayloadTest(parseT)
	parseRawPatchPayload, parseRawPatchPayloadErr := BuildStructuredClonePatchPayloadJSON(parsePatchStream)
	if parseRawPatchPayloadErr != nil {
		parseT.Fatalf("BuildStructuredClonePatchPayloadJSON returned error: %v", parseRawPatchPayloadErr)
	}
	parseStructuredEnvelope := StructuredClonePatchEnvelope{
		RegionInstanceID: RegionInstanceID(parsePatchStream.GetHeader.RegionID),
		Epoch:            parsePatchStream.GetHeader.Epoch,
		PatchVersion:     parsePatchStream.GetHeader.PatchVersion,
		PatchPayload:     parseRawPatchPayload,
	}
	parseStructuredEnvelopePayload, parseStructuredEnvelopePayloadErr := BuildStructuredClonePatchEnvelopeJSON(parseStructuredEnvelope)
	if parseStructuredEnvelopePayloadErr != nil {
		parseT.Fatalf("BuildStructuredClonePatchEnvelopeJSON returned error: %v", parseStructuredEnvelopePayloadErr)
	}
	return parsePatchStream, parseStructuredEnvelope, parseStructuredEnvelopePayload
}

// TestParseHostPatchPayloadWithFallbackStructuredEnvelope verifies host patch parsing accepts structured-clone patch envelope payloads.
func TestParseHostPatchPayloadWithFallbackStructuredEnvelope(parseT *testing.T) {
	parsePatchStream, _, parseStructuredEnvelopePayload := buildStructuredClonePatchEnvelopeForHostParseTest(parseT)
	parsePatchReadyEnvelope, parsePatchReadyEnvelopeErr := BuildControlPatchReadyEnvelope(
		RegionInstanceID(parsePatchStream.GetHeader.RegionID),
		parsePatchStream.GetHeader.PatchVersion,
		parsePatchStream.GetHeader.InputVersion,
		TransportTierStructuredClone,
	)
	if parsePatchReadyEnvelopeErr != nil {
		parseT.Fatalf("BuildControlPatchReadyEnvelope returned error: %v", parsePatchReadyEnvelopeErr)
	}
	parseTransportTier, parseDecodedPatchStream, parseDecodedPatchStreamErr := ParseHostPatchPayloadWithFallback(
		parsePatchReadyEnvelope,
		parseStructuredEnvelopePayload,
		nil,
	)
	if parseDecodedPatchStreamErr != nil {
		parseT.Fatalf("ParseHostPatchPayloadWithFallback returned error: %v", parseDecodedPatchStreamErr)
	}
	if parseTransportTier != TransportTierStructuredClone {
		parseT.Fatalf("expected structured-clone tier, got %q", parseTransportTier)
	}
	if parseDecodedPatchStream.GetHeader.PatchVersion != parsePatchStream.GetHeader.PatchVersion {
		parseT.Fatalf("expected patch version %d, got %d", parsePatchStream.GetHeader.PatchVersion, parseDecodedPatchStream.GetHeader.PatchVersion)
	}
}

// TestParseHostPatchPayloadWithFallbackBinaryTierFallsBackToStructured verifies binary-tier parse falls back to structured-clone payload parsing when binary decode fails.
func TestParseHostPatchPayloadWithFallbackBinaryTierFallsBackToStructured(parseT *testing.T) {
	parsePatchStream, _, parseStructuredEnvelopePayload := buildStructuredClonePatchEnvelopeForHostParseTest(parseT)
	parsePatchReadyEnvelope, parsePatchReadyEnvelopeErr := BuildControlPatchReadyEnvelope(
		RegionInstanceID(parsePatchStream.GetHeader.RegionID),
		parsePatchStream.GetHeader.PatchVersion,
		parsePatchStream.GetHeader.InputVersion,
		TransportTierBinary,
	)
	if parsePatchReadyEnvelopeErr != nil {
		parseT.Fatalf("BuildControlPatchReadyEnvelope returned error: %v", parsePatchReadyEnvelopeErr)
	}
	parseTransportTier, _, parseDecodedPatchStreamErr := ParseHostPatchPayloadWithFallback(
		parsePatchReadyEnvelope,
		parseStructuredEnvelopePayload,
		nil,
	)
	if parseDecodedPatchStreamErr != nil {
		parseT.Fatalf("ParseHostPatchPayloadWithFallback returned error: %v", parseDecodedPatchStreamErr)
	}
	if parseTransportTier != TransportTierStructuredClone {
		parseT.Fatalf("expected structured-clone fallback tier, got %q", parseTransportTier)
	}
}

// TestParseHostPatchPayloadWithFallbackSharedTier verifies shared-buffer tier parse reads patch payload from shared patch page.
func TestParseHostPatchPayloadWithFallbackSharedTier(parseT *testing.T) {
	parsePatchStream, parseStructuredEnvelope, _ := buildStructuredClonePatchEnvelopeForHostParseTest(parseT)
	parseSharedPatchPage, parseSharedPatchPageErr := BuildSharedPatchPage(4096)
	if parseSharedPatchPageErr != nil {
		parseT.Fatalf("BuildSharedPatchPage returned error: %v", parseSharedPatchPageErr)
	}
	parseSharedTransportResult, parseSharedTransportResultErr := BuildSharedPatchTransportResult(
		parseStructuredEnvelope,
		buildSharedPatchCapabilityReportForTest(),
		parseSharedPatchPage,
	)
	if parseSharedTransportResultErr != nil {
		parseT.Fatalf("BuildSharedPatchTransportResult returned error: %v", parseSharedTransportResultErr)
	}
	if parseSharedTransportResult.GetTransportTier != TransportTierSharedBuffer {
		parseT.Fatalf("expected shared-buffer tier, got %q", parseSharedTransportResult.GetTransportTier)
	}
	parsePatchReadyEnvelope, parsePatchReadyEnvelopeErr := BuildControlPatchReadyEnvelope(
		RegionInstanceID(parsePatchStream.GetHeader.RegionID),
		parsePatchStream.GetHeader.PatchVersion,
		parsePatchStream.GetHeader.InputVersion,
		TransportTierSharedBuffer,
	)
	if parsePatchReadyEnvelopeErr != nil {
		parseT.Fatalf("BuildControlPatchReadyEnvelope returned error: %v", parsePatchReadyEnvelopeErr)
	}
	parseTransportTier, parseDecodedPatchStream, parseDecodedPatchStreamErr := ParseHostPatchPayloadWithFallback(
		parsePatchReadyEnvelope,
		parseSharedTransportResult.GetMessagePayload,
		parseSharedPatchPage,
	)
	if parseDecodedPatchStreamErr != nil {
		parseT.Fatalf("ParseHostPatchPayloadWithFallback returned error: %v", parseDecodedPatchStreamErr)
	}
	if parseTransportTier != TransportTierSharedBuffer {
		parseT.Fatalf("expected shared-buffer parse tier, got %q", parseTransportTier)
	}
	if parseDecodedPatchStream.GetHeader.RegionID != parsePatchStream.GetHeader.RegionID {
		parseT.Fatalf("expected decoded region %q, got %q", parsePatchStream.GetHeader.RegionID, parseDecodedPatchStream.GetHeader.RegionID)
	}
}

// TestParseHostPatchPayloadWithFallbackRejectsMalformedPayload verifies malformed patch payloads fail decode before host commit.
func TestParseHostPatchPayloadWithFallbackRejectsMalformedPayload(parseT *testing.T) {
	parsePatchReadyEnvelope, parsePatchReadyEnvelopeErr := BuildControlPatchReadyEnvelope(
		"region-1",
		2,
		2,
		TransportTierStructuredClone,
	)
	if parsePatchReadyEnvelopeErr != nil {
		parseT.Fatalf("BuildControlPatchReadyEnvelope returned error: %v", parsePatchReadyEnvelopeErr)
	}
	if _, _, parsePatchErr := ParseHostPatchPayloadWithFallback(parsePatchReadyEnvelope, []byte(`{"invalid"`), nil); parsePatchErr == nil {
		parseT.Fatal("expected malformed patch payload to fail parse before host commit")
	}
}

// TestParseHostPatchPayloadWithFallbackRejectsWrongRegionBeforeCommit verifies patch payload region mismatches fail before host commit.
func TestParseHostPatchPayloadWithFallbackRejectsWrongRegionBeforeCommit(parseT *testing.T) {
	parsePatchStream, _, parseStructuredEnvelopePayload := buildStructuredClonePatchEnvelopeForHostParseTest(parseT)
	parsePatchReadyEnvelope, parsePatchReadyEnvelopeErr := BuildControlPatchReadyEnvelope(
		"region-other",
		parsePatchStream.GetHeader.PatchVersion,
		parsePatchStream.GetHeader.InputVersion,
		TransportTierStructuredClone,
	)
	if parsePatchReadyEnvelopeErr != nil {
		parseT.Fatalf("BuildControlPatchReadyEnvelope returned error: %v", parsePatchReadyEnvelopeErr)
	}
	if _, _, parsePatchErr := ParseHostPatchPayloadWithFallback(parsePatchReadyEnvelope, parseStructuredEnvelopePayload, nil); parsePatchErr == nil {
		parseT.Fatal("expected wrong-region patch payload mismatch to fail parse before host commit")
	}
}

// TestParseHostPatchPayloadWithFallbackRejectsWrongVersionBeforeCommit verifies patch payload version mismatches fail before host commit.
func TestParseHostPatchPayloadWithFallbackRejectsWrongVersionBeforeCommit(parseT *testing.T) {
	parsePatchStream, _, parseStructuredEnvelopePayload := buildStructuredClonePatchEnvelopeForHostParseTest(parseT)
	parsePatchReadyEnvelope, parsePatchReadyEnvelopeErr := BuildControlPatchReadyEnvelope(
		RegionInstanceID(parsePatchStream.GetHeader.RegionID),
		parsePatchStream.GetHeader.PatchVersion+1,
		parsePatchStream.GetHeader.InputVersion,
		TransportTierStructuredClone,
	)
	if parsePatchReadyEnvelopeErr != nil {
		parseT.Fatalf("BuildControlPatchReadyEnvelope returned error: %v", parsePatchReadyEnvelopeErr)
	}
	if _, _, parsePatchErr := ParseHostPatchPayloadWithFallback(parsePatchReadyEnvelope, parseStructuredEnvelopePayload, nil); parsePatchErr == nil {
		parseT.Fatal("expected wrong-version patch payload mismatch to fail parse before host commit")
	}
}
