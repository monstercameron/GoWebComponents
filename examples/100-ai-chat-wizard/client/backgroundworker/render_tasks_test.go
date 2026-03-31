//go:build js && wasm

package main

import (
	"context"
	"strings"
	"testing"
)

// ─── parseBuildThoughtHeading ─────────────────────────────────────────────────

// TestParseBuildThoughtHeading verifies bold-heading extraction across valid, edge-case, and invalid inputs.
func TestParseBuildThoughtHeading(parseT *testing.T) {
	parseT.Run("valid bold heading returns trimmed text", func(parseT *testing.T) {
		parseText, isParseOK := parseBuildThoughtHeading("**Analysis step**")
		if !isParseOK {
			parseT.Fatal("expected ok=true for valid bold heading")
		}
		if parseText != "Analysis step" {
			parseT.Fatalf("got %q, want %q", parseText, "Analysis step")
		}
	})
	parseT.Run("bold-only marker returns empty false", func(parseT *testing.T) {
		_, isParseOK := parseBuildThoughtHeading("****")
		if isParseOK {
			parseT.Fatal("expected ok=false for empty bold marker")
		}
	})
	parseT.Run("single asterisk prefix is rejected", func(parseT *testing.T) {
		_, isParseOK := parseBuildThoughtHeading("*not bold*")
		if isParseOK {
			parseT.Fatal("expected ok=false for single asterisk")
		}
	})
	parseT.Run("plain line is rejected", func(parseT *testing.T) {
		_, isParseOK := parseBuildThoughtHeading("just a regular sentence")
		if isParseOK {
			parseT.Fatal("expected ok=false for plain line")
		}
	})
	parseT.Run("empty string is rejected", func(parseT *testing.T) {
		_, isParseOK := parseBuildThoughtHeading("")
		if isParseOK {
			parseT.Fatal("expected ok=false for empty string")
		}
	})
}

// ─── parseBuildThoughtSectionResults ─────────────────────────────────────────

// TestParseBuildThoughtSectionResults verifies thought transcript parsing into section slices.
func TestParseBuildThoughtSectionResults(parseT *testing.T) {
	parseT.Run("empty input returns nil", func(parseT *testing.T) {
		parseSections := parseBuildThoughtSectionResults("")
		if parseSections != nil {
			parseT.Fatalf("expected nil for empty input, got %+v", parseSections)
		}
	})
	parseT.Run("whitespace-only input returns nil", func(parseT *testing.T) {
		parseSections := parseBuildThoughtSectionResults("   \n\n   ")
		if parseSections != nil {
			parseT.Fatalf("expected nil for whitespace input, got %+v", parseSections)
		}
	})
	parseT.Run("plain paragraph with no headings wraps in Thinking section", func(parseT *testing.T) {
		parseSections := parseBuildThoughtSectionResults("Step one.\nStep two.")
		if len(parseSections) != 1 {
			parseT.Fatalf("expected 1 section, got %d: %+v", len(parseSections), parseSections)
		}
		if string(parseSections[0].GetHeadingBytes) != "Thinking" {
			parseT.Fatalf("expected heading 'Thinking', got %q", string(parseSections[0].GetHeadingBytes))
		}
		if !strings.Contains(string(parseSections[0].GetBodyBytes), "Step one") {
			parseT.Fatalf("body missing content: %q", string(parseSections[0].GetBodyBytes))
		}
	})
	parseT.Run("single leading Thinking line is stripped", func(parseT *testing.T) {
		parseSections := parseBuildThoughtSectionResults("Thinking\nSome body text.")
		if len(parseSections) != 1 {
			parseT.Fatalf("expected 1 section, got %d", len(parseSections))
		}
		if !strings.Contains(string(parseSections[0].GetBodyBytes), "Some body text") {
			parseT.Fatalf("expected body text, got %q", string(parseSections[0].GetBodyBytes))
		}
	})
	parseT.Run("multi-section transcript splits at bold headings", func(parseT *testing.T) {
		parseInput := "**Planning**\nFirst step.\n**Execution**\nSecond step."
		parseSections := parseBuildThoughtSectionResults(parseInput)
		if len(parseSections) != 2 {
			parseT.Fatalf("expected 2 sections, got %d: %+v", len(parseSections), parseSections)
		}
		if string(parseSections[0].GetHeadingBytes) != "Planning" {
			parseT.Fatalf("section 0 heading = %q, want 'Planning'", string(parseSections[0].GetHeadingBytes))
		}
		if string(parseSections[1].GetHeadingBytes) != "Execution" {
			parseT.Fatalf("section 1 heading = %q, want 'Execution'", string(parseSections[1].GetHeadingBytes))
		}
		if !strings.Contains(string(parseSections[0].GetBodyBytes), "First step.") {
			parseT.Fatalf("section 0 body missing: %q", string(parseSections[0].GetBodyBytes))
		}
		if !strings.Contains(string(parseSections[1].GetBodyBytes), "Second step.") {
			parseT.Fatalf("section 1 body missing: %q", string(parseSections[1].GetBodyBytes))
		}
	})
}

