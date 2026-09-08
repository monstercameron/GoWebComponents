//go:build js && wasm && !production

package hotreload

import (
	"encoding/json"
	"errors"
	"strings"
	"syscall/js"
	"testing"

	runtimepkg "github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/state"
)

// storeHotReloadTestGlobal replaces a global value for the duration of a wasm hotreload test.
func storeHotReloadTestGlobal(parseT *testing.T, parseName string, parseValue js.Value) func() {
	parseT.Helper()
	parseGlobal := js.Global()
	parseOriginal := parseGlobal.Get(parseName)
	parseGlobal.Set(parseName, parseValue)
	return func() {
		parseGlobal.Set(parseName, parseOriginal)
	}
}

// buildHotReloadTestLiveReload creates a GoLiveReload stub that serves one stored payload and records clearStoredState calls.
func buildHotReloadTestLiveReload(parsePayload string, parseClearCount *int) (js.Value, func()) {
	parseLiveReload := js.Global().Get("Object").New()
	parseGetStoredState := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		_ = parseThis
		_ = parseArgs
		return parsePayload
	})
	parseClearStoredState := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		_ = parseThis
		_ = parseArgs
		if parseClearCount != nil {
			*parseClearCount = *parseClearCount + 1
		}
		return nil
	})
	parseLiveReload.Set("getStoredState", parseGetStoredState)
	parseLiveReload.Set("clearStoredState", parseClearStoredState)
	return parseLiveReload, func() {
		parseGetStoredState.Release()
		parseClearStoredState.Release()
	}
}

// TestGetSnapshotWasmHonorsAtomFilterAndResetKey verifies filtered snapshot export and bridge metadata wiring.
func TestGetSnapshotWasmHonorsAtomFilterAndResetKey(parseT *testing.T) {
	Disable()
	runtimepkg.InitGlobalRuntime(runtimepkg.Config{Scheduler: noOpScheduler{}, Reset: true})
	runtimepkg.ClearDiagnostics()
	runtimepkg.ClearLogs()
	parseT.Cleanup(func() {
		Disable()
		runtimepkg.ClearDiagnostics()
		runtimepkg.ClearLogs()
	})

	if parseErr := runtimepkg.GetGlobalRuntime().SetAtomValue("theme", "dark"); parseErr != nil {
		parseT.Fatalf("unexpected setup error: %v", parseErr)
	}
	if parseErr := runtimepkg.GetGlobalRuntime().SetAtomValue("sidebar", "expanded"); parseErr != nil {
		parseT.Fatalf("unexpected setup error: %v", parseErr)
	}

	Configure(Config{
		AtomIDs:  []string{"theme"},
		ResetKey: " build-v1 ",
	})
	if !IsEnabled() {
		parseT.Fatal("expected compatibility IsEnabled wrapper to report enabled bridge")
	}

	parsePayload, parseErr := GetSnapshot()
	if parseErr != nil {
		parseT.Fatalf("expected snapshot export to succeed, got %v", parseErr)
	}
	var parseSnapshot bridgeSnapshot
	if parseErr2 := json.Unmarshal([]byte(parsePayload), &parseSnapshot); parseErr2 != nil {
		parseT.Fatalf("expected structured snapshot payload, got %v", parseErr2)
	}
	if parseSnapshot.Protocol != hotReloadSnapshotProtocol || parseSnapshot.Version != currentHotReloadSnapshotVersion {
		parseT.Fatalf("expected versioned snapshot payload, got protocol=%q version=%d", parseSnapshot.Protocol, parseSnapshot.Version)
	}
	if parseSnapshot.SnapshotVersion != defaultHotReloadSchemaVersion {
		parseT.Fatalf("expected default snapshot schema version, got %d", parseSnapshot.SnapshotVersion)
	}
	if parseSnapshot.ResetKey != "build-v1" {
		parseT.Fatalf("expected trimmed reset key, got %q", parseSnapshot.ResetKey)
	}
	if len(parseSnapshot.State) != 1 || parseSnapshot.State["theme"] != "dark" {
		parseT.Fatalf("expected filtered snapshot state, got %#v", parseSnapshot.State)
	}
	if _, parseFound := parseSnapshot.State["sidebar"]; parseFound {
		parseT.Fatalf("expected unselected atom to be excluded, got %#v", parseSnapshot.State)
	}

	parseBridge, parseErr := appBridge()
	if parseErr != nil {
		parseT.Fatalf("expected installed bridge, got %v", parseErr)
	}
	if parseResetKey := parseBridge.Get("resetKey"); !parseResetKey.Present() || parseResetKey.String() != "build-v1" {
		parseT.Fatalf("expected bridge reset key metadata, got %#v", parseResetKey)
	}
	if parseProtocol := parseBridge.Get("protocol"); !parseProtocol.Present() || parseProtocol.String() != hotReloadSnapshotProtocol {
		parseT.Fatalf("expected bridge protocol metadata, got %#v", parseProtocol)
	}
	if parseVersion := parseBridge.Get("version"); !parseVersion.Present() || parseVersion.Int() != currentHotReloadSnapshotVersion {
		parseT.Fatalf("expected bridge version metadata, got %#v", parseVersion)
	}
	if parseSnapshotVersion := parseBridge.Get("snapshotVersion"); !parseSnapshotVersion.Present() || parseSnapshotVersion.Int() != defaultHotReloadSchemaVersion {
		parseT.Fatalf("expected bridge snapshot version metadata, got %#v", parseSnapshotVersion)
	}
	parseAtomIDs, parseErr := parseBridge.Get("atomIDs").ToGo()
	if parseErr != nil {
		parseT.Fatalf("expected bridge atom filter metadata to decode, got %v", parseErr)
	}
	parseAtomIDList, parseOk := parseAtomIDs.([]any)
	if !parseOk || len(parseAtomIDList) != 1 || parseAtomIDList[0] != "theme" {
		parseT.Fatalf("expected bridge atom filter metadata, got %#v", parseAtomIDs)
	}
}

