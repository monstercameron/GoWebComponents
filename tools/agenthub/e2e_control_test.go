package agenthub

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/monstercameron/GoWebComponents/agentbridge"
	"github.com/monstercameron/GoWebComponents/events"
	"github.com/monstercameron/GoWebComponents/state"
)

// describeResult mirrors the bridge.describe ack payload for assertions.
type describeResult struct {
	Commands []string `json:"commands"`
	Atoms    []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"atoms"`
	Events []struct {
		Topic       string `json:"topic"`
		Subscribers int    `json:"subscribers"`
	} `json:"events"`
}

// TestEndToEndDescribeAndWaitForOverWebSocket proves the two control verbs that
// make the bridge agentically self-describing actually work across a real
// socket: bridge.describe returns the atom + event vocabulary (so an agent
// learns what it can drive without reading source), and bridge.wait-for
// resolves an atom predicate that is already satisfied. Both are reached
// through the hub's HTTP command API, the same path the MCP layer uses.
func TestEndToEndDescribeAndWaitForOverWebSocket(t *testing.T) {
	parseHub, parseErr := NewAgentHub()
	if parseErr != nil {
		t.Fatalf("new hub: %v", parseErr)
	}
	parseServer := httptest.NewServer(parseHub)
	defer parseServer.Close()

	agentbridge.RegisterReadCommands()
	agentbridge.RegisterWriteCommands()
	agentbridge.RegisterControlCommands()
	agentbridge.SetAgentModeActive(true)
	defer agentbridge.SetAgentModeActive(false)

	// Seed a known atom and a known event topic so describe has vocabulary to
	// report and an agent could discover both.
	if parseSeedErr := state.ApplySnapshot(state.Snapshot{"control.mode": "edit"}); parseSeedErr != nil {
		t.Fatalf("seed atom: %v", parseSeedErr)
	}
	parseUnsub := events.Subscribe[string]("control.saved", func(string) {})
	defer parseUnsub()

	parseWSURL := "ws" + strings.TrimPrefix(parseServer.URL, "http") + "/gwc-agent?token=" + parseHub.Token()
	parseConn, _, parseDialErr := websocket.DefaultDialer.Dial(parseWSURL, nil)
	if parseDialErr != nil {
		t.Fatalf("dial hub: %v", parseDialErr)
	}
	parseClient := agentbridge.NewBridgeClient("control-app", "build-1")
	go parseClient.RunLoop(&gorillaSocket{conn: parseConn})
	parseSessionID := waitForActiveSession(t, parseHub)

	// --- describe: the agent's "what can I control here?" call. ---
	parseAck := postCommand(t, parseServer.URL, parseHub.Token(), apiCommandRequest{
		Session: parseSessionID,
		Name:    "bridge.describe",
	})
	if parseAck.OK == nil || !*parseAck.OK {
		t.Fatalf("describe ack not ok: %+v", parseAck.Error)
	}
	var parseDesc describeResult
	if parseErr := json.Unmarshal(parseAck.Payload, &parseDesc); parseErr != nil {
		t.Fatalf("decode describe: %v", parseErr)
	}
	if !describeHasAtom(parseDesc, "control.mode") {
		t.Fatalf("describe omitted the seeded atom; atoms=%+v", parseDesc.Atoms)
	}
	if !describeHasCommand(parseDesc, "bridge.set-atom") {
		t.Fatalf("describe omitted the registered commands; got %v", parseDesc.Commands)
	}
	// The event topic with its subscriber count is what makes publish
	// discoverable rather than a silent guess.
	if !describeHasEvent(parseDesc, "control.saved", 1) {
		t.Fatalf("describe omitted the seeded event topic (or wrong count); events=%+v", parseDesc.Events)
	}

	// --- wait-for: an already-satisfied atom predicate resolves promptly. ---
	parseWaitAck := postCommand(t, parseServer.URL, parseHub.Token(), apiCommandRequest{
		Session: parseSessionID,
		Name:    "bridge.wait-for",
		Payload: json.RawMessage(`{"timeoutMs":1000,"atom":{"id":"control.mode","equals":"edit"}}`),
	})
	if parseWaitAck.OK == nil || !*parseWaitAck.OK {
		t.Fatalf("wait-for ack not ok: %+v", parseWaitAck.Error)
	}

	// --- wait-for: an unsatisfiable predicate times out with a structured code. ---
	parseTimeoutAck := postCommand(t, parseServer.URL, parseHub.Token(), apiCommandRequest{
		Session: parseSessionID,
		Name:    "bridge.wait-for",
		Payload: json.RawMessage(`{"timeoutMs":80,"atom":{"id":"control.mode","equals":"never"}}`),
	})
	if parseTimeoutAck.OK == nil || *parseTimeoutAck.OK {
		t.Fatalf("expected wait-for to time out")
	}
	if parseTimeoutAck.Error == nil || parseTimeoutAck.Error.Code != agentbridge.ErrorCodeTimeout {
		t.Fatalf("expected timeout code, got %+v", parseTimeoutAck.Error)
	}
}

func describeHasAtom(parseDesc describeResult, parseID string) bool {
	for _, parseAtom := range parseDesc.Atoms {
		if parseAtom.ID == parseID {
			return true
		}
	}
	return false
}

func describeHasCommand(parseDesc describeResult, parseName string) bool {
	for _, parseCmd := range parseDesc.Commands {
		if parseCmd == parseName {
			return true
		}
	}
	return false
}

func describeHasEvent(parseDesc describeResult, parseTopic string, parseWantSubs int) bool {
	for _, parseEvent := range parseDesc.Events {
		if parseEvent.Topic == parseTopic {
			return parseEvent.Subscribers == parseWantSubs
		}
	}
	return false
}
