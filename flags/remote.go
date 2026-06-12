package flags

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// RemoteFetchFunc is the caller-supplied transport hook that retrieves a
// fresh Set from any source (HTTP poll, SSE stream, file watch, etc.).
// The package is transport-agnostic; it calls this function and reacts to
// the result.
type RemoteFetchFunc func(parseCtx context.Context) (Set, error)

// RemoteProvider holds a live, mutex-guarded snapshot of a remotely
// managed flag Set, and notifies subscribers on every successful refresh.
// A remote payload that sets a flag's Enabled field to false acts as a
// kill switch: IsKilled returns true and all subscribers are notified
// without requiring a page reload.
type RemoteProvider struct {
	mu          sync.Mutex
	fetch       RemoteFetchFunc
	current     Set
	lastUpdated time.Time
	lastErr     error
	subscribers map[int]func(Set)
	nextID      int
	now         func() time.Time
}

// NewRemoteProvider returns a RemoteProvider seeded with parseInitial as
// the last-known-good Set. parseFetch supplies updates; it is called once
// per Refresh and once per Poll interval.
func NewRemoteProvider(parseInitial Set, parseFetch RemoteFetchFunc) *RemoteProvider {
	parseP := &RemoteProvider{
		fetch:       parseFetch,
		current:     BuildSet(parseInitial.Flags, parseInitial.Experiments),
		subscribers: make(map[int]func(Set)),
		now:         time.Now,
	}
	parseP.lastUpdated = parseP.now()
	return parseP
}

// setNow replaces the clock used by Age and IsStale, and resets
// lastUpdated to the new clock's current reading so that staleness
// calculations in tests start from a known baseline. Intended only for
// unit tests; not part of the public API.
func (parseP *RemoteProvider) setNow(parseClock func() time.Time) {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	parseP.now = parseClock
	parseP.lastUpdated = parseClock()
}

// subscriberCount returns the number of currently registered subscribers.
// It is exported only to the same package (unexported to callers) and is
// intended for use in unit tests.
func (parseP *RemoteProvider) subscriberCount() int {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	return len(parseP.subscribers)
}

// Refresh calls the injected fetch function once. On success it replaces
// the current Set, records the update time, clears the last error, and
// notifies all subscribers. On failure it preserves the previous Set and
// last-known-good time, stores the error, and returns it. A failed refresh
// never clobbers good values.
func (parseP *RemoteProvider) Refresh(parseCtx context.Context) error {
	parseNext, parseErr := parseP.fetch(parseCtx)
	if parseErr != nil {
		parseP.mu.Lock()
		parseP.lastErr = parseErr
		parseP.mu.Unlock()
		return parseErr
	}

	parseCopied := BuildSet(parseNext.Flags, parseNext.Experiments)

	parseP.mu.Lock()
	parseP.current = parseCopied
	parseP.lastUpdated = parseP.now()
	parseP.lastErr = nil
	parseHandlers := make([]func(Set), 0, len(parseP.subscribers))
	for _, parseFn := range parseP.subscribers {
		parseHandlers = append(parseHandlers, parseFn)
	}
	parseP.mu.Unlock()

	for _, parseFn := range parseHandlers {
		parseCallSubscriber(parseFn, parseCopied)
	}
	return nil
}

// parseCallSubscriber invokes parseFn, recovering from any panic so that
// one misbehaving subscriber cannot stop delivery to the others.
func parseCallSubscriber(parseFn func(Set), parseSet Set) {
	defer func() { recover() }() //nolint:errcheck
	parseFn(parseSet)
}

// Current returns the current last-known-good Set. It is always valid;
// even after a failed Refresh it holds the last successfully fetched value.
func (parseP *RemoteProvider) Current() Set {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	return parseP.current
}

// LastError returns the error from the most recent failed Refresh, or nil
// if the last Refresh succeeded.
func (parseP *RemoteProvider) LastError() error {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	return parseP.lastErr
}

// Age returns the time elapsed since the last successful Refresh, measured
// relative to parseNow. Inject the same clock you pass to setNow when
// testing staleness.
func (parseP *RemoteProvider) Age(parseNow time.Time) time.Duration {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	return parseNow.Sub(parseP.lastUpdated)
}

// IsStale reports whether the last successful Refresh is older than
// parseMaxAge as of parseNow.
func (parseP *RemoteProvider) IsStale(parseMaxAge time.Duration, parseNow time.Time) bool {
	return parseP.Age(parseNow) > parseMaxAge
}

