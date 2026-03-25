package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

const cerebrasBaseURL = "https://api.cerebras.ai/v1"
const cerebrasMaxCompletionTokens int64 = 4096

type CerebrasProvider struct {
	client  *openai.Client
	catalog Catalog
}

func NewCerebrasProvider(apiKey string, catalog Catalog) *CerebrasProvider {
	trimmedAPIKey := strings.TrimSpace(apiKey)
	resolvedCatalog := normalizeCatalog("cerebras", "Cerebras", catalog)
	if trimmedAPIKey == "" {
		return &CerebrasProvider{catalog: resolvedCatalog}
	}
	client := openai.NewClient(
		option.WithAPIKey(trimmedAPIKey),
		option.WithBaseURL(cerebrasBaseURL),
	)
	return &CerebrasProvider{client: &client, catalog: resolvedCatalog}
}

func (p *CerebrasProvider) ID() string {
	return "cerebras"
}

func (p *CerebrasProvider) Available() bool {
	return p != nil && p.client != nil && len(p.catalog.Options) > 0
}

func (p *CerebrasProvider) Info() ProviderInfo {
	return ProviderInfo{
		ID:                 p.ID(),
		Label:              "Cerebras",
		BaseURL:            cerebrasBaseURL,
		AuthConfigured:     p.Available(),
		Available:          p.Available(),
		StreamingSupported: true,
		ReasoningSupported: true,
		ToolUseSupported:   false,
	}
}

func (p *CerebrasProvider) DefaultModel() string {
	return strings.TrimSpace(p.catalog.DefaultModel)
}

func (p *CerebrasProvider) SupportsModel(model string) bool {
	return p.catalog.SupportsModel(model)
}

func (p *CerebrasProvider) ModelOptions() []ModelOption {
	return p.catalog.ModelOptions()
}

func (p *CerebrasProvider) ModelMetadata(model string) (ModelMetadata, bool) {
	return p.catalog.ModelMetadata(model)
}

func (p *CerebrasProvider) Capabilities(model string) ModelCapabilities {
	if metadata, ok := p.catalog.ModelMetadata(model); ok {
		return metadata.Capabilities
	}
	return ModelCapabilities{ProviderID: p.ID(), ProviderLabel: "Cerebras"}
}

func (p *CerebrasProvider) Health() ProviderHealth {
	status := ProviderHealthUnavailable
	if p.Available() {
		status = ProviderHealthUnknown
	}
	return ProviderHealth{ProviderID: p.ID(), Status: status}
}

func (p *CerebrasProvider) CurrentRateLimits() RateLimitSnapshot {
	return RateLimitSnapshot{}
}

func (p *CerebrasProvider) GenerateTitle(ctx context.Context, req TitleRequest) (string, error) {
	if !p.Available() {
		return "", ErrNoProvidersAvailable
	}

	response, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: shared.ChatModel(p.catalog.TitleModel),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(strings.TrimSpace(req.SystemPrompt)),
			openai.UserMessage(strings.TrimSpace(req.Prompt)),
		},
		MaxCompletionTokens: openai.Int(64),
	})
	if err != nil {
		return "", fmt.Errorf("cerebras title: %w", err)
	}
	if len(response.Choices) == 0 {
		return "", errors.New("cerebras title: empty response")
	}
	title := strings.TrimSpace(response.Choices[0].Message.Content)
	if title == "" {
		return "", errors.New("cerebras title: empty title")
	}
	return title, nil
}

func (p *CerebrasProvider) ExtractUserMemories(ctx context.Context, req MemoryExtractionRequest) ([]UserMemoryCandidate, error) {
	_ = ctx
	_ = req
	if !p.Available() {
		return nil, ErrNoProvidersAvailable
	}
	return nil, errors.New("cerebras memory extraction is not implemented")
}

