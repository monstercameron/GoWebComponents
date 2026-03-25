package provider

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNewStubProviderEmptyAndInfoHealth(parseT *testing.T) {
	parseProvider := ParseNewStubProvider("   ", Catalog{})
	if parseProvider == nil {
		parseT.Fatal("expected provider")
	}
	if parseProvider.ParseID() != "" {
		parseT.Fatalf("unexpected id: %q", parseProvider.ParseID())
	}
	if parseProvider.ParseAvailable() {
		parseT.Fatal("expected empty provider to be unavailable")
	}

	parseInfo := parseProvider.ParseInfo()
	if parseInfo.BaseURL != "stub://" || parseInfo.ParseAvailable {
		parseT.Fatalf("unexpected info: %+v", parseInfo)
	}
	if parseInfo.StreamingSupported != true || parseInfo.ReasoningSupported != true || parseInfo.ToolUseSupported {
		parseT.Fatalf("unexpected capability flags: %+v", parseInfo)
	}

	parseHealth := parseProvider.ParseHealth()
	if parseHealth.ProviderID != "" || parseHealth.ParseStatus != ProviderHealthUnavailable {
		parseT.Fatalf("unexpected health: %+v", parseHealth)
	}
	if parseLimits := parseProvider.ParseCurrentRateLimits(); parseLimits != (RateLimitSnapshot{}) {
		parseT.Fatalf("unexpected rate limits: %+v", parseLimits)
	}
}

func TestStubProviderGenerateTitleAndSpeech(parseT *testing.T) {
	parseProvider := ParseNewStubProvider("openai", Catalog{
		DefaultModel: "gpt-5-mini",
		Models: []ModelMetadata{{
			ID:          "gpt-5-mini",
			DisplayName: "GPT-5 Mini",
			ProviderID:  "openai",
		}},
	})

	parseTitle, parseErr := parseProvider.ParseGenerateTitle(context.Background(), TitleRequest{})
	if parseErr != nil {
		parseT.Fatalf("GenerateTitle empty: %v", parseErr)
	}
	if parseTitle != "OpenAI stub conversation" {
		parseT.Fatalf("unexpected empty title: %q", parseTitle)
	}

	parseTitle, parseErr = parseProvider.ParseGenerateTitle(context.Background(), TitleRequest{
		Prompt: strings.Repeat("word ", 20),
	})
	if parseErr != nil {
		parseT.Fatalf("GenerateTitle prompt: %v", parseErr)
	}
	if !strings.HasPrefix(parseTitle, "OpenAI stub: ") || !strings.Contains(parseTitle, "…") {
		parseT.Fatalf("unexpected truncated title: %q", parseTitle)
	}

	parseOptions := parseProvider.ParseModelOptions()
	if len(parseOptions) != 1 || parseOptions[0].ParseID != "gpt-5-mini" {
		parseT.Fatalf("unexpected model options: %+v", parseOptions)
	}
	parseCapabilities := parseProvider.ParseCapabilities("gpt-5-mini")
	if parseCapabilities.ProviderID != "openai" {
		parseT.Fatalf("unexpected capabilities: %+v", parseCapabilities)
	}
	parseMissingCapabilities := parseProvider.ParseCapabilities("missing")
	if parseMissingCapabilities.ProviderID != "openai" || parseMissingCapabilities.ProviderLabel != "OpenAI" {
		parseT.Fatalf("unexpected fallback capabilities: %+v", parseMissingCapabilities)
	}

	parseMemories, parseErr := parseProvider.ParseExtractUserMemories(context.Background(), MemoryExtractionRequest{})
	if parseErr != nil {
		parseT.Fatalf("ExtractUserMemories: %v", parseErr)
	}
	if parseMemories != nil {
		parseT.Fatalf("expected nil memories, got %+v", parseMemories)
	}

	var parseChunks []SpeechChunk
	parseResult, parseErr := parseProvider.ParseSynthesizeSpeech(context.Background(), SpeechRequest{
		Text: "hello",
	}, func(parseChunk SpeechChunk) error {
		parseChunks = append(parseChunks, parseChunk)
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("SynthesizeSpeech: %v", parseErr)
	}
	if parseResult.Model != "gpt-5-mini" || parseResult.MimeType != "audio/mpeg" || parseResult.Voice != "stub" {
		parseT.Fatalf("unexpected speech result: %+v", parseResult)
	}
	if len(parseChunks) != 2 {
		parseT.Fatalf("expected two chunks, got %+v", parseChunks)
	}
	if string(parseChunks[0].AudioChunk) != "stub-audio" || parseChunks[0].Done {
		parseT.Fatalf("unexpected first chunk: %+v", parseChunks[0])
	}
	if !parseChunks[1].Done || len(parseChunks[1].AudioChunk) != 0 {
		parseT.Fatalf("unexpected done chunk: %+v", parseChunks[1])
	}
}

func TestStubProviderEmitFailuresAndHelpers(parseT *testing.T) {
	parseProvider := ParseNewStubProvider("cerebras", Catalog{
		DefaultModel: "cerebras-gpt",
		Models: []ModelMetadata{{
			ID:          "cerebras-gpt",
			DisplayName: "Cerebras GPT",
			ProviderID:  "cerebras",
		}},
	})

	parseChatErr := errors.New("chat emit failed")
	_, parseErr := parseProvider.ParseStreamChat(context.Background(), ChatRequest{
		ThinkingEnabled: true,
	}, func(parseEvent ChatEvent) error {
		if parseEvent.ThoughtDelta != "" {
			return parseChatErr
		}
		return nil
	})
	if !errors.Is(parseErr, parseChatErr) {
		parseT.Fatalf("expected chat emit failure, got %v", parseErr)
	}

	parseSpeechErr := errors.New("speech emit failed")
	_, parseErr = parseProvider.ParseSynthesizeSpeech(context.Background(), SpeechRequest{
		Model: "cerebras-gpt",
		Text:  "hello",
	}, func(SpeechChunk) error {
		return parseSpeechErr
	})
	if !errors.Is(parseErr, parseSpeechErr) {
		parseT.Fatalf("expected speech emit failure, got %v", parseErr)
	}

	if parseGot := parseStubProviderLabel(" OpenAI "); parseGot != "OpenAI" {
		parseT.Fatalf("stubProviderLabel openai = %q", parseGot)
	}
	if parseGot2 := parseStubProviderLabel(" unknown "); parseGot2 != "UNKNOWN" {
		parseT.Fatalf("stubProviderLabel default = %q", parseGot2)
	}
	if parseGot3 := parseNormalizeStubThinkingEffort(" HIGH "); parseGot3 != "high" {
		parseT.Fatalf("normalizeStubThinkingEffort high = %q", parseGot3)
	}
	if parseGot4 := parseNormalizeStubThinkingEffort("turbo"); parseGot4 != "medium" {
		parseT.Fatalf("normalizeStubThinkingEffort default = %q", parseGot4)
	}
}
