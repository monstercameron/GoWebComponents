# F2.1 Design — state ↔ SQLite persistence binding (`kvstate`)

Opt any piece of state into **transparent, durable KV** backed by the `db/sqlite` driver, with every
configurable axis also a **public interface** users can implement. Models `ui.UsePersistedState`
(localStorage) but routed through SQLite/IndexedDB.

Status: **design** (task F2.1). Implements into F2.2. Depends on `db/sqlite` (done).

---

## Package & entry points

New package `kvstate` (working name — bikeshed later; alternatives `db/kv`, `state/persist`). It imports
`ui`, `state`, `db/sqlite`, `interop` — none of those import it, so no cycle. Two ergonomic entry points,
one engine:

```
kvstate/
  doc.go
  engine.go          // the durable-state engine: owns *sqlite.DB + the KV table (default backend)
  options.go         // Options + defaults
  backend.go         // PersistenceBackend interface + SQLite default impl
  strategy.go        // WriteStrategy interface + Immediate/Debounced/OnUnload
  codec.go           // Codec interface + JSON/CBOR
  conflict.go        // ConflictResolver interface + LastWriteWins/Versioned
  registry.go        // named registry for user strategies
  hook_wasm.go       // UsePersistedState[T] (component-bound; calls ui.UseState/UseEffect)
  hook_native.go     // native degrade: plain in-memory state
  atom.go            // BindAtom[T] for global atoms
  *_test.go, README.md
```

Why async changes the shape: `db/sqlite.Open` and `Flush` are context/async; the DB is not ready at
hook-call time. So unlike `UsePersistedState` (synchronous localStorage read), the value starts at
`initial` and is **hydrated then `Set`** once the engine is ready (inside `UseEffect`, off the render path).

---

## The engine (`engine.go`) — default `PersistenceBackend`

A lazily-created singleton per `(Name, backend)` that owns one `*sqlite.DB` and the KV table. All keys in
an app share one DB/connection (we do **not** open a DB per key).

Table (created `IF NOT EXISTS`, name = `Options.Table`, default `gwc_state`):
```sql
CREATE TABLE gwc_state (
  k          TEXT PRIMARY KEY,
  v          BLOB    NOT NULL,   -- codec output
  version    INTEGER NOT NULL,   -- monotonic per key; drives Versioned conflict
  updated_at INTEGER NOT NULL    -- ms; drives LastWriteWins
);
```

Set = `INSERT … ON CONFLICT(k) DO UPDATE` then a `Flush` governed by the WriteStrategy. (The row write
is cheap in-wasm; the expensive durable step is the IndexedDB image snapshot in `db/sqlite.Flush`, which
is exactly what the strategy paces.)

---

## Configurable surface (`options.go`)

```go
type Options struct {
    Name     string             // logical DB name → db/sqlite Options.Name (default "gwc")
    Table    string             // KV table name (default "gwc_state")
    Codec    Codec              // default JSONCodec
    Strategy WriteStrategy      // default Debounced(150ms)
    Conflict ConflictResolver   // default LastWriteWins
    Hydrate  HydrateMode        // Eager (load on mount) | Lazy (load on first Get) — default Eager
    Persistence sqlite.Persistence // default sqlite.IndexedDB
    Backend  PersistenceBackend // default nil → the SQLite engine above
}
```
Any field left zero takes its default. A user overrides exactly the axes they care about.

---

## Extension interfaces (the "users build their own strategies" requirement)

Every axis is an interface with shipped built-ins, following the house struct-of-closures / registry style.

