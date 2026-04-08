package cachecore

import (
	"context"
	"strings"
	"sync"
)

// Storage defines the shared cache/outbox persistence contract for client runtimes.
type Storage interface {
	Get(parseCtx context.Context, parseScopedResourceKey string) (CacheRecordEnvelope, bool, error)
	List(parseCtx context.Context, parseScopePrefix string) ([]CacheRecordEnvelope, error)
	Set(parseCtx context.Context, parseRecord CacheRecordEnvelope) error
	Delete(parseCtx context.Context, parseScopedResourceKey string) error
	DeleteByPrefix(parseCtx context.Context, parseScopePrefix string) (int, error)
	UpdateQueueAtomic(parseCtx context.Context, parseQueueKey string, parseMutate func([]OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error)) ([]OutboxRecordEnvelope, error)
}

// MemoryStorage stores cache/outbox records in-process for tests and non-persistent runtimes.
type MemoryStorage struct {
	parseMu      sync.RWMutex
	parseRecords map[string]CacheRecordEnvelope
	parseQueues  map[string][]OutboxRecordEnvelope
}

// BuildMemoryStorage creates one in-memory cache/outbox storage implementation.
func BuildMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		parseRecords: map[string]CacheRecordEnvelope{},
		parseQueues:  map[string][]OutboxRecordEnvelope{},
	}
}

// Get returns one cache record by scoped resource key.
func (parseStorage *MemoryStorage) Get(parseCtx context.Context, parseScopedResourceKey string) (CacheRecordEnvelope, bool, error) {
	_ = parseCtx
	if parseStorage == nil {
		return CacheRecordEnvelope{}, false, nil
	}
	parseScopedResourceKey = strings.TrimSpace(parseScopedResourceKey)
	if parseScopedResourceKey == "" {
		return CacheRecordEnvelope{}, false, nil
	}
	parseStorage.parseMu.RLock()
	defer parseStorage.parseMu.RUnlock()
	parseRecord, isParseFound := parseStorage.parseRecords[parseScopedResourceKey]
	if !isParseFound {
		return CacheRecordEnvelope{}, false, nil
	}
	return NormalizeCacheRecordEnvelope(parseRecord), true, nil
}

// List returns all cache records matching one scope-prefix filter.
func (parseStorage *MemoryStorage) List(parseCtx context.Context, parseScopePrefix string) ([]CacheRecordEnvelope, error) {
	_ = parseCtx
	if parseStorage == nil {
		return []CacheRecordEnvelope{}, nil
	}
	parseScopePrefix = strings.TrimSpace(parseScopePrefix)
	parseStorage.parseMu.RLock()
	defer parseStorage.parseMu.RUnlock()
	parseRecords := make([]CacheRecordEnvelope, 0, len(parseStorage.parseRecords))
	for parseScopedResourceKey, parseRecord := range parseStorage.parseRecords {
		if parseScopePrefix != "" && !strings.HasPrefix(parseScopedResourceKey, parseScopePrefix) {
			continue
		}
		parseRecords = append(parseRecords, NormalizeCacheRecordEnvelope(parseRecord))
	}
	return parseRecords, nil
}

// Set stores one normalized cache record.
func (parseStorage *MemoryStorage) Set(parseCtx context.Context, parseRecord CacheRecordEnvelope) error {
	_ = parseCtx
	if parseStorage == nil {
		return nil
	}
	parseRecord = NormalizeCacheRecordEnvelope(parseRecord)
	parseScopedResourceKey := BuildScopedResourceKey(parseRecord.ScopeKey, parseRecord.ResourceKey)
	if strings.TrimSpace(parseScopedResourceKey) == "" {
		return nil
	}
	parseStorage.parseMu.Lock()
	defer parseStorage.parseMu.Unlock()
	parseStorage.parseRecords[parseScopedResourceKey] = parseRecord
	return nil
}

// Delete removes one cache record by scoped resource key.
func (parseStorage *MemoryStorage) Delete(parseCtx context.Context, parseScopedResourceKey string) error {
	_ = parseCtx
	if parseStorage == nil {
		return nil
	}
	parseScopedResourceKey = strings.TrimSpace(parseScopedResourceKey)
	if parseScopedResourceKey == "" {
		return nil
	}
	parseStorage.parseMu.Lock()
	defer parseStorage.parseMu.Unlock()
	delete(parseStorage.parseRecords, parseScopedResourceKey)
	return nil
}

// DeleteByPrefix removes all cache records matching one scoped-prefix filter.
func (parseStorage *MemoryStorage) DeleteByPrefix(parseCtx context.Context, parseScopePrefix string) (int, error) {
	_ = parseCtx
	if parseStorage == nil {
		return 0, nil
	}
	parseScopePrefix = strings.TrimSpace(parseScopePrefix)
	parseStorage.parseMu.Lock()
	defer parseStorage.parseMu.Unlock()
	parseDeletedCount := 0
	for parseScopedResourceKey := range parseStorage.parseRecords {
		if parseScopePrefix != "" && !strings.HasPrefix(parseScopedResourceKey, parseScopePrefix) {
			continue
		}
		delete(parseStorage.parseRecords, parseScopedResourceKey)
		parseDeletedCount++
	}
	return parseDeletedCount, nil
}

// UpdateQueueAtomic applies one atomic queue mutation and returns the final queue snapshot.
func (parseStorage *MemoryStorage) UpdateQueueAtomic(parseCtx context.Context, parseQueueKey string, parseMutate func([]OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error)) ([]OutboxRecordEnvelope, error) {
	_ = parseCtx
	if parseStorage == nil {
		return []OutboxRecordEnvelope{}, nil
	}
	parseQueueKey = strings.TrimSpace(parseQueueKey)
	if parseQueueKey == "" || parseMutate == nil {
		return []OutboxRecordEnvelope{}, nil
	}
	parseStorage.parseMu.Lock()
	defer parseStorage.parseMu.Unlock()
	parseCurrentQueue := parseCloneOutboxRecordEnvelopeSlice(parseStorage.parseQueues[parseQueueKey])
	parseNextQueue, parseErr := parseMutate(parseCurrentQueue)
	if parseErr != nil {
		return parseCurrentQueue, parseErr
	}
	parseStorage.parseQueues[parseQueueKey] = parseNormalizeOutboxRecordEnvelopeSlice(parseNextQueue)
	return parseCloneOutboxRecordEnvelopeSlice(parseStorage.parseQueues[parseQueueKey]), nil
}

// parseNormalizeOutboxRecordEnvelopeSlice normalizes one outbox envelope slice.
func parseNormalizeOutboxRecordEnvelopeSlice(parseRecords []OutboxRecordEnvelope) []OutboxRecordEnvelope {
	if len(parseRecords) == 0 {
		return []OutboxRecordEnvelope{}
	}
	parseNormalized := make([]OutboxRecordEnvelope, 0, len(parseRecords))
	for _, parseRecord := range parseRecords {
		parseNormalized = append(parseNormalized, NormalizeOutboxRecordEnvelope(parseRecord))
	}
	return parseNormalized
}

// parseCloneOutboxRecordEnvelopeSlice clones one outbox envelope slice.
func parseCloneOutboxRecordEnvelopeSlice(parseRecords []OutboxRecordEnvelope) []OutboxRecordEnvelope {
	if len(parseRecords) == 0 {
		return []OutboxRecordEnvelope{}
	}
	parseClone := make([]OutboxRecordEnvelope, 0, len(parseRecords))
	for _, parseRecord := range parseRecords {
		parseClone = append(parseClone, NormalizeOutboxRecordEnvelope(parseRecord))
	}
	return parseClone
}
