//go:build !(js && wasm)

package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// openDriver (native) backs the database with modernc.org/sqlite so the same
// calling code runs in the gwc native test lane. Memory maps to an in-memory
// database; durable backends are approximated by a temp file on disk.
func openDriver(parseCtx context.Context, parseOptions Options) (*DB, error) {
	var parseDSN string
	switch parseOptions.Persistence {
	case Memory:
		parseDSN = ":memory:"
	default:
		parseDir := filepath.Join(os.TempDir(), "gwc-sqlite")
		if parseErr := os.MkdirAll(parseDir, 0o755); parseErr != nil {
			return nil, fmt.Errorf("create sqlite dir: %w", parseErr)
		}
		parsePath := filepath.Join(parseDir, sanitizeName(parseOptions.Name)+".db")
		parseDSN = "file:" + filepath.ToSlash(parsePath)
	}

	parseSDB, parseErr := sql.Open("sqlite", parseDSN)
	if parseErr != nil {
		return nil, fmt.Errorf("open sqlite: %w", parseErr)
	}
	parseSDB.SetMaxOpenConns(1)
	if parseErr := parseSDB.PingContext(parseCtx); parseErr != nil {
		parseSDB.Close()
		return nil, fmt.Errorf("ping sqlite: %w", parseErr)
	}
	if parseErr := applyPragmas(parseCtx, parseSDB, defaultPragmas(parseOptions.Pragmas)); parseErr != nil {
		parseSDB.Close()
		return nil, parseErr
	}
	// Durability on native is the OS file itself; no flush hook needed.
	return &DB{sdb: parseSDB}, nil
}
