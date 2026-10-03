//go:build !js || !wasm

package fetch

import (
	"testing"
	"testing/synctest"
	"time"
)

// TestRealtimeControllerShortLivedConnectionsExhaustReconnectBudget proves a server that accepts
// the connection and drops it straight away cannot keep the retry budget topped up forever: opens
// shorter than minStableDuration do not reset the consecutive reconnect count.
func TestRealtimeControllerShortLivedConnectionsExhaustReconnectBudget(parseT *testing.T) {
	synctest.Test(parseT, func(parseT *testing.T) {
		parseNow := time.Date(2026, time.June, 13, 10, 0, 0, 0, time.UTC)
		parseController, parseState, parseCalls := newRealtimeTestController("WebSocket", realtimeResolvedOptions{
			reconnect:         true,
			maxReconnects:     2,
			initialBackoff:    5 * time.Millisecond,
			maxBackoff:        20 * time.Millisecond,
			backoffFactor:     2,
			minStableDuration: 100 * time.Millisecond,
			maxMessages:       4,
			maxErrors:         4,
			now:               func() time.Time { return parseNow },
		})
		parseT.Cleanup(func() { parseController.stop(true) })

		parseController.start()
		parseCalls.get(0).parseCallbacks.handleOpen()
		parseNow = parseNow.Add(10 * time.Millisecond)
		parseCalls.get(0).parseCallbacks.handleClose()
		if parseGot := parseState.Get(); parseGot.ReconnectAttempts != 1 {
			parseT.Fatalf("first short-lived drop: ReconnectAttempts = %d, want 1", parseGot.ReconnectAttempts)
		}

		waitFetchTestCondition(parseT, time.Second, func() bool { return parseCalls.count() == 2 })
		parseCalls.get(1).parseCallbacks.handleOpen()
		if parseGot := parseState.Get(); parseGot.ReconnectAttempts != 1 {
			parseT.Fatalf("a reopened short-lived connection must keep the count, got %d", parseGot.ReconnectAttempts)
		}
		parseNow = parseNow.Add(10 * time.Millisecond)
		parseCalls.get(1).parseCallbacks.handleClose()
		if parseGot := parseState.Get(); parseGot.ReconnectAttempts != 2 {
			parseT.Fatalf("second short-lived drop: ReconnectAttempts = %d, want 2", parseGot.ReconnectAttempts)
		}

		waitFetchTestCondition(parseT, time.Second, func() bool { return parseCalls.count() == 3 })
		parseCalls.get(2).parseCallbacks.handleOpen()
		parseNow = parseNow.Add(10 * time.Millisecond)
		parseCalls.get(2).parseCallbacks.handleClose()
		if parseGot := parseState.Get(); parseGot.Status != RealtimeClosed || !parseGot.Closed {
			parseT.Fatalf("repeated short-lived connections must exhaust the budget and close, got %+v", parseGot)
		}
	})
}

// TestRealtimeControllerStableConnectionResetsAfterShortLivedOnes proves one connection that stays
// up for at least minStableDuration restores the full retry budget.
func TestRealtimeControllerStableConnectionResetsAfterShortLivedOnes(parseT *testing.T) {
	synctest.Test(parseT, func(parseT *testing.T) {
		parseNow := time.Date(2026, time.June, 13, 10, 0, 0, 0, time.UTC)
		parseController, parseState, parseCalls := newRealtimeTestController("WebSocket", realtimeResolvedOptions{
			reconnect:         true,
			maxReconnects:     3,
			initialBackoff:    5 * time.Millisecond,
			maxBackoff:        20 * time.Millisecond,
			backoffFactor:     2,
			minStableDuration: 100 * time.Millisecond,
			maxMessages:       4,
			maxErrors:         4,
			now:               func() time.Time { return parseNow },
		})
		parseT.Cleanup(func() { parseController.stop(true) })

		parseController.start()
		parseCalls.get(0).parseCallbacks.handleOpen()
		parseNow = parseNow.Add(10 * time.Millisecond)
		parseCalls.get(0).parseCallbacks.handleClose()
		waitFetchTestCondition(parseT, time.Second, func() bool { return parseCalls.count() == 2 })

		parseCalls.get(1).parseCallbacks.handleOpen()
		parseNow = parseNow.Add(500 * time.Millisecond)
		parseCalls.get(1).parseCallbacks.handleClose()
		if parseGot := parseState.Get(); parseGot.ReconnectAttempts != 1 {
			parseT.Fatalf("a stable connection must restart the count, got ReconnectAttempts = %d", parseGot.ReconnectAttempts)
		}
	})
}
