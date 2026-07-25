package server_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/db/offthread"
	"github.com/monstercameron/GoWebComponents/v4/db/offthread/server"
	"github.com/monstercameron/GoWebComponents/v4/db/sqlite"
)

// v5 P3.5 — the engine-side half, against a real SQLite database.
//
// The client tests use a scripted transport and prove what the client SENDS.
// These prove the two halves actually compose: a statement issued through the
// off-thread API reaches SQLite, and its result comes back intact.

func buildTestServer(parseT *testing.T) (*server.Server, *sqlite.DB) {
	parseT.Helper()

	parseDB, parseErr := sqlite.Open(context.Background(), sqlite.Options{
		Name:        "offthread-test",
		Persistence: sqlite.Memory,
	})
	if parseErr != nil {
		parseT.Fatalf("open sqlite: %v", parseErr)
	}
	parseT.Cleanup(func() { _ = parseDB.Close() })

	parseServer, parseServerErr := server.New(parseDB)
	if parseServerErr != nil {
		parseT.Fatalf("new server: %v", parseServerErr)
	}
	return parseServer, parseDB
}

// buildTestHandle wires a client to a server in-process, which is the
// LocalTransport development path.
func buildTestHandle(parseT *testing.T) (*offthread.DB, *server.Server) {
	parseT.Helper()

	parseServer, _ := buildTestServer(parseT)
	parseHandle, parseErr := offthread.Open(server.NewLocalTransport(parseServer), offthread.Options{})
	if parseErr != nil {
		parseT.Fatalf("open handle: %v", parseErr)
	}

	if _, parseErr := parseHandle.Exec(context.Background(),
		"CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT, weight REAL, data BLOB)"); parseErr != nil {
		parseT.Fatalf("create table: %v", parseErr)
	}
	return parseHandle, parseServer
}

// --------------------------------------------------------------- round trip

func TestStatementsReachTheDatabaseAndResultsComeBack(parseT *testing.T) {
	parseHandle, _ := buildTestHandle(parseT)
	parseCtx := context.Background()

	parseResult, parseErr := parseHandle.Exec(parseCtx,
		"INSERT INTO items (name, weight, data) VALUES (?, ?, ?)", "widget", 1.5, []byte{7, 8})
	if parseErr != nil {
		parseT.Fatalf("insert: %v", parseErr)
	}
	if parseResult.RowsAffected != 1 {
		parseT.Errorf("rows affected = %d, want 1", parseResult.RowsAffected)
	}
	if parseResult.LastInsertID != 1 {
		parseT.Errorf("last insert id = %d, want 1", parseResult.LastInsertID)
	}

	parseRows, parseQueryErr := parseHandle.Query(parseCtx, "SELECT id, name, weight, data FROM items")
	if parseQueryErr != nil {
		parseT.Fatalf("query: %v", parseQueryErr)
	}
	if parseRows.Len() != 1 {
		parseT.Fatalf("rows = %d, want 1", parseRows.Len())
	}

	parseRow := parseRows.Values[0]
	// Each storage class must survive the boundary as itself, not as whatever a
	// generic decoder guessed.
	if parseRow[0].Kind != offthread.ValueInt || parseRow[0].Int != 1 {
		parseT.Errorf("id = %+v, want int 1", parseRow[0])
	}
	if parseRow[1].Kind != offthread.ValueText || parseRow[1].Text != "widget" {
		parseT.Errorf("name = %+v, want text widget", parseRow[1])
	}
	if parseRow[2].Kind != offthread.ValueFloat || parseRow[2].Float != 1.5 {
		parseT.Errorf("weight = %+v, want float 1.5", parseRow[2])
	}
	if parseRow[3].Kind != offthread.ValueBlob || len(parseRow[3].Blob) != 2 {
		parseT.Errorf("data = %+v, want a 2-byte blob", parseRow[3])
	}
}

func TestNullsSurviveTheBoundary(parseT *testing.T) {
	parseHandle, _ := buildTestHandle(parseT)
	parseCtx := context.Background()

	if _, parseErr := parseHandle.Exec(parseCtx, "INSERT INTO items (name) VALUES (?)", nil); parseErr != nil {
		parseT.Fatalf("insert: %v", parseErr)
	}
	parseRows, parseErr := parseHandle.Query(parseCtx, "SELECT name FROM items")
	if parseErr != nil {
		parseT.Fatalf("query: %v", parseErr)
	}
	if parseRows.Values[0][0].Kind != offthread.ValueNull {
		parseT.Errorf("name = %+v, want null", parseRows.Values[0][0])
	}
}

