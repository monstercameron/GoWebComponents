package provider

import (
	"context"
	"strings"
	"testing"
)

func TestOpenAIProviderAdditionalMetadataAndUnavailableBranches(parseT *testing.T) {
	parseProvider := ParseNewOpenAIProvider("test-key", parseTestOpenAICatalog())
	if parseInfo := parseProvider.ParseInfo(); parseInfo.ParseID != "openai" || parseInfo.BaseURL != openAIBaseURL || !parseInfo.ParseAvailable || !parseInfo.AuthConfigured {
		parseT.Fatalf("Info() = %+v, want available OpenAI metadata", parseInfo)
	}
	if parseHealth := parseProvider.ParseHealth(); parseHealth.ProviderID != "openai" || parseHealth.ParseStatus != ProviderHealthUnknown {
		parseT.Fatalf("Health() = %+v, want unknown available health", parseHealth)
	}
	if parseLimits := parseProvider.ParseCurrentRateLimits(); !parseLimits.ParseEmpty() {
		parseT.Fatalf("CurrentRateLimits() = %+v, want empty snapshot", parseLimits)
	}
	if parseMetadata, parseOk := parseProvider.ParseModelMetadata(" GPT-5.4 "); !parseOk || parseMetadata.ParseID != "gpt-5.4" {
		parseT.Fatalf("ModelMetadata() = %+v, %t; want gpt-5.4", parseMetadata, parseOk)
	}
	if parseMetadata2, parseOk2 := parseProvider.ParseModelMetadata("unsupported-model"); parseOk2 || parseMetadata2.ParseID != "" {
		parseT.Fatalf("ModelMetadata(unsupported) = %+v, %t; want zero,false", parseMetadata2, parseOk2)
	}
	if parseFallback := parseProvider.parseMustModelMetadata(" custom-openai "); parseFallback.ParseID != "custom-openai" || parseFallback.ProviderID != "openai" {
		parseT.Fatalf("mustModelMetadata() = %+v, want fallback metadata", parseFallback)
	}
	if parseGot := parseNormalizeOpenAIModel(" GPT-5.4-MINI "); parseGot != "gpt-5.4-mini" {
		parseT.Fatalf("normalizeOpenAIModel() = %q, want gpt-5.4-mini", parseGot)
	}
	if parseGot2 := parseExtractJSONObject("```json\n{\"memories\":[]}\n```"); parseGot2 != `{"memories":[]}` {
		parseT.Fatalf("extractJSONObject(fenced) = %q, want JSON body", parseGot2)
	}
	if parseGot3 := parseExtractJSONObject("plain text"); parseGot3 != "plain text" {
		parseT.Fatalf("extractJSONObject(plain) = %q, want original text", parseGot3)
	}
	if parseGot4 := parseExtractJSONObject(""); parseGot4 != `{"memories":[]}` {
		parseT.Fatalf("extractJSONObject(empty) = %q, want empty memories object", parseGot4)
	}

	parseUnavailable := ParseNewOpenAIProvider("", parseTestOpenAICatalog())
	if _, parseErr := parseUnavailable.ParseGenerateTitle(context.Background(), TitleRequest{Prompt: "hello"}); parseErr != ErrNoProvidersAvailable {
		parseT.Fatalf("GenerateTitle() error = %v, want ErrNoProvidersAvailable", parseErr)
	}
	if _, parseErr2 := parseUnavailable.ParseExtractUserMemories(context.Background(), MemoryExtractionRequest{UserMessage: "remember"}); parseErr2 != ErrNoProvidersAvailable {
		parseT.Fatalf("ExtractUserMemories() error = %v, want ErrNoProvidersAvailable", parseErr2)
	}
	if _, parseErr3 := parseUnavailable.ParseStreamChat(context.Background(), ChatRequest{UserMessage: "hello"}, func(ChatEvent) error { return nil }); parseErr3 != ErrNoProvidersAvailable {
		parseT.Fatalf("StreamChat() error = %v, want ErrNoProvidersAvailable", parseErr3)
	}
	if _, parseErr4 := parseProvider.ParseSynthesizeSpeech(context.Background(), SpeechRequest{Model: "unknown-model", Text: "hello"}, func(SpeechChunk) error { return nil }); parseErr4 == nil || !strings.Contains(parseErr4.ParseError(), `does not support speech`) {
		parseT.Fatalf("SynthesizeSpeech() error = %v, want unsupported speech capability error", parseErr4)
	}
}

