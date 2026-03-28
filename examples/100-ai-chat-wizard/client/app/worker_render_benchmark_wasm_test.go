//go:build js && wasm
// +build js,wasm

package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/interop"
)

var parseRenderWorkerExample100BenchmarkSink int

// BenchmarkRenderWorkerExample100MainThreadMetadataAndCostLongThread measures synchronous metadata and cost derivation on one long thread.
func BenchmarkRenderWorkerExample100MainThreadMetadataAndCostLongThread(parseB *testing.B) {
	parseMessages, parseModels := parseBuildRenderWorkerExample100Fixture(180)
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseB.Loop() {
		parseClearRenderWorkerExample100Caches()
		parseThoughtSectionCount := 0
		parseCanvasArtifactCount := 0
		for parseMessageIndex, parseMessageItem := range parseMessages {
			if parseMessageItem.Role != roleAssistant || parseMessageItem.Pending {
				continue
			}
			parseThoughtSections := parseThoughtSections(parseMessageIndex, parseMessageItem.Thought)
			parseCanvasArtifacts := canvasArtifactsFromMarkdown(parseMessageIndex, parseMessageItem.Content)
			parseThoughtSectionCount += len(parseThoughtSections)
			parseCanvasArtifactCount += len(parseCanvasArtifacts)
		}
		parseSummary := parseDeriveThreadCostSummary(parseMessages, parseModels)
		if len(parseSummary.AssistantMessageCosts) == 0 {
			parseB.Fatal("parseDeriveThreadCostSummary returned no assistant costs")
		}
		parseRenderWorkerExample100BenchmarkSink = parseThoughtSectionCount + parseCanvasArtifactCount + len(parseSummary.AssistantMessageCosts)
	}
}

// BenchmarkRenderWorkerExample100WorkerPathMetadataAndCostLongThread measures worker-path request shape derivation on one long thread.
func BenchmarkRenderWorkerExample100WorkerPathMetadataAndCostLongThread(parseB *testing.B) {
	parseMessages, parseModels := parseBuildRenderWorkerExample100Fixture(180)
	parseRequester := renderWorkerExample100BenchmarkRequester{}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	var parseGeneration uint64
	for parseB.Loop() {
		parseClearRenderWorkerExample100Caches()
		parseGeneration++
		parseMessageItems := parseBuildAssistantMessageMetadataItems(parseMessages)
		parseThoughtCacheByMessage, parseCanvasCacheByMessage, parseErr := parseRequestAssistantMessageMetadata(
			context.Background(),
			parseRequester,
			parseGeneration,
			parseMessageItems,
			backgroundWorkerRenderPoolSize,
		)
		if parseErr != nil {
			parseB.Fatalf("parseRequestAssistantMessageMetadata: %v", parseErr)
		}
		parseSummary, parseErr2 := parseRequestWorkerThreadCostSummary(
			context.Background(),
			parseRequester,
			parseGeneration,
			parseMessages,
			parseModels,
		)
		if parseErr2 != nil {
			parseB.Fatalf("parseRequestWorkerThreadCostSummary: %v", parseErr2)
		}
		parseRenderWorkerExample100BenchmarkSink = len(parseThoughtCacheByMessage) + len(parseCanvasCacheByMessage) + len(parseSummary.AssistantMessageCosts)
	}
}

