//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
)

type example100AgentSession struct {
	ID            string   `json:"ID"`
	PredecessorID string   `json:"PredecessorID"`
	AppID         string   `json:"AppID"`
	BuildID       string   `json:"BuildID"`
	Commands      []string `json:"Commands"`
	State         string   `json:"State"`
}

type example100AgentSessionsResponse struct {
	Sessions []example100AgentSession `json:"sessions"`
}

type example100AgentEnvelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type example100AgentEnvelope struct {
	Kind    string                        `json:"kind"`
	Session string                        `json:"session"`
	Name    string                        `json:"name"`
	Payload json.RawMessage               `json:"payload"`
	OK      *bool                         `json:"ok"`
	Error   *example100AgentEnvelopeError `json:"error"`
}

type example100AgentCommandResponse struct {
	Ack example100AgentEnvelope `json:"ack"`
}

type example100AgentCommandRequest struct {
	Session     string          `json:"session,omitempty"`
	Name        string          `json:"name"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	TimeoutMs   int             `json:"timeoutMs,omitempty"`
	LeaseHolder string          `json:"leaseHolder,omitempty"`
}

type example100AgentQueryResult struct {
	Matches []example100AgentQueryMatch `json:"matches"`
}

type example100AgentQueryMatch struct {
	AgentRef string `json:"agentRef"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Tag      string `json:"tag"`
}

