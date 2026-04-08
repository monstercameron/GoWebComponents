package cachecore

import (
	"context"
	"strings"
	"sync"
	"time"
)

// WorkerEventType identifies one worker-to-main cache/outbox maintenance signal.
type WorkerEventType string

const (
	WorkerEventUpdated        WorkerEventType = "updated"
	WorkerEventStale          WorkerEventType = "stale"
	WorkerEventRefreshing     WorkerEventType = "refreshing"
	WorkerEventAcked          WorkerEventType = "acked"
	WorkerEventRetryScheduled WorkerEventType = "retry_scheduled"
	WorkerEventFailed         WorkerEventType = "failed"
	WorkerEventEvicted        WorkerEventType = "evicted"
)

// WorkerEvent stores one payload-agnostic maintenance event.
type WorkerEvent struct {
	EventType    WorkerEventType
	ScopeKey     string
	ResourceKey  string
	QueueKey     string
	OperationKey string
	Reason       string
	ErrorCode    string
	UpdatedAt    string
}

// WorkerEventHandler handles one worker maintenance event callback.
type WorkerEventHandler func(context.Context, WorkerEvent)

// WorkerEventContract defines the worker event publish/subscribe API.
type WorkerEventContract interface {
	Publish(context.Context, WorkerEvent)
	Subscribe(string, WorkerEventHandler)
	Unsubscribe(string)
}

// WorkerEventBus provides one in-process worker event contract implementation.
type WorkerEventBus struct {
	parseMu       sync.Mutex
	parseHandlers map[string]WorkerEventHandler
}

// BuildWorkerEventBus creates one worker event bus.
func BuildWorkerEventBus() *WorkerEventBus {
	return &WorkerEventBus{
		parseHandlers: map[string]WorkerEventHandler{},
	}
}

// Publish emits one normalized worker event to all subscribers.
func (parseBus *WorkerEventBus) Publish(parseCtx context.Context, parseEvent WorkerEvent) {
	if parseBus == nil {
		return
	}
	parseEvent = normalizeWorkerEvent(parseEvent)
	parseBus.parseMu.Lock()
	parseHandlers := make([]WorkerEventHandler, 0, len(parseBus.parseHandlers))
	for _, parseHandler := range parseBus.parseHandlers {
		parseHandlers = append(parseHandlers, parseHandler)
	}
	parseBus.parseMu.Unlock()
	for _, parseHandler := range parseHandlers {
		parseHandler(parseCtx, parseEvent)
	}
}

// Subscribe registers one named worker event handler.
func (parseBus *WorkerEventBus) Subscribe(parseHandlerKey string, parseHandler WorkerEventHandler) {
	if parseBus == nil || parseHandler == nil {
		return
	}
	parseHandlerKey = strings.TrimSpace(parseHandlerKey)
	if parseHandlerKey == "" {
		return
	}
	parseBus.parseMu.Lock()
	defer parseBus.parseMu.Unlock()
	parseBus.parseHandlers[parseHandlerKey] = parseHandler
}

// Unsubscribe removes one named worker event handler.
func (parseBus *WorkerEventBus) Unsubscribe(parseHandlerKey string) {
	if parseBus == nil {
		return
	}
	parseHandlerKey = strings.TrimSpace(parseHandlerKey)
	if parseHandlerKey == "" {
		return
	}
	parseBus.parseMu.Lock()
	defer parseBus.parseMu.Unlock()
	delete(parseBus.parseHandlers, parseHandlerKey)
}

// normalizeWorkerEvent normalizes one worker event shape.
func normalizeWorkerEvent(parseEvent WorkerEvent) WorkerEvent {
	parseEvent.ScopeKey = strings.TrimSpace(parseEvent.ScopeKey)
	parseEvent.ResourceKey = strings.TrimSpace(parseEvent.ResourceKey)
	parseEvent.QueueKey = strings.TrimSpace(parseEvent.QueueKey)
	parseEvent.OperationKey = strings.TrimSpace(parseEvent.OperationKey)
	parseEvent.Reason = strings.TrimSpace(parseEvent.Reason)
	parseEvent.ErrorCode = strings.TrimSpace(parseEvent.ErrorCode)
	parseEvent.UpdatedAt = strings.TrimSpace(parseEvent.UpdatedAt)
	if parseEvent.UpdatedAt == "" {
		parseEvent.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	parseEvent.EventType = normalizeWorkerEventType(parseEvent.EventType)
	return parseEvent
}

// normalizeWorkerEventType resolves one canonical worker event type fallback.
func normalizeWorkerEventType(parseEventType WorkerEventType) WorkerEventType {
	switch parseEventType {
	case WorkerEventUpdated,
		WorkerEventStale,
		WorkerEventRefreshing,
		WorkerEventAcked,
		WorkerEventRetryScheduled,
		WorkerEventFailed,
		WorkerEventEvicted:
		return parseEventType
	default:
		return WorkerEventUpdated
	}
}
