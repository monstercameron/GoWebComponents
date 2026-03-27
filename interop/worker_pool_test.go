package interop

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestOpenWorkerPoolValidatesOptions verifies invalid pool construction inputs
// fail before any worker startup happens.
func TestOpenWorkerPoolValidatesOptions(parseT *testing.T) {
	if _, parseErr := OpenWorkerPool(context.Background(), WorkerPoolOptions{}); !IsCode(parseErr, CodeInvalid) {
		parseT.Fatalf("expected invalid worker pool size, got %v", parseErr)
	}
	if _, parseErr := OpenWorkerPool(context.Background(), WorkerPoolOptions{
		Size:       1,
		QueueLimit: -1,
		OpenWorker: func(context.Context) (Worker, error) { return Worker{}, nil },
	}); !IsCode(parseErr, CodeInvalid) {
		parseT.Fatalf("expected invalid worker pool queue limit, got %v", parseErr)
	}
	if _, parseErr := OpenWorkerPool(context.Background(), WorkerPoolOptions{
		Size: 1,
	}); !IsCode(parseErr, CodeInvalid) {
		parseT.Fatalf("expected invalid nil worker pool factory, got %v", parseErr)
	}
}

// TestOpenWorkerPoolRoutesRequestsAcrossWorkers verifies concurrent requests are
// distributed across the fixed worker set.
func TestOpenWorkerPoolRoutesRequestsAcrossWorkers(parseT *testing.T) {
	parseStart := make(chan int, 2)
	parseRelease := make(chan struct{})
	parseNextWorkerID := 0

	parsePool, parseErr := OpenWorkerPool(context.Background(), WorkerPoolOptions{
		Size:       2,
		QueueLimit: 0,
		OpenWorker: func(context.Context) (Worker, error) {
			parseNextWorkerID++
			parseWorkerID := parseNextWorkerID
			return Worker{
				request: func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error) {
					parseStart <- parseWorkerID
					<-parseRelease
					return WorkerMessage{Phase: "result", Name: "task", Payload: map[string]any{"worker": parseWorkerID}}, nil
				},
				terminate: func() error { return nil },
			}, nil
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker pool, got %v", parseErr)
	}

	parseResultACh := make(chan int, 1)
	go func() {
		parseResult, parseErr := RequestWorkerDecoded[map[string]int, map[string]int, map[string]int](context.Background(), parsePool, "task", map[string]int{"job": 1}, nil)
		if parseErr != nil {
			parseT.Errorf("expected pooled decoded request A, got %v", parseErr)
			return
		}
		parseResultACh <- parseResult["worker"]
	}()
	parseResultBCh := make(chan int, 1)
	go func() {
		parseResult, parseErr := RequestWorkerDecoded[map[string]int, map[string]int, map[string]int](context.Background(), parsePool, "task", map[string]int{"job": 2}, nil)
		if parseErr != nil {
			parseT.Errorf("expected pooled decoded request B, got %v", parseErr)
			return
		}
		parseResultBCh <- parseResult["worker"]
	}()

	parseWorkerA := readWorkerPoolInt(parseT, parseStart, "first pooled worker start")
	parseWorkerB := readWorkerPoolInt(parseT, parseStart, "second pooled worker start")
	if parseWorkerA == parseWorkerB {
		parseT.Fatalf("expected two distinct pooled workers, got %d and %d", parseWorkerA, parseWorkerB)
	}

	close(parseRelease)

	parseResultA := readWorkerPoolInt(parseT, parseResultACh, "first pooled request result")
	parseResultB := readWorkerPoolInt(parseT, parseResultBCh, "second pooled request result")
	if parseResultA == parseResultB {
		parseT.Fatalf("expected pooled requests to finish on distinct workers, got %d and %d", parseResultA, parseResultB)
	}
	if parsePool.GetSize() != 2 || parsePool.GetQueueLimit() != 0 {
		parseT.Fatalf("expected pooled limits size=2 queue=0, got size=%d queue=%d", parsePool.GetSize(), parsePool.GetQueueLimit())
	}
	if parseErr := parsePool.Close(); parseErr != nil {
		parseT.Fatalf("expected pooled close, got %v", parseErr)
	}
}

// TestWorkerPoolHonorsQueueBackpressure verifies bounded queueing blocks later
// requests until capacity frees or the request context expires.
func TestWorkerPoolHonorsQueueBackpressure(parseT *testing.T) {
	parseRelease := make(chan struct{})
	parseStarted := make(chan struct{}, 1)

	parsePool, parseErr := OpenWorkerPool(context.Background(), WorkerPoolOptions{
		Size:       1,
		QueueLimit: 0,
		OpenWorker: func(context.Context) (Worker, error) {
			return Worker{
				request: func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error) {
					parseStarted <- struct{}{}
					<-parseRelease
					return WorkerMessage{Phase: "result", Name: "task"}, nil
				},
				terminate: func() error { return nil },
			}, nil
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker pool, got %v", parseErr)
	}

	parseFirstDone := make(chan error, 1)
	go func() {
		_, parseErr := parsePool.Request(context.Background(), "task", nil, nil)
		parseFirstDone <- parseErr
	}()
	readWorkerPoolSignal(parseT, parseStarted, "first pooled request start")

	parseCtx, parseCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer parseCancel()
	if _, parseErr := parsePool.Request(parseCtx, "task", nil, nil); !IsCode(parseErr, CodeTimeout) {
		parseT.Fatalf("expected pooled queue timeout, got %v", parseErr)
	}

	close(parseRelease)
	if parseErr := readWorkerPoolError(parseT, parseFirstDone, "first pooled request completion"); parseErr != nil {
		parseT.Fatalf("expected first pooled request to succeed, got %v", parseErr)
	}
	if parseErr := parsePool.Close(); parseErr != nil {
		parseT.Fatalf("expected pooled close, got %v", parseErr)
	}
}

// TestWorkerPoolDrainWaitsForAcceptedWork verifies drain rejects new requests
// while allowing already accepted queued work to finish.
func TestWorkerPoolDrainWaitsForAcceptedWork(parseT *testing.T) {
	parseStarted := make(chan int, 2)
	parseReleaseFirst := make(chan struct{})
	parseReleaseSecond := make(chan struct{})
	parseRequestCount := 0
	parseTerminateCount := 0
	var parseTerminateMu sync.Mutex

	parsePool, parseErr := OpenWorkerPool(context.Background(), WorkerPoolOptions{
		Size:       1,
		QueueLimit: 1,
		OpenWorker: func(context.Context) (Worker, error) {
			return Worker{
				request: func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error) {
					parseRequestCount++
					parseStarted <- parseRequestCount
					if parseRequestCount == 1 {
						<-parseReleaseFirst
					} else {
						<-parseReleaseSecond
					}
					return WorkerMessage{Phase: "result", Name: "task", Payload: map[string]any{"count": parseRequestCount}}, nil
				},
				terminate: func() error {
					parseTerminateMu.Lock()
					parseTerminateCount++
					parseTerminateMu.Unlock()
					return nil
				},
			}, nil
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker pool, got %v", parseErr)
	}

	parseFirstDone := make(chan error, 1)
	go func() {
		_, parseErr := parsePool.Request(context.Background(), "task", nil, nil)
		parseFirstDone <- parseErr
	}()
	if parseStartedCount := readWorkerPoolInt(parseT, parseStarted, "first pooled request start"); parseStartedCount != 1 {
		parseT.Fatalf("expected first pooled request start marker 1, got %d", parseStartedCount)
	}

	parseSecondDone := make(chan error, 1)
	go func() {
		_, parseErr := parsePool.Request(context.Background(), "task", nil, nil)
		parseSecondDone <- parseErr
	}()
	time.Sleep(10 * time.Millisecond)

	parseDrainDone := make(chan error, 1)
	go func() {
		parseDrainDone <- parsePool.Drain(context.Background())
	}()

	time.Sleep(10 * time.Millisecond)
	if _, parseErr := parsePool.Request(context.Background(), "task", nil, nil); !IsCode(parseErr, CodeDisposed) {
		parseT.Fatalf("expected pooled request rejection after drain, got %v", parseErr)
	}

	close(parseReleaseFirst)
	if parseErr := readWorkerPoolError(parseT, parseFirstDone, "first pooled request completion"); parseErr != nil {
		parseT.Fatalf("expected first pooled request to succeed, got %v", parseErr)
	}
	if parseStartedCount := readWorkerPoolInt(parseT, parseStarted, "second pooled request start"); parseStartedCount != 2 {
		parseT.Fatalf("expected second pooled request start marker 2, got %d", parseStartedCount)
	}
	close(parseReleaseSecond)
	if parseErr := readWorkerPoolError(parseT, parseSecondDone, "second pooled request completion"); parseErr != nil {
		parseT.Fatalf("expected second pooled request to succeed, got %v", parseErr)
	}
	if parseErr := readWorkerPoolError(parseT, parseDrainDone, "pooled drain completion"); parseErr != nil {
		parseT.Fatalf("expected pooled drain to succeed, got %v", parseErr)
	}

	parseTerminateMu.Lock()
	parseClosed := parseTerminateCount
	parseTerminateMu.Unlock()
	if parseClosed != 1 {
		parseT.Fatalf("expected pooled drain to terminate one worker, got %d", parseClosed)
	}
}

// TestWorkerPoolCloseCancelsQueuedAndRunningRequests verifies immediate close
// interrupts active work and rejects queued or future requests.
func TestWorkerPoolCloseCancelsQueuedAndRunningRequests(parseT *testing.T) {
	parseStarted := make(chan struct{}, 1)
	parseTerminated := make(chan struct{})
	parseTerminateCount := 0
	var parseTerminateMu sync.Mutex

	parsePool, parseErr := OpenWorkerPool(context.Background(), WorkerPoolOptions{
		Size:       1,
		QueueLimit: 1,
		OpenWorker: func(context.Context) (Worker, error) {
			return Worker{
				request: func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error) {
					parseStarted <- struct{}{}
					select {
					case <-parseTerminated:
						return WorkerMessage{}, wrapError("Worker.Request", "task", CodeDisposed, context.Canceled)
					case <-time.After(250 * time.Millisecond):
						return WorkerMessage{Phase: "result", Name: "task"}, nil
					}
				},
				terminate: func() error {
					parseTerminateMu.Lock()
					parseTerminateCount++
					parseTerminateMu.Unlock()
					select {
					case <-parseTerminated:
					default:
						close(parseTerminated)
					}
					return nil
				},
			}, nil
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker pool, got %v", parseErr)
	}

	parseFirstDone := make(chan error, 1)
	go func() {
		_, parseErr := parsePool.Request(context.Background(), "task", nil, nil)
		parseFirstDone <- parseErr
	}()
	readWorkerPoolSignal(parseT, parseStarted, "running pooled request start")

	parseSecondDone := make(chan error, 1)
	go func() {
		_, parseErr := parsePool.Request(context.Background(), "task", nil, nil)
		parseSecondDone <- parseErr
	}()

	if parseErr := parsePool.Close(); parseErr != nil {
		parseT.Fatalf("expected pooled close, got %v", parseErr)
	}
	if parseErr := readWorkerPoolError(parseT, parseFirstDone, "running pooled request close result"); !IsCode(parseErr, CodeDisposed) {
		parseT.Fatalf("expected running pooled request disposed on close, got %v", parseErr)
	}
	if parseErr := readWorkerPoolError(parseT, parseSecondDone, "queued pooled request close result"); !IsCode(parseErr, CodeDisposed) {
		parseT.Fatalf("expected queued pooled request disposed on close, got %v", parseErr)
	}
	if _, parseErr := parsePool.Request(context.Background(), "task", nil, nil); !IsCode(parseErr, CodeDisposed) {
		parseT.Fatalf("expected future pooled request disposed after close, got %v", parseErr)
	}

	parseTerminateMu.Lock()
	parseClosed := parseTerminateCount
	parseTerminateMu.Unlock()
	if parseClosed != 1 {
		parseT.Fatalf("expected pooled close to terminate one worker, got %d", parseClosed)
	}
}

// TestWorkerPoolReplacesDisposedWorkers verifies unexpected worker disposal
// triggers automatic capacity repair for later requests.
func TestWorkerPoolReplacesDisposedWorkers(parseT *testing.T) {
	parseOpenCount := 0
	parseTerminateCount := 0
	var parseTerminateMu sync.Mutex

	parsePool, parseErr := OpenWorkerPool(context.Background(), WorkerPoolOptions{
		Size:       1,
		QueueLimit: 0,
		OpenWorker: func(context.Context) (Worker, error) {
			parseOpenCount++
			parseWorkerID := parseOpenCount
			if parseWorkerID == 1 {
				return Worker{
					request: func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error) {
						return WorkerMessage{}, wrapError("Worker.Request", "task", CodeDisposed, errors.New("worker disposed"))
					},
					terminate: func() error {
						parseTerminateMu.Lock()
						parseTerminateCount++
						parseTerminateMu.Unlock()
						return nil
					},
				}, nil
			}
			return Worker{
				request: func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error) {
					return WorkerMessage{Phase: "result", Name: "task", Payload: map[string]any{"worker": parseWorkerID}}, nil
				},
				terminate: func() error {
					parseTerminateMu.Lock()
					parseTerminateCount++
					parseTerminateMu.Unlock()
					return nil
				},
			}, nil
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker pool, got %v", parseErr)
	}

	if _, parseErr := parsePool.Request(context.Background(), "task", nil, nil); !IsCode(parseErr, CodeDisposed) {
		parseT.Fatalf("expected first pooled request disposed, got %v", parseErr)
	}

	parseResult, parseErr := RequestWorkerDecoded[map[string]int, map[string]int, map[string]int](context.Background(), parsePool, "task", nil, nil)
	if parseErr != nil {
		parseT.Fatalf("expected repaired pooled request to succeed, got %v", parseErr)
	}
	if parseResult["worker"] != 2 {
		parseT.Fatalf("expected repaired pooled request to use replacement worker 2, got %v", parseResult)
	}
	if parseOpenCount != 2 {
		parseT.Fatalf("expected pooled replacement worker open count 2, got %d", parseOpenCount)
	}
	if parseErr := parsePool.Close(); parseErr != nil {
		parseT.Fatalf("expected pooled close after repair, got %v", parseErr)
	}

	parseTerminateMu.Lock()
	parseClosed := parseTerminateCount
	parseTerminateMu.Unlock()
	if parseClosed != 1 {
		parseT.Fatalf("expected pooled close after repair to terminate one live worker, got %d", parseClosed)
	}
}

// TestWorkerPoolClosesWhenReplacementFails verifies replacement failure closes
// the pool so later requests fail fast with a clear repair error.
func TestWorkerPoolClosesWhenReplacementFails(parseT *testing.T) {
	parseOpenCount := 0
	parseTerminateCount := 0
	parseReplacementErr := errors.New("replacement failed")
	parseReplacementStarted := make(chan struct{}, 1)
	var parseTerminateMu sync.Mutex

	parsePool, parseErr := OpenWorkerPool(context.Background(), WorkerPoolOptions{
		Size:       1,
		QueueLimit: 0,
		OpenWorker: func(context.Context) (Worker, error) {
			parseOpenCount++
			if parseOpenCount == 1 {
				return Worker{
					request: func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error) {
						return WorkerMessage{}, wrapError("Worker.Request", "task", CodeDisposed, errors.New("worker disposed"))
					},
					terminate: func() error {
						parseTerminateMu.Lock()
						parseTerminateCount++
						parseTerminateMu.Unlock()
						return nil
					},
				}, nil
			}
			select {
			case parseReplacementStarted <- struct{}{}:
			default:
			}
			return Worker{}, parseReplacementErr
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker pool, got %v", parseErr)
	}

	_, parseErr = parsePool.Request(context.Background(), "task", nil, nil)
	if !IsCode(parseErr, CodeDisposed) {
		parseT.Fatalf("expected disposed pooled request while repair is starting, got %v", parseErr)
	}
	readWorkerPoolSignal(parseT, parseReplacementStarted, "pooled replacement attempt")

	var parseRepairErr error
	for parseAttempt := 0; parseAttempt < 20; parseAttempt++ {
		_, parseRepairErr = parsePool.Request(context.Background(), "task", nil, nil)
		if IsCode(parseRepairErr, CodeRemote) {
			break
		}
		if !IsCode(parseRepairErr, CodeDisposed) {
			parseT.Fatalf("expected pooled disposed or repair failure after replacement failure, got %v", parseRepairErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !IsCode(parseRepairErr, CodeRemote) {
		parseT.Fatalf("expected pooled repair failure to surface on later requests, got %v", parseRepairErr)
	}
	if !strings.Contains(parseRepairErr.Error(), "WorkerPool.ReplaceWorker") || !strings.Contains(parseRepairErr.Error(), "replacement failed") {
		parseT.Fatalf("expected pooled replacement failure details, got %v", parseRepairErr)
	}
	parseTerminateMu.Lock()
	parseClosed := parseTerminateCount
	parseTerminateMu.Unlock()
	if parseClosed != 1 {
		parseT.Fatalf("expected pooled replacement failure to terminate tracked worker set, got %d", parseClosed)
	}
}

// TestWorkerPoolCancelsReplacementWhenClosed verifies shutdown cancels any
// in-flight worker repair so the pool does not keep booting a replacement
// after close.
func TestWorkerPoolCancelsReplacementWhenClosed(parseT *testing.T) {
	parseRepairStarted := make(chan struct{}, 1)
	parseRepairCanceled := make(chan struct{}, 1)
	parseOpenCount := 0
	parseTerminateCount := 0
	var parseTerminateMu sync.Mutex

	parsePool, parseErr := OpenWorkerPool(context.Background(), WorkerPoolOptions{
		Size:       1,
		QueueLimit: 0,
		OpenWorker: func(parseCtx context.Context) (Worker, error) {
			parseOpenCount++
			if parseOpenCount == 1 {
				return Worker{
					request: func(context.Context, string, any, func(WorkerMessage, error)) (WorkerMessage, error) {
						return WorkerMessage{}, wrapError("Worker.Request", "task", CodeDisposed, errors.New("worker disposed"))
					},
					terminate: func() error {
						parseTerminateMu.Lock()
						parseTerminateCount++
						parseTerminateMu.Unlock()
						return nil
					},
				}, nil
			}
			select {
			case parseRepairStarted <- struct{}{}:
			default:
			}
			select {
			case <-parseCtx.Done():
				select {
				case parseRepairCanceled <- struct{}{}:
				default:
				}
				return Worker{}, parseCtx.Err()
			case <-time.After(250 * time.Millisecond):
				return Worker{}, errors.New("repair worker open timed out waiting for cancellation")
			}
		},
	})
	if parseErr != nil {
		parseT.Fatalf("expected worker pool, got %v", parseErr)
	}

	if _, parseErr := parsePool.Request(context.Background(), "task", nil, nil); !IsCode(parseErr, CodeDisposed) {
		parseT.Fatalf("expected disposed pooled request before repair cancellation, got %v", parseErr)
	}
	readWorkerPoolSignal(parseT, parseRepairStarted, "pooled repair start")

	if parseErr := parsePool.Close(); parseErr != nil {
		parseT.Fatalf("expected pooled close while cancelling repair, got %v", parseErr)
	}
	readWorkerPoolSignal(parseT, parseRepairCanceled, "pooled repair cancellation")

	if _, parseErr := parsePool.Request(context.Background(), "task", nil, nil); !IsCode(parseErr, CodeDisposed) {
		parseT.Fatalf("expected disposed pooled request after close, got %v", parseErr)
	}

	parseTerminateMu.Lock()
	parseClosed := parseTerminateCount
	parseTerminateMu.Unlock()
	if parseClosed != 1 {
		parseT.Fatalf("expected pooled close to terminate the live worker set once, got %d", parseClosed)
	}
}

// readWorkerPoolInt waits for one integer result from a test channel.
func readWorkerPoolInt(parseT *testing.T, parseValues <-chan int, parseLabel string) int {
	parseT.Helper()
	select {
	case parseValue := <-parseValues:
		return parseValue
	case <-time.After(200 * time.Millisecond):
		parseT.Fatalf("timed out waiting for %s", parseLabel)
		return 0
	}
}

// readWorkerPoolSignal waits for one signal from a test channel.
func readWorkerPoolSignal(parseT *testing.T, parseValues <-chan struct{}, parseLabel string) {
	parseT.Helper()
	select {
	case <-parseValues:
	case <-time.After(200 * time.Millisecond):
		parseT.Fatalf("timed out waiting for %s", parseLabel)
	}
}

// readWorkerPoolError waits for one error result from a test channel.
func readWorkerPoolError(parseT *testing.T, parseValues <-chan error, parseLabel string) error {
	parseT.Helper()
	select {
	case parseValue := <-parseValues:
		return parseValue
	case <-time.After(200 * time.Millisecond):
		parseT.Fatalf("timed out waiting for %s", parseLabel)
		return nil
	}
}
