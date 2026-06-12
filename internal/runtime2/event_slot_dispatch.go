package runtime2

import "fmt"

// EventSlotDispatch stores one semantic event dispatch for one declared event slot.
type EventSlotDispatch struct {
	SlotID    string `json:"slot_id,omitempty"`
	EventType string `json:"event_type,omitempty"`
	Payload   any    `json:"payload,omitempty"`
}

// ValidateEventSlotDispatch verifies one semantic event dispatch uses a declared-slot compatible shape.
func ValidateEventSlotDispatch(parseDispatch EventSlotDispatch) error {
	_, parseErr := buildEventSlotDispatchNormalized(parseDispatch)
	return parseErr
}

// buildEventSlotDispatchNormalized validates one semantic event dispatch and returns the normalized dispatch payload.
func buildEventSlotDispatchNormalized(parseDispatch EventSlotDispatch) (EventSlotDispatch, error) {
	parseNormalizedRecord, parseRecordErr := buildEventSlotRecordNormalized(EventSlotRecord{
		SlotID:    parseDispatch.SlotID,
		EventType: parseDispatch.EventType,
	})
	if parseRecordErr != nil {
		return EventSlotDispatch{}, fmt.Errorf("runtime2: event-slot dispatch is invalid: %w", parseRecordErr)
	}
	if parsePayloadErr := ValidateSerializableProps(parseDispatch.Payload); parsePayloadErr != nil {
		return EventSlotDispatch{}, fmt.Errorf("runtime2: event-slot payload is not serializable: %w", parsePayloadErr)
	}
	return EventSlotDispatch{
		SlotID:    parseNormalizedRecord.SlotID,
		EventType: parseNormalizedRecord.EventType,
		Payload:   parseDispatch.Payload,
	}, nil
}

// buildEventSlotDispatchRecord converts one normalized event dispatch into the equivalent slot declaration record.
func buildEventSlotDispatchRecord(parseDispatch EventSlotDispatch) EventSlotRecord {
	return EventSlotRecord{
		SlotID:    parseDispatch.SlotID,
		EventType: parseDispatch.EventType,
	}
}

// hasEventSlotMetadataDispatch reports whether one normalized event dispatch is declared in renderer metadata.
func hasEventSlotMetadataDispatch(parseMetadata EventSlotMetadata, parseDispatch EventSlotDispatch) bool {
	parseDispatchKey := buildEventSlotRecordKey(buildEventSlotDispatchRecord(parseDispatch))
	for _, parseSlot := range parseMetadata.Slots {
		if buildEventSlotRecordKey(parseSlot) == parseDispatchKey {
			return true
		}
	}
	return false
}