// TestConfigureWasmReusesInstalledBridgeWhenConfigMatches verifies same-config configure calls restore pending state without reinstalling the bridge.
func TestConfigureWasmReusesInstalledBridgeWhenConfigMatches(parseT *testing.T) {
	Disable()
	runtimepkg.InitGlobalRuntime(runtimepkg.Config{Scheduler: noOpScheduler{}, Reset: true})
	runtimepkg.ClearDiagnostics()
	runtimepkg.ClearLogs()
	parseT.Cleanup(func() {
		Disable()
		runtimepkg.ClearDiagnostics()
		runtimepkg.ClearLogs()
	})

	if parseErr := runtimepkg.GetGlobalRuntime().SetAtomValue("hot-reload-theme", "dark"); parseErr != nil {
		parseT.Fatalf("unexpected setup error: %v", parseErr)
	}

	Configure(Config{ResetKey: " build-v1 "})
	parsePayload, parseErr := GetSnapshot()
	if parseErr != nil {
		parseT.Fatalf("expected snapshot export to succeed, got %v", parseErr)
	}
	parseBridgeBefore := js.Global().Get(appBridgeGlobal)
	if !parseBridgeBefore.Truthy() {
		parseT.Fatal("expected configured bridge object")
	}

	if parseErr2 := runtimepkg.GetGlobalRuntime().SetAtomValue("hot-reload-theme", "light"); parseErr2 != nil {
		parseT.Fatalf("unexpected mutation error: %v", parseErr2)
	}

	parseClearCount := 0
	parseLiveReload, parseLiveReloadCleanup := buildHotReloadTestLiveReload(parsePayload, &parseClearCount)
	defer parseLiveReloadCleanup()
	parseRestoreLiveReload := storeHotReloadTestGlobal(parseT, liveReloadGlobal, parseLiveReload)
	defer parseRestoreLiveReload()

	Configure(Config{ResetKey: "build-v1"})
	parseBridgeAfter := js.Global().Get(appBridgeGlobal)
	if !parseBridgeAfter.Equal(parseBridgeBefore) {
		parseT.Fatal("expected same-config configure call to preserve the installed bridge object")
	}

	parseValue, parseOk := runtimepkg.GetGlobalRuntime().GetAtomValue("hot-reload-theme")
	if !parseOk || parseValue != "dark" {
		parseT.Fatalf("expected pending snapshot restore to recover atom state, got %#v (ok=%v)", parseValue, parseOk)
	}
	if parseClearCount != 1 {
		parseT.Fatalf("expected stored snapshot to be cleared after restore, got %d clears", parseClearCount)
	}
}

// TestImportSnapshotWasmHandlesEmptyLegacyAndInvalidPayloads verifies the skipped, legacy, and malformed snapshot branches.
func TestImportSnapshotWasmHandlesEmptyLegacyAndInvalidPayloads(parseT *testing.T) {
	Disable()
	runtimepkg.InitGlobalRuntime(runtimepkg.Config{Scheduler: noOpScheduler{}, Reset: true})
	runtimepkg.ClearDiagnostics()
	runtimepkg.ClearLogs()
	parseT.Cleanup(func() {
		Disable()
		runtimepkg.ClearDiagnostics()
		runtimepkg.ClearLogs()
	})

	parseResult, parseErr := importSnapshot("   ")
	if parseErr != nil || parseResult.Outcome != "skipped-empty" {
		parseT.Fatalf("expected empty payload to be skipped, got result=%+v err=%v", parseResult, parseErr)
	}

	if parseErr2 := runtimepkg.GetGlobalRuntime().SetAtomValue("legacy-theme", "light"); parseErr2 != nil {
		parseT.Fatalf("unexpected setup error: %v", parseErr2)
	}
	parseLegacyResult, parseErr := importSnapshot(`{"legacy-theme":"dark"}`)
	if parseErr != nil || parseLegacyResult.Outcome != "restored-legacy" {
		parseT.Fatalf("expected legacy payload to restore, got result=%+v err=%v", parseLegacyResult, parseErr)
	}
	if parseValue, parseOk := runtimepkg.GetGlobalRuntime().GetAtomValue("legacy-theme"); !parseOk || parseValue != "dark" {
		parseT.Fatalf("expected legacy state restore to update atom, got %#v (ok=%v)", parseValue, parseOk)
	}

	parseInvalidResult, parseErr := importSnapshot(`{`)
	if parseErr == nil || parseInvalidResult.Outcome != "error" {
		parseT.Fatalf("expected malformed payload to fail, got result=%+v err=%v", parseInvalidResult, parseErr)
	}
	if !strings.Contains(parseInvalidResult.Message, "unexpected end of JSON input") {
		parseT.Fatalf("expected malformed payload error detail, got %q", parseInvalidResult.Message)
	}
}

