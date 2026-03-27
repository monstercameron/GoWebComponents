package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
)

const openAITTSDaultModel = openai.SpeechModelGPT4oMiniTTS
const openAITTSDefaultVoice = openai.AudioSpeechNewParamsVoiceSage
const openAITTSDefaultMimeType = "audio/mpeg"
const openAITTSStreamChunkSize = 32 * 1024
const openAIBaseURL = "https://api.openai.com/v1"

var openAIMemoryExtractionSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
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
	},
	"required":             []string{"memories"},
	"additionalProperties": false,
}

type OpenAIProvider struct {
	client  *openai.Client
	catalog Catalog
}

func ParseNewOpenAIProvider(parseApiKey string, parseCatalog Catalog) *OpenAIProvider {
	parseTrimmedAPIKey := strings.TrimSpace(parseApiKey)
	parseResolvedCatalog := parseNormalizeCatalog("openai", "OpenAI", parseCatalog)
	if parseTrimmedAPIKey == "" {
		return &OpenAIProvider{catalog: parseResolvedCatalog}
	}
	parseClient := openai.NewClient(option.WithAPIKey(parseTrimmedAPIKey))
	return &OpenAIProvider{client: &parseClient, catalog: parseResolvedCatalog}
}

func (parseP *OpenAIProvider) ParseID() string {
	return "openai"
}

func (parseP *OpenAIProvider) ParseAvailable() bool {
	return parseP != nil && parseP.client != nil && len(parseP.catalog.Options) > 0
}

func (parseP *OpenAIProvider) ParseInfo() ProviderInfo {
	return ProviderInfo{
		ID:                 parseP.ParseID(),
		Label:              "OpenAI",
		BaseURL:            openAIBaseURL,
		AuthConfigured:     parseP.ParseAvailable(),
		Available:          parseP.ParseAvailable(),
		StreamingSupported: true,
		ReasoningSupported: true,
		ToolUseSupported:   true,
	}
}

func (parseP *OpenAIProvider) ParseDefaultModel() string {
	return strings.TrimSpace(parseP.catalog.DefaultModel)
}

func (parseP *OpenAIProvider) ParseSupportsModel(parseModel string) bool {
	return parseP.catalog.ParseSupportsModel(parseModel)
}

func (parseP *OpenAIProvider) ParseModelOptions() []ModelOption {
	return parseP.catalog.ParseModelOptions()
}

func (parseP *OpenAIProvider) ParseModelMetadata(parseModel string) (ModelMetadata, bool) {
	return parseP.catalog.ParseModelMetadata(parseModel)
}

func (parseP *OpenAIProvider) ParseCapabilities(parseModel string) ModelCapabilities {
	if parseMetadata, parseOk := parseP.catalog.ParseModelMetadata(parseModel); parseOk {
		return parseMetadata.Capabilities
	}
	return ModelCapabilities{ProviderID: parseP.ParseID(), ProviderLabel: "OpenAI"}
}

func (parseP *OpenAIProvider) ParseHealth() ProviderHealth {
	parseStatus := ProviderHealthUnavailable
	if parseP.ParseAvailable() {
		parseStatus = ProviderHealthUnknown
	}
	return ProviderHealth{ProviderID: parseP.ParseID(), Status: parseStatus}
}

func (parseP *OpenAIProvider) ParseCurrentRateLimits() RateLimitSnapshot {
	return RateLimitSnapshot{}
}

func (parseP *OpenAIProvider) ParseGenerateTitle(parseCtx context.Context, parseReq TitleRequest) (string, error) {
	if !parseP.ParseAvailable() {
		return "", ErrNoProvidersAvailable
	}

	parseResponse, parseErr := parseP.client.Responses.New(parseCtx, responses.ResponseNewParams{
		Model:        shared.ResponsesModel(parseP.catalog.TitleModel),
		Instructions: openai.String(parseReq.SystemPrompt),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(parseReq.Prompt),
		},
	})
	if parseErr != nil {
		return "", fmt.Errorf("openai title: %w", parseErr)
	}

	parseTitle := strings.TrimSpace(parseResponse.OutputText())
	if parseTitle == "" {
		return "", errors.New("openai title: empty title")
	}
	return parseTitle, nil
}

