//go:build !js || !wasm

package ui

import (
	"errors"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/query"
)

// TestUseQueryReturnsCachedSnapshot proves the hook renders cached data on first call
// (the pure no-fetch read), so a server-seeded or already-fetched key paints immediately
// with no loading flash.
func TestUseQueryReturnsCachedSnapshot(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("user/42", "Ada")

	parseRes := UseQuery(parseCache, "user/42", func() (string, error) {
		parseT.Fatal("fetcher must not run for a fresh cached key on first read")
		return "", nil
	})
	if parseRes.Data != "Ada" || parseRes.Status != query.StatusSuccess {
		parseT.Fatalf("expected fresh cached Ada, got %+v", parseRes)
	}
}

// TestUseMutationOptimisticRollback proves the hook's mutate function applies optimism
// and rolls back to the prior cached value when the mutation fails.
func TestUseMutationOptimisticRollback(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("count", 10)

	parseMutate := UseMutation[int](parseCache, "count")
	parseBoom := errors.New("save failed")
	parseRes := parseMutate(11, func() (int, error) { return 0, parseBoom })

	if !errors.Is(parseRes.Err, parseBoom) {
		parseT.Fatalf("expected the mutation error to surface, got %+v", parseRes)
	}
	if parsePeek, _ := parseCache.Peek("count"); parsePeek != 10 {
		parseT.Fatalf("expected rollback to 10, cache holds %v", parsePeek)
	}
}