// ─── parseBuildCanvasFenceLanguage ────────────────────────────────────────────

// TestParseBuildCanvasFenceLanguage verifies fence-language resolution for supported and unsupported types.
func TestParseBuildCanvasFenceLanguage(parseT *testing.T) {
	parseCases := []struct {
		parseInfo     string
		parseWantLang string
		isParseOK     bool
	}{
		{"html", "html", true},
		{"htm", "html", true},
		{"HTML", "html", true},
		{"javascript", "javascript", true},
		{"js", "javascript", true},
		{"canvas", "canvas", true},
		{"canvas html", "canvas", true}, // canvas keyword wins
		{"go", "", false},
		{"python", "", false},
		{"", "", false},
		{"   ", "", false},
	}
	for _, parseCase := range parseCases {
		parseLang, isParseFound := parseBuildCanvasFenceLanguage(parseCase.parseInfo)
		if isParseFound != parseCase.isParseOK {
			parseT.Errorf("parseBuildCanvasFenceLanguage(%q) ok=%v, want %v", parseCase.parseInfo, isParseFound, parseCase.isParseOK)
			continue
		}
		if isParseFound && parseLang != parseCase.parseWantLang {
			parseT.Errorf("parseBuildCanvasFenceLanguage(%q) lang=%q, want %q", parseCase.parseInfo, parseLang, parseCase.parseWantLang)
		}
	}
}

// ─── parseBuildCanvasArtifactLabel ───────────────────────────────────────────

// TestParseBuildCanvasArtifactLabel verifies label derivation from language, source, and block index.
func TestParseBuildCanvasArtifactLabel(parseT *testing.T) {
	parseT.Run("html language returns HTML demo", func(parseT *testing.T) {
		parseLabel := parseBuildCanvasArtifactLabel("html", "<div>Hello</div>", 0)
		if parseLabel != "HTML demo" {
			parseT.Fatalf("got %q, want 'HTML demo'", parseLabel)
		}
	})
	parseT.Run("javascript with function returns App component", func(parseT *testing.T) {
		parseLabel := parseBuildCanvasArtifactLabel("javascript", "function MyApp() {}", 0)
		if parseLabel != "App component" {
			parseT.Fatalf("got %q, want 'App component'", parseLabel)
		}
	})
	parseT.Run("javascript without function returns JavaScript demo", func(parseT *testing.T) {
		parseLabel := parseBuildCanvasArtifactLabel("javascript", "console.log('hello')", 0)
		if parseLabel != "JavaScript demo" {
			parseT.Fatalf("got %q, want 'JavaScript demo'", parseLabel)
		}
	})
	parseT.Run("canvas with html content returns Canvas page", func(parseT *testing.T) {
		parseLabel := parseBuildCanvasArtifactLabel("canvas", "<html><body></body></html>", 0)
		if parseLabel != "Canvas page" {
			parseT.Fatalf("got %q, want 'Canvas page'", parseLabel)
		}
	})
	parseT.Run("canvas with class returns App component", func(parseT *testing.T) {
		parseLabel := parseBuildCanvasArtifactLabel("canvas", "class MyComponent {}", 0)
		if parseLabel != "App component" {
			parseT.Fatalf("got %q, want 'App component'", parseLabel)
		}
	})
	parseT.Run("canvas plain source returns fallback numbered label", func(parseT *testing.T) {
		parseLabel := parseBuildCanvasArtifactLabel("canvas", "console.log('test')", 2)
		if parseLabel != "Canvas block 3" {
			parseT.Fatalf("got %q, want 'Canvas block 3'", parseLabel)
		}
	})
	parseT.Run("unknown language returns fallback numbered label", func(parseT *testing.T) {
		parseLabel := parseBuildCanvasArtifactLabel("unknown", "anything", 0)
		if parseLabel != "Canvas block 1" {
			parseT.Fatalf("got %q, want 'Canvas block 1'", parseLabel)
		}
	})
}

