//go:build !js || !wasm

package fetch

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetchNativeQueryTagsAndOptimisticRollback(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	parseQuery := UseQuery("users", func(parseCtx context.Context) ([]string, error) {
		_ = parseCtx
		return []string{"server"}, nil
	}, QueryOptions{Tags: []string{" users ", "team", "users"}})
	parseQuery.Set([]string{"Ada"})

	if parseTags := parseQuery.Tags(); !reflect.DeepEqual(parseTags, []string{"team", "users"}) {
		parseT.Fatalf("expected normalized tags, got %v", parseTags)
	}
	parseTagsCopy := parseQuery.Tags()
	parseTagsCopy[0] = "mutated"
	if parseTags := parseQuery.Tags(); !reflect.DeepEqual(parseTags, []string{"team", "users"}) {
		parseT.Fatalf("Tags returned mutable backing slice, got %v", parseTags)
	}
	if parseQuery.CacheKey() != "users" {
		parseT.Fatalf("expected query cache key, got %q", parseQuery.CacheKey())
	}
	if parseKeys := QueryKeysForTag("users"); !reflect.DeepEqual(parseKeys, []string{"users"}) {
		parseT.Fatalf("expected users tag to include users query, got %v", parseKeys)
	}
	if parseTagsForKey := QueryTagsForKey("users"); !reflect.DeepEqual(parseTagsForKey, []string{"team", "users"}) {
		parseT.Fatalf("expected tags for key, got %v", parseTagsForKey)
	}

	parseQuery.Update(func(parsePrev []string) []string {
		return append(append([]string{}, parsePrev...), "Katherine")
	})
	if parseState := parseQuery.Get(); !reflect.DeepEqual(parseState.Value, []string{"Ada", "Katherine"}) {
		parseT.Fatalf("expected Update to derive from previous value, got %+v", parseState)
	}
	parseQuery.Set([]string{"Ada"})

	parseRollback := parseQuery.OptimisticUpdate(func(parsePrev []string) []string {
		parseNext := append([]string{}, parsePrev...)
		return append(parseNext, "Linus")
	})
	if !parseRollback.Active() {
		parseT.Fatal("expected rollback handle to be active")
	}
	if parseState := parseQuery.Get(); !reflect.DeepEqual(parseState.Value, []string{"Ada", "Linus"}) {
		parseT.Fatalf("expected optimistic value, got %+v", parseState)
	}
	parseRollback.Rollback()
	if parseRollback.Active() {
		parseT.Fatal("expected rollback handle to be inactive after rollback")
	}
	if parseState2 := parseQuery.Get(); !reflect.DeepEqual(parseState2.Value, []string{"Ada"}) {
		parseT.Fatalf("expected rollback to restore previous value, got %+v", parseState2)
	}

	parseCommit := parseQuery.OptimisticUpdate(func(parsePrev []string) []string {
		return append(append([]string{}, parsePrev...), "Grace")
	})
	parseCommit.Commit()
	parseCommit.Rollback()
	if parseState3 := parseQuery.Get(); !reflect.DeepEqual(parseState3.Value, []string{"Ada", "Grace"}) {
		parseT.Fatalf("expected committed optimistic value to remain, got %+v", parseState3)
	}

	if parseInvalidated := InvalidateQueryTag("team"); parseInvalidated != 1 {
		parseT.Fatalf("expected one invalidated query, got %d", parseInvalidated)
	}
	if parseState4 := parseQuery.Get(); !parseState4.Stale {
		parseT.Fatalf("expected tag invalidation to mark query stale, got %+v", parseState4)
	}
	parseQuery.Cancel()
	parseQuery.Reload()
	waitFetchTestCondition(parseT, 2*time.Second, func() bool {
		parseState := parseQuery.Get()
		return parseState.Ready && !parseState.Loading && !parseState.Stale && reflect.DeepEqual(parseState.Value, []string{"server"})
	})
	parseQuery.Invalidate()
	if parseState := parseQuery.Get(); !parseState.Stale {
		parseT.Fatalf("expected direct query invalidation to mark stale, got %+v", parseState)
	}

	parseInspections := InspectCachedResources()
	if len(parseInspections) != 1 || !reflect.DeepEqual(parseInspections[0].Tags, []string{"team", "users"}) {
		parseT.Fatalf("expected inspection to include query tags, got %+v", parseInspections)
	}

	parseQuery.Dispose()
	if parseKeys2 := QueryKeysForTag("users"); len(parseKeys2) != 0 {
		parseT.Fatalf("expected disposed query to unregister tags, got %v", parseKeys2)
	}
}

