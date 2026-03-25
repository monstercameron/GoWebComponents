package provider

import (
	"errors"
	"strings"
	"testing"
)

func TestProviderMetadataHealthAndErrorHelpers(t *testing.T) {
	openaiConfigured := NewOpenAIProvider("test-key", testOpenAICatalog())
	if info := openaiConfigured.Info(); info.ID != "openai" || info.Label != "OpenAI" || !info.Available || !info.AuthConfigured || info.BaseURL != openAIBaseURL {
		t.Fatalf("unexpected OpenAI info: %+v", info)
	}
	if health := openaiConfigured.Health(); health.ProviderID != "openai" || health.Status != ProviderHealthUnknown {
		t.Fatalf("unexpected OpenAI health: %+v", health)
	}
	if limits := openaiConfigured.CurrentRateLimits(); !limits.Empty() {
		t.Fatalf("expected OpenAI rate limits to be empty, got %+v", limits)
	}
	if metadata, ok := openaiConfigured.ModelMetadata(" GPT-5.4-NANO "); !ok || metadata.ID != "gpt-5.4-nano" {
		t.Fatalf("unexpected OpenAI metadata resolution: ok=%v metadata=%+v", ok, metadata)
	}
	if metadata, ok := openaiConfigured.ModelMetadata("unsupported-model"); ok || metadata.ID != "" {
		t.Fatalf("expected unknown OpenAI model metadata to be unavailable, got ok=%v metadata=%+v", ok, metadata)
	}
	if fallback := openaiConfigured.mustModelMetadata(" custom-model "); fallback.ID != "custom-model" || fallback.DisplayName != "custom-model" || fallback.ProviderID != "openai" {
		t.Fatalf("unexpected OpenAI fallback metadata: %+v", fallback)
	}

	openaiUnavailable := NewOpenAIProvider("", testOpenAICatalog())
	if info := openaiUnavailable.Info(); info.Available || info.AuthConfigured {
		t.Fatalf("expected unavailable OpenAI info to reflect missing auth, got %+v", info)
	}
	if health := openaiUnavailable.Health(); health.Status != ProviderHealthUnavailable {
		t.Fatalf("expected unavailable OpenAI health, got %+v", health)
	}

	anthropicConfigured := NewAnthropicProvider("test-key", testAnthropicCatalog())
	if info := anthropicConfigured.Info(); info.ID != "anthropic" || info.Label != "Anthropic" || !info.Available || !info.AuthConfigured || info.BaseURL != anthropicBaseURL {
		t.Fatalf("unexpected Anthropic info: %+v", info)
	}
	if health := anthropicConfigured.Health(); health.ProviderID != "anthropic" || health.Status != ProviderHealthUnknown {
		t.Fatalf("unexpected Anthropic health: %+v", health)
	}
	if limits := anthropicConfigured.CurrentRateLimits(); !limits.Empty() {
		t.Fatalf("expected Anthropic rate limits to be empty, got %+v", limits)
	}
	if metadata, ok := anthropicConfigured.ModelMetadata(" CLAUDE-HAIKU-4-5 "); !ok || metadata.ID != "claude-haiku-4-5" {
		t.Fatalf("unexpected Anthropic metadata resolution: ok=%v metadata=%+v", ok, metadata)
	}
	if metadata, ok := anthropicConfigured.ModelMetadata("unsupported-model"); ok || metadata.ID != "" {
		t.Fatalf("expected unknown Anthropic model metadata to be unavailable, got ok=%v metadata=%+v", ok, metadata)
	}
	if fallback := anthropicConfigured.mustModelMetadata(" custom-claude "); fallback.ID != "custom-claude" || fallback.DisplayName != "custom-claude" || fallback.ProviderID != "anthropic" {
		t.Fatalf("unexpected Anthropic fallback metadata: %+v", fallback)
	}

	anthropicUnavailable := NewAnthropicProvider("", testAnthropicCatalog())
	if info := anthropicUnavailable.Info(); info.Available || info.AuthConfigured {
		t.Fatalf("expected unavailable Anthropic info to reflect missing auth, got %+v", info)
	}
	if health := anthropicUnavailable.Health(); health.Status != ProviderHealthUnavailable {
		t.Fatalf("expected unavailable Anthropic health, got %+v", health)
	}

	withModel := (&UnsupportedCapabilityError{Capability: CapabilitySpeech, Model: "gpt-5.4-mini", ProviderID: "openai"}).Error()
	if !strings.Contains(withModel, "gpt-5.4-mini") || !strings.Contains(withModel, "speech") {
		t.Fatalf("unexpected unsupported capability error with model: %q", withModel)
	}
	withoutModel := (&UnsupportedCapabilityError{Capability: CapabilityThinking, ProviderID: "anthropic"}).Error()
	if !strings.Contains(withoutModel, "anthropic") || !strings.Contains(withoutModel, "thinking") {
		t.Fatalf("unexpected unsupported capability error without model: %q", withoutModel)
	}
	if got := (*UnsupportedCapabilityError)(nil).Error(); got != "unsupported capability" {
		t.Fatalf("unexpected nil unsupported capability error text: %q", got)
	}

	rootErr := errors.New("upstream exploded")
	normalized := &NormalizedError{
		Kind:       ErrorKindUpstream,
		ProviderID: "openai",
		Model:      "gpt-5.4",
		Message:    "request failed",
		Err:        rootErr,
	}
	if got := normalized.Error(); !strings.Contains(got, "openai") || !strings.Contains(got, "gpt-5.4") || !strings.Contains(got, "request failed") {
		t.Fatalf("unexpected normalized provider error text: %q", got)
	}
	if !errors.Is(normalized, rootErr) {
		t.Fatalf("expected NormalizedError to unwrap underlying error")
	}
	if got := (&NormalizedError{Message: "plain"}).Error(); got != "plain" {
		t.Fatalf("unexpected normalized provider error without provider: %q", got)
	}
	if got := (*NormalizedError)(nil).Error(); got != "provider error" {
		t.Fatalf("unexpected nil normalized provider error text: %q", got)
	}
	if err := (*NormalizedError)(nil).Unwrap(); err != nil {
		t.Fatalf("expected nil normalized provider unwrap to be nil, got %v", err)
	}
}
