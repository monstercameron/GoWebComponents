package state

import (
	"fmt"
	"strconv"
	"sync/atomic"
)

// Signal is a fine-grained reactive value: a terse, ergonomic handle over the
// shared atom registry that updates exactly the DOM nodes bound to it (via
// [Signal.Text]) and the regions/components subscribed to it — without forcing a
// re-render of the component that owns it.
//
// It is the recommended fine-grained primitive. Compared to its neighbors:
//
//   - vs [UseAtom]: a Signal needs no caller-managed string id (one is minted),
//     and it is created with [NewSignal] outside the hook/render lifecycle, so it
//     can live in a package var, an event handler, or a goroutine.
//   - vs Solid/Svelte signals: GWC tracks derivation dependencies EXPLICITLY
//     ([NewComputed] names its sources) rather than discovering them through a
//     hidden runtime graph. That is a deliberate design choice — predictable,
//     auditable reactivity over implicit magic.
//
// A Signal is a small value (an id + default); copy it freely. Reads and writes
// go through the global atom registry and are safe from any goroutine.
type Signal[T any] struct {
	atom GlobalAtom[T]
}

// signalSeq mints process-unique ids for anonymous signals and computeds.
var signalSeq atomic.Uint64

// signalID returns a process-unique atom id for an anonymous signal/computed.
func signalID() string {
	return "gwc.signal." + strconv.FormatUint(signalSeq.Add(1), 10)
}

// NewSignal creates a fine-grained reactive value seeded with initial.
//
// It may be called anywhere — including outside a component render (package init,
// event handlers, goroutines) — because, unlike a hook, it does not depend on
// call order within a render. Each call mints a fresh, process-unique identity;
// use [NewKeyedSignal] when several call sites must address the same signal.
func NewSignal[T any](parseInitial T) Signal[T] {
	return Signal[T]{atom: NewGlobalAtom(signalID(), parseInitial)}
}

// NewKeyedSignal creates a signal with an explicit, shared id, so independent
// call sites can address the same reactive value (the way [UseAtom] does). Prefer
// [NewSignal] unless the shared identity is the point. Call sites sharing an id
// must agree on the initial value.
func NewKeyedSignal[T any](parseID string, parseInitial T) Signal[T] {
	return Signal[T]{atom: NewGlobalAtom(parseID, parseInitial)}
}

// Get returns the current value of the signal.
func (parseS Signal[T]) Get() T {
	return parseS.atom.Get()
}

// Peek returns the current value without implying a reactive subscription. It is
// identical to [Signal.Get] today, named for parity with signal libraries and to
// document intent at call sites that must read a value without binding to it.
func (parseS Signal[T]) Peek() T {
	return parseS.atom.Get()
}

// Set writes a new value and notifies bound text nodes, reactive regions, and
// [UseAtom] subscribers of the same id. Writing a value equal to the current one
// is a no-op (no notification), matching UseState semantics. Safe from any
// goroutine; a no-op when the runtime is unavailable (e.g. native SSR).
func (parseS Signal[T]) Set(parseValue T) {
	parseS.atom.Set(parseValue)
}

// Update applies fn to the current value and stores the result.
func (parseS Signal[T]) Update(parseFn func(T) T) {
	parseS.atom.Update(parseFn)
}

// ID returns the underlying atom id — the source key used by reactive regions and
// selectors, and the key a [UseAtom] reader would share.
func (parseS Signal[T]) ID() string {
	return parseS.atom.ID()
}

// Text renders a fine-grained reactive text node bound to this signal: it updates
// in place when the signal changes, WITHOUT re-rendering the component that
// returned it. This is the primary fine-grained authoring path.
//
//	count := state.NewSignal(0)
//	// ...
//	h.Span(count.Text(func(n int) string { return fmt.Sprintf("%d", n) }))
func (parseS Signal[T]) Text(render func(T) string) *Element {
	return createReactiveTextNode(parseS.atom.ID(), parseS.Get, render)
}

// TextValue is the zero-argument form of Text: it binds a reactive text node that renders
// the signal's value with default fmt formatting — the common case (a string signal, or any
// value whose fmt form is what you want) where no custom formatter is needed.
//
//	name := state.NewSignal("Ada")
//	h.Span(name.TextValue()) // no render func
func (parseS Signal[T]) TextValue() *Element {
	return parseS.Text(func(parseValue T) string { return fmt.Sprint(parseValue) })
}

