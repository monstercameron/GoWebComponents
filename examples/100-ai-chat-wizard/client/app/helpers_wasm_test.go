//go:build js && wasm

package app

import (
	"math"
	"strings"
	"testing"
)

var testAvailableModels = []modelOption{
	{
		ID:           "gpt-5.4",
		Label:        "GPT-5.4",
		Note:         "Best",
		Capabilities: modelCapabilities{ProviderID: "openai", ProviderLabel: "OpenAI", SupportsThinking: true, SupportsSpeech: true},
		Pricing:      modelPricing{InputDollarsPerMillion: 1.25, OutputDollarsPerMillion: 10.00, Currency: "USD"},
	},
	{
		ID:           "gpt-5.4-mini",
		Label:        "GPT-5.4 mini",
		Note:         "Fast",
		Capabilities: modelCapabilities{ProviderID: "openai", ProviderLabel: "OpenAI", SupportsThinking: true, SupportsSpeech: true},
		Pricing:      modelPricing{InputDollarsPerMillion: 0.25, OutputDollarsPerMillion: 2.00, Currency: "USD"},
	},
	{
		ID:           "gpt-5.4-nano",
		Label:        "GPT-5.4 nano",
		Note:         "Cheap",
		Capabilities: modelCapabilities{ProviderID: "openai", ProviderLabel: "OpenAI", SupportsThinking: true, SupportsSpeech: true},
		Pricing:      modelPricing{InputDollarsPerMillion: 0.05, OutputDollarsPerMillion: 0.40, Currency: "USD"},
	},
}

const testDefaultModel = "gpt-5.4-mini"

func TestNormalizeSelectedModelIDHandlesAliasesAndFallbacks(parseT *testing.T) {
	parseT.Parallel()

	parseTests := []struct {
		name     string
		modelID  string
		models   []modelOption
		fallback string
		want     string
	}{
		{
			name:     "maps dated alias to stable model",
			modelID:  "gpt-5.4-mini-2026-03-17",
			models:   testAvailableModels,
			fallback: testDefaultModel,
			want:     "gpt-5.4-mini",
		},
		{
			name:     "uses fallback when model is unknown",
			modelID:  "does-not-exist",
			models:   testAvailableModels,
			fallback: "gpt-5.4",
			want:     "gpt-5.4",
		},
		{
			name:     "falls back to first model when fallback is invalid",
			modelID:  "",
			models:   []modelOption{{ID: "alpha"}, {ID: "beta"}},
			fallback: "missing",
			want:     "alpha",
		},
		{
			name:     "returns empty when models and fallback are empty",
			modelID:  "",
			models:   nil,
			fallback: "",
			want:     "",
		},
	}

	for _, parseTt := range parseTests {
		parseTt2 := parseTt
		parseT.Run(parseTt2.name, func(parseT2 *testing.T) {
			parseT2.Parallel()
			if parseGot := parseNormalizeSelectedModelID(parseTt2.modelID, parseTt2.models, parseTt2.fallback); parseGot != parseTt2.want {
				parseT2.Fatalf("normalizeSelectedModelID(%q) = %q, want %q", parseTt2.modelID, parseGot, parseTt2.want)
			}
		})
	}
}

