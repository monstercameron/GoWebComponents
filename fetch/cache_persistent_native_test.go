//go:build !js || !wasm

package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/interop"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func TestFetchNativePersistentCachePoliciesAndRetry(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	if parseBootstrap, parseErr := readCacheBootstrap(nil); parseErr != nil || len(parseBootstrap.Entries) != 0 {
		parseT.Fatalf("expected empty bootstrap decode, bootstrap=%+v err=%v", parseBootstrap, parseErr)
	}
	if _, parseErr := readCacheBootstrap(map[string]any{CacheBootstrapDataKey: func() {}}); parseErr == nil {
		parseT.Fatal("expected invalid bootstrap payload to fail JSON encoding")
	}

	parseNow := time.Date(2026, time.April, 6, 12, 0, 0, 0, time.UTC)
	parsePayload := ui.SSRBootstrap{
		Data: map[string]any{
			CacheBootstrapDataKey: CacheBootstrap{
				Entries: []CacheBootstrapEntry{
					{Key: "profile", Value: "Ada", UpdatedAt: parseNow, ResumePolicy: CacheResumeAlwaysRefetch},
				},
			},
		},
	}
	if parseErr := RestoreCacheBootstrap(parsePayload); parseErr != nil {
		parseT.Fatalf("expected RestoreCacheBootstrap success, got %v", parseErr)
	}
	parseSnapshot := currentCachedSnapshot("profile")
	if !parseSnapshot.Ready || !parseSnapshot.Stale || parseSnapshot.Value != "Ada" {
		parseT.Fatalf("unexpected restored bootstrap snapshot: %+v", parseSnapshot)
	}

	if normalizeResumePolicy("") != CacheResumeTrustOnce || normalizeResumePolicy(CacheResumeStaleWhileRevalidate) != CacheResumeStaleWhileRevalidate {
		parseT.Fatal("expected normalizeResumePolicy to preserve known values and default unknown values")
	}
	if !bootstrapEntryShouldStartStale(parseNow, CacheBootstrapEntry{ResumePolicy: CacheResumeAlwaysRefetch}) {
		parseT.Fatal("expected always-refetch bootstrap entry to start stale")
	}
	if !bootstrapEntryShouldStartStale(parseNow, CacheBootstrapEntry{ResumePolicy: CacheResumeStaleWhileRevalidate}) {
		parseT.Fatal("expected zero-time stale-while-revalidate entry to start stale")
	}
	if bootstrapEntryShouldStartStale(parseNow, CacheBootstrapEntry{ResumePolicy: CacheResumeTrustOnce, UpdatedAt: parseNow}) {
		parseT.Fatal("expected trust-once bootstrap entry to stay fresh")
	}

	if parseValue, parseErr := decodePersistedCachedValue(json.RawMessage(`{"count":3}`), nil); parseErr != nil {
		parseT.Fatalf("expected untyped persisted decode success, got %v", parseErr)
	} else if parseMap, parseOk := parseValue.(map[string]any); !parseOk || parseMap["count"] != float64(3) {
		parseT.Fatalf("unexpected untyped persisted decode value: %#v", parseValue)
	}
	type parseProfile struct {
		Name string `json:"name"`
	}
	if parseValue, parseErr := decodePersistedCachedValue(json.RawMessage(`{"name":"Ada"}`), reflect.TypeFor[parseProfile]()); parseErr != nil {
		parseT.Fatalf("expected typed persisted decode success, got %v", parseErr)
	} else if parseTyped, parseOk := parseValue.(parseProfile); !parseOk || parseTyped.Name != "Ada" {
		parseT.Fatalf("unexpected typed persisted decode value: %#v", parseValue)
	}
	if _, parseErr := decodePersistedCachedValue(json.RawMessage(`{`), reflect.TypeFor[parseProfile]()); parseErr == nil {
		parseT.Fatal("expected invalid persisted JSON to fail typed decode")
	}

	parseEntry := &cachedResourceEntry{staleAfter: time.Minute, maxAge: time.Minute}
	parseRecord := persistedCachedResource{UpdatedAt: parseNow.Add(-3 * time.Minute), LastLoaded: parseNow.Add(-2 * time.Minute)}
	if persistedEntryShouldStartStale(parseNow, nil, parseRecord) || persistedEntryExpired(parseNow, nil, parseRecord) {
		parseT.Fatal("expected nil persisted-entry helpers to return false")
	}
	if !persistedEntryShouldStartStale(parseNow, parseEntry, parseRecord) {
		parseT.Fatal("expected aged persisted entry to start stale")
	}
	if !persistedEntryExpired(parseNow, parseEntry, parseRecord) {
		parseT.Fatal("expected aged persisted entry to expire")
	}

	parseStore, _ := buildFetchTestPersistentStore(parseT)
	var parseAttempts int
	ConfigurePersistentCache(PersistentCacheOptions{
		StoreResolver: func(parseCtx context.Context) (interop.PersistentStore, error) {
			_ = parseCtx
			parseAttempts++
			if parseAttempts == 1 {
				return interop.PersistentStore{}, errors.New("temporary open failure")
			}
			return parseStore, nil
		},
	})
	if _, parseErr := openPersistentCacheStore(context.Background()); parseErr == nil {
		parseT.Fatal("expected first persistent-cache open to fail")
	}
	if _, parseErr := openPersistentCacheStore(context.Background()); parseErr != nil || parseAttempts != 2 {
		parseT.Fatalf("expected second persistent-cache open to retry and succeed, attempts=%d err=%v", parseAttempts, parseErr)
	}
}

