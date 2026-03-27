package runtime2

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BuildEventSlotMetadataPlaceholderJSON encodes one placeholder event-slot metadata payload.
func BuildEventSlotMetadataPlaceholderJSON(parseMetadata EventSlotMetadata) ([]byte, error) {
	if parseErr := ValidateEventSlotMetadata(parseMetadata); parseErr != nil {
		return nil, parseErr
	}
	parsePayload, parseErr := json.Marshal(parseMetadata)
	if parseErr != nil {
		return nil, fmt.Errorf("runtime2: encode event-slot metadata placeholder payload: %w", parseErr)
	}
	return parsePayload, nil
}

// ParseEventSlotMetadataPlaceholderJSON decodes one placeholder event-slot metadata payload.
func ParseEventSlotMetadataPlaceholderJSON(parsePayload []byte) (EventSlotMetadata, error) {
	if strings.TrimSpace(string(parsePayload)) == "" {
		return EventSlotMetadata{}, fmt.Errorf("runtime2: event-slot metadata placeholder payload is required")
	}
	var parseMetadata EventSlotMetadata
	if parseErr := json.Unmarshal(parsePayload, &parseMetadata); parseErr != nil {
		return EventSlotMetadata{}, fmt.Errorf("runtime2: decode event-slot metadata placeholder payload: %w", parseErr)
	}
	if parseErr := ValidateEventSlotMetadata(parseMetadata); parseErr != nil {
		return EventSlotMetadata{}, parseErr
	}
	return parseMetadata, nil
}
