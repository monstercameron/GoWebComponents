package query

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestFetchRecoversPanickingFetcher proves a panicking fetcher is converted to an error and does
// NOT wedge the key: the flight is cleared so a subsequent read reports StatusError, not a permanent
// StatusLoading. Regression — without recovery the panic unwinds past runFetch's flight cleanup,
// leaving entry.flight non-nil and the done channel open forever.
func TestFetchRecoversPanickingFetcher(parseT *testing.T) {
	parseCache := New(WithStaleTime(time.Hour))

	parseRes := Fetch(parseCache, "k", func() (string, error) { panic("kaboom") })
	if parseRes.Err == nil || !strings.Contains(parseRes.Err.Error(), "panicked") {
		parseT.Fatalf("expected the fetcher panic surfaced as an error, got %+v", parseRes)
	}

	parseSnap := Snapshot[string](parseCache, "k")
	if parseSnap.Status == StatusLoading {
		parseT.Fatal("flight left open after fetcher panic — key wedged in StatusLoading")
	}
	if parseSnap.Status != StatusError {
		parseT.Fatalf("expected StatusError after a panicking fetcher, got %v", parseSnap.Status)
	}
}

// fakeClock is a deterministic, concurrency-safe clock for staleness tests.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (parseFC *fakeClock) now() time.Time {
	parseFC.mu.Lock()
	defer parseFC.mu.Unlock()
	return parseFC.t
}

func (parseFC *fakeClock) advance(parseD time.Duration) {
	parseFC.mu.Lock()
	defer parseFC.mu.Unlock()
	parseFC.t = parseFC.t.Add(parseD)
}

// TestFetchCoalescesConcurrentCallsAndCaches proves request de-duplication: eight
// concurrent Fetches for one key invoke the fetcher exactly once, and a later Fetch
// within the stale window serves from cache without calling it again.
func TestFetchCoalescesConcurrentCallsAndCaches(parseT *testing.T) {
	parseCache := New(WithStaleTime(time.Hour))
	var parseCalls atomic.Int64
	parseRelease := make(chan struct{})

	var parseWG sync.WaitGroup
	parseResults := make([]Result[int], 8)
	for parseI := range parseResults {
		parseWG.Add(1)
		go func(parseIdx int) {
			defer parseWG.Done()
			parseResults[parseIdx] = Fetch(parseCache, "answer", func() (int, error) {
				parseCalls.Add(1)
				<-parseRelease
				return 42, nil
			})
		}(parseI)
	}

	// Let all goroutines arrive and join the single in-flight fetch before releasing it.
	time.Sleep(25 * time.Millisecond)
	close(parseRelease)
	parseWG.Wait()

	if parseGot := parseCalls.Load(); parseGot != 1 {
		parseT.Fatalf("expected exactly 1 coalesced fetch, got %d", parseGot)
	}
	for parseI, parseRes := range parseResults {
		if parseRes.Data != 42 || parseRes.Status != StatusSuccess {
			parseT.Fatalf("result %d: got %+v, want Data=42 success", parseI, parseRes)
		}
	}

	// A fresh cache hit must not call the fetcher again.
	parseRes := Fetch(parseCache, "answer", func() (int, error) {
		parseCalls.Add(1)
		return -1, nil
	})
	if parseRes.Data != 42 {
		parseT.Fatalf("cache hit returned %d, want 42", parseRes.Data)
	}
	if parseGot := parseCalls.Load(); parseGot != 1 {
		parseT.Fatalf("fresh cache hit should not refetch; calls=%d", parseGot)
	}
}

// TestFetchErrorThenRecovery proves an error surfaces as StatusError with no data, and a
// later successful Fetch caches normally.
func TestFetchErrorThenRecovery(parseT *testing.T) {
	parseCache := New(WithStaleTime(time.Hour))
	parseBoom := errors.New("boom")

	parseRes := Fetch(parseCache, "k", func() (int, error) { return 0, parseBoom })
	if parseRes.Status != StatusError || !errors.Is(parseRes.Err, parseBoom) {
		parseT.Fatalf("expected StatusError with boom, got %+v", parseRes)
	}

	parseRes = Fetch(parseCache, "k", func() (int, error) { return 7, nil })
	if parseRes.Status != StatusSuccess || parseRes.Data != 7 || parseRes.Err != nil {
		parseT.Fatalf("expected recovery to 7, got %+v", parseRes)
	}
}

