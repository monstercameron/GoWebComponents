//go:build js && wasm
// +build js,wasm

package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"syscall/js"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

type noOpScheduler struct{}

func (noOpScheduler) RequestIdleCallback(parseCallback func(runtime.Deadline)) {}

func (noOpScheduler) SetTimeout(parseCallback func(), parseDelay int) {}

func installFetchHookContext(parseT *testing.T) {
	parseT.Helper()
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: noOpScheduler{}})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	resetCachedResourcesForTest()
	ConfigurePersistentCache(PersistentCacheOptions{})
	parseT.Cleanup(func() {
		resetCachedResourcesForTest()
		ConfigurePersistentCache(PersistentCacheOptions{})
		runtime.SetCurrentFiber(nil)
	})
}

func resetCachedResourcesForTest() {
	var parseKeys []string
	cachedResourceRegistry.Range(func(parseKey2, parseValue interface{}) bool {
		cacheKey, _ := parseKey2.(string)
		if cacheKey != "" {
			parseKeys = append(parseKeys, cacheKey)
		}
		return true
	})
	for _, parseKey := range parseKeys {
		DisposeResource(parseKey)
	}
}

func setGlobalJSValue(parseName string, parseValue interface{}) func() {
	parseGlobal := js.Global()
	parsePrev := parseGlobal.Get(parseName)
	parseGlobal.Set(parseName, parseValue)
	return func() {
		parseGlobal.Set(parseName, parsePrev)
	}
}

func makeResolvedPromise(parseValue js.Value) js.Value {
	return js.Global().Get("Promise").Call("resolve", parseValue)
}

func waitForResult(parseT *testing.T, parseCh <-chan Result) Result {
	parseT.Helper()
	select {
	case parseResult := <-parseCh:
		return parseResult
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for fetch result")
		return Result{}
	}
}

