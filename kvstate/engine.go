package kvstate

import (
	"context"
	"database/sql"
	"sync"

	"github.com/monstercameron/GoWebComponents/v4/db/sqlite"
)

// engine bundles a PersistenceBackend with the durable flush it is paced by.
type engine struct {
	backend PersistenceBackend
	flush   func(context.Context) error
	close   func() error
}

var (
	enginesMu sync.Mutex
	engines   = map[string]*engine{}
)

// acquireEngine returns the shared engine for opts.Name, creating it (and its
// SQLite database + table) on first use. When a custom Backend is supplied, it
// is used directly and no SQLite database is opened.
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
	if parseExisting, parseOK := engines[parseOptions.Name]; parseOK {
		return parseExisting, nil
	}

	parseDB, parseErr := sqlite.Open(parseCtx, sqlite.Options{
		Name:        parseOptions.Name,
		Persistence: parseOptions.Durability.toSQLite(),
	})
	if parseErr != nil {
		return nil, parseErr
	}
	if _, parseErr := parseDB.Exec(parseCtx,
		"CREATE TABLE IF NOT EXISTS "+parseOptions.Table+
			" (k TEXT PRIMARY KEY, v BLOB NOT NULL, version INTEGER NOT NULL, updated_at INTEGER NOT NULL)",
	); parseErr != nil {
		parseDB.Close()
		return nil, parseErr
	}

	parseEngine := &engine{
		backend: &sqliteBackend{db: parseDB, table: parseOptions.Table},
		flush:   parseDB.Flush,
		close:   parseDB.Close,
	}
	engines[parseOptions.Name] = parseEngine
	return parseEngine, nil
}

// resetEnginesForTest drops the shared engine cache (closing databases). Used by
// tests to start clean; not part of the public API.
func resetEnginesForTest() {
	enginesMu.Lock()
	defer enginesMu.Unlock()
	for parseName, parseEngine := range engines {
		if parseEngine.close != nil {
			parseEngine.close()
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
	_, parseErr := parseB.db.Exec(parseCtx,
		"INSERT INTO "+parseB.table+" (k, v, version, updated_at) VALUES (?, ?, ?, ?)"+
			" ON CONFLICT(k) DO UPDATE SET v = excluded.v, version = excluded.version, updated_at = excluded.updated_at",
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