func TestFetchNativePersistentCacheRestorePersistAndDelete(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	parseStore, parseData := buildFetchTestPersistentStore(parseT)
	ConfigurePersistentCache(PersistentCacheOptions{
		StoreResolver: func(parseCtx context.Context) (interop.PersistentStore, error) {
			_ = parseCtx
			return parseStore, nil
		},
	})

	parseKey := "persisted-profile"
	parseEntry := getCachedResourceEntry(parseKey)
	configureCachedResourceEntry[string](parseKey, parseEntry, CacheOptions{Persist: true, StaleAfter: time.Hour, MaxAge: 2 * time.Hour})

	parseRecord := persistedCachedResource{
		Value:      json.RawMessage(`"restored"`),
		UpdatedAt:  time.Now().Add(-time.Minute),
		LastLoaded: time.Now().Add(-time.Minute),
	}
	parseEncoded, parseErr := json.Marshal(parseRecord)
	if parseErr != nil {
		parseT.Fatalf("expected persisted cache test record to encode, got %v", parseErr)
	}
	parseData[parseKey] = string(parseEncoded)

	parseEntry.mu.Lock()
	parseEntry.restore = newCachedResourceWaiters()
	parseEntry.mu.Unlock()
	startPersistentCachedRestore(parseKey, parseEntry)
	waitFetchTestCondition(parseT, 2*time.Second, func() bool {
		parseSnapshot := currentCachedSnapshot(parseKey)
		return parseSnapshot.Ready && parseSnapshot.Value == "restored"
	})

	setCachedValue(parseKey, "updated")
	waitFetchTestCondition(parseT, 2*time.Second, func() bool {
		parseStored, _, parseStoreErr := parseStore.GetItem(context.Background(), parseKey)
		return parseStoreErr == nil && strings.Contains(parseStored, `"updated"`)
	})

	deletePersistentCachedSnapshot(parseKey)
	waitFetchTestCondition(parseT, 2*time.Second, func() bool {
		_, parseOk, parseStoreErr := parseStore.GetItem(context.Background(), parseKey)
		if parseStoreErr != nil {
			return false
		}
		return !parseOk
	})

	parseCorruptKey := "persisted-corrupt"
	parseCorruptEntry := getCachedResourceEntry(parseCorruptKey)
	configureCachedResourceEntry[string](parseCorruptKey, parseCorruptEntry, CacheOptions{Persist: true})
	parseData[parseCorruptKey] = `{"value":{`
	parseCorruptEntry.mu.Lock()
	parseCorruptEntry.restore = newCachedResourceWaiters()
	parseCorruptEntry.mu.Unlock()
	startPersistentCachedRestore(parseCorruptKey, parseCorruptEntry)
	waitFetchTestCondition(parseT, 2*time.Second, func() bool {
		parseCorruptEntry.mu.Lock()
		defer parseCorruptEntry.mu.Unlock()
		return parseCorruptEntry.restored && parseCorruptEntry.restore == nil
	})
}