func (parseP *OpenAIProvider) ParseExtractUserMemories(parseCtx context.Context, parseReq MemoryExtractionRequest) ([]UserMemoryCandidate, error) {
	if !parseP.ParseAvailable() {
		return nil, ErrNoProvidersAvailable
	}

	parseResolvedModel := strings.TrimSpace(parseReq.Model)
	if parseResolvedModel == "" {
		parseResolvedModel = parseP.ParseDefaultModel()
	}

	parseResponse, parseErr := parseP.client.Responses.New(parseCtx, responses.ResponseNewParams{
		Model: shared.ResponsesModel(parseResolvedModel),
		Instructions: openai.String(strings.TrimSpace(`You extract stable, reusable user memory candidates from a single user message.
Rubric:
- Score 0-39: ephemeral, one-off, or not useful later.
- Score 40-59: maybe useful, but weak or uncertain.
- Score 60-79: clearly useful future preference/detail/constraint.
- Score 80-100: highly reusable stable preference, identity detail, or ongoing constraint/project context.

Only include memories that are likely to help future replies. Prefer stable preferences, durable personal details, ongoing projects, recurring constraints, and explicit likes/dislikes.
Do not store secrets, passwords, API keys, payment details, government IDs, or exact street addresses.
If nothing qualifies, return {"memories":[]}.`)),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String("User message:\n" + strings.TrimSpace(parseReq.UserMessage)),
		},
		Text: responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigUnionParam{
				OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{
					Name:   "user_memories",
					Schema: openAIMemoryExtractionSchema,
					Strict: openai.Bool(true),
				},
			},
		},
	})
	if parseErr != nil {
		return nil, fmt.Errorf("openai memory extraction: %w", parseErr)
	}

	var parsePayload struct {
		Memories []UserMemoryCandidate `json:"memories"`
	}
	parseOutput := strings.TrimSpace(parseResponse.OutputText())
	if parseErr2 := json.Unmarshal([]byte(parseOutput), &parsePayload); parseErr2 != nil {
		parseFallbackOutput := parseExtractJSONObject(parseOutput)
		if parseFallbackErr := json.Unmarshal([]byte(parseFallbackOutput), &parsePayload); parseFallbackErr != nil {
			return nil, fmt.Errorf("openai memory extraction parse: strict=%v fallback=%v", parseErr2, parseFallbackErr)
		}
	}
	return parsePayload.Memories, nil
}

