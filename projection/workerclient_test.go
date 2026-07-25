package projection_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/projection"
)

// v5 — the worker request/reply plumbing.
//
// postMessage is fire-and-forget in both directions, so replies arrive as events
// with no inherent connection to the request that caused them. Correlating them
// is where every bug in this shape lives, and all of them are about a slot in
// the table outliving the request that owned it:
//
//   - a reply for a request that already timed out, delivered to whatever now
//     holds that id
//   - a worker death leaving in-flight requests waiting forever
//   - cancelled requests never releasing their slots, so the table grows for the
//     life of the session
//
// These tests are about the table, not about postMessage.

// recordingPoster captures posts and lets a test reply on its own schedule.
type recordingPoster struct {
	mutex    sync.Mutex
	posts    []postedRequest
	failWith error
	// onPost, when set, runs after each post — a place for a test to reply
	// immediately or after a delay.
	onPost func(parseRequestID uint64, parseName string)
}

type postedRequest struct {
	id      uint64
	name    string
	payload []byte
}

func (parsePoster *recordingPoster) Post(parseRequestID uint64, parseName string, parseRequest []byte) error {
	parsePoster.mutex.Lock()
	if parsePoster.failWith != nil {
		parsePoster.mutex.Unlock()
		return parsePoster.failWith
	}
	parsePoster.posts = append(parsePoster.posts, postedRequest{id: parseRequestID, name: parseName, payload: parseRequest})
	parseCallback := parsePoster.onPost
	parsePoster.mutex.Unlock()

	if parseCallback != nil {
		parseCallback(parseRequestID, parseName)
	}
	return nil
}

func (parsePoster *recordingPoster) lastID() uint64 {
	parsePoster.mutex.Lock()
	defer parsePoster.mutex.Unlock()
	if len(parsePoster.posts) == 0 {
		return 0
	}
	return parsePoster.posts[len(parsePoster.posts)-1].id
}

func buildWorkerClient(parseT *testing.T, parsePoster projection.Poster) *projection.WorkerClient {
	parseT.Helper()
	parseClient, parseErr := projection.NewWorkerClient(parsePoster)
	if parseErr != nil {
		parseT.Fatalf("NewWorkerClient: %v", parseErr)
	}
	return parseClient
}

// ------------------------------------------------------------ correlation

func TestReplyReachesItsOwnRequest(parseT *testing.T) {
	var parseClient *projection.WorkerClient
	parsePoster := &recordingPoster{}
	parsePoster.onPost = func(parseRequestID uint64, parseName string) {
		// Reply with the request id encoded, so a mis-correlation is visible in
		// the payload rather than only in a count.
		parseClient.Deliver(parseRequestID, []byte(fmt.Sprintf(`{"id":%d}`, parseRequestID)), nil)
	}
	parseClient = buildWorkerClient(parseT, parsePoster)

	for parseAttempt := range 5 {
		parseResponse, parseErr := parseClient.Send(context.Background(), "cmd", []byte("{}"))
		if parseErr != nil {
			parseT.Fatalf("attempt %d: %v", parseAttempt, parseErr)
		}
		parseWant := fmt.Sprintf(`{"id":%d}`, parseAttempt+1)
		if string(parseResponse) != parseWant {
			parseT.Errorf("attempt %d got %q, want %q — a reply reached the wrong request",
				parseAttempt, parseResponse, parseWant)
		}
	}

	if parseClient.InFlight() != 0 {
		parseT.Errorf("%d requests still in flight after all replies arrived", parseClient.InFlight())
	}
}

