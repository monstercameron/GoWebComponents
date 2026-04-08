package cachecore

import (
	"context"
	"strings"
	"sync"
)

// SnapshotStore stores one main-thread mirror of cache records for synchronous render reads.
type SnapshotStore struct {
	parseMu      sync.RWMutex
	parseRecords map[string]CacheRecordEnvelope
}

// BuildSnapshotStore creates one main-thread snapshot store.
func BuildSnapshotStore() *SnapshotStore {
	return &SnapshotStore{
		parseRecords: map[string]CacheRecordEnvelope{},
	}
}

// GetSync returns one mirrored cache record synchronously for one scoped resource key.
func (parseStore *SnapshotStore) GetSync(parseScopeKey string, parseResourceKey string) (CacheRecordEnvelope, bool) {
	if parseStore == nil {
		return CacheRecordEnvelope{}, false
	}
	parseScopedResourceKey := BuildScopedResourceKey(parseScopeKey, parseResourceKey)
	parseStore.parseMu.RLock()
	defer parseStore.parseMu.RUnlock()
	parseRecord, isParseFound := parseStore.parseRecords[parseScopedResourceKey]
	if !isParseFound {
		return CacheRecordEnvelope{}, false
	}
	return NormalizeCacheRecordEnvelope(parseRecord), true
}

// ListSync returns one snapshot list for one scope prefix synchronously.
func (parseStore *SnapshotStore) ListSync(parseScopePrefix string) []CacheRecordEnvelope {
	if parseStore == nil {
		return []CacheRecordEnvelope{}
	}
	parseScopePrefix = strings.TrimSpace(parseScopePrefix)
	parseStore.parseMu.RLock()
	defer parseStore.parseMu.RUnlock()
	parseRecords := make([]CacheRecordEnvelope, 0, len(parseStore.parseRecords))
	for parseScopedResourceKey, parseRecord := range parseStore.parseRecords {
		if parseScopePrefix != "" && !strings.HasPrefix(parseScopedResourceKey, parseScopePrefix) {
			continue
		}
		parseRecords = append(parseRecords, NormalizeCacheRecordEnvelope(parseRecord))
	}
	return parseRecords
}

// SetSync writes one mirrored cache record synchronously.
func (parseStore *SnapshotStore) SetSync(parseRecord CacheRecordEnvelope) {
	if parseStore == nil {
		return
	}
	parseRecord = NormalizeCacheRecordEnvelope(parseRecord)
	parseScopedResourceKey := BuildScopedResourceKey(parseRecord.ScopeKey, parseRecord.ResourceKey)
	if strings.TrimSpace(parseScopedResourceKey) == "" {
		return
	}
	parseStore.parseMu.Lock()
	defer parseStore.parseMu.Unlock()
	parseStore.parseRecords[parseScopedResourceKey] = parseRecord
}

// DeleteSync removes one mirrored cache record synchronously.
func (parseStore *SnapshotStore) DeleteSync(parseScopeKey string, parseResourceKey string) {
	if parseStore == nil {
		return
	}
	parseScopedResourceKey := BuildScopedResourceKey(parseScopeKey, parseResourceKey)
	if strings.TrimSpace(parseScopedResourceKey) == "" {
		return
	}
	parseStore.parseMu.Lock()
	defer parseStore.parseMu.Unlock()
	delete(parseStore.parseRecords, parseScopedResourceKey)
}

// ApplyFromStorage mirrors records from one storage implementation into snapshot state.
func (parseStore *SnapshotStore) ApplyFromStorage(parseCtx context.Context, parseStorage Storage, parseScopePrefix string) error {
	if parseStore == nil || parseStorage == nil {
		return nil
	}
	parseRecords, parseErr := parseStorage.List(parseCtx, strings.TrimSpace(parseScopePrefix))
	if parseErr != nil {
		return parseErr
	}
	for _, parseRecord := range parseRecords {
		parseStore.SetSync(parseRecord)
	}
	return nil
}
