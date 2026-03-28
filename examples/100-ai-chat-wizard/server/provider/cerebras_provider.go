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
const cerebrasMemoryExtractionMaxCompletionTokens int64 = 1024

type CerebrasProvider struct {
	client  *openai.Client
	catalog Catalog
}

// ParseNewCerebrasProvider creates a Cerebras provider wired to the supplied catalog and API key.
func ParseNewCerebrasProvider(parseApiKey string, parseCatalog Catalog) *CerebrasProvider {
	parseTrimmedAPIKey := strings.TrimSpace(parseApiKey)
	parseResolvedCatalog := parseNormalizeCatalog("cerebras", "Cerebras", parseCatalog)
	if parseTrimmedAPIKey == "" {
		return &CerebrasProvider{catalog: parseResolvedCatalog}
	}
	parseClient := openai.NewClient(
		option.WithAPIKey(parseTrimmedAPIKey),
		option.WithBaseURL(cerebrasBaseURL),
		option.WithMiddleware(parseBuildTraceabilityMiddleware()),
	)
	return &CerebrasProvider{client: &parseClient, catalog: parseResolvedCatalog}
}

// ParseID returns the provider identifier.
func (parseP *CerebrasProvider) ParseID() string {
	return "cerebras"
}

// ParseAvailable reports whether the provider is available.
func (parseP *CerebrasProvider) ParseAvailable() bool {
	return parseP != nil && parseP.client != nil && len(parseP.catalog.Options) > 0
}

// ParseInfo returns the provider info snapshot.
func (parseP *CerebrasProvider) ParseInfo() ProviderInfo {
	return ProviderInfo{
		ID:                 parseP.ParseID(),
		Label:              "Cerebras",
		BaseURL:            cerebrasBaseURL,
		AuthConfigured:     parseP.ParseAvailable(),
		Available:          parseP.ParseAvailable(),
		StreamingSupported: true,
		ReasoningSupported: true,
		ToolUseSupported:   false,
	}
}

// ParseDefaultModel returns the default model for the provider.
func (parseP *CerebrasProvider) ParseDefaultModel() string {
	return strings.TrimSpace(parseP.catalog.DefaultModel)
}

// ParseSupportsModel reports whether the provider supports the requested model.
func (parseP *CerebrasProvider) ParseSupportsModel(parseModel string) bool {
	return parseP.catalog.ParseSupportsModel(parseModel)
}

// ParseModelOptions returns the provider model options.
func (parseP *CerebrasProvider) ParseModelOptions() []ModelOption {
	return parseP.catalog.ParseModelOptions()
}

// ParseModelMetadata returns the provider model metadata.
func (parseP *CerebrasProvider) ParseModelMetadata(parseModel string) (ModelMetadata, bool) {
	return parseP.catalog.ParseModelMetadata(parseModel)
}

// ParseCapabilities returns the capability snapshot.
func (parseP *CerebrasProvider) ParseCapabilities(parseModel string) ModelCapabilities {
	if parseMetadata, parseOk := parseP.catalog.ParseModelMetadata(parseModel); parseOk {
		return parseMetadata.Capabilities
	}
	return ModelCapabilities{ProviderID: parseP.ParseID(), ProviderLabel: "Cerebras"}
}

// ParseHealth returns the health snapshot.
func (parseP *CerebrasProvider) ParseHealth() ProviderHealth {
	parseStatus := ProviderHealthUnavailable
	if parseP.ParseAvailable() {
		parseStatus = ProviderHealthUnknown
	}
	return ProviderHealth{ProviderID: parseP.ParseID(), Status: parseStatus}
}

// ParseCurrentRateLimits returns the current rate-limit snapshot.
func (parseP *CerebrasProvider) ParseCurrentRateLimits() RateLimitSnapshot {
	return RateLimitSnapshot{}
}