// TestConcurrentRequestsDoNotCrossReplies is the property under contention: many
// requests outstanding, replies arriving out of order.
func TestConcurrentRequestsDoNotCrossReplies(parseT *testing.T) {
	var parseClient *projection.WorkerClient
	parsePoster := &recordingPoster{}
	parsePoster.onPost = func(parseRequestID uint64, parseName string) {
		// Reply out of order, and after a jitter that depends on the id, so
		// replies genuinely interleave.
		go func() {
			time.Sleep(time.Duration(parseRequestID%5) * time.Millisecond)
			parseClient.Deliver(parseRequestID, []byte(fmt.Sprintf("%d", parseRequestID)), nil)
		}()
	}
	parseClient = buildWorkerClient(parseT, parsePoster)

	const parseCount = 40
	var parseWait sync.WaitGroup
	parseErrors := make(chan error, parseCount)

	for range parseCount {
		parseWait.Add(1)
		go func() {
			defer parseWait.Done()
			parseResponse, parseErr := parseClient.Send(context.Background(), "cmd", nil)
			if parseErr != nil {
				parseErrors <- parseErr
				return
			}
			if len(parseResponse) == 0 {
				parseErrors <- errors.New("empty reply")
			}
		}()
	}
	parseWait.Wait()
	close(parseErrors)

	for parseErr := range parseErrors {
		parseT.Errorf("concurrent send: %v", parseErr)
	}
	if parseClient.InFlight() != 0 {
		parseT.Errorf("%d requests leaked", parseClient.InFlight())
	}
}

// ------------------------------------------------------- the slot lifecycle

// TestALateReplyIsDroppedNotMisdelivered is the bug this design is shaped to
// avoid: a reply arriving after its request gave up, delivered to whatever now
// holds that id.
func TestALateReplyIsDroppedNotMisdelivered(parseT *testing.T) {
	parsePoster := &recordingPoster{}
	parseClient := buildWorkerClient(parseT, parsePoster)

	parseCtx, parseCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer parseCancel()

	if _, parseErr := parseClient.Send(parseCtx, "abandoned", nil); parseErr == nil {
		parseT.Fatal("the request should have timed out")
	}
	parseAbandonedID := parsePoster.lastID()

	// The slot must be gone the moment the caller stopped waiting.
	if parseClient.InFlight() != 0 {
		parseT.Errorf("%d slots held after a timeout; the table grows for the session", parseClient.InFlight())
	}

	// The worker's late reply now finds nothing.
	if parseClient.Deliver(parseAbandonedID, []byte("too late"), nil) {
		parseT.Error("a late reply was delivered; it must be dropped")
	}
}

func TestCancellationReleasesTheSlot(parseT *testing.T) {
	parsePoster := &recordingPoster{}
	parseClient := buildWorkerClient(parseT, parsePoster)

	parseCtx, parseCancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		parseCancel()
	}()

	_, parseErr := parseClient.Send(parseCtx, "cmd", nil)
	if !errors.Is(parseErr, context.Canceled) {
		parseT.Errorf("err = %v, want context.Canceled", parseErr)
	}
	if parseClient.InFlight() != 0 {
		parseT.Errorf("%d slots held after cancellation", parseClient.InFlight())
	}
}

// TestDeliverNeverBlocksOnAnAbandonedRequest: an unbuffered reply channel would
// wedge the message-event callback — and with it every OTHER reply — behind one
// caller that had already given up.
func TestDeliverNeverBlocksOnAnAbandonedRequest(parseT *testing.T) {
	parsePoster := &recordingPoster{}
	parseClient := buildWorkerClient(parseT, parsePoster)

	parseCtx, parseCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer parseCancel()
	go func() { _, _ = parseClient.Send(parseCtx, "cmd", nil) }()

	// Wait until the request is actually registered.
	for range 200 {
		if parseClient.InFlight() > 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}

	parseDone := make(chan struct{})
	go func() {
		defer close(parseDone)
		// Whether this finds the slot or not, it must return promptly.
		parseClient.Deliver(parsePoster.lastID(), []byte("reply"), nil)
	}()

	select {
	case <-parseDone:
	case <-time.After(time.Second):
		parseT.Fatal("Deliver blocked; the message callback would be wedged behind one abandoned request")
	}
}

// ------------------------------------------------------------ worker death

