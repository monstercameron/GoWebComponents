package state

// AtomKey is a typed, declared-once handle to a shared atom. Declaring the id, value type, and
// default in ONE package-level var —
//
//	var ThemeAtom = state.NewAtomKey("app.theme", "light")
//
// and passing that key to UseAtomKey / ThemeAtom.Global() everywhere the atom is read or written
// gives three things the stringly UseAtom(id, default) form cannot:
//
//   - Type + default consistency: a UseAtom("app.theme", "light") and a NewGlobalAtom("app.theme", 0)
//     elsewhere silently disagree on type and default; with a key they share the one declaration.
//   - A compile-checked name: a typo in the key var is a compile error, whereas a typo'd id string
//     silently creates a brand-new, empty atom.
//   - A stringless call site: you pass ThemeAtom, not "app.theme" + "light" repeated everywhere.
//
// Two keys that nonetheless choose the same id string still address the same atom — AtomKey
// centralizes the contract, it does not namespace ids. Share one key var for one logical atom.
// (Corollary: an AtomKey[string] and an AtomKey[int] declared with the SAME id hit one registry
// slot; whichever seeds first wins and the other's reads fall back to its default — exactly the
// raw-UseAtom type-mismatch footgun, not a new one. For a pointer/slice/map T, Default() hands
// every caller the same reference, so mutating it aliases; keep defaults immutable.)
type AtomKey[T any] struct {
	id      string
	initial T
}

// NewAtomKey declares a typed atom key. Call it once at package scope and reuse the returned value.
func NewAtomKey[T any](parseID string, parseInitial T) AtomKey[T] {
	return AtomKey[T]{id: parseID, initial: parseInitial}
}

// ID returns the underlying atom id this key maps to.
func (parseKey AtomKey[T]) ID() string { return parseKey.id }

// Default returns the key's seeded default value.
func (parseKey AtomKey[T]) Default() T { return parseKey.initial }

// Global returns the out-of-render handle for this key — the typed sibling of NewGlobalAtom — so
// code outside a component can read or write the atom: ThemeAtom.Global().Set("dark"). The default
// is seeded only if the atom has no value yet (it never clobbers an existing write).
func (parseKey AtomKey[T]) Global() GlobalAtom[T] {
	return NewGlobalAtom(parseKey.id, parseKey.initial)
}

// UseAtomKey subscribes the current component to a typed atom key — the typed sibling of UseAtom.
// Prefer it over UseAtom for any atom shared across components or packages, so the id, type, and
// default are declared once and can't drift between call sites.
func UseAtomKey[T any](parseKey AtomKey[T]) Atom[T] {
	return UseAtom(parseKey.id, parseKey.initial)
}
