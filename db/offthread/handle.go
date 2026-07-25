package offthread

import (
	"context"
	"errors"
	"fmt"
)

// Transport carries one request to the database thread and returns its response.
//
// An interface rather than a concrete worker binding so the client is testable
// without a browser, and so the same client works over a Web Worker, a
// MessagePort, or an in-process server during development. A transport is
// responsible for delivery and for surfacing worker death as an error; it is not
// responsible for interpreting Response.Err, which is the database's own answer
// rather than a transport failure.
type Transport interface {
	Call(parseCtx context.Context, parseRequest Request) (Response, error)
}

// Options configure an off-thread database.
type Options struct {
	// MaxRows caps every query's result set unless the query overrides it.
	//
	// It defaults to DefaultMaxRows rather than to unlimited, and that default is
	// the point: a query returning every row of a large table has to marshal the
	// whole set across the thread boundary, which is the exact stall the
	// off-thread model exists to remove. A cap turns that mistake into a visible
	// Truncated flag instead of a frozen UI.
	MaxRows int
}

// DefaultMaxRows is the result-set cap applied when Options.MaxRows is unset.
const DefaultMaxRows = 10000

// DB is a handle to a database running on another thread.
//
// It deliberately does NOT expose database/sql types. *sql.Rows is a cursor over
// a live connection, and there is no live connection here — pretending otherwise
// would mean a thread round-trip per row, which is precisely the pattern this
// package exists to prevent. Results are materialized, capped, and returned
// whole.
type DB struct {
	transport Transport
	maxRows   int
}

// Open connects to a database on another thread.
//
// The transport is assumed to already address a database; opening the underlying
// file is the server's job, on its own thread. That split is what keeps the
// engine out of this binary.
func Open(parseTransport Transport, parseOptions Options) (*DB, error) {
	if parseTransport == nil {
		return nil, errors.New("offthread: a transport is required")
	}
	parseMaxRows := parseOptions.MaxRows
	if parseMaxRows <= 0 {
		parseMaxRows = DefaultMaxRows
	}
	return &DB{transport: parseTransport, maxRows: parseMaxRows}, nil
}

// Result reports what a statement changed.
type Result struct {
	RowsAffected int64
	LastInsertID int64
}

// Rows is a materialized result set.
type Rows struct {
	Columns []string
	Values  [][]Value
	// Truncated reports that the result hit the row cap and is incomplete.
	Truncated bool
}

// Len reports the row count.
func (parseRows Rows) Len() int { return len(parseRows.Values) }

// Exec runs a statement that returns no rows.
func (parseDB *DB) Exec(parseCtx context.Context, parseSQL string, parseArgs ...any) (Result, error) {
	parseResponse, parseErr := parseDB.call(parseCtx, Request{Op: OpExec, SQL: parseSQL}, parseArgs)
	if parseErr != nil {
		return Result{}, parseErr
	}
	return Result{RowsAffected: parseResponse.RowsAffected, LastInsertID: parseResponse.LastInsertID}, nil
}

// Query runs a statement and returns its rows, capped at the handle's row limit.
func (parseDB *DB) Query(parseCtx context.Context, parseSQL string, parseArgs ...any) (Rows, error) {
	parseResponse, parseErr := parseDB.call(parseCtx, Request{Op: OpQuery, SQL: parseSQL, MaxRows: parseDB.maxRows}, parseArgs)
	if parseErr != nil {
		return Rows{}, parseErr
	}
	return Rows{Columns: parseResponse.Columns, Values: parseResponse.Rows, Truncated: parseResponse.Truncated}, nil
}

// Tx runs fn inside a transaction, committing on success and rolling back on
// error or panic.
//
// The rollback on panic matters more here than in an in-process driver: an
// abandoned transaction holds a write lock on the database thread, so every
// later write from every caller blocks behind it. A panicking callback that left
// the transaction open would look like the whole database had hung.
func (parseDB *DB) Tx(parseCtx context.Context, parseFn func(*Tx) error) (parseErr error) {
	if parseDB == nil {
		return errors.New("offthread: database handle is nil")
	}
	if parseFn == nil {
		return errors.New("offthread: transaction function is required")
	}

	parseBegin, parseBeginErr := parseDB.transport.Call(parseCtx, Request{Op: OpBegin})
	if parseBeginErr != nil {
		return fmt.Errorf("offthread: begin: %w", parseBeginErr)
	}
	if parseBegin.Err != "" {
		return &RemoteError{Op: OpBegin, Message: parseBegin.Err}
	}
	if parseBegin.TxID == "" {
		return errors.New("offthread: the database thread began a transaction without returning its id")
	}

	parseTx := &Tx{db: parseDB, id: parseBegin.TxID}
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseDB.finishTx(parseCtx, OpRollback, parseTx.id)
			panic(parseRecovered)
		}
	}()

	if parseErr = parseFn(parseTx); parseErr != nil {
		parseDB.finishTx(parseCtx, OpRollback, parseTx.id)
		return parseErr
	}

	parseCommit, parseCommitErr := parseDB.transport.Call(parseCtx, Request{Op: OpCommit, TxID: parseTx.id})
	if parseCommitErr != nil {
		return fmt.Errorf("offthread: commit: %w", parseCommitErr)
	}
	if parseCommit.Err != "" {
		return &RemoteError{Op: OpCommit, Message: parseCommit.Err}
	}
	return nil
}

