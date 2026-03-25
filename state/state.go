//go:build js && wasm
// +build js,wasm

package state

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/interop"
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
type Snapshot map[string]interface{}

// StorageArea names a browser storage backend.
type StorageArea string

type PersistentSnapshotOptions struct {
	DatabaseName       string
	StoreName          string
	DeleteOnCorruption bool
	FallbackResolver   func() (interop.Storage, error)
	FallbackBackend    string
	StoreResolver      func(context.Context) (interop.PersistentStore, error)
}

const (
	// LocalStorage stores snapshots in window.localStorage.
	LocalStorage StorageArea = "localStorage"
	// SessionStorage stores snapshots in window.sessionStorage.
	SessionStorage StorageArea = "sessionStorage"
)

const defaultPersistentSnapshotStoreName = "state-snapshots"

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
// Returns:
//   - A getter function that returns the current atom value
//   - A setter function that updates the atom and triggers re-renders in all subscribers
//
// Example - Theme Management:
//
//	// In theme switcher component
//	func ThemeSwitcher(props dom.Attrs) *fiber.Element {
//	    theme, setTheme := state.UseAtom("appTheme", "light")
//
//	    toggle := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	        if theme() == "light" {
//	            setTheme("dark")
//	        } else {
//	            setTheme("light")
//	        }
//	        return nil
//	    })
//
//	    return dom.Button(map[string]interface{}{"onclick": toggle},
//	        fmt.Sprintf("Switch to %s mode", theme()))
//	}
//
//	// In header component - automatically updates when theme changes
//	func Header(props dom.Attrs) *fiber.Element {
//	    theme, _ := state.UseAtom("appTheme", "light")
//
//	    bgColor := "white"
//	    if theme() == "dark" {
//	        bgColor = "#333"
//	    }
//
//	    return dom.Header(map[string]interface{}{
//	        "style": map[string]string{"background-color": bgColor},
//	    }, dom.H1(nil, "My App"))
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
//	func LoginButton(props dom.Attrs) *fiber.Element {
//	    user, setUser := state.UseAtom("currentUser", User{})
//
//	    login := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	        setUser(User{ID: 1, Username: "johndoe", Email: "john@example.com"})
//	        return nil
//	    })
//
//	    if user().ID == 0 {
//	        return dom.Button(map[string]interface{}{"onclick": login}, "Login")
//	    }
//	    return dom.Span(nil, fmt.Sprintf("Welcome, %s", user().Username))
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
//
// Consider using structured types (structs) for complex state
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
func UseComputed[T any](parseCompute func() T, parseDeps ...interface{}) Computed[T] {
	parseValue := runtime.GoUseMemoGlobal(func() interface{} {
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
func UseDerived[T any](parseId string, parseCompute func() T, parseDeps ...string) Derived[T] {
	var parseZero T
	if parseErr := runtime.GetGlobalRuntime().RegisterDerivedAtom(parseId, parseDeps, func() interface{} {
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
func Select[T any, U any](parseId string, parseSource selectorSource[T], parseProject func(T) U) Derived[U] {
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
	return createReactiveTextNode(parseA.id, parseA.Get, render)
}

// Text renders a derived-value-backed reactive text node that can update without rerendering the owning component.
func (parseD Derived[T]) Text(render func(T) string) *Element {
	return createReactiveTextNode(parseD.id, parseD.Get, render)
}

// createReactiveTextNode is a core package helper.
func createReactiveTextNode[T any](parseId string, parseGetter func() T, render func(T) string) *Element {
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
	return runtime.CreateElement(runtime.ReactiveTextNodeType, map[string]interface{}{
		runtimeReactiveTextAtomIDProp(): parseId,
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
	for parseKey, parseValue := range parseRaw {
		parseSnapshot[parseKey] = parseValue
	}
	return parseSnapshot, nil
}

// ExportSnapshot returns a copy of all atoms currently registered in the global runtime.
//
// It preserves the original public API name and behavior for callers that still
// use the export/import snapshot terminology.
func ExportSnapshot() (Snapshot, error) {
	return GetSnapshot()
}

// Select returns a filtered snapshot containing only the requested atom keys.
func (parseS Snapshot) Select(parseKeys ...string) Snapshot {
	if len(parseKeys) == 0 {
		parseClone := make(Snapshot, len(parseS))
		for parseKey, parseValue := range parseS {
			parseClone[parseKey] = parseValue
		}
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
// It preserves the original public API name and behavior for callers that still
// use the export/import snapshot terminology.
func ImportSnapshot(parseSnapshot Snapshot) error {
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
	return json.Marshal(parseSnapshot)
}

// UnmarshalSnapshotJSON decodes a JSON snapshot produced by MarshalSnapshotJSON.
func UnmarshalSnapshotJSON(parseData []byte) (Snapshot, error) {
	if len(parseData) == 0 {
		return Snapshot{}, nil
	}

	var parseSnapshot Snapshot
	if parseErr := json.Unmarshal(parseData, &parseSnapshot); parseErr != nil {
		return nil, parseErr
	}
	if parseSnapshot == nil {
		return Snapshot{}, nil
	}
	return normalizeSnapshot(parseSnapshot).(Snapshot), nil
}

// SaveSnapshot stores a JSON-encoded snapshot in browser storage.
func SaveSnapshot(parseKey string, parseSnapshot Snapshot, parseArea StorageArea) error {
	parseStorage := getStorage(parseArea)
	if !parseStorage.Truthy() {
		return fmt.Errorf("%s is not available", parseArea)
	}

	parseData, parseErr := MarshalSnapshotJSON(parseSnapshot)
	if parseErr != nil {
		return parseErr
	}
	parseStorage.Call("setItem", parseKey, string(parseData))
	return nil
}

// LoadSnapshot reads and decodes a snapshot from browser storage.
func LoadSnapshot(parseKey string, parseArea StorageArea) (Snapshot, bool, error) {
	parseStorage := getStorage(parseArea)
	if !parseStorage.Truthy() {
		return nil, false, fmt.Errorf("%s is not available", parseArea)
	}

	parseValue := parseStorage.Call("getItem", parseKey)
	if parseValue.IsNull() || parseValue.IsUndefined() {
		return nil, false, nil
	}

	parseSnapshot, parseErr := UnmarshalSnapshotJSON([]byte(parseValue.String()))
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

// getStorage is a core package helper.
func getStorage(parseArea StorageArea) js.Value {
	parseGlobalObject := js.Global()
	parseStorage := parseGlobalObject.Get(string(parseArea))
	if parseStorage.Truthy() {
		return parseStorage
	}
	return js.Undefined()
}

// normalizeSnapshot is a core package helper.
func normalizeSnapshot(parseValue interface{}) interface{} {
	switch parseTyped := parseValue.(type) {
	case Snapshot:
		parseNormalized := make(Snapshot, len(parseTyped))
		for parseKey, parseNested := range parseTyped {
			parseNormalized[parseKey] = normalizeSnapshot(parseNested)
		}
		return parseNormalized
	case map[string]interface{}:
		parseNormalized2 := make(map[string]interface{}, len(parseTyped))
		for parseKey2, parseNested2 := range parseTyped {
			parseNormalized2[parseKey2] = normalizeSnapshot(parseNested2)
		}
		return parseNormalized2
	case []interface{}:
		parseNormalized3 := make([]interface{}, len(parseTyped))
		for parseIndex, parseNested3 := range parseTyped {
			parseNormalized3[parseIndex] = normalizeSnapshot(parseNested3)
		}
		return parseNormalized3
	case float64:
		if math.Trunc(parseTyped) == parseTyped {
			return int(parseTyped)
		}
		return parseTyped
	default:
		return parseValue
	}
}
