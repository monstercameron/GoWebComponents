package cachecore

import (
	"strings"
	"sync"
	"time"
)

// InvalidationEvent stores one cache invalidation trigger payload.
type InvalidationEvent struct {
	Reason         string
	ScopePrefix    string
	ResourcePrefix string
	TriggeredAt    string
}

// InvalidationHook handles one invalidation event notification.
type InvalidationHook func(InvalidationEvent)

// SWRCoordinator coordinates stale-while-revalidate refresh de-duplication.
type SWRCoordinator struct {
	parseMu       sync.Mutex
	parseInFlight map[string]struct{}
}

// Invalidator stores named invalidation subscribers for cache and outbox consumers.
type Invalidator struct {
	parseMu    sync.Mutex
	parseHooks map[string]InvalidationHook
}

// BuildSWRCoordinator creates one stale-while-revalidate refresh coordinator.
func BuildSWRCoordinator() *SWRCoordinator {
	return &SWRCoordinator{
		parseInFlight: map[string]struct{}{},
	}
}

// StartRefresh registers one refresh key and returns one release callback when start succeeds.
func (parseCoordinator *SWRCoordinator) StartRefresh(parseScopeKey string, parseResourceKey string) (func(), bool) {
	if parseCoordinator == nil {
		return func() {}, false
	}
	parseScopedKey := BuildScopedResourceKey(parseScopeKey, parseResourceKey)
	if strings.TrimSpace(parseScopedKey) == "" {
		return func() {}, false
	}
	parseCoordinator.parseMu.Lock()
	defer parseCoordinator.parseMu.Unlock()
	if _, isParseExists := parseCoordinator.parseInFlight[parseScopedKey]; isParseExists {
		return func() {}, false
	}
	parseCoordinator.parseInFlight[parseScopedKey] = struct{}{}
	return func() {
		parseCoordinator.finishRefresh(parseScopedKey)
	}, true
}

// finishRefresh clears one in-flight refresh key.
func (parseCoordinator *SWRCoordinator) finishRefresh(parseScopedKey string) {
	if parseCoordinator == nil {
		return
	}
	parseScopedKey = strings.TrimSpace(parseScopedKey)
	if parseScopedKey == "" {
		return
	}
	parseCoordinator.parseMu.Lock()
	defer parseCoordinator.parseMu.Unlock()
	delete(parseCoordinator.parseInFlight, parseScopedKey)
}

// BuildInvalidator creates one invalidation dispatcher.
func BuildInvalidator() *Invalidator {
	return &Invalidator{
		parseHooks: map[string]InvalidationHook{},
	}
}

// SetHook registers one named invalidation hook.
func (parseInvalidator *Invalidator) SetHook(parseHookKey string, parseHook InvalidationHook) {
	if parseInvalidator == nil || parseHook == nil {
		return
	}
	parseHookKey = strings.TrimSpace(parseHookKey)
	if parseHookKey == "" {
		return
	}
	parseInvalidator.parseMu.Lock()
	defer parseInvalidator.parseMu.Unlock()
	parseInvalidator.parseHooks[parseHookKey] = parseHook
}

// DeleteHook removes one named invalidation hook.
func (parseInvalidator *Invalidator) DeleteHook(parseHookKey string) {
	if parseInvalidator == nil {
		return
	}
	parseHookKey = strings.TrimSpace(parseHookKey)
	if parseHookKey == "" {
		return
	}
	parseInvalidator.parseMu.Lock()
	defer parseInvalidator.parseMu.Unlock()
	delete(parseInvalidator.parseHooks, parseHookKey)
}

// ApplyInvalidation broadcasts one invalidation event to registered hooks.
func (parseInvalidator *Invalidator) ApplyInvalidation(parseEvent InvalidationEvent) {
	if parseInvalidator == nil {
		return
	}
	parseEvent.Reason = strings.TrimSpace(parseEvent.Reason)
	if parseEvent.TriggeredAt == "" {
		parseEvent.TriggeredAt = time.Now().UTC().Format(time.RFC3339)
	}
	parseInvalidator.parseMu.Lock()
	parseHooks := make([]InvalidationHook, 0, len(parseInvalidator.parseHooks))
	for _, parseHook := range parseInvalidator.parseHooks {
		parseHooks = append(parseHooks, parseHook)
	}
	parseInvalidator.parseMu.Unlock()
	for _, parseHook := range parseHooks {
		parseHook(parseEvent)
	}
}
