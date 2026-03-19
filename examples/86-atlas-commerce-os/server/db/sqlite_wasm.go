//go:build js && wasm
// +build js,wasm

package db

import (
	"context"
	"database/sql"
	"fmt"
)

func Open(ctx context.Context, sqlitePath string) (*sql.DB, error) {
	_ = ctx
	_ = sqlitePath
	return nil, fmt.Errorf("sqlite is unavailable for js/wasm builds")
}
