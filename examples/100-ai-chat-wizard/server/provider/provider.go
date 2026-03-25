package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrNoProvidersAvailable = errors.New("no configured model providers available")

type Capability string

const (
	CapabilityThinking Capability = "thinking"
	CapabilitySpeech   Capability = "speech"
)

type ModelCapabilities struct {
	ProviderID       string
	ProviderLabel    string
	SupportsThinking bool
	SupportsSpeech   bool
}

func (parseC ModelCapabilities) ParseSupports(parseCapability Capability) bool {
	switch parseCapability {
	case CapabilityThinking:
		return parseC.SupportsThinking
	case CapabilitySpeech:
		return parseC.SupportsSpeech
	default:
		return false
	}
}

type ModelOption struct {
	ID           string
	Label        string
	Note         string
	Capabilities ModelCapabilities
	Pricing      ModelPricing
}

type UnsupportedCapabilityError struct {
	Capability Capability
	Model      string
	ProviderID string
}

func (parseE *UnsupportedCapabilityError) ParseError() string {
	if parseE == nil {
		return "unsupported capability"
	}
	if strings.TrimSpace(parseE.Model) == "" {
		return fmt.Sprintf("provider %q does not support %s", parseE.ProviderID, parseE.Capability)
	}
	return fmt.Sprintf("model %q does not support %s", parseE.Model, parseE.Capability)
}

type ChatMessage struct {
	Role    string
	Content string
}

type ChatRequest struct {
	Model           string
	SystemPrompt    string
	History         []ChatMessage
	UserMessage     string
	ThinkingEnabled bool
	ThinkingEffort  string
}

type ChatEvent struct {
	TextDelta    string
	ThoughtDelta string
	ThoughtDone  bool
}

type ChatResult struct {
	Model            string
	PromptTokens     int64
	CompletionTokens int64
}

type TitleRequest struct {
	Model        string
	SystemPrompt string
	Prompt       string
}

type MemoryExtractionRequest struct {
	Model       string
	UserMessage string
}

type UserMemoryCandidate struct {
	Key             string
	Category        string
	Summary         string
	Detail          string
	UsefulnessScore int
	ConfidenceScore float64
	RubricReason    string
}

type SpeechRequest struct {
	Model string
	Text  string
}

type SpeechChunk struct {
	AudioChunk []byte
	Done       bool
	MimeType   string
	Model      string
	Voice      string
	Script     string
}

type SpeechResult struct {
	MimeType string
	Model    string
	Voice    string
	Script   string
}

type ChatProvider interface {
	ParseID() string
	ParseAvailable() bool
	ParseInfo() ProviderInfo
	ParseDefaultModel() string
	ParseSupportsModel(model string) bool
	ParseModelOptions() []ModelOption
	ParseModelMetadata(model string) (ModelMetadata, bool)
	ParseCapabilities(model string) ModelCapabilities
	ParseHealth() ProviderHealth
	ParseCurrentRateLimits() RateLimitSnapshot
	ParseStreamChat(ctx context.Context, req ChatRequest, emit func(ChatEvent) error) (ChatResult, error)
	ParseGenerateTitle(ctx context.Context, req TitleRequest) (string, error)
	ParseExtractUserMemories(ctx context.Context, req MemoryExtractionRequest) ([]UserMemoryCandidate, error)
	ParseSynthesizeSpeech(ctx context.Context, req SpeechRequest, emit func(SpeechChunk) error) (SpeechResult, error)
}

type Registry struct {
	providers []ChatProvider
}

func ParseNewRegistry(parseProviders ...ChatProvider) *Registry {
	parseAvailable := make([]ChatProvider, 0, len(parseProviders))
	for _, parseCurrentProvider := range parseProviders {
		if parseCurrentProvider == nil || !parseCurrentProvider.ParseAvailable() {
			continue
		}
		parseAvailable = append(parseAvailable, parseCurrentProvider)
	}
	return &Registry{providers: parseAvailable}
}

func (parseR *Registry) ParseResolve(parseModel string) (ChatProvider, string, error) {
	if parseR == nil || len(parseR.providers) == 0 {
		return nil, "", ErrNoProvidersAvailable
	}

	parseTrimmedModel := strings.TrimSpace(parseModel)
	if parseTrimmedModel != "" {
		for _, parseCurrentProvider := range parseR.providers {
			if parseCurrentProvider.ParseSupportsModel(parseTrimmedModel) {
				return parseCurrentProvider, parseTrimmedModel, nil
			}
		}
		return nil, "", fmt.Errorf("unsupported model %q", parseTrimmedModel)
	}

	return parseR.providers[0], parseR.providers[0].ParseDefaultModel(), nil
}

