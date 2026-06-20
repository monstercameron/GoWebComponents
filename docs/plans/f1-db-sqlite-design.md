# F1.2 Design — `db/sqlite` package

Client-side, browser-wasm SQLite for GWC. Engine: `ncruces/go-sqlite3` (spike-confirmed: builds+runs
under `js/wasm`, no cgo, 2.5 MB gzip, public `vfs.Register` seam). `database/sql`-flavored API, ctx-first.
Native builds run the same calling code over `modernc.org/sqlite` so unit tests pass in the `gwc` native lane.

Status: **design** (task F1.2). Implements into task F1.3. Feeds Feature 2 (transparent KV state).

---

## Package & file layout

```
db/sqlite/
  doc.go              // package godoc
  sqlite.go           // shared: DB/Tx/Rows/Row types, Options, errors (build-tag-free)
  open_wasm.go        //go:build js && wasm     — Open() wiring ncruces + chosen VFS
  open_native.go      //go:build !(js && wasm)  — Open() over modernc.org/sqlite (test parity)
  persist.go          // shared Persistence enum + the persistence seam (interface)
  persist_indexeddb_wasm.go  // v1: memdb image <-> interop.PersistentStore (IndexedDB)
  persist_opfs_wasm.go       // v2 stub: OPFS-in-worker VFS (interface satisfied, impl later)
  README.md
  sqlite_test.go      // native lane
  sqlite_wasm_test.go // wasm lane (gwc test -lane wasm)
```

Module path: `github.com/monstercameron/GoWebComponents/db/sqlite`. House style: `parse`-prefixed params.

---

## Public API (`sqlite.go`)

`database/sql`-shaped but trimmed to what wasm needs, ctx-first throughout.

```go
type Persistence int
const (
    Memory     Persistence = iota // in-wasm only, lost on reload
    IndexedDB                     // v1 durability: DB image serialized to IndexedDB (interop.PersistentStore)
    OPFS                          // v2 durability: incremental, DB runs in a Web Worker; falls back to IndexedDB
)

type Options struct {
    Name        string            // logical db name → IndexedDB/OPFS key (default "gwc")
    Persistence Persistence       // default Memory
    Pragmas     map[string]string // e.g. {"foreign_keys":"1","busy_timeout":"5000"} → applied on open
    // v2 seam (ignored in v1): when set, IndexedDB/OPFS writes are debounced by this much.
    FlushDebounce time.Duration
}

type DB struct { /* unexported: conn, persistence backend, mutex */ }

func Open(parseCtx context.Context, parseOptions Options) (*DB, error)

func (parseDB *DB) Exec(parseCtx context.Context, parseSQL string, parseArgs ...any) (Result, error)
func (parseDB *DB) Query(parseCtx context.Context, parseSQL string, parseArgs ...any) (*Rows, error)
func (parseDB *DB) QueryRow(parseCtx context.Context, parseSQL string, parseArgs ...any) *Row
func (parseDB *DB) Tx(parseCtx context.Context, parseFn func(*Tx) error) error // begins, runs fn, commits/rolls back
func (parseDB *DB) Flush(parseCtx context.Context) error // force-persist now (also called on Close)
func (parseDB *DB) Close() error

type Tx struct { /* same Exec/Query/QueryRow surface */ }
type Result interface{ LastInsertId() (int64, error); RowsAffected() (int64, error) }
type Rows  struct{ /* Next() bool; Scan(...any) error; Close() error; Columns() []string */ }
type Row   struct{ /* Scan(...any) error */ }
```

Notes:
- `Result/Rows/Row` are thin wrappers over `database/sql` on native and over ncruces stmts on wasm,
  so the *exported* surface is identical across builds — that's what makes the native test lane honest.
- No connection pool exposed: a browser app is single-process; `DB` holds one connection guarded by a mutex.
  (Pooling is meaningless in-wasm and a footgun with the memdb VFS.)

---

## The persistence seam (`persist.go`)

