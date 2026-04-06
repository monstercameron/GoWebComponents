package runtime2

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestEventSlotMetadataVersionV1IsDefined verifies the active event-slot schema version constant is stable and non-empty.
func TestEventSlotMetadataVersionV1IsDefined(parseT *testing.T) {
	if EventSlotMetadataVersionV1 == "" {
		parseT.Fatal("expected event-slot metadata version to be defined")
	}
}

// TestEventSlotMetadataRoundTrip verifies event-slot metadata round-trips through JSON with stable fields.
func TestEventSlotMetadataRoundTrip(parseT *testing.T) {
	parseMetadata := EventSlotMetadata{
		Version: EventSlotMetadataVersionV1,
		Slots: []EventSlotRecord{
			{
				SlotID:    "slot-1",
				EventType: "click",
			},
			{
				SlotID:    "slot-1",
				EventType: "keydown",
			},
		},
	}
	parsePayload, parseMarshalErr := json.Marshal(parseMetadata)
	if parseMarshalErr != nil {
		parseT.Fatalf("json.Marshal(EventSlotMetadata) returned error: %v", parseMarshalErr)
	}
	var parseDecoded EventSlotMetadata
	if parseUnmarshalErr := json.Unmarshal(parsePayload, &parseDecoded); parseUnmarshalErr != nil {
		parseT.Fatalf("json.Unmarshal(EventSlotMetadata) returned error: %v", parseUnmarshalErr)
	}
	if parseDecoded.Version != EventSlotMetadataVersionV1 {
		parseT.Fatalf("decoded event-slot metadata version = %q, want %q", parseDecoded.Version, EventSlotMetadataVersionV1)
	}
	if len(parseDecoded.Slots) != 2 {
		parseT.Fatalf("decoded event-slot metadata slots = %d, want 2", len(parseDecoded.Slots))
	}
}

// TestValidateEventSlotMetadataAcceptsEmptyMetadata verifies metadata is optional when no version or slots are provided.
func TestValidateEventSlotMetadataAcceptsEmptyMetadata(parseT *testing.T) {
	if parseErr := ValidateEventSlotMetadata(EventSlotMetadata{}); parseErr != nil {
		parseT.Fatalf("ValidateEventSlotMetadata(empty) returned error: %v", parseErr)
	}
}

// TestValidateEventSlotMetadataAcceptsOperationalSlots verifies the active schema accepts declared event slots.
func TestValidateEventSlotMetadataAcceptsOperationalSlots(parseT *testing.T) {
	parseErr := ValidateEventSlotMetadata(EventSlotMetadata{
		Version: EventSlotMetadataVersionV1,
		Slots: []EventSlotRecord{
			{
				SlotID:    "slot-1",
				EventType: "click",
			},
			{
				SlotID:    "slot-1",
				EventType: "keydown",
			},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("ValidateEventSlotMetadata(valid) returned error: %v", parseErr)
	}
}

// TestValidateEventSlotMetadataRejectsUnknownVersion verifies unknown versions fail validation.
func TestValidateEventSlotMetadataRejectsUnknownVersion(parseT *testing.T) {
	parseErr := ValidateEventSlotMetadata(EventSlotMetadata{
		Version: EventSlotMetadataVersion("gwc.parallel.event-slot.v99"),
	})
	if parseErr == nil {
		parseT.Fatal("expected unknown event-slot metadata version to fail")
	}
	if !strings.Contains(parseErr.Error(), "unsupported") {
		parseT.Fatalf("expected unknown-version error details, got %v", parseErr)
	}
}

// TestValidateEventSlotMetadataRejectsSlotsWithoutVersion verifies declared slots require an explicit schema version.
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
	if !strings.Contains(parseErr.Error(), "version is required") {
		parseT.Fatalf("expected missing-version error details, got %v", parseErr)
	}
}

// TestValidateEventSlotMetadataRejectsBlankSlotID verifies slot declarations require a slot ID.
func TestValidateEventSlotMetadataRejectsBlankSlotID(parseT *testing.T) {
	parseErr := ValidateEventSlotMetadata(EventSlotMetadata{
		Version: EventSlotMetadataVersionV1,
		Slots: []EventSlotRecord{
			{
				SlotID:    "",
				EventType: "click",
			},
		},
	})
	if parseErr == nil {
		parseT.Fatal("expected blank slot ID to fail")
	}
	if !strings.Contains(parseErr.Error(), "slot ID is required") {
		parseT.Fatalf("expected slot-id error details, got %v", parseErr)
	}
}

// TestValidateEventSlotMetadataRejectsBlankEventType verifies slot declarations require an event type.
func TestValidateEventSlotMetadataRejectsBlankEventType(parseT *testing.T) {
	parseErr := ValidateEventSlotMetadata(EventSlotMetadata{
		Version: EventSlotMetadataVersionV1,
		Slots: []EventSlotRecord{
			{
				SlotID:    "slot-1",
				EventType: "",
			},
		},
	})
	if parseErr == nil {
		parseT.Fatal("expected blank event type to fail")
	}
	if !strings.Contains(parseErr.Error(), "event type is required") {
		parseT.Fatalf("expected event-type error details, got %v", parseErr)
	}
}

// TestValidateEventSlotMetadataRejectsDuplicateSlotEventPair verifies duplicate slot and event declarations fail validation.
func TestValidateEventSlotMetadataRejectsDuplicateSlotEventPair(parseT *testing.T) {
	parseErr := ValidateEventSlotMetadata(EventSlotMetadata{
		Version: EventSlotMetadataVersionV1,
		Slots: []EventSlotRecord{
			{
				SlotID:    "slot-1",
				EventType: "click",
			},
			{
				SlotID:    "slot-1",
				EventType: "click",
			},
		},
	})
	if parseErr == nil {
		parseT.Fatal("expected duplicate slot and event pair to fail")
	}
	if !strings.Contains(parseErr.Error(), "duplicate event-slot declaration") {
		parseT.Fatalf("expected duplicate-slot error details, got %v", parseErr)
	}
}
