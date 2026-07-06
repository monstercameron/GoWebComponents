package kvstate

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/monstercameron/GoWebComponents/v4/db/sqlite"
)

// engine bundles a PersistenceBackend with the durable flush it is paced by.
type engine struct {
	backend PersistenceBackend
	flush   func(context.Context) error
	close   func() error
}

// namedDB is one opened SQLite database shared by every table-scoped engine
// under one Options.Name.
type namedDB struct {
	db         *sqlite.DB
	durability Durability
	engines    map[string]*engine // keyed by table
}

var (
	enginesMu sync.Mutex
	engines   = map[string]*namedDB{}
)

// acquireEngine returns the shared engine for (opts.Name, opts.Table), creating
// the SQLite database on first use of the Name and the table on first use of
// the (Name, Table) pair. The cache used to key by Name alone, which silently
// routed a second Table (and ignored a different Durability) into whatever the
// first caller opened. When a custom Backend is supplied, it is used directly
// and no SQLite database is opened.
func acquireEngine(parseCtx context.Context, parseOptions Options) (*engine, error) {
	parseOptions = parseOptions.withDefaults()

	if parseOptions.Backend != nil {
		return &engine{
			backend: parseOptions.Backend,
			flush:   func(context.Context) error { return nil }, // custom backend persists on Save
		}, nil
	}

	enginesMu.Lock()
	defer enginesMu.Unlock()
	parseEntry, parseOK := engines[parseOptions.Name]
	if parseOK {
		if parseEntry.durability != parseOptions.Durability {
			return nil, fmt.Errorf("kvstate: store %q is already open with a different durability; all Options sharing a Name must agree on Durability", parseOptions.Name)
		}
		if parseExisting, hasTable := parseEntry.engines[parseOptions.Table]; hasTable {
			return parseExisting, nil
		}
	} else {
		parseDB, parseErr := sqlite.Open(parseCtx, sqlite.Options{
			Name:        parseOptions.Name,
			Persistence: parseOptions.Durability.toSQLite(),
		})
		if parseErr != nil {
			return nil, parseErr
		}
		parseEntry = &namedDB{db: parseDB, durability: parseOptions.Durability, engines: map[string]*engine{}}
		engines[parseOptions.Name] = parseEntry
	}

	if _, parseErr := parseEntry.db.Exec(parseCtx,
		"CREATE TABLE IF NOT EXISTS "+parseOptions.Table+
			" (k TEXT PRIMARY KEY, v BLOB NOT NULL, version INTEGER NOT NULL, updated_at INTEGER NOT NULL)",
	); parseErr != nil {
		if len(parseEntry.engines) == 0 {
			parseEntry.db.Close()
			delete(engines, parseOptions.Name)
		}
		return nil, parseErr
	}

	parseEngine := &engine{
		backend: &sqliteBackend{db: parseEntry.db, table: parseOptions.Table},
		flush:   parseEntry.db.Flush,
		close:   parseEntry.db.Close,
	}
	parseEntry.engines[parseOptions.Table] = parseEngine
	return parseEngine, nil
}

// resetEnginesForTest drops the shared engine cache (closing databases). Used by
// tests to start clean; not part of the public API.
func resetEnginesForTest() {
	enginesMu.Lock()
	defer enginesMu.Unlock()
	for parseName, parseEntry := range engines {
		if parseEntry.db != nil {
			parseEntry.db.Close()
		}
		delete(engines, parseName)
	}
}

// sqliteBackend is the default PersistenceBackend, storing each value as a row
// in the KV table.
type sqliteBackend struct {
	db    *sqlite.DB
	table string
}

func (parseB *sqliteBackend) Load(parseCtx context.Context, parseKey string) (Record, bool, error) {
	parseRec := Record{Key: parseKey}
	parseErr := parseB.db.QueryRow(parseCtx,
		"SELECT v, version, updated_at FROM "+parseB.table+" WHERE k = ?", parseKey,
	).Scan(&parseRec.Value, &parseRec.Version, &parseRec.UpdatedAt)
	if parseErr == sql.ErrNoRows {
		return Record{}, false, nil
	}
	if parseErr != nil {
		return Record{}, false, parseErr
	}
	return parseRec, true, nil
}

func (parseB *sqliteBackend) Save(parseCtx context.Context, parseRecord Record) error {
	// Version-conditional upsert: only overwrite when the incoming version is at
	// least the stored one. Writes for a single key are versioned monotonically, but
	// they persist from per-write goroutines that can land out of order; an
	// unconditional DO UPDATE would let a stale (lower-version) write clobber a newer
	// value already on disk. The WHERE clause drops the stale write (no row change,
	// no error), completing the version-race fix at the durability layer.
	_, parseErr := parseB.db.Exec(parseCtx,
		"INSERT INTO "+parseB.table+" (k, v, version, updated_at) VALUES (?, ?, ?, ?)"+
			" ON CONFLICT(k) DO UPDATE SET v = excluded.v, version = excluded.version, updated_at = excluded.updated_at"+
			" WHERE excluded.version >= "+parseB.table+".version",
		parseRecord.Key, parseRecord.Value, parseRecord.Version, parseRecord.UpdatedAt,
	)
	return parseErr
}

func (parseB *sqliteBackend) Delete(parseCtx context.Context, parseKey string) error {
	_, parseErr := parseB.db.Exec(parseCtx, "DELETE FROM "+parseB.table+" WHERE k = ?", parseKey)
	return parseErr
}

func (parseB *sqliteBackend) Keys(parseCtx context.Context) ([]string, error) {
	parseRows, parseErr := parseB.db.Query(parseCtx, "SELECT k FROM "+parseB.table)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()
	var parseKeys []string
	for parseRows.Next() {
		var parseKey string
		if parseErr := parseRows.Scan(&parseKey); parseErr != nil {
			return nil, parseErr
		}
		parseKeys = append(parseKeys, parseKey)
	}
	return parseKeys, parseRows.Err()
}
