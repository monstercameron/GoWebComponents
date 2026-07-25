// Package projection is v5's render-thread view of data owned by a worker
// (plan item P3.7).
//
// A projection is a slice of domain state made resident on the render thread so
// the UI can read it without asking anyone. That is the whole point: the §1.2
// filter keystroke must perform ZERO domain round-trips, because a round trip
// per keystroke reintroduces exactly the latency the off-thread architecture
// removed. Filtering, sorting, and windowing over a resident projection are
// local operations on local memory.
//
// The worker keeps the authority and publishes deltas (delta); this
// package applies them and answers questions. It never queries.
//
// Residency is bounded on purpose. Making the render thread hold an unbounded
// copy of a worker's data would trade a message-passing stall for a GC one —
// the risk M12 exists to bound — so Options.Resident caps it, and the cap is
// visible rather than silent.
package projection

import (
	"errors"
	"fmt"

	"github.com/monstercameron/GoWebComponents/v5/delta"
)

// Key identifies one row, re-exported from delta so a caller reading a
// projection does not have to import the publication engine as well.
type Key = delta.Key

// Op is one published delta operation.
type Op = delta.Op

// OpKind names one delta operation.
type OpKind = delta.OpKind

// The op kinds, re-exported.
//
// Aliases rather than new constants, so a kind produced by delta and a kind
// named through projection are the same value and switch statements over either
// stay exhaustive. Declaring separate constants would create two vocabularies
// for one concept and a conversion nobody would remember to write.
const (
	OpInsert = delta.OpInsert
	OpUpdate = delta.OpUpdate
	OpRemove = delta.OpRemove
	OpMove   = delta.OpMove
)

// Entry is one resident row.
type Entry[T any] struct {
	Key   Key
	Value T
}

// DefaultResident is the residency cap applied when Options.Resident is unset.
//
// The number comes from M12: it is the row count at which render-thread
// projection memory and its GC cost stay inside the frame budget for a typical
// row. See memory_test.go, which both measures it and fails if this default
// drifts away from what was measured.
const DefaultResident = 20000

// Options configure a projection.
type Options struct {
	// Resident caps how many rows are held on the render thread. Zero uses
	// DefaultResident. A negative value means unbounded, which is supported but
	// deliberately awkward to ask for.
	Resident int
}

// Decoder turns a published payload into a value.
//
// Decoding is the caller's because the payload format is the caller's; the
// engine that produced it treats payloads as opaque bytes.
type Decoder[T any] func(parsePayload []byte) (T, error)

// Projection is a resident, ordered view of published rows.
//
// Not safe for concurrent use. It belongs to the render thread, which is
// single-threaded by construction in wasm.
type Projection[T any] struct {
	decode   Decoder[T]
	resident int

	entries    []Entry[T]
	indexByKey map[Key]int
	// indexStale marks the key index as needing a rebuild after a structural
	// change. Deferring the rebuild to the first lookup after a batch is what
	// keeps applying a batch of ops O(batch + N) instead of O(batch * N).
	indexStale bool

	// droppedByResidency counts rows refused because the cap was reached, so a
	// truncated view is visible rather than silently short.
	droppedByResidency int

	// ready reports that initial contents have arrived — see Ready(), and note
	// that it is deliberately NOT derived from the row count.
	ready bool
	// complete reports that the resident rows are the whole dataset rather than
	// a server-sent prefix.
	complete bool
}

// New creates an empty projection.
func New[T any](parseDecode Decoder[T], parseOptions Options) (*Projection[T], error) {
	if parseDecode == nil {
		return nil, errors.New("projection: a decoder is required")
	}
	parseResident := parseOptions.Resident
	if parseResident == 0 {
		parseResident = DefaultResident
	}
	return &Projection[T]{
		decode:     parseDecode,
		resident:   parseResident,
		indexByKey: make(map[Key]int),
	}, nil
}

// Len reports how many rows are resident.
func (parseProjection *Projection[T]) Len() int {
	if parseProjection == nil {
		return 0
	}
	return len(parseProjection.entries)
}

// DroppedByResidency reports how many rows were refused because the residency
// cap was reached.
//
// Non-zero means the projection is a prefix of what the worker published, and
// any local count or aggregate computed from it is a lower bound. Surfacing
// this is the difference between a bounded view and a wrong one.
func (parseProjection *Projection[T]) DroppedByResidency() int {
	if parseProjection == nil {
		return 0
	}
	return parseProjection.droppedByResidency
}

