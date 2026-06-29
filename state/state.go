package state

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"strings"
	"sync"

	"github.com/monstercameron/GoWebComponents/v4/deprecation"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/interop"
)

// Element aliases the runtime element type for state package examples and helpers.
type Element = runtime.Element

// Atom exposes shared read/write state keyed by ID.
type Atom[T any] struct {
	id  string
	get func() T
	set func(T)
}

// Computed exposes a memoized derived value local to the current component.
type Computed[T any] struct {
	get func() T
}

// Derived exposes a shared read-only derived atom keyed by ID.
type Derived[T any] struct {
	id  string
	get func() T
}

type selectorSource[T any] interface {
	Get() T
	selectorSourceID() string
}

// Snapshot stores exported atom values by atom ID.
type Snapshot map[string]any

// SnapshotMigration upgrades one snapshot payload between adjacent schema
// versions. Implementations must return a fresh snapshot or an error; errors
// abort the restore before any atom state is changed.
type SnapshotMigration func(Snapshot) (Snapshot, error)

// StorageArea names a browser storage backend.
type StorageArea string

// PersistentSnapshotOptions configures durable IndexedDB-first snapshot storage used by
// SavePersistentSnapshot / LoadPersistentSnapshot. The zero value is valid: it uses the
// default database and store names with no fallback. Set fields only to override defaults.
type PersistentSnapshotOptions struct {
	// DatabaseName is the IndexedDB database to open (default used when empty).
	DatabaseName string
	// StoreName is the object store within the database (default used when empty).
	StoreName string
	// DeleteOnCorruption drops and recreates the store if it fails to open, trading data
	// loss for availability instead of surfacing a hard error.
	DeleteOnCorruption bool
	// FallbackResolver supplies a synchronous Storage when IndexedDB is unavailable.
	FallbackResolver func() (interop.Storage, error)
	// FallbackBackend names the fallback for diagnostics (e.g. "localStorage").
	FallbackBackend string
	// StoreResolver overrides how the durable store is opened; when nil the default
	// IndexedDB resolver is used.
	StoreResolver func(context.Context) (interop.PersistentStore, error)
}

const (
	// LocalStorage stores snapshots in window.localStorage.
	LocalStorage StorageArea = "localStorage"
	// SessionStorage stores snapshots in window.sessionStorage.
	SessionStorage StorageArea = "sessionStorage"
)

const (
	defaultPersistentSnapshotStoreName = "state-snapshots"
	snapshotWireProtocol               = "gwc.state.snapshot"
	currentSnapshotVersion             = 1
)

type snapshotWireEnvelope struct {
	Protocol string   `json:"protocol,omitempty"`
	Version  int      `json:"version,omitempty"`
	State    Snapshot `json:"state,omitempty"`
}

var (
	loadStateLocalStorage   = interop.GetLocalStorage
	loadStateSessionStorage = interop.GetSessionStorage
	snapshotMigrationMu     sync.RWMutex
	snapshotMigrations      = map[int]SnapshotMigration{}
)

// RegisterSnapshotMigration registers one adjacent snapshot schema migration
// from parseFromVersion to parseFromVersion+1. Register migrations at app boot
// before loading stored snapshots. Passing nil removes the migration.
func RegisterSnapshotMigration(parseFromVersion int, parseMigration SnapshotMigration) error {
	if parseFromVersion < 0 || parseFromVersion >= currentSnapshotVersion {
		return fmt.Errorf("state: snapshot migration source version %d is unsupported", parseFromVersion)
	}
	snapshotMigrationMu.Lock()
	defer snapshotMigrationMu.Unlock()
	if parseMigration == nil {
		delete(snapshotMigrations, parseFromVersion)
		return nil
	}
	snapshotMigrations[parseFromVersion] = parseMigration
	return nil
}

