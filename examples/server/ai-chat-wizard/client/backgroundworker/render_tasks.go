//go:build js && wasm

package main

import (
	"bytes"
	"context"
	"math"
	"regexp"
	"strconv"
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
	GetContentBytes     []byte `json:"contentBytes"`
	GetThoughtBytes     []byte `json:"thoughtBytes"`
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

const parseSignatureSeed uint64 = 14695981039346656037
const parseSignaturePrime uint64 = 1099511628211

var renderWorkerClassPattern = regexp.MustCompile(`^\s*(?:export\s+default\s+|export\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)`)
var renderWorkerConstPattern = regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=`)

// handleRenderMessageMetadataBatchRequest derives thought-section and canvas-label metadata for one or more message chunks.
func handleRenderMessageMetadataBatchRequest(parseCtx context.Context, parseRequest renderWorkerMessageMetadataBatchRequest) (renderWorkerMessageMetadataBatchResult, error) {
	_ = parseCtx
	parseChunkResults := make([]renderWorkerMessageMetadataChunkResult, len(parseRequest.GetChunkRequest))
	for parseChunkIndex, parseChunkRequest := range parseRequest.GetChunkRequest {
		parseMessageResults := make([]renderWorkerMessageMetadataMessageResult, len(parseChunkRequest.GetMessageItems))
		for parseMessageIndex, parseMessageItem := range parseChunkRequest.GetMessageItems {
			parseMessageResults[parseMessageIndex] = renderWorkerMessageMetadataMessageResult{
				GetMessageIndex:   parseMessageItem.GetMessageIndex,
				GetThoughtSection: parseBuildThoughtSectionResults(string(parseMessageItem.GetThoughtBytes)),
				GetCanvasArtifact: parseBuildCanvasArtifactResultsFromBytes(parseMessageItem.GetMessageIndex, parseMessageItem.GetContentBytes),
			}
		}
		parseChunkResults[parseChunkIndex] = renderWorkerMessageMetadataChunkResult{
			GetGeneration: parseRequest.GetGeneration,
			GetChunkIndex: parseChunkRequest.GetChunkIndex,
			GetMessage:    parseMessageResults,
		}
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

// parseApplySignatureByte mixes one delimiter or scalar byte into the rolling signature accumulator.
func parseApplySignatureByte(parseHash *uint64, parseByte byte) {
	*parseHash ^= uint64(parseByte)
	*parseHash *= parseSignaturePrime
}

// parseApplySignatureBytes mixes one byte slice into the rolling signature accumulator with a field terminator.
func parseApplySignatureBytes(parseHash *uint64, parseBytes []byte) {
	for _, parseByte := range parseBytes {
		parseApplySignatureByte(parseHash, parseByte)
	}
	parseApplySignatureByte(parseHash, 0)
}

// parseApplySignatureInt mixes one base-10 integer field into the rolling signature accumulator.
func parseApplySignatureInt(parseHash *uint64, parseValue int) {
	var parseScratch [24]byte
	parseEncoded := strconv.AppendInt(parseScratch[:0], int64(parseValue), 10)
	parseApplySignatureBytes(parseHash, parseEncoded)
}

// parseApplySignatureScaledFloat mixes one pricing field rounded to six decimal places into the rolling signature accumulator.
func parseApplySignatureScaledFloat(parseHash *uint64, parseValue float64) {
	parseApplySignatureInt(parseHash, int(math.Round(parseValue*1_000_000)))
}

// parseBuildSignatureString formats one rolling signature accumulator into a compact hexadecimal key.
func parseBuildSignatureString(parseHash uint64) string {
	return strconv.FormatUint(parseHash, 16)
}

// parseBuildRenderCompletedMarkdownSignature builds one signature over completed assistant markdown content.
func parseBuildRenderCompletedMarkdownSignature(parseMessages []renderWorkerSignatureMessageRequest) string {
	parseHash := parseSignatureSeed
	hasParseContent := false
	for _, parseMessageItem := range parseMessages {
		parseContentText := strings.TrimSpace(string(parseMessageItem.GetContentBytes))
		if parseContentText == "" {
			continue
		}
		hasParseContent = true
		parseApplySignatureByte(&parseHash, 'm')
		parseApplySignatureBytes(&parseHash, parseMessageItem.GetContentBytes)
	}
	if !hasParseContent {
		return ""
	}
	return parseBuildSignatureString(parseHash)
}

// parseBuildRenderAssistantMetadataSignature builds one signature over assistant metadata inputs.
func parseBuildRenderAssistantMetadataSignature(parseMessages []renderWorkerSignatureMessageRequest) string {
	parseHash := parseSignatureSeed
	hasParseMessage := false
	for _, parseMessageItem := range parseMessages {
		hasParseMessage = true
		parseApplySignatureByte(&parseHash, 'm')
		parseApplySignatureInt(&parseHash, parseMessageItem.GetMessageIndex)
		parseApplySignatureBytes(&parseHash, parseMessageItem.GetContentBytes)
		parseApplySignatureBytes(&parseHash, parseMessageItem.GetThoughtBytes)
	}
	if !hasParseMessage {
		return ""
	}
	return parseBuildSignatureString(parseHash)
}

// parseBuildRenderThreadCostSignature builds one signature over model pricing and assistant usage rows.
func parseBuildRenderThreadCostSignature(parseMessages []renderWorkerSignatureMessageRequest, parseModels []renderWorkerCostModelRequest) string {
	parseHash := parseSignatureSeed
	for _, parseModel := range parseModels {
		parseApplySignatureByte(&parseHash, 'o')
		parseApplySignatureBytes(&parseHash, parseModel.GetModelIDBytes)
		parseApplySignatureScaledFloat(&parseHash, parseModel.GetInputDollarsPerMillion)
		parseApplySignatureScaledFloat(&parseHash, parseModel.GetOutputDollarsPerMillion)
		parseApplySignatureBytes(&parseHash, parseModel.GetCurrencyBytes)
	}
	parseApplySignatureByte(&parseHash, '-')
	for _, parseMessageItem := range parseMessages {
		if strings.TrimSpace(string(parseMessageItem.GetContentBytes)) == "" {
			continue
		}
		parseApplySignatureByte(&parseHash, 'm')
		parseApplySignatureInt(&parseHash, parseMessageItem.GetMessageIndex)
		parseApplySignatureBytes(&parseHash, parseMessageItem.GetModelIDBytes)
		parseApplySignatureInt(&parseHash, parseMessageItem.GetPromptTokens)
		parseApplySignatureInt(&parseHash, parseMessageItem.GetCompletionTokens)
	}
	return parseBuildSignatureString(parseHash)
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
	parseNormalizedText := parseThoughtText
	if strings.Contains(parseNormalizedText, "\r\n") {
		parseNormalizedText = strings.ReplaceAll(parseNormalizedText, "\r\n", "\n")
	}
	parseNormalizedText = strings.TrimSpace(parseNormalizedText)
	if parseNormalizedText == "" {
		return nil
	}
	parseSectionResults := make([]renderWorkerThoughtSectionResult, 0, 4)
	parseCurrentHeading := ""
	parseCurrentBodyStart := -1
	parseCurrentBodyEnd := -1
	parseFlushCurrent := func() {
		if parseCurrentHeading == "" && parseCurrentBodyStart < 0 {
			return
		}
		parseHeading := strings.TrimSpace(parseCurrentHeading)
		parseBody := ""
		if parseCurrentBodyStart >= 0 {
			parseBody = strings.TrimSpace(parseNormalizedText[parseCurrentBodyStart:parseCurrentBodyEnd])
		}
		if parseHeading == "" {
			parseHeading = "Thinking"
		}
		parseSectionResults = append(parseSectionResults, renderWorkerThoughtSectionResult{
			GetHeadingBytes: []byte(parseHeading),
			GetBodyBytes:    []byte(parseBody),
		})
		parseCurrentHeading = ""
		parseCurrentBodyStart = -1
		parseCurrentBodyEnd = -1
	}

	// Scan the normalized transcript once so section bodies reuse the original backing string instead of split/join staging.
	for parseLineStart := 0; parseLineStart < len(parseNormalizedText); {
		parseLineEnd := parseLineStart
		for parseLineEnd < len(parseNormalizedText) && parseNormalizedText[parseLineEnd] != '\n' {
			parseLineEnd++
		}
		parseNextLineStart := parseLineEnd
		if parseNextLineStart < len(parseNormalizedText) && parseNormalizedText[parseNextLineStart] == '\n' {
			parseNextLineStart++
		}
		parseTrimmedLine := strings.TrimSpace(parseNormalizedText[parseLineStart:parseLineEnd])
		if len(parseSectionResults) == 0 && parseCurrentHeading == "" && parseCurrentBodyStart < 0 && strings.EqualFold(parseTrimmedLine, "thinking") {
			parseLineStart = parseNextLineStart
			continue
		}
		if parseHeading, hasParseHeading := parseBuildThoughtHeading(parseTrimmedLine); hasParseHeading {
			parseFlushCurrent()
			parseCurrentHeading = parseHeading
			parseLineStart = parseNextLineStart
			continue
		}
		if parseCurrentBodyStart < 0 {
			parseCurrentBodyStart = parseLineStart
		}
		parseCurrentBodyEnd = parseNextLineStart
		parseLineStart = parseNextLineStart
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
	return parseBuildCanvasArtifactResultsFromBytes(parseMessageIndex, []byte(parseMarkdown))
}

// parseBuildCanvasArtifactResultsFromBytes derives one compact canvas artifact list from markdown bytes without first copying the entire payload into a string.
func parseBuildCanvasArtifactResultsFromBytes(parseMessageIndex int, parseMarkdown []byte) []renderWorkerCanvasArtifactResult {
	parseFenceMarker := []byte("```")
	parseFenceClosePrefix := []byte("\n```")
	if !bytes.Contains(parseMarkdown, parseFenceMarker) {
		return nil
	}
	parseArtifacts := make([]renderWorkerCanvasArtifactResult, 0, 4)
	parseSearchStart := 0
	parseBlockIndex := 0

	// Scan fenced code blocks directly so metadata extraction avoids regex match-slice staging and full markdown string copies on worker requests.
	for parseSearchStart < len(parseMarkdown) {
		parseOpenRel := bytes.Index(parseMarkdown[parseSearchStart:], parseFenceMarker)
		if parseOpenRel < 0 {
			break
		}
		parseOpenStart := parseSearchStart + parseOpenRel
		parseInfoStart := parseOpenStart + len(parseFenceMarker)
		parseInfoEndRel := bytes.IndexByte(parseMarkdown[parseInfoStart:], '\n')
		if parseInfoEndRel < 0 {
			break
		}
		parseInfoEnd := parseInfoStart + parseInfoEndRel
		parseInfoRaw := parseMarkdown[parseInfoStart:parseInfoEnd]
		if bytes.IndexByte(parseInfoRaw, '`') >= 0 {
			parseSearchStart = parseOpenStart + 1
			continue
		}
		parseContentStart := parseInfoEnd + 1
		parseCloseRel := bytes.Index(parseMarkdown[parseContentStart:], parseFenceClosePrefix)
		if parseCloseRel < 0 {
			break
		}
		parseCloseStart := parseContentStart + parseCloseRel
		parseInfo := bytes.TrimSpace(parseInfoRaw)
		parseSource := bytes.TrimSpace(parseMarkdown[parseContentStart:parseCloseStart])
		if len(parseSource) == 0 {
			parseBlockIndex++
			parseSearchStart = parseCloseStart + len(parseFenceClosePrefix)
			continue
		}
		parseLanguage, hasParseLanguage := parseBuildCanvasFenceLanguageBytes(parseInfo)
		if !hasParseLanguage {
			parseBlockIndex++
			parseSearchStart = parseCloseStart + len(parseFenceClosePrefix)
			continue
		}
		parseArtifacts = append(parseArtifacts, renderWorkerCanvasArtifactResult{
			GetIDBytes:    parseBuildCanvasArtifactID(parseMessageIndex, parseBlockIndex),
			GetLabelBytes: []byte(parseBuildCanvasArtifactLabel(parseLanguage, string(parseSource), parseBlockIndex)),
		})
		parseBlockIndex++
		parseSearchStart = parseCloseStart + len(parseFenceClosePrefix)
	}
	if len(parseArtifacts) == 0 {
		return nil
	}
	return parseArtifacts
}

