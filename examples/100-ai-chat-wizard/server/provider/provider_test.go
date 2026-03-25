package provider

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type stubProvider struct {
	id           string
	available    bool
	defaultModel string
	models       map[string]ModelCapabilities
	options      []ModelOption
	info         ProviderInfo
	metadata     map[string]ModelMetadata
}

func (p *stubProvider) ID() string { return p.id }

func (p *stubProvider) Available() bool { return p.available }

func (p *stubProvider) Info() ProviderInfo {
	if p.info.ID == "" {
		return ProviderInfo{ID: p.id, Label: p.id, Available: p.available, AuthConfigured: p.available}
	}
	return p.info
}

func (p *stubProvider) DefaultModel() string { return p.defaultModel }

func (p *stubProvider) SupportsModel(model string) bool {
	_, ok := p.models[strings.TrimSpace(model)]
	return ok
}

func (p *stubProvider) ModelOptions() []ModelOption { return p.options }

func (p *stubProvider) ModelMetadata(model string) (ModelMetadata, bool) {
	metadata, ok := p.metadata[strings.TrimSpace(model)]
	return metadata, ok
}

func (p *stubProvider) Capabilities(model string) ModelCapabilities {
	return p.models[strings.TrimSpace(model)]
}

func (p *stubProvider) Health() ProviderHealth {
	return ProviderHealth{ProviderID: p.id, Status: ProviderHealthUnknown}
}

func (p *stubProvider) CurrentRateLimits() RateLimitSnapshot {
	return RateLimitSnapshot{}
}

func (p *stubProvider) StreamChat(context.Context, ChatRequest, func(ChatEvent) error) (ChatResult, error) {
	return ChatResult{}, nil
}

func (p *stubProvider) GenerateTitle(context.Context, TitleRequest) (string, error) { return "", nil }

func (p *stubProvider) ExtractUserMemories(context.Context, MemoryExtractionRequest) ([]UserMemoryCandidate, error) {
	return nil, nil
}

func (p *stubProvider) SynthesizeSpeech(context.Context, SpeechRequest, func(SpeechChunk) error) (SpeechResult, error) {
	return SpeechResult{}, nil
}

func TestRegistryResolveAndCapabilityChecks(t *testing.T) {
	capabilities := ModelCapabilities{ProviderID: "stub", ProviderLabel: "Stub", SupportsThinking: true, SupportsSpeech: false}
	provider := &stubProvider{
		id:           "stub",
		available:    true,
		defaultModel: "stub-default",
		models:       map[string]ModelCapabilities{"model-a": capabilities},
		options:      []ModelOption{{ID: "model-a", Label: "Model A", Capabilities: capabilities}},
		metadata: map[string]ModelMetadata{
			"model-a": {
				ID:            "model-a",
				DisplayName:   "Model A",
				ProviderID:    "stub",
				ProviderLabel: "Stub",
				Capabilities:  capabilities,
			},
		},
	}
	registry := NewRegistry(provider)

	resolvedProvider, resolvedModel, err := registry.Resolve("model-a")
	if err != nil || resolvedProvider.ID() != "stub" || resolvedModel != "model-a" {
		t.Fatalf("Resolve returned provider=%v model=%q err=%v", resolvedProvider, resolvedModel, err)
	}

	defaultProvider, defaultModel, err := registry.Resolve("")
	if err != nil || defaultProvider.ID() != "stub" || defaultModel != "stub-default" {
		t.Fatalf("Resolve default returned provider=%v model=%q err=%v", defaultProvider, defaultModel, err)
	}

	_, _, capabilityState, err := registry.RequireCapability("model-a", CapabilitySpeech)
	var unsupportedErr *UnsupportedCapabilityError
	if !errors.As(err, &unsupportedErr) {
		t.Fatalf("expected UnsupportedCapabilityError, got %v", err)
	}
	if capabilityState.SupportsSpeech {
		t.Fatal("expected speech capability to be false")
	}

	if _, _, err := registry.Resolve("missing"); err == nil {
		t.Fatal("expected Resolve to reject unsupported model")
	}
	if _, _, err := (*Registry)(nil).Resolve(""); !errors.Is(err, ErrNoProvidersAvailable) {
		t.Fatalf("expected ErrNoProvidersAvailable for nil registry, got %v", err)
	}
}

func TestConversationInputAndRoleNormalization(t *testing.T) {
	if got := NormalizeRole(" Developer "); got != "developer" {
		t.Fatalf("NormalizeRole mismatch: %q", got)
	}
	if got := NormalizeRole("unknown"); got != "user" {
		t.Fatalf("expected unknown role to normalize to user, got %q", got)
	}

	conversation := BuildConversationInput([]ChatMessage{{Role: "assistant", Content: " Hello "}}, " Latest ")
	if !strings.Contains(conversation, "assistant:\nHello") {
		t.Fatalf("conversation input missing normalized history: %q", conversation)
	}
	if !strings.HasSuffix(conversation, "user:\nLatest") {
		t.Fatalf("conversation input missing latest user turn: %q", conversation)
	}

	var nilErr *UnsupportedCapabilityError
	if nilErr.Error() != "unsupported capability" {
		t.Fatalf("unexpected nil UnsupportedCapabilityError string: %q", nilErr.Error())
	}
	withModel := (&UnsupportedCapabilityError{Capability: CapabilityThinking, Model: "model-a", ProviderID: "stub"}).Error()
	if !strings.Contains(withModel, "model \"model-a\" does not support thinking") {
		t.Fatalf("unexpected UnsupportedCapabilityError text: %q", withModel)
	}
}
