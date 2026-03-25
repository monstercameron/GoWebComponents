//go:build js && wasm

package app

import (
	"math"
	"strings"
	"testing"
)

func TestNormalizeSelectedModelIDHandlesAliasesAndFallbacks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		modelID  string
		models   []modelOption
		fallback string
		want     string
	}{
		{
			name:     "maps dated alias to stable model",
			modelID:  "gpt-5.4-mini-2026-03-17",
			models:   availableModels,
			fallback: defaultModel,
			want:     "gpt-5.4-mini",
		},
		{
			name:     "uses fallback when model is unknown",
			modelID:  "does-not-exist",
			models:   availableModels,
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
			name:     "uses built in catalog when models are empty",
			modelID:  "",
			models:   nil,
			fallback: "",
			want:     defaultModel,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeSelectedModelID(tt.modelID, tt.models, tt.fallback); got != tt.want {
				t.Fatalf("normalizeSelectedModelID(%q) = %q, want %q", tt.modelID, got, tt.want)
			}
		})
	}
}

func TestSelectedModelForConversationPrefersLatestSwitchThenAssistant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []message
		want     string
	}{
		{
			name: "latest switch wins over earlier assistant model",
			messages: []message{
				{Role: roleAssistant, ModelID: "gpt-5.4"},
				{Role: roleSwitch, Content: "gpt-5.4-nano"},
			},
			want: "gpt-5.4-nano",
		},
		{
			name: "blank switch falls back to latest assistant model",
			messages: []message{
				{Role: roleAssistant, ModelID: "gpt-5.4"},
				{Role: roleSwitch, Content: "   "},
			},
			want: "gpt-5.4",
		},
		{
			name: "unknown assistant model falls back to default",
			messages: []message{
				{Role: roleAssistant, ModelID: "unknown-model"},
			},
			want: defaultModel,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := selectedModelForConversation(tt.messages, availableModels, defaultModel); got != tt.want {
				t.Fatalf("selectedModelForConversation() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeSelectedThinkingEffortHandlesCaseWhitespaceAndInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: " HIGH ", want: "high"},
		{input: "", want: defaultThinkingEffort},
		{input: "unknown", want: defaultThinkingEffort},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := normalizeSelectedThinkingEffort(tt.input); got != tt.want {
				t.Fatalf("normalizeSelectedThinkingEffort(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestProviderHelpersGroupModelsAndChooseProviderDefault(t *testing.T) {
	t.Parallel()

	models := []modelOption{
		{ID: "openai-fast", Label: "OpenAI Fast", Capabilities: modelCapabilities{ProviderID: "openai", ProviderLabel: "OpenAI"}},
		{ID: "openai-best", Label: "OpenAI Best", Capabilities: modelCapabilities{ProviderID: "openai", ProviderLabel: "OpenAI"}},
		{ID: "cerebras-code", Label: "Cerebras Code", Capabilities: modelCapabilities{ProviderID: "cerebras", ProviderLabel: "Cerebras"}},
	}

	providers := providerOptionsForModels(models)
	if len(providers) != 2 {
		t.Fatalf("len(providerOptionsForModels()) = %d, want 2", len(providers))
	}
	if providers[0].ID != "openai" || providers[1].ID != "cerebras" {
		t.Fatalf("provider ordering = %#v, want openai then cerebras", providers)
	}

	activeProvider := providerForModel("cerebras-code", models, "openai-fast")
	if activeProvider.ID != "cerebras" {
		t.Fatalf("providerForModel() = %#v, want cerebras", activeProvider)
	}

	filteredModels := modelsForProvider(models, "openai")
	if len(filteredModels) != 2 {
		t.Fatalf("len(modelsForProvider(openai)) = %d, want 2", len(filteredModels))
	}

	if got := defaultModelForProvider("cerebras", models, "openai-best"); got != "cerebras-code" {
		t.Fatalf("defaultModelForProvider(cerebras) = %q, want cerebras-code", got)
	}
}

func TestCanvasPreviewFromMarkdownIgnoresInvalidFencesAndUsesLatestCompletedCanvas(t *testing.T) {
	t.Parallel()

	markdown := strings.Join([]string{
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

	preview, ok := canvasPreviewFromMarkdown(markdown)
	if !ok {
		t.Fatal("canvasPreviewFromMarkdown() = not found, want preview")
	}
	if preview.Source != "<div>latest</div>" {
		t.Fatalf("preview.Source = %q, want latest canvas block", preview.Source)
	}

	statefulPreview, ok := latestCanvasPreview([]message{
		{Role: roleAssistant, Pending: true, Content: "```canvas\n<div>pending</div>\n```"},
		{Role: roleAssistant, Content: "```canvas\n<div>stable</div>\n```"},
	})
	if !ok {
		t.Fatal("latestCanvasPreview() = not found, want completed preview")
	}
	if statefulPreview.Source != "<div>stable</div>" {
		t.Fatalf("latestCanvasPreview() picked %q, want completed message preview", statefulPreview.Source)
	}
}

func TestBuildCanvasDocumentHandlesScriptOnlyAndFullHTML(t *testing.T) {
	t.Parallel()

	scriptDoc := buildCanvasDocument(`console.log("hi")`)
	if !strings.Contains(scriptDoc, `<div id="canvas"></div>`) {
		t.Fatalf("script document missing #canvas root: %q", scriptDoc)
	}
	if !strings.Contains(scriptDoc, `<script>console.log("hi")</script>`) {
		t.Fatalf("script document missing wrapped script: %q", scriptDoc)
	}

	fullHTML := "<!DOCTYPE html><html><body><main>ready</main></body></html>"
	if got := buildCanvasDocument(fullHTML); !strings.Contains(got, "<main>ready</main>") || !strings.Contains(got, "__gwcCanvas") {
		t.Fatalf("buildCanvasDocument(full html) missing preserved content or runtime bridge: got %q", got)
	}
}

func TestCanvasArtifactsFromMarkdownRecognizesHTMLAndJavaScript(t *testing.T) {
	t.Parallel()

	markdown := strings.Join([]string{
		"```html",
		"<main>ready</main>",
		"```",
		"",
		"```javascript",
		`console.log("hi")`,
		"```",
	}, "\n")

	artifacts := canvasArtifactsFromMarkdown(4, markdown)
	if len(artifacts) != 2 {
		t.Fatalf("len(canvasArtifactsFromMarkdown()) = %d, want 2", len(artifacts))
	}
	if artifacts[0].Language != "html" {
		t.Fatalf("artifacts[0].Language = %q, want html", artifacts[0].Language)
	}
	if artifacts[1].Language != "javascript" {
		t.Fatalf("artifacts[1].Language = %q, want javascript", artifacts[1].Language)
	}
}

func TestBuildCanvasRuntimeDocumentInjectsBridge(t *testing.T) {
	t.Parallel()

	document := buildCanvasRuntimeDocument(`<main>ok</main>`, "session-1", "artifact-1", 7)
	if !strings.Contains(document, "__gwcCanvas") {
		t.Fatalf("buildCanvasRuntimeDocument() missing bridge marker: %q", document)
	}
	if !strings.Contains(document, `"session-1"`) {
		t.Fatalf("buildCanvasRuntimeDocument() missing session id: %q", document)
	}
}

func TestDeriveThreadCostSummaryMarksPartialExactCoverage(t *testing.T) {
	t.Parallel()

	summary := deriveThreadCostSummary([]message{
		{Role: roleAssistant, Content: "priced", ModelID: "gpt-5.4-mini", PromptTokens: 1000, CompletionTokens: 500},
		{Role: roleAssistant, Content: "missing usage", ModelID: "gpt-5.4-mini", PromptTokens: 0, CompletionTokens: 500},
		{Role: roleAssistant, Pending: true, Content: "pending should be ignored", ModelID: "gpt-5.4-mini", PromptTokens: 999, CompletionTokens: 999},
	})

	if !summary.HasAnyExactCosts {
		t.Fatal("HasAnyExactCosts = false, want true")
	}
	if summary.AllAssistantCostsExact {
		t.Fatal("AllAssistantCostsExact = true, want false for partial coverage")
	}
	if len(summary.AssistantMessageCosts) != 1 {
		t.Fatalf("len(AssistantMessageCosts) = %d, want 1", len(summary.AssistantMessageCosts))
	}

	gotCost := summary.AssistantMessageCosts[0].Cost
	wantCost := (1000 * 0.25 / 1_000_000) + (500 * 2.00 / 1_000_000)
	if math.Abs(gotCost-wantCost) > 1e-12 {
		t.Fatalf("cost = %.12f, want %.12f", gotCost, wantCost)
	}
}

func TestExactAssistantMessageCostAllowsCompletionOnlyUsage(t *testing.T) {
	t.Parallel()

	cost, ok := exactAssistantMessageCost("gpt-5.4-mini", 0, 500)
	if !ok {
		t.Fatal("exactAssistantMessageCost() = not exact, want exact for completion-only usage")
	}

	wantCost := 500 * 2.00 / 1_000_000
	if math.Abs(cost.Cost-wantCost) > 1e-12 {
		t.Fatalf("cost = %.12f, want %.12f", cost.Cost, wantCost)
	}
}
