package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/shared"
)

func TestRegistryHelperMethodsAndCapabilities(t *testing.T) {
	capabilities := ModelCapabilities{ProviderID: "stub", ProviderLabel: "Stub", SupportsThinking: true, SupportsSpeech: false}
	registry := NewRegistry(&stubProvider{
		id:           "stub",
		available:    true,
		defaultModel: " stub-default ",
		models:       map[string]ModelCapabilities{"model-a": capabilities},
		options:      []ModelOption{{ID: "model-a", Label: "Model A", Capabilities: capabilities}},
		metadata: map[string]ModelMetadata{
			"model-a": {
				ID:            "model-a",
				DisplayName:   "Model A",
				ProviderID:    "stub",
				ProviderLabel: "Stub",
				Capabilities:  capabilities,
				Pricing: ModelPricing{
					InputPerMillionUSD:  0.5,
					OutputPerMillionUSD: 1.25,
				},
			},
		},
	})

	if !capabilities.Supports(CapabilityThinking) {
		t.Fatal("expected thinking capability to be supported")
	}
	if capabilities.Supports(Capability("unknown")) {
		t.Fatal("expected unknown capability to be unsupported")
	}
	if registry.DefaultModel() != "stub-default" {
		t.Fatalf("unexpected registry default model: %q", registry.DefaultModel())
	}
	if len(registry.ModelOptions()) != 1 {
		t.Fatalf("expected one registry model option, got %d", len(registry.ModelOptions()))
	}
	resolvedCapabilities, resolvedModel, err := registry.Capabilities("model-a")
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if resolvedModel != "model-a" || !resolvedCapabilities.SupportsThinking || resolvedCapabilities.SupportsSpeech {
		t.Fatalf("unexpected capabilities resolution: model=%q caps=%+v", resolvedModel, resolvedCapabilities)
	}
	if _, _, err := (*Registry)(nil).Capabilities("model-a"); !errors.Is(err, ErrNoProvidersAvailable) {
		t.Fatalf("expected nil registry capabilities to fail with ErrNoProvidersAvailable, got %v", err)
	}
	if infos := registry.ProviderInfos(); len(infos) != 1 || infos[0].ID != "stub" {
		t.Fatalf("unexpected provider infos: %+v", infos)
	}
	metadata, resolvedModel, err := registry.ModelMetadata("model-a")
	if err != nil {
		t.Fatalf("ModelMetadata: %v", err)
	}
	if resolvedModel != "model-a" || metadata.DisplayName != "Model A" {
		t.Fatalf("unexpected metadata resolution: model=%q metadata=%+v", resolvedModel, metadata)
	}
	pricing, resolvedModel, err := registry.Pricing("model-a")
	if err != nil {
		t.Fatalf("Pricing: %v", err)
	}
	if resolvedModel != "model-a" || pricing.InputPerMillionUSD != 0.5 || pricing.OutputPerMillionUSD != 1.25 {
		t.Fatalf("unexpected pricing resolution: model=%q pricing=%+v", resolvedModel, pricing)
	}
	if snapshots := registry.HealthSnapshots(); len(snapshots) != 1 || snapshots[0].ProviderID != "stub" {
		t.Fatalf("unexpected health snapshots: %+v", snapshots)
	}
}

func TestOpenAIProviderUnavailableAndUnsupportedBranches(t *testing.T) {
	provider := NewOpenAIProvider("")
	ctx := context.Background()

	if _, err := provider.GenerateTitle(ctx, TitleRequest{Prompt: "hello"}); !errors.Is(err, ErrNoProvidersAvailable) {
		t.Fatalf("expected unavailable GenerateTitle to return ErrNoProvidersAvailable, got %v", err)
	}
	if _, err := provider.ExtractUserMemories(ctx, MemoryExtractionRequest{UserMessage: "remember this"}); !errors.Is(err, ErrNoProvidersAvailable) {
		t.Fatalf("expected unavailable ExtractUserMemories to return ErrNoProvidersAvailable, got %v", err)
	}
	if _, err := provider.StreamChat(ctx, ChatRequest{UserMessage: "hello"}, func(ChatEvent) error { return nil }); !errors.Is(err, ErrNoProvidersAvailable) {
		t.Fatalf("expected unavailable StreamChat to return ErrNoProvidersAvailable, got %v", err)
	}
	if _, err := provider.SynthesizeSpeech(ctx, SpeechRequest{Model: "gpt-5.4-mini", Text: "hello"}, func(SpeechChunk) error { return nil }); !errors.Is(err, ErrNoProvidersAvailable) {
		t.Fatalf("expected unavailable SynthesizeSpeech to return ErrNoProvidersAvailable, got %v", err)
	}

	availableProvider := NewOpenAIProvider("test-key")
	_, err := availableProvider.SynthesizeSpeech(ctx, SpeechRequest{Model: "claude-sonnet-4-5", Text: "hello"}, func(SpeechChunk) error { return nil })
	var unsupportedErr *UnsupportedCapabilityError
	if !errors.As(err, &unsupportedErr) {
		t.Fatalf("expected unsupported capability error for OpenAI speech model mismatch, got %v", err)
	}
	if unsupportedErr.ProviderID != "openai" || unsupportedErr.Capability != CapabilitySpeech {
		t.Fatalf("unexpected unsupported capability details: %+v", unsupportedErr)
	}
	if got := openAIReasoningEffort("low"); got != shared.ReasoningEffortLow {
		t.Fatalf("expected low reasoning effort, got %q", got)
	}
	if got := openAIReasoningEffort("unexpected"); got != shared.ReasoningEffortMedium {
		t.Fatalf("expected medium reasoning effort fallback, got %q", got)
	}
}