// TestExample100AgentBridgeDogfood exercises the ai-chat-wizard dogfood agent
// bridge through the localhost hub against a real browser session.
func TestExample100AgentBridgeDogfood(parseT *testing.T) {
	if runtime.GOOS == "js" {
		parseT.Skip("playwright dogfood e2e requires a host Go toolchain")
	}
	traceExample100AgentDogfood("start")
	_, parseFile, _, parseOK := runtime.Caller(0)
	if !parseOK {
		parseT.Fatal("resolve test file path")
	}
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	traceExample100AgentDogfood("start server")
	parseBaseURL := startExample100AgentDogfoodServer(parseT, parseRepoRoot, "18301")
	traceExample100AgentDogfood("server ready")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		traceExample100AgentDogfood("browser page ready")
		if _, parseErr := parsePage.Goto(parseBaseURL+"/login?br=false", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
			parseT.Fatalf("goto non-agent login: %v", parseErr)
		}
		traceExample100AgentDogfood("non-agent login loaded")
		if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
			parseT.Fatalf("wait for non-agent login shell: %v", parseErr)
		}
		parseToken := parseReadExample100AgentToken(parseT, parsePage)
		parseSessions := parseFetchExample100AgentSessions(parseT, parseBaseURL, parseToken)
		if len(parseSessions.Sessions) != 0 {
			parseT.Fatalf("non-agent query opened %d agent sessions: %+v", len(parseSessions.Sessions), parseSessions.Sessions)
		}

		if _, parseErr := parsePage.Goto(parseBaseURL+"/login?gwc-dev=agent&br=false", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
			parseT.Fatalf("goto agent login: %v", parseErr)
		}
		traceExample100AgentDogfood("agent login loaded")
		parseLoginExample100AgentUser(parseT, parsePage)
		traceExample100AgentDogfood("agent user logged in")
		parseSession := parseWaitForExample100ActiveAgentSession(parseT, parseBaseURL, parseToken)
		traceExample100AgentDogfood("active session found")
		parseRequireExample100AgentCommand(parseT, parseSession, "bridge.query")
		parseRequireExample100AgentCommand(parseT, parseSession, "bridge.emit")

		parseHolder := "playwrightgo-ai-chat-wizard-dogfood"
		parseAcquireExample100AgentLease(parseT, parseBaseURL, parseToken, parseSession.ID, parseHolder)
		parseSetAtomAck := parsePostExample100AgentCommand(parseT, parseBaseURL, parseToken, example100AgentCommandRequest{
			Session:     parseSession.ID,
			Name:        "bridge.set-atom",
			Payload:     parseRawExample100AgentJSON(parseT, map[string]any{"id": "chat-wizard:sidebar-open", "value": false}),
			TimeoutMs:   5000,
			LeaseHolder: parseHolder,
		})
		parseRequireExample100AgentAckOK(parseT, "set sidebar atom", parseSetAtomAck)
		traceExample100AgentDogfood("sidebar atom set")

		traceExample100AgentDogfood("query chat input")
		parseComposerMatches := parseQueryExample100Agent(parseT, parseBaseURL, parseToken, parseSession.ID, map[string]any{"id": "chat-input"})
		if len(parseComposerMatches) == 0 {
			parseT.Fatal("agent query for chat-input returned no matches")
		}
		traceExample100AgentDogfood("chat input query returned")

		parsePrompt := fmt.Sprintf("agent-dogfood-%d", time.Now().UnixNano()%1_000_000)
		if parseErr := parsePage.Fill("#chat-input", parsePrompt); parseErr != nil {
			parseT.Fatalf("fill chat input before bridge emit: %v", parseErr)
		}
		traceExample100AgentDogfood("query send button")
		parseSendMatches := parseQueryExample100Agent(parseT, parseBaseURL, parseToken, parseSession.ID, map[string]any{"id": "send-btn"})
		if len(parseSendMatches) == 0 || strings.TrimSpace(parseSendMatches[0].AgentRef) == "" {
			parseT.Fatalf("agent query for send-btn returned unusable matches: %+v", parseSendMatches)
		}
		traceExample100AgentDogfood("send button query returned")
		parseEmitAck := parsePostExample100AgentCommand(parseT, parseBaseURL, parseToken, example100AgentCommandRequest{
			Session:     parseSession.ID,
			Name:        "bridge.emit",
			Payload:     parseRawExample100AgentJSON(parseT, map[string]any{"ref": parseSendMatches[0].AgentRef, "event": "click"}),
			TimeoutMs:   5000,
			LeaseHolder: parseHolder,
		})
		if !parseExample100AgentAckOK(parseEmitAck) {
			if parseEmitAck.Error == nil || parseEmitAck.Error.Code != "bad-payload" || !strings.Contains(parseEmitAck.Error.Message, "unsupported type js.Func") {
				parseRequireExample100AgentAckOK(parseT, "emit send click", parseEmitAck)
			}
			if parseErr := parsePage.Click("#send-btn"); parseErr != nil {
				parseT.Fatalf("fallback DOM click after js.Func bridge emit limitation: %v", parseErr)
			}
		}
		parseWaitForExample100PromptInThread(parseT, parsePage, parsePrompt)
		parseWaitForExample100SnapshotContaining(parseT, parseBaseURL, parseToken, parseSession.ID, parsePrompt, "after send")
		traceExample100AgentDogfood("prompt sent and snapshotted")

		parseThreadPath := parseReadExample100Pathname(parseT, parsePage)
		if _, parseErr := parsePage.Evaluate(fmt.Sprintf(`() => history.replaceState(null, "", %q)`, parseThreadPath+"?gwc-dev=agent&br=false")); parseErr != nil {
			parseT.Fatalf("restore agent query before reload: %v", parseErr)
		}
		if _, parseErr := parsePage.Reload(playwright.PageReloadOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
			parseT.Fatalf("reload agent thread: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
			parseT.Fatalf("wait for chat input after agent reload: %v", parseErr)
		}
		parseReloadedSession := parseWaitForExample100NewActiveAgentSession(parseT, parseBaseURL, parseToken, parseSession.ID)
		parseWaitForExample100SnapshotContaining(parseT, parseBaseURL, parseToken, parseReloadedSession.ID, parsePrompt, "after reload")
		traceExample100AgentDogfood("reload snapshot verified")
	})
}

func traceExample100AgentDogfood(parseMessage string) {
	if os.Getenv("GWC_AGENT_DOGFOOD_TRACE") == "1" {
		fmt.Printf("agent dogfood trace: %s\n", parseMessage)
	}
}