// UseAtom provides shared global atoms with subscription-scoped rerenders.
// Atoms are accessible from anywhere in the component tree by ID and
// automatically trigger re-renders in all subscribed components when updated.
//
// The first component to call UseAtom with a specific ID initializes the atom
// with the provided initial value. Subsequent calls from other components will
// use the existing value and subscribe to updates.
//
// Type parameter T can be any Go type. The hook uses generic type parameters
// for type safety.
//
// Returns an [Atom][T] handle with Get, Set, and Update methods.
//
// Example - Theme Management:
//
//	// In a theme-switcher component
//	theme := state.UseAtom("appTheme", "light")
//
//	toggle := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	    if theme.Get() == "light" {
//	        theme.Set("dark")
//	    } else {
//	        theme.Set("light")
//	    }
//	    return nil
//	})
//
//	// In a header component — re-renders automatically when theme changes
//	theme := state.UseAtom("appTheme", "light")
//	bgColor := "white"
//	if theme.Get() == "dark" {
//	    bgColor = "#333"
//	}
//
// Example - User Authentication:
//
//	type User struct {
//	    ID       int
//	    Username string
//	    Email    string
//	}
//
//	user := state.UseAtom("currentUser", User{})
//
//	login := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	    user.Set(User{ID: 1, Username: "johndoe", Email: "john@example.com"})
//	    return nil
//	})
//
//	if user.Get().ID == 0 {
//	    // render login button
//	} else {
//	    // render welcome message using user.Get().Username
//	}
//
// Thread Safety:
// UseAtom is thread-safe and can be safely called from multiple goroutines.
// The internal atom registry uses mutex-based synchronization.
//
// Cleanup:
// When a component unmounts, it is automatically unsubscribed from all atoms
// to prevent memory leaks and unnecessary updates.
//
// Best Practices:
//   - Use descriptive atom IDs (e.g., "currentUser", "appTheme", "shoppingCart")
//   - Initialize atoms with appropriate default values
//   - Use structured types (structs) for complex state
//   - Avoid storing large amounts of data in atoms (use for coordination, not caching)
func UseAtom[T any](parseId string, parseInitialValue T) Atom[T] {
	get, set := runtime.GoUseAtomGlobal(parseId, parseInitialValue)
	return Atom[T]{id: parseId, get: get, set: set}
}

// Get returns the current atom value.
func (parseA Atom[T]) Get() T {
	return parseA.get()
}

// Set replaces the atom value.
func (parseA Atom[T]) Set(parseValue T) {
	parseA.set(parseValue)
}

// Update replaces the atom value using the previous value.
func (parseA Atom[T]) Update(parseFn func(T) T) {
	parseA.set(parseFn(parseA.get()))
}

// UseComputed derives a typed value from other state used by the current component.
//
// It is intended for render-time derived values, especially when a component is
// already reading one or more atoms and wants a typed handle instead of using
// ui.UseMemo directly in every call site.
//
// The computed value is memoized according to the provided dependency list.
// Callers should pass the values that should trigger recomputation.
//
// Deprecated: prefer ui.UseMemo, which returns the value directly (UseComputed only wraps it in a
// Computed handle). For a shared, cross-component derived value use state.UseDerived instead.
func UseComputed[T any](parseCompute func() T, parseDeps ...any) Computed[T] {
	parseValue := runtime.GoUseMemoGlobal(func() any {
		return parseCompute()
	}, parseDeps...)

	parseCast, parseOk := parseValue.(T)
	if !parseOk {
		var parseZero T
		return Computed[T]{get: func() T { return parseZero }}
	}

	return Computed[T]{get: func() T { return parseCast }}
}

// Get returns the current computed value.
func (parseC Computed[T]) Get() T {
	if parseC.get == nil {
		var parseZero T
		return parseZero
	}

	return parseC.get()
}