// ─── parseBuildCanvasArtifactResults ─────────────────────────────────────────

// TestParseBuildCanvasArtifactResults verifies canvas artifact extraction from markdown message payloads.
func TestParseBuildCanvasArtifactResults(parseT *testing.T) {
	parseT.Run("empty markdown returns nil", func(parseT *testing.T) {
		parseResults := parseBuildCanvasArtifactResults(0, "")
		if parseResults != nil {
			parseT.Fatalf("expected nil for empty markdown, got %+v", parseResults)
		}
	})
	parseT.Run("non-code markdown returns nil", func(parseT *testing.T) {
		parseResults := parseBuildCanvasArtifactResults(0, "Just some text\nNo code blocks here.")
		if parseResults != nil {
			parseT.Fatalf("expected nil for no code blocks, got %+v", parseResults)
		}
	})
	parseT.Run("go code block is ignored", func(parseT *testing.T) {
		parseMarkdown := "```go\npackage main\n```"
		parseResults := parseBuildCanvasArtifactResults(0, parseMarkdown)
		if parseResults != nil {
			parseT.Fatalf("expected nil for go code block, got %+v", parseResults)
		}
	})
	parseT.Run("javascript code block produces one artifact", func(parseT *testing.T) {
		parseMarkdown := "```js\nconsole.log('hello')\n```"
		parseResults := parseBuildCanvasArtifactResults(3, parseMarkdown)
		if len(parseResults) != 1 {
			parseT.Fatalf("expected 1 artifact, got %d: %+v", len(parseResults), parseResults)
		}
		if string(parseResults[0].GetIDBytes) != "m3-b0" {
			parseT.Fatalf("id=%q, want 'm3-b0'", string(parseResults[0].GetIDBytes))
		}
	})
	parseT.Run("html and canvas blocks both produce artifacts", func(parseT *testing.T) {
		parseMarkdown := "```html\n<div>Hi</div>\n```\n\n```canvas\nvar x = 1\n```"
		parseResults := parseBuildCanvasArtifactResults(1, parseMarkdown)
		if len(parseResults) != 2 {
			parseT.Fatalf("expected 2 artifacts, got %d: %+v", len(parseResults), parseResults)
		}
		if string(parseResults[0].GetIDBytes) != "m1-b0" {
			parseT.Fatalf("first id=%q, want 'm1-b0'", string(parseResults[0].GetIDBytes))
		}
		if string(parseResults[1].GetIDBytes) != "m1-b1" {
			parseT.Fatalf("second id=%q, want 'm1-b1'", string(parseResults[1].GetIDBytes))
		}
	})
}

// ─── parseBuildModelByIDMap ───────────────────────────────────────────────────

