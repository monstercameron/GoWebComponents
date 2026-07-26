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
	// gen increments on every data write. Async completions (background
	// fetches, optimistic mutations) capture it at their start and commit only
	// if it is unchanged, so the SEMANTICALLY newest write wins instead of the
	// slowest-to-finish one.
	gen uint64
	// lastUse orders entries for eviction. A counter rather than a timestamp so
	// recency does not depend on the injected clock, which a test may freeze.
	lastUse uint64
}

// flight is one in-progress fetch that concurrent callers join instead of duplicating.
type flight struct {
	done chan struct{}
	gen  uint64 // entry.gen when this fetch started
	// waiters are the SWR onUpdate callbacks of every caller that started or
	// joined this fetch. runFetch invokes them all once, after completion,
	// outside the lock — so a component that joins an in-flight fetch is still
	// notified when the fresh data lands. Guarded by Cache.mu.
	waiters []func()
}

// Cache is a concurrency-safe keyed query cache. Create one per logical data scope
// (commonly one app-wide cache) and share it; all methods are safe for concurrent use.
type Cache struct {
	mu        sync.Mutex
	entries   map[string]*entry
	staleTime time.Duration
	now       func() time.Time
	// maxEntries bounds the map; useClock stamps entry.lastUse for eviction
	// order. See DefaultMaxEntries.
	maxEntries int
	useClock   uint64
}

// DefaultMaxEntries bounds a cache that does not set its own limit.
//
// The map had no bound at all, and ensureEntry creates a record on every read
// path — including misses and fetches that error — so a parameterized key
// pattern retained one entry per distinct key forever. Search-as-you-type is the
// clearest case: one key per keystroke, none ever collected.
//
// A COUNT bound rather than a time-to-live, deliberately. A default TTL would
// change behaviour for every existing caller, including apps whose working set
// is a handful of stable keys that are supposed to stay cached. A count only
// engages at a scale where unbounded growth is a bug rather than a working set,
// and the entry it drops is the least recently used — by definition the one
// nothing is looking at. Apps that want time-based collection can still call
// Evict / EvictPrefix on their own schedule.
const DefaultMaxEntries = 512

// WithMaxEntries bounds how many keys the cache retains, evicting the
// least-recently-used entry when a new key would exceed the limit. The default
// is DefaultMaxEntries. Zero or negative disables the bound entirely, which
// restores the previous unbounded behaviour — appropriate only when the key
// space is known to be small and fixed.
func WithMaxEntries(parseMax int) Option {
	return func(parseC *Cache) { parseC.maxEntries = parseMax }
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
		entries:    map[string]*entry{},
		now:        time.Now,
		maxEntries: DefaultMaxEntries,
	}
	for _, parseOpt := range parseOpts {
		parseOpt(parseC)
	}
	return parseC
}

// ensureEntry returns the entry for key, creating it if absent, and marks it as
// the most recently used. Caller holds mu.
func (parseC *Cache) ensureEntry(parseKey string) *entry {
	parseE := parseC.entries[parseKey]
	isNew := parseE == nil
	if isNew {
		parseE = &entry{}
		parseC.entries[parseKey] = parseE
	}
	// Stamped BEFORE any eviction: a brand-new entry carries lastUse 0 until it
	// is, which makes it the coldest thing in the map and the first candidate to
	// be dropped — the new key would evict itself and the fetch would never be
	// cached.
	parseC.markUsedLocked(parseE)
	if isNew {
		parseC.evictLeastRecentlyUsedLocked()
	}
	return parseE
}

// markUsedLocked stamps one entry as the most recently used. Caller holds mu.
func (parseC *Cache) markUsedLocked(parseE *entry) {
	parseC.useClock++
	parseE.lastUse = parseC.useClock
}

