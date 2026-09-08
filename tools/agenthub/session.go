package agenthub

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/monstercameron/GoWebComponents/v5/agentbridge"
)

// SessionState is the lifecycle state of an agent session.
type SessionState string

const (
	// StateActive means the session's WebSocket is open and healthy.
	StateActive SessionState = "active"
	// StateReloading means a rebuild was signaled; the next hello will be
	// linked as a successor. The socket may or may not be closed yet.
	StateReloading SessionState = "reloading"
	// StateCrashed means the socket died outside a known rebuild window.
	StateCrashed SessionState = "crashed"
	// StateClosed means the session was gracefully closed.
	StateClosed SessionState = "closed"
)

// Session is one connected (or formerly connected) app instance. Fields are
// only written under mu except conn, which may only be written before the
// session is exposed to other goroutines.
type Session struct {
	// ID is the hub-assigned monotonic session identifier.
	ID string
	// PredecessorID is the ID of the session this one succeeded (reload/crash
	// recovery), or empty for the first session in a chain.
	PredecessorID string

	// AppID and BuildID come from the hello payload.
	AppID   string
	BuildID string
	// Commands is the list of command names the app advertises.
	Commands []string
	// ConnectedAt is the wall-clock time the hello frame was received.
	ConnectedAt time.Time

	// State transitions: active -> reloading -> (new session); active -> crashed.
	State SessionState

	mu           sync.Mutex
	conn         *websocket.Conn
	writeMu      sync.Mutex
	events       *eventRing
	logs         *eventRing
	recording    *recordingRing
	lastSnapshot json.RawMessage
	crashReport  *CrashReport
	lease        WriteLease
	pendingAcks  map[uint64]chan agentbridge.Envelope
	outSeq       *atomic.Uint64
	pongWait     time.Duration
	pingPeriod   time.Duration
	writeWait    time.Duration
}

// SessionInfo is a read-only snapshot of session metadata for ListSessions.
type SessionInfo struct {
	// ID is the hub-assigned session identifier.
	ID string
	// PredecessorID links this session to its predecessor in a reload chain.
	PredecessorID string
	// AppID from the hello payload.
	AppID string
	// BuildID from the hello payload.
	BuildID string
	// Commands advertised by the app.
	Commands []string
	// ConnectedAt is when the hello was received.
	ConnectedAt time.Time
	// State is the current lifecycle state.
	State SessionState
	// DropCount is the number of events dropped from the ring buffer due to
	// overflow since this session connected.
	DropCount int
	// LogDropCount is the number of diagnostic/log frames dropped from the
	// retained log buffer.
	LogDropCount int
	// RecordingCount is the number of retained command records.
	RecordingCount int
	// HasCrashReport reports whether a crash report is available.
	HasCrashReport bool
	// LeaseHolder is the current write lease holder, when one exists.
	LeaseHolder string
	// LeaseExpiresAt is the current write lease expiry, when one exists.
	LeaseExpiresAt time.Time
}

// info builds a read-only snapshot of the session for ListSessions callers.
func (parseSess *Session) info() SessionInfo {
	parseSess.mu.Lock()
	defer parseSess.mu.Unlock()
	return SessionInfo{
		ID:             parseSess.ID,
		PredecessorID:  parseSess.PredecessorID,
		AppID:          parseSess.AppID,
		BuildID:        parseSess.BuildID,
		Commands:       parseSess.Commands,
		ConnectedAt:    parseSess.ConnectedAt,
		State:          parseSess.State,
		DropCount:      parseSess.events.dropCount(),
		LogDropCount:   parseSess.logs.dropCount(),
		RecordingCount: parseSess.recording.len(),
		HasCrashReport: parseSess.crashReport != nil,
		LeaseHolder:    parseSess.lease.Holder,
		LeaseExpiresAt: parseSess.lease.ExpiresAt,
	}
}

// drainPendingAcks closes all pending SendCommand waiters when the socket dies.
func (parseSess *Session) drainPendingAcks() {
	parseSess.mu.Lock()
	parsePending := parseSess.pendingAcks
	parseSess.pendingAcks = make(map[uint64]chan agentbridge.Envelope)
	parseSess.mu.Unlock()
	for _, parseCh := range parsePending {
		close(parseCh)
	}
}