// ParseGenerateTitle generates one conversation title.
func (parseP *CerebrasProvider) ParseGenerateTitle(parseCtx context.Context, parseReq TitleRequest) (string, error) {
	if !parseP.ParseAvailable() {
		return "", ErrNoProvidersAvailable
	}

	parseResponse, parseErr := parseP.client.Chat.Completions.New(parseCtx, openai.ChatCompletionNewParams{
		Model: shared.ChatModel(parseP.catalog.TitleModel),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(strings.TrimSpace(parseReq.SystemPrompt)),
			openai.UserMessage(strings.TrimSpace(parseReq.Prompt)),
		},
		MaxCompletionTokens: openai.Int(64),
	})
	if parseErr != nil {
		return "", fmt.Errorf("cerebras title: %w", parseErr)
	}
	if len(parseResponse.Choices) == 0 {
		return "", errors.New("cerebras title: empty response")
	}
	parseTitle := strings.TrimSpace(parseResponse.Choices[0].Message.Content)
	if parseTitle == "" {
		return "", errors.New("cerebras title: empty title")
	}
	return parseTitle, nil
}

// ParseExtractUserMemories extracts user-memory candidates from the current conversation.
func (parseP *CerebrasProvider) ParseExtractUserMemories(parseCtx context.Context, parseReq MemoryExtractionRequest) ([]UserMemoryCandidate, error) {
	if !parseP.ParseAvailable() {
		return nil, ErrNoProvidersAvailable
	}
	if strings.TrimSpace(parseReq.UserMessage) == "" {
		return nil, nil
	}
	parseResolvedModel := strings.TrimSpace(parseReq.Model)
	if parseResolvedModel == "" {
		parseResolvedModel = parseP.ParseDefaultModel()
	}
	parseResponse, parseErr := parseP.client.Chat.Completions.New(parseCtx, openai.ChatCompletionNewParams{
		Model: shared.ChatModel(parseResolvedModel),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(strings.TrimSpace(`You extract stable, reusable user memory candidates from one user message.
Return one JSON object only with shape {"memories":[...]}.
Each memory item must include:
- key (string)
- category (one of: preference, profile, constraint, project, other)
- summary (string)
- detail (string)
- usefulness_score (integer 0-100)
- confidence_score (number 0-1)
- rubric_reason (string)

Rules:
- Prefer stable preferences, durable personal details, ongoing constraints, and long-lived project context.
- Exclude one-off requests, secrets, passwords, API keys, payment details, government IDs, and exact street addresses.
- If nothing qualifies, return {"memories":[]}.`)),
			openai.UserMessage("User message:\n" + strings.TrimSpace(parseReq.UserMessage)),
		},
		MaxCompletionTokens: openai.Int(cerebrasMemoryExtractionMaxCompletionTokens),
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &shared.ResponseFormatJSONObjectParam{Type: "json_object"},
		},
	})
	if parseErr != nil {
		return nil, fmt.Errorf("cerebras memory extraction: %w", parseErr)
	}
	parseCandidates, parseErr := parseResolveCerebrasMemoryCandidates(parseResponse)
	if parseErr != nil {
		return nil, fmt.Errorf("cerebras memory extraction parse: %w", parseErr)
	}
	return parseNormalizeMemoryCandidates(parseCandidates), nil
}

// parseResolveCerebrasMemoryCandidates resolves one chat-completion extraction response into memory candidates.
func parseResolveCerebrasMemoryCandidates(parseResponse *openai.ChatCompletion) ([]UserMemoryCandidate, error) {
	if parseResponse == nil || len(parseResponse.Choices) == 0 {
		return []UserMemoryCandidate{}, nil
	}
	parseOutput := strings.TrimSpace(parseResponse.Choices[0].Message.Content)
	if parseOutput == "" {
		return []UserMemoryCandidate{}, nil
	}
	parseCandidates, isParseDecoded := parseDecodeMemoryCandidateList([]byte(parseOutput))
	if !isParseDecoded {
		parseFallbackOutput := parseExtractJSONObject(parseOutput)
		parseCandidates, isParseDecoded = parseDecodeMemoryCandidateList([]byte(parseFallbackOutput))
		if !isParseDecoded {
			return []UserMemoryCandidate{}, nil
		}
	}
	return parseCandidates, nil
}

