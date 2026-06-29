//go:build !js || !wasm

package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/interop"
)

// stateNativeNoOpScheduler keeps native state tests deterministic without scheduling background work.
type stateNativeNoOpScheduler struct{}

// RequestIdleCallback ignores native idle callback scheduling in state tests.
func (stateNativeNoOpScheduler) RequestIdleCallback(parseCallback func(runtime.Deadline)) {}

// SetTimeout ignores native timeout scheduling in state tests.
func (stateNativeNoOpScheduler) SetTimeout(parseCallback func(), parseDelay int) {}

// setStateTestStructField writes one unexported interop field so state tests can build lightweight native doubles.
func setStateTestStructField(parseT *testing.T, parseTarget any, parseField string, parseValue any) {
	parseT.Helper()
	parseStructValue := reflect.ValueOf(parseTarget).Elem()
	parseFieldValue := parseStructValue.FieldByName(parseField)
	if !parseFieldValue.IsValid() {
		parseT.Fatalf("missing field %q on %T", parseField, parseTarget)
	}
	reflect.NewAt(parseFieldValue.Type(), unsafe.Pointer(parseFieldValue.UnsafeAddr())).Elem().Set(reflect.ValueOf(parseValue))
}

// installStateNativeHookContext resets the global runtime and installs one active fiber for hook-based state tests.
func installStateNativeHookContext(parseT *testing.T) {
	parseT.Helper()
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: stateNativeNoOpScheduler{}, Reset: true})
	runtime.SetCurrentFiber(&runtime.Fiber{})
	parseT.Cleanup(func() {
		runtime.SetCurrentFiber(nil)
	})
}

// buildStateTestStorage returns one in-memory interop.Storage double for native snapshot tests.
func buildStateTestStorage(parseT *testing.T) (interop.Storage, map[string]string) {
	parseT.Helper()
	parseStorage := interop.Storage{}
	parseData := map[string]string{}
	setStateTestStructField(parseT, &parseStorage, "getItem", func(parseKey string) (string, bool, error) {
		parseValue, parseOk := parseData[parseKey]
		return parseValue, parseOk, nil
	})
	setStateTestStructField(parseT, &parseStorage, "setItem", func(parseKey string, parseValue string) error {
		parseData[parseKey] = parseValue
		return nil
	})
	setStateTestStructField(parseT, &parseStorage, "removeItem", func(parseKey string) error {
		delete(parseData, parseKey)
		return nil
	})
	return parseStorage, parseData
}

// buildStateTestPersistentStore returns one in-memory interop.PersistentStore double for native persistent snapshot tests.
func buildStateTestPersistentStore(parseT *testing.T) (interop.PersistentStore, map[string]string) {
	parseT.Helper()
	parseStore := interop.PersistentStore{}
	parseData := map[string]string{}
	setStateTestStructField(parseT, &parseStore, "backend", func() string { return "memory" })
	setStateTestStructField(parseT, &parseStore, "getItem", func(parseCtx context.Context, parseKey string) (string, bool, error) {
		_ = parseCtx
		parseValue, parseOk := parseData[parseKey]
		return parseValue, parseOk, nil
	})
	setStateTestStructField(parseT, &parseStore, "setItem", func(parseCtx context.Context, parseKey string, parseValue string) error {
		_ = parseCtx
		parseData[parseKey] = parseValue
		return nil
	})
	return parseStore, parseData
}

