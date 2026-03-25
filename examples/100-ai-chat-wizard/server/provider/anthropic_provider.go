package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	anthropicoption "github.com/anthropics/anthropic-sdk-go/option"
)

const anthropicDefaultModel = "claude-sonnet-4-5"
const anthropicTitleModel = "claude-haiku-4-5"
const anthropicMaxTokens int64 = 4096
const anthropicBaseURL = "https://api.anthropic.com/v1"

type AnthropicProvider struct {
	client *anthropic.Client
}

func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	trimmedAPIKey := strings.TrimSpace(apiKey)
	if trimmedAPIKey == "" {
		return &AnthropicProvider{}
	}
	client := anthropic.NewClient(anthropicoption.WithAPIKey(trimmedAPIKey))
	return &AnthropicProvider{client: &client}
}

func (p *AnthropicProvider) ID() string {
	return "anthropic"
}

func (p *AnthropicProvider) Available() bool {
	return p != nil && p.client != nil
}

func (p *AnthropicProvider) Info() ProviderInfo {
	return ProviderInfo{
		ID:                 p.ID(),
		Label:              "Anthropic",
		BaseURL:            anthropicBaseURL,
		AuthConfigured:     p.Available(),
		Available:          p.Available(),
		StreamingSupported: true,
		ReasoningSupported: true,
		ToolUseSupported:   true,
	}
}

func (p *AnthropicProvider) DefaultModel() string {
	return anthropicDefaultModel
}

func (p *AnthropicProvider) SupportsModel(model string) bool {
	resolvedModel := strings.ToLower(strings.TrimSpace(model))
	if resolvedModel == "" {
		return false
	}
	return strings.Contains(resolvedModel, "claude")
}

func (p *AnthropicProvider) ModelOptions() []ModelOption {
	return []ModelOption{
		ModelOptionFromMetadata(p.mustModelMetadata(anthropicDefaultModel), "Reasoning"),
		ModelOptionFromMetadata(p.mustModelMetadata(anthropicTitleModel), "Fast"),
	}
}

func (p *AnthropicProvider) ModelMetadata(model string) (ModelMetadata, bool) {
	switch normalizeAnthropicModel(model) {
	case anthropicDefaultModel:
		return ModelMetadata{
			ID:                 anthropicDefaultModel,
			DisplayName:        "Claude Sonnet 4.5",
			Description:        "Balanced Anthropic reasoning model for general chat and coding flows.",
			ProviderID:         p.ID(),
			ProviderLabel:      "Anthropic",
			ProviderFamily:     "anthropic",
			Capabilities:       p.Capabilities(anthropicDefaultModel),
			StreamingSupported: true,
			ReasoningSupported: true,
			ToolUseSupported:   true,
			OnboardingReady:    true,
		}, true
	case anthropicTitleModel:
		return ModelMetadata{
			ID:                 anthropicTitleModel,
			DisplayName:        "Claude Haiku 4.5",
			Description:        "Faster Anthropic model for lightweight generation tasks.",
			ProviderID:         p.ID(),
			ProviderLabel:      "Anthropic",
			ProviderFamily:     "anthropic",
			Capabilities:       p.Capabilities(anthropicTitleModel),
			StreamingSupported: true,
			ReasoningSupported: true,
			ToolUseSupported:   true,
			OnboardingReady:    true,
		}, true
	default:
		return ModelMetadata{}, false
	}
}

func (p *AnthropicProvider) Capabilities(model string) ModelCapabilities {
	return ModelCapabilities{
		ProviderID:       p.ID(),
		ProviderLabel:    "Anthropic",
		SupportsThinking: p.SupportsModel(model),
		SupportsSpeech:   false,
	}
}

func (p *AnthropicProvider) Health() ProviderHealth {
	status := ProviderHealthUnavailable
	if p.Available() {
		status = ProviderHealthUnknown
	}
	return ProviderHealth{ProviderID: p.ID(), Status: status}
}

func (p *AnthropicProvider) CurrentRateLimits() RateLimitSnapshot {
	return RateLimitSnapshot{}
}

func (p *AnthropicProvider) GenerateTitle(ctx context.Context, req TitleRequest) (string, error) {
	if !p.Available() {
		return "", ErrNoProvidersAvailable
	}

	message, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: 64,
		Messages: []anthropic.MessageParam{
			anthropicTextMessage("user", req.Prompt),
		},
		Model:  anthropic.Model(anthropicTitleModel),
		System: anthropicSystemPrompt(req.SystemPrompt),
	})
	if err != nil {
		return "", fmt.Errorf("anthropic title: %w", err)
	}

	title := strings.TrimSpace(anthropicMessageText(message))
	if title == "" {
		return "", errors.New("anthropic title: empty title")
	}
	return title, nil
}

func (p *AnthropicProvider) ExtractUserMemories(ctx context.Context, req MemoryExtractionRequest) ([]UserMemoryCandidate, error) {
	_ = ctx
	_ = req
	if !p.Available() {
		return nil, ErrNoProvidersAvailable
	}
	return nil, errors.New("anthropic memory extraction is not implemented")
}