// ParseStreamChat streams one chat completion.
func (parseP *CerebrasProvider) ParseStreamChat(parseCtx context.Context, parseReq ChatRequest, parseEmit func(ChatEvent) error) (ChatResult, error) {
	if !parseP.ParseAvailable() {
		return ChatResult{}, ErrNoProvidersAvailable
	}

	parseResolvedModel := strings.TrimSpace(parseReq.Model)
	if parseResolvedModel == "" {
		parseResolvedModel = parseP.ParseDefaultModel()
	}

	parseParams := openai.ChatCompletionNewParams{
		Model:               shared.ChatModel(parseResolvedModel),
		Messages:            parseCerebrasChatMessages(parseReq.SystemPrompt, parseReq.History, parseReq.UserMessage),
		MaxCompletionTokens: openai.Int(cerebrasMaxCompletionTokens),
		StreamOptions: openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		},
	}
	if parseReq.ThinkingEnabled && parseP.ParseCapabilities(parseResolvedModel).SupportsThinking {
		parseParams.ReasoningEffort = parseCerebrasReasoningEffort(parseReq.ThinkingEffort)
	}

	parseStream := parseP.client.Chat.Completions.NewStreaming(parseCtx, parseParams)
	isParseThoughtStarted := false
	isParseThoughtDoneSent := false
	var parsePromptTokens int64
	var parseCompletionTokens int64
	parseProviderRequestID := ""

	parseEmitThoughtDone := func() error {
		if isParseThoughtDoneSent {
			return nil
		}
		isParseThoughtDoneSent = true
		return parseEmit(ChatEvent{ThoughtDone: true})
	}

	for parseStream.Next() {
		parseChunk := parseStream.Current()
		if parseCurrentRequestID := strings.TrimSpace(fmt.Sprintf("%v", parseChunk.ID)); parseCurrentRequestID != "" {
			parseProviderRequestID = parseCurrentRequestID
		}
		if parseChunk.Usage.CompletionTokens > 0 || parseChunk.Usage.PromptTokens > 0 {
			parsePromptTokens = parseChunk.Usage.PromptTokens
			parseCompletionTokens = parseChunk.Usage.CompletionTokens
		}
		parseReasoningDelta := parseCerebrasReasoningDelta(parseChunk.RawJSON())
		if parseReasoningDelta != "" {
			isParseThoughtStarted = true
			if parseErr := parseEmit(ChatEvent{ThoughtDelta: parseReasoningDelta}); parseErr != nil {
				return ChatResult{}, parseErr
			}
		}
		for _, parseChoice := range parseChunk.Choices {
			if parseChoice.Delta.Content == "" {
				continue
			}
			if isParseThoughtStarted && !isParseThoughtDoneSent {
				if parseErr2 := parseEmitThoughtDone(); parseErr2 != nil {
					return ChatResult{}, parseErr2
				}
			}
			if parseErr3 := parseEmit(ChatEvent{TextDelta: parseChoice.Delta.Content}); parseErr3 != nil {
				return ChatResult{}, parseErr3
			}
		}
	}
	if parseErr4 := parseStream.Err(); parseErr4 != nil {
		return ChatResult{}, fmt.Errorf("cerebras stream: %w", parseErr4)
	}
	if isParseThoughtStarted && !isParseThoughtDoneSent {
		if parseErr5 := parseEmitThoughtDone(); parseErr5 != nil {
			return ChatResult{}, parseErr5
		}
	}
	parseUsageSource := UsageSourceMissing
	if parsePromptTokens > 0 || parseCompletionTokens > 0 {
		parseUsageSource = UsageSourceExact
	}

	return ChatResult{
		Model:             parseResolvedModel,
		PromptTokens:      parsePromptTokens,
		CompletionTokens:  parseCompletionTokens,
		UsageSource:       parseUsageSource,
		ProviderRequestID: parseProviderRequestID,
	}, nil
}