func TestAnthropicProviderHelperAndFallbackBranches(t *testing.T) {
	provider := NewAnthropicProvider("")
	ctx := context.Background()

	if _, err := provider.GenerateTitle(ctx, TitleRequest{Prompt: "hello"}); !errors.Is(err, ErrNoProvidersAvailable) {
		t.Fatalf("expected unavailable GenerateTitle to return ErrNoProvidersAvailable, got %v", err)
	}
	if _, err := provider.StreamChat(ctx, ChatRequest{UserMessage: "hello"}, func(ChatEvent) error { return nil }); !errors.Is(err, ErrNoProvidersAvailable) {
		t.Fatalf("expected unavailable StreamChat to return ErrNoProvidersAvailable, got %v", err)
	}
	if _, err := provider.ExtractUserMemories(ctx, MemoryExtractionRequest{UserMessage: "remember this"}); !errors.Is(err, ErrNoProvidersAvailable) {
		t.Fatalf("expected unavailable ExtractUserMemories to return ErrNoProvidersAvailable, got %v", err)
	}
	if _, err := provider.SynthesizeSpeech(ctx, SpeechRequest{Model: anthropicDefaultModel, Text: "hello"}, func(SpeechChunk) error { return nil }); !errors.Is(err, ErrNoProvidersAvailable) {
		t.Fatalf("expected unavailable SynthesizeSpeech to return ErrNoProvidersAvailable, got %v", err)
	}

	availableProvider := NewAnthropicProvider("test-key")
	if _, err := availableProvider.ExtractUserMemories(ctx, MemoryExtractionRequest{UserMessage: "remember this"}); err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected configured Anthropic memory extraction to surface not implemented, got %v", err)
	}
	_, err := availableProvider.SynthesizeSpeech(ctx, SpeechRequest{Model: anthropicDefaultModel, Text: "hello"}, func(SpeechChunk) error { return nil })
	var unsupportedErr *UnsupportedCapabilityError
	if !errors.As(err, &unsupportedErr) {
		t.Fatalf("expected unsupported capability error for Anthropic speech, got %v", err)
	}

	messageParam := anthropicTextMessage(" assistant ", " hello ")
	if string(messageParam.Role) != "assistant" {
		t.Fatalf("unexpected anthropic message role: %q", messageParam.Role)
	}
	if messageParam.Content[0].OfText == nil || messageParam.Content[0].OfText.Text != "hello" {
		t.Fatalf("unexpected anthropic text message content: %+v", messageParam.Content)
	}

	conversation := anthropicMessages([]ChatMessage{{Role: "developer", Content: " plan "}}, " latest ")
	if len(conversation) != 2 || string(conversation[0].Role) != "developer" || string(conversation[1].Role) != "user" {
		t.Fatalf("unexpected anthropic messages conversion: %+v", conversation)
	}
	if prompt := anthropicSystemPrompt("   "); prompt != nil {
		t.Fatalf("expected blank system prompt to collapse to nil, got %+v", prompt)
	}
	if prompt := anthropicSystemPrompt(" system "); len(prompt) != 1 || prompt[0].Text != "system" {
		t.Fatalf("unexpected anthropic system prompt: %+v", prompt)
	}
	if anthropicMessageText(nil) != "" {
		t.Fatal("expected nil anthropic message text to be empty")
	}
	message := &anthropic.Message{}
	if err := json.Unmarshal([]byte(`{"content":[{"type":"text","text":"Hello"},{"type":"text","text":" world"}]}`), message); err != nil {
		t.Fatalf("json.Unmarshal anthropic message: %v", err)
	}
	if got := anthropicMessageText(message); got != "Hello world" {
		t.Fatalf("unexpected anthropic message text: %q", got)
	}
	if got := anthropicThinkingBudget("medium"); got != 2048 {
		t.Fatalf("unexpected medium thinking budget: %d", got)
	}
	if anthropicThinkingUnsupported(errors.New("plain failure")) {
		t.Fatal("expected plain error to not be treated as unsupported thinking")
	}
}

func TestPricingAndRateLimitHelpers(t *testing.T) {
	estimate := EstimateCost(250000, 500000, ModelPricing{
		InputPerMillionUSD:  2.25,
		OutputPerMillionUSD: 2.75,
	})
	if estimate.InputCostUSD != 0.5625 || estimate.OutputCostUSD != 1.375 || estimate.TotalCostUSD != 1.9375 {
		t.Fatalf("unexpected cost estimate: %+v", estimate)
	}
	if !(RateLimitSnapshot{}).Empty() {
		t.Fatal("expected zero-value rate-limit snapshot to be empty")
	}
	if (RateLimitSnapshot{Source: "headers"}).Empty() {
		t.Fatal("expected non-zero rate-limit snapshot to be non-empty")
	}
}
