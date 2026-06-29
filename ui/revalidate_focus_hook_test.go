//go:build !js || !wasm

package ui

import (
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/query"
)

// TestRevalidateOnSignalStalesEveryKey proves the focus/reconnect action marks all cached
// queries stale, so the next read refetches (the core of UseRevalidateOnFocus).
func TestRevalidateOnSignalStalesEveryKey(parseT *testing.T) {
	parseCache := query.New(query.WithStaleTime(time.Hour))
	parseCache.Set("a", 1)
	parseCache.Set("b", 2)

	// Fresh before: a Fetch would hit cache.
	var parseCalls int
	parseFetch := func() (int, error) { parseCalls++; return 0, nil }
	_ = query.Fetch(parseCache, "a", parseFetch)
	if parseCalls != 0 {
		parseT.Fatalf("fresh key should not refetch before the focus signal, calls=%d", parseCalls)
	}

	revalidateOnSignal(parseCache) // simulate focus/online

	_ = query.Fetch(parseCache, "a", parseFetch)
	_ = query.Fetch(parseCache, "b", parseFetch)
	if parseCalls != 2 {
		parseT.Fatalf("after the focus signal every key should revalidate, calls=%d", parseCalls)
	}
}

// TestRevalidateOnSignalNilCacheSafe proves a nil cache is a no-op.
func TestRevalidateOnSignalNilCacheSafe(parseT *testing.T) {
	revalidateOnSignal(nil) // must not panic
}
