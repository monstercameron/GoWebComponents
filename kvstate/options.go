package kvstate

import (
	"time"

	"github.com/monstercameron/GoWebComponents/v6/db/sqlite"
)

// Durability selects the db/sqlite persistence backend. Its zero value
// (DurableDefault) means IndexedDB, so callers who want in-memory state must ask
// for it explicitly.
type Durability int

const (
	// DurableDefault maps to IndexedDB.
	DurableDefault Durability = iota
	// DurableMemory keeps state in wasm memory only (lost on reload).
	DurableMemory
	// DurableIndexedDB snapshots to IndexedDB (the v1 durable backend).
	DurableIndexedDB
	// DurableOPFS is reserved; falls back to IndexedDB in v1.
	DurableOPFS
)

func (parseD Durability) toSQLite() sqlite.Persistence {
	switch parseD {
	case DurableMemory:
		return sqlite.Memory
	case DurableOPFS:
		return sqlite.OPFS
	default:
		return sqlite.IndexedDB
	}
}

// HydrateMode controls when a value is loaded from the backend.
type HydrateMode int

const (
	// HydrateEager loads the stored value when the binding mounts.
	HydrateEager HydrateMode = iota
	// HydrateLazy does NOT auto-load the stored value: the binding starts at the supplied initial
	// value and the first Set persists forward over whatever was stored. Use it for write-through
	// state that should not read its prior value back on mount.
	//
	// NOTE: this is "skip the eager load", not "load on first read" — true load-on-first-read is not
	// implemented in the asynchronous-engine hook model (there is no synchronous read path to defer
	// into). Use HydrateEager when you need the stored value available after mount.
	HydrateLazy
)

// Options configure a persisted binding. Any zero field takes its default, so
// the common case needs no Options at all.
type Options struct {
	// Name is the logical database name (db/sqlite Options.Name). Default "gwc".
	Name string
	// Table is the KV table name. Default "gwc_state".
	Table string
	// Codec encodes/decodes values. Default JSONCodec.
	Codec Codec
	// Strategy decides when writes become durable. Default Debounced(150ms).
	Strategy WriteStrategy
	// Conflict reconciles concurrent/cross-tab writes. Default LastWriteWins.
	Conflict ConflictResolver
	// Hydrate controls load timing. Default HydrateEager.
	Hydrate HydrateMode
	// Durability selects the SQLite backend. Default DurableDefault (IndexedDB).
	Durability Durability
	// Backend overrides the persistence layer entirely. When nil, the shared
	// SQLite engine is used.
	Backend PersistenceBackend
	// ExternalInvalidation disables BroadcastChannel for this binding. A host
	// adapter must call Invalidate(Name) after durable commits instead.
	ExternalInvalidation bool
}

func (parseO Options) withDefaults() Options {
	if parseO.Name == "" {
		parseO.Name = "gwc"
	}
	if parseO.Table == "" {
		parseO.Table = "gwc_state"
	}
	parseO.Table = sanitizeIdent(parseO.Table)
	if parseO.Codec == nil {
		parseO.Codec = JSONCodec{}
	}
	if parseO.Strategy == nil {
		parseO.Strategy = Debounced(150 * time.Millisecond)
	}
	if parseO.Conflict == nil {
		parseO.Conflict = LastWriteWins{}
	}
	return parseO
}

// sanitizeIdent keeps a SQL identifier (table name) safe for interpolation.
func sanitizeIdent(parseName string) string {
	parseOut := make([]rune, 0, len(parseName))
	for _, parseR := range parseName {
		switch {
		case parseR >= 'a' && parseR <= 'z', parseR >= 'A' && parseR <= 'Z', parseR >= '0' && parseR <= '9', parseR == '_':
			parseOut = append(parseOut, parseR)
		default:
			parseOut = append(parseOut, '_')
		}
	}
	if len(parseOut) == 0 {
		return "gwc_state"
	}
	// The result is interpolated UNQUOTED into DDL/DML (CREATE TABLE <t>, FROM <t>),
	// and an unquoted SQL identifier may not begin with a digit — "123abc" would be
	// a syntax error at table-create time. Prefix a leading digit with '_'.
	if parseOut[0] >= '0' && parseOut[0] <= '9' {
		parseOut = append([]rune{'_'}, parseOut...)
	}
	return string(parseOut)
}