func TestImportSnapshotWasmRejectsFutureBridgeSnapshotVersion(parseT *testing.T) {
	Disable()
	runtimepkg.InitGlobalRuntime(runtimepkg.Config{Scheduler: noOpScheduler{}, Reset: true})
	runtimepkg.ClearDiagnostics()
	runtimepkg.ClearLogs()
	parseT.Cleanup(func() {
		Disable()
		runtimepkg.ClearDiagnostics()
		runtimepkg.ClearLogs()
	})

	if parseErr := runtimepkg.GetGlobalRuntime().SetAtomValue("future-theme", "light"); parseErr != nil {
		parseT.Fatalf("unexpected setup error: %v", parseErr)
	}
	parsePayload, parseErr := json.Marshal(bridgeSnapshot{
		Protocol: hotReloadSnapshotProtocol,
		Version:  currentHotReloadSnapshotVersion + 1,
		State:    state.Snapshot{"future-theme": "dark"},
	})
	if parseErr != nil {
		parseT.Fatalf("expected future payload marshal to succeed, got %v", parseErr)
	}
	parseResult, parseErr2 := importSnapshot(string(parsePayload))
	if parseErr2 == nil || parseResult.Outcome != "error" {
		parseT.Fatalf("expected future snapshot version to fail, got result=%+v err=%v", parseResult, parseErr2)
	}
	parseValue, parseOk := runtimepkg.GetGlobalRuntime().GetAtomValue("future-theme")
	if !parseOk || parseValue != "light" {
		parseT.Fatalf("expected rejected future snapshot not to mutate atom, got %#v ok=%t", parseValue, parseOk)
	}
	parseDiagnostics := runtimepkg.GetDiagnostics()
	if len(parseDiagnostics) == 0 || !strings.Contains(parseDiagnostics[0].Message, "hot reload snapshot protocol mismatch") {
		parseT.Fatalf("expected protocol mismatch diagnostic, got %+v", parseDiagnostics)
	}
}

func TestImportSnapshotWasmAppliesConfiguredSnapshotMigration(parseT *testing.T) {
	Disable()
	runtimepkg.InitGlobalRuntime(runtimepkg.Config{Scheduler: noOpScheduler{}, Reset: true})
	runtimepkg.ClearDiagnostics()
	runtimepkg.ClearLogs()
	parseT.Cleanup(func() {
		Disable()
		runtimepkg.ClearDiagnostics()
		runtimepkg.ClearLogs()
	})

	if parseErr := runtimepkg.GetGlobalRuntime().SetAtomValue("new-theme", "light"); parseErr != nil {
		parseT.Fatalf("unexpected setup error: %v", parseErr)
	}
	Configure(Config{
		SnapshotVersion: 2,
		SnapshotMigrations: []SnapshotMigration{{
			FromVersion: 1,
			ToVersion:   2,
			MigrateState: func(parseCtx SnapshotMigrationContext) (state.Snapshot, error) {
				if parseCtx.FromVersion != 1 || parseCtx.ToVersion != 2 {
					parseT.Fatalf("unexpected migration context versions: %+v", parseCtx)
				}
				parseCtx.State["new-theme"] = parseCtx.State["old-theme"]
				delete(parseCtx.State, "old-theme")
				return parseCtx.State, nil
			},
			ComponentPathAliases: map[string]string{
				"old/path": "new/path",
			},
			ComponentIdentityAliases: map[string]string{
				"example.Old": "example.New",
			},
		}},
	})

	parseSnapshot := bridgeSnapshot{
		Protocol:        hotReloadSnapshotProtocol,
		Version:         currentHotReloadSnapshotVersion,
		SnapshotVersion: 1,
		State:           state.Snapshot{"old-theme": "dark"},
		Components: []runtimepkg.HotReloadComponentSnapshot{{
			Path:          "old/path",
			IdentityTrail: []string{"example.Old"},
			Signature: runtimepkg.ComponentSignature{
				Kind:          "component",
				Name:          "Old",
				QualifiedName: "example.Old",
				HookKinds:     []string{"state"},
			},
			States: []interface{}{"preserved"},
		}},
	}
	parseMigrated, parseErr := migrateBridgeSnapshot(parseSnapshot)
	if parseErr != nil {
		parseT.Fatalf("expected direct snapshot migration to succeed, got %v", parseErr)
	}
	if parseMigrated.SnapshotVersion != 2 || parseMigrated.State["new-theme"] != "dark" {
		parseT.Fatalf("expected migrated snapshot state and version, got %+v", parseMigrated)
	}
	if parseMigrated.Components[0].Path != "new/path" ||
		parseMigrated.Components[0].IdentityTrail[0] != "example.New" ||
		parseMigrated.Components[0].Signature.QualifiedName != "example.New" {
		parseT.Fatalf("expected migrated component aliases, got %+v", parseMigrated.Components[0])
	}

	parsePayload, parseErr2 := json.Marshal(parseSnapshot)
	if parseErr2 != nil {
		parseT.Fatalf("expected migration payload marshal to succeed, got %v", parseErr2)
	}
	parseResult, parseErr3 := importSnapshot(string(parsePayload))
	if parseErr3 != nil || parseResult.Outcome != "restored" {
		parseT.Fatalf("expected migrated snapshot import to restore, got result=%+v err=%v", parseResult, parseErr3)
	}
	parseValue, parseOk := runtimepkg.GetGlobalRuntime().GetAtomValue("new-theme")
	if !parseOk || parseValue != "dark" {
		parseT.Fatalf("expected migrated atom to restore under new key, got %#v ok=%t", parseValue, parseOk)
	}
}