// Subscribe registers parseHandler to be called after every successful
// Refresh. It returns an unsubscribe function; calling it removes the
// handler and is leak-free (the internal registry returns to baseline size).
// A panicking handler does not interrupt delivery to the remaining ones.
func (parseP *RemoteProvider) Subscribe(parseHandler func(Set)) (parseUnsubscribe func()) {
	parseP.mu.Lock()
	parseID := parseP.nextID
	parseP.nextID++
	parseP.subscribers[parseID] = parseHandler
	parseP.mu.Unlock()

	return func() {
		parseP.mu.Lock()
		delete(parseP.subscribers, parseID)
		parseP.mu.Unlock()
	}
}

// IsKilled reports whether parseFeature is actively killed. A flag is
// considered killed when the current Set contains it AND its Enabled field
// is false. A missing flag is not killed — it is simply absent.
//
// Kill-switch pattern: deploy a remote payload with the flag present and
// Enabled set to false. Components that gate a subtree on IsKilled and
// react via Subscribe will disable themselves immediately without a reload.
// Re-enabling the flag (Enabled true) in a subsequent payload un-kills it.
func (parseP *RemoteProvider) IsKilled(parseFeature string) bool {
	parseP.mu.Lock()
	parseSet := parseP.current
	parseP.mu.Unlock()

	if parseSet.Flags == nil {
		return false
	}
	parseFlag, parseOk := parseSet.Flags[parseFeature]
	if !parseOk {
		return false
	}
	return !parseFlag.Enabled
}

// Poll calls Refresh on every parseInterval tick until parseCtx is
// cancelled. parseSleep is called between iterations; inject a no-op in
// tests so they run instantly. Errors from individual Refresh calls are
// recorded in LastError but do not stop the loop (outage resilience).
// Poll returns parseCtx.Err() when the context is done.
func (parseP *RemoteProvider) Poll(
	parseCtx context.Context,
	parseInterval time.Duration,
	parseSleep func(context.Context, time.Duration) error,
) error {
	for {
		_ = parseP.Refresh(parseCtx)
		if parseErr := parseSleep(parseCtx, parseInterval); parseErr != nil {
			return parseCtx.Err()
		}
		select {
		case <-parseCtx.Done():
			return parseCtx.Err()
		default:
		}
	}
}

// remotePayload mirrors the JSON shape expected by ParseRemoteSet.
type remotePayload struct {
	Flags       map[string]remoteFlag       `json:"flags"`
	Experiments map[string]remoteExperiment `json:"experiments"`
}

type remoteFlag struct {
	Enabled bool   `json:"enabled"`
	Value   string `json:"value"`
	Reason  string `json:"reason"`
}

type remoteExperiment struct {
	Enabled  bool            `json:"enabled"`
	Salt     string          `json:"salt"`
	Variants []remoteVariant `json:"variants"`
}

type remoteVariant struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Weight int    `json:"weight"`
}

// ParseRemoteSet decodes parseRaw from the canonical remote payload shape:
//
//	{"flags": {"name": {"enabled": true, "value": "...", "reason": "..."}},
//	 "experiments": {"name": {"enabled": true, "salt": "...", "variants": [...]}}}
//
// A malformed payload returns a descriptive error; the caller should keep
// the last-known-good Set rather than replacing it. ParseRemoteSet never
// panics; it is safe to call from any goroutine.
func ParseRemoteSet(parseRaw []byte) (Set, error) {
	var parsePayload remotePayload
	if parseErr := json.Unmarshal(parseRaw, &parsePayload); parseErr != nil {
		return Set{}, parseErr
	}

	parseFlags := make(map[string]Flag, len(parsePayload.Flags))
	for parseName, parseF := range parsePayload.Flags {
		parseFlags[parseName] = Flag(parseF)
	}

	parseExperiments := make(map[string]Experiment, len(parsePayload.Experiments))
	for parseName, parseE := range parsePayload.Experiments {
		parseVariants := make([]Variant, len(parseE.Variants))
		for parseI, parseV := range parseE.Variants {
			parseVariants[parseI] = Variant(parseV)
		}
		parseExperiments[parseName] = Experiment{
			Enabled:  parseE.Enabled,
			Salt:     parseE.Salt,
			Variants: parseVariants,
		}
	}

	return BuildSet(parseFlags, parseExperiments), nil
}
