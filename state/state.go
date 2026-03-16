//go:build js && wasm
// +build js,wasm

package state

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// Type alias for Element
type Element = runtime.Element

type Atom[T any] struct {
	get func() T
	set func(T)
}

type Computed[T any] struct {
	get func() T
}

type Derived[T any] struct {
	get func() T
}

type Snapshot map[string]interface{}

type StorageArea string

const (
	LocalStorage   StorageArea = "localStorage"
	SessionStorage StorageArea = "sessionStorage"
)

// UseAtom provides SolidJS-style fine-grained reactivity with global atoms.
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
	return Atom[T]{get: get, set: set}
}

func (a Atom[T]) Get() T {
	return a.get()
}

func (a Atom[T]) Set(value T) {
	a.set(value)
}

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
		return Derived[T]{get: func() T { return zero }}
	}

	atom := UseAtom(id, zero)
	return Derived[T]{get: atom.Get}
}

func (d Derived[T]) Get() T {
	if d.get == nil {
		var zero T
		return zero
	}
	return d.get()
}

// ExportSnapshot returns a copy of all atoms currently registered in the global runtime.
//
// The returned snapshot preserves in-memory Go values exactly for same-process
// restore via ImportSnapshot. When serializing to JSON or browser storage, only
// JSON-compatible atom values should be relied on as stable persisted data.
func ExportSnapshot() Snapshot {
	raw := runtime.GetGlobalRuntime().SnapshotAtoms()
	snapshot := make(Snapshot, len(raw))
	for key, value := range raw {
		snapshot[key] = value
	}
	return snapshot
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

// ImportSnapshot merges atom values from snapshot into the global runtime and
// schedules subscribed components for updates.
func ImportSnapshot(snapshot Snapshot) error {
	return runtime.GetGlobalRuntime().RestoreAtomSnapshot(snapshot)
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
	return snapshot, nil
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
	return true, ImportSnapshot(snapshot)
}

func getStorage(area StorageArea) js.Value {
	window := js.Global()
	storage := window.Get(string(area))
	if storage.Truthy() {
		return storage
	}
	return js.Undefined()
}
