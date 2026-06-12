package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestSelectSnapshotTransportTierPrefersBinary verifies binary is selected when the capability report allows it.
func TestSelectSnapshotTransportTierPrefersBinary(parseT *testing.T) {
	parseTier, parseErr := runtime2.SelectSnapshotTransportTier(runtime2.CapabilityReport{
		HasWorkerSupport:          true,
		HasStructuredCloneSupport: true,
		HasBinaryTransportSupport: true,
	})
	if parseErr != nil {
		parseT.Fatalf("SelectSnapshotTransportTier returned error: %v", parseErr)
	}
	if parseTier != runtime2.TransportTierBinary {
		parseT.Fatalf("expected binary transport tier, got %q", parseTier)
	}
}

// TestBuildSnapshotTransportPayloadWithFallbackDowngradesOnBinaryEncodeFailure verifies binary encode failures fall back to structured-clone.
func TestBuildSnapshotTransportPayloadWithFallbackDowngradesOnBinaryEncodeFailure(parseT *testing.T) {
	parseProps := make(map[string]any, 65)
	for parseIndex := range 65 {
		parseProps[string(rune('a'+(parseIndex%26)))+string(rune('A'+(parseIndex/26)))] = parseIndex
	}
	parseTier, parsePayload, parseErr := runtime2.BuildSnapshotTransportPayloadWithFallback(runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     1,
		Props:            parseProps,
	}, runtime2.CapabilityReport{
		HasWorkerSupport:          true,
		HasStructuredCloneSupport: true,
		HasBinaryTransportSupport: true,
	})
	if parseErr != nil {
		parseT.Fatalf("BuildSnapshotTransportPayloadWithFallback returned error: %v", parseErr)
	}
	if parseTier != runtime2.TransportTierStructuredClone {
		parseT.Fatalf("expected structured-clone downgrade, got %q", parseTier)
	}
	parseDecoded, parseDecodedTier, parseErr := parseSnapshotTransportPayload(parseTier, parsePayload)
	if parseErr != nil {
		parseT.Fatalf("parseSnapshotTransportPayload returned error: %v", parseErr)
	}
	if parseDecodedTier != runtime2.TransportTierStructuredClone || parseDecoded.RegionInstanceID != runtime2.RegionInstanceID("region-1") {
		parseT.Fatalf("unexpected fallback decode result tier=%q envelope=%#v", parseDecodedTier, parseDecoded)
	}
}

// TestParseSnapshotTransportPayloadWithFallbackDowngradesOnBinaryDecodeFailure verifies binary decode failure can fall back to structured-clone bytes.
func TestParseSnapshotTransportPayloadWithFallbackDowngradesOnBinaryDecodeFailure(parseT *testing.T) {
	parsePayload, parseErr := runtime2.BuildStructuredCloneSnapshotEnvelopeJSON(runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     1,
	})
	if parseErr != nil {
		parseT.Fatalf("BuildStructuredCloneSnapshotEnvelopeJSON returned error: %v", parseErr)
	}
	parseTier, parseEnvelope, parseErr := runtime2.ParseSnapshotTransportPayloadWithFallback(runtime2.TransportTierBinary, parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseSnapshotTransportPayloadWithFallback returned error: %v", parseErr)
	}
	if parseTier != runtime2.TransportTierStructuredClone {
		parseT.Fatalf("expected structured-clone fallback tier, got %q", parseTier)
	}
	if parseEnvelope.RegionInstanceID != runtime2.RegionInstanceID("region-1") {
		parseT.Fatalf("expected region-1, got %#v", parseEnvelope)
	}
}

