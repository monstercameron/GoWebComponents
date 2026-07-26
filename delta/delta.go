// Package delta is v5's delta publication engine (plan item P3.15a).
//
// A projection published to the render thread must not be re-sent whenever
// anything about it changes. At 20,000 rows that is a multi-megabyte structured
// clone per keystroke, which is precisely the stall the off-thread model exists
// to remove. This engine turns a new state of the world into the ops needed to
// get there from the old one.
//
// Two design decisions carry the guarantee, and both are about what an op says:
//
//  1. Ops name a key and an ANCHOR ("after this other key"), never an integer
//     index. Index-based ops force the publisher to maintain a key→index map,
//     and every insert then shifts every following index — O(N) bookkeeping per
//     change, which is the cost being avoided. Anchors are O(1) to produce and
//     the consumer already knows where its own keys are.
//
//  2. Change detection compares VERSIONS, not payloads. So the engine's resident
//     index is one integer per key rather than a copy of the data, which is what
//     M11 bounds — see memory_test.go.
//
// The engine is not safe for concurrent use; it belongs to the worker that owns
// the projection.
package delta

import "fmt"

// Key identifies one row within a projection.
type Key string

// Row is one published row.
//
// Payload is opaque: the engine never inspects it, which is what lets the same
// engine publish table rows, chart points, or search results without learning
// anything about them.
type Row struct {
	Key Key
	// Version changes whenever Payload changes. It is the producer's
	// responsibility and the engine's only means of detecting an update — a
	// producer that reuses a version for changed data will publish a stale row,
	// and one that bumps it needlessly will publish a redundant update.
	Version uint64
	Payload []byte
}

// OpKind names one delta operation.
type OpKind uint8

const (
	// OpInsert adds a key that was not present.
	OpInsert OpKind = iota
	// OpUpdate replaces the payload of a key that was already present.
	OpUpdate
	// OpRemove drops a key.
	OpRemove
	// OpMove relocates a key without changing its payload.
	OpMove
)

func (parseKind OpKind) String() string {
	switch parseKind {
	case OpInsert:
		return "insert"
	case OpUpdate:
		return "update"
	case OpRemove:
		return "remove"
	case OpMove:
		return "move"
	default:
		return fmt.Sprintf("unknown-op(%d)", uint8(parseKind))
	}
}

// Op is one delta operation.
type Op struct {
	Kind OpKind
	Key  Key
	// AfterKey anchors an insert or a move. An empty AfterKey means the head of
	// the projection. Unused by updates and removes.
	AfterKey Key
	// Payload is carried by inserts and updates only. Removes and moves do not
	// resend data, which is most of the saving on a reorder.
	Payload []byte
}

// Engine maintains a published projection and computes deltas against it.
type Engine struct {
	// order is the published key order. Needed to compute anchors and to detect
	// reordering; it holds keys, never payloads.
	order []Key
	// versionByKey is the resident index: one integer per key.
	versionByKey map[Key]uint64
}

// New creates an empty engine.
func New() *Engine {
	return &Engine{versionByKey: make(map[Key]uint64)}
}

// Len reports how many rows are currently published.
func (parseEngine *Engine) Len() int {
	if parseEngine == nil {
		return 0
	}
	return len(parseEngine.order)
}

// Keys returns a copy of the published key order, for diagnostics and tests.
func (parseEngine *Engine) Keys() []Key {
	if parseEngine == nil {
		return nil
	}
	parseKeys := make([]Key, len(parseEngine.order))
	copy(parseKeys, parseEngine.order)
	return parseKeys
}

// Version reports a published key's version, and whether it is published.
func (parseEngine *Engine) Version(parseKey Key) (uint64, bool) {
	if parseEngine == nil {
		return 0, false
	}
	parseVersion, hasKey := parseEngine.versionByKey[parseKey]
	return parseVersion, hasKey
}

