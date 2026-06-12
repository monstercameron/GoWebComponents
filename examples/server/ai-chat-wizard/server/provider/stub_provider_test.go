package provider

import (
	"context"
	"strings"
	"testing"
)

func TestStubProviderSupportsCatalogAndStreaming(parseT *testing.T) {
	parseCatalog := Catalog{
		DefaultModel: "claude-sonnet-4-5",
		Models: []ModelMetadata{{
			ID:            "claude-sonnet-4-5",
			DisplayName:   "Claude Sonnet 4.5",
			ProviderID:    "anthropic",
			ProviderLabel: "Anthropic",
			Capabilities: ModelCapabilities{
				ProviderID:       "anthropic",
				ProviderLabel:    "Anthropic",
				SupportsThinking: true,
				SupportsSpeech:   true,
			},
		}},
	}
	parseProvider := ParseNewStubProvider("anthropic", parseCatalog)
	if !parseProvider.ParseAvailable() {
		parseT.Fatal("expected stub provider to be available")
	}
	if parseProvider.ParseDefaultModel() != "claude-sonnet-4-5" {
		parseT.Fatalf("unexpected default model: %q", parseProvider.ParseDefaultModel())
	}
	if !parseProvider.ParseSupportsModel("claude-sonnet-4-5") {
		parseT.Fatal("expected stub provider to support catalog model")
	}

	parseT.Setenv("CHAT_STUB_CHUNK_DELAY_MS", "1")
	parseEvents := []ChatEvent{}
	parseResult, parseErr := parseProvider.ParseStreamChat(context.Background(), ChatRequest{
		Model:           "claude-sonnet-4-5",
		UserMessage:     "Switch providers without restarting",
		ThinkingEnabled: true,
		ThinkingEffort:  "high",
	}, func(parseEvent ChatEvent) error {
		parseEvents = append(parseEvents, parseEvent)
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("StreamChat: %v", parseErr)
	}
	if parseResult.Model != "claude-sonnet-4-5" {
		parseT.Fatalf("unexpected result: %+v", parseResult)
	}
	if parseResult.UsageSource != UsageSourceEstimated {
		parseT.Fatalf("expected estimated usage source for stub provider, got %+v", parseResult)
	}
	// Contract: thought, thought-done, then a paced sequence of text deltas
	// whose concatenation is the full reply (streaming UIs need >1 delta).
	if len(parseEvents) < 4 {
		parseT.Fatalf("expected thought, thought done, and multiple reply deltas, got %+v", parseEvents)
	}
	if !strings.Contains(parseEvents[0].ThoughtDelta, "Anthropic stub reasoning") {
		parseT.Fatalf("unexpected thought event: %+v", parseEvents[0])
	}
	if !parseEvents[1].ThoughtDone {
		parseT.Fatalf("expected thought completion event, got %+v", parseEvents[1])
	}
	parseReply := strings.Builder{}
	for _, parseEvent := range parseEvents[2:] {
		if parseEvent.ThoughtDelta != "" || parseEvent.ThoughtDone {
			parseT.Fatalf("unexpected thought event after completion: %+v", parseEvent)
		}
		parseReply.WriteString(parseEvent.TextDelta)
	}
	if !strings.Contains(parseReply.String(), "Anthropic stub reply from claude-sonnet-4-5") {
		parseT.Fatalf("unexpected concatenated reply: %q", parseReply.String())
	}
}

// TestStubProviderSingleShotWhenDelayDisabled pins the CHAT_STUB_CHUNK_DELAY_MS=0
// escape hatch: the reply collapses back to one TextDelta.
func TestStubProviderSingleShotWhenDelayDisabled(parseT *testing.T) {
	parseT.Setenv("CHAT_STUB_CHUNK_DELAY_MS", "0")
	parseProvider := ParseNewStubProvider("openai", Catalog{})
	parseEvents := []ChatEvent{}
	if _, parseErr := parseProvider.ParseStreamChat(context.Background(), ChatRequest{
		UserMessage: "hello",
	}, func(parseEvent ChatEvent) error {
		parseEvents = append(parseEvents, parseEvent)
		return nil
	}); parseErr != nil {
		parseT.Fatalf("StreamChat: %v", parseErr)
	}
	if len(parseEvents) != 1 || !strings.Contains(parseEvents[0].TextDelta, "OpenAI stub reply") {
		parseT.Fatalf("expected one single-shot reply delta, got %+v", parseEvents)
	}
}
