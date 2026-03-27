package runtime2

import (
	"strings"
	"testing"
)

// TestBuildAndParseEventSlotMetadataPlaceholderJSONRoundTrip verifies placeholder metadata has a dedicated stable JSON path.
func TestBuildAndParseEventSlotMetadataPlaceholderJSONRoundTrip(parseT *testing.T) {
	parseMetadata := EventSlotMetadata{
		Version: EventSlotMetadataVersionPlaceholderV1,
	}
	parsePayload, parseBuildErr := BuildEventSlotMetadataPlaceholderJSON(parseMetadata)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildEventSlotMetadataPlaceholderJSON returned error: %v", parseBuildErr)
	}
	parseDecoded, parseParseErr := ParseEventSlotMetadataPlaceholderJSON(parsePayload)
	if parseParseErr != nil {
		parseT.Fatalf("ParseEventSlotMetadataPlaceholderJSON returned error: %v", parseParseErr)
	}
	if parseDecoded.Version != EventSlotMetadataVersionPlaceholderV1 {
		parseT.Fatalf("decoded event-slot metadata version = %q, want %q", parseDecoded.Version, EventSlotMetadataVersionPlaceholderV1)
	}
	if len(parseDecoded.Slots) != 0 {
		parseT.Fatalf("decoded event-slot metadata slots = %+v, want non-operative empty slots", parseDecoded.Slots)
	}
}

// TestBuildEventSlotMetadataPlaceholderJSONRejectsInvalidShape verifies the placeholder encoder enforces metadata validation.
func TestBuildEventSlotMetadataPlaceholderJSONRejectsInvalidShape(parseT *testing.T) {
	_, parseErr := BuildEventSlotMetadataPlaceholderJSON(EventSlotMetadata{
		Slots: []EventSlotRecord{
			{
				SlotID:    "slot-1",
				EventType: "click",
			},
		},
	})
	if parseErr == nil {
		parseT.Fatal("expected BuildEventSlotMetadataPlaceholderJSON invalid shape to fail")
	}
}

// TestParseEventSlotMetadataPlaceholderJSONRejectsMalformedPayload verifies malformed payloads fail decode.
func TestParseEventSlotMetadataPlaceholderJSONRejectsMalformedPayload(parseT *testing.T) {
	_, parseErr := ParseEventSlotMetadataPlaceholderJSON([]byte(`{"version":`))
	if parseErr == nil {
		parseT.Fatal("expected ParseEventSlotMetadataPlaceholderJSON malformed payload to fail")
	}
	if !strings.Contains(parseErr.Error(), "decode event-slot metadata placeholder payload") {
		parseT.Fatalf("expected decode error details, got %v", parseErr)
	}
}

// TestParseEventSlotMetadataPlaceholderJSONRejectsUnsupportedVersion verifies unknown versions fail on decode.
func TestParseEventSlotMetadataPlaceholderJSONRejectsUnsupportedVersion(parseT *testing.T) {
	_, parseErr := ParseEventSlotMetadataPlaceholderJSON([]byte(`{"version":"gwc.parallel.event-slot.placeholder.v99"}`))
	if parseErr == nil {
		parseT.Fatal("expected ParseEventSlotMetadataPlaceholderJSON unknown version to fail")
	}
	if !strings.Contains(parseErr.Error(), "unsupported") {
		parseT.Fatalf("expected unsupported-version error details, got %v", parseErr)
	}
}

// TestParseEventSlotMetadataPlaceholderJSONRejectsActiveSlots verifies decode blocks active slot declarations in slice one.
func TestParseEventSlotMetadataPlaceholderJSONRejectsActiveSlots(parseT *testing.T) {
	_, parseErr := ParseEventSlotMetadataPlaceholderJSON([]byte(`{"version":"gwc.parallel.event-slot.placeholder.v1","slots":[{"slot_id":"slot-1","event_type":"click"}]}`))
	if parseErr == nil {
		parseT.Fatal("expected ParseEventSlotMetadataPlaceholderJSON active slots to fail")
	}
	if !strings.Contains(parseErr.Error(), "active slots are unsupported") {
		parseT.Fatalf("expected non-operative-slot error details, got %v", parseErr)
	}
}
