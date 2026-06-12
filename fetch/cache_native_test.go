//go:build !js || !wasm

package fetch

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func TestFetchNativeCachedResourceDisposedHandleNoOps(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	var parseLoads int32
	parseResource := UseCachedResource("users", func(parseCtx context.Context) (string, error) {
		_ = parseCtx
		atomic.AddInt32(&parseLoads, 1)
		return "server", nil
	})
	parseResource.Set("cached")
	if parseState := parseResource.Get(); !parseState.Ready || parseState.Value != "cached" {
		parseT.Fatalf("expected cached value before dispose, got %+v", parseState)
	}

	parseResource.Dispose()
	if parseState := parseResource.Get(); parseState.Ready || parseState.Value != "" || parseState.Error != nil {
		parseT.Fatalf("expected disposed handle to read as empty, got %+v", parseState)
	}

	parseResource.Set("stale")
	parseResource.Update(func(parsePrev string) string { return parsePrev + "-ignored" })
	parseResource.Invalidate()
	parseResource.Reload()
	parseResource.Cancel()
	if parseState := parseResource.Get(); parseState.Ready || parseState.Value != "" || parseState.Stale {
		parseT.Fatalf("expected disposed handle to stay inactive after mutations, got %+v", parseState)
	}
	if atomic.LoadInt32(&parseLoads) != 0 {
		parseT.Fatalf("expected disposed handle reload to no-op, got %d loader calls", atomic.LoadInt32(&parseLoads))
	}

	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseFresh := UseCachedResource("users", func(parseCtx context.Context) (string, error) {
		_ = parseCtx
		return "fresh", nil
	})
	parseFresh.Set("fresh")
	if parseState := parseFresh.Get(); !parseState.Ready || parseState.Value != "fresh" {
		parseT.Fatalf("expected fresh handle value, got %+v", parseState)
	}

	parseResource.Dispose()
	if parseState := parseFresh.Get(); !parseState.Ready || parseState.Value != "fresh" {
		parseT.Fatalf("expected stale handle dispose to leave live cache entry intact, got %+v", parseState)
	}
}

func TestFetchNativeCachedResourceSetCancelsInflightLoad(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	parseRelease := make(chan struct{})
	var parseLoads int32
	parseResource := UseCachedResource("profile", func(parseCtx context.Context) (string, error) {
		atomic.AddInt32(&parseLoads, 1)
		select {
		case <-parseCtx.Done():
			return "", parseCtx.Err()
		case <-parseRelease:
			return "server", nil
		}
	})

	parseResource.Reload()
	waitFetchTestCondition(parseT, 2*time.Second, func() bool {
		return parseResource.Get().Loading
	})

	parseResource.Set("optimistic")
	parseResource.Update(func(parsePrev string) string { return parsePrev + "-v2" })
	close(parseRelease)
	waitFetchTestCondition(parseT, 2*time.Second, func() bool {
		return atomic.LoadInt32(&parseLoads) == 1
	})

	waitFetchTestCondition(parseT, 2*time.Second, func() bool {
		parseState := parseResource.Get()
		return !parseState.Loading && parseState.Ready
	})
	parseState := parseResource.Get()
	if parseState.Value != "optimistic-v2" || parseState.Error != nil || parseState.Stale {
		parseT.Fatalf("expected optimistic value to survive canceled load, got %+v", parseState)
	}
	if atomic.LoadInt32(&parseLoads) != 1 {
		parseT.Fatalf("expected exactly one loader call, got %d", atomic.LoadInt32(&parseLoads))
	}
}

