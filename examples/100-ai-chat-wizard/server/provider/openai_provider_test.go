package provider

import (
	"testing"

	"github.com/openai/openai-go/shared"
)

func TestOpenAIProviderNonNetworkHelpers(t *testing.T) {
	provider := NewOpenAIProvider("test-key", testOpenAICatalog())
	if !provider.Available() {
		t.Fatal("expected provider with test key to be available")
	}
	if provider.ID() != "openai" {
		t.Fatalf("unexpected provider ID: %q", provider.ID())
	}
	if provider.DefaultModel() != "gpt-5.4-mini" {
		t.Fatalf("unexpected default model: %q", provider.DefaultModel())
	}
	if !provider.SupportsModel("gpt-5.4-mini") || provider.SupportsModel("claude-sonnet-4-5") {
		t.Fatal("unexpected SupportsModel behavior for OpenAI provider")
	}
	capabilities := provider.Capabilities("gpt-5.4-mini")
	if !capabilities.SupportsThinking || !capabilities.SupportsSpeech {
		t.Fatalf("unexpected capabilities: %+v", capabilities)
	}
	if len(provider.ModelOptions()) != 3 {
		t.Fatalf("expected three model options, got %d", len(provider.ModelOptions()))
	}
	if got := openAIReasoningEffort("HIGH"); got != shared.ReasoningEffortHigh {
		t.Fatalf("unexpected reasoning effort normalization: %v", got)
	}
	if metadata := provider.mustModelMetadata("gpt-5.4-mini"); metadata.ID != "gpt-5.4-mini" || metadata.ProviderID != "openai" {
		t.Fatalf("expected known-model metadata lookup path, got %+v", metadata)
	}
	if NewOpenAIProvider("", testOpenAICatalog()).Available() {
		t.Fatal("expected provider without key to be unavailable")
	}
}
