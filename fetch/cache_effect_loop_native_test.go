//go:build !js || !wasm

package fetch

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// driveCachedEffectUntilQuiet emulates the UseCachedResource load effect on the
// native target, where ui.UseEffect is a no-op. The effect re-fires whenever its
// dependency tuple (Ready, Stale, Loading) changes, exactly as the hook declares.
// It returns the number of times the effect ran before the tuple stopped changing,
// or parseMaxRuns when the effect never settles (a retry loop).
func driveCachedEffectUntilQuiet(parseKey string, parseEntry *cachedResourceEntry, parseLoader func(context.Context) (any, error), parseMaxRuns int) int {
	type parseDeps struct{ ready, stale, loading bool }
	var parseLast *parseDeps
	parseRuns := 0
	for parseStep := 0; parseStep < parseMaxRuns*4; parseStep++ {
		parseSnap := currentCachedSnapshot(parseKey)
		parseNow := parseDeps{parseSnap.Ready, parseSnap.Stale, parseSnap.Loading}
		if parseLast != nil && *parseLast == parseNow {
			time.Sleep(50 * time.Millisecond)
			if parseAfter := currentCachedSnapshot(parseKey); (parseDeps{parseAfter.Ready, parseAfter.Stale, parseAfter.Loading}) == parseNow {
				return parseRuns
			}
			continue
		}
		parseLast = &parseNow
		parseRuns++
		if parseRuns >= parseMaxRuns {
			return parseRuns
		}
		startCachedLoad(parseKey, parseEntry, parseLoader, false, nil)
		// A real render observes the synchronous Loading transition before the
		// load settles, so record the post-start tuple as the last seen one.
		parseStarted := currentCachedSnapshot(parseKey)
		parseLast = &parseDeps{parseStarted.Ready, parseStarted.Stale, parseStarted.Loading}
		time.Sleep(50 * time.Millisecond)
	}
	return parseRuns
}

// TestFetchNativeCachedResourceFailedRevalidationDoesNotLoop pins that a failed
// revalidation of a Ready resource does not re-fire the load effect forever.
func TestFetchNativeCachedResourceFailedRevalidationDoesNotLoop(parseT *testing.T) {
	synctest.Test(parseT, func(parseT *testing.T) {
		installFetchTestHookContext(parseT)

		parseKey := "revalidate-fails"
		parseEntry := getCachedResourceEntry(parseKey)
		configureCachedResourceEntry[string](parseKey, parseEntry, CacheOptions{})
		setCachedValue(parseKey, "good")

		var parseLoads int32
		parseLoader := func(parseCtx context.Context) (any, error) {
			_ = parseCtx
			atomic.AddInt32(&parseLoads, 1)
			return nil, errors.New("revalidation failed")
		}

		InvalidateResource(parseKey)
		driveCachedEffectUntilQuiet(parseKey, parseEntry, parseLoader, 50)

		if parseCount := atomic.LoadInt32(&parseLoads); parseCount > 2 {
			parseT.Fatalf("failed revalidation retried %d times; want a bounded count (<=2)", parseCount)
		}
		parseSnap := currentCachedSnapshot(parseKey)
		if !parseSnap.Ready || parseSnap.Value != "good" || parseSnap.Error == nil {
			parseT.Fatalf("expected stale-but-ready value with error kept, got %+v", parseSnap)
		}

		// A fresh Invalidate must still trigger exactly one more attempt.
		parseBefore := atomic.LoadInt32(&parseLoads)
		InvalidateResource(parseKey)
		driveCachedEffectUntilQuiet(parseKey, parseEntry, parseLoader, 50)
		if parseGot := atomic.LoadInt32(&parseLoads) - parseBefore; parseGot != 1 {
			parseT.Fatalf("expected one retry after a new Invalidate, got %d", parseGot)
		}
	})
}