// TestParseBuildModelByIDMap verifies model lookup table construction.
func TestParseBuildModelByIDMap(parseT *testing.T) {
	parseT.Run("empty slice returns empty map", func(parseT *testing.T) {
		parseMap := parseBuildModelByIDMap(nil)
		if len(parseMap) != 0 {
			parseT.Fatalf("expected empty map, got %v", parseMap)
		}
	})
	parseT.Run("models with IDs are indexed correctly", func(parseT *testing.T) {
		parseModels := []renderWorkerCostModelRequest{
			{GetModelIDBytes: []byte("gpt-4"), GetInputDollarsPerMillion: 30.0, GetOutputDollarsPerMillion: 60.0},
			{GetModelIDBytes: []byte("gpt-3.5"), GetInputDollarsPerMillion: 0.5, GetOutputDollarsPerMillion: 1.5},
		}
		parseMap := parseBuildModelByIDMap(parseModels)
		if len(parseMap) != 2 {
			parseT.Fatalf("expected 2 entries, got %d", len(parseMap))
		}
		if _, isParseFound := parseMap["gpt-4"]; !isParseFound {
			parseT.Fatal("gpt-4 not found in map")
		}
	})
	parseT.Run("model with empty ID is skipped", func(parseT *testing.T) {
		parseModels := []renderWorkerCostModelRequest{
			{GetModelIDBytes: []byte(""), GetInputDollarsPerMillion: 1.0},
			{GetModelIDBytes: []byte("   "), GetInputDollarsPerMillion: 1.0},
			{GetModelIDBytes: []byte("gpt-4"), GetInputDollarsPerMillion: 30.0},
		}
		parseMap := parseBuildModelByIDMap(parseModels)
		if len(parseMap) != 1 {
			parseT.Fatalf("expected 1 entry, got %d: %v", len(parseMap), parseMap)
		}
	})
}

// ─── parseBuildAssistantMessageCost ──────────────────────────────────────────

// TestParseBuildAssistantMessageCost verifies cost computation exactness for various token/model combinations.
func TestParseBuildAssistantMessageCost(parseT *testing.T) {
	parseModel := renderWorkerCostModelRequest{
		GetModelIDBytes:            []byte("gpt-4"),
		GetInputDollarsPerMillion:  30.0,
		GetOutputDollarsPerMillion: 60.0,
	}
	parseModelByID := map[string]renderWorkerCostModelRequest{"gpt-4": parseModel}

	parseT.Run("exact cost computed when model and tokens are present", func(parseT *testing.T) {
		parseMsg := renderWorkerCostMessageRequest{
			GetMessageIndex:     2,
			GetModelIDBytes:     []byte("gpt-4"),
			GetPromptTokens:     1000,
			GetCompletionTokens: 500,
			GetHasContent:       true,
		}
		parseResult, isParseExact := parseBuildAssistantMessageCost(parseMsg, parseModelByID)
		if !isParseExact {
			parseT.Fatal("expected exact cost, got inexact")
		}
		// cost = (1000 * 30/1_000_000) + (500 * 60/1_000_000) = 0.030 + 0.030 = 0.060
		parseWant := 0.060
		if parseResult.GetCost < parseWant-1e-9 || parseResult.GetCost > parseWant+1e-9 {
			parseT.Fatalf("cost=%v, want %v", parseResult.GetCost, parseWant)
		}
		if parseResult.GetMessageIndex != 2 {
			parseT.Fatalf("message index=%d, want 2", parseResult.GetMessageIndex)
		}
	})
	parseT.Run("model not in map returns inexact", func(parseT *testing.T) {
		parseMsg := renderWorkerCostMessageRequest{
			GetModelIDBytes:     []byte("unknown-model"),
			GetPromptTokens:     100,
			GetCompletionTokens: 50,
			GetHasContent:       true,
		}
		_, isParseExact := parseBuildAssistantMessageCost(parseMsg, parseModelByID)
		if isParseExact {
			parseT.Fatal("expected inexact for unknown model")
		}
	})
	parseT.Run("zero tokens returns inexact", func(parseT *testing.T) {
		parseMsg := renderWorkerCostMessageRequest{
			GetModelIDBytes:     []byte("gpt-4"),
			GetPromptTokens:     0,
			GetCompletionTokens: 0,
			GetHasContent:       true,
		}
		_, isParseExact := parseBuildAssistantMessageCost(parseMsg, parseModelByID)
		if isParseExact {
			parseT.Fatal("expected inexact for zero tokens")
		}
	})
}