// BenchmarkRenderWorkerExample100WorkerPathMetadataAndCostLongThreadScale measures worker-path request shape derivation while scaling worker lanes from 1 through 4.
func BenchmarkRenderWorkerExample100WorkerPathMetadataAndCostLongThreadScale(parseB *testing.B) {
	parseMessages, parseModels := parseBuildRenderWorkerExample100Fixture(180)
	parseRequester := renderWorkerExample100BenchmarkRequester{}
	for parseWorkerCount := 1; parseWorkerCount <= 4; parseWorkerCount++ {
		parseWorkerCount := parseWorkerCount
		parseB.Run(fmt.Sprintf("workers_%d", parseWorkerCount), func(parseScaleB *testing.B) {
			parseScaleB.ReportAllocs()
			parseScaleB.ResetTimer()
			var parseGeneration uint64
			for parseScaleB.Loop() {
				parseClearRenderWorkerExample100Caches()
				parseGeneration++
				parseMessageItems := parseBuildAssistantMessageMetadataItems(parseMessages)
				parseThoughtCacheByMessage, parseCanvasCacheByMessage, parseErr := parseRequestAssistantMessageMetadata(
					context.Background(),
					parseRequester,
					parseGeneration,
					parseMessageItems,
					parseWorkerCount,
				)
				if parseErr != nil {
					parseScaleB.Fatalf("parseRequestAssistantMessageMetadata: %v", parseErr)
				}
				parseSummary, parseErr2 := parseRequestWorkerThreadCostSummary(
					context.Background(),
					parseRequester,
					parseGeneration,
					parseMessages,
					parseModels,
				)
				if parseErr2 != nil {
					parseScaleB.Fatalf("parseRequestWorkerThreadCostSummary: %v", parseErr2)
				}
				parseRenderWorkerExample100BenchmarkSink = len(parseThoughtCacheByMessage) + len(parseCanvasCacheByMessage) + len(parseSummary.AssistantMessageCosts)
			}
		})
	}
}

// parseBuildRenderWorkerExample100Fixture builds one deterministic long-thread benchmark fixture with alternating user/assistant messages.
func parseBuildRenderWorkerExample100Fixture(parseAssistantCount int) ([]message, []modelOption) {
	if parseAssistantCount < 1 {
		parseAssistantCount = 1
	}
	parseMessages := make([]message, 0, parseAssistantCount*2)
	for parseIndex := 0; parseIndex < parseAssistantCount; parseIndex++ {
		parseMessages = append(parseMessages, message{
			Role:    roleUser,
			Content: fmt.Sprintf("User prompt %d asks for code + reasoning details.", parseIndex+1),
		})
		parseMessages = append(parseMessages, message{
			Role:             roleAssistant,
			Content:          parseBuildRenderWorkerExample100AssistantContent(parseIndex),
			Thought:          parseBuildRenderWorkerExample100AssistantThought(parseIndex),
			ModelID:          parseBuildRenderWorkerExample100ModelID(parseIndex),
			PromptTokens:     1100 + (parseIndex % 170),
			CompletionTokens: 1700 + (parseIndex % 240),
		})
	}
	return parseMessages, parseBuildRenderWorkerExample100Models()
}

// parseBuildRenderWorkerExample100Models builds one deterministic model list with per-model pricing for cost derivation benchmarking.
func parseBuildRenderWorkerExample100Models() []modelOption {
	return []modelOption{
		{
			ID:           "gpt-5.4",
			Label:        "GPT-5.4",
			Capabilities: modelCapabilities{ProviderID: "openai", ProviderLabel: "OpenAI", SupportsThinking: true, SupportsSpeech: true},
			Pricing:      modelPricing{InputDollarsPerMillion: 1.25, OutputDollarsPerMillion: 10.00, Currency: "USD"},
		},
		{
			ID:           "gpt-5.4-mini",
			Label:        "GPT-5.4 mini",
			Capabilities: modelCapabilities{ProviderID: "openai", ProviderLabel: "OpenAI", SupportsThinking: true, SupportsSpeech: true},
			Pricing:      modelPricing{InputDollarsPerMillion: 0.25, OutputDollarsPerMillion: 2.00, Currency: "USD"},
		},
		{
			ID:           "gpt-5.4-nano",
			Label:        "GPT-5.4 nano",
			Capabilities: modelCapabilities{ProviderID: "openai", ProviderLabel: "OpenAI", SupportsThinking: true, SupportsSpeech: true},
			Pricing:      modelPricing{InputDollarsPerMillion: 0.05, OutputDollarsPerMillion: 0.40, Currency: "USD"},
		},
	}
}