// TestWorkerDeathFailsEveryInFlightRequest: without this each one waits for its
// own timeout, and a caller with no timeout waits forever — which presents as
// the app hanging rather than as the worker crashing.
func TestWorkerDeathFailsEveryInFlightRequest(parseT *testing.T) {
	parsePoster := &recordingPoster{}
	parseClient := buildWorkerClient(parseT, parsePoster)

	const parseCount = 6
	parseResults := make(chan error, parseCount)
	for range parseCount {
		go func() {
			_, parseErr := parseClient.Send(context.Background(), "cmd", nil)
			parseResults <- parseErr
		}()
	}

	for range 500 {
		if parseClient.InFlight() == parseCount {
			break
		}
		time.Sleep(time.Millisecond)
	}

	parseFailed := parseClient.WorkerDied("the worker crashed")
	if parseFailed != parseCount {
		parseT.Errorf("failed %d requests, want all %d", parseFailed, parseCount)
	}

	for range parseCount {
		select {
		case parseErr := <-parseResults:
			if parseErr == nil {
				parseT.Error("an in-flight request succeeded after the worker died")
				continue
			}
			// The kind matters: worker death MAY have applied, and a caller
			// needs that to decide whether retrying is safe.
			if parseKind := projection.Classify(parseErr); parseKind != projection.FailureWorkerDied {
				parseT.Errorf("kind = %s, want worker-died", parseKind)
			}
		case <-time.After(2 * time.Second):
			parseT.Fatal("a request was still waiting after the worker died")
		}
	}

	if parseClient.InFlight() != 0 {
		parseT.Errorf("%d slots survived the worker's death", parseClient.InFlight())
	}
}

// TestRequestsAfterDeathAreRefusedImmediately: a dead worker never replies, so
// posting into the void only buys a timeout that nothing can shorten.
func TestRequestsAfterDeathAreRefusedImmediately(parseT *testing.T) {
	parsePoster := &recordingPoster{}
	parseClient := buildWorkerClient(parseT, parsePoster)
	parseClient.WorkerDied("gone")

	parseStart := time.Now()
	_, parseErr := parseClient.Send(context.Background(), "cmd", nil)
	parseElapsed := time.Since(parseStart)

	if parseErr == nil {
		parseT.Fatal("a request to a dead worker must fail")
	}
	if projection.Classify(parseErr) != projection.FailureWorkerDied {
		parseT.Errorf("kind = %s, want worker-died", projection.Classify(parseErr))
	}
	if parseElapsed > 100*time.Millisecond {
		parseT.Errorf("took %v to refuse; it must not wait", parseElapsed)
	}
	if len(parsePoster.posts) != 0 {
		parseT.Error("a request was posted to a worker known to be dead")
	}
}

func TestRestartAcceptsRequestsAgain(parseT *testing.T) {
	var parseClient *projection.WorkerClient
	parsePoster := &recordingPoster{}
	parsePoster.onPost = func(parseRequestID uint64, parseName string) {
		parseClient.Deliver(parseRequestID, []byte("ok"), nil)
	}
	parseClient = buildWorkerClient(parseT, parsePoster)

	parseClient.WorkerDied("gone")
	parseClient.WorkerRestarted()

	parseResponse, parseErr := parseClient.Send(context.Background(), "cmd", nil)
	if parseErr != nil {
		parseT.Fatalf("after restart: %v", parseErr)
	}
	if string(parseResponse) != "ok" {
		parseT.Errorf("response = %q, want ok", parseResponse)
	}
}

// ----------------------------------------------------------- error kinds

// TestDomainRefusalsArriveAsRejections: the constructors exist so a transport
// cannot forget to wrap, and the classification a retry policy depends on comes
// from that wrapping.
func TestDomainRefusalsArriveAsRejections(parseT *testing.T) {
	var parseClient *projection.WorkerClient
	parsePoster := &recordingPoster{}
	parsePoster.onPost = func(parseRequestID uint64, parseName string) {
		parseClient.DeliverRejection(parseRequestID, "insufficient funds")
	}
	parseClient = buildWorkerClient(parseT, parsePoster)

	_, parseErr := parseClient.Send(context.Background(), "withdraw", nil)
	if parseErr == nil {
		parseT.Fatal("a refusal must surface")
	}
	if parseKind := projection.Classify(parseErr); parseKind != projection.FailureRejected {
		parseT.Errorf("kind = %s, want rejected", parseKind)
	}
	if parseKind := projection.Classify(parseErr); parseKind.Retryable() {
		parseT.Error("a refusal must not be retryable; the answer would be the same")
	}
}

