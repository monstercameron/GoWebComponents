package runtime2

import "testing"

// TestParsePatchStreamHeaderValidHeaderDecodes verifies a valid patch-stream header decodes successfully.
func TestParsePatchStreamHeaderValidHeaderDecodes(parseTesting *testing.T) {
	parseRawHeader := PatchStreamHeaderRaw{
		ProtocolVersion: PatchStreamProtocolVersion,
		RegionID:        "region-42",
		Epoch:           3,
		InputVersion:    9,
		PatchVersion:    5,
	}
	parseHeader, parseErr := ParsePatchStreamHeader(parseRawHeader, "region-42")
	if parseErr != nil {
		parseTesting.Fatalf("ParsePatchStreamHeader(valid) error = %v", parseErr)
	}
	if parseHeader.RegionID != "region-42" {
		parseTesting.Fatalf("ParsePatchStreamHeader(valid) RegionID = %q, want %q", parseHeader.RegionID, "region-42")
	}
}

// TestParsePatchStreamHeaderWrongRegionIDFails verifies region mismatches are rejected.
func TestParsePatchStreamHeaderWrongRegionIDFails(parseTesting *testing.T) {
	parseRawHeader := PatchStreamHeaderRaw{
		ProtocolVersion: PatchStreamProtocolVersion,
		RegionID:        "region-99",
		Epoch:           1,
		InputVersion:    2,
		PatchVersion:    2,
	}
	_, parseErr := ParsePatchStreamHeader(parseRawHeader, "region-42")
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchStreamHeader(wrong region) error = nil, want error")
	}
}

// TestParsePatchStreamHeaderWrongVersionFails verifies protocol version mismatches are rejected.
func TestParsePatchStreamHeaderWrongVersionFails(parseTesting *testing.T) {
	parseRawHeader := PatchStreamHeaderRaw{
		ProtocolVersion: "gwc.parallel.v0",
		RegionID:        "region-42",
		Epoch:           1,
		InputVersion:    2,
		PatchVersion:    2,
	}
	_, parseErr := ParsePatchStreamHeader(parseRawHeader, "region-42")
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchStreamHeader(wrong version) error = nil, want error")
	}
}