// parseBuildRenderWorkerExample100ModelID picks one deterministic model ID for fixture variability.
func parseBuildRenderWorkerExample100ModelID(parseIndex int) string {
	switch parseIndex % 3 {
	case 0:
		return "gpt-5.4"
	case 1:
		return "gpt-5.4-mini"
	default:
		return "gpt-5.4-nano"
	}
}

// parseBuildRenderWorkerExample100AssistantContent builds one markdown-heavy assistant response with multiple code fences for canvas extraction work.
func parseBuildRenderWorkerExample100AssistantContent(parseIndex int) string {
	var parseBuilder strings.Builder
	parseBuilder.WriteString(fmt.Sprintf("### Response %d\n", parseIndex+1))
	parseBuilder.WriteString("Here is a practical implementation and explanation.\n\n")
	parseBuilder.WriteString("```javascript\n")
	parseBuilder.WriteString(fmt.Sprintf("function App%d(){\n", parseIndex+1))
	parseBuilder.WriteString("  const total = [1,2,3,4,5].reduce((sum, value) => sum + value, 0);\n")
	parseBuilder.WriteString("  return `<section>${total}</section>`;\n")
	parseBuilder.WriteString("}\n")
	parseBuilder.WriteString("const mount = () => document.body.innerHTML = App1();\n")
	parseBuilder.WriteString("```\n\n")
	parseBuilder.WriteString("```html\n")
	parseBuilder.WriteString("<!doctype html>\n<html><body>\n")
	parseBuilder.WriteString(fmt.Sprintf("<div id=\"card-%d\" class=\"card\">", parseIndex+1))
	parseBuilder.WriteString("Benchmark sample paragraph with enough text to stress splitting and parsing.")
	parseBuilder.WriteString("</div>\n")
	parseBuilder.WriteString("</body></html>\n")
	parseBuilder.WriteString("```\n")
	return parseBuilder.String()
}

// parseBuildRenderWorkerExample100AssistantThought builds one structured thought transcript with multiple sections for section parsing work.
func parseBuildRenderWorkerExample100AssistantThought(parseIndex int) string {
	return strings.Join([]string{
		"Thinking",
		"",
		"**Plan**",
		fmt.Sprintf("Evaluate alternatives for prompt %d and pick the most robust path.", parseIndex+1),
		"",
		"**Checks**",
		"- Validate token accounting and price coverage.",
		"- Validate markdown and code-fence extraction outcomes.",
		"- Validate final output formatting.",
	}, "\n")
}

// parseClearRenderWorkerExample100Caches clears shared app-level derivation caches between benchmark iterations for stable apples-to-apples runs.
func parseClearRenderWorkerExample100Caches() {
	renderedMarkdownCache = map[string]string{}
	thoughtSectionsCache = map[string][]thoughtSection{}
	threadCostSummaryCache = map[string]threadCostSummary{}
}

type renderWorkerExample100BenchmarkRequester struct{}

// Request emulates one worker request-response cycle for benchmark dispatch paths.
func (parseRequester renderWorkerExample100BenchmarkRequester) Request(parseCtx context.Context, parseName string, parsePayload any, parseOnProgress func(interop.WorkerMessage, error)) (interop.WorkerMessage, error) {
	_ = parseRequester
	_ = parseCtx
	_ = parseOnProgress
	switch parseName {
	case backgroundWorkerRequestRenderMessageMetadataBatch:
		parseRequest, parseErr := parseBuildRenderWorkerExample100MetadataBatchRequest(parsePayload)
		if parseErr != nil {
			return interop.WorkerMessage{}, parseErr
		}
		parseResult := parseBuildRenderWorkerExample100MetadataBatchResult(parseRequest)
		return interop.WorkerMessage{Phase: "result", Name: parseName, Payload: parseResult}, nil
	case backgroundWorkerRequestRenderThreadCostSummary:
		parseRequest, parseErr := parseBuildRenderWorkerExample100ThreadCostSummaryRequest(parsePayload)
		if parseErr != nil {
			return interop.WorkerMessage{}, parseErr
		}
		parseResult := parseBuildRenderWorkerExample100ThreadCostSummaryResult(parseRequest)
		return interop.WorkerMessage{Phase: "result", Name: parseName, Payload: parseResult}, nil
	default:
		return interop.WorkerMessage{}, fmt.Errorf("unknown request name %q", parseName)
	}
}

