package provider

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNewStubProviderEmptyAndInfoHealth(t *testing.T) {
	provider := NewStubProvider("   ", Catalog{})
	if provider == nil {
		t.Fatal("expected provider")
	}
	if provider.ID() != "" {
		t.Fatalf("unexpected id: %q", provider.ID())
	}
	if provider.Available() {
		t.Fatal("expected empty provider to be unavailable")
	}

	info := provider.Info()
	if info.BaseURL != "stub://" || info.Available {
		t.Fatalf("unexpected info: %+v", info)
	}
	if info.StreamingSupported != true || info.ReasoningSupported != true || info.ToolUseSupported {
		t.Fatalf("unexpected capability flags: %+v", info)
	}

	health := provider.Health()
	if health.ProviderID != "" || health.Status != ProviderHealthUnavailable {
		t.Fatalf("unexpected health: %+v", health)
	}
	if limits := provider.CurrentRateLimits(); limits != (RateLimitSnapshot{}) {
		t.Fatalf("unexpected rate limits: %+v", limits)
	}
}

func TestStubProviderGenerateTitleAndSpeech(t *testing.T) {
	provider := NewStubProvider("openai", Catalog{
		DefaultModel: "gpt-5-mini",
		Models: []ModelMetadata{{
			ID:          "gpt-5-mini",
			DisplayName: "GPT-5 Mini",
			ProviderID:  "openai",
		}},
	})

	title, err := provider.GenerateTitle(context.Background(), TitleRequest{})
	if err != nil {
		t.Fatalf("GenerateTitle empty: %v", err)
	}
	if title != "OpenAI stub conversation" {
		t.Fatalf("unexpected empty title: %q", title)
	}

	title, err = provider.GenerateTitle(context.Background(), TitleRequest{
		Prompt: strings.Repeat("word ", 20),
	})
	if err != nil {
		t.Fatalf("GenerateTitle prompt: %v", err)
	}
	if !strings.HasPrefix(title, "OpenAI stub: ") || !strings.Contains(title, "…") {
		t.Fatalf("unexpected truncated title: %q", title)
	}

	options := provider.ModelOptions()
	if len(options) != 1 || options[0].ID != "gpt-5-mini" {
		t.Fatalf("unexpected model options: %+v", options)
	}
	capabilities := provider.Capabilities("gpt-5-mini")
	if capabilities.ProviderID != "openai" {
		t.Fatalf("unexpected capabilities: %+v", capabilities)
	}
	missingCapabilities := provider.Capabilities("missing")
	if missingCapabilities.ProviderID != "openai" || missingCapabilities.ProviderLabel != "OpenAI" {
		t.Fatalf("unexpected fallback capabilities: %+v", missingCapabilities)
	}

	memories, err := provider.ExtractUserMemories(context.Background(), MemoryExtractionRequest{})
	if err != nil {
		t.Fatalf("ExtractUserMemories: %v", err)
	}
	if memories != nil {
		t.Fatalf("expected nil memories, got %+v", memories)
	}

	var chunks []SpeechChunk
	result, err := provider.SynthesizeSpeech(context.Background(), SpeechRequest{
		Text: "hello",
	}, func(chunk SpeechChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("SynthesizeSpeech: %v", err)
	}
	if result.Model != "gpt-5-mini" || result.MimeType != "audio/mpeg" || result.Voice != "stub" {
		t.Fatalf("unexpected speech result: %+v", result)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected two chunks, got %+v", chunks)
	}
	if string(chunks[0].AudioChunk) != "stub-audio" || chunks[0].Done {
		t.Fatalf("unexpected first chunk: %+v", chunks[0])
	}
	if !chunks[1].Done || len(chunks[1].AudioChunk) != 0 {
		t.Fatalf("unexpected done chunk: %+v", chunks[1])
	}
}

func TestStubProviderEmitFailuresAndHelpers(t *testing.T) {
	provider := NewStubProvider("cerebras", Catalog{
		DefaultModel: "cerebras-gpt",
		Models: []ModelMetadata{{
			ID:          "cerebras-gpt",
			DisplayName: "Cerebras GPT",
			ProviderID:  "cerebras",
		}},
	})

	chatErr := errors.New("chat emit failed")
	_, err := provider.StreamChat(context.Background(), ChatRequest{
		ThinkingEnabled: true,
	}, func(event ChatEvent) error {
		if event.ThoughtDelta != "" {
			return chatErr
		}
		return nil
	})
	if !errors.Is(err, chatErr) {
		t.Fatalf("expected chat emit failure, got %v", err)
	}

	speechErr := errors.New("speech emit failed")
	_, err = provider.SynthesizeSpeech(context.Background(), SpeechRequest{
		Model: "cerebras-gpt",
		Text:  "hello",
	}, func(SpeechChunk) error {
		return speechErr
	})
	if !errors.Is(err, speechErr) {
		t.Fatalf("expected speech emit failure, got %v", err)
	}

	if got := stubProviderLabel(" OpenAI "); got != "OpenAI" {
		t.Fatalf("stubProviderLabel openai = %q", got)
	}
	if got := stubProviderLabel(" unknown "); got != "UNKNOWN" {
		t.Fatalf("stubProviderLabel default = %q", got)
	}
	if got := normalizeStubThinkingEffort(" HIGH "); got != "high" {
		t.Fatalf("normalizeStubThinkingEffort high = %q", got)
	}
	if got := normalizeStubThinkingEffort("turbo"); got != "medium" {
		t.Fatalf("normalizeStubThinkingEffort default = %q", got)
	}
}
