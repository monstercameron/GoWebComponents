//go:build !js || !wasm

package ui

import (
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/query"
)

// TestUseOptimisticAppliesImmediately proves the optimistic setter writes the cache at once
// (the UI would re-render with it before any server round-trip).
func TestUseOptimisticAppliesImmediately(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("likes", 10)

	parseValue, parseSet := UseOptimistic[int](parseCache, "likes")
	if parseValue.Data != 10 {
		parseT.Fatalf("expected current 10, got %d", parseValue.Data)
	}
	parseSet(11)
	if parsePeek, _ := parseCache.Peek("likes"); parsePeek != 11 {
		parseT.Fatalf("optimistic set should update the cache immediately, got %v", parsePeek)
	}
}

// TestUseAsyncMutationOptimisticThenCommits proves the async action shows the optimistic
// value at once and commits the server's result in the background.
func TestUseAsyncMutationOptimisticThenCommits(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("likes", 10)

	parseRun := UseAsyncMutation[int](parseCache, "likes")
	parseRelease := make(chan struct{})
	parseRun(11, func() (int, error) {
		<-parseRelease
		return 12, nil // server's authoritative value
	})

	// Optimistic value is visible before the server settles.
	if parsePeek, _ := parseCache.Peek("likes"); parsePeek != 11 {
		parseT.Fatalf("expected optimistic 11 immediately, got %v", parsePeek)
	}

	close(parseRelease)
	if !waitForCache(parseCache, "likes", 12) {
		parsePeek, _ := parseCache.Peek("likes")
		parseT.Fatalf("expected committed server value 12, got %v", parsePeek)
	}
}

// TestUseActionRollsBackOnError proves UseAction's run rolls the cache back to the prior
// value when the action fails.
func TestUseActionRollsBackOnError(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("likes", 10)

	_, parseRun := UseAction[int](parseCache, "likes")
	parseRun(11, func() (int, error) { return 0, errString("server rejected") })

	if !waitForCache(parseCache, "likes", 10) {
		parsePeek, _ := parseCache.Peek("likes")
		parseT.Fatalf("expected rollback to 10 after a failed action, got %v", parsePeek)
	}
}

// waitForCache polls the cache until key equals want, up to a second.
func waitForCache(parseCache *query.Cache, parseKey string, parseWant int) bool {
	parseDeadline := time.Now().Add(time.Second)
	for time.Now().Before(parseDeadline) {
		if parsePeek, parseOk := parseCache.Peek(parseKey); parseOk && parsePeek == parseWant {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return false
}

type errString string

func (parseE errString) Error() string { return string(parseE) }
