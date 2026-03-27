package runtime2

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestEventSlotMetadataVersionPlaceholderIsDefined verifies the placeholder event-slot schema version constant is stable and non-empty.
func TestEventSlotMetadataVersionPlaceholderIsDefined(parseT *testing.T) {
	if EventSlotMetadataVersionPlaceholderV1 == "" {
		parseT.Fatal("expected placeholder event-slot metadata version to be defined")
	}
}

// TestEventSlotMetadataPlaceholderRoundTrip verifies placeholder event-slot metadata round-trips through JSON with stable fields.
func TestEventSlotMetadataPlaceholderRoundTrip(parseT *testing.T) {
	parseMetadata := EventSlotMetadata{
		Version: EventSlotMetadataVersionPlaceholderV1,
	}
	parsePayload, parseMarshalErr := json.Marshal(parseMetadata)
	if parseMarshalErr != nil {
		parseT.Fatalf("json.Marshal(EventSlotMetadata) returned error: %v", parseMarshalErr)
	}
	var parseDecoded EventSlotMetadata
	if parseUnmarshalErr := json.Unmarshal(parsePayload, &parseDecoded); parseUnmarshalErr != nil {
		parseT.Fatalf("json.Unmarshal(EventSlotMetadata) returned error: %v", parseUnmarshalErr)
	}
	if parseDecoded.Version != EventSlotMetadataVersionPlaceholderV1 {
		parseT.Fatalf("decoded event-slot metadata version = %q, want %q", parseDecoded.Version, EventSlotMetadataVersionPlaceholderV1)
	}
	if len(parseDecoded.Slots) != 0 {
		parseT.Fatalf("decoded event-slot metadata slots = %d, want 0 for non-operative placeholder", len(parseDecoded.Slots))
	}
}

// TestValidateEventSlotMetadataAcceptsEmptyPlaceholder verifies metadata is optional when no slots are provided.
func TestValidateEventSlotMetadataAcceptsEmptyPlaceholder(parseT *testing.T) {
	if parseErr := ValidateEventSlotMetadata(EventSlotMetadata{}); parseErr != nil {
		parseT.Fatalf("ValidateEventSlotMetadata(empty) returned error: %v", parseErr)
	}
}

// TestValidateEventSlotMetadataRejectsUnknownVersion verifies unknown versions fail validation.
func TestValidateEventSlotMetadataRejectsUnknownVersion(parseT *testing.T) {
	parseErr := ValidateEventSlotMetadata(EventSlotMetadata{
		Version: EventSlotMetadataVersion("gwc.parallel.event-slot.placeholder.v99"),
	})
	if parseErr == nil {
		parseT.Fatal("expected unknown event-slot metadata version to fail")
	}
	if !strings.Contains(parseErr.Error(), "unsupported") {
		parseT.Fatalf("expected unknown-version error details, got %v", parseErr)
	}
}

// TestValidateEventSlotMetadataRejectsSlotsWithoutVersion verifies active slots are blocked in the first-slice non-operative placeholder mode.
func TestValidateEventSlotMetadataRejectsSlotsWithoutVersion(parseT *testing.T) {
	parseErr := ValidateEventSlotMetadata(EventSlotMetadata{
		Slots: []EventSlotRecord{
			{
				SlotID:    "slot-1",
				EventType: "click",
			},
		},
	})
	if parseErr == nil {
		parseT.Fatal("expected slots without version to fail")
	}
	if !strings.Contains(parseErr.Error(), "active slots are unsupported") {
		parseT.Fatalf("expected non-operative-slot error details, got %v", parseErr)
	}
}

// TestValidateEventSlotMetadataRejectsActiveSlotsWithPlaceholderVersion verifies active slot declarations are blocked even when version is present.
func TestValidateEventSlotMetadataRejectsActiveSlotsWithPlaceholderVersion(parseT *testing.T) {
	parseErr := ValidateEventSlotMetadata(EventSlotMetadata{
		Version: EventSlotMetadataVersionPlaceholderV1,
		Slots: []EventSlotRecord{
			{
				SlotID:    "slot-1",
				EventType: "click",
			},
		},
	})
	if parseErr == nil {
		parseT.Fatal("expected active slots with placeholder version to fail")
	}
	if !strings.Contains(parseErr.Error(), "active slots are unsupported") {
		parseT.Fatalf("expected non-operative-slot error details, got %v", parseErr)
	}
}