// parseBuildRenderWorkerExample100MetadataBatchRequest decodes one generic payload into the typed metadata batch request shape.
func parseBuildRenderWorkerExample100MetadataBatchRequest(parsePayload any) (renderWorkerMessageMetadataBatchRequest, error) {
	if parseRequest, hasParseRequest := parsePayload.(renderWorkerMessageMetadataBatchRequest); hasParseRequest {
		return parseRequest, nil
	}
	var parseRequest renderWorkerMessageMetadataBatchRequest
	if parseErr := interop.Decode(parsePayload, &parseRequest); parseErr != nil {
		return renderWorkerMessageMetadataBatchRequest{}, parseErr
	}
	return parseRequest, nil
}

// parseBuildRenderWorkerExample100ThreadCostSummaryRequest decodes one generic payload into the typed thread cost request shape.
func parseBuildRenderWorkerExample100ThreadCostSummaryRequest(parsePayload any) (renderWorkerThreadCostSummaryRequest, error) {
	if parseRequest, hasParseRequest := parsePayload.(renderWorkerThreadCostSummaryRequest); hasParseRequest {
		return parseRequest, nil
	}
	var parseRequest renderWorkerThreadCostSummaryRequest
	if parseErr := interop.Decode(parsePayload, &parseRequest); parseErr != nil {
		return renderWorkerThreadCostSummaryRequest{}, parseErr
	}
	return parseRequest, nil
}

// parseBuildRenderWorkerExample100MetadataBatchResult derives one metadata batch result from one metadata batch request.
func parseBuildRenderWorkerExample100MetadataBatchResult(parseRequest renderWorkerMessageMetadataBatchRequest) renderWorkerMessageMetadataBatchResult {
	parseChunkResults := make([]renderWorkerMessageMetadataChunkResult, 0, len(parseRequest.GetChunkRequest))
	for _, parseChunkRequest := range parseRequest.GetChunkRequest {
		parseMessageResults := make([]renderWorkerMessageMetadataMessageResult, 0, len(parseChunkRequest.GetMessageItems))
		for _, parseMessageItem := range parseChunkRequest.GetMessageItems {
			parseMessageIndex := parseMessageItem.GetMessageIndex
			parseContentText := string(parseMessageItem.GetContentBytes)
			parseThoughtText := string(parseMessageItem.GetThoughtBytes)
			parseSections := parseBuildRenderWorkerExample100ThoughtSectionResults(parseThoughtText)
			parseArtifacts := canvasArtifactsFromMarkdown(parseMessageIndex, parseContentText)
			parseMessageResults = append(parseMessageResults, renderWorkerMessageMetadataMessageResult{
				GetMessageIndex:   parseMessageIndex,
				GetContentBytes:   append([]byte(nil), parseMessageItem.GetContentBytes...),
				GetThoughtBytes:   append([]byte(nil), parseMessageItem.GetThoughtBytes...),
				GetThoughtSection: parseBuildRenderWorkerExample100ThoughtSectionBytes(parseSections),
				GetCanvasArtifact: parseBuildRenderWorkerExample100CanvasArtifactBytes(parseArtifacts),
			})
		}
		parseChunkResults = append(parseChunkResults, renderWorkerMessageMetadataChunkResult{
			GetGeneration: parseRequest.GetGeneration,
			GetChunkIndex: parseChunkRequest.GetChunkIndex,
			GetMessage:    parseMessageResults,
		})
	}
	return renderWorkerMessageMetadataBatchResult{
		GetGeneration: parseRequest.GetGeneration,
		GetChunk:      parseChunkResults,
	}
}