func TestImportSnapshotWasmRejectsMissingSnapshotMigration(parseT *testing.T) {
	Disable()
	runtimepkg.InitGlobalRuntime(runtimepkg.Config{Scheduler: noOpScheduler{}})
	runtimepkg.ClearDiagnostics()
	runtimepkg.ClearLogs()
	parseT.Cleanup(func() {
		Disable()
		runtimepkg.ClearDiagnostics()
		runtimepkg.ClearLogs()
	})

	if parseErr := runtimepkg.GetGlobalRuntime().SetAtomValue("migration-theme", "light"); parseErr != nil {
		parseT.Fatalf("unexpected setup error: %v", parseErr)
	}
	Configure(Config{SnapshotVersion: 2})
	parsePayload, parseErr := json.Marshal(bridgeSnapshot{
		Protocol:        hotReloadSnapshotProtocol,
		Version:         currentHotReloadSnapshotVersion,
		SnapshotVersion: 1,
		State:           state.Snapshot{"migration-theme": "dark"},
	})
	if parseErr != nil {
		parseT.Fatalf("expected missing migration payload marshal to succeed, got %v", parseErr)
	}
	parseResult, parseErr2 := importSnapshot(string(parsePayload))
	if parseErr2 == nil || parseResult.Outcome != "error" {
		parseT.Fatalf("expected missing migration to fail, got result=%+v err=%v", parseResult, parseErr2)
	}
	if !strings.Contains(parseResult.Message, "no snapshot migration") {
		parseT.Fatalf("expected missing migration detail, got %q", parseResult.Message)
	}
	parseValue, parseOk := runtimepkg.GetGlobalRuntime().GetAtomValue("migration-theme")
	if !parseOk || parseValue != "light" {
		parseT.Fatalf("expected rejected migration not to mutate atom, got %#v ok=%t", parseValue, parseOk)
	}
}

func TestImportSnapshotWasmRejectsNewerConfiguredSnapshotSchema(parseT *testing.T) {
	Disable()
	runtimepkg.InitGlobalRuntime(runtimepkg.Config{Scheduler: noOpScheduler{}})
	runtimepkg.ClearDiagnostics()
	runtimepkg.ClearLogs()
	parseT.Cleanup(func() {
		Disable()
		runtimepkg.ClearDiagnostics()
		runtimepkg.ClearLogs()
	})

	Configure(Config{SnapshotVersion: 2})
	parsePayload, parseErr := json.Marshal(bridgeSnapshot{
		Protocol:        hotReloadSnapshotProtocol,
		Version:         currentHotReloadSnapshotVersion,
		SnapshotVersion: 3,
		State:           state.Snapshot{"theme": "dark"},
	})
	if parseErr != nil {
		parseT.Fatalf("expected newer schema payload marshal to succeed, got %v", parseErr)
	}
	parseResult, parseErr2 := importSnapshot(string(parsePayload))
	if parseErr2 == nil || parseResult.Outcome != "error" {
		parseT.Fatalf("expected newer schema snapshot to fail, got result=%+v err=%v", parseResult, parseErr2)
	}
	if !strings.Contains(parseResult.Message, "newer than configured") {
		parseT.Fatalf("expected newer schema detail, got %q", parseResult.Message)
	}
}

func TestImportSnapshotWasmStateMigrationFailureIsAtomic(parseT *testing.T) {
	Disable()
	runtimepkg.InitGlobalRuntime(runtimepkg.Config{Scheduler: noOpScheduler{}, Reset: true})
	runtimepkg.ClearDiagnostics()
	runtimepkg.ClearLogs()
	parseT.Cleanup(func() {
		Disable()
		runtimepkg.ClearDiagnostics()
		runtimepkg.ClearLogs()
	})

	if parseErr := runtimepkg.GetGlobalRuntime().SetAtomValue("atomic-theme", "light"); parseErr != nil {
		parseT.Fatalf("unexpected setup error: %v", parseErr)
	}
	Configure(Config{
		SnapshotVersion: 2,
		SnapshotMigrations: []SnapshotMigration{{
			FromVersion: 1,
			ToVersion:   2,
			MigrateState: func(SnapshotMigrationContext) (state.Snapshot, error) {
				return nil, errors.New("migration failed")
			},
		}},
	})
	parsePayload, parseErr := json.Marshal(bridgeSnapshot{
		Protocol:        hotReloadSnapshotProtocol,
		Version:         currentHotReloadSnapshotVersion,
		SnapshotVersion: 1,
		State:           state.Snapshot{"atomic-theme": "dark"},
		Components: []runtimepkg.HotReloadComponentSnapshot{{
			Path: "example/App@0",
		}},
	})
	if parseErr != nil {
		parseT.Fatalf("expected atomic payload marshal to succeed, got %v", parseErr)
	}
	parseResult, parseErr2 := importSnapshot(string(parsePayload))
	if parseErr2 == nil || parseResult.Outcome != "error" {
		parseT.Fatalf("expected migration failure to return an error, got result=%+v err=%v", parseResult, parseErr2)
	}
	parseValue, parseOk := runtimepkg.GetGlobalRuntime().GetAtomValue("atomic-theme")
	if !parseOk || parseValue != "light" {
		parseT.Fatalf("expected failed migration not to mutate atom, got %#v ok=%t", parseValue, parseOk)
	}
	if runtimepkg.GetGlobalRuntime().HasPendingHotReloadSnapshot() {
		parseT.Fatal("expected failed migration not to queue component snapshots")
	}
}