// UseDerived registers and subscribes to a read-only derived atom.
//
// Derived atoms are shared state values keyed by id. They recompute when one of
// the named source atom IDs changes and expose a typed read-only handle to the
// current derived value. Dependency tracking is explicit through atom IDs so
// recomputation remains predictable and avoids hidden runtime graph discovery.
//
// The id is GLOBAL and shared — this is intentional and is the whole point: two components that
// pass the same id share one derived atom (the same model as UseAtom). That deliberately differs
// from UseSelector, whose id is auto-scoped to the calling component because a selector is a
// component-local projection, not shared state. So: pick a unique, descriptive id for a UseDerived
// you intend to share, exactly as you would for a UseAtom; reach for UseSelector when you want a
// local, collision-free projection instead.
func UseDerived[T any](parseId string, parseCompute func() T, parseDeps ...string) Derived[T] {
	var parseZero T
	if parseErr := runtime.GetGlobalRuntime().RegisterDerivedAtom(parseId, parseDeps, func() any {
		return parseCompute()
	}); parseErr != nil {
		return Derived[T]{id: parseId, get: func() T { return parseZero }}
	}

	parseAtom := UseAtom(parseId, parseZero)
	return Derived[T]{id: parseId, get: parseAtom.Get}
}

// Get returns the current derived value.
func (parseD Derived[T]) Get() T {
	if parseD.get == nil {
		var parseZero T
		return parseZero
	}
	return parseD.get()
}

// selectorSourceID is a core package helper.
func (parseA Atom[T]) selectorSourceID() string {
	return parseA.id
}

// ReactiveRegionSourceIDs is a core package helper.
func (parseA Atom[T]) ReactiveRegionSourceIDs() []string {
	if parseA.id == "" {
		return nil
	}
	return []string{parseA.id}
}

// selectorSourceID is a core package helper.
func (parseD Derived[T]) selectorSourceID() string {
	return parseD.id
}

// ReactiveRegionSourceIDs is a core package helper.
func (parseD Derived[T]) ReactiveRegionSourceIDs() []string {
	if parseD.id == "" {
		return nil
	}
	return []string{parseD.id}
}

// UseSelector creates a read-only projected shared value from an atom or derived source.
//
// The selector remains explicit: callers provide the derived ID to register and the
// source handle to project from. When the projected value is unchanged, subscribers
// are not notified, which makes it suitable for fine-grained hot-value paths.
func UseSelector[T any, U any](parseId string, parseSource selectorSource[T], parseProject func(T) U) Derived[U] {
	var parseZero U
	if parseSource == nil || parseProject == nil {
		return Derived[U]{id: parseId, get: func() U { return parseZero }}
	}
	parseSelectorID := scopedSelectorID(parseId, parseSource.selectorSourceID())

	return UseDerived(parseSelectorID, func() U {
		return parseProject(parseSource.Get())
	}, parseSource.selectorSourceID())
}

// Select is a compatibility wrapper around UseSelector.
//
// Deprecated: Use UseSelector.
func Select[T any, U any](parseId string, parseSource selectorSource[T], parseProject func(T) U) Derived[U] {
	deprecation.Warn("state.Select", "state.UseSelector")
	return UseSelector(parseId, parseSource, parseProject)
}

// scopedSelectorID is a core package helper.
func scopedSelectorID(parseRequestedID string, parseSourceID string) string {
	parseStableID := runtime.GoUseIdGlobal()
	parseParts := []string{"selector", parseStableID}
	if strings.TrimSpace(parseRequestedID) != "" {
		parseParts = append(parseParts, parseRequestedID)
	}
	if strings.TrimSpace(parseSourceID) != "" {
		parseParts = append(parseParts, parseSourceID)
	}
	return strings.Join(parseParts, ":")
}

// Text renders an atom-backed reactive text node that can update without rerendering the owning component.
func (parseA Atom[T]) Text(render func(T) string) *Element {
	return createReactiveTextNode([]string{parseA.id}, parseA.Get, render)
}

// Text renders a derived-value-backed reactive text node that can update without rerendering the owning component.
func (parseD Derived[T]) Text(render func(T) string) *Element {
	return createReactiveTextNode([]string{parseD.id}, parseD.Get, render)
}

