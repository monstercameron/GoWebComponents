// Package kvstate opts GWC state into transparent, durable key/value storage
// backed by the client-side SQLite driver (db/sqlite).
//
// It mirrors ui.UsePersistedState (which uses localStorage) but routes through
// SQLite/IndexedDB, and every configurable axis is also a public interface a
// user can implement to build their own strategy:
//
//   - PersistenceBackend — where values live (default: a shared SQLite engine).
//   - WriteStrategy      — when a dirty value becomes durable (Immediate, Debounced, OnUnload).
//   - Codec              — how values are encoded (JSONCodec, CBORCodec).
//   - ConflictResolver   — how concurrent/cross-tab writes are reconciled (LastWriteWins, Versioned).
//
// Two ergonomic entry points share one engine:
//
//   - UsePersistedState[T] — a component-bound hook (async hydrate, write-through).
//   - BindAtom[T]          — a global-atom adapter.
//
// On native/SSR builds the underlying ui/interop stubs degrade gracefully; the
// SQLite engine itself still works (file-backed) so the durable logic is testable.
package kvstate
