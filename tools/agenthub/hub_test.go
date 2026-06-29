package agenthub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/monstercameron/GoWebComponents/agentbridge"
)

// dialAgentSession is a test helper that dials /gwc-agent with the correct
// token and sends a hello frame, returning the connection and the assigned
// session ID.
func dialAgentSession(parseT *testing.T, parseServer *httptest.Server, parseHub *AgentHub, parseAppID string, parseBuildID string, parseCommands []string) (*websocket.Conn, string) {
	parseT.Helper()
	parseWsURL := "ws" + strings.TrimPrefix(parseServer.URL, "http") + "/gwc-agent?token=" + parseHub.Token()
	parseConn, _, parseDialErr := websocket.DefaultDialer.Dial(parseWsURL, nil)
	if parseDialErr != nil {
		parseT.Fatalf("dial /gwc-agent: %v", parseDialErr)
	}

	parsePayload, parseMarshErr := json.Marshal(map[string]any{
		"appId":    parseAppID,
		"buildId":  parseBuildID,
		"commands": parseCommands,
	})
	if parseMarshErr != nil {
		parseT.Fatalf("marshal hello payload: %v", parseMarshErr)
	}
	parseHelloEnv := agentbridge.BuildHelloEnvelope(1, parsePayload)
	parseHelloJSON, parseFormatErr := agentbridge.FormatEnvelopeJSON(parseHelloEnv)
	if parseFormatErr != nil {
		parseT.Fatalf("format hello envelope: %v", parseFormatErr)
	}
	if parseWriteErr := parseConn.WriteMessage(websocket.TextMessage, []byte(parseHelloJSON)); parseWriteErr != nil {
		parseT.Fatalf("write hello: %v", parseWriteErr)
	}

	// Wait for the session to appear.
	parseDeadline := time.Now().Add(2 * time.Second)
	var parseSessID string
	for time.Now().Before(parseDeadline) {
		parseSessions := parseHub.ListSessions()
		for _, parseSess := range parseSessions {
			if parseSess.AppID == parseAppID && parseSess.State == StateActive {
				parseSessID = parseSess.ID
				return parseConn, parseSessID
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	parseT.Fatalf("session never appeared for appId=%q", parseAppID)
	return parseConn, ""
}

// newTestHub creates a test hub and returns it with a httptest.Server.
// The server's RemoteAddr will be loopback (127.0.0.1) so it passes the guard.
func newTestHub(parseT *testing.T) (*AgentHub, *httptest.Server) {
	parseT.Helper()
	parseHub, parseHubErr := NewAgentHub()
	if parseHubErr != nil {
		parseT.Fatalf("NewAgentHub: %v", parseHubErr)
	}
	parseServer := httptest.NewServer(parseHub)
	return parseHub, parseServer
}

// --- (a) hello registers a session with metadata ---

func TestHelloRegistersSessionWithMetadata(parseT *testing.T) {
	parseHub, parseServer := newTestHub(parseT)
	defer parseServer.Close()

	parseConn, parseSessID := dialAgentSession(parseT, parseServer, parseHub, "my-app", "build-abc", []string{"snapshot", "query"})
	defer parseConn.Close()

	parseSessions := parseHub.ListSessions()
	if len(parseSessions) != 1 {
		parseT.Fatalf("expected 1 session, got %d", len(parseSessions))
	}
	parseSess := parseSessions[0]
	if parseSessID == "" {
		parseT.Fatal("expected non-empty session ID")
	}
	if parseSess.ID != parseSessID {
		parseT.Fatalf("session ID mismatch: got %q, want %q", parseSess.ID, parseSessID)
	}
	if parseSess.AppID != "my-app" {
		parseT.Fatalf("expected appId %q, got %q", "my-app", parseSess.AppID)
	}
	if parseSess.BuildID != "build-abc" {
		parseT.Fatalf("expected buildId %q, got %q", "build-abc", parseSess.BuildID)
	}
	if len(parseSess.Commands) != 2 || parseSess.Commands[0] != "snapshot" || parseSess.Commands[1] != "query" {
		parseT.Fatalf("expected commands [snapshot query], got %v", parseSess.Commands)
	}
	if parseSess.State != StateActive {
		parseT.Fatalf("expected state active, got %s", parseSess.State)
	}
	if parseSess.ConnectedAt.IsZero() {
		parseT.Fatal("expected non-zero ConnectedAt")
	}
}

// --- (b) bad token -> 403 and no session ---

func TestBadTokenRejectsWith403(parseT *testing.T) {
	parseHub, parseServer := newTestHub(parseT)
	defer parseServer.Close()

	// Missing token.
	parseWsURL := "ws" + strings.TrimPrefix(parseServer.URL, "http") + "/gwc-agent"
	parseConn, parseResp, parseDialErr := websocket.DefaultDialer.Dial(parseWsURL, nil)
	if parseDialErr == nil {
		parseConn.Close()
		parseT.Fatal("expected dial with no token to fail")
	}
	if parseResp == nil || parseResp.StatusCode != http.StatusForbidden {
		parseStatus := 0
		if parseResp != nil {
			parseStatus = parseResp.StatusCode
		}
		parseT.Fatalf("expected 403 for missing token, got %d", parseStatus)
	}

	// Wrong token.
	parseWsURL2 := "ws" + strings.TrimPrefix(parseServer.URL, "http") + "/gwc-agent?token=wrongtoken"
	parseConn2, parseResp2, parseDialErr2 := websocket.DefaultDialer.Dial(parseWsURL2, nil)
	if parseDialErr2 == nil {
		parseConn2.Close()
		parseT.Fatal("expected dial with wrong token to fail")
	}
	if parseResp2 == nil || parseResp2.StatusCode != http.StatusForbidden {
		parseStatus2 := 0
		if parseResp2 != nil {
			parseStatus2 = parseResp2.StatusCode
		}
		parseT.Fatalf("expected 403 for wrong token, got %d", parseStatus2)
	}

	// No sessions should be registered.
	if parseSessions := parseHub.ListSessions(); len(parseSessions) != 0 {
		parseT.Fatalf("expected 0 sessions after rejected token attempts, got %d", len(parseSessions))
	}
}

// --- (c) two connections = two sessions ---

func TestTwoConnectionsTwoSessions(parseT *testing.T) {
	parseHub, parseServer := newTestHub(parseT)
	defer parseServer.Close()

	parseConnA, parseSessIDA := dialAgentSession(parseT, parseServer, parseHub, "app-a", "build-1", nil)
	defer parseConnA.Close()

	parseConnB, parseSessIDB := dialAgentSession(parseT, parseServer, parseHub, "app-b", "build-2", nil)
	defer parseConnB.Close()

	if parseSessIDA == parseSessIDB {
		parseT.Fatalf("expected distinct session IDs, both are %q", parseSessIDA)
	}
	parseSessions := parseHub.ListSessions()
	if len(parseSessions) != 2 {
		parseT.Fatalf("expected 2 sessions, got %d", len(parseSessions))
	}
}

// --- (d) SendCommand round-trips against a fake app loop that acks ---

func TestSendCommandRoundTrips(parseT *testing.T) {
	parseHub, parseServer := newTestHub(parseT)
	defer parseServer.Close()

	parseConn, parseSessID := dialAgentSession(parseT, parseServer, parseHub, "app-cmd", "build-cmd", []string{"ping"})
	defer parseConn.Close()

	// Run the fake app ack loop.
	var parseFakeSeq atomic.Uint64
	go func() {
		for {
			_, parseRaw, parseReadErr := parseConn.ReadMessage()
			if parseReadErr != nil {
				return
			}
			parseEnv, parseParseErr := agentbridge.ParseEnvelope(string(parseRaw))
			if parseParseErr != nil || parseEnv.Kind != agentbridge.KindCommand {
				continue
			}
			parseAckSeq := parseFakeSeq.Add(1)
			parseAck := agentbridge.BuildAckEnvelope(parseAckSeq, parseSessID, parseEnv.Seq, 42, json.RawMessage(`{"pong":true}`))
			parseAckJSON, _ := agentbridge.FormatEnvelopeJSON(parseAck)
			_ = parseConn.WriteMessage(websocket.TextMessage, []byte(parseAckJSON))
		}
	}()

	parseCtx, parseCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer parseCancel()
	parseResult, parseSendErr := parseHub.SendCommand(parseCtx, parseSessID, "ping", json.RawMessage(`{}`))
	if parseSendErr != nil {
		parseT.Fatalf("SendCommand failed: %v", parseSendErr)
	}
	if parseResult.Kind != agentbridge.KindAck {
		parseT.Fatalf("expected ack kind, got %q", parseResult.Kind)
	}
	if parseResult.OK == nil || !*parseResult.OK {
		parseT.Fatalf("expected successful ack, got ok=%v", parseResult.OK)
	}
	if parseResult.StateVersion != 42 {
		parseT.Fatalf("expected stateVersion 42, got %d", parseResult.StateVersion)
	}
}

func TestJSONAPIListsSessionsAndRelaysCommand(parseT *testing.T) {
	parseHub, parseServer := newTestHub(parseT)
	defer parseServer.Close()

	parseConn, parseSessID := dialAgentSession(parseT, parseServer, parseHub, "app-api", "build-api", []string{"bridge.snapshot"})
	defer parseConn.Close()

	go func() {
		for {
			_, parseRaw, parseReadErr := parseConn.ReadMessage()
			if parseReadErr != nil {
				return
			}
			parseEnv, parseParseErr := agentbridge.ParseEnvelope(string(parseRaw))
			if parseParseErr != nil || parseEnv.Kind != agentbridge.KindCommand {
				continue
			}
			parseAck := agentbridge.BuildAckEnvelope(2, parseSessID, parseEnv.Seq, 77, json.RawMessage(`{"ok":true}`))
			parseAckJSON, _ := agentbridge.FormatEnvelopeJSON(parseAck)
			_ = parseConn.WriteMessage(websocket.TextMessage, []byte(parseAckJSON))
		}
	}()

	parseSessionsResp, parseGetErr := http.Get(parseServer.URL + "/__gwc-agent/sessions?token=" + parseHub.Token())
	if parseGetErr != nil {
		parseT.Fatalf("get sessions: %v", parseGetErr)
	}
	defer parseSessionsResp.Body.Close()
	if parseSessionsResp.StatusCode != http.StatusOK {
		parseT.Fatalf("sessions status = %d", parseSessionsResp.StatusCode)
	}
	var parseSessions apiSessionsResponse
	if parseErr := json.NewDecoder(parseSessionsResp.Body).Decode(&parseSessions); parseErr != nil {
		parseT.Fatalf("decode sessions: %v", parseErr)
	}
	if len(parseSessions.Sessions) != 1 || parseSessions.Sessions[0].ID != parseSessID {
		parseT.Fatalf("sessions = %#v, want %s", parseSessions.Sessions, parseSessID)
	}

	parseBody := []byte(`{"name":"bridge.snapshot","payload":{},"timeoutMs":1000}`)
	parseCommandResp, parsePostErr := http.Post(parseServer.URL+"/__gwc-agent/command?token="+parseHub.Token(), "application/json", bytes.NewReader(parseBody))
	if parsePostErr != nil {
		parseT.Fatalf("post command: %v", parsePostErr)
	}
	defer parseCommandResp.Body.Close()
	if parseCommandResp.StatusCode != http.StatusOK {
		parseT.Fatalf("command status = %d", parseCommandResp.StatusCode)
	}
	var parseCommand apiCommandResponse
	if parseErr := json.NewDecoder(parseCommandResp.Body).Decode(&parseCommand); parseErr != nil {
		parseT.Fatalf("decode command: %v", parseErr)
	}
	if parseCommand.Ack.StateVersion != 77 || parseCommand.Ack.OK == nil || !*parseCommand.Ack.OK {
		parseT.Fatalf("ack = %#v", parseCommand.Ack)
	}
}

func TestWriteLeaseRequiredForMutatingAPICommands(parseT *testing.T) {
	parseHub, parseServer := newTestHub(parseT)
	defer parseServer.Close()

	parseConn, parseSessID := dialAgentSession(parseT, parseServer, parseHub, "app-lease", "build-lease", []string{"bridge.set-atom"})
	defer parseConn.Close()

	go func() {
		for {
			_, parseRaw, parseReadErr := parseConn.ReadMessage()
			if parseReadErr != nil {
				return
			}
			parseEnv, parseParseErr := agentbridge.ParseEnvelope(string(parseRaw))
			if parseParseErr != nil || parseEnv.Kind != agentbridge.KindCommand {
				continue
			}
			parseAck := agentbridge.BuildAckEnvelope(parseEnv.Seq+100, parseSessID, parseEnv.Seq, 88, json.RawMessage(`{"ok":true}`))
			parseAckJSON, _ := agentbridge.FormatEnvelopeJSON(parseAck)
			_ = parseConn.WriteMessage(websocket.TextMessage, []byte(parseAckJSON))
		}
	}()

	parseBody := []byte(`{"session":"` + parseSessID + `","name":"bridge.set-atom","payload":{"id":"x","value":1},"timeoutMs":1000}`)
	parseResp := postAgentHubRaw(parseT, parseServer.URL+"/__gwc-agent/command?token="+parseHub.Token(), parseBody)
	if parseResp.StatusCode != http.StatusForbidden {
		parseT.Fatalf("expected mutating command without lease to fail with 403, got %d", parseResp.StatusCode)
	}
	parseResp.Body.Close()

	parseLeaseBody := []byte(`{"session":"` + parseSessID + `","action":"acquire","holder":"agent-a"}`)
	parseLeaseResp := postAgentHubRaw(parseT, parseServer.URL+"/__gwc-agent/lease?token="+parseHub.Token(), parseLeaseBody)
	defer parseLeaseResp.Body.Close()
	if parseLeaseResp.StatusCode != http.StatusOK {
		parseT.Fatalf("expected lease acquire OK, got %d", parseLeaseResp.StatusCode)
	}
	var parseLease apiLeaseResponse
	if parseErr := json.NewDecoder(parseLeaseResp.Body).Decode(&parseLease); parseErr != nil {
		parseT.Fatalf("decode lease: %v", parseErr)
	}
	if parseLease.Lease.Holder != "agent-a" || parseLease.Lease.ExpiresAt.IsZero() {
		parseT.Fatalf("unexpected lease response: %#v", parseLease)
	}

	parseWrongHolder := []byte(`{"session":"` + parseSessID + `","name":"bridge.set-atom","leaseHolder":"agent-b","payload":{"id":"x","value":2},"timeoutMs":1000}`)
	parseWrongResp := postAgentHubRaw(parseT, parseServer.URL+"/__gwc-agent/command?token="+parseHub.Token(), parseWrongHolder)
	if parseWrongResp.StatusCode != http.StatusForbidden {
		parseT.Fatalf("expected wrong holder to fail with 403, got %d", parseWrongResp.StatusCode)
	}
	parseWrongResp.Body.Close()

	parseAllowed := []byte(`{"session":"` + parseSessID + `","name":"bridge.set-atom","leaseHolder":"agent-a","payload":{"id":"x","value":3},"timeoutMs":1000}`)
	parseAllowedResp := postAgentHubRaw(parseT, parseServer.URL+"/__gwc-agent/command?token="+parseHub.Token(), parseAllowed)
	defer parseAllowedResp.Body.Close()
	if parseAllowedResp.StatusCode != http.StatusOK {
		parseT.Fatalf("expected leased command OK, got %d", parseAllowedResp.StatusCode)
	}

	parseStealMissing := []byte(`{"session":"` + parseSessID + `","action":"steal","holder":"agent-b"}`)
	parseStealMissingResp := postAgentHubRaw(parseT, parseServer.URL+"/__gwc-agent/lease?token="+parseHub.Token(), parseStealMissing)
	if parseStealMissingResp.StatusCode != http.StatusConflict {
		parseT.Fatalf("expected steal without flag to fail with 409, got %d", parseStealMissingResp.StatusCode)
	}
	parseStealMissingResp.Body.Close()

	parseSteal := []byte(`{"session":"` + parseSessID + `","action":"steal","holder":"agent-b","steal":true}`)
	parseStealResp := postAgentHubRaw(parseT, parseServer.URL+"/__gwc-agent/lease?token="+parseHub.Token(), parseSteal)
	defer parseStealResp.Body.Close()
	if parseStealResp.StatusCode != http.StatusOK {
		parseT.Fatalf("expected explicit steal OK, got %d", parseStealResp.StatusCode)
	}
	parseEvents := parseHub.ReadEvents(parseSessID, 10)
	if len(parseEvents) == 0 || parseEvents[len(parseEvents)-1].Name != "lease.stolen" {
		parseT.Fatalf("expected lease.stolen notification event, got %#v", parseEvents)
	}
}

func TestLogsRecordingAndCrashReportAPI(parseT *testing.T) {
	parseHub, parseServer := newTestHub(parseT)
	defer parseServer.Close()

	parseConn, parseSessID := dialAgentSession(parseT, parseServer, parseHub, "app-observe", "build-observe", []string{"bridge.snapshot", "bridge.query"})

	go func() {
		for {
			_, parseRaw, parseReadErr := parseConn.ReadMessage()
			if parseReadErr != nil {
				return
			}
			parseEnv, parseParseErr := agentbridge.ParseEnvelope(string(parseRaw))
			if parseParseErr != nil || parseEnv.Kind != agentbridge.KindCommand {
				continue
			}
			parsePayload := json.RawMessage(`{"ok":true}`)
			if parseEnv.Name == "bridge.snapshot" {
				parsePayload = json.RawMessage(`{"root":{"agentRef":"root-1","token":"snapshot-secret"}}`)
			}
			parseAck := agentbridge.BuildAckEnvelope(parseEnv.Seq+200, parseSessID, parseEnv.Seq, parseEnv.Seq, parsePayload)
			parseAckJSON, _ := agentbridge.FormatEnvelopeJSON(parseAck)
			_ = parseConn.WriteMessage(websocket.TextMessage, []byte(parseAckJSON))
		}
	}()

	parseCtx, parseCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer parseCancel()
	if _, parseErr := parseHub.SendCommand(parseCtx, parseSessID, "bridge.snapshot", json.RawMessage(`{"maxDepth":2}`)); parseErr != nil {
		parseT.Fatalf("snapshot command: %v", parseErr)
	}
	if _, parseErr := parseHub.SendCommand(parseCtx, parseSessID, "bridge.query", json.RawMessage(`{"role":"button"}`)); parseErr != nil {
		parseT.Fatalf("query command: %v", parseErr)
	}

	parseDiag := agentbridge.BuildEventEnvelope(99, parseSessID, "runtime.diagnostic", json.RawMessage(`{"message":"panic contained","token":"diag-secret","nested":{"authorization":"Bearer abc","safe":true}}`))
	parseDiagJSON, _ := agentbridge.FormatEnvelopeJSON(parseDiag)
	if parseErr := parseConn.WriteMessage(websocket.TextMessage, []byte(parseDiagJSON)); parseErr != nil {
		parseT.Fatalf("write diagnostic event: %v", parseErr)
	}

	parseLogsURL := parseServer.URL + "/__gwc-agent/logs?token=" + parseHub.Token() + "&session=" + parseSessID
	var parseLogs apiLogsResponse
	waitForJSON(parseT, parseLogsURL, &parseLogs, func() bool { return len(parseLogs.Logs) == 1 })
	if parseLogs.Logs[0].Name != "runtime.diagnostic" {
		parseT.Fatalf("unexpected logs response: %#v", parseLogs)
	}
	if bytes.Contains(parseLogs.Logs[0].Payload, []byte("diag-secret")) || bytes.Contains(parseLogs.Logs[0].Payload, []byte("authorization")) {
		parseT.Fatalf("expected logs payload to redact sensitive fields, got %s", parseLogs.Logs[0].Payload)
	}

	parseRecordingURL := parseServer.URL + "/__gwc-agent/recording?token=" + parseHub.Token() + "&session=" + parseSessID
	var parseRecording apiRecordingResponse
	getAgentHubJSON(parseT, parseRecordingURL, &parseRecording)
	if len(parseRecording.Records) != 2 || parseRecording.Records[0].Name != "bridge.snapshot" || parseRecording.Records[1].Name != "bridge.query" {
		parseT.Fatalf("unexpected recording: %#v", parseRecording)
	}

	parseConn.Close()
	waitForState(parseT, parseHub, parseSessID, StateCrashed)
	parseCrashURL := parseServer.URL + "/__gwc-agent/crash-report?token=" + parseHub.Token() + "&session=" + parseSessID
	var parseCrash apiCrashReportResponse
	getAgentHubJSON(parseT, parseCrashURL, &parseCrash)
	if parseCrash.Report == nil {
		parseT.Fatal("expected crash report")
	}
	if !bytes.Contains(parseCrash.Report.LastSnapshot, []byte("root-1")) {
		parseT.Fatalf("expected last snapshot in crash report, got %s", parseCrash.Report.LastSnapshot)
	}
	if bytes.Contains(parseCrash.Report.LastSnapshot, []byte("snapshot-secret")) || bytes.Contains(parseCrash.Report.LastSnapshot, []byte("token")) {
		parseT.Fatalf("expected crash snapshot to redact sensitive fields, got %s", parseCrash.Report.LastSnapshot)
	}
	if len(parseCrash.Report.Logs) != 1 || len(parseCrash.Report.Commands) != 2 {
		parseT.Fatalf("unexpected crash report: %#v", parseCrash.Report)
	}

	parseClearReq, parseErr := http.NewRequest(http.MethodDelete, parseRecordingURL, nil)
	if parseErr != nil {
		parseT.Fatalf("build delete recording request: %v", parseErr)
	}
	parseClearResp, parseErr := http.DefaultClient.Do(parseClearReq)
	if parseErr != nil {
		parseT.Fatalf("delete recording: %v", parseErr)
	}
	parseClearResp.Body.Close()
	if parseClearResp.StatusCode != http.StatusOK {
		parseT.Fatalf("expected clear recording OK, got %d", parseClearResp.StatusCode)
	}
	var parseCleared apiRecordingResponse
	getAgentHubJSON(parseT, parseRecordingURL, &parseCleared)
	if len(parseCleared.Records) != 0 {
		parseT.Fatalf("expected cleared recording, got %#v", parseCleared)
	}
}

func TestReloadAndSuccessorEndpoints(parseT *testing.T) {
	parseHub, parseServer := newTestHub(parseT)
	defer parseServer.Close()

	parseConnA, parseSessIDA := dialAgentSession(parseT, parseServer, parseHub, "app-rebuild", "build-old", nil)
	defer parseConnA.Close()

	parseReloadBody := []byte(`{"session":"` + parseSessIDA + `","buildId":"build-new","artifact":"app.wasm"}`)
	parseReloadResp := postAgentHubRaw(parseT, parseServer.URL+"/__gwc-agent/reload?token="+parseHub.Token(), parseReloadBody)
	defer parseReloadResp.Body.Close()
	if parseReloadResp.StatusCode != http.StatusOK {
		parseT.Fatalf("expected reload OK, got %d", parseReloadResp.StatusCode)
	}
	waitForState(parseT, parseHub, parseSessIDA, StateReloading)

	parseConnB, parseSessIDB := dialAgentSession(parseT, parseServer, parseHub, "app-rebuild", "build-new", nil)
	defer parseConnB.Close()

	parseSuccessorURL := parseServer.URL + "/__gwc-agent/successor?token=" + parseHub.Token() + "&session=" + parseSessIDA + "&buildId=build-new&timeoutMs=1000"
	var parseSuccessor apiSuccessorResponse
	getAgentHubJSON(parseT, parseSuccessorURL, &parseSuccessor)
	if parseSuccessor.SessionID != parseSessIDB || parseSuccessor.PredecessorID != parseSessIDA || parseSuccessor.BuildID != "build-new" {
		parseT.Fatalf("unexpected successor response: %#v", parseSuccessor)
	}
}

// --- (e) socket kill -> crashed; MarkReloadExpected then kill -> reloading,
//     and a reconnect links PredecessorID ---

func TestSocketDeathTransitions(parseT *testing.T) {
	// Part 1: plain kill -> crashed.
	parseHubA, parseServerA := newTestHub(parseT)
	defer parseServerA.Close()

	parseConnA, parseSessIDA := dialAgentSession(parseT, parseServerA, parseHubA, "app-death", "build-1", nil)
	parseConnA.Close() // kill without MarkReloadExpected

	parseDeadlineA := time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadlineA) {
		parseSess := parseHubA.findSession(parseSessIDA)
		if parseSess != nil {
			parseSess.mu.Lock()
			parseState := parseSess.State
			parseSess.mu.Unlock()
			if parseState == StateCrashed {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	parseFoundA := parseHubA.findSession(parseSessIDA)
	if parseFoundA == nil {
		parseT.Fatal("session A not found after kill")
	}
	parseFoundA.mu.Lock()
	parseStateA := parseFoundA.State
	parseFoundA.mu.Unlock()
	if parseStateA != StateCrashed {
		parseT.Fatalf("expected state crashed, got %s", parseStateA)
	}

	// Part 2: MarkReloadExpected then kill -> reloading.
	parseHubB, parseServerB := newTestHub(parseT)
	defer parseServerB.Close()

	parseConnB, parseSessIDB := dialAgentSession(parseT, parseServerB, parseHubB, "app-reload", "build-2", nil)
	parseHubB.MarkReloadExpected(parseSessIDB)
	parseConnB.Close()

	parseDeadlineB := time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadlineB) {
		parseSessB := parseHubB.findSession(parseSessIDB)
		if parseSessB != nil {
			parseSessB.mu.Lock()
			parseStateB := parseSessB.State
			parseSessB.mu.Unlock()
			if parseStateB == StateReloading || parseStateB == StateCrashed {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	parseFoundB := parseHubB.findSession(parseSessIDB)
	if parseFoundB == nil {
		parseT.Fatal("session B not found")
	}
	parseFoundB.mu.Lock()
	parseStateB := parseFoundB.State
	parseFoundB.mu.Unlock()
	if parseStateB != StateReloading {
		parseT.Fatalf("expected state reloading after MarkReloadExpected+kill, got %s", parseStateB)
	}

	// Part 3: reconnect after reloading links PredecessorID.
	parseConnC, parseSessIDC := dialAgentSession(parseT, parseServerB, parseHubB, "app-reload", "build-3", nil)
	defer parseConnC.Close()

	parseFoundC := parseHubB.findSession(parseSessIDC)
	if parseFoundC == nil {
		parseT.Fatal("successor session not found")
	}
	parseFoundC.mu.Lock()
	parsePredID := parseFoundC.PredecessorID
	parseFoundC.mu.Unlock()
	if parsePredID != parseSessIDB {
		parseT.Fatalf("expected PredecessorID=%q, got %q", parseSessIDB, parsePredID)
	}
}

// --- (f) frames for session A never reach session B ---

func TestFramesDoNotCrossSessionBoundary(parseT *testing.T) {
	parseHub, parseServer := newTestHub(parseT)
	defer parseServer.Close()

	parseConnA, parseSessIDA := dialAgentSession(parseT, parseServer, parseHub, "app-fa", "build-1", []string{"cmd"})
	defer parseConnA.Close()

	parseConnB, _ := dialAgentSession(parseT, parseServer, parseHub, "app-fb", "build-1", []string{"cmd"})
	defer parseConnB.Close()

	// Set a read deadline on B so we know if it receives anything.
	if parseSetErr := parseConnB.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); parseSetErr != nil {
		parseT.Fatalf("set read deadline on B: %v", parseSetErr)
	}

	// Send command to A with a fake ack loop on A only.
	var parseFakeSeqA atomic.Uint64
	go func() {
		for {
			_, parseRaw, parseReadErr := parseConnA.ReadMessage()
			if parseReadErr != nil {
				return
			}
			parseEnv, parseParseErr := agentbridge.ParseEnvelope(string(parseRaw))
			if parseParseErr != nil || parseEnv.Kind != agentbridge.KindCommand {
				continue
			}
			parseSeqF := parseFakeSeqA.Add(1)
			parseAck := agentbridge.BuildAckEnvelope(parseSeqF, parseSessIDA, parseEnv.Seq, 0, nil)
			parseAckJSON, _ := agentbridge.FormatEnvelopeJSON(parseAck)
			_ = parseConnA.WriteMessage(websocket.TextMessage, []byte(parseAckJSON))
		}
	}()

	parseCtx, parseCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer parseCancel()
	_, parseSendErr := parseHub.SendCommand(parseCtx, parseSessIDA, "cmd", nil)
	if parseSendErr != nil {
		parseT.Fatalf("SendCommand to A failed: %v", parseSendErr)
	}

	// B must receive nothing (deadline will fire).
	_, _, parseReadErrB := parseConnB.ReadMessage()
	if parseReadErrB == nil {
		parseT.Fatal("expected session B to receive no frames, but got a message")
	}
	// A timeout error is the expected outcome.
	if !strings.Contains(parseReadErrB.Error(), "timeout") && !strings.Contains(parseReadErrB.Error(), "deadline") {
		// Could also be a close frame – acceptable as long as it's not a data frame.
		parseT.Logf("session B read ended with non-timeout error (acceptable): %v", parseReadErrB)
	}
}

// --- (g) non-loopback RemoteAddr refused ---

func TestNonLoopbackRemoteAddrRefused(parseT *testing.T) {
	parseHub, parseHubErr := NewAgentHub()
	if parseHubErr != nil {
		parseT.Fatalf("NewAgentHub: %v", parseHubErr)
	}

	// Build a fake request whose RemoteAddr is a non-loopback address.
	parseReq := httptest.NewRequest(http.MethodGet, "/gwc-agent?token="+parseHub.Token(), nil)
	parseReq.RemoteAddr = "10.0.0.1:12345"
	parseRec := httptest.NewRecorder()
	parseHub.ServeHTTP(parseRec, parseReq)
	if parseRec.Code != http.StatusForbidden {
		parseT.Fatalf("expected 403 for non-loopback RemoteAddr, got %d", parseRec.Code)
	}
}

// --- extra: RouteAgentHub wires the hub into a mux ---

func TestRouteAgentHubRegistersEndpoint(parseT *testing.T) {
	parseHub, parseHubErr := NewAgentHub()
	if parseHubErr != nil {
		parseT.Fatalf("NewAgentHub: %v", parseHubErr)
	}
	parseMux := http.NewServeMux()
	RouteAgentHub(parseMux, parseHub)

	// A request with bad token should hit the handler and return 403.
	parseReq := httptest.NewRequest(http.MethodGet, "/gwc-agent?token=bad", nil)
	parseReq.RemoteAddr = "127.0.0.1:9999"
	parseRec := httptest.NewRecorder()
	parseMux.ServeHTTP(parseRec, parseReq)
	if parseRec.Code != http.StatusForbidden {
		parseT.Fatalf("expected 403 from mux-registered hub, got %d", parseRec.Code)
	}
}

// --- extra: NewAgentHub mints a token ---

func TestNewAgentHubMintsToken(parseT *testing.T) {
	parseHubA, parseErrA := NewAgentHub()
	if parseErrA != nil {
		parseT.Fatalf("NewAgentHub: %v", parseErrA)
	}
	parseHubB, parseErrB := NewAgentHub()
	if parseErrB != nil {
		parseT.Fatalf("NewAgentHub second: %v", parseErrB)
	}
	if parseHubA.Token() == "" {
		parseT.Fatal("expected non-empty token")
	}
	if parseHubA.Token() == parseHubB.Token() {
		parseT.Fatal("expected distinct tokens for distinct hubs")
	}
}

// --- extra: eventRing ring-buffer semantics ---

func TestEventRingDropsOldestOnOverflow(parseT *testing.T) {
	parseRing := newEventRing(3)
	for parseIdx := 1; parseIdx <= 4; parseIdx++ {
		parseRing.push(agentbridge.BuildEventEnvelope(uint64(parseIdx), "s1", "ev", json.RawMessage(fmt.Sprintf(`{"i":%d}`, parseIdx))))
	}
	if parseRing.dropCount() != 1 {
		parseT.Fatalf("expected 1 drop, got %d", parseRing.dropCount())
	}
	parseDrained := parseRing.drain(10)
	if len(parseDrained) != 3 {
		parseT.Fatalf("expected 3 events, got %d", len(parseDrained))
	}
	// Oldest (seq=1) dropped; remaining are seq 2,3,4.
	if parseDrained[0].Seq != 2 {
		parseT.Fatalf("expected oldest surviving seq=2, got seq=%d", parseDrained[0].Seq)
	}
}

// --- extra: originAllowed tests ---

func TestOriginAllowed(parseT *testing.T) {
	parseHub, _ := NewAgentHub()

	parseAllowed := []string{
		"",
		"http://localhost:8080",
		"http://127.0.0.1:8080",
		"http://[::1]:8080",
	}
	for _, parseOrigin := range parseAllowed {
		parseReq := httptest.NewRequest(http.MethodGet, "/gwc-agent", nil)
		if parseOrigin != "" {
			parseReq.Header.Set("Origin", parseOrigin)
		}
		if !parseHub.originAllowed(parseReq) {
			parseT.Fatalf("expected origin %q to be allowed", parseOrigin)
		}
	}

	parseRejected := []string{
		"https://evil.example.com",
		"http://attacker.localhost:8080",
	}
	for _, parseOrigin := range parseRejected {
		parseReq := httptest.NewRequest(http.MethodGet, "/gwc-agent", nil)
		parseReq.Header.Set("Origin", parseOrigin)
		if parseHub.originAllowed(parseReq) {
			parseT.Fatalf("expected origin %q to be rejected", parseOrigin)
		}
	}
}

// --- extra: isLoopback helper ---

func TestIsLoopback(parseT *testing.T) {
	parseHub, _ := NewAgentHub()
	parseLoopbackCases := []string{"127.0.0.1:9999", "[::1]:9999", "localhost:9999"}
	for _, parseAddr := range parseLoopbackCases {
		// Resolve localhost for the test.
		parseHost, parsePort, _ := net.SplitHostPort(parseAddr)
		if parseHost == "localhost" {
			parseResolved, parseResErr := net.ResolveIPAddr("ip", "localhost")
			if parseResErr == nil && parseResolved.IP.IsLoopback() {
				parseAddr = net.JoinHostPort(parseResolved.IP.String(), parsePort)
			}
		}
		if !parseHub.isLoopback(parseAddr) {
			parseT.Fatalf("expected %q to be loopback", parseAddr)
		}
	}
	parseNonLoopback := []string{"10.0.0.1:80", "192.168.1.1:80", "8.8.8.8:53"}
	for _, parseAddr := range parseNonLoopback {
		if parseHub.isLoopback(parseAddr) {
			parseT.Fatalf("expected %q to NOT be loopback", parseAddr)
		}
	}
}

func postAgentHubRaw(parseT *testing.T, parseURL string, parseBody []byte) *http.Response {
	parseT.Helper()
	parseResp, parseErr := http.Post(parseURL, "application/json", bytes.NewReader(parseBody))
	if parseErr != nil {
		parseT.Fatalf("post %s: %v", parseURL, parseErr)
	}
	return parseResp
}

func getAgentHubJSON(parseT *testing.T, parseURL string, parseOut any) {
	parseT.Helper()
	parseResp, parseErr := http.Get(parseURL)
	if parseErr != nil {
		parseT.Fatalf("get %s: %v", parseURL, parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusOK {
		parseBody, _ := io.ReadAll(parseResp.Body)
		parseT.Fatalf("get %s status=%d body=%s", parseURL, parseResp.StatusCode, string(parseBody))
	}
	if parseErr := json.NewDecoder(parseResp.Body).Decode(parseOut); parseErr != nil {
		parseT.Fatalf("decode %s: %v", parseURL, parseErr)
	}
}

func waitForJSON(parseT *testing.T, parseURL string, parseOut any, parseReady func() bool) {
	parseT.Helper()
	parseDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) {
		getAgentHubJSON(parseT, parseURL, parseOut)
		if parseReady() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	parseT.Fatalf("condition never became true for %s", parseURL)
}

func waitForState(parseT *testing.T, parseHub *AgentHub, parseSessionID string, parseWant SessionState) {
	parseT.Helper()
	parseDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) {
		parseSess := parseHub.findSession(parseSessionID)
		if parseSess != nil {
			parseSess.mu.Lock()
			parseState := parseSess.State
			parseSess.mu.Unlock()
			if parseState == parseWant {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	parseT.Fatalf("session %s never reached state %s", parseSessionID, parseWant)
}
