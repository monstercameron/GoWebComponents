package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	anthropicoption "github.com/anthropics/anthropic-sdk-go/option"
)

const anthropicMaxTokens int64 = 4096
const anthropicBaseURL = "https://api.anthropic.com/v1"

type AnthropicProvider struct {
	client  *anthropic.Client
	catalog Catalog
}

func ParseNewAnthropicProvider(parseApiKey string, parseCatalog Catalog) *AnthropicProvider {
	parseTrimmedAPIKey := strings.TrimSpace(parseApiKey)
	parseResolvedCatalog := parseNormalizeCatalog("anthropic", "Anthropic", parseCatalog)
	if parseTrimmedAPIKey == "" {
		return &AnthropicProvider{catalog: parseResolvedCatalog}
	}
	parseClient := anthropic.NewClient(anthropicoption.WithAPIKey(parseTrimmedAPIKey))
	return &AnthropicProvider{client: &parseClient, catalog: parseResolvedCatalog}
}

func (parseP *AnthropicProvider) ParseID() string {
	return "anthropic"
}

func (parseP *AnthropicProvider) ParseAvailable() bool {
	return parseP != nil && parseP.client != nil && len(parseP.catalog.Options) > 0
}

func (parseP *AnthropicProvider) ParseInfo() ProviderInfo {
	return ProviderInfo{
		ID:                 parseP.ParseID(),
		Label:              "Anthropic",
		BaseURL:            anthropicBaseURL,
		AuthConfigured:     parseP.ParseAvailable(),
		Available:          parseP.ParseAvailable(),
		StreamingSupported: true,
		ReasoningSupported: true,
		ToolUseSupported:   true,
	}
}

func (parseP *AnthropicProvider) ParseDefaultModel() string {
	return strings.TrimSpace(parseP.catalog.DefaultModel)
}

func (parseP *AnthropicProvider) ParseSupportsModel(parseModel string) bool {
	return parseP.catalog.ParseSupportsModel(parseModel)
}

func (parseP *AnthropicProvider) ParseModelOptions() []ModelOption {
	return parseP.catalog.ParseModelOptions()
}

func (parseP *AnthropicProvider) ParseModelMetadata(parseModel string) (ModelMetadata, bool) {
	return parseP.catalog.ParseModelMetadata(parseModel)
}

func (parseP *AnthropicProvider) ParseCapabilities(parseModel string) ModelCapabilities {
	if parseMetadata, parseOk := parseP.catalog.ParseModelMetadata(parseModel); parseOk {
		return parseMetadata.Capabilities
	}
	return ModelCapabilities{ProviderID: parseP.ParseID(), ProviderLabel: "Anthropic"}
}

func (parseP *AnthropicProvider) ParseHealth() ProviderHealth {
	parseStatus := ProviderHealthUnavailable
	if parseP.ParseAvailable() {
		parseStatus = ProviderHealthUnknown
	}
	return ProviderHealth{ProviderID: parseP.ParseID(), Status: parseStatus}
}

func (parseP *AnthropicProvider) ParseCurrentRateLimits() RateLimitSnapshot {
	return RateLimitSnapshot{}
}

func (parseP *AnthropicProvider) ParseGenerateTitle(parseCtx context.Context, parseReq TitleRequest) (string, error) {
	if !parseP.ParseAvailable() {
		return "", ErrNoProvidersAvailable
	}

	parseMessage, parseErr := parseP.client.Messages.New(parseCtx, anthropic.MessageNewParams{
		MaxTokens: 64,
		Messages: []anthropic.MessageParam{
			parseAnthropicTextMessage("user", parseReq.Prompt),
		},
		Model:  anthropic.Model(parseP.catalog.TitleModel),
		System: parseAnthropicSystemPrompt(parseReq.SystemPrompt),
	})
	if parseErr != nil {
		return "", fmt.Errorf("anthropic title: %w", parseErr)
	}

	parseTitle := strings.TrimSpace(parseAnthropicMessageText(parseMessage))
	if parseTitle == "" {
		return "", errors.New("anthropic title: empty title")
	}
	return parseTitle, nil
}

