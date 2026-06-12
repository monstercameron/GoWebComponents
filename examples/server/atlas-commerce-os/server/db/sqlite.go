//go:build !(js && wasm)

package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open(parseCtx context.Context, parseSqlitePath string) (*sql.DB, error) {
	if parseErr := os.MkdirAll(filepath.Dir(parseSqlitePath), 0o755); parseErr != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", parseErr)
	}
	parseDsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", filepath.ToSlash(parseSqlitePath))
	parseDatabase, parseErr2 := sql.Open("sqlite", parseDsn)
	if parseErr2 != nil {
		return nil, fmt.Errorf("open sqlite: %w", parseErr2)
	}
	if parseErr3 := parseDatabase.PingContext(parseCtx); parseErr3 != nil {
		parseDatabase.Close()
		return nil, fmt.Errorf("ping sqlite: %w", parseErr3)
	}
	return parseDatabase, nil
}