// evictLeastRecentlyUsedLocked drops the coldest entry while the cache is over
// its bound. Caller holds mu.
//
// An entry with a fetch in flight is never chosen: callers are blocked on that
// flight's done channel and resolve by reading the entry back, so evicting it
// would strand them on a record nothing will ever complete. If EVERY entry is
// in flight the cache is briefly allowed over its bound rather than breaking
// that guarantee — a transient state that resolves as the flights land.
func (parseC *Cache) evictLeastRecentlyUsedLocked() {
	if parseC.maxEntries <= 0 {
		return
	}
	for len(parseC.entries) > parseC.maxEntries {
		parseColdestKey := ""
		parseColdestUse := uint64(0)
		for parseKey, parseEntry := range parseC.entries {
			if parseEntry.flight != nil {
				continue
			}
			if parseColdestKey == "" || parseEntry.lastUse < parseColdestUse {
				parseColdestKey = parseKey
				parseColdestUse = parseEntry.lastUse
			}
		}
		if parseColdestKey == "" {
			return
		}
		delete(parseC.entries, parseColdestKey)
	}
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

	parseFlight := &flight{done: make(chan struct{}), gen: parseE.gen}
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

	// Fresh data: no refresh needed, so onUpdate is intentionally not registered.
	if parseC.fresh(parseE) {
		parseC.mu.Unlock()
		return parseSnapshot
	}

	// A refresh is already in flight: JOIN it. Register this caller's onUpdate as
	// a waiter on the existing flight so it is notified when that fetch completes
	// — previously a joining caller's onUpdate was silently dropped and its
	// component never re-rendered on the fresh data.
	if parseE.flight != nil {
		if parseOnUpdate != nil {
			parseE.flight.waiters = append(parseE.flight.waiters, func() {
				parseOnUpdate(resultAfterFlight[T](parseC, parseKey))
			})
		}
		parseC.mu.Unlock()
		return parseSnapshot
	}

	parseFlight := &flight{done: make(chan struct{}), gen: parseE.gen}
	if parseOnUpdate != nil {
		parseFlight.waiters = append(parseFlight.waiters, func() {
			parseOnUpdate(resultAfterFlight[T](parseC, parseKey))
		})
	}
	parseE.flight = parseFlight
	parseSnapshot.Fetching = true
	parseC.mu.Unlock()

	go runFetch(parseC, parseKey, parseFn, parseFlight)
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
	parseE.gen++
	parseGen := parseE.gen
	parseC.mu.Unlock()

	parseVal, parseErr := parseFn()

	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	if parseE.gen != parseGen {
		// A newer write (another mutation, Set, or a completed fetch) landed
		// while fn ran; committing or rolling back would clobber it. Report
		// the outcome against the current state instead.
		parseRes := resultFromEntry[T](parseE, parseC.now(), parseC.staleTime)
		if parseErr != nil {
			parseRes.Err = parseErr
		}
		return parseRes
	}
	if parseErr != nil {
		parseE.data, parseE.hasData, parseE.err, parseE.updatedAt = parsePrevData, parsePrevHas, parsePrevErr, parsePrevAt
		parseE.gen++
		parseRes := resultFromEntry[T](parseE, parseC.now(), parseC.staleTime)
		parseRes.Err = parseErr
		return parseRes
	}
	parseE.data, parseE.hasData, parseE.err, parseE.updatedAt = parseVal, true, nil, parseC.now()
	parseE.gen++
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
	parseE.gen++
	parseGen := parseE.gen
	parseC.mu.Unlock()

	go func() {
		parseVal, parseErr := parseFn()
		parseC.mu.Lock()
		var parseRes Result[T]
		switch {
		case parseE.gen != parseGen:
			// A newer write landed while fn ran; leave it in place (see Mutate).
			parseRes = resultFromEntry[T](parseE, parseC.now(), parseC.staleTime)
			if parseErr != nil {
				parseRes.Err = parseErr
			}
		case parseErr != nil:
			parseE.data, parseE.hasData, parseE.err, parseE.updatedAt = parsePrevData, parsePrevHas, parsePrevErr, parsePrevAt
			parseE.gen++
			parseRes = resultFromEntry[T](parseE, parseC.now(), parseC.staleTime)
			parseRes.Err = parseErr
		default:
			parseE.data, parseE.hasData, parseE.err, parseE.updatedAt = parseVal, true, nil, parseC.now()
			parseE.gen++
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
	parseE.gen++
}

// Evict removes key from the cache entirely, releasing its retained value.
// Unlike Invalidate (which keeps the value and marks it stale), Evict is for
// keys that will not be read again — pagination cursors, per-item detail keys,
// search strings — so a long-lived session does not grow the cache without
// bound. An in-flight fetch for the key completes harmlessly: its result is
// dropped (the entry is gone) and joined callers wake normally.
func (parseC *Cache) Evict(parseKey string) {
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	delete(parseC.entries, parseKey)
}

// EvictPrefix removes every key sharing prefix — the by-scope sibling of Evict.
func (parseC *Cache) EvictPrefix(parsePrefix string) {
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	for parseKey := range parseC.entries {
		if strings.HasPrefix(parseKey, parsePrefix) {
			delete(parseC.entries, parseKey)
		}
	}
}

// EvictAll removes every cached entry, releasing all retained values.
func (parseC *Cache) EvictAll() {
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	clear(parseC.entries)
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
	// A read is a use. A component that reads a still-fresh key every render
	// never reaches a fetch path, so without this its entry would look colder on
	// every tick and eventually be evicted out from under an active reader.
	parseC.markUsedLocked(parseE)
	return resultFromEntry[T](parseE, parseC.now(), parseC.staleTime)
}

// Peek returns the cached value for key without triggering a fetch.
func (parseC *Cache) Peek(parseKey string) (any, bool) {
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	if parseE := parseC.entries[parseKey]; parseE != nil && parseE.hasData {
		parseC.markUsedLocked(parseE)
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
// joined callers wake. Caller must NOT hold mu. The outcome commits only when no
// newer write (Set/Mutate) landed while the fetch was in flight, and only when the
// entry still exists (it may have been Evicted).
func runFetch[T any](parseC *Cache, parseKey string, parseFn func() (T, error), parseFlight *flight) {
	parseVal, parseErr := callFetch(parseFn)
	parseC.mu.Lock()
	// Commit only when the entry still OWNS this flight. If the key was Evicted
	// and later re-created, the map holds a fresh entry whose flight differs (and
	// whose gen restarted at 0, so the bare gen check could ALIAS this stale
	// flight's captured gen — the ABA poisoning). Requiring flight identity drops
	// this stale result harmlessly onto the recreated entry.
	if parseE := parseC.entries[parseKey]; parseE != nil && parseE.flight == parseFlight {
		if parseE.gen == parseFlight.gen {
			if parseErr == nil {
				parseE.data, parseE.err, parseE.hasData, parseE.updatedAt = parseVal, nil, true, parseC.now()
				parseE.gen++
			} else {
				parseE.err = parseErr
			}
		}
		parseE.flight = nil
	}
	// Capture and clear the waiters under the lock, then invoke them after
	// releasing it — a waiter re-reads the cache (resultAfterFlight locks).
	parseWaiters := parseFlight.waiters
	parseFlight.waiters = nil
	close(parseFlight.done)
	parseC.mu.Unlock()

	for _, parseWaiter := range parseWaiters {
		parseWaiter()
	}
}

// resultAfterFlight builds a snapshot for key under the lock; used once a fetch finishes.
func resultAfterFlight[T any](parseC *Cache, parseKey string) Result[T] {
	parseC.mu.Lock()
	defer parseC.mu.Unlock()
	return resultFromEntry[T](parseC.entries[parseKey], parseC.now(), parseC.staleTime)
}

// resultFromEntry projects an entry into a typed Result. Caller holds mu.
// A nil entry (evicted mid-flight) reports StatusIdle.
func resultFromEntry[T any](parseE *entry, parseNow time.Time, parseStale time.Duration) Result[T] {
	if parseE == nil {
		return Result[T]{Status: StatusIdle}
	}
	var parseData T
	isTypeMismatch := false
	if parseE.hasData {
		if parseTyped, parseOk := parseE.data.(T); parseOk {
			parseData = parseTyped
		} else {
			isTypeMismatch = true
		}
	}
	parseStatus := StatusIdle
	parseErr := parseE.err
	switch {
	case isTypeMismatch:
		// The key was populated with a different concrete type (key reused
		// across features). Silently reporting Success with a zero value hid
		// the mismatch entirely.
		parseStatus = StatusError
		parseErr = fmt.Errorf("query: cached value has type %T, not the requested type", parseE.data)
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
		Err:       parseErr,
		Status:    parseStatus,
		UpdatedAt: parseE.updatedAt,
		Stale:     parseIsStale,
		Fetching:  parseE.flight != nil,
	}
}
