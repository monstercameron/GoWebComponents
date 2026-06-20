# Plan: SQLite driver, state persistence, typed CSS

Three new "batteries" for GWC, planned to be built **one at a time**. Tracked as Claude Code
tasks (see `TaskList`). This doc holds the design detail that won't fit in task titles.

Decisions locked with the user (2026-06-15):
- **SQLite runs client-side**, inside the browser wasm app, persisted to OPFS/IndexedDB. Local-first.
- **Engine: `ncruces/go-sqlite3`** (no cgo, wasm + OPFS VFS). Hard rule: **no cgo anywhere**.
- **DB API is `database/sql`-flavored** (Open/Exec/Query/QueryRow/Tx, ctx-first, wasm-safe).
- **State↔DB is transparent KV persistence** — state serialized into a generic table behind the
  existing persisted-state/atom APIs. SQLite is an upgraded durability backend. Live relational
  queries are explicitly **out of scope** for now (possible later layer, same driver).
- **Typed CSS replaces Tailwind as the primary system**: a typed raw-CSS layer (scoped, real CSS) is
  the foundation; Tailwind-style utilities become a typed convenience built on top. No Tailwind
  toolchain dependency. Existing class strings keep working for gradual migration.
- **CSS authoring must feel as natural as today's `Class(...)` / shorthand mixed-args.** The emission
  mechanism (runtime injection vs build-time extraction) is **subordinate to that ergonomic** and is a
  deliberate design item, not yet decided — see Feature 3.

---

## Context that shapes everything

GWC compiles primarily to **`GOOS=js GOARCH=wasm`** (every component file is `//go:build js && wasm`).

The repo already uses SQLite, but **only on the native/server side**. The current code path
*explicitly stubs it out for the browser*:

- `examples/server/atlas-commerce-os/server/db/sqlite.go` — native: `_ "modernc.org/sqlite"`, real `Open`.
- `examples/server/atlas-commerce-os/server/db/sqlite_wasm.go` — `//go:build js && wasm` returns
  `"sqlite is unavailable for js/wasm builds"`.