func TestSelectedModelForConversationPrefersLatestSwitchThenAssistant(parseT *testing.T) {
	parseT.Parallel()

	parseTests := []struct {
		name     string
		messages []message
		fallback string
		want     string
	}{
		{
			name: "latest switch wins over earlier assistant model",
			messages: []message{
				{Role: roleAssistant, ModelID: "gpt-5.4"},
				{Role: roleSwitch, Content: "gpt-5.4-nano"},
			},
			fallback: testDefaultModel,
			want:     "gpt-5.4-nano",
		},
		{
			name: "blank switch falls back to latest assistant model",
			messages: []message{
				{Role: roleAssistant, ModelID: "gpt-5.4"},
				{Role: roleSwitch, Content: "   "},
			},
			fallback: testDefaultModel,
			want:     "gpt-5.4",
		},
		{
			name: "unknown assistant model falls back to default",
			messages: []message{
				{Role: roleAssistant, ModelID: "unknown-model"},
			},
			fallback: testDefaultModel,
			want:     testDefaultModel,
		},
	}

	for _, parseTt := range parseTests {
		parseTt2 := parseTt
		parseT.Run(parseTt2.name, func(parseT2 *testing.T) {
			parseT2.Parallel()
			if parseGot := parseSelectedModelForConversation(parseTt2.messages, testAvailableModels, parseTt2.fallback); parseGot != parseTt2.want {
				parseT2.Fatalf("selectedModelForConversation() = %q, want %q", parseGot, parseTt2.want)
			}
		})
	}
}

func TestSelectedModelForConversationKeepsFallbackForNewOrUserOnlyThreads(parseT *testing.T) {
	parseT.Parallel()

	parseModels := []modelOption{
		{ID: "claude-sonnet-4-5", Capabilities: modelCapabilities{ProviderID: "anthropic", ProviderLabel: "Anthropic"}},
		{ID: "gpt-5.4-mini", Capabilities: modelCapabilities{ProviderID: "openai", ProviderLabel: "OpenAI"}},
	}

	parseTests := []struct {
		name     string
		messages []message
	}{
		{
			name:     "empty thread",
			messages: []message{},
		},
		{
			name: "user-only thread",
			messages: []message{
				{Role: roleUser, Content: "hello"},
			},
		},
	}

	for _, parseTt := range parseTests {
		parseTt2 := parseTt
		parseT.Run(parseTt2.name, func(parseT2 *testing.T) {
			parseT2.Parallel()
			if parseGot := parseSelectedModelForConversation(parseTt2.messages, parseModels, "claude-sonnet-4-5"); parseGot != "claude-sonnet-4-5" {
				parseT2.Fatalf("selectedModelForConversation() = %q, want claude-sonnet-4-5", parseGot)
			}
		})
	}
}

func TestNormalizeSelectedThinkingEffortHandlesCaseWhitespaceAndInvalidInput(parseT *testing.T) {
	parseT.Parallel()

	parseTests := []struct {
		input string
		want  string
	}{
		{input: " HIGH ", want: "high"},
		{input: "", want: defaultThinkingEffort},
		{input: "unknown", want: defaultThinkingEffort},
	}

	for _, parseTt := range parseTests {
		parseTt2 := parseTt
		parseT.Run(parseTt2.input, func(parseT2 *testing.T) {
			parseT2.Parallel()
			if parseGot := parseNormalizeSelectedThinkingEffort(parseTt2.input); parseGot != parseTt2.want {
				parseT2.Fatalf("normalizeSelectedThinkingEffort(%q) = %q, want %q", parseTt2.input, parseGot, parseTt2.want)
			}
		})
	}
}

func TestProviderHelpersGroupModelsAndChooseProviderDefault(parseT *testing.T) {
	parseT.Parallel()

	parseModels := []modelOption{
		{ID: "openai-fast", Label: "OpenAI Fast", Capabilities: modelCapabilities{ProviderID: "openai", ProviderLabel: "OpenAI"}},
		{ID: "openai-best", Label: "OpenAI Best", Capabilities: modelCapabilities{ProviderID: "openai", ProviderLabel: "OpenAI"}},
		{ID: "cerebras-code", Label: "Cerebras Code", Capabilities: modelCapabilities{ProviderID: "cerebras", ProviderLabel: "Cerebras"}},
	}

	parseProviders := parseProviderOptionsForModels(parseModels)
	if len(parseProviders) != 2 {
		parseT.Fatalf("len(providerOptionsForModels()) = %d, want 2", len(parseProviders))
	}
	if parseProviders[0].ID != "openai" || parseProviders[1].ID != "cerebras" {
		parseT.Fatalf("provider ordering = %#v, want openai then cerebras", parseProviders)
	}

	parseActiveProvider := parseProviderForModel("cerebras-code", parseModels, "openai-fast")
	if parseActiveProvider.ID != "cerebras" {
		parseT.Fatalf("providerForModel() = %#v, want cerebras", parseActiveProvider)
	}

	parseFilteredModels := parseModelsForProvider(parseModels, "openai")
	if len(parseFilteredModels) != 2 {
		parseT.Fatalf("len(modelsForProvider(openai)) = %d, want 2", len(parseFilteredModels))
	}

	if parseGot := parseDefaultModelForProvider("cerebras", parseModels, "openai-best"); parseGot != "cerebras-code" {
		parseT.Fatalf("defaultModelForProvider(cerebras) = %q, want cerebras-code", parseGot)
	}
}

