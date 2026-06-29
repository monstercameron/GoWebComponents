package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	anthropicoption "github.com/anthropics/anthropic-sdk-go/option"
)

const anthropicMaxTokens int64 = 4096
const anthropicMemoryExtractionMaxTokens int64 = 1024
const anthropicBaseURL = "https://api.anthropic.com/v1"
const anthropicMemoryExtractionToolName = "extract_user_memories"

var anthropicMemoryExtractionSchemaProperties = map[string]any{
	"memories": map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key": map[string]any{
					"type": "string",
				},
				"category": map[string]any{
					"type": "string",
					"enum": []string{"preference", "profile", "constraint", "project", "other"},
				},
				"summary": map[string]any{
					"type": "string",
				},
				"detail": map[string]any{
					"type": "string",
				},
				"usefulness_score": map[string]any{
					"type":    "integer",
					"minimum": 0,
					"maximum": 100,
				},
				"confidence_score": map[string]any{
					"type":    "number",
					"minimum": 0,
					"maximum": 1,
				},
				"rubric_reason": map[string]any{
					"type": "string",
				},
			},
			"required": []string{
				"key",
				"category",
				"summary",
				"detail",
				"usefulness_score",
				"confidence_score",
				"rubric_reason",
			},
			"additionalProperties": false,
		},
		"maxItems": 5,
	},
}

type AnthropicProvider struct {
	client  *anthropic.Client
	catalog Catalog
}

// ParseNewAnthropicProvider creates an Anthropic provider wired to the supplied catalog and API key.
func ParseNewAnthropicProvider(parseApiKey string, parseCatalog Catalog) *AnthropicProvider {
	parseTrimmedAPIKey := strings.TrimSpace(parseApiKey)
	parseResolvedCatalog := parseNormalizeCatalog("anthropic", "Anthropic", parseCatalog)
	if parseTrimmedAPIKey == "" {
		return &AnthropicProvider{catalog: parseResolvedCatalog}
	}
	parseClient := anthropic.NewClient(
		anthropicoption.WithAPIKey(parseTrimmedAPIKey),
		anthropicoption.WithMiddleware(parseBuildTraceabilityMiddleware()),
	)
	return &AnthropicProvider{client: &parseClient, catalog: parseResolvedCatalog}
}

// ParseID returns the provider identifier.
func (parseP *AnthropicProvider) ParseID() string {
	return "anthropic"
}

// ParseAvailable reports whether the provider is available.
func (parseP *AnthropicProvider) ParseAvailable() bool {
	return parseP != nil && parseP.client != nil && len(parseP.catalog.Options) > 0
}

// ParseInfo returns the provider info snapshot.
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

// ParseDefaultModel returns the default model for the provider.
func (parseP *AnthropicProvider) ParseDefaultModel() string {
	return strings.TrimSpace(parseP.catalog.DefaultModel)
}

// ParseSupportsModel reports whether the provider supports the requested model.
func (parseP *AnthropicProvider) ParseSupportsModel(parseModel string) bool {
	return parseP.catalog.ParseSupportsModel(parseModel)
}

// ParseModelOptions returns the provider model options.
func (parseP *AnthropicProvider) ParseModelOptions() []ModelOption {
	return parseP.catalog.ParseModelOptions()
}

// ParseModelMetadata returns the provider model metadata.
func (parseP *AnthropicProvider) ParseModelMetadata(parseModel string) (ModelMetadata, bool) {
	return parseP.catalog.ParseModelMetadata(parseModel)
}

// ParseCapabilities returns the capability snapshot.
func (parseP *AnthropicProvider) ParseCapabilities(parseModel string) ModelCapabilities {
	if parseMetadata, parseOk := parseP.catalog.ParseModelMetadata(parseModel); parseOk {
		return parseMetadata.Capabilities
	}
	return ModelCapabilities{ProviderID: parseP.ParseID(), ProviderLabel: "Anthropic"}
}

// ParseHealth returns the health snapshot.
func (parseP *AnthropicProvider) ParseHealth() ProviderHealth {
	parseStatus := ProviderHealthUnavailable
	if parseP.ParseAvailable() {
		parseStatus = ProviderHealthUnknown
	}
	return ProviderHealth{ProviderID: parseP.ParseID(), Status: parseStatus}
}

// ParseCurrentRateLimits returns the current rate-limit snapshot.
func (parseP *AnthropicProvider) ParseCurrentRateLimits() RateLimitSnapshot {
	return RateLimitSnapshot{}
}