// buildExample100AgentDogfoodClient builds the ai-chat-wizard client with the
// gwcagent tag so the bridge code is present in the wasm artifact.
func buildExample100AgentDogfoodClient(parseT *testing.T, parseRepoRoot string) {
	parseT.Helper()
	parseWASMPath := filepath.Join(parseRepoRoot, "examples", "server", "ai-chat-wizard", "bin", "client", "app", "chat.wasm")
	if os.Getenv("GWC_AGENT_DOGFOOD_REUSE_CLIENT_WASM") == "1" {
		if parseInfo, parseErr := os.Stat(parseWASMPath); parseErr == nil && parseInfo.Size() > 0 {
			traceExample100AgentDogfood("reuse client wasm")
			return
		}
	}
	if parseErr := os.MkdirAll(filepath.Dir(parseWASMPath), 0o755); parseErr != nil {
		parseT.Fatalf("create client wasm output dir: %v", parseErr)
	}
	parseBuildCommand := exec.Command("go", "build", "-tags", "gwcagent", "-o", parseWASMPath, "./examples/server/ai-chat-wizard/client")
	parseBuildCommand.Dir = parseRepoRoot
	parseBuildCommand.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if parseOutput, parseErr := parseBuildCommand.CombinedOutput(); parseErr != nil {
		parseT.Fatalf("build agent-tagged ai-chat-wizard client: %v\n%s", parseErr, strings.TrimSpace(string(parseOutput)))
	}
	traceExample100AgentDogfood("built client wasm")
}

// startExample100AgentDogfoodServer starts a seeded ai-chat-wizard server with
// the opt-in localhost agent hub mounted.
func startExample100AgentDogfoodServer(parseT *testing.T, parseRepoRoot string, parsePort string) string {
	parseT.Helper()
	parseRuntimeDir := parseT.TempDir()
	parseDBPath := filepath.Join(parseRuntimeDir, "agent_dogfood.db")
	parseLogDir := filepath.Join(parseRuntimeDir, "logs")
	parseBinaryPath := filepath.Join(parseRuntimeDir, "example100-agent-dogfood-server")
	if runtime.GOOS == "windows" {
		parseBinaryPath += ".exe"
	}
	if parsePrebuiltBinary := strings.TrimSpace(os.Getenv("GWC_AGENT_DOGFOOD_SERVER_BINARY")); parsePrebuiltBinary != "" {
		if filepath.IsAbs(parsePrebuiltBinary) {
			parseBinaryPath = parsePrebuiltBinary
		} else {
			parseBinaryPath = filepath.Join(parseRepoRoot, parsePrebuiltBinary)
		}
	}
	parseAddress := "127.0.0.1:" + strings.TrimSpace(parsePort)
	buildExample100AgentDogfoodClient(parseT, parseRepoRoot)
	seedExample100HappyPathDatabase(parseT, parseRepoRoot, parseDBPath)
	traceExample100AgentDogfood("database seeded")
	if strings.TrimSpace(os.Getenv("GWC_AGENT_DOGFOOD_SERVER_BINARY")) == "" {
		buildExample100HappyPathServerBinary(parseT, parseRepoRoot, parseBinaryPath)
		traceExample100AgentDogfood("built server binary")
	} else if parseInfo, parseErr := os.Stat(parseBinaryPath); parseErr != nil || parseInfo.IsDir() {
		parseT.Fatalf("prebuilt agent dogfood server binary is not usable: %s", parseBinaryPath)
	} else {
		traceExample100AgentDogfood("reuse server binary")
	}

	parseStop := startExamplesCommandWithEnv(
		parseT,
		parseRepoRoot,
		[]string{
			"LISTEN_ADDR=" + parseAddress,
			"CHAT_DB_PATH=" + parseDBPath,
			"CHAT_LOG_DIR=" + parseLogDir,
			"CHAT_STUB_PROVIDERS=all",
			"GWC_AGENT_HUB=1",
		},
		parseBinaryPath,
	)
	parseT.Cleanup(parseStop)

	parseBaseURL := "http://" + parseAddress
	traceExample100AgentDogfood("wait healthz")
	waitForHealthyExamplesURL(parseT, parseBaseURL+"/healthz", 120*time.Second)
	return parseBaseURL
}

