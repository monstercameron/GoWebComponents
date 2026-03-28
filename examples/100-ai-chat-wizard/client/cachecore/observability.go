package cachecore

import (
	"context"
	"strings"
	"sync"
	"time"
)

// ObservabilityEventType identifies one cache/outbox lifecycle telemetry event.
type ObservabilityEventType string

const (
	ObservabilityEventCacheHit         ObservabilityEventType = "cache_hit"
	ObservabilityEventCacheMiss        ObservabilityEventType = "cache_miss"
	ObservabilityEventCacheStaleReturn ObservabilityEventType = "cache_stale_return"
	ObservabilityEventRefreshStart     ObservabilityEventType = "refresh_start"
	ObservabilityEventRefreshFailure   ObservabilityEventType = "refresh_failure"
	ObservabilityEventQueueEnqueue     ObservabilityEventType = "queue_enqueue"
	ObservabilityEventQueueAck         ObservabilityEventType = "queue_ack"
	ObservabilityEventQueueDrop        ObservabilityEventType = "queue_drop"
	ObservabilityEventEviction         ObservabilityEventType = "eviction"
)

// ObservabilityEvent stores one payload-safe diagnostics event.
type ObservabilityEvent struct {
	EventType    ObservabilityEventType
	ScopeKey     string
	ResourceKey  string
	QueueKey     string
	Reason       string
	ErrorCode    string
	AttemptCount int
	IsStale      bool
	EmittedAt    string
}

// ObservabilitySink receives payload-safe observability events.
type ObservabilitySink interface {
	Emit(context.Context, ObservabilityEvent)
}

// ObservabilityEmitter emits normalized observability events to one sink.
type ObservabilityEmitter struct {
	parseSink ObservabilitySink
}

// InMemoryObservabilitySink stores emitted events for tests and diagnostics.
type InMemoryObservabilitySink struct {
	parseMu     sync.Mutex
	parseEvents []ObservabilityEvent
}

// BuildObservabilityEmitter creates one event emitter bound to one sink.
func BuildObservabilityEmitter(parseSink ObservabilitySink) *ObservabilityEmitter {
	return &ObservabilityEmitter{parseSink: parseSink}
}

// Emit sends one normalized observability event to the configured sink.
func (parseEmitter *ObservabilityEmitter) Emit(parseCtx context.Context, parseEvent ObservabilityEvent) {
	if parseEmitter == nil || parseEmitter.parseSink == nil {
		return
	}
	parseEmitter.parseSink.Emit(parseCtx, normalizeObservabilityEvent(parseEvent))
}

// BuildInMemoryObservabilitySink creates one in-memory sink for testing.
func BuildInMemoryObservabilitySink() *InMemoryObservabilitySink {
	return &InMemoryObservabilitySink{
		parseEvents: []ObservabilityEvent{},
	}
}

// Emit appends one normalized event to in-memory sink state.
func (parseSink *InMemoryObservabilitySink) Emit(parseCtx context.Context, parseEvent ObservabilityEvent) {
	_ = parseCtx
	if parseSink == nil {
		return
	}
	parseSink.parseMu.Lock()
	defer parseSink.parseMu.Unlock()
	parseSink.parseEvents = append(parseSink.parseEvents, normalizeObservabilityEvent(parseEvent))
}

// List returns one snapshot of emitted in-memory events.
func (parseSink *InMemoryObservabilitySink) List() []ObservabilityEvent {
	if parseSink == nil {
		return []ObservabilityEvent{}
	}
	parseSink.parseMu.Lock()
	defer parseSink.parseMu.Unlock()
	parseEvents := make([]ObservabilityEvent, 0, len(parseSink.parseEvents))
	parseEvents = append(parseEvents, parseSink.parseEvents...)
	return parseEvents
}

// normalizeObservabilityEvent normalizes one observability event to payload-safe metadata fields.
func normalizeObservabilityEvent(parseEvent ObservabilityEvent) ObservabilityEvent {
	parseEvent.ScopeKey = strings.TrimSpace(parseEvent.ScopeKey)
	parseEvent.ResourceKey = strings.TrimSpace(parseEvent.ResourceKey)
	parseEvent.QueueKey = strings.TrimSpace(parseEvent.QueueKey)
	parseEvent.Reason = strings.TrimSpace(parseEvent.Reason)
	parseEvent.ErrorCode = strings.TrimSpace(parseEvent.ErrorCode)
	if parseEvent.AttemptCount < 0 {
		parseEvent.AttemptCount = 0
	}
	parseEvent.EmittedAt = strings.TrimSpace(parseEvent.EmittedAt)
	if parseEvent.EmittedAt == "" {
		parseEvent.EmittedAt = time.Now().UTC().Format(time.RFC3339)
	}
	parseEvent.EventType = normalizeObservabilityEventType(parseEvent.EventType)
	return parseEvent
}

// normalizeObservabilityEventType resolves one canonical event type fallback.
func normalizeObservabilityEventType(parseEventType ObservabilityEventType) ObservabilityEventType {
	switch parseEventType {
	case ObservabilityEventCacheHit,
		ObservabilityEventCacheMiss,
		ObservabilityEventCacheStaleReturn,
		ObservabilityEventRefreshStart,
		ObservabilityEventRefreshFailure,
		ObservabilityEventQueueEnqueue,
		ObservabilityEventQueueAck,
		ObservabilityEventQueueDrop,
		ObservabilityEventEviction:
		return parseEventType
	default:
		return ObservabilityEventCacheMiss
	}
}