// ParseGenerateTitle generates one conversation title.
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

// ParseExtractUserMemories extracts user-memory candidates from the current conversation.
func (parseP *AnthropicProvider) ParseExtractUserMemories(parseCtx context.Context, parseReq MemoryExtractionRequest) ([]UserMemoryCandidate, error) {
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
	parseMessage, parseErr := parseP.client.Messages.New(parseCtx, anthropic.MessageNewParams{
		MaxTokens: anthropicMemoryExtractionMaxTokens,
		Messages: []anthropic.MessageParam{
			parseAnthropicTextMessage("user", "User message:\n"+strings.TrimSpace(parseReq.UserMessage)),
		},
		Model: anthropic.Model(parseResolvedModel),
		System: parseAnthropicSystemPrompt(strings.TrimSpace(`You extract stable, reusable user memory candidates from one user message.
Use the provided extraction tool exactly once and return only tool input that matches its JSON schema.

Rubric:
- Score 0-39: ephemeral, one-off, or not useful later.
- Score 40-59: maybe useful, but weak or uncertain.
- Score 60-79: clearly useful future preference/detail/constraint.
- Score 80-100: highly reusable stable preference, identity detail, or ongoing constraint/project context.

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
- If nothing qualifies, return a tool input object with {"memories":[]}.`)),
		ToolChoice: anthropic.ToolChoiceParamOfTool(anthropicMemoryExtractionToolName),
		Tools: []anthropic.ToolUnionParam{
			parseBuildAnthropicMemoryExtractionTool(),
		},
	})
	if parseErr != nil {
		return nil, fmt.Errorf("anthropic memory extraction: %w", parseErr)
	}
	parseCandidates, isParseResolved := parseResolveAnthropicMemoryCandidates(parseMessage)
	if !isParseResolved {
		return []UserMemoryCandidate{}, nil
	}
	return parseNormalizeMemoryCandidates(parseCandidates), nil
}

// ParseStreamChat streams one chat completion.
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
	parseUsageSource := UsageSourceMissing
	if parseMessage.Usage.InputTokens > 0 || parseMessage.Usage.OutputTokens > 0 {
		parseUsageSource = UsageSourceExact
	}

	return ChatResult{
		Model:             parseResolvedModel,
		PromptTokens:      parseMessage.Usage.InputTokens,
		CompletionTokens:  parseMessage.Usage.OutputTokens,
		UsageSource:       parseUsageSource,
		ProviderRequestID: strings.TrimSpace(fmt.Sprintf("%v", parseMessage.ID)),
	}, nil
}

// ParseSynthesizeSpeech streams one speech-synthesis response.
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

// parseBuildAnthropicMemoryExtractionTool builds the extraction tool schema used for deterministic memory output.
func parseBuildAnthropicMemoryExtractionTool() anthropic.ToolUnionParam {
	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        anthropicMemoryExtractionToolName,
			Description: anthropic.String("Extract normalized user memory candidates from one user message."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: anthropicMemoryExtractionSchemaProperties,
				Required:   []string{"memories"},
				ExtraFields: map[string]any{
					"additionalProperties": false,
				},
			},
			Strict: anthropic.Bool(true),
		},
	}
}

// parseResolveAnthropicMemoryCandidates resolves one extraction response into candidate rows with tool-first fallback behavior.
func parseResolveAnthropicMemoryCandidates(parseMessage *anthropic.Message) ([]UserMemoryCandidate, bool) {
	if parseToolCandidates, isParseResolved := parseResolveAnthropicMemoryCandidatesFromToolUse(parseMessage); isParseResolved {
		return parseToolCandidates, true
	}
	if parseTextCandidates, isParseResolved := parseResolveAnthropicMemoryCandidatesFromText(parseMessage); isParseResolved {
		return parseTextCandidates, true
	}
	return nil, false
}

// parseResolveAnthropicMemoryCandidatesFromToolUse parses one tool_use block payload into memory candidates.
func parseResolveAnthropicMemoryCandidatesFromToolUse(parseMessage *anthropic.Message) ([]UserMemoryCandidate, bool) {
	if parseMessage == nil {
		return nil, false
	}
	for _, parseContentBlock := range parseMessage.Content {
		parseToolUseBlock, isParseToolUse := parseContentBlock.AsAny().(anthropic.ToolUseBlock)
		if !isParseToolUse || strings.TrimSpace(parseToolUseBlock.Name) != anthropicMemoryExtractionToolName {
			continue
		}
		if len(parseToolUseBlock.Input) == 0 {
			return nil, false
		}
		parseCandidates, isParseDecoded := parseDecodeMemoryCandidateList(parseToolUseBlock.Input)
		if !isParseDecoded {
			return nil, false
		}
		return parseCandidates, true
	}
	return nil, false
}

