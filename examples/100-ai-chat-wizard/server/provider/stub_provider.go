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

func NewStubProvider(providerID string, catalog Catalog) *StubProvider {
	trimmedID := strings.TrimSpace(strings.ToLower(providerID))
	if trimmedID == "" {
		return &StubProvider{}
	}
	return &StubProvider{
		id:      trimmedID,
		label:   stubProviderLabel(trimmedID),
		catalog: normalizeCatalog(trimmedID, stubProviderLabel(trimmedID), catalog),
	}
}

func (p *StubProvider) ID() string {
	return p.id
}

func (p *StubProvider) Available() bool {
	return p != nil && p.id != "" && len(p.catalog.Options) > 0
}

func (p *StubProvider) Info() ProviderInfo {
	return ProviderInfo{
		ID:                 p.id,
		Label:              p.label,
		BaseURL:            "stub://" + p.id,
		AuthConfigured:     false,
		Available:          p.Available(),
		StreamingSupported: true,
		ReasoningSupported: true,
		ToolUseSupported:   false,
		Notes:              []string{"Local development stub provider"},
	}
}

func (p *StubProvider) DefaultModel() string {
	return strings.TrimSpace(p.catalog.DefaultModel)
}

func (p *StubProvider) SupportsModel(model string) bool {
	return p.catalog.SupportsModel(model)
}

func (p *StubProvider) ModelOptions() []ModelOption {
	return p.catalog.ModelOptions()
}

func (p *StubProvider) ModelMetadata(model string) (ModelMetadata, bool) {
	return p.catalog.ModelMetadata(model)
}

func (p *StubProvider) Capabilities(model string) ModelCapabilities {
	if metadata, ok := p.catalog.ModelMetadata(model); ok {
		return metadata.Capabilities
	}
	return ModelCapabilities{ProviderID: p.id, ProviderLabel: p.label}
}

func (p *StubProvider) Health() ProviderHealth {
	if !p.Available() {
		return ProviderHealth{ProviderID: p.id, Status: ProviderHealthUnavailable}
	}
	return ProviderHealth{ProviderID: p.id, Status: ProviderHealthUnknown}
}

func (p *StubProvider) CurrentRateLimits() RateLimitSnapshot {
	return RateLimitSnapshot{}
}

func (p *StubProvider) GenerateTitle(_ context.Context, req TitleRequest) (string, error) {
	userPrompt := strings.TrimSpace(req.Prompt)
	if userPrompt == "" {
		return fmt.Sprintf("%s stub conversation", p.label), nil
	}
	title := userPrompt
	if len(title) > 48 {
		title = strings.TrimSpace(title[:48]) + "…"
	}
	return fmt.Sprintf("%s stub: %s", p.label, title), nil
}

func (p *StubProvider) ExtractUserMemories(_ context.Context, _ MemoryExtractionRequest) ([]UserMemoryCandidate, error) {
	return nil, nil
}

func (p *StubProvider) StreamChat(_ context.Context, req ChatRequest, emit func(ChatEvent) error) (ChatResult, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = p.DefaultModel()
	}
	if req.ThinkingEnabled {
		if err := emit(ChatEvent{ThoughtDelta: fmt.Sprintf("%s stub reasoning in %s mode.", p.label, normalizeStubThinkingEffort(req.ThinkingEffort))}); err != nil {
			return ChatResult{}, err
		}
		if err := emit(ChatEvent{ThoughtDone: true}); err != nil {
			return ChatResult{}, err
		}
	}
	response := fmt.Sprintf("%s stub reply from %s to %q", p.label, model, strings.TrimSpace(req.UserMessage))
	if err := emit(ChatEvent{TextDelta: response}); err != nil {
		return ChatResult{}, err
	}
	return ChatResult{
		Model:            model,
		PromptTokens:     int64(len(strings.Fields(req.UserMessage))) * 8,
		CompletionTokens: int64(len(strings.Fields(response))) * 6,
	}, nil
}

func (p *StubProvider) SynthesizeSpeech(_ context.Context, req SpeechRequest, emit func(SpeechChunk) error) (SpeechResult, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = p.DefaultModel()
	}
	mimeType := "audio/mpeg"
	chunk := SpeechChunk{
		AudioChunk: []byte("stub-audio"),
		MimeType:   mimeType,
		Model:      model,
		Voice:      "stub",
		Script:     req.Text,
	}
	if err := emit(chunk); err != nil {
		return SpeechResult{}, err
	}
	done := chunk
	done.AudioChunk = nil
	done.Done = true
	if err := emit(done); err != nil {
		return SpeechResult{}, err
	}
	return SpeechResult{MimeType: mimeType, Model: model, Voice: "stub", Script: req.Text}, nil
}

func stubProviderLabel(providerID string) string {
	switch strings.TrimSpace(strings.ToLower(providerID)) {
	case "openai":
		return "OpenAI"
	case "anthropic":
		return "Anthropic"
	case "cerebras":
		return "Cerebras"
	default:
		return strings.ToUpper(strings.TrimSpace(providerID))
	}
}

func normalizeStubThinkingEffort(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "low", "medium", "high":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "medium"
	}
}
