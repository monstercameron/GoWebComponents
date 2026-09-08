// Package agenthub hosts the /gwc-agent WebSocket endpoint for the dev
// tooling. The hub mints a per-run token, validates every upgrade against
// that token, tracks app sessions through hello/death/reload, and exposes
// a Go API for relaying command/ack/event frames to/from connected apps.
//
// Security model: the endpoint refuses non-loopback RemoteAddr and
// cross-origin upgrades (Origin must be empty or a localhost variant),
// so a visited website cannot drive the agent socket while the dev server
// is running.
package agenthub

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/monstercameron/GoWebComponents/v6/agentbridge"
)

const (
	// eventRingSize is the per-session ring buffer capacity for KindEvent frames.
	eventRingSize = 512
	// recordingRingSize is the retained per-session command journal capacity.
	recordingRingSize = 256
	// leaseDuration is the write lease TTL refreshed on acquire or steal.
	leaseDuration = 5 * time.Minute
)

// WebSocket keepalive tunables. After the hello handshake the read deadline was
// previously cleared, so a half-open peer (client crash without TCP FIN, network
// partition) left ReadMessage blocked forever — a leaked session goroutine plus a
// stale entry in hub.sessions. The hub now pings every agentPingPeriod and drops a
// connection whose read (data or pong) goes quiet for agentPongWait. Vars, not
// consts, so tests can shrink them.
var (
	agentPongWait   = 60 * time.Second
	agentPingPeriod = 54 * time.Second // must be < agentPongWait
	agentWriteWait  = 10 * time.Second
)

// AgentHub is the server-side session registry and WebSocket handler for
// /gwc-agent. Create one with NewAgentHub; the zero value is not valid.
type AgentHub struct {
	// token is the per-run secret every upgrade must present as ?token=...
	token string

	mu       sync.Mutex
	sessions []*Session // append-only; newest last
	nextID   atomic.Uint64

	upgrader websocket.Upgrader
}

// NewAgentHub constructs an AgentHub, minting a fresh random token.
// The token is available via hub.Token() and should be injected into
// the served page so the app can present it on connect.
func NewAgentHub() (*AgentHub, error) {
	parseBytes := make([]byte, 16)
	if _, parseErr := rand.Read(parseBytes); parseErr != nil {
		return nil, fmt.Errorf("agenthub: mint token: %w", parseErr)
	}
	parseHub := &AgentHub{
		token: hex.EncodeToString(parseBytes),
	}
	parseHub.upgrader = websocket.Upgrader{
		CheckOrigin: parseHub.originAllowed,
	}
	return parseHub, nil
}

// Token returns the per-run secret that callers must present as ?token=... on
// every /gwc-agent upgrade request.
func (parseHub *AgentHub) Token() string {
	return parseHub.token
}

// ServeHTTP is the http.Handler for the /gwc-agent endpoint. It enforces
// loopback-only access, origin validation, token match, upgrades the
// connection to WebSocket, reads the hello frame, and registers the session.
func (parseHub *AgentHub) ServeHTTP(parseW http.ResponseWriter, parseR *http.Request) {
	switch parseR.URL.Path {
	case "/gwc-agent", "":
		parseHub.serveWebSocket(parseW, parseR)
	case "/__gwc-agent/sessions":
		parseHub.handleAPISessions(parseW, parseR)
	case "/__gwc-agent/command":
		parseHub.handleAPICommand(parseW, parseR)
	case "/__gwc-agent/logs":
		parseHub.handleAPILogs(parseW, parseR)
	case "/__gwc-agent/crash-report":
		parseHub.handleAPICrashReport(parseW, parseR)
	case "/__gwc-agent/recording":
		parseHub.handleAPIRecording(parseW, parseR)
	case "/__gwc-agent/lease":
		parseHub.handleAPILease(parseW, parseR)
	case "/__gwc-agent/reload":
		parseHub.handleAPIReload(parseW, parseR)
	case "/__gwc-agent/successor":
		parseHub.handleAPISuccessor(parseW, parseR)
	default:
		http.NotFound(parseW, parseR)
	}
}

func (parseHub *AgentHub) serveWebSocket(parseW http.ResponseWriter, parseR *http.Request) {
	// Guard: only loopback callers may reach the agent socket.
	if !parseHub.isLoopback(parseR.RemoteAddr) {
		http.Error(parseW, "agent endpoint is localhost-only", http.StatusForbidden)
		return
	}

	// Guard: token must match before any upgrade attempt.
	parseQueryToken := strings.TrimSpace(parseR.URL.Query().Get("token"))
	if !parseHub.tokenMatches(parseQueryToken) {
		http.Error(parseW, "missing or invalid agent token", http.StatusForbidden)
		return
	}

	// Upgrade (upgrader.CheckOrigin enforces origin policy).
	parseConn, parseUpgradeErr := parseHub.upgrader.Upgrade(parseW, parseR, nil)
	if parseUpgradeErr != nil {
		// upgrader already wrote the HTTP error response.
		return
	}

	parseHub.serveSession(parseConn)
}

