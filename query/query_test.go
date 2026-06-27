package query

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

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
