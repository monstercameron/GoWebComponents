package main

import (
	"encoding/json"

	"github.com/monstercameron/GoWebComponents/agentui"
)

// agentUICatalogToolName is the MCP tool that exposes the agentui component allow-list, so an
// agent can query exactly which components and props it may emit BEFORE generating a UI tree —
// turning the allow-list from after-the-fact rejection into up-front guidance (FC3).
const agentUICatalogToolName = "gwc_agentui_catalog"

// agentUICatalogMCPTool describes the read-only catalog tool for the MCP manifest.
func agentUICatalogMCPTool() mcpTool {
	return mcpTool{
		Name:        agentUICatalogToolName,
		Description: "List the agentui allow-listed components and their permitted prop keys, the contract an agent must satisfy before emitting an agent-driven UI tree.",
		InputSchema: map[string]interface{}{
			"type":                 "object",
			"additionalProperties": false,
			"properties":           map[string]interface{}{},
		},
		Annotations: map[string]interface{}{
			"readOnlyHint":    true,
			"destructiveHint": false,
			"title":           "agentui catalog",
		},
	}
}

// agentUICatalogMCPResult returns the MCP tool-call result carrying the catalog as JSON text.
func agentUICatalogMCPResult() (map[string]interface{}, error) {
	parseCatalog := agentui.DefaultRegistry().Catalog()
	parseJSON, parseErr := json.MarshalIndent(map[string]interface{}{"components": parseCatalog}, "", "  ")
	if parseErr != nil {
		return nil, parseErr
	}
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": string(parseJSON)},
		},
		"isError": false,
	}, nil
}
