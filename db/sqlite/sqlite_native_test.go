//go:build !(js && wasm)

package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestDurableReopenNative verifies rows survive Close + fresh Open under the
// same Name, using the native temp-file backend.
func TestDurableReopenNative(t *testing.T) {
	parseCtx := context.Background()
	parseName := "test-durable-reopen"
	cleanupNativeFile(parseName)
	t.Cleanup(func() { cleanupNativeFile(parseName) })

	parseDB, parseErr := Open(parseCtx, Options{Name: parseName, Persistence: IndexedDB})
	if parseErr != nil {
		t.Fatalf("open: %v", parseErr)
	}
	if _, parseErr := parseDB.Exec(parseCtx, `CREATE TABLE IF NOT EXISTS kv(k TEXT PRIMARY KEY, v TEXT)`); parseErr != nil {
		t.Fatalf("create: %v", parseErr)
	}
	if _, parseErr := parseDB.Exec(parseCtx, `INSERT INTO kv(k,v) VALUES('greeting','hi')`); parseErr != nil {
		t.Fatalf("insert: %v", parseErr)
	}
	if parseErr := parseDB.Close(); parseErr != nil {
		t.Fatalf("close: %v", parseErr)
	}

	parseDB2, parseErr := Open(parseCtx, Options{Name: parseName, Persistence: IndexedDB})
	if parseErr != nil {
		t.Fatalf("reopen: %v", parseErr)
	}
	defer parseDB2.Close()

	var parseV string
	if parseErr := parseDB2.QueryRow(parseCtx, `SELECT v FROM kv WHERE k='greeting'`).Scan(&parseV); parseErr != nil {
		t.Fatalf("select after reopen: %v", parseErr)
	}
	if parseV != "hi" {
		t.Fatalf("durability failed: got %q want %q", parseV, "hi")
	}
}

func cleanupNativeFile(parseName string) {
	parsePath := filepath.Join(os.TempDir(), "gwc-sqlite", sanitizeName(parseName)+".db")
	for _, parseSuffix := range []string{"", "-journal", "-wal", "-shm"} {
		os.Remove(parsePath + parseSuffix)
	}
}
