//go:build !(js && wasm)

package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/ncruces/go-sqlite3/driver" // registers database/sql driver "sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"  // embeds the SQLite wasm binary
)

// openDriver (native) backs the database with ncruces/go-sqlite3 so native and
// js/wasm builds share the same SQLite engine and driver behavior without cgo.
// Memory maps to an in-memory database; durable backends are approximated by a
// temp file on disk.
func openDriver(ctx context.Context, options Options) (*DB, error) {
	var dsn string
	switch options.Persistence {
	case Memory:
		dsn = ":memory:"
	default:
		dir := filepath.Join(os.TempDir(), "gwc-sqlite")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create sqlite dir: %w", err)
		}
		path := filepath.Join(dir, sanitizeName(options.Name)+".db")
		dsn = "file:" + filepath.ToSlash(path)
	}

	sdb, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	sdb.SetMaxOpenConns(1)
	if err := sdb.PingContext(ctx); err != nil {
		if closeErr := sdb.Close(); closeErr != nil {
			return nil, fmt.Errorf("ping sqlite: %v (close: %w)", err, closeErr)
		}
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := applyPragmas(ctx, sdb, defaultPragmas(options.Pragmas)); err != nil {
		if closeErr := sdb.Close(); closeErr != nil {
			return nil, fmt.Errorf("apply sqlite pragmas: %v (close: %w)", err, closeErr)
		}
		return nil, err
	}
	// Durability on native is the OS file itself; no flush hook needed.
	return &DB{sdb: sdb}, nil
}
