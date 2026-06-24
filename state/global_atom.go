package state

import "github.com/monstercameron/GoWebComponents/internal/runtime"

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
func (parseAtom GlobalAtom[T]) Set(parseValue T) {
	if parseRt := runtime.GetGlobalRuntime(); parseRt != nil {
		_ = parseRt.SetAtomValue(parseAtom.id, parseValue)
	}
}

// Update applies fn to the current value and stores the result.
func (parseAtom GlobalAtom[T]) Update(parseFn func(T) T) {
	parseAtom.Set(parseFn(parseAtom.Get()))
}
