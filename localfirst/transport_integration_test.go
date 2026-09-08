package localfirst_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/localfirst"
	"github.com/monstercameron/GoWebComponents/v6/serverfn"
)

// pushReq/pushResp and pullReq/pullResp are the sync shape carried over the //gwc:server
// transport — exactly what a real app would define and expose with `gwc server gen`.
type pushReq struct {
	Mutations []localfirst.Mutation `json:"mutations"`
}
type pushResp struct {
	Changes []localfirst.Record `json:"changes"`
}
type pullReq struct{}
type pullResp struct {
	Changes []localfirst.Record `json:"changes"`
}

// TestSyncConvergesOverServerfnTransport is the FC1×FB1 end-to-end proof: the sync engine
// is driven entirely over the real serverfn HTTP transport (push + pull server functions
// backed by an Authority), and two replicas that edit the same key offline still converge
// to the one last-write-wins value after reconnecting.
func TestSyncConvergesOverServerfnTransport(parseT *testing.T) {
	parseAuthority := localfirst.NewAuthority()

	parseMux := http.NewServeMux()
	serverfn.Handle(parseMux, "SyncPush", func(parseCtx context.Context, parseReq pushReq) (pushResp, error) {
		return pushResp{Changes: parseAuthority.Receive(parseReq.Mutations)}, nil
	})
	serverfn.Handle(parseMux, "SyncPull", func(parseCtx context.Context, parseReq pullReq) (pullResp, error) {
		return pullResp{Changes: parseAuthority.Changes()}, nil
	})
	parseServer := httptest.NewServer(parseMux)
	defer parseServer.Close()
	serverfn.Configure(parseServer.URL)
	defer serverfn.Configure("")

	parseSyncOverWire := func(parseReplica *localfirst.Replica) error {
		if _, parseErr := serverfn.Call[pushReq, pushResp](context.Background(), "SyncPush", pushReq{Mutations: parseReplica.Pending()}); parseErr != nil {
			return parseErr
		}
		parsePull, parseErr := serverfn.Call[pullReq, pullResp](context.Background(), "SyncPull", pullReq{})
		if parseErr != nil {
			return parseErr
		}
		parseReplica.Merge(parsePull.Changes)
		return nil
	}

	parseReplicaA := localfirst.NewReplica("A")
	parseReplicaB := localfirst.NewReplica("B")

	// Offline edits to the same key.
	parseReplicaA.Set("doc", "from-A")
	parseReplicaB.Set("doc", "from-B")

	// Reconnect in sequence; B wins the tie. A pulls again to converge.
	for _, parseStep := range []struct {
		name    string
		replica *localfirst.Replica
	}{{"A", parseReplicaA}, {"B", parseReplicaB}, {"A-again", parseReplicaA}} {
		if parseErr := parseSyncOverWire(parseStep.replica); parseErr != nil {
			parseT.Fatalf("sync %s over transport failed: %v", parseStep.name, parseErr)
		}
	}

	parseWant := "from-B"
	if parseGot := parseAuthority.Snapshot()["doc"]; parseGot != parseWant {
		parseT.Fatalf("authority did not converge over the wire: got %q, want %q", parseGot, parseWant)
	}
	if parseValue, parseOk := parseReplicaA.Get("doc"); !parseOk || parseValue != parseWant {
		parseT.Fatalf("replica A did not converge over the wire: got %q", parseValue)
	}
	if parseValue, parseOk := parseReplicaB.Get("doc"); !parseOk || parseValue != parseWant {
		parseT.Fatalf("replica B did not converge over the wire: got %q", parseValue)
	}
}
