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
func UseAtom[T any](id string, initialValue T) Atom[T] {
	get, set := runtime.GoUseAtomGlobal(id, initialValue)
	return Atom[T]{id: id, get: get, set: set}
}

// Get returns the current atom value.
func (a Atom[T]) Get() T {
	return a.get()
}

// Set replaces the atom value.
func (a Atom[T]) Set(value T) {
	a.set(value)
}

// Update replaces the atom value using the previous value.
func (a Atom[T]) Update(fn func(T) T) {
	a.set(fn(a.get()))
}

// UseComputed derives a typed value from other state used by the current component.
//
// It is intended for render-time derived values, especially when a component is
// already reading one or more atoms and wants a typed handle instead of using
// ui.UseMemo directly in every call site.
//
// The computed value is memoized according to the provided dependency list.
// Callers should pass the values that should trigger recomputation.
func UseComputed[T any](compute func() T, deps ...interface{}) Computed[T] {
	value := runtime.GoUseMemoGlobal(func() interface{} {
		return compute()
	}, deps...)

	cast, ok := value.(T)
	if !ok {
		var zero T
		return Computed[T]{get: func() T { return zero }}
	}

	return Computed[T]{get: func() T { return cast }}
}

// Get returns the current computed value.
func (c Computed[T]) Get() T {
	if c.get == nil {
		var zero T
		return zero
	}

	return c.get()
}

// UseDerived registers and subscribes to a read-only derived atom.
//
// Derived atoms are shared state values keyed by id. They recompute when one of
// the named source atom IDs changes and expose a typed read-only handle to the
// current derived value. Dependency tracking is explicit through atom IDs so
// recomputation remains predictable and avoids hidden runtime graph discovery.
func UseDerived[T any](id string, compute func() T, deps ...string) Derived[T] {
	var zero T
	if err := runtime.GetGlobalRuntime().RegisterDerivedAtom(id, deps, func() interface{} {
		return compute()
	}); err != nil {
		return Derived[T]{id: id, get: func() T { return zero }}
	}

	atom := UseAtom(id, zero)
	return Derived[T]{id: id, get: atom.Get}
}

// Get returns the current derived value.
func (d Derived[T]) Get() T {
	if d.get == nil {
		var zero T
		return zero
	}
	return d.get()
}

func (a Atom[T]) selectorSourceID() string {
	return a.id
}

func (a Atom[T]) ReactiveRegionSourceIDs() []string {
	if a.id == "" {
		return nil
	}
	return []string{a.id}
}

func (d Derived[T]) selectorSourceID() string {
	return d.id
}

func (d Derived[T]) ReactiveRegionSourceIDs() []string {
	if d.id == "" {
		return nil
	}
	return []string{d.id}
}

// Select creates a read-only projected shared value from an atom or derived source.
//
// The selector remains explicit: callers provide the derived ID to register and the
// source handle to project from. When the projected value is unchanged, subscribers
// are not notified, which makes it suitable for fine-grained hot-value paths.
func Select[T any, U any](id string, source selectorSource[T], project func(T) U) Derived[U] {
	var zero U
	if source == nil || project == nil {
		return Derived[U]{id: id, get: func() U { return zero }}
	}
	selectorID := scopedSelectorID(id, source.selectorSourceID())

	return UseDerived(selectorID, func() U {
		return project(source.Get())
	}, source.selectorSourceID())
}

func scopedSelectorID(requestedID string, sourceID string) string {
	stableID := runtime.GoUseIdGlobal()
	parts := []string{"selector", stableID}
	if strings.TrimSpace(requestedID) != "" {
		parts = append(parts, requestedID)
	}
	if strings.TrimSpace(sourceID) != "" {
		parts = append(parts, sourceID)
	}
	return strings.Join(parts, ":")
}

// Text renders an atom-backed reactive text node that can update without rerendering the owning component.
func (a Atom[T]) Text(render func(T) string) *Element {
	return createReactiveTextNode(a.id, a.Get, render)
}

// Text renders a derived-value-backed reactive text node that can update without rerendering the owning component.
func (d Derived[T]) Text(render func(T) string) *Element {
	return createReactiveTextNode(d.id, d.Get, render)
}

