package state

import (
	"reflect"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// GlobalAtom is a non-hook handle to a shared atom, readable and writable from
// ANY context — crucially from OUTSIDE a render: global keyboard handlers,
// undo/redo, post-decrypt hydration, network/event callbacks, or any goroutine.
//
// It targets the same atom registry as [UseAtom], keyed by the same id, so a
// component that reads the id via UseAtom(id, default) re-renders automatically
// when a GlobalAtom write changes the value. This is the answer to G39: it
// retires the render-phase "capture variable + captured bool" triad and removes
// the pre-render-write silent-drop bug — a value written through GlobalAtom
// before the first render persists (UseAtom seeds its default only when the atom
// is absent) and is observed by the first UseAtom read.
//
// Contract: a GlobalAtom and any UseAtom sharing an id MUST agree on the default
// value. Construct a GlobalAtom once (e.g. a package var) and share it.
type GlobalAtom[T any] struct {
	id  string
	def T
}

// NewGlobalAtom returns a handle for the atom identified by id and seeds the
// registry with defaultValue if (and only if) the atom has no value yet. Seeding
// never notifies subscribers and never clobbers an existing value, so it is safe
// to construct at package-init time or after an external write.
func NewGlobalAtom[T any](parseID string, parseDefault T) GlobalAtom[T] {
	parseAtom := GlobalAtom[T]{id: parseID, def: parseDefault}
	if parseRt := runtime.GetGlobalRuntime(); parseRt != nil {
		parseRt.InitAtomValue(parseID, parseDefault)
	}
	return parseAtom
}

// ID returns the atom's identifier (the key shared with UseAtom).
func (parseAtom GlobalAtom[T]) ID() string { return parseAtom.id }

// Get returns the current value, or the handle's default when the runtime is
// unavailable or the stored value is not of type T.
func (parseAtom GlobalAtom[T]) Get() T {
	parseRt := runtime.GetGlobalRuntime()
	if parseRt == nil {
		return parseAtom.def
	}
	parseValue, parseOk := parseRt.GetAtomValue(parseAtom.id)
	if !parseOk {
		return parseAtom.def
	}
	if parseTyped, parseOk := parseValue.(T); parseOk {
		return parseTyped
	}
	return parseAtom.def
}

// Set writes a new value and schedules a re-render of every component subscribed
// to this id via UseAtom. Safe to call from any goroutine or callback. A no-op
// when the runtime is unavailable (e.g. native SSR with no global runtime).
//
// Writing a value equal to the current one is a no-op (no re-render), matching
// UseState's behavior. Equality is a fast == with a structural reflect.DeepEqual fallback
// for non-comparable types (slice/map), so an equal slice/map is also a no-op.
func (parseAtom GlobalAtom[T]) Set(parseValue T) {
	parseRt := runtime.GetGlobalRuntime()
	if parseRt == nil {
		return
	}
	if parseCurrent, parseOk := parseRt.GetAtomValue(parseAtom.id); parseOk && atomValuesEqual(parseCurrent, parseValue) {
		return
	}
	_ = parseRt.SetAtomValue(parseAtom.id, parseValue)
}

// atomValuesEqual reports whether two atom values are equal. It first tries a fast ==; on a
// non-comparable type (slice/map/func) == panics, and it falls back to a structural
// reflect.DeepEqual so that writing an equal slice/map does NOT needlessly re-notify (the
// previous behavior always re-rendered for non-comparable payloads).
func atomValuesEqual(parseA any, parseB any) (parseEqual bool) {
	defer func() {
		if recover() != nil {
			parseEqual = reflect.DeepEqual(parseA, parseB)
		}
	}()
	return parseA == parseB
}

// Update applies fn to the current value and stores the result.
func (parseAtom GlobalAtom[T]) Update(parseFn func(T) T) {
	parseAtom.Set(parseFn(parseAtom.Get()))
}
