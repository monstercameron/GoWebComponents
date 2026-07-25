package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/agentui"
)

// TestAgentUICheckAcceptsValidAndRejectsInvalid proves `gwc agentui check` validates an
// agent-UI schema file against the allow-list — accepting allow-listed components and
// rejecting a non-allow-listed type, so bad agent output is caught in CI before render.
func TestAgentUICheckAcceptsValidAndRejectsInvalid(parseT *testing.T) {
	parseDir := parseT.TempDir()

	parseValid := filepath.Join(parseDir, "ok.json")
	if parseErr := os.WriteFile(parseValid, []byte(`{"type":"stack","children":[{"type":"text","text":"hi"}]}`), 0644); parseErr != nil {
		parseT.Fatalf("write valid: %v", parseErr)
	}
	if parseErr := (launcher{}).runAgentUI([]string{"check", parseValid}); parseErr != nil {
		parseT.Fatalf("expected the allow-listed schema to pass, got %v", parseErr)
	}

	parseBad := filepath.Join(parseDir, "bad.json")
	if parseErr := os.WriteFile(parseBad, []byte(`{"type":"script","props":{"src":"evil.js"}}`), 0644); parseErr != nil {
		parseT.Fatalf("write bad: %v", parseErr)
	}
	parseErr := (launcher{}).runAgentUI([]string{"check", parseBad})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "allow-list") {
		parseT.Fatalf("expected a non-allow-listed type to be rejected, got %v", parseErr)
	}
}

// TestAgentUICheckRequiresFile proves a missing path is reported.
func TestAgentUICheckRequiresFile(parseT *testing.T) {
	if parseErr := (launcher{}).runAgentUI([]string{"check"}); parseErr == nil {
		parseT.Fatal("expected an error when no file is given")
	}
}

// TestAgentUICatalogEmitsJSON proves `gwc agentui catalog` emits the allow-list (the data an
// MCP tool serves an agent).
func TestAgentUICatalogEmitsJSON(parseT *testing.T) {
	if parseErr := printAgentUICatalog(); parseErr != nil {
		parseT.Fatalf("catalog should emit JSON, got %v", parseErr)
	}
	// And the registry catalog itself is non-empty + has the known components.
	parseNames := map[string]bool{}
	for _, parseInfo := range agentui.DefaultRegistry().Catalog() {
		parseNames[parseInfo.Name] = true
	}
	for _, parseWant := range []string{"stack", "heading", "text"} {
		if !parseNames[parseWant] {
			parseT.Fatalf("catalog missing %q: %v", parseWant, parseNames)
		}
	}
}
