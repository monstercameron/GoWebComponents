# db/sqlite

Client-side SQLite for GWC — a `database/sql`-flavored, context-first API that runs **inside the
browser wasm app with no cgo**.

- **wasm:** backed by [`ncruces/go-sqlite3`](https://github.com/ncruces/go-sqlite3) (SQLite compiled
  to wasm, executed via wazero). ~2.5 MB gzip added to the bundle.
- **native:** backed by `modernc.org/sqlite`, so the same calling code runs in the `gwc` native test lane.

## Public API

```go
db, err := sqlite.Open(ctx, sqlite.Options{
    Name:        "myapp",
    Persistence: sqlite.IndexedDB, // Memory | IndexedDB | OPFS
})
defer db.Close()

db.Exec(ctx, `CREATE TABLE kv(k TEXT PRIMARY KEY, v TEXT)`)
db.Exec(ctx, `INSERT INTO kv VALUES(?, ?)`, "greeting", "hi")

var v string
db.QueryRow(ctx, `SELECT v FROM kv WHERE k = ?`, "greeting").Scan(&v)

rows, _ := db.Query(ctx, `SELECT k, v FROM kv`)
db.Tx(ctx, func(tx *sql.Tx) error { /* ... */ return nil })

db.Flush(ctx) // force-persist to the durable backend now
```

The package exposes the standard library's `*sql.Rows`, `sql.Result`, `*sql.Row` and `*sql.Tx`
directly — the surface is identical across builds.

## Persistence

| Mode        | Behavior                                                                                   |
|-------------|--------------------------------------------------------------------------------------------|
| `Memory`    | In-wasm only; lost on reload.                                                               |
| `IndexedDB` | DB image snapshotted to IndexedDB on `Flush`/`Close`, rehydrated on `Open`. Main-thread, no worker. **(v1 default durable backend)** |
| `OPFS`      | Reserved for incremental, Web-Worker-backed durability. **Falls back to `IndexedDB` in v1.** |

Durability happens at `Flush`/`Close` points. Consumers that need write-through (e.g. the state
binding) drive their own debounced `Flush`.

### How IndexedDB durability works
`Memory`/`IndexedDB` use `gwcmem`, a snapshot-able in-memory `vfs.VFS` (adapted from ncruces' `memdb`)
that owns its byte buffer and exposes snapshot/restore. On `Open` the image is read from IndexedDB
(via `interop.PersistentStore`, base64) and restored before the connection opens; on `Flush` the image
is snapshotted back out and written. Whole-file granularity — ideal for small KV state.

## Files
- `sqlite.go` — shared types (`DB`, `Options`, `Persistence`), pragmas, helpers.
- `open_wasm.go` / `open_native.go` — build-specific `Open`.
- `gwcmem_wasm.go` — the snapshot-able in-memory VFS.
- `persist_wasm.go` — the IndexedDB durability backend.

## Tests
- Native: `go test ./db/sqlite/`
- Wasm:   `GOOS=js GOARCH=wasm go test -exec=<node wasm runner> ./db/sqlite/`
