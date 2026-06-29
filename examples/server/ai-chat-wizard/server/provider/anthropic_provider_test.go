package provider

import (
	"errors"
	"testing"
)

func TestAnthropicProviderNonNetworkHelpers(parseT *testing.T) {
	parseProvider := ParseNewAnthropicProvider("test-key", parseTestAnthropicCatalog())
	if !parseProvider.ParseAvailable() {
		parseT.Fatal("expected anthropic provider with test key to be available")
	}
	if parseProvider.ParseID() != "anthropic" {
		parseT.Fatalf("unexpected provider ID: %q", parseProvider.ParseID())
	}
	if parseProvider.ParseDefaultModel() != "claude-sonnet-4-5" {
		parseT.Fatalf("unexpected default model: %q", parseProvider.ParseDefaultModel())
	}
	if !parseProvider.ParseSupportsModel("claude-sonnet-4-5") || parseProvider.ParseSupportsModel("gpt-5.4-mini") {
		parseT.Fatal("unexpected SupportsModel behavior for Anthropic provider")
	}
	parseCapabilities := parseProvider.ParseCapabilities("claude-sonnet-4-5")
	if !parseCapabilities.SupportsThinking || parseCapabilities.SupportsSpeech {
		parseT.Fatalf("unexpected capabilities: %+v", parseCapabilities)
	}
	if len(parseProvider.ParseModelOptions()) != 2 {
		parseT.Fatalf("expected two Anthropic model options, got %d", len(parseProvider.ParseModelOptions()))
	}
	if parseGot := parseAnthropicThinkingBudget("HIGH"); parseGot != 4096 {
		parseT.Fatalf("unexpected high thinking budget: %d", parseGot)
	}
	if parseGot2 := parseAnthropicThinkingBudget("low"); parseGot2 != 1024 {
		parseT.Fatalf("unexpected low thinking budget: %d", parseGot2)
	}
	if parseMetadata := parseProvider.parseMustModelMetadata("claude-sonnet-4-5"); parseMetadata.ID != "claude-sonnet-4-5" || parseMetadata.ProviderID != "anthropic" {
		parseT.Fatalf("expected known-model metadata lookup path, got %+v", parseMetadata)
	}
	if !parseAnthropicThinkingUnsupported(errors.New("thinking unsupported invalid_request_error")) {
		parseT.Fatal("expected unsupported thinking error to be detected")
	}
	if parseAnthropicThinkingUnsupported(nil) {
		parseT.Fatal("expected nil error to return false")
	}
	if ParseNewAnthropicProvider("", parseTestAnthropicCatalog()).ParseAvailable() {
		parseT.Fatal("expected provider without key to be unavailable")
	}
}