// eventRing is a bounded FIFO ring buffer for KindEvent envelopes. It drops
// the oldest entry when full, incrementing a drop counter.
type eventRing struct {
	mu    sync.Mutex
	buf   []agentbridge.Envelope
	head  int // index of the oldest entry
	count int // number of live entries
	cap   int // ring capacity
	drops int
}

// newEventRing creates an eventRing with the given capacity.
func newEventRing(parseCapacity int) *eventRing {
	return &eventRing{
		buf: make([]agentbridge.Envelope, parseCapacity),
		cap: parseCapacity,
	}
}

// push appends parseEnv to the ring, dropping the oldest entry if full.
func (parseRing *eventRing) push(parseEnv agentbridge.Envelope) {
	parseRing.mu.Lock()
	defer parseRing.mu.Unlock()
	if parseRing.count == parseRing.cap {
		// Drop oldest.
		parseRing.head = (parseRing.head + 1) % parseRing.cap
		parseRing.count--
		parseRing.drops++
	}
	parseTail := (parseRing.head + parseRing.count) % parseRing.cap
	parseRing.buf[parseTail] = parseEnv
	parseRing.count++
}

// drain removes and returns up to parseMax entries from the ring, oldest first.
func (parseRing *eventRing) drain(parseMax int) []agentbridge.Envelope {
	parseRing.mu.Lock()
	defer parseRing.mu.Unlock()
	if parseRing.count == 0 {
		return nil
	}
	parseN := parseRing.count
	if parseMax > 0 && parseMax < parseN {
		parseN = parseMax
	}
	parseOut := make([]agentbridge.Envelope, parseN)
	for parseIdx := 0; parseIdx < parseN; parseIdx++ {
		parseOut[parseIdx] = parseRing.buf[(parseRing.head+parseIdx)%parseRing.cap]
	}
	parseRing.head = (parseRing.head + parseN) % parseRing.cap
	parseRing.count -= parseN
	return parseOut
}

// snapshot returns up to parseMax retained entries without removing them.
func (parseRing *eventRing) snapshot(parseMax int) []agentbridge.Envelope {
	parseRing.mu.Lock()
	defer parseRing.mu.Unlock()
	if parseRing.count == 0 {
		return nil
	}
	parseN := parseRing.count
	if parseMax > 0 && parseMax < parseN {
		parseN = parseMax
	}
	parseStart := (parseRing.head + parseRing.count - parseN + parseRing.cap) % parseRing.cap
	parseOut := make([]agentbridge.Envelope, parseN)
	for parseIdx := 0; parseIdx < parseN; parseIdx++ {
		parseOut[parseIdx] = parseRing.buf[(parseStart+parseIdx)%parseRing.cap]
	}
	return parseOut
}

// dropCount returns the number of events that have been dropped due to overflow.
func (parseRing *eventRing) dropCount() int {
	parseRing.mu.Lock()
	defer parseRing.mu.Unlock()
	return parseRing.drops
}

