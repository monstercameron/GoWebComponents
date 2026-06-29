//go:build js && wasm

package sqlite

import (
	"context"
	"database/sql"
	"testing"
)

// TestGwcmemSnapshotRestore exercises the v1 durability primitive directly:
// write into a gwcmem-backed database, snapshot its image, then restore that
// image under a new name and read the row back. This is exactly what the
// IndexedDB backend does around a reload, minus the IndexedDB round-trip (which
// requires a real browser and is covered by the example).
func TestGwcmemSnapshotRestore(t *testing.T) {
	parseCtx := context.Background()

	parseSrc, parseErr := sql.Open("sqlite3", "file:/snaptest?vfs=gwcmem")
	if parseErr != nil {
		t.Fatalf("open src: %v", parseErr)
	}
	parseSrc.SetMaxOpenConns(1)
	if _, parseErr := parseSrc.ExecContext(parseCtx, `CREATE TABLE kv(k TEXT PRIMARY KEY, v TEXT)`); parseErr != nil {
		t.Fatalf("create: %v", parseErr)
	}
	if _, parseErr := parseSrc.ExecContext(parseCtx, `INSERT INTO kv(k,v) VALUES('greeting','hi')`); parseErr != nil {
		t.Fatalf("insert: %v", parseErr)
	}

	parseImage, parseOK := gwcmemSnapshot("snaptest")
	if !parseOK || len(parseImage) == 0 {
		t.Fatalf("snapshot failed: ok=%v len=%d", parseOK, len(parseImage))
	}
	parseSrc.Close()

	// Simulate a reload: a brand-new name seeded from the snapshot image.
	gwcmemRestore("snaptest-restored", parseImage)
	parseDst, parseErr := sql.Open("sqlite3", "file:/snaptest-restored?vfs=gwcmem")
	if parseErr != nil {
		t.Fatalf("open dst: %v", parseErr)
	}
	parseDst.SetMaxOpenConns(1)
	defer parseDst.Close()

	var parseV string
	if parseErr := parseDst.QueryRowContext(parseCtx, `SELECT v FROM kv WHERE k='greeting'`).Scan(&parseV); parseErr != nil {
		t.Fatalf("select after restore: %v", parseErr)
	}
	if parseV != "hi" {
		t.Fatalf("snapshot/restore failed: got %q want %q", parseV, "hi")
	}
}
