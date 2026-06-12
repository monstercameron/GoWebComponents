package agentbridge

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// AgentSocket abstracts one bidirectional frame-oriented transport. The
// bridge client calls ReadFrame and WriteFrame from a single goroutine; Close
// may be called from any goroutine and must be idempotent.
type AgentSocket interface {
	// ReadFrame blocks until a text frame arrives or the connection closes.
	// It returns ("", io.EOF) or any non-nil error when the socket is done.
	ReadFrame() (string, error)
	// WriteFrame sends one text frame synchronously. An error means the
	// connection is broken; the client will stop the loop.
	WriteFrame(parseFrame string) error
	// Close terminates the connection. Safe to call more than once.
	Close() error
}

// helloPayload is the first payload the client sends so the hub learns the
// app identity and its command surface.
type helloPayload struct {
	AppID    string   `json:"appId"`
	BuildID  string   `json:"buildId"`
	Protocol string   `json:"protocol"`
	Commands []string `json:"commands"`
}

// BridgeClient is the transport-agnostic client loop. Create one with
// NewBridgeClient; the zero value is not valid.
type BridgeClient struct {
	appID   string
	buildID string
	outSeq  atomic.Uint64

	eventMu      sync.Mutex
	eventSock    AgentSocket // nil until hello has been written
	eventBacklog []Envelope
}

// NewBridgeClient constructs a BridgeClient for the given app and build
// identifiers. Neither string is validated; empty strings are legal for
// tests.
func NewBridgeClient(parseAppID string, parseBuildID string) *BridgeClient {
	return &BridgeClient{appID: parseAppID, buildID: parseBuildID}
}

// nextSeq returns the next outbound sequence number (monotonically increasing,
// starting at 1).
func (parseC *BridgeClient) nextSeq() uint64 {
	return parseC.outSeq.Add(1)
}

// buildHelloJSON formats the hello envelope as a JSON string.
func (parseC *BridgeClient) buildHelloJSON() (string, error) {
	parsePayloadStruct := helloPayload{
		AppID:    parseC.appID,
		BuildID:  parseC.buildID,
		Protocol: ProtocolName + "/v" + fmt.Sprint(ProtocolVersion),
		Commands: ListAgentCommands(),
	}
	parsePayloadBytes, parseErr := json.Marshal(parsePayloadStruct)
	if parseErr != nil {
		return "", fmt.Errorf("agentbridge: marshal hello payload: %w", parseErr)
	}
	parseEnv := BuildHelloEnvelope(parseC.nextSeq(), json.RawMessage(parsePayloadBytes))
	parseJSON, parseErr := FormatEnvelopeJSON(parseEnv)
	if parseErr != nil {
		return "", fmt.Errorf("agentbridge: format hello envelope: %w", parseErr)
	}
	return parseJSON, nil
}

