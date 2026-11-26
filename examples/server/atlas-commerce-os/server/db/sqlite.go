//go:build !(js && wasm)

package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// Open opens or creates the SQLite database file for the Atlas Commerce OS example server.
func Open(ctx context.Context, sqlitePath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(sqlitePath), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", filepath.ToSlash(sqlitePath))
	database, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return database, nil
}
