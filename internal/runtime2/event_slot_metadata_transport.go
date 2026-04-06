package runtime2

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BuildEventSlotMetadataJSON encodes one validated event-slot metadata payload.
func BuildEventSlotMetadataJSON(parseMetadata EventSlotMetadata) ([]byte, error) {
	parseNormalizedMetadata, parseErr := buildEventSlotMetadataNormalized(parseMetadata)
	if parseErr != nil {
		return nil, parseErr
	}
	parsePayload, parseErr := json.Marshal(parseNormalizedMetadata)
	if parseErr != nil {
		return nil, fmt.Errorf("runtime2: encode event-slot metadata payload: %w", parseErr)
	}
	return parsePayload, nil
}

// ParseEventSlotMetadataJSON decodes one validated event-slot metadata payload.
func ParseEventSlotMetadataJSON(parsePayload []byte) (EventSlotMetadata, error) {
	if strings.TrimSpace(string(parsePayload)) == "" {
		return EventSlotMetadata{}, fmt.Errorf("runtime2: event-slot metadata payload is required")
	}
	var parseMetadata EventSlotMetadata
	if parseErr := json.Unmarshal(parsePayload, &parseMetadata); parseErr != nil {
		return EventSlotMetadata{}, fmt.Errorf("runtime2: decode event-slot metadata payload: %w", parseErr)
	}
	parseNormalizedMetadata, parseErr := buildEventSlotMetadataNormalized(parseMetadata)
	if parseErr != nil {
		return EventSlotMetadata{}, parseErr
	}
	return parseNormalizedMetadata, nil
}

// BuildEventSlotMetadataPlaceholderJSON encodes one event-slot metadata payload through the legacy placeholder helper name.
func BuildEventSlotMetadataPlaceholderJSON(parseMetadata EventSlotMetadata) ([]byte, error) {
	return BuildEventSlotMetadataJSON(parseMetadata)
}

// ParseEventSlotMetadataPlaceholderJSON decodes one event-slot metadata payload through the legacy placeholder helper name.
func ParseEventSlotMetadataPlaceholderJSON(parsePayload []byte) (EventSlotMetadata, error) {
	return ParseEventSlotMetadataJSON(parsePayload)
}