// selectorSourceID lets a Signal be used as a [UseSelector] source.
func (parseS Signal[T]) selectorSourceID() string {
	return parseS.atom.ID()
}

// ReactiveRegionSourceIDs lets a Signal drive a ui.ReactiveRegion, so a subtree
// can re-render on signal changes without rerunning its owner.
func (parseS Signal[T]) ReactiveRegionSourceIDs() []string {
	if parseS.atom.ID() == "" {
		return nil
	}
	return []string{parseS.atom.ID()}
}

// reactiveSource is any reactive handle (Atom, Derived, Signal, ComputedSignal)
// usable as an explicit dependency of a [NewComputed].
type reactiveSource interface {
	selectorSourceID() string
}

// ComputedSignal is a read-only value derived from one or more reactive sources.
//
// Its dependencies are EXPLICIT — passed to [NewComputed] — so recomputation is
// predictable and never depends on a hidden tracking graph. [ComputedSignal.Get]
// is lazy (it recomputes from the live sources on read); pass the computed to
// ui.ReactiveRegion to render a subtree that updates fine-grained when any named
// source changes.
type ComputedSignal[T any] struct {
	compute   func() T
	sourceIDs []string
}

// NewComputed derives a value from compute, declaring the reactive sources it
// depends on. Dependencies are explicit by design: only changes to the named
// sources are guaranteed to refresh consumers bound through ui.ReactiveRegion.
//
//	first := state.NewSignal("Ada")
//	last := state.NewSignal("Lovelace")
//	full := state.NewComputed(func() string { return first.Get() + " " + last.Get() }, first, last)
func NewComputed[T any](parseCompute func() T, parseSources ...reactiveSource) ComputedSignal[T] {
	parseIDs := make([]string, 0, len(parseSources))
	parseSeen := make(map[string]struct{}, len(parseSources))
	for _, parseSource := range parseSources {
		if parseSource == nil {
			continue
		}
		parseID := parseSource.selectorSourceID()
		if parseID == "" {
			continue
		}
		if _, parseDup := parseSeen[parseID]; parseDup {
			continue
		}
		parseSeen[parseID] = struct{}{}
		parseIDs = append(parseIDs, parseID)
	}
	return ComputedSignal[T]{compute: parseCompute, sourceIDs: parseIDs}
}

// Get recomputes and returns the derived value from the live sources.
func (parseC ComputedSignal[T]) Get() T {
	if parseC.compute == nil {
		var parseZero T
		return parseZero
	}
	return parseC.compute()
}

// Peek returns the derived value; identical to [ComputedSignal.Get], named for
// parity with [Signal.Peek].
func (parseC ComputedSignal[T]) Peek() T {
	return parseC.Get()
}

// Text renders a fine-grained reactive text node for the computed value. It binds
// to the computed's first declared source; for a computed spanning several sources
// that must all refresh the text, prefer ui.ReactiveRegion with this computed as
// the source (which subscribes to every declared source id).
func (parseC ComputedSignal[T]) Text(render func(T) string) *Element {
	parseID := ""
	if len(parseC.sourceIDs) > 0 {
		parseID = parseC.sourceIDs[0]
	}
	return createReactiveTextNode(parseID, parseC.Get, render)
}

// TextValue is the zero-argument form of Text for a computed signal: it renders the computed
// value with default fmt formatting (see Signal.TextValue).
func (parseC ComputedSignal[T]) TextValue() *Element {
	return parseC.Text(func(parseValue T) string { return fmt.Sprint(parseValue) })
}

// selectorSourceID returns the computed's primary source id (empty when it has no
// declared sources), so a ComputedSignal can itself be a source of another.
func (parseC ComputedSignal[T]) selectorSourceID() string {
	if len(parseC.sourceIDs) == 0 {
		return ""
	}
	return parseC.sourceIDs[0]
}

// ReactiveRegionSourceIDs returns every declared source id, so ui.ReactiveRegion
// re-renders the region when ANY dependency of the computed changes.
func (parseC ComputedSignal[T]) ReactiveRegionSourceIDs() []string {
	if len(parseC.sourceIDs) == 0 {
		return nil
	}
	return append([]string(nil), parseC.sourceIDs...)
}