The engine is settled: **`ncruces/go-sqlite3`** (already a direct dep). It is **no-cgo** (runs
SQLite-compiled-to-wasm via the **wazero** runtime, an indirect dep) and ships **browser/wasm support
with an OPFS VFS**. Hard constraint from the user: **no cgo** — this rules out `mattn/go-sqlite3` and
any cgo driver. `modernc.org/sqlite` stays as the native-build adapter only (it doesn't target js/wasm).

Feature 1 still opens with a short **validation spike** (does it build+run+persist under our `gwc`
wasm toolchain, and what's the binary-size/latency cost?) — but it's a confidence check on a chosen
engine, not an open engine search.

Conventions to mirror (house style):
- `parse`-prefixed params, `Use*` hooks for component-bound APIs, dual-build `*_wasm.go` / `*_native.go`.
- Pluggable backends are structs-of-closures (see `interop.Storage`); persistence already has a tiered
  model in `interop` (`Storage` = localStorage, `PersistentStore` = IndexedDB+fallback).
- State layers: local `ui.UseState`, persisted `ui.UsePersistedState`, global `state.UseAtom` +
  snapshots (`state.SavePersistentSnapshot` / `LoadPersistentSnapshot`, IndexedDB-backed).
- Styling today: Tailwind class strings + `html.ClassNames` / `When` / `ClassMap` (`html/sugar.go`).
  No typed CSS exists — net-new, no collisions.

---

## Feature 1 — Client-side pure-Go SQLite driver (`db/sqlite`)

Goal: a SQLite engine usable from inside a GWC wasm app, no cgo, durable across reloads.

### Engine: `ncruces/go-sqlite3` (no cgo, wasm + OPFS VFS)
The spike is a confidence check, not an engine search:
- Confirm it **builds and runs under the `gwc` wasm toolchain** (`GOOS=js GOARCH=wasm`), wired through
  wazero, from inside an actual GWC component — not just a bare `main`.
- Confirm **OPFS persistence survives a reload** (and feature-detect → fall back when OPFS is absent,
  e.g. non-secure context or unsupported browser).
- Record **binary-size delta** (wazero + SQLite wasm blob is not free) and first-query latency.
- Hard constraint: **no cgo anywhere** in the chosen path.

Deliverable: a throwaway example that opens a DB, runs `CREATE/INSERT/SELECT` in the browser, reloads,
and shows the data persisted. If size/latency is unacceptable, the only fallback considered is the
official `sqlite3.wasm` + OPFS driven via `syscall/js` (also no-cgo) — but `ncruces` is the default.

### Spike results (2026-06-15) — GO ✅
Probed `ncruces/go-sqlite3@v0.32.0` (already a direct dep):
- **Compiles** for `GOOS=js GOARCH=wasm`, **no cgo**. **Runs**: executed under Node via Go's
  `wasm_exec_node.js`, ran CREATE/INSERT/SELECT through wazero-hosted SQLite, returned the row.
- **Size**: 8.9 MB uncompressed → **2.5 MB gzip** (SQLite + wazero over Go's ~2 MB wasm baseline).
- **VFS extensibility confirmed**: public `vfs.Register(name, vfs.VFS)` + clean `vfs.VFS`/`vfs.File`
  interfaces; the `memdb` package is a complete ~300-line custom-VFS reference. A custom OPFS VFS is
  scoped engineering, not a research risk. Select a VFS per-connection via the DSN (`?vfs=<name>`).
- **Constraint that drives F1.2**: `vfs.File` is **synchronous** (`ReadAt`/`WriteAt`). Browser OPFS
  *sync access handles* (`createSyncAccessHandle`) exist **only in a Web Worker**. So:
  - **v1 (main-thread, ship first)**: run the `memdb` in-wasm VFS and **serialize the DB image to
    IndexedDB** via the existing `interop.PersistentStore` on a debounced write. Whole-file
    granularity, zero worker plumbing — and exactly enough for Feature 2's transparent-KV need.
  - **v2 (best, later)**: a true OPFS-backed `vfs.VFS` running the DB **in a Web Worker** for
    incremental, large-DB durability. Feature-detect OPFS; fall back to v1.
- Throwaway probe was removed after the run; findings live here.

### API shape (after engine chosen)
A small `database/sql`-flavored surface that is wasm-safe and async-aware:
- `Open(ctx, Options) (*DB, error)` — Options: `Name`, `Persistence` (`Memory|OPFS|IndexedDB`), pragmas.
- `DB.Exec(ctx, sql, args...)`, `DB.Query(ctx, sql, args...)`, `DB.QueryRow`, `DB.Tx(ctx, fn)`.
- An **OPFS-backed VFS** for durability + `Memory` and `IndexedDB` fallbacks (feature-detect OPFS).
- Native build parity: thin adapter over `modernc.org/sqlite` so the same code runs in `gwc test` unit lane.

### Tasks: spike → design API+VFS → implement+example+tests.

---

## Feature 2 — State ↔ SQLite persistence binding (configurable)

Goal: opt a piece of state into SQLite-backed durability with one call, configurably.

Reuse, don't reinvent: model after `ui.UsePersistedState` (localStorage) and the snapshot system, but
back it with Feature 1.

### API shape
- `UseSQLitePersistedState[T](ctx, key, initial, Options) PersistedState[T]` — component-bound.
- Atom adapter: `state.BindAtomToSQLite(atomID, Options)` for global state durability.
- `Options` (the "configurable" ask): `Table` (default `gwc_state`), `Codec` (JSON|CBOR — CBOR dep
  already present), `WriteStrategy` (`Immediate|Debounced(d)|OnUnload`), `Hydrate` (eager|lazy),
  `Conflict` (last-write-wins | versioned via existing `RegisterSnapshotMigration`).
- Cross-tab consistency: reuse the storage-event pattern already in `UsePersistedState`.

### Extensibility — users develop their own strategies (required)
Every configurable axis is also a **public interface a user can implement**, not just a built-in enum.
Follow GWC's house pattern (structs-of-closures like `interop.Storage`, registered like `plugin`):
- `PersistenceBackend` — the store contract (`Load(ctx, key)`, `Save(ctx, key, blob)`, `Delete`,
  `Keys`, `Watch`). SQLite is the default impl; users can supply their own (e.g. REST, custom OPFS
  layout) and bind state to it through the same `UseSQLitePersistedState`/`BindAtom*` surface.
- `WriteStrategy` interface (when/how a dirty value is flushed) — Immediate/Debounced/OnUnload ship as
  built-ins, but it's an interface so users write batching, throttle, or network-aware strategies.
- `Codec` interface (encode/decode) and `ConflictResolver` interface (merge/last-write-wins/versioned).
- A small registry so a user strategy can be named and reused across the app (mirror `flags.BuildSet`).
Generalize the names if needed (it's really a pluggable durable-state engine that *defaults* to SQLite).

### Tasks: design binding+config+extension interfaces → implement built-ins + a custom-strategy example + tests.
Depends on Feature 1's driver API.

---

## Feature 3 — Typed CSS system (typed CSS as primary; Tailwind utilities built on top)

Goal: replace the Tailwind toolchain with a **typed, type-safe CSS system** where the raw-CSS layer is
the foundation and Tailwind-style utilities are a typed convenience layer over it. No external Tailwind
build step. Existing class strings keep working during migration.

### The non-negotiable: authoring ergonomics
The user's hard requirement: **it must feel as natural as the way `html`/`shorthand` take classes
today.** The design task's *first job* is a deep read of the real authoring surface before any API is
proposed — concretely:
- `html/sugar.go` — `Class()`, `ClassNames(parts ...any)`, `When`, `ClassMap`.
- `html/shorthand/shorthand.go` — the mixed-arg variadic sugar (how `Div(Class(...), ...)` and bare
  string/attr args are interpreted and ordered).
- `html/html.go` — `Props` (incl. `Style map[string]string`), how attrs are applied.

The typed CSS API must drop into those call sites with **no ceremony** — i.e. whatever `css.*` produces
has to be accepted by `Class(...)` / the shorthand arg list exactly like a string is today. That
interop contract drives every other choice below.

### Layer 1 (foundation) — typed raw CSS
- Typed style values with valid property names/units → emit **real scoped CSS** (hashed class + a
  stylesheet), expressing what inline styles can't (`:hover`, media, keyframes).
  Sketch: `css.New(css.Display.Flex, css.Gap(css.Px(8)), css.Hover(css.Bg(css.Slate900)))` → a class name.
- Must integrate with SSR (`ui/ssr_*`) so styles are present on first paint (no FOUC).

### Layer 2 (convenience) — typed utilities over Layer 1
- A typed, Tailwind-shaped vocabulary (`u.Flex`, `u.Gap(3)`, `u.Text(u.Lg)`) implemented as named
  bundles of Layer-1 rules — autocompletable, typo-proof, no Tailwind config to maintain.
- Curated token set to start (spacing/color/type scales), expandable; generated where it pays off.

### Open design items the design task must settle (with the ergonomic as the deciding vote)
- **Interop shape**: does `css.New(...)` return a `string` (drops straight into `Class`/`ClassNames`),
  or a typed value the shorthand layer is taught to accept? Pick whichever reads most naturally at the
  call site — likely a string-yielding type so zero plumbing changes are needed.
- **Emission**: runtime `<style>` injection (simplest, ships first) vs build-time extraction via
  `tools/gwc` (zero runtime cost). Decide *after* the interop shape, since extraction constrains how
  dynamic the API can be. Lean runtime-first, extraction as a later optimization — but only if it
  doesn't compromise the authoring feel.

### Extensibility — users develop their own strategies (required)
The system must be open at every layer so users build their own design systems on top, not just consume ours:
- **Custom utilities/tokens**: a registration API to define new typed utilities and token scales
  (`css.DefineUtility(...)`, `css.Theme(tokens)`) that compose with Layer-1 rules exactly like the
  built-ins — so a user can ship their own `u.*`-style vocabulary or a branded theme.
- **Custom variants**: an interface to register new variant selectors/conditions
  (`css.DefineVariant("group-hover", ...)`, container queries, data-attr states) usable like `Hover`.
- **Pluggable emission `Sink`**: the thing that turns rules into output is an interface (default =
  runtime `<style>` injection). Users (or the SSR/build path) can supply an extraction sink, a
  CSS-file writer, or a custom target — without touching authoring code.
- **Interop contract is public**: whatever `css.New(...)` returns is a documented type other libraries
  can produce/consume, so third-party styling layers drop into `Class(...)`/shorthand too.

### Tasks: deep-study authoring surface + design layers, emission & extension APIs → implement built-ins + a custom-utility/theme example + tests.
Independent of Features 1–2.

---

## Sequencing

One at a time, in task-ID order. Feature 1 gates Feature 2. Feature 3 is independent and can slot in
whenever. Each "implement" task ends with a runnable example under `examples/public/` and tests in both
the native and wasm lanes (`gwc test`).
