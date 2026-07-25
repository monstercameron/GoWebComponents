package localfirst_test

import (
	"encoding/json"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/localfirst"
)

// TestSSRSeedsSyncedStore proves D6: the server's authoritative state seeds a fresh client
// replica during hydration, so the first paint has data with no client fetch — and the
// replica is immediately convergent (a later edit reconciles normally). The server's
// Authority.Changes() is the SSR bootstrap payload; the client merges it into a new replica.
func TestSSRSeedsSyncedStore(parseT *testing.T) {
	// Server side: authoritative store already holds the page's data.
	parseAuthority := localfirst.NewAuthority()
	parseSeed := localfirst.NewReplica("server")
	parseSeed.Set("todo/1", "buy milk")
	parseSeed.Set("todo/2", "write tests")
	localfirst.Sync(parseSeed, parseAuthority)

	// SSR bootstrap: serialize the authoritative records into the page.
	parseBootstrap, parseErr := json.Marshal(parseAuthority.Changes())
	if parseErr != nil {
		parseT.Fatalf("marshal SSR bootstrap: %v", parseErr)
	}

	// Client side on hydration: a fresh replica seeded from the bootstrap — no fetch.
	var parseRecords []localfirst.Record
	if parseErr := json.Unmarshal(parseBootstrap, &parseRecords); parseErr != nil {
		parseT.Fatalf("unmarshal SSR bootstrap: %v", parseErr)
	}
	parseClient := localfirst.NewReplica("client")
	parseClient.Merge(parseRecords)

	if parseValue, parseOk := parseClient.Get("todo/1"); !parseOk || parseValue != "buy milk" {
		parseT.Fatalf("client should be seeded with todo/1 at first paint, got %q", parseValue)
	}
	if len(parseClient.Pending()) != 0 {
		parseT.Fatal("seeding from server state must not create pending writes (it is authoritative)")
	}

	// The seeded replica is fully convergent: a local edit syncs back normally.
	parseClient.Set("todo/2", "write MORE tests")
	localfirst.Sync(parseClient, parseAuthority)
	if parseAuthority.Snapshot()["todo/2"] != "write MORE tests" {
		parseT.Fatalf("a post-hydration edit should converge, got %v", parseAuthority.Snapshot())
	}
}
