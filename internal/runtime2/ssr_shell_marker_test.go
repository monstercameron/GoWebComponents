package runtime2

import (
	"encoding/json"
	"testing"
)

// TestValidateSSRShellMarkerAcceptsValidMarker verifies the SSR shell marker format accepts a minimal valid payload.
func TestValidateSSRShellMarkerAcceptsValidMarker(parseT *testing.T) {
	parseMarker := SSRShellMarker{
		Version:          SSRShellMarkerVersionV1,
		RegionInstanceID: RegionInstanceID("region-1"),
		RendererID:       RendererID("dashboard.hot-panel"),
	}
	if parseErr := ValidateSSRShellMarker(parseMarker); parseErr != nil {
		parseT.Fatalf("ValidateSSRShellMarker(valid) returned error: %v", parseErr)
	}
}

// TestValidateSSRShellMarkerRejectsMissingRegion verifies SSR shell marker payloads require region identity.
func TestValidateSSRShellMarkerRejectsMissingRegion(parseT *testing.T) {
	parseMarker := SSRShellMarker{
		Version:    SSRShellMarkerVersionV1,
		RendererID: RendererID("dashboard.hot-panel"),
	}
	if parseErr := ValidateSSRShellMarker(parseMarker); parseErr == nil {
		parseT.Fatal("expected SSR shell marker missing region to fail")
	}
}

// TestParseSSRShellMarkerVersionRejectsUnsupportedVersion verifies unsupported marker versions fail clearly.
func TestParseSSRShellMarkerVersionRejectsUnsupportedVersion(parseT *testing.T) {
	if _, parseErr := ParseSSRShellMarkerVersion("gwc.runtime2.ssr-shell.v99"); parseErr == nil {
		parseT.Fatal("expected unsupported SSR shell marker version to fail")
	}
}

// TestBuildSSRShellMarkerAttributeValueEncodesValidatedMarker verifies local SSR encoding produces a parseable attribute payload.
func TestBuildSSRShellMarkerAttributeValueEncodesValidatedMarker(parseT *testing.T) {
	parseMarker := SSRShellMarker{
		Version:          SSRShellMarkerVersionV1,
		RegionInstanceID: RegionInstanceID("region-1"),
		RendererID:       RendererID("dashboard.hot-panel"),
	}
	parsePayload, parseErr := BuildSSRShellMarkerAttributeValue(parseMarker)
	if parseErr != nil {
		parseT.Fatalf("BuildSSRShellMarkerAttributeValue(valid) returned error: %v", parseErr)
	}
	var parseDecoded SSRShellMarker
	if parseDecodeErr := json.Unmarshal([]byte(parsePayload), &parseDecoded); parseDecodeErr != nil {
		parseT.Fatalf("json.Unmarshal(shell marker payload) returned error: %v", parseDecodeErr)
	}
	if parseDecoded != parseMarker {
		parseT.Fatalf("decoded shell marker = %+v, want %+v", parseDecoded, parseMarker)
	}
}

// TestBuildSSRShellMarkerAttributeValueRejectsInvalidMarker verifies local SSR encoding rejects incomplete marker payloads.
func TestBuildSSRShellMarkerAttributeValueRejectsInvalidMarker(parseT *testing.T) {
	_, parseErr := BuildSSRShellMarkerAttributeValue(SSRShellMarker{
		Version: SSRShellMarkerVersionV1,
	})
	if parseErr == nil {
		parseT.Fatal("expected BuildSSRShellMarkerAttributeValue(invalid marker) to fail")
	}
}

// TestParseSSRShellMarkerAttributeValueAcceptsEncodedPayload verifies client parsing accepts encoded marker payloads.
func TestParseSSRShellMarkerAttributeValueAcceptsEncodedPayload(parseT *testing.T) {
	parseMarker := SSRShellMarker{
		Version:          SSRShellMarkerVersionV1,
		RegionInstanceID: RegionInstanceID("region-1"),
		RendererID:       RendererID("dashboard.hot-panel"),
	}
	parsePayload, parseBuildErr := BuildSSRShellMarkerAttributeValue(parseMarker)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildSSRShellMarkerAttributeValue(valid) returned error: %v", parseBuildErr)
	}
	parseParsedMarker, parseParseErr := ParseSSRShellMarkerAttributeValue(parsePayload)
	if parseParseErr != nil {
		parseT.Fatalf("ParseSSRShellMarkerAttributeValue(valid) returned error: %v", parseParseErr)
	}
	if parseParsedMarker != parseMarker {
		parseT.Fatalf("ParseSSRShellMarkerAttributeValue(valid) marker = %+v, want %+v", parseParsedMarker, parseMarker)
	}
}

// TestParseSSRShellMarkerAttributeValueRejectsMalformedPayload verifies client parsing rejects malformed marker payloads.
func TestParseSSRShellMarkerAttributeValueRejectsMalformedPayload(parseT *testing.T) {
	if _, parseErr := ParseSSRShellMarkerAttributeValue(`{"version":`); parseErr == nil {
		parseT.Fatal("expected ParseSSRShellMarkerAttributeValue(malformed payload) to fail")
	}
}

// TestParseSSRShellMarkerRejectsMissingVersionAndPayload verifies SSR shell marker parsing rejects missing version identifiers and empty payloads.
func TestParseSSRShellMarkerRejectsMissingVersionAndPayload(parseT *testing.T) {
	if _, parseErr := ParseSSRShellMarkerVersion(""); parseErr == nil {
		parseT.Fatal("expected ParseSSRShellMarkerVersion(empty) to fail")
	}
	if parseErr := ValidateSSRShellMarker(SSRShellMarker{
		RegionInstanceID: RegionInstanceID("region-1"),
		RendererID:       RendererID("dashboard.hot-panel"),
	}); parseErr == nil {
		parseT.Fatal("expected ValidateSSRShellMarker(missing version) to fail")
	}
	if _, parseErr := ParseSSRShellMarkerAttributeValue(" \n\t "); parseErr == nil {
		parseT.Fatal("expected ParseSSRShellMarkerAttributeValue(empty payload) to fail")
	}
}
