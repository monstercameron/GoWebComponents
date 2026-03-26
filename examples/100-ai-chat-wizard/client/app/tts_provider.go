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

			resolvedRequestModel := parseNormalizeSelectedModelID(requestModel, models, fallback)
			if resolvedRequestModel != "" && parseModelSupportsSpeech(resolvedRequestModel, models, fallback) {
				requestProvider := strings.TrimSpace(strings.ToLower(parseProviderForModel(resolvedRequestModel, models, fallback).ID))
				if requestProvider == openAIProviderID {
					return resolvedRequestModel
				}
			}

			preferredModel := strings.TrimSpace(parseDefaultModelForProvider(openAIProviderID, models, fallback))
			if preferredModel != "" {
				preferredProvider := strings.TrimSpace(strings.ToLower(parseProviderForModel(preferredModel, models, fallback).ID))
				if preferredProvider == openAIProviderID && parseModelSupportsSpeech(preferredModel, models, fallback) {
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

func parseResolveTTSProviderID(parseProviderID string) string {
	parseNormalized := strings.TrimSpace(strings.ToLower(parseProviderID))
	for _, parseProvider := range ttsProviderDefinitions {
		if parseProvider.ID == parseNormalized {
			return parseProvider.ID
		}
	}
	return defaultTTSProvider
}

func parseTtsProviderLabel(parseProviderID string) string {
	parseResolvedProviderID := parseResolveTTSProviderID(parseProviderID)
	for _, parseProvider := range ttsProviderDefinitions {
		if parseProvider.ID == parseResolvedProviderID {
			return parseProvider.Label
		}
	}
	return strings.ToUpper(parseResolvedProviderID)
}

func parseTtsProviderOptionsForModels(parseModels []modelOption, parseFallback string) []ttsProviderOption {
	parseOptions := make([]ttsProviderOption, 0, len(ttsProviderDefinitions))
	for _, parseProvider := range ttsProviderDefinitions {
		parseResolvedModel := strings.TrimSpace(parseProvider.ResolveModel("", parseModels, parseFallback))
		parseOptions = append(parseOptions, ttsProviderOption{
			ID:            parseProvider.ID,
			Label:         parseProvider.Label,
			ResolvedModel: parseResolvedModel,
			Available:     parseResolvedModel != "",
		})
	}
	return parseOptions
}

func parseResolveTTSProviderModel(parseProviderID, parseRequestModel string, parseModels []modelOption, parseFallback string) string {
	parseResolvedProviderID := parseResolveTTSProviderID(parseProviderID)
	for _, parseProvider := range ttsProviderDefinitions {
		if parseProvider.ID != parseResolvedProviderID {
			continue
		}
		return strings.TrimSpace(parseProvider.ResolveModel(parseRequestModel, parseModels, parseFallback))
	}
	return ""
}

func parseTtsProviderSupportsSpeech(parseProviderID string, parseModels []modelOption, parseFallback string) bool {
	parseResolvedModel := parseResolveTTSProviderModel(parseProviderID, "", parseModels, parseFallback)
	return parseModelSupportsSpeech(parseResolvedModel, parseModels, parseFallback)
}

// openAITTSSynthesisModel stays as a compatibility helper for existing tests.
func parseOpenAITTSSynthesisModel(parseModels []modelOption, parseFallback string) string {
	return parseResolveTTSProviderModel(ttsProviderOpenAI, "", parseModels, parseFallback)
}

func parseResolveSpeechSynthesisModelForProvider(parseRequestModel string, parseModels []modelOption, parseFallback string, parseProviderID string) (string, bool) {
	parseResolvedModel := parseResolveTTSProviderModel(parseProviderID, parseRequestModel, parseModels, parseFallback)
	if !parseModelSupportsSpeech(parseResolvedModel, parseModels, parseFallback) {
		return "", false
	}
	return parseResolvedModel, true
}

// resolveSpeechSynthesisModel stays as a compatibility helper for the older
// boolean fallback mode and delegates to the provider abstraction.
func parseResolveSpeechSynthesisModel(parseRequestModel string, parseModels []modelOption, parseFallback string, isUseOpenAITTSFallback bool) (string, bool) {
	parseResolvedModel := parseNormalizeSelectedModelID(parseRequestModel, parseModels, parseFallback)
	if parseModelSupportsSpeech(parseResolvedModel, parseModels, parseFallback) {
		return parseResolvedModel, true
	}
	if !isUseOpenAITTSFallback {
		return "", false
	}
	return parseResolveSpeechSynthesisModelForProvider(parseRequestModel, parseModels, parseFallback, ttsProviderOpenAI)
}