// Publish computes the ops that turn the published projection into rows, and
// adopts rows as the new published state.
//
// Reading the whole new state is O(N) by necessity — the engine cannot know what
// changed without looking. What P3.15a requires, and what this delivers, is that
// the OPS are O(change): a 20,000-row projection with one edited row publishes
// exactly one op, not 20,000. A producer that can describe its own changes
// should use Apply instead and skip the scan entirely.
//
// Duplicate keys are rejected rather than deduplicated. A projection with two
// rows claiming one key has no correct delta — the second silently winning would
// make the consumer's state depend on iteration order.
func (parseEngine *Engine) Publish(parseRows []Row) ([]Op, error) {
	if parseEngine == nil {
		return nil, fmt.Errorf("delta: engine is nil")
	}

	parseNewVersionByKey := make(map[Key]uint64, len(parseRows))
	parseNewOrder := make([]Key, 0, len(parseRows))
	parseRowByKey := make(map[Key]Row, len(parseRows))
	for _, parseRow := range parseRows {
		if parseRow.Key == "" {
			return nil, fmt.Errorf("delta: a published row has no key")
		}
		if _, hasKey := parseNewVersionByKey[parseRow.Key]; hasKey {
			return nil, fmt.Errorf("delta: key %q appears more than once in one publish", parseRow.Key)
		}
		parseNewVersionByKey[parseRow.Key] = parseRow.Version
		parseRowByKey[parseRow.Key] = parseRow
		parseNewOrder = append(parseNewOrder, parseRow.Key)
	}

	var parseOps []Op

	// Removals first. Doing them before anything else means later anchors never
	// reference a key that is on its way out.
	for _, parseKey := range parseEngine.order {
		if _, hasKey := parseNewVersionByKey[parseKey]; !hasKey {
			parseOps = append(parseOps, Op{Kind: OpRemove, Key: parseKey})
		}
	}

	// Survivors, in their OLD relative order, is what move minimization works on.
	parseOldPositionByKey := make(map[Key]int, len(parseEngine.order))
	for parsePosition, parseKey := range parseEngine.order {
		parseOldPositionByKey[parseKey] = parsePosition
	}

	parseSurvivingOldPositions := make([]int, 0, len(parseNewOrder))
	parseSurvivingKeys := make([]Key, 0, len(parseNewOrder))
	for _, parseKey := range parseNewOrder {
		if parsePosition, hasKey := parseOldPositionByKey[parseKey]; hasKey {
			parseSurvivingOldPositions = append(parseSurvivingOldPositions, parsePosition)
			parseSurvivingKeys = append(parseSurvivingKeys, parseKey)
		}
	}

	// Keys already in relative order need no move. The longest increasing
	// subsequence of their old positions is the largest such set, so moving
	// everything else is the minimum number of moves. Emitting a move per key
	// instead would make a single row dragged to the top cost N ops.
	parseStayingKeys := make(map[Key]bool, len(parseSurvivingKeys))
	for _, parseOffset := range longestIncreasingSubsequence(parseSurvivingOldPositions) {
		parseStayingKeys[parseSurvivingKeys[parseOffset]] = true
	}

	// Walk the new order, emitting inserts, moves, and updates with anchors that
	// are already correct because every earlier key is in place.
	var parseAnchor Key
	for _, parseKey := range parseNewOrder {
		parseRow := parseRowByKey[parseKey]
		parseOldVersion, hadKey := parseEngine.versionByKey[parseKey]

		switch {
		case !hadKey:
			parseOps = append(parseOps, Op{Kind: OpInsert, Key: parseKey, AfterKey: parseAnchor, Payload: parseRow.Payload})
		case !parseStayingKeys[parseKey]:
			parseOps = append(parseOps, Op{Kind: OpMove, Key: parseKey, AfterKey: parseAnchor})
			// A moved row whose data also changed needs both: the move carries no
			// payload by design, so the update is not redundant.
			if parseOldVersion != parseRow.Version {
				parseOps = append(parseOps, Op{Kind: OpUpdate, Key: parseKey, Payload: parseRow.Payload})
			}
		case parseOldVersion != parseRow.Version:
			parseOps = append(parseOps, Op{Kind: OpUpdate, Key: parseKey, Payload: parseRow.Payload})
		}
		parseAnchor = parseKey
	}

	parseEngine.order = parseNewOrder
	parseEngine.versionByKey = parseNewVersionByKey
	return parseOps, nil
}

// ChangeKind names one producer-described change.
type ChangeKind uint8

const (
	// ChangeUpsert inserts a key or updates it if already present.
	ChangeUpsert ChangeKind = iota
	// ChangeRemove drops a key.
	ChangeRemove
)

// Change is one edit a producer already knows about.
type Change struct {
	Kind ChangeKind
	Row  Row
	// AfterKey anchors an insert. Empty means the head. Ignored when the key is
	// already published, since an upsert never relocates a row — use Publish if
	// the order genuinely changed.
	AfterKey Key
}