func TestAnthropicProviderAdditionalMetadataAndHelpers(parseT *testing.T) {
	parseProvider := ParseNewAnthropicProvider("test-key", parseTestAnthropicCatalog())
	if parseInfo := parseProvider.ParseInfo(); parseInfo.ParseID != "anthropic" || parseInfo.BaseURL != anthropicBaseURL || !parseInfo.ParseAvailable {
		parseT.Fatalf("Info() = %+v, want available anthropic metadata", parseInfo)
	}
	if parseHealth := parseProvider.ParseHealth(); parseHealth.ProviderID != "anthropic" || parseHealth.ParseStatus != ProviderHealthUnknown {
		parseT.Fatalf("Health() = %+v, want unknown available health", parseHealth)
	}
	if parseLimits := parseProvider.ParseCurrentRateLimits(); !parseLimits.ParseEmpty() {
		parseT.Fatalf("CurrentRateLimits() = %+v, want empty snapshot", parseLimits)
	}
	if parseMetadata, parseOk := parseProvider.ParseModelMetadata(" Claude-Sonnet-4-5 "); !parseOk || parseMetadata.ParseID != "claude-sonnet-4-5" {
		parseT.Fatalf("ModelMetadata() = %+v, %t; want claude-sonnet-4-5", parseMetadata, parseOk)
	}
	if parseFallback := parseProvider.parseMustModelMetadata(" custom-claude "); parseFallback.ParseID != "custom-claude" || parseFallback.ProviderID != "anthropic" {
		parseT.Fatalf("mustModelMetadata() = %+v, want fallback metadata", parseFallback)
	}
	if parseGot := parseNormalizeAnthropicModel(" Claude-SONNET-4-5 "); parseGot != "claude-sonnet-4-5" {
		parseT.Fatalf("normalizeAnthropicModel() = %q, want claude-sonnet-4-5", parseGot)
	}
	if parseSystem := parseAnthropicSystemPrompt("  system  "); len(parseSystem) != 1 || parseSystem[0].Text != "system" {
		parseT.Fatalf("anthropicSystemPrompt() = %+v, want trimmed system prompt", parseSystem)
	}
	if parseSystem2 := parseAnthropicSystemPrompt("   "); parseSystem2 != nil {
		parseT.Fatalf("anthropicSystemPrompt(blank) = %+v, want nil", parseSystem2)
	}
	if parseMessages := parseAnthropicMessages([]ChatMessage{{Role: "assistant", Content: "done"}}, "hello"); len(parseMessages) != 2 {
		parseT.Fatalf("anthropicMessages() len = %d, want 2", len(parseMessages))
	}
	if parseText := parseAnthropicMessageText(nil); parseText != "" {
		parseT.Fatalf("anthropicMessageText(nil) = %q, want empty string", parseText)
	}

	parseUnavailable := ParseNewAnthropicProvider("", parseTestAnthropicCatalog())
	if _, parseErr := parseUnavailable.ParseGenerateTitle(context.Background(), TitleRequest{Prompt: "hello"}); parseErr != ErrNoProvidersAvailable {
		parseT.Fatalf("GenerateTitle() error = %v, want ErrNoProvidersAvailable", parseErr)
	}
	if _, parseErr2 := parseUnavailable.ParseExtractUserMemories(context.Background(), MemoryExtractionRequest{UserMessage: "remember"}); parseErr2 != ErrNoProvidersAvailable {
		parseT.Fatalf("ExtractUserMemories() error = %v, want ErrNoProvidersAvailable", parseErr2)
	}
	if _, parseErr3 := parseUnavailable.ParseStreamChat(context.Background(), ChatRequest{UserMessage: "hello"}, func(ChatEvent) error { return nil }); parseErr3 != ErrNoProvidersAvailable {
		parseT.Fatalf("StreamChat() error = %v, want ErrNoProvidersAvailable", parseErr3)
	}
	if _, parseErr4 := parseProvider.ParseSynthesizeSpeech(context.Background(), SpeechRequest{Model: "claude-sonnet-4-5", Text: "hello"}, func(SpeechChunk) error { return nil }); parseErr4 == nil || !strings.Contains(parseErr4.ParseError(), `does not support speech`) {
		parseT.Fatalf("SynthesizeSpeech() error = %v, want unsupported speech capability error", parseErr4)
	}
}