// parseBuildRenderWorkerExample100ThoughtSectionBytes converts thought sections into worker thought-section byte payload records.
func parseBuildRenderWorkerExample100ThoughtSectionBytes(parseSections []thoughtSection) []renderWorkerThoughtSectionResult {
	parseResults := make([]renderWorkerThoughtSectionResult, 0, len(parseSections))
	for _, parseSection := range parseSections {
		parseResults = append(parseResults, renderWorkerThoughtSectionResult{
			GetHeadingBytes: []byte(parseSection.Heading),
			GetBodyBytes:    []byte(parseSection.Body),
		})
	}
	return parseResults
}

// parseBuildRenderWorkerExample100CanvasArtifactBytes converts canvas artifacts into worker canvas byte payload records.
func parseBuildRenderWorkerExample100CanvasArtifactBytes(parseArtifacts []canvasArtifact) []renderWorkerCanvasArtifactResult {
	parseResults := make([]renderWorkerCanvasArtifactResult, 0, len(parseArtifacts))
	for _, parseArtifact := range parseArtifacts {
		parseResults = append(parseResults, renderWorkerCanvasArtifactResult{
			GetIDBytes:    []byte(parseArtifact.ID),
			GetLabelBytes: []byte(parseArtifact.Label),
		})
	}
	return parseResults
}

// parseBuildRenderWorkerExample100ThoughtSectionResults derives one thought-section list without shared-cache writes for safe concurrent benchmark worker emulation.
func parseBuildRenderWorkerExample100ThoughtSectionResults(parseThoughtText string) []thoughtSection {
	parseNormalizedText := strings.TrimSpace(strings.ReplaceAll(parseThoughtText, "\r\n", "\n"))
	if parseNormalizedText == "" {
		return nil
	}
	parseLines := strings.Split(parseNormalizedText, "\n")
	parseSections := make([]thoughtSection, 0, 4)
	parseCurrentHeading := ""
	parseCurrentBodyLines := make([]string, 0, len(parseLines))
	parseFlushCurrent := func() {
		if parseCurrentHeading == "" && len(parseCurrentBodyLines) == 0 {
			return
		}
		parseHeading := strings.TrimSpace(parseCurrentHeading)
		parseBody := strings.TrimSpace(strings.Join(parseCurrentBodyLines, "\n"))
		if parseHeading == "" {
			parseHeading = "Thinking"
		}
		parseSections = append(parseSections, thoughtSection{
			Heading: parseHeading,
			Body:    parseBody,
		})
		parseCurrentHeading = ""
		parseCurrentBodyLines = parseCurrentBodyLines[:0]
	}
	for _, parseLine := range parseLines {
		parseTrimmedLine := strings.TrimSpace(parseLine)
		if len(parseSections) == 0 && parseCurrentHeading == "" && len(parseCurrentBodyLines) == 0 && strings.EqualFold(parseTrimmedLine, "thinking") {
			continue
		}
		if parseHeading, hasParseHeading := parseBuildRenderWorkerExample100ThoughtHeading(parseTrimmedLine); hasParseHeading {
			parseFlushCurrent()
			parseCurrentHeading = parseHeading
			continue
		}
		parseCurrentBodyLines = append(parseCurrentBodyLines, parseLine)
	}
	parseFlushCurrent()
	if len(parseSections) == 0 {
		return []thoughtSection{{
			Heading: "Thinking",
			Body:    parseNormalizedText,
		}}
	}
	return parseSections
}

// parseBuildRenderWorkerExample100ThoughtHeading extracts one markdown bold heading line.
func parseBuildRenderWorkerExample100ThoughtHeading(parseLine string) (string, bool) {
	parseTrimmedLine := strings.TrimSpace(parseLine)
	if !strings.HasPrefix(parseTrimmedLine, "**") || !strings.HasSuffix(parseTrimmedLine, "**") || len(parseTrimmedLine) <= 4 {
		return "", false
	}
	parseHeading := strings.TrimSpace(parseTrimmedLine[2 : len(parseTrimmedLine)-2])
	return parseHeading, parseHeading != ""
}

