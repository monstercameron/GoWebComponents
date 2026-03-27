package runtime2

import (
	"bytes"
	"testing"
)

func buildSharedPatchEnvelopeForTest() StructuredClonePatchEnvelope {
	return StructuredClonePatchEnvelope{
		RegionInstanceID: "region-1",
		Epoch:            2,
		PatchVersion:     4,
		PatchPayload:     []byte("patch-stream-bytes"),
	}
}

func buildSharedPatchCapabilityReportForTest() CapabilityReport {
	return BuildCapabilityReport(CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasBinaryTransportSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
}

// TestHandleSharedPatchPublishAndReadRoundTrip verifies shared patch pages publish and read one complete payload round-trip.
func TestHandleSharedPatchPublishAndReadRoundTrip(parseT *testing.T) {
	parseSharedPatchPage, parseSharedPatchPageErr := BuildSharedPatchPage(1024)
	if parseSharedPatchPageErr != nil {
		parseT.Fatalf("BuildSharedPatchPage returned error: %v", parseSharedPatchPageErr)
	}
	parsePayload := []byte("patch-bytes")
	_, parsePublishErr := parseSharedPatchPage.HandleSharedPatchPublishPayload(parsePayload)
	if parsePublishErr != nil {
		parseT.Fatalf("HandleSharedPatchPublishPayload returned error: %v", parsePublishErr)
	}
	parseReadPayload, parseReadPayloadErr := parseSharedPatchPage.GetSharedPatchReadPayload()
	if parseReadPayloadErr != nil {
		parseT.Fatalf("GetSharedPatchReadPayload returned error: %v", parseReadPayloadErr)
	}
	if !bytes.Equal(parseReadPayload, parsePayload) {
		parseT.Fatalf("expected shared patch payload %q, got %q", string(parsePayload), string(parseReadPayload))
	}
}

// TestBuildSharedPatchTransportResultPrefersSharedBuffer verifies patch transport prefers shared-buffer when capability and page availability requirements are met.
func TestBuildSharedPatchTransportResultPrefersSharedBuffer(parseT *testing.T) {
	parseSharedPatchPage, parseSharedPatchPageErr := BuildSharedPatchPage(2048)
	if parseSharedPatchPageErr != nil {
		parseT.Fatalf("BuildSharedPatchPage returned error: %v", parseSharedPatchPageErr)
	}
	parseResult, parseResultErr := BuildSharedPatchTransportResult(
		buildSharedPatchEnvelopeForTest(),
		buildSharedPatchCapabilityReportForTest(),
		parseSharedPatchPage,
	)
	if parseResultErr != nil {
		parseT.Fatalf("BuildSharedPatchTransportResult returned error: %v", parseResultErr)
	}
	if parseResult.GetTransportTier != TransportTierSharedBuffer {
		parseT.Fatalf("expected shared-buffer transport tier, got %q", parseResult.GetTransportTier)
	}
	parseDecodedEnvelope, parseDecodeErr := ParseSharedPatchPayloadFromPage(parseSharedPatchPage)
	if parseDecodeErr != nil {
		parseT.Fatalf("ParseSharedPatchPayloadFromPage returned error: %v", parseDecodeErr)
	}
	if parseDecodedEnvelope.PatchVersion != 4 {
		parseT.Fatalf("expected decoded patch version 4, got %d", parseDecodedEnvelope.PatchVersion)
	}
}

// TestBuildSharedPatchTransportResultDowngradesWhenSharedUnavailable verifies shared patch transport downgrades to structured-clone when shared memory is unavailable or invalid.
func TestBuildSharedPatchTransportResultDowngradesWhenSharedUnavailable(parseT *testing.T) {
	parseUnavailableResult, parseUnavailableErr := BuildSharedPatchTransportResult(
		buildSharedPatchEnvelopeForTest(),
		buildSharedPatchCapabilityReportForTest(),
		nil,
	)
	if parseUnavailableErr != nil {
		parseT.Fatalf("BuildSharedPatchTransportResult(unavailable) returned error: %v", parseUnavailableErr)
	}
	if parseUnavailableResult.GetTransportTier != TransportTierStructuredClone {
		parseT.Fatalf("expected structured-clone tier for unavailable shared memory, got %q", parseUnavailableResult.GetTransportTier)
	}
	if parseUnavailableResult.GetDowngradeReason != SharedPatchDowngradeReasonSharedMemoryUnavailable {
		parseT.Fatalf("expected unavailable downgrade reason, got %q", parseUnavailableResult.GetDowngradeReason)
	}
	parseSmallPage, parseSmallPageErr := BuildSharedPatchPage(sharedSnapshotPageHeaderSize + 1)
	if parseSmallPageErr != nil {
		parseT.Fatalf("BuildSharedPatchPage returned error: %v", parseSmallPageErr)
	}
	parseInvalidResult, parseInvalidErr := BuildSharedPatchTransportResult(
		buildSharedPatchEnvelopeForTest(),
		buildSharedPatchCapabilityReportForTest(),
		parseSmallPage,
	)
	if parseInvalidErr != nil {
		parseT.Fatalf("BuildSharedPatchTransportResult(invalid page) returned error: %v", parseInvalidErr)
	}
	if parseInvalidResult.GetTransportTier != TransportTierStructuredClone {
		parseT.Fatalf("expected structured-clone tier for invalid shared page, got %q", parseInvalidResult.GetTransportTier)
	}
	if parseInvalidResult.GetDowngradeReason != SharedPatchDowngradeReasonInvalidSharedPage {
		parseT.Fatalf("expected invalid-page downgrade reason, got %q", parseInvalidResult.GetDowngradeReason)
	}
}