func TestStubProviderAdditionalBranches(parseT *testing.T) {
	parseCatalog := parseTestOpenAICatalog()
	parseProvider := ParseNewStubProvider(" openai ", parseCatalog)
	if parseInfo := parseProvider.ParseInfo(); parseInfo.ParseID != "openai" || parseInfo.BaseURL != "stub://openai" || parseInfo.AuthConfigured || !parseInfo.ParseAvailable {
		parseT.Fatalf("Info() = %+v, want available stub provider metadata", parseInfo)
	}
	if parseHealth := parseProvider.ParseHealth(); parseHealth.ProviderID != "openai" || parseHealth.ParseStatus != ProviderHealthUnknown {
		parseT.Fatalf("Health() = %+v, want unknown available health", parseHealth)
	}
	if parseLimits := parseProvider.ParseCurrentRateLimits(); !parseLimits.ParseEmpty() {
		parseT.Fatalf("CurrentRateLimits() = %+v, want empty snapshot", parseLimits)
	}
	if parseMetadata, parseOk := parseProvider.ParseModelMetadata("gpt-5.4-mini"); !parseOk || parseMetadata.ParseID != "gpt-5.4-mini" {
		parseT.Fatalf("ModelMetadata() = %+v, %t; want gpt-5.4-mini", parseMetadata, parseOk)
	}
	if parseGot, parseErr := parseProvider.ParseGenerateTitle(context.Background(), TitleRequest{Prompt: strings.Repeat("word ", 20)}); parseErr != nil || !strings.Contains(parseGot, "OpenAI stub:") {
		parseT.Fatalf("GenerateTitle() = %q, %v; want stub title", parseGot, parseErr)
	}
	if parseGot2, parseErr2 := parseProvider.ParseGenerateTitle(context.Background(), TitleRequest{}); parseErr2 != nil || !strings.Contains(parseGot2, "stub conversation") {
		parseT.Fatalf("GenerateTitle(blank) = %q, %v; want default stub title", parseGot2, parseErr2)
	}
	if parseMemories, parseErr3 := parseProvider.ParseExtractUserMemories(context.Background(), MemoryExtractionRequest{}); parseErr3 != nil || parseMemories != nil {
		parseT.Fatalf("ExtractUserMemories() = %+v, %v; want nil,nil", parseMemories, parseErr3)
	}

	parseChunks := make([]SpeechChunk, 0, 2)
	parseResult, parseErr4 := parseProvider.ParseSynthesizeSpeech(context.Background(), SpeechRequest{Text: "hello"}, func(parseChunk SpeechChunk) error {
		parseChunks = append(parseChunks, parseChunk)
		return nil
	})
	if parseErr4 != nil {
		parseT.Fatalf("SynthesizeSpeech() error = %v", parseErr4)
	}
	if parseResult.MimeType != "audio/mpeg" || len(parseChunks) != 2 || len(parseChunks[0].AudioChunk) == 0 || !parseChunks[1].Done {
		parseT.Fatalf("SynthesizeSpeech() = %+v chunks=%+v, want first audio chunk and final done chunk", parseResult, parseChunks)
	}

	if parseGot3 := parseStubProviderLabel(" custom "); parseGot3 != "CUSTOM" {
		parseT.Fatalf("stubProviderLabel() = %q, want CUSTOM", parseGot3)
	}
	if parseGot4 := parseNormalizeStubThinkingEffort("odd"); parseGot4 != "medium" {
		parseT.Fatalf("normalizeStubThinkingEffort() = %q, want medium", parseGot4)
	}
	if parseUnavailable := ParseNewStubProvider("", parseCatalog); parseUnavailable.ParseAvailable() || parseUnavailable.ParseInfo().ParseID != "" || parseUnavailable.ParseHealth().ParseStatus != ProviderHealthUnavailable {
		parseT.Fatalf("blank stub provider = %+v info=%+v health=%+v, want unavailable provider", parseUnavailable, parseUnavailable.ParseInfo(), parseUnavailable.ParseHealth())
	}
}