// TestFetchNativeInfiniteQueryMountedLoadNextKeepsAppendedPage pins that LoadNext
// on a mounted infinite query does not trigger a first-page reload that discards
// the appended page.
func TestFetchNativeInfiniteQueryMountedLoadNextKeepsAppendedPage(parseT *testing.T) {
	synctest.Test(parseT, func(parseT *testing.T) {
		installFetchTestHookContext(parseT)

		var parseFirstLoads int32
		parsePageLoader := func(parseCtx context.Context, parseRequest QueryPageRequest[int]) (QueryPage[string, int], error) {
			_ = parseCtx
			if parseRequest.PageIndex == 0 {
				atomic.AddInt32(&parseFirstLoads, 1)
			}
			time.Sleep(10 * time.Millisecond)
			return QueryPage[string, int]{
				Items:      []string{fmt.Sprintf("page-%d", parseRequest.PageIndex)},
				NextCursor: parseRequest.Cursor + 1,
				HasNext:    parseRequest.PageIndex == 0,
			}, nil
		}
		parseQuery := UseInfiniteQuery[string, int]("mounted-feed", parsePageLoader)
		parseEntry := getCachedResourceEntry("mounted-feed")
		// The effect's loader is the hook's first-page loader.
		parseEffectLoader := func(parseCtx context.Context) (any, error) {
			parsePage, parseErr := parsePageLoader(parseCtx, QueryPageRequest[int]{PageIndex: 0})
			if parseErr != nil {
				return InfiniteQueryData[string, int]{}, parseErr
			}
			return buildInfiniteQueryData([]QueryPage[string, int]{parsePage}), nil
		}

		driveCachedEffectUntilQuiet("mounted-feed", parseEntry, parseEffectLoader, 20)
		if parseState := parseQuery.Get(); !parseState.Ready || len(parseState.Pages) != 1 {
			parseT.Fatalf("expected one loaded page after mount, got %+v", parseState)
		}

		parseQuery.LoadNext()
		driveCachedEffectUntilQuiet("mounted-feed", parseEntry, parseEffectLoader, 20)

		parseState := parseQuery.Get()
		if len(parseState.Pages) != 2 || len(parseState.Items) != 2 || parseState.Items[1] != "page-1" {
			parseT.Fatalf("LoadNext page was discarded: pages=%d items=%v firstPageLoads=%d", len(parseState.Pages), parseState.Items, atomic.LoadInt32(&parseFirstLoads))
		}
		if parseCount := atomic.LoadInt32(&parseFirstLoads); parseCount != 1 {
			parseT.Fatalf("LoadNext triggered %d first-page loads; want 1", parseCount)
		}
	})
}

// TestFetchNativeDisposedKeyLateLoaderDoesNotTouchNewEntry pins that a loader
// finishing after its key was disposed and re-created cannot clear the Loading
// flag of the new entry's in-flight load.
func TestFetchNativeDisposedKeyLateLoaderDoesNotTouchNewEntry(parseT *testing.T) {
	synctest.Test(parseT, func(parseT *testing.T) {
		installFetchTestHookContext(parseT)

		parseKey := "disposed-late"
		parseOldEntry := getCachedResourceEntry(parseKey)
		configureCachedResourceEntry[string](parseKey, parseOldEntry, CacheOptions{})
		parseReleaseOld := make(chan struct{})
		startCachedLoad(parseKey, parseOldEntry, func(parseCtx context.Context) (any, error) {
			<-parseReleaseOld // ignores cancellation: it finishes late, after the dispose
			return "old", nil
		}, false, nil)
		time.Sleep(10 * time.Millisecond)

		DisposeResource(parseKey)

		parseNewEntry := getCachedResourceEntry(parseKey)
		configureCachedResourceEntry[string](parseKey, parseNewEntry, CacheOptions{})
		parseReleaseNew := make(chan struct{})
		startCachedLoad(parseKey, parseNewEntry, func(parseCtx context.Context) (any, error) {
			<-parseReleaseNew
			return "new", nil
		}, false, nil)
		time.Sleep(10 * time.Millisecond)
		if !currentCachedSnapshot(parseKey).Loading {
			parseT.Fatal("new entry should be loading")
		}

		close(parseReleaseOld)
		time.Sleep(10 * time.Millisecond)
		if !currentCachedSnapshot(parseKey).Loading {
			parseT.Fatalf("late loader of the disposed entry cleared Loading on the new entry: %+v", currentCachedSnapshot(parseKey))
		}

		close(parseReleaseNew)
		time.Sleep(10 * time.Millisecond)
		if parseSnap := currentCachedSnapshot(parseKey); parseSnap.Loading || parseSnap.Value != "new" {
			parseT.Fatalf("expected the new load to land, got %+v", parseSnap)
		}
	})
}
