//go:build js && wasm

package sqlite

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/monstercameron/GoWebComponents/interop"
)

const (
	persistDatabaseName = "gwc-sqlite" // IndexedDB database
	persistStoreName    = "images"     // IndexedDB object store
)

// openPersistent opens a durable database: it rehydrates the gwcmem image from
// IndexedDB (if any), opens the connection over the gwcmem VFS, and installs a
// Flush hook that snapshots the image back to IndexedDB.
//
// Persistence == OPFS falls back to this IndexedDB path in v1 (the OPFS,
// worker-backed VFS is a later, non-breaking addition behind the same Options).
func openPersistent(parseCtx context.Context, parseOptions Options) (*DB, error) {
	parseName := sanitizeName(parseOptions.Name)

	parseStore, parseErr := interop.OpenPersistentStore(parseCtx, interop.PersistentStoreOptions{
		Name:         persistStoreName,
		DatabaseName: persistDatabaseName,
		Version:      1,
	})
	if parseErr != nil {
		return nil, fmt.Errorf("open persistent store: %w", parseErr)
	}

	// Rehydrate the image before the connection opens.
	if parseEncoded, parseOK, parseGetErr := parseStore.GetItem(parseCtx, parseName); parseGetErr == nil && parseOK && parseEncoded != "" {
		if parseImage, parseDecErr := base64.StdEncoding.DecodeString(parseEncoded); parseDecErr == nil && len(parseImage) > 0 {
			gwcmemRestore(parseName, parseImage)
		}
	}

	parseDSN := "file:/" + parseName + "?vfs=gwcmem"
	parseSDB, parseErr := openConn(parseCtx, parseDSN, parseOptions)
	if parseErr != nil {
		_ = parseStore.Close()
		return nil, parseErr
	}

	parseDB := &DB{sdb: parseSDB}
	parseDB.flush = func(parseFlushCtx context.Context) error {
		parseImage, parseHas := gwcmemSnapshot(parseName)
		if !parseHas {
			return nil
		}
		parseEncoded := base64.StdEncoding.EncodeToString(parseImage)
		if parseSetErr := parseStore.SetItem(parseFlushCtx, parseName, parseEncoded); parseSetErr != nil {
			return fmt.Errorf("persist sqlite image: %w", parseSetErr)
		}
		return nil
	}
	parseDB.closeExtra = parseStore.Close
	return parseDB, nil
}
