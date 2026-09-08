package fetch

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/interop"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

const CacheBootstrapDataKey = "fetchCache"

type PersistentCacheOptions struct {
	DatabaseName     string
	StoreName        string
	FallbackResolver func() (interop.Storage, error)
	FallbackBackend  string
	StoreResolver    func(context.Context) (interop.PersistentStore, error)
}

type CacheBootstrapEntry struct {
	Key          string            `json:"key,omitempty"`
	Value        any               `json:"value,omitempty"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	ResumePolicy CacheResumePolicy `json:"resumePolicy,omitempty"`
	StaleAfter   time.Duration     `json:"staleAfter,omitempty"`
}

type CacheBootstrap struct {
	Entries []CacheBootstrapEntry `json:"entries,omitempty"`
}

type persistedCachedResource struct {
	Value      json.RawMessage `json:"value,omitempty"`
	UpdatedAt  time.Time       `json:"updatedAt"`
	LastLoaded time.Time       `json:"lastLoaded"`
}

var persistentCacheState = struct {
	mu      sync.Mutex
	options PersistentCacheOptions
	opened  bool
	store   interop.PersistentStore
	err     error
}{}

// RestoreCacheBootstrap seeds shared cached resources from a UI bootstrap payload.
func RestoreCacheBootstrap(parsePayload ui.SSRBootstrap) error {
	parseBootstrap, parseErr := readCacheBootstrap(parsePayload.Data)
	if parseErr != nil {
		return parseErr
	}
	restoreCacheBootstrapEntries(parseBootstrap.Entries)
	return nil
}

// ConfigurePersistentCache sets the options for the persistent cache store, closing any existing store.
func ConfigurePersistentCache(parseOptions PersistentCacheOptions) {
	persistentCacheState.mu.Lock()
	store := persistentCacheState.store
	parseOpened := persistentCacheState.opened
	persistentCacheState.options = parseOptions
	persistentCacheState.store = interop.PersistentStore{}
	persistentCacheState.err = nil
	persistentCacheState.opened = false
	persistentCacheState.mu.Unlock()
	if parseOpened {
		if parseCloseErr := store.Close(); parseCloseErr != nil {
			runtime.ReportLogWithFields("fetch", runtime.LogWarn, runtime.DiagnosticRecovered, "persistent cache store close failed", "", map[string]string{
				"message": parseCloseErr.Error(),
			})
		}
	}
}

// readCacheBootstrap is an internal cache helper.
func readCacheBootstrap(parseData map[string]any) (CacheBootstrap, error) {
	if len(parseData) == 0 {
		return CacheBootstrap{}, nil
	}
	parseRaw, parseOk := parseData[CacheBootstrapDataKey]
	if !parseOk || parseRaw == nil {
		return CacheBootstrap{}, nil
	}
	parseEncoded, parseErr := json.Marshal(parseRaw)
	if parseErr != nil {
		return CacheBootstrap{}, parseErr
	}
	var parseBootstrap CacheBootstrap
	if parseErr2 := json.Unmarshal(parseEncoded, &parseBootstrap); parseErr2 != nil {
		return CacheBootstrap{}, parseErr2
	}
	return parseBootstrap, nil
}

// restoreCacheBootstrapEntries is an internal cache helper.
func restoreCacheBootstrapEntries(parseEntries []CacheBootstrapEntry) {
	parseNow := time.Now()
	for _, parseItem := range parseEntries {
		parseKey := strings.TrimSpace(parseItem.Key)
		if parseKey == "" {
			continue
		}
		parseEntry := getCachedResourceEntry(parseKey)
		parseEntry.mu.Lock()
		parseEntry.lastLoaded = parseItem.UpdatedAt
		parseEntry.lastAccess = parseNow
		parseEntry.resumePolicy = normalizeResumePolicy(parseItem.ResumePolicy)
		parseEntry.bootstrapped = true
		parseEntry.pending = false
		parseEntry.cancel = nil
		parseEntry.done = nil
		parseEntry.mu.Unlock()

		updateCachedSnapshot(parseKey, func(parsePrev cachedResourceSnapshot) cachedResourceSnapshot {
			return cachedResourceSnapshot{
				Value:     parseItem.Value,
				Loading:   false,
				Error:     nil,
				Ready:     true,
				Stale:     bootstrapEntryShouldStartStale(parseNow, parseItem),
				UpdatedAt: parseItem.UpdatedAt,
			}
		})
	}
}

// normalizeResumePolicy is an internal cache helper.
func normalizeResumePolicy(parsePolicy CacheResumePolicy) CacheResumePolicy {
	switch parsePolicy {
	case CacheResumeStaleWhileRevalidate, CacheResumeAlwaysRefetch:
		return parsePolicy
	default:
		return CacheResumeTrustOnce
	}
}

// bootstrapEntryShouldStartStale is an internal cache helper.
func bootstrapEntryShouldStartStale(parseNow time.Time, parseEntry CacheBootstrapEntry) bool {
	switch normalizeResumePolicy(parseEntry.ResumePolicy) {
	case CacheResumeAlwaysRefetch:
		return true
	case CacheResumeStaleWhileRevalidate:
		if parseEntry.UpdatedAt.IsZero() || parseEntry.StaleAfter <= 0 {
			return true
		}
		return parseNow.Sub(parseEntry.UpdatedAt) >= parseEntry.StaleAfter
	default:
		return false
	}
}

// openPersistentCacheStore is an internal cache helper.
func openPersistentCacheStore(parseCtx context.Context) (interop.PersistentStore, error) {
	persistentCacheState.mu.Lock()
	if persistentCacheState.opened {
		store := persistentCacheState.store
		parseErr := persistentCacheState.err
		persistentCacheState.mu.Unlock()
		return store, parseErr
	}
	parseOptions := persistentCacheState.options
	persistentCacheState.mu.Unlock()

	parseResolver := parseOptions.StoreResolver
	if parseResolver == nil {
		storeName := strings.TrimSpace(parseOptions.StoreName)
		if storeName == "" {
			storeName = "fetch-cache"
		}
		parseFallbackResolver := parseOptions.FallbackResolver
		if parseFallbackResolver == nil {
			parseFallbackResolver = interop.LocalStorage
		}
		parseFallbackBackend := strings.TrimSpace(parseOptions.FallbackBackend)
		if parseFallbackBackend == "" {
			parseFallbackBackend = "localStorage"
		}
		parseResolver = func(parseCtx2 context.Context) (interop.PersistentStore, error) {
			return interop.OpenPersistentStore(parseCtx2, interop.PersistentStoreOptions{
				Name:               storeName,
				DatabaseName:       parseOptions.DatabaseName,
				DeleteOnCorruption: true,
				FallbackResolver:   parseFallbackResolver,
				FallbackBackend:    parseFallbackBackend,
			})
		}
	}
	store, parseErr2 := parseResolver(parseCtx)

	persistentCacheState.mu.Lock()
	if persistentCacheState.opened {
		parseExisting := persistentCacheState.store
		parseExistingErr := persistentCacheState.err
		persistentCacheState.mu.Unlock()
		if parseErr2 == nil {
			if parseDupCloseErr := store.Close(); parseDupCloseErr != nil {
				runtime.ReportLogWithFields("fetch", runtime.LogWarn, runtime.DiagnosticRecovered, "persistent cache duplicate store close failed", "", map[string]string{
					"message": parseDupCloseErr.Error(),
				})
			}
		}
		return parseExisting, parseExistingErr
	}
	if parseErr2 == nil {
		persistentCacheState.store = store
		persistentCacheState.opened = true
	} else {
		persistentCacheState.store = interop.PersistentStore{}
		persistentCacheState.opened = false
	}
	persistentCacheState.err = parseErr2
	persistentCacheState.mu.Unlock()
	return store, parseErr2
}

// startPersistentCachedRestore is an internal cache helper.
func startPersistentCachedRestore(parseKey string, parseEntry *cachedResourceEntry) {
	if parseKey == "" || parseEntry == nil {
		return
	}
	parseEntry.mu.Lock()
	parseWaiters := parseEntry.restore
	parseValueType := parseEntry.valueType
	parseEntry.mu.Unlock()
	if parseWaiters == nil || parseValueType == nil {
		return
	}

	go func(parseDone *cachedResourceWaiters, parseDesiredType reflect.Type) {
		defer runtime.RecoverContainedPanic("fetch", "persistent cache hydrate")
		defer parseDone.Close()
		store, parseErr := openPersistentCacheStore(context.Background())
		if parseErr == nil {
			var parseRecord persistedCachedResource
			parseOk, parseDecodeErr := store.DecodeJSON(context.Background(), parseKey, &parseRecord)
			if parseDecodeErr != nil {
				parseErr = parseDecodeErr
			} else if parseOk && len(parseRecord.Value) > 0 {
				parseNow := time.Now()
				if persistedEntryExpired(parseNow, parseEntry, parseRecord) {
					deletePersistentCachedSnapshot(parseKey)
				} else {
					parseValue, parseValueErr := decodePersistedCachedValue(parseRecord.Value, parseDesiredType)
					if parseValueErr != nil {
						parseErr = parseValueErr
					} else {
						updateCachedSnapshot(parseKey, func(parsePrev cachedResourceSnapshot) cachedResourceSnapshot {
							return cachedResourceSnapshot{
								Value:     parseValue,
								Loading:   false,
								Error:     nil,
								Ready:     true,
								Stale:     persistedEntryShouldStartStale(parseNow, parseEntry, parseRecord),
								UpdatedAt: parseRecord.UpdatedAt,
							}
						})
						parseEntry.mu.Lock()
						parseEntry.lastLoaded = parseRecord.LastLoaded
						if parseEntry.lastLoaded.IsZero() {
							parseEntry.lastLoaded = parseRecord.UpdatedAt
						}
						parseEntry.lastAccess = time.Now()
						parseEntry.mu.Unlock()
					}
				}
			}
		}

		parseEntry.mu.Lock()
		if parseEntry.restore == parseDone {
			parseEntry.restore = nil
		}
		parseEntry.restored = true
		parseEntry.mu.Unlock()

		if parseErr != nil {
			runtime.ReportLogWithFields("fetch", runtime.LogWarn, runtime.DiagnosticRecovered, "persistent cache restore failed", "", map[string]string{
				"key":     parseKey,
				"message": parseErr.Error(),
			})
		}
	}(parseWaiters, parseValueType)
}

// persistCachedSnapshot is an internal cache helper.
func persistCachedSnapshot(parseKey string) {
	if parseKey == "" {
		return
	}
	parseRaw, parseOk := cachedResourceRegistry.Load(parseKey)
	if !parseOk {
		return
	}
	parseEntry := parseRaw.(*cachedResourceEntry)
	parseEntry.mu.Lock()
	parsePersist := parseEntry.persist
	parseLastLoaded := parseEntry.lastLoaded
	// Capture the request sequence under the lock so the background goroutine can
	// skip the write if a newer update has already superseded this snapshot.
	parseCapturedSeq := parseEntry.requestSeq
	parseEntry.mu.Unlock()
	if !parsePersist {
		return
	}
	parseSnapshot := currentCachedSnapshot(parseKey)
	if !parseSnapshot.Ready || parseSnapshot.Error != nil {
		return
	}
	parseEncoded, parseErr := json.Marshal(parseSnapshot.Value)
	if parseErr != nil {
		runtime.ReportLogWithFields("fetch", runtime.LogWarn, runtime.DiagnosticRecovered, "persistent cache encode failed", "", map[string]string{
			"key":     parseKey,
			"message": parseErr.Error(),
		})
		return
	}
	parseRecord := persistedCachedResource{Value: parseEncoded, UpdatedAt: parseSnapshot.UpdatedAt, LastLoaded: parseLastLoaded}
	go func() {
		defer runtime.RecoverContainedPanic("fetch", "persistent cache write")
		// Skip the write if a newer load has already updated the entry since we
		// captured the snapshot, to avoid persisting a superseded value.
		parseEntryRaw, parseStillPresent := cachedResourceRegistry.Load(parseKey)
		if parseStillPresent {
			parseWriteEntry := parseEntryRaw.(*cachedResourceEntry)
			parseWriteEntry.mu.Lock()
			parseCurrentSeq := parseWriteEntry.requestSeq
			parseWriteEntry.mu.Unlock()
			if parseCurrentSeq != parseCapturedSeq {
				return
			}
		}
		store, storeErr := openPersistentCacheStore(context.Background())
		if storeErr != nil {
			runtime.ReportLogWithFields("fetch", runtime.LogWarn, runtime.DiagnosticRecovered, "persistent cache write failed", "", map[string]string{
				"key":     parseKey,
				"message": storeErr.Error(),
			})
			return
		}
		if parseWriteErr := store.SetJSON(context.Background(), parseKey, parseRecord); parseWriteErr != nil {
			runtime.ReportLogWithFields("fetch", runtime.LogWarn, runtime.DiagnosticRecovered, "persistent cache write failed", "", map[string]string{
				"key":     parseKey,
				"message": parseWriteErr.Error(),
			})
		}
	}()
}

// deletePersistentCachedSnapshot is an internal cache helper.
func deletePersistentCachedSnapshot(parseKey string) {
	if parseKey == "" {
		return
	}
	go func() {
		defer runtime.RecoverContainedPanic("fetch", "persistent cache invalidate")
		store, parseErr := openPersistentCacheStore(context.Background())
		if parseErr != nil {
			return
		}
		if parseRemoveErr := store.RemoveItem(context.Background(), parseKey); parseRemoveErr != nil {
			runtime.ReportLogWithFields("fetch", runtime.LogWarn, runtime.DiagnosticRecovered, "persistent cache delete failed", "", map[string]string{
				"key":     parseKey,
				"message": parseRemoveErr.Error(),
			})
		}
	}()
}

// decodePersistedCachedValue is an internal cache helper.
func decodePersistedCachedValue(parseRaw json.RawMessage, parseDesiredType reflect.Type) (any, error) {
	if parseDesiredType == nil {
		var parseValue any
		if parseErr := json.Unmarshal(parseRaw, &parseValue); parseErr != nil {
			return nil, parseErr
		}
		return parseValue, nil
	}
	parseValuePtr := reflect.New(parseDesiredType)
	if parseErr2 := json.Unmarshal(parseRaw, parseValuePtr.Interface()); parseErr2 != nil {
		return nil, parseErr2
	}
	return parseValuePtr.Elem().Interface(), nil
}

// persistedEntryShouldStartStale is an internal cache helper.
func persistedEntryShouldStartStale(parseNow time.Time, parseEntry *cachedResourceEntry, parseItem persistedCachedResource) bool {
	if parseEntry == nil {
		return false
	}
	parseEntry.mu.Lock()
	parseStaleAfter := parseEntry.staleAfter
	parseEntry.mu.Unlock()
	if parseStaleAfter <= 0 {
		return false
	}
	parseReference := parseItem.LastLoaded
	if parseReference.IsZero() {
		parseReference = parseItem.UpdatedAt
	}
	if parseReference.IsZero() {
		return true
	}
	return parseNow.Sub(parseReference) >= parseStaleAfter
}

// persistedEntryExpired is an internal cache helper.
func persistedEntryExpired(parseNow time.Time, parseEntry *cachedResourceEntry, parseItem persistedCachedResource) bool {
	if parseEntry == nil {
		return false
	}
	parseEntry.mu.Lock()
	parseMaxAge := parseEntry.maxAge
	parseEntry.mu.Unlock()
	if parseMaxAge <= 0 {
		return false
	}
	parseReference := parseItem.LastLoaded
	if parseReference.IsZero() {
		parseReference = parseItem.UpdatedAt
	}
	if parseReference.IsZero() {
		return false
	}
	return parseNow.Sub(parseReference) >= parseMaxAge
}