func TestRecoverPersistedModelSelection(parseT *testing.T) {
	parseT.Parallel()

	parseTests := []struct {
		name           string
		persistedModel string
		models         []modelOption
		wantModel      string
		wantFallback   bool
	}{
		{
			name:           "keeps catalog model",
			persistedModel: "gpt-5.4-mini",
			models:         testAvailableModels,
			wantModel:      "gpt-5.4-mini",
			wantFallback:   false,
		},
		{
			name:           "maps dated alias without fallback",
			persistedModel: "gpt-5.4-mini-2026-03-17",
			models:         testAvailableModels,
			wantModel:      "gpt-5.4-mini",
			wantFallback:   false,
		},
		{
			name:           "falls back to first model when persisted selection is invalid",
			persistedModel: "missing-model",
			models:         testAvailableModels,
			wantModel:      "gpt-5.4",
			wantFallback:   true,
		},
	}

	for _, parseTt := range parseTests {
		parseTt2 := parseTt
		parseT.Run(parseTt2.name, func(parseT2 *testing.T) {
			parseT2.Parallel()
			parseGotModel, parseGotFallback := parseRecoverPersistedModelSelection(parseTt2.persistedModel, parseTt2.models)
			if parseGotModel != parseTt2.wantModel || parseGotFallback != parseTt2.wantFallback {
				parseT2.Fatalf("recoverPersistedModelSelection(%q) = (%q, %v), want (%q, %v)", parseTt2.persistedModel, parseGotModel, parseGotFallback, parseTt2.wantModel, parseTt2.wantFallback)
			}
		})
	}
}

func TestSelectedModelCrossTabChannelName(parseT *testing.T) {
	parseT.Parallel()

	parseTests := []struct {
		name        string
		sessionMail string
		want        string
	}{
		{
			name:        "uses base channel when session email missing",
			sessionMail: "   ",
			want:        crossTabChannelSelectedModel,
		},
		{
			name:        "normalizes email into channel suffix",
			sessionMail: "Demo.User+QA@example.com ",
			want:        "chat-wizard:selected-model:demo-user-qa-example-com",
		},
	}

	for _, parseTt := range parseTests {
		parseTt2 := parseTt
		parseT.Run(parseTt2.name, func(parseT2 *testing.T) {
			parseT2.Parallel()
			if parseGot := parseSelectedModelCrossTabChannelName(parseTt2.sessionMail); parseGot != parseTt2.want {
				parseT2.Fatalf("selectedModelCrossTabChannelName(%q) = %q, want %q", parseTt2.sessionMail, parseGot, parseTt2.want)
			}
		})
	}
}