func (parseP *OpenAIProvider) ParseStreamChat(parseCtx context.Context, parseReq ChatRequest, parseEmit func(ChatEvent) error) (ChatResult, error) {
	if !parseP.ParseAvailable() {
		return ChatResult{}, ErrNoProvidersAvailable
	}

	parseResolvedModel := strings.TrimSpace(parseReq.Model)
	if parseResolvedModel == "" {
		parseResolvedModel = parseP.ParseDefaultModel()
	}

	buildResponseParams := func(isIncludeReasoningSummary bool) responses.ResponseNewParams {
		parseParams := responses.ResponseNewParams{
			Model:        shared.ResponsesModel(parseResolvedModel),
			Instructions: openai.String(parseReq.SystemPrompt),
			Input: responses.ResponseNewParamsInputUnion{
				OfString: openai.String(BuildConversationInput(parseReq.History, parseReq.UserMessage)),
			},
		}
		if parseReq.ThinkingEnabled {
			parseReasoningParam := shared.ReasoningParam{
				Effort: parseOpenAIReasoningEffort(parseReq.ThinkingEffort),
			}
			if isIncludeReasoningSummary {
				parseReasoningParam.Summary = shared.ReasoningSummaryDetailed
			}
			parseParams.Reasoning = parseReasoningParam
		}
		return parseParams
	}

	parseResponseStream := parseP.client.Responses.NewStreaming(parseCtx, buildResponseParams(parseReq.ThinkingEnabled))
	isParseRetriedWithoutReasoningSummary := false
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

	var parseProcessStream func() error
	parseProcessStream = func() error {
		for parseResponseStream.Next() {
			switch parseEvent := parseResponseStream.Current().AsAny().(type) {
			case responses.ResponseCompletedEvent:
				parseProviderRequestID = strings.TrimSpace(parseEvent.Response.ID)
				parsePromptTokens = parseEvent.Response.Usage.InputTokens
				parseCompletionTokens = parseEvent.Response.Usage.OutputTokens
			case responses.ResponseReasoningSummaryTextDeltaEvent:
				if parseEvent.Delta == "" {
					continue
				}
				isParseThoughtStarted = true
				if parseErr := parseEmit(ChatEvent{ThoughtDelta: parseEvent.Delta}); parseErr != nil {
					return parseErr
				}
			case responses.ResponseReasoningSummaryTextDoneEvent:
				if parseErr2 := parseEmitThoughtDone(); parseErr2 != nil {
					return parseErr2
				}
			case responses.ResponseTextDeltaEvent:
				if parseEvent.Delta == "" {
					continue
				}
				if isParseThoughtStarted && !isParseThoughtDoneSent {
					if parseErr3 := parseEmitThoughtDone(); parseErr3 != nil {
						return parseErr3
					}
				}
				if parseErr4 := parseEmit(ChatEvent{TextDelta: parseEvent.Delta}); parseErr4 != nil {
					return parseErr4
				}
			}
		}
		if parseErr5 := parseResponseStream.Err(); parseErr5 != nil {
			if parseReq.ThinkingEnabled && !isParseRetriedWithoutReasoningSummary && strings.Contains(parseErr5.Error(), "reasoning.summary") && strings.Contains(parseErr5.Error(), "unsupported_value") {
				isParseRetriedWithoutReasoningSummary = true
				if !isParseThoughtStarted {
					if parseSendErr := parseEmit(ChatEvent{ThoughtDelta: "Reasoning summaries are unavailable for this account, so continuing without live thought output."}); parseSendErr != nil {
						return parseSendErr
					}
					isParseThoughtStarted = true
				}
				if parseSendErr2 := parseEmitThoughtDone(); parseSendErr2 != nil {
					return parseSendErr2
				}
				parseResponseStream = parseP.client.Responses.NewStreaming(parseCtx, buildResponseParams(false))
				return parseProcessStream()
			}
			return fmt.Errorf("openai stream: %w", parseErr5)
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

func (parseP *OpenAIProvider) ParseSynthesizeSpeech(parseCtx context.Context, parseReq SpeechRequest, parseEmit func(SpeechChunk) error) (SpeechResult, error) {
	if !parseP.ParseAvailable() {
		return SpeechResult{}, ErrNoProvidersAvailable
	}
	if !parseP.ParseCapabilities(parseReq.Model).SupportsSpeech {
		return SpeechResult{}, &UnsupportedCapabilityError{Capability: CapabilitySpeech, Model: strings.TrimSpace(parseReq.Model), ProviderID: parseP.ParseID()}
	}

	parseResponse, parseErr := parseP.client.Audio.Speech.New(parseCtx, openai.AudioSpeechNewParams{
		Input:          strings.TrimSpace(parseReq.Text),
		Model:          openAITTSDaultModel,
		Voice:          openAITTSDefaultVoice,
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
		StreamFormat:   openai.AudioSpeechNewParamsStreamFormatAudio,
	})
	if parseErr != nil {
		return SpeechResult{}, fmt.Errorf("openai speech: %w", parseErr)
	}
	defer parseResponse.Body.Close()

	parseResult := SpeechResult{
		MimeType: openAITTSDefaultMimeType,
		Model:    string(openAITTSDaultModel),
		Voice:    string(openAITTSDefaultVoice),
		Script:   strings.TrimSpace(parseReq.Text),
	}

	parseBuffer := make([]byte, openAITTSStreamChunkSize)
	parseTotalAudioBytes := 0
	isParseMetadataSent := false
	for {
		parseReadBytes, parseReadErr := parseResponse.Body.Read(parseBuffer)
		if parseReadBytes > 0 {
			parseChunk := SpeechChunk{AudioChunk: append([]byte(nil), parseBuffer[:parseReadBytes]...)}
			if !isParseMetadataSent {
				parseChunk.MimeType = parseResult.MimeType
				parseChunk.Model = parseResult.Model
				parseChunk.Voice = parseResult.Voice
				parseChunk.Script = parseResult.Script
				isParseMetadataSent = true
			}
			if parseErr2 := parseEmit(parseChunk); parseErr2 != nil {
				return SpeechResult{}, parseErr2
			}
			parseTotalAudioBytes += parseReadBytes
		}
		if parseReadErr == io.EOF {
			break
		}
		if parseReadErr != nil {
			return SpeechResult{}, fmt.Errorf("openai speech read: %w", parseReadErr)
		}
	}
	if parseTotalAudioBytes == 0 {
		return SpeechResult{}, errors.New("openai speech: synthesized audio was empty")
	}
	if parseErr3 := parseEmit(SpeechChunk{
		Done:     true,
		MimeType: parseResult.MimeType,
		Model:    parseResult.Model,
		Voice:    parseResult.Voice,
		Script:   parseResult.Script,
	}); parseErr3 != nil {
		return SpeechResult{}, parseErr3
	}

	return parseResult, nil
}

func parseOpenAIReasoningEffort(parseEffort string) shared.ReasoningEffort {
	switch strings.TrimSpace(strings.ToLower(parseEffort)) {
	case "low":
		return shared.ReasoningEffortLow
	case "high":
		return shared.ReasoningEffortHigh
	default:
		return shared.ReasoningEffortMedium
	}
}

var jsonFencePattern = regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")

func parseExtractJSONObject(parseSource string) string {
	parseTrimmed := strings.TrimSpace(parseSource)
	if parseTrimmed == "" {
		return `{"memories":[]}`
	}
	if parseMatches := jsonFencePattern.FindStringSubmatch(parseTrimmed); len(parseMatches) == 2 {
		parseTrimmed = strings.TrimSpace(parseMatches[1])
	}
	parseStart := strings.Index(parseTrimmed, "{")
	parseEnd := strings.LastIndex(parseTrimmed, "}")
	if parseStart >= 0 && parseEnd > parseStart {
		return parseTrimmed[parseStart : parseEnd+1]
	}
	return parseTrimmed
}

func parseNormalizeOpenAIModel(parseModel string) string {
	return strings.TrimSpace(strings.ToLower(parseModel))
}

func (parseP *OpenAIProvider) parseMustModelMetadata(parseModel string) ModelMetadata {
	parseMetadata, parseOk := parseP.ParseModelMetadata(parseModel)
	if !parseOk {
		return ModelMetadata{
			ID:            strings.TrimSpace(parseModel),
			DisplayName:   strings.TrimSpace(parseModel),
			ProviderID:    parseP.ParseID(),
			ProviderLabel: "OpenAI",
			Capabilities:  parseP.ParseCapabilities(parseModel),
		}
	}
	return parseMetadata
}
