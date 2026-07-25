package kvstate

import (
	"context"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/db/sqlite"
)

// TestSqliteBackendSaveIsVersionConditional pins that the sqlite backend's Save
// only overwrites when the incoming version is >= the stored version. Per-key
// writes are versioned monotonically but persist from goroutines that can land out
// of order; an unconditional upsert would let a stale (lower-version) write clobber
// a newer value on disk. This completes the kvstate version-race fix at the
// durability layer.
func TestSqliteBackendSaveIsVersionConditional(parseT *testing.T) {
	parseCtx := context.Background()
	parseDB, parseErr := sqlite.Open(parseCtx, sqlite.Options{Name: "kvtest-verscond", Persistence: sqlite.Memory})
	if parseErr != nil {
		parseT.Fatalf("sqlite.Open: %v", parseErr)
	}
	defer parseDB.Close()

	parseTable := "kv_verscond"
	if _, parseErr := parseDB.Exec(parseCtx,
		"CREATE TABLE IF NOT EXISTS "+parseTable+" (k TEXT PRIMARY KEY, v BLOB NOT NULL, version INTEGER NOT NULL, updated_at INTEGER NOT NULL)",
	); parseErr != nil {
		parseT.Fatalf("create table: %v", parseErr)
	}
	parseBackend := &sqliteBackend{db: parseDB, table: parseTable}

	parseSave := func(parseValue string, parseVersion int64) {
		if parseErr := parseBackend.Save(parseCtx, Record{Key: "k1", Value: []byte(parseValue), Version: parseVersion, UpdatedAt: parseVersion * 10}); parseErr != nil {
			parseT.Fatalf("save v%d: %v", parseVersion, parseErr)
		}
	}
	parseLoad := func() Record {
		parseRec, parseOk, parseErr := parseBackend.Load(parseCtx, "k1")
		if parseErr != nil || !parseOk {
			parseT.Fatalf("load: ok=%t err=%v", parseOk, parseErr)
		}
		return parseRec
	}

	parseSave("A", 5)
	// A stale, lower-version write must be dropped, not applied.
	parseSave("B", 3)
	if parseRec := parseLoad(); parseRec.Version != 5 || string(parseRec.Value) != "A" {
		parseT.Fatalf("stale v3 write clobbered v5: version=%d value=%q", parseRec.Version, parseRec.Value)
	}
	// A newer version applies.
	parseSave("C", 7)
	if parseRec := parseLoad(); parseRec.Version != 7 || string(parseRec.Value) != "C" {
		parseT.Fatalf("newer v7 write not applied: version=%d value=%q", parseRec.Version, parseRec.Value)
	}
	// An equal-version write applies (>= permits idempotent / last-write-wins at the same version).
	parseSave("D", 7)
	if parseRec := parseLoad(); string(parseRec.Value) != "D" {
		parseT.Fatalf("equal-version write should apply under >=: value=%q", parseRec.Value)
	}
}
