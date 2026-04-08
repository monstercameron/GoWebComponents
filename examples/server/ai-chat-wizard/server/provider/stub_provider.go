package provider

import (
	"context"
	"fmt"
	"strings"
)

type StubProvider struct {
	id      string
	label   string
	catalog Catalog
}

// ParseNewStubProvider creates one stub provider for local testing.
func ParseNewStubProvider(parseProviderID string, parseCatalog Catalog) *StubProvider {
	parseTrimmedID := strings.TrimSpace(strings.ToLower(parseProviderID))
	if parseTrimmedID == "" {
		return &StubProvider{}
	}
	return &StubProvider{
		id:      parseTrimmedID,
		label:   parseStubProviderLabel(parseTrimmedID),
		catalog: parseNormalizeCatalog(parseTrimmedID, parseStubProviderLabel(parseTrimmedID), parseCatalog),
	}
}

// ParseID returns the provider identifier.
func (parseP *StubProvider) ParseID() string {
	return parseP.id
}

// ParseAvailable reports whether the provider is available.
func (parseP *StubProvider) ParseAvailable() bool {
	return parseP != nil && parseP.id != "" && len(parseP.catalog.Options) > 0
}

// ParseInfo returns the provider info snapshot.
func (parseP *StubProvider) ParseInfo() ProviderInfo {
	return ProviderInfo{
		ID:                 parseP.id,
		Label:              parseP.label,
		BaseURL:            "stub://" + parseP.id,
		AuthConfigured:     false,
		Available:          parseP.ParseAvailable(),
		StreamingSupported: true,
		ReasoningSupported: true,
		ToolUseSupported:   false,
		Notes:              []string{"Local development stub provider"},
	}
}

// ParseDefaultModel returns the default model for the provider.
func (parseP *StubProvider) ParseDefaultModel() string {
	return strings.TrimSpace(parseP.catalog.DefaultModel)
}

// ParseSupportsModel reports whether the provider supports the requested model.
func (parseP *StubProvider) ParseSupportsModel(parseModel string) bool {
	return parseP.catalog.ParseSupportsModel(parseModel)
}

// ParseModelOptions returns the provider model options.
func (parseP *StubProvider) ParseModelOptions() []ModelOption {
	return parseP.catalog.ParseModelOptions()
}

// ParseModelMetadata returns the provider model metadata.
func (parseP *StubProvider) ParseModelMetadata(parseModel string) (ModelMetadata, bool) {
	return parseP.catalog.ParseModelMetadata(parseModel)
}

// ParseCapabilities returns the capability snapshot.
func (parseP *StubProvider) ParseCapabilities(parseModel string) ModelCapabilities {
	if parseMetadata, parseOk := parseP.catalog.ParseModelMetadata(parseModel); parseOk {
		return parseMetadata.Capabilities
	}
	return ModelCapabilities{ProviderID: parseP.id, ProviderLabel: parseP.label}
}

// ParseHealth returns the health snapshot.
func (parseP *StubProvider) ParseHealth() ProviderHealth {
	if !parseP.ParseAvailable() {
		return ProviderHealth{ProviderID: parseP.id, Status: ProviderHealthUnavailable}
	}
	return ProviderHealth{ProviderID: parseP.id, Status: ProviderHealthUnknown}
}

// ParseCurrentRateLimits returns the current rate-limit snapshot.
func (parseP *StubProvider) ParseCurrentRateLimits() RateLimitSnapshot {
	return RateLimitSnapshot{}
}

// ParseGenerateTitle generates one conversation title.
func (parseP *StubProvider) ParseGenerateTitle(_ context.Context, parseReq TitleRequest) (string, error) {
	parseUserPrompt := strings.TrimSpace(parseReq.Prompt)
	if parseUserPrompt == "" {
		return fmt.Sprintf("%s stub conversation", parseP.label), nil
	}
	parseTitle := parseUserPrompt
	if len(parseTitle) > 48 {
		parseTitle = strings.TrimSpace(parseTitle[:48]) + "..."
	}
	return fmt.Sprintf("%s stub: %s", parseP.label, parseTitle), nil
}

// ParseExtractUserMemories extracts user-memory candidates from the current conversation.
func (parseP *StubProvider) ParseExtractUserMemories(_ context.Context, _ MemoryExtractionRequest) ([]UserMemoryCandidate, error) {
	return nil, nil
}

// ParseStreamChat streams one chat completion.
func (parseP *StubProvider) ParseStreamChat(_ context.Context, parseReq ChatRequest, parseEmit func(ChatEvent) error) (ChatResult, error) {
	parseModel := strings.TrimSpace(parseReq.Model)
	if parseModel == "" {
		parseModel = parseP.ParseDefaultModel()
	}
	if parseReq.ThinkingEnabled {
		if parseErr := parseEmit(ChatEvent{ThoughtDelta: fmt.Sprintf("%s stub reasoning in %s mode.", parseP.label, parseNormalizeStubThinkingEffort(parseReq.ThinkingEffort))}); parseErr != nil {
			return ChatResult{}, parseErr
		}
		if parseErr2 := parseEmit(ChatEvent{ThoughtDone: true}); parseErr2 != nil {
			return ChatResult{}, parseErr2
		}
	}
	parseResponse := fmt.Sprintf("%s stub reply from %s to %q", parseP.label, parseModel, strings.TrimSpace(parseReq.UserMessage))
	if parseErr3 := parseEmit(ChatEvent{TextDelta: parseResponse}); parseErr3 != nil {
		return ChatResult{}, parseErr3
	}
	return ChatResult{
		Model:            parseModel,
		PromptTokens:     int64(len(strings.Fields(parseReq.UserMessage))) * 8,
		CompletionTokens: int64(len(strings.Fields(parseResponse))) * 6,
		UsageSource:      UsageSourceEstimated,
	}, nil
}

// ParseSynthesizeSpeech streams one speech-synthesis response.
func (parseP *StubProvider) ParseSynthesizeSpeech(_ context.Context, parseReq SpeechRequest, parseEmit func(SpeechChunk) error) (SpeechResult, error) {
	parseModel := strings.TrimSpace(parseReq.Model)
	if parseModel == "" {
		parseModel = parseP.ParseDefaultModel()
	}
	parseMimeType := "audio/mpeg"
	parseChunk := SpeechChunk{
		AudioChunk: []byte("stub-audio"),
		MimeType:   parseMimeType,
		Model:      parseModel,
		Voice:      "stub",
		Script:     parseReq.Text,
	}
	if parseErr := parseEmit(parseChunk); parseErr != nil {
		return SpeechResult{}, parseErr
	}
	parseDone := parseChunk
	parseDone.AudioChunk = nil
	parseDone.Done = true
	if parseErr2 := parseEmit(parseDone); parseErr2 != nil {
		return SpeechResult{}, parseErr2
	}
	return SpeechResult{MimeType: parseMimeType, Model: parseModel, Voice: "stub", Script: parseReq.Text}, nil
}

func parseStubProviderLabel(parseProviderID string) string {
	switch strings.TrimSpace(strings.ToLower(parseProviderID)) {
	case "openai":
		return "OpenAI"
	case "anthropic":
		return "Anthropic"
	case "cerebras":
		return "Cerebras"
	default:
		return strings.ToUpper(strings.TrimSpace(parseProviderID))
	}
}

func parseNormalizeStubThinkingEffort(parseValue string) string {
	switch strings.TrimSpace(strings.ToLower(parseValue)) {
	case "low", "medium", "high":
		return strings.TrimSpace(strings.ToLower(parseValue))
	default:
		return "medium"
	}
}
