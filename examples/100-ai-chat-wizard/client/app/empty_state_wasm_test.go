//go:build js && wasm

package app

import "testing"

// TestParseBuildStarterPrompts verifies first-run starter prompts stay populated and usable.
func TestParseBuildStarterPrompts(parseT *testing.T) {
	parsePrompts := parseBuildStarterPrompts()
	if len(parsePrompts) < 3 {
		parseT.Fatalf("parseBuildStarterPrompts() returned %d prompts, want at least 3", len(parsePrompts))
	}
	for parseIndex, parsePrompt := range parsePrompts {
		if parsePrompt.parseLabel == "" {
			parseT.Fatalf("parseBuildStarterPrompts()[%d] label is empty", parseIndex)
		}
		if parsePrompt.parseText == "" {
			parseT.Fatalf("parseBuildStarterPrompts()[%d] text is empty", parseIndex)
		}
	}
}