// parseBuildCanvasFenceLanguageBytes resolves one fenced code info byte slice into one supported preview language.
func parseBuildCanvasFenceLanguageBytes(parseInfo []byte) (string, bool) {
	parseInfo = bytes.TrimSpace(parseInfo)
	if len(parseInfo) == 0 {
		return "", false
	}
	parseFirstToken := []byte(nil)
	parseTokenStart := -1
	for parseIndex := 0; parseIndex <= len(parseInfo); parseIndex++ {
		hasParseTokenByte := parseIndex < len(parseInfo)
		if hasParseTokenByte && !parseIsCanvasFenceWhitespace(parseInfo[parseIndex]) {
			if parseTokenStart < 0 {
				parseTokenStart = parseIndex
			}
			continue
		}
		if parseTokenStart < 0 {
			continue
		}
		parseToken := parseInfo[parseTokenStart:parseIndex]
		if parseMatchCanvasFenceTokenBytes(parseToken, "canvas") {
			return "canvas", true
		}
		if len(parseFirstToken) == 0 {
			parseFirstToken = parseToken
		}
		parseTokenStart = -1
	}
	switch {
	case parseMatchCanvasFenceTokenBytes(parseFirstToken, "html"), parseMatchCanvasFenceTokenBytes(parseFirstToken, "htm"):
		return "html", true
	case parseMatchCanvasFenceTokenBytes(parseFirstToken, "javascript"), parseMatchCanvasFenceTokenBytes(parseFirstToken, "js"):
		return "javascript", true
	default:
		return "", false
	}
}