// finishTx ends a transaction, deliberately discarding any error.
//
// It runs on paths that are already failing — a rollback after an error or a
// panic. Replacing the original cause with a rollback error would hide why the
// transaction failed in the first place, which is the information the caller
// actually needs.
func (parseDB *DB) finishTx(parseCtx context.Context, parseOp Op, parseTxID string) {
	_, _ = parseDB.transport.Call(parseCtx, Request{Op: parseOp, TxID: parseTxID})
}

// Flush forces the database image to its durable backend.
func (parseDB *DB) Flush(parseCtx context.Context) error {
	_, parseErr := parseDB.call(parseCtx, Request{Op: OpFlush}, nil)
	return parseErr
}

// Close flushes and closes the database on its thread.
func (parseDB *DB) Close(parseCtx context.Context) error {
	_, parseErr := parseDB.call(parseCtx, Request{Op: OpClose}, nil)
	return parseErr
}

// call encodes arguments, sends a request, and separates transport failure from
// database failure.
func (parseDB *DB) call(parseCtx context.Context, parseRequest Request, parseArgs []any) (Response, error) {
	if parseDB == nil {
		return Response{}, errors.New("offthread: database handle is nil")
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	if parseRequest.Op != OpFlush && parseRequest.Op != OpClose && parseRequest.SQL == "" &&
		(parseRequest.Op == OpExec || parseRequest.Op == OpQuery) {
		return Response{}, errors.New("offthread: sql is required")
	}

	parseValues, parseValueErr := NewValues(parseArgs)
	if parseValueErr != nil {
		return Response{}, parseValueErr
	}
	parseRequest.Args = parseValues

	parseResponse, parseTransportErr := parseDB.transport.Call(parseCtx, parseRequest)
	if parseTransportErr != nil {
		// Transport failure: the database may or may not have run the statement.
		// Kept distinct from RemoteError so a caller can tell an unanswered
		// request from a rejected one.
		return Response{}, fmt.Errorf("offthread: %s could not reach the database thread: %w", parseRequest.Op, parseTransportErr)
	}
	if parseResponse.Err != "" {
		return Response{}, &RemoteError{Op: parseRequest.Op, Message: parseResponse.Err}
	}
	return parseResponse, nil
}

// Tx is an open transaction on the database thread.
type Tx struct {
	db *DB
	id string
}

// Exec runs a statement inside the transaction.
func (parseTx *Tx) Exec(parseCtx context.Context, parseSQL string, parseArgs ...any) (Result, error) {
	if parseTx == nil {
		return Result{}, errors.New("offthread: transaction is nil")
	}
	parseResponse, parseErr := parseTx.db.call(parseCtx, Request{Op: OpExec, SQL: parseSQL, TxID: parseTx.id}, parseArgs)
	if parseErr != nil {
		return Result{}, parseErr
	}
	return Result{RowsAffected: parseResponse.RowsAffected, LastInsertID: parseResponse.LastInsertID}, nil
}

// Query runs a query inside the transaction.
func (parseTx *Tx) Query(parseCtx context.Context, parseSQL string, parseArgs ...any) (Rows, error) {
	if parseTx == nil {
		return Rows{}, errors.New("offthread: transaction is nil")
	}
	parseResponse, parseErr := parseTx.db.call(parseCtx,
		Request{Op: OpQuery, SQL: parseSQL, TxID: parseTx.id, MaxRows: parseTx.db.maxRows}, parseArgs)
	if parseErr != nil {
		return Rows{}, parseErr
	}
	return Rows{Columns: parseResponse.Columns, Values: parseResponse.Rows, Truncated: parseResponse.Truncated}, nil
}
