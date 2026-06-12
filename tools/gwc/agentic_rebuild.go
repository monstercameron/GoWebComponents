package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// runRebuildCommand routes the live-session rebuild command.
var runRebuildCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runRebuild(parseArgs)
}

type rebuildConfig struct {
	appPath   string
	rootPath  string
	output    string
	profile   string
	session   string
	hub       string
	token     string
	timeoutMs int
	json      bool
}

type rebuildReport struct {
	OK                bool          `json:"ok"`
	Hub               string        `json:"hub"`
	OldSession        string        `json:"oldSession,omitempty"`
	SuccessorSession  string        `json:"successorSession,omitempty"`
	BuildID           string        `json:"buildId,omitempty"`
	SnapshotCaptured  bool          `json:"snapshotCaptured"`
	StateRestored     bool          `json:"stateRestored"`
	Phase             string        `json:"phase,omitempty"`
	Build             *buildSummary `json:"build,omitempty"`
	CompilerOutput    string        `json:"compilerOutput,omitempty"`
	APIPaths          []string      `json:"apiPaths"`
	RestoredStateNote string        `json:"restoredStateNote,omitempty"`
}

func (parseL launcher) runRebuild(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"app", "main", "root", "out", "output", "profile", "session", "hub", "token", "timeout"}, []string{"json"})
	parseFlags := flag.NewFlagSet("rebuild", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseApp := parseFlags.String("app", "", "Path to the app main.go file or app directory")
	parseMain := parseFlags.String("main", "", "(deprecated) alias for -app; use -app")
	parseRoot := parseFlags.String("root", "", "Project root used for output resolution")
	parseOut := parseFlags.String("out", "", "WASM output path")
	parseOutput := parseFlags.String("output", "", "(deprecated) alias for -out; use -out")
	parseProfile := parseFlags.String("profile", "development", "Build profile: development, debug, ci, benchmark, release, or tinygo")
	parseSession := parseFlags.String("session", "", "Optional live agent session id; defaults to the most recently active session")
	parseHub := parseFlags.String("hub", firstNonEmpty(os.Getenv("GWC_AGENT_HUB_URL"), ""), "Agent hub base URL, for example http://127.0.0.1:8090")
	parseToken := parseFlags.String("token", firstNonEmpty(os.Getenv("GWC_AGENT_TOKEN"), ""), "Agent hub token; defaults to GWC_AGENT_TOKEN")
	parseTimeout := parseFlags.Duration("timeout", 10*time.Second, "Timeout for live hub successor/apply operations")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig := rebuildConfig{
		appPath:   firstNonEmpty(*parseApp, *parseMain),
		rootPath:  *parseRoot,
		output:    firstNonEmpty(*parseOut, *parseOutput),
		profile:   *parseProfile,
		session:   *parseSession,
		hub:       *parseHub,
		token:     *parseToken,
		timeoutMs: int(parseTimeout.Milliseconds()),
		json:      *parseJSON,
	}
	parseReport, parseErr := executeRebuild(parseConfig)
	if parseConfig.json {
		parseDiagnostics := []agenticDiagnostic(nil)
		if parseErr != nil {
			parseDiagnostics = append(parseDiagnostics, buildAgenticCommandError("GWC-REBUILD", parseErr))
		}
		if parseWriteErr := writeAgenticEnvelope("rebuild", parseErr == nil, parseReport, parseDiagnostics, parseErr); parseWriteErr != nil {
			return parseWriteErr
		}
	}
	if parseErr != nil {
		return parseErr
	}
	if !parseConfig.json {
		printAgenticHumanSummary("GWC rebuild", true, []string{
			"old session: " + firstNonEmpty(parseReport.OldSession, "<hub default>"),
			"successor: " + firstNonEmpty(parseReport.SuccessorSession, "<unknown>"),
			"build id: " + firstNonEmpty(parseReport.BuildID, "<none>"),
		})
	}
	return nil
}

