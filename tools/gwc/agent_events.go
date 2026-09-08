package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// buildAgentTraceRepresentations summarizes hydration and commit evidence from
// structured runtime/devtools records. With no records it still returns a
// machine-readable unavailable state, so callers never need a prose sentinel.
func buildAgentTraceRepresentations(parseRecords ...map[string]any) (agentHydrationTraceRecord, agentCommitTraceRecord) {
	parseHydration := agentHydrationTraceRecord{
		Status:   "unavailable",
		Evidence: "no runtime hydration trace records were provided",
	}
	parseCommit := agentCommitTraceRecord{
		Status:   "unavailable",
		Evidence: "no runtime commit trace records were provided",
	}
	hasHydrationRecord := false
	for _, parseRecord := range parseRecords {
		if len(parseRecord) == 0 {
			continue
		}
		if isAgentHydrationTraceRecord(parseRecord) {
			hasHydrationRecord = true
			parseMismatchCount := agentTraceIntField(parseRecord, "mismatchCount", "hydration.mismatchCount")
			if parseMismatchCount > parseHydration.MismatchCount {
				parseHydration.MismatchCount = parseMismatchCount
			}
			parseMismatch := agentHydrationMismatch{
				Path:     firstNonEmpty(agentTraceStringField(parseRecord, "path", "hydration.path", "nodePath"), "/"),
				SSR:      agentTraceStringField(parseRecord, "ssr", "server", "hydration.ssr"),
				Client:   agentTraceStringField(parseRecord, "client", "browser", "hydration.client"),
				Severity: firstNonEmpty(agentTraceStringField(parseRecord, "severity", "level"), "error"),
			}
			if parseMismatch.SSR != "" || parseMismatch.Client != "" || agentTraceContainsAny(parseRecord, "mismatch", "GWC-HYDRATION") {
				parseHydration.Mismatches = append(parseHydration.Mismatches, parseMismatch)
			}
			if agentTraceBoolField(parseRecord, "failed", "hydration.failed") {
				parseHydration.Status = "failed"
			}
		}
		if parseEvent, parseOK := buildAgentCommitTraceEvent(parseRecord); parseOK {
			parseCommit.Events = append(parseCommit.Events, parseEvent)
		}
	}
	if hasHydrationRecord {
		if parseHydration.MismatchCount < len(parseHydration.Mismatches) {
			parseHydration.MismatchCount = len(parseHydration.Mismatches)
		}
		if parseHydration.MismatchCount > 0 || parseHydration.Status == "failed" {
			parseHydration.Status = "failed"
			parseHydration.Evidence = "hydration mismatch evidence captured from runtime records"
		} else {
			parseHydration.Status = "passed"
			parseHydration.Evidence = "hydration records were captured without mismatches"
		}
	}
	if len(parseCommit.Events) > 0 {
		parseCommit.Status = "captured"
		parseCommit.Evidence = "commit trace events captured from runtime records"
	}
	return parseHydration, parseCommit
}

func isAgentHydrationTraceRecord(parseRecord map[string]any) bool {
	return agentTraceContainsAny(parseRecord, "hydration", "hydrate", "GWC-HYDRATION") ||
		agentTraceStringField(parseRecord, "hydration.path", "hydration.ssr", "hydration.client") != "" ||
		agentTraceIntField(parseRecord, "mismatchCount", "hydration.mismatchCount") > 0
}

func buildAgentCommitTraceEvent(parseRecord map[string]any) (agentCommitEvent, bool) {
	isCommit := agentTraceContainsAny(parseRecord, "commit", "committed") ||
		agentTraceIntField(parseRecord, "commitCount", "commits") > 0 ||
		agentTraceStringField(parseRecord, "write", "stateWrite", "atomWrite", "component") != ""
	if !isCommit {
		return agentCommitEvent{}, false
	}
	parseEvent := agentCommitEvent{
		Source:     firstNonEmpty(agentTraceStringField(parseRecord, "source", "trigger", "atom", "state"), "runtime"),
		Write:      agentTraceStringField(parseRecord, "write", "stateWrite", "atomWrite", "operation"),
		Components: agentTraceStringSliceField(parseRecord, "components", "component"),
		Commits:    firstPositiveInt(agentTraceIntField(parseRecord, "commits", "commitCount", "count"), 1),
	}
	return parseEvent, true
}