func (parseP *AnthropicProvider) ParseExtractUserMemories(parseCtx context.Context, parseReq MemoryExtractionRequest) ([]UserMemoryCandidate, error) {
	_ = parseCtx
	_ = parseReq
	if !parseP.ParseAvailable() {
		return nil, ErrNoProvidersAvailable
	}
	return nil, errors.New("anthropic memory extraction is not implemented")
}

func (parseP *AnthropicProvider) ParseStreamChat(parseCtx context.Context, parseReq ChatRequest, parseEmit func(ChatEvent) error) (ChatResult, error) {
	if !parseP.ParseAvailable() {
		return ChatResult{}, ErrNoProvidersAvailable
	}

	parseResolvedModel := strings.TrimSpace(parseReq.Model)
	if parseResolvedModel == "" {
		parseResolvedModel = parseP.ParseDefaultModel()
	}

	buildParams := func(isIncludeThinking bool) anthropic.MessageNewParams {
		parseParams := anthropic.MessageNewParams{
			MaxTokens: anthropicMaxTokens,
			Messages:  parseAnthropicMessages(parseReq.History, parseReq.UserMessage),
			Model:     anthropic.Model(parseResolvedModel),
			System:    parseAnthropicSystemPrompt(parseReq.SystemPrompt),
		}
		if isIncludeThinking {
			parseParams.Thinking = anthropic.ThinkingConfigParamOfEnabled(parseAnthropicThinkingBudget(parseReq.ThinkingEffort))
		}
		return parseParams
	}

	parseMessage := anthropic.Message{}
	parseStream := parseP.client.Messages.NewStreaming(parseCtx, buildParams(parseReq.ThinkingEnabled))
	isParseRetriedWithoutThinking := false
	isParseThoughtStarted := false
	isParseThoughtDoneSent := false

	parseEmitThoughtDone := func() error {
		if isParseThoughtDoneSent {
			return nil
		}
		isParseThoughtDoneSent = true
		return parseEmit(ChatEvent{ThoughtDone: true})
	}

	var parseProcessStream func() error
	parseProcessStream = func() error {
		for parseStream.Next() {
			parseEvent := parseStream.Current()
			if parseErr := parseMessage.Accumulate(parseEvent); parseErr != nil {
				return fmt.Errorf("anthropic accumulate: %w", parseErr)
			}

			switch parseCurrentEvent := parseEvent.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch parseDelta := parseCurrentEvent.Delta.AsAny().(type) {
				case anthropic.ThinkingDelta:
					if parseDelta.Thinking == "" {
						continue
					}
					isParseThoughtStarted = true
					if parseErr2 := parseEmit(ChatEvent{ThoughtDelta: parseDelta.Thinking}); parseErr2 != nil {
						return parseErr2
					}
				case anthropic.TextDelta:
					if parseDelta.Text == "" {
						continue
					}
					if isParseThoughtStarted && !isParseThoughtDoneSent {
						if parseErr3 := parseEmitThoughtDone(); parseErr3 != nil {
							return parseErr3
						}
					}
					if parseErr4 := parseEmit(ChatEvent{TextDelta: parseDelta.Text}); parseErr4 != nil {
						return parseErr4
					}
				}
			}
		}

		if parseErr5 := parseStream.Err(); parseErr5 != nil {
			if parseReq.ThinkingEnabled && !isParseRetriedWithoutThinking && parseAnthropicThinkingUnsupported(parseErr5) {
				isParseRetriedWithoutThinking = true
				if !isParseThoughtStarted {
					if parseSendErr := parseEmit(ChatEvent{ThoughtDelta: "Extended thinking is unavailable for this Claude configuration, so continuing without live thought output."}); parseSendErr != nil {
						return parseSendErr
					}
					isParseThoughtStarted = true
				}
				if parseSendErr2 := parseEmitThoughtDone(); parseSendErr2 != nil {
					return parseSendErr2
				}
				parseMessage = anthropic.Message{}
				parseStream = parseP.client.Messages.NewStreaming(parseCtx, buildParams(false))
				return parseProcessStream()
			}
			return fmt.Errorf("anthropic stream: %w", parseErr5)
		}

		return nil
	}

	if parseErr6 := parseProcessStream(); parseErr6 != nil {
		return ChatResult{}, parseErr6
	}
	if isParseThoughtStarted && !isParseThoughtDoneSent {
		if parseErr7 := parseEmitThoughtDone(); parseErr7 != nil {
			return ChatResult{}, parseErr7
		}
	}

	return ChatResult{
		Model:            parseResolvedModel,
		PromptTokens:     parseMessage.Usage.InputTokens,
		CompletionTokens: parseMessage.Usage.OutputTokens,
	}, nil
}