func executeRebuild(parseConfig rebuildConfig) (rebuildReport, error) {
	parseHub := strings.TrimRight(strings.TrimSpace(parseConfig.hub), "/")
	parseReport := rebuildReport{
		Hub:      parseHub,
		Phase:    "init",
		APIPaths: []string{"/__gwc-agent/sessions", "/__gwc-agent/command", "/__gwc-agent/reload", "/__gwc-agent/successor"},
	}
	if parseHub == "" {
		return parseReport, errors.New("rebuild requires -hub or GWC_AGENT_HUB_URL for session-aware reload")
	}
	if strings.TrimSpace(parseConfig.token) == "" {
		return parseReport, errors.New("rebuild requires -token or GWC_AGENT_TOKEN")
	}

	parseSession := strings.TrimSpace(parseConfig.session)
	if parseSession == "" {
		parseReport.Phase = "sessions"
		parseSessions, parseErr := callLiveBridgeHub(liveBridgeCall{command: "sessions", hub: parseHub, token: parseConfig.token})
		if parseErr != nil {
			return parseReport, fmt.Errorf("select live session: %w", parseErr)
		}
		parseSelected := selectLiveBridgeSessionID(parseSessions.Result)
		if parseSelected == "" {
			return parseReport, errors.New("no live agent session connected; launch with ?gwc-dev=agent before calling rebuild")
		}
		parseSession = parseSelected
	}
	parseReport.OldSession = parseSession

	parseReport.Phase = "snapshot"
	parseSnapshot, parseErr := postLiveBridgeCommand(parseHub, parseConfig.token, parseSession, "bridge.snapshot", map[string]any{"includeState": true}, parseConfig.timeoutMs)
	if parseErr != nil {
		return parseReport, fmt.Errorf("capture hotreload snapshot: %w", parseErr)
	}
	parseReport.SnapshotCaptured = true

	parseReport.Phase = "build"
	parseBuildConfig, parseErr := resolveBuildConfig(buildConfig{
		appPath:    parseConfig.appPath,
		rootPath:   parseConfig.rootPath,
		outputPath: parseConfig.output,
		profile:    firstNonEmpty(parseConfig.profile, "development"),
		json:       false,
	})
	if parseErr != nil {
		return parseReport, parseErr
	}
	parseBuild, parseErr := buildExecuteBuild(parseBuildConfig)
	if parseErr != nil {
		parseReport.CompilerOutput = parseErr.Error()
		return parseReport, parseErr
	}
	parseReport.Build = &parseBuild
	parseReport.BuildID = firstNonEmpty(parseBuild.SHA256, parseBuild.OutputPath)

	parseReport.Phase = "reload"
	parseReloadBody, parseErr := json.Marshal(map[string]any{
		"session":  parseSession,
		"buildId":  parseReport.BuildID,
		"artifact": parseBuild.OutputPath,
	})
	if parseErr != nil {
		return parseReport, parseErr
	}
	var parseReloadResult any
	parseReloadURL := parseHub + "/__gwc-agent/reload?token=" + url.QueryEscape(parseConfig.token)
	if parseErr := liveBridgeHTTPJSON(http.MethodPost, parseReloadURL, parseReloadBody, &parseReloadResult); parseErr != nil {
		return parseReport, fmt.Errorf("trigger live reload: %w", parseErr)
	}

	parseReport.Phase = "successor"
	parseSuccessor, parseErr := waitForLiveBridgeSuccessor(parseHub, parseConfig.token, parseSession, parseReport.BuildID, parseConfig.timeoutMs)
	if parseErr != nil {
		return parseReport, parseErr
	}
	parseReport.SuccessorSession = parseSuccessor

	parseReport.Phase = "restore"
	_, parseErr = postLiveBridgeCommand(parseHub, parseConfig.token, parseSuccessor, "bridge.apply-snapshot", map[string]any{"snapshot": parseSnapshot}, parseConfig.timeoutMs)
	if parseErr != nil {
		parseReport.RestoredStateNote = parseErr.Error()
		return parseReport, fmt.Errorf("apply hotreload snapshot to successor session: %w", parseErr)
	}
	parseReport.StateRestored = true
	parseReport.OK = true
	parseReport.Phase = "done"
	return parseReport, nil
}