This is the v1/v2 swap point. `Open` picks a backend by `Options.Persistence`; the backend is an
interface so OPFS slots in later with zero API change.

```go
// persistBackend is how a *DB makes itself durable. memory => nil backend.
type persistBackend interface {
    // hydrate loads a previously-saved DB image (nil, false if none) before the connection opens.
    hydrate(ctx context.Context, name string) ([]byte, bool, error)
    // persist saves the current DB image. Called on Flush/Close (and debounced writes in v2).
    persist(ctx context.Context, name string, image []byte) error
}
```

### v1 — `IndexedDB` (`persist_indexeddb_wasm.go` + `gwcmem_wasm.go`)
- **Implementation note (corrected during F1.3):** ncruces' stock `memdb` has *no public image
  export* and there's no `sqlite3_serialize` binding, so we ship **our own** snapshot-able in-memory
  VFS `gwcmem` — a direct adaptation of `memdb` (the public `vfs.VFS`/`vfs.File` template) that owns
  its byte buffer and adds `gwcmemSnapshot(name)`/`gwcmemRestore(name, image)`. Registered via
  `vfs.Register("gwcmem", …)`; selected with DSN `file:/<name>?vfs=gwcmem`. Synchronous, main-thread,
  no worker. This is the v1 durability primitive.
- On `Open`: `hydrate` reads the image from `interop.PersistentStore` (base64 string value) and seeds
  it via `gwcmemRestore(name, image)` **before** the SQLite connection opens.
- On `Flush`/`Close`: `gwcmemSnapshot(name)` reads the image back out and `persist`s it (base64) to
  `interop.PersistentStore`. (Durability is at Flush/Close points — Feature 2 drives debounced Flush.)
- Both builds talk to SQLite through `database/sql` (ncruces registers driver `sqlite3`; modernc
  registers `sqlite`), so the package exposes stdlib `*sql.Rows`/`sql.Result`/`*sql.Row`/`*sql.Tx`
  directly — only `Open` + the persistence wiring are build-specific. `SetMaxOpenConns(1)` keeps the
  single in-wasm connection coherent. Pragmas are applied via `PRAGMA` Exec after open (driver-agnostic).
- `interop.PersistentStore` is the existing IndexedDB wrapper (`OpenPersistentStore`, `SetItem`/`GetItem`,
  string values → base64 the binary image; v2 can store raw ArrayBuffer for efficiency).
- Whole-file granularity. Perfectly adequate for transparent-KV state (Feature 2), which writes small blobs.

### v2 — `OPFS` (`persist_opfs_wasm.go`, stub now)
- A real `vfs.VFS`/`vfs.File` backed by OPFS **sync access handles**, which require a **Web Worker**
  (`interop.OpenGoWASMWorker` already exists). DB runs in the worker; main thread talks to it via the
  existing message-channel interop. Incremental writes, large DBs, no whole-file rewrite.
- Feature-detect OPFS (`navigator.storage.getDirectory` + secure context); fall back to v1 `IndexedDB`.
- Designed-in now via the `persistBackend` interface so adding it is non-breaking.

### native (`open_native.go`)
- `Memory` → `modernc.org/sqlite` `:memory:`. `IndexedDB`/`OPFS` → a temp-file DB under the OS temp dir
  (durability semantics approximated for tests). Mirrors the existing `examples/.../db/sqlite.go` adapter.

---

## Open question for review (small)
- **Args binding style**: positional `?`/`$1` only (matches `database/sql`) — yes, keep it minimal; named
  params can come later via a helper. No struct-scan in v1 (that's the "slim API" path we explicitly did
  not choose).

## What F1.3 implements, in order
1. `sqlite.go` shared types + `open_native.go` (get the native test lane green first — fastest feedback).
2. `open_wasm.go` with `Memory`.
3. `persist.go` seam + `persist_indexeddb_wasm.go` (v1 durability).
4. `persist_opfs_wasm.go` stub (interface satisfied, returns "use IndexedDB" until v2).
5. Tests both lanes + `examples/public/` demo: counter table that survives reload.
