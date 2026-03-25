package provider

import (
	"context"
	"strings"
	"testing"
)

func TestOpenAIProviderAdditionalMetadataAndUnavailableBranches(t *testing.T) {
	provider := NewOpenAIProvider("test-key", testOpenAICatalog())
	if info := provider.Info(); info.ID != "openai" || info.BaseURL != openAIBaseURL || !info.Available || !info.AuthConfigured {
		t.Fatalf("Info() = %+v, want available OpenAI metadata", info)
	}
	if health := provider.Health(); health.ProviderID != "openai" || health.Status != ProviderHealthUnknown {
		t.Fatalf("Health() = %+v, want unknown available health", health)
	}
	if limits := provider.CurrentRateLimits(); !limits.Empty() {
		t.Fatalf("CurrentRateLimits() = %+v, want empty snapshot", limits)
	}
	if metadata, ok := provider.ModelMetadata(" GPT-5.4 "); !ok || metadata.ID != "gpt-5.4" {
		t.Fatalf("ModelMetadata() = %+v, %t; want gpt-5.4", metadata, ok)
	}
	if metadata, ok := provider.ModelMetadata("unsupported-model"); ok || metadata.ID != "" {
		t.Fatalf("ModelMetadata(unsupported) = %+v, %t; want zero,false", metadata, ok)
	}
	if fallback := provider.mustModelMetadata(" custom-openai "); fallback.ID != "custom-openai" || fallback.ProviderID != "openai" {
		t.Fatalf("mustModelMetadata() = %+v, want fallback metadata", fallback)
	}
	if got := normalizeOpenAIModel(" GPT-5.4-MINI "); got != "gpt-5.4-mini" {
		t.Fatalf("normalizeOpenAIModel() = %q, want gpt-5.4-mini", got)
	}
	if got := extractJSONObject("```json\n{\"memories\":[]}\n```"); got != `{"memories":[]}` {
		t.Fatalf("extractJSONObject(fenced) = %q, want JSON body", got)
	}
	if got := extractJSONObject("plain text"); got != "plain text" {
		t.Fatalf("extractJSONObject(plain) = %q, want original text", got)
	}
	if got := extractJSONObject(""); got != `{"memories":[]}` {
		t.Fatalf("extractJSONObject(empty) = %q, want empty memories object", got)
	}

	unavailable := NewOpenAIProvider("", testOpenAICatalog())
	if _, err := unavailable.GenerateTitle(context.Background(), TitleRequest{Prompt: "hello"}); err != ErrNoProvidersAvailable {
		t.Fatalf("GenerateTitle() error = %v, want ErrNoProvidersAvailable", err)
	}
	if _, err := unavailable.ExtractUserMemories(context.Background(), MemoryExtractionRequest{UserMessage: "remember"}); err != ErrNoProvidersAvailable {
		t.Fatalf("ExtractUserMemories() error = %v, want ErrNoProvidersAvailable", err)
	}
	if _, err := unavailable.StreamChat(context.Background(), ChatRequest{UserMessage: "hello"}, func(ChatEvent) error { return nil }); err != ErrNoProvidersAvailable {
		t.Fatalf("StreamChat() error = %v, want ErrNoProvidersAvailable", err)
	}
	if _, err := provider.SynthesizeSpeech(context.Background(), SpeechRequest{Model: "unknown-model", Text: "hello"}, func(SpeechChunk) error { return nil }); err == nil || !strings.Contains(err.Error(), `does not support speech`) {
		t.Fatalf("SynthesizeSpeech() error = %v, want unsupported speech capability error", err)
	}
}