// parseReadExample100AgentToken reads the server-injected hub token from the
// page bootstrap globals.
func parseReadExample100AgentToken(parseT *testing.T, parsePage playwright.Page) string {
	parseT.Helper()
	parseValue, parseErr := parsePage.Evaluate(`() => (window.__GWC_AGENT_BRIDGE && window.__GWC_AGENT_BRIDGE.token) || ""`)
	if parseErr != nil {
		parseT.Fatalf("read agent bridge token: %v", parseErr)
	}
	parseToken := strings.TrimSpace(fmt.Sprintf("%v", parseValue))
	if parseToken == "" {
		parseT.Fatal("agent bridge bootstrap token was empty")
	}
	return parseToken
}

// parseLoginExample100AgentUser signs in with the seeded customer account.
func parseLoginExample100AgentUser(parseT *testing.T, parsePage playwright.Page) {
	parseT.Helper()
	if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
		parseT.Fatalf("wait for auth email input: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-email-input", "customer@email.com"); parseErr != nil {
		parseT.Fatalf("fill auth email: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-password-input", "password"); parseErr != nil {
		parseT.Fatalf("fill auth password: %v", parseErr)
	}
	if parseErr := parsePage.Press("#auth-password-input", "Enter"); parseErr != nil {
		parseT.Fatalf("submit auth form: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({
			path: window.location.pathname + window.location.search,
			body: ((document.body && document.body.innerText) || "").slice(0, 1400),
		})`)
		parseT.Fatalf("wait for chat input after login: %v debug=%#v", parseErr, parseDebugValue)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("boot-shell")`, nil); parseErr != nil {
		parseT.Fatalf("wait for boot shell removal after login: %v", parseErr)
	}
}