// parseMatchCanvasFenceTokenBytes reports whether one fence-info token equals the expected keyword ignoring ASCII case.
func parseMatchCanvasFenceTokenBytes(parseToken []byte, parseWant string) bool {
	if len(parseToken) != len(parseWant) {
		return false
	}
	for parseIndex := 0; parseIndex < len(parseWant); parseIndex++ {
		parseByte := parseToken[parseIndex]
		if parseByte >= 'A' && parseByte <= 'Z' {
			parseByte += 'a' - 'A'
		}
		if parseByte != parseWant[parseIndex] {
			return false
		}
	}
	return true
}

// parseBuildCanvasArtifactID builds one stable canvas artifact ID without formatted string staging.
func parseBuildCanvasArtifactID(parseMessageIndex int, parseBlockIndex int) []byte {
	var parseScratch [32]byte
	parseIDBytes := parseScratch[:0]
	parseIDBytes = append(parseIDBytes, 'm')
	parseIDBytes = strconv.AppendInt(parseIDBytes, int64(parseMessageIndex), 10)
	parseIDBytes = append(parseIDBytes, '-', 'b')
	parseIDBytes = strconv.AppendInt(parseIDBytes, int64(parseBlockIndex), 10)
	return append([]byte(nil), parseIDBytes...)
}

