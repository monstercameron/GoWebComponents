package agenthub

import (
	"testing"
	"time"
)

// TestKeepaliveDropsUnresponsivePeer pins that after the hello handshake a
// half-open connection — one that stops reading and never pongs — is detected and
// dropped by the ping/read-deadline keepalive. Before this, the post-handshake
// read deadline was cleared, so ReadMessage blocked forever and the session
// goroutine (plus its hub.sessions entry) leaked until process exit.
func TestKeepaliveDropsUnresponsivePeer(parseT *testing.T) {
	// Shrink the keepalive so the test is fast; restore afterward.
	parseOrigPong, parseOrigPing, parseOrigWrite := agentPongWait, agentPingPeriod, agentWriteWait
	agentPongWait = 200 * time.Millisecond
	agentPingPeriod = 60 * time.Millisecond
	agentWriteWait = 200 * time.Millisecond
	defer func() {
		agentPongWait, agentPingPeriod, agentWriteWait = parseOrigPong, parseOrigPing, parseOrigWrite
	}()

	parseHub, parseServer := newTestHub(parseT)
	defer parseServer.Close()

	parseConn, parseSessID := dialAgentSession(parseT, parseServer, parseHub, "dead-peer", "b1", nil)
	defer parseConn.Close()

	// Simulate a dead peer: never read (so the server's pings are never answered
	// with a pong) and explicitly swallow any ping so no pong is ever produced.
	parseConn.SetPingHandler(func(string) error { return nil })

	// The server's read deadline should expire within a few pongWait windows and
	// move the session out of the Active state.
	parseDeadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(parseDeadline) {
		for _, parseSess := range parseHub.ListSessions() {
			if parseSess.ID == parseSessID && parseSess.State != StateActive {
				return // success: the unresponsive peer was dropped
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	parseT.Fatal("expected keepalive to drop the unresponsive session, but it stayed Active")
}

// TestKeepaliveKeepsResponsivePeerAlive pins that a normal peer (one that keeps
// reading, so gorilla auto-pongs the server's pings) is NOT dropped across
// multiple ping periods — the keepalive must not evict healthy connections.
func TestKeepaliveKeepsResponsivePeerAlive(parseT *testing.T) {
	parseOrigPong, parseOrigPing, parseOrigWrite := agentPongWait, agentPingPeriod, agentWriteWait
	agentPongWait = 200 * time.Millisecond
	agentPingPeriod = 60 * time.Millisecond
	agentWriteWait = 200 * time.Millisecond
	defer func() {
		agentPongWait, agentPingPeriod, agentWriteWait = parseOrigPong, parseOrigPing, parseOrigWrite
	}()

	parseHub, parseServer := newTestHub(parseT)
	defer parseServer.Close()

	parseConn, parseSessID := dialAgentSession(parseT, parseServer, parseHub, "live-peer", "b1", nil)
	defer parseConn.Close()

	// Keep reading (blocking, no deadline) so gorilla's default ping handler
	// auto-pongs the server's pings. On any error (e.g. the deferred Close at test
	// end) the goroutine returns without re-reading — gorilla panics on a repeated
	// read of a failed connection.
	go func() {
		for {
			if _, _, parseErr := parseConn.ReadMessage(); parseErr != nil {
				return
			}
		}
	}()

	// Well past several ping periods, the session must still be Active.
	time.Sleep(5 * agentPongWait)
	for _, parseSess := range parseHub.ListSessions() {
		if parseSess.ID == parseSessID {
			if parseSess.State != StateActive {
				parseT.Fatalf("responsive session was wrongly dropped: state=%v", parseSess.State)
			}
			return
		}
	}
	parseT.Fatal("session disappeared entirely")
}