// createReactiveTextNode builds a fine-grained reactive text node. It subscribes the node to
// EVERY source id passed (a multi-source computed contributes all of its dependency ids), so the
// text flushes when ANY source changes — not only the first. The ids are stored comma-joined in
// the atom-id prop; the reconciler splits them when wiring subscriptions.
func createReactiveTextNode[T any](parseIDs []string, parseGetter func() T, render func(T) string) *Element {
	parseTextGetter := func() string {
		if parseGetter == nil {
			return ""
		}
		parseValue := parseGetter()
		if render != nil {
			return render(parseValue)
		}
		return fmt.Sprint(parseValue)
	}
	parseFiltered := make([]string, 0, len(parseIDs))
	for _, parseID := range parseIDs {
		if strings.TrimSpace(parseID) != "" {
			parseFiltered = append(parseFiltered, parseID)
		}
	}
	return runtime.CreateElement(runtime.ReactiveTextNodeType, map[string]any{
		runtimeReactiveTextAtomIDProp(): strings.Join(parseFiltered, ","),
		runtimeReactiveTextGetterProp(): parseTextGetter,
	})
}

// runtimeReactiveTextAtomIDProp is a core package helper.
func runtimeReactiveTextAtomIDProp() string {
	return "__gwc_reactive_text_atom_id"
}

// runtimeReactiveTextGetterProp is a core package helper.
func runtimeReactiveTextGetterProp() string {
	return "__gwc_reactive_text_getter"
}

// GetSnapshot returns a copy of all atoms currently registered in the global runtime.
//
// The returned snapshot preserves in-memory Go values exactly for same-process
// restore via ApplySnapshot. When serializing to JSON or browser storage, only
// JSON-compatible atom values should be relied on as stable persisted data.
func GetSnapshot() (Snapshot, error) {
	parseRaw := runtime.GetGlobalRuntime().SnapshotAtoms()
	parseSnapshot := make(Snapshot, len(parseRaw))
	maps.Copy(parseSnapshot, parseRaw)
	return parseSnapshot, nil
}

// ExportSnapshot returns a copy of all atoms currently registered in the global runtime.
//
// Deprecated: use GetSnapshot. ExportSnapshot is the older export/import-terminology name and
// is kept only for source compatibility.
func ExportSnapshot() (Snapshot, error) {
	deprecation.Warn("state.ExportSnapshot", "state.GetSnapshot")
	return GetSnapshot()
}

// Select returns a filtered snapshot containing only the requested atom keys.
func (parseS Snapshot) Select(parseKeys ...string) Snapshot {
	if len(parseKeys) == 0 {
		parseClone := make(Snapshot, len(parseS))
		maps.Copy(parseClone, parseS)
		return parseClone
	}

	parseSelected := make(Snapshot, len(parseKeys))
	for _, parseKey2 := range parseKeys {
		if parseValue2, parseOk := parseS[parseKey2]; parseOk {
			parseSelected[parseKey2] = parseValue2
		}
	}
	return parseSelected
}

// ApplySnapshot merges atom values from snapshot into the global runtime and
// schedules subscribed components for updates.
func ApplySnapshot(parseSnapshot Snapshot) error {
	return runtime.GetGlobalRuntime().RestoreAtomSnapshot(parseSnapshot)
}

// ImportSnapshot merges atom values from snapshot into the global runtime and
// schedules subscribed components for updates.
//
// Deprecated: use ApplySnapshot. ImportSnapshot is the older export/import-terminology name and
// is kept only for source compatibility.
func ImportSnapshot(parseSnapshot Snapshot) error {
	deprecation.Warn("state.ImportSnapshot", "state.ApplySnapshot")
	return ApplySnapshot(parseSnapshot)
}

// MarshalSnapshotJSON serializes snapshot for browser storage or transport.
//
// Persisted snapshots should only contain JSON-compatible values if stable
// round-tripping is required. Composite Go structs restore as generic JSON
// objects unless callers provide their own typed serialization layer.
func MarshalSnapshotJSON(parseSnapshot Snapshot) ([]byte, error) {
	if parseSnapshot == nil {
		parseSnapshot = Snapshot{}
	}
	return json.Marshal(snapshotWireEnvelope{
		Protocol: snapshotWireProtocol,
		Version:  currentSnapshotVersion,
		State:    parseSnapshot,
	})
}

