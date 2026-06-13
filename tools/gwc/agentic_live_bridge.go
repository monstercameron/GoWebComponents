package main

import (
	"bytes"
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

// runLiveBridgeCommand routes live-session agent bridge CLI/MCP commands.
var runLiveBridgeCommand = func(parseL launcher, parseCommand string, parseArgs []string) error {
	return parseL.runLiveBridge(parseCommand, parseArgs)
}

type liveBridgeReport struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Session string `json:"session,omitempty"`
	Hub     string `json:"hub,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Fix     string `json:"fix,omitempty"`
	Result  any    `json:"result,omitempty"`
}

func (parseL launcher) runLiveBridge(parseCommand string, parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"session", "hub", "token", "payload", "timeout", "lease-holder", "holder", "action", "max"}, []string{"json", "steal", "clear"})
	parseFlags := flag.NewFlagSet(parseCommand, flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseSession := parseFlags.String("session", "", "Optional live agent session id; defaults to the most recently active session")
	parseHub := parseFlags.String("hub", firstNonEmpty(os.Getenv("GWC_AGENT_HUB_URL"), ""), "Agent hub base URL, for example http://127.0.0.1:8090")
	parseToken := parseFlags.String("token", firstNonEmpty(os.Getenv("GWC_AGENT_TOKEN"), ""), "Agent hub token; defaults to GWC_AGENT_TOKEN")
	parsePayload := parseFlags.String("payload", "{}", "Bridge command payload JSON")
	parseTimeout := parseFlags.Duration("timeout", 5*time.Second, "Bridge command timeout")
	parseLeaseHolder := parseFlags.String("lease-holder", firstNonEmpty(os.Getenv("GWC_AGENT_LEASE_HOLDER"), ""), "Write lease holder for mutating bridge commands")
	parseHolder := parseFlags.String("holder", firstNonEmpty(os.Getenv("GWC_AGENT_LEASE_HOLDER"), ""), "Lease holder for the lease command")
	parseAction := parseFlags.String("action", "acquire", "Lease action: acquire, release, or steal")
	parseMax := parseFlags.Int("max", 0, "Maximum logs or recording records to return")
	parseSteal := parseFlags.Bool("steal", false, "Allow the lease command to steal an existing write lease")
	parseClear := parseFlags.Bool("clear", false, "Clear the session recording after reading it")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	if strings.TrimSpace(*parseHub) != "" {
		parseReport, parseErr := callLiveBridgeHub(liveBridgeCall{
			command:     parseCommand,
			hub:         *parseHub,
			token:       *parseToken,
			session:     *parseSession,
			payload:     *parsePayload,
			timeoutMs:   int(parseTimeout.Milliseconds()),
			leaseHolder: *parseLeaseHolder,
			holder:      *parseHolder,
			action:      *parseAction,
			max:         *parseMax,
			steal:       *parseSteal,
			clear:       *parseClear,
		})
		if *parseJSON {
			parseDiagnostics := []agenticDiagnostic(nil)
			if parseErr != nil {
				parseDiagnostics = append(parseDiagnostics, buildAgenticCommandError("GWC-AGENTBRIDGE-HUB", parseErr))
			}
			if parseWriteErr := writeAgenticEnvelope(parseCommand, parseErr == nil, parseReport, parseDiagnostics, parseErr); parseWriteErr != nil {
				return parseWriteErr
			}
		}
		if parseErr != nil {
			return parseErr
		}
		if !*parseJSON {
			printAgenticHumanSummary("GWC "+parseCommand, true, []string{"hub: " + parseReport.Hub})
		}
		return nil
	}

	parseErr := fmt.Errorf("no live agent session connected; launch the app through `gwc dev`, open it with `?gwc-dev=agent`, and ensure the wasm build includes the `gwcagent` tag")
	parseReport := liveBridgeReport{
		OK:      false,
		Command: strings.TrimSpace(parseCommand),
		Session: strings.TrimSpace(*parseSession),
		Reason:  "no-session-connected",
		Fix:     "launch with ?gwc-dev=agent and connect a live wasm session before calling bridge tools",
	}
	if *parseJSON {
		parseDiagnostic := buildAgenticCommandError("GWC-AGENTBRIDGE-NO-SESSION", parseErr)
		if parseWriteErr := writeAgenticEnvelope(parseCommand, false, parseReport, []agenticDiagnostic{parseDiagnostic}, parseErr); parseWriteErr != nil {
			return parseWriteErr
		}
		return parseErr
	}
	printAgenticHumanSummary("GWC "+parseCommand, false, []string{
		"reason: " + parseReport.Reason,
		"fix: " + parseReport.Fix,
	})
	return parseErr
}

type liveBridgeCall struct {
	command     string
	hub         string
	token       string
	session     string
	payload     string
	timeoutMs   int
	leaseHolder string
	holder      string
	action      string
	max         int
	steal       bool
	clear       bool
}

func callLiveBridgeHub(parseCall liveBridgeCall) (liveBridgeReport, error) {
	parseHub := strings.TrimRight(strings.TrimSpace(parseCall.hub), "/")
	parseReport := liveBridgeReport{OK: false, Command: parseCall.command, Session: strings.TrimSpace(parseCall.session), Hub: parseHub}
	if parseHub == "" {
		return parseReport, fmt.Errorf("agent hub URL is required")
	}
	if strings.TrimSpace(parseCall.token) == "" {
		return parseReport, fmt.Errorf("agent hub token is required")
	}
	switch parseCall.command {
	case "sessions":
		parseURL := parseHub + "/__gwc-agent/sessions?token=" + url.QueryEscape(parseCall.token)
		var parseResult any
		if parseErr := liveBridgeHTTPJSON(http.MethodGet, parseURL, nil, &parseResult); parseErr != nil {
			return parseReport, parseErr
		}
		parseReport.OK = true
		parseReport.Result = parseResult
		return parseReport, nil
	case "logs":
		parseURL := liveBridgeEndpointURL(parseHub, "/__gwc-agent/logs", parseCall)
		var parseResult any
		if parseErr := liveBridgeHTTPJSON(http.MethodGet, parseURL, nil, &parseResult); parseErr != nil {
			return parseReport, parseErr
		}
		parseReport.OK = true
		parseReport.Result = parseResult
		return parseReport, nil
	case "crash-report":
		parseURL := liveBridgeEndpointURL(parseHub, "/__gwc-agent/crash-report", parseCall)
		var parseResult any
		if parseErr := liveBridgeHTTPJSON(http.MethodGet, parseURL, nil, &parseResult); parseErr != nil {
			return parseReport, parseErr
		}
		parseReport.OK = true
		parseReport.Result = parseResult
		return parseReport, nil
	case "recording":
		parseURL := liveBridgeEndpointURL(parseHub, "/__gwc-agent/recording", parseCall)
		parseMethod := http.MethodGet
		if parseCall.clear {
			parseMethod = http.MethodDelete
		}
		var parseResult any
		if parseErr := liveBridgeHTTPJSON(parseMethod, parseURL, nil, &parseResult); parseErr != nil {
			return parseReport, parseErr
		}
		parseReport.OK = true
		parseReport.Result = parseResult
		return parseReport, nil
	case "lease":
		parseBody, parseErr := json.Marshal(map[string]any{
			"session": strings.TrimSpace(parseCall.session),
			"action":  strings.TrimSpace(parseCall.action),
			"holder":  strings.TrimSpace(parseCall.holder),
			"steal":   parseCall.steal,
		})
		if parseErr != nil {
			return parseReport, parseErr
		}
		parseURL := parseHub + "/__gwc-agent/lease?token=" + url.QueryEscape(parseCall.token)
		var parseResult any
		if parseErr := liveBridgeHTTPJSON(http.MethodPost, parseURL, parseBody, &parseResult); parseErr != nil {
			return parseReport, parseErr
		}
		parseReport.OK = true
		parseReport.Result = parseResult
		return parseReport, nil
	}
	parseBridgeName, parseOK := liveBridgeCommandName(parseCall.command)
	if !parseOK {
		return parseReport, fmt.Errorf("unsupported live bridge command %q", parseCall.command)
	}
	var parsePayload json.RawMessage
	if strings.TrimSpace(parseCall.payload) != "" {
		if parseErr := json.Unmarshal([]byte(parseCall.payload), &parsePayload); parseErr != nil {
			return parseReport, fmt.Errorf("payload must be JSON: %w", parseErr)
		}
	}
	parseBody, parseErr := json.Marshal(map[string]any{
		"session":     strings.TrimSpace(parseCall.session),
		"name":        parseBridgeName,
		"payload":     parsePayload,
		"timeoutMs":   parseCall.timeoutMs,
		"leaseHolder": strings.TrimSpace(parseCall.leaseHolder),
	})
	if parseErr != nil {
		return parseReport, parseErr
	}
	parseURL := parseHub + "/__gwc-agent/command?token=" + url.QueryEscape(parseCall.token)
	var parseResult any
	if parseErr := liveBridgeHTTPJSON(http.MethodPost, parseURL, parseBody, &parseResult); parseErr != nil {
		return parseReport, parseErr
	}
	parseReport.OK = true
	parseReport.Result = parseResult
	return parseReport, nil
}

func liveBridgeEndpointURL(parseHub string, parsePath string, parseCall liveBridgeCall) string {
	parseValues := url.Values{}
	parseValues.Set("token", parseCall.token)
	if strings.TrimSpace(parseCall.session) != "" {
		parseValues.Set("session", strings.TrimSpace(parseCall.session))
	}
	if parseCall.max > 0 {
		parseValues.Set("max", fmt.Sprintf("%d", parseCall.max))
	}
	return parseHub + parsePath + "?" + parseValues.Encode()
}

func liveBridgeCommandName(parseCommand string) (string, bool) {
	switch parseCommand {
	case "snapshot":
		return "bridge.snapshot", true
	case "query":
		return "bridge.query", true
	case "set-atom":
		return "bridge.set-atom", true
	case "set-state":
		return "bridge.set-state", true
	case "mount":
		return "bridge.mount", true
	case "unmount":
		return "bridge.unmount", true
	case "delete-atom":
		return "bridge.delete-atom", true
	case "emit":
		return "bridge.emit", true
	case "publish":
		return "bridge.publish", true
	case "navigate":
		return "bridge.navigate", true
	case "describe":
		return "bridge.describe", true
	case "wait-for":
		return "bridge.wait-for", true
	case "audit":
		return "bridge.audit", true
	case "undo":
		return "bridge.undo", true
	case "replay":
		return "bridge.replay", true
	case "render-tree":
		return "bridge.render-tree", true
	default:
		return "", false
	}
}

func liveBridgeHTTPJSON(parseMethod string, parseURL string, parseBody []byte, parseOut any) error {
	parseClient := &http.Client{Timeout: 10 * time.Second}
	var parseReader *bytes.Reader
	if parseBody == nil {
		parseReader = bytes.NewReader(nil)
	} else {
		parseReader = bytes.NewReader(parseBody)
	}
	parseReq, parseErr := http.NewRequest(parseMethod, parseURL, parseReader)
	if parseErr != nil {
		return parseErr
	}
	if parseBody != nil {
		parseReq.Header.Set("Content-Type", "application/json")
	}
	parseResp, parseErr := parseClient.Do(parseReq)
	if parseErr != nil {
		return parseErr
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode < 200 || parseResp.StatusCode >= 300 {
		return fmt.Errorf("agent hub returned %s", parseResp.Status)
	}
	return json.NewDecoder(parseResp.Body).Decode(parseOut)
}