func TestApplyOptimisticUpdateNilFnReturnsInactiveHandle(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	parseUpdate := ApplyOptimisticUpdate[string]("nil-optimistic", nil)
	if parseUpdate.Active() {
		parseT.Fatal("expected nil optimistic update function to return inactive handle")
	}
	if parseUpdateEmptyKey := ApplyOptimisticUpdate[string]("", func(parsePrev string) string { return "next" }); parseUpdateEmptyKey.Active() {
		parseT.Fatal("expected empty optimistic update key to return inactive handle")
	}

	parseUpdate.Commit()
	parseUpdate.Rollback()
	if _, parseOk := cachedResourceRegistry.Load("nil-optimistic"); parseOk {
		parseT.Fatal("expected nil optimistic update to avoid creating a cache entry")
	}
}

func TestFetchNativeLoadQueryRegistersTagsAndInvalidatesOnce(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	parseValue, parseErr := LoadQuery(context.Background(), "profile", func(parseCtx context.Context) (string, error) {
		_ = parseCtx
		return "Ada", nil
	}, QueryOptions{Tags: []string{"user", "profile"}})
	if parseErr != nil || parseValue != "Ada" {
		parseT.Fatalf("expected LoadQuery value, got value=%q err=%v", parseValue, parseErr)
	}

	if parseKeys := QueryKeysForTag("user"); !reflect.DeepEqual(parseKeys, []string{"profile"}) {
		parseT.Fatalf("expected LoadQuery to register tag, got %v", parseKeys)
	}
	if parseInvalidated := InvalidateQueryTags("user", "profile"); parseInvalidated != 1 {
		parseT.Fatalf("expected overlapping tags to invalidate one query, got %d", parseInvalidated)
	}
	if parseSnapshot := currentCachedSnapshot("profile"); !parseSnapshot.Stale {
		parseT.Fatalf("expected LoadQuery cache entry to be stale after tag invalidation, got %+v", parseSnapshot)
	}
	if parseDisposed := DisposeQueryTag("user"); parseDisposed != 1 {
		parseT.Fatalf("expected one disposed query, got %d", parseDisposed)
	}
	if parseDisposedAgain := DisposeQueryTag("user"); parseDisposedAgain != 0 {
		parseT.Fatalf("expected disposed tag to be idempotent, got %d", parseDisposedAgain)
	}
}

// TestFetchNativeInfiniteQueryLoadNextDropsResultSupersededByReload pins the #85
// stale-result fix: a page fetched by LoadNext must NOT be appended if a Reload
// superseded the query while that page was in flight — otherwise a page fetched
// from a now-stale cursor corrupts the freshly reloaded data.
func TestFetchNativeInfiniteQueryLoadNextDropsResultSupersededByReload(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	var parsePageZeroCount int32
	parseReleaseNext := make(chan struct{})
	parseStartedNext := make(chan struct{})
	var parseSignalOnce int32

	parseQuery := UseInfiniteQuery[string, int]("feed-stale", func(parseCtx context.Context, parseRequest QueryPageRequest[int]) (QueryPage[string, int], error) {
		_ = parseCtx
		if parseRequest.PageIndex == 0 {
			parseN := atomic.AddInt32(&parsePageZeroCount, 1)
			return QueryPage[string, int]{
				Items:      []string{fmt.Sprintf("p0-%d", parseN)},
				NextCursor: 1,
				HasNext:    true,
			}, nil
		}
		// The LoadNext page: signal in-flight once, then block so the test can
		// supersede it with a Reload.
		if atomic.CompareAndSwapInt32(&parseSignalOnce, 0, 1) {
			close(parseStartedNext)
		}
		<-parseReleaseNext
		return QueryPage[string, int]{Items: []string{"p1-STALE"}, NextCursor: 2, HasNext: false}, nil
	}, InfiniteQueryOptions[int]{InitialCursor: 0})

	parseQuery.Reload()
	waitFetchTestCondition(parseT, 2*time.Second, func() bool {
		parseState := parseQuery.Get()
		return parseState.Ready && !parseState.Loading && len(parseState.Items) == 1
	})

	parseQuery.LoadNext()   // captures load gen, blocks in the loader
	<-parseStartedNext      // the page load is in flight

	parseQuery.Reload()     // supersedes it: bumps requestSeq, reloads page 0 as p0-2
	waitFetchTestCondition(parseT, 2*time.Second, func() bool {
		parseState := parseQuery.Get()
		return parseState.Ready && !parseState.Loading && len(parseState.Items) == 1 && parseState.Items[0] == "p0-2"
	})

	close(parseReleaseNext) // the stale LoadNext now completes and must be dropped
	// Give the released goroutine time to attempt (and drop) its commit.
	time.Sleep(150 * time.Millisecond)

	parseFinal := parseQuery.Get()
	for _, parseItem := range parseFinal.Items {
		if parseItem == "p1-STALE" {
			parseT.Fatalf("stale LoadNext page corrupted reloaded data: %+v", parseFinal.Items)
		}
	}
	if len(parseFinal.Items) != 1 || parseFinal.Items[0] != "p0-2" {
		parseT.Fatalf("expected reloaded single page p0-2 to survive, got %+v", parseFinal.Items)
	}
}