func createReactiveTextNode[T any](id string, getter func() T, render func(T) string) *Element {
	textGetter := func() string {
		if getter == nil {
			return ""
		}
		value := getter()
		if render != nil {
			return render(value)
		}
		return fmt.Sprint(value)
	}
	return runtime.CreateElement(runtime.ReactiveTextNodeType, map[string]interface{}{
		runtimeReactiveTextAtomIDProp(): id,
		runtimeReactiveTextGetterProp(): textGetter,
	})
}

func runtimeReactiveTextAtomIDProp() string {
	return "__gwc_reactive_text_atom_id"
}

func runtimeReactiveTextGetterProp() string {
	return "__gwc_reactive_text_getter"
}

// GetSnapshot returns a copy of all atoms currently registered in the global runtime.
//
// The returned snapshot preserves in-memory Go values exactly for same-process
// restore via ApplySnapshot. When serializing to JSON or browser storage, only
// JSON-compatible atom values should be relied on as stable persisted data.
func GetSnapshot() (Snapshot, error) {
	raw := runtime.GetGlobalRuntime().SnapshotAtoms()
	snapshot := make(Snapshot, len(raw))
	for key, value := range raw {
		snapshot[key] = value
	}
	return snapshot, nil
}

// ExportSnapshot returns a copy of all atoms currently registered in the global runtime.
//
// It preserves the original public API name and behavior for callers that still
// use the export/import snapshot terminology.
func ExportSnapshot() (Snapshot, error) {
	return GetSnapshot()
}

// Select returns a filtered snapshot containing only the requested atom keys.
func (s Snapshot) Select(keys ...string) Snapshot {
	if len(keys) == 0 {
		clone := make(Snapshot, len(s))
		for key, value := range s {
			clone[key] = value
		}
		return clone
	}

	selected := make(Snapshot, len(keys))
	for _, key := range keys {
		if value, ok := s[key]; ok {
			selected[key] = value
		}
	}
	return selected
}

// ApplySnapshot merges atom values from snapshot into the global runtime and
// schedules subscribed components for updates.
func ApplySnapshot(snapshot Snapshot) error {
	return runtime.GetGlobalRuntime().RestoreAtomSnapshot(snapshot)
}

// ImportSnapshot merges atom values from snapshot into the global runtime and
// schedules subscribed components for updates.
//
// It preserves the original public API name and behavior for callers that still
// use the export/import snapshot terminology.
func ImportSnapshot(snapshot Snapshot) error {
	return ApplySnapshot(snapshot)
}

// MarshalSnapshotJSON serializes snapshot for browser storage or transport.
//
// Persisted snapshots should only contain JSON-compatible values if stable
// round-tripping is required. Composite Go structs restore as generic JSON
// objects unless callers provide their own typed serialization layer.
func MarshalSnapshotJSON(snapshot Snapshot) ([]byte, error) {
	if snapshot == nil {
		snapshot = Snapshot{}
	}
	return json.Marshal(snapshot)
}

// UnmarshalSnapshotJSON decodes a JSON snapshot produced by MarshalSnapshotJSON.
func UnmarshalSnapshotJSON(data []byte) (Snapshot, error) {
	if len(data) == 0 {
		return Snapshot{}, nil
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}
	if snapshot == nil {
		return Snapshot{}, nil
	}
	return normalizeSnapshot(snapshot).(Snapshot), nil
}

// SaveSnapshot stores a JSON-encoded snapshot in browser storage.
func SaveSnapshot(key string, snapshot Snapshot, area StorageArea) error {
	storage := getStorage(area)
	if !storage.Truthy() {
		return fmt.Errorf("%s is not available", area)
	}

	data, err := MarshalSnapshotJSON(snapshot)
	if err != nil {
		return err
	}
	storage.Call("setItem", key, string(data))
	return nil
}

// LoadSnapshot reads and decodes a snapshot from browser storage.
func LoadSnapshot(key string, area StorageArea) (Snapshot, bool, error) {
	storage := getStorage(area)
	if !storage.Truthy() {
		return nil, false, fmt.Errorf("%s is not available", area)
	}

	value := storage.Call("getItem", key)
	if value.IsNull() || value.IsUndefined() {
		return nil, false, nil
	}

	snapshot, err := UnmarshalSnapshotJSON([]byte(value.String()))
	if err != nil {
		return nil, false, err
	}
	return snapshot, true, nil
}

