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

func TestRegistryHelperMethodsAndCapabilities(parseT *testing.T) {
	parseCapabilities := ModelCapabilities{ProviderID: "stub", ProviderLabel: "Stub", SupportsThinking: true, SupportsSpeech: false}
	parseRegistry := ParseNewRegistry(&stubProvider{
		id:           "stub",
		available:    true,
		defaultModel: " stub-default ",
		models:       map[string]ModelCapabilities{"model-a": parseCapabilities},
		options:      []ModelOption{{ID: "model-a", Label: "Model A", Capabilities: parseCapabilities}},
		metadata: map[string]ModelMetadata{
			"model-a": {
				ID:            "model-a",
				DisplayName:   "Model A",
				ProviderID:    "stub",
				ProviderLabel: "Stub",
				Capabilities:  parseCapabilities,
				Pricing: ModelPricing{
					InputPerMillionUSD:  0.5,
					OutputPerMillionUSD: 1.25,
				},
			},
		},
	})

	if !parseCapabilities.ParseSupports(CapabilityThinking) {
		parseT.Fatal("expected thinking capability to be supported")
	}
	if parseCapabilities.ParseSupports(Capability("unknown")) {
		parseT.Fatal("expected unknown capability to be unsupported")
	}
	if parseRegistry.ParseDefaultModel() != "stub-default" {
		parseT.Fatalf("unexpected registry default model: %q", parseRegistry.ParseDefaultModel())
	}
	if len(parseRegistry.ParseModelOptions()) != 1 {
		parseT.Fatalf("expected one registry model option, got %d", len(parseRegistry.ParseModelOptions()))
	}
	parseResolvedCapabilities, parseResolvedModel, parseErr := parseRegistry.ParseCapabilities("model-a")
	if parseErr != nil {
		parseT.Fatalf("Capabilities: %v", parseErr)
	}
	if parseResolvedModel != "model-a" || !parseResolvedCapabilities.SupportsThinking || parseResolvedCapabilities.SupportsSpeech {
		parseT.Fatalf("unexpected capabilities resolution: model=%q caps=%+v", parseResolvedModel, parseResolvedCapabilities)
	}
	if _, _, parseErr2 := (*Registry)(nil).ParseCapabilities("model-a"); !errors.Is(parseErr2, ErrNoProvidersAvailable) {
		parseT.Fatalf("expected nil registry capabilities to fail with ErrNoProvidersAvailable, got %v", parseErr2)
	}
	if parseInfos := parseRegistry.ParseProviderInfos(); len(parseInfos) != 1 || parseInfos[0].ID != "stub" {
		parseT.Fatalf("unexpected provider infos: %+v", parseInfos)
	}
	parseMetadata, parseResolvedModel, parseErr := parseRegistry.ParseModelMetadata("model-a")
	if parseErr != nil {
		parseT.Fatalf("ModelMetadata: %v", parseErr)
	}
	if parseResolvedModel != "model-a" || parseMetadata.DisplayName != "Model A" {
		parseT.Fatalf("unexpected metadata resolution: model=%q metadata=%+v", parseResolvedModel, parseMetadata)
	}
	parsePricing, parseResolvedModel, parseErr := parseRegistry.ParsePricing("model-a")
	if parseErr != nil {
		parseT.Fatalf("Pricing: %v", parseErr)
	}
	if parseResolvedModel != "model-a" || parsePricing.InputPerMillionUSD != 0.5 || parsePricing.OutputPerMillionUSD != 1.25 {
		parseT.Fatalf("unexpected pricing resolution: model=%q pricing=%+v", parseResolvedModel, parsePricing)
	}
	if parseSnapshots := parseRegistry.ParseHealthSnapshots(); len(parseSnapshots) != 1 || parseSnapshots[0].ProviderID != "stub" {
		parseT.Fatalf("unexpected health snapshots: %+v", parseSnapshots)
	}
}

