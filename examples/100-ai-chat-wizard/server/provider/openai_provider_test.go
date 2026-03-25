package provider

import (
	"testing"

	"github.com/openai/openai-go/shared"
)

func TestOpenAIProviderNonNetworkHelpers(parseT *testing.T) {
	parseProvider := ParseNewOpenAIProvider("test-key", parseTestOpenAICatalog())
	if !parseProvider.ParseAvailable() {
		parseT.Fatal("expected provider with test key to be available")
	}
	if parseProvider.ParseID() != "openai" {
		parseT.Fatalf("unexpected provider ID: %q", parseProvider.ParseID())
	}
	if parseProvider.ParseDefaultModel() != "gpt-5.4-mini" {
		parseT.Fatalf("unexpected default model: %q", parseProvider.ParseDefaultModel())
	}
	if !parseProvider.ParseSupportsModel("gpt-5.4-mini") || parseProvider.ParseSupportsModel("claude-sonnet-4-5") {
		parseT.Fatal("unexpected SupportsModel behavior for OpenAI provider")
	}
	parseCapabilities := parseProvider.ParseCapabilities("gpt-5.4-mini")
	if !parseCapabilities.SupportsThinking || !parseCapabilities.SupportsSpeech {
		parseT.Fatalf("unexpected capabilities: %+v", parseCapabilities)
	}
	if len(parseProvider.ParseModelOptions()) != 3 {
		parseT.Fatalf("expected three model options, got %d", len(parseProvider.ParseModelOptions()))
	}
	if parseGot := parseOpenAIReasoningEffort("HIGH"); parseGot != shared.ReasoningEffortHigh {
		parseT.Fatalf("unexpected reasoning effort normalization: %v", parseGot)
	}
	if parseMetadata := parseProvider.parseMustModelMetadata("gpt-5.4-mini"); parseMetadata.ParseID != "gpt-5.4-mini" || parseMetadata.ProviderID != "openai" {
		parseT.Fatalf("expected known-model metadata lookup path, got %+v", parseMetadata)
	}
	if ParseNewOpenAIProvider("", parseTestOpenAICatalog()).ParseAvailable() {
		parseT.Fatal("expected provider without key to be unavailable")
	}
}
