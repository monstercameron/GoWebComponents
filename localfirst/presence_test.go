package localfirst

import (
	"strings"
	"testing"
)

// TestPresenceUpdateAndLive proves peers appear once they publish presence, sorted by id.
func TestPresenceUpdateAndLive(parseT *testing.T) {
	parsePresence := NewPresenceSet(2)
	parsePresence.Update(Presence{ClientID: "B", State: `{"name":"Bob"}`})
	parsePresence.Update(Presence{ClientID: "A", State: `{"name":"Ada"}`})

	parseLive := parsePresence.Live()
	if len(parseLive) != 2 || parseLive[0].ClientID != "A" || parseLive[1].ClientID != "B" {
		parseT.Fatalf("expected sorted [A B], got %+v", parseLive)
	}
	if !strings.Contains(parseLive[0].State, "Ada") {
		parseT.Fatalf("expected A's state payload, got %q", parseLive[0].State)
	}
}

// TestPresenceExpiresWithoutHeartbeat proves a peer that stops refreshing is dropped after
// ttl ticks — a crashed tab leaves the session on its own.
func TestPresenceExpiresWithoutHeartbeat(parseT *testing.T) {
	parsePresence := NewPresenceSet(2)
	parsePresence.Update(Presence{ClientID: "A"})

	parsePresence.Tick() // age 1 — still within ttl
	if parsePresence.Count() != 1 {
		parseT.Fatal("peer should survive within ttl")
	}
	parsePresence.Tick() // age 2 — still within ttl (now-lastSeen == 2, not > 2)
	if parsePresence.Count() != 1 {
		parseT.Fatal("peer should survive at exactly ttl")
	}
	parsePresence.Tick() // age 3 — exceeds ttl
	if parsePresence.Count() != 0 {
		parseT.Fatal("peer should expire after ttl ticks without a heartbeat")
	}
}

// TestPresenceHeartbeatKeepsAlive proves refreshing within ttl keeps a peer present
// indefinitely.
func TestPresenceHeartbeatKeepsAlive(parseT *testing.T) {
	parsePresence := NewPresenceSet(1)
	parsePresence.Update(Presence{ClientID: "A"})

	for range 5 {
		parsePresence.Tick()
		parsePresence.Update(Presence{ClientID: "A"}) // heartbeat each interval
	}
	if parsePresence.Count() != 1 {
		parseT.Fatal("a peer that heartbeats every interval must stay present")
	}
}

// TestPresenceRemoveLeavesImmediately proves an explicit leave drops the peer at once.
func TestPresenceRemoveLeavesImmediately(parseT *testing.T) {
	parsePresence := NewPresenceSet(10)
	parsePresence.Update(Presence{ClientID: "A"})
	parsePresence.Update(Presence{ClientID: "B"})

	parsePresence.Remove("A")
	parseLive := parsePresence.Live()
	if len(parseLive) != 1 || parseLive[0].ClientID != "B" {
		parseT.Fatalf("expected only B after A leaves, got %+v", parseLive)
	}
}