func (p *AnthropicProvider) StreamChat(ctx context.Context, req ChatRequest, emit func(ChatEvent) error) (ChatResult, error) {
	if !p.Available() {
		return ChatResult{}, ErrNoProvidersAvailable
	}

	resolvedModel := strings.TrimSpace(req.Model)
	if resolvedModel == "" {
		resolvedModel = p.DefaultModel()
	}

	buildParams := func(includeThinking bool) anthropic.MessageNewParams {
		params := anthropic.MessageNewParams{
			MaxTokens: anthropicMaxTokens,
			Messages:  anthropicMessages(req.History, req.UserMessage),
			Model:     anthropic.Model(resolvedModel),
			System:    anthropicSystemPrompt(req.SystemPrompt),
		}
		if includeThinking {
			params.Thinking = anthropic.ThinkingConfigParamOfEnabled(anthropicThinkingBudget(req.ThinkingEffort))
		}
		return params
	}

	message := anthropic.Message{}
	stream := p.client.Messages.NewStreaming(ctx, buildParams(req.ThinkingEnabled))
	retriedWithoutThinking := false
	thoughtStarted := false
	thoughtDoneSent := false

	emitThoughtDone := func() error {
		if thoughtDoneSent {
			return nil
		}
		thoughtDoneSent = true
		return emit(ChatEvent{ThoughtDone: true})
	}

	var processStream func() error
	processStream = func() error {
		for stream.Next() {
			event := stream.Current()
			if err := message.Accumulate(event); err != nil {
				return fmt.Errorf("anthropic accumulate: %w", err)
			}

			switch currentEvent := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch delta := currentEvent.Delta.AsAny().(type) {
				case anthropic.ThinkingDelta:
					if delta.Thinking == "" {
						continue
					}
					thoughtStarted = true
					if err := emit(ChatEvent{ThoughtDelta: delta.Thinking}); err != nil {
						return err
					}
				case anthropic.TextDelta:
					if delta.Text == "" {
						continue
					}
					if thoughtStarted && !thoughtDoneSent {
						if err := emitThoughtDone(); err != nil {
							return err
						}
					}
					if err := emit(ChatEvent{TextDelta: delta.Text}); err != nil {
						return err
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			if req.ThinkingEnabled && !retriedWithoutThinking && anthropicThinkingUnsupported(err) {
				retriedWithoutThinking = true
				if !thoughtStarted {
					if sendErr := emit(ChatEvent{ThoughtDelta: "Extended thinking is unavailable for this Claude configuration, so continuing without live thought output."}); sendErr != nil {
						return sendErr
					}
					thoughtStarted = true
				}
				if sendErr := emitThoughtDone(); sendErr != nil {
					return sendErr
				}
				message = anthropic.Message{}
				stream = p.client.Messages.NewStreaming(ctx, buildParams(false))
				return processStream()
			}
			return fmt.Errorf("anthropic stream: %w", err)
		}

		return nil
	}

	if err := processStream(); err != nil {
		return ChatResult{}, err
	}
	if thoughtStarted && !thoughtDoneSent {
		if err := emitThoughtDone(); err != nil {
			return ChatResult{}, err
		}
	}

	return ChatResult{
		Model:            resolvedModel,
		PromptTokens:     message.Usage.InputTokens,
		CompletionTokens: message.Usage.OutputTokens,
	}, nil
}

func (p *AnthropicProvider) SynthesizeSpeech(ctx context.Context, req SpeechRequest, emit func(SpeechChunk) error) (SpeechResult, error) {
	_ = ctx
	_ = req
	_ = emit
	if !p.Available() {
		return SpeechResult{}, ErrNoProvidersAvailable
	}
	return SpeechResult{}, &UnsupportedCapabilityError{Capability: CapabilitySpeech, Model: strings.TrimSpace(req.Model), ProviderID: p.ID()}
}

func anthropicTextMessage(role, content string) anthropic.MessageParam {
	return anthropic.MessageParam{
		Role: anthropic.MessageParamRole(NormalizeRole(role)),
		Content: []anthropic.ContentBlockParamUnion{
			{
				OfText: &anthropic.TextBlockParam{Text: strings.TrimSpace(content)},
			},
		},
	}
}

func anthropicMessages(history []ChatMessage, userMessage string) []anthropic.MessageParam {
	resolvedMessages := make([]anthropic.MessageParam, 0, len(history)+1)
	for _, historyMessage := range history {
		resolvedMessages = append(resolvedMessages, anthropicTextMessage(historyMessage.Role, historyMessage.Content))
	}
	resolvedMessages = append(resolvedMessages, anthropicTextMessage("user", userMessage))
	return resolvedMessages
}

func anthropicSystemPrompt(prompt string) []anthropic.TextBlockParam {
	trimmedPrompt := strings.TrimSpace(prompt)
	if trimmedPrompt == "" {
		return nil
	}
	return []anthropic.TextBlockParam{{Text: trimmedPrompt}}
}

func anthropicMessageText(message *anthropic.Message) string {
	if message == nil {
		return ""
	}

	var builder strings.Builder
	for _, contentBlock := range message.Content {
		switch block := contentBlock.AsAny().(type) {
		case anthropic.TextBlock:
			builder.WriteString(block.Text)
		}
	}
	return builder.String()
}

func anthropicThinkingBudget(effort string) int64 {
	switch strings.TrimSpace(strings.ToLower(effort)) {
	case "low":
		return 1024
	case "high":
		return 4096
	default:
		return 2048
	}
}

func anthropicThinkingUnsupported(err error) bool {
	if err == nil {
		return false
	}
	resolvedError := strings.ToLower(err.Error())
	return strings.Contains(resolvedError, "thinking") && (strings.Contains(resolvedError, "unsupported") || strings.Contains(resolvedError, "not available") || strings.Contains(resolvedError, "invalid_request_error"))
}

func normalizeAnthropicModel(model string) string {
	return strings.TrimSpace(strings.ToLower(model))
}

func (p *AnthropicProvider) mustModelMetadata(model string) ModelMetadata {
	metadata, ok := p.ModelMetadata(model)
	if !ok {
		return ModelMetadata{
			ID:            strings.TrimSpace(model),
			DisplayName:   strings.TrimSpace(model),
			ProviderID:    p.ID(),
			ProviderLabel: "Anthropic",
			Capabilities:  p.Capabilities(model),
		}
	}
	return metadata
}