// TestArgumentsAreBoundNotInterpolated: the boundary must not become an
// injection surface. A value that would be catastrophic as SQL has to arrive as
// data.
func TestArgumentsAreBoundNotInterpolated(parseT *testing.T) {
	parseHandle, _ := buildTestHandle(parseT)
	parseCtx := context.Background()

	const parseHostile = "'); DROP TABLE items; --"
	if _, parseErr := parseHandle.Exec(parseCtx, "INSERT INTO items (name) VALUES (?)", parseHostile); parseErr != nil {
		parseT.Fatalf("insert: %v", parseErr)
	}

	parseRows, parseErr := parseHandle.Query(parseCtx, "SELECT name FROM items")
	if parseErr != nil {
		parseT.Fatalf("the table is gone — the argument was interpolated: %v", parseErr)
	}
	if parseRows.Len() != 1 || parseRows.Values[0][0].Text != parseHostile {
		parseT.Errorf("rows = %+v, want the hostile string stored verbatim as data", parseRows.Values)
	}
}

func TestQueryErrorsSurfaceAsRemoteErrors(parseT *testing.T) {
	parseHandle, _ := buildTestHandle(parseT)

	_, parseErr := parseHandle.Query(context.Background(), "SELECT nope FROM items")
	var parseRemoteErr *offthread.RemoteError
	if !errors.As(parseErr, &parseRemoteErr) {
		parseT.Fatalf("err = %v, want a *RemoteError from the database thread", parseErr)
	}
}

// ------------------------------------------------------------- truncation

func TestQueryTruncatesAtTheCapAndSaysSo(parseT *testing.T) {
	parseServer, _ := buildTestServer(parseT)
	parseHandle, parseErr := offthread.Open(server.NewLocalTransport(parseServer), offthread.Options{MaxRows: 10})
	if parseErr != nil {
		parseT.Fatalf("open handle: %v", parseErr)
	}
	parseCtx := context.Background()

	if _, parseErr := parseHandle.Exec(parseCtx, "CREATE TABLE n (v INTEGER)"); parseErr != nil {
		parseT.Fatalf("create: %v", parseErr)
	}
	for parseIndex := range 25 {
		if _, parseErr := parseHandle.Exec(parseCtx, "INSERT INTO n (v) VALUES (?)", parseIndex); parseErr != nil {
			parseT.Fatalf("insert %d: %v", parseIndex, parseErr)
		}
	}

	parseRows, parseQueryErr := parseHandle.Query(parseCtx, "SELECT v FROM n ORDER BY v")
	if parseQueryErr != nil {
		parseT.Fatalf("query: %v", parseQueryErr)
	}
	if parseRows.Len() != 10 {
		parseT.Errorf("rows = %d, want the cap of 10", parseRows.Len())
	}
	if !parseRows.Truncated {
		parseT.Error("a capped result must report truncation — a short set that looks complete stops pagination early")
	}
}

func TestQueryAtExactlyTheCapIsNotReportedTruncated(parseT *testing.T) {
	parseServer, _ := buildTestServer(parseT)
	parseHandle, _ := offthread.Open(server.NewLocalTransport(parseServer), offthread.Options{MaxRows: 5})
	parseCtx := context.Background()

	if _, parseErr := parseHandle.Exec(parseCtx, "CREATE TABLE n (v INTEGER)"); parseErr != nil {
		parseT.Fatalf("create: %v", parseErr)
	}
	for parseIndex := range 5 {
		if _, parseErr := parseHandle.Exec(parseCtx, "INSERT INTO n (v) VALUES (?)", parseIndex); parseErr != nil {
			parseT.Fatalf("insert: %v", parseErr)
		}
	}

	parseRows, parseErr := parseHandle.Query(parseCtx, "SELECT v FROM n")
	if parseErr != nil {
		parseT.Fatalf("query: %v", parseErr)
	}
	// Off-by-one here would cry wolf on every full page and make the flag useless.
	if parseRows.Truncated {
		parseT.Error("a result that exactly fills the cap is complete, not truncated")
	}
	if parseRows.Len() != 5 {
		parseT.Errorf("rows = %d, want 5", parseRows.Len())
	}
}

// ------------------------------------------------------------ transactions