func TestUseFetchWrapper(parseT *testing.T) {
	installFetchHookContext(parseT)
	parseRestoreFetch := setGlobalJSValue("fetch", js.Null())
	defer parseRestoreFetch()

	parseResource := UseFetch("/api/demo", Options{Method: "POST"})
	parseInitial := parseResource.Get()
	if parseInitial.Loading || parseInitial.Error != "" || parseInitial.Data != nil {
		parseT.Fatalf("unexpected initial state: %+v", parseInitial)
	}

	parseResource.Refetch()
	parseDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) {
		parseState := parseResource.Get()
		if !parseState.Loading {
			if parseState.Error != "fetch API unavailable" {
				parseT.Fatalf("expected unavailable fetch error, got %q", parseState.Error)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	parseT.Fatal("timed out waiting for fetch hook to settle")
}

func TestUseFetchReturnsStableHandleShape(parseT *testing.T) {
	installFetchHookContext(parseT)
	parseResource := UseFetch("/api/demo")
	if parseState := parseResource.Get(); parseState.Loading || parseState.Error != "" || parseState.Data != nil {
		parseT.Fatalf("unexpected initial fetch state: %+v", parseState)
	}
}

func TestUseResourceReturnsStableHandleShape(parseT *testing.T) {
	installFetchHookContext(parseT)

	parseResource := UseResource(func(parseCtx context.Context) (string, error) {
		return "ok", nil
	}, "dep")

	parseState := parseResource.Get()
	if parseState.Loading || parseState.Error != nil || parseState.Ready {
		parseT.Fatalf("unexpected initial resource state: %+v", parseState)
	}
	parseResource.Reload()
	parseResource.Cancel()
}

func TestAsyncResourceZeroValue(parseT *testing.T) {
	var parseResource AsyncResource[string]
	parseState := parseResource.Get()
	if parseState.Loading || parseState.Error != nil || parseState.Ready || parseState.Value != "" {
		parseT.Fatalf("expected zero-value resource state, got %+v", parseState)
	}
	parseResource.Reload()
	parseResource.Cancel()
}

func TestCachedResourceZeroValue(parseT *testing.T) {
	var parseResource CachedResource[string]
	parseState := parseResource.Get()
	if parseState.Loading || parseState.Error != nil || parseState.Ready || parseState.Stale || parseState.Value != "" || !parseState.UpdatedAt.IsZero() {
		parseT.Fatalf("expected zero-value cached resource state, got %+v", parseState)
	}
	parseResource.Reload()
	parseResource.Cancel()
	parseResource.Invalidate()
	parseResource.Set("ignored")
	parseResource.Update(func(parsePrev string) string { return parsePrev + "x" })
}

func TestUseCachedResourceReturnsStableHandleShape(parseT *testing.T) {
	installFetchHookContext(parseT)
	parseResource := UseCachedResource("users", func(parseCtx context.Context) (string, error) {
		return "ok", nil
	})

	parseState := parseResource.Get()
	if parseState.Loading || parseState.Error != nil || parseState.Ready || parseState.Stale {
		parseT.Fatalf("unexpected initial cached resource state: %+v", parseState)
	}
	parseResource.Reload()
	parseResource.Cancel()
	parseResource.Invalidate()
}

func TestUseCachedResourceDeduplicatesInflightRequests(parseT *testing.T) {
	installFetchHookContext(parseT)
	var parseLoads int32
	parseLoader := func(parseCtx context.Context) (string, error) {
		atomic.AddInt32(&parseLoads, 1)
		select {
		case <-parseCtx.Done():
			return "", parseCtx.Err()
		case <-time.After(25 * time.Millisecond):
			return "shared", nil
		}
	}

	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseFirst := UseCachedResource("shared-users", parseLoader)
	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseSecond := UseCachedResource("shared-users", parseLoader)

	parseFirst.Reload()
	parseSecond.Reload()

	parseDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) {
		parseFirstState := parseFirst.Get()
		parseSecondState := parseSecond.Get()
		if parseFirstState.Ready && parseSecondState.Ready {
			if parseFirstState.Value != "shared" || parseSecondState.Value != "shared" {
				parseT.Fatalf("expected both cached resources to share the same value, got %+v and %+v", parseFirstState, parseSecondState)
			}
			if atomic.LoadInt32(&parseLoads) != 1 {
				parseT.Fatalf("expected exactly one loader call, got %d", atomic.LoadInt32(&parseLoads))
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	parseT.Fatalf("timed out waiting for cached resource to settle; first=%+v second=%+v", parseFirst.Get(), parseSecond.Get())
}

func TestInvalidateResourceMarksSnapshotStale(parseT *testing.T) {
	installFetchHookContext(parseT)
	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseResource := UseCachedResource("invalidate-demo", func(parseCtx context.Context) (string, error) {
		return "fresh", nil
	})

	parseResource.Set("ready")
	parseResource.Invalidate()

	parseState := parseResource.Get()
	if !parseState.Ready || !parseState.Stale || parseState.Error != nil || parseState.Value != "ready" {
		parseT.Fatalf("expected invalidated cached state to preserve ready data and become stale, got %+v", parseState)
	}
}

func TestCachedResourceUpdateSharesOptimisticValue(parseT *testing.T) {
	installFetchHookContext(parseT)
	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseFirst := UseCachedResource("optimistic", func(parseCtx context.Context) (int, error) {
		return 0, nil
	})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseSecond := UseCachedResource("optimistic", func(parseCtx2 context.Context) (int, error) {
		return 0, nil
	})

	parseFirst.Set(3)
	parseSecond.Update(func(parsePrev int) int {
		return parsePrev + 4
	})

	parseFirstState := parseFirst.Get()
	parseSecondState := parseSecond.Get()
	if !parseFirstState.Ready || !parseSecondState.Ready || parseFirstState.Value != 7 || parseSecondState.Value != 7 {
		parseT.Fatalf("expected optimistic update to be shared, got %+v and %+v", parseFirstState, parseSecondState)
	}
}

func TestCachedResourceReloadKeepsStaleValueOnError(parseT *testing.T) {
	installFetchHookContext(parseT)
	var parseFailNext atomic.Bool
	parseLoader := func(parseCtx context.Context) (string, error) {
		if parseFailNext.Load() {
			return "", errors.New("reload failed")
		}
		return "fresh", nil
	}

	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseResource := UseCachedResource("reload-error", parseLoader)
	parseResource.Set("cached")
	parseFailNext.Store(true)
	parseResource.Reload()

	parseDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) {
		parseState := parseResource.Get()
		if !parseState.Loading {
			if parseState.Value != "cached" || parseState.Error == nil || !parseState.Ready || !parseState.Stale {
				parseT.Fatalf("expected stale cached value to survive reload error, got %+v", parseState)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	parseT.Fatalf("timed out waiting for cached resource reload error state: %+v", parseResource.Get())
}

func TestDisposeResourceClearsSharedCachedValue(parseT *testing.T) {
	installFetchHookContext(parseT)
	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseResource := UseCachedResource("dispose-demo", func(parseCtx context.Context) (string, error) {
		return "fresh", nil
	})

	parseResource.Set("cached")
	parseResource.Dispose()

	parseState := parseResource.Get()
	if parseState.Ready || parseState.Loading || parseState.Stale || parseState.Error != nil || parseState.Value != "" || !parseState.UpdatedAt.IsZero() {
		parseT.Fatalf("expected disposed cached state to be cleared, got %+v", parseState)
	}
	if _, parseOk := cachedResourceRegistry.Load("dispose-demo"); parseOk {
		parseT.Fatal("expected disposed cached entry to leave the registry")
	}
}

func TestSweepCachedResourcesDisposesIdleEntries(parseT *testing.T) {
	installFetchHookContext(parseT)
	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseResource := UseCachedResource("sweep-demo", func(parseCtx context.Context) (string, error) {
		return "fresh", nil
	}, CacheOptions{DisposeAfter: time.Millisecond})
	parseResource.Set("cached")

	parseRaw, parseOk := cachedResourceRegistry.Load("sweep-demo")
	if !parseOk {
		parseT.Fatal("expected cache entry to be registered")
	}
	parseEntry := parseRaw.(*cachedResourceEntry)
	parseEntry.mu.Lock()
	parseEntry.lastAccess = time.Now().Add(-10 * time.Millisecond)
	parseEntry.mu.Unlock()

	if parseDisposed := SweepCachedResources(); parseDisposed != 1 {
		parseT.Fatalf("expected one swept cache entry, got %d", parseDisposed)
	}
	if parseState := parseResource.Get(); parseState.Ready || parseState.Value != "" {
		parseT.Fatalf("expected swept cache state to be cleared, got %+v", parseState)
	}
}

func TestLoadCachedSharesResultWithUseCachedResource(parseT *testing.T) {
	installFetchHookContext(parseT)
	var parseLoads int32
	parseValue, parseErr := LoadCached(context.Background(), "loader-bridge", func(parseCtx context.Context) (string, error) {
		atomic.AddInt32(&parseLoads, 1)
		return "shared", nil
	}, CacheOptions{StaleAfter: time.Minute})
	if parseErr != nil {
		parseT.Fatalf("expected cached load to succeed, got %v", parseErr)
	}
	if parseValue != "shared" {
		parseT.Fatalf("expected cached load value, got %q", parseValue)
	}

	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseResource := UseCachedResource("loader-bridge", func(parseCtx2 context.Context) (string, error) {
		atomic.AddInt32(&parseLoads, 1)
		return "should-not-run", nil
	}, CacheOptions{StaleAfter: time.Minute})

	parseState := parseResource.Get()
	if !parseState.Ready || parseState.Value != "shared" || parseState.Error != nil {
		parseT.Fatalf("expected hook reader to see seeded shared cache, got %+v", parseState)
	}
	if atomic.LoadInt32(&parseLoads) != 1 {
		parseT.Fatalf("expected shared cache bridge to avoid a second load, got %d loads", atomic.LoadInt32(&parseLoads))
	}
}

func TestLoadCachedDeduplicatesConcurrentImperativeReaders(parseT *testing.T) {
	installFetchHookContext(parseT)
	var parseLoads int32
	parseLoader := func(parseCtx context.Context) (string, error) {
		atomic.AddInt32(&parseLoads, 1)
		select {
		case <-parseCtx.Done():
			return "", parseCtx.Err()
		case <-time.After(20 * time.Millisecond):
			return "shared", nil
		}
	}

	parseResults := make(chan string, 2)
	parseErrs := make(chan error, 2)
	for parseI := 0; parseI < 2; parseI++ {
		go func() {
			parseValue, parseErr := LoadCached(context.Background(), "dedupe-loadcached", parseLoader)
			if parseErr == nil {
				parseResults <- parseValue
			}
			parseErrs <- parseErr
		}()
	}

	for parseI2 := 0; parseI2 < 2; parseI2++ {
		if parseErr2 := <-parseErrs; parseErr2 != nil {
			parseT.Fatalf("expected cached imperative load to succeed, got %v", parseErr2)
		}
	}
	parseFirst := <-parseResults
	parseSecond := <-parseResults
	if parseFirst != "shared" || parseSecond != "shared" {
		parseT.Fatalf("expected both imperative readers to share one result, got %q and %q", parseFirst, parseSecond)
	}
	if atomic.LoadInt32(&parseLoads) != 1 {
		parseT.Fatalf("expected exactly one imperative load, got %d", atomic.LoadInt32(&parseLoads))
	}
}

func TestRestoreCacheBootstrapSeedsTrustOnceCache(parseT *testing.T) {
	installFetchHookContext(parseT)
	parsePayload := ui.SSRBootstrap{
		Data: map[string]interface{}{
			CacheBootstrapDataKey: CacheBootstrap{
				Entries: []CacheBootstrapEntry{{
					Key:          "bootstrap-users",
					Value:        "server-seeded",
					UpdatedAt:    time.Now(),
					ResumePolicy: CacheResumeTrustOnce,
					StaleAfter:   time.Minute,
				}},
			},
		},
	}
	if parseErr := RestoreCacheBootstrap(parsePayload); parseErr != nil {
		parseT.Fatalf("expected bootstrap restore to succeed, got %v", parseErr)
	}

	var parseLoads int32
	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseResource := UseCachedResource("bootstrap-users", func(parseCtx context.Context) (string, error) {
		atomic.AddInt32(&parseLoads, 1)
		return "client", nil
	}, CacheOptions{StaleAfter: time.Minute})

	parseState := parseResource.Get()
	if !parseState.Ready || parseState.Stale || parseState.Value != "server-seeded" {
		parseT.Fatalf("expected trust-once bootstrap value to seed ready cache state, got %+v", parseState)
	}
	if atomic.LoadInt32(&parseLoads) != 0 {
		parseT.Fatalf("expected trust-once bootstrap to skip immediate reload, got %d loads", atomic.LoadInt32(&parseLoads))
	}
}

func TestRestoreCacheBootstrapResumePoliciesMarkEntriesForAuthoritativeClientLoad(parseT *testing.T) {
	installFetchHookContext(parseT)
	parsePayload := ui.SSRBootstrap{
		Data: map[string]interface{}{
			CacheBootstrapDataKey: CacheBootstrap{
				Entries: []CacheBootstrapEntry{
					{
						Key:          "bootstrap-swr",
						Value:        "server-stale",
						UpdatedAt:    time.Now().Add(-time.Minute),
						ResumePolicy: CacheResumeStaleWhileRevalidate,
						StaleAfter:   time.Second,
					},
					{
						Key:          "bootstrap-always",
						Value:        "server-always",
						UpdatedAt:    time.Now(),
						ResumePolicy: CacheResumeAlwaysRefetch,
						StaleAfter:   time.Hour,
					},
				},
			},
		},
	}
	if parseErr := RestoreCacheBootstrap(parsePayload); parseErr != nil {
		parseT.Fatalf("expected bootstrap restore to succeed, got %v", parseErr)
	}

	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseSwr := UseCachedResource("bootstrap-swr", func(parseCtx context.Context) (string, error) {
		return "client-swr", nil
	}, CacheOptions{StaleAfter: time.Minute})

	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseAlways := UseCachedResource("bootstrap-always", func(parseCtx2 context.Context) (string, error) {
		return "client-always", nil
	}, CacheOptions{StaleAfter: time.Minute})

	parseSwrState := parseSwr.Get()
	parseAlwaysState := parseAlways.Get()
	if !parseSwrState.Ready || !parseSwrState.Stale {
		parseT.Fatalf("expected stale-while-revalidate bootstrap to preserve ready data and mark stale, got %+v", parseSwrState)
	}
	if !parseAlwaysState.Ready || !parseAlwaysState.Stale {
		parseT.Fatalf("expected always-refetch bootstrap to preserve ready data and mark stale, got %+v", parseAlwaysState)
	}
	if !shouldLoadCachedEntry(currentCachedSnapshot("bootstrap-swr"), getCachedResourceEntry("bootstrap-swr")) {
		parseT.Fatal("expected stale-while-revalidate bootstrap entry to require an authoritative client load")
	}
	if !shouldLoadCachedEntry(currentCachedSnapshot("bootstrap-always"), getCachedResourceEntry("bootstrap-always")) {
		parseT.Fatal("expected always-refetch bootstrap entry to require an authoritative client load")
	}
}

func TestInspectCachedResourcesReportsKeyPolicyAndSubscribers(parseT *testing.T) {
	installFetchHookContext(parseT)
	parsePayload := ui.SSRBootstrap{
		Data: map[string]interface{}{
			CacheBootstrapDataKey: CacheBootstrap{
				Entries: []CacheBootstrapEntry{{
					Key:          "inspect-cache",
					Value:        "server",
					UpdatedAt:    time.Now(),
					ResumePolicy: CacheResumeTrustOnce,
				}},
			},
		},
	}
	if parseErr := RestoreCacheBootstrap(parsePayload); parseErr != nil {
		parseT.Fatalf("expected bootstrap restore to succeed, got %v", parseErr)
	}

	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseResource := UseCachedResource("inspect-cache", func(parseCtx context.Context) (string, error) {
		return "client", nil
	})
	if parseState := parseResource.Get(); !parseState.Ready {
		parseT.Fatalf("expected seeded resource to start ready, got %+v", parseState)
	}
	retainCachedResource("inspect-cache", "App > InspectCache")

	parseEntries := InspectCachedResources()
	if len(parseEntries) != 1 {
		parseT.Fatalf("expected one inspected cache entry, got %#v", parseEntries)
	}
	parseEntry := parseEntries[0]
	if parseEntry.Key != "inspect-cache" || parseEntry.ResumePolicy != CacheResumeTrustOnce {
		parseT.Fatalf("unexpected inspected cache entry: %+v", parseEntry)
	}
	if parseEntry.SubscriberCount != 1 {
		parseT.Fatalf("expected one active subscriber, got %+v", parseEntry)
	}
	if len(parseEntry.OwnerPaths) != 1 || parseEntry.OwnerPaths[0] != "App > InspectCache" {
		parseT.Fatalf("expected inspected owner path, got %+v", parseEntry)
	}
}

func TestLoadCachedPersistsAndRestoresFromDurableStore(parseT *testing.T) {
	installFetchHookContext(parseT)
	parseStorage := installMockPersistentLocalStorage(parseT)
	parseRestoreIndexedDB := setGlobalJSValue("indexedDB", js.Undefined())
	defer parseRestoreIndexedDB()

	var parseLoads int32
	parseValue, parseErr := LoadCached(context.Background(), "persisted-users", func(parseCtx context.Context) (string, error) {
		atomic.AddInt32(&parseLoads, 1)
		return "from-network", nil
	}, CacheOptions{Persist: true, StaleAfter: time.Hour})
	if parseErr != nil {
		parseT.Fatalf("expected persisted cached load to succeed, got %v", parseErr)
	}
	if parseValue != "from-network" {
		parseT.Fatalf("expected network value, got %q", parseValue)
	}

	parseDeadline := time.Now().Add(2 * time.Second)
	parsePersisted := ""
	for time.Now().Before(parseDeadline) {
		if parseStored, parseOk := parseStorage["persisted-users"]; parseOk && parseStored != "" {
			parsePersisted = parseStored
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if parsePersisted == "" {
		parseT.Fatal("expected first cached load to persist durable snapshot before restore")
	}

	// Clear only in-memory cache state so durable storage remains available for restore.
	cachedResourceRegistry = sync.Map{}
	clearCachedSnapshot("persisted-users")
	var parseRestoreLoads int32
	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseResource := UseCachedResource("persisted-users", func(parseCtx2 context.Context) (string, error) {
		atomic.AddInt32(&parseRestoreLoads, 1)
		return "unexpected", nil
	}, CacheOptions{Persist: true, StaleAfter: time.Hour})

	parseDeadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) {
		parseState := parseResource.Get()
		if parseState.Ready {
			if parseState.Value != "from-network" || parseState.Stale || parseState.Error != nil {
				parseT.Fatalf("expected durable cache restore, got %+v", parseState)
			}
			if atomic.LoadInt32(&parseRestoreLoads) != 0 {
				parseT.Fatalf("expected durable restore to avoid cold load, got %d loads", atomic.LoadInt32(&parseRestoreLoads))
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	parseT.Fatalf("timed out waiting for durable cache restore: %+v", parseResource.Get())
}

func TestDisposeResourceRemovesDurableCachedValue(parseT *testing.T) {
	installFetchHookContext(parseT)
	parseStorage := installMockPersistentLocalStorage(parseT)
	parseRestoreIndexedDB := setGlobalJSValue("indexedDB", js.Undefined())
	defer parseRestoreIndexedDB()

	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseResource := UseCachedResource("persisted-dispose", func(parseCtx context.Context) (string, error) {
		return "fresh", nil
	}, CacheOptions{Persist: true, StaleAfter: time.Hour})
	parseResource.Set("cached")

	parseDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) {
		if _, parseOk := parseStorage["persisted-dispose"]; parseOk {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	parseResource.Dispose()
	parseDeadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) {
		if _, parseOk2 := parseStorage["persisted-dispose"]; !parseOk2 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	parseT.Fatal("expected persistent cache entry to be removed on dispose")
}

func installMockPersistentLocalStorage(parseT *testing.T) map[string]string {
	parseT.Helper()
	parseData := map[string]string{}
	parseStorage := js.Global().Get("Object").New()
	getItemFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if parseValue, parseOk := parseData[parseArgs[0].String()]; parseOk {
			return parseValue
		}
		return js.Null()
	})
	setItemFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseData[parseArgs2[0].String()] = parseArgs2[1].String()
		parseStorage.Set("length", len(parseData))
		return nil
	})
	parseRemoveItemFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		delete(parseData, parseArgs3[0].String())
		parseStorage.Set("length", len(parseData))
		return nil
	})
	clearFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		for parseKey := range parseData {
			delete(parseData, parseKey)
		}
		parseStorage.Set("length", 0)
		return nil
	})
	parseKeyFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		parseIndex := parseArgs5[0].Int()
		parseKeys := make([]string, 0, len(parseData))
		for parseKey2 := range parseData {
			parseKeys = append(parseKeys, parseKey2)
		}
		if parseIndex < 0 || parseIndex >= len(parseKeys) {
			return js.Null()
		}
		return parseKeys[parseIndex]
	})
	parseStorage.Set("getItem", getItemFn)
	parseStorage.Set("setItem", setItemFn)
	parseStorage.Set("removeItem", parseRemoveItemFn)
	parseStorage.Set("clear", clearFn)
	parseStorage.Set("key", parseKeyFn)
	parseStorage.Set("length", 0)
	parseRestore := setGlobalJSValue("localStorage", parseStorage)
	parseT.Cleanup(func() {
		parseRestore()
		getItemFn.Release()
		setItemFn.Release()
		parseRemoveItemFn.Release()
		clearFn.Release()
		parseKeyFn.Release()
	})
	return parseData
}

func TestPersistedCachedValueRoundTripsJSONEnvelope(parseT *testing.T) {
	parseRecord := persistedCachedResource{}
	parseEncoded, parseErr := json.Marshal(persistedCachedResource{
		Value:      json.RawMessage(`{"name":"Ada"}`),
		UpdatedAt:  time.Date(2026, time.March, 21, 10, 0, 0, 0, time.UTC),
		LastLoaded: time.Date(2026, time.March, 21, 10, 0, 0, 0, time.UTC),
	})
	if parseErr != nil {
		parseT.Fatalf("expected record marshal to succeed, got %v", parseErr)
	}
	if parseErr2 := json.Unmarshal(parseEncoded, &parseRecord); parseErr2 != nil {
		parseT.Fatalf("expected record unmarshal to succeed, got %v", parseErr2)
	}
	parseValue, parseErr := decodePersistedCachedValue(parseRecord.Value, reflect.TypeOf(struct {
		name string `json:"name"`
	}{}))
	if parseErr != nil {
		parseT.Fatalf("expected persisted value decode, got %v", parseErr)
	}
	parseDecoded := parseValue.(struct {
		name string `json:"name"`
	})
	if parseDecoded.Name != "Ada" {
		parseT.Fatalf("unexpected persisted decoded value: %+v", parseDecoded)
	}
}

func TestFetchUnavailable(parseT *testing.T) {
	parseRestoreFetch := setGlobalJSValue("fetch", js.Null())
	defer parseRestoreFetch()

	parseResult := waitForResult(parseT, Fetch("/api/demo", Options{}))
	if parseResult.Err == nil || !strings.Contains(parseResult.Err.Error(), "fetch API unavailable") {
		parseT.Fatalf("expected unavailable fetch error, got %#v", parseResult.Err)
	}
}

func TestBuildMultipartFormDataAppendsFieldsAndFiles(parseT *testing.T) {
	var parseAppendCalls []string
	parseAppendFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) >= 2 {
			parseEntry := parseArgs[0].String() + "="
			if parseArgs[1].Type() == js.TypeString {
				parseEntry = parseArgs[0].String() + "=" + parseArgs[1].String()
			} else {
				parseEntry += parseArgs[1].Get("name").String()
			}
			parseAppendCalls = append(parseAppendCalls, parseEntry)
		}
		return nil
	})
	defer parseAppendFn.Release()

	parseFormDataCtor := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseObject := js.Global().Get("Object").New()
		parseObject.Set("append", parseAppendFn)
		return parseObject
	})
	defer parseFormDataCtor.Release()
	parseRestoreFormData := setGlobalJSValue("FormData", parseFormDataCtor)
	defer parseRestoreFormData()

	parseRawFile := js.Global().Get("Object").New()
	parseRawFile.Set("name", "avatar.png")
	parseForm, parseErr := buildMultipartFormData(MultipartBody{
		Fields: map[string]string{"title": "Demo"},
		Files:  []MultipartFile{{FieldName: "asset", File: ui.WrapJSFile(parseRawFile)}},
	})
	if parseErr != nil {
		parseT.Fatalf("expected multipart form-data to build, got %v", parseErr)
	}
	if !parseForm.Truthy() {
		parseT.Fatal("expected multipart form-data js value")
	}
	if len(parseAppendCalls) != 2 {
		parseT.Fatalf("expected two multipart append calls, got %v", parseAppendCalls)
	}
	if parseAppendCalls[0] != "title=Demo" || parseAppendCalls[1] != "asset=avatar.png" {
		parseT.Fatalf("unexpected multipart append calls: %v", parseAppendCalls)
	}
}