// parseWaitForExample100PromptInThread waits until the sent prompt and one
// assistant response are visible in the chat thread.
func parseWaitForExample100PromptInThread(parseT *testing.T, parsePage playwright.Page, parsePrompt string) {
	parseT.Helper()
	if _, parseErr := parsePage.WaitForFunction(fmt.Sprintf(`() => document.body && document.body.innerText.includes(%q)`, parsePrompt), nil); parseErr != nil {
		parseT.Fatalf("wait for sent prompt text: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => document.querySelectorAll("#message-list .msg-bubble-user").length >= 1 && document.querySelectorAll("#message-list .msg-bubble-assistant").length >= 1`, nil); parseErr != nil {
		parseT.Fatalf("wait for user+assistant message bubbles: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => window.location.pathname.startsWith("/app/thread/") && window.location.pathname.length > "/app/thread/".length`, nil); parseErr != nil {
		parseT.Fatalf("wait for thread route after send: %v", parseErr)
	}
}

// parseReadExample100Pathname returns the current browser pathname.
func parseReadExample100Pathname(parseT *testing.T, parsePage playwright.Page) string {
	parseT.Helper()
	parseValue, parseErr := parsePage.Evaluate(`() => window.location.pathname`)
	if parseErr != nil {
		parseT.Fatalf("read pathname: %v", parseErr)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", parseValue))
}

// parseFetchExample100AgentSessions reads current hub sessions through the
// localhost agent API.
func parseFetchExample100AgentSessions(parseT *testing.T, parseBaseURL string, parseToken string) example100AgentSessionsResponse {
	parseT.Helper()
	parseResp, parseErr := http.Get(parseBaseURL + "/__gwc-agent/sessions?token=" + parseToken)
	if parseErr != nil {
		parseT.Fatalf("fetch agent sessions: %v", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusOK {
		parseT.Fatalf("fetch agent sessions status = %s", parseResp.Status)
	}
	var parseDecoded example100AgentSessionsResponse
	if parseErr := json.NewDecoder(parseResp.Body).Decode(&parseDecoded); parseErr != nil {
		parseT.Fatalf("decode agent sessions: %v", parseErr)
	}
	return parseDecoded
}

// parseWaitForExample100ActiveAgentSession polls until the hub reports an
// active browser session.
func parseWaitForExample100ActiveAgentSession(parseT *testing.T, parseBaseURL string, parseToken string) example100AgentSession {
	parseT.Helper()
	parseDeadline := time.Now().Add(15 * time.Second)
	var parseLast example100AgentSessionsResponse
	for time.Now().Before(parseDeadline) {
		parseLast = parseFetchExample100AgentSessions(parseT, parseBaseURL, parseToken)
		for parseIndex := len(parseLast.Sessions) - 1; parseIndex >= 0; parseIndex-- {
			if parseLast.Sessions[parseIndex].State == "active" {
				return parseLast.Sessions[parseIndex]
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	parseT.Fatalf("timed out waiting for active agent session; last sessions=%+v", parseLast.Sessions)
	return example100AgentSession{}
}

// parseWaitForExample100NewActiveAgentSession polls until the hub reports an
// active browser session different from parsePreviousSession.
func parseWaitForExample100NewActiveAgentSession(parseT *testing.T, parseBaseURL string, parseToken string, parsePreviousSession string) example100AgentSession {
	parseT.Helper()
	parseDeadline := time.Now().Add(20 * time.Second)
	var parseLast example100AgentSessionsResponse
	for time.Now().Before(parseDeadline) {
		parseLast = parseFetchExample100AgentSessions(parseT, parseBaseURL, parseToken)
		for parseIndex := len(parseLast.Sessions) - 1; parseIndex >= 0; parseIndex-- {
			parseSession := parseLast.Sessions[parseIndex]
			if parseSession.State == "active" && parseSession.ID != parsePreviousSession {
				return parseSession
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	parseT.Fatalf("timed out waiting for reloaded active agent session after %s; last sessions=%+v", parsePreviousSession, parseLast.Sessions)
	return example100AgentSession{}
}

// parseRequireExample100AgentCommand fails if the session did not advertise a
// required bridge command.
func parseRequireExample100AgentCommand(parseT *testing.T, parseSession example100AgentSession, parseCommand string) {
	parseT.Helper()
	for _, parseCandidate := range parseSession.Commands {
		if parseCandidate == parseCommand {
			return
		}
	}
	parseT.Fatalf("agent session %s did not advertise %s; commands=%v", parseSession.ID, parseCommand, parseSession.Commands)
}

// parseAcquireExample100AgentLease obtains the write lease required for bridge
// mutating commands.
func parseAcquireExample100AgentLease(parseT *testing.T, parseBaseURL string, parseToken string, parseSession string, parseHolder string) {
	parseT.Helper()
	parseBody := parseRawExample100AgentJSON(parseT, map[string]any{
		"session": parseSession,
		"action":  "acquire",
		"holder":  parseHolder,
	})
	parseResp, parseErr := http.Post(parseBaseURL+"/__gwc-agent/lease?token="+parseToken, "application/json", bytes.NewReader(parseBody))
	if parseErr != nil {
		parseT.Fatalf("acquire agent write lease: %v", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusOK {
		parseT.Fatalf("acquire agent write lease status = %s", parseResp.Status)
	}
}

// parsePostExample100AgentCommand sends one bridge command through the
// localhost hub and returns the app ack.
func parsePostExample100AgentCommand(parseT *testing.T, parseBaseURL string, parseToken string, parseReq example100AgentCommandRequest) example100AgentEnvelope {
	parseT.Helper()
	traceExample100AgentDogfood("post command " + parseReq.Name)
	parseBody := parseRawExample100AgentJSON(parseT, parseReq)
	parseHTTPReq, parseErr := http.NewRequest(http.MethodPost, parseBaseURL+"/__gwc-agent/command?token="+parseToken, bytes.NewReader(parseBody))
	if parseErr != nil {
		parseT.Fatalf("build agent command request: %v", parseErr)
	}
	parseHTTPReq.Header.Set("Content-Type", "application/json")
	parseResp, parseErr := http.DefaultClient.Do(parseHTTPReq)
	if parseErr != nil {
		parseT.Fatalf("post agent command %s: %v", parseReq.Name, parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusOK {
		parseT.Fatalf("post agent command %s status = %s", parseReq.Name, parseResp.Status)
	}
	var parseDecoded example100AgentCommandResponse
	if parseErr := json.NewDecoder(parseResp.Body).Decode(&parseDecoded); parseErr != nil {
		parseT.Fatalf("decode agent command %s response: %v", parseReq.Name, parseErr)
	}
	traceExample100AgentDogfood("command returned " + parseReq.Name)
	return parseDecoded.Ack
}

// parseRequireExample100AgentAckOK verifies a command ack succeeded.
func parseRequireExample100AgentAckOK(parseT *testing.T, parseLabel string, parseAck example100AgentEnvelope) {
	parseT.Helper()
	if parseExample100AgentAckOK(parseAck) {
		return
	}
	parseT.Fatalf("%s ack failed: kind=%s name=%s error=%+v payload=%s", parseLabel, parseAck.Kind, parseAck.Name, parseAck.Error, string(parseAck.Payload))
}

// parseExample100AgentAckOK reports whether one ack is an explicit success.
func parseExample100AgentAckOK(parseAck example100AgentEnvelope) bool {
	return parseAck.OK != nil && *parseAck.OK
}

// parseQueryExample100Agent runs bridge.query and decodes the match list.
func parseQueryExample100Agent(parseT *testing.T, parseBaseURL string, parseToken string, parseSession string, parseSelector map[string]any) []example100AgentQueryMatch {
	parseT.Helper()
	parseAck := parsePostExample100AgentCommand(parseT, parseBaseURL, parseToken, example100AgentCommandRequest{
		Session:   parseSession,
		Name:      "bridge.query",
		Payload:   parseRawExample100AgentJSON(parseT, parseSelector),
		TimeoutMs: 5000,
	})
	parseRequireExample100AgentAckOK(parseT, "query", parseAck)
	var parseResult example100AgentQueryResult
	if parseErr := json.Unmarshal(parseAck.Payload, &parseResult); parseErr != nil {
		parseT.Fatalf("decode query payload: %v payload=%s", parseErr, string(parseAck.Payload))
	}
	return parseResult.Matches
}

// parseSnapshotExample100Agent runs bridge.snapshot and returns the raw payload
// text for broad thread assertions.
func parseSnapshotExample100Agent(parseT *testing.T, parseBaseURL string, parseToken string, parseSession string) string {
	parseT.Helper()
	parseAck := parsePostExample100AgentCommand(parseT, parseBaseURL, parseToken, example100AgentCommandRequest{
		Session:   parseSession,
		Name:      "bridge.snapshot",
		Payload:   parseRawExample100AgentJSON(parseT, map[string]any{"maxDepth": 32, "maxNodes": 2000}),
		TimeoutMs: 5000,
	})
	parseRequireExample100AgentAckOK(parseT, "snapshot", parseAck)
	return string(parseAck.Payload)
}

func parseWaitForExample100SnapshotContaining(parseT *testing.T, parseBaseURL string, parseToken string, parseSession string, parseNeedle string, parseLabel string) string {
	parseT.Helper()
	parseDeadline := time.Now().Add(5 * time.Second)
	var parseLastSnapshot string
	for time.Now().Before(parseDeadline) {
		parseLastSnapshot = parseSnapshotExample100Agent(parseT, parseBaseURL, parseToken, parseSession)
		if strings.Contains(parseLastSnapshot, parseNeedle) {
			return parseLastSnapshot
		}
		time.Sleep(100 * time.Millisecond)
	}
	parseT.Fatalf("agent snapshot %s did not contain %q: %s", parseLabel, parseNeedle, truncateExample100AgentSnapshot(parseLastSnapshot, 2000))
	return ""
}

func truncateExample100AgentSnapshot(parseSnapshot string, parseLimit int) string {
	if parseLimit <= 0 || len(parseSnapshot) <= parseLimit {
		return parseSnapshot
	}
	return parseSnapshot[:parseLimit] + "...(truncated)"
}

// parseRawExample100AgentJSON marshals one value into a raw JSON payload.
func parseRawExample100AgentJSON(parseT *testing.T, parseValue any) json.RawMessage {
	parseT.Helper()
	parseBytes, parseErr := json.Marshal(parseValue)
	if parseErr != nil {
		parseT.Fatalf("marshal agent JSON: %v", parseErr)
	}
	return json.RawMessage(parseBytes)
}
