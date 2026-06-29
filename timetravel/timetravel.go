// Package timetravel is the snapshot step-back replay engine behind GoWebComponents'
// time-travel devtools (C4) and undo/redo (FB6) — one engine, two consumers. It records
// immutable state snapshots over time and lets you step back (undo), step forward (redo),
// or scrub to any point, with a bounded ring so history never grows without limit.
//
// It is pure and deterministic: it owns no clock, no DOM, and no runtime, so it unit-tests
// trivially and the devtools panel is just a thin view over Labels()/ScrubTo. State is held
// by value, so snapshots are immutable as long as T is (use value types or copy on Record).
package timetravel

import "sync"

// Snapshot is one recorded point in history: a human-facing Label and the state value.
type Snapshot[T any] struct {
	Label string
	State T
}

// History is a bounded, navigable timeline of state snapshots. Safe for concurrent use.
type History[T any] struct {
	mu       sync.Mutex
	entries  []Snapshot[T]
	cursor   int
	capacity int
}

// New creates a history seeded with an initial snapshot. capacity bounds the number of
// retained snapshots (the oldest are evicted past it); a capacity <= 0 means unbounded.
func New[T any](parseCapacity int, parseInitial T) *History[T] {
	return &History[T]{
		entries:  []Snapshot[T]{{Label: "initial", State: parseInitial}},
		cursor:   0,
		capacity: parseCapacity,
	}
}

// Record appends a new snapshot as the current state. If the cursor is not at the end
// (because of prior Undo calls), the redo branch ahead of it is discarded first — the
// standard undo/redo rule — then the oldest snapshots are evicted to honor capacity.
func (parseH *History[T]) Record(parseLabel string, parseState T) {
	parseH.mu.Lock()
	defer parseH.mu.Unlock()

	parseH.entries = append(parseH.entries[:parseH.cursor+1], Snapshot[T]{Label: parseLabel, State: parseState})
	parseH.cursor = len(parseH.entries) - 1
	parseH.evictLocked()
}

// evictLocked drops the oldest snapshots when over capacity, keeping the cursor valid.
func (parseH *History[T]) evictLocked() {
	if parseH.capacity <= 0 {
		return
	}
	parseOverflow := len(parseH.entries) - parseH.capacity
	if parseOverflow <= 0 {
		return
	}
	parseH.entries = append([]Snapshot[T](nil), parseH.entries[parseOverflow:]...)
	parseH.cursor -= parseOverflow
	if parseH.cursor < 0 {
		parseH.cursor = 0
	}
}

// Current returns the state at the cursor.
func (parseH *History[T]) Current() T {
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	return parseH.entries[parseH.cursor].State
}

// CanUndo reports whether there is an earlier snapshot to step back to.
func (parseH *History[T]) CanUndo() bool {
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	return parseH.cursor > 0
}

// CanRedo reports whether there is a later snapshot to step forward to.
func (parseH *History[T]) CanRedo() bool {
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	return parseH.cursor < len(parseH.entries)-1
}

// Undo steps back one snapshot and returns it; ok is false at the start of history.
func (parseH *History[T]) Undo() (T, bool) {
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	if parseH.cursor <= 0 {
		return parseH.entries[parseH.cursor].State, false
	}
	parseH.cursor--
	return parseH.entries[parseH.cursor].State, true
}

// Redo steps forward one snapshot and returns it; ok is false at the end of history.
func (parseH *History[T]) Redo() (T, bool) {
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	if parseH.cursor >= len(parseH.entries)-1 {
		return parseH.entries[parseH.cursor].State, false
	}
	parseH.cursor++
	return parseH.entries[parseH.cursor].State, true
}

// ScrubTo jumps the cursor to an arbitrary index (the scrubber/timeline in the devtools
// panel), returning that state; ok is false if the index is out of range.
func (parseH *History[T]) ScrubTo(parseIndex int) (T, bool) {
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	if parseIndex < 0 || parseIndex >= len(parseH.entries) {
		var parseZero T
		return parseZero, false
	}
	parseH.cursor = parseIndex
	return parseH.entries[parseIndex].State, true
}

// Cursor returns the current position in history.
func (parseH *History[T]) Cursor() int {
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	return parseH.cursor
}

// Len returns the number of retained snapshots.
func (parseH *History[T]) Len() int {
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	return len(parseH.entries)
}

// Labels returns the snapshot labels in order — the timeline the devtools scrubber renders.
func (parseH *History[T]) Labels() []string {
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	parseLabels := make([]string, len(parseH.entries))
	for parseIndex, parseEntry := range parseH.entries {
		parseLabels[parseIndex] = parseEntry.Label
	}
	return parseLabels
}

// Snapshots returns a copy of every retained snapshot (label + state) in order, WITHOUT moving the
// cursor — so a devtools panel or undo-list UI can read the whole timeline at once instead of
// looping ScrubTo (which mutates the cursor). The returned slice is a copy; mutating it is safe.
func (parseH *History[T]) Snapshots() []Snapshot[T] {
	parseH.mu.Lock()
	defer parseH.mu.Unlock()
	parseOut := make([]Snapshot[T], len(parseH.entries))
	copy(parseOut, parseH.entries)
	return parseOut
}