// RunLoop runs the client loop on the given socket. It sends the hello frame,
// then reads, parses, and dispatches frames until the socket closes or
// WriteFrame returns an error. RunLoop blocks; run it in a goroutine. The
// socket is closed before RunLoop returns.
func (parseC *BridgeClient) RunLoop(parseSock AgentSocket) {
	defer parseSock.Close()

	defer func() {
		parseC.eventMu.Lock()
		if parseC.eventSock == parseSock {
			parseC.eventSock = nil
		}
		parseC.eventMu.Unlock()
	}()

	// Send hello as the very first frame.
	parseHelloJSON, parseErr := parseC.buildHelloJSON()
	if parseErr != nil {
		runtime.ReportDiagnostic("agentbridge", runtime.DiagnosticWarning,
			"bridge client: build hello: "+parseErr.Error())
		return
	}
	if parseWriteErr := parseSock.WriteFrame(parseHelloJSON); parseWriteErr != nil {
		runtime.ReportDiagnostic("agentbridge", runtime.DiagnosticWarning,
			"bridge client: write hello: "+parseWriteErr.Error())
		return
	}
	parseC.eventMu.Lock()
	parseC.eventSock = parseSock
	parseBacklog := parseC.eventBacklog
	parseC.eventBacklog = nil
	parseC.eventMu.Unlock()
	for _, parseEvent := range parseBacklog {
		parseEventJSON, parseFormatErr := FormatEnvelopeJSON(parseEvent)
		if parseFormatErr != nil {
			runtime.ReportDiagnostic("agentbridge", runtime.DiagnosticWarning,
				"bridge client: format queued event: "+parseFormatErr.Error())
			continue
		}
		if parseWriteErr := parseSock.WriteFrame(parseEventJSON); parseWriteErr != nil {
			runtime.ReportDiagnostic("agentbridge", runtime.DiagnosticWarning,
				"bridge client: write queued event: "+parseWriteErr.Error())
			return
		}
	}

	// Main dispatch loop.
	for {
		parseFrame, parseReadErr := parseSock.ReadFrame()
		if parseReadErr != nil {
			// Normal EOF or closed socket — not a diagnostic.
			return
		}

		parseEnv, parseParseErr := ParseEnvelope(parseFrame)
		if parseParseErr != nil {
			// Malformed frame: report but continue — never panic, never stop.
			runtime.ReportDiagnostic("agentbridge", runtime.DiagnosticWarning,
				"bridge client: malformed inbound frame: "+parseParseErr.Error())
			continue
		}

		switch parseEnv.Kind {
		case KindCommand:
			parseC.handleCommand(parseSock, parseEnv)
		default:
			// Unexpected kinds (hello, ack, event from hub) are ignored.
		}
	}
}

// handleCommand executes one KindCommand envelope and writes the ack.
func (parseC *BridgeClient) handleCommand(parseSock AgentSocket, parseEnv Envelope) {
	parseResultPayload, parseExecErr := ExecuteAgentCommand(parseEnv.Name, parseEnv.Payload)

	var parseReply Envelope
	if parseExecErr != nil {
		parseReply = BuildErrorAckEnvelope(
			parseC.nextSeq(),
			parseEnv.Session,
			parseEnv.Seq,
			parseExecErr.Code,
			parseExecErr.Message,
		)
	} else {
		parseReply = BuildAckEnvelope(
			parseC.nextSeq(),
			parseEnv.Session,
			parseEnv.Seq,
			runtime.GetGlobalRuntime().AgentStateVersion(),
			parseResultPayload,
		)
	}

	parseReplyJSON, parseFormatErr := FormatEnvelopeJSON(parseReply)
	if parseFormatErr != nil {
		runtime.ReportDiagnostic("agentbridge", runtime.DiagnosticWarning,
			"bridge client: format ack: "+parseFormatErr.Error())
		return
	}
	if parseWriteErr := parseSock.WriteFrame(parseReplyJSON); parseWriteErr != nil {
		runtime.ReportDiagnostic("agentbridge", runtime.DiagnosticWarning,
			"bridge client: write ack: "+parseWriteErr.Error())
	}
}

// SendEvent pushes an unsolicited event frame on the current socket, if one is
// active. It returns an error if no socket is connected or the write fails.
// SendEvent is safe to call from any goroutine.
func (parseC *BridgeClient) SendEvent(parseSession string, parseName string, parsePayload json.RawMessage) error {
	parseEnv := BuildEventEnvelope(parseC.nextSeq(), parseSession, parseName, parsePayload)
	parseC.eventMu.Lock()
	parseSock := parseC.eventSock
	if parseSock == nil {
		parseC.queueEventLocked(parseEnv)
		parseC.eventMu.Unlock()
		return nil
	}
	parseC.eventMu.Unlock()

	parseJSON, parseErr := FormatEnvelopeJSON(parseEnv)
	if parseErr != nil {
		return fmt.Errorf("agentbridge: SendEvent: %w", parseErr)
	}
	return parseSock.WriteFrame(parseJSON)
}

func (parseC *BridgeClient) queueEventLocked(parseEnv Envelope) {
	const parseMaxBacklog = 64
	if len(parseC.eventBacklog) == parseMaxBacklog {
		copy(parseC.eventBacklog, parseC.eventBacklog[1:])
		parseC.eventBacklog[parseMaxBacklog-1] = parseEnv
		return
	}
	parseC.eventBacklog = append(parseC.eventBacklog, parseEnv)
}