func TestFetchNativeInfiniteQueryLoadsNextPage(parseT *testing.T) {
	installFetchTestHookContext(parseT)

	parseQuery := UseInfiniteQuery[string, int]("feed", func(parseCtx context.Context, parseRequest QueryPageRequest[int]) (QueryPage[string, int], error) {
		_ = parseCtx
		return QueryPage[string, int]{
			Items:      []string{fmt.Sprintf("page:%d cursor:%d", parseRequest.PageIndex, parseRequest.Cursor)},
			NextCursor: parseRequest.Cursor + 1,
			HasNext:    parseRequest.PageIndex == 0,
		}, nil
	}, InfiniteQueryOptions[int]{Tags: []string{"feed"}, InitialCursor: 10})

	if parseQuery.CacheKey() != "feed" || !reflect.DeepEqual(parseQuery.Tags(), []string{"feed"}) {
		parseT.Fatalf("unexpected infinite query metadata key=%q tags=%v", parseQuery.CacheKey(), parseQuery.Tags())
	}

	parseQuery.Reload()
	waitFetchTestCondition(parseT, 2*time.Second, func() bool {
		parseState := parseQuery.Get()
		return parseState.Ready && !parseState.Loading
	})

	parseState := parseQuery.Get()
	if !reflect.DeepEqual(parseState.Items, []string{"page:0 cursor:10"}) || !parseState.HasNext || parseState.NextCursor != 11 {
		parseT.Fatalf("unexpected first page state: %+v", parseState)
	}

	parseQuery.LoadNext()
	waitFetchTestCondition(parseT, 2*time.Second, func() bool {
		parseState2 := parseQuery.Get()
		return parseState2.Ready && !parseState2.Loading && len(parseState2.Items) == 2
	})

	parseState2 := parseQuery.Get()
	parseWantItems := []string{"page:0 cursor:10", "page:1 cursor:11"}
	if !reflect.DeepEqual(parseState2.Items, parseWantItems) || parseState2.HasNext || parseState2.NextCursor != 12 || len(parseState2.Pages) != 2 {
		parseT.Fatalf("unexpected appended page state: %+v", parseState2)
	}

	parseQuery.Invalidate()
	if parseState3 := parseQuery.Get(); !parseState3.Stale {
		parseT.Fatalf("expected infinite query invalidation to mark stale, got %+v", parseState3)
	}
	parseQuery.Set(InfiniteQueryData[string, int]{Items: []string{"manual"}, HasNext: true, NextCursor: 42})
	parseQuery.Update(func(parsePrev InfiniteQueryData[string, int]) InfiniteQueryData[string, int] {
		parsePrev.Items = append(parsePrev.Items, "updated")
		parsePrev.HasNext = false
		return parsePrev
	})
	parseState4 := parseQuery.Get()
	if !reflect.DeepEqual(parseState4.Items, []string{"manual", "updated"}) || parseState4.HasNext {
		parseT.Fatalf("Set/Update did not update infinite query data: %+v", parseState4)
	}
	parseRollback := parseQuery.OptimisticUpdate(func(parsePrev InfiniteQueryData[string, int]) InfiniteQueryData[string, int] {
		parsePrev.Items = append(parsePrev.Items, "optimistic")
		return parsePrev
	})
	if !parseRollback.Active() {
		parseT.Fatal("expected infinite optimistic update to be active")
	}
	parseRollback.Rollback()
	if parseState5 := parseQuery.Get(); !reflect.DeepEqual(parseState5.Items, []string{"manual", "updated"}) {
		parseT.Fatalf("rollback did not restore infinite query data: %+v", parseState5)
	}
	parseQuery.Cancel()
	parseQuery.Dispose()
	if parseKeys := QueryKeysForTag("feed"); len(parseKeys) != 0 {
		parseT.Fatalf("expected disposed infinite query to unregister tags, got %v", parseKeys)
	}
}
