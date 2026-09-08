package hotreload

import (
	"encoding/json"

	"github.com/monstercameron/GoWebComponents/v6/state"
)

// snapshotWedgeShouldClear decides whether a stored hot-reload snapshot that
// FAILED to restore should be cleared to avoid wedging the restore path. A
// well-formed JSON payload that still failed to apply is an un-restorable
// snapshot (protocol/schema/migration mismatch or bad state) that will re-fail
// identically on every future reload — clearing it lets the next reload start
// fresh instead of silently retrying the same broken restore forever. An
// ill-formed payload, by contrast, is likely a truncated mid-write the dev
// server is about to rewrite, so it is kept (clearing could destroy recoverable
// data). JSON validity is the exact discriminator between the two classes.
func snapshotWedgeShouldClear(parsePayload string) bool {
	return json.Valid([]byte(parsePayload))
}

// Config controls how the hot reload bridge captures state.
type Config struct {
	// AtomIDs limits exported atom state to the provided ids.
	// When empty, all exported atom state is included.
	AtomIDs []string

	// ResetKey opts into explicit snapshot invalidation. When the reset key
	// changes between builds, previously exported local and shared state is
	// discarded instead of being restored into the new module.
	ResetKey string

	// SnapshotVersion names the app-owned hot reload snapshot schema.
	//
	// It defaults to 1. Increment it when refactors require atom-state or
	// component identity migrations before an old browser snapshot can be
	// restored into the new module.
	SnapshotVersion int

	// SnapshotMigrations migrates old app-owned snapshot schemas into
	// SnapshotVersion before restore. Migrations are applied in version order
	// and may rewrite shared atom state plus component path or identity aliases.
	SnapshotMigrations []SnapshotMigration
}

// SnapshotMigration rewrites one app-owned hot reload snapshot schema version.
type SnapshotMigration struct {
	FromVersion int
	ToVersion   int

	// MigrateState rewrites the shared atom snapshot for this version step.
	MigrateState func(SnapshotMigrationContext) (state.Snapshot, error)

	// ComponentPathAliases rewrites stable component paths captured by the
	// runtime. Use it when a component moved but should keep compatible local
	// hook state.
	ComponentPathAliases map[string]string

	// ComponentIdentityAliases rewrites component identity strings captured in
	// signatures and identity trails. Use it when a component was renamed or
	// moved but its hook shape remains compatible.
	ComponentIdentityAliases map[string]string
}

// SnapshotMigrationContext is passed to SnapshotMigration.MigrateState.
type SnapshotMigrationContext struct {
	FromVersion int
	ToVersion   int
	State       state.Snapshot
}

// Enable installs the default hot reload bridge for development builds.
func Enable() {
	Configure(Config{})
}
