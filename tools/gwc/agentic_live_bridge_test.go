package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallLiveBridgeHubSessionsAndCommand(parseT *testing.T) {
	parseSeen := map[string]bool{}
	parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		if parseR.URL.Query().Get("token") != "tok" {
			http.Error(parseW, "bad token", http.StatusForbidden)
			return
		}
		switch parseR.URL.Path {
		case "/__gwc-agent/sessions":
			parseSeen["sessions"] = true
			_, _ = parseW.Write([]byte(`{"sessions":[{"id":"sess-1","state":"active"}]}`))
		case "/__gwc-agent/command":
			parseSeen["command"] = true
			var parseBody map[string]any
			if parseErr := json.NewDecoder(parseR.Body).Decode(&parseBody); parseErr != nil {
				parseT.Errorf("decode command: %v", parseErr)
				http.Error(parseW, "bad json", http.StatusBadRequest)
				return
			}
			if parseBody["name"] != "bridge.snapshot" {
				parseT.Errorf("command name = %#v", parseBody["name"])
			}
			if parseBody["session"] != "sess-1" {
				parseT.Errorf("session = %#v", parseBody["session"])
			}
			if parseBody["leaseHolder"] != "" {
				parseT.Errorf("unexpected read leaseHolder = %#v", parseBody["leaseHolder"])
			}
			_, _ = parseW.Write([]byte(`{"ack":{"ok":true,"stateVersion":9}}`))
		case "/__gwc-agent/logs":
			parseSeen["logs"] = true
			if parseR.URL.Query().Get("max") != "10" {
				parseT.Errorf("logs max = %q", parseR.URL.Query().Get("max"))
			}
			_, _ = parseW.Write([]byte(`{"session":"sess-1","logs":[]}`))
		case "/__gwc-agent/crash-report":
			parseSeen["crash"] = true
			_, _ = parseW.Write([]byte(`{"session":"sess-1","report":{"sessionId":"sess-1"}}`))
		case "/__gwc-agent/recording":
			parseSeen["recording"] = true
			if parseR.Method != http.MethodDelete {
				parseT.Errorf("recording method = %s", parseR.Method)
			}
			_, _ = parseW.Write([]byte(`{"session":"sess-1","records":[]}`))
		case "/__gwc-agent/lease":
			parseSeen["lease"] = true
			var parseBody map[string]any
			if parseErr := json.NewDecoder(parseR.Body).Decode(&parseBody); parseErr != nil {
				parseT.Errorf("decode lease: %v", parseErr)
				http.Error(parseW, "bad json", http.StatusBadRequest)
				return
			}
			if parseBody["holder"] != "agent-a" || parseBody["action"] != "steal" || parseBody["steal"] != true {
				parseT.Errorf("lease body = %#v", parseBody)
			}
			_, _ = parseW.Write([]byte(`{"session":"sess-1","lease":{"holder":"agent-a"}}`))
		default:
			http.NotFound(parseW, parseR)
		}
	}))
	defer parseServer.Close()

	parseSessions, parseErr := callLiveBridgeHub(liveBridgeCall{command: "sessions", hub: parseServer.URL, token: "tok"})
	if parseErr != nil {
		parseT.Fatalf("sessions call: %v", parseErr)
	}
	if !parseSessions.OK || !strings.Contains(parseSessions.Hub, "http://") {
		parseT.Fatalf("sessions report = %#v", parseSessions)
	}

	parseCommand, parseErr := callLiveBridgeHub(liveBridgeCall{command: "snapshot", hub: parseServer.URL, token: "tok", session: "sess-1", payload: `{"maxDepth":2}`, timeoutMs: 1000})
	if parseErr != nil {
		parseT.Fatalf("command call: %v", parseErr)
	}
	if !parseCommand.OK {
		parseT.Fatalf("command report = %#v", parseCommand)
	}

	parseLogs, parseErr := callLiveBridgeHub(liveBridgeCall{command: "logs", hub: parseServer.URL, token: "tok", session: "sess-1", max: 10})
	if parseErr != nil || !parseLogs.OK {
		parseT.Fatalf("logs call report=%#v err=%v", parseLogs, parseErr)
	}
	parseCrash, parseErr := callLiveBridgeHub(liveBridgeCall{command: "crash-report", hub: parseServer.URL, token: "tok", session: "sess-1"})
	if parseErr != nil || !parseCrash.OK {
		parseT.Fatalf("crash call report=%#v err=%v", parseCrash, parseErr)
	}
	parseRecording, parseErr := callLiveBridgeHub(liveBridgeCall{command: "recording", hub: parseServer.URL, token: "tok", session: "sess-1", clear: true})
	if parseErr != nil || !parseRecording.OK {
		parseT.Fatalf("recording call report=%#v err=%v", parseRecording, parseErr)
	}
	parseLease, parseErr := callLiveBridgeHub(liveBridgeCall{command: "lease", hub: parseServer.URL, token: "tok", session: "sess-1", holder: "agent-a", action: "steal", steal: true})
	if parseErr != nil || !parseLease.OK {
		parseT.Fatalf("lease call report=%#v err=%v", parseLease, parseErr)
	}
	for _, parseKey := range []string{"sessions", "command", "logs", "crash", "recording", "lease"} {
		if !parseSeen[parseKey] {
			parseT.Fatalf("expected %s endpoint to be called, saw %#v", parseKey, parseSeen)
		}
	}
}