func TestFetchNativeCacheHelperBranches(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	var parseNilWaiters *cachedResourceWaiters
	if parseNilWaiters.Done() != nil {
		parseT.Fatal("expected nil cachedResourceWaiters.Done to return nil")
	}
	parseNilWaiters.Close()

	if parseResolved := resolveCacheOptions(nil); parseResolved != (CacheOptions{}) {
		parseT.Fatalf("expected zero cache options, got %+v", parseResolved)
	}
	parseResolved := resolveCacheOptions([]CacheOptions{{StaleAfter: time.Second, Persist: true}})
	if parseResolved.StaleAfter != time.Second || !parseResolved.Persist {
		parseT.Fatalf("unexpected resolved cache options: %+v", parseResolved)
	}

	if parseID := cachedResourceAtomID("users"); parseID != "__fetch_cached_resource:users" {
		parseT.Fatalf("unexpected cached resource atom id: %q", parseID)
	}
	if parseEntry := getCachedResourceEntry(""); parseEntry == nil {
		parseT.Fatal("expected empty-key cache entry accessor to return a zero entry")
	}
	if parseOwners := cloneCachedOwnerPaths(map[string]int{"b": 1, "a": 2}); !reflect.DeepEqual(parseOwners, []string{"a", "b"}) {
		parseT.Fatalf("expected sorted owner paths, got %v", parseOwners)
	}
	if _, parseOk := castCachedValue[string](5); parseOk {
		parseT.Fatal("expected castCachedValue to report false for a mismatched type")
	}
	if parseValue, parseOk := castCachedValue[string]("ready"); !parseOk || parseValue != "ready" {
		parseT.Fatalf("expected castCachedValue success, got value=%q ok=%t", parseValue, parseOk)
	}

	if parseErr := waitForCachedResource(nil, nil); parseErr != nil { //nolint:staticcheck // nil context is the case under test
		parseT.Fatalf("expected nil waiters to succeed, got %v", parseErr)
	}
	if parseSnapshot := currentCachedSnapshot("missing"); parseSnapshot != (cachedResourceSnapshot{}) {
		parseT.Fatalf("expected missing snapshot lookup to return zero value, got %+v", parseSnapshot)
	}
	parseClosedWaiters := newCachedResourceWaiters()
	parseClosedWaiters.Close()
	if parseErr := waitForCachedResource(context.Background(), parseClosedWaiters); parseErr != nil {
		parseT.Fatalf("expected closed waiters to succeed, got %v", parseErr)
	}
	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseWaiting := newCachedResourceWaiters()
	parseCancel()
	if parseErr := waitForCachedResource(parseCtx, parseWaiting); !errors.Is(parseErr, context.Canceled) {
		parseT.Fatalf("expected context cancellation while waiting, got %v", parseErr)
	}
	if resolveCachedContext(nil) == nil { //nolint:staticcheck // nil context fallback is the case under test
		parseT.Fatal("expected resolveCachedContext to fall back to context.Background")
	}
	if parseValue, parseErr := LoadCached(context.Background(), "", func(parseCtx2 context.Context) (string, error) {
		_ = parseCtx2
		return "direct", nil
	}); parseErr != nil || parseValue != "direct" {
		parseT.Fatalf("expected empty-key LoadCached to bypass shared cache, value=%q err=%v", parseValue, parseErr)
	}
	if _, parseErr := LoadCached[string](context.Background(), "users", nil); parseErr == nil {
		parseT.Fatal("expected LoadCached to reject a nil loader")
	}

	parseEntry := getCachedResourceEntry("inspect")
	configureCachedResourceEntry[string]("inspect", parseEntry, CacheOptions{DisposeAfter: time.Second, MaxAge: time.Second})
	retainCachedResource("inspect", " Thread/B ")
	retainCachedResource("inspect", "Thread/A")
	updateCachedSnapshot("", nil)
	parseInspections := InspectCachedResources()
	if len(parseInspections) == 0 || parseInspections[0].Key != "inspect" || !reflect.DeepEqual(parseInspections[0].OwnerPaths, []string{"Thread/A", "Thread/B"}) {
		parseT.Fatalf("unexpected cache inspection snapshot: %+v", parseInspections)
	}
	releaseCachedResource("inspect", "Thread/A")
	releaseCachedResource("inspect", "Thread/B")

	setCachedValue("inspect", "ready")
	InvalidateResource("inspect")
	if parseState := currentCachedSnapshot("inspect"); !parseState.Stale || parseState.Value != "ready" {
		parseT.Fatalf("expected InvalidateResource to preserve ready data and mark the snapshot stale, got %+v", parseState)
	}
	parseEntry.lastLoaded = time.Now().Add(-2 * time.Second)
	parseEntry.lastAccess = time.Now().Add(-2 * time.Second)
	if !shouldLoadCachedEntry(currentCachedSnapshot("inspect"), &cachedResourceEntry{invalidated: true}) {
		parseT.Fatal("expected invalidated cached entry to require a reload")
	}
	parseRestoring := &cachedResourceEntry{restore: newCachedResourceWaiters()}
	if shouldLoadCachedEntry(cachedResourceSnapshot{}, parseRestoring) {
		parseT.Fatal("expected restoring cache entry to suppress reloads")
	}
	prepareCachedResourceEntry("inspect", parseEntry)
	if parseState := currentCachedSnapshot("inspect"); parseState.Ready {
		parseT.Fatalf("expected expired idle cache snapshot to clear, got %+v", parseState)
	}

	parseSweepEntry := getCachedResourceEntry("sweep")
	configureCachedResourceEntry[string]("sweep", parseSweepEntry, CacheOptions{DisposeAfter: time.Millisecond})
	setCachedValue("sweep", "old")
	parseSweepEntry.lastAccess = time.Now().Add(-time.Second)
	if !shouldDisposeCachedEntry(time.Now(), parseSweepEntry) {
		parseT.Fatal("expected shouldDisposeCachedEntry to report true for an idle entry")
	}
	if parseDisposed := SweepCachedResources(); parseDisposed != 1 {
		parseT.Fatalf("expected one swept cache entry, got %d", parseDisposed)
	}
}