func TestTransactionCommitsAcrossMessages(parseT *testing.T) {
	parseHandle, parseServer := buildTestHandle(parseT)
	parseCtx := context.Background()

	if parseErr := parseHandle.Tx(parseCtx, func(parseTx *offthread.Tx) error {
		for _, parseName := range []string{"a", "b", "c"} {
			if _, parseErr := parseTx.Exec(parseCtx, "INSERT INTO items (name) VALUES (?)", parseName); parseErr != nil {
				return parseErr
			}
		}
		return nil
	}); parseErr != nil {
		parseT.Fatalf("Tx: %v", parseErr)
	}

	parseRows, _ := parseHandle.Query(parseCtx, "SELECT name FROM items ORDER BY name")
	if parseRows.Len() != 3 {
		parseT.Errorf("rows = %d, want 3 committed", parseRows.Len())
	}
	if parseServer.OpenTransactionCount() != 0 {
		parseT.Errorf("%d transactions left open after a commit", parseServer.OpenTransactionCount())
	}
}

func TestTransactionRollbackDiscardsWrites(parseT *testing.T) {
	parseHandle, parseServer := buildTestHandle(parseT)
	parseCtx := context.Background()

	parseWantErr := errors.New("changed my mind")
	if parseErr := parseHandle.Tx(parseCtx, func(parseTx *offthread.Tx) error {
		if _, parseErr := parseTx.Exec(parseCtx, "INSERT INTO items (name) VALUES (?)", "ghost"); parseErr != nil {
			return parseErr
		}
		return parseWantErr
	}); !errors.Is(parseErr, parseWantErr) {
		parseT.Fatalf("err = %v, want the callback's error", parseErr)
	}

	parseRows, _ := parseHandle.Query(parseCtx, "SELECT name FROM items")
	if parseRows.Len() != 0 {
		parseT.Errorf("rows = %d, want 0 — the rollback did not discard the write", parseRows.Len())
	}
	if parseServer.OpenTransactionCount() != 0 {
		parseT.Errorf("%d transactions left open after a rollback", parseServer.OpenTransactionCount())
	}
}

// TestASecondTransactionIsRefusedRatherThanBlocking: the pool holds a single
// connection, so a concurrent transaction would wait on it forever. Refusing
// turns a permanent hang into a message.
func TestASecondTransactionIsRefusedRatherThanBlocking(parseT *testing.T) {
	parseServer, _ := buildTestServer(parseT)
	parseCtx := context.Background()

	parseFirst := parseServer.Handle(parseCtx, offthread.Request{Op: offthread.OpBegin})
	if parseFirst.Err != "" || parseFirst.TxID == "" {
		parseT.Fatalf("first begin = %+v, want a transaction id", parseFirst)
	}

	parseSecond := parseServer.Handle(parseCtx, offthread.Request{Op: offthread.OpBegin})
	if parseSecond.Err == "" {
		parseT.Error("a second concurrent transaction must be refused")
	}

	parseServer.Handle(parseCtx, offthread.Request{Op: offthread.OpRollback, TxID: parseFirst.TxID})
}

// TestCommittingAnUnknownTransactionIsAnError: reporting success would leave the
// two sides permanently disagreeing about what is durable.
func TestCommittingAnUnknownTransactionIsAnError(parseT *testing.T) {
	parseServer, _ := buildTestServer(parseT)
	parseResponse := parseServer.Handle(context.Background(),
		offthread.Request{Op: offthread.OpCommit, TxID: "tx-does-not-exist"})
	if parseResponse.Err == "" {
		parseT.Error("committing a transaction the server does not have must be an error")
	}
}

func TestStatementsInAnUnknownTransactionAreRefused(parseT *testing.T) {
	parseServer, _ := buildTestServer(parseT)
	parseResponse := parseServer.Handle(context.Background(),
		offthread.Request{Op: offthread.OpExec, SQL: "SELECT 1", TxID: "tx-nope"})
	if parseResponse.Err == "" {
		parseT.Error("a statement naming an unknown transaction must be refused, not run in autocommit")
	}
}

// TestFlushDuringATransactionIsRefusedRatherThanDeadlocking is a liveness guard.
//
// Flush acquires the single pooled connection so it can never snapshot a torn
// image; an open transaction already holds it; and this server answers one
// request at a time, so nothing would arrive to release it. Attempting it hangs
// the database thread permanently.
func TestFlushDuringATransactionIsRefusedRatherThanDeadlocking(parseT *testing.T) {
	parseServer, _ := buildTestServer(parseT)
	parseCtx := context.Background()

	parseBegin := parseServer.Handle(parseCtx, offthread.Request{Op: offthread.OpBegin})
	if parseBegin.TxID == "" {
		parseT.Fatalf("begin = %+v", parseBegin)
	}

	parseFlush := parseServer.Handle(parseCtx, offthread.Request{Op: offthread.OpFlush})
	if parseFlush.Err == "" {
		parseT.Error("flushing during an open transaction must be refused rather than attempted")
	}

	parseServer.Handle(parseCtx, offthread.Request{Op: offthread.OpRollback, TxID: parseBegin.TxID})
	if parseAfter := parseServer.Handle(parseCtx, offthread.Request{Op: offthread.OpFlush}); parseAfter.Err != "" {
		parseT.Errorf("flush after the rollback = %q, want success", parseAfter.Err)
	}
}

