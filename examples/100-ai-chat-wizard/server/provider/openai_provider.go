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

type OpenAIProvider struct {
	client  *openai.Client
	catalog Catalog
}

func NewOpenAIProvider(apiKey string, catalog Catalog) *OpenAIProvider {
	trimmedAPIKey := strings.TrimSpace(apiKey)
	resolvedCatalog := normalizeCatalog("openai", "OpenAI", catalog)
	if trimmedAPIKey == "" {
		return &OpenAIProvider{catalog: resolvedCatalog}
	}
	client := openai.NewClient(option.WithAPIKey(trimmedAPIKey))
	return &OpenAIProvider{client: &client, catalog: resolvedCatalog}
}

func (p *OpenAIProvider) ID() string {
	return "openai"
}

func (p *OpenAIProvider) Available() bool {
	return p != nil && p.client != nil && len(p.catalog.Options) > 0
}

func (p *OpenAIProvider) Info() ProviderInfo {
	return ProviderInfo{
		ID:                 p.ID(),
		Label:              "OpenAI",
		BaseURL:            openAIBaseURL,
		AuthConfigured:     p.Available(),
		Available:          p.Available(),
		StreamingSupported: true,
		ReasoningSupported: true,
		ToolUseSupported:   true,
	}
}

func (p *OpenAIProvider) DefaultModel() string {
	return strings.TrimSpace(p.catalog.DefaultModel)
}

func (p *OpenAIProvider) SupportsModel(model string) bool {
	return p.catalog.SupportsModel(model)
}

func (p *OpenAIProvider) ModelOptions() []ModelOption {
	return p.catalog.ModelOptions()
}

func (p *OpenAIProvider) ModelMetadata(model string) (ModelMetadata, bool) {
	return p.catalog.ModelMetadata(model)
}

func (p *OpenAIProvider) Capabilities(model string) ModelCapabilities {
	if metadata, ok := p.catalog.ModelMetadata(model); ok {
		return metadata.Capabilities
	}
	return ModelCapabilities{ProviderID: p.ID(), ProviderLabel: "OpenAI"}
}

func (p *OpenAIProvider) Health() ProviderHealth {
	status := ProviderHealthUnavailable
	if p.Available() {
		status = ProviderHealthUnknown
	}
	return ProviderHealth{ProviderID: p.ID(), Status: status}
}

func (p *OpenAIProvider) CurrentRateLimits() RateLimitSnapshot {
	return RateLimitSnapshot{}
}

func (p *OpenAIProvider) GenerateTitle(ctx context.Context, req TitleRequest) (string, error) {
	if !p.Available() {
		return "", ErrNoProvidersAvailable
	}

	response, err := p.client.Responses.New(ctx, responses.ResponseNewParams{
		Model:        shared.ResponsesModel(p.catalog.TitleModel),
		Instructions: openai.String(req.SystemPrompt),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(req.Prompt),
		},
	})
	if err != nil {
		return "", fmt.Errorf("openai title: %w", err)
	}

	title := strings.TrimSpace(response.OutputText())
	if title == "" {
		return "", errors.New("openai title: empty title")
	}
	return title, nil
}

func (p *OpenAIProvider) ExtractUserMemories(ctx context.Context, req MemoryExtractionRequest) ([]UserMemoryCandidate, error) {
	if !p.Available() {
		return nil, ErrNoProvidersAvailable
	}

	resolvedModel := strings.TrimSpace(req.Model)
	if resolvedModel == "" {
		resolvedModel = p.DefaultModel()
	}

	response, err := p.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: shared.ResponsesModel(resolvedModel),
		Instructions: openai.String(strings.TrimSpace(`You extract stable, reusable user memory candidates from a single user message.
Return strict JSON only with this shape:
{"memories":[{"key":"","category":"","summary":"","detail":"","usefulness_score":0,"confidence_score":0,"rubric_reason":""}]}

Rubric:
- Score 0-39: ephemeral, one-off, or not useful later.
- Score 40-59: maybe useful, but weak or uncertain.
- Score 60-79: clearly useful future preference/detail/constraint.
- Score 80-100: highly reusable stable preference, identity detail, or ongoing constraint/project context.

Only include memories that are likely to help future replies. Prefer stable preferences, durable personal details, ongoing projects, recurring constraints, and explicit likes/dislikes.
Do not store secrets, passwords, API keys, payment details, government IDs, or exact street addresses.
Return at most 5 memories. If nothing qualifies, return {"memories":[]}.`)),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String("User message:\n" + strings.TrimSpace(req.UserMessage)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("openai memory extraction: %w", err)
	}

	var payload struct {
		Memories []UserMemoryCandidate `json:"memories"`
	}
	if err := json.Unmarshal([]byte(extractJSONObject(response.OutputText())), &payload); err != nil {
		return nil, fmt.Errorf("openai memory extraction parse: %w", err)
	}
	return payload.Memories, nil
}

