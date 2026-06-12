package flags

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// parseFixedTime is a stable reference instant used by clock-injecting tests.
var parseFixedTime = time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

// parseSetA and parseSetB are two distinct non-empty Sets used across tests.
var parseSetA = BuildSet(
	map[string]Flag{"alpha": {Enabled: true, Value: "on", Reason: "test"}},
	nil,
)

var parseSetB = BuildSet(
	map[string]Flag{
		"alpha": {Enabled: true, Value: "updated", Reason: "remote"},
		"beta":  {Enabled: false, Value: "", Reason: "killed"},
	},
	nil,
)

// parseSuccessFetch returns parseSet unconditionally.
func parseSuccessFetch(parseSet Set) RemoteFetchFunc {
	return func(_ context.Context) (Set, error) {
		return parseSet, nil
	}
}

// parseErrorFetch always returns the given error.
func parseErrorFetch(parseErr error) RemoteFetchFunc {
	return func(_ context.Context) (Set, error) {
		return Set{}, parseErr
	}
}

// TestRemoteRefreshUpdatesCurrentAndNotifiesSubscribers verifies that a
// successful Refresh replaces Current and delivers the new Set to all
// registered subscribers.
func TestRemoteRefreshUpdatesCurrentAndNotifiesSubscribers(parseT *testing.T) {
	parseP := NewRemoteProvider(parseSetA, parseSuccessFetch(parseSetB))

	var parseGot Set
	var parseCalled int32
	parseUnsub := parseP.Subscribe(func(parseS Set) {
		atomic.StoreInt32(&parseCalled, 1)
		parseGot = parseS
	})
	defer parseUnsub()

	if parseErr := parseP.Refresh(context.Background()); parseErr != nil {
		parseT.Fatalf("unexpected error: %v", parseErr)
	}

	parseCurrent := parseP.Current()
	if parseCurrent.GetValue("alpha", "") != "updated" {
		parseT.Fatalf("Current not updated: %+v", parseCurrent)
	}
	if atomic.LoadInt32(&parseCalled) == 0 {
		parseT.Fatal("subscriber was not notified")
	}
	if parseGot.GetValue("alpha", "") != "updated" {
		parseT.Fatalf("subscriber received wrong Set: %+v", parseGot)
	}
	if parseP.LastError() != nil {
		parseT.Fatalf("LastError should be nil after success, got %v", parseP.LastError())
	}
}

// TestRemoteRefreshFailureKeepsLastKnownGood verifies that a failed Refresh
// preserves the previous Current, records LastError, and that IsStale
// becomes true once the injected clock advances past the max age.
func TestRemoteRefreshFailureKeepsLastKnownGood(parseT *testing.T) {
	parseSentinel := errors.New("network down")

	parseP := NewRemoteProvider(parseSetA, parseErrorFetch(parseSentinel))
	parseP.setNow(func() time.Time { return parseFixedTime })

	parseErr := parseP.Refresh(context.Background())
	if parseErr == nil {
		parseT.Fatal("expected error from failed fetch")
	}
	if !errors.Is(parseP.LastError(), parseSentinel) {
		parseT.Fatalf("LastError mismatch: %v", parseP.LastError())
	}

	// Current must still be the original seed.
	parseCurrent := parseP.Current()
	if parseCurrent.GetValue("alpha", "") != "on" {
		parseT.Fatalf("Current was clobbered after failed refresh: %+v", parseCurrent)
	}

	// Not yet stale at T+0.
	if parseP.IsStale(5*time.Minute, parseFixedTime) {
		parseT.Fatal("should not be stale immediately")
	}

	// Advance clock past maxAge.
	parseLater := parseFixedTime.Add(6 * time.Minute)
	if !parseP.IsStale(5*time.Minute, parseLater) {
		parseT.Fatal("should be stale after max age elapsed")
	}
}

// TestParseRemoteSetRoundTrip verifies that a valid JSON payload decodes
// into a Set with all fields intact, and that a malformed payload returns
// an error without disturbing a provider seeded with the good Set.
func TestParseRemoteSetRoundTrip(parseT *testing.T) {
	parseRaw := []byte(`{
		"flags": {
			"dark-mode": {"enabled": true, "value": "forced", "reason": "config"}
		},
		"experiments": {
			"checkout": {
				"enabled": true,
				"salt": "v2",
				"variants": [
					{"name": "control", "value": "A", "weight": 50},
					{"name": "treatment", "value": "B", "weight": 50}
				]
			}
		}
	}`)

	parseSet, parseErr := ParseRemoteSet(parseRaw)
	if parseErr != nil {
		parseT.Fatalf("unexpected error: %v", parseErr)
	}
	if !parseSet.GetEnabled("dark-mode", false) {
		parseT.Fatal("dark-mode flag not enabled")
	}
	if parseSet.GetValue("dark-mode", "") != "forced" {
		parseT.Fatal("dark-mode value wrong")
	}
	parseAssign := parseSet.GetAssignment("checkout", "user-42")
	if parseAssign.Reason != "assigned" {
		parseT.Fatalf("expected assigned experiment: %+v", parseAssign)
	}

	// Malformed payload.
	parseBad, parseBadErr := ParseRemoteSet([]byte(`{not valid json`))
	if parseBadErr == nil {
		parseT.Fatal("expected error for malformed JSON")
	}
	if parseBad.Flags != nil || parseBad.Experiments != nil {
		parseT.Fatal("malformed parse should return zero Set")
	}

	// Provider is unaffected — last-known-good still intact.
	parseP := NewRemoteProvider(parseSet, parseErrorFetch(errors.New("bad payload")))
	_ = parseP.Refresh(context.Background())
	if !parseP.Current().GetEnabled("dark-mode", false) {
		parseT.Fatal("provider corrupted after malformed refresh")
	}
}

