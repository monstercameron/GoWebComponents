package main

import (
	"encoding/json"
	"testing"
)

// TestMCPServerInitializeAndToolsList is the FB3 MCP-server integration smoke: it drives the
// gwc MCP server's JSON-RPC handler through `initialize` and `tools/list` and asserts a
// well-formed response advertising the gwc commands as tools — so an MCP client can discover
// and (per tools/call) invoke them.
func TestMCPServerInitializeAndToolsList(parseT *testing.T) {
	parseLauncher := launcher{}

	// initialize handshake.
	parseInit := handleMCPRequest(parseLauncher, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	if parseInit == nil || parseInit.JSONRPC != "2.0" {
		parseT.Fatalf("initialize should return a 2.0 response, got %+v", parseInit)
	}
	parseResult, parseOk := parseInit.Result.(map[string]interface{})
	if !parseOk || parseResult["protocolVersion"] == nil {
		parseT.Fatalf("initialize result missing protocolVersion: %+v", parseInit.Result)
	}

	// tools/list must advertise tools.
	parseList := handleMCPRequest(parseLauncher, []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`))
	if parseList == nil {
		parseT.Fatal("tools/list returned nil")
	}
	parseListResult, _ := parseList.Result.(map[string]interface{})
	parseTools, parseOk := parseListResult["tools"].([]mcpTool)
	if !parseOk || len(parseTools) == 0 {
		parseT.Fatalf("tools/list should advertise the gwc commands, got %#v", parseListResult["tools"])
	}

	// The advertised tools must be JSON-serializable (the wire form an MCP client receives).
	if _, parseErr := json.Marshal(parseList.Result); parseErr != nil {
		parseT.Fatalf("tools/list result must marshal to JSON: %v", parseErr)
	}

	// An unknown method is a proper JSON-RPC method-not-found error.
	parseUnknown := handleMCPRequest(parseLauncher, []byte(`{"jsonrpc":"2.0","id":3,"method":"bogus/method"}`))
	if parseUnknown == nil || parseUnknown.Error == nil {
		parseT.Fatalf("an unknown method should return a JSON-RPC error, got %+v", parseUnknown)
	}
}
