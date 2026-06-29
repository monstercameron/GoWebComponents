package cachecore

import (
	"context"
	"sync"
	"testing"
)

// TestNormalizeWorkerEventTypeAllowed verifies the required event types stay canonical.
func TestNormalizeWorkerEventTypeAllowed(parseT *testing.T) {
	parseTypes := []WorkerEventType{
		WorkerEventUpdated,
		WorkerEventStale,
		WorkerEventRefreshing,
		WorkerEventAcked,
		WorkerEventRetryScheduled,
		WorkerEventFailed,
		WorkerEventEvicted,
	}
	for _, parseType := range parseTypes {
		if parseNormalized := normalizeWorkerEventType(parseType); parseNormalized != parseType {
			parseT.Fatalf("normalizeWorkerEventType(%q) = %q, want %q", parseType, parseNormalized, parseType)
		}
	}
	if parseFallback := normalizeWorkerEventType(WorkerEventType("unknown")); parseFallback != WorkerEventUpdated {
		parseT.Fatalf("expected fallback type %q, got %q", WorkerEventUpdated, parseFallback)
	}
}

// TestWorkerEventBusPublishSubscribeUnsubscribe verifies contract fan-out and unsubscribe behavior.
func TestWorkerEventBusPublishSubscribeUnsubscribe(parseT *testing.T) {
	parseBus := BuildWorkerEventBus()
	parseCtx := context.Background()

	parseMu := sync.Mutex{}
	parseCaptured := []WorkerEvent{}
	parseHandler := func(parseCtx context.Context, parseEvent WorkerEvent) {
		if parseCtx == nil {
			parseT.Fatalf("expected context to be provided")
		}
		parseMu.Lock()
		defer parseMu.Unlock()
		parseCaptured = append(parseCaptured, parseEvent)
	}
	parseBus.Subscribe("handler-one", parseHandler)
	parseBus.Subscribe("handler-two", parseHandler)

	parseBus.Publish(parseCtx, WorkerEvent{
		EventType:   WorkerEventType("unknown"),
		ScopeKey:    " scope-a ",
		ResourceKey: " resource-a ",
		QueueKey:    " queue-a ",
		Reason:      " retry ",
	})

	parseMu.Lock()
	if len(parseCaptured) != 2 {
		parseMu.Unlock()
		parseT.Fatalf("expected 2 captured events, got %d", len(parseCaptured))
	}
	parseFirstEvent := parseCaptured[0]
	parseMu.Unlock()
	if parseFirstEvent.EventType != WorkerEventUpdated {
		parseT.Fatalf("expected fallback type %q, got %q", WorkerEventUpdated, parseFirstEvent.EventType)
	}
	if parseFirstEvent.ScopeKey != "scope-a" || parseFirstEvent.ResourceKey != "resource-a" || parseFirstEvent.QueueKey != "queue-a" || parseFirstEvent.Reason != "retry" {
		parseT.Fatalf("expected normalized event fields, got %+v", parseFirstEvent)
	}
	if parseFirstEvent.UpdatedAt == "" {
		parseT.Fatalf("expected UpdatedAt to be populated")
	}

	parseBus.Unsubscribe("handler-two")
	parseBus.Publish(parseCtx, WorkerEvent{EventType: WorkerEventStale})

	parseMu.Lock()
	defer parseMu.Unlock()
	if len(parseCaptured) != 3 {
		parseT.Fatalf("expected 3 captured events after unsubscribe, got %d", len(parseCaptured))
	}
	if parseCaptured[2].EventType != WorkerEventStale {
		parseT.Fatalf("expected stale event after unsubscribe, got %q", parseCaptured[2].EventType)
	}
}
