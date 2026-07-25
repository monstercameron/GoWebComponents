package offthread

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// Transaction cleanup must not inherit the caller's cancellation.
//
// Rollback usually runs BECAUSE the caller's context was cancelled or timed out.
// A transport that honours context — every real one does — refuses a request on
// an already-cancelled context before it reaches the database thread, so reusing
// that context means the rollback silently does nothing. The transaction stays
// open holding the single write lock, and every later write from every caller
// blocks behind it: the "whole database has hung" symptom Tx documents itself as
// preventing.

// contextHonouringTransport refuses calls on a cancelled context, the way a real
// worker transport does, and records the ops that got through.
type contextHonouringTransport struct {
	mutex sync.Mutex
	ops   []Op
	// failFn, when set, decides the error for a given op.
	failFn func(Op) error
}

func (parseTransport *contextHonouringTransport) Call(parseCtx context.Context, parseRequest Request) (Response, error) {
	if parseErr := parseCtx.Err(); parseErr != nil {
		return Response{}, parseErr
	}

	parseTransport.mutex.Lock()
	parseTransport.ops = append(parseTransport.ops, parseRequest.Op)
	parseTransport.mutex.Unlock()

	if parseTransport.failFn != nil {
		if parseErr := parseTransport.failFn(parseRequest.Op); parseErr != nil {
			return Response{}, parseErr
		}
	}
	if parseRequest.Op == OpBegin {
		return Response{TxID: "tx-1"}, nil
	}
	return Response{}, nil
}

func (parseTransport *contextHonouringTransport) sawOp(parseOp Op) bool {
	parseTransport.mutex.Lock()
	defer parseTransport.mutex.Unlock()
	for _, parseSeen := range parseTransport.ops {
		if parseSeen == parseOp {
			return true
		}
	}
	return false
}

// TestRollbackRunsAfterTheCallerContextIsCancelled is the finding, stated
// directly: the callback fails because its context died, and the rollback must
// still reach the database thread.
func TestRollbackRunsAfterTheCallerContextIsCancelled(parseT *testing.T) {
	parseTransport := &contextHonouringTransport{}
	parseDB, parseErr := Open(parseTransport, Options{})
	if parseErr != nil {
		parseT.Fatalf("Open: %v", parseErr)
	}

	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseTxErr := parseDB.Tx(parseCtx, func(*Tx) error {
		// The shape of a real timeout: the work is abandoned mid-transaction
		// because the context expired, and that same context is what cleanup
		// would otherwise be issued on.
		parseCancel()
		return errors.New("work abandoned because the context expired")
	})
	if parseTxErr == nil {
		parseT.Fatal("the transaction should have failed")
	}

	if !parseTransport.sawOp(OpRollback) {
		parseT.Error("no rollback reached the database thread; the transaction and its write lock are still open, and every later write will block behind them")
	}
}

// TestRollbackRunsWhenTheCallbackPanicsWithACancelledContext is the same
// property on the panic path, which is the one Tx's own doc comment calls out.
func TestRollbackRunsWhenTheCallbackPanicsWithACancelledContext(parseT *testing.T) {
	parseTransport := &contextHonouringTransport{}
	parseDB, parseErr := Open(parseTransport, Options{})
	if parseErr != nil {
		parseT.Fatalf("Open: %v", parseErr)
	}

	parseCtx, parseCancel := context.WithCancel(context.Background())
	func() {
		defer func() {
			if parseRecovered := recover(); parseRecovered == nil {
				parseT.Error("the panic should have propagated to the caller")
			}
		}()
		_ = parseDB.Tx(parseCtx, func(*Tx) error {
			parseCancel()
			panic("callback exploded")
		})
	}()

	if !parseTransport.sawOp(OpRollback) {
		parseT.Error("a panic with a cancelled context left the transaction open")
	}
}

// TestFailedCommitRollsBack covers the path that previously did no cleanup at
// all: a commit whose reply never arrived leaves the transaction open on the
// database thread, holding the write lock with nothing left to close it.
func TestFailedCommitRollsBack(parseT *testing.T) {
	parseTransport := &contextHonouringTransport{
		failFn: func(parseOp Op) error {
			if parseOp == OpCommit {
				return errors.New("worker died before answering")
			}
			return nil
		},
	}
	parseDB, parseErr := Open(parseTransport, Options{})
	if parseErr != nil {
		parseT.Fatalf("Open: %v", parseErr)
	}

	if parseTxErr := parseDB.Tx(context.Background(), func(*Tx) error { return nil }); parseTxErr == nil {
		parseT.Fatal("a failed commit must surface as an error")
	}
	if !parseTransport.sawOp(OpRollback) {
		parseT.Error("a commit that never got an answer left the transaction open")
	}
}

// TestSuccessfulCommitDoesNotRollBack keeps the fix honest from the other side.
func TestSuccessfulCommitDoesNotRollBack(parseT *testing.T) {
	parseTransport := &contextHonouringTransport{}
	parseDB, parseErr := Open(parseTransport, Options{})
	if parseErr != nil {
		parseT.Fatalf("Open: %v", parseErr)
	}

	if parseTxErr := parseDB.Tx(context.Background(), func(*Tx) error { return nil }); parseTxErr != nil {
		parseT.Fatalf("Tx: %v", parseTxErr)
	}
	if parseTransport.sawOp(OpRollback) {
		parseT.Error("a committed transaction must not also roll back")
	}
}
