package localfirst

import (
	"sort"
	"sync"
)

// Presence is one client's ephemeral awareness state — cursor position, name, colour,
// "is typing", etc. It is deliberately NOT a synced Record: "who is here right now" has no
// history and must never be conflict-merged or persisted, so presence is last-write plus
// heartbeat-expiry. With the converging document store (Replica/Authority), this is the
// second half of real-time collaboration (FC6), which the sync engine gives nearly for
// free.
type Presence struct {
	// ClientID identifies the peer (one per browser tab/session).
	ClientID string `json:"client"`
	// State is an opaque, app-defined JSON payload (e.g. a cursor and a display name).
	State string `json:"state"`
}

// presenceRecord pairs a presence with the heartbeat tick it was last seen at.
type presenceRecord struct {
	presence Presence
	lastSeen uint64
}

// PresenceSet tracks the live peers in a collaboration session, expiring any that miss
// heartbeats. It owns no wall clock: the app advances time by calling Tick on its
// heartbeat interval, which keeps the set deterministic and unit-testable. Safe for
// concurrent use.
type PresenceSet struct {
	mu      sync.Mutex
	now     uint64
	ttl     uint64
	entries map[string]presenceRecord
}

// NewPresenceSet creates a presence set whose peers expire after ttl heartbeat ticks
// without a refresh. A ttl of 0 means a peer expires on the next Tick unless refreshed.
func NewPresenceSet(parseTTL uint64) *PresenceSet {
	return &PresenceSet{ttl: parseTTL, entries: map[string]presenceRecord{}}
}

// Update records or refreshes a peer's presence, marking it seen at the current tick.
func (parseP *PresenceSet) Update(parsePresence Presence) {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	parseP.entries[parsePresence.ClientID] = presenceRecord{presence: parsePresence, lastSeen: parseP.now}
}

// Tick advances the heartbeat clock by one and removes peers that have not been refreshed
// within ttl ticks — the expiry that makes a crashed or closed tab disappear from the
// session without an explicit leave.
func (parseP *PresenceSet) Tick() {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	parseP.now++
	for parseID, parseRecord := range parseP.entries {
		if parseP.now-parseRecord.lastSeen > parseP.ttl {
			delete(parseP.entries, parseID)
		}
	}
}

// Remove drops a peer immediately (an explicit leave / tab close).
func (parseP *PresenceSet) Remove(parseClientID string) {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	delete(parseP.entries, parseClientID)
}

// Live returns the currently-present peers, sorted by client id for stable rendering.
func (parseP *PresenceSet) Live() []Presence {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	parsePresences := make([]Presence, 0, len(parseP.entries))
	for _, parseRecord := range parseP.entries {
		parsePresences = append(parsePresences, parseRecord.presence)
	}
	sort.Slice(parsePresences, func(parseA, parseB int) bool {
		return parsePresences[parseA].ClientID < parsePresences[parseB].ClientID
	})
	return parsePresences
}

// Count returns the number of live peers.
func (parseP *PresenceSet) Count() int {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	return len(parseP.entries)
}
