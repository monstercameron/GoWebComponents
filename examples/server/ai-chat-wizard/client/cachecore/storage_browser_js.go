//go:build js && wasm

package cachecore

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"syscall/js"
)

const parseBrowserRecordStoragePrefix = "cachecore:record:"
const parseBrowserQueueStoragePrefix = "cachecore:queue:"

// BrowserStorage stores cache/outbox records in browser localStorage.
type BrowserStorage struct {
	parseLocalStorage js.Value
}

// BuildBrowserStorage creates one browser-backed storage implementation.
func BuildBrowserStorage() (*BrowserStorage, error) {
	parseWindow := js.Global().Get("window")
	if parseWindow.IsUndefined() || parseWindow.IsNull() {
		return nil, errors.New("cachecore browser storage: window unavailable")
	}
	parseLocalStorage := parseWindow.Get("localStorage")
	if parseLocalStorage.IsUndefined() || parseLocalStorage.IsNull() {
		return nil, errors.New("cachecore browser storage: localStorage unavailable")
	}
	return &BrowserStorage{
		parseLocalStorage: parseLocalStorage,
	}, nil
}

// Get returns one cache record by scoped resource key from localStorage.
func (parseStorage *BrowserStorage) Get(parseCtx context.Context, parseScopedResourceKey string) (CacheRecordEnvelope, bool, error) {
	_ = parseCtx
	parseScopedResourceKey = strings.TrimSpace(parseScopedResourceKey)
	if parseStorage == nil || parseScopedResourceKey == "" {
		return CacheRecordEnvelope{}, false, nil
	}
	parseRaw := parseStorage.parseLocalStorage.Call("getItem", parseBrowserRecordStoragePrefix+parseScopedResourceKey)
	if parseRaw.IsUndefined() || parseRaw.IsNull() {
		return CacheRecordEnvelope{}, false, nil
	}
	parseRecord, parseErr := ParseCacheRecordEnvelopeJSON([]byte(parseRaw.String()))
	if parseErr != nil {
		return CacheRecordEnvelope{}, false, parseErr
	}
	return parseRecord, true, nil
}

// List returns all cache records matching one scope-prefix filter from localStorage.
func (parseStorage *BrowserStorage) List(parseCtx context.Context, parseScopePrefix string) ([]CacheRecordEnvelope, error) {
	_ = parseCtx
	if parseStorage == nil {
		return []CacheRecordEnvelope{}, nil
	}
	parseScopePrefix = strings.TrimSpace(parseScopePrefix)
	parseRecords := make([]CacheRecordEnvelope, 0)
	parseLength := parseStorage.parseLocalStorage.Get("length").Int()
	for parseIndex := 0; parseIndex < parseLength; parseIndex++ {
		parseStorageKey := parseStorage.parseLocalStorage.Call("key", parseIndex).String()
		if !strings.HasPrefix(parseStorageKey, parseBrowserRecordStoragePrefix) {
			continue
		}
		parseScopedResourceKey := strings.TrimPrefix(parseStorageKey, parseBrowserRecordStoragePrefix)
		if parseScopePrefix != "" && !strings.HasPrefix(parseScopedResourceKey, parseScopePrefix) {
			continue
		}
		parseRaw := parseStorage.parseLocalStorage.Call("getItem", parseStorageKey)
		if parseRaw.IsUndefined() || parseRaw.IsNull() {
			continue
		}
		parseRecord, parseErr := ParseCacheRecordEnvelopeJSON([]byte(parseRaw.String()))
		if parseErr != nil {
			return parseRecords, parseErr
		}
		parseRecords = append(parseRecords, parseRecord)
	}
	return parseRecords, nil
}

// Set stores one normalized cache record in localStorage.
func (parseStorage *BrowserStorage) Set(parseCtx context.Context, parseRecord CacheRecordEnvelope) error {
	_ = parseCtx
	if parseStorage == nil {
		return nil
	}
	parseRecord = NormalizeCacheRecordEnvelope(parseRecord)
	parseScopedResourceKey := BuildScopedResourceKey(parseRecord.ScopeKey, parseRecord.ResourceKey)
	if strings.TrimSpace(parseScopedResourceKey) == "" {
		return nil
	}
	parseRecordJSON, parseErr := BuildCacheRecordEnvelopeJSON(parseRecord)
	if parseErr != nil {
		return parseErr
	}
	parseStorage.parseLocalStorage.Call("setItem", parseBrowserRecordStoragePrefix+parseScopedResourceKey, string(parseRecordJSON))
	return nil
}

