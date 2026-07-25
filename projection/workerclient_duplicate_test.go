package projection

import (
	"context"
	"sync"
	"testing"
	"time"
)

// Deliver must claim a pending slot, not merely read it.
//
// The reply channel has capacity one. If Deliver looks the channel up under the
// lock but leaves the entry in place, two callers can both find it and both
// send — and the second blocks forever on a full buffer. Deliver runs inside the
// worker's message-event callback, so that does not stall one request: it stops
// the JS event loop, and with it every later message, timer, and frame on the
// thread. A wedged page, from a duplicate reply.
//
// These tests run each suspect call under a timeout, because the failure mode is
// a hang. A plain call would take the whole package down with it rather than
// reporting which case blocked.

// silentPoster accepts every post. These tests exercise the DELIVERY side,
// so the transport only has to not fail.
type silentPoster struct{}

func (*silentPoster) Post(uint64, string, []byte) error { return nil }

// runWithin runs parseFn and reports whether it finished in time.
func runWithin(parseT *testing.T, parseLimit time.Duration, parseFn func()) bool {
	parseT.Helper()
	parseDone := make(chan struct{})
	go func() {
		defer close(parseDone)
		parseFn()
	}()
	select {
	case <-parseDone:
		return true
	case <-time.After(parseLimit):
		return false
	}
}

// newPendingClient returns a client with one in-flight request and its id.
//
// The request is registered directly rather than through Send so the test does
// not need a waiter goroutine; what is under test is the delivery side.
func newPendingClient(parseT *testing.T) (*WorkerClient, uint64) {
	parseT.Helper()
	parseClient, parseErr := NewWorkerClient(&silentPoster{})
	if parseErr != nil {
		parseT.Fatalf("NewWorkerClient: %v", parseErr)
	}
	parseClient.mutex.Lock()
	parseClient.nextID++
	parseRequestID := parseClient.nextID
	parseClient.pending[parseRequestID] = make(chan reply, 1)
	parseClient.mutex.Unlock()
	return parseClient, parseRequestID
}

// TestDeliverTwiceDoesNotBlock is the duplicate-reply case.
func TestDeliverTwiceDoesNotBlock(parseT *testing.T) {
	parseClient, parseRequestID := newPendingClient(parseT)

	if !parseClient.Deliver(parseRequestID, []byte("first"), nil) {
		parseT.Fatal("the first delivery should have found a waiter")
	}
	if !runWithin(parseT, 2*time.Second, func() {
		if parseClient.Deliver(parseRequestID, []byte("second"), nil) {
			parseT.Error("a second delivery for the same id must find nothing; the slot was already claimed")
		}
	}) {
		parseT.Fatal("a duplicate reply blocked Deliver — in a browser this wedges the JS event loop, not just this request")
	}
}

// TestDeliverRacingWorkerDeathDoesNotBlock is the case that needs no duplicate
// at all: a reply arriving as the transport reports the worker dead.
func TestDeliverRacingWorkerDeathDoesNotBlock(parseT *testing.T) {
	for parseAttempt := range 200 {
		parseClient, parseRequestID := newPendingClient(parseT)

		var parseGroup sync.WaitGroup
		parseGroup.Add(2)
		if !runWithin(parseT, 5*time.Second, func() {
			go func() {
				defer parseGroup.Done()
				parseClient.Deliver(parseRequestID, []byte("reply"), nil)
			}()
			go func() {
				defer parseGroup.Done()
				parseClient.WorkerDied("transport closed")
			}()
			parseGroup.Wait()
		}) {
			parseT.Fatalf("Deliver and WorkerDied both sent on the same one-slot channel and one blocked (attempt %d)", parseAttempt)
		}
	}
}

// TestDeliverAfterCancellationIsDropped pins the behaviour the claim must not
// change: a reply for a request whose caller already gave up finds nothing.
func TestDeliverAfterCancellationIsDropped(parseT *testing.T) {
	parseClient, parseErr := NewWorkerClient(&silentPoster{})
	if parseErr != nil {
		parseT.Fatalf("NewWorkerClient: %v", parseErr)
	}

	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	if _, parseSendErr := parseClient.Send(parseCtx, "doThing", nil); parseSendErr == nil {
		parseT.Fatal("a cancelled context must fail the send")
	}

	if !runWithin(parseT, 2*time.Second, func() {
		if parseClient.Deliver(1, []byte("late"), nil) {
			parseT.Error("a reply for an abandoned request must be dropped, not delivered")
		}
	}) {
		parseT.Fatal("delivering a late reply blocked")
	}
}