func TestFilterModelsByCapability(parseT *testing.T) {
	parseT.Parallel()

	parseModels := []modelOption{
		{ID: "reasoning", Capabilities: modelCapabilities{SupportsThinking: true, SupportsSpeech: false}},
		{ID: "voice", Capabilities: modelCapabilities{SupportsThinking: false, SupportsSpeech: true}},
		{ID: "basic", Capabilities: modelCapabilities{SupportsThinking: false, SupportsSpeech: false}},
	}

	parseThinking := filterModelsByCapability(parseModels, "thinking")
	if len(parseThinking) != 1 || parseThinking[0].ID != "reasoning" {
		parseT.Fatalf("filterModelsByCapability(thinking) = %#v, want reasoning-only list", parseThinking)
	}

	parseSpeech := filterModelsByCapability(parseModels, "speech")
	if len(parseSpeech) != 1 || parseSpeech[0].ID != "voice" {
		parseT.Fatalf("filterModelsByCapability(speech) = %#v, want voice-only list", parseSpeech)
	}

	parseUnknown := filterModelsByCapability(parseModels, "unknown")
	if len(parseUnknown) != len(parseModels) {
		parseT.Fatalf("filterModelsByCapability(unknown) len = %d, want %d", len(parseUnknown), len(parseModels))
	}
}

func TestOpenAITTSSynthesisModel(parseT *testing.T) {
	parseT.Parallel()

	parseT.Run("prefers openai default when speech-capable", func(parseT2 *testing.T) {
		parseT2.Parallel()
		parseModels := []modelOption{
			{ID: "gpt-5.4-mini", Capabilities: modelCapabilities{ProviderID: "openai", SupportsSpeech: true}},
			{ID: "claude-4", Capabilities: modelCapabilities{ProviderID: "anthropic", SupportsSpeech: false}},
		}
		if parseGot := parseOpenAITTSSynthesisModel(parseModels, "gpt-5.4-mini"); parseGot != "gpt-5.4-mini" {
			parseT2.Fatalf("openAITTSSynthesisModel() = %q, want gpt-5.4-mini", parseGot)
		}
	})

	parseT.Run("falls back to first openai speech model when provider default lacks speech", func(parseT3 *testing.T) {
		parseT3.Parallel()
		parseModels2 := []modelOption{
			{ID: "gpt-5.4-mini", Capabilities: modelCapabilities{ProviderID: "openai", SupportsSpeech: false}},
			{ID: "gpt-5.4-tts", Capabilities: modelCapabilities{ProviderID: "openai", SupportsSpeech: true}},
			{ID: "cerebras-voice", Capabilities: modelCapabilities{ProviderID: "cerebras", SupportsSpeech: true}},
		}
		if parseGot2 := parseOpenAITTSSynthesisModel(parseModels2, "gpt-5.4-mini"); parseGot2 != "gpt-5.4-tts" {
			parseT3.Fatalf("openAITTSSynthesisModel() = %q, want gpt-5.4-tts", parseGot2)
		}
	})

	parseT.Run("returns empty when no openai speech model exists", func(parseT4 *testing.T) {
		parseT4.Parallel()
		parseModels3 := []modelOption{
			{ID: "claude-4", Capabilities: modelCapabilities{ProviderID: "anthropic", SupportsSpeech: false}},
			{ID: "cerebras-voice", Capabilities: modelCapabilities{ProviderID: "cerebras", SupportsSpeech: true}},
		}
		if parseGot3 := parseOpenAITTSSynthesisModel(parseModels3, "claude-4"); parseGot3 != "" {
			parseT4.Fatalf("openAITTSSynthesisModel() = %q, want empty", parseGot3)
		}
	})
}

func TestTTSProviderOptionsForModelsIncludesOpenAI(parseT *testing.T) {
	parseT.Parallel()

	parseModels := []modelOption{
		{ID: "gpt-5.4-mini", Capabilities: modelCapabilities{ProviderID: "openai", SupportsSpeech: true}},
	}

	parseOptions := parseTtsProviderOptionsForModels(parseModels, "gpt-5.4-mini")
	if len(parseOptions) != 1 {
		parseT.Fatalf("len(ttsProviderOptionsForModels()) = %d, want 1", len(parseOptions))
	}
	if parseOptions[0].ID != ttsProviderOpenAI || parseOptions[0].Label != "OpenAI" {
		parseT.Fatalf("ttsProviderOptionsForModels()[0] = %#v, want OpenAI provider", parseOptions[0])
	}
	if !parseOptions[0].Available || parseOptions[0].ResolvedModel != "gpt-5.4-mini" {
		parseT.Fatalf("ttsProviderOptionsForModels()[0] availability = %#v, want available gpt-5.4-mini", parseOptions[0])
	}
}