// TestStateNativeAtomAndDerivedHelpers covers the shared hook wrappers and reactive text helpers on native builds.
func TestStateNativeAtomAndDerivedHelpers(parseT *testing.T) {
	installStateNativeHookContext(parseT)

	parseCount := UseAtom("state-native-count", 2)
	parseCount.Set(3)
	parseCount.Update(func(parsePrev int) int { return parsePrev + 1 })
	if parseCount.Get() != 4 {
		parseT.Fatalf("expected atom wrapper to update shared value to 4, got %d", parseCount.Get())
	}
	if parseIDs := parseCount.ReactiveRegionSourceIDs(); len(parseIDs) != 1 || parseIDs[0] != "state-native-count" {
		parseT.Fatalf("expected atom reactive source ids to include the atom id, got %v", parseIDs)
	}

	parseComputed := UseComputed(func() int {
		return parseCount.Get() * 2
	}, parseCount.Get())
	if parseComputed.Get() != 8 {
		parseT.Fatalf("expected computed wrapper to return 8, got %d", parseComputed.Get())
	}

	parseDerived := UseDerived("state-native-double", func() int {
		return parseCount.Get() * 2
	}, "state-native-count")
	if parseDerived.Get() != 8 {
		parseT.Fatalf("expected derived wrapper to return 8, got %d", parseDerived.Get())
	}
	parseCount.Set(5)
	if parseDerived.Get() != 10 {
		parseT.Fatalf("expected derived wrapper to observe latest atom value, got %d", parseDerived.Get())
	}
	if parseIDs2 := parseDerived.ReactiveRegionSourceIDs(); len(parseIDs2) != 1 || parseIDs2[0] != "state-native-double" {
		parseT.Fatalf("expected derived reactive source ids to include the derived id, got %v", parseIDs2)
	}

	parseSelector := Select("state-native-parity", parseCount, func(parseValue int) string {
		if parseValue%2 == 0 {
			return "even"
		}
		return "odd"
	})
	if parseSelector.Get() != "odd" {
		parseT.Fatalf("expected selector wrapper to project odd, got %q", parseSelector.Get())
	}
	if parseSelector.id == "" || !strings.Contains(parseSelector.id, "state-native-parity") || !strings.Contains(parseSelector.id, "state-native-count") {
		parseT.Fatalf("expected selector id to include requested and source ids, got %q", parseSelector.id)
	}

	parseAtomText := parseCount.Text(func(parseValue int) string { return fmt.Sprintf("count:%d", parseValue) })
	parseAtomGetter, _ := parseAtomText.Props[runtimeReactiveTextGetterProp()].(func() string)
	if parseAtomGetter == nil || parseAtomGetter() != "count:5" {
		parseT.Fatalf("expected atom text getter to render latest atom value, got nil=%t", parseAtomGetter == nil)
	}
	parseDerivedText := parseSelector.Text(func(parseValue string) string { return "parity:" + parseValue })
	parseDerivedGetter, _ := parseDerivedText.Props[runtimeReactiveTextGetterProp()].(func() string)
	if parseDerivedGetter == nil || parseDerivedGetter() != "parity:odd" {
		parseT.Fatalf("expected derived text getter to render projected value, got nil=%t", parseDerivedGetter == nil)
	}

	if UseSelector[string, string]("state-native-empty-selector", nil, func(parseValue string) string { return parseValue }).Get() != "" {
		parseT.Fatal("expected selector with nil source to return the zero value")
	}
	if UseSelector[int, string]("state-native-empty-project", parseCount, (func(int) string)(nil)).Get() != "" {
		parseT.Fatal("expected selector with nil projector to return the zero value")
	}

	var parseZeroComputed Computed[string]
	if parseZeroComputed.Get() != "" {
		parseT.Fatalf("expected zero-value computed handle to return empty string, got %q", parseZeroComputed.Get())
	}
	var parseZeroDerived Derived[string]
	if parseZeroDerived.Get() != "" {
		parseT.Fatalf("expected zero-value derived handle to return empty string, got %q", parseZeroDerived.Get())
	}
}