// UnmarshalSnapshotJSON decodes a JSON snapshot produced by MarshalSnapshotJSON.
func UnmarshalSnapshotJSON(parseData []byte) (Snapshot, error) {
	if len(parseData) == 0 {
		return Snapshot{}, nil
	}

	var parseRaw map[string]json.RawMessage
	if parseErr := json.Unmarshal(parseData, &parseRaw); parseErr != nil {
		return nil, parseErr
	}
	if parseRaw == nil {
		return Snapshot{}, nil
	}

	if parseProtocolData, parseFound := parseRaw["protocol"]; parseFound {
		var parseProtocol string
		if parseErr := json.Unmarshal(parseProtocolData, &parseProtocol); parseErr != nil {
			return nil, fmt.Errorf("state: invalid snapshot protocol: %w", parseErr)
		}
		parseProtocol = strings.TrimSpace(parseProtocol)
		if parseProtocol != "" {
			var parseEnvelope snapshotWireEnvelope
			if parseErr2 := json.Unmarshal(parseData, &parseEnvelope); parseErr2 != nil {
				return nil, parseErr2
			}
			if parseProtocol != snapshotWireProtocol {
				return nil, fmt.Errorf("state: unsupported snapshot protocol %q", parseProtocol)
			}
			parseVersion, parseErr3 := normalizeSnapshotVersion(parseEnvelope.Version)
			if parseErr3 != nil {
				return nil, parseErr3
			}
			if parseEnvelope.State == nil {
				return Snapshot{}, nil
			}
			parseNormalized, parseOk := normalizeSnapshot(parseEnvelope.State).(Snapshot)
			if !parseOk {
				return nil, fmt.Errorf("state: snapshot normalization returned unexpected type")
			}
			return migrateSnapshot(parseVersion, parseNormalized)
		}
	}

	var parseSnapshot Snapshot
	if parseErr := json.Unmarshal(parseData, &parseSnapshot); parseErr != nil {
		return nil, parseErr
	}
	if parseSnapshot == nil {
		return Snapshot{}, nil
	}
	parseNormalized, parseOk := normalizeSnapshot(parseSnapshot).(Snapshot)
	if !parseOk {
		return nil, fmt.Errorf("state: snapshot normalization returned unexpected type")
	}
	return parseNormalized, nil
}

func normalizeSnapshotVersion(parseVersion int) (int, error) {
	if parseVersion < 0 {
		return 0, fmt.Errorf("state: unsupported snapshot version %d", parseVersion)
	}
	if parseVersion == 0 {
		return 0, nil
	}
	if parseVersion > currentSnapshotVersion {
		return 0, fmt.Errorf("state: unsupported snapshot version %d", parseVersion)
	}
	return parseVersion, nil
}

func migrateSnapshot(parseVersion int, parseSnapshot Snapshot) (Snapshot, error) {
	if parseVersion >= currentSnapshotVersion {
		return parseSnapshot, nil
	}
	parseCurrent := parseSnapshot.Select()
	for parseFrom := parseVersion; parseFrom < currentSnapshotVersion; parseFrom++ {
		snapshotMigrationMu.RLock()
		parseMigration := snapshotMigrations[parseFrom]
		snapshotMigrationMu.RUnlock()
		if parseMigration == nil {
			if parseFrom == 0 {
				return parseCurrent, nil
			}
			return nil, fmt.Errorf("state: missing snapshot migration from version %d to %d", parseFrom, parseFrom+1)
		}
		parseNext, parseErr := parseMigration(parseCurrent.Select())
		if parseErr != nil {
			return nil, fmt.Errorf("state: migrate snapshot from version %d to %d: %w", parseFrom, parseFrom+1, parseErr)
		}
		if parseNext == nil {
			parseNext = Snapshot{}
		}
		parseCurrent = parseNext.Select()
	}
	return parseCurrent, nil
}