// Apply publishes changes the producer already knows about, in O(change).
//
// This is the path that makes P3.15a's guarantee about TIME as well as ops:
// nothing here touches a row that was not named. A producer that knows it
// inserted one row — a command handler, a SQLite hook — should use this rather
// than re-reading 20,000 rows so the engine can rediscover the same fact.
//
// The cost is that the producer is now responsible for correctness: a missed
// change leaves the consumer permanently stale, with nothing to detect it. A
// producer that cannot guarantee completeness should use Publish, which pays
// O(N) to be sure.
func (parseEngine *Engine) Apply(parseChanges []Change) ([]Op, error) {
	if parseEngine == nil {
		return nil, fmt.Errorf("delta: engine is nil")
	}

	var parseOps []Op
	for _, parseChange := range parseChanges {
		if parseChange.Row.Key == "" {
			return nil, fmt.Errorf("delta: a change has no key")
		}
		parseKey := parseChange.Row.Key

		switch parseChange.Kind {
		case ChangeRemove:
			if _, hasKey := parseEngine.versionByKey[parseKey]; !hasKey {
				// Removing an absent key is a no-op rather than an error: a
				// producer replaying a change log should converge, not fail.
				continue
			}
			delete(parseEngine.versionByKey, parseKey)
			parseEngine.removeFromOrder(parseKey)
			parseOps = append(parseOps, Op{Kind: OpRemove, Key: parseKey})

		case ChangeUpsert:
			parseOldVersion, hadKey := parseEngine.versionByKey[parseKey]
			if !hadKey {
				parseInsertErr := parseEngine.insertAfter(parseKey, parseChange.AfterKey)
				if parseInsertErr != nil {
					return nil, parseInsertErr
				}
				parseEngine.versionByKey[parseKey] = parseChange.Row.Version
				parseOps = append(parseOps, Op{
					Kind: OpInsert, Key: parseKey, AfterKey: parseChange.AfterKey, Payload: parseChange.Row.Payload,
				})
				continue
			}
			if parseOldVersion == parseChange.Row.Version {
				// Same version means same data by the producer's own contract, so
				// publishing an update would send bytes to change nothing.
				continue
			}
			parseEngine.versionByKey[parseKey] = parseChange.Row.Version
			parseOps = append(parseOps, Op{Kind: OpUpdate, Key: parseKey, Payload: parseChange.Row.Payload})

		default:
			return nil, fmt.Errorf("delta: unknown change kind %d for key %q", parseChange.Kind, parseKey)
		}
	}
	return parseOps, nil
}

// insertAfter places a key in the published order.
//
// Appending — the bulk-import case — is O(1). Inserting into the middle moves
// the tail of a key slice, which is a memmove of pointers rather than of data
// and does not affect the op count. Making this O(1) too would need a linked
// order, whose per-node overhead would cost more against M11 than the memmove
// costs in time.
func (parseEngine *Engine) insertAfter(parseKey Key, parseAfterKey Key) error {
	if parseAfterKey == "" {
		parseEngine.order = append([]Key{parseKey}, parseEngine.order...)
		return nil
	}
	for parsePosition, parseExisting := range parseEngine.order {
		if parseExisting == parseAfterKey {
			parseEngine.order = append(parseEngine.order, "")
			copy(parseEngine.order[parsePosition+2:], parseEngine.order[parsePosition+1:])
			parseEngine.order[parsePosition+1] = parseKey
			return nil
		}
	}
	// An anchor that is not published would place the row nowhere. Guessing —
	// appending, say — would put it somewhere the producer did not ask for and
	// the consumer would never learn the order was wrong.
	return fmt.Errorf("delta: cannot insert %q after %q, which is not published", parseKey, parseAfterKey)
}

func (parseEngine *Engine) removeFromOrder(parseKey Key) {
	for parsePosition, parseExisting := range parseEngine.order {
		if parseExisting == parseKey {
			// Release the slot the shift vacates rather than leaving the old
			// last key in it past the new length.
			copy(parseEngine.order[parsePosition:], parseEngine.order[parsePosition+1:])
			parseEngine.order[len(parseEngine.order)-1] = ""
			parseEngine.order = parseEngine.order[:len(parseEngine.order)-1]
			return
		}
	}
}

// Reset drops all published state, so the next Publish emits a full insert set.
//
// For a projection whose subject changed entirely — a different query, a
// different table — where a diff against the previous subject would emit a
// remove-everything/insert-everything pair anyway, at twice the size.
func (parseEngine *Engine) Reset() {
	if parseEngine == nil {
		return
	}
	parseEngine.order = nil
	parseEngine.versionByKey = make(map[Key]uint64)
}