// parseResolveAnthropicMemoryCandidatesFromText parses one text response payload into memory candidates as a compatibility fallback.
func parseResolveAnthropicMemoryCandidatesFromText(parseMessage *anthropic.Message) ([]UserMemoryCandidate, bool) {
	parseOutput := strings.TrimSpace(parseAnthropicMessageText(parseMessage))
	if parseOutput == "" {
		return nil, false
	}
	parseCandidates, isParseDecoded := parseDecodeMemoryCandidateList([]byte(parseOutput))
	if !isParseDecoded {
		parseFallbackOutput := parseExtractJSONObject(parseOutput)
		parseCandidates, isParseDecoded = parseDecodeMemoryCandidateList([]byte(parseFallbackOutput))
		if !isParseDecoded {
			return nil, false
		}
	}
	return parseCandidates, true
}

// parseDecodeMemoryCandidateList decodes one memory-extraction JSON object into provider-agnostic candidate rows.
func parseDecodeMemoryCandidateList(parseRawPayload []byte) ([]UserMemoryCandidate, bool) {
	var parsePayload struct {
		Memories []map[string]any `json:"memories"`
	}
	if parseErr := json.Unmarshal(parseRawPayload, &parsePayload); parseErr != nil {
		return nil, false
	}
	parseCandidates := make([]UserMemoryCandidate, 0, len(parsePayload.Memories))
	for _, parseRawCandidate := range parsePayload.Memories {
		parseCandidates = append(parseCandidates, parseBuildMemoryCandidateFromMap(parseRawCandidate))
	}
	return parseCandidates, true
}

// parseBuildMemoryCandidateFromMap normalizes one raw memory row map into the shared memory-candidate struct.
func parseBuildMemoryCandidateFromMap(parseRawCandidate map[string]any) UserMemoryCandidate {
	return UserMemoryCandidate{
		Key:             parseResolveMemoryFieldStringFromMap(parseRawCandidate, "key"),
		Category:        parseResolveMemoryFieldStringFromMap(parseRawCandidate, "category"),
		Summary:         parseResolveMemoryFieldStringFromMap(parseRawCandidate, "summary"),
		Detail:          parseResolveMemoryFieldStringFromMap(parseRawCandidate, "detail"),
		UsefulnessScore: parseResolveMemoryFieldIntFromMap(parseRawCandidate, "usefulness_score", "usefulnessScore"),
		ConfidenceScore: parseResolveMemoryFieldFloatFromMap(parseRawCandidate, "confidence_score", "confidenceScore"),
		RubricReason:    parseResolveMemoryFieldStringFromMap(parseRawCandidate, "rubric_reason", "rubricReason"),
	}
}

// parseResolveMemoryFieldStringFromMap resolves the first non-empty string value found for one set of candidate keys.
func parseResolveMemoryFieldStringFromMap(parseRawCandidate map[string]any, parseCandidateKeys ...string) string {
	for _, parseCandidateKey := range parseCandidateKeys {
		parseRawValue, hasParseValue := parseRawCandidate[parseCandidateKey]
		if !hasParseValue {
			continue
		}
		parseResolvedValue, isParseString := parseRawValue.(string)
		if !isParseString {
			continue
		}
		if parseResolvedValue = strings.TrimSpace(parseResolvedValue); parseResolvedValue != "" {
			return parseResolvedValue
		}
	}
	return ""
}

// parseResolveMemoryFieldIntFromMap resolves one integer field from string or numeric JSON values.
func parseResolveMemoryFieldIntFromMap(parseRawCandidate map[string]any, parseCandidateKeys ...string) int {
	for _, parseCandidateKey := range parseCandidateKeys {
		parseRawValue, hasParseValue := parseRawCandidate[parseCandidateKey]
		if !hasParseValue {
			continue
		}
		switch parseValue := parseRawValue.(type) {
		case int:
			return parseValue
		case int64:
			return int(parseValue)
		case float64:
			return int(parseValue)
		case json.Number:
			parseResolvedValue, parseErr := parseValue.Int64()
			if parseErr == nil {
				return int(parseResolvedValue)
			}
		case string:
			parseResolvedValue, parseErr := strconv.Atoi(strings.TrimSpace(parseValue))
			if parseErr == nil {
				return parseResolvedValue
			}
		}
	}
	return 0
}

