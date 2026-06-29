// Package localfirst is GoWebComponents' built-in local-first sync engine (FC1). It gives
// a client an optimistic, offline-capable local store that converges with a server-
// authoritative store the moment connectivity returns — the model behind Zero / Electric /
// TanStack DB, in pure Go on both sides.
//
// The core is a last-write-wins register per key (an LWW-Register CRDT). Every write
// carries a logical Clock; conflicts are resolved by the SAME deterministic rule on every
// replica and on the server, so all participants converge to one state regardless of the
// order in which offline edits arrive. A Replica applies writes locally at once (optimistic
// UI), keeps unsynced writes in a durable pending log (offline), and converges via Merge;
// an Authority is the server-side store. Mutation and Record are JSON-serializable, so they
// ride the //gwc:server transport (serverfn) unchanged.
//
// This is the convergence engine. The query-driven "shapes" (`//gwc:sync`) and the
// db/sqlite + kvstate persistence adapters layer on top of it; kvstate supplies the
// single-replica conflict resolver, localfirst supplies the multi-replica protocol.
package localfirst

import (
	"maps"
	"sort"
	"sync"
)

// Clock is a logical, Lamport-style timestamp for last-write-wins resolution: the higher
// Counter wins, and ReplicaID breaks ties so every replica resolves any conflict
// identically — the property that makes the store convergent rather than merely
// eventually-something.
type Clock struct {
	Counter   uint64 `json:"counter"`
	ReplicaID string `json:"replica"`
}

// After reports whether c should win over o under last-write-wins.
func (parseC Clock) After(parseOther Clock) bool {
	if parseC.Counter != parseOther.Counter {
		return parseC.Counter > parseOther.Counter
	}
	return parseC.ReplicaID > parseOther.ReplicaID
}

// Record is one synced key/value with the clock that stamped it and a tombstone flag for
// deletes (deletes must propagate, so they are records, not absences).
type Record struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Clock   Clock  `json:"clock"`
	Deleted bool   `json:"deleted"`
}

// Mutation is one local change awaiting sync — the durable offline-queue entry.
type Mutation struct {
	Record Record `json:"record"`
}

// Replica is a client-side local-first store: optimistic local writes, a durable pending
// log of unsynced writes (survives offline), and convergent Merge of authoritative records.
// Safe for concurrent use.
type Replica struct {
	mu      sync.Mutex
	id      string
	counter uint64
	records map[string]Record
	pending []Mutation
}

// NewReplica creates an empty replica with a stable, globally-unique id (the id breaks
// write-conflict ties, so distinct replicas must use distinct ids).
func NewReplica(parseID string) *Replica {
	return &Replica{id: parseID, records: map[string]Record{}}
}

// nextClock mints the next logical clock for a local write. Caller holds mu.
func (parseR *Replica) nextClock() Clock {
	parseR.counter++
	return Clock{Counter: parseR.counter, ReplicaID: parseR.id}
}

// observeClock advances the local counter past a clock we have seen, preserving Lamport
// monotonicity so a re-edit made after pulling a remote change outranks it. Caller holds mu.
func (parseR *Replica) observeClock(parseClock Clock) {
	if parseClock.Counter > parseR.counter {
		parseR.counter = parseClock.Counter
	}
}

// Set writes key=value optimistically (visible immediately) and queues it for sync.
func (parseR *Replica) Set(parseKey, parseValue string) Record {
	parseR.mu.Lock()
	defer parseR.mu.Unlock()
	parseRecord := Record{Key: parseKey, Value: parseValue, Clock: parseR.nextClock()}
	parseR.records[parseKey] = parseRecord
	parseR.pending = append(parseR.pending, Mutation{Record: parseRecord})
	return parseRecord
}

// Delete tombstones key optimistically and queues the delete for sync.
func (parseR *Replica) Delete(parseKey string) {
	parseR.mu.Lock()
	defer parseR.mu.Unlock()
	parseRecord := Record{Key: parseKey, Clock: parseR.nextClock(), Deleted: true}
	parseR.records[parseKey] = parseRecord
	parseR.pending = append(parseR.pending, Mutation{Record: parseRecord})
}

// Get returns the live value for key (absent for unknown or tombstoned keys).
func (parseR *Replica) Get(parseKey string) (string, bool) {
	parseR.mu.Lock()
	defer parseR.mu.Unlock()
	parseRecord, parseOk := parseR.records[parseKey]
	if !parseOk || parseRecord.Deleted {
		return "", false
	}
	return parseRecord.Value, true
}

// Pending returns a copy of the unsynced mutation log (what Push would send).
func (parseR *Replica) Pending() []Mutation {
	parseR.mu.Lock()
	defer parseR.mu.Unlock()
	return append([]Mutation(nil), parseR.pending...)
}

// Merge converges the replica toward a batch of authoritative records: each record is
// applied iff its clock wins under LWW, the local counter advances past it, and any pending
// mutation the authority has now acknowledged (its clock is equal-or-newer) is dropped.
func (parseR *Replica) Merge(parseChanges []Record) {
	parseR.mu.Lock()
	defer parseR.mu.Unlock()

	parseAuthClock := make(map[string]Clock, len(parseChanges))
	for _, parseIncoming := range parseChanges {
		parseR.observeClock(parseIncoming.Clock)
		parseCurrent, parseExists := parseR.records[parseIncoming.Key]
		if !parseExists || parseIncoming.Clock.After(parseCurrent.Clock) {
			parseR.records[parseIncoming.Key] = parseIncoming
		}
		if parsePrev, parseSeen := parseAuthClock[parseIncoming.Key]; !parseSeen || parseIncoming.Clock.After(parsePrev) {
			parseAuthClock[parseIncoming.Key] = parseIncoming.Clock
		}
	}

	// Acknowledge pending writes the authority has now caught up to (equal-or-newer clock).
	var parseKept []Mutation
	for _, parseMutation := range parseR.pending {
		parseClock, parseHasAuth := parseAuthClock[parseMutation.Record.Key]
		if parseHasAuth && !parseMutation.Record.Clock.After(parseClock) {
			continue
		}
		parseKept = append(parseKept, parseMutation)
	}
	parseR.pending = parseKept
}