// parseBuildCanvasFenceLanguage resolves one fenced code info string into one supported preview language.
func parseBuildCanvasFenceLanguage(parseInfo string) (string, bool) {
	parseInfo = strings.TrimSpace(parseInfo)
	if parseInfo == "" {
		return "", false
	}
	parseFirstToken := ""
	parseTokenStart := -1
	for parseIndex := 0; parseIndex <= len(parseInfo); parseIndex++ {
		hasParseTokenByte := parseIndex < len(parseInfo)
		if hasParseTokenByte && !parseIsCanvasFenceWhitespace(parseInfo[parseIndex]) {
			if parseTokenStart < 0 {
				parseTokenStart = parseIndex
			}
			continue
		}
		if parseTokenStart < 0 {
			continue
		}
		parseToken := parseInfo[parseTokenStart:parseIndex]
		if parseMatchCanvasFenceToken(parseToken, "canvas") {
			return "canvas", true
		}
		if parseFirstToken == "" {
			parseFirstToken = parseToken
		}
		parseTokenStart = -1
	}
	switch {
	case parseMatchCanvasFenceToken(parseFirstToken, "html"), parseMatchCanvasFenceToken(parseFirstToken, "htm"):
		return "html", true
	case parseMatchCanvasFenceToken(parseFirstToken, "javascript"), parseMatchCanvasFenceToken(parseFirstToken, "js"):
		return "javascript", true
	default:
		return "", false
	}
}