func TestUploadUnavailableWithoutXMLHttpRequest(parseT *testing.T) {
	parseRestoreXHR := setGlobalJSValue("XMLHttpRequest", js.Null())
	defer parseRestoreXHR()

	parseCtx, parseCancel := context.WithCancel(context.Background())
	defer parseCancel()
	parseUpdate := <-Upload(parseCtx, "/api/upload", Options{Body: MultipartBody{}})
	if !parseUpdate.Done || parseUpdate.Result.Err == nil || !strings.Contains(parseUpdate.Result.Err.Error(), "XMLHttpRequest unavailable") {
		parseT.Fatalf("expected unavailable upload transport error, got %+v", parseUpdate)
	}
}

func TestFetchHTTPErrorIncludesStatusHeadersAndDecodesJSON(parseT *testing.T) {
	parseTextFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return makeResolvedPromise(js.ValueOf(`{"message":"invalid upload"}`))
	})
	defer parseTextFn.Release()

	parseHeaders := js.Global().Get("Object").New()
	parseForEachFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		if len(parseArgs2) == 0 {
			return nil
		}
		parseCallback := parseArgs2[0]
		parseCallback.Invoke("application/json", "Content-Type")
		parseCallback.Invoke("trace-123", "X-Trace")
		return nil
	})
	defer parseForEachFn.Release()
	parseHeaders.Set("forEach", parseForEachFn)

	parseResponse := js.Global().Get("Object").New()
	parseResponse.Set("status", 422)
	parseResponse.Set("statusText", "Unprocessable Entity")
	parseResponse.Set("headers", parseHeaders)
	parseResponse.Set("text", parseTextFn)

	parseFetchFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		return makeResolvedPromise(parseResponse)
	})
	defer parseFetchFn.Release()

	parseRestoreFetch := setGlobalJSValue("fetch", parseFetchFn)
	defer parseRestoreFetch()

	parseResult := waitForResult(parseT, Fetch("/api/demo", Options{}))
	if parseResult.Status != 422 {
		parseT.Fatalf("expected status 422, got %d", parseResult.Status)
	}
	parseHttpErr, parseOk := parseResult.Err.(HTTPError)
	if !parseOk {
		parseT.Fatalf("expected HTTPError, got %#v", parseResult.Err)
	}
	if parseHttpErr.Status != 422 || parseHttpErr.StatusText != "Unprocessable Entity" {
		parseT.Fatalf("unexpected http error metadata: %+v", parseHttpErr)
	}
	if parseResult.Headers["Content-Type"] != "application/json" || parseResult.Headers["X-Trace"] != "trace-123" {
		parseT.Fatalf("expected response headers to be preserved, got %#v", parseResult.Headers)
	}
	var parsePayload map[string]string
	if parseErr := parseResult.DecodeJSON(&parsePayload); parseErr != nil {
		parseT.Fatalf("expected json response to decode, got %v", parseErr)
	}
	if parsePayload["message"] != "invalid upload" {
		parseT.Fatalf("unexpected decoded payload: %#v", parsePayload)
	}
}

