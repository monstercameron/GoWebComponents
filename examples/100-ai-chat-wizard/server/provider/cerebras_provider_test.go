package provider

import (
	"context"
	"testing"

	"github.com/openai/openai-go/shared"
)

func TestCerebrasProviderNonNetworkHelpers(t *testing.T) {
	provider := NewCerebrasProvider("test-key", testCerebrasCatalog())
	if !provider.Available() {
		t.Fatal("expected cerebras provider with test key to be available")
	}
	if provider.ID() != "cerebras" {
		t.Fatalf("unexpected provider ID: %q", provider.ID())
	}
	if provider.DefaultModel() != "gpt-oss-120b" {
		t.Fatalf("unexpected default model: %q", provider.DefaultModel())
	}
	if !provider.SupportsModel("gpt-oss-120b") || provider.SupportsModel("gpt-5.4-mini") {
		t.Fatal("unexpected SupportsModel behavior for Cerebras provider")
	}
	if len(provider.ModelOptions()) != 4 {
		t.Fatalf("expected four Cerebras model options, got %d", len(provider.ModelOptions()))
	}
	if caps := provider.Capabilities("gpt-oss-120b"); !caps.SupportsThinking || caps.SupportsSpeech {
		t.Fatalf("unexpected default Cerebras capabilities: %+v", caps)
	}
	if caps := provider.Capabilities("llama3.1-8b"); caps.SupportsThinking || caps.SupportsSpeech {
		t.Fatalf("unexpected fast Cerebras capabilities: %+v", caps)
	}
	if got := cerebrasReasoningEffort("HIGH"); got != shared.ReasoningEffortHigh {
		t.Fatalf("unexpected reasoning effort normalization: %q", got)
	}
	if got := cerebrasReasoningEffort("weird"); got != shared.ReasoningEffortMedium {
		t.Fatalf("expected medium reasoning effort fallback, got %q", got)
	}
	if delta := cerebrasReasoningDelta(`{"choices":[{"delta":{"reasoning":"step 1"}}]}`); delta != "step 1" {
		t.Fatalf("unexpected reasoning delta parse: %q", delta)
	}
	if delta := cerebrasReasoningDelta(`{"choices":[{"delta":{"reasoning_content":"step 2"}}]}`); delta != "step 2" {
		t.Fatalf("unexpected reasoning_content delta parse: %q", delta)
	}
	if delta := cerebrasReasoningDelta(`{}`); delta != "" {
		t.Fatalf("expected empty reasoning delta, got %q", delta)
	}
}

func TestCerebrasProviderMetadataAndUnavailableBranches(t *testing.T) {
	provider := NewCerebrasProvider("test-key", testCerebrasCatalog())
	if info := provider.Info(); info.ID != "cerebras" || info.Label != "Cerebras" || !info.Available || !info.AuthConfigured || info.BaseURL != cerebrasBaseURL {
		t.Fatalf("unexpected Cerebras info: %+v", info)
	}
	if health := provider.Health(); health.ProviderID != "cerebras" || health.Status != ProviderHealthUnknown {
		t.Fatalf("unexpected Cerebras health: %+v", health)
	}
	if limits := provider.CurrentRateLimits(); !limits.Empty() {
		t.Fatalf("expected Cerebras rate limits to be empty, got %+v", limits)
	}
	if metadata, ok := provider.ModelMetadata(" GPT-OSS-120B "); !ok || metadata.ID != "gpt-oss-120b" {
		t.Fatalf("unexpected Cerebras metadata resolution: ok=%v metadata=%+v", ok, metadata)
	}
	if metadata, ok := provider.ModelMetadata("unsupported-model"); ok || metadata.ID != "" {
		t.Fatalf("expected unknown Cerebras model metadata to be unavailable, got ok=%v metadata=%+v", ok, metadata)
	}
	if fallback := provider.mustModelMetadata(" custom-cerebras "); fallback.ID != "custom-cerebras" || fallback.DisplayName != "custom-cerebras" || fallback.ProviderID != "cerebras" {
		t.Fatalf("unexpected Cerebras fallback metadata: %+v", fallback)
	}

	unavailable := NewCerebrasProvider("", testCerebrasCatalog())
	if unavailable.Available() {
		t.Fatal("expected provider without key to be unavailable")
	}
	if _, err := unavailable.GenerateTitle(context.Background(), TitleRequest{Prompt: "hello"}); err != ErrNoProvidersAvailable {
		t.Fatalf("expected unavailable GenerateTitle to return ErrNoProvidersAvailable, got %v", err)
	}
	if _, err := unavailable.ExtractUserMemories(context.Background(), MemoryExtractionRequest{UserMessage: "remember"}); err != ErrNoProvidersAvailable {
		t.Fatalf("expected unavailable ExtractUserMemories to return ErrNoProvidersAvailable, got %v", err)
	}
	if _, err := unavailable.StreamChat(context.Background(), ChatRequest{UserMessage: "hello"}, func(ChatEvent) error { return nil }); err != ErrNoProvidersAvailable {
		t.Fatalf("expected unavailable StreamChat to return ErrNoProvidersAvailable, got %v", err)
	}
	if _, err := unavailable.SynthesizeSpeech(context.Background(), SpeechRequest{Model: "gpt-oss-120b", Text: "hello"}, func(SpeechChunk) error { return nil }); err != ErrNoProvidersAvailable {
		t.Fatalf("expected unavailable SynthesizeSpeech to return ErrNoProvidersAvailable, got %v", err)
	}
}