func TestCallLiveBridgeHubMutatingCommandSendsLeaseHolder(parseT *testing.T) {
	parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		if parseR.URL.Path != "/__gwc-agent/command" {
			http.NotFound(parseW, parseR)
			return
		}
		var parseBody map[string]any
		if parseErr := json.NewDecoder(parseR.Body).Decode(&parseBody); parseErr != nil {
			parseT.Errorf("decode command: %v", parseErr)
			http.Error(parseW, "bad json", http.StatusBadRequest)
			return
		}
		if parseBody["name"] != "bridge.set-state" || parseBody["leaseHolder"] != "agent-a" {
			parseT.Errorf("mutating body = %#v", parseBody)
		}
		_, _ = parseW.Write([]byte(`{"ack":{"ok":true,"stateVersion":10}}`))
	}))
	defer parseServer.Close()

	parseCommand, parseErr := callLiveBridgeHub(liveBridgeCall{command: "set-state", hub: parseServer.URL, token: "tok", session: "sess-1", payload: `{"ref":"root","slot":0,"value":true}`, leaseHolder: "agent-a"})
	if parseErr != nil {
		parseT.Fatalf("set-state call: %v", parseErr)
	}
	if !parseCommand.OK {
		parseT.Fatalf("set-state report = %#v", parseCommand)
	}
}

func TestCallLiveBridgeHubRequiresToken(parseT *testing.T) {
	_, parseErr := callLiveBridgeHub(liveBridgeCall{command: "sessions", hub: "http://127.0.0.1:8090"})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "token") {
		parseT.Fatalf("expected token error, got %v", parseErr)
	}
}

func TestMCPToolCallCanReachLiveBridgeHub(parseT *testing.T) {
	parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		if parseR.URL.Path != "/__gwc-agent/command" {
			http.NotFound(parseW, parseR)
			return
		}
		if parseR.URL.Query().Get("token") != "tok" {
			http.Error(parseW, "bad token", http.StatusForbidden)
			return
		}
		_, _ = parseW.Write([]byte(`{"ack":{"ok":true,"stateVersion":12,"payload":{"ok":true}}}`))
	}))
	defer parseServer.Close()

	parseArguments, parseErr := json.Marshal(map[string]any{
		"name": "gwc_snapshot",
		"arguments": map[string]any{
			"args": []string{"-hub", parseServer.URL, "-token", "tok", "-session", "sess-1", "-payload", "{}"},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("marshal mcp args: %v", parseErr)
	}
	parseResult, parseErr := executeMCPToolCall(launcher{}, parseArguments)
	if parseErr != nil {
		parseT.Fatalf("execute mcp live bridge call: %v", parseErr)
	}
	if parseResult["isError"].(bool) {
		parseT.Fatalf("expected successful MCP bridge call, got %#v", parseResult)
	}
	parseContent := parseResult["content"].([]map[string]interface{})
	if !strings.Contains(parseContent[0]["text"].(string), `"stateVersion": 12`) {
		parseT.Fatalf("expected ack stateVersion in MCP content, got %s", parseContent[0]["text"].(string))
	}
}