func TestResolveSpeechSynthesisModelForProvider(parseT *testing.T) {
	parseT.Parallel()

	parseT.Run("resolves openai provider model for non-openai active chat model", func(parseT2 *testing.T) {
		parseT2.Parallel()
		parseModels := []modelOption{
			{ID: "claude-4", Capabilities: modelCapabilities{ProviderID: "anthropic", SupportsSpeech: false}},
			{ID: "gpt-5.4-mini", Capabilities: modelCapabilities{ProviderID: "openai", SupportsSpeech: true}},
		}

		parseGotModel, parseSupported := parseResolveSpeechSynthesisModelForProvider("claude-4", parseModels, "claude-4", ttsProviderOpenAI)
		if !parseSupported || parseGotModel != "gpt-5.4-mini" {
			parseT2.Fatalf("resolveSpeechSynthesisModelForProvider() = (%q, %v), want (gpt-5.4-mini, true)", parseGotModel, parseSupported)
		}
	})

	parseT.Run("returns unsupported when provider has no speech-capable model", func(parseT3 *testing.T) {
		parseT3.Parallel()
		parseModels2 := []modelOption{
			{ID: "claude-4", Capabilities: modelCapabilities{ProviderID: "anthropic", SupportsSpeech: false}},
		}

		parseGotModel2, parseSupported2 := parseResolveSpeechSynthesisModelForProvider("claude-4", parseModels2, "claude-4", ttsProviderOpenAI)
		if parseSupported2 || parseGotModel2 != "" {
			parseT3.Fatalf("resolveSpeechSynthesisModelForProvider() = (%q, %v), want (\"\", false)", parseGotModel2, parseSupported2)
		}
	})
}