// WriteLease describes the single writer allowed to send mutating commands to
// a session.
type WriteLease struct {
	Holder    string    `json:"holder,omitempty"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
}

// CommandRecord captures one hub-to-app command and the ack or error that
// completed it.
type CommandRecord struct {
	Seq                 uint64                `json:"seq"`
	Name                string                `json:"name"`
	Payload             json.RawMessage       `json:"payload,omitempty"`
	StartedAt           time.Time             `json:"startedAt"`
	CompletedAt         time.Time             `json:"completedAt,omitempty"`
	DurationMs          int64                 `json:"durationMs,omitempty"`
	WaitSincePreviousMs int64                 `json:"waitSincePreviousMs,omitempty"`
	StateVersion        uint64                `json:"stateVersion,omitempty"`
	OK                  *bool                 `json:"ok,omitempty"`
	Ack                 *agentbridge.Envelope `json:"ack,omitempty"`
	Error               string                `json:"error,omitempty"`
}

type recordingRing struct {
	mu       sync.Mutex
	buf      []CommandRecord
	head     int
	count    int
	cap      int
	drops    int
	lastDone time.Time
}

func newRecordingRing(parseCapacity int) *recordingRing {
	return &recordingRing{buf: make([]CommandRecord, parseCapacity), cap: parseCapacity}
}

func (parseRing *recordingRing) begin(parseRecord CommandRecord) CommandRecord {
	parseRing.mu.Lock()
	defer parseRing.mu.Unlock()
	if !parseRing.lastDone.IsZero() {
		parseRecord.WaitSincePreviousMs = parseRecord.StartedAt.Sub(parseRing.lastDone).Milliseconds()
	}
	return parseRecord
}

func (parseRing *recordingRing) push(parseRecord CommandRecord) {
	parseRing.mu.Lock()
	defer parseRing.mu.Unlock()
	if parseRing.count == parseRing.cap {
		parseRing.head = (parseRing.head + 1) % parseRing.cap
		parseRing.count--
		parseRing.drops++
	}
	parseTail := (parseRing.head + parseRing.count) % parseRing.cap
	parseRing.buf[parseTail] = parseRecord
	parseRing.count++
	if !parseRecord.CompletedAt.IsZero() {
		parseRing.lastDone = parseRecord.CompletedAt
	}
}

func (parseRing *recordingRing) snapshot(parseMax int) []CommandRecord {
	parseRing.mu.Lock()
	defer parseRing.mu.Unlock()
	if parseRing.count == 0 {
		return nil
	}
	parseN := parseRing.count
	if parseMax > 0 && parseMax < parseN {
		parseN = parseMax
	}
	parseStart := (parseRing.head + parseRing.count - parseN + parseRing.cap) % parseRing.cap
	parseOut := make([]CommandRecord, parseN)
	for parseIdx := 0; parseIdx < parseN; parseIdx++ {
		parseOut[parseIdx] = parseRing.buf[(parseStart+parseIdx)%parseRing.cap]
	}
	return parseOut
}

func (parseRing *recordingRing) clear() {
	parseRing.mu.Lock()
	defer parseRing.mu.Unlock()
	parseRing.head = 0
	parseRing.count = 0
	parseRing.drops = 0
	parseRing.lastDone = time.Time{}
}

func (parseRing *recordingRing) len() int {
	parseRing.mu.Lock()
	defer parseRing.mu.Unlock()
	return parseRing.count
}

func (parseRing *recordingRing) dropCount() int {
	parseRing.mu.Lock()
	defer parseRing.mu.Unlock()
	return parseRing.drops
}

// CrashReport is assembled when a session socket dies outside a rebuild
// window.
type CrashReport struct {
	SessionID    string                 `json:"sessionId"`
	AppID        string                 `json:"appId,omitempty"`
	BuildID      string                 `json:"buildId,omitempty"`
	CrashedAt    time.Time              `json:"crashedAt"`
	LastSnapshot json.RawMessage        `json:"lastSnapshot,omitempty"`
	Logs         []agentbridge.Envelope `json:"logs,omitempty"`
	Commands     []CommandRecord        `json:"commands,omitempty"`
}

// RouteAgentHub registers the /gwc-agent handler on parseMux. This is the
// one-line integration point called from the livereload HTTP handler setup.
func RouteAgentHub(parseMux *http.ServeMux, parseHub *AgentHub) {
	parseMux.Handle("/gwc-agent", parseHub)
	parseMux.Handle("/__gwc-agent/sessions", parseHub)
	parseMux.Handle("/__gwc-agent/command", parseHub)
	parseMux.Handle("/__gwc-agent/logs", parseHub)
	parseMux.Handle("/__gwc-agent/crash-report", parseHub)
	parseMux.Handle("/__gwc-agent/recording", parseHub)
	parseMux.Handle("/__gwc-agent/lease", parseHub)
	parseMux.Handle("/__gwc-agent/reload", parseHub)
	parseMux.Handle("/__gwc-agent/successor", parseHub)
}
