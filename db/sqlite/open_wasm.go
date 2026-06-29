//go:build js && wasm

package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/ncruces/go-sqlite3/driver" // registers database/sql driver "sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"  // embeds the SQLite wasm binary
)

// openDriver (wasm) backs the database with ncruces/go-sqlite3. Memory uses an
// in-wasm database; durable backends use the snapshot-able gwcmem VFS plus a
// persistence backend (see persist_wasm.go).
func openDriver(parseCtx context.Context, parseOptions Options) (*DB, error) {
	if parseOptions.Persistence == Memory {
		parseSDB, parseErr := openConn(parseCtx, ":memory:", parseOptions)
		if parseErr != nil {
			return nil, parseErr
		}
		return &DB{sdb: parseSDB}, nil
	}
	return openPersistent(parseCtx, parseOptions)
}

// openConn opens a single-connection *sql.DB and applies pragmas.
func openConn(parseCtx context.Context, parseDSN string, parseOptions Options) (*sql.DB, error) {
	parseSDB, parseErr := sql.Open("sqlite3", parseDSN)
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
	return parseSDB, nil
}