// parseIsCanvasFenceWhitespace reports whether one byte is treated as fence-info whitespace.
func parseIsCanvasFenceWhitespace(parseByte byte) bool {
	switch parseByte {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}

// parseMatchCanvasFenceToken reports whether one fence-info token equals the expected keyword ignoring ASCII case.
func parseMatchCanvasFenceToken(parseToken string, parseWant string) bool {
	if len(parseToken) != len(parseWant) {
		return false
	}
	for parseIndex := 0; parseIndex < len(parseWant); parseIndex++ {
		parseByte := parseToken[parseIndex]
		if parseByte >= 'A' && parseByte <= 'Z' {
			parseByte += 'a' - 'A'
		}
		if parseByte != parseWant[parseIndex] {
			return false
		}
	}
	return true
}

// parseBuildCanvasArtifactLabel builds one user-facing label for one canvas-preview artifact.
func parseBuildCanvasArtifactLabel(parseLanguage string, parseSource string, parseBlockIndex int) string {
	switch parseLanguage {
	case "html":
		return "HTML demo"
	case "javascript":
		if parseHasCanvasSourcePrefix(parseSource, "function ") ||
			parseHasCanvasSourcePrefix(parseSource, "export function ") ||
			parseHasCanvasSourcePrefix(parseSource, "export default function ") ||
			strings.Contains(parseSource, "const App") {
			return "App component"
		}
		return "JavaScript demo"
	case "canvas":
		if parseContainsCanvasSourceFold(parseSource, "<html") {
			return "Canvas page"
		}
		if parseHasCanvasSourcePrefix(parseSource, "class ") ||
			parseHasCanvasSourcePrefix(parseSource, "export class ") ||
			parseHasCanvasSourcePrefix(parseSource, "export default class ") ||
			parseHasCanvasSourcePrefix(parseSource, "const ") ||
			parseHasCanvasSourcePrefix(parseSource, "let ") ||
			parseHasCanvasSourcePrefix(parseSource, "var ") ||
			parseHasCanvasSourcePrefix(parseSource, "export const ") ||
			parseHasCanvasSourcePrefix(parseSource, "export let ") ||
			parseHasCanvasSourcePrefix(parseSource, "export var ") ||
			strings.Contains(parseSource, "function App") ||
			strings.Contains(parseSource, "const App") {
			return "App component"
		}
		return parseBuildCanvasBlockLabel(parseBlockIndex)
	default:
		return parseBuildCanvasBlockLabel(parseBlockIndex)
	}
}

// parseHasCanvasSourcePrefix reports whether one source starts with the expected prefix after leading whitespace.
func parseHasCanvasSourcePrefix(parseSource string, parsePrefix string) bool {
	parseIndex := 0
	for parseIndex < len(parseSource) && parseIsCanvasFenceWhitespace(parseSource[parseIndex]) {
		parseIndex++
	}
	return strings.HasPrefix(parseSource[parseIndex:], parsePrefix)
}

// parseContainsCanvasSourceFold reports whether one source contains the expected ASCII token ignoring case.
func parseContainsCanvasSourceFold(parseSource string, parseNeedle string) bool {
	if parseNeedle == "" {
		return true
	}
	if len(parseNeedle) > len(parseSource) {
		return false
	}
	parseLimit := len(parseSource) - len(parseNeedle)
	for parseIndex := 0; parseIndex <= parseLimit; parseIndex++ {
		if parseMatchCanvasSourceFold(parseSource[parseIndex:parseIndex+len(parseNeedle)], parseNeedle) {
			return true
		}
	}
	return false
}

// parseMatchCanvasSourceFold reports whether two ASCII strings match ignoring case.
func parseMatchCanvasSourceFold(parseSource string, parseNeedle string) bool {
	if len(parseSource) != len(parseNeedle) {
		return false
	}
	for parseIndex := 0; parseIndex < len(parseNeedle); parseIndex++ {
		parseByte := parseSource[parseIndex]
		if parseByte >= 'A' && parseByte <= 'Z' {
			parseByte += 'a' - 'A'
		}
		if parseByte != parseNeedle[parseIndex] {
			return false
		}
	}
	return true
}

// parseBuildCanvasBlockLabel builds one fallback block label without formatted string staging.
func parseBuildCanvasBlockLabel(parseBlockIndex int) string {
	return "Canvas block " + strconv.Itoa(parseBlockIndex+1)
}