```go
// backend.go — swap SQLite for anything (REST, custom OPFS layout, a mock).
type PersistenceBackend interface {
    Load(ctx context.Context, key string) (record Record, found bool, err error)
    Save(ctx context.Context, rec Record) error
    Delete(ctx context.Context, key string) error
    Keys(ctx context.Context) ([]string, error)
    Watch(ctx context.Context, key string, onChange func(Record)) (cancel func(), err error) // cross-tab/live
}
type Record struct { Key string; Value []byte; Version int64; UpdatedAt int64 }

// strategy.go — when does a dirty value become durable?
type WriteStrategy interface {
    // OnWrite is called after each Set; it decides when to invoke flush.
    OnWrite(ctx context.Context, key string, flush func(context.Context) error)
    Close(ctx context.Context) error // final flush on teardown
}
// built-ins: Immediate{}, Debounced(d), OnUnload{} (flushes on pagehide/visibilitychange)

// codec.go
type Codec interface { Encode(v any) ([]byte, error); Decode(data []byte, v any) error }
// built-ins: JSONCodec{}, CBORCodec{} (fxamacker/cbor — already a dep)

// conflict.go — incoming (cross-tab / backend) vs local
type ConflictResolver interface { Resolve(local, incoming Record) (winner Record) }
// built-ins: LastWriteWins{} (max updated_at), Versioned{} (max version; reject stale)
```

### Registry (`registry.go`)
```go
func RegisterStrategy(name string, s WriteStrategy)   // then Options can reference by name app-wide
func RegisterCodec(name string, c Codec)
func RegisterBackend(name string, b PersistenceBackend)
func StrategyByName(name string) (WriteStrategy, bool) // … etc.
```
So a user defines a custom strategy once and reuses it everywhere by name (mirrors `flags.BuildSet`).

---

## Entry point 1 — component-bound hook (`hook_wasm.go`)

```go
func UsePersistedState[T any](parseKey string, parseInitial T, parseOptions ...Options) PersistedState[T]
// PersistedState[T]: Get() T / Set(T) / Loading() bool / Err() error
```
Behavior:
- `parseState := ui.UseState(parseInitial)`; `loading := ui.UseState(true)`.
- `ui.UseEffect` (deps: key) → engine ready → `Load` the key via Codec → `Set` state + `loading=false`;
  open a `Watch` (BroadcastChannel via `interop.OpenCrossTabChannel`) so other tabs' writes re-`Load`
  and apply through `ConflictResolver`. Cleanup cancels the watch.
- Returned `Set` is write-through: updates in-memory state, `Save`s via backend (bumping version +
  updated_at), and hands `flush` to the `WriteStrategy`.
- `hook_native.go` degrades to plain `ui.UseState` (no engine), matching `UsePersistedState`'s native path.

## Entry point 2 — global atom adapter (`atom.go`)

```go
func BindAtom[T any](parseCtx context.Context, parseAtom state.Atom[T], parseKey string, parseOptions ...Options) BoundAtom[T]
// BoundAtom[T]: Get/Set/Update (write-through) + Loading()/Err()
```
- On bind: async `Load` key → `parseAtom.Set(value)`; subscribe `Watch` for cross-tab.
- Returns a write-through wrapper whose `Set/Update` persist. (Atoms expose no global out-of-component
  subscribe, so — exactly like `UsePersistedState` — persistence is captured through the returned
  handle. Open follow-up: a runtime-level atom observer to capture writes from *any* handle; not needed
  for v1.)

---

## Cross-tab consistency
SQLite writes do **not** fire `storage` events, so we use `interop.OpenCrossTabChannel`
(BroadcastChannel): after a durable `Save`, broadcast `{key, version}`; other tabs re-`Load` that key and
apply via `ConflictResolver`. This replaces the `storage`-event mechanism `UsePersistedState` relies on.

---

## Defaults (so the simple case is one line)
```go
s := kvstate.UsePersistedState("draft", "")          // JSON, Debounced(150ms), IndexedDB, LWW, eager
s.Set("hello")                                        // persists; survives reload
```

## What F2.2 implements, in order
1. `engine.go` + `backend.go` (SQLite default) + `options.go` defaults — get a durable round-trip.
2. `codec.go`, `strategy.go`, `conflict.go` built-ins.
3. `hook_wasm.go`/`hook_native.go` + `atom.go`.
4. `registry.go`.
5. Cross-tab `Watch` (BroadcastChannel).
6. Tests (native + wasm) + an example (persisted form draft) + a **custom-strategy example** proving the
   extension interfaces plug in through the public API.
