package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestMCPManifestIncludesAgentUICatalog proves the agentui catalog is exposed as a first-class,
// read-only MCP tool in tools/list (FC3).
func TestMCPManifestIncludesAgentUICatalog(parseT *testing.T) {
	parseManifest := buildMCPManifest()
	parseFound := false
	for _, parseTool := range parseManifest.Tools {
		if parseTool.Name == agentUICatalogToolName {
			parseFound = true
			if parseTool.Annotations["readOnlyHint"] != true {
				parseT.Fatalf("agentui catalog tool should be read-only, got %#v", parseTool.Annotations)
			}
		}
	}
	if !parseFound {
		parseT.Fatalf("expected %s in the MCP manifest", agentUICatalogToolName)
	}
}

// TestMCPToolCallReturnsAgentUICatalog proves invoking the tool returns the allow-list as JSON
// content an agent can read before emitting a UI tree.
func TestMCPToolCallReturnsAgentUICatalog(parseT *testing.T) {
	parseResult, parseErr := agentUICatalogMCPResult()
	if parseErr != nil {
		parseT.Fatalf("agentUICatalogMCPResult: %v", parseErr)
	}
	if parseResult["isError"] == true {
		parseT.Fatal("catalog result should not be an error")
	}
	parseContent, parseOk := parseResult["content"].([]map[string]interface{})
	if !parseOk || len(parseContent) == 0 {
		parseT.Fatalf("expected text content, got %#v", parseResult["content"])
	}
	parseText, _ := parseContent[0]["text"].(string)
	if !strings.Contains(parseText, "components") {
		parseT.Fatalf("catalog JSON should carry a components array, got %q", parseText)
	}
	// The payload must be valid JSON with a components array.
	var parsePayload struct {
		Components []struct {
			Name         string   `json:"name"`
			AllowedProps []string `json:"allowedProps"`
		} `json:"components"`
	}
	if parseErr := json.Unmarshal([]byte(parseText), &parsePayload); parseErr != nil {
		parseT.Fatalf("catalog content must be valid JSON: %v", parseErr)
	}
	if len(parsePayload.Components) == 0 {
		parseT.Fatal("default registry should expose at least one component")
	}
}

// TestMCPHandleRequestRoutesCatalogToolCall proves the JSON-RPC tools/call path dispatches the
// catalog tool end-to-end.
func TestMCPHandleRequestRoutesCatalogToolCall(parseT *testing.T) {
	parsePayload := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + agentUICatalogToolName + `","arguments":{}}}`
	parseResponse := handleMCPRequest(launcher{}, []byte(parsePayload))
	if parseResponse == nil || parseResponse.Error != nil {
		parseT.Fatalf("expected a successful catalog tool/call, got %#v", parseResponse)
	}
	parseResult, parseOk := parseResponse.Result.(map[string]interface{})
	if !parseOk || parseResult["isError"] == true {
		parseT.Fatalf("expected non-error catalog result, got %#v", parseResponse.Result)
	}
}
