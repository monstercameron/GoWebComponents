//go:build js && wasm

package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

type renderWorkerMessageMetadataMessageRequest struct {
	GetMessageIndex int    `json:"messageIndex"`
	GetContentBytes []byte `json:"contentBytes"`
	GetThoughtBytes []byte `json:"thoughtBytes"`
}

type renderWorkerMessageMetadataChunkRequest struct {
	GetChunkIndex   int                                         `json:"chunkIndex"`
	GetMessageItems []renderWorkerMessageMetadataMessageRequest `json:"messageItems"`
}

type renderWorkerMessageMetadataBatchRequest struct {
	GetGeneration   uint64                                    `json:"generation"`
	GetChunkRequest []renderWorkerMessageMetadataChunkRequest `json:"chunkRequest"`
}

type renderWorkerThoughtSectionResult struct {
	GetHeadingBytes []byte `json:"headingBytes"`
	GetBodyBytes    []byte `json:"bodyBytes"`
}

type renderWorkerCanvasArtifactResult struct {
	GetIDBytes    []byte `json:"idBytes"`
	GetLabelBytes []byte `json:"labelBytes"`
}

type renderWorkerMessageMetadataMessageResult struct {
	GetMessageIndex   int                                `json:"messageIndex"`
	GetContentBytes   []byte                             `json:"contentBytes"`
	GetThoughtBytes   []byte                             `json:"thoughtBytes"`
	GetThoughtSection []renderWorkerThoughtSectionResult `json:"thoughtSection"`
	GetCanvasArtifact []renderWorkerCanvasArtifactResult `json:"canvasArtifact"`
}

type renderWorkerMessageMetadataChunkResult struct {
	GetGeneration uint64                                     `json:"generation"`
	GetChunkIndex int                                        `json:"chunkIndex"`
	GetMessage    []renderWorkerMessageMetadataMessageResult `json:"message"`
}

type renderWorkerMessageMetadataBatchResult struct {
	GetGeneration uint64                                   `json:"generation"`
	GetChunk      []renderWorkerMessageMetadataChunkResult `json:"chunk"`
}

type renderWorkerCostMessageRequest struct {
	GetMessageIndex     int    `json:"messageIndex"`
	GetRoleBytes        []byte `json:"roleBytes"`
	GetHasContent       bool   `json:"hasContent"`
	GetPending          bool   `json:"pending"`
	GetModelIDBytes     []byte `json:"modelIDBytes"`
	GetPromptTokens     int    `json:"promptTokens"`
	GetCompletionTokens int    `json:"completionTokens"`
}

type renderWorkerCostModelRequest struct {
	GetModelIDBytes            []byte  `json:"modelIDBytes"`
	GetInputDollarsPerMillion  float64 `json:"inputDollarsPerMillion"`
	GetOutputDollarsPerMillion float64 `json:"outputDollarsPerMillion"`
	GetCurrencyBytes           []byte  `json:"currencyBytes"`
}

type renderWorkerThreadCostSummaryRequest struct {
	GetGeneration uint64                           `json:"generation"`
	GetMessage    []renderWorkerCostMessageRequest `json:"message"`
	GetModel      []renderWorkerCostModelRequest   `json:"model"`
}

type renderWorkerAssistantMessageCostResult struct {
	GetMessageIndex     int     `json:"messageIndex"`
	GetModelIDBytes     []byte  `json:"modelIDBytes"`
	GetPromptTokens     int     `json:"promptTokens"`
	GetCompletionTokens int     `json:"completionTokens"`
	GetCost             float64 `json:"cost"`
	GetHasExactCost     bool    `json:"hasExactCost"`
}

type renderWorkerThreadCostSummaryResult struct {
	GetGeneration             uint64                                   `json:"generation"`
	GetTotalCost              float64                                  `json:"totalCost"`
	GetAssistantMessageCost   []renderWorkerAssistantMessageCostResult `json:"assistantMessageCost"`
	GetHasAnyExactCosts       bool                                     `json:"hasAnyExactCosts"`
	GetAllAssistantCostsExact bool                                     `json:"allAssistantCostsExact"`
}