// SaveSnapshot stores a JSON-encoded snapshot in browser storage.
func SaveSnapshot(parseKey string, parseSnapshot Snapshot, parseArea StorageArea) error {
	parseStorage, parseErr := openSnapshotStorage(parseArea)
	if parseErr != nil {
		return parseErr
	}

	parseData, parseErr := MarshalSnapshotJSON(parseSnapshot)
	if parseErr != nil {
		return parseErr
	}
	return parseStorage.SetItem(parseKey, string(parseData))
}

// LoadSnapshot reads and decodes a snapshot from browser storage.
func LoadSnapshot(parseKey string, parseArea StorageArea) (Snapshot, bool, error) {
	parseStorage, parseErr := openSnapshotStorage(parseArea)
	if parseErr != nil {
		return nil, false, parseErr
	}

	parseValue, parseOk, parseErr := parseStorage.GetItem(parseKey)
	if parseErr != nil || !parseOk {
		return nil, parseOk, parseErr
	}

	parseSnapshot, parseErr := UnmarshalSnapshotJSON([]byte(parseValue))
	if parseErr != nil {
		return nil, false, parseErr
	}
	return parseSnapshot, true, nil
}

// RestoreSnapshot loads a snapshot from browser storage and imports it.
func RestoreSnapshot(parseKey string, parseArea StorageArea) (bool, error) {
	parseSnapshot, parseOk, parseErr := LoadSnapshot(parseKey, parseArea)
	if parseErr != nil || !parseOk {
		return parseOk, parseErr
	}
	return true, ApplySnapshot(parseSnapshot)
}

// SavePersistentSnapshot stores a JSON-encoded snapshot in IndexedDB-first durable browser storage.
//
// If parseCtx is nil it is replaced with context.Background(). If parseCtx is
// already cancelled the underlying store resolver will receive a cancelled
// context and is expected to return an error, which is propagated to the caller.
func SavePersistentSnapshot(parseCtx context.Context, parseKey string, parseSnapshot Snapshot, parseOptions ...PersistentSnapshotOptions) error {
	store, parseErr := openPersistentSnapshotStore(parseCtx, parseOptions)
	if parseErr != nil {
		return parseErr
	}
	parseData, parseErr := MarshalSnapshotJSON(parseSnapshot)
	if parseErr != nil {
		return parseErr
	}
	return store.SetItem(resolvePersistentSnapshotContext(parseCtx), parseKey, string(parseData))
}

// LoadPersistentSnapshot reads and decodes a snapshot from IndexedDB-first durable browser storage.
//
// If parseCtx is nil it is replaced with context.Background(). If parseCtx is
// already cancelled the underlying store resolver will receive a cancelled
// context and is expected to return an error, which is propagated to the caller.
func LoadPersistentSnapshot(parseCtx context.Context, parseKey string, parseOptions ...PersistentSnapshotOptions) (Snapshot, bool, error) {
	store, parseErr := openPersistentSnapshotStore(parseCtx, parseOptions)
	if parseErr != nil {
		return nil, false, parseErr
	}
	parseValue, parseOk, parseErr := store.GetItem(resolvePersistentSnapshotContext(parseCtx), parseKey)
	if parseErr != nil || !parseOk {
		return nil, parseOk, parseErr
	}
	parseSnapshot, parseErr := UnmarshalSnapshotJSON([]byte(parseValue))
	if parseErr != nil {
		return nil, false, parseErr
	}
	return parseSnapshot, true, nil
}

// RestorePersistentSnapshot loads a durable snapshot and imports it into the current runtime.
//
// If parseCtx is nil it is replaced with context.Background(). If parseCtx is
// already cancelled the underlying store resolver will receive a cancelled
// context and is expected to return an error, which is propagated to the caller.
func RestorePersistentSnapshot(parseCtx context.Context, parseKey string, parseOptions ...PersistentSnapshotOptions) (bool, error) {
	parseSnapshot, parseOk, parseErr := LoadPersistentSnapshot(parseCtx, parseKey, parseOptions...)
	if parseErr != nil || !parseOk {
		return parseOk, parseErr
	}
	return true, ApplySnapshot(parseSnapshot)
}

