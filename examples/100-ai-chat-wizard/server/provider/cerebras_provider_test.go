package provider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/openai/openai-go/shared"
)

func TestCerebrasProviderNonNetworkHelpers(parseT *testing.T) {
	parseProvider := ParseNewCerebrasProvider("test-key", parseTestCerebrasCatalog())
	if !parseProvider.ParseAvailable() {
		parseT.Fatal("expected cerebras provider with test key to be available")
	}
	if parseProvider.ParseID() != "cerebras" {
		parseT.Fatalf("unexpected provider ID: %q", parseProvider.ParseID())
	}
	if parseProvider.ParseDefaultModel() != "gpt-oss-120b" {
		parseT.Fatalf("unexpected default model: %q", parseProvider.ParseDefaultModel())
	}
	if !parseProvider.ParseSupportsModel("gpt-oss-120b") || parseProvider.ParseSupportsModel("gpt-5.4-mini") {
		parseT.Fatal("unexpected SupportsModel behavior for Cerebras provider")
	}
	if len(parseProvider.ParseModelOptions()) != 4 {
		parseT.Fatalf("expected four Cerebras model options, got %d", len(parseProvider.ParseModelOptions()))
	}
	if parseCaps := parseProvider.ParseCapabilities("gpt-oss-120b"); !parseCaps.SupportsThinking || parseCaps.SupportsSpeech {
		parseT.Fatalf("unexpected default Cerebras capabilities: %+v", parseCaps)
	}
	if parseCaps2 := parseProvider.ParseCapabilities("llama3.1-8b"); parseCaps2.SupportsThinking || parseCaps2.SupportsSpeech {
		parseT.Fatalf("unexpected fast Cerebras capabilities: %+v", parseCaps2)
	}
	if parseGot := parseCerebrasReasoningEffort("HIGH"); parseGot != shared.ReasoningEffortHigh {
		parseT.Fatalf("unexpected reasoning effort normalization: %q", parseGot)
	}
	if parseGot2 := parseCerebrasReasoningEffort("weird"); parseGot2 != shared.ReasoningEffortMedium {
		parseT.Fatalf("expected medium reasoning effort fallback, got %q", parseGot2)
	}
	if parseDelta := parseCerebrasReasoningDelta(`{"choices":[{"delta":{"reasoning":"step 1"}}]}`); parseDelta != "step 1" {
		parseT.Fatalf("unexpected reasoning delta parse: %q", parseDelta)
	}
	if parseDelta2 := parseCerebrasReasoningDelta(`{"choices":[{"delta":{"reasoning_content":"step 2"}}]}`); parseDelta2 != "step 2" {
		parseT.Fatalf("unexpected reasoning_content delta parse: %q", parseDelta2)
	}
	if parseDelta3 := parseCerebrasReasoningDelta(`{}`); parseDelta3 != "" {
		parseT.Fatalf("expected empty reasoning delta, got %q", parseDelta3)
	}
}

func TestCerebrasProviderMetadataAndUnavailableBranches(parseT *testing.T) {
	parseProvider := ParseNewCerebrasProvider("test-key", parseTestCerebrasCatalog())
	if parseInfo := parseProvider.ParseInfo(); parseInfo.ParseID != "cerebras" || parseInfo.Label != "Cerebras" || !parseInfo.ParseAvailable || !parseInfo.AuthConfigured || parseInfo.BaseURL != cerebrasBaseURL {
		parseT.Fatalf("unexpected Cerebras info: %+v", parseInfo)
	}
	if parseHealth := parseProvider.ParseHealth(); parseHealth.ProviderID != "cerebras" || parseHealth.ParseStatus != ProviderHealthUnknown {
		parseT.Fatalf("unexpected Cerebras health: %+v", parseHealth)
	}
	if parseLimits := parseProvider.ParseCurrentRateLimits(); !parseLimits.ParseEmpty() {
		parseT.Fatalf("expected Cerebras rate limits to be empty, got %+v", parseLimits)
	}
	if parseMetadata, parseOk := parseProvider.ParseModelMetadata(" GPT-OSS-120B "); !parseOk || parseMetadata.ParseID != "gpt-oss-120b" {
		parseT.Fatalf("unexpected Cerebras metadata resolution: ok=%v metadata=%+v", parseOk, parseMetadata)
	}
	if parseMetadata2, parseOk2 := parseProvider.ParseModelMetadata("unsupported-model"); parseOk2 || parseMetadata2.ParseID != "" {
		parseT.Fatalf("expected unknown Cerebras model metadata to be unavailable, got ok=%v metadata=%+v", parseOk2, parseMetadata2)
	}
	if parseFallback := parseProvider.parseMustModelMetadata(" custom-cerebras "); parseFallback.ParseID != "custom-cerebras" || parseFallback.DisplayName != "custom-cerebras" || parseFallback.ProviderID != "cerebras" {
		parseT.Fatalf("unexpected Cerebras fallback metadata: %+v", parseFallback)
	}

	parseUnavailable := ParseNewCerebrasProvider("", parseTestCerebrasCatalog())
	if parseUnavailable.ParseAvailable() {
		parseT.Fatal("expected provider without key to be unavailable")
	}
	if _, parseErr := parseUnavailable.ParseGenerateTitle(context.Background(), TitleRequest{Prompt: "hello"}); parseErr != ErrNoProvidersAvailable {
		parseT.Fatalf("expected unavailable GenerateTitle to return ErrNoProvidersAvailable, got %v", parseErr)
	}
	if _, parseErr2 := parseUnavailable.ParseExtractUserMemories(context.Background(), MemoryExtractionRequest{UserMessage: "remember"}); parseErr2 != ErrNoProvidersAvailable {
		parseT.Fatalf("expected unavailable ExtractUserMemories to return ErrNoProvidersAvailable, got %v", parseErr2)
	}
	if _, parseErr3 := parseUnavailable.ParseStreamChat(context.Background(), ChatRequest{UserMessage: "hello"}, func(ChatEvent) error { return nil }); parseErr3 != ErrNoProvidersAvailable {
		parseT.Fatalf("expected unavailable StreamChat to return ErrNoProvidersAvailable, got %v", parseErr3)
	}
	if _, parseErr4 := parseUnavailable.ParseSynthesizeSpeech(context.Background(), SpeechRequest{Model: "gpt-oss-120b", Text: "hello"}, func(SpeechChunk) error { return nil }); parseErr4 != ErrNoProvidersAvailable {
		parseT.Fatalf("expected unavailable SynthesizeSpeech to return ErrNoProvidersAvailable, got %v", parseErr4)
	}
}