func TestResolveSpeechSynthesisModel(parseT *testing.T) {
	parseT.Parallel()

	parseT.Run("uses requested model when it already supports speech", func(parseT2 *testing.T) {
		parseT2.Parallel()
		parseModels := []modelOption{
			{ID: "gpt-5.4-mini", Capabilities: modelCapabilities{ProviderID: "openai", SupportsSpeech: true}},
		}

		parseGotModel, parseSupported := parseResolveSpeechSynthesisModel("gpt-5.4-mini", parseModels, "gpt-5.4-mini", false)
		if !parseSupported || parseGotModel != "gpt-5.4-mini" {
			parseT2.Fatalf("resolveSpeechSynthesisModel() = (%q, %v), want (gpt-5.4-mini, true)", parseGotModel, parseSupported)
		}
	})

	parseT.Run("does not fall back when openai tts fallback is disabled", func(parseT3 *testing.T) {
		parseT3.Parallel()
		parseModels2 := []modelOption{
			{ID: "claude-4", Capabilities: modelCapabilities{ProviderID: "anthropic", SupportsSpeech: false}},
			{ID: "gpt-5.4-mini", Capabilities: modelCapabilities{ProviderID: "openai", SupportsSpeech: true}},
		}

		parseGotModel2, parseSupported2 := parseResolveSpeechSynthesisModel("claude-4", parseModels2, "claude-4", false)
		if parseSupported2 || parseGotModel2 != "" {
			parseT3.Fatalf("resolveSpeechSynthesisModel() = (%q, %v), want (\"\", false)", parseGotModel2, parseSupported2)
		}
	})

	parseT.Run("uses openai tts fallback while keeping non-openai chat provider selection", func(parseT4 *testing.T) {
		parseT4.Parallel()
		parseModels3 := []modelOption{
			{ID: "claude-4", Capabilities: modelCapabilities{ProviderID: "anthropic", SupportsSpeech: false}},
			{ID: "gpt-5.4-mini", Capabilities: modelCapabilities{ProviderID: "openai", SupportsSpeech: true}},
		}

		parseActiveChatModel := "claude-4"
		parseGotModel3, parseSupported3 := parseResolveSpeechSynthesisModel(parseActiveChatModel, parseModels3, parseActiveChatModel, true)
		if !parseSupported3 || parseGotModel3 != "gpt-5.4-mini" {
			parseT4.Fatalf("resolveSpeechSynthesisModel() = (%q, %v), want (gpt-5.4-mini, true)", parseGotModel3, parseSupported3)
		}
		if parseActiveChatModel != "claude-4" {
			parseT4.Fatalf("activeChatModel mutated to %q, want claude-4", parseActiveChatModel)
		}
		parseActiveProvider := parseProviderForModel(parseActiveChatModel, parseModels3, parseActiveChatModel)
		if parseActiveProvider.ID != "anthropic" {
			parseT4.Fatalf("active provider = %q, want anthropic", parseActiveProvider.ID)
		}
	})

	parseT.Run("does not use non-openai speech models for fallback", func(parseT5 *testing.T) {
		parseT5.Parallel()
		parseModels4 := []modelOption{
			{ID: "claude-4", Capabilities: modelCapabilities{ProviderID: "anthropic", SupportsSpeech: false}},
			{ID: "cerebras-voice", Capabilities: modelCapabilities{ProviderID: "cerebras", SupportsSpeech: true}},
		}

		parseGotModel4, parseSupported4 := parseResolveSpeechSynthesisModel("claude-4", parseModels4, "claude-4", true)
		if parseSupported4 || parseGotModel4 != "" {
			parseT5.Fatalf("resolveSpeechSynthesisModel() = (%q, %v), want (\"\", false)", parseGotModel4, parseSupported4)
		}
	})
}

func TestCanvasPreviewFromMarkdownIgnoresInvalidFencesAndUsesLatestCompletedCanvas(parseT *testing.T) {
	parseT.Parallel()

	parseMarkdown := strings.Join([]string{
		"```go",
		`fmt.Println("ignore")`,
		"```",
		"",
		"```canvas",
		"",
		"```",
		"",
		"```canvas preview",
		"<div>latest</div>",
		"```",
	}, "\n")

	parsePreview, parseOk := canvasPreviewFromMarkdown(parseMarkdown)
	if !parseOk {
		parseT.Fatal("canvasPreviewFromMarkdown() = not found, want preview")
	}
	if parsePreview.Source != "<div>latest</div>" {
		parseT.Fatalf("preview.Source = %q, want latest canvas block", parsePreview.Source)
	}

	parseStatefulPreview, parseOk := parseLatestCanvasPreview([]message{
		{Role: roleAssistant, Pending: true, Content: "```canvas\n<div>pending</div>\n```"},
		{Role: roleAssistant, Content: "```canvas\n<div>stable</div>\n```"},
	})
	if !parseOk {
		parseT.Fatal("latestCanvasPreview() = not found, want completed preview")
	}
	if parseStatefulPreview.Source != "<div>stable</div>" {
		parseT.Fatalf("latestCanvasPreview() picked %q, want completed message preview", parseStatefulPreview.Source)
	}
}

func TestBuildCanvasDocumentHandlesScriptOnlyAndFullHTML(parseT *testing.T) {
	parseT.Parallel()

	parseScriptDoc := buildCanvasDocument(`console.log("hi")`)
	if !strings.Contains(parseScriptDoc, `<div id="canvas"></div>`) {
		parseT.Fatalf("script document missing #canvas root: %q", parseScriptDoc)
	}
	if !strings.Contains(parseScriptDoc, `<script>console.log("hi")</script>`) {
		parseT.Fatalf("script document missing wrapped script: %q", parseScriptDoc)
	}

	parseFullHTML := "<!DOCTYPE html><html><body><main>ready</main></body></html>"
	if parseGot := buildCanvasDocument(parseFullHTML); !strings.Contains(parseGot, "<main>ready</main>") || !strings.Contains(parseGot, "__gwcCanvas") {
		parseT.Fatalf("buildCanvasDocument(full html) missing preserved content or runtime bridge: got %q", parseGot)
	}
}