// Delete removes one cache record by scoped resource key in localStorage.
func (parseStorage *BrowserStorage) Delete(parseCtx context.Context, parseScopedResourceKey string) error {
	_ = parseCtx
	if parseStorage == nil {
		return nil
	}
	parseScopedResourceKey = strings.TrimSpace(parseScopedResourceKey)
	if parseScopedResourceKey == "" {
		return nil
	}
	parseStorage.parseLocalStorage.Call("removeItem", parseBrowserRecordStoragePrefix+parseScopedResourceKey)
	return nil
}

// DeleteByPrefix removes all cache records matching one scope-prefix filter in localStorage.
func (parseStorage *BrowserStorage) DeleteByPrefix(parseCtx context.Context, parseScopePrefix string) (int, error) {
	_ = parseCtx
	if parseStorage == nil {
		return 0, nil
	}
	parseScopePrefix = strings.TrimSpace(parseScopePrefix)
	parseDeletedCount := 0
	parseStorageKeys := make([]string, 0)
	parseLength := parseStorage.parseLocalStorage.Get("length").Int()
	for parseIndex := 0; parseIndex < parseLength; parseIndex++ {
		parseStorageKey := parseStorage.parseLocalStorage.Call("key", parseIndex).String()
		if !strings.HasPrefix(parseStorageKey, parseBrowserRecordStoragePrefix) {
			continue
		}
		parseScopedResourceKey := strings.TrimPrefix(parseStorageKey, parseBrowserRecordStoragePrefix)
		if parseScopePrefix != "" && !strings.HasPrefix(parseScopedResourceKey, parseScopePrefix) {
			continue
		}
		parseStorageKeys = append(parseStorageKeys, parseStorageKey)
	}
	for _, parseStorageKey := range parseStorageKeys {
		parseStorage.parseLocalStorage.Call("removeItem", parseStorageKey)
		parseDeletedCount++
	}
	return parseDeletedCount, nil
}

// UpdateQueueAtomic applies one queue mutation against one localStorage queue envelope.
func (parseStorage *BrowserStorage) UpdateQueueAtomic(parseCtx context.Context, parseQueueKey string, parseMutate func([]OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error)) ([]OutboxRecordEnvelope, error) {
	_ = parseCtx
	if parseStorage == nil {
		return []OutboxRecordEnvelope{}, nil
	}
	parseQueueKey = strings.TrimSpace(parseQueueKey)
	if parseQueueKey == "" || parseMutate == nil {
		return []OutboxRecordEnvelope{}, nil
	}
	parseStorageKey := parseBrowserQueueStoragePrefix + parseQueueKey
	parseCurrentQueue := []OutboxRecordEnvelope{}
	parseRawQueue := parseStorage.parseLocalStorage.Call("getItem", parseStorageKey)
	if !parseRawQueue.IsUndefined() && !parseRawQueue.IsNull() && strings.TrimSpace(parseRawQueue.String()) != "" {
		if parseErr := json.Unmarshal([]byte(parseRawQueue.String()), &parseCurrentQueue); parseErr != nil {
			return []OutboxRecordEnvelope{}, parseErr
		}
	}
	parseCurrentQueue = parseNormalizeOutboxRecordEnvelopeSlice(parseCurrentQueue)
	parseNextQueue, parseErr := parseMutate(parseCloneOutboxRecordEnvelopeSlice(parseCurrentQueue))
	if parseErr != nil {
		return parseCurrentQueue, parseErr
	}
	parseNextQueue = parseNormalizeOutboxRecordEnvelopeSlice(parseNextQueue)
	parseNextQueueJSON, parseErr := json.Marshal(parseNextQueue)
	if parseErr != nil {
		return parseCurrentQueue, parseErr
	}
	parseStorage.parseLocalStorage.Call("setItem", parseStorageKey, string(parseNextQueueJSON))
	return parseCloneOutboxRecordEnvelopeSlice(parseNextQueue), nil
}