// Apply applies published ops in order.
//
// Ops from one publish are a sequence, not a set: an insert may anchor on a key
// an earlier op in the same batch created. Applying them out of order, or
// filtering them, breaks that.
func (parseProjection *Projection[T]) Apply(parseOps []Op) error {
	if parseProjection == nil {
		return errors.New("projection: projection is nil")
	}

	for _, parseOp := range parseOps {
		var parseErr error
		switch parseOp.Kind {
		case delta.OpInsert:
			parseErr = parseProjection.applyInsert(parseOp)
		case delta.OpUpdate:
			parseErr = parseProjection.applyUpdate(parseOp)
		case delta.OpRemove:
			parseProjection.applyRemove(parseOp)
		case delta.OpMove:
			parseErr = parseProjection.applyMove(parseOp)
		default:
			parseErr = fmt.Errorf("projection: unknown op kind %d for key %q", parseOp.Kind, parseOp.Key)
		}
		if parseErr != nil {
			return parseErr
		}
	}
	return nil
}

func (parseProjection *Projection[T]) applyInsert(parseOp Op) error {
	if parseProjection.resident >= 0 && len(parseProjection.entries) >= parseProjection.resident {
		// Refused rather than evicting something else: the projection is an
		// ordered prefix, and dropping an arbitrary row to make room would make
		// the resident window neither a prefix nor a window.
		parseProjection.droppedByResidency++
		return nil
	}

	parseValue, parseDecodeErr := parseProjection.decode(parseOp.Payload)
	if parseDecodeErr != nil {
		return fmt.Errorf("projection: decoding %q: %w", parseOp.Key, parseDecodeErr)
	}

	parsePosition, parseAnchorErr := parseProjection.positionAfter(parseOp.AfterKey, parseOp.Key)
	if parseAnchorErr != nil {
		return parseAnchorErr
	}

	parseProjection.entries = append(parseProjection.entries, Entry[T]{})
	copy(parseProjection.entries[parsePosition+1:], parseProjection.entries[parsePosition:])
	parseProjection.entries[parsePosition] = Entry[T]{Key: parseOp.Key, Value: parseValue}
	parseProjection.indexStale = true
	return nil
}

func (parseProjection *Projection[T]) applyUpdate(parseOp Op) error {
	parsePosition, hasKey := parseProjection.lookup(parseOp.Key)
	if !hasKey {
		// A row outside the resident window legitimately receives updates it
		// cannot apply. That is not an error — but it is also not silent, since
		// DroppedByResidency already tells the caller the view is a prefix.
		if parseProjection.droppedByResidency > 0 {
			return nil
		}
		return fmt.Errorf("projection: update for unknown key %q", parseOp.Key)
	}

	parseValue, parseDecodeErr := parseProjection.decode(parseOp.Payload)
	if parseDecodeErr != nil {
		return fmt.Errorf("projection: decoding %q: %w", parseOp.Key, parseDecodeErr)
	}
	// Only the value changes; the key and position are untouched, so the index
	// stays valid and an update never triggers a rebuild. This is the path a
	// keystroke-driven re-publish takes, so it is the one that must stay cheap.
	parseProjection.entries[parsePosition].Value = parseValue
	return nil
}

func (parseProjection *Projection[T]) applyRemove(parseOp Op) {
	parsePosition, hasKey := parseProjection.lookup(parseOp.Key)
	if !hasKey {
		// Removing something not resident is a no-op, so a projection that
		// dropped rows at the cap still converges.
		return
	}
	parseProjection.entries = append(parseProjection.entries[:parsePosition], parseProjection.entries[parsePosition+1:]...)
	parseProjection.indexStale = true

	if parseProjection.droppedByResidency > 0 {
		// Room freed up, but the row that would fill it is not being resent. The
		// count stays as a standing signal that the view is incomplete rather
		// than being decremented into looking healthy.
		_ = parseProjection.droppedByResidency
	}
}

func (parseProjection *Projection[T]) applyMove(parseOp Op) error {
	parsePosition, hasKey := parseProjection.lookup(parseOp.Key)
	if !hasKey {
		return nil
	}
	parseEntry := parseProjection.entries[parsePosition]
	parseProjection.entries = append(parseProjection.entries[:parsePosition], parseProjection.entries[parsePosition+1:]...)
	parseProjection.indexStale = true

	// positionAfter may rebuild the index, which marks it clean. Every mutation
	// below therefore has to mark it stale AGAIN — a rebuild that happens between
	// two halves of one structural change leaves the index describing neither.
	parseTarget, parseAnchorErr := parseProjection.positionAfter(parseOp.AfterKey, parseOp.Key)
	if parseAnchorErr != nil {
		// Put it back rather than losing the row. A move to an unknown anchor is
		// a protocol error, and dropping data is a worse response than reporting
		// one.
		parseProjection.entries = append(parseProjection.entries, Entry[T]{})
		copy(parseProjection.entries[parsePosition+1:], parseProjection.entries[parsePosition:])
		parseProjection.entries[parsePosition] = parseEntry
		parseProjection.indexStale = true
		return parseAnchorErr
	}

	parseProjection.entries = append(parseProjection.entries, Entry[T]{})
	copy(parseProjection.entries[parseTarget+1:], parseProjection.entries[parseTarget:])
	parseProjection.entries[parseTarget] = parseEntry
	parseProjection.indexStale = true
	return nil
}