func TestCanvasArtifactsFromMarkdownRecognizesHTMLAndJavaScript(parseT *testing.T) {
	parseT.Parallel()

	parseMarkdown := strings.Join([]string{
		"```html",
		"<main>ready</main>",
		"```",
		"",
		"```javascript",
		`console.log("hi")`,
		"```",
	}, "\n")

	parseArtifacts := canvasArtifactsFromMarkdown(4, parseMarkdown)
	if len(parseArtifacts) != 2 {
		parseT.Fatalf("len(canvasArtifactsFromMarkdown()) = %d, want 2", len(parseArtifacts))
	}
	if parseArtifacts[0].Language != "html" {
		parseT.Fatalf("artifacts[0].Language = %q, want html", parseArtifacts[0].Language)
	}
	if parseArtifacts[1].Language != "javascript" {
		parseT.Fatalf("artifacts[1].Language = %q, want javascript", parseArtifacts[1].Language)
	}
}

func TestBuildCanvasRuntimeDocumentInjectsBridge(parseT *testing.T) {
	parseT.Parallel()

	parseDocument := buildCanvasRuntimeDocument(`<main>ok</main>`, "session-1", "artifact-1", 7)
	if !strings.Contains(parseDocument, "__gwcCanvas") {
		parseT.Fatalf("buildCanvasRuntimeDocument() missing bridge marker: %q", parseDocument)
	}
	if !strings.Contains(parseDocument, `"session-1"`) {
		parseT.Fatalf("buildCanvasRuntimeDocument() missing session id: %q", parseDocument)
	}
}

func TestDeriveThreadCostSummaryMarksPartialExactCoverage(parseT *testing.T) {
	parseT.Parallel()

	parseSummary := parseDeriveThreadCostSummary([]message{
		{Role: roleAssistant, Content: "priced", ModelID: "gpt-5.4-mini", PromptTokens: 1000, CompletionTokens: 500},
		{Role: roleAssistant, Content: "missing usage", ModelID: "gpt-5.4-mini", PromptTokens: 0, CompletionTokens: 0},
		{Role: roleAssistant, Pending: true, Content: "pending should be ignored", ModelID: "gpt-5.4-mini", PromptTokens: 999, CompletionTokens: 999},
	}, testAvailableModels)

	if !parseSummary.HasAnyExactCosts {
		parseT.Fatal("HasAnyExactCosts = false, want true")
	}
	if parseSummary.AllAssistantCostsExact {
		parseT.Fatal("AllAssistantCostsExact = true, want false for partial coverage")
	}
	if len(parseSummary.AssistantMessageCosts) != 1 {
		parseT.Fatalf("len(AssistantMessageCosts) = %d, want 1", len(parseSummary.AssistantMessageCosts))
	}

	parseGotCost := parseSummary.AssistantMessageCosts[0].Cost
	parseWantCost := (1000 * 0.25 / 1_000_000) + (500 * 2.00 / 1_000_000)
	if math.Abs(parseGotCost-parseWantCost) > 1e-12 {
		parseT.Fatalf("cost = %.12f, want %.12f", parseGotCost, parseWantCost)
	}
}

