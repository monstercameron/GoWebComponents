package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const agentEventSchemaVersion = "gwc.agent.event.v1"

type agentEventStream struct {
	command string
	writer  io.Writer
	encoder *json.Encoder
	mutex   sync.Mutex
}

type agentEvent struct {
	SchemaVersion string            `json:"schemaVersion"`
	Command       string            `json:"command"`
	Event         string            `json:"event"`
	Phase         string            `json:"phase,omitempty"`
	OK            *bool             `json:"ok,omitempty"`
	Timestamp     time.Time         `json:"timestamp"`
	Data          any               `json:"data,omitempty"`
	Diagnostics   []agentDiagnostic `json:"diagnostics,omitempty"`
	Error         *agentEventError  `json:"error,omitempty"`
}

type agentDiagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Severity string `json:"severity,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

type agentEventError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type agentDevPlanRecord struct {
	App          string            `json:"app"`
	Root         string            `json:"root"`
	ProjectRoot  string            `json:"projectRoot"`
	AppMode      string            `json:"appMode"`
	ServerMode   string            `json:"serverMode"`
	HTML         string            `json:"html,omitempty"`
	WASM         string            `json:"wasm,omitempty"`
	Host         string            `json:"host"`
	Port         string            `json:"port"`
	Hot          bool              `json:"hot"`
	ListeningURL string            `json:"listeningURL"`
	StatusURL    string            `json:"statusURL,omitempty"`
	WebSocketURL string            `json:"webSocketURL,omitempty"`
	Resolution   map[string]string `json:"resolution,omitempty"`
}

type agentLiveReloadMessage struct {
	Protocol  string          `json:"protocol,omitempty"`
	Version   int             `json:"version,omitempty"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

type agentHydrationTraceRecord struct {
	Status        string                   `json:"status"`
	MismatchCount int                      `json:"mismatchCount"`
	Mismatches    []agentHydrationMismatch `json:"mismatches,omitempty"`
	Evidence      string                   `json:"evidence,omitempty"`
}

type agentHydrationMismatch struct {
	Path     string `json:"path"`
	SSR      string `json:"ssr,omitempty"`
	Client   string `json:"client,omitempty"`
	Severity string `json:"severity,omitempty"`
}

type agentCommitTraceRecord struct {
	Status   string             `json:"status"`
	Events   []agentCommitEvent `json:"events,omitempty"`
	Evidence string             `json:"evidence,omitempty"`
}

type agentCommitEvent struct {
	Source     string   `json:"source"`
	Write      string   `json:"write,omitempty"`
	Components []string `json:"components,omitempty"`
	Commits    int      `json:"commits,omitempty"`
}