func (p *OpenAIProvider) StreamChat(ctx context.Context, req ChatRequest, emit func(ChatEvent) error) (ChatResult, error) {
	if !p.Available() {
		return ChatResult{}, ErrNoProvidersAvailable
	}

	resolvedModel := strings.TrimSpace(req.Model)
	if resolvedModel == "" {
		resolvedModel = p.DefaultModel()
	}

	buildResponseParams := func(includeReasoningSummary bool) responses.ResponseNewParams {
		params := responses.ResponseNewParams{
			Model:        shared.ResponsesModel(resolvedModel),
			Instructions: openai.String(req.SystemPrompt),
			Input: responses.ResponseNewParamsInputUnion{
				OfString: openai.String(BuildConversationInput(req.History, req.UserMessage)),
			},
		}
		if req.ThinkingEnabled {
			reasoningParam := shared.ReasoningParam{
				Effort: openAIReasoningEffort(req.ThinkingEffort),
			}
			if includeReasoningSummary {
				reasoningParam.Summary = shared.ReasoningSummaryDetailed
			}
			params.Reasoning = reasoningParam
		}
		return params
	}

	responseStream := p.client.Responses.NewStreaming(ctx, buildResponseParams(req.ThinkingEnabled))
	retriedWithoutReasoningSummary := false
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

	var processStream func() error
	processStream = func() error {
		for responseStream.Next() {
			switch event := responseStream.Current().AsAny().(type) {
			case responses.ResponseCompletedEvent:
				promptTokens = event.Response.Usage.InputTokens
				completionTokens = event.Response.Usage.OutputTokens
			case responses.ResponseReasoningSummaryTextDeltaEvent:
				if event.Delta == "" {
					continue
				}
				thoughtStarted = true
				if err := emit(ChatEvent{ThoughtDelta: event.Delta}); err != nil {
					return err
				}
			case responses.ResponseReasoningSummaryTextDoneEvent:
				if err := emitThoughtDone(); err != nil {
					return err
				}
			case responses.ResponseTextDeltaEvent:
				if event.Delta == "" {
					continue
				}
				if thoughtStarted && !thoughtDoneSent {
					if err := emitThoughtDone(); err != nil {
						return err
					}
				}
				if err := emit(ChatEvent{TextDelta: event.Delta}); err != nil {
					return err
				}
			}
		}
		if err := responseStream.Err(); err != nil {
			if req.ThinkingEnabled && !retriedWithoutReasoningSummary && strings.Contains(err.Error(), "reasoning.summary") && strings.Contains(err.Error(), "unsupported_value") {
				retriedWithoutReasoningSummary = true
				if !thoughtStarted {
					if sendErr := emit(ChatEvent{ThoughtDelta: "Reasoning summaries are unavailable for this account, so continuing without live thought output."}); sendErr != nil {
						return sendErr
					}
					thoughtStarted = true
				}
				if sendErr := emitThoughtDone(); sendErr != nil {
					return sendErr
				}
				responseStream = p.client.Responses.NewStreaming(ctx, buildResponseParams(false))
				return processStream()
			}
			return fmt.Errorf("openai stream: %w", err)
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
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
	}, nil
}

func (p *OpenAIProvider) SynthesizeSpeech(ctx context.Context, req SpeechRequest, emit func(SpeechChunk) error) (SpeechResult, error) {
	if !p.Available() {
		return SpeechResult{}, ErrNoProvidersAvailable
	}
	if !p.Capabilities(req.Model).SupportsSpeech {
		return SpeechResult{}, &UnsupportedCapabilityError{Capability: CapabilitySpeech, Model: strings.TrimSpace(req.Model), ProviderID: p.ID()}
	}

	response, err := p.client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{
		Input:          strings.TrimSpace(req.Text),
		Model:          openAITTSDaultModel,
		Voice:          openAITTSDefaultVoice,
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
		StreamFormat:   openai.AudioSpeechNewParamsStreamFormatAudio,
	})
	if err != nil {
		return SpeechResult{}, fmt.Errorf("openai speech: %w", err)
	}
	defer response.Body.Close()

	result := SpeechResult{
		MimeType: openAITTSDefaultMimeType,
		Model:    string(openAITTSDaultModel),
		Voice:    string(openAITTSDefaultVoice),
		Script:   strings.TrimSpace(req.Text),
	}

	buffer := make([]byte, openAITTSStreamChunkSize)
	totalAudioBytes := 0
	metadataSent := false
	for {
		readBytes, readErr := response.Body.Read(buffer)
		if readBytes > 0 {
			chunk := SpeechChunk{AudioChunk: append([]byte(nil), buffer[:readBytes]...)}
			if !metadataSent {
				chunk.MimeType = result.MimeType
				chunk.Model = result.Model
				chunk.Voice = result.Voice
				chunk.Script = result.Script
				metadataSent = true
			}
			if err := emit(chunk); err != nil {
				return SpeechResult{}, err
			}
			totalAudioBytes += readBytes
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return SpeechResult{}, fmt.Errorf("openai speech read: %w", readErr)
		}
	}
	if totalAudioBytes == 0 {
		return SpeechResult{}, errors.New("openai speech: synthesized audio was empty")
	}
	if err := emit(SpeechChunk{
		Done:     true,
		MimeType: result.MimeType,
		Model:    result.Model,
		Voice:    result.Voice,
		Script:   result.Script,
	}); err != nil {
		return SpeechResult{}, err
	}

	return result, nil
}

func openAIReasoningEffort(effort string) shared.ReasoningEffort {
	switch strings.TrimSpace(strings.ToLower(effort)) {
	case "low":
		return shared.ReasoningEffortLow
	case "high":
		return shared.ReasoningEffortHigh
	default:
		return shared.ReasoningEffortMedium
	}
}

var jsonFencePattern = regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")

func extractJSONObject(source string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return `{"memories":[]}`
	}
	if matches := jsonFencePattern.FindStringSubmatch(trimmed); len(matches) == 2 {
		trimmed = strings.TrimSpace(matches[1])
	}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		return trimmed[start : end+1]
	}
	return trimmed
}

func normalizeOpenAIModel(model string) string {
	return strings.TrimSpace(strings.ToLower(model))
}

func (p *OpenAIProvider) mustModelMetadata(model string) ModelMetadata {
	metadata, ok := p.ModelMetadata(model)
	if !ok {
		return ModelMetadata{
			ID:            strings.TrimSpace(model),
			DisplayName:   strings.TrimSpace(model),
			ProviderID:    p.ID(),
			ProviderLabel: "OpenAI",
			Capabilities:  p.Capabilities(model),
		}
	}
	return metadata
}