// parseBuildRenderWorkerExample100ThreadCostSummaryResult derives one worker-style thread cost summary payload from request inputs.
func parseBuildRenderWorkerExample100ThreadCostSummaryResult(parseRequest renderWorkerThreadCostSummaryRequest) renderWorkerThreadCostSummaryResult {
	parseMessages := parseBuildRenderWorkerExample100MessagesFromWorkerRequest(parseRequest.GetMessage)
	parseModels := parseBuildRenderWorkerExample100ModelsFromWorkerRequest(parseRequest.GetModel)
	parseSummary := parseDeriveThreadCostSummary(parseMessages, parseModels)
	parseMessageIndexes := make([]int, 0, len(parseSummary.AssistantMessageCosts))
	for parseMessageIndex := range parseSummary.AssistantMessageCosts {
		parseMessageIndexes = append(parseMessageIndexes, parseMessageIndex)
	}
	sort.Ints(parseMessageIndexes)
	parseAssistantMessageCosts := make([]renderWorkerAssistantMessageCostResult, 0, len(parseMessageIndexes))
	for _, parseMessageIndex := range parseMessageIndexes {
		parseMessageCost := parseSummary.AssistantMessageCosts[parseMessageIndex]
		parseAssistantMessageCosts = append(parseAssistantMessageCosts, renderWorkerAssistantMessageCostResult{
			GetMessageIndex:     parseMessageIndex,
			GetModelIDBytes:     []byte(parseMessageCost.ModelID),
			GetPromptTokens:     parseMessageCost.PromptTokens,
			GetCompletionTokens: parseMessageCost.CompletionTokens,
			GetCost:             parseMessageCost.Cost,
			GetHasExactCost:     true,
		})
	}
	return renderWorkerThreadCostSummaryResult{
		GetGeneration:             parseRequest.GetGeneration,
		GetTotalCost:              parseSummary.TotalCost,
		GetAssistantMessageCost:   parseAssistantMessageCosts,
		GetHasAnyExactCosts:       parseSummary.HasAnyExactCosts,
		GetAllAssistantCostsExact: parseSummary.AllAssistantCostsExact,
	}
}

// parseBuildRenderWorkerExample100MessagesFromWorkerRequest converts worker message request rows into app message rows.
func parseBuildRenderWorkerExample100MessagesFromWorkerRequest(parseMessageRows []renderWorkerCostMessageRequest) []message {
	parseMessages := make([]message, 0, len(parseMessageRows))
	for _, parseMessageRow := range parseMessageRows {
		parseContent := ""
		if parseMessageRow.GetHasContent {
			parseContent = "content"
		}
		parseMessages = append(parseMessages, message{
			Role:             string(parseMessageRow.GetRoleBytes),
			Content:          parseContent,
			Pending:          parseMessageRow.GetPending,
			ModelID:          string(parseMessageRow.GetModelIDBytes),
			PromptTokens:     parseMessageRow.GetPromptTokens,
			CompletionTokens: parseMessageRow.GetCompletionTokens,
		})
	}
	return parseMessages
}

// parseBuildRenderWorkerExample100ModelsFromWorkerRequest converts worker model request rows into app model rows.
func parseBuildRenderWorkerExample100ModelsFromWorkerRequest(parseModelRows []renderWorkerCostModelRequest) []modelOption {
	parseModels := make([]modelOption, 0, len(parseModelRows))
	for _, parseModelRow := range parseModelRows {
		parseModels = append(parseModels, modelOption{
			ID: string(parseModelRow.GetModelIDBytes),
			Pricing: modelPricing{
				InputDollarsPerMillion:  parseModelRow.GetInputDollarsPerMillion,
				OutputDollarsPerMillion: parseModelRow.GetOutputDollarsPerMillion,
				Currency:                string(parseModelRow.GetCurrencyBytes),
			},
		})
	}
	return parseModels
}
