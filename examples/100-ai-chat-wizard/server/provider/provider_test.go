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

func (parseP *stubProvider) ParseID() string { return parseP.id }

func (parseP *stubProvider) ParseAvailable() bool { return parseP.available }

func (parseP *stubProvider) ParseInfo() ProviderInfo {
	if parseP.info.ParseID == "" {
		return ProviderInfo{ID: parseP.id, Label: parseP.id, Available: parseP.available, AuthConfigured: parseP.available}
	}
	return parseP.info
}

func (parseP *stubProvider) ParseDefaultModel() string { return parseP.defaultModel }

func (parseP *stubProvider) ParseSupportsModel(parseModel string) bool {
	_, parseOk := parseP.models[strings.TrimSpace(parseModel)]
	return parseOk
}

func (parseP *stubProvider) ParseModelOptions() []ModelOption { return parseP.options }

func (parseP *stubProvider) ParseModelMetadata(parseModel string) (ModelMetadata, bool) {
	parseMetadata, parseOk := parseP.parseMetadata[strings.TrimSpace(parseModel)]
	return parseMetadata, parseOk
}

func (parseP *stubProvider) ParseCapabilities(parseModel string) ModelCapabilities {
	return parseP.models[strings.TrimSpace(parseModel)]
}

func (parseP *stubProvider) ParseHealth() ProviderHealth {
	return ProviderHealth{ProviderID: parseP.id, Status: ProviderHealthUnknown}
}

func (parseP *stubProvider) ParseCurrentRateLimits() RateLimitSnapshot {
	return RateLimitSnapshot{}
}

func (parseP *stubProvider) ParseStreamChat(context.Context, ChatRequest, func(ChatEvent) error) (ChatResult, error) {
	return ChatResult{}, nil
}

func (parseP *stubProvider) ParseGenerateTitle(context.Context, TitleRequest) (string, error) {
	return "", nil
}

func (parseP *stubProvider) ParseExtractUserMemories(context.Context, MemoryExtractionRequest) ([]UserMemoryCandidate, error) {
	return nil, nil
}

func (parseP *stubProvider) ParseSynthesizeSpeech(context.Context, SpeechRequest, func(SpeechChunk) error) (SpeechResult, error) {
	return SpeechResult{}, nil
}

func TestRegistryResolveAndCapabilityChecks(parseT *testing.T) {
	parseCapabilities := ModelCapabilities{ProviderID: "stub", ProviderLabel: "Stub", SupportsThinking: true, SupportsSpeech: false}
	parseProvider := &stubProvider{
		id:           "stub",
		available:    true,
		defaultModel: "stub-default",
		models:       map[string]ModelCapabilities{"model-a": parseCapabilities},
		options:      []ModelOption{{ID: "model-a", Label: "Model A", Capabilities: parseCapabilities}},
		metadata: map[string]ModelMetadata{
			"model-a": {
				ID:            "model-a",
				DisplayName:   "Model A",
				ProviderID:    "stub",
				ProviderLabel: "Stub",
				Capabilities:  parseCapabilities,
			},
		},
	}
	parseRegistry := ParseNewRegistry(parseProvider)

	parseResolvedProvider, parseResolvedModel, parseErr := parseRegistry.ParseResolve("model-a")
	if parseErr != nil || parseResolvedProvider.ParseID() != "stub" || parseResolvedModel != "model-a" {
		parseT.Fatalf("Resolve returned provider=%v model=%q err=%v", parseResolvedProvider, parseResolvedModel, parseErr)
	}

	parseDefaultProvider, parseDefaultModel, parseErr := parseRegistry.ParseResolve("")
	if parseErr != nil || parseDefaultProvider.ParseID() != "stub" || parseDefaultModel != "stub-default" {
		parseT.Fatalf("Resolve default returned provider=%v model=%q err=%v", parseDefaultProvider, parseDefaultModel, parseErr)
	}

	_, _, parseCapabilityState, parseErr := parseRegistry.ParseRequireCapability("model-a", CapabilitySpeech)
	var parseUnsupportedErr *UnsupportedCapabilityError
	if !errors.As(parseErr, &parseUnsupportedErr) {
		parseT.Fatalf("expected UnsupportedCapabilityError, got %v", parseErr)
	}
	if parseCapabilityState.SupportsSpeech {
		parseT.Fatal("expected speech capability to be false")
	}

	if _, _, parseErr2 := parseRegistry.ParseResolve("missing"); parseErr2 == nil {
		parseT.Fatal("expected Resolve to reject unsupported model")
	}
	if _, _, parseErr3 := (*Registry)(nil).ParseResolve(""); !errors.Is(parseErr3, ErrNoProvidersAvailable) {
		parseT.Fatalf("expected ErrNoProvidersAvailable for nil registry, got %v", parseErr3)
	}
}

func TestConversationInputAndRoleNormalization(parseT *testing.T) {
	if parseGot := ParseNormalizeRole(" Developer "); parseGot != "developer" {
		parseT.Fatalf("NormalizeRole mismatch: %q", parseGot)
	}
	if parseGot2 := ParseNormalizeRole("unknown"); parseGot2 != "user" {
		parseT.Fatalf("expected unknown role to normalize to user, got %q", parseGot2)
	}

	parseConversation := BuildConversationInput([]ChatMessage{{Role: "assistant", Content: " Hello "}}, " Latest ")
	if !strings.Contains(parseConversation, "assistant:\nHello") {
		parseT.Fatalf("conversation input missing normalized history: %q", parseConversation)
	}
	if !strings.HasSuffix(parseConversation, "user:\nLatest") {
		parseT.Fatalf("conversation input missing latest user turn: %q", parseConversation)
	}

	var parseNilErr *UnsupportedCapabilityError
	if parseNilErr.ParseError() != "unsupported capability" {
		parseT.Fatalf("unexpected nil UnsupportedCapabilityError string: %q", parseNilErr.ParseError())
	}
	parseWithModel := (&UnsupportedCapabilityError{Capability: CapabilityThinking, Model: "model-a", ProviderID: "stub"}).ParseError()
	if !strings.Contains(parseWithModel, "model \"model-a\" does not support thinking") {
		parseT.Fatalf("unexpected UnsupportedCapabilityError text: %q", parseWithModel)
	}
}
