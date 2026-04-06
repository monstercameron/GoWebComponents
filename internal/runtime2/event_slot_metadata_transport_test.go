package runtime2

import (
	"strings"
	"testing"
)

// TestBuildAndParseEventSlotMetadataJSONRoundTrip verifies event-slot metadata has a stable JSON path.
func TestBuildAndParseEventSlotMetadataJSONRoundTrip(parseT *testing.T) {
	parseMetadata := EventSlotMetadata{
		Version: EventSlotMetadataVersionV1,
		Slots: []EventSlotRecord{
			{
				SlotID:    "slot-1",
				EventType: "click",
			},
		},
	}
	parsePayload, parseBuildErr := BuildEventSlotMetadataJSON(parseMetadata)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildEventSlotMetadataJSON returned error: %v", parseBuildErr)
	}
	parseDecoded, parseParseErr := ParseEventSlotMetadataJSON(parsePayload)
	if parseParseErr != nil {
		parseT.Fatalf("ParseEventSlotMetadataJSON returned error: %v", parseParseErr)
	}
	if parseDecoded.Version != EventSlotMetadataVersionV1 {
		parseT.Fatalf("decoded event-slot metadata version = %q, want %q", parseDecoded.Version, EventSlotMetadataVersionV1)
	}
	if len(parseDecoded.Slots) != 1 || parseDecoded.Slots[0].EventType != "click" {
		parseT.Fatalf("decoded event-slot metadata slots = %+v, want one click slot", parseDecoded.Slots)
	}
}

// TestBuildAndParseEventSlotMetadataPlaceholderJSONRoundTrip verifies the legacy placeholder helper names route through the stable JSON path.
func TestBuildAndParseEventSlotMetadataPlaceholderJSONRoundTrip(parseT *testing.T) {
	parseMetadata := EventSlotMetadata{
		Version: EventSlotMetadataVersionV1,
		Slots: []EventSlotRecord{
			{
				SlotID:    "slot-1",
				EventType: "click",
			},
		},
	}
	parsePayload, parseBuildErr := BuildEventSlotMetadataPlaceholderJSON(parseMetadata)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildEventSlotMetadataPlaceholderJSON returned error: %v", parseBuildErr)
	}
	parseDecoded, parseParseErr := ParseEventSlotMetadataPlaceholderJSON(parsePayload)
	if parseParseErr != nil {
		parseT.Fatalf("ParseEventSlotMetadataPlaceholderJSON returned error: %v", parseParseErr)
	}
	if parseDecoded.Version != EventSlotMetadataVersionV1 {
		parseT.Fatalf("decoded event-slot metadata version = %q, want %q", parseDecoded.Version, EventSlotMetadataVersionV1)
	}
}

// TestBuildEventSlotMetadataJSONRejectsInvalidShape verifies the encoder enforces metadata validation.
func TestBuildEventSlotMetadataJSONRejectsInvalidShape(parseT *testing.T) {
	_, parseErr := BuildEventSlotMetadataJSON(EventSlotMetadata{
		Version: EventSlotMetadataVersionV1,
		Slots: []EventSlotRecord{
			{
				SlotID:    "",
				EventType: "click",
			},
		},
	})
	if parseErr == nil {
		parseT.Fatal("expected BuildEventSlotMetadataJSON invalid shape to fail")
	}
}

// TestParseEventSlotMetadataJSONRejectsMalformedPayload verifies malformed payloads fail decode.
func TestParseEventSlotMetadataJSONRejectsMalformedPayload(parseT *testing.T) {
	_, parseErr := ParseEventSlotMetadataJSON([]byte(`{"version":`))
	if parseErr == nil {
		parseT.Fatal("expected ParseEventSlotMetadataJSON malformed payload to fail")
	}
	if !strings.Contains(parseErr.Error(), "decode event-slot metadata payload") {
		parseT.Fatalf("expected decode error details, got %v", parseErr)
	}
}

// TestParseEventSlotMetadataJSONRejectsUnsupportedVersion verifies unknown versions fail on decode.
func TestParseEventSlotMetadataJSONRejectsUnsupportedVersion(parseT *testing.T) {
	_, parseErr := ParseEventSlotMetadataJSON([]byte(`{"version":"gwc.parallel.event-slot.v99"}`))
	if parseErr == nil {
		parseT.Fatal("expected ParseEventSlotMetadataJSON unknown version to fail")
	}
	if !strings.Contains(parseErr.Error(), "unsupported") {
		parseT.Fatalf("expected unsupported-version error details, got %v", parseErr)
	}
}

// TestParseEventSlotMetadataJSONNormalizesLegacyPlaceholderVersion verifies legacy placeholder payloads still decode through the new schema.
func TestParseEventSlotMetadataJSONNormalizesLegacyPlaceholderVersion(parseT *testing.T) {
	parseDecoded, parseErr := ParseEventSlotMetadataJSON([]byte(`{"version":"gwc.parallel.event-slot.placeholder.v1","slots":[{"slot_id":"slot-1","event_type":"click"}]}`))
	if parseErr != nil {
		parseT.Fatalf("ParseEventSlotMetadataJSON returned error: %v", parseErr)
	}
	if parseDecoded.Version != EventSlotMetadataVersionV1 {
		parseT.Fatalf("decoded event-slot metadata version = %q, want normalized %q", parseDecoded.Version, EventSlotMetadataVersionV1)
	}
	if len(parseDecoded.Slots) != 1 || parseDecoded.Slots[0].SlotID != "slot-1" {
		parseT.Fatalf("decoded event-slot metadata slots = %+v, want one normalized slot", parseDecoded.Slots)
	}
}
