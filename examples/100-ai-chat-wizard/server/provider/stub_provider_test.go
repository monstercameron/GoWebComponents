package provider

import (
	"context"
	"strings"
	"testing"
)

func TestStubProviderSupportsCatalogAndStreaming(t *testing.T) {
	catalog := Catalog{
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
	provider := NewStubProvider("anthropic", catalog)
	if !provider.Available() {
		t.Fatal("expected stub provider to be available")
	}
	if provider.DefaultModel() != "claude-sonnet-4-5" {
		t.Fatalf("unexpected default model: %q", provider.DefaultModel())
	}
	if !provider.SupportsModel("claude-sonnet-4-5") {
		t.Fatal("expected stub provider to support catalog model")
	}

	events := []ChatEvent{}
	result, err := provider.StreamChat(context.Background(), ChatRequest{
		Model:           "claude-sonnet-4-5",
		UserMessage:     "Switch providers without restarting",
		ThinkingEnabled: true,
		ThinkingEffort:  "high",
	}, func(event ChatEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}
	if result.Model != "claude-sonnet-4-5" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(events) != 3 {
		t.Fatalf("expected thought, thought done, and reply events, got %+v", events)
	}
	if !strings.Contains(events[0].ThoughtDelta, "Anthropic stub reasoning") {
		t.Fatalf("unexpected thought event: %+v", events[0])
	}
	if !events[1].ThoughtDone {
		t.Fatalf("expected thought completion event, got %+v", events[1])
	}
	if !strings.Contains(events[2].TextDelta, "Anthropic stub reply from claude-sonnet-4-5") {
		t.Fatalf("unexpected reply event: %+v", events[2])
	}
}