func (parseR *Registry) ParseDefaultModel() string {
	if parseR == nil || len(parseR.providers) == 0 {
		return ""
	}
	return strings.TrimSpace(parseR.providers[0].ParseDefaultModel())
}

func (parseR *Registry) ParseModelOptions() []ModelOption {
	if parseR == nil || len(parseR.providers) == 0 {
		return nil
	}
	parseOptions := make([]ModelOption, 0, len(parseR.providers)*4)
	for _, parseCurrentProvider := range parseR.providers {
		parseOptions = append(parseOptions, parseCurrentProvider.ParseModelOptions()...)
	}
	return parseOptions
}

func (parseR *Registry) ParseCapabilities(parseModel string) (ModelCapabilities, string, error) {
	parseCurrentProvider, parseResolvedModel, parseErr := parseR.ParseResolve(parseModel)
	if parseErr != nil {
		return ModelCapabilities{}, "", parseErr
	}
	return parseCurrentProvider.ParseCapabilities(parseResolvedModel), parseResolvedModel, nil
}

func (parseR *Registry) ParseProviderInfos() []ProviderInfo {
	if parseR == nil || len(parseR.providers) == 0 {
		return nil
	}
	parseInfos := make([]ProviderInfo, 0, len(parseR.providers))
	for _, parseCurrentProvider := range parseR.providers {
		parseInfos = append(parseInfos, parseCurrentProvider.ParseInfo())
	}
	return parseInfos
}

func (parseR *Registry) ParseModelMetadata(parseModel string) (ModelMetadata, string, error) {
	parseCurrentProvider, parseResolvedModel, parseErr := parseR.ParseResolve(parseModel)
	if parseErr != nil {
		return ModelMetadata{}, "", parseErr
	}
	parseMetadata, parseOk := parseCurrentProvider.ParseModelMetadata(parseResolvedModel)
	if !parseOk {
		return ModelMetadata{}, parseResolvedModel, fmt.Errorf("metadata unavailable for model %q", parseResolvedModel)
	}
	return parseMetadata, parseResolvedModel, nil
}

func (parseR *Registry) ParsePricing(parseModel string) (ModelPricing, string, error) {
	parseMetadata, parseResolvedModel, parseErr := parseR.ParseModelMetadata(parseModel)
	if parseErr != nil {
		return ModelPricing{}, "", parseErr
	}
	return parseMetadata.ParsePricing, parseResolvedModel, nil
}

func (parseR *Registry) ParseHealthSnapshots() []ProviderHealth {
	if parseR == nil || len(parseR.providers) == 0 {
		return nil
	}
	parseSnapshots := make([]ProviderHealth, 0, len(parseR.providers))
	for _, parseCurrentProvider := range parseR.providers {
		parseSnapshots = append(parseSnapshots, parseCurrentProvider.ParseHealth())
	}
	return parseSnapshots
}

func (parseR *Registry) ParseRequireCapability(parseModel string, parseCapability Capability) (ChatProvider, string, ModelCapabilities, error) {
	parseCurrentProvider, parseResolvedModel, parseErr := parseR.ParseResolve(parseModel)
	if parseErr != nil {
		return nil, "", ModelCapabilities{}, parseErr
	}
	parseCapabilities := parseCurrentProvider.ParseCapabilities(parseResolvedModel)
	if !parseCapabilities.ParseSupports(parseCapability) {
		return nil, parseResolvedModel, parseCapabilities, &UnsupportedCapabilityError{
			Capability: parseCapability,
			Model:      parseResolvedModel,
			ProviderID: parseCurrentProvider.ParseID(),
		}
	}
	return parseCurrentProvider, parseResolvedModel, parseCapabilities, nil
}

func ParseNormalizeRole(parseRole string) string {
	parseResolvedRole := strings.TrimSpace(strings.ToLower(parseRole))
	switch parseResolvedRole {
	case "assistant", "system", "developer", "tool":
		return parseResolvedRole
	default:
		return "user"
	}
}

func BuildConversationInput(parseHistory []ChatMessage, parseUserMessage string) string {
	var parseBuilder strings.Builder
	parseBuilder.WriteString("Continue this conversation naturally. The latest user turn is last.\n\n")
	for _, parseChatMessage := range parseHistory {
		parseBuilder.WriteString(ParseNormalizeRole(parseChatMessage.Role))
		parseBuilder.WriteString(":\n")
		parseBuilder.WriteString(strings.TrimSpace(parseChatMessage.Content))
		parseBuilder.WriteString("\n\n")
	}
	parseBuilder.WriteString("user:\n")
	parseBuilder.WriteString(strings.TrimSpace(parseUserMessage))
	return parseBuilder.ParseString()
}
