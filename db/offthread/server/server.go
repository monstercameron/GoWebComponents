// Package server executes off-thread database requests against a real SQLite
// database (plan item P3.5).
//
// This is the half that links the engine. It imports db/sqlite, which imports
// ncruces/go-sqlite3, which embeds wazero — roughly a megabyte of wasm
// interpreter. Compiling this package into domain.wasm and NOT into app.wasm is
// the entire point of the split, and it is what criterion (b) checks.
//
// A Server is single-threaded by construction, matching both the worker it runs
// in and the one-connection pool underneath it.
package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/monstercameron/GoWebComponents/v5/db/offthread"
	"github.com/monstercameron/GoWebComponents/v5/db/sqlite"
)

// defaultMaxRows caps a result set when a request does not.
//
// A server-side cap as well as a client-side one, because the server is what
// actually allocates the rows. A client asking for everything should not be able
// to exhaust the worker's memory.
const defaultMaxRows = 10000

// Server answers offthread.Request against a SQLite database.
type Server struct {
	db *sqlite.DB
	// openTransactions holds transactions the client has begun and not finished.
	openTransactions map[string]*sql.Tx
	transactionSeq   uint64
}

// New wraps an open database in a request server.
func New(parseDB *sqlite.DB) (*Server, error) {
	if parseDB == nil {
		return nil, errors.New("offthread/server: a database is required")
	}
	return &Server{db: parseDB, openTransactions: make(map[string]*sql.Tx)}, nil
}

