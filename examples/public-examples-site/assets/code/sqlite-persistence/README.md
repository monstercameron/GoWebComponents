# SQLite Persistence

A counter whose value is stored in a **client-side SQLite database** (`db/sqlite`) backed by
IndexedDB. Increment/decrement, then **reload the page** — the count is still there.

No server and no cgo: SQLite runs as wasm in the browser (via `ncruces/go-sqlite3` + wazero), and the
database image is snapshotted to IndexedDB on each `Flush` and rehydrated on `Open`.

## What it shows
- Opening a durable database: `sqlite.Open(ctx, sqlite.Options{Name, Persistence: sqlite.IndexedDB})`.
- Reading initial state from SQLite on mount (inside `UseEffect`, off the render path).
- Writing + `Flush` on each change so the value survives a reload.
- Cleaning up the connection on unmount.

## Run
```
gwc dev examples/public/sqlite-persistence
```
Open the page, change the count, reload — it persists. (Requires a browser; IndexedDB is not available
under headless Node.)
