package projection

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Worker command client — the request/reply plumbing that makes the rest of this
// package run against a real worker.
//
// Everything else in v5 assumes a CommandClient exists: something that takes a
// command name and bytes, and eventually produces the reply. postMessage does
// not work that way. It is fire-and-forget in both directions, so replies arrive
// as unrelated events with no inherent connection to the request that caused
// them. Correlating them is this file's whole job.
//
// The transport itself is injected, because the actual postMessage call is
// wasm-only and this must be testable natively. What is NOT injected — and is
// where every bug in this shape lives — is the correlation table:
//
//   - a reply for a request that already timed out must be dropped, not
//     delivered to whatever now holds that slot
//   - a worker that dies must fail every in-flight request, or they hang forever
//   - a cancelled request must stop waiting AND release its slot, or the table
//     grows for the life of the session

// Poster sends one encoded request toward the worker.
//
// Returns as soon as the message is handed over. It does not and cannot return
// the reply; that arrives later through Deliver.
type Poster interface {
	Post(parseRequestID uint64, parseName string, parseRequest []byte) error
}

// WorkerClient correlates postMessage replies back to their requests.
//
// Safe for concurrent use: a render-thread caller issues requests while the
// message-event callback delivers replies, and in wasm those are different
// turns of the same thread rather than different threads — but the callback can
// still interleave between any two statements here.
type WorkerClient struct {
	poster Poster

	mutex     sync.Mutex
	nextID    uint64
	pending   map[uint64]chan reply
	deadCause error
}

// reply is one worker response.
type reply struct {
	payload []byte
	err     error
}

// NewWorkerClient wraps a poster.
func NewWorkerClient(parsePoster Poster) (*WorkerClient, error) {
	if parsePoster == nil {
		return nil, errors.New("projection: a poster is required")
	}
	return &WorkerClient{
		poster:  parsePoster,
		pending: make(map[uint64]chan reply),
	}, nil
}

// Send issues a command and waits for its reply.
func (parseClient *WorkerClient) Send(parseCtx context.Context, parseName string, parseRequest []byte) ([]byte, error) {
	if parseClient == nil {
		return nil, errors.New("projection: worker client is nil")
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}

	parseClient.mutex.Lock()
	if parseClient.deadCause != nil {
		parseClient.mutex.Unlock()
		// Refused immediately rather than posted into the void. A dead worker
		// will never reply, so every request after its death would otherwise
		// wait for a timeout that nothing can shorten.
		return nil, parseClient.deadCause
	}
	parseClient.nextID++
	parseRequestID := parseClient.nextID
	// Buffered so Deliver never blocks on a caller that has already given up.
	// An unbuffered channel here would wedge the message-event callback — and
	// with it every other reply — behind one abandoned request.
	parseChannel := make(chan reply, 1)
	parseClient.pending[parseRequestID] = parseChannel
	parseClient.mutex.Unlock()

	if parsePostErr := parseClient.poster.Post(parseRequestID, parseName, parseRequest); parsePostErr != nil {
		parseClient.release(parseRequestID)
		return nil, WorkerDeath(fmt.Sprintf("posting %q: %v", parseName, parsePostErr))
	}

	select {
	case parseReply := <-parseChannel:
		parseClient.release(parseRequestID)
		if parseReply.err != nil {
			return nil, parseReply.err
		}
		return parseReply.payload, nil

	case <-parseCtx.Done():
		// The slot is released here and not on reply arrival, so a late reply
		// finds nothing and is dropped. Leaving it would let the reply be
		// delivered to whichever request later reused the id.
		parseClient.release(parseRequestID)
		return nil, parseCtx.Err()
	}
}

// Deliver hands a worker reply back to whoever is waiting for it.
//
// Called from the message-event callback. A reply for an unknown id is dropped
// and reported as such rather than treated as an error: it is the normal
// outcome for a request that timed out or was cancelled while in flight, and
// making it an error would fill a console with noise from working code.
func (parseClient *WorkerClient) Deliver(parseRequestID uint64, parsePayload []byte, parseWorkerErr error) bool {
	if parseClient == nil {
		return false
	}

	parseClient.mutex.Lock()
	parseChannel, hasPending := parseClient.pending[parseRequestID]
	parseClient.mutex.Unlock()

	if !hasPending {
		return false
	}
	parseChannel <- reply{payload: parsePayload, err: parseWorkerErr}
	return true
}

// DeliverRejection hands back a domain refusal, preserving its message.
//
// Separate from Deliver so a transport does not have to remember to wrap with
// Rejection — the classification that decides whether a retry policy will spend
// attempts on it depends on that wrapping, and "remember to wrap" is the kind of
// instruction followed on three call sites out of four.
func (parseClient *WorkerClient) DeliverRejection(parseRequestID uint64, parseMessage string) bool {
	return parseClient.Deliver(parseRequestID, nil, Rejection(parseMessage))
}

// DeliverPanic hands back a contained domain panic.
func (parseClient *WorkerClient) DeliverPanic(parseRequestID uint64, parseMessage string) bool {
	return parseClient.Deliver(parseRequestID, nil, DomainPanic(parseMessage))
}

// WorkerDied fails every in-flight request and refuses new ones.
//
// Called when the worker's error or terminate event fires. Without it, every
// request outstanding at the moment of death waits for its own timeout — and a
// caller with no timeout waits forever, which presents as the app hanging rather
// than as the worker crashing.
//
// Returns how many requests were failed, which is worth surfacing: a death that
// takes twenty in-flight commands with it is a different event from one that
// takes none.
func (parseClient *WorkerClient) WorkerDied(parseReason string) int {
	if parseClient == nil {
		return 0
	}

	parseCause := WorkerDeath(parseReason)

	parseClient.mutex.Lock()
	parseClient.deadCause = parseCause
	parseChannels := make([]chan reply, 0, len(parseClient.pending))
	for parseRequestID, parseChannel := range parseClient.pending {
		parseChannels = append(parseChannels, parseChannel)
		delete(parseClient.pending, parseRequestID)
	}
	parseClient.mutex.Unlock()

	// Sent outside the lock: a caller waking on its reply may re-enter this
	// client, and holding the mutex across that would deadlock.
	for _, parseChannel := range parseChannels {
		parseChannel <- reply{err: parseCause}
	}
	return len(parseChannels)
}

// WorkerRestarted clears the dead state after a replacement worker is ready.
//
// Requests issued before the restart are NOT retried here. Whether a command
// that may have applied can safely be reissued is the caller's decision and
// depends on whether it is idempotent — exactly the judgement FailureKind
// exists to inform, and not one a transport should make on its own.
func (parseClient *WorkerClient) WorkerRestarted() {
	if parseClient == nil {
		return
	}
	parseClient.mutex.Lock()
	defer parseClient.mutex.Unlock()
	parseClient.deadCause = nil
}

// InFlight reports how many requests are awaiting a reply.
//
// Exists so a leak is observable. A correlation table that grows for the life of
// a session is the classic failure of this shape, and it is invisible without a
// count.
func (parseClient *WorkerClient) InFlight() int {
	if parseClient == nil {
		return 0
	}
	parseClient.mutex.Lock()
	defer parseClient.mutex.Unlock()
	return len(parseClient.pending)
}

// release drops a request's slot.
func (parseClient *WorkerClient) release(parseRequestID uint64) {
	parseClient.mutex.Lock()
	defer parseClient.mutex.Unlock()
	delete(parseClient.pending, parseRequestID)
}
