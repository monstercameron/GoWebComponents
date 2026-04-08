package cachecore

import (
	"context"
	"testing"
)

// TestObservabilityEmitterAndSink verifies event emission and sink capture behavior.
func TestObservabilityEmitterAndSink(parseT *testing.T) {
	parseSink := BuildInMemoryObservabilitySink()
	parseEmitter := BuildObservabilityEmitter(parseSink)
	parseEmitter.Emit(context.Background(), ObservabilityEvent{
		EventType:    ObservabilityEventQueueEnqueue,
		ScopeKey:     "scope-1",
		ResourceKey:  "resource-1",
		QueueKey:     "queue-1",
		Reason:       "user_action",
		AttemptCount: 1,
		IsStale:      false,
	})
	parseEvents := parseSink.List()
	if len(parseEvents) != 1 {
		parseT.Fatalf("expected one emitted event, got %d", len(parseEvents))
	}
	parseEvent := parseEvents[0]
	if parseEvent.EventType != ObservabilityEventQueueEnqueue || parseEvent.EmittedAt == "" {
		parseT.Fatalf("unexpected emitted event: %+v", parseEvent)
	}
}

// TestNormalizeObservabilityEventTypeFallback verifies unknown event types normalize safely.
func TestNormalizeObservabilityEventTypeFallback(parseT *testing.T) {
	parseEvent := normalizeObservabilityEvent(ObservabilityEvent{
		EventType:    ObservabilityEventType("unknown"),
		AttemptCount: -10,
	})
	if parseEvent.EventType != ObservabilityEventCacheMiss {
		parseT.Fatalf("expected unknown event type fallback to cache_miss, got %q", parseEvent.EventType)
	}
	if parseEvent.AttemptCount != 0 {
		parseT.Fatalf("expected negative attempt count normalization, got %d", parseEvent.AttemptCount)
	}
}