// TestStateNativeSnapshotHelpers covers shared in-memory snapshot selection, import, and JSON normalization.
func TestStateNativeSnapshotHelpers(parseT *testing.T) {
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: stateNativeNoOpScheduler{}, Reset: true})
	if parseErr := runtime.GetGlobalRuntime().SetAtomValue("state-native-theme", "dark"); parseErr != nil {
		parseT.Fatalf("expected atom seed to succeed, got %v", parseErr)
	}
	if parseErr2 := runtime.GetGlobalRuntime().SetAtomValue("state-native-count", 3); parseErr2 != nil {
		parseT.Fatalf("expected second atom seed to succeed, got %v", parseErr2)
	}

	parseSnapshot, parseErr := GetSnapshot()
	if parseErr != nil {
		parseT.Fatalf("expected GetSnapshot to succeed, got %v", parseErr)
	}
	if parseSnapshot["state-native-theme"] != "dark" || parseSnapshot["state-native-count"] != 3 {
		parseT.Fatalf("unexpected snapshot contents: %#v", parseSnapshot)
	}
	parseSelected := parseSnapshot.Select("state-native-theme")
	if len(parseSelected) != 1 || parseSelected["state-native-theme"] != "dark" {
		parseT.Fatalf("expected Select to keep only the requested key, got %#v", parseSelected)
	}
	parseClone := parseSnapshot.Select()
	if len(parseClone) != len(parseSnapshot) {
		parseT.Fatalf("expected zero-arg Select to clone the snapshot, got %#v", parseClone)
	}

	parseExported, parseErr2 := ExportSnapshot()
	if parseErr2 != nil || parseExported["state-native-theme"] != "dark" {
		parseT.Fatalf("expected ExportSnapshot to preserve atom values, snapshot=%#v err=%v", parseExported, parseErr2)
	}

	if parseErr3 := ApplySnapshot(Snapshot{"state-native-theme": "light"}); parseErr3 != nil {
		parseT.Fatalf("expected ApplySnapshot to succeed, got %v", parseErr3)
	}
	if parseErr4 := ImportSnapshot(Snapshot{"state-native-count": 7}); parseErr4 != nil {
		parseT.Fatalf("expected ImportSnapshot to succeed, got %v", parseErr4)
	}
	if parseValue, parseOk := runtime.GetGlobalRuntime().GetAtomValue("state-native-theme"); !parseOk || parseValue != "light" {
		parseT.Fatalf("expected ApplySnapshot to update theme atom, got %#v ok=%t", parseValue, parseOk)
	}
	if parseValue2, parseOk2 := runtime.GetGlobalRuntime().GetAtomValue("state-native-count"); !parseOk2 || parseValue2 != 7 {
		parseT.Fatalf("expected ImportSnapshot to update count atom, got %#v ok=%t", parseValue2, parseOk2)
	}

	parseData, parseErr5 := MarshalSnapshotJSON(nil)
	if parseErr5 != nil {
		parseT.Fatalf("expected nil snapshot JSON to encode, got %v", parseErr5)
	}
	var parseEnvelope snapshotWireEnvelope
	if parseErrEnvelope := json.Unmarshal(parseData, &parseEnvelope); parseErrEnvelope != nil {
		parseT.Fatalf("expected nil snapshot JSON to decode as versioned envelope, got %v", parseErrEnvelope)
	}
	if parseEnvelope.Protocol != snapshotWireProtocol || parseEnvelope.Version != currentSnapshotVersion || len(parseEnvelope.State) != 0 {
		parseT.Fatalf("expected nil snapshot JSON to encode as empty versioned envelope, got %q", string(parseData))
	}
	if parseEmpty, parseErr6 := UnmarshalSnapshotJSON(nil); parseErr6 != nil || len(parseEmpty) != 0 {
		parseT.Fatalf("expected empty snapshot JSON to decode to an empty snapshot, snapshot=%#v err=%v", parseEmpty, parseErr6)
	}
	parseDecoded, parseErr7 := UnmarshalSnapshotJSON([]byte(`{"count":1,"ratio":1.5,"nested":{"items":[2,2.5]}}`))
	if parseErr7 != nil {
		parseT.Fatalf("expected UnmarshalSnapshotJSON to succeed, got %v", parseErr7)
	}
	if parseDecoded["count"] != 1 || parseDecoded["ratio"] != 1.5 {
		parseT.Fatalf("expected snapshot normalization to preserve whole and fractional numbers, got %#v", parseDecoded)
	}
	parseNested, _ := parseDecoded["nested"].(map[string]any)
	parseItems, _ := parseNested["items"].([]any)
	if len(parseItems) != 2 || parseItems[0] != 2 || parseItems[1] != 2.5 {
		parseT.Fatalf("expected nested snapshot normalization to preserve slice values, got %#v", parseItems)
	}
	if _, parseErr8 := UnmarshalSnapshotJSON([]byte(`{`)); parseErr8 == nil {
		parseT.Fatal("expected invalid JSON to return an error")
	}
	parseVersioned, parseErr9 := UnmarshalSnapshotJSON([]byte(`{"protocol":"gwc.state.snapshot","version":1,"state":{"theme":"dark"}}`))
	if parseErr9 != nil || parseVersioned["theme"] != "dark" {
		parseT.Fatalf("expected versioned snapshot envelope to decode, snapshot=%#v err=%v", parseVersioned, parseErr9)
	}
	if _, parseErr10 := UnmarshalSnapshotJSON([]byte(`{"protocol":"gwc.state.snapshot","version":99,"state":{"theme":"dark"}}`)); parseErr10 == nil {
		parseT.Fatal("expected future snapshot version to be rejected")
	}
	if _, parseErr11 := UnmarshalSnapshotJSON([]byte(`{"protocol":"other.snapshot","version":1,"state":{"theme":"dark"}}`)); parseErr11 == nil {
		parseT.Fatal("expected unknown snapshot protocol to be rejected")
	}
}