// Handle executes one request and returns its response.
//
// It never returns a Go error: every failure travels in Response.Err, because
// the response is what crosses the thread boundary. A transport that also
// returned an error would give the client two channels for the same failure and
// invite it to check only one.
func (parseServer *Server) Handle(parseCtx context.Context, parseRequest offthread.Request) offthread.Response {
	if parseServer == nil {
		return offthread.Response{Err: "server is nil"}
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	if !offthread.IsKnownOp(parseRequest.Op) {
		return offthread.Response{Err: fmt.Sprintf("unknown op %q", parseRequest.Op)}
	}

	switch parseRequest.Op {
	case offthread.OpBegin:
		return parseServer.handleBegin(parseCtx)
	case offthread.OpCommit, offthread.OpRollback:
		return parseServer.handleFinishTransaction(parseRequest)
	case offthread.OpExec:
		return parseServer.handleExec(parseCtx, parseRequest)
	case offthread.OpQuery:
		return parseServer.handleQuery(parseCtx, parseRequest)
	case offthread.OpFlush:
		// Refused rather than attempted while a transaction is open, and this is
		// a deadlock guard, not a policy preference. Flush acquires the single
		// pooled connection so it can never snapshot a torn mid-transaction
		// image; an open transaction already holds that connection; and this
		// server answers one request at a time, so nothing would ever arrive to
		// release it. Attempting it hangs the database thread permanently.
		if len(parseServer.openTransactions) > 0 {
			return offthread.Response{Err: "cannot flush while a transaction is open; commit or roll back first"}
		}
		if parseErr := parseServer.db.Flush(parseCtx); parseErr != nil {
			return offthread.Response{Err: parseErr.Error()}
		}
		return offthread.Response{}
	case offthread.OpClose:
		return parseServer.handleClose()
	default:
		return offthread.Response{Err: fmt.Sprintf("unhandled op %q", parseRequest.Op)}
	}
}

func (parseServer *Server) handleBegin(parseCtx context.Context) offthread.Response {
	// One writer at a time. The underlying pool is capped at a single connection,
	// so a second concurrent transaction would block on it forever rather than
	// fail — refusing here turns a hang into a message.
	if len(parseServer.openTransactions) > 0 {
		return offthread.Response{Err: "a transaction is already open on this database"}
	}

	parseTx, parseErr := parseServer.db.Begin(parseCtx)
	if parseErr != nil {
		return offthread.Response{Err: parseErr.Error()}
	}
	parseServer.transactionSeq++
	parseTxID := "tx-" + strconv.FormatUint(parseServer.transactionSeq, 10)
	parseServer.openTransactions[parseTxID] = parseTx
	return offthread.Response{TxID: parseTxID}
}

func (parseServer *Server) handleFinishTransaction(parseRequest offthread.Request) offthread.Response {
	parseTx, hasTx := parseServer.openTransactions[parseRequest.TxID]
	if !hasTx {
		// Deliberately an error rather than a silent success. A commit naming a
		// transaction the server does not have means the two sides disagree about
		// what is durable, and reporting success would make that permanent.
		return offthread.Response{Err: fmt.Sprintf("no open transaction %q", parseRequest.TxID)}
	}
	delete(parseServer.openTransactions, parseRequest.TxID)

	var parseErr error
	if parseRequest.Op == offthread.OpCommit {
		parseErr = parseTx.Commit()
	} else {
		parseErr = parseTx.Rollback()
	}
	if parseErr != nil {
		return offthread.Response{Err: parseErr.Error()}
	}
	return offthread.Response{}
}

func (parseServer *Server) handleExec(parseCtx context.Context, parseRequest offthread.Request) offthread.Response {
	parseArgs := toGoArgs(parseRequest.Args)

	var parseResult sql.Result
	var parseErr error
	if parseRequest.TxID != "" {
		parseTx, hasTx := parseServer.openTransactions[parseRequest.TxID]
		if !hasTx {
			return offthread.Response{Err: fmt.Sprintf("no open transaction %q", parseRequest.TxID)}
		}
		parseResult, parseErr = parseTx.ExecContext(parseCtx, parseRequest.SQL, parseArgs...)
	} else {
		parseResult, parseErr = parseServer.db.Exec(parseCtx, parseRequest.SQL, parseArgs...)
	}
	if parseErr != nil {
		return offthread.Response{Err: parseErr.Error()}
	}

	parseResponse := offthread.Response{}
	// Both are optional in database/sql and unsupported by some statements, so a
	// failure to read them is not a failure of the statement itself.
	if parseAffected, parseAffectedErr := parseResult.RowsAffected(); parseAffectedErr == nil {
		parseResponse.RowsAffected = parseAffected
	}
	if parseLastID, parseLastIDErr := parseResult.LastInsertId(); parseLastIDErr == nil {
		parseResponse.LastInsertID = parseLastID
	}
	return parseResponse
}

func (parseServer *Server) handleQuery(parseCtx context.Context, parseRequest offthread.Request) offthread.Response {
	parseArgs := toGoArgs(parseRequest.Args)

	var parseRows *sql.Rows
	var parseErr error
	if parseRequest.TxID != "" {
		parseTx, hasTx := parseServer.openTransactions[parseRequest.TxID]
		if !hasTx {
			return offthread.Response{Err: fmt.Sprintf("no open transaction %q", parseRequest.TxID)}
		}
		parseRows, parseErr = parseTx.QueryContext(parseCtx, parseRequest.SQL, parseArgs...)
	} else {
		parseRows, parseErr = parseServer.db.Query(parseCtx, parseRequest.SQL, parseArgs...)
	}
	if parseErr != nil {
		return offthread.Response{Err: parseErr.Error()}
	}
	defer parseRows.Close()

	return materializeRows(parseRows, parseRequest.MaxRows)
}

// materializeRows drains a cursor into a wire result set.
//
// Draining rather than streaming is the deliberate choice described on
// offthread.DB: a cursor cannot cross a thread boundary, and emulating one would
// cost a round-trip per row.
func materializeRows(parseRows *sql.Rows, parseMaxRows int) offthread.Response {
	parseColumns, parseColumnsErr := parseRows.Columns()
	if parseColumnsErr != nil {
		return offthread.Response{Err: parseColumnsErr.Error()}
	}
	if parseMaxRows <= 0 {
		parseMaxRows = defaultMaxRows
	}

	parseResponse := offthread.Response{Columns: parseColumns}
	parseScanTargets := make([]any, len(parseColumns))
	parseScanValues := make([]any, len(parseColumns))
	for parseIndex := range parseScanTargets {
		parseScanTargets[parseIndex] = &parseScanValues[parseIndex]
	}

	for parseRows.Next() {
		if len(parseResponse.Rows) >= parseMaxRows {
			// There is at least one more row than the cap allows. Say so rather
			// than returning a short set that looks complete.
			parseResponse.Truncated = true
			break
		}
		if parseScanErr := parseRows.Scan(parseScanTargets...); parseScanErr != nil {
			return offthread.Response{Err: parseScanErr.Error()}
		}

		parseRow := make([]offthread.Value, len(parseColumns))
		for parseIndex, parseScanned := range parseScanValues {
			parseValue, parseValueErr := offthread.NewValue(parseScanned)
			if parseValueErr != nil {
				return offthread.Response{Err: fmt.Sprintf("column %q: %v", parseColumns[parseIndex], parseValueErr)}
			}
			parseRow[parseIndex] = parseValue
		}
		parseResponse.Rows = append(parseResponse.Rows, parseRow)
	}
	if parseIterErr := parseRows.Err(); parseIterErr != nil {
		return offthread.Response{Err: parseIterErr.Error()}
	}
	return parseResponse
}

func (parseServer *Server) handleClose() offthread.Response {
	// Roll back anything still open first. Closing over a live transaction would
	// discard its writes anyway; doing it explicitly means the client's next
	// request gets "no open transaction" rather than a driver-level surprise.
	for parseTxID, parseTx := range parseServer.openTransactions {
		_ = parseTx.Rollback()
		delete(parseServer.openTransactions, parseTxID)
	}
	if parseErr := parseServer.db.Close(); parseErr != nil {
		return offthread.Response{Err: parseErr.Error()}
	}
	return offthread.Response{}
}

// OpenTransactionCount reports how many transactions are open, for diagnostics
// and for tests asserting that failure paths do not leak them.
func (parseServer *Server) OpenTransactionCount() int {
	if parseServer == nil {
		return 0
	}
	return len(parseServer.openTransactions)
}

func toGoArgs(parseValues []offthread.Value) []any {
	if len(parseValues) == 0 {
		return nil
	}
	parseArgs := make([]any, 0, len(parseValues))
	for _, parseValue := range parseValues {
		parseArgs = append(parseArgs, parseValue.Go())
	}
	return parseArgs
}

// LocalTransport runs a Server in-process, implementing offthread.Transport.
//
// It exists so an app can develop and test against the off-thread API without a
// worker, and so criterion (c) — a mixed app using both models — has something
// to exercise. It is NOT the production path: running the server in-process puts
// the engine back on the calling thread, which is the thing being avoided.
type LocalTransport struct {
	server *Server
}

// NewLocalTransport wraps a server as an in-process transport.
func NewLocalTransport(parseServer *Server) *LocalTransport {
	return &LocalTransport{server: parseServer}
}

// Call executes a request in-process.
func (parseTransport *LocalTransport) Call(parseCtx context.Context, parseRequest offthread.Request) (offthread.Response, error) {
	if parseTransport == nil || parseTransport.server == nil {
		return offthread.Response{}, errors.New("offthread/server: local transport has no server")
	}
	return parseTransport.server.Handle(parseCtx, parseRequest), nil
}