// ----------------------------------------------------------------- protocol

func TestUnknownOpsAreRefusedByTheServer(parseT *testing.T) {
	parseServer, _ := buildTestServer(parseT)
	parseResponse := parseServer.Handle(context.Background(), offthread.Request{Op: offthread.Op("truncate-everything")})
	if parseResponse.Err == "" {
		parseT.Error("an unknown op must be refused, not silently ignored")
	}
}

// TestHandleNeverReturnsAGoError pins the design decision that every failure
// travels in Response.Err. Two channels for one failure invites a caller to
// check only one.
func TestHandleNeverReturnsAGoError(parseT *testing.T) {
	parseServer, _ := buildTestServer(parseT)
	parseTransport := server.NewLocalTransport(parseServer)

	for _, parseRequest := range []offthread.Request{
		{Op: offthread.OpExec, SQL: "THIS IS NOT SQL"},
		{Op: offthread.Op("nonsense")},
		{Op: offthread.OpCommit, TxID: "missing"},
	} {
		parseResponse, parseErr := parseTransport.Call(context.Background(), parseRequest)
		if parseErr != nil {
			parseT.Errorf("%+v: transport returned a Go error %v; failures belong in Response.Err", parseRequest, parseErr)
		}
		if parseResponse.Err == "" {
			parseT.Errorf("%+v: expected a failure message", parseRequest)
		}
	}
}

// ------------------------------------------------- criterion (c): mixed app

// TestMixedAppUsesBothModelsSideBySide is P3.5 criterion (c).
//
// The two models must not interfere: an app can hold a conventional in-process
// sqlite.DB and an off-thread handle at once, and neither sees the other's data
// or transactions.
func TestMixedAppUsesBothModelsSideBySide(parseT *testing.T) {
	parseCtx := context.Background()

	// The conventional model, used directly.
	parseLocalDB, parseErr := sqlite.Open(parseCtx, sqlite.Options{Name: "mixed-local", Persistence: sqlite.Memory})
	if parseErr != nil {
		parseT.Fatalf("open local: %v", parseErr)
	}
	defer parseLocalDB.Close()

	if _, parseErr := parseLocalDB.Exec(parseCtx, "CREATE TABLE local_only (v TEXT)"); parseErr != nil {
		parseT.Fatalf("create local: %v", parseErr)
	}
	if _, parseErr := parseLocalDB.Exec(parseCtx, "INSERT INTO local_only VALUES (?)", "local-value"); parseErr != nil {
		parseT.Fatalf("insert local: %v", parseErr)
	}

	// The off-thread model, in the same program.
	parseOffThread, _ := buildTestHandle(parseT)
	if _, parseErr := parseOffThread.Exec(parseCtx, "INSERT INTO items (name) VALUES (?)", "remote-value"); parseErr != nil {
		parseT.Fatalf("insert off-thread: %v", parseErr)
	}

	// Each sees its own data.
	var parseLocalValue string
	if parseErr := parseLocalDB.QueryRow(parseCtx, "SELECT v FROM local_only").Scan(&parseLocalValue); parseErr != nil {
		parseT.Fatalf("read local: %v", parseErr)
	}
	if parseLocalValue != "local-value" {
		parseT.Errorf("local value = %q, want local-value", parseLocalValue)
	}

	parseRemoteRows, parseRemoteErr := parseOffThread.Query(parseCtx, "SELECT name FROM items")
	if parseRemoteErr != nil {
		parseT.Fatalf("read off-thread: %v", parseRemoteErr)
	}
	if parseRemoteRows.Len() != 1 || parseRemoteRows.Values[0][0].Text != "remote-value" {
		parseT.Errorf("off-thread rows = %+v, want remote-value", parseRemoteRows.Values)
	}

	// And neither sees the other's tables, which is what "side by side" has to
	// mean for two independent databases.
	if _, parseErr := parseOffThread.Query(parseCtx, "SELECT v FROM local_only"); parseErr == nil {
		parseT.Error("the off-thread database must not see the local database's tables")
	}
	if _, parseErr := parseLocalDB.Query(parseCtx, "SELECT name FROM items"); parseErr == nil {
		parseT.Error("the local database must not see the off-thread database's tables")
	}
}