type agentVerifyCheckRecord struct {
	Name       string            `json:"name"`
	Status     string            `json:"status"`
	OK         bool              `json:"ok"`
	Skipped    bool              `json:"skipped,omitempty"`
	Evidence   any               `json:"evidence,omitempty"`
	Diagnostic *agentDiagnostic  `json:"diagnostic,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// newAgentEventStream builds one synchronized NDJSON event writer.
func newAgentEventStream(parseWriter io.Writer, parseCommand string) *agentEventStream {
	if parseWriter == nil {
		parseWriter = io.Discard
	}
	return &agentEventStream{
		command: strings.TrimSpace(parseCommand),
		writer:  parseWriter,
		encoder: json.NewEncoder(parseWriter),
	}
}

// emit writes one versioned agent event as a single flushed NDJSON line.
func (parseStream *agentEventStream) emit(parseEvent agentEvent) error {
	if parseStream == nil {
		return nil
	}
	parseStream.mutex.Lock()
	defer parseStream.mutex.Unlock()
	if strings.TrimSpace(parseEvent.SchemaVersion) == "" {
		parseEvent.SchemaVersion = agentEventSchemaVersion
	}
	if strings.TrimSpace(parseEvent.Command) == "" {
		parseEvent.Command = parseStream.command
	}
	if parseEvent.Timestamp.IsZero() {
		parseEvent.Timestamp = time.Now().UTC()
	}
	if parseErr := parseStream.encoder.Encode(parseEvent); parseErr != nil {
		return parseErr
	}
	if parseFlusher, parseOk := parseStream.writer.(interface{ Flush() error }); parseOk {
		_ = parseFlusher.Flush()
	}
	return nil
}

// buildAgentBool returns a pointer suitable for optional JSON bool fields.
func buildAgentBool(parseValue bool) *bool {
	return &parseValue
}

// buildAgentDevPlanRecord converts a resolved dev config to the agent plan payload.
func buildAgentDevPlanRecord(parseConfig devConfig) agentDevPlanRecord {
	parsePlan := describeDevPlan(parseConfig)
	return agentDevPlanRecord{
		App:          parseConfig.appPath,
		Root:         parseConfig.rootPath,
		ProjectRoot:  parsePlan.ProjectRoot,
		AppMode:      parsePlan.AppMode,
		ServerMode:   parsePlan.ServerMode,
		HTML:         parseConfig.htmlPath,
		WASM:         parseConfig.wasmPath,
		Host:         parseConfig.host,
		Port:         parseConfig.port,
		Hot:          parseConfig.hot,
		ListeningURL: parsePlan.ListeningURL,
		StatusURL:    parsePlan.StatusURL,
		WebSocketURL: buildAgentDevWebSocketURL(parsePlan.ListeningURL),
		Resolution:   cloneResolutionTrace(parseConfig.resolution),
	}
}

// buildAgentDevWebSocketURL derives the livereload websocket URL from the listening URL.
func buildAgentDevWebSocketURL(parseListeningURL string) string {
	parseTrimmed := strings.TrimSpace(parseListeningURL)
	if parseTrimmed == "" {
		return ""
	}
	parseParsed, parseErr := url.Parse(parseTrimmed)
	if parseErr != nil || parseParsed.Host == "" {
		return ""
	}
	parseScheme := "ws"
	if parseParsed.Scheme == "https" {
		parseScheme = "wss"
	}
	return parseScheme + "://" + parseParsed.Host + "/ws"
}

// buildAgentTraceRepresentations returns currently available hydration and commit trace records.
func buildAgentTraceRepresentations() (agentHydrationTraceRecord, agentCommitTraceRecord) {
	return agentHydrationTraceRecord{
			Status:        "skipped",
			MismatchCount: 0,
			Evidence:      "hydration browser diff collection is represented but not yet wired into this command",
		}, agentCommitTraceRecord{
			Status:   "skipped",
			Evidence: "runtime commit trace collection is represented but not yet wired into this command",
		}
}

// buildAgentDiagnosticFromError creates one stable diagnostic from an error message.
func buildAgentDiagnosticFromError(parseCode string, parseMessage string, parseSeverity string) agentDiagnostic {
	parseDiagnostic := agentDiagnostic{
		Code:     strings.TrimSpace(parseCode),
		Message:  strings.TrimSpace(parseMessage),
		Severity: strings.TrimSpace(parseSeverity),
	}
	parseFile, parseLine, parseColumn := extractAgentFileLocation(parseMessage)
	parseDiagnostic.File = parseFile
	parseDiagnostic.Line = parseLine
	parseDiagnostic.Column = parseColumn
	return parseDiagnostic
}

// extractAgentFileLocation finds a Go compiler-style file:line[:column] location.
func extractAgentFileLocation(parseMessage string) (string, int, int) {
	parseMatcher := regexp.MustCompile(`(?m)([^\s:]+\.go):(\d+)(?::(\d+))?`)
	parseMatch := parseMatcher.FindStringSubmatch(parseMessage)
	if len(parseMatch) == 0 {
		return "", 0, 0
	}
	parseLine, _ := strconv.Atoi(parseMatch[2])
	parseColumn := 0
	if len(parseMatch) > 3 && strings.TrimSpace(parseMatch[3]) != "" {
		parseColumn, _ = strconv.Atoi(parseMatch[3])
	}
	return parseMatch[1], parseLine, parseColumn
}

// runDevAgent runs the dev server and converts process plus websocket signals to NDJSON.
func runDevAgent(parseL launcher, parseConfig devConfig, parseForwarded []string, isDryRun bool, shouldNoDoctor bool) error {
	parseStream := newAgentEventStream(os.Stdout, "dev")
	parsePlan := buildAgentDevPlanRecord(parseConfig)
	if parseErr := parseStream.emit(agentEvent{
		Event: "dev.plan",
		Phase: "plan",
		OK:    buildAgentBool(true),
		Data:  parsePlan,
	}); parseErr != nil {
		return parseErr
	}
	parseHydration, parseCommit := buildAgentTraceRepresentations()
	_ = parseStream.emit(agentEvent{
		Event: "dev.trace.representations",
		Phase: "plan",
		OK:    buildAgentBool(true),
		Data: map[string]any{
			"hydrationDiff": parseHydration,
			"commitTrace":   parseCommit,
		},
	})
	if isDryRun {
		return parseStream.emit(agentEvent{
			Event: "dev.summary",
			Phase: "complete",
			OK:    buildAgentBool(true),
			Data: map[string]any{
				"dryRun": true,
			},
		})
	}

	parseCmd := exec.Command("go", parseForwarded...)
	parseCmd.Dir = parseL.repoRoot
	parseCmd.Env = os.Environ()
	parseCmd.Stdin = os.Stdin
	parseStdout, parseErr := parseCmd.StdoutPipe()
	if parseErr != nil {
		return parseErr
	}
	parseStderr, parseErr := parseCmd.StderrPipe()
	if parseErr != nil {
		return parseErr
	}
	if parseErr2 := parseCmd.Start(); parseErr2 != nil {
		return parseErr2
	}
	_ = parseStream.emit(agentEvent{
		Event: "dev.process.started",
		Phase: "startup",
		OK:    buildAgentBool(true),
		Data: map[string]any{
			"pid": parseCmd.Process.Pid,
		},
	})

	parseCtx, parseCancel := context.WithCancel(context.Background())
	defer parseCancel()
	parsePumpDone := make(chan struct{}, 2)
	go streamAgentProcessLines(parseStream, parseStdout, "stdout", parsePumpDone)
	go streamAgentProcessLines(parseStream, parseStderr, "stderr", parsePumpDone)
	parseWebSocketDone := make(chan struct{})
	if strings.TrimSpace(parsePlan.WebSocketURL) != "" && parsePlan.ServerMode == "livereload-wasm" {
		go streamAgentDevWebSocket(parseCtx, parseStream, parsePlan.WebSocketURL, parseWebSocketDone)
	} else {
		close(parseWebSocketDone)
	}

	parseRunErr := parseCmd.Wait()
	parseCancel()
	<-parsePumpDone
	<-parsePumpDone
	select {
	case <-parseWebSocketDone:
	case <-time.After(1500 * time.Millisecond):
	}

	if parseRunErr != nil && !shouldNoDoctor {
		parseReport := parseL.buildDoctorReport(doctorConfig{host: defaultHost, port: "8080"})
		parseSummary, parseShow := formatEnvironmentDiagnosis(parseReport)
		if parseShow {
			_ = parseStream.emit(agentEvent{
				Event: "doctor.diagnosis",
				Phase: "diagnose",
				OK:    buildAgentBool(false),
				Data: map[string]any{
					"summary": strings.TrimSpace(parseSummary),
					"report":  parseReport,
				},
			})
		}
	}
	parseOK := parseRunErr == nil
	parseEvent := agentEvent{
		Event: "dev.process.exited",
		Phase: "complete",
		OK:    buildAgentBool(parseOK),
		Data: map[string]any{
			"pid": parseCmd.Process.Pid,
		},
	}
	if parseRunErr != nil {
		parseMessage := parseRunErr.Error()
		parseDiagnostic := buildAgentDiagnosticFromError("GWC_AGENT_DEV_EXIT", parseMessage, "error")
		parseEvent.Diagnostics = []agentDiagnostic{parseDiagnostic}
		parseEvent.Error = &agentEventError{Code: parseDiagnostic.Code, Message: parseMessage}
	}
	if parseErr2 := parseStream.emit(parseEvent); parseErr2 != nil {
		return parseErr2
	}
	return parseRunErr
}

// streamAgentProcessLines emits child process stdout/stderr as structured log events.
func streamAgentProcessLines(parseStream *agentEventStream, parseReader io.Reader, parseStreamName string, parseDone chan<- struct{}) {
	defer func() {
		parseDone <- struct{}{}
	}()
	parseScanner := bufio.NewScanner(parseReader)
	parseScanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for parseScanner.Scan() {
		parseLine := strings.TrimSpace(parseScanner.Text())
		if parseLine == "" {
			continue
		}
		_ = parseStream.emit(agentEvent{
			Event: "dev.process.log",
			Phase: "runtime",
			Data: map[string]any{
				"stream": parseStreamName,
				"line":   parseLine,
			},
		})
	}
	if parseErr := parseScanner.Err(); parseErr != nil {
		parseDiagnostic := buildAgentDiagnosticFromError("GWC_AGENT_PROCESS_LOG", parseErr.Error(), "warning")
		_ = parseStream.emit(agentEvent{
			Event:       "dev.process.log_error",
			Phase:       "runtime",
			OK:          buildAgentBool(false),
			Diagnostics: []agentDiagnostic{parseDiagnostic},
			Error:       &agentEventError{Code: parseDiagnostic.Code, Message: parseErr.Error()},
		})
	}
}

// streamAgentDevWebSocket tails livereload websocket messages into agent events.
func streamAgentDevWebSocket(parseCtx context.Context, parseStream *agentEventStream, parseWebSocketURL string, parseDone chan<- struct{}) {
	defer close(parseDone)
	parseHadFailure := false
	parseConnected := false
	for {
		select {
		case <-parseCtx.Done():
			return
		default:
		}
		parseConn, _, parseErr := websocket.DefaultDialer.Dial(parseWebSocketURL, nil)
		if parseErr != nil {
			if !parseConnected {
				_ = parseStream.emit(agentEvent{
					Event: "dev.websocket.waiting",
					Phase: "startup",
					Data: map[string]any{
						"url":   parseWebSocketURL,
						"error": parseErr.Error(),
					},
				})
				parseConnected = true
			}
			if !sleepAgentContext(parseCtx, 300*time.Millisecond) {
				return
			}
			continue
		}
		_ = parseStream.emit(agentEvent{
			Event: "dev.websocket.connected",
			Phase: "startup",
			OK:    buildAgentBool(true),
			Data: map[string]any{
				"url": parseWebSocketURL,
			},
		})
		for {
			_ = parseConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			var parseMessage agentLiveReloadMessage
			parseReadErr := parseConn.ReadJSON(&parseMessage)
			if parseReadErr != nil {
				if parseNetErr, parseOk := parseReadErr.(net.Error); parseOk && parseNetErr.Timeout() {
					select {
					case <-parseCtx.Done():
						_ = parseConn.Close()
						return
					default:
						continue
					}
				}
				_ = parseConn.Close()
				break
			}
			parseEvent, parseFailed, parseRecovered := buildAgentEventFromLiveReloadMessage(parseMessage, parseHadFailure)
			if parseEvent.Event != "" {
				_ = parseStream.emit(parseEvent)
			}
			if parseRecovered {
				_ = parseStream.emit(agentEvent{
					Event: "recovered",
					Phase: "build",
					OK:    buildAgentBool(true),
					Data: map[string]any{
						"source": "build_complete",
					},
				})
			}
			parseHadFailure = parseFailed
		}
	}
}

// buildAgentEventFromLiveReloadMessage maps one livereload message to an agent event.
func buildAgentEventFromLiveReloadMessage(parseMessage agentLiveReloadMessage, wasFailed bool) (agentEvent, bool, bool) {
	parsePayload := parseRawPayloadMap(parseMessage.Payload)
	parseEvent := agentEvent{
		Event:     "dev." + strings.ReplaceAll(strings.TrimSpace(parseMessage.Type), "_", "."),
		Phase:     "runtime",
		Timestamp: parseMessage.Timestamp,
		Data: map[string]any{
			"type":    parseMessage.Type,
			"payload": parsePayload,
		},
	}
	parseFailed := wasFailed
	parseRecovered := false
	switch parseMessage.Type {
	case "build_start":
		parseEvent.Event = "build.started"
		parseEvent.Phase = "build"
	case "build_complete", "build_error":
		parseEvent.Phase = "build"
		parseSuccess, parseHasSuccess := parsePayloadBool(parsePayload, "success")
		if parseHasSuccess && parseSuccess {
			parseEvent.Event = "recompiled"
			parseEvent.OK = buildAgentBool(true)
			parseRecovered = wasFailed
			parseFailed = false
			break
		}
		parseEvent.Event = "error"
		parseEvent.OK = buildAgentBool(false)
		parseMessageText := parsePayloadString(parsePayload, "error")
		if parseMessageText == "" {
			parseMessageText = "dev build failed"
		}
		parseDiagnostic := buildAgentDiagnosticFromError("GWC_AGENT_BUILD_ERROR", parseMessageText, "error")
		parseEvent.Diagnostics = []agentDiagnostic{parseDiagnostic}
		parseEvent.Error = &agentEventError{Code: parseDiagnostic.Code, Message: parseMessageText}
		parseFailed = true
	case "asset_swap":
		parseEvent.Event = "asset.swap"
		parseEvent.Phase = "runtime"
		parseEvent.OK = buildAgentBool(true)
	case "watcher_degraded":
		parseEvent.Event = "watcher.degraded"
		parseEvent.Phase = "runtime"
		parseEvent.OK = buildAgentBool(false)
	case "current_status":
		parseEvent.Event = "dev.status"
	default:
		if strings.TrimSpace(parseMessage.Type) == "" {
			parseEvent.Event = ""
		}
	}
	return parseEvent, parseFailed, parseRecovered
}

// parseRawPayloadMap decodes a JSON object payload for agent forwarding.
func parseRawPayloadMap(parsePayload json.RawMessage) map[string]any {
	if len(parsePayload) == 0 {
		return map[string]any{}
	}
	var parseDecoded map[string]any
	if parseErr := json.Unmarshal(parsePayload, &parseDecoded); parseErr != nil {
		return map[string]any{"raw": string(parsePayload)}
	}
	return parseDecoded
}

// parsePayloadBool reads a bool from a decoded payload map.
func parsePayloadBool(parsePayload map[string]any, parseKey string) (bool, bool) {
	parseValue, parseOk := parsePayload[parseKey]
	if !parseOk {
		return false, false
	}
	parseBool, parseOk := parseValue.(bool)
	return parseBool, parseOk
}

// parsePayloadString reads a string from a decoded payload map.
func parsePayloadString(parsePayload map[string]any, parseKey string) string {
	parseValue, parseOk := parsePayload[parseKey]
	if !parseOk || parseValue == nil {
		return ""
	}
	parseString, parseOk := parseValue.(string)
	if parseOk {
		return strings.TrimSpace(parseString)
	}
	return strings.TrimSpace(fmt.Sprint(parseValue))
}

// sleepAgentContext waits for a duration or exits when the context is done.
func sleepAgentContext(parseCtx context.Context, parseDuration time.Duration) bool {
	parseTimer := time.NewTimer(parseDuration)
	defer parseTimer.Stop()
	select {
	case <-parseCtx.Done():
		return false
	case <-parseTimer.C:
		return true
	}
}

// emitAgentVerifyCheck writes one structured verify check event.
func emitAgentVerifyCheck(parseStream *agentEventStream, parseCheck agentVerifyCheckRecord) error {
	if parseStream == nil {
		return nil
	}
	parseOK := parseCheck.OK
	if parseCheck.Skipped {
		parseOK = true
	}
	parseEvent := agentEvent{
		Event: "verify.check",
		Phase: "verify",
		OK:    buildAgentBool(parseOK),
		Data:  parseCheck,
	}
	if parseCheck.Diagnostic != nil {
		parseEvent.Diagnostics = []agentDiagnostic{*parseCheck.Diagnostic}
		if !parseOK {
			parseEvent.Error = &agentEventError{Code: parseCheck.Diagnostic.Code, Message: parseCheck.Diagnostic.Message}
		}
	}
	return parseStream.emit(parseEvent)
}

// emitAgentVerifySummary writes the final verify summary event.
func emitAgentVerifySummary(parseStream *agentEventStream, parseSummary verifySummary) error {
	if parseStream == nil {
		return nil
	}
	return parseStream.emit(agentEvent{
		Event: "verify.summary",
		Phase: "complete",
		OK:    buildAgentBool(parseSummary.OK),
		Data:  parseSummary,
	})
}