// Snapshot returns the replica's live key/value state (tombstones excluded) — used to
// assert convergence.
func (parseR *Replica) Snapshot() map[string]string {
	parseR.mu.Lock()
	defer parseR.mu.Unlock()
	return liveValues(parseR.records)
}

// ReplicaState is the full, JSON-serializable state of a replica — its records, its
// unsynced pending log, and its logical counter. Persist it (localStorage / IndexedDB /
// db/sqlite) so a replica survives a page reload with its offline writes intact, then
// Restore it: the durable offline queue that makes "edit offline, close the tab, reopen,
// reconnect, converge" actually work.
type ReplicaState struct {
	ID      string            `json:"id"`
	Counter uint64            `json:"counter"`
	Records map[string]Record `json:"records"`
	Pending []Mutation        `json:"pending"`
}

// Export captures the replica's full state for durable persistence.
func (parseR *Replica) Export() ReplicaState {
	parseR.mu.Lock()
	defer parseR.mu.Unlock()
	parseRecords := make(map[string]Record, len(parseR.records))
	maps.Copy(parseRecords, parseR.records)
	return ReplicaState{
		ID:      parseR.id,
		Counter: parseR.counter,
		Records: parseRecords,
		Pending: append([]Mutation(nil), parseR.pending...),
	}
}

// RestoreReplica rebuilds a replica from persisted state — typically read back from
// storage on page load — so its records and, crucially, its unsynced pending writes are
// exactly as they were before the reload.
func RestoreReplica(parseState ReplicaState) *Replica {
	parseRecords := make(map[string]Record, len(parseState.Records))
	maps.Copy(parseRecords, parseState.Records)
	return &Replica{
		id:      parseState.ID,
		counter: parseState.Counter,
		records: parseRecords,
		pending: append([]Mutation(nil), parseState.Pending...),
	}
}

// Authority is the server-authoritative store: it resolves incoming client mutations under
// the same LWW rule and serves the converged state back. Safe for concurrent use.
type Authority struct {
	mu      sync.Mutex
	records map[string]Record
}

// NewAuthority creates an empty server-side store.
func NewAuthority() *Authority {
	return &Authority{records: map[string]Record{}}
}

// Receive applies a batch of client mutations under LWW and returns the resulting
// authoritative records for the affected keys (their post-merge winning state), sorted by
// key, so the client can pull and converge.
func (parseA *Authority) Receive(parseMutations []Mutation) []Record {
	parseA.mu.Lock()
	defer parseA.mu.Unlock()
	parseAffected := map[string]struct{}{}
	for _, parseMutation := range parseMutations {
		parseKey := parseMutation.Record.Key
		parseCurrent, parseExists := parseA.records[parseKey]
		if !parseExists || parseMutation.Record.Clock.After(parseCurrent.Clock) {
			parseA.records[parseKey] = parseMutation.Record
		}
		parseAffected[parseKey] = struct{}{}
	}
	parseChanges := make([]Record, 0, len(parseAffected))
	for parseKey := range parseAffected {
		parseChanges = append(parseChanges, parseA.records[parseKey])
	}
	sortRecords(parseChanges)
	return parseChanges
}

// Changes returns every authoritative record (including tombstones), sorted by key — the
// full-state pull a replica merges to converge.
func (parseA *Authority) Changes() []Record {
	parseA.mu.Lock()
	defer parseA.mu.Unlock()
	parseChanges := make([]Record, 0, len(parseA.records))
	for _, parseRecord := range parseA.records {
		parseChanges = append(parseChanges, parseRecord)
	}
	sortRecords(parseChanges)
	return parseChanges
}

// Snapshot returns the authority's live key/value state (tombstones excluded).
func (parseA *Authority) Snapshot() map[string]string {
	parseA.mu.Lock()
	defer parseA.mu.Unlock()
	return liveValues(parseA.records)
}

// Sync performs one round-trip of the protocol: it pushes the replica's pending mutations
// to the authority and merges the full authoritative state back, converging the replica.
// Calling it again with an empty pending log is a pure pull (used to fan out another
// replica's accepted writes). Returns the records merged.
func Sync(parseReplica *Replica, parseAuthority *Authority) []Record {
	parseAuthority.Receive(parseReplica.Pending())
	parseChanges := parseAuthority.Changes()
	parseReplica.Merge(parseChanges)
	return parseChanges
}

// liveValues projects a record map to its non-tombstoned key/value pairs.
func liveValues(parseRecords map[string]Record) map[string]string {
	parseOut := map[string]string{}
	for parseKey, parseRecord := range parseRecords {
		if !parseRecord.Deleted {
			parseOut[parseKey] = parseRecord.Value
		}
	}
	return parseOut
}

// sortRecords orders records by key for deterministic output.
func sortRecords(parseRecords []Record) {
	sort.Slice(parseRecords, func(parseA, parseB int) bool { return parseRecords[parseA].Key < parseRecords[parseB].Key })
}