func TestOpenAIProviderUnavailableAndUnsupportedBranches(parseT *testing.T) {
	parseProvider := ParseNewOpenAIProvider("", parseTestOpenAICatalog())
	parseCtx := context.Background()

	if _, parseErr := parseProvider.ParseGenerateTitle(parseCtx, TitleRequest{Prompt: "hello"}); !errors.Is(parseErr, ErrNoProvidersAvailable) {
		parseT.Fatalf("expected unavailable GenerateTitle to return ErrNoProvidersAvailable, got %v", parseErr)
	}
	if _, parseErr2 := parseProvider.ParseExtractUserMemories(parseCtx, MemoryExtractionRequest{UserMessage: "remember this"}); !errors.Is(parseErr2, ErrNoProvidersAvailable) {
		parseT.Fatalf("expected unavailable ExtractUserMemories to return ErrNoProvidersAvailable, got %v", parseErr2)
	}
	if _, parseErr3 := parseProvider.ParseStreamChat(parseCtx, ChatRequest{UserMessage: "hello"}, func(ChatEvent) error { return nil }); !errors.Is(parseErr3, ErrNoProvidersAvailable) {
		parseT.Fatalf("expected unavailable StreamChat to return ErrNoProvidersAvailable, got %v", parseErr3)
	}
	if _, parseErr4 := parseProvider.ParseSynthesizeSpeech(parseCtx, SpeechRequest{Model: "gpt-5.4-mini", Text: "hello"}, func(SpeechChunk) error { return nil }); !errors.Is(parseErr4, ErrNoProvidersAvailable) {
		parseT.Fatalf("expected unavailable SynthesizeSpeech to return ErrNoProvidersAvailable, got %v", parseErr4)
	}

	parseAvailableProvider := ParseNewOpenAIProvider("test-key", parseTestOpenAICatalog())
	_, parseErr5 := parseAvailableProvider.ParseSynthesizeSpeech(parseCtx, SpeechRequest{Model: "claude-sonnet-4-5", Text: "hello"}, func(SpeechChunk) error { return nil })
	var parseUnsupportedErr *UnsupportedCapabilityError
	if !errors.As(parseErr5, &parseUnsupportedErr) {
		parseT.Fatalf("expected unsupported capability error for OpenAI speech model mismatch, got %v", parseErr5)
	}
	if parseUnsupportedErr.ProviderID != "openai" || parseUnsupportedErr.Capability != CapabilitySpeech {
		parseT.Fatalf("unexpected unsupported capability details: %+v", parseUnsupportedErr)
	}
	if parseGot := parseOpenAIReasoningEffort("low"); parseGot != shared.ReasoningEffortLow {
		parseT.Fatalf("expected low reasoning effort, got %q", parseGot)
	}
	if parseGot2 := parseOpenAIReasoningEffort("unexpected"); parseGot2 != shared.ReasoningEffortMedium {
		parseT.Fatalf("expected medium reasoning effort fallback, got %q", parseGot2)
	}
}