// TestBuildSnapshotTransportPayloadWithFallbackSupportsStructuredOnlyAndRejectsNoSupport verifies snapshot transport build succeeds on structured-only capability reports and fails clearly when no transport tier is available.
func TestBuildSnapshotTransportPayloadWithFallbackSupportsStructuredOnlyAndRejectsNoSupport(parseT *testing.T) {
	parseEnvelope := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     1,
		Props:            map[string]any{"title": "Orders"},
	}
	parseTier, parsePayload, parseErr := runtime2.BuildSnapshotTransportPayloadWithFallback(parseEnvelope, runtime2.CapabilityReport{
		HasWorkerSupport:          true,
		HasStructuredCloneSupport: true,
	})
	if parseErr != nil {
		parseT.Fatalf("BuildSnapshotTransportPayloadWithFallback(structured only) returned error: %v", parseErr)
	}
	if parseTier != runtime2.TransportTierStructuredClone {
		parseT.Fatalf("expected structured-only transport tier, got %q", parseTier)
	}
	parseDecodedTier, parseDecodedEnvelope, parseDecodeErr := runtime2.ParseSnapshotTransportPayloadWithFallback(parseTier, parsePayload)
	if parseDecodeErr != nil {
		parseT.Fatalf("ParseSnapshotTransportPayloadWithFallback(structured only) returned error: %v", parseDecodeErr)
	}
	if parseDecodedTier != runtime2.TransportTierStructuredClone || parseDecodedEnvelope.RegionInstanceID != runtime2.RegionInstanceID("region-1") {
		parseT.Fatalf("unexpected structured-only decode result tier=%q envelope=%#v", parseDecodedTier, parseDecodedEnvelope)
	}

	if _, _, parseErr := runtime2.BuildSnapshotTransportPayloadWithFallback(parseEnvelope, runtime2.CapabilityReport{}); parseErr == nil {
		parseT.Fatal("expected missing snapshot transport support to fail")
	}
}

// TestBuildSnapshotTransportPayloadWithFallbackReturnsBinaryErrorWithoutStructuredFallback verifies binary snapshot build failures are returned directly when structured-clone fallback is unavailable.
func TestBuildSnapshotTransportPayloadWithFallbackReturnsBinaryErrorWithoutStructuredFallback(parseT *testing.T) {
	parseProps := make(map[string]any, 65)
	for parseIndex := range 65 {
		parseProps[string(rune('a'+(parseIndex%26)))+string(rune('A'+(parseIndex/26)))] = parseIndex
	}
	if _, _, parseErr := runtime2.BuildSnapshotTransportPayloadWithFallback(runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     1,
		Props:            parseProps,
	}, runtime2.CapabilityReport{
		HasWorkerSupport:          true,
		HasBinaryTransportSupport: true,
	}); parseErr == nil {
		parseT.Fatal("expected binary-only snapshot transport build to fail when binary encode fails")
	}
}

// TestParseSnapshotTransportPayloadWithFallbackRejectsMalformedFallbacksAndUnsupportedTiers verifies snapshot transport parse reports failures when neither binary nor structured-clone decoding succeeds and when the requested tier is unsupported.
func TestParseSnapshotTransportPayloadWithFallbackRejectsMalformedFallbacksAndUnsupportedTiers(parseT *testing.T) {
	if _, _, parseErr := runtime2.ParseSnapshotTransportPayloadWithFallback(runtime2.TransportTierBinary, []byte("not-a-binary-or-structured-payload")); parseErr == nil {
		parseT.Fatal("expected malformed binary-tier payload to fail both binary and structured fallback decode")
	}
	if _, _, parseErr := runtime2.ParseSnapshotTransportPayloadWithFallback(runtime2.TransportTierStructuredClone, []byte("{")); parseErr == nil {
		parseT.Fatal("expected malformed structured-clone payload to fail")
	}
	if _, _, parseErr := runtime2.ParseSnapshotTransportPayloadWithFallback(runtime2.TransportTier("shared-buffer"), []byte("{}")); parseErr == nil {
		parseT.Fatal("expected unsupported snapshot decode tier to fail")
	}
}

// parseSnapshotTransportPayload decodes one snapshot payload using the given transport tier.
func parseSnapshotTransportPayload(parseTier runtime2.TransportTier, parsePayload []byte) (runtime2.SnapshotEnvelope, runtime2.TransportTier, error) {
	parseDecodedTier, parseEnvelope, parseErr := runtime2.ParseSnapshotTransportPayloadWithFallback(parseTier, parsePayload)
	return parseEnvelope, parseDecodedTier, parseErr
}
