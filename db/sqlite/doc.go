// Package sqlite is GWC's client-side SQLite battery: a database/sql-flavored,
// context-first API that runs inside the browser wasm app with no cgo.
//
// On js/wasm it is backed by github.com/ncruces/go-sqlite3 (SQLite compiled to
// wasm, executed via wazero). On native builds it is backed by
// modernc.org/sqlite so the same calling code runs in the gwc native test lane.
//
// Durability options (see Persistence):
//
//   - Memory:    in-wasm only, lost on reload.
//   - IndexedDB: the DB image is snapshotted to IndexedDB (via interop) on
//     Flush/Close and rehydrated on Open. Whole-file granularity, main-thread,
//     no Web Worker required. This is the v1 durable backend.
//   - OPFS:      reserved for incremental, worker-backed durability. In v1 it
//     transparently falls back to IndexedDB.
//
// The package deliberately exposes the standard library's *sql.Rows,
// sql.Result, *sql.Row and *sql.Tx rather than re-wrapping them, so the API is
// identical across the wasm and native builds.
package sqlite