// openPersistentSnapshotStore is a core package helper.
func openPersistentSnapshotStore(parseCtx context.Context, parseOptions []PersistentSnapshotOptions) (interop.PersistentStore, error) {
	parseResolved := resolvePersistentSnapshotOptions(parseOptions)
	parseResolver := parseResolved.StoreResolver
	if parseResolver == nil {
		parseFallbackResolver := parseResolved.FallbackResolver
		if parseFallbackResolver == nil {
			parseFallbackResolver = interop.GetLocalStorage
		}
		parseFallbackBackend := strings.TrimSpace(parseResolved.FallbackBackend)
		if parseFallbackBackend == "" {
			parseFallbackBackend = "localStorage"
		}
		storeName := strings.TrimSpace(parseResolved.StoreName)
		if storeName == "" {
			storeName = defaultPersistentSnapshotStoreName
		}
		parseResolver = func(parseCtx2 context.Context) (interop.PersistentStore, error) {
			return interop.OpenPersistentStore(parseCtx2, interop.PersistentStoreOptions{
				Name:               storeName,
				DatabaseName:       parseResolved.DatabaseName,
				DeleteOnCorruption: parseResolved.DeleteOnCorruption,
				FallbackResolver:   parseFallbackResolver,
				FallbackBackend:    parseFallbackBackend,
			})
		}
	}
	return parseResolver(resolvePersistentSnapshotContext(parseCtx))
}

// resolvePersistentSnapshotOptions is a core package helper.
func resolvePersistentSnapshotOptions(parseOptions []PersistentSnapshotOptions) PersistentSnapshotOptions {
	if len(parseOptions) == 0 {
		return PersistentSnapshotOptions{}
	}
	return parseOptions[0]
}

// resolvePersistentSnapshotContext is a core package helper.
func resolvePersistentSnapshotContext(parseCtx context.Context) context.Context {
	if parseCtx != nil {
		return parseCtx
	}
	return context.Background()
}

// openSnapshotStorage is a core package helper.
func openSnapshotStorage(parseArea StorageArea) (interop.Storage, error) {
	switch parseArea {
	case LocalStorage:
		return loadStateLocalStorage()
	case SessionStorage:
		return loadStateSessionStorage()
	default:
		return interop.Storage{}, fmt.Errorf("storage area %q is not available; valid areas are %q (state.LocalStorage) and %q (state.SessionStorage)", parseArea, LocalStorage, SessionStorage)
	}
}

// normalizeSnapshot is a core package helper.
func normalizeSnapshot(parseValue any) any {
	switch parseTyped := parseValue.(type) {
	case Snapshot:
		parseNormalized := make(Snapshot, len(parseTyped))
		for parseKey, parseNested := range parseTyped {
			parseNormalized[parseKey] = normalizeSnapshot(parseNested)
		}
		return parseNormalized
	case map[string]any:
		parseNormalized2 := make(map[string]any, len(parseTyped))
		for parseKey2, parseNested2 := range parseTyped {
			parseNormalized2[parseKey2] = normalizeSnapshot(parseNested2)
		}
		return parseNormalized2
	case []any:
		parseNormalized3 := make([]any, len(parseTyped))
		for parseIndex, parseNested3 := range parseTyped {
			parseNormalized3[parseIndex] = normalizeSnapshot(parseNested3)
		}
		return parseNormalized3
	case float64:
		if math.Trunc(parseTyped) == parseTyped &&
			parseTyped >= math.MinInt64 && parseTyped <= math.MaxInt64 &&
			parseTyped >= -1<<53 && parseTyped <= 1<<53 {
			return int(parseTyped)
		}
		return parseTyped
	default:
		return parseValue
	}
}