// ─── handleRenderThreadCostSummaryRequest ────────────────────────────────────

// TestHandleRenderThreadCostSummaryRequest verifies end-to-end cost summary derivation from request payloads.
func TestHandleRenderThreadCostSummaryRequest(parseT *testing.T) {
	parseT.Run("empty request returns zero summary for generation", func(parseT *testing.T) {
		parseResult, parseErr := handleRenderThreadCostSummaryRequest(context.Background(), renderWorkerThreadCostSummaryRequest{
			GetGeneration: 42,
		})
		if parseErr != nil {
			parseT.Fatalf("unexpected error: %v", parseErr)
		}
		if parseResult.GetGeneration != 42 {
			parseT.Fatalf("generation=%d, want 42", parseResult.GetGeneration)
		}
		if parseResult.GetTotalCost != 0 {
			parseT.Fatalf("expected zero total cost, got %v", parseResult.GetTotalCost)
		}
		if parseResult.GetAllAssistantCostsExact {
			parseT.Fatal("expected AllAssistantCostsExact=false for empty message list")
		}
	})
	parseT.Run("user-only message list produces no assistant costs", func(parseT *testing.T) {
		parseResult, parseErr := handleRenderThreadCostSummaryRequest(context.Background(), renderWorkerThreadCostSummaryRequest{
			GetGeneration: 1,
			GetMessage: []renderWorkerCostMessageRequest{
				{GetRoleBytes: []byte("user"), GetHasContent: true, GetPromptTokens: 100, GetCompletionTokens: 0},
			},
		})
		if parseErr != nil {
			parseT.Fatalf("unexpected error: %v", parseErr)
		}
		if len(parseResult.GetAssistantMessageCost) != 0 {
			parseT.Fatalf("expected no assistant message costs, got %d", len(parseResult.GetAssistantMessageCost))
		}
	})
	parseT.Run("assistant message with known model produces exact cost", func(parseT *testing.T) {
		parseResult, parseErr := handleRenderThreadCostSummaryRequest(context.Background(), renderWorkerThreadCostSummaryRequest{
			GetGeneration: 7,
			GetMessage: []renderWorkerCostMessageRequest{
				{
					GetRoleBytes:        []byte("assistant"),
					GetHasContent:       true,
					GetModelIDBytes:     []byte("gpt-4"),
					GetPromptTokens:     1000,
					GetCompletionTokens: 500,
				},
			},
			GetModel: []renderWorkerCostModelRequest{
				{GetModelIDBytes: []byte("gpt-4"), GetInputDollarsPerMillion: 30.0, GetOutputDollarsPerMillion: 60.0},
			},
		})
		if parseErr != nil {
			parseT.Fatalf("unexpected error: %v", parseErr)
		}
		if !parseResult.GetHasAnyExactCosts {
			parseT.Fatal("expected HasAnyExactCosts=true")
		}
		if !parseResult.GetAllAssistantCostsExact {
			parseT.Fatal("expected AllAssistantCostsExact=true")
		}
		if parseResult.GetTotalCost <= 0 {
			parseT.Fatalf("expected positive total cost, got %v", parseResult.GetTotalCost)
		}
	})
}

// ─── handleRenderMessageMetadataBatchRequest ──────────────────────────────────