// RestoreSnapshot loads a snapshot from browser storage and imports it.
func RestoreSnapshot(key string, area StorageArea) (bool, error) {
	snapshot, ok, err := LoadSnapshot(key, area)
	if err != nil || !ok {
		return ok, err
	}
	return true, ApplySnapshot(snapshot)
}

// SavePersistentSnapshot stores a JSON-encoded snapshot in IndexedDB-first durable browser storage.
func SavePersistentSnapshot(ctx context.Context, key string, snapshot Snapshot, options ...PersistentSnapshotOptions) error {
	store, err := openPersistentSnapshotStore(ctx, options)
	if err != nil {
		return err
	}
	data, err := MarshalSnapshotJSON(snapshot)
	if err != nil {
		return err
	}
	return store.SetItem(resolvePersistentSnapshotContext(ctx), key, string(data))
}

// LoadPersistentSnapshot reads and decodes a snapshot from IndexedDB-first durable browser storage.
func LoadPersistentSnapshot(ctx context.Context, key string, options ...PersistentSnapshotOptions) (Snapshot, bool, error) {
	store, err := openPersistentSnapshotStore(ctx, options)
	if err != nil {
		return nil, false, err
	}
	value, ok, err := store.GetItem(resolvePersistentSnapshotContext(ctx), key)
	if err != nil || !ok {
		return nil, ok, err
	}
	snapshot, err := UnmarshalSnapshotJSON([]byte(value))
	if err != nil {
		return nil, false, err
	}
	return snapshot, true, nil
}

// RestorePersistentSnapshot loads a durable snapshot and imports it into the current runtime.
func RestorePersistentSnapshot(ctx context.Context, key string, options ...PersistentSnapshotOptions) (bool, error) {
	snapshot, ok, err := LoadPersistentSnapshot(ctx, key, options...)
	if err != nil || !ok {
		return ok, err
	}
	return true, ApplySnapshot(snapshot)
}

func openPersistentSnapshotStore(ctx context.Context, options []PersistentSnapshotOptions) (interop.PersistentStore, error) {
	resolved := resolvePersistentSnapshotOptions(options)
	resolver := resolved.StoreResolver
	if resolver == nil {
		fallbackResolver := resolved.FallbackResolver
		if fallbackResolver == nil {
			fallbackResolver = interop.GetLocalStorage
		}
		fallbackBackend := strings.TrimSpace(resolved.FallbackBackend)
		if fallbackBackend == "" {
			fallbackBackend = "localStorage"
		}
		storeName := strings.TrimSpace(resolved.StoreName)
		if storeName == "" {
			storeName = defaultPersistentSnapshotStoreName
		}
		resolver = func(ctx context.Context) (interop.PersistentStore, error) {
			return interop.OpenPersistentStore(ctx, interop.PersistentStoreOptions{
				Name:               storeName,
				DatabaseName:       resolved.DatabaseName,
				DeleteOnCorruption: resolved.DeleteOnCorruption,
				FallbackResolver:   fallbackResolver,
				FallbackBackend:    fallbackBackend,
			})
		}
	}
	return resolver(resolvePersistentSnapshotContext(ctx))
}

func resolvePersistentSnapshotOptions(options []PersistentSnapshotOptions) PersistentSnapshotOptions {
	if len(options) == 0 {
		return PersistentSnapshotOptions{}
	}
	return options[0]
}

func resolvePersistentSnapshotContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func getStorage(area StorageArea) js.Value {
	globalObject := js.Global()
	storage := globalObject.Get(string(area))
	if storage.Truthy() {
		return storage
	}
	return js.Undefined()
}

func normalizeSnapshot(value interface{}) interface{} {
	switch typed := value.(type) {
	case Snapshot:
		normalized := make(Snapshot, len(typed))
		for key, nested := range typed {
			normalized[key] = normalizeSnapshot(nested)
		}
		return normalized
	case map[string]interface{}:
		normalized := make(map[string]interface{}, len(typed))
		for key, nested := range typed {
			normalized[key] = normalizeSnapshot(nested)
		}
		return normalized
	case []interface{}:
		normalized := make([]interface{}, len(typed))
		for index, nested := range typed {
			normalized[index] = normalizeSnapshot(nested)
		}
		return normalized
	case float64:
		if math.Trunc(typed) == typed {
			return int(typed)
		}
		return typed
	default:
		return value
	}
}