func TestStateNativeSnapshotSchemaMigration(parseT *testing.T) {
	parsePrev := snapshotMigrations
	snapshotMigrations = map[int]SnapshotMigration{}
	parseT.Cleanup(func() {
		snapshotMigrations = parsePrev
	})

	if parseErr := RegisterSnapshotMigration(0, func(parseSnapshot Snapshot) (Snapshot, error) {
		parseNext := parseSnapshot.Select()
		parseNext["theme"] = parseNext["old_theme"]
		delete(parseNext, "old_theme")
		parseNext["migrated"] = true
		return parseNext, nil
	}); parseErr != nil {
		parseT.Fatalf("RegisterSnapshotMigration returned error: %v", parseErr)
	}

	parseMigrated, parseErr2 := UnmarshalSnapshotJSON([]byte(`{"protocol":"gwc.state.snapshot","version":0,"state":{"old_theme":"dark"}}`))
	if parseErr2 != nil {
		parseT.Fatalf("expected v0 snapshot migration to succeed, got %v", parseErr2)
	}
	if parseMigrated["theme"] != "dark" || parseMigrated["migrated"] != true {
		parseT.Fatalf("unexpected migrated snapshot: %#v", parseMigrated)
	}

	if parseErr3 := RegisterSnapshotMigration(0, nil); parseErr3 != nil {
		parseT.Fatalf("expected nil migration to unregister, got %v", parseErr3)
	}
	parseLegacy, parseErr4 := UnmarshalSnapshotJSON([]byte(`{"protocol":"gwc.state.snapshot","state":{"theme":"legacy"}}`))
	if parseErr4 != nil || parseLegacy["theme"] != "legacy" {
		parseT.Fatalf("expected missing-version legacy envelope to restore without migration, snapshot=%#v err=%v", parseLegacy, parseErr4)
	}
}

func TestStateNativeSnapshotMigrationFailureIsAtomic(parseT *testing.T) {
	parsePrev := snapshotMigrations
	snapshotMigrations = map[int]SnapshotMigration{}
	parseT.Cleanup(func() {
		snapshotMigrations = parsePrev
	})
	if parseErr := RegisterSnapshotMigration(0, func(Snapshot) (Snapshot, error) {
		return nil, errors.New("boom")
	}); parseErr != nil {
		parseT.Fatalf("RegisterSnapshotMigration returned error: %v", parseErr)
	}

	runtime.InitGlobalRuntime(runtime.Config{Scheduler: stateNativeNoOpScheduler{}, Reset: true})
	if parseErr := ApplySnapshot(Snapshot{"state-native-migration-atomic": "before"}); parseErr != nil {
		parseT.Fatalf("seed ApplySnapshot returned error: %v", parseErr)
	}
	parseSnapshot, parseErr2 := UnmarshalSnapshotJSON([]byte(`{"protocol":"gwc.state.snapshot","version":0,"state":{"state-native-migration-atomic":"after"}}`))
	if parseErr2 == nil || parseSnapshot != nil {
		parseT.Fatalf("expected migration failure to return no snapshot, snapshot=%#v err=%v", parseSnapshot, parseErr2)
	}
	if parseValue, parseOk := runtime.GetGlobalRuntime().GetAtomValue("state-native-migration-atomic"); !parseOk || parseValue != "before" {
		parseT.Fatalf("expected failed migration not to mutate atom state, got %#v ok=%t", parseValue, parseOk)
	}
}

