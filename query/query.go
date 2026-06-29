// Package query is GoWebComponents' stable query/data layer: a keyed, concurrency-safe
// cache with request de-duplication, stale-while-revalidate (SWR) refresh,
// invalidate-by-key, and one-call optimistic mutations with automatic rollback.
//
// It is the Go-native answer to TanStack Query / SWR, with two deliberate differences:
//
//   - It is EXPLICIT, not magical. You pass the cache key and the fetcher to every call;
//     nothing is auto-tracked behind your back. This matches the framework's core design
//     direction (no hidden reactive graph) and keeps every call site readable.
//   - It is PURE Go. The fetcher is an ordinary func() (T, error), so the entire layer
//     compiles and is fully unit-testable both natively and as wasm — no JS, no network
//     stubs, no fake timers needed (the clock is injectable for tests).
//
// Typical use:
//
//	c := query.New(query.WithStaleTime(30 * time.Second))
//	res := query.Fetch(c, "user/42", func() (User, error) { return api.GetUser(42) })
//	// concurrent Fetch("user/42", …) calls coalesce into ONE api.GetUser call.
package query

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Status is the lifecycle state of a cached query result.
type Status int

const (
	// StatusIdle means the key has never been fetched and holds no data.
	StatusIdle Status = iota
	// StatusLoading means a fetch is in flight and no data is cached yet.
	StatusLoading
	// StatusSuccess means data is cached (it may be stale and revalidating).
	StatusSuccess
	// StatusError means the last fetch failed and no prior data is cached.
	StatusError
)

