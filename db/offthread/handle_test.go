package offthread

import (
	"context"
	"errors"
	"math"
	"testing"
)

// v5 P3.5 — the off-thread client.
//
// Tested against a scripted transport rather than a database, because what is
// under test here is the CLIENT's contract: which requests it emits, how it
// separates a rejected statement from an unreachable worker, and whether a
// transaction can be left open by a failing callback.

// scriptedTransport records requests and returns canned responses.
type scriptedTransport struct {
	requests  []Request
	responses []Response
	failWith  error
}

func (parseTransport *scriptedTransport) Call(parseCtx context.Context, parseRequest Request) (Response, error) {
	parseTransport.requests = append(parseTransport.requests, parseRequest)
	if parseTransport.failWith != nil {
		return Response{}, parseTransport.failWith
	}
	if len(parseTransport.responses) == 0 {
		return Response{}, nil
	}
	parseResponse := parseTransport.responses[0]
	parseTransport.responses = parseTransport.responses[1:]
	return parseResponse, nil
}

func (parseTransport *scriptedTransport) ops() []Op {
	parseOps := make([]Op, 0, len(parseTransport.requests))
	for _, parseRequest := range parseTransport.requests {
		parseOps = append(parseOps, parseRequest.Op)
	}
	return parseOps
}

func buildTestDB(parseT *testing.T, parseTransport Transport) *DB {
	parseT.Helper()
	parseDB, parseErr := Open(parseTransport, Options{})
	if parseErr != nil {
		parseT.Fatalf("Open: %v", parseErr)
	}
	return parseDB
}

// ------------------------------------------------------------------ basics

func TestExecSendsBoundArguments(parseT *testing.T) {
	parseTransport := &scriptedTransport{responses: []Response{{RowsAffected: 1, LastInsertID: 7}}}
	parseDB := buildTestDB(parseT, parseTransport)

	parseResult, parseErr := parseDB.Exec(context.Background(),
		"INSERT INTO t (a, b) VALUES (?, ?)", 42, "hello")
	if parseErr != nil {
		parseT.Fatalf("Exec: %v", parseErr)
	}
	if parseResult.RowsAffected != 1 || parseResult.LastInsertID != 7 {
		parseT.Errorf("result = %+v, want 1 row affected and id 7", parseResult)
	}

	parseRequest := parseTransport.requests[0]
	if parseRequest.Op != OpExec {
		parseT.Errorf("op = %q, want exec", parseRequest.Op)
	}
	// The SQL must travel unmodified with the values as bind parameters. If the
	// client ever interpolated them, the boundary would become an injection
	// surface that no amount of server-side care could close.
	if parseRequest.SQL != "INSERT INTO t (a, b) VALUES (?, ?)" {
		parseT.Errorf("sql = %q, want it unmodified", parseRequest.SQL)
	}
	if len(parseRequest.Args) != 2 ||
		parseRequest.Args[0].Kind != ValueInt || parseRequest.Args[0].Int != 42 ||
		parseRequest.Args[1].Kind != ValueText || parseRequest.Args[1].Text != "hello" {
		parseT.Errorf("args = %+v, want [int 42, text hello]", parseRequest.Args)
	}
}

func TestQueryAppliesTheRowCap(parseT *testing.T) {
	parseTransport := &scriptedTransport{responses: []Response{{Columns: []string{"a"}}}}
	parseDB, parseErr := Open(parseTransport, Options{MaxRows: 25})
	if parseErr != nil {
		parseT.Fatalf("Open: %v", parseErr)
	}

	if _, parseErr := parseDB.Query(context.Background(), "SELECT a FROM t"); parseErr != nil {
		parseT.Fatalf("Query: %v", parseErr)
	}
	if parseTransport.requests[0].MaxRows != 25 {
		parseT.Errorf("MaxRows = %d, want 25", parseTransport.requests[0].MaxRows)
	}
}

// TestQueryDefaultsToACapRatherThanUnlimited: an unbounded default would make
// the easiest possible query — SELECT * FROM a big table — marshal the whole
// table across the thread boundary, which is the stall this model exists to
// remove.
func TestQueryDefaultsToACapRatherThanUnlimited(parseT *testing.T) {
	parseTransport := &scriptedTransport{responses: []Response{{}}}
	parseDB := buildTestDB(parseT, parseTransport)

	if _, parseErr := parseDB.Query(context.Background(), "SELECT * FROM t"); parseErr != nil {
		parseT.Fatalf("Query: %v", parseErr)
	}
	if parseTransport.requests[0].MaxRows != DefaultMaxRows {
		parseT.Errorf("MaxRows = %d, want the default %d", parseTransport.requests[0].MaxRows, DefaultMaxRows)
	}
}

