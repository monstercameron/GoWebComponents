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

func (c ModelCapabilities) Supports(capability Capability) bool {
	switch capability {
	case CapabilityThinking:
		return c.SupportsThinking
	case CapabilitySpeech:
		return c.SupportsSpeech
	default:
		return false
	}
}

type ModelOption struct {
	ID           string
	Label        string
	Note         string
	Capabilities ModelCapabilities
}

type UnsupportedCapabilityError struct {
	Capability Capability
	Model      string
	ProviderID string
}

func (e *UnsupportedCapabilityError) Error() string {
	if e == nil {
		return "unsupported capability"
	}
	if strings.TrimSpace(e.Model) == "" {
		return fmt.Sprintf("provider %q does not support %s", e.ProviderID, e.Capability)
	}
	return fmt.Sprintf("model %q does not support %s", e.Model, e.Capability)
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
	ID() string
	Available() bool
	Info() ProviderInfo
	DefaultModel() string
	SupportsModel(model string) bool
	ModelOptions() []ModelOption
	ModelMetadata(model string) (ModelMetadata, bool)
	Capabilities(model string) ModelCapabilities
	Health() ProviderHealth
	CurrentRateLimits() RateLimitSnapshot
	StreamChat(ctx context.Context, req ChatRequest, emit func(ChatEvent) error) (ChatResult, error)
	GenerateTitle(ctx context.Context, req TitleRequest) (string, error)
	ExtractUserMemories(ctx context.Context, req MemoryExtractionRequest) ([]UserMemoryCandidate, error)
	SynthesizeSpeech(ctx context.Context, req SpeechRequest, emit func(SpeechChunk) error) (SpeechResult, error)
}

type Registry struct {
	providers []ChatProvider
}

func NewRegistry(providers ...ChatProvider) *Registry {
	available := make([]ChatProvider, 0, len(providers))
	for _, currentProvider := range providers {
		if currentProvider == nil || !currentProvider.Available() {
			continue
		}
		available = append(available, currentProvider)
	}
	return &Registry{providers: available}
}

func (r *Registry) Resolve(model string) (ChatProvider, string, error) {
	if r == nil || len(r.providers) == 0 {
		return nil, "", ErrNoProvidersAvailable
	}

	trimmedModel := strings.TrimSpace(model)
	if trimmedModel != "" {
		for _, currentProvider := range r.providers {
			if currentProvider.SupportsModel(trimmedModel) {
				return currentProvider, trimmedModel, nil
			}
		}
		return nil, "", fmt.Errorf("unsupported model %q", trimmedModel)
	}

	return r.providers[0], r.providers[0].DefaultModel(), nil
}

func (r *Registry) DefaultModel() string {
	if r == nil || len(r.providers) == 0 {
		return ""
	}
	return strings.TrimSpace(r.providers[0].DefaultModel())
}

func (r *Registry) ModelOptions() []ModelOption {
	if r == nil || len(r.providers) == 0 {
		return nil
	}
	options := make([]ModelOption, 0, len(r.providers)*4)
	for _, currentProvider := range r.providers {
		options = append(options, currentProvider.ModelOptions()...)
	}
	return options
}

func (r *Registry) Capabilities(model string) (ModelCapabilities, string, error) {
	currentProvider, resolvedModel, err := r.Resolve(model)
	if err != nil {
		return ModelCapabilities{}, "", err
	}
	return currentProvider.Capabilities(resolvedModel), resolvedModel, nil
}

func (r *Registry) ProviderInfos() []ProviderInfo {
	if r == nil || len(r.providers) == 0 {
		return nil
	}
	infos := make([]ProviderInfo, 0, len(r.providers))
	for _, currentProvider := range r.providers {
		infos = append(infos, currentProvider.Info())
	}
	return infos
}

func (r *Registry) ModelMetadata(model string) (ModelMetadata, string, error) {
	currentProvider, resolvedModel, err := r.Resolve(model)
	if err != nil {
		return ModelMetadata{}, "", err
	}
	metadata, ok := currentProvider.ModelMetadata(resolvedModel)
	if !ok {
		return ModelMetadata{}, resolvedModel, fmt.Errorf("metadata unavailable for model %q", resolvedModel)
	}
	return metadata, resolvedModel, nil
}

func (r *Registry) Pricing(model string) (ModelPricing, string, error) {
	metadata, resolvedModel, err := r.ModelMetadata(model)
	if err != nil {
		return ModelPricing{}, "", err
	}
	return metadata.Pricing, resolvedModel, nil
}

func (r *Registry) HealthSnapshots() []ProviderHealth {
	if r == nil || len(r.providers) == 0 {
		return nil
	}
	snapshots := make([]ProviderHealth, 0, len(r.providers))
	for _, currentProvider := range r.providers {
		snapshots = append(snapshots, currentProvider.Health())
	}
	return snapshots
}

func (r *Registry) RequireCapability(model string, capability Capability) (ChatProvider, string, ModelCapabilities, error) {
	currentProvider, resolvedModel, err := r.Resolve(model)
	if err != nil {
		return nil, "", ModelCapabilities{}, err
	}
	capabilities := currentProvider.Capabilities(resolvedModel)
	if !capabilities.Supports(capability) {
		return nil, resolvedModel, capabilities, &UnsupportedCapabilityError{
			Capability: capability,
			Model:      resolvedModel,
			ProviderID: currentProvider.ID(),
		}
	}
	return currentProvider, resolvedModel, capabilities, nil
}

func NormalizeRole(role string) string {
	resolvedRole := strings.TrimSpace(strings.ToLower(role))
	switch resolvedRole {
	case "assistant", "system", "developer", "tool":
		return resolvedRole
	default:
		return "user"
	}
}

func BuildConversationInput(history []ChatMessage, userMessage string) string {
	var builder strings.Builder
	builder.WriteString("Continue this conversation naturally. The latest user turn is last.\n\n")
	for _, chatMessage := range history {
		builder.WriteString(NormalizeRole(chatMessage.Role))
		builder.WriteString(":\n")
		builder.WriteString(strings.TrimSpace(chatMessage.Content))
		builder.WriteString("\n\n")
	}
	builder.WriteString("user:\n")
	builder.WriteString(strings.TrimSpace(userMessage))
	return builder.String()
}