func TestUploadSuccessPreservesResponseMetadataAndIgnoresLateCancel(parseT *testing.T) {
	var parseUploadProgressListener js.Value
	var parseLoadListener js.Value
	var setHeaders []string
	var parseAbortCalls int

	parseUploadAddEventListenerFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) >= 2 && parseArgs[0].String() == "progress" {
			parseUploadProgressListener = parseArgs[1]
		}
		return nil
	})
	defer parseUploadAddEventListenerFn.Release()

	parseAddEventListenerFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		if len(parseArgs2) >= 2 && parseArgs2[0].String() == "load" {
			parseLoadListener = parseArgs2[1]
		}
		return nil
	})
	defer parseAddEventListenerFn.Release()

	setRequestHeaderFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		if len(parseArgs3) >= 2 {
			setHeaders = append(setHeaders, parseArgs3[0].String()+"="+parseArgs3[1].String())
		}
		return nil
	})
	defer setRequestHeaderFn.Release()

	parseAbortFn := js.FuncOf(func(parseThis4 js.Value, parseArgs4 []js.Value) interface{} {
		parseAbortCalls++
		return nil
	})
	defer parseAbortFn.Release()

	getAllResponseHeadersFn := js.FuncOf(func(parseThis5 js.Value, parseArgs5 []js.Value) interface{} {
		return "Content-Type: application/json\r\nX-Upload-ID: up-42\r\n"
	})
	defer getAllResponseHeadersFn.Release()

	parseSendFn := js.FuncOf(func(parseThis6 js.Value, parseArgs6 []js.Value) interface{} {
		parseProgressEvent := js.Global().Get("Object").New()
		parseProgressEvent.Set("loaded", 3)
		parseProgressEvent.Set("total", 9)
		parseProgressEvent.Set("lengthComputable", true)
		if parseUploadProgressListener.Truthy() {
			parseUploadProgressListener.Invoke(parseProgressEvent)
		}
		parseLoadEvent := js.Global().Get("Object").New()
		parseLoadEvent.Set("loaded", 9)
		parseLoadEvent.Set("total", 9)
		parseLoadEvent.Set("lengthComputable", true)
		if parseLoadListener.Truthy() {
			parseLoadListener.Invoke(parseLoadEvent)
		}
		return nil
	})
	defer parseSendFn.Release()

	parseOpenFn := js.FuncOf(func(parseThis7 js.Value, parseArgs7 []js.Value) interface{} { return nil })
	defer parseOpenFn.Release()

	parseXhrUpload := js.Global().Get("Object").New()
	parseXhrUpload.Set("addEventListener", parseUploadAddEventListenerFn)

	parseXhr := js.Global().Get("Object").New()
	parseXhr.Set("status", 201)
	parseXhr.Set("statusText", "Created")
	parseXhr.Set("responseText", `{"asset_id":"asset-1"}`)
	parseXhr.Set("open", parseOpenFn)
	parseXhr.Set("setRequestHeader", setRequestHeaderFn)
	parseXhr.Set("addEventListener", parseAddEventListenerFn)
	parseXhr.Set("send", parseSendFn)
	parseXhr.Set("abort", parseAbortFn)
	parseXhr.Set("getAllResponseHeaders", getAllResponseHeadersFn)
	parseXhr.Set("upload", parseXhrUpload)

	parseXhrCtor := js.FuncOf(func(parseThis8 js.Value, parseArgs8 []js.Value) interface{} {
		return parseXhr
	})
	defer parseXhrCtor.Release()
	parseRestoreXHR := setGlobalJSValue("XMLHttpRequest", parseXhrCtor)
	defer parseRestoreXHR()

	parseFormDataCtor := js.FuncOf(func(parseThis9 js.Value, parseArgs9 []js.Value) interface{} {
		parseObject := js.Global().Get("Object").New()
		parseObject.Set("append", js.FuncOf(func(parseThis10 js.Value, parseArgs10 []js.Value) interface{} { return nil }))
		return parseObject
	})
	defer parseFormDataCtor.Release()
	parseRestoreFormData := setGlobalJSValue("FormData", parseFormDataCtor)
	defer parseRestoreFormData()

	parseCtx, parseCancel := context.WithCancel(context.Background())
	defer parseCancel()

	var parseUpdates []UploadUpdate
	for parseUpdate := range Upload(parseCtx, "/api/upload", Options{
		Method: "POST",
		Headers: map[string]interface{}{
			"X-Test":       "ok",
			"Content-Type": "multipart/form-data",
		},
		Body: MultipartBody{Fields: map[string]string{"title": "Demo"}},
	}) {
		parseUpdates = append(parseUpdates, parseUpdate)
	}
	parseCancel()
	time.Sleep(20 * time.Millisecond)

	if parseAbortCalls != 0 {
		parseT.Fatalf("expected completed upload to ignore late context cancellation, got %d abort calls", parseAbortCalls)
	}
	if len(parseUpdates) != 2 {
		parseT.Fatalf("expected progress plus final upload update, got %d updates: %+v", len(parseUpdates), parseUpdates)
	}
	if len(setHeaders) != 1 || setHeaders[0] != "X-Test=ok" {
		parseT.Fatalf("expected multipart upload to keep custom headers and drop manual content-type, got %v", setHeaders)
	}
	parseFinal := parseUpdates[len(parseUpdates)-1]
	if !parseFinal.Done || parseFinal.Result.Err != nil {
		parseT.Fatalf("expected successful final update, got %+v", parseFinal)
	}
	if parseFinal.Result.Status != 201 || parseFinal.Result.Headers["X-Upload-ID"] != "up-42" {
		parseT.Fatalf("expected upload result metadata, got %+v", parseFinal.Result)
	}
	if parseFinal.Loaded != 9 || parseFinal.Total != 9 || !parseFinal.LengthComputable {
		parseT.Fatalf("expected final upload progress to reflect load event, got %+v", parseFinal)
	}
	var parsePayload map[string]string
	if parseErr := parseFinal.Result.DecodeJSON(&parsePayload); parseErr != nil {
		parseT.Fatalf("expected upload json response to decode, got %v", parseErr)
	}
	if parsePayload["asset_id"] != "asset-1" {
		parseT.Fatalf("unexpected upload payload: %#v", parsePayload)
	}
}