// positionAfter resolves an anchor to the insertion position following it.
func (parseProjection *Projection[T]) positionAfter(parseAfterKey Key, parseKey Key) (int, error) {
	if parseAfterKey == "" {
		return 0, nil
	}
	parsePosition, hasAnchor := parseProjection.lookup(parseAfterKey)
	if !hasAnchor {
		if parseProjection.droppedByResidency > 0 {
			// The anchor fell outside the resident window; appending keeps the
			// resident rows in published order relative to each other.
			return len(parseProjection.entries), nil
		}
		return 0, fmt.Errorf("projection: cannot place %q after %q, which is not resident", parseKey, parseAfterKey)
	}
	return parsePosition + 1, nil
}

// lookup finds a key's position, rebuilding the index if a structural change
// invalidated it.
func (parseProjection *Projection[T]) lookup(parseKey Key) (int, bool) {
	if parseProjection.indexStale {
		parseProjection.rebuildIndex()
	}
	parsePosition, hasKey := parseProjection.indexByKey[parseKey]
	return parsePosition, hasKey
}

func (parseProjection *Projection[T]) rebuildIndex() {
	clear(parseProjection.indexByKey)
	for parsePosition, parseEntry := range parseProjection.entries {
		parseProjection.indexByKey[parseEntry.Key] = parsePosition
	}
	parseProjection.indexStale = false
}

// ------------------------------------------------------------ local queries

// Rows returns the resident rows in published order.
//
// The returned slice aliases the projection's own storage: it is read-only to
// the caller, and the next Apply may change it. Copying on every read would
// allocate a full projection per frame, which is the cost this package exists to
// avoid.
func (parseProjection *Projection[T]) Rows() []Entry[T] {
	if parseProjection == nil {
		return nil
	}
	return parseProjection.entries
}

// Get returns one row by key.
func (parseProjection *Projection[T]) Get(parseKey Key) (T, bool) {
	var parseZero T
	if parseProjection == nil {
		return parseZero, false
	}
	parsePosition, hasKey := parseProjection.lookup(parseKey)
	if !hasKey {
		return parseZero, false
	}
	return parseProjection.entries[parsePosition].Value, true
}

// Filter returns rows matching a predicate, evaluated locally.
//
// This is P3.7 criterion (c): the §1.2 filter keystroke performs zero domain
// round-trips. Nothing here sends a message, and that is asserted by message
// count in the tests rather than left as a claim.
func (parseProjection *Projection[T]) Filter(parsePredicate func(T) bool) []Entry[T] {
	if parseProjection == nil || parsePredicate == nil {
		return nil
	}
	var parseMatches []Entry[T]
	for _, parseEntry := range parseProjection.entries {
		if parsePredicate(parseEntry.Value) {
			parseMatches = append(parseMatches, parseEntry)
		}
	}
	return parseMatches
}

// Window returns a contiguous slice of rows, for virtualized rendering.
//
// Out-of-range bounds are clamped rather than refused: a scroll position is
// naturally transient and can outrun a projection that just shrank, and
// panicking on a stale scroll offset would be a worse answer than an empty
// window.
func (parseProjection *Projection[T]) Window(parseOffset int, parseCount int) []Entry[T] {
	if parseProjection == nil || parseCount <= 0 {
		return nil
	}
	parseOffset = max(parseOffset, 0)
	if parseOffset >= len(parseProjection.entries) {
		return nil
	}
	parseEnd := min(parseOffset+parseCount, len(parseProjection.entries))
	return parseProjection.entries[parseOffset:parseEnd]
}

// Reset drops every resident row, for a projection whose subject changed.
func (parseProjection *Projection[T]) Reset() {
	if parseProjection == nil {
		return
	}
	parseProjection.entries = nil
	clear(parseProjection.indexByKey)
	parseProjection.indexStale = false
	parseProjection.droppedByResidency = 0
	// A reset projection is unloaded again: its subject changed, and whatever
	// made it ready described the old subject. Leaving ready set would render
	// "no results" for the new one before anything was fetched.
	parseProjection.ready = false
	parseProjection.complete = false
}