func TestImportSnapshotWasmAppliesMultiStepSnapshotMigrationsInOrder(parseT *testing.T) {
	Disable()
	runtimepkg.InitGlobalRuntime(runtimepkg.Config{Scheduler: noOpScheduler{}})
	runtimepkg.ClearDiagnostics()
	runtimepkg.ClearLogs()
	parseT.Cleanup(func() {
		Disable()
		runtimepkg.ClearDiagnostics()
		runtimepkg.ClearLogs()
	})

	parseSteps := []int{}
	Configure(Config{
		SnapshotVersion: 3,
		SnapshotMigrations: []SnapshotMigration{
			{
				FromVersion: 1,
				ToVersion:   2,
				MigrateState: func(parseCtx SnapshotMigrationContext) (state.Snapshot, error) {
					parseSteps = append(parseSteps, parseCtx.ToVersion)
					parseCtx.State["theme-v2"] = parseCtx.State["theme"]
					delete(parseCtx.State, "theme")
					return parseCtx.State, nil
				},
			},
			{
				FromVersion: 2,
				ToVersion:   3,
				MigrateState: func(parseCtx SnapshotMigrationContext) (state.Snapshot, error) {
					parseSteps = append(parseSteps, parseCtx.ToVersion)
					parseCtx.State["theme-v3"] = parseCtx.State["theme-v2"]
					delete(parseCtx.State, "theme-v2")
					return parseCtx.State, nil
				},
			},
		},
	})
	parsePayload, parseErr := json.Marshal(bridgeSnapshot{
		Protocol:        hotReloadSnapshotProtocol,
		Version:         currentHotReloadSnapshotVersion,
		SnapshotVersion: 1,
		State:           state.Snapshot{"theme": "dark"},
	})
	if parseErr != nil {
		parseT.Fatalf("expected multistep payload marshal to succeed, got %v", parseErr)
	}
	parseResult, parseErr2 := importSnapshot(string(parsePayload))
	if parseErr2 != nil || parseResult.Outcome != "restored" {
		parseT.Fatalf("expected multistep migration to restore, got result=%+v err=%v", parseResult, parseErr2)
	}
	if len(parseSteps) != 2 || parseSteps[0] != 2 || parseSteps[1] != 3 {
		parseT.Fatalf("expected migrations to run in order, got %v", parseSteps)
	}
	parseValue, parseOk := runtimepkg.GetGlobalRuntime().GetAtomValue("theme-v3")
	if !parseOk || parseValue != "dark" {
		parseT.Fatalf("expected final migrated atom, got %#v ok=%t", parseValue, parseOk)
	}
}

func TestConfigureWasmUpdatesSnapshotMigrationsWithoutBridgeReinstall(parseT *testing.T) {
	Disable()
	runtimepkg.InitGlobalRuntime(runtimepkg.Config{Scheduler: noOpScheduler{}})
	runtimepkg.ClearDiagnostics()
	runtimepkg.ClearLogs()
	parseT.Cleanup(func() {
		Disable()
		runtimepkg.ClearDiagnostics()
		runtimepkg.ClearLogs()
	})

	Configure(Config{
		SnapshotVersion: 2,
		SnapshotMigrations: []SnapshotMigration{{
			FromVersion: 1,
			ToVersion:   2,
			MigrateState: func(parseCtx SnapshotMigrationContext) (state.Snapshot, error) {
				parseCtx.State["first-theme"] = parseCtx.State["old-theme"]
				return parseCtx.State, nil
			},
		}},
	})
	parseBridgeBefore := js.Global().Get(appBridgeGlobal)
	if !parseBridgeBefore.Truthy() {
		parseT.Fatal("expected configured bridge object")
	}

	Configure(Config{
		SnapshotVersion: 2,
		SnapshotMigrations: []SnapshotMigration{{
			FromVersion: 1,
			ToVersion:   2,
			MigrateState: func(parseCtx SnapshotMigrationContext) (state.Snapshot, error) {
				parseCtx.State["second-theme"] = parseCtx.State["old-theme"]
				return parseCtx.State, nil
			},
		}},
	})
	parseBridgeAfter := js.Global().Get(appBridgeGlobal)
	if !parseBridgeAfter.Equal(parseBridgeBefore) {
		parseT.Fatal("expected same bridge object when only migration callbacks changed")
	}

	parsePayload, parseErr := json.Marshal(bridgeSnapshot{
		Protocol:        hotReloadSnapshotProtocol,
		Version:         currentHotReloadSnapshotVersion,
		SnapshotVersion: 1,
		State:           state.Snapshot{"old-theme": "dark"},
	})
	if parseErr != nil {
		parseT.Fatalf("expected bridge-refresh payload marshal to succeed, got %v", parseErr)
	}
	parseResult, parseErr2 := importSnapshot(string(parsePayload))
	if parseErr2 != nil || parseResult.Outcome != "restored" {
		parseT.Fatalf("expected refreshed migration callback to restore, got result=%+v err=%v", parseResult, parseErr2)
	}
	if parseValue, parseOk := runtimepkg.GetGlobalRuntime().GetAtomValue("second-theme"); !parseOk || parseValue != "dark" {
		parseT.Fatalf("expected refreshed migration callback to run, got %#v ok=%t", parseValue, parseOk)
	}
	if _, parseFound := runtimepkg.GetGlobalRuntime().GetAtomValue("first-theme"); parseFound {
		parseT.Fatal("expected stale migration callback not to run")
	}
}