func postLiveBridgeCommand(parseHub string, parseToken string, parseSession string, parseName string, parsePayload any, parseTimeoutMs int) (any, error) {
	parseBody, parseErr := json.Marshal(map[string]any{
		"session":   strings.TrimSpace(parseSession),
		"name":      strings.TrimSpace(parseName),
		"payload":   parsePayload,
		"timeoutMs": parseTimeoutMs,
	})
	if parseErr != nil {
		return nil, parseErr
	}
	parseURL := strings.TrimRight(strings.TrimSpace(parseHub), "/") + "/__gwc-agent/command?token=" + url.QueryEscape(parseToken)
	var parseResult any
	if parseErr := liveBridgeHTTPJSON(http.MethodPost, parseURL, parseBody, &parseResult); parseErr != nil {
		return nil, parseErr
	}
	return parseResult, nil
}

func waitForLiveBridgeSuccessor(parseHub string, parseToken string, parseSession string, parseBuildID string, parseTimeoutMs int) (string, error) {
	parseValues := url.Values{}
	parseValues.Set("token", parseToken)
	parseValues.Set("session", parseSession)
	parseValues.Set("buildId", parseBuildID)
	parseValues.Set("timeoutMs", fmt.Sprintf("%d", parseTimeoutMs))
	parseURL := strings.TrimRight(strings.TrimSpace(parseHub), "/") + "/__gwc-agent/successor?" + parseValues.Encode()
	var parseResult any
	if parseErr := liveBridgeHTTPJSON(http.MethodGet, parseURL, nil, &parseResult); parseErr != nil {
		return "", fmt.Errorf("wait for successor session: %w", parseErr)
	}
	parseSessionID := extractStringField(parseResult, "session", "sessionId", "successor", "successorSession", "id")
	if parseSessionID == "" {
		return "", fmt.Errorf("successor response did not include a session id")
	}
	return parseSessionID, nil
}

func selectLiveBridgeSessionID(parseValue any) string {
	parseSessions := extractArrayField(parseValue, "sessions")
	if len(parseSessions) == 0 {
		return extractStringField(parseValue, "session", "sessionId", "id")
	}
	for parseIndex := len(parseSessions) - 1; parseIndex >= 0; parseIndex-- {
		parseSession, parseOK := parseSessions[parseIndex].(map[string]any)
		if !parseOK {
			continue
		}
		parseState := strings.ToLower(strings.TrimSpace(extractStringField(parseSession, "state", "status")))
		if parseState == "" || parseState == "active" || parseState == "connected" {
			if parseID := extractStringField(parseSession, "id", "session", "sessionId"); parseID != "" {
				return parseID
			}
		}
	}
	for parseIndex := len(parseSessions) - 1; parseIndex >= 0; parseIndex-- {
		if parseSession, parseOK := parseSessions[parseIndex].(map[string]any); parseOK {
			if parseID := extractStringField(parseSession, "id", "session", "sessionId"); parseID != "" {
				return parseID
			}
		}
	}
	return ""
}

func extractArrayField(parseValue any, parseKeys ...string) []any {
	parseMap, parseOK := parseValue.(map[string]any)
	if !parseOK {
		return nil
	}
	for _, parseKey := range parseKeys {
		if parseArray, parseOK := parseMap[parseKey].([]any); parseOK {
			return parseArray
		}
	}
	return nil
}

func extractStringField(parseValue any, parseKeys ...string) string {
	parseMap, parseOK := parseValue.(map[string]any)
	if !parseOK {
		return ""
	}
	for _, parseKey := range parseKeys {
		if parseString, parseOK := parseMap[parseKey].(string); parseOK && strings.TrimSpace(parseString) != "" {
			return strings.TrimSpace(parseString)
		}
	}
	return ""
}