// TestStateNativeSnapshotStorageHelpers covers browser-storage helpers through injected native storage doubles.
func TestStateNativeSnapshotStorageHelpers(parseT *testing.T) {
	parseStorage, parseData := buildStateTestStorage(parseT)
	parsePrevLocal := loadStateLocalStorage
	parsePrevSession := loadStateSessionStorage
	loadStateLocalStorage = func() (interop.Storage, error) { return parseStorage, nil }
	loadStateSessionStorage = func() (interop.Storage, error) { return interop.Storage{}, errors.New("session unavailable") }
	parseT.Cleanup(func() {
		loadStateLocalStorage = parsePrevLocal
		loadStateSessionStorage = parsePrevSession
	})
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: stateNativeNoOpScheduler{}, Reset: true})

	parseSnapshot := Snapshot{"state-native-user": "alice", "state-native-ready": true}
	if parseErr := SaveSnapshot("state-native-app", parseSnapshot, LocalStorage); parseErr != nil {
		parseT.Fatalf("expected SaveSnapshot to succeed with injected storage, got %v", parseErr)
	}
	if _, parseOk := parseData["state-native-app"]; !parseOk {
		parseT.Fatalf("expected SaveSnapshot to write into injected storage, data=%#v", parseData)
	}

	parseLoaded, parseOk, parseErr2 := LoadSnapshot("state-native-app", LocalStorage)
	if parseErr2 != nil || !parseOk {
		parseT.Fatalf("expected LoadSnapshot to succeed, snapshot=%#v ok=%t err=%v", parseLoaded, parseOk, parseErr2)
	}
	if parseLoaded["state-native-user"] != "alice" || parseLoaded["state-native-ready"] != true {
		parseT.Fatalf("unexpected loaded snapshot: %#v", parseLoaded)
	}
	if parseRestored, parseErr3 := RestoreSnapshot("state-native-app", LocalStorage); parseErr3 != nil || !parseRestored {
		parseT.Fatalf("expected RestoreSnapshot to succeed, restored=%t err=%v", parseRestored, parseErr3)
	}
	if parseValue, parseOk2 := runtime.GetGlobalRuntime().GetAtomValue("state-native-user"); !parseOk2 || parseValue != "alice" {
		parseT.Fatalf("expected RestoreSnapshot to import the stored atom, got %#v ok=%t", parseValue, parseOk2)
	}
	if _, parseMissing, parseErr4 := LoadSnapshot("state-native-missing", LocalStorage); parseErr4 != nil || parseMissing {
		parseT.Fatalf("expected missing storage key to miss cleanly, found=%t err=%v", parseMissing, parseErr4)
	}

	if parseErr5 := SaveSnapshot("state-native-bad", Snapshot{"bad": math.Inf(1)}, LocalStorage); parseErr5 == nil {
		parseT.Fatal("expected SaveSnapshot to surface JSON encoding failures")
	}
	if _, _, parseErr6 := LoadSnapshot("state-native-app", StorageArea("cookieStorage")); parseErr6 == nil {
		parseT.Fatal("expected LoadSnapshot to reject unsupported storage areas")
	}
	if parseRestored2, parseErr7 := RestoreSnapshot("state-native-app", SessionStorage); parseErr7 == nil || parseRestored2 {
		parseT.Fatalf("expected unavailable session storage to fail restore, restored=%t err=%v", parseRestored2, parseErr7)
	}
}