// TestImportSnapshotWasmHandlesSelectiveModes verifies selective restore outcomes for both compatible and legacy fallback payloads.
func TestImportSnapshotWasmHandlesSelectiveModes(parseT *testing.T) {
	Disable()
	runtimepkg.InitGlobalRuntime(runtimepkg.Config{Scheduler: noOpScheduler{}})
	runtimepkg.ClearDiagnostics()
	runtimepkg.ClearLogs()
	parseT.Cleanup(func() {
		Disable()
		runtimepkg.ClearDiagnostics()
		runtimepkg.ClearLogs()
	})

	parseSelectivePayload, parseErr := json.Marshal(bridgeSnapshot{
		RestoreMode:       "selective",
		ChangedComponents: []string{"example/Other"},
		Components: []runtimepkg.HotReloadComponentSnapshot{{
			Path:          "example/App",
			IdentityTrail: []string{"example/App"},
		}},
	})
	if parseErr != nil {
		parseT.Fatalf("expected selective payload marshal to succeed, got %v", parseErr)
	}
	parseSelectiveResult, parseErr := importSnapshot(string(parseSelectivePayload))
	if parseErr != nil || parseSelectiveResult.Outcome != "restored-selective" {
		parseT.Fatalf("expected selective restore result, got result=%+v err=%v", parseSelectiveResult, parseErr)
	}
	if !strings.Contains(parseSelectiveResult.Message, "1 changed component identities") {
		parseT.Fatalf("expected selective restore summary, got %q", parseSelectiveResult.Message)
	}

	parseLegacyPayload, parseErr := json.Marshal(bridgeSnapshot{
		RestoreMode: "selective",
		Components: []runtimepkg.HotReloadComponentSnapshot{{
			Path:          "example/App",
			IdentityTrail: []string{"example/App"},
		}},
	})
	if parseErr != nil {
		parseT.Fatalf("expected fallback payload marshal to succeed, got %v", parseErr)
	}
	parseLegacyResult, parseErr := importSnapshot(string(parseLegacyPayload))
	if parseErr != nil || parseLegacyResult.Outcome != "restored-legacy" {
		parseT.Fatalf("expected selective legacy fallback result, got result=%+v err=%v", parseLegacyResult, parseErr)
	}
	if !strings.Contains(parseLegacyResult.Message, "Selective hot reload fell back to full compatible restore") {
		parseT.Fatalf("expected legacy fallback detail, got %q", parseLegacyResult.Message)
	}
}

