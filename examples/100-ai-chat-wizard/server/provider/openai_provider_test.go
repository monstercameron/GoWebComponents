package provider

import (
	"testing"

	"github.com/openai/openai-go/shared"
)

func TestOpenAIProviderNonNetworkHelpers(t *testing.T) {
	provider := NewOpenAIProvider("test-key")
	if !provider.Available() {
		t.Fatal("expected provider with test key to be available")
	}
	if provider.ID() != "openai" {
		t.Fatalf("unexpected provider ID: %q", provider.ID())
	}
	if provider.DefaultModel() != openAIDefaultModel {
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
	if NewOpenAIProvider("").Available() {
		t.Fatal("expected provider without key to be unavailable")
	}
}
