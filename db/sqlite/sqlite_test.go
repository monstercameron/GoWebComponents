package sqlite

import (
	"context"
	"database/sql"
	"testing"
)

func TestMemoryCRUD(t *testing.T) {
	parseCtx := context.Background()
	parseDB, parseErr := Open(parseCtx, Options{Persistence: Memory})
	if parseErr != nil {
		t.Fatalf("open: %v", parseErr)
	}
	defer parseDB.Close()

	if _, parseErr := parseDB.Exec(parseCtx, `CREATE TABLE t(id INTEGER PRIMARY KEY, v TEXT)`); parseErr != nil {
		t.Fatalf("create: %v", parseErr)
	}
	if _, parseErr := parseDB.Exec(parseCtx, `INSERT INTO t(v) VALUES(?)`, "hello"); parseErr != nil {
		t.Fatalf("insert: %v", parseErr)
	}

	var parseV string
	if parseErr := parseDB.QueryRow(parseCtx, `SELECT v FROM t WHERE id=1`).Scan(&parseV); parseErr != nil {
		t.Fatalf("select: %v", parseErr)
	}
	if parseV != "hello" {
		t.Fatalf("got %q want %q", parseV, "hello")
	}
}

func TestTxRollback(t *testing.T) {
	parseCtx := context.Background()
	parseDB, _ := Open(parseCtx, Options{Persistence: Memory})
	defer parseDB.Close()
	parseDB.Exec(parseCtx, `CREATE TABLE t(id INTEGER PRIMARY KEY)`)

	parseWant := errSentinel
	parseErr := parseDB.Tx(parseCtx, func(parseTx *sql.Tx) error {
		parseTx.ExecContext(parseCtx, `INSERT INTO t(id) VALUES(1)`)
		return parseWant
	})
	if parseErr != parseWant {
		t.Fatalf("Tx err = %v want %v", parseErr, parseWant)
	}

	var parseN int
	parseDB.QueryRow(parseCtx, `SELECT COUNT(*) FROM t`).Scan(&parseN)
	if parseN != 0 {
		t.Fatalf("rollback failed: %d rows", parseN)
	}
}

type sentinelError struct{}

func (sentinelError) Error() string { return "sentinel" }

var errSentinel = sentinelError{}