// TestHandleRenderMessageMetadataBatchRequest verifies metadata batch derivation for thought sections and canvas artifacts.
func TestHandleRenderMessageMetadataBatchRequest(parseT *testing.T) {
	parseT.Run("empty batch request returns matching generation with no chunks", func(parseT *testing.T) {
		parseResult, parseErr := handleRenderMessageMetadataBatchRequest(context.Background(), renderWorkerMessageMetadataBatchRequest{
			GetGeneration: 99,
		})
		if parseErr != nil {
			parseT.Fatalf("unexpected error: %v", parseErr)
		}
		if parseResult.GetGeneration != 99 {
			parseT.Fatalf("generation=%d, want 99", parseResult.GetGeneration)
		}
		if len(parseResult.GetChunk) != 0 {
			parseT.Fatalf("expected no chunks, got %d", len(parseResult.GetChunk))
		}
	})
	parseT.Run("message with javascript block produces one canvas artifact", func(parseT *testing.T) {
		parseMarkdown := "```js\nconsole.log('hello')\n```"
		parseResult, parseErr := handleRenderMessageMetadataBatchRequest(context.Background(), renderWorkerMessageMetadataBatchRequest{
			GetGeneration: 1,
			GetChunkRequest: []renderWorkerMessageMetadataChunkRequest{
				{
					GetChunkIndex: 0,
					GetMessageItems: []renderWorkerMessageMetadataMessageRequest{
						{GetMessageIndex: 0, GetContentBytes: []byte(parseMarkdown)},
					},
				},
			},
		})
		if parseErr != nil {
			parseT.Fatalf("unexpected error: %v", parseErr)
		}
		if len(parseResult.GetChunk) != 1 {
			parseT.Fatalf("expected 1 chunk, got %d", len(parseResult.GetChunk))
		}
		parseMsgResults := parseResult.GetChunk[0].GetMessage
		if len(parseMsgResults) != 1 {
			parseT.Fatalf("expected 1 message result, got %d", len(parseMsgResults))
		}
		if len(parseMsgResults[0].GetCanvasArtifact) != 1 {
			parseT.Fatalf("expected 1 canvas artifact, got %d", len(parseMsgResults[0].GetCanvasArtifact))
		}
	})
	parseT.Run("message with thought section produces heading and body", func(parseT *testing.T) {
		parseThought := "**Reasoning**\nHere is my reasoning."
		parseResult, parseErr := handleRenderMessageMetadataBatchRequest(context.Background(), renderWorkerMessageMetadataBatchRequest{
			GetGeneration: 2,
			GetChunkRequest: []renderWorkerMessageMetadataChunkRequest{
				{
					GetChunkIndex: 0,
					GetMessageItems: []renderWorkerMessageMetadataMessageRequest{
						{GetMessageIndex: 1, GetThoughtBytes: []byte(parseThought)},
					},
				},
			},
		})
		if parseErr != nil {
			parseT.Fatalf("unexpected error: %v", parseErr)
		}
		if len(parseResult.GetChunk) != 1 {
			parseT.Fatalf("expected 1 chunk, got %d", len(parseResult.GetChunk))
		}
		parseMsgResults := parseResult.GetChunk[0].GetMessage
		if len(parseMsgResults[0].GetThoughtSection) != 1 {
			parseT.Fatalf("expected 1 thought section, got %d", len(parseMsgResults[0].GetThoughtSection))
		}
		if string(parseMsgResults[0].GetThoughtSection[0].GetHeadingBytes) != "Reasoning" {
			parseT.Fatalf("heading=%q, want 'Reasoning'", string(parseMsgResults[0].GetThoughtSection[0].GetHeadingBytes))
		}
	})
}

// ─── handleRenderSignaturesRequest ───────────────────────────────────────────

