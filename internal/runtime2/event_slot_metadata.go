package runtime2

import (
	"fmt"
	"strings"
)

// EventSlotMetadataVersion identifies one placeholder event-slot metadata schema version.
type EventSlotMetadataVersion string

const (
	// EventSlotMetadataVersionPlaceholderV1 identifies the slice-one non-operative placeholder event-slot schema.
	EventSlotMetadataVersionPlaceholderV1 EventSlotMetadataVersion = "gwc.parallel.event-slot.placeholder.v1"
)

// EventSlotMetadata stores optional placeholder event-slot metadata for future interactive slices.
type EventSlotMetadata struct {
	Version EventSlotMetadataVersion `json:"version,omitempty"`
	Slots   []EventSlotRecord        `json:"slots,omitempty"`
}

// EventSlotRecord stores one placeholder event-slot declaration.
type EventSlotRecord struct {
	SlotID    string `json:"slot_id,omitempty"`
	EventType string `json:"event_type,omitempty"`
}

// ValidateEventSlotMetadata verifies placeholder event-slot metadata shape and version rules.
func ValidateEventSlotMetadata(parseMetadata EventSlotMetadata) error {
	parseVersion := string(parseMetadata.Version)
	parseTrimmedVersion := strings.TrimSpace(parseVersion)
	if parseTrimmedVersion != parseVersion {
		return fmt.Errorf("runtime2: event-slot metadata version must not contain surrounding whitespace")
	}
	if parseTrimmedVersion == "" {
		if len(parseMetadata.Slots) == 0 {
			return nil
		}
		return fmt.Errorf("runtime2: event-slot metadata active slots are unsupported in first-slice runtime2")
	}
	if parseMetadata.Version != EventSlotMetadataVersionPlaceholderV1 {
		return fmt.Errorf("runtime2: event-slot metadata version %q is unsupported", parseMetadata.Version)
	}
	if len(parseMetadata.Slots) > 0 {
		return fmt.Errorf("runtime2: event-slot metadata active slots are unsupported in first-slice runtime2")
	}
	return nil
}