// TestHotReloadDiagnosticsAndActivityWasmFilterRecentEntries verifies hot reload diagnostics and activity filtering behavior.
func TestHotReloadDiagnosticsAndActivityWasmFilterRecentEntries(parseT *testing.T) {
	Disable()
	runtimepkg.InitGlobalRuntime(runtimepkg.Config{Scheduler: noOpScheduler{}})
	runtimepkg.ClearDiagnostics()
	runtimepkg.ClearLogs()
	parseT.Cleanup(func() {
		Disable()
		runtimepkg.ClearDiagnostics()
		runtimepkg.ClearLogs()
	})

	runtimepkg.ReportDiagnostic("runtime", runtimepkg.DiagnosticWarning, "hot reload fell back to remount for App")
	runtimepkg.ReportDiagnostic("runtime", runtimepkg.DiagnosticWarning, "plain runtime warning")
	parseDiagnostics := hotReloadDiagnostics()
	if len(parseDiagnostics) != 1 || !strings.Contains(parseDiagnostics[0].Message, "hot reload") {
		parseT.Fatalf("expected only hot reload diagnostics, got %+v", parseDiagnostics)
	}
	if len(lastRestoreResult.Diagnostics) != 1 {
		parseT.Fatalf("expected last restore result diagnostics to mirror filtered diagnostics, got %+v", lastRestoreResult.Diagnostics)
	}

	runtimepkg.ReportLogWithFields("hotreload", runtimepkg.LogInfo, runtimepkg.DiagnosticInformational, "snapshot captured", "", map[string]string{"build": "1"})
	runtimepkg.ReportLogWithFields("router", runtimepkg.LogInfo, runtimepkg.DiagnosticInformational, "route loader started", "", nil)
	runtimepkg.ReportLogWithFields("router", runtimepkg.LogInfo, runtimepkg.DiagnosticInformational, "before-enter guard passed", "", nil)
	runtimepkg.ReportLogWithFields("router", runtimepkg.LogInfo, runtimepkg.DiagnosticInformational, "before-leave guard passed", "", nil)
	runtimepkg.ReportLogWithFields("router", runtimepkg.LogInfo, runtimepkg.DiagnosticInformational, "navigation commit queued", "", nil)
	runtimepkg.ReportLogWithFields("router", runtimepkg.LogInfo, runtimepkg.DiagnosticInformational, "navigation commit finished", "", nil)
	runtimepkg.ReportLogWithFields("router", runtimepkg.LogInfo, runtimepkg.DiagnosticInformational, "navigation cache refreshed", "", nil)
	runtimepkg.ReportLogWithFields("router", runtimepkg.LogInfo, runtimepkg.DiagnosticInformational, "cache write skipped", "", nil)

	parseActivity := hotReloadActivity()
	if len(parseActivity) != 6 {
		parseT.Fatalf("expected hot reload activity to retain only the last six entries, got %+v", parseActivity)
	}
	if parseActivity[0].Message != "route loader started" {
		parseT.Fatalf("expected earliest included entry to be trimmed, got first message %q", parseActivity[0].Message)
	}
	if parseActivity[len(parseActivity)-1].Message != "navigation cache refreshed" {
		parseT.Fatalf("expected latest navigation entry to be preserved, got %q", parseActivity[len(parseActivity)-1].Message)
	}
	if parseActivity[len(parseActivity)-1].Domain != "router" {
		parseT.Fatalf("expected router navigation entry, got %+v", parseActivity[len(parseActivity)-1])
	}
}

// TestBridgeRestoreSnapshotWasmReportsEmptyAndErrorResponses verifies restoreSnapshot exports skipped-empty and error metadata through the bridge API.
func TestBridgeRestoreSnapshotWasmReportsEmptyAndErrorResponses(parseT *testing.T) {
	Disable()
	runtimepkg.InitGlobalRuntime(runtimepkg.Config{Scheduler: noOpScheduler{}})
	runtimepkg.ClearDiagnostics()
	runtimepkg.ClearLogs()
	parseT.Cleanup(func() {
		Disable()
		runtimepkg.ClearDiagnostics()
		runtimepkg.ClearLogs()
	})

	Configure(Config{ResetKey: "bridge-v1"})
	parseBridge, parseErr := appBridge()
	if parseErr != nil {
		parseT.Fatalf("expected installed bridge, got %v", parseErr)
	}

	parseRestoreFn := parseBridge.Get("restoreSnapshot")
	parseEmptyResponse, parseErr := parseRestoreFn.Invoke()
	if parseErr != nil {
		parseT.Fatalf("expected empty restore invocation to succeed, got %v", parseErr)
	}
	parseEmptyValue, parseErr := parseEmptyResponse.ToGo()
	if parseErr != nil {
		parseT.Fatalf("expected empty restore payload to decode, got %v", parseErr)
	}
	parseEmptyMap, parseOk := parseEmptyValue.(map[string]any)
	if !parseOk || parseEmptyMap["outcome"] != "skipped-empty" {
		parseT.Fatalf("expected skipped-empty bridge response, got %#v", parseEmptyValue)
	}

	parseInvalidResponse, parseErr := parseRestoreFn.Invoke("{")
	if parseErr != nil {
		parseT.Fatalf("expected malformed restore invocation to return structured error payload, got %v", parseErr)
	}
	parseInvalidValue, parseErr := parseInvalidResponse.ToGo()
	if parseErr != nil {
		parseT.Fatalf("expected invalid restore payload to decode, got %v", parseErr)
	}
	parseInvalidMap, parseOk := parseInvalidValue.(map[string]any)
	if !parseOk || parseInvalidMap["outcome"] != "error" {
		parseT.Fatalf("expected error bridge response, got %#v", parseInvalidValue)
	}
	if _, parseFound := parseInvalidMap["error"]; !parseFound {
		parseT.Fatalf("expected bridge error payload to include string error detail, got %#v", parseInvalidValue)
	}

	runtimepkg.ReportDiagnostic("runtime", runtimepkg.DiagnosticWarning, "hot reload fell back to remount for App")
	parseRestoredResponse, parseErr := parseRestoreFn.Invoke(`{"bridge-theme":"dark"}`)
	if parseErr != nil {
		parseT.Fatalf("expected restored bridge response to succeed, got %v", parseErr)
	}
	parseRestoredValue, parseErr := parseRestoredResponse.ToGo()
	if parseErr != nil {
		parseT.Fatalf("expected restored bridge payload to decode, got %v", parseErr)
	}
	parseRestoredMap, parseOk := parseRestoredValue.(map[string]any)
	if !parseOk || parseRestoredMap["outcome"] != "restored-legacy" {
		parseT.Fatalf("expected restored-legacy bridge response, got %#v", parseRestoredValue)
	}
	parseDiagnostics, parseFound := parseRestoredMap["diagnostics"]
	if !parseFound {
		parseT.Fatalf("expected restored bridge response to include hot reload diagnostics, got %#v", parseRestoredValue)
	}
	parseDiagnosticList, parseOk := parseDiagnostics.([]any)
	if !parseOk || len(parseDiagnosticList) != 1 {
		parseT.Fatalf("expected one hot reload diagnostic in bridge response, got %#v", parseDiagnostics)
	}
}

