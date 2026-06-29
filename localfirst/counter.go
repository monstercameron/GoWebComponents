package localfirst

import "maps"

// Counter is a positive-negative counter CRDT (PN-Counter). Unlike the last-write-wins
// Register (where two replicas that both edit the same key keep only one write), every
// replica's increments and decrements SURVIVE a merge: each replica tracks its own positive
// and negative tallies, Value is their global sum, and Merge takes the per-replica maximum —
// which is commutative, associative, and idempotent, so all replicas converge to the same
// count regardless of the order merges arrive. This is the op-based answer to "concurrent
// same-field edits must not silently last-write-wins" for the counter case (likes, votes,
// inventory, …). Safe for concurrent use is the caller's responsibility (guard with a mutex
// if shared across goroutines); the type is value-semantic per replica.
type Counter struct {
	pos map[string]uint64
	neg map[string]uint64
}

// NewCounter creates a zero counter.
func NewCounter() *Counter {
	return &Counter{pos: map[string]uint64{}, neg: map[string]uint64{}}
}

// Inc adds by to the calling replica's positive tally (a local, conflict-free increment).
func (parseC *Counter) Inc(parseReplica string, parseBy uint64) {
	parseC.pos[parseReplica] += parseBy
}

// Dec adds by to the calling replica's negative tally.
func (parseC *Counter) Dec(parseReplica string, parseBy uint64) {
	parseC.neg[parseReplica] += parseBy
}

// Value returns the current global count: the sum of every replica's increments minus the
// sum of every replica's decrements.
func (parseC *Counter) Value() int64 {
	var parsePos, parseNeg uint64
	for _, parseV := range parseC.pos {
		parsePos += parseV
	}
	for _, parseV := range parseC.neg {
		parseNeg += parseV
	}
	return int64(parsePos) - int64(parseNeg)
}

// Merge converges this counter with another by taking the per-replica maximum of each
// tally. It is commutative/associative/idempotent, so merging in any order — even merging
// the same peer twice — yields the same value, and no replica's increments are ever lost.
func (parseC *Counter) Merge(parseOther *Counter) {
	if parseOther == nil {
		return
	}
	for parseReplica, parseValue := range parseOther.pos {
		if parseValue > parseC.pos[parseReplica] {
			parseC.pos[parseReplica] = parseValue
		}
	}
	for parseReplica, parseValue := range parseOther.neg {
		if parseValue > parseC.neg[parseReplica] {
			parseC.neg[parseReplica] = parseValue
		}
	}
}

// CounterState is the JSON-serializable state of a Counter, for persistence/transport.
type CounterState struct {
	Pos map[string]uint64 `json:"pos"`
	Neg map[string]uint64 `json:"neg"`
}

// Export captures the counter's state.
func (parseC *Counter) Export() CounterState {
	return CounterState{Pos: maps.Clone(parseC.pos), Neg: maps.Clone(parseC.neg)}
}

// RestoreCounter rebuilds a counter from persisted state.
func RestoreCounter(parseState CounterState) *Counter {
	parseCounter := NewCounter()
	if parseState.Pos != nil {
		maps.Copy(parseCounter.pos, parseState.Pos)
	}
	if parseState.Neg != nil {
		maps.Copy(parseCounter.neg, parseState.Neg)
	}
	return parseCounter
}