func TestQuerySurfacesTruncation(parseT *testing.T) {
	parseTransport := &scriptedTransport{responses: []Response{{
		Columns:   []string{"a"},
		Rows:      [][]Value{{{Kind: ValueInt, Int: 1}}},
		Truncated: true,
	}}}
	parseDB := buildTestDB(parseT, parseTransport)

	parseRows, parseErr := parseDB.Query(context.Background(), "SELECT a FROM t")
	if parseErr != nil {
		parseT.Fatalf("Query: %v", parseErr)
	}
	if !parseRows.Truncated {
		parseT.Error("truncation must reach the caller — a short set that looks complete stops pagination early")
	}
	if parseRows.Len() != 1 {
		parseT.Errorf("Len = %d, want 1", parseRows.Len())
	}
}

// --------------------------------------------------------------- error kinds

// TestRemoteErrorIsDistinctFromTransportError is the distinction a caller needs
// to react correctly: a rejected statement is a bug in the query, an unreachable
// worker is a liveness problem, and retrying is right for exactly one of them.
func TestRemoteErrorIsDistinctFromTransportError(parseT *testing.T) {
	parseRejected := buildTestDB(parseT, &scriptedTransport{responses: []Response{{Err: "no such table: t"}}})
	_, parseRejectErr := parseRejected.Exec(context.Background(), "INSERT INTO t VALUES (1)")

	var parseRemoteErr *RemoteError
	if !errors.As(parseRejectErr, &parseRemoteErr) {
		parseT.Fatalf("err = %v, want a *RemoteError", parseRejectErr)
	}
	if parseRemoteErr.Op != OpExec || parseRemoteErr.Message != "no such table: t" {
		parseT.Errorf("remote error = %+v, want the op and message preserved", parseRemoteErr)
	}

	parseUnreachable := buildTestDB(parseT, &scriptedTransport{failWith: errors.New("worker terminated")})
	_, parseTransportErr := parseUnreachable.Exec(context.Background(), "INSERT INTO t VALUES (1)")

	if errors.As(parseTransportErr, &parseRemoteErr) {
		parseT.Error("a transport failure must not be reported as a database rejection")
	}
	if parseTransportErr == nil {
		parseT.Error("a transport failure must surface")
	}
}

// -------------------------------------------------------------- transactions

func TestTxCommitsOnSuccess(parseT *testing.T) {
	parseTransport := &scriptedTransport{responses: []Response{
		{TxID: "tx-1"}, // begin
		{RowsAffected: 1},
		{}, // commit
	}}
	parseDB := buildTestDB(parseT, parseTransport)

	if parseErr := parseDB.Tx(context.Background(), func(parseTx *Tx) error {
		_, parseExecErr := parseTx.Exec(context.Background(), "INSERT INTO t VALUES (1)")
		return parseExecErr
	}); parseErr != nil {
		parseT.Fatalf("Tx: %v", parseErr)
	}

	parseWantOps := []Op{OpBegin, OpExec, OpCommit}
	parseGotOps := parseTransport.ops()
	if len(parseGotOps) != len(parseWantOps) {
		parseT.Fatalf("ops = %v, want %v", parseGotOps, parseWantOps)
	}
	for parseIndex, parseWant := range parseWantOps {
		if parseGotOps[parseIndex] != parseWant {
			parseT.Fatalf("ops = %v, want %v", parseGotOps, parseWantOps)
		}
	}
	// Statements inside the transaction must carry its id, or they run in
	// autocommit and the transaction wraps nothing.
	if parseTransport.requests[1].TxID != "tx-1" {
		parseT.Errorf("statement TxID = %q, want tx-1", parseTransport.requests[1].TxID)
	}
}

func TestTxRollsBackOnError(parseT *testing.T) {
	parseTransport := &scriptedTransport{responses: []Response{{TxID: "tx-1"}, {}}}
	parseDB := buildTestDB(parseT, parseTransport)

	parseWantErr := errors.New("business rule violated")
	if parseErr := parseDB.Tx(context.Background(), func(*Tx) error {
		return parseWantErr
	}); !errors.Is(parseErr, parseWantErr) {
		parseT.Fatalf("err = %v, want the callback's own error preserved", parseErr)
	}

	parseOps := parseTransport.ops()
	if len(parseOps) != 2 || parseOps[1] != OpRollback {
		parseT.Errorf("ops = %v, want [begin rollback]", parseOps)
	}
}