type apiCommandRequest struct {
	Session     string          `json:"session,omitempty"`
	Name        string          `json:"name"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	TimeoutMs   int             `json:"timeoutMs,omitempty"`
	LeaseHolder string          `json:"leaseHolder,omitempty"`
}

type apiCommandResponse struct {
	Ack agentbridge.Envelope `json:"ack"`
}

type apiSessionsResponse struct {
	Sessions []SessionInfo `json:"sessions"`
}

type apiLogsResponse struct {
	Session string                 `json:"session"`
	Logs    []agentbridge.Envelope `json:"logs"`
	Dropped int                    `json:"dropped"`
}

type apiCrashReportResponse struct {
	Session string       `json:"session"`
	Report  *CrashReport `json:"report,omitempty"`
}

type apiRecordingResponse struct {
	Session string          `json:"session"`
	Records []CommandRecord `json:"records"`
	Dropped int             `json:"dropped"`
}

type apiLeaseRequest struct {
	Session string `json:"session,omitempty"`
	Action  string `json:"action"`
	Holder  string `json:"holder"`
	Steal   bool   `json:"steal,omitempty"`
}

type apiLeaseResponse struct {
	Session string     `json:"session"`
	Lease   WriteLease `json:"lease"`
}

type apiReloadRequest struct {
	Session  string `json:"session"`
	BuildID  string `json:"buildId,omitempty"`
	Artifact string `json:"artifact,omitempty"`
}

type apiReloadResponse struct {
	OK      bool   `json:"ok"`
	Session string `json:"session"`
	BuildID string `json:"buildId,omitempty"`
}

type apiSuccessorResponse struct {
	SessionID     string `json:"sessionId"`
	PredecessorID string `json:"predecessorId"`
	BuildID       string `json:"buildId,omitempty"`
}

func (parseHub *AgentHub) handleAPISessions(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method != http.MethodGet {
		parseW.Header().Set("Allow", http.MethodGet)
		http.Error(parseW, "sessions endpoint only supports GET", http.StatusMethodNotAllowed)
		return
	}
	if !parseHub.authorizeAPI(parseW, parseR) {
		return
	}
	parseHub.writeAPIJSON(parseW, apiSessionsResponse{Sessions: parseHub.ListSessions()})
}

func (parseHub *AgentHub) handleAPICommand(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method != http.MethodPost {
		parseW.Header().Set("Allow", http.MethodPost)
		http.Error(parseW, "command endpoint only supports POST", http.StatusMethodNotAllowed)
		return
	}
	if !parseHub.authorizeAPI(parseW, parseR) {
		return
	}
	defer parseR.Body.Close()
	var parseReq apiCommandRequest
	if parseErr := json.NewDecoder(parseR.Body).Decode(&parseReq); parseErr != nil {
		http.Error(parseW, "invalid command request JSON", http.StatusBadRequest)
		return
	}
	parseReq.Name = strings.TrimSpace(parseReq.Name)
	if parseReq.Name == "" {
		http.Error(parseW, "command name is required", http.StatusBadRequest)
		return
	}
	parseSessionID := strings.TrimSpace(parseReq.Session)
	// Fall forward to the latest active session when none was given OR the
	// requested one is no longer active (e.g. the page reloaded into a new
	// session). This keeps agent commands working across normal reloads instead
	// of failing against a stale/crashed session id.
	if parseSessionID == "" || !parseHub.isSessionActive(parseSessionID) {
		if parseLatest := parseHub.latestActiveSessionID(); parseLatest != "" {
			parseSessionID = parseLatest
		}
	}
	if parseSessionID == "" {
		http.Error(parseW, "no active agent session connected", http.StatusConflict)
		return
	}
	if commandRequiresLease(parseReq.Name, parseReq.Payload) {
		if parseErr := parseHub.requireWriteLease(parseSessionID, parseReq.LeaseHolder); parseErr != nil {
			http.Error(parseW, parseErr.Error(), http.StatusForbidden)
			return
		}
	}
	parseTimeout := time.Duration(parseReq.TimeoutMs) * time.Millisecond
	if parseTimeout <= 0 {
		parseTimeout = 5 * time.Second
	}
	parseCtx, parseCancel := context.WithTimeout(parseR.Context(), parseTimeout)
	defer parseCancel()
	parseAck, parseErr := parseHub.SendCommand(parseCtx, parseSessionID, parseReq.Name, parseReq.Payload)
	if parseErr != nil {
		http.Error(parseW, parseErr.Error(), http.StatusBadGateway)
		return
	}
	parseHub.writeAPIJSON(parseW, apiCommandResponse{Ack: parseAck})
}

func (parseHub *AgentHub) handleAPILogs(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method != http.MethodGet {
		parseW.Header().Set("Allow", http.MethodGet)
		http.Error(parseW, "logs endpoint only supports GET", http.StatusMethodNotAllowed)
		return
	}
	if !parseHub.authorizeAPI(parseW, parseR) {
		return
	}
	parseSess := parseHub.sessionFromRequest(parseR)
	if parseSess == nil {
		http.Error(parseW, "agent session not found", http.StatusNotFound)
		return
	}
	parseMax := parsePositiveInt(parseR.URL.Query().Get("max"))
	parseHub.writeAPIJSON(parseW, apiLogsResponse{
		Session: parseSess.ID,
		Logs:    parseSess.logs.snapshot(parseMax),
		Dropped: parseSess.logs.dropCount(),
	})
}

func (parseHub *AgentHub) handleAPICrashReport(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method != http.MethodGet {
		parseW.Header().Set("Allow", http.MethodGet)
		http.Error(parseW, "crash-report endpoint only supports GET", http.StatusMethodNotAllowed)
		return
	}
	if !parseHub.authorizeAPI(parseW, parseR) {
		return
	}
	parseSess := parseHub.sessionFromRequest(parseR)
	if parseSess == nil {
		http.Error(parseW, "agent session not found", http.StatusNotFound)
		return
	}
	parseSess.mu.Lock()
	parseReport := parseSess.crashReport
	parseSess.mu.Unlock()
	parseHub.writeAPIJSON(parseW, apiCrashReportResponse{Session: parseSess.ID, Report: parseReport})
}

func (parseHub *AgentHub) handleAPIRecording(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method != http.MethodGet && parseR.Method != http.MethodDelete {
		parseW.Header().Set("Allow", "GET, DELETE")
		http.Error(parseW, "recording endpoint only supports GET and DELETE", http.StatusMethodNotAllowed)
		return
	}
	if !parseHub.authorizeAPI(parseW, parseR) {
		return
	}
	parseSess := parseHub.sessionFromRequest(parseR)
	if parseSess == nil {
		http.Error(parseW, "agent session not found", http.StatusNotFound)
		return
	}
	if parseR.Method == http.MethodDelete || parseR.URL.Query().Get("clear") == "1" {
		parseSess.recording.clear()
	}
	parseMax := parsePositiveInt(parseR.URL.Query().Get("max"))
	parseHub.writeAPIJSON(parseW, apiRecordingResponse{
		Session: parseSess.ID,
		Records: parseHub.recordingChain(parseSess, parseMax),
		Dropped: parseSess.recording.dropCount(),
	})
}

func (parseHub *AgentHub) handleAPILease(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method != http.MethodPost {
		parseW.Header().Set("Allow", http.MethodPost)
		http.Error(parseW, "lease endpoint only supports POST", http.StatusMethodNotAllowed)
		return
	}
	if !parseHub.authorizeAPI(parseW, parseR) {
		return
	}
	defer parseR.Body.Close()
	var parseReq apiLeaseRequest
	if parseErr := json.NewDecoder(parseR.Body).Decode(&parseReq); parseErr != nil {
		http.Error(parseW, "invalid lease request JSON", http.StatusBadRequest)
		return
	}
	parseSess := parseHub.sessionFromRequestBody(parseReq.Session)
	if parseSess == nil {
		http.Error(parseW, "agent session not found", http.StatusNotFound)
		return
	}
	parseHolder := strings.TrimSpace(parseReq.Holder)
	if parseHolder == "" {
		http.Error(parseW, "lease holder is required", http.StatusBadRequest)
		return
	}
	parseLease, parseErr := parseHub.updateWriteLease(parseSess, strings.TrimSpace(parseReq.Action), parseHolder, parseReq.Steal)
	if parseErr != nil {
		http.Error(parseW, parseErr.Error(), http.StatusConflict)
		return
	}
	parseHub.writeAPIJSON(parseW, apiLeaseResponse{Session: parseSess.ID, Lease: parseLease})
}

func (parseHub *AgentHub) handleAPIReload(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method != http.MethodPost {
		parseW.Header().Set("Allow", http.MethodPost)
		http.Error(parseW, "reload endpoint only supports POST", http.StatusMethodNotAllowed)
		return
	}
	if !parseHub.authorizeAPI(parseW, parseR) {
		return
	}
	defer parseR.Body.Close()
	var parseReq apiReloadRequest
	if parseErr := json.NewDecoder(parseR.Body).Decode(&parseReq); parseErr != nil {
		http.Error(parseW, "invalid reload request JSON", http.StatusBadRequest)
		return
	}
	parseSessionID := strings.TrimSpace(parseReq.Session)
	if parseSessionID == "" {
		parseSessionID = parseHub.latestActiveSessionID()
	}
	if parseSessionID == "" || parseHub.findSession(parseSessionID) == nil {
		http.Error(parseW, "agent session not found", http.StatusNotFound)
		return
	}
	parseHub.MarkReloadExpected(parseSessionID)
	parseHub.writeAPIJSON(parseW, apiReloadResponse{OK: true, Session: parseSessionID, BuildID: strings.TrimSpace(parseReq.BuildID)})
}

func (parseHub *AgentHub) handleAPISuccessor(parseW http.ResponseWriter, parseR *http.Request) {
	if parseR.Method != http.MethodGet {
		parseW.Header().Set("Allow", http.MethodGet)
		http.Error(parseW, "successor endpoint only supports GET", http.StatusMethodNotAllowed)
		return
	}
	if !parseHub.authorizeAPI(parseW, parseR) {
		return
	}
	parseSessionID := strings.TrimSpace(parseR.URL.Query().Get("session"))
	if parseSessionID == "" {
		http.Error(parseW, "session query parameter is required", http.StatusBadRequest)
		return
	}
	parseBuildID := strings.TrimSpace(parseR.URL.Query().Get("buildId"))
	parseTimeout := time.Duration(parsePositiveInt(parseR.URL.Query().Get("timeoutMs"))) * time.Millisecond
	if parseTimeout <= 0 {
		parseTimeout = 10 * time.Second
	}
	parseDeadline := time.Now().Add(parseTimeout)
	for {
		if parseSuccessor := parseHub.findSuccessor(parseSessionID, parseBuildID); parseSuccessor != nil {
			parseSuccessor.mu.Lock()
			parseResp := apiSuccessorResponse{
				SessionID:     parseSuccessor.ID,
				PredecessorID: parseSuccessor.PredecessorID,
				BuildID:       parseSuccessor.BuildID,
			}
			parseSuccessor.mu.Unlock()
			parseHub.writeAPIJSON(parseW, parseResp)
			return
		}
		if time.Now().After(parseDeadline) {
			http.Error(parseW, "successor session timed out", http.StatusGatewayTimeout)
			return
		}
		select {
		case <-parseR.Context().Done():
			http.Error(parseW, "successor wait cancelled", http.StatusRequestTimeout)
			return
		case <-time.After(25 * time.Millisecond):
		}
	}
}

// maxAgentAPIBodyBytes caps the request body the plain-HTTP agent API will read.
const maxAgentAPIBodyBytes = 4 << 20 // 4 MiB

// maxAgentFrameBytes caps a single inbound WebSocket frame.
const maxAgentFrameBytes = 16 << 20 // 16 MiB

func (parseHub *AgentHub) authorizeAPI(parseW http.ResponseWriter, parseR *http.Request) bool {
	if !parseHub.isLoopback(parseR.RemoteAddr) {
		http.Error(parseW, "agent API is localhost-only", http.StatusForbidden)
		return false
	}
	// Defense-in-depth parity with the WebSocket upgrade path (which enforces
	// originAllowed): reject a present cross-origin Origin so a visited website
	// cannot drive the agent API even if the per-run token leaked via a side
	// channel. Empty Origin (non-browser dev tooling) is still allowed.
	if !parseHub.originAllowed(parseR) {
		http.Error(parseW, "agent API rejects cross-origin requests", http.StatusForbidden)
		return false
	}
	parseToken := strings.TrimSpace(parseR.URL.Query().Get("token"))
	if parseToken == "" {
		parseToken = strings.TrimSpace(parseR.Header.Get("X-GWC-Agent-Token"))
	}
	if !parseHub.tokenMatches(parseToken) {
		http.Error(parseW, "missing or invalid agent token", http.StatusForbidden)
		return false
	}
	// Cap the request body: the API handlers json-decode parseR.Body with no
	// size limit, so an authorized-but-buggy/compromised loopback peer could OOM
	// the dev box with one huge body.
	parseR.Body = http.MaxBytesReader(parseW, parseR.Body, maxAgentAPIBodyBytes)
	return true
}

func (parseHub *AgentHub) latestActiveSessionID() string {
	parseHub.mu.Lock()
	defer parseHub.mu.Unlock()
	for parseIdx := len(parseHub.sessions) - 1; parseIdx >= 0; parseIdx-- {
		parseSess := parseHub.sessions[parseIdx]
		parseSess.mu.Lock()
		parseState := parseSess.State
		parseID := parseSess.ID
		parseSess.mu.Unlock()
		if parseState == StateActive {
			return parseID
		}
	}
	return ""
}

// isSessionActive reports whether the named session is currently connected and
// active (not crashed/reloading/closed).
func (parseHub *AgentHub) isSessionActive(parseID string) bool {
	parseID = strings.TrimSpace(parseID)
	if parseID == "" {
		return false
	}
	parseHub.mu.Lock()
	defer parseHub.mu.Unlock()
	for _, parseSess := range parseHub.sessions {
		parseSess.mu.Lock()
		parseMatch := parseSess.ID == parseID && parseSess.State == StateActive
		parseSess.mu.Unlock()
		if parseMatch {
			return true
		}
	}
	return false
}

func (parseHub *AgentHub) sessionFromRequest(parseR *http.Request) *Session {
	return parseHub.sessionFromRequestBody(strings.TrimSpace(parseR.URL.Query().Get("session")))
}

func (parseHub *AgentHub) sessionFromRequestBody(parseSessionID string) *Session {
	parseSessionID = strings.TrimSpace(parseSessionID)
	if parseSessionID == "" {
		parseSessionID = parseHub.latestActiveSessionID()
	}
	if parseSessionID == "" {
		return nil
	}
	return parseHub.findSession(parseSessionID)
}

func (parseHub *AgentHub) writeAPIJSON(parseW http.ResponseWriter, parseValue any) {
	parseW.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(parseW).Encode(parseValue)
}

// serveSession runs the read-loop for one accepted WebSocket connection.
func (parseHub *AgentHub) serveSession(parseConn *websocket.Conn) {
	defer parseConn.Close()
	parsePongWait := agentPongWait
	parsePingPeriod := agentPingPeriod
	parseWriteWait := agentWriteWait

	// Bound per-frame allocation: gorilla/websocket allows unlimited frame sizes
	// when no read limit is set, so a compromised/buggy loopback peer holding the
	// token could OOM the dev box with a single enormous frame.
	parseConn.SetReadLimit(maxAgentFrameBytes)

	// Read the mandatory hello frame. Give it a generous deadline so a slow
	// wasm boot does not time out a legitimate connection, but a dangling
	// socket does not live forever.
	if parseSetErr := parseConn.SetReadDeadline(time.Now().Add(30 * time.Second)); parseSetErr != nil {
		return
	}
	_, parseHelloRaw, parseReadErr := parseConn.ReadMessage()
	if parseReadErr != nil {
		return
	}
	// Arm the keepalive read deadline for the rest of the session; each inbound
	// frame and each pong pushes it forward (see runFrameLoop + the pong handler).
	if parseSetErr := parseConn.SetReadDeadline(time.Now().Add(parsePongWait)); parseSetErr != nil {
		return
	}
	parseConn.SetPongHandler(func(string) error {
		return parseConn.SetReadDeadline(time.Now().Add(parsePongWait))
	})

	parseEnvelope, parseParseErr := agentbridge.ParseEnvelope(string(parseHelloRaw))
	if parseParseErr != nil || parseEnvelope.Kind != agentbridge.KindHello {
		return
	}

	// Decode hello payload for app/build metadata and command names.
	var parseHello helloPayload
	if len(parseEnvelope.Payload) > 0 {
		_ = json.Unmarshal(parseEnvelope.Payload, &parseHello)
	}

	parseSessionID := fmt.Sprintf("sess-%d", parseHub.nextID.Add(1))

	parseSess := &Session{
		ID:          parseSessionID,
		AppID:       parseHello.AppID,
		BuildID:     parseHello.BuildID,
		Commands:    parseHello.Commands,
		ConnectedAt: time.Now().UTC(),
		State:       StateActive,
		conn:        parseConn,
		events:      newEventRing(eventRingSize),
		logs:        newEventRing(eventRingSize),
		recording:   newRecordingRing(recordingRingSize),
		pendingAcks: make(map[uint64]chan agentbridge.Envelope),
		outSeq:      &atomic.Uint64{},
		pongWait:    parsePongWait,
		pingPeriod:  parsePingPeriod,
		writeWait:   parseWriteWait,
	}

	// Link to predecessor if one is in reloading/crashed state.
	parseHub.mu.Lock()
	for parseIdx := len(parseHub.sessions) - 1; parseIdx >= 0; parseIdx-- {
		parsePrev := parseHub.sessions[parseIdx]
		parsePrev.mu.Lock()
		parsePrevState := parsePrev.State
		parsePrev.mu.Unlock()
		if parsePrevState == StateReloading || parsePrevState == StateCrashed {
			parseSess.PredecessorID = parsePrev.ID
			break
		}
	}
	parseHub.appendSessionLocked(parseSess)
	parseHub.mu.Unlock()

	// Run the inbound frame loop.
	parseHub.runFrameLoop(parseSess)
}

// runFrameLoop reads inbound frames from the session's socket until it closes.
func (parseHub *AgentHub) runFrameLoop(parseSess *Session) {
	// Keepalive: ping periodically so a half-open connection is detected. gorilla's
	// WriteControl is safe to call concurrently with the WriteMessage path (guarded
	// by writeMu elsewhere), so the ping needs no extra locking. The ticker stops
	// when the read loop returns (defer close(parseDone)).
	parseDone := make(chan struct{})
	defer close(parseDone)
	go func() {
		parseTicker := time.NewTicker(parseSess.pingPeriod)
		defer parseTicker.Stop()
		for {
			select {
			case <-parseDone:
				return
			case <-parseTicker.C:
				if parseErr := parseSess.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(parseSess.writeWait)); parseErr != nil {
					return
				}
			}
		}
	}()

	for {
		_, parseRaw, parseReadErr := parseSess.conn.ReadMessage()
		if parseReadErr != nil {
			// Socket closed. Determine terminal state.
			parseSess.mu.Lock()
			if parseSess.State == StateReloading {
				// Already flagged by MarkReloadExpected – keep reloading.
			} else {
				parseSess.State = StateCrashed
				parseSess.crashReport = parseHub.buildCrashReport(parseSess)
			}
			parseSess.lease = WriteLease{}
			parseSess.mu.Unlock()

			// Drain any pending SendCommand waiters with an error ack.
			parseSess.drainPendingAcks()
			return
		}

		// A live frame arrived: push the keepalive deadline forward.
		_ = parseSess.conn.SetReadDeadline(time.Now().Add(parseSess.pongWait))

		parseEnv, parseParseErr := agentbridge.ParseEnvelope(string(parseRaw))
		if parseParseErr != nil {
			continue
		}

		switch parseEnv.Kind {
		case agentbridge.KindAck:
			parseSess.mu.Lock()
			parseCh, parseFound := parseSess.pendingAcks[parseEnv.AckSeq]
			if parseFound {
				delete(parseSess.pendingAcks, parseEnv.AckSeq)
			}
			parseSess.mu.Unlock()
			if parseFound {
				select {
				case parseCh <- parseEnv:
				default:
				}
			}

		case agentbridge.KindEvent:
			parseEnv.Payload = redactRawMessage(parseEnv.Payload)
			parseSess.events.push(parseEnv)
			if isLogOrDiagnosticEvent(parseEnv.Name) {
				parseSess.logs.push(parseEnv)
			}

		default:
			// hello mid-session and unknown kinds are silently dropped.
		}
	}
}

// MarkReloadExpected flags the active session identified by parseSessionID so
// that an imminent socket death is interpreted as reloading (a rebuild is in
// flight) rather than crashed. Call this before triggering a rebuild.
func (parseHub *AgentHub) MarkReloadExpected(parseSessionID string) {
	parseHub.mu.Lock()
	defer parseHub.mu.Unlock()
	for _, parseSess := range parseHub.sessions {
		if parseSess.ID == parseSessionID {
			parseSess.mu.Lock()
			if parseSess.State == StateActive {
				parseSess.State = StateReloading
			}
			parseSess.mu.Unlock()
			return
		}
	}
}

// ListSessions returns a snapshot of all session metadata, newest first.
func (parseHub *AgentHub) ListSessions() []SessionInfo {
	parseHub.mu.Lock()
	parseCopy := make([]*Session, len(parseHub.sessions))
	copy(parseCopy, parseHub.sessions)
	parseHub.mu.Unlock()

	parseResult := make([]SessionInfo, len(parseCopy))
	for parseIdx, parseSess := range parseCopy {
		parseResult[len(parseCopy)-1-parseIdx] = parseSess.info()
	}
	return parseResult
}

// SendCommand writes a command frame to the session identified by
// parseSessionID and blocks until the matching ack arrives or parseCtx is
// done. Concurrent calls are serialized per session via the per-session
// write mutex so frame ordering is preserved.
func (parseHub *AgentHub) SendCommand(parseCtx context.Context, parseSessionID string, parseName string, parsePayload json.RawMessage) (agentbridge.Envelope, error) {
	parseSess := parseHub.findSession(parseSessionID)
	if parseSess == nil {
		return agentbridge.Envelope{}, fmt.Errorf("agenthub: session %q not found", parseSessionID)
	}

	parseSeq := parseSess.outSeq.Add(1)
	parseCh := make(chan agentbridge.Envelope, 1)
	parseStarted := time.Now().UTC()
	parseRecord := parseSess.recording.begin(CommandRecord{
		Seq:       parseSeq,
		Name:      strings.TrimSpace(parseName),
		Payload:   redactRawMessage(parsePayload),
		StartedAt: parseStarted,
	})

	parseSess.mu.Lock()
	if parseSess.State != StateActive {
		parseSess.mu.Unlock()
		return agentbridge.Envelope{}, fmt.Errorf("agenthub: session %q is not active (state=%s)", parseSessionID, parseSess.State)
	}
	parseSess.pendingAcks[parseSeq] = parseCh
	parseSess.mu.Unlock()

	parseFrame := agentbridge.NewCommandEnvelope(parseSeq, parseSessionID, parseName, parsePayload)
	parseFormatted, parseFormatErr := agentbridge.FormatEnvelopeJSON(parseFrame)
	if parseFormatErr != nil {
		parseSess.mu.Lock()
		delete(parseSess.pendingAcks, parseSeq)
		parseSess.mu.Unlock()
		return agentbridge.Envelope{}, fmt.Errorf("agenthub: format command: %w", parseFormatErr)
	}

	parseSess.writeMu.Lock()
	parseWriteErr := parseSess.conn.WriteMessage(websocket.TextMessage, []byte(parseFormatted))
	parseSess.writeMu.Unlock()
	if parseWriteErr != nil {
		parseSess.mu.Lock()
		delete(parseSess.pendingAcks, parseSeq)
		parseSess.mu.Unlock()
		parseRecord.CompletedAt = time.Now().UTC()
		parseRecord.DurationMs = parseRecord.CompletedAt.Sub(parseRecord.StartedAt).Milliseconds()
		parseRecord.Error = parseWriteErr.Error()
		parseSess.recording.push(parseRecord)
		return agentbridge.Envelope{}, fmt.Errorf("agenthub: write command: %w", parseWriteErr)
	}

	select {
	case parseAck, parseOpen := <-parseCh:
		if !parseOpen {
			// drainPendingAcks closed the channel because the socket died.
			parseRecord.CompletedAt = time.Now().UTC()
			parseRecord.DurationMs = parseRecord.CompletedAt.Sub(parseRecord.StartedAt).Milliseconds()
			parseRecord.Error = "socket closed before ack"
			parseSess.recording.push(parseRecord)
			return agentbridge.Envelope{}, fmt.Errorf("agenthub: session %q socket closed before ack", parseSessionID)
		}
		parseRecord.CompletedAt = time.Now().UTC()
		parseRecord.DurationMs = parseRecord.CompletedAt.Sub(parseRecord.StartedAt).Milliseconds()
		parseRecord.StateVersion = parseAck.StateVersion
		parseRecord.OK = parseAck.OK
		parseAckCopy := parseAck
		parseRecord.Ack = &parseAckCopy
		parseSess.recording.push(parseRecord)
		parseHub.maybeCaptureSnapshot(parseSess, parseName, parseAck)
		return parseAck, nil
	case <-parseCtx.Done():
		parseSess.mu.Lock()
		delete(parseSess.pendingAcks, parseSeq)
		parseSess.mu.Unlock()
		parseRecord.CompletedAt = time.Now().UTC()
		parseRecord.DurationMs = parseRecord.CompletedAt.Sub(parseRecord.StartedAt).Milliseconds()
		parseRecord.Error = parseCtx.Err().Error()
		parseSess.recording.push(parseRecord)
		return agentbridge.Envelope{}, fmt.Errorf("agenthub: command timed out: %w", parseCtx.Err())
	}
}

// ReadEvents returns up to parseMax buffered KindEvent envelopes from the
// session identified by parseSessionID, oldest first. It returns nil if the
// session is not found or has no events.
func (parseHub *AgentHub) ReadEvents(parseSessionID string, parseMax int) []agentbridge.Envelope {
	parseSess := parseHub.findSession(parseSessionID)
	if parseSess == nil {
		return nil
	}
	return parseSess.events.drain(parseMax)
}

func (parseHub *AgentHub) maybeCaptureSnapshot(parseSess *Session, parseName string, parseAck agentbridge.Envelope) {
	if parseName != "bridge.snapshot" || parseAck.OK == nil || !*parseAck.OK || len(parseAck.Payload) == 0 {
		return
	}
	parseSess.mu.Lock()
	parseSess.lastSnapshot = redactRawMessage(parseAck.Payload)
	parseSess.mu.Unlock()
}

func (parseHub *AgentHub) buildCrashReport(parseSess *Session) *CrashReport {
	return &CrashReport{
		SessionID:    parseSess.ID,
		AppID:        parseSess.AppID,
		BuildID:      parseSess.BuildID,
		CrashedAt:    time.Now().UTC(),
		LastSnapshot: cloneRawMessage(parseSess.lastSnapshot),
		Logs:         parseSess.logs.snapshot(64),
		Commands:     parseSess.recording.snapshot(32),
	}
}

func (parseHub *AgentHub) recordingChain(parseSess *Session, parseMax int) []CommandRecord {
	return parseHub.recordingChainGuarded(parseSess, parseMax, map[string]bool{}, 0)
}

// maxRecordingChainDepth caps how far back a reload/crash predecessor chain is
// walked, bounding work even before the visited-set catches a cycle.
const maxRecordingChainDepth = 128

// maxRetainedSessions bounds the append-only hub.sessions history. Crashed/reloaded
// predecessors are kept for crash-report linkage and were never pruned, so a long
// dev session with repeated hot-reloads grew the slice (and its per-session ring
// buffers) without bound. The oldest sessions are dropped past this cap. It sits
// comfortably above maxRecordingChainDepth: a predecessor is always older (earlier)
// than its successor and chains are walked at most maxRecordingChainDepth deep, so
// keeping the newest maxRetainedSessions preserves every reachable crash-report
// chain; anything older can no longer be reached by a bounded walk anyway. A var
// (not const) so tests can lower it without opening hundreds of connections.
var maxRetainedSessions = 512

// appendSessionLocked appends a session to the append-only history and prunes the
// oldest past maxRetainedSessions. Caller holds parseHub.mu. A fresh slice is
// allocated on prune so the dropped *Session pointers (and their ring buffers) are
// unreachable and GC-able.
func (parseHub *AgentHub) appendSessionLocked(parseSess *Session) {
	parseHub.sessions = append(parseHub.sessions, parseSess)
	if parseOverflow := len(parseHub.sessions) - maxRetainedSessions; parseOverflow > 0 {
		parseHub.sessions = append([]*Session(nil), parseHub.sessions[parseOverflow:]...)
	}
}

// recordingChainGuarded walks the predecessor chain with a visited-set and a
// depth cap so a self-referential or cyclic PredecessorID cannot infinite-loop
// and a pathologically long chain cannot blow up allocation.
func (parseHub *AgentHub) recordingChainGuarded(parseSess *Session, parseMax int, parseVisited map[string]bool, parseDepth int) []CommandRecord {
	parseRecords := parseSess.recording.snapshot(parseMax)
	parseVisited[parseSess.ID] = true
	if parseSess.PredecessorID == "" || parseVisited[parseSess.PredecessorID] || parseDepth >= maxRecordingChainDepth {
		return parseRecords
	}
	parsePrev := parseHub.findSession(parseSess.PredecessorID)
	if parsePrev == nil {
		return parseRecords
	}
	parsePrevRecords := parseHub.recordingChainGuarded(parsePrev, 0, parseVisited, parseDepth+1)
	parseJoined := append(parsePrevRecords, parseRecords...)
	if parseMax > 0 && len(parseJoined) > parseMax {
		parseJoined = parseJoined[len(parseJoined)-parseMax:]
	}
	return parseJoined
}

func (parseHub *AgentHub) requireWriteLease(parseSessionID string, parseHolder string) error {
	parseSess := parseHub.findSession(parseSessionID)
	if parseSess == nil {
		return fmt.Errorf("agenthub: session %q not found", parseSessionID)
	}
	parseHolder = strings.TrimSpace(parseHolder)
	if parseHolder == "" {
		return fmt.Errorf("agenthub: mutating command requires a write lease holder")
	}
	parseSess.mu.Lock()
	defer parseSess.mu.Unlock()
	if parseSess.lease.Holder == "" || time.Now().UTC().After(parseSess.lease.ExpiresAt) {
		return fmt.Errorf("agenthub: write lease is not held for session %q", parseSessionID)
	}
	if parseSess.lease.Holder != parseHolder {
		return fmt.Errorf("agenthub: write lease held by %q", parseSess.lease.Holder)
	}
	return nil
}

func (parseHub *AgentHub) updateWriteLease(parseSess *Session, parseAction string, parseHolder string, parseSteal bool) (WriteLease, error) {
	parseNow := time.Now().UTC()
	parseExpires := parseNow.Add(leaseDuration)
	parseSess.mu.Lock()
	defer parseSess.mu.Unlock()
	parseExpired := parseSess.lease.Holder == "" || parseNow.After(parseSess.lease.ExpiresAt)
	switch parseAction {
	case "acquire":
		if !parseExpired && parseSess.lease.Holder != parseHolder {
			return parseSess.lease, fmt.Errorf("agenthub: write lease held by %q", parseSess.lease.Holder)
		}
		parseSess.lease = WriteLease{Holder: parseHolder, ExpiresAt: parseExpires}
	case "release":
		if parseSess.lease.Holder != "" && parseSess.lease.Holder != parseHolder {
			return parseSess.lease, fmt.Errorf("agenthub: write lease held by %q", parseSess.lease.Holder)
		}
		parseSess.lease = WriteLease{}
	case "steal":
		if !parseSteal {
			return parseSess.lease, fmt.Errorf("agenthub: steal requires explicit steal flag")
		}
		parseSess.lease = WriteLease{Holder: parseHolder, ExpiresAt: parseExpires}
		parsePayload := json.RawMessage(fmt.Sprintf(`{"holder":%q}`, parseHolder))
		parseSess.events.push(agentbridge.NewEventEnvelope(parseSess.outSeq.Add(1), parseSess.ID, "lease.stolen", parsePayload))
	default:
		return parseSess.lease, fmt.Errorf("agenthub: unsupported lease action %q", parseAction)
	}
	return parseSess.lease, nil
}

// tokenMatches reports whether parseToken equals the hub token using a
// constant-time comparison so a timing oracle cannot recover the token byte by
// byte. An empty token never matches.
func (parseHub *AgentHub) tokenMatches(parseToken string) bool {
	if parseToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(parseToken), []byte(parseHub.token)) == 1
}

// commandRequiresLease reports whether a command needs the session write lease.
// It is payload-aware for bridge.replay: only the "replay" action mutates live
// app state and needs the lease; start/stop/status are recorder control and do
// not, so a reader can poll replay status without holding the lease.
func commandRequiresLease(parseName string, parsePayload json.RawMessage) bool {
	if strings.TrimSpace(parseName) == "bridge.replay" {
		var parseBody struct {
			Action string `json:"action"`
		}
		_ = json.Unmarshal(parsePayload, &parseBody)
		return strings.TrimSpace(parseBody.Action) == "replay"
	}
	return isMutatingBridgeCommand(parseName)
}

func isMutatingBridgeCommand(parseName string) bool {
	switch strings.TrimSpace(parseName) {
	case "bridge.set-atom", "bridge.set-state", "bridge.mount", "bridge.unmount", "bridge.delete-atom", "bridge.emit", "bridge.publish", "bridge.navigate",
		// bridge.undo reverses live atom state and bridge.replay's "replay"
		// action re-applies recorded updates — both must hold the write lease
		// so one agent cannot mutate around another's lease.
		"bridge.undo", "bridge.replay":
		return true
	default:
		return false
	}
}

func isLogOrDiagnosticEvent(parseName string) bool {
	parseName = strings.ToLower(strings.TrimSpace(parseName))
	return strings.Contains(parseName, "log") || strings.Contains(parseName, "diagnostic") || strings.Contains(parseName, "panic")
}

func cloneRawMessage(parsePayload json.RawMessage) json.RawMessage {
	if len(parsePayload) == 0 {
		return nil
	}
	parseCopy := make(json.RawMessage, len(parsePayload))
	copy(parseCopy, parsePayload)
	return parseCopy
}

func redactRawMessage(parsePayload json.RawMessage) json.RawMessage {
	if len(parsePayload) == 0 {
		return nil
	}
	var parseValue any
	if parseErr := json.Unmarshal(parsePayload, &parseValue); parseErr != nil {
		return cloneRawMessage(parsePayload)
	}
	parseOut, parseErr2 := json.Marshal(redactJSONValue(parseValue))
	if parseErr2 != nil {
		return cloneRawMessage(parsePayload)
	}
	return parseOut
}

func redactJSONValue(parseValue any) any {
	switch parseTyped := parseValue.(type) {
	case map[string]any:
		parseOut := make(map[string]any, len(parseTyped))
		for parseKey, parseChild := range parseTyped {
			if isSensitiveJSONKey(parseKey) {
				continue
			}
			parseOut[parseKey] = redactJSONValue(parseChild)
		}
		return parseOut
	case []any:
		parseOut := make([]any, len(parseTyped))
		for parseIdx, parseChild := range parseTyped {
			parseOut[parseIdx] = redactJSONValue(parseChild)
		}
		return parseOut
	default:
		return parseTyped
	}
}

func isSensitiveJSONKey(parseKey string) bool {
	parseNormalized := strings.ToLower(strings.NewReplacer("-", "", "_", "", ".", "").Replace(strings.TrimSpace(parseKey)))
	if parseNormalized == "" {
		return false
	}
	for _, parseNeedle := range []string{"authorization", "cookie", "password", "secret", "token", "apikey", "accesskey", "privatekey"} {
		if strings.Contains(parseNormalized, parseNeedle) {
			return true
		}
	}
	return false
}

func parsePositiveInt(parseValue string) int {
	var parseN int
	if _, parseErr := fmt.Sscanf(strings.TrimSpace(parseValue), "%d", &parseN); parseErr != nil || parseN < 0 {
		return 0
	}
	return parseN
}

// findSession locates the first session matching parseSessionID.
func (parseHub *AgentHub) findSession(parseSessionID string) *Session {
	parseHub.mu.Lock()
	defer parseHub.mu.Unlock()
	for _, parseSess := range parseHub.sessions {
		if parseSess.ID == parseSessionID {
			return parseSess
		}
	}
	return nil
}

func (parseHub *AgentHub) findSuccessor(parsePredecessorID string, parseBuildID string) *Session {
	parseHub.mu.Lock()
	parseCopy := make([]*Session, len(parseHub.sessions))
	copy(parseCopy, parseHub.sessions)
	parseHub.mu.Unlock()
	for parseIdx := len(parseCopy) - 1; parseIdx >= 0; parseIdx-- {
		parseSess := parseCopy[parseIdx]
		parseSess.mu.Lock()
		parseMatch := parseSess.PredecessorID == parsePredecessorID && parseSess.State == StateActive
		if parseMatch && parseBuildID != "" {
			parseMatch = parseSess.BuildID == parseBuildID
		}
		parseSess.mu.Unlock()
		if parseMatch {
			return parseSess
		}
	}
	return nil
}

// isLoopback reports whether parseAddr (host:port) is a loopback address.
func (parseHub *AgentHub) isLoopback(parseAddr string) bool {
	parseHost, _, parseErr := net.SplitHostPort(parseAddr)
	if parseErr != nil {
		parseHost = parseAddr
	}
	parseIP := net.ParseIP(strings.TrimSpace(parseHost))
	if parseIP == nil {
		return false
	}
	return parseIP.IsLoopback()
}

// originAllowed is the websocket.Upgrader.CheckOrigin callback. It accepts
// requests with no Origin header (non-browser clients) and requires any
// present Origin to be a localhost variant.
func (parseHub *AgentHub) originAllowed(parseR *http.Request) bool {
	parseOrigin := strings.TrimSpace(parseR.Header.Get("Origin"))
	if parseOrigin == "" {
		return true
	}
	parseURL, parseErr := url.Parse(parseOrigin)
	if parseErr != nil || parseURL.Host == "" {
		return false
	}
	parseHost := strings.ToLower(parseURL.Hostname())
	parseAllowed := map[string]bool{
		"localhost": true,
		"127.0.0.1": true,
		"::1":       true,
	}
	return parseAllowed[parseHost]
}

// helloPayload mirrors the first frame an app sends to identify itself.
type helloPayload struct {
	AppID    string   `json:"appId"`
	BuildID  string   `json:"buildId"`
	Commands []string `json:"commands"`
}
