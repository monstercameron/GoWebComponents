package runtime2

import (
	"fmt"
	"strings"
)

// EventSlotMetadataVersion identifies one event-slot metadata schema version.
type EventSlotMetadataVersion string

const (
	// EventSlotMetadataVersionV1 identifies the current event-slot metadata schema.
	EventSlotMetadataVersionV1 EventSlotMetadataVersion = "gwc.parallel.event-slot.v1"
	// EventSlotMetadataVersionPlaceholderV1 is the legacy placeholder schema identifier accepted on decode and normalized to v1.
	EventSlotMetadataVersionPlaceholderV1 EventSlotMetadataVersion = "gwc.parallel.event-slot.placeholder.v1"
)

// EventSlotMetadata stores optional event-slot metadata for renderer capability declarations.
type EventSlotMetadata struct {
	Version EventSlotMetadataVersion `json:"version,omitempty"`
	Slots   []EventSlotRecord        `json:"slots,omitempty"`
}

// EventSlotRecord stores one event-slot declaration.
type EventSlotRecord struct {
	SlotID    string `json:"slot_id,omitempty"`
	EventType string `json:"event_type,omitempty"`
}

// ValidateEventSlotMetadata verifies event-slot metadata shape and version rules.
func ValidateEventSlotMetadata(parseMetadata EventSlotMetadata) error {
	_, parseErr := buildEventSlotMetadataNormalized(parseMetadata)
	return parseErr
}

// buildEventSlotMetadataNormalized validates one event-slot metadata payload and returns the normalized v1 shape.
func buildEventSlotMetadataNormalized(parseMetadata EventSlotMetadata) (EventSlotMetadata, error) {
	parseVersion := string(parseMetadata.Version)
	parseTrimmedVersion := strings.TrimSpace(parseVersion)
	if parseTrimmedVersion != parseVersion {
		return EventSlotMetadata{}, fmt.Errorf("runtime2: event-slot metadata version must not contain surrounding whitespace")
	}
	if parseTrimmedVersion == "" {
		if len(parseMetadata.Slots) == 0 {
			return EventSlotMetadata{}, nil
		}
		return EventSlotMetadata{}, fmt.Errorf("runtime2: event-slot metadata version is required when slots are declared")
	}
	switch parseMetadata.Version {
	case EventSlotMetadataVersionV1, EventSlotMetadataVersionPlaceholderV1:
	default:
		return EventSlotMetadata{}, fmt.Errorf("runtime2: event-slot metadata version %q is unsupported", parseMetadata.Version)
	}
	parseNormalizedSlots := make([]EventSlotRecord, 0, len(parseMetadata.Slots))
	parseSeenSlotKeys := make(map[string]bool, len(parseMetadata.Slots))
	for parseIndex, parseSlot := range parseMetadata.Slots {
		parseNormalizedSlot, parseSlotErr := buildEventSlotRecordNormalized(parseSlot)
		if parseSlotErr != nil {
			return EventSlotMetadata{}, fmt.Errorf("runtime2: event-slot metadata slot %d is invalid: %w", parseIndex, parseSlotErr)
		}
		parseSlotKey := buildEventSlotRecordKey(parseNormalizedSlot)
		if parseSeenSlotKeys[parseSlotKey] {
			return EventSlotMetadata{}, fmt.Errorf(
				"runtime2: duplicate event-slot declaration for slot %q and event %q",
				parseNormalizedSlot.SlotID,
				parseNormalizedSlot.EventType,
			)
		}
		parseSeenSlotKeys[parseSlotKey] = true
		parseNormalizedSlots = append(parseNormalizedSlots, parseNormalizedSlot)
	}
	return EventSlotMetadata{
		Version: EventSlotMetadataVersionV1,
		Slots:   parseNormalizedSlots,
	}, nil
}

// buildEventSlotRecordNormalized validates one event-slot declaration and returns the normalized slot record.
func buildEventSlotRecordNormalized(parseSlot EventSlotRecord) (EventSlotRecord, error) {
	parseTrimmedSlotID := strings.TrimSpace(parseSlot.SlotID)
	if parseTrimmedSlotID == "" {
		return EventSlotRecord{}, fmt.Errorf("slot ID is required")
	}
	if parseTrimmedSlotID != parseSlot.SlotID {
		return EventSlotRecord{}, fmt.Errorf("slot ID must not contain surrounding whitespace")
	}
	parseTrimmedEventType := strings.TrimSpace(parseSlot.EventType)
	if parseTrimmedEventType == "" {
		return EventSlotRecord{}, fmt.Errorf("event type is required")
	}
	if parseTrimmedEventType != parseSlot.EventType {
		return EventSlotRecord{}, fmt.Errorf("event type must not contain surrounding whitespace")
	}
	return EventSlotRecord{
		SlotID:    parseTrimmedSlotID,
		EventType: parseTrimmedEventType,
	}, nil
}

// buildEventSlotRecordKey builds the duplicate-detection key for one normalized event-slot declaration.
func buildEventSlotRecordKey(parseSlot EventSlotRecord) string {
	return parseSlot.SlotID + "\x00" + parseSlot.EventType
}