func TestFetchMarshalError(parseT *testing.T) {
	parseResult := waitForResult(parseT, Fetch("/api/demo", Options{
		Body: map[string]float64{"bad": math.NaN()},
	}))
	if parseResult.Err == nil || !strings.Contains(parseResult.Err.Error(), "failed to encode body") {
		parseT.Fatalf("expected marshal error, got %#v", parseResult.Err)
	}
}

func TestFetchSuccess(parseT *testing.T) {
	var parseCapturedURL string
	var parseCapturedMethod string
	var parseCapturedHeader string
	var parseCapturedBody string

	parseTextFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return makeResolvedPromise(js.ValueOf("response body"))
	})
	defer parseTextFn.Release()

	parseResponse := js.Global().Get("Object").New()
	parseResponse.Set("status", 200)
	parseResponse.Set("statusText", "OK")
	parseHeaders := js.Global().Get("Object").New()
	parseHeaders.Set("forEach", js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} { return nil }))
	parseResponse.Set("headers", parseHeaders)
	parseResponse.Set("text", parseTextFn)

	parseFetchFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		parseCapturedURL = parseArgs3[0].String()
		parseOpts := parseArgs3[1]
		parseCapturedMethod = parseOpts.Get("method").String()
		parseCapturedHeader = parseOpts.Get("headers").Get("X-Test").String()
		parseCapturedBody = parseOpts.Get("body").String()
		return makeResolvedPromise(parseResponse)
	})
	defer parseFetchFn.Release()

	parseRestoreFetch := setGlobalJSValue("fetch", parseFetchFn)
	defer parseRestoreFetch()

	parseResult := waitForResult(parseT, Fetch("/api/demo", Options{
		Method:  "POST",
		Headers: map[string]interface{}{"X-Test": "ok"},
		Body:    "body",
	}))

	if parseResult.Err != nil {
		parseT.Fatalf("expected successful fetch result, got error %v", parseResult.Err)
	}
	if parseResult.Data != "response body" {
		parseT.Fatalf("expected response body, got %#v", parseResult.Data)
	}
	if parseCapturedURL != "/api/demo" || parseCapturedMethod != "POST" || parseCapturedHeader != "ok" || parseCapturedBody != "body" {
		parseT.Fatalf("unexpected fetch invocation: url=%q method=%q header=%q body=%q", parseCapturedURL, parseCapturedMethod, parseCapturedHeader, parseCapturedBody)
	}
}