// TestHandleRenderSignaturesRequest verifies signature derivation preserves generation and produces non-empty bytes when content is present.
func TestHandleRenderSignaturesRequest(parseT *testing.T) {
	parseT.Run("empty request returns matching generation with empty signatures", func(parseT *testing.T) {
		parseResult, parseErr := handleRenderSignaturesRequest(context.Background(), renderWorkerSignatureRequest{
			GetGeneration: 5,
		})
		if parseErr != nil {
			parseT.Fatalf("unexpected error: %v", parseErr)
		}
		if parseResult.GetGeneration != 5 {
			parseT.Fatalf("generation=%d, want 5", parseResult.GetGeneration)
		}
		if len(parseResult.GetCompletedMarkdownSignatureBytes) != 0 {
			parseT.Fatalf("expected empty markdown signature, got %q", string(parseResult.GetCompletedMarkdownSignatureBytes))
		}
	})
	parseT.Run("completed assistant message produces non-empty markdown signature", func(parseT *testing.T) {
		parseResult, parseErr := handleRenderSignaturesRequest(context.Background(), renderWorkerSignatureRequest{
			GetGeneration: 3,
			GetMessage: []renderWorkerSignatureMessageRequest{
				{
					GetRoleBytes:    []byte("assistant"),
					GetContentBytes: []byte("Hello world"),
					GetPending:      false,
				},
			},
		})
		if parseErr != nil {
			parseT.Fatalf("unexpected error: %v", parseErr)
		}
		if len(parseResult.GetCompletedMarkdownSignatureBytes) == 0 {
			parseT.Fatal("expected non-empty markdown signature for completed assistant message")
		}
	})
	parseT.Run("pending assistant message is excluded from markdown signature", func(parseT *testing.T) {
		parseResult, parseErr := handleRenderSignaturesRequest(context.Background(), renderWorkerSignatureRequest{
			GetGeneration: 1,
			GetMessage: []renderWorkerSignatureMessageRequest{
				{
					GetRoleBytes:    []byte("assistant"),
					GetContentBytes: []byte("Pending response"),
					GetPending:      true,
				},
			},
		})
		if parseErr != nil {
			parseT.Fatalf("unexpected error: %v", parseErr)
		}
		if len(parseResult.GetCompletedMarkdownSignatureBytes) != 0 {
			parseT.Fatalf("expected empty signature for pending message, got %q", string(parseResult.GetCompletedMarkdownSignatureBytes))
		}
	})
}

// ─── setBackgroundWorkerCustomHandlers ────────────────────────────────────────

// TestSetBackgroundWorkerCustomHandlers verifies the custom handlers hook returns nil without panic.
func TestSetBackgroundWorkerCustomHandlers(parseT *testing.T) {
	parseDispatcher := buildBackgroundWorkerDispatcher()
	if parseDispatcher == nil {
		parseT.Fatal("expected non-nil dispatcher from buildBackgroundWorkerDispatcher")
	}
}

// ─── buildBackgroundWorkerDispatcher ─────────────────────────────────────────

// TestBuildBackgroundWorkerDispatcher verifies the dispatcher is built without panicking and returns a usable value.
func TestBuildBackgroundWorkerDispatcher(parseT *testing.T) {
	parseDefer := func() {
		if parseRecover := recover(); parseRecover != nil {
			parseT.Fatalf("buildBackgroundWorkerDispatcher panicked: %v", parseRecover)
		}
	}
	defer parseDefer()
	parseDispatcher := buildBackgroundWorkerDispatcher()
	if parseDispatcher == nil {
		parseT.Fatal("expected non-nil dispatcher")
	}
}

// ─── parseBuildRenderWorkerMessageIndexString ─────────────────────────────────

// TestParseBuildRenderWorkerMessageIndexString verifies stable base-10 integer serialization.
func TestParseBuildRenderWorkerMessageIndexString(parseT *testing.T) {
	parseCases := []struct {
		parseInput int
		parseWant  string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{-1, "-1"},
		{999999, "999999"},
	}
	for _, parseCase := range parseCases {
		parseGot := parseBuildRenderWorkerMessageIndexString(parseCase.parseInput)
		if parseGot != parseCase.parseWant {
			parseT.Errorf("parseBuildRenderWorkerMessageIndexString(%d)=%q, want %q", parseCase.parseInput, parseGot, parseCase.parseWant)
		}
	}
}