func TestCerebrasMessageMappingAndModelNormalization(parseT *testing.T) {
	parseHistory := []ChatMessage{
		{Role: "assistant", Content: " assistant message "},
		{Role: "developer", Content: " developer message "},
		{Role: "system", Content: " system message "},
		{Role: "tool", Content: " tool message "},
		{Role: "user", Content: " user message "},
		{Role: "unknown", Content: " unknown role message "},
	}

	parseMessages := parseCerebrasChatMessages(" system prompt ", parseHistory, " final user message ")
	if len(parseMessages) != 8 {
		parseT.Fatalf("expected 8 chat messages (system + history + user), got %d", len(parseMessages))
	}

	parseExpectedRoles := []string{"system", "assistant", "developer", "system", "tool", "user", "user", "user"}
	for parseIndex, parseExpectedRole := range parseExpectedRoles {
		parsePayload, parseErr := json.Marshal(parseMessages[parseIndex])
		if parseErr != nil {
			parseT.Fatalf("marshal chat message %d: %v", parseIndex, parseErr)
		}
		if !strings.Contains(string(parsePayload), `"role":"`+parseExpectedRole+`"`) {
			parseT.Fatalf("expected role %q at index %d, got payload %s", parseExpectedRole, parseIndex, string(parsePayload))
		}
	}

	parseToolCallID := parseMessages[4].GetToolCallID()
	if parseToolCallID == nil || *parseToolCallID != "tool" {
		parseT.Fatalf("expected tool-call ID \"tool\", got %v", parseToolCallID)
	}

	parsePayload2, parseErr2 := json.Marshal(parseMessages)
	if parseErr2 != nil {
		parseT.Fatalf("marshal mapped chat messages: %v", parseErr2)
	}
	parseBody := string(parsePayload2)
	for _, parseSnippet := range []string{
		`"content":"system prompt"`,
		`"content":"assistant message"`,
		`"content":"developer message"`,
		`"content":"system message"`,
		`"content":"tool message"`,
		`"content":"user message"`,
		`"content":"unknown role message"`,
		`"content":"final user message"`,
	} {
		if !strings.Contains(parseBody, parseSnippet) {
			parseT.Fatalf("expected mapped message payload to contain %q, got %s", parseSnippet, parseBody)
		}
	}

	if parseGot := parseCerebrasReasoningEffort("LOW"); parseGot != shared.ReasoningEffortLow {
		parseT.Fatalf("unexpected low reasoning effort normalization: %q", parseGot)
	}
	if parseGot2 := parseNormalizeCerebrasModel(" GPT-OSS-120B "); parseGot2 != "gpt-oss-120b" {
		parseT.Fatalf("unexpected cerebras model normalization: %q", parseGot2)
	}

	parseProvider := ParseNewCerebrasProvider("test-key", parseTestCerebrasCatalog())
	if parseMetadata := parseProvider.parseMustModelMetadata("gpt-oss-120b"); parseMetadata.ParseID != "gpt-oss-120b" || parseMetadata.ProviderID != "cerebras" {
		parseT.Fatalf("expected known model metadata lookup path, got %+v", parseMetadata)
	}
}
