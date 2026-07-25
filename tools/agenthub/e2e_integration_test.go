package agenthub

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/monstercameron/GoWebComponents/v5/agentbridge"
	"github.com/monstercameron/GoWebComponents/v5/state"
)

// gorillaSocket adapts a gorilla *websocket.Conn to agentbridge.AgentSocket so
// the transport-agnostic BridgeClient loop can run against the real hub. This
// is the role the browser WebSocket plays in production.
type gorillaSocket struct {
	conn *websocket.Conn
}

// ReadFrame blocks for one text frame from the hub.
func (parseSock *gorillaSocket) ReadFrame() (string, error) {
	_, parseData, parseErr := parseSock.conn.ReadMessage()
	if parseErr != nil {
		return "", parseErr
	}
	return string(parseData), nil
}

// WriteFrame sends one text frame to the hub.
func (parseSock *gorillaSocket) WriteFrame(parseFrame string) error {
	return parseSock.conn.WriteMessage(websocket.TextMessage, []byte(parseFrame))
}

// Close terminates the underlying connection.
func (parseSock *gorillaSocket) Close() error {
	return parseSock.conn.Close()
}

// waitForActiveSession polls until the hub reports one active session or fails.
func waitForActiveSession(parseTB testing.TB, parseHub *AgentHub) string {
	parseTB.Helper()
	parseDeadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(parseDeadline) {
		for _, parseInfo := range parseHub.ListSessions() {
			if parseInfo.State == StateActive {
				return parseInfo.ID
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	parseTB.Fatalf("no active session registered before deadline")
	return ""
}

// TestEndToEndCommandLoopOverRealWebSocket drives the WHOLE bridge loop with a
// real WebSocket: the hub's HTTP command API (the surface the MCP layer calls)
// -> hub.SendCommand -> WS frame -> BridgeClient.RunLoop -> ExecuteAgentCommand
// -> the real set-atom/snapshot handlers -> ack back through the hub to the
// HTTP response. This is the agent's actual round trip minus the browser
// syscall/js socket and the MCP stdio shell.
func TestEndToEndCommandLoopOverRealWebSocket(t *testing.T) {
	parseHub, parseErr := NewAgentHub()
	if parseErr != nil {
		t.Fatalf("new hub: %v", parseErr)
	}
	parseServer := httptest.NewServer(parseHub)
	defer parseServer.Close()

	// --- App side: register real commands, seed state, enable agent mode. ---
	agentbridge.RegisterReadCommands()
	agentbridge.RegisterWriteCommands()
	agentbridge.SetAgentModeActive(true)
	defer agentbridge.SetAgentModeActive(false)
	if parseSeedErr := state.ApplySnapshot(state.Snapshot{"e2e.theme": "light"}); parseSeedErr != nil {
		t.Fatalf("seed atom: %v", parseSeedErr)
	}

	// Dial the hub the way the wasm app does: ws:// with the minted token.
	parseWSURL := "ws" + strings.TrimPrefix(parseServer.URL, "http") + "/gwc-agent?token=" + parseHub.Token()
	parseConn, _, parseDialErr := websocket.DefaultDialer.Dial(parseWSURL, nil)
	if parseDialErr != nil {
		t.Fatalf("dial hub: %v", parseDialErr)
	}
	parseClient := agentbridge.NewBridgeClient("e2e-app", "build-1")
	go parseClient.RunLoop(&gorillaSocket{conn: parseConn})

	parseSessionID := waitForActiveSession(t, parseHub)
	postLease(t, parseServer.URL, parseHub.Token(), apiLeaseRequest{
		Session: parseSessionID,
		Action:  "acquire",
		Holder:  "e2e-agent",
	})

	// --- Agent side: send bridge.set-atom through the HTTP command API. ---
	parseAck := postCommand(t, parseServer.URL, parseHub.Token(), apiCommandRequest{
		Session:     parseSessionID,
		Name:        "bridge.set-atom",
		Payload:     json.RawMessage(`{"id":"e2e.theme","value":"dark"}`),
		LeaseHolder: "e2e-agent",
	})
	if parseAck.OK == nil || !*parseAck.OK {
		t.Fatalf("set-atom ack not ok: %+v", parseAck.Error)
	}
	// Prove the command actually executed in the app: the atom changed. Read
	// it back through the public state package (the same view a component has).
	parseSnap, parseSnapErr := state.GetSnapshot()
	if parseSnapErr != nil {
		t.Fatalf("read state snapshot: %v", parseSnapErr)
	}
	if parseGot := parseSnap["e2e.theme"]; parseGot != "dark" {
		t.Fatalf("set-atom did not take effect across the wire: got %#v want \"dark\"", parseGot)
	}

	// --- A read command round-trips too: bridge.snapshot returns an ok ack. ---
	parseSnapAck := postCommand(t, parseServer.URL, parseHub.Token(), apiCommandRequest{
		Session: parseSessionID,
		Name:    "bridge.snapshot",
		Payload: json.RawMessage(`{"maxDepth":4,"maxNodes":50}`),
	})
	if parseSnapAck.OK == nil || !*parseSnapAck.OK {
		t.Fatalf("snapshot ack not ok: %+v", parseSnapAck.Error)
	}

	// --- An unknown command fails with a structured code, not a hang. ---
	parseBadAck := postCommand(t, parseServer.URL, parseHub.Token(), apiCommandRequest{
		Session: parseSessionID,
		Name:    "bridge.does-not-exist",
		Payload: json.RawMessage(`{}`),
	})
	if parseBadAck.OK == nil || *parseBadAck.OK {
		t.Fatalf("expected unknown command to fail")
	}
	if parseBadAck.Error == nil || parseBadAck.Error.Code != agentbridge.ErrorCodeUnknownCommand {
		t.Fatalf("expected unknown-command code, got %+v", parseBadAck.Error)
	}
}

func postLease(parseTB testing.TB, parseBaseURL string, parseToken string, parseReq apiLeaseRequest) apiLeaseResponse {
	parseTB.Helper()
	parseBody, _ := json.Marshal(parseReq)
	parseHTTPReq, parseErr := http.NewRequestWithContext(context.Background(), http.MethodPost,
		parseBaseURL+"/__gwc-agent/lease?token="+parseToken, bytes.NewReader(parseBody))
	if parseErr != nil {
		parseTB.Fatalf("build lease request: %v", parseErr)
	}
	parseHTTPReq.Header.Set("Content-Type", "application/json")
	parseResp, parseErr := http.DefaultClient.Do(parseHTTPReq)
	if parseErr != nil {
		parseTB.Fatalf("post lease: %v", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusOK {
		parseTB.Fatalf("lease HTTP status %d", parseResp.StatusCode)
	}
	var parseDecoded apiLeaseResponse
	if parseErr := json.NewDecoder(parseResp.Body).Decode(&parseDecoded); parseErr != nil {
		parseTB.Fatalf("decode lease response: %v", parseErr)
	}
	return parseDecoded
}

// postCommand POSTs a command to the hub's HTTP API and returns the ack.
func postCommand(parseTB testing.TB, parseBaseURL string, parseToken string, parseReq apiCommandRequest) agentbridge.Envelope {
	parseTB.Helper()
	parseBody, _ := json.Marshal(parseReq)
	parseHTTPReq, parseErr := http.NewRequestWithContext(context.Background(), http.MethodPost,
		parseBaseURL+"/__gwc-agent/command?token="+parseToken, bytes.NewReader(parseBody))
	if parseErr != nil {
		parseTB.Fatalf("build request: %v", parseErr)
	}
	parseHTTPReq.Header.Set("Content-Type", "application/json")
	parseResp, parseErr := http.DefaultClient.Do(parseHTTPReq)
	if parseErr != nil {
		parseTB.Fatalf("post command: %v", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusOK {
		parseTB.Fatalf("command HTTP status %d", parseResp.StatusCode)
	}
	var parseDecoded apiCommandResponse
	if parseErr := json.NewDecoder(parseResp.Body).Decode(&parseDecoded); parseErr != nil {
		parseTB.Fatalf("decode ack: %v", parseErr)
	}
	return parseDecoded.Ack
}