// String renders the status for logs and diagnostics.
func (parseS Status) String() string {
	switch parseS {
	case StatusIdle:
		return "idle"
	case StatusLoading:
		return "loading"
	case StatusSuccess:
		return "success"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// Result is an immutable snapshot of a cached query at one moment.
type Result[T any] struct {
	// Data is the cached value (zero if none has been fetched successfully).
	Data T
	// Err is the most recent error for this key, or nil.
	Err error
	// Status is the lifecycle state at snapshot time.
	Status Status
	// UpdatedAt is when Data was last written, zero if never.
	UpdatedAt time.Time
	// Stale is true when Data is older than the cache's stale time (SWR will refresh).
	Stale bool
	// Fetching is true when a background or foreground fetch is in flight.
	Fetching bool
}

// entry is the cache's per-key record. All access is guarded by Cache.mu.
type entry struct {
	data      any
	err       error
	hasData   bool
	updatedAt time.Time
	flight    *flight // non-nil while a fetch is in progress (the dedupe handle)
}

// flight is one in-progress fetch that concurrent callers join instead of duplicating.
type flight struct {
	done chan struct{}
}

// Cache is a concurrency-safe keyed query cache. Create one per logical data scope
// (commonly one app-wide cache) and share it; all methods are safe for concurrent use.
type Cache struct {
	mu        sync.Mutex
	entries   map[string]*entry
	staleTime time.Duration
	now       func() time.Time
}

// Option configures a Cache at construction.
type Option func(*Cache)

// WithStaleTime sets how long fetched data is considered fresh. Within this window
// Fetch returns cached data without calling the fetcher and SWR skips revalidation.
// The default is 0: every read revalidates (concurrent reads still coalesce).
func WithStaleTime(parseD time.Duration) Option {
	return func(parseC *Cache) { parseC.staleTime = parseD }
}

// WithClock injects the clock used for freshness decisions (defaults to time.Now).
// Supply a deterministic clock to make staleness testable without sleeping, or a
// server-synchronized clock so cache TTLs agree with an authoritative time source.
func WithClock(parseNow func() time.Time) Option {
	return func(parseC *Cache) { parseC.now = parseNow }
}

// New creates an empty cache with the given options.
func New(parseOpts ...Option) *Cache {
	parseC := &Cache{
		entries: map[string]*entry{},
		now:     time.Now,
	}
	for _, parseOpt := range parseOpts {
		parseOpt(parseC)
	}
	return parseC
}

// ensureEntry returns the entry for key, creating it if absent. Caller holds mu.
func (parseC *Cache) ensureEntry(parseKey string) *entry {
	parseE := parseC.entries[parseKey]
	if parseE == nil {
		parseE = &entry{}
		parseC.entries[parseKey] = parseE
	}
	return parseE
}

// fresh reports whether the entry has data within the stale window. Caller holds mu.
func (parseC *Cache) fresh(parseE *entry) bool {
	return parseE.hasData && parseC.staleTime > 0 && parseC.now().Sub(parseE.updatedAt) < parseC.staleTime
}

// Fetch returns the cached value for key when fresh; otherwise it invokes fn — coalescing
// all concurrent Fetch calls for the same key into a single fn invocation — caches the
// result, and returns it. This is the blocking, read-through path.
func Fetch[T any](parseC *Cache, parseKey string, parseFn func() (T, error)) Result[T] {
	parseC.mu.Lock()
	parseE := parseC.ensureEntry(parseKey)

	if parseC.fresh(parseE) && parseE.flight == nil {
		parseRes := resultFromEntry[T](parseE, parseC.now(), parseC.staleTime)
		parseC.mu.Unlock()
		return parseRes
	}
	if parseE.flight != nil {
		parseFlight := parseE.flight
		parseC.mu.Unlock()
		<-parseFlight.done
		return resultAfterFlight[T](parseC, parseKey)
	}

	parseFlight := &flight{done: make(chan struct{})}
	parseE.flight = parseFlight
	parseC.mu.Unlock()

	runFetch(parseC, parseKey, parseFn, parseFlight)
	return resultAfterFlight[T](parseC, parseKey)
}

// SWR returns the cached value immediately (possibly empty or stale) and, when the data
// is missing or stale and no fetch is already running, starts a background refresh.
// onUpdate (may be nil) is called once with the fresh result when that refresh completes,
// which is where a component re-renders. Concurrent SWR/Fetch calls share one refresh.
func SWR[T any](parseC *Cache, parseKey string, parseFn func() (T, error), parseOnUpdate func(Result[T])) Result[T] {
	parseC.mu.Lock()
	parseE := parseC.ensureEntry(parseKey)
	parseSnapshot := resultFromEntry[T](parseE, parseC.now(), parseC.staleTime)

	if parseC.fresh(parseE) || parseE.flight != nil {
		parseC.mu.Unlock()
		return parseSnapshot
	}

	parseFlight := &flight{done: make(chan struct{})}
	parseE.flight = parseFlight
	parseSnapshot.Fetching = true
	parseC.mu.Unlock()

	go func() {
		runFetch(parseC, parseKey, parseFn, parseFlight)
		if parseOnUpdate != nil {
			parseOnUpdate(resultAfterFlight[T](parseC, parseKey))
		}
	}()
	return parseSnapshot
}

// Mutate applies an optimistic value to key immediately, runs fn, and on success commits
// fn's returned value to the cache — or, on error, rolls the cache back to exactly the
// pre-mutation state and reports the error in the returned Result. This is the
// optimistic-update-with-rollback path that JS query libs make you wire by hand.
func Mutate[T any](parseC *Cache, parseKey string, parseOptimistic T, parseFn func() (T, error)) Result[T] {
	parseC.mu.Lock()
	parseE := parseC.ensureEntry(parseKey)
	parsePrevData, parsePrevHas, parsePrevErr, parsePrevAt := parseE.data, parseE.hasData, parseE.err, parseE.updatedAt
	parseE.data, parseE.hasData, parseE.err, parseE.updatedAt = parseOptimistic, true, nil, parseC.now()
	parseC.mu.Unlock()

	parseVal, parseErr := parseFn()

	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	if parseErr != nil {
		parseE.data, parseE.hasData, parseE.err, parseE.updatedAt = parsePrevData, parsePrevHas, parsePrevErr, parsePrevAt
		parseRes := resultFromEntry[T](parseE, parseC.now(), parseC.staleTime)
		parseRes.Err = parseErr
		return parseRes
	}
	parseE.data, parseE.hasData, parseE.err, parseE.updatedAt = parseVal, true, nil, parseC.now()
	return resultFromEntry[T](parseE, parseC.now(), parseC.staleTime)
}

// MutateAsync applies the optimistic value to key immediately (so a Snapshot taken right
// after already reflects it) and runs fn in the background, committing fn's result on
// success or rolling back to the exact prior state on error, then invoking onSettled (may
// be nil) with the outcome. This is the fire-and-forget optimistic action: the UI updates
// now and the server reconciles later — pair fn with serverfn.Call to make a one-call
// optimistic server action.
func MutateAsync[T any](parseC *Cache, parseKey string, parseOptimistic T, parseFn func() (T, error), parseOnSettled func(Result[T])) {
	parseC.mu.Lock()
	parseE := parseC.ensureEntry(parseKey)
	parsePrevData, parsePrevHas, parsePrevErr, parsePrevAt := parseE.data, parseE.hasData, parseE.err, parseE.updatedAt
	parseE.data, parseE.hasData, parseE.err, parseE.updatedAt = parseOptimistic, true, nil, parseC.now()
	parseC.mu.Unlock()

	go func() {
		parseVal, parseErr := parseFn()
		parseC.mu.Lock()
		var parseRes Result[T]
		if parseErr != nil {
			parseE.data, parseE.hasData, parseE.err, parseE.updatedAt = parsePrevData, parsePrevHas, parsePrevErr, parsePrevAt
			parseRes = resultFromEntry[T](parseE, parseC.now(), parseC.staleTime)
			parseRes.Err = parseErr
		} else {
			parseE.data, parseE.hasData, parseE.err, parseE.updatedAt = parseVal, true, nil, parseC.now()
			parseRes = resultFromEntry[T](parseE, parseC.now(), parseC.staleTime)
		}
		parseC.mu.Unlock()
		if parseOnSettled != nil {
			parseOnSettled(parseRes)
		}
	}()
}

// Set seeds or overwrites the cached value for key (e.g. priming from SSR data or a
// mutation response), marking it freshly updated.
func (parseC *Cache) Set(parseKey string, parseData any) {
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	parseE := parseC.ensureEntry(parseKey)
	parseE.data, parseE.hasData, parseE.err, parseE.updatedAt = parseData, true, nil, parseC.now()
}

// Snapshot returns the current typed result for key WITHOUT triggering a fetch. It is
// the pure read a render uses to project current cache state into the view; pair it with
// SWR (or the ui.UseQuery hook) to drive the actual revalidation.
func Snapshot[T any](parseC *Cache, parseKey string) Result[T] {
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	parseE := parseC.entries[parseKey]
	if parseE == nil {
		return Result[T]{Status: StatusIdle}
	}
	return resultFromEntry[T](parseE, parseC.now(), parseC.staleTime)
}

// Peek returns the cached value for key without triggering a fetch.
func (parseC *Cache) Peek(parseKey string) (any, bool) {
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	if parseE := parseC.entries[parseKey]; parseE != nil && parseE.hasData {
		return parseE.data, true
	}
	return nil, false
}

// Invalidate marks key stale so the next Fetch/SWR refetches it. In-flight fetches are
// left untouched.
func (parseC *Cache) Invalidate(parseKey string) {
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	if parseE := parseC.entries[parseKey]; parseE != nil {
		parseE.updatedAt = time.Time{}
	}
}

// InvalidateAll marks every cached key stale so the next read of each refetches. This is
// the global cache-bust used on sign-in/sign-out or a window-focus/reconnect revalidation.
func (parseC *Cache) InvalidateAll() {
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	for _, parseE := range parseC.entries {
		parseE.updatedAt = time.Time{}
	}
}

// EntryInfo is a non-generic, type-erased view of one cache entry for observability — the
// devtools surface that does not know each key's concrete type.
type EntryInfo struct {
	Key       string
	HasData   bool
	Stale     bool
	Fetching  bool
	UpdatedAt time.Time
}

// Inspect returns a sorted, type-erased view of every cache entry — the data a devtools
// cache panel renders (which keys are cached, fresh/stale, currently fetching).
func (parseC *Cache) Inspect() []EntryInfo {
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	parseInfos := make([]EntryInfo, 0, len(parseC.entries))
	for parseKey, parseEntry := range parseC.entries {
		parseStale := parseEntry.hasData && (parseC.staleTime <= 0 || parseC.now().Sub(parseEntry.updatedAt) >= parseC.staleTime)
		parseInfos = append(parseInfos, EntryInfo{
			Key:       parseKey,
			HasData:   parseEntry.hasData,
			Stale:     parseStale,
			Fetching:  parseEntry.flight != nil,
			UpdatedAt: parseEntry.updatedAt,
		})
	}
	sort.Slice(parseInfos, func(parseA, parseB int) bool { return parseInfos[parseA].Key < parseInfos[parseB].Key })
	return parseInfos
}

// Keys returns a sorted snapshot of the cached keys — for devtools, introspection, and
// deciding which queries a focus/reconnect handler should revalidate.
func (parseC *Cache) Keys() []string {
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	parseKeys := make([]string, 0, len(parseC.entries))
	for parseKey := range parseC.entries {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	return parseKeys
}

// InvalidatePrefix marks every key sharing prefix stale — the by-scope invalidation that
// makes keys like "user/42/posts" cheap to refresh as a group.
func (parseC *Cache) InvalidatePrefix(parsePrefix string) {
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	for parseKey, parseE := range parseC.entries {
		if strings.HasPrefix(parseKey, parsePrefix) {
			parseE.updatedAt = time.Time{}
		}
	}
}

// callFetch runs fn, converting a panic into an error. Without this a panicking fetcher would
// unwind past runFetch's flight cleanup, leaving entry.flight non-nil and parseFlight.done open
// forever — wedging every future read of the key in StatusLoading and blocking any joined caller.
func callFetch[T any](parseFn func() (T, error)) (parseVal T, parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseErr = fmt.Errorf("query fetcher panicked: %v", parseRecovered)
		}
	}()
	return parseFn()
}

// runFetch executes fn and stores the outcome under key, then releases the flight so
// joined callers wake. Caller must NOT hold mu.
func runFetch[T any](parseC *Cache, parseKey string, parseFn func() (T, error), parseFlight *flight) {
	parseVal, parseErr := callFetch(parseFn)
	parseC.mu.Lock()
	parseE := parseC.entries[parseKey]
	if parseErr == nil {
		parseE.data, parseE.err, parseE.hasData, parseE.updatedAt = parseVal, nil, true, parseC.now()
	} else {
		parseE.err = parseErr
	}
	parseE.flight = nil
	close(parseFlight.done)
	parseC.mu.Unlock()
}

// resultAfterFlight builds a snapshot for key under the lock; used once a fetch finishes.
func resultAfterFlight[T any](parseC *Cache, parseKey string) Result[T] {
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	return resultFromEntry[T](parseC.entries[parseKey], parseC.now(), parseC.staleTime)
}

// resultFromEntry projects an entry into a typed Result. Caller holds mu.
func resultFromEntry[T any](parseE *entry, parseNow time.Time, parseStale time.Duration) Result[T] {
	var parseData T
	if parseE.hasData {
		parseData, _ = parseE.data.(T)
	}
	parseStatus := StatusIdle
	switch {
	case parseE.hasData:
		parseStatus = StatusSuccess
	case parseE.flight != nil:
		parseStatus = StatusLoading
	case parseE.err != nil:
		parseStatus = StatusError
	}
	parseIsStale := parseE.hasData && (parseStale <= 0 || parseNow.Sub(parseE.updatedAt) >= parseStale)
	return Result[T]{
		Data:      parseData,
		Err:       parseE.err,
		Status:    parseStatus,
		UpdatedAt: parseE.updatedAt,
		Stale:     parseIsStale,
		Fetching:  parseE.flight != nil,
	}
}