// TestConfigEqualWasmComparesResetKeysAndAtomIDs verifies the config comparison helper covers reset-key, length, and element differences.
func TestConfigEqualWasmComparesResetKeysAndAtomIDs(parseT *testing.T) {
	if !configEqual(Config{ResetKey: "build-v1", AtomIDs: []string{"theme"}}, Config{ResetKey: "build-v1", AtomIDs: []string{"theme"}}) {
		parseT.Fatal("expected identical config values to compare equal")
	}
	if configEqual(Config{ResetKey: "build-v1"}, Config{ResetKey: "build-v2"}) {
		parseT.Fatal("expected differing reset keys to compare unequal")
	}
	if configEqual(Config{SnapshotVersion: 1}, Config{SnapshotVersion: 2}) {
		parseT.Fatal("expected differing snapshot versions to compare unequal")
	}
	if configEqual(Config{AtomIDs: []string{"theme"}}, Config{AtomIDs: []string{"theme", "sidebar"}}) {
		parseT.Fatal("expected differing atom-id lengths to compare unequal")
	}
	if configEqual(Config{AtomIDs: []string{"theme", "sidebar"}}, Config{AtomIDs: []string{"theme", "layout"}}) {
		parseT.Fatal("expected differing atom-id values to compare unequal")
	}
}

// TestRestorePendingSnapshotWasmSkipsBlankInvalidAndOptionalClearPaths verifies stored-state restore guards around empty payloads, malformed payloads, and missing clear handlers.
func TestRestorePendingSnapshotWasmSkipsBlankInvalidAndOptionalClearPaths(parseT *testing.T) {
	Disable()
	runtimepkg.InitGlobalRuntime(runtimepkg.Config{Scheduler: noOpScheduler{}})
	runtimepkg.ClearDiagnostics()
	runtimepkg.ClearLogs()
	parseT.Cleanup(func() {
		Disable()
		runtimepkg.ClearDiagnostics()
		runtimepkg.ClearLogs()
	})

	if parseErr := runtimepkg.GetGlobalRuntime().SetAtomValue("restore-theme", "light"); parseErr != nil {
		parseT.Fatalf("unexpected setup error: %v", parseErr)
	}

	parseRestoreLiveReload := storeHotReloadTestGlobal(parseT, liveReloadGlobal, js.Global().Get("Object").New())
	restorePendingSnapshot()
	parseRestoreLiveReload()

	parseBlankClearCount := 0
	parseBlankLiveReload, parseBlankCleanup := buildHotReloadTestLiveReload("   ", &parseBlankClearCount)
	defer parseBlankCleanup()
	parseRestoreLiveReload = storeHotReloadTestGlobal(parseT, liveReloadGlobal, parseBlankLiveReload)
	restorePendingSnapshot()
	parseRestoreLiveReload()
	if parseBlankClearCount != 0 {
		parseT.Fatalf("expected blank stored payload to skip clearStoredState, got %d clears", parseBlankClearCount)
	}

	parseInvalidClearCount := 0
	parseInvalidLiveReload, parseInvalidCleanup := buildHotReloadTestLiveReload("{", &parseInvalidClearCount)
	defer parseInvalidCleanup()
	parseRestoreLiveReload = storeHotReloadTestGlobal(parseT, liveReloadGlobal, parseInvalidLiveReload)
	restorePendingSnapshot()
	parseRestoreLiveReload()
	if parseInvalidClearCount != 0 {
		parseT.Fatalf("expected malformed stored payload to skip clearStoredState, got %d clears", parseInvalidClearCount)
	}
	if parseValue, parseOk := runtimepkg.GetGlobalRuntime().GetAtomValue("restore-theme"); !parseOk || parseValue != "light" {
		parseT.Fatalf("expected malformed payload to leave atom state unchanged, got %#v (ok=%v)", parseValue, parseOk)
	}

	parseNoClearLiveReload, parseNoClearCleanup := buildHotReloadTestLiveReload(`{"restore-theme":"dark"}`, nil)
	defer parseNoClearCleanup()
	parseNoClearLiveReload.Set("clearStoredState", js.Undefined())
	parseRestoreLiveReload = storeHotReloadTestGlobal(parseT, liveReloadGlobal, parseNoClearLiveReload)
	restorePendingSnapshot()
	parseRestoreLiveReload()
	if parseValue, parseOk := runtimepkg.GetGlobalRuntime().GetAtomValue("restore-theme"); !parseOk || parseValue != "dark" {
		parseT.Fatalf("expected valid stored payload to restore even without clear handler, got %#v (ok=%v)", parseValue, parseOk)
	}
}

// TestAppBridgeWasmReportsMissingBridge verifies the helper returns a descriptive error when the bridge has not been installed.
func TestAppBridgeWasmReportsMissingBridge(parseT *testing.T) {
	Disable()
	if _, parseErr := appBridge(); parseErr == nil {
		parseT.Fatal("expected appBridge to fail when the bridge is not installed")
	}
}