func TestStateNativeSnapshotMigrationAndAtomicRestore(parseT *testing.T) {
	parseStorage, parseData := buildStateTestStorage(parseT)
	parsePrevLocal := loadStateLocalStorage
	loadStateLocalStorage = func() (interop.Storage, error) { return parseStorage, nil }
	parseT.Cleanup(func() {
		loadStateLocalStorage = parsePrevLocal
		_ = RegisterSnapshotMigration(0, nil)
	})
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: stateNativeNoOpScheduler{}, Reset: true})
	if parseErr := runtime.GetGlobalRuntime().SetAtomValue("state-native-migrated", "current"); parseErr != nil {
		parseT.Fatalf("seed current atom: %v", parseErr)
	}

	if parseErr := RegisterSnapshotMigration(0, func(parseSnapshot Snapshot) (Snapshot, error) {
		parseNext := parseSnapshot.Select()
		parseNext["state-native-migrated"] = "upgraded:" + fmt.Sprint(parseNext["legacy"])
		delete(parseNext, "legacy")
		return parseNext, nil
	}); parseErr != nil {
		parseT.Fatalf("RegisterSnapshotMigration: %v", parseErr)
	}
	parseData["state-native-legacy"] = `{"protocol":"gwc.state.snapshot","version":0,"state":{"legacy":"v0"}}`
	parseRestored, parseErr := RestoreSnapshot("state-native-legacy", LocalStorage)
	if parseErr != nil || !parseRestored {
		parseT.Fatalf("RestoreSnapshot migrated legacy payload = %t, %v", parseRestored, parseErr)
	}
	if parseValue, parseOk := runtime.GetGlobalRuntime().GetAtomValue("state-native-migrated"); !parseOk || parseValue != "upgraded:v0" {
		parseT.Fatalf("expected migrated atom value, got %#v ok=%t", parseValue, parseOk)
	}

	if parseErr := RegisterSnapshotMigration(0, func(Snapshot) (Snapshot, error) {
		return nil, errors.New("migration failed")
	}); parseErr != nil {
		parseT.Fatalf("RegisterSnapshotMigration failing hook: %v", parseErr)
	}
	parseData["state-native-failing-migration"] = `{"protocol":"gwc.state.snapshot","version":0,"state":{"state-native-migrated":"bad"}}`
	parseRestored, parseErr = RestoreSnapshot("state-native-failing-migration", LocalStorage)
	if parseErr == nil || parseRestored {
		parseT.Fatalf("expected failed migration to abort restore, restored=%t err=%v", parseRestored, parseErr)
	}
	if parseValue, parseOk := runtime.GetGlobalRuntime().GetAtomValue("state-native-migrated"); !parseOk || parseValue != "upgraded:v0" {
		parseT.Fatalf("expected failed migration to leave atom unchanged, got %#v ok=%t", parseValue, parseOk)
	}
}

// TestStateNativePersistentSnapshotHelpers covers persistent snapshot helpers with one injected in-memory store.
func TestStateNativePersistentSnapshotHelpers(parseT *testing.T) {
	parseStore, parseData := buildStateTestPersistentStore(parseT)
	parseOptions := PersistentSnapshotOptions{
		StoreResolver: func(parseCtx context.Context) (interop.PersistentStore, error) {
			if parseCtx == nil {
				parseT.Fatal("expected persistent resolver context to be normalized")
			}
			return parseStore, nil
		},
	}
	runtime.InitGlobalRuntime(runtime.Config{Scheduler: stateNativeNoOpScheduler{}, Reset: true})

	parseSnapshot := Snapshot{"state-native-project": "atlas", "state-native-ready": true}
	if parseErr := SavePersistentSnapshot(nil, "state-native-persistent", parseSnapshot, parseOptions); parseErr != nil { //nolint:staticcheck // nil-context normalization is the case under test
		parseT.Fatalf("expected SavePersistentSnapshot to succeed, got %v", parseErr)
	}
	if _, parseOk := parseData["state-native-persistent"]; !parseOk {
		parseT.Fatalf("expected persistent save to write into the injected store, data=%#v", parseData)
	}

	parseLoaded, parseOk, parseErr2 := LoadPersistentSnapshot(nil, "state-native-persistent", parseOptions) //nolint:staticcheck // nil-context normalization is the case under test
	if parseErr2 != nil || !parseOk {
		parseT.Fatalf("expected LoadPersistentSnapshot to succeed, snapshot=%#v ok=%t err=%v", parseLoaded, parseOk, parseErr2)
	}
	if parseLoaded["state-native-project"] != "atlas" || parseLoaded["state-native-ready"] != true {
		parseT.Fatalf("unexpected persistent snapshot contents: %#v", parseLoaded)
	}
	if parseRestored, parseErr3 := RestorePersistentSnapshot(nil, "state-native-persistent", parseOptions); parseErr3 != nil || !parseRestored { //nolint:staticcheck // nil-context normalization is the case under test
		parseT.Fatalf("expected RestorePersistentSnapshot to succeed, restored=%t err=%v", parseRestored, parseErr3)
	}
	if parseValue, parseOk2 := runtime.GetGlobalRuntime().GetAtomValue("state-native-project"); !parseOk2 || parseValue != "atlas" {
		parseT.Fatalf("expected persistent restore to import the stored atom, got %#v ok=%t", parseValue, parseOk2)
	}

	if parseResolved := resolvePersistentSnapshotOptions(nil); parseResolved.DatabaseName != "" || parseResolved.StoreName != "" || parseResolved.DeleteOnCorruption || parseResolved.FallbackResolver != nil || parseResolved.FallbackBackend != "" || parseResolved.StoreResolver != nil {
		parseT.Fatalf("expected zero options to resolve to the zero value, got %#v", parseResolved)
	}
	if parseResolved2 := resolvePersistentSnapshotContext(nil); parseResolved2 == nil { //nolint:staticcheck // nil-context normalization is the case under test
		parseT.Fatal("expected resolvePersistentSnapshotContext to replace nil with context.Background")
	}

	parseFallbackStorage, _ := buildStateTestStorage(parseT)
	parseStore2, parseErr4 := openPersistentSnapshotStore(nil, []PersistentSnapshotOptions{{ //nolint:staticcheck // nil-context normalization is the case under test
		StoreResolver: nil,
		FallbackResolver: func() (interop.Storage, error) {
			return parseFallbackStorage, nil
		},
		FallbackBackend: "memory",
		StoreName:       "native-state",
		DatabaseName:    "native-state-db",
	}})
	if parseErr4 == nil || !strings.Contains(parseErr4.Error(), "unavailable") || parseStore2.Backend() != "" {
		parseT.Fatalf("expected native openPersistentSnapshotStore fallback path to surface unavailability, store=%#v err=%v", parseStore2, parseErr4)
	}

	_, parseErr5 := openPersistentSnapshotStore(nil, []PersistentSnapshotOptions{{ //nolint:staticcheck // nil-context normalization is the case under test
		StoreResolver: func(context.Context) (interop.PersistentStore, error) {
			return interop.PersistentStore{}, errors.New("open failed")
		},
	}})
	if parseErr5 == nil || !strings.Contains(parseErr5.Error(), "open failed") {
		parseT.Fatalf("expected store resolver errors to surface, got %v", parseErr5)
	}
}

