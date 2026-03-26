package provider

import (
	"errors"
	"strings"
	"testing"
)

func TestProviderMetadataHealthAndErrorHelpers(parseT *testing.T) {
	parseOpenaiConfigured := ParseNewOpenAIProvider("test-key", parseTestOpenAICatalog())
	if parseInfo := parseOpenaiConfigured.ParseInfo(); parseInfo.ID != "openai" || parseInfo.Label != "OpenAI" || !parseInfo.Available || !parseInfo.AuthConfigured || parseInfo.BaseURL != openAIBaseURL {
		parseT.Fatalf("unexpected OpenAI info: %+v", parseInfo)
	}
	if parseHealth := parseOpenaiConfigured.ParseHealth(); parseHealth.ProviderID != "openai" || parseHealth.Status != ProviderHealthUnknown {
		parseT.Fatalf("unexpected OpenAI health: %+v", parseHealth)
	}
	if parseLimits := parseOpenaiConfigured.ParseCurrentRateLimits(); !parseLimits.ParseEmpty() {
		parseT.Fatalf("expected OpenAI rate limits to be empty, got %+v", parseLimits)
	}
	if parseMetadata, parseOk := parseOpenaiConfigured.ParseModelMetadata(" GPT-5.4-NANO "); !parseOk || parseMetadata.ID != "gpt-5.4-nano" {
		parseT.Fatalf("unexpected OpenAI metadata resolution: ok=%v metadata=%+v", parseOk, parseMetadata)
	}
	if parseMetadata2, parseOk2 := parseOpenaiConfigured.ParseModelMetadata("unsupported-model"); parseOk2 || parseMetadata2.ID != "" {
		parseT.Fatalf("expected unknown OpenAI model metadata to be unavailable, got ok=%v metadata=%+v", parseOk2, parseMetadata2)
	}
	if parseFallback := parseOpenaiConfigured.parseMustModelMetadata(" custom-model "); parseFallback.ID != "custom-model" || parseFallback.DisplayName != "custom-model" || parseFallback.ProviderID != "openai" {
		parseT.Fatalf("unexpected OpenAI fallback metadata: %+v", parseFallback)
	}

	parseOpenaiUnavailable := ParseNewOpenAIProvider("", parseTestOpenAICatalog())
	if parseInfo2 := parseOpenaiUnavailable.ParseInfo(); parseInfo2.Available || parseInfo2.AuthConfigured {
		parseT.Fatalf("expected unavailable OpenAI info to reflect missing auth, got %+v", parseInfo2)
	}
	if parseHealth2 := parseOpenaiUnavailable.ParseHealth(); parseHealth2.Status != ProviderHealthUnavailable {
		parseT.Fatalf("expected unavailable OpenAI health, got %+v", parseHealth2)
	}

	parseAnthropicConfigured := ParseNewAnthropicProvider("test-key", parseTestAnthropicCatalog())
	if parseInfo3 := parseAnthropicConfigured.ParseInfo(); parseInfo3.ID != "anthropic" || parseInfo3.Label != "Anthropic" || !parseInfo3.Available || !parseInfo3.AuthConfigured || parseInfo3.BaseURL != anthropicBaseURL {
		parseT.Fatalf("unexpected Anthropic info: %+v", parseInfo3)
	}
	if parseHealth3 := parseAnthropicConfigured.ParseHealth(); parseHealth3.ProviderID != "anthropic" || parseHealth3.Status != ProviderHealthUnknown {
		parseT.Fatalf("unexpected Anthropic health: %+v", parseHealth3)
	}
	if parseLimits2 := parseAnthropicConfigured.ParseCurrentRateLimits(); !parseLimits2.ParseEmpty() {
		parseT.Fatalf("expected Anthropic rate limits to be empty, got %+v", parseLimits2)
	}
	if parseMetadata3, parseOk3 := parseAnthropicConfigured.ParseModelMetadata(" CLAUDE-HAIKU-4-5 "); !parseOk3 || parseMetadata3.ID != "claude-haiku-4-5" {
		parseT.Fatalf("unexpected Anthropic metadata resolution: ok=%v metadata=%+v", parseOk3, parseMetadata3)
	}
	if parseMetadata4, parseOk4 := parseAnthropicConfigured.ParseModelMetadata("unsupported-model"); parseOk4 || parseMetadata4.ID != "" {
		parseT.Fatalf("expected unknown Anthropic model metadata to be unavailable, got ok=%v metadata=%+v", parseOk4, parseMetadata4)
	}
	if parseFallback2 := parseAnthropicConfigured.parseMustModelMetadata(" custom-claude "); parseFallback2.ID != "custom-claude" || parseFallback2.DisplayName != "custom-claude" || parseFallback2.ProviderID != "anthropic" {
		parseT.Fatalf("unexpected Anthropic fallback metadata: %+v", parseFallback2)
	}

	parseAnthropicUnavailable := ParseNewAnthropicProvider("", parseTestAnthropicCatalog())
	if parseInfo4 := parseAnthropicUnavailable.ParseInfo(); parseInfo4.Available || parseInfo4.AuthConfigured {
		parseT.Fatalf("expected unavailable Anthropic info to reflect missing auth, got %+v", parseInfo4)
	}
	if parseHealth4 := parseAnthropicUnavailable.ParseHealth(); parseHealth4.Status != ProviderHealthUnavailable {
		parseT.Fatalf("expected unavailable Anthropic health, got %+v", parseHealth4)
	}

	parseWithModel := (&UnsupportedCapabilityError{Capability: CapabilitySpeech, Model: "gpt-5.4-mini", ProviderID: "openai"}).Error()
	if !strings.Contains(parseWithModel, "gpt-5.4-mini") || !strings.Contains(parseWithModel, "speech") {
		parseT.Fatalf("unexpected unsupported capability error with model: %q", parseWithModel)
	}
	parseWithoutModel := (&UnsupportedCapabilityError{Capability: CapabilityThinking, ProviderID: "anthropic"}).Error()
	if !strings.Contains(parseWithoutModel, "anthropic") || !strings.Contains(parseWithoutModel, "thinking") {
		parseT.Fatalf("unexpected unsupported capability error without model: %q", parseWithoutModel)
	}
	if parseGot := (*UnsupportedCapabilityError)(nil).Error(); parseGot != "unsupported capability" {
		parseT.Fatalf("unexpected nil unsupported capability error text: %q", parseGot)
	}

	parseRootErr := errors.New("upstream exploded")
	parseNormalized := &NormalizedError{
		Kind:       ErrorKindUpstream,
		ProviderID: "openai",
		Model:      "gpt-5.4",
		Message:    "request failed",
		Err:        parseRootErr,
	}
	if parseGot2 := parseNormalized.Error(); !strings.Contains(parseGot2, "openai") || !strings.Contains(parseGot2, "gpt-5.4") || !strings.Contains(parseGot2, "request failed") {
		parseT.Fatalf("unexpected normalized provider error text: %q", parseGot2)
	}
	if !errors.Is(parseNormalized, parseRootErr) {
		parseT.Fatalf("expected NormalizedError to unwrap underlying error")
	}
	if parseGot3 := (&NormalizedError{Message: "plain"}).Error(); parseGot3 != "plain" {
		parseT.Fatalf("unexpected normalized provider error without provider: %q", parseGot3)
	}
	if parseGot4 := (*NormalizedError)(nil).Error(); parseGot4 != "provider error" {
		parseT.Fatalf("unexpected nil normalized provider error text: %q", parseGot4)
	}
	if parseErr := (*NormalizedError)(nil).ParseUnwrap(); parseErr != nil {
		parseT.Fatalf("expected nil normalized provider unwrap to be nil, got %v", parseErr)
	}
}