func agentTraceContainsAny(parseRecord map[string]any, parseNeedles ...string) bool {
	parseHaystack := strings.ToLower(fmt.Sprint(parseRecord))
	for _, parseNeedle := range parseNeedles {
		if strings.Contains(parseHaystack, strings.ToLower(strings.TrimSpace(parseNeedle))) {
			return true
		}
	}
	return false
}

func agentTraceStringField(parseRecord map[string]any, parseKeys ...string) string {
	for _, parseKey := range parseKeys {
		if parseValue, parseOK := agentTraceLookup(parseRecord, parseKey); parseOK && parseValue != nil {
			return strings.TrimSpace(fmt.Sprint(parseValue))
		}
	}
	return ""
}

func agentTraceStringSliceField(parseRecord map[string]any, parseKeys ...string) []string {
	parseValues := []string{}
	for _, parseKey := range parseKeys {
		parseValue, parseOK := agentTraceLookup(parseRecord, parseKey)
		if !parseOK || parseValue == nil {
			continue
		}
		switch parseTyped := parseValue.(type) {
		case []string:
			parseValues = append(parseValues, parseTyped...)
		case []any:
			for _, parseItem := range parseTyped {
				parseText := strings.TrimSpace(fmt.Sprint(parseItem))
				if parseText != "" {
					parseValues = append(parseValues, parseText)
				}
			}
		default:
			parseText := strings.TrimSpace(fmt.Sprint(parseTyped))
			if parseText != "" {
				parseValues = append(parseValues, parseText)
			}
		}
	}
	return dedupeObserveStrings(parseValues)
}

func agentTraceIntField(parseRecord map[string]any, parseKeys ...string) int {
	for _, parseKey := range parseKeys {
		parseValue, parseOK := agentTraceLookup(parseRecord, parseKey)
		if !parseOK || parseValue == nil {
			continue
		}
		switch parseTyped := parseValue.(type) {
		case int:
			return parseTyped
		case int64:
			return int(parseTyped)
		case float64:
			return int(parseTyped)
		case json.Number:
			parseInt, _ := parseTyped.Int64()
			return int(parseInt)
		default:
			parseInt, parseErr := strconv.Atoi(strings.TrimSpace(fmt.Sprint(parseTyped)))
			if parseErr == nil {
				return parseInt
			}
		}
	}
	return 0
}

func agentTraceBoolField(parseRecord map[string]any, parseKeys ...string) bool {
	for _, parseKey := range parseKeys {
		parseValue, parseOK := agentTraceLookup(parseRecord, parseKey)
		if !parseOK || parseValue == nil {
			continue
		}
		switch parseTyped := parseValue.(type) {
		case bool:
			return parseTyped
		case string:
			return strings.EqualFold(strings.TrimSpace(parseTyped), "true")
		default:
			return strings.EqualFold(strings.TrimSpace(fmt.Sprint(parseTyped)), "true")
		}
	}
	return false
}

func agentTraceLookup(parseRecord map[string]any, parseKey string) (any, bool) {
	parseKey = strings.TrimSpace(parseKey)
	if parseKey == "" {
		return nil, false
	}
	if parseValue, parseOK := parseRecord[parseKey]; parseOK {
		return parseValue, true
	}
	parseParts := strings.Split(parseKey, ".")
	var parseCurrent any = parseRecord
	for _, parsePart := range parseParts {
		parseMap, parseOK := parseCurrent.(map[string]any)
		if !parseOK {
			return nil, false
		}
		parseCurrent, parseOK = parseMap[parsePart]
		if !parseOK {
			return nil, false
		}
	}
	return parseCurrent, true
}

func firstPositiveInt(parseValues ...int) int {
	for _, parseValue := range parseValues {
		if parseValue > 0 {
			return parseValue
		}
	}
	return 0
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
	parseCmd.Env = buildDevChildEnv(parseConfig)
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
		parseConn, _, parseErr := websocket.DefaultDialer.DialContext(parseCtx, parseWebSocketURL, nil)
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
		// A Gorilla read timeout permanently poisons the connection. Close on cancellation
		// instead, allowing idle development sessions to remain connected indefinitely.
		parseStopClose := context.AfterFunc(parseCtx, func() { _ = parseConn.Close() })
		for {
			var parseMessage agentLiveReloadMessage
			parseReadErr := parseConn.ReadJSON(&parseMessage)
			if parseReadErr != nil {
				parseStopClose()
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
		if !sleepAgentContext(parseCtx, 300*time.Millisecond) {
			return
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