func TestAnthropicProviderAdditionalMetadataAndHelpers(t *testing.T) {
	provider := NewAnthropicProvider("test-key", testAnthropicCatalog())
	if info := provider.Info(); info.ID != "anthropic" || info.BaseURL != anthropicBaseURL || !info.Available {
		t.Fatalf("Info() = %+v, want available anthropic metadata", info)
	}
	if health := provider.Health(); health.ProviderID != "anthropic" || health.Status != ProviderHealthUnknown {
		t.Fatalf("Health() = %+v, want unknown available health", health)
	}
	if limits := provider.CurrentRateLimits(); !limits.Empty() {
		t.Fatalf("CurrentRateLimits() = %+v, want empty snapshot", limits)
	}
	if metadata, ok := provider.ModelMetadata(" Claude-Sonnet-4-5 "); !ok || metadata.ID != "claude-sonnet-4-5" {
		t.Fatalf("ModelMetadata() = %+v, %t; want claude-sonnet-4-5", metadata, ok)
	}
	if fallback := provider.mustModelMetadata(" custom-claude "); fallback.ID != "custom-claude" || fallback.ProviderID != "anthropic" {
		t.Fatalf("mustModelMetadata() = %+v, want fallback metadata", fallback)
	}
	if got := normalizeAnthropicModel(" Claude-SONNET-4-5 "); got != "claude-sonnet-4-5" {
		t.Fatalf("normalizeAnthropicModel() = %q, want claude-sonnet-4-5", got)
	}
	if system := anthropicSystemPrompt("  system  "); len(system) != 1 || system[0].Text != "system" {
		t.Fatalf("anthropicSystemPrompt() = %+v, want trimmed system prompt", system)
	}
	if system := anthropicSystemPrompt("   "); system != nil {
		t.Fatalf("anthropicSystemPrompt(blank) = %+v, want nil", system)
	}
	if messages := anthropicMessages([]ChatMessage{{Role: "assistant", Content: "done"}}, "hello"); len(messages) != 2 {
		t.Fatalf("anthropicMessages() len = %d, want 2", len(messages))
	}
	if text := anthropicMessageText(nil); text != "" {
		t.Fatalf("anthropicMessageText(nil) = %q, want empty string", text)
	}

	unavailable := NewAnthropicProvider("", testAnthropicCatalog())
	if _, err := unavailable.GenerateTitle(context.Background(), TitleRequest{Prompt: "hello"}); err != ErrNoProvidersAvailable {
		t.Fatalf("GenerateTitle() error = %v, want ErrNoProvidersAvailable", err)
	}
	if _, err := unavailable.ExtractUserMemories(context.Background(), MemoryExtractionRequest{UserMessage: "remember"}); err != ErrNoProvidersAvailable {
		t.Fatalf("ExtractUserMemories() error = %v, want ErrNoProvidersAvailable", err)
	}
	if _, err := unavailable.StreamChat(context.Background(), ChatRequest{UserMessage: "hello"}, func(ChatEvent) error { return nil }); err != ErrNoProvidersAvailable {
		t.Fatalf("StreamChat() error = %v, want ErrNoProvidersAvailable", err)
	}
	if _, err := provider.SynthesizeSpeech(context.Background(), SpeechRequest{Model: "claude-sonnet-4-5", Text: "hello"}, func(SpeechChunk) error { return nil }); err == nil || !strings.Contains(err.Error(), `does not support speech`) {
		t.Fatalf("SynthesizeSpeech() error = %v, want unsupported speech capability error", err)
	}
}

func TestStubProviderAdditionalBranches(t *testing.T) {
	catalog := testOpenAICatalog()
	provider := NewStubProvider(" openai ", catalog)
	if info := provider.Info(); info.ID != "openai" || info.BaseURL != "stub://openai" || info.AuthConfigured || !info.Available {
		t.Fatalf("Info() = %+v, want available stub provider metadata", info)
	}
	if health := provider.Health(); health.ProviderID != "openai" || health.Status != ProviderHealthUnknown {
		t.Fatalf("Health() = %+v, want unknown available health", health)
	}
	if limits := provider.CurrentRateLimits(); !limits.Empty() {
		t.Fatalf("CurrentRateLimits() = %+v, want empty snapshot", limits)
	}
	if metadata, ok := provider.ModelMetadata("gpt-5.4-mini"); !ok || metadata.ID != "gpt-5.4-mini" {
		t.Fatalf("ModelMetadata() = %+v, %t; want gpt-5.4-mini", metadata, ok)
	}
	if got, err := provider.GenerateTitle(context.Background(), TitleRequest{Prompt: strings.Repeat("word ", 20)}); err != nil || !strings.Contains(got, "OpenAI stub:") {
		t.Fatalf("GenerateTitle() = %q, %v; want stub title", got, err)
	}
	if got, err := provider.GenerateTitle(context.Background(), TitleRequest{}); err != nil || !strings.Contains(got, "stub conversation") {
		t.Fatalf("GenerateTitle(blank) = %q, %v; want default stub title", got, err)
	}
	if memories, err := provider.ExtractUserMemories(context.Background(), MemoryExtractionRequest{}); err != nil || memories != nil {
		t.Fatalf("ExtractUserMemories() = %+v, %v; want nil,nil", memories, err)
	}

	chunks := make([]SpeechChunk, 0, 2)
	result, err := provider.SynthesizeSpeech(context.Background(), SpeechRequest{Text: "hello"}, func(chunk SpeechChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("SynthesizeSpeech() error = %v", err)
	}
	if result.MimeType != "audio/mpeg" || len(chunks) != 2 || len(chunks[0].AudioChunk) == 0 || !chunks[1].Done {
		t.Fatalf("SynthesizeSpeech() = %+v chunks=%+v, want first audio chunk and final done chunk", result, chunks)
	}

	if got := stubProviderLabel(" custom "); got != "CUSTOM" {
		t.Fatalf("stubProviderLabel() = %q, want CUSTOM", got)
	}
	if got := normalizeStubThinkingEffort("odd"); got != "medium" {
		t.Fatalf("normalizeStubThinkingEffort() = %q, want medium", got)
	}
	if unavailable := NewStubProvider("", catalog); unavailable.Available() || unavailable.Info().ID != "" || unavailable.Health().Status != ProviderHealthUnavailable {
		t.Fatalf("blank stub provider = %+v info=%+v health=%+v, want unavailable provider", unavailable, unavailable.Info(), unavailable.Health())
	}
}