func (p *CerebrasProvider) StreamChat(ctx context.Context, req ChatRequest, emit func(ChatEvent) error) (ChatResult, error) {
	if !p.Available() {
		return ChatResult{}, ErrNoProvidersAvailable
	}

	resolvedModel := strings.TrimSpace(req.Model)
	if resolvedModel == "" {
		resolvedModel = p.DefaultModel()
	}

	params := openai.ChatCompletionNewParams{
		Model:               shared.ChatModel(resolvedModel),
		Messages:            cerebrasChatMessages(req.SystemPrompt, req.History, req.UserMessage),
		MaxCompletionTokens: openai.Int(cerebrasMaxCompletionTokens),
		StreamOptions: openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		},
	}
		if req.ThinkingEnabled && p.Capabilities(resolvedModel).SupportsThinking {
		params.ReasoningEffort = cerebrasReasoningEffort(req.ThinkingEffort)
	}

	stream := p.client.Chat.Completions.NewStreaming(ctx, params)
	thoughtStarted := false
	thoughtDoneSent := false
	var promptTokens int64
	var completionTokens int64

	emitThoughtDone := func() error {
		if thoughtDoneSent {
			return nil
		}
		thoughtDoneSent = true
		return emit(ChatEvent{ThoughtDone: true})
	}

	for stream.Next() {
		chunk := stream.Current()
		if chunk.Usage.CompletionTokens > 0 || chunk.Usage.PromptTokens > 0 {
			promptTokens = chunk.Usage.PromptTokens
			completionTokens = chunk.Usage.CompletionTokens
		}
		reasoningDelta := cerebrasReasoningDelta(chunk.RawJSON())
		if reasoningDelta != "" {
			thoughtStarted = true
			if err := emit(ChatEvent{ThoughtDelta: reasoningDelta}); err != nil {
				return ChatResult{}, err
			}
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content == "" {
				continue
			}
			if thoughtStarted && !thoughtDoneSent {
				if err := emitThoughtDone(); err != nil {
					return ChatResult{}, err
				}
			}
			if err := emit(ChatEvent{TextDelta: choice.Delta.Content}); err != nil {
				return ChatResult{}, err
			}
		}
	}
	if err := stream.Err(); err != nil {
		return ChatResult{}, fmt.Errorf("cerebras stream: %w", err)
	}
	if thoughtStarted && !thoughtDoneSent {
		if err := emitThoughtDone(); err != nil {
			return ChatResult{}, err
		}
	}

	return ChatResult{
		Model:            resolvedModel,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
	}, nil
}

func (p *CerebrasProvider) SynthesizeSpeech(ctx context.Context, req SpeechRequest, emit func(SpeechChunk) error) (SpeechResult, error) {
	_ = ctx
	_ = req
	_ = emit
	if !p.Available() {
		return SpeechResult{}, ErrNoProvidersAvailable
	}
	return SpeechResult{}, &UnsupportedCapabilityError{Capability: CapabilitySpeech, Model: strings.TrimSpace(req.Model), ProviderID: p.ID()}
}

func cerebrasChatMessages(systemPrompt string, history []ChatMessage, userMessage string) []openai.ChatCompletionMessageParamUnion {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(history)+2)
	if trimmedPrompt := strings.TrimSpace(systemPrompt); trimmedPrompt != "" {
		messages = append(messages, openai.SystemMessage(trimmedPrompt))
	}
	for _, historyMessage := range history {
		switch NormalizeRole(historyMessage.Role) {
		case "assistant":
			messages = append(messages, openai.AssistantMessage(strings.TrimSpace(historyMessage.Content)))
		case "developer":
			messages = append(messages, openai.DeveloperMessage(strings.TrimSpace(historyMessage.Content)))
		case "system":
			messages = append(messages, openai.SystemMessage(strings.TrimSpace(historyMessage.Content)))
		case "tool":
			messages = append(messages, openai.ToolMessage(strings.TrimSpace(historyMessage.Content), "tool"))
		default:
			messages = append(messages, openai.UserMessage(strings.TrimSpace(historyMessage.Content)))
		}
	}
	messages = append(messages, openai.UserMessage(strings.TrimSpace(userMessage)))
	return messages
}

func cerebrasReasoningDelta(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var chunk struct {
		Choices []struct {
			Delta struct {
				Reasoning        string `json:"reasoning"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(raw), &chunk); err != nil {
		return ""
	}
	for _, choice := range chunk.Choices {
		if choice.Delta.Reasoning != "" {
			return choice.Delta.Reasoning
		}
		if choice.Delta.ReasoningContent != "" {
			return choice.Delta.ReasoningContent
		}
	}
	return ""
}

func cerebrasReasoningEffort(effort string) shared.ReasoningEffort {
	switch strings.TrimSpace(strings.ToLower(effort)) {
	case "low":
		return shared.ReasoningEffortLow
	case "high":
		return shared.ReasoningEffortHigh
	default:
		return shared.ReasoningEffortMedium
	}
}

func normalizeCerebrasModel(model string) string {
	return strings.TrimSpace(strings.ToLower(model))
}

func (p *CerebrasProvider) mustModelMetadata(model string) ModelMetadata {
	metadata, ok := p.ModelMetadata(model)
	if !ok {
		return ModelMetadata{
			ID:            strings.TrimSpace(model),
			DisplayName:   strings.TrimSpace(model),
			ProviderID:    p.ID(),
			ProviderLabel: "Cerebras",
			Capabilities:  p.Capabilities(model),
		}
	}
	return metadata
}