func TestFetchNativeLoadCachedBranches(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	var parseLoads int32
	parseValue, parseErr := LoadCached(context.Background(), "article", func(parseCtx context.Context) (string, error) {
		_ = parseCtx
		atomic.AddInt32(&parseLoads, 1)
		return "loaded", nil
	})
	if parseErr != nil || parseValue != "loaded" {
		parseT.Fatalf("expected first LoadCached success, value=%q err=%v", parseValue, parseErr)
	}
	parseValue, parseErr = LoadCached(context.Background(), "article", func(parseCtx context.Context) (string, error) {
		_ = parseCtx
		atomic.AddInt32(&parseLoads, 1)
		return "unexpected", nil
	})
	if parseErr != nil || parseValue != "loaded" || atomic.LoadInt32(&parseLoads) != 1 {
		parseT.Fatalf("expected ready LoadCached to reuse cached data, value=%q err=%v loads=%d", parseValue, parseErr, atomic.LoadInt32(&parseLoads))
	}

	if _, parseErr2 := LoadCached(context.Background(), "broken", func(parseCtx context.Context) (string, error) {
		_ = parseCtx
		return "", errors.New("boom")
	}); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "boom") {
		parseT.Fatalf("expected failing LoadCached loader to return its error, got %v", parseErr2)
	}

	parseSlowEntry := getCachedResourceEntry("slow")
	configureCachedResourceEntry[string]("slow", parseSlowEntry, CacheOptions{})
	parseRelease := make(chan struct{})
	if _, parseStarted := startCachedLoad("slow", parseSlowEntry, func(parseCtx context.Context) (any, error) {
		select {
		case <-parseCtx.Done():
			return nil, parseCtx.Err()
		case <-parseRelease:
			return "slow", nil
		}
	}, true, nil); !parseStarted {
		parseT.Fatal("expected startCachedLoad to launch the pending request")
	}
	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	if _, parseErr2 := LoadCached[string](parseCtx, "slow", func(parseLoadCtx context.Context) (string, error) {
		_ = parseLoadCtx
		return "ignored", nil
	}); !errors.Is(parseErr2, context.Canceled) {
		parseT.Fatalf("expected waiting LoadCached call to respect context cancellation, got %v", parseErr2)
	}
	cancelCachedLoad("slow")
	close(parseRelease)
}

// TestFetchNativeCachedResourceFailingLoaderNoRetryStorm is a regression test for
// finding #50: UseCachedResource previously included parseSnapshot.UpdatedAt in its
// UseEffect dependency list. updateCachedSnapshot bumps UpdatedAt even on error,
// which caused the effect to re-fire after every failed load, creating an infinite
// retry storm. The fix removes UpdatedAt from the effect deps, so a failing loader
// fires exactly once and does not re-schedule itself.
func TestFetchNativeCachedResourceFailingLoaderNoRetryStorm(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	var parseLoads int32
	parseKey := "failing-resource"
	parseEntry := getCachedResourceEntry(parseKey)
	configureCachedResourceEntry[string](parseKey, parseEntry, CacheOptions{})

	// Run the loader directly (simulating what UseCachedResource would trigger) and
	// let it complete with an error.
	parseWaiters, parseStarted := startCachedLoad(parseKey, parseEntry, func(parseCtx context.Context) (any, error) {
		_ = parseCtx
		atomic.AddInt32(&parseLoads, 1)
		return nil, fmt.Errorf("load error")
	}, false, nil)
	if !parseStarted {
		parseT.Fatal("expected startCachedLoad to launch the first load")
	}

	// Wait for the first load to finish.
	waitForCachedResource(context.Background(), parseWaiters) //nolint:errcheck
	waitFetchTestCondition(parseT, 2*time.Second, func() bool {
		parseSnapshot := currentCachedSnapshot(parseKey)
		return !parseSnapshot.Loading && parseSnapshot.Error != nil
	})

	// Snapshot has an error and UpdatedAt is set. Simulate the effect re-evaluation
	// that the old code would have triggered: call startCachedLoad again with the same
	// snapshot (no shouldLoad predicate changes other than UpdatedAt). Under the old
	// code this would start another load; under the fix it must not.
	parseSnapshot := currentCachedSnapshot(parseKey)
	if parseSnapshot.Error == nil {
		parseT.Fatal("expected error snapshot after failing load")
	}

	// shouldLoadCachedEntry must report false for a non-ready, non-stale, non-pending
	// entry that already has an error — there is nothing that would trigger a reload.
	parseEntry.mu.Lock()
	parseShouldLoad := shouldLoadCachedEntry(parseSnapshot, parseEntry)
	parseEntry.mu.Unlock()
	if parseShouldLoad {
		parseT.Fatal("shouldLoadCachedEntry must return false after a failed load to prevent retry storms")
	}

	// A second startCachedLoad call (not forced) must be a no-op.
	_, parseStarted2 := startCachedLoad(parseKey, parseEntry, func(parseCtx context.Context) (any, error) {
		_ = parseCtx
		atomic.AddInt32(&parseLoads, 1)
		return nil, fmt.Errorf("should not run")
	}, false, nil)
	if parseStarted2 {
		parseT.Fatal("startCachedLoad must not start a second load after a failed non-stale non-invalidated load")
	}
	if atomic.LoadInt32(&parseLoads) != 1 {
		parseT.Fatalf("expected exactly one loader invocation, got %d", atomic.LoadInt32(&parseLoads))
	}
}