// TestStateNativeUnmarshalSnapshotJSONTypeAssertion is a regression test for #42.
// It verifies that UnmarshalSnapshotJSON does not panic on the normal path and
// returns a properly typed Snapshot (not a bare map[string]interface{}).
func TestStateNativeUnmarshalSnapshotJSONTypeAssertion(parseT *testing.T) {
	// Normal round-trip: marshal a Snapshot then unmarshal it — must not panic.
	parseOriginal := Snapshot{"alpha": "hello", "beta": true, "gamma": 42}
	parseData, parseErr := MarshalSnapshotJSON(parseOriginal)
	if parseErr != nil {
		parseT.Fatalf("expected MarshalSnapshotJSON to succeed, got %v", parseErr)
	}
	parseResult, parseErr2 := UnmarshalSnapshotJSON(parseData)
	if parseErr2 != nil {
		parseT.Fatalf("expected UnmarshalSnapshotJSON to succeed, got %v", parseErr2)
	}
	if parseResult["alpha"] != "hello" || parseResult["beta"] != true || parseResult["gamma"] != 42 {
		parseT.Fatalf("expected round-tripped snapshot to preserve values, got %#v", parseResult)
	}

	// Verify the returned value is usable as a Snapshot (Select, ApplySnapshot, etc.).
	parseSelected := parseResult.Select("alpha")
	if len(parseSelected) != 1 || parseSelected["alpha"] != "hello" {
		parseT.Fatalf("expected Select on round-tripped snapshot to work, got %#v", parseSelected)
	}

	// Deeply nested JSON must also not panic.
	parseNested, parseErr3 := UnmarshalSnapshotJSON([]byte(`{"outer":{"inner":{"deep":99}},"list":[1,2,3]}`))
	if parseErr3 != nil {
		parseT.Fatalf("expected nested UnmarshalSnapshotJSON to succeed, got %v", parseErr3)
	}
	parseOuter, parseOk := parseNested["outer"].(map[string]any)
	if !parseOk {
		parseT.Fatalf("expected outer to be map[string]interface{}, got %T", parseNested["outer"])
	}
	parseInner, parseOk2 := parseOuter["inner"].(map[string]any)
	if !parseOk2 || parseInner["deep"] != 99 {
		parseT.Fatalf("expected deep nested value to be 99, got %#v", parseOuter)
	}
}

