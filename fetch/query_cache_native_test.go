//go:build !js || !wasm

package fetch

import (
	"context"
	"fmt"
	"reflect"
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
	if parseKeys := QueryKeysForTag("users"); !reflect.DeepEqual(parseKeys, []string{"users"}) {
		parseT.Fatalf("expected users tag to include users query, got %v", parseKeys)
	}
	if parseTagsForKey := QueryTagsForKey("users"); !reflect.DeepEqual(parseTagsForKey, []string{"team", "users"}) {
		parseT.Fatalf("expected tags for key, got %v", parseTagsForKey)
	}

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

	parseInspections := InspectCachedResources()
	if len(parseInspections) != 1 || !reflect.DeepEqual(parseInspections[0].Tags, []string{"team", "users"}) {
		parseT.Fatalf("expected inspection to include query tags, got %+v", parseInspections)
	}

	parseQuery.Dispose()
	if parseKeys2 := QueryKeysForTag("users"); len(parseKeys2) != 0 {
		parseT.Fatalf("expected disposed query to unregister tags, got %v", parseKeys2)
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
}
