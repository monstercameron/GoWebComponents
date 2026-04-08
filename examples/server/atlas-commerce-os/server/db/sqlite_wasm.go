//go:build js && wasm
// +build js,wasm

package db

import (
	"context"
	"database/sql"
	"fmt"
)

func Open(parseCtx context.Context, parseSqlitePath string) (*sql.DB, error) {
	_ = parseCtx
	_ = parseSqlitePath
	return nil, fmt.Errorf("sqlite is unavailable for js/wasm builds")
}