// parseResolveMemoryFieldFloatFromMap resolves one float field from string or numeric JSON values.
func parseResolveMemoryFieldFloatFromMap(parseRawCandidate map[string]any, parseCandidateKeys ...string) float64 {
	for _, parseCandidateKey := range parseCandidateKeys {
		parseRawValue, hasParseValue := parseRawCandidate[parseCandidateKey]
		if !hasParseValue {
			continue
		}
		switch parseValue := parseRawValue.(type) {
		case float64:
			return parseValue
		case int:
			return float64(parseValue)
		case int64:
			return float64(parseValue)
		case json.Number:
			parseResolvedValue, parseErr := parseValue.Float64()
			if parseErr == nil {
				return parseResolvedValue
			}
		case string:
			parseResolvedValue, parseErr := strconv.ParseFloat(strings.TrimSpace(parseValue), 64)
			if parseErr == nil {
				return parseResolvedValue
			}
		}
	}
	return 0
}

// parseNormalizeMemoryCandidates normalizes one memory-candidate slice into stable score/category bounds and unique keys.
func parseNormalizeMemoryCandidates(parseCandidates []UserMemoryCandidate) []UserMemoryCandidate {
	parseNormalizedCandidates := make([]UserMemoryCandidate, 0, len(parseCandidates))
	parseSeenKeys := make(map[string]int)
	for parseIndex, parseCandidate := range parseCandidates {
		parseCategory := strings.TrimSpace(strings.ToLower(parseCandidate.Category))
		switch parseCategory {
		case "preference", "profile", "constraint", "project", "other":
		default:
			parseCategory = "other"
		}
		parseSummary := strings.TrimSpace(parseCandidate.Summary)
		parseDetail := strings.TrimSpace(parseCandidate.Detail)
		parseRubricReason := strings.TrimSpace(parseCandidate.RubricReason)
		parseKey := parseBuildMemoryCandidateKey(parseCandidate.Key, parseSummary, parseDetail, parseIndex)
		if parseKey == "" {
			continue
		}
		if parseSeenCount, hasParseSeen := parseSeenKeys[parseKey]; hasParseSeen {
			parseSeenKeys[parseKey] = parseSeenCount + 1
			parseKey = parseKey + "-" + strconv.Itoa(parseSeenCount+1)
		} else {
			parseSeenKeys[parseKey] = 1
		}
		parseUsefulness := min(max(parseCandidate.UsefulnessScore, 0), 100)
		parseConfidence := parseCandidate.ConfidenceScore
		if parseConfidence < 0 {
			parseConfidence = 0
		}
		if parseConfidence > 1 {
			parseConfidence = 1
		}
		parseNormalizedCandidates = append(parseNormalizedCandidates, UserMemoryCandidate{
			Key:             parseKey,
			Category:        parseCategory,
			Summary:         parseSummary,
			Detail:          parseDetail,
			UsefulnessScore: parseUsefulness,
			ConfidenceScore: parseConfidence,
			RubricReason:    parseRubricReason,
		})
	}
	return parseNormalizedCandidates
}

// parseBuildMemoryCandidateKey resolves one stable memory key using explicit key first and summary/detail fallback.
func parseBuildMemoryCandidateKey(parseRawKey, parseSummary, parseDetail string, parseIndex int) string {
	parseResolvedKey := strings.TrimSpace(strings.ToLower(parseRawKey))
	if parseResolvedKey == "" {
		parseResolvedKey = strings.TrimSpace(strings.ToLower(parseSummary))
	}
	if parseResolvedKey == "" {
		parseResolvedKey = strings.TrimSpace(strings.ToLower(parseDetail))
	}
	parseBuilder := strings.Builder{}
	parseLastDash := false
	for _, parseRune := range parseResolvedKey {
		if (parseRune >= 'a' && parseRune <= 'z') || (parseRune >= '0' && parseRune <= '9') {
			parseBuilder.WriteRune(parseRune)
			parseLastDash = false
			continue
		}
		if !parseLastDash {
			parseBuilder.WriteRune('-')
			parseLastDash = true
		}
	}
	parseNormalizedKey := strings.Trim(strings.TrimSpace(parseBuilder.String()), "-")
	if parseNormalizedKey == "" {
		parseNormalizedKey = fmt.Sprintf("memory-%d", parseIndex+1)
	}
	return parseNormalizedKey
}
