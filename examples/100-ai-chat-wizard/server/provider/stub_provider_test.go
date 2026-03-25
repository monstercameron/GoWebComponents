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
	if len(parseEvents) != 3 {
		parseT.Fatalf("expected thought, thought done, and reply events, got %+v", parseEvents)
	}
	if !strings.Contains(parseEvents[0].ThoughtDelta, "Anthropic stub reasoning") {
		parseT.Fatalf("unexpected thought event: %+v", parseEvents[0])
	}
	if !parseEvents[1].ThoughtDone {
		parseT.Fatalf("expected thought completion event, got %+v", parseEvents[1])
	}
	if !strings.Contains(parseEvents[2].TextDelta, "Anthropic stub reply from claude-sonnet-4-5") {
		parseT.Fatalf("unexpected reply event: %+v", parseEvents[2])
	}
}