// TestBothModelsRunTransactionsIndependently is the harder half of criterion
// (c): an open transaction on one model must not block the other.
func TestBothModelsRunTransactionsIndependently(parseT *testing.T) {
	parseCtx := context.Background()

	parseLocalDB, parseErr := sqlite.Open(parseCtx, sqlite.Options{Name: "mixed-tx-local", Persistence: sqlite.Memory})
	if parseErr != nil {
		parseT.Fatalf("open local: %v", parseErr)
	}
	defer parseLocalDB.Close()
	if _, parseErr := parseLocalDB.Exec(parseCtx, "CREATE TABLE t (v INTEGER)"); parseErr != nil {
		parseT.Fatalf("create: %v", parseErr)
	}

	parseServer, _ := buildTestServer(parseT)
	parseOffThread, _ := offthread.Open(server.NewLocalTransport(parseServer), offthread.Options{})
	if _, parseErr := parseOffThread.Exec(parseCtx, "CREATE TABLE t (v INTEGER)"); parseErr != nil {
		parseT.Fatalf("create off-thread: %v", parseErr)
	}

	// Open a transaction on the off-thread database and, from inside it, run a
	// full transaction on the local one.
	if parseErr := parseOffThread.Tx(parseCtx, func(parseTx *offthread.Tx) error {
		if _, parseErr := parseTx.Exec(parseCtx, "INSERT INTO t VALUES (?)", 1); parseErr != nil {
			return parseErr
		}
		return parseLocalDB.Tx(parseCtx, func(parseLocalTx *sql.Tx) error {
			_, parseErr := parseLocalTx.ExecContext(parseCtx, "INSERT INTO t VALUES (?)", 2)
			return parseErr
		})
	}); parseErr != nil {
		parseT.Fatalf("nested independent transactions: %v", parseErr)
	}

	parseOffThreadRows, _ := parseOffThread.Query(parseCtx, "SELECT v FROM t")
	if parseOffThreadRows.Len() != 1 || parseOffThreadRows.Values[0][0].Int != 1 {
		parseT.Errorf("off-thread rows = %+v, want [1]", parseOffThreadRows.Values)
	}

	var parseLocalValue int64
	if parseErr := parseLocalDB.QueryRow(parseCtx, "SELECT v FROM t").Scan(&parseLocalValue); parseErr != nil {
		parseT.Fatalf("read local: %v", parseErr)
	}
	if parseLocalValue != 2 {
		parseT.Errorf("local value = %d, want 2", parseLocalValue)
	}
}

// TestServerRejectsANilDatabase covers construction.
func TestServerRejectsANilDatabase(parseT *testing.T) {
	if _, parseErr := server.New(nil); parseErr == nil {
		parseT.Error("a nil database must be rejected")
	}
	var parseNilTransport *server.LocalTransport
	if _, parseErr := parseNilTransport.Call(context.Background(), offthread.Request{Op: offthread.OpExec}); parseErr == nil {
		parseT.Error("a nil transport must error rather than panic")
	}
}

// exerciseRowCount is a small helper used by the fuzz-ish breadth test below.
func exerciseRowCount(parseT *testing.T, parseHandle *offthread.DB, parseCount int) {
	parseT.Helper()
	parseCtx := context.Background()

	if _, parseErr := parseHandle.Exec(parseCtx, "DELETE FROM items"); parseErr != nil {
		parseT.Fatalf("clear: %v", parseErr)
	}
	for parseIndex := range parseCount {
		if _, parseErr := parseHandle.Exec(parseCtx,
			"INSERT INTO items (name) VALUES (?)", fmt.Sprintf("row-%d", parseIndex)); parseErr != nil {
			parseT.Fatalf("insert %d: %v", parseIndex, parseErr)
		}
	}
	parseRows, parseErr := parseHandle.Query(parseCtx, "SELECT name FROM items")
	if parseErr != nil {
		parseT.Fatalf("query: %v", parseErr)
	}
	if parseRows.Len() != parseCount {
		parseT.Errorf("rows = %d, want %d", parseRows.Len(), parseCount)
	}
}

// TestResultSetsOfVaryingSize guards the boundary conditions around an empty
// result and a single row, where an off-by-one in materialization hides.
func TestResultSetsOfVaryingSize(parseT *testing.T) {
	parseHandle, _ := buildTestHandle(parseT)
	for _, parseCount := range []int{0, 1, 2, 37} {
		exerciseRowCount(parseT, parseHandle, parseCount)
	}
}
