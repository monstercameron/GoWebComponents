package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"sort"
	"strings"
	"time"
)

// Persistence selects how a DB survives (or does not survive) a page reload.
type Persistence int

const (
	// Memory keeps the database in wasm memory only; it is lost on reload.
	Memory Persistence = iota
	// IndexedDB snapshots the database image to IndexedDB on Flush/Close and
	// rehydrates it on Open. Main-thread, no worker. The v1 durable backend.
	IndexedDB
	// OPFS is reserved for incremental, Web-Worker-backed durability. In v1 it
	// falls back to IndexedDB behavior.
	OPFS
)

func (parseP Persistence) String() string {
	switch parseP {
	case Memory:
		return "memory"
	case IndexedDB:
		return "indexeddb"
	case OPFS:
		return "opfs"
	default:
		return fmt.Sprintf("Persistence(%d)", int(parseP))
	}
}

// Options configure a DB opened with Open.
type Options struct {
	// Name is the logical database name; it keys the IndexedDB/OPFS entry.
	// Defaults to "gwc". Keep it to letters, digits, '-' and '_'.
	Name string
	// Persistence selects the durability backend. Defaults to Memory.
	Persistence Persistence
	// Pragmas are applied via "PRAGMA k=v" after the connection opens. When nil,
	// sensible defaults are used (foreign_keys=ON, busy_timeout=5000).
	Pragmas map[string]string
	// FlushDebounce is advisory metadata for consumers (e.g. the state binding)
	// that drive their own debounced Flush; the driver itself does not debounce.
	FlushDebounce time.Duration
	// Encryptor, when non-nil, seals the database image before it is written to
	// the persistent store and opens it on restore — encryption at rest. Use
	// NewPassphraseEncryptor for the passphrase-derived AES-256-GCM scheme. nil
	// stores the image unencrypted (base64 only). See encryption.go for the
	// threat model. Ignored for Persistence == Memory (nothing is persisted).
	Encryptor Encryptor
}

// DB is a single client-side SQLite database. It is safe for sequential use;
// the underlying connection pool is capped to one connection.
type DB struct {
	sdb        *sql.DB
	flush      func(context.Context) error // nil when durability is handled by the OS/file
	closeExtra func() error                // nil unless a backend resource must be closed
}

// Open opens (and, for durable backends, rehydrates) a database.
func Open(parseCtx context.Context, parseOptions Options) (*DB, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	if parseOptions.Name == "" {
		parseOptions.Name = "gwc"
	}
	return openDriver(parseCtx, parseOptions)
}

// Exec runs a statement that returns no rows.
func (parseDB *DB) Exec(parseCtx context.Context, parseSQL string, parseArgs ...any) (sql.Result, error) {
	return parseDB.sdb.ExecContext(parseCtx, parseSQL, parseArgs...)
}

// Query runs a query that returns rows. Close the returned *sql.Rows.
func (parseDB *DB) Query(parseCtx context.Context, parseSQL string, parseArgs ...any) (*sql.Rows, error) {
	return parseDB.sdb.QueryContext(parseCtx, parseSQL, parseArgs...)
}

// QueryRow runs a query expected to return at most one row.
func (parseDB *DB) QueryRow(parseCtx context.Context, parseSQL string, parseArgs ...any) *sql.Row {
	return parseDB.sdb.QueryRowContext(parseCtx, parseSQL, parseArgs...)
}

// Tx runs fn inside a transaction, committing on success and rolling back on
// error (including a panic, which is re-raised after rollback).
func (parseDB *DB) Tx(parseCtx context.Context, parseFn func(*sql.Tx) error) (parseErr error) {
	parseTx, parseBeginErr := parseDB.sdb.BeginTx(parseCtx, nil)
	if parseBeginErr != nil {
		return parseBeginErr
	}
	defer func() {
		if parseRecover := recover(); parseRecover != nil {
			_ = parseTx.Rollback()
			panic(parseRecover)
		}
	}()
	if parseErr = parseFn(parseTx); parseErr != nil {
		_ = parseTx.Rollback()
		return parseErr
	}
	return parseTx.Commit()
}

// Flush forces the current database image to its durable backend. It is a no-op
// for Memory and for native file-backed databases.
func (parseDB *DB) Flush(parseCtx context.Context) error {
	if parseDB.flush == nil {
		return nil
	}
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	return parseDB.flush(parseCtx)
}

// Close flushes (for durable backends) and closes the database.
func (parseDB *DB) Close() error {
	parseErr := parseDB.Flush(context.Background())
	if parseCloseErr := parseDB.sdb.Close(); parseErr == nil {
		parseErr = parseCloseErr
	}
	if parseDB.closeExtra != nil {
		if parseExtraErr := parseDB.closeExtra(); parseErr == nil {
			parseErr = parseExtraErr
		}
	}
	return parseErr
}

// defaultPragmas merges the caller's pragmas over the package defaults.
func defaultPragmas(parsePragmas map[string]string) map[string]string {
	parseMerged := map[string]string{
		"foreign_keys": "ON",
		"busy_timeout": "5000",
	}
	maps.Copy(parseMerged, parsePragmas)
	return parseMerged
}

// sanitizeName keeps a name safe for use as a filename / VFS key.
func sanitizeName(parseName string) string {
	parseSafe := strings.Map(func(parseR rune) rune {
		switch {
		case parseR >= 'a' && parseR <= 'z', parseR >= 'A' && parseR <= 'Z', parseR >= '0' && parseR <= '9', parseR == '-', parseR == '_':
			return parseR
		default:
			return '_'
		}
	}, parseName)
	if parseSafe == "" {
		return "gwc"
	}
	return parseSafe
}

// applyPragmas runs each pragma as "PRAGMA k=v" in deterministic order.
func applyPragmas(parseCtx context.Context, parseSDB *sql.DB, parsePragmas map[string]string) error {
	parseKeys := make([]string, 0, len(parsePragmas))
	for parseKey := range parsePragmas {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	for _, parseKey := range parseKeys {
		if _, parseErr := parseSDB.ExecContext(parseCtx, fmt.Sprintf("PRAGMA %s=%s", parseKey, parsePragmas[parseKey])); parseErr != nil {
			return fmt.Errorf("apply pragma %s: %w", parseKey, parseErr)
		}
	}
	return nil
}
