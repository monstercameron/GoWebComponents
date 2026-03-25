//go:build js && wasm

package app

import "strings"

type ttsProviderDefinition struct {
	ID           string
	Label        string
	ResolveModel func(requestModel string, models []modelOption, fallback string) string
}

type ttsProviderOption struct {
	ID            string
	Label         string
	ResolvedModel string
	Available     bool
}

var ttsProviderDefinitions = []ttsProviderDefinition{
	{
		ID:    ttsProviderOpenAI,
		Label: "OpenAI",
		ResolveModel: func(requestModel string, models []modelOption, fallback string) string {
			const openAIProviderID = "openai"

			resolvedRequestModel := normalizeSelectedModelID(requestModel, models, fallback)
			if resolvedRequestModel != "" && modelSupportsSpeech(resolvedRequestModel, models, fallback) {
				requestProvider := strings.TrimSpace(strings.ToLower(providerForModel(resolvedRequestModel, models, fallback).ID))
				if requestProvider == openAIProviderID {
					return resolvedRequestModel
				}
			}

			preferredModel := strings.TrimSpace(defaultModelForProvider(openAIProviderID, models, fallback))
			if preferredModel != "" {
				preferredProvider := strings.TrimSpace(strings.ToLower(providerForModel(preferredModel, models, fallback).ID))
				if preferredProvider == openAIProviderID && modelSupportsSpeech(preferredModel, models, fallback) {
					return preferredModel
				}
			}

			for _, option := range models {
				if !option.Capabilities.SupportsSpeech {
					continue
				}
				if strings.TrimSpace(strings.ToLower(option.Capabilities.ProviderID)) != openAIProviderID {
					continue
				}
				if modelID := strings.TrimSpace(option.ID); modelID != "" {
					return modelID
				}
			}

			return ""
		},
	},
}

func resolveTTSProviderID(providerID string) string {
	normalized := strings.TrimSpace(strings.ToLower(providerID))
	for _, provider := range ttsProviderDefinitions {
		if provider.ID == normalized {
			return provider.ID
		}
	}
	return defaultTTSProvider
}

func ttsProviderLabel(providerID string) string {
	resolvedProviderID := resolveTTSProviderID(providerID)
	for _, provider := range ttsProviderDefinitions {
		if provider.ID == resolvedProviderID {
			return provider.Label
		}
	}
	return strings.ToUpper(resolvedProviderID)
}

func ttsProviderOptionsForModels(models []modelOption, fallback string) []ttsProviderOption {
	options := make([]ttsProviderOption, 0, len(ttsProviderDefinitions))
	for _, provider := range ttsProviderDefinitions {
		resolvedModel := strings.TrimSpace(provider.ResolveModel("", models, fallback))
		options = append(options, ttsProviderOption{
			ID:            provider.ID,
			Label:         provider.Label,
			ResolvedModel: resolvedModel,
			Available:     resolvedModel != "",
		})
	}
	return options
}

func resolveTTSProviderModel(providerID, requestModel string, models []modelOption, fallback string) string {
	resolvedProviderID := resolveTTSProviderID(providerID)
	for _, provider := range ttsProviderDefinitions {
		if provider.ID != resolvedProviderID {
			continue
		}
		return strings.TrimSpace(provider.ResolveModel(requestModel, models, fallback))
	}
	return ""
}

func ttsProviderSupportsSpeech(providerID string, models []modelOption, fallback string) bool {
	resolvedModel := resolveTTSProviderModel(providerID, "", models, fallback)
	return modelSupportsSpeech(resolvedModel, models, fallback)
}

// openAITTSSynthesisModel stays as a compatibility helper for existing tests.
func openAITTSSynthesisModel(models []modelOption, fallback string) string {
	return resolveTTSProviderModel(ttsProviderOpenAI, "", models, fallback)
}

func resolveSpeechSynthesisModelForProvider(requestModel string, models []modelOption, fallback string, providerID string) (string, bool) {
	resolvedModel := resolveTTSProviderModel(providerID, requestModel, models, fallback)
	if !modelSupportsSpeech(resolvedModel, models, fallback) {
		return "", false
	}
	return resolvedModel, true
}

// resolveSpeechSynthesisModel stays as a compatibility helper for the older
// boolean fallback mode and delegates to the provider abstraction.
func resolveSpeechSynthesisModel(requestModel string, models []modelOption, fallback string, useOpenAITTSFallback bool) (string, bool) {
	resolvedModel := normalizeSelectedModelID(requestModel, models, fallback)
	if modelSupportsSpeech(resolvedModel, models, fallback) {
		return resolvedModel, true
	}
	if !useOpenAITTSFallback {
		return "", false
	}
	return resolveSpeechSynthesisModelForProvider(requestModel, models, fallback, ttsProviderOpenAI)
}