// TestSWRServesStaleThenRevalidates proves stale-while-revalidate: the stale value is
// returned synchronously while a background refresh runs, and onUpdate fires with fresh
// data.
func TestSWRServesStaleThenRevalidates(parseT *testing.T) {
	parseClock := &fakeClock{t: time.Unix(1_000_000, 0)}
	parseCache := New(WithStaleTime(time.Minute), WithClock(parseClock.now))

	parseCache.Set("k", 1)              // updatedAt = now
	parseClock.advance(2 * time.Minute) // now stale

	parseDone := make(chan Result[int], 1)
	parseSnapshot := SWR(parseCache, "k", func() (int, error) {
		return 2, nil
	}, func(parseFresh Result[int]) { parseDone <- parseFresh })

	if parseSnapshot.Data != 1 || !parseSnapshot.Stale || !parseSnapshot.Fetching {
		parseT.Fatalf("expected stale=1 fetching snapshot, got %+v", parseSnapshot)
	}

	select {
	case parseFresh := <-parseDone:
		if parseFresh.Data != 2 || parseFresh.Status != StatusSuccess {
			parseT.Fatalf("expected refreshed 2, got %+v", parseFresh)
		}
	case <-time.After(time.Second):
		parseT.Fatal("SWR background refresh never completed")
	}
}

// TestSWRSkipsRefreshWhenFresh proves a fresh entry is returned without revalidating.
func TestSWRSkipsRefreshWhenFresh(parseT *testing.T) {
	parseClock := &fakeClock{t: time.Unix(1_000_000, 0)}
	parseCache := New(WithStaleTime(time.Minute), WithClock(parseClock.now))
	parseCache.Set("k", 5)

	var parseCalled atomic.Bool
	parseRes := SWR(parseCache, "k", func() (int, error) {
		parseCalled.Store(true)
		return 9, nil
	}, nil)

	if parseRes.Data != 5 || parseRes.Stale || parseRes.Fetching {
		parseT.Fatalf("expected fresh 5, got %+v", parseRes)
	}
	if parseCalled.Load() {
		parseT.Fatal("fresh SWR must not call the fetcher")
	}
}

// TestMutateOptimisticRollbackOnError proves the optimistic value is applied, then on a
// failed mutation the cache is restored to its exact prior value and the error reported.
func TestMutateOptimisticRollbackOnError(parseT *testing.T) {
	parseCache := New(WithStaleTime(time.Hour))
	parseCache.Set("count", 10)

	parseBoom := errors.New("save failed")
	parseRes := Mutate(parseCache, "count", 11, func() (int, error) {
		// While the mutation runs, the optimistic value is visible to other readers.
		if parsePeek, _ := parseCache.Peek("count"); parsePeek != 11 {
			parseT.Fatalf("optimistic value not applied during mutation: %v", parsePeek)
		}
		return 0, parseBoom
	})

	if !errors.Is(parseRes.Err, parseBoom) {
		parseT.Fatalf("expected boom error, got %+v", parseRes)
	}
	if parsePeek, _ := parseCache.Peek("count"); parsePeek != 10 {
		parseT.Fatalf("expected rollback to 10, cache holds %v", parsePeek)
	}
}

// TestMutateCommitsOnSuccess proves a successful mutation replaces the cached value.
func TestMutateCommitsOnSuccess(parseT *testing.T) {
	parseCache := New(WithStaleTime(time.Hour))
	parseCache.Set("count", 10)

	parseRes := Mutate(parseCache, "count", 11, func() (int, error) { return 12, nil })
	if parseRes.Data != 12 || parseRes.Err != nil {
		parseT.Fatalf("expected committed 12, got %+v", parseRes)
	}
	if parsePeek, _ := parseCache.Peek("count"); parsePeek != 12 {
		parseT.Fatalf("cache should hold the server value 12, got %v", parsePeek)
	}
}

// TestMutateAsyncOptimisticThenSettles proves the fire-and-forget action: the optimistic
// value is visible immediately, and the server result is committed when fn settles.
func TestMutateAsyncOptimisticThenSettles(parseT *testing.T) {
	parseCache := New(WithStaleTime(time.Hour))
	parseCache.Set("likes", 10)

	parseRelease := make(chan struct{})
	parseSettled := make(chan Result[int], 1)
	MutateAsync(parseCache, "likes", 11, func() (int, error) {
		<-parseRelease
		return 12, nil // server's authoritative value
	}, func(parseRes Result[int]) { parseSettled <- parseRes })

	// Optimistic value is visible before the server settles.
	if parsePeek, _ := parseCache.Peek("likes"); parsePeek != 11 {
		parseT.Fatalf("expected optimistic 11 immediately, got %v", parsePeek)
	}

	close(parseRelease)
	select {
	case parseRes := <-parseSettled:
		if parseRes.Err != nil || parseRes.Data != 12 {
			parseT.Fatalf("expected committed 12, got %+v", parseRes)
		}
	case <-time.After(time.Second):
		parseT.Fatal("MutateAsync never settled")
	}
	if parsePeek, _ := parseCache.Peek("likes"); parsePeek != 12 {
		parseT.Fatalf("expected committed server value 12, got %v", parsePeek)
	}
}