func TestContainedPanicsArriveAsRejectionsNotDeaths(parseT *testing.T) {
	var parseClient *projection.WorkerClient
	parsePoster := &recordingPoster{}
	parsePoster.onPost = func(parseRequestID uint64, parseName string) {
		parseClient.DeliverPanic(parseRequestID, "nil map write")
	}
	parseClient = buildWorkerClient(parseT, parsePoster)

	_, parseErr := parseClient.Send(context.Background(), "cmd", nil)
	if parseKind := projection.Classify(parseErr); parseKind != projection.FailureRejected {
		parseT.Errorf("kind = %s, want rejected — the worker contained it and is still alive", parseKind)
	}
}

// TestAFailedPostIsWorkerDeathNotASilentDrop.
func TestAFailedPostIsWorkerDeathNotASilentDrop(parseT *testing.T) {
	parsePoster := &recordingPoster{failWith: errors.New("port closed")}
	parseClient := buildWorkerClient(parseT, parsePoster)

	_, parseErr := parseClient.Send(context.Background(), "cmd", nil)
	if parseErr == nil {
		parseT.Fatal("a failed post must surface")
	}
	if projection.Classify(parseErr) != projection.FailureWorkerDied {
		parseT.Errorf("kind = %s, want worker-died", projection.Classify(parseErr))
	}
	if parseClient.InFlight() != 0 {
		parseT.Errorf("%d slots held after a failed post", parseClient.InFlight())
	}
}

// ---------------------------------------------------- composes with retry

// TestWorkerClientComposesWithTheRetryPolicy is the end-to-end shape an app
// actually uses: a declared command, over a resilient client, over the worker
// correlation table.
func TestWorkerClientComposesWithTheRetryPolicy(parseT *testing.T) {
	var parseClient *projection.WorkerClient
	parseAttempts := 0
	parsePoster := &recordingPoster{}
	parsePoster.onPost = func(parseRequestID uint64, parseName string) {
		parseAttempts++
		if parseAttempts < 3 {
			parseClient.Deliver(parseRequestID, nil, errors.New("channel busy"))
			return
		}
		parseClient.Deliver(parseRequestID, []byte(`{"id":9}`), nil)
	}
	parseClient = buildWorkerClient(parseT, parsePoster)

	parseResilient, parseErr := projection.NewResilient(parseClient, projection.Policy{
		MaxAttempts: 3,
		Sleep:       func(context.Context, time.Duration) error { return nil },
	})
	if parseErr != nil {
		parseT.Fatalf("NewResilient: %v", parseErr)
	}

	parseResult, parseInvokeErr := addItem.Invoke(context.Background(), parseResilient, jsonCodec{}, addItemArgs{Name: "x"})
	if parseInvokeErr != nil {
		parseT.Fatalf("Invoke: %v", parseInvokeErr)
	}
	if parseResult.ID != 9 {
		parseT.Errorf("result = %+v, want id 9", parseResult)
	}
	if parseAttempts != 3 {
		parseT.Errorf("attempts = %d, want 3", parseAttempts)
	}
	if parseClient.InFlight() != 0 {
		parseT.Errorf("%d slots leaked across retries", parseClient.InFlight())
	}
}

// --------------------------------------------------------------- guards

func TestWorkerClientRejectsBadInput(parseT *testing.T) {
	if _, parseErr := projection.NewWorkerClient(nil); parseErr == nil {
		parseT.Error("a nil poster must be rejected")
	}

	var parseNil *projection.WorkerClient
	if _, parseErr := parseNil.Send(context.Background(), "cmd", nil); parseErr == nil {
		parseT.Error("a nil client must error rather than panic")
	}
	if parseNil.Deliver(1, nil, nil) || parseNil.InFlight() != 0 || parseNil.WorkerDied("x") != 0 {
		parseT.Error("a nil client is inert")
	}
	parseNil.WorkerRestarted()
}

func TestUnknownReplyIsReportedAsUndelivered(parseT *testing.T) {
	parseClient := buildWorkerClient(parseT, &recordingPoster{})
	if parseClient.Deliver(999, []byte("x"), nil) {
		parseT.Error("a reply for an id nobody is waiting on must report undelivered")
	}
}