// ParseSynthesizeSpeech streams one speech-synthesis response.
func (parseP *CerebrasProvider) ParseSynthesizeSpeech(parseCtx context.Context, parseReq SpeechRequest, parseEmit func(SpeechChunk) error) (SpeechResult, error) {
	_ = parseCtx
	_ = parseReq
	_ = parseEmit
	if !parseP.ParseAvailable() {
		return SpeechResult{}, ErrNoProvidersAvailable
	}
	return SpeechResult{}, &UnsupportedCapabilityError{Capability: CapabilitySpeech, Model: strings.TrimSpace(parseReq.Model), ProviderID: parseP.ParseID()}
}

func parseCerebrasChatMessages(parseSystemPrompt string, parseHistory []ChatMessage, parseUserMessage string) []openai.ChatCompletionMessageParamUnion {
	parseMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(parseHistory)+2)
	if parseTrimmedPrompt := strings.TrimSpace(parseSystemPrompt); parseTrimmedPrompt != "" {
		parseMessages = append(parseMessages, openai.SystemMessage(parseTrimmedPrompt))
	}
	for _, parseHistoryMessage := range parseHistory {
		switch ParseNormalizeRole(parseHistoryMessage.Role) {
		case "assistant":
			parseMessages = append(parseMessages, openai.AssistantMessage(strings.TrimSpace(parseHistoryMessage.Content)))
		case "developer":
			parseMessages = append(parseMessages, openai.DeveloperMessage(strings.TrimSpace(parseHistoryMessage.Content)))
		case "system":
			parseMessages = append(parseMessages, openai.SystemMessage(strings.TrimSpace(parseHistoryMessage.Content)))
		case "tool":
			parseMessages = append(parseMessages, openai.ToolMessage(strings.TrimSpace(parseHistoryMessage.Content), "tool"))
		default:
			parseMessages = append(parseMessages, openai.UserMessage(strings.TrimSpace(parseHistoryMessage.Content)))
		}
	}
	parseMessages = append(parseMessages, openai.UserMessage(strings.TrimSpace(parseUserMessage)))
	return parseMessages
}

func parseCerebrasReasoningDelta(parseRaw string) string {
	if strings.TrimSpace(parseRaw) == "" {
		return ""
	}
	var parseChunk struct {
		Choices []struct {
			Delta struct {
				Reasoning        string `json:"reasoning"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if parseErr := json.Unmarshal([]byte(parseRaw), &parseChunk); parseErr != nil {
		return ""
	}
	for _, parseChoice := range parseChunk.Choices {
		if parseChoice.Delta.Reasoning != "" {
			return parseChoice.Delta.Reasoning
		}
		if parseChoice.Delta.ReasoningContent != "" {
			return parseChoice.Delta.ReasoningContent
		}
	}
	return ""
}

func parseCerebrasReasoningEffort(parseEffort string) shared.ReasoningEffort {
	switch strings.TrimSpace(strings.ToLower(parseEffort)) {
	case "low":
		return shared.ReasoningEffortLow
	case "high":
		return shared.ReasoningEffortHigh
	default:
		return shared.ReasoningEffortMedium
	}
}

func parseNormalizeCerebrasModel(parseModel string) string {
	return strings.TrimSpace(strings.ToLower(parseModel))
}

func (parseP *CerebrasProvider) parseMustModelMetadata(parseModel string) ModelMetadata {
	parseMetadata, parseOk := parseP.ParseModelMetadata(parseModel)
	if !parseOk {
		return ModelMetadata{
			ID:            strings.TrimSpace(parseModel),
			DisplayName:   strings.TrimSpace(parseModel),
			ProviderID:    parseP.ParseID(),
			ProviderLabel: "Cerebras",
			Capabilities:  parseP.ParseCapabilities(parseModel),
		}
	}
	return parseMetadata
}