// TestStateNativeNormalizeSnapshotFloatRangeGuard is a regression test for #43.
// It verifies that float64 values outside the safe-integer range are not
// converted to int (avoiding precision loss and platform overflow).
func TestStateNativeNormalizeSnapshotFloatRangeGuard(parseT *testing.T) {
	// Values within safe-integer range should be converted to int.
	parseCases := []struct {
		parseInput    float64
		parseWantInt  bool
		parseWantDesc string
	}{
		{0, true, "zero"},
		{1, true, "one"},
		{-1, true, "negative one"},
		{1 << 53, true, "2^53 boundary"},
		{-(1 << 53), true, "-(2^53) boundary"},
		{1.5, false, "fractional"},
		{float64(1<<53) + 2, false, "2^53+2 (above safe boundary)"},
		{-(float64(1<<53) + 2), false, "-(2^53+2) (below safe boundary)"},
		{math.MaxFloat64, false, "MaxFloat64"},
		{-math.MaxFloat64, false, "-MaxFloat64"},
	}
	for _, parseCase := range parseCases {
		parseGot := normalizeSnapshot(parseCase.parseInput)
		if parseCase.parseWantInt {
			if _, parseIsInt := parseGot.(int); !parseIsInt {
				parseT.Errorf("normalizeSnapshot(%v) [%s]: expected int, got %T(%v)", parseCase.parseInput, parseCase.parseWantDesc, parseGot, parseGot)
			}
		} else {
			if _, parseIsFloat := parseGot.(float64); !parseIsFloat {
				parseT.Errorf("normalizeSnapshot(%v) [%s]: expected float64, got %T(%v)", parseCase.parseInput, parseCase.parseWantDesc, parseGot, parseGot)
			}
		}
	}

	// JSON round-trip: a large integer-valued float must survive without truncation.
	parseLarge := float64(1<<53) + 2 // 9007199254740994 — above safe range
	parseJSON := fmt.Appendf(nil, `{"big":%v}`, parseLarge)
	parseSnap, parseErr := UnmarshalSnapshotJSON(parseJSON)
	if parseErr != nil {
		parseT.Fatalf("expected large-float unmarshal to succeed, got %v", parseErr)
	}
	if _, parseIsFloat := parseSnap["big"].(float64); !parseIsFloat {
		parseT.Errorf("expected out-of-safe-range whole float64 to stay as float64, got %T(%v)", parseSnap["big"], parseSnap["big"])
	}
}

func TestStateNativeSnapshotEdgeValues(parseT *testing.T) {
	parseNaN := normalizeSnapshot(math.NaN())
	parseNaNFloat, parseNaNOk := parseNaN.(float64)
	if !parseNaNOk || !math.IsNaN(parseNaNFloat) {
		parseT.Fatalf("normalizeSnapshot(NaN) = %T(%v), want float64 NaN", parseNaN, parseNaN)
	}

	parseInf := normalizeSnapshot(math.Inf(1))
	parseInfFloat, parseInfOk := parseInf.(float64)
	if !parseInfOk || !math.IsInf(parseInfFloat, 1) {
		parseT.Fatalf("normalizeSnapshot(+Inf) = %T(%v), want float64 +Inf", parseInf, parseInf)
	}

	parseSnapshot := Snapshot{"nil-value": nil, "number": 1}
	parseSelected := parseSnapshot.Select("nil-value", "missing")
	if len(parseSelected) != 1 {
		parseT.Fatalf("Select should preserve present nil value only, got %#v", parseSelected)
	}
	if parseValue, parseExists := parseSelected["nil-value"]; !parseExists || parseValue != nil {
		parseT.Fatalf("Select did not preserve nil value, got value=%#v exists=%t", parseValue, parseExists)
	}
	if parseErr := ApplySnapshot(Snapshot{"state-native-nil": nil}); parseErr != nil {
		parseT.Fatalf("ApplySnapshot with nil value returned error: %v", parseErr)
	}
}