func (parseP *AnthropicProvider) ParseSynthesizeSpeech(parseCtx context.Context, parseReq SpeechRequest, parseEmit func(SpeechChunk) error) (SpeechResult, error) {
	_ = parseCtx
	_ = parseReq
	_ = parseEmit
	if !parseP.ParseAvailable() {
		return SpeechResult{}, ErrNoProvidersAvailable
	}
	return SpeechResult{}, &UnsupportedCapabilityError{Capability: CapabilitySpeech, Model: strings.TrimSpace(parseReq.Model), ProviderID: parseP.ParseID()}
}

func parseAnthropicTextMessage(parseRole, parseContent string) anthropic.MessageParam {
	return anthropic.MessageParam{
		Role: anthropic.MessageParamRole(ParseNormalizeRole(parseRole)),
		Content: []anthropic.ContentBlockParamUnion{
			{
				OfText: &anthropic.TextBlockParam{Text: strings.TrimSpace(parseContent)},
			},
		},
	}
}

func parseAnthropicMessages(parseHistory []ChatMessage, parseUserMessage string) []anthropic.MessageParam {
	parseResolvedMessages := make([]anthropic.MessageParam, 0, len(parseHistory)+1)
	for _, parseHistoryMessage := range parseHistory {
		parseResolvedMessages = append(parseResolvedMessages, parseAnthropicTextMessage(parseHistoryMessage.Role, parseHistoryMessage.Content))
	}
	parseResolvedMessages = append(parseResolvedMessages, parseAnthropicTextMessage("user", parseUserMessage))
	return parseResolvedMessages
}

func parseAnthropicSystemPrompt(parsePrompt string) []anthropic.TextBlockParam {
	parseTrimmedPrompt := strings.TrimSpace(parsePrompt)
	if parseTrimmedPrompt == "" {
		return nil
	}
	return []anthropic.TextBlockParam{{Text: parseTrimmedPrompt}}
}

func parseAnthropicMessageText(parseMessage *anthropic.Message) string {
	if parseMessage == nil {
		return ""
	}

	var parseBuilder strings.Builder
	for _, parseContentBlock := range parseMessage.Content {
		switch parseBlock := parseContentBlock.AsAny().(type) {
		case anthropic.TextBlock:
			parseBuilder.WriteString(parseBlock.Text)
		}
	}
	return parseBuilder.String()
}

func parseAnthropicThinkingBudget(parseEffort string) int64 {
	switch strings.TrimSpace(strings.ToLower(parseEffort)) {
	case "low":
		return 1024
	case "high":
		return 4096
	default:
		return 2048
	}
}

func parseAnthropicThinkingUnsupported(parseErr error) bool {
	if parseErr == nil {
		return false
	}
	parseResolvedError := strings.ToLower(parseErr.Error())
	return strings.Contains(parseResolvedError, "thinking") && (strings.Contains(parseResolvedError, "unsupported") || strings.Contains(parseResolvedError, "not available") || strings.Contains(parseResolvedError, "invalid_request_error"))
}

func parseNormalizeAnthropicModel(parseModel string) string {
	return strings.TrimSpace(strings.ToLower(parseModel))
}

func (parseP *AnthropicProvider) parseMustModelMetadata(parseModel string) ModelMetadata {
	parseMetadata, parseOk := parseP.ParseModelMetadata(parseModel)
	if !parseOk {
		return ModelMetadata{
			ID:            strings.TrimSpace(parseModel),
			DisplayName:   strings.TrimSpace(parseModel),
			ProviderID:    parseP.ParseID(),
			ProviderLabel: "Anthropic",
			Capabilities:  parseP.ParseCapabilities(parseModel),
		}
	}
	return parseMetadata
}