func TestAnthropicProviderHelperAndFallbackBranches(parseT *testing.T) {
	parseProvider := ParseNewAnthropicProvider("", parseTestAnthropicCatalog())
	parseCtx := context.Background()

	if _, parseErr := parseProvider.ParseGenerateTitle(parseCtx, TitleRequest{Prompt: "hello"}); !errors.Is(parseErr, ErrNoProvidersAvailable) {
		parseT.Fatalf("expected unavailable GenerateTitle to return ErrNoProvidersAvailable, got %v", parseErr)
	}
	if _, parseErr2 := parseProvider.ParseStreamChat(parseCtx, ChatRequest{UserMessage: "hello"}, func(ChatEvent) error { return nil }); !errors.Is(parseErr2, ErrNoProvidersAvailable) {
		parseT.Fatalf("expected unavailable StreamChat to return ErrNoProvidersAvailable, got %v", parseErr2)
	}
	if _, parseErr3 := parseProvider.ParseExtractUserMemories(parseCtx, MemoryExtractionRequest{UserMessage: "remember this"}); !errors.Is(parseErr3, ErrNoProvidersAvailable) {
		parseT.Fatalf("expected unavailable ExtractUserMemories to return ErrNoProvidersAvailable, got %v", parseErr3)
	}
	if _, parseErr4 := parseProvider.ParseSynthesizeSpeech(parseCtx, SpeechRequest{Model: "claude-sonnet-4-5", Text: "hello"}, func(SpeechChunk) error { return nil }); !errors.Is(parseErr4, ErrNoProvidersAvailable) {
		parseT.Fatalf("expected unavailable SynthesizeSpeech to return ErrNoProvidersAvailable, got %v", parseErr4)
	}

	parseAvailableProvider := ParseNewAnthropicProvider("test-key", parseTestAnthropicCatalog())
	if _, parseErr5 := parseAvailableProvider.ParseExtractUserMemories(parseCtx, MemoryExtractionRequest{UserMessage: "remember this"}); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "not implemented") {
		parseT.Fatalf("expected configured Anthropic memory extraction to surface not implemented, got %v", parseErr5)
	}
	_, parseErr6 := parseAvailableProvider.ParseSynthesizeSpeech(parseCtx, SpeechRequest{Model: "claude-sonnet-4-5", Text: "hello"}, func(SpeechChunk) error { return nil })
	var parseUnsupportedErr *UnsupportedCapabilityError
	if !errors.As(parseErr6, &parseUnsupportedErr) {
		parseT.Fatalf("expected unsupported capability error for Anthropic speech, got %v", parseErr6)
	}

	parseMessageParam := parseAnthropicTextMessage(" assistant ", " hello ")
	if string(parseMessageParam.Role) != "assistant" {
		parseT.Fatalf("unexpected anthropic message role: %q", parseMessageParam.Role)
	}
	if parseMessageParam.Content[0].OfText == nil || parseMessageParam.Content[0].OfText.Text != "hello" {
		parseT.Fatalf("unexpected anthropic text message content: %+v", parseMessageParam.Content)
	}

	parseConversation := parseAnthropicMessages([]ChatMessage{{Role: "developer", Content: " plan "}}, " latest ")
	if len(parseConversation) != 2 || string(parseConversation[0].Role) != "developer" || string(parseConversation[1].Role) != "user" {
		parseT.Fatalf("unexpected anthropic messages conversion: %+v", parseConversation)
	}
	if parsePrompt := parseAnthropicSystemPrompt("   "); parsePrompt != nil {
		parseT.Fatalf("expected blank system prompt to collapse to nil, got %+v", parsePrompt)
	}
	if parsePrompt2 := parseAnthropicSystemPrompt(" system "); len(parsePrompt2) != 1 || parsePrompt2[0].Text != "system" {
		parseT.Fatalf("unexpected anthropic system prompt: %+v", parsePrompt2)
	}
	if parseAnthropicMessageText(nil) != "" {
		parseT.Fatal("expected nil anthropic message text to be empty")
	}
	parseMessage := &anthropic.Message{}
	if parseErr7 := json.Unmarshal([]byte(`{"content":[{"type":"text","text":"Hello"},{"type":"text","text":" world"}]}`), parseMessage); parseErr7 != nil {
		parseT.Fatalf("json.Unmarshal anthropic message: %v", parseErr7)
	}
	if parseGot := parseAnthropicMessageText(parseMessage); parseGot != "Hello world" {
		parseT.Fatalf("unexpected anthropic message text: %q", parseGot)
	}
	if parseGot2 := parseAnthropicThinkingBudget("medium"); parseGot2 != 2048 {
		parseT.Fatalf("unexpected medium thinking budget: %d", parseGot2)
	}
	if parseAnthropicThinkingUnsupported(errors.New("plain failure")) {
		parseT.Fatal("expected plain error to not be treated as unsupported thinking")
	}
}

func TestPricingAndRateLimitHelpers(parseT *testing.T) {
	parseEstimate := ParseEstimateCost(250000, 500000, ModelPricing{
		InputPerMillionUSD:  2.25,
		OutputPerMillionUSD: 2.75,
	})
	if parseEstimate.InputCostUSD != 0.5625 || parseEstimate.OutputCostUSD != 1.375 || parseEstimate.TotalCostUSD != 1.9375 {
		parseT.Fatalf("unexpected cost estimate: %+v", parseEstimate)
	}
	if !(RateLimitSnapshot{}).ParseEmpty() {
		parseT.Fatal("expected zero-value rate-limit snapshot to be empty")
	}
	if (RateLimitSnapshot{Source: "headers"}).ParseEmpty() {
		parseT.Fatal("expected non-zero rate-limit snapshot to be non-empty")
	}
}