// longestIncreasingSubsequence returns the OFFSETS into values that form a
// longest strictly increasing subsequence.
//
// Patience sorting, O(n log n). This is what makes a reorder cost the minimum
// number of moves: everything in the subsequence is already in relative order
// and can stay put.
func longestIncreasingSubsequence(parseValues []int) []int {
	if len(parseValues) == 0 {
		return nil
	}

	// tailOffsetByLength[l] is the offset of the smallest tail of an increasing
	// subsequence of length l+1 found so far.
	parseTailOffsetByLength := make([]int, 0, len(parseValues))
	parsePredecessor := make([]int, len(parseValues))
	for parseOffset := range parsePredecessor {
		parsePredecessor[parseOffset] = -1
	}

	for parseOffset, parseValue := range parseValues {
		parseLow, parseHigh := 0, len(parseTailOffsetByLength)
		for parseLow < parseHigh {
			parseMid := (parseLow + parseHigh) / 2
			if parseValues[parseTailOffsetByLength[parseMid]] < parseValue {
				parseLow = parseMid + 1
			} else {
				parseHigh = parseMid
			}
		}
		if parseLow > 0 {
			parsePredecessor[parseOffset] = parseTailOffsetByLength[parseLow-1]
		}
		if parseLow == len(parseTailOffsetByLength) {
			parseTailOffsetByLength = append(parseTailOffsetByLength, parseOffset)
		} else {
			parseTailOffsetByLength[parseLow] = parseOffset
		}
	}

	parseResult := make([]int, len(parseTailOffsetByLength))
	parseCursor := parseTailOffsetByLength[len(parseTailOffsetByLength)-1]
	for parsePosition := len(parseResult) - 1; parsePosition >= 0; parsePosition-- {
		parseResult[parsePosition] = parseCursor
		parseCursor = parsePredecessor[parseCursor]
	}
	return parseResult
}

// ------------------------------------------------------------ hot reload

// State is an engine's publication index, extracted for transfer across a
// worker reload (plan item P3.14).
//
// It is the index and nothing else: keys, their order, and their versions. No
// payloads, because the engine never held any — which is what makes carrying
// this across a reload cheap enough to be worth doing.
type State struct {
	// Order is the published key order.
	Order []Key `json:"order,omitempty"`
	// Versions parallels Order.
	//
	// Parallel arrays rather than a map so the JSON is compact and the order is
	// carried by the structure rather than needing a separate sort on restore.
	Versions []uint64 `json:"versions,omitempty"`
}

// ExportState captures the engine's publication index.
func (parseEngine *Engine) ExportState() State {
	if parseEngine == nil {
		return State{}
	}
	parseState := State{
		Order:    make([]Key, len(parseEngine.order)),
		Versions: make([]uint64, len(parseEngine.order)),
	}
	copy(parseState.Order, parseEngine.order)
	for parseIndex, parseKey := range parseEngine.order {
		parseState.Versions[parseIndex] = parseEngine.versionByKey[parseKey]
	}
	return parseState
}

// Restore rebuilds an engine from an exported index.
//
// This is what makes P3.14's guarantee possible. Without it, a reloaded worker
// starts with an empty engine and its first publish emits an insert for every
// row — a full re-send of a 20,000-row projection across the boundary, on a code
// edit. With it, the first publish after a reload emits only what actually
// changed, which for an edit that changed no data is nothing at all.
func Restore(parseState State) (*Engine, error) {
	if len(parseState.Order) != len(parseState.Versions) {
		return nil, fmt.Errorf("delta: restored state has %d keys and %d versions",
			len(parseState.Order), len(parseState.Versions))
	}

	parseEngine := New()
	parseEngine.order = make([]Key, 0, len(parseState.Order))
	for parseIndex, parseKey := range parseState.Order {
		if parseKey == "" {
			return nil, fmt.Errorf("delta: restored state has an empty key at %d", parseIndex)
		}
		if _, hasKey := parseEngine.versionByKey[parseKey]; hasKey {
			// A duplicate would make the restored order ambiguous and the next
			// diff wrong in a way nothing downstream could detect.
			return nil, fmt.Errorf("delta: restored state repeats key %q", parseKey)
		}
		parseEngine.order = append(parseEngine.order, parseKey)
		parseEngine.versionByKey[parseKey] = parseState.Versions[parseIndex]
	}
	return parseEngine, nil
}
