package runtime2_test

import (
	"bytes"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestValidateProtocolVersionMatchAcceptsMatchingVersions verifies matching versions compare equal.
func TestValidateProtocolVersionMatchAcceptsMatchingVersions(parseT *testing.T) {
	parseVersion, parseErr := runtime2.ParseProtocolVersion("gwc.parallel.v1")
	if parseErr != nil {
		parseT.Fatalf("ParseProtocolVersion returned error: %v", parseErr)
	}
	if parseErr := runtime2.ValidateProtocolVersionMatch(runtime2.ProtocolVersionParallelV1, parseVersion); parseErr != nil {
		parseT.Fatalf("ValidateProtocolVersionMatch returned error: %v", parseErr)
	}
}

// TestValidateProtocolVersionMatchRejectsMismatchedVersions verifies mismatched versions fail clearly.
func TestValidateProtocolVersionMatchRejectsMismatchedVersions(parseT *testing.T) {
	parseErr := runtime2.ValidateProtocolVersionMatch(runtime2.ProtocolVersionParallelV1, runtime2.ProtocolVersion("gwc.parallel.v2"))
	if parseErr == nil {
		parseT.Fatal("expected mismatched protocol versions to fail")
	}
}

// TestParseProtocolVersionRejectsEmptyOrUnknownVersions verifies invalid versions fail validation.
func TestParseProtocolVersionRejectsEmptyOrUnknownVersions(parseT *testing.T) {
	parseCases := []string{"", "gwc.parallel.unknown"}
	for _, parseCase := range parseCases {
		if _, parseErr := runtime2.ParseProtocolVersion(parseCase); parseErr == nil {
			parseT.Fatalf("expected ParseProtocolVersion(%q) to fail", parseCase)
		}
	}
}

// TestBuildCapabilityFixtureJSONRoundTrips verifies fixture encoding and decoding stays stable.
func TestBuildCapabilityFixtureJSONRoundTrips(parseT *testing.T) {
	parseFixture := runtime2.CapabilityFixture{
		ProtocolVersion: runtime2.ProtocolVersionParallelV1,
		Capabilities: runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
			HasWorkerSupport:          true,
			HasStructuredCloneSupport: true,
			HasMessagePortSupport:     true,
		}),
	}
	parseEncoded, parseErr := runtime2.BuildCapabilityFixtureJSON(parseFixture)
	if parseErr != nil {
		parseT.Fatalf("BuildCapabilityFixtureJSON returned error: %v", parseErr)
	}
	parseDecoded, parseErr := runtime2.ParseCapabilityFixtureJSON(parseEncoded)
	if parseErr != nil {
		parseT.Fatalf("ParseCapabilityFixtureJSON returned error: %v", parseErr)
	}
	parseRoundTrip, parseErr := runtime2.BuildCapabilityFixtureJSON(parseDecoded)
	if parseErr != nil {
		parseT.Fatalf("BuildCapabilityFixtureJSON round trip returned error: %v", parseErr)
	}
	if !bytes.Equal(parseEncoded, parseRoundTrip) {
		parseT.Fatalf("expected stable fixture round trip, first=%s second=%s", string(parseEncoded), string(parseRoundTrip))
	}
}

// TestParseCapabilityFixtureJSONRejectsInvalidVersion verifies invalid fixture versions fail validation.
func TestParseCapabilityFixtureJSONRejectsInvalidVersion(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v9","capabilities":{"HasWorkerSupport":true}}`)
	if _, parseErr := runtime2.ParseCapabilityFixtureJSON(parseValue); parseErr == nil {
		parseT.Fatal("expected invalid fixture version to fail")
	}
}

// TestParseCapabilityFixtureJSONRejectsMissingCapabilities verifies missing capability fields fail clearly.
func TestParseCapabilityFixtureJSONRejectsMissingCapabilities(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1"}`)
	if _, parseErr := runtime2.ParseCapabilityFixtureJSON(parseValue); parseErr == nil {
		parseT.Fatal("expected missing capabilities to fail")
	}
}