// TestMutateAsyncRollsBackOnError proves an async failure restores the prior value.
func TestMutateAsyncRollsBackOnError(parseT *testing.T) {
	parseCache := New(WithStaleTime(time.Hour))
	parseCache.Set("likes", 10)

	parseSettled := make(chan Result[int], 1)
	MutateAsync(parseCache, "likes", 11, func() (int, error) {
		return 0, errors.New("server rejected")
	}, func(parseRes Result[int]) { parseSettled <- parseRes })

	select {
	case parseRes := <-parseSettled:
		if parseRes.Err == nil {
			parseT.Fatal("expected the server error to surface")
		}
	case <-time.After(time.Second):
		parseT.Fatal("MutateAsync never settled")
	}
	if parsePeek, _ := parseCache.Peek("likes"); parsePeek != 10 {
		parseT.Fatalf("expected rollback to 10 after async failure, got %v", parsePeek)
	}
}

// TestInvalidateForcesRefetch proves Invalidate makes a fresh entry refetch on next read.
func TestInvalidateForcesRefetch(parseT *testing.T) {
	parseCache := New(WithStaleTime(time.Hour))
	var parseCalls atomic.Int64
	parseFetch := func() (int, error) { parseCalls.Add(1); return int(parseCalls.Load()), nil }

	_ = Fetch(parseCache, "k", parseFetch) // calls=1
	_ = Fetch(parseCache, "k", parseFetch) // cache hit, still 1
	if parseCalls.Load() != 1 {
		parseT.Fatalf("expected 1 call before invalidate, got %d", parseCalls.Load())
	}

	parseCache.Invalidate("k")
	parseRes := Fetch(parseCache, "k", parseFetch) // calls=2
	if parseCalls.Load() != 2 || parseRes.Data != 2 {
		parseT.Fatalf("expected refetch after invalidate (calls=2,data=2), got calls=%d %+v", parseCalls.Load(), parseRes)
	}
}

// TestInvalidateAllAndKeys proves the global cache-bust stales every key and Keys returns
// a sorted snapshot of cached keys.
func TestInvalidateAllAndKeys(parseT *testing.T) {
	parseCache := New(WithStaleTime(time.Hour))
	parseCache.Set("b", 2)
	parseCache.Set("a", 1)
	parseCache.Set("c", 3)

	if parseGot := strings.Join(parseCache.Keys(), ","); parseGot != "a,b,c" {
		parseT.Fatalf("expected sorted keys a,b,c, got %q", parseGot)
	}

	// Fresh before, so a Fetch would hit cache; after InvalidateAll it must refetch.
	var parseCalls atomic.Int64
	parseCache.InvalidateAll()
	for _, parseKey := range parseCache.Keys() {
		_ = Fetch(parseCache, parseKey, func() (int, error) { parseCalls.Add(1); return 0, nil })
	}
	if parseCalls.Load() != 3 {
		parseT.Fatalf("expected all 3 keys to refetch after InvalidateAll, got %d", parseCalls.Load())
	}
}

// TestSnapshotReadsWithoutFetching proves Snapshot is a pure read: it never invokes a
// fetcher and reports idle for an unknown key.
func TestSnapshotReadsWithoutFetching(parseT *testing.T) {
	parseCache := New(WithStaleTime(time.Hour))

	if parseRes := Snapshot[int](parseCache, "missing"); parseRes.Status != StatusIdle {
		parseT.Fatalf("expected idle for an unknown key, got %+v", parseRes)
	}

	parseCache.Set("k", 7)
	parseRes := Snapshot[int](parseCache, "k")
	if parseRes.Data != 7 || parseRes.Status != StatusSuccess || parseRes.Stale {
		parseT.Fatalf("expected fresh 7, got %+v", parseRes)
	}
}

// TestInvalidatePrefixGroupsKeys proves prefix invalidation stales a whole key scope.
func TestInvalidatePrefixGroupsKeys(parseT *testing.T) {
	parseCache := New(WithStaleTime(time.Hour))
	parseCache.Set("user/42", "a")
	parseCache.Set("user/42/posts", "b")
	parseCache.Set("user/99", "c")

	parseCache.InvalidatePrefix("user/42")

	parseCache.mu.Lock()
	defer parseCache.mu.Unlock()
	if !parseCache.entries["user/42"].updatedAt.IsZero() || !parseCache.entries["user/42/posts"].updatedAt.IsZero() {
		parseT.Fatal("expected the user/42 scope to be invalidated")
	}
	if parseCache.entries["user/99"].updatedAt.IsZero() {
		parseT.Fatal("user/99 is outside the prefix and must stay fresh")
	}
}