// TestTxRollsBackOnPanic matters more off-thread than in-process: an abandoned
// transaction holds the database thread's only write lock, so every later write
// from every caller blocks behind it and the database appears hung.
func TestTxRollsBackOnPanic(parseT *testing.T) {
	parseTransport := &scriptedTransport{responses: []Response{{TxID: "tx-1"}, {}}}
	parseDB := buildTestDB(parseT, parseTransport)

	func() {
		defer func() {
			if parseRecovered := recover(); parseRecovered == nil {
				parseT.Error("the panic must be re-raised after the rollback, not swallowed")
			}
		}()
		_ = parseDB.Tx(context.Background(), func(*Tx) error {
			panic("callback blew up")
		})
	}()

	parseOps := parseTransport.ops()
	if len(parseOps) != 2 || parseOps[1] != OpRollback {
		parseT.Errorf("ops = %v, want [begin rollback] — a panicking callback left the transaction open", parseOps)
	}
}

// TestTxRollbackFailureDoesNotMaskTheOriginalError: the rollback runs on a path
// that is already failing, and the caller needs to know why it failed, not that
// the cleanup also had trouble.
func TestTxRollbackFailureDoesNotMaskTheOriginalError(parseT *testing.T) {
	parseTransport := &scriptedTransport{responses: []Response{{TxID: "tx-1"}, {Err: "rollback failed too"}}}
	parseDB := buildTestDB(parseT, parseTransport)

	parseWantErr := errors.New("the real problem")
	if parseErr := parseDB.Tx(context.Background(), func(*Tx) error {
		return parseWantErr
	}); !errors.Is(parseErr, parseWantErr) {
		parseT.Errorf("err = %v, want the original cause", parseErr)
	}
}

func TestTxRejectsABeginWithoutAnID(parseT *testing.T) {
	parseTransport := &scriptedTransport{responses: []Response{{}}}
	parseDB := buildTestDB(parseT, parseTransport)

	if parseErr := parseDB.Tx(context.Background(), func(*Tx) error { return nil }); parseErr == nil {
		parseT.Error("a begin that returns no transaction id must be refused, not treated as autocommit")
	}
}

// ------------------------------------------------------------------- guards

func TestOpenRequiresATransport(parseT *testing.T) {
	if _, parseErr := Open(nil, Options{}); parseErr == nil {
		parseT.Error("a nil transport must be rejected")
	}
}

func TestExecRejectsEmptySQL(parseT *testing.T) {
	parseDB := buildTestDB(parseT, &scriptedTransport{})
	if _, parseErr := parseDB.Exec(context.Background(), ""); parseErr == nil {
		parseT.Error("empty sql must be rejected before it reaches the worker")
	}
	if _, parseErr := parseDB.Query(context.Background(), ""); parseErr == nil {
		parseT.Error("empty sql must be rejected before it reaches the worker")
	}
}

func TestExecRejectsUnsupportedArgumentTypes(parseT *testing.T) {
	parseDB := buildTestDB(parseT, &scriptedTransport{})
	if _, parseErr := parseDB.Exec(context.Background(), "INSERT INTO t VALUES (?)", struct{ A int }{1}); parseErr == nil {
		parseT.Error("a value SQLite cannot store must be rejected at the call site")
	}
	if _, parseErr := parseDB.Exec(context.Background(), "INSERT INTO t VALUES (?)", uint64(math.MaxUint64)); parseErr == nil {
		parseT.Error("an integer wider than int64 must be rejected rather than silently wrapped")
	}
}

func TestNilHandleIsSafe(parseT *testing.T) {
	var parseDB *DB
	if _, parseErr := parseDB.Exec(context.Background(), "SELECT 1"); parseErr == nil {
		parseT.Error("a nil handle must error rather than panic")
	}
	if parseErr := parseDB.Tx(context.Background(), func(*Tx) error { return nil }); parseErr == nil {
		parseT.Error("a nil handle must error rather than panic")
	}

	var parseTx *Tx
	if _, parseErr := parseTx.Exec(context.Background(), "SELECT 1"); parseErr == nil {
		parseT.Error("a nil transaction must error rather than panic")
	}
	if _, parseErr := parseTx.Query(context.Background(), "SELECT 1"); parseErr == nil {
		parseT.Error("a nil transaction must error rather than panic")
	}
}