func TestExactAssistantMessageCostAllowsCompletionOnlyUsage(parseT *testing.T) {
	parseT.Parallel()

	parseCost, parseOk := parseExactAssistantMessageCost("gpt-5.4-mini", testAvailableModels, 0, 500)
	if !parseOk {
		parseT.Fatal("exactAssistantMessageCost() = not exact, want exact for completion-only usage")
	}

	parseWantCost := 500 * 2.00 / 1_000_000
	if math.Abs(parseCost.Cost-parseWantCost) > 1e-12 {
		parseT.Fatalf("cost = %.12f, want %.12f", parseCost.Cost, parseWantCost)
	}
}

func TestDeriveAccountCostSummaryAppliesPremiumAndTracksCoverage(parseT *testing.T) {
	parseT.Parallel()

	parseThreadA := threadCostSummary{
		TotalCost:              0.125,
		HasAnyExactCosts:       true,
		AllAssistantCostsExact: true,
	}
	parseThreadB := threadCostSummary{
		TotalCost:              0.375,
		HasAnyExactCosts:       true,
		AllAssistantCostsExact: false,
	}
	parseSummary := parseDeriveAccountCostSummary([]threadCostSummary{parseThreadA, parseThreadB}, 5, 1)

	if parseSummary.ThreadCount != 3 {
		parseT.Fatalf("ThreadCount = %d, want 3", parseSummary.ThreadCount)
	}
	if !parseSummary.HasAnyExactCosts {
		parseT.Fatal("HasAnyExactCosts = false, want true")
	}
	if parseSummary.ExactThreadCostCount != 2 {
		parseT.Fatalf("ExactThreadCostCount = %d, want 2", parseSummary.ExactThreadCostCount)
	}
	if parseSummary.AllThreadCostsExact {
		parseT.Fatal("AllThreadCostsExact = true, want false due to partial + failed lookups")
	}
	if !parseSummary.HasCoverageGaps {
		parseT.Fatal("HasCoverageGaps = false, want true")
	}
	if parseSummary.FailedThreadLookups != 1 {
		parseT.Fatalf("FailedThreadLookups = %d, want 1", parseSummary.FailedThreadLookups)
	}
	if math.Abs(parseSummary.UsageCost-0.5) > 1e-12 {
		parseT.Fatalf("UsageCost = %.12f, want 0.500000000000", parseSummary.UsageCost)
	}
	if math.Abs(parseSummary.PremiumCost-0.025) > 1e-12 {
		parseT.Fatalf("PremiumCost = %.12f, want 0.025000000000", parseSummary.PremiumCost)
	}
	if math.Abs(parseSummary.TotalCost-0.525) > 1e-12 {
		parseT.Fatalf("TotalCost = %.12f, want 0.525000000000", parseSummary.TotalCost)
	}
}

func TestSanitizeUsagePremiumPercentGuardsInvalidInput(parseT *testing.T) {
	parseT.Parallel()

	if parseGot := parseSanitizeUsagePremiumPercent(7.5, 5); parseGot != 7.5 {
		parseT.Fatalf("sanitizeUsagePremiumPercent(valid) = %.2f, want 7.50", parseGot)
	}
	if parseGot2 := parseSanitizeUsagePremiumPercent(-2, 5); parseGot2 != 5 {
		parseT.Fatalf("sanitizeUsagePremiumPercent(negative) = %.2f, want fallback 5.00", parseGot2)
	}
	if parseGot3 := parseSanitizeUsagePremiumPercent(math.Inf(1), 5); parseGot3 != 5 {
		parseT.Fatalf("sanitizeUsagePremiumPercent(inf) = %.2f, want fallback 5.00", parseGot3)
	}
	if parseGot4 := parseSanitizeUsagePremiumPercent(math.NaN(), 5); parseGot4 != 5 {
		parseT.Fatalf("sanitizeUsagePremiumPercent(nan) = %.2f, want fallback 5.00", parseGot4)
	}
	if parseGot5 := parseSanitizeUsagePremiumPercent(5000, 5); parseGot5 != 1000 {
		parseT.Fatalf("sanitizeUsagePremiumPercent(clamp) = %.2f, want 1000.00", parseGot5)
	}
}