type renderWorkerSignatureMessageRequest struct {
	GetMessageIndex     int    `json:"messageIndex"`
	GetRoleBytes        []byte `json:"roleBytes"`
	GetContentBytes     []byte `json:"contentBytes"`
	GetThoughtBytes     []byte `json:"thoughtBytes"`
	GetPending          bool   `json:"pending"`
	GetModelIDBytes     []byte `json:"modelIDBytes"`
	GetPromptTokens     int    `json:"promptTokens"`
	GetCompletionTokens int    `json:"completionTokens"`
}

type renderWorkerSignatureRequest struct {
	GetGeneration uint64                                `json:"generation"`
	GetMessage    []renderWorkerSignatureMessageRequest `json:"message"`
	GetModel      []renderWorkerCostModelRequest        `json:"model"`
}

type renderWorkerSignatureResult struct {
	GetGeneration                      uint64 `json:"generation"`
	GetCompletedMarkdownSignatureBytes []byte `json:"completedMarkdownSignatureBytes"`
	GetAssistantMetadataSignatureBytes []byte `json:"assistantMetadataSignatureBytes"`
	GetThreadCostSignatureBytes        []byte `json:"threadCostSignatureBytes"`
}

var renderWorkerFencedBlockPattern = regexp.MustCompile("(?s)```([^\\n`]*)\\n(.*?)\\n```")
var renderWorkerFunctionPattern = regexp.MustCompile(`^\s*(?:export\s+default\s+|export\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)`)
var renderWorkerClassPattern = regexp.MustCompile(`^\s*(?:export\s+default\s+|export\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)`)
var renderWorkerConstPattern = regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=`)

// handleRenderMessageMetadataBatchRequest derives thought-section and canvas-label metadata for one or more message chunks.
func handleRenderMessageMetadataBatchRequest(parseCtx context.Context, parseRequest renderWorkerMessageMetadataBatchRequest) (renderWorkerMessageMetadataBatchResult, error) {
	_ = parseCtx
	parseChunkResults := make([]renderWorkerMessageMetadataChunkResult, 0, len(parseRequest.GetChunkRequest))
	for _, parseChunkRequest := range parseRequest.GetChunkRequest {
		parseMessageResults := make([]renderWorkerMessageMetadataMessageResult, 0, len(parseChunkRequest.GetMessageItems))
		for _, parseMessageItem := range parseChunkRequest.GetMessageItems {
			parseContentBytes := parseMessageItem.GetContentBytes
			parseThoughtBytes := parseMessageItem.GetThoughtBytes
			parseMessageResults = append(parseMessageResults, renderWorkerMessageMetadataMessageResult{
				GetMessageIndex:   parseMessageItem.GetMessageIndex,
				GetContentBytes:   parseContentBytes,
				GetThoughtBytes:   parseThoughtBytes,
				GetThoughtSection: parseBuildThoughtSectionResults(string(parseThoughtBytes)),
				GetCanvasArtifact: parseBuildCanvasArtifactResults(parseMessageItem.GetMessageIndex, string(parseContentBytes)),
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
	}, nil
}

// handleRenderThreadCostSummaryRequest derives one exact-cost summary for assistant messages in the request payload.
func handleRenderThreadCostSummaryRequest(parseCtx context.Context, parseRequest renderWorkerThreadCostSummaryRequest) (renderWorkerThreadCostSummaryResult, error) {
	_ = parseCtx
	parseSummary := renderWorkerThreadCostSummaryResult{
		GetGeneration:             parseRequest.GetGeneration,
		GetAssistantMessageCost:   make([]renderWorkerAssistantMessageCostResult, 0, len(parseRequest.GetMessage)),
		GetAllAssistantCostsExact: true,
	}
	parseModelByID := parseBuildModelByIDMap(parseRequest.GetModel)
	parseAssistantMessageCount := 0
	for _, parseMessageItem := range parseRequest.GetMessage {
		if strings.TrimSpace(string(parseMessageItem.GetRoleBytes)) != "assistant" || parseMessageItem.GetPending {
			continue
		}
		if !parseMessageItem.GetHasContent {
			continue
		}
		parseAssistantMessageCount++
		parseMessageCost, hasParseExactCost := parseBuildAssistantMessageCost(parseMessageItem, parseModelByID)
		if !hasParseExactCost {
			parseSummary.GetAllAssistantCostsExact = false
			continue
		}
		parseSummary.GetAssistantMessageCost = append(parseSummary.GetAssistantMessageCost, parseMessageCost)
		parseSummary.GetTotalCost += parseMessageCost.GetCost
		parseSummary.GetHasAnyExactCosts = true
	}
	if parseAssistantMessageCount == 0 {
		parseSummary.GetAllAssistantCostsExact = false
	}
	return parseSummary, nil
}

// handleRenderSignaturesRequest derives one signature bundle for markdown, metadata, and thread-cost effects.
func handleRenderSignaturesRequest(parseCtx context.Context, parseRequest renderWorkerSignatureRequest) (renderWorkerSignatureResult, error) {
	_ = parseCtx
	return renderWorkerSignatureResult{
		GetGeneration:                      parseRequest.GetGeneration,
		GetCompletedMarkdownSignatureBytes: []byte(parseBuildRenderCompletedMarkdownSignature(parseRequest.GetMessage)),
		GetAssistantMetadataSignatureBytes: []byte(parseBuildRenderAssistantMetadataSignature(parseRequest.GetMessage)),
		GetThreadCostSignatureBytes:        []byte(parseBuildRenderThreadCostSignature(parseRequest.GetMessage, parseRequest.GetModel)),
	}, nil
}

// parseBuildModelByIDMap builds one lookup table from model ID to pricing metadata.
func parseBuildModelByIDMap(parseModels []renderWorkerCostModelRequest) map[string]renderWorkerCostModelRequest {
	parseModelByID := make(map[string]renderWorkerCostModelRequest, len(parseModels))
	for _, parseModel := range parseModels {
		parseModelID := strings.TrimSpace(string(parseModel.GetModelIDBytes))
		if parseModelID == "" {
			continue
		}
		parseModelByID[parseModelID] = parseModel
	}
	return parseModelByID
}

// parseBuildRenderCompletedMarkdownSignature builds one signature over completed assistant markdown content.
func parseBuildRenderCompletedMarkdownSignature(parseMessages []renderWorkerSignatureMessageRequest) string {
	var parseBuilder strings.Builder
	for _, parseMessageItem := range parseMessages {
		if strings.TrimSpace(string(parseMessageItem.GetRoleBytes)) != "assistant" || parseMessageItem.GetPending {
			continue
		}
		parseContentText := strings.TrimSpace(string(parseMessageItem.GetContentBytes))
		if parseContentText == "" {
			continue
		}
		parseBuilder.WriteString(string(parseMessageItem.GetContentBytes))
		parseBuilder.WriteString("\n\x1f\n")
	}
	return parseBuilder.String()
}

// parseBuildRenderAssistantMetadataSignature builds one signature over assistant metadata inputs.
func parseBuildRenderAssistantMetadataSignature(parseMessages []renderWorkerSignatureMessageRequest) string {
	var parseBuilder strings.Builder
	for _, parseMessageItem := range parseMessages {
		if strings.TrimSpace(string(parseMessageItem.GetRoleBytes)) != "assistant" || parseMessageItem.GetPending {
			continue
		}
		parseBuilder.WriteString(parseBuildRenderWorkerMessageIndexString(parseMessageItem.GetMessageIndex))
		parseBuilder.WriteString("|")
		parseBuilder.WriteString(string(parseMessageItem.GetContentBytes))
		parseBuilder.WriteString("|")
		parseBuilder.WriteString(string(parseMessageItem.GetThoughtBytes))
		parseBuilder.WriteString("\n\x1e\n")
	}
	return parseBuilder.String()
}

// parseBuildRenderThreadCostSignature builds one signature over model pricing and assistant usage rows.
func parseBuildRenderThreadCostSignature(parseMessages []renderWorkerSignatureMessageRequest, parseModels []renderWorkerCostModelRequest) string {
	var parseBuilder strings.Builder
	for _, parseModel := range parseModels {
		parseBuilder.WriteString(fmt.Sprintf("model|%s|%.6f|%.6f|%s\n",
			string(parseModel.GetModelIDBytes),
			parseModel.GetInputDollarsPerMillion,
			parseModel.GetOutputDollarsPerMillion,
			string(parseModel.GetCurrencyBytes),
		))
	}
	parseBuilder.WriteString("--\n")
	for _, parseMessageItem := range parseMessages {
		if strings.TrimSpace(string(parseMessageItem.GetRoleBytes)) != "assistant" || parseMessageItem.GetPending {
			continue
		}
		if strings.TrimSpace(string(parseMessageItem.GetContentBytes)) == "" {
			continue
		}
		parseBuilder.WriteString(fmt.Sprintf("%d|%s|%d|%d\n",
			parseMessageItem.GetMessageIndex,
			string(parseMessageItem.GetModelIDBytes),
			parseMessageItem.GetPromptTokens,
			parseMessageItem.GetCompletionTokens,
		))
	}
	return parseBuilder.String()
}

// parseBuildRenderWorkerMessageIndexString converts one integer index into a stable base-10 string.
func parseBuildRenderWorkerMessageIndexString(parseIndex int) string {
	if parseIndex == 0 {
		return "0"
	}
	isParseNegative := parseIndex < 0
	if isParseNegative {
		parseIndex = -parseIndex
	}
	parseDigits := [20]byte{}
	parseWrite := len(parseDigits)
	for parseIndex > 0 {
		parseWrite--
		parseDigits[parseWrite] = byte('0' + (parseIndex % 10))
		parseIndex /= 10
	}
	if isParseNegative {
		parseWrite--
		parseDigits[parseWrite] = '-'
	}
	return string(parseDigits[parseWrite:])
}

// parseBuildAssistantMessageCost computes one assistant message cost and reports exactness based on available model pricing and token counts.
func parseBuildAssistantMessageCost(parseMessage renderWorkerCostMessageRequest, parseModelByID map[string]renderWorkerCostModelRequest) (renderWorkerAssistantMessageCostResult, bool) {
	parseModelID := strings.TrimSpace(string(parseMessage.GetModelIDBytes))
	parseResult := renderWorkerAssistantMessageCostResult{
		GetMessageIndex:     parseMessage.GetMessageIndex,
		GetModelIDBytes:     []byte(parseModelID),
		GetPromptTokens:     parseMessage.GetPromptTokens,
		GetCompletionTokens: parseMessage.GetCompletionTokens,
	}
	parseModel, hasParseModel := parseModelByID[parseModelID]
	if !hasParseModel || (parseMessage.GetPromptTokens <= 0 && parseMessage.GetCompletionTokens <= 0) {
		return parseResult, false
	}
	parseResult.GetCost = (float64(parseMessage.GetPromptTokens) * parseModel.GetInputDollarsPerMillion / 1_000_000) +
		(float64(parseMessage.GetCompletionTokens) * parseModel.GetOutputDollarsPerMillion / 1_000_000)
	parseResult.GetHasExactCost = true
	return parseResult, true
}

// parseBuildThoughtSectionResults parses one thought transcript into heading/body sections.
func parseBuildThoughtSectionResults(parseThoughtText string) []renderWorkerThoughtSectionResult {
	parseNormalizedText := strings.TrimSpace(strings.ReplaceAll(parseThoughtText, "\r\n", "\n"))
	if parseNormalizedText == "" {
		return nil
	}
	parseLines := strings.Split(parseNormalizedText, "\n")
	parseSectionResults := make([]renderWorkerThoughtSectionResult, 0, 4)
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
		parseSectionResults = append(parseSectionResults, renderWorkerThoughtSectionResult{
			GetHeadingBytes: []byte(parseHeading),
			GetBodyBytes:    []byte(parseBody),
		})
		parseCurrentHeading = ""
		parseCurrentBodyLines = parseCurrentBodyLines[:0]
	}
	for _, parseLine := range parseLines {
		parseTrimmedLine := strings.TrimSpace(parseLine)
		if len(parseSectionResults) == 0 && parseCurrentHeading == "" && len(parseCurrentBodyLines) == 0 && strings.EqualFold(parseTrimmedLine, "thinking") {
			continue
		}
		if parseHeading, hasParseHeading := parseBuildThoughtHeading(parseTrimmedLine); hasParseHeading {
			parseFlushCurrent()
			parseCurrentHeading = parseHeading
			continue
		}
		parseCurrentBodyLines = append(parseCurrentBodyLines, parseLine)
	}
	parseFlushCurrent()
	if len(parseSectionResults) == 0 {
		parseSectionResults = append(parseSectionResults, renderWorkerThoughtSectionResult{
			GetHeadingBytes: []byte("Thinking"),
			GetBodyBytes:    []byte(parseNormalizedText),
		})
	}
	return parseSectionResults
}

// parseBuildThoughtHeading extracts one markdown bold heading line and reports whether extraction succeeded.
func parseBuildThoughtHeading(parseLine string) (string, bool) {
	parseTrimmedLine := strings.TrimSpace(parseLine)
	if !strings.HasPrefix(parseTrimmedLine, "**") || !strings.HasSuffix(parseTrimmedLine, "**") || len(parseTrimmedLine) <= 4 {
		return "", false
	}
	parseHeading := strings.TrimSpace(parseTrimmedLine[2 : len(parseTrimmedLine)-2])
	return parseHeading, parseHeading != ""
}

// parseBuildCanvasArtifactResults derives one compact canvas artifact list with stable IDs and labels for one markdown message payload.
func parseBuildCanvasArtifactResults(parseMessageIndex int, parseMarkdown string) []renderWorkerCanvasArtifactResult {
	parseMatches := renderWorkerFencedBlockPattern.FindAllStringSubmatch(parseMarkdown, -1)
	if len(parseMatches) == 0 {
		return nil
	}
	parseArtifacts := make([]renderWorkerCanvasArtifactResult, 0, len(parseMatches))
	for parseBlockIndex, parseMatch := range parseMatches {
		if len(parseMatch) < 3 {
			continue
		}
		parseInfo := strings.TrimSpace(parseMatch[1])
		parseSource := strings.TrimSpace(parseMatch[2])
		if parseSource == "" {
			continue
		}
		parseLanguage, hasParseLanguage := parseBuildCanvasFenceLanguage(parseInfo)
		if !hasParseLanguage {
			continue
		}
		parseArtifacts = append(parseArtifacts, renderWorkerCanvasArtifactResult{
			GetIDBytes:    []byte(fmt.Sprintf("m%d-b%d", parseMessageIndex, parseBlockIndex)),
			GetLabelBytes: []byte(parseBuildCanvasArtifactLabel(parseLanguage, parseSource, parseBlockIndex)),
		})
	}
	return parseArtifacts
}

// parseBuildCanvasFenceLanguage resolves one fenced code info string into one supported preview language.
func parseBuildCanvasFenceLanguage(parseInfo string) (string, bool) {
	parseFields := strings.Fields(strings.ToLower(strings.TrimSpace(parseInfo)))
	if len(parseFields) == 0 {
		return "", false
	}
	for _, parseField := range parseFields {
		if parseField == "canvas" {
			return "canvas", true
		}
	}
	switch parseFields[0] {
	case "html", "htm":
		return "html", true
	case "javascript", "js":
		return "javascript", true
	default:
		return "", false
	}
}

// parseBuildCanvasArtifactLabel builds one user-facing label for one canvas-preview artifact.
func parseBuildCanvasArtifactLabel(parseLanguage string, parseSource string, parseBlockIndex int) string {
	switch parseLanguage {
	case "html":
		return "HTML demo"
	case "javascript":
		if renderWorkerFunctionPattern.MatchString(parseSource) || strings.Contains(parseSource, "const App") {
			return "App component"
		}
		return "JavaScript demo"
	case "canvas":
		if strings.Contains(strings.ToLower(parseSource), "<html") {
			return "Canvas page"
		}
		if renderWorkerClassPattern.MatchString(parseSource) || renderWorkerConstPattern.MatchString(parseSource) || strings.Contains(parseSource, "function App") || strings.Contains(parseSource, "const App") {
			return "App component"
		}
		return fmt.Sprintf("Canvas block %d", parseBlockIndex+1)
	default:
		return fmt.Sprintf("Canvas block %d", parseBlockIndex+1)
	}
}