// TestKillSwitchFlips verifies the kill-switch semantics: a Set with a flag
// disabled makes IsKilled true and notifies subscribers; re-enabling flips
// it back to false.
func TestKillSwitchFlips(parseT *testing.T) {
	parseKilledSet := BuildSet(
		map[string]Flag{"beta": {Enabled: false, Reason: "kill-switch"}},
		nil,
	)
	parseEnabledSet := BuildSet(
		map[string]Flag{"beta": {Enabled: true, Reason: "re-enabled"}},
		nil,
	)

	parseP := NewRemoteProvider(parseSetA, parseSuccessFetch(parseKilledSet))

	var parseNotified int32
	parseUnsub := parseP.Subscribe(func(_ Set) { atomic.AddInt32(&parseNotified, 1) })
	defer parseUnsub()

	// First refresh: deliver killed set.
	if parseErr := parseP.Refresh(context.Background()); parseErr != nil {
		parseT.Fatalf("refresh error: %v", parseErr)
	}
	if !parseP.IsKilled("beta") {
		parseT.Fatal("beta should be killed after first refresh")
	}
	if atomic.LoadInt32(&parseNotified) != 1 {
		parseT.Fatalf("expected 1 subscriber notification, got %d", parseNotified)
	}

	// Second refresh: re-enable beta.
	parseP.fetch = parseSuccessFetch(parseEnabledSet)
	if parseErr := parseP.Refresh(context.Background()); parseErr != nil {
		parseT.Fatalf("refresh error: %v", parseErr)
	}
	if parseP.IsKilled("beta") {
		parseT.Fatal("beta should not be killed after re-enable")
	}
	if atomic.LoadInt32(&parseNotified) != 2 {
		parseT.Fatalf("expected 2 subscriber notifications, got %d", parseNotified)
	}

	// A flag that is completely absent is not killed.
	if parseP.IsKilled("nonexistent") {
		parseT.Fatal("absent flag should not be killed")
	}
}

// TestSubscribeUnsubscribeIsLeakFree verifies that after all unsubscribe
// functions are called the subscriber registry returns to size zero, and
// that a panicking subscriber does not stop delivery to the remaining ones.
func TestSubscribeUnsubscribeIsLeakFree(parseT *testing.T) {
	parseP := NewRemoteProvider(parseSetA, parseSuccessFetch(parseSetB))

	// Register a panicking subscriber and a counting one.
	var parseDelivered int32
	parseUnsub1 := parseP.Subscribe(func(_ Set) { panic("intentional test panic") })
	parseUnsub2 := parseP.Subscribe(func(_ Set) { atomic.AddInt32(&parseDelivered, 1) })

	if parseP.subscriberCount() != 2 {
		parseT.Fatalf("expected 2 subscribers, got %d", parseP.subscriberCount())
	}

	// Refresh must deliver to the counting subscriber despite the panic.
	if parseErr := parseP.Refresh(context.Background()); parseErr != nil {
		parseT.Fatalf("unexpected error: %v", parseErr)
	}
	if atomic.LoadInt32(&parseDelivered) != 1 {
		parseT.Fatalf("counting subscriber not called: delivered=%d", parseDelivered)
	}

	// Unsubscribe both — registry must be empty.
	parseUnsub1()
	parseUnsub2()
	if parseP.subscriberCount() != 0 {
		parseT.Fatalf("expected 0 subscribers after unsubscribe, got %d", parseP.subscriberCount())
	}

	// Subsequent refresh delivers to nobody (no crash).
	_ = parseP.Refresh(context.Background())
}

// TestPollKeepsThroughFailuresThenSucceeds verifies that Poll swallows
// individual Refresh errors (outage resilience), eventually reflects a
// successful fetch, and returns ctx.Err() on cancellation.
func TestPollKeepsThroughFailuresThenSucceeds(parseT *testing.T) {
	var parseFetchCount int32

	// Fail twice then succeed.
	parseSentinel := errors.New("transient")
	parseFetch := func(_ context.Context) (Set, error) {
		parseN := atomic.AddInt32(&parseFetchCount, 1)
		if parseN <= 2 {
			return Set{}, parseSentinel
		}
		return parseSetB, nil
	}

	parseP := NewRemoteProvider(parseSetA, parseFetch)

	// Count sleep calls to know when to cancel.
	var parseSleepCount int32
	parseCtx, parseCancel := context.WithCancel(context.Background())

	parseSleep := func(parseC context.Context, _ time.Duration) error {
		parseN := atomic.AddInt32(&parseSleepCount, 1)
		// Cancel after the third sleep (i.e., after the successful fetch).
		if parseN >= 3 {
			parseCancel()
			return parseC.Err()
		}
		return nil
	}

	parseErr := parseP.Poll(parseCtx, time.Millisecond, parseSleep)
	if !errors.Is(parseErr, context.Canceled) {
		parseT.Fatalf("expected context.Canceled, got %v", parseErr)
	}

	// Must have fetched at least 3 times (two failures + one success).
	if atomic.LoadInt32(&parseFetchCount) < 3 {
		parseT.Fatalf("expected ≥3 fetches, got %d", parseFetchCount)
	}

	// After the successful fetch the current set must reflect parseSetB.
	if parseP.Current().GetValue("alpha", "") != "updated" {
		parseT.Fatalf("provider did not reflect successful fetch: %+v", parseP.Current())
	}

	// After the two failures LastError was set; after success it was cleared.
	if parseP.LastError() != nil {
		parseT.Fatalf("LastError should be nil after success, got %v", parseP.LastError())
	}
}