func TestReturnChannelIsNoOp(parseT *testing.T) {
	parseCh := make(chan Result, 1)
	ReturnChannel(parseCh)
	parseCh <- Result{Data: "ok"}
	if parseResult := <-parseCh; parseResult.Data != "ok" {
		parseT.Fatalf("expected channel contents to remain untouched, got %#v", parseResult)
	}
}

func BenchmarkBuildMultipartFormData(parseB *testing.B) {
	parseAppendFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} { return nil })
	defer parseAppendFn.Release()

	parseFormDataCtor := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseObject := js.Global().Get("Object").New()
		parseObject.Set("append", parseAppendFn)
		return parseObject
	})
	defer parseFormDataCtor.Release()
	parseRestoreFormData := setGlobalJSValue("FormData", parseFormDataCtor)
	defer parseRestoreFormData()

	parseRawFile := js.Global().Get("Object").New()
	parseRawFile.Set("name", "hero.png")
	parseBody := MultipartBody{
		Fields: map[string]string{
			"title":    "Demo",
			"audience": "buyers",
		},
		Files: []MultipartFile{{FieldName: "asset", File: ui.WrapJSFile(parseRawFile)}},
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseForm, parseErr := buildMultipartFormData(parseBody)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
		if !parseForm.Truthy() {
			parseB.Fatal("expected form data result")
		}
	}
}
