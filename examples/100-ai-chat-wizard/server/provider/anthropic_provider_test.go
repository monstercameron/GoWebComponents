package provider

import (
	"errors"
	"testing"
)

func TestAnthropicProviderNonNetworkHelpers(t *testing.T) {
	provider := NewAnthropicProvider("test-key")
	if !provider.Available() {
		t.Fatal("expected anthropic provider with test key to be available")
	}
	if provider.ID() != "anthropic" {
		t.Fatalf("unexpected provider ID: %q", provider.ID())
	}
	if provider.DefaultModel() != anthropicDefaultModel {
		t.Fatalf("unexpected default model: %q", provider.DefaultModel())
	}
	if !provider.SupportsModel("claude-sonnet-4-5") || provider.SupportsModel("gpt-5.4-mini") {
		t.Fatal("unexpected SupportsModel behavior for Anthropic provider")
	}
	capabilities := provider.Capabilities("claude-sonnet-4-5")
	if !capabilities.SupportsThinking || capabilities.SupportsSpeech {
		t.Fatalf("unexpected capabilities: %+v", capabilities)
	}
	if len(provider.ModelOptions()) != 2 {
		t.Fatalf("expected two Anthropic model options, got %d", len(provider.ModelOptions()))
	}
	if got := anthropicThinkingBudget("HIGH"); got != 4096 {
		t.Fatalf("unexpected high thinking budget: %d", got)
	}
	if got := anthropicThinkingBudget("low"); got != 1024 {
		t.Fatalf("unexpected low thinking budget: %d", got)
	}
	if !anthropicThinkingUnsupported(errors.New("thinking unsupported invalid_request_error")) {
		t.Fatal("expected unsupported thinking error to be detected")
	}
	if anthropicThinkingUnsupported(nil) {
		t.Fatal("expected nil error to return false")
	}
	if NewAnthropicProvider("").Available() {
		t.Fatal("expected provider without key to be unavailable")
	}
}
