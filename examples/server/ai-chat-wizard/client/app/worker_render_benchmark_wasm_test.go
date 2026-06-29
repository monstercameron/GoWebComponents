//go:build js && wasm

package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/interop"
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

// BenchmarkRenderWorkerExample100WorkerPathMetadataSingleMessageRefresh compares full metadata refresh against a delta refresh when only one assistant message changes.
func BenchmarkRenderWorkerExample100WorkerPathMetadataSingleMessageRefresh(parseB *testing.B) {
	parseBaseMessages, _ := parseBuildRenderWorkerExample100Fixture(180)
	parseRequester := renderWorkerExample100BenchmarkRequester{}
	parseWarmMessageItems := parseBuildAssistantMessageMetadataItems(parseBaseMessages)
	parseWarmThoughtCacheByMessage, parseWarmCanvasCacheByMessage, parseWarmErr := parseRequestAssistantMessageMetadata(
		context.Background(),
		parseRequester,
		1,
		parseWarmMessageItems,
		backgroundWorkerRenderPoolSize,
	)
	if parseWarmErr != nil {
		parseB.Fatalf("parseRequestAssistantMessageMetadata(warm): %v", parseWarmErr)
	}
	parseTargetMessageIndex := 133
	parseBaseContentText := parseBaseMessages[parseTargetMessageIndex].Content
	parseBuildChangedMessages := func(parseIndex int) []message {
		parseMessages := append([]message(nil), parseBaseMessages...)
		parseMessages[parseTargetMessageIndex].Content = parseBaseContentText + fmt.Sprintf("\n<!-- delta-%d -->", parseIndex)
		return parseMessages
	}
	parseB.Run("full_refresh", func(parseFullB *testing.B) {
		parseFullB.ReportAllocs()
		parseFullB.ResetTimer()
		for parseIndex := 0; parseFullB.Loop(); parseIndex++ {
			parseMessages := parseBuildChangedMessages(parseIndex)
			parseMessageItems := parseBuildAssistantMessageMetadataItems(parseMessages)
			parseThoughtCacheByMessage, parseCanvasCacheByMessage, parseErr := parseRequestAssistantMessageMetadata(
				context.Background(),
				parseRequester,
				uint64(parseIndex+2),
				parseMessageItems,
				backgroundWorkerRenderPoolSize,
			)
			if parseErr != nil {
				parseFullB.Fatalf("parseRequestAssistantMessageMetadata(full): %v", parseErr)
			}
			parseRenderWorkerExample100BenchmarkSink = len(parseThoughtCacheByMessage) + len(parseCanvasCacheByMessage)
		}
	})
	parseB.Run("delta_refresh", func(parseDeltaB *testing.B) {
		parseDeltaB.ReportAllocs()
		parseDeltaB.ResetTimer()
		for parseIndex := 0; parseDeltaB.Loop(); parseIndex++ {
			parseMessages := parseBuildChangedMessages(parseIndex)
			parseMessageItems, parseRetainedThoughtCacheByMessage, parseRetainedCanvasCacheByMessage := parseBuildAssistantMessageMetadataDelta(
				parseMessages,
				parseWarmThoughtCacheByMessage,
				parseWarmCanvasCacheByMessage,
			)
			parseThoughtCacheByMessage, parseCanvasCacheByMessage, parseErr := parseRequestAssistantMessageMetadata(
				context.Background(),
				parseRequester,
				uint64(parseIndex+2),
				parseMessageItems,
				backgroundWorkerRenderPoolSize,
			)
			if parseErr != nil {
				parseDeltaB.Fatalf("parseRequestAssistantMessageMetadata(delta): %v", parseErr)
			}
			parseThoughtMergedCacheByMessage, parseCanvasMergedCacheByMessage := parseMergeAssistantMessageMetadataCaches(
				parseRetainedThoughtCacheByMessage,
				parseRetainedCanvasCacheByMessage,
				parseThoughtCacheByMessage,
				parseCanvasCacheByMessage,
			)
			parseRenderWorkerExample100BenchmarkSink = len(parseMessageItems) + len(parseThoughtMergedCacheByMessage) + len(parseCanvasMergedCacheByMessage)
		}
	})
}

// BenchmarkBuildAssistantMessageMetadataLookupCurrentVsLegacy compares the old string-reconstruction map against the current dense index lookup used when mapping worker metadata results back into caches.
func BenchmarkBuildAssistantMessageMetadataLookupCurrentVsLegacy(parseB *testing.B) {
	parseMessages, _ := parseBuildRenderWorkerExample100Fixture(180)
	parseItems := parseBuildAssistantMessageMetadataItems(parseMessages)
	parseRequest := renderWorkerMessageMetadataBatchRequest{
		GetGeneration: 1,
		GetChunkRequest: []renderWorkerMessageMetadataChunkRequest{{
			GetChunkIndex:   0,
			GetMessageItems: parseItems,
		}},
	}
	parseResult := parseBuildRenderWorkerExample100MetadataBatchResult(parseRequest)

	parseB.Run("legacy", func(parseLegacyB *testing.B) {
		parseLegacyB.ReportAllocs()
		parseLegacyB.ResetTimer()
		for parseLegacyB.Loop() {
			parseTextByMessage := parseBuildAssistantMessageMetadataTextByMessageLegacy(parseItems)
			parseThoughtCacheByMessage := make(map[int]renderWorkerThoughtCacheEntry, len(parseItems))
			parseCanvasCacheByMessage := make(map[int]renderWorkerCanvasCacheEntry, len(parseItems))
			for _, parseChunkResult := range parseResult.GetChunk {
				for _, parseMessageResult := range parseChunkResult.GetMessage {
					parseMessageText := parseTextByMessage[parseMessageResult.GetMessageIndex]
					parseThoughtCacheByMessage[parseMessageResult.GetMessageIndex] = renderWorkerThoughtCacheEntry{
						GetThoughtText: parseMessageText.GetThoughtText,
						GetSection:     parseBuildThoughtSectionResultsFromWorker(parseMessageResult.GetMessageIndex, parseMessageResult.GetThoughtSection),
					}
					parseCanvasCacheByMessage[parseMessageResult.GetMessageIndex] = renderWorkerCanvasCacheEntry{
						GetContentText: parseMessageText.GetContentText,
						GetArtifact:    parseBuildCanvasArtifactResultsFromWorker(parseMessageResult.GetMessageIndex, parseMessageResult.GetCanvasArtifact),
					}
				}
			}
			parseRenderWorkerExample100BenchmarkSink = len(parseThoughtCacheByMessage) + len(parseCanvasCacheByMessage)
		}
	})
	parseB.Run("current", func(parseCurrentB *testing.B) {
		parseCurrentB.ReportAllocs()
		parseCurrentB.ResetTimer()
		for parseCurrentB.Loop() {
			parseTextByIndex := parseBuildAssistantMessageMetadataTextByIndex(parseItems)
			parseThoughtCacheByMessage := make(map[int]renderWorkerThoughtCacheEntry, len(parseItems))
			parseCanvasCacheByMessage := make(map[int]renderWorkerCanvasCacheEntry, len(parseItems))
			for _, parseChunkResult := range parseResult.GetChunk {
				for _, parseMessageResult := range parseChunkResult.GetMessage {
					parseMessageText := parseLookupAssistantMessageMetadataText(parseTextByIndex, parseMessageResult.GetMessageIndex)
					parseThoughtCacheByMessage[parseMessageResult.GetMessageIndex] = renderWorkerThoughtCacheEntry{
						GetThoughtText: parseMessageText.GetThoughtText,
						GetSection:     parseBuildThoughtSectionResultsFromWorker(parseMessageResult.GetMessageIndex, parseMessageResult.GetThoughtSection),
					}
					parseCanvasCacheByMessage[parseMessageResult.GetMessageIndex] = renderWorkerCanvasCacheEntry{
						GetContentText: parseMessageText.GetContentText,
						GetArtifact:    parseBuildCanvasArtifactResultsFromWorker(parseMessageResult.GetMessageIndex, parseMessageResult.GetCanvasArtifact),
					}
				}
			}
			parseRenderWorkerExample100BenchmarkSink = len(parseThoughtCacheByMessage) + len(parseCanvasCacheByMessage)
		}
	})
}

// BenchmarkBuildAssistantMessageMetadataConversionCurrentVsLegacy compares the legacy append-heavy metadata result conversion path against the current fixed-size conversion helpers.
func BenchmarkBuildAssistantMessageMetadataConversionCurrentVsLegacy(parseB *testing.B) {
	parseMessages, _ := parseBuildRenderWorkerExample100Fixture(180)
	parseItems := parseBuildAssistantMessageMetadataItems(parseMessages)
	parseRequest := renderWorkerMessageMetadataBatchRequest{
		GetGeneration: 1,
		GetChunkRequest: []renderWorkerMessageMetadataChunkRequest{{
			GetChunkIndex:   0,
			GetMessageItems: parseItems,
		}},
	}
	parseResult := parseBuildRenderWorkerExample100MetadataBatchResult(parseRequest)
	parseTextByIndex := parseBuildAssistantMessageMetadataTextByIndex(parseItems)

	parseB.Run("legacy", func(parseLegacyB *testing.B) {
		parseLegacyB.ReportAllocs()
		parseLegacyB.ResetTimer()
		for parseLegacyB.Loop() {
			parseThoughtCacheByMessage := make(map[int]renderWorkerThoughtCacheEntry, len(parseItems))
			parseCanvasCacheByMessage := make(map[int]renderWorkerCanvasCacheEntry, len(parseItems))
			for _, parseChunkResult := range parseResult.GetChunk {
				for _, parseMessageResult := range parseChunkResult.GetMessage {
					parseMessageText := parseLookupAssistantMessageMetadataText(parseTextByIndex, parseMessageResult.GetMessageIndex)
					parseThoughtCacheByMessage[parseMessageResult.GetMessageIndex] = renderWorkerThoughtCacheEntry{
						GetThoughtText: parseMessageText.GetThoughtText,
						GetSection:     parseBuildThoughtSectionResultsFromWorkerLegacy(parseMessageResult.GetMessageIndex, parseMessageResult.GetThoughtSection),
					}
					parseCanvasCacheByMessage[parseMessageResult.GetMessageIndex] = renderWorkerCanvasCacheEntry{
						GetContentText: parseMessageText.GetContentText,
						GetArtifact:    parseBuildCanvasArtifactResultsFromWorkerLegacy(parseMessageResult.GetMessageIndex, parseMessageResult.GetCanvasArtifact),
					}
				}
			}
			parseRenderWorkerExample100BenchmarkSink = len(parseThoughtCacheByMessage) + len(parseCanvasCacheByMessage)
		}
	})
	parseB.Run("current", func(parseCurrentB *testing.B) {
		parseCurrentB.ReportAllocs()
		parseCurrentB.ResetTimer()
		for parseCurrentB.Loop() {
			parseThoughtCacheByMessage := make(map[int]renderWorkerThoughtCacheEntry, len(parseItems))
			parseCanvasCacheByMessage := make(map[int]renderWorkerCanvasCacheEntry, len(parseItems))
			for _, parseChunkResult := range parseResult.GetChunk {
				for _, parseMessageResult := range parseChunkResult.GetMessage {
					parseMessageText := parseLookupAssistantMessageMetadataText(parseTextByIndex, parseMessageResult.GetMessageIndex)
					parseThoughtCacheByMessage[parseMessageResult.GetMessageIndex] = renderWorkerThoughtCacheEntry{
						GetThoughtText: parseMessageText.GetThoughtText,
						GetSection:     parseBuildThoughtSectionResultsFromWorker(parseMessageResult.GetMessageIndex, parseMessageResult.GetThoughtSection),
					}
					parseCanvasCacheByMessage[parseMessageResult.GetMessageIndex] = renderWorkerCanvasCacheEntry{
						GetContentText: parseMessageText.GetContentText,
						GetArtifact:    parseBuildCanvasArtifactResultsFromWorker(parseMessageResult.GetMessageIndex, parseMessageResult.GetCanvasArtifact),
					}
				}
			}
			parseRenderWorkerExample100BenchmarkSink = len(parseThoughtCacheByMessage) + len(parseCanvasCacheByMessage)
		}
	})
}

// BenchmarkRenderWorkerExample100WorkerPathStableThreadCostRefresh compares a redundant stable-signature thread-cost worker refresh against the signature-gated no-op path.
func BenchmarkRenderWorkerExample100WorkerPathStableThreadCostRefresh(parseB *testing.B) {
	parseMessages, parseModels := parseBuildRenderWorkerExample100Fixture(180)
	parseRequester := renderWorkerExample100BenchmarkRequester{}
	parseSignature := parseThreadCostSummarySignature(parseMessages, parseModels)
	parseB.Run("redundant_request", func(parseRequestB *testing.B) {
		parseRequestB.ReportAllocs()
		parseRequestB.ResetTimer()
		for parseIndex := 0; parseRequestB.Loop(); parseIndex++ {
			parseSummary, parseErr := parseRequestWorkerThreadCostSummary(
				context.Background(),
				parseRequester,
				uint64(parseIndex+1),
				parseMessages,
				parseModels,
			)
			if parseErr != nil {
				parseRequestB.Fatalf("parseRequestWorkerThreadCostSummary(redundant): %v", parseErr)
			}
			parseRenderWorkerExample100BenchmarkSink = len(parseSummary.AssistantMessageCosts)
		}
	})
	parseB.Run("signature_gated_noop", func(parseNoopB *testing.B) {
		parseNoopB.ReportAllocs()
		parseAppliedSignature := parseSignature
		parseNoopB.ResetTimer()
		for parseNoopB.Loop() {
			if !parseShouldRefreshWorkerDerivedSignature(parseSignature, parseAppliedSignature) {
				parseRenderWorkerExample100BenchmarkSink++
				continue
			}
			parseSummary, parseErr := parseRequestWorkerThreadCostSummary(
				context.Background(),
				parseRequester,
				1,
				parseMessages,
				parseModels,
			)
			if parseErr != nil {
				parseNoopB.Fatalf("parseRequestWorkerThreadCostSummary(noop): %v", parseErr)
			}
			parseRenderWorkerExample100BenchmarkSink = len(parseSummary.AssistantMessageCosts)
			parseAppliedSignature = parseSignature
		}
	})
}

// BenchmarkBuildRenderWorkerThreadCostSummaryRequestCurrentVsLegacy compares the filtered worker request builder against the previous full-payload shape under a large mostly-unused model catalog.
func BenchmarkBuildRenderWorkerThreadCostSummaryRequestCurrentVsLegacy(parseB *testing.B) {
	parseMessages, parseModels := parseBuildRenderWorkerExample100Fixture(180)
	parseExpandedModels := parseBuildRenderWorkerExample100ExpandedModels(parseModels, 64)
	parseB.Run("legacy", func(parseLegacyB *testing.B) {
		parseLegacyB.ReportAllocs()
		parseLegacyB.ResetTimer()
		for parseIndex := 0; parseLegacyB.Loop(); parseIndex++ {
			parseRequest := parseBuildRenderWorkerExample100LegacyThreadCostSummaryRequest(uint64(parseIndex+1), parseMessages, parseExpandedModels)
			parseRenderWorkerExample100BenchmarkSink = len(parseRequest.GetMessage) + len(parseRequest.GetModel)
		}
	})
	parseB.Run("current", func(parseCurrentB *testing.B) {
		parseCurrentB.ReportAllocs()
		parseCurrentB.ResetTimer()
		for parseIndex := 0; parseCurrentB.Loop(); parseIndex++ {
			parseRequest := parseBuildWorkerThreadCostSummaryRequest(uint64(parseIndex+1), parseMessages, parseExpandedModels)
			parseRenderWorkerExample100BenchmarkSink = len(parseRequest.GetMessage) + len(parseRequest.GetModel)
		}
	})
}

type renderWorkerLegacyCostMessageRequest struct {
	GetMessageIndex     int    `json:"messageIndex"`
	GetRoleBytes        []byte `json:"roleBytes"`
	GetHasContent       bool   `json:"hasContent"`
	GetPending          bool   `json:"pending"`
	GetModelIDBytes     []byte `json:"modelIDBytes"`
	GetPromptTokens     int    `json:"promptTokens"`
	GetCompletionTokens int    `json:"completionTokens"`
}

type renderWorkerLegacyThreadCostSummaryRequest struct {
	GetGeneration uint64                                 `json:"generation"`
	GetMessage    []renderWorkerLegacyCostMessageRequest `json:"message"`
	GetModel      []renderWorkerCostModelRequest         `json:"model"`
}

type renderWorkerLegacySignatureMessageRequest struct {
	GetMessageIndex     int    `json:"messageIndex"`
	GetRoleBytes        []byte `json:"roleBytes"`
	GetContentBytes     []byte `json:"contentBytes"`
	GetThoughtBytes     []byte `json:"thoughtBytes"`
	GetPending          bool   `json:"pending"`
	GetModelIDBytes     []byte `json:"modelIDBytes"`
	GetPromptTokens     int    `json:"promptTokens"`
	GetCompletionTokens int    `json:"completionTokens"`
}

type renderWorkerLegacySignatureRequest struct {
	GetGeneration uint64                                      `json:"generation"`
	GetMessage    []renderWorkerLegacySignatureMessageRequest `json:"message"`
	GetModel      []renderWorkerCostModelRequest              `json:"model"`
}

// BenchmarkBuildRenderWorkerRenderSignatureRequestCurrentVsLegacy compares the filtered render-signature request builder against the previous full-payload shape.
func BenchmarkBuildRenderWorkerRenderSignatureRequestCurrentVsLegacy(parseB *testing.B) {
	parseMessages, parseModels := parseBuildRenderWorkerExample100Fixture(180)
	parseExpandedModels := parseBuildRenderWorkerExample100ExpandedModels(parseModels, 64)
	parseB.Run("legacy", func(parseLegacyB *testing.B) {
		parseLegacyB.ReportAllocs()
		parseLegacyB.ResetTimer()
		for parseIndex := 0; parseLegacyB.Loop(); parseIndex++ {
			parseRequest := parseBuildRenderWorkerExample100LegacyRenderSignatureRequest(uint64(parseIndex+1), parseMessages, parseExpandedModels)
			parseRenderWorkerExample100BenchmarkSink = len(parseRequest.GetMessage) + len(parseRequest.GetModel)
		}
	})
	parseB.Run("current", func(parseCurrentB *testing.B) {
		parseCurrentB.ReportAllocs()
		parseCurrentB.ResetTimer()
		for parseIndex := 0; parseCurrentB.Loop(); parseIndex++ {
			parseRequest := parseBuildWorkerRenderSignatureRequest(uint64(parseIndex+1), parseMessages, parseExpandedModels)
			parseRenderWorkerExample100BenchmarkSink = len(parseRequest.GetMessage) + len(parseRequest.GetModel)
		}
	})
}

// BenchmarkBuildRenderSignatureStateCurrentVsLegacy compares compact hash signatures against the previous full-string concatenation path.
func BenchmarkBuildRenderSignatureStateCurrentVsLegacy(parseB *testing.B) {
	parseMessages, parseModels := parseBuildRenderWorkerExample100Fixture(180)
	parseExpandedModels := parseBuildRenderWorkerExample100ExpandedModels(parseModels, 64)
	parseB.Run("legacy", func(parseLegacyB *testing.B) {
		parseLegacyB.ReportAllocs()
		parseLegacyB.ResetTimer()
		for parseLegacyB.Loop() {
			parseSignatureState := parseBuildRenderSignatureStateLegacy(parseMessages, parseExpandedModels)
			parseRenderWorkerExample100BenchmarkSink = len(parseSignatureState.GetCompletedMarkdownSignature) + len(parseSignatureState.GetAssistantMetadataSignature) + len(parseSignatureState.GetThreadCostSignature)
		}
	})
	parseB.Run("current", func(parseCurrentB *testing.B) {
		parseCurrentB.ReportAllocs()
		parseCurrentB.ResetTimer()
		for parseCurrentB.Loop() {
			parseSignatureState := parseBuildRenderSignatureState(parseMessages, parseExpandedModels)
			parseRenderWorkerExample100BenchmarkSink = len(parseSignatureState.GetCompletedMarkdownSignature) + len(parseSignatureState.GetAssistantMetadataSignature) + len(parseSignatureState.GetThreadCostSignature)
		}
	})
}

// BenchmarkBuildRenderSignatureWorkerDispatchCurrentVsLegacy compares the legacy eager fallback-signature staging path against the current lazy-fallback worker dispatch path.
func BenchmarkBuildRenderSignatureWorkerDispatchCurrentVsLegacy(parseB *testing.B) {
	parseMessages, parseModels := parseBuildRenderWorkerExample100Fixture(180)
	parseB.Run("legacy_eager_fallback", func(parseLegacyB *testing.B) {
		parseLegacyB.ReportAllocs()
		parseLegacyB.ResetTimer()
		for parseIndex := 0; parseLegacyB.Loop(); parseIndex++ {
			parseWorkerMessages := append([]message(nil), parseMessages...)
			parseWorkerModels := append([]modelOption(nil), parseModels...)
			parseFallbackSignatures := parseBuildRenderSignatureState(parseWorkerMessages, parseWorkerModels)
			parseRequest := parseBuildWorkerRenderSignatureRequest(uint64(parseIndex+1), parseWorkerMessages, parseWorkerModels)
			parseRenderWorkerExample100BenchmarkSink = len(parseRequest.GetMessage) + len(parseRequest.GetModel) + len(parseFallbackSignatures.GetCompletedMarkdownSignature) + len(parseFallbackSignatures.GetAssistantMetadataSignature) + len(parseFallbackSignatures.GetThreadCostSignature)
		}
	})
	parseB.Run("current_lazy_fallback", func(parseCurrentB *testing.B) {
		parseCurrentB.ReportAllocs()
		parseCurrentB.ResetTimer()
		for parseIndex := 0; parseCurrentB.Loop(); parseIndex++ {
			parseWorkerMessages := append([]message(nil), parseMessages...)
			parseWorkerModels := append([]modelOption(nil), parseModels...)
			parseRequest := parseBuildWorkerRenderSignatureRequest(uint64(parseIndex+1), parseWorkerMessages, parseWorkerModels)
			parseRenderWorkerExample100BenchmarkSink = len(parseRequest.GetMessage) + len(parseRequest.GetModel)
		}
	})
}

// BenchmarkBuildRenderSignatureWorkerDispatchCloneVsDirect compares worker dispatch staging with the old render-snapshot slice clones against the direct immutable snapshot path.
func BenchmarkBuildRenderSignatureWorkerDispatchCloneVsDirect(parseB *testing.B) {
	parseMessages, parseModels := parseBuildRenderWorkerExample100Fixture(180)
	parseB.Run("with_clone", func(parseCloneB *testing.B) {
		parseCloneB.ReportAllocs()
		parseCloneB.ResetTimer()
		for parseIndex := 0; parseCloneB.Loop(); parseIndex++ {
			parseWorkerMessages := append([]message(nil), parseMessages...)
			parseWorkerModels := append([]modelOption(nil), parseModels...)
			parseRequest := parseBuildWorkerRenderSignatureRequest(uint64(parseIndex+1), parseWorkerMessages, parseWorkerModels)
			parseRenderWorkerExample100BenchmarkSink = len(parseRequest.GetMessage) + len(parseRequest.GetModel)
		}
	})
	parseB.Run("direct_snapshot", func(parseDirectB *testing.B) {
		parseDirectB.ReportAllocs()
		parseDirectB.ResetTimer()
		for parseIndex := 0; parseDirectB.Loop(); parseIndex++ {
			parseRequest := parseBuildWorkerRenderSignatureRequest(uint64(parseIndex+1), parseMessages, parseModels)
			parseRenderWorkerExample100BenchmarkSink = len(parseRequest.GetMessage) + len(parseRequest.GetModel)
		}
	})
}

// BenchmarkBuildRenderWorkerThreadCostDispatchCloneVsDirect compares thread-cost worker dispatch staging with cloned slices against the direct immutable snapshot path.
func BenchmarkBuildRenderWorkerThreadCostDispatchCloneVsDirect(parseB *testing.B) {
	parseMessages, parseModels := parseBuildRenderWorkerExample100Fixture(180)
	parseB.Run("with_clone", func(parseCloneB *testing.B) {
		parseCloneB.ReportAllocs()
		parseCloneB.ResetTimer()
		for parseIndex := 0; parseCloneB.Loop(); parseIndex++ {
			parseWorkerMessages := append([]message(nil), parseMessages...)
			parseWorkerModels := append([]modelOption(nil), parseModels...)
			parseRequest := parseBuildWorkerThreadCostSummaryRequest(uint64(parseIndex+1), parseWorkerMessages, parseWorkerModels)
			parseRenderWorkerExample100BenchmarkSink = len(parseRequest.GetMessage) + len(parseRequest.GetModel)
		}
	})
	parseB.Run("direct_snapshot", func(parseDirectB *testing.B) {
		parseDirectB.ReportAllocs()
		parseDirectB.ResetTimer()
		for parseIndex := 0; parseDirectB.Loop(); parseIndex++ {
			parseRequest := parseBuildWorkerThreadCostSummaryRequest(uint64(parseIndex+1), parseMessages, parseModels)
			parseRenderWorkerExample100BenchmarkSink = len(parseRequest.GetMessage) + len(parseRequest.GetModel)
		}
	})
}

// BenchmarkBuildAssistantMessageMetadataDeltaDispatchCloneVsDirect compares the old metadata-item clone before worker dispatch against the direct freshly-built delta item list.
func BenchmarkBuildAssistantMessageMetadataDeltaDispatchCloneVsDirect(parseB *testing.B) {
	parseMessages, _ := parseBuildRenderWorkerExample100Fixture(180)
	parseMessageItems := parseBuildAssistantMessageMetadataItems(parseMessages)
	parseB.Run("with_clone", func(parseCloneB *testing.B) {
		parseCloneB.ReportAllocs()
		parseCloneB.ResetTimer()
		for parseCloneB.Loop() {
			parseClonedItems := append([]renderWorkerMessageMetadataMessageRequest(nil), parseMessageItems...)
			parseRenderWorkerExample100BenchmarkSink = len(parseClonedItems)
		}
	})
	parseB.Run("direct_snapshot", func(parseDirectB *testing.B) {
		parseDirectB.ReportAllocs()
		parseDirectB.ResetTimer()
		for parseDirectB.Loop() {
			parseRenderWorkerExample100BenchmarkSink = len(parseMessageItems)
		}
	})
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

// parseBuildRenderWorkerExample100ExpandedModels appends deterministic unused models so request-builder benchmarks can measure pruning behavior.
func parseBuildRenderWorkerExample100ExpandedModels(parseBaseModels []modelOption, parseTargetCount int) []modelOption {
	parseModels := append([]modelOption(nil), parseBaseModels...)
	for len(parseModels) < parseTargetCount {
		parseIndex := len(parseModels) + 1
		parseModels = append(parseModels, modelOption{
			ID:    fmt.Sprintf("unused-model-%d", parseIndex),
			Label: fmt.Sprintf("Unused model %d", parseIndex),
			Pricing: modelPricing{
				InputDollarsPerMillion:  float64(parseIndex) * 0.1,
				OutputDollarsPerMillion: float64(parseIndex) * 0.2,
				Currency:                "USD",
			},
		})
	}
	return parseModels
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

// parseBuildRenderWorkerExample100LegacyThreadCostSummaryRequest preserves the previous full-payload worker request shape for benchmark comparison.
func parseBuildRenderWorkerExample100LegacyThreadCostSummaryRequest(parseGeneration uint64, parseMessages []message, parseModels []modelOption) renderWorkerLegacyThreadCostSummaryRequest {
	parseWorkerMessages := make([]renderWorkerLegacyCostMessageRequest, 0, len(parseMessages))
	for parseMessageIndex, parseMessageItem := range parseMessages {
		parseWorkerMessages = append(parseWorkerMessages, renderWorkerLegacyCostMessageRequest{
			GetMessageIndex:     parseMessageIndex,
			GetRoleBytes:        []byte(parseMessageItem.Role),
			GetHasContent:       strings.TrimSpace(parseMessageItem.Content) != "",
			GetPending:          parseMessageItem.Pending,
			GetModelIDBytes:     []byte(parseMessageItem.ModelID),
			GetPromptTokens:     parseMessageItem.PromptTokens,
			GetCompletionTokens: parseMessageItem.CompletionTokens,
		})
	}
	parseWorkerModels := make([]renderWorkerCostModelRequest, 0, len(parseModels))
	for _, parseModel := range parseModels {
		parseWorkerModels = append(parseWorkerModels, renderWorkerCostModelRequest{
			GetModelIDBytes:            []byte(parseModel.ID),
			GetInputDollarsPerMillion:  parseModel.Pricing.InputDollarsPerMillion,
			GetOutputDollarsPerMillion: parseModel.Pricing.OutputDollarsPerMillion,
			GetCurrencyBytes:           []byte(parseModel.Pricing.Currency),
		})
	}
	return renderWorkerLegacyThreadCostSummaryRequest{
		GetGeneration: parseGeneration,
		GetMessage:    parseWorkerMessages,
		GetModel:      parseWorkerModels,
	}
}

// parseBuildRenderWorkerExample100LegacyRenderSignatureRequest preserves the previous full-payload render-signature request shape for benchmark comparison.
func parseBuildRenderWorkerExample100LegacyRenderSignatureRequest(parseGeneration uint64, parseMessages []message, parseModels []modelOption) renderWorkerLegacySignatureRequest {
	parseWorkerMessages := make([]renderWorkerLegacySignatureMessageRequest, 0, len(parseMessages))
	for parseMessageIndex, parseMessageItem := range parseMessages {
		parseWorkerMessages = append(parseWorkerMessages, renderWorkerLegacySignatureMessageRequest{
			GetMessageIndex:     parseMessageIndex,
			GetRoleBytes:        []byte(parseMessageItem.Role),
			GetContentBytes:     []byte(parseMessageItem.Content),
			GetThoughtBytes:     []byte(parseMessageItem.Thought),
			GetPending:          parseMessageItem.Pending,
			GetModelIDBytes:     []byte(parseMessageItem.ModelID),
			GetPromptTokens:     parseMessageItem.PromptTokens,
			GetCompletionTokens: parseMessageItem.CompletionTokens,
		})
	}
	parseWorkerModels := make([]renderWorkerCostModelRequest, 0, len(parseModels))
	for _, parseModel := range parseModels {
		parseWorkerModels = append(parseWorkerModels, renderWorkerCostModelRequest{
			GetModelIDBytes:            []byte(parseModel.ID),
			GetInputDollarsPerMillion:  parseModel.Pricing.InputDollarsPerMillion,
			GetOutputDollarsPerMillion: parseModel.Pricing.OutputDollarsPerMillion,
			GetCurrencyBytes:           []byte(parseModel.Pricing.Currency),
		})
	}
	return renderWorkerLegacySignatureRequest{
		GetGeneration: parseGeneration,
		GetMessage:    parseWorkerMessages,
		GetModel:      parseWorkerModels,
	}
}

// parseBuildRenderSignatureStateLegacy preserves the previous full-string local signature path for benchmark comparison.
func parseBuildRenderSignatureStateLegacy(parseMessages []message, parseModels []modelOption) renderWorkerSignatureState {
	return renderWorkerSignatureState{
		GetCompletedMarkdownSignature: parseBuildCompletedAssistantMessagesMarkdownSignatureLegacy(parseMessages),
		GetAssistantMetadataSignature: parseBuildAssistantMessageMetadataSignatureLegacy(parseMessages),
		GetThreadCostSignature:        parseBuildThreadCostSummarySignatureLegacy(parseMessages, parseModels),
	}
}

// parseBuildCompletedAssistantMessagesMarkdownSignatureLegacy preserves the previous concatenated markdown signature path for benchmark comparison.
func parseBuildCompletedAssistantMessagesMarkdownSignatureLegacy(parseMessages []message) string {
	var parseBuilder strings.Builder
	for _, parseMessageItem := range parseMessages {
		if parseMessageItem.Role != roleAssistant || parseMessageItem.Pending {
			continue
		}
		if strings.TrimSpace(parseMessageItem.Content) == "" {
			continue
		}
		parseBuilder.WriteString(parseMessageItem.Content)
		parseBuilder.WriteString("\n\x1f\n")
	}
	return parseBuilder.String()
}

// parseBuildAssistantMessageMetadataSignatureLegacy preserves the previous concatenated metadata signature path for benchmark comparison.
func parseBuildAssistantMessageMetadataSignatureLegacy(parseMessages []message) string {
	var parseBuilder strings.Builder
	for parseMessageIndex, parseMessageItem := range parseMessages {
		if parseMessageItem.Role != roleAssistant || parseMessageItem.Pending {
			continue
		}
		parseBuilder.WriteString(parseBuildMessageIndexString(parseMessageIndex))
		parseBuilder.WriteString("|")
		parseBuilder.WriteString(parseMessageItem.Content)
		parseBuilder.WriteString("|")
		parseBuilder.WriteString(parseMessageItem.Thought)
		parseBuilder.WriteString("\n\x1e\n")
	}
	return parseBuilder.String()
}

// parseBuildAssistantMessageMetadataTextByMessageLegacy preserves the previous byte-to-string map reconstruction path for benchmark comparison.
func parseBuildAssistantMessageMetadataTextByMessageLegacy(parseItems []renderWorkerMessageMetadataMessageRequest) map[int]renderWorkerMessageMetadataText {
	parseTextByMessage := make(map[int]renderWorkerMessageMetadataText, len(parseItems))
	for _, parseItem := range parseItems {
		parseTextByMessage[parseItem.GetMessageIndex] = renderWorkerMessageMetadataText{
			GetContentText: string(parseItem.GetContentBytes),
			GetThoughtText: string(parseItem.GetThoughtBytes),
		}
	}
	return parseTextByMessage
}

// parseBuildThoughtSectionResultsFromWorkerLegacy preserves the previous append-heavy thought-section conversion path for benchmark comparison.
func parseBuildThoughtSectionResultsFromWorkerLegacy(parseMessageIndex int, parseSectionResults []renderWorkerThoughtSectionResult) []thoughtSection {
	if len(parseSectionResults) == 0 {
		return nil
	}
	parseSections := make([]thoughtSection, 0, len(parseSectionResults))
	for parseSectionIndex, parseSectionResult := range parseSectionResults {
		parseHeading := string(parseSectionResult.GetHeadingBytes)
		parseSections = append(parseSections, thoughtSection{
			Key:     fmt.Sprintf("%d:%d:%s", parseMessageIndex, parseSectionIndex, strings.TrimSpace(parseHeading)),
			Heading: parseHeading,
			Body:    string(parseSectionResult.GetBodyBytes),
		})
	}
	return parseSections
}

// parseBuildCanvasArtifactResultsFromWorkerLegacy preserves the previous append-heavy canvas-artifact conversion path for benchmark comparison.
func parseBuildCanvasArtifactResultsFromWorkerLegacy(parseMessageIndex int, parseArtifactResults []renderWorkerCanvasArtifactResult) []canvasArtifact {
	if len(parseArtifactResults) == 0 {
		return nil
	}
	parseArtifacts := make([]canvasArtifact, 0, len(parseArtifactResults))
	for _, parseArtifactResult := range parseArtifactResults {
		parseArtifacts = append(parseArtifacts, canvasArtifact{
			ID:           string(parseArtifactResult.GetIDBytes),
			MessageIndex: parseMessageIndex,
			Label:        string(parseArtifactResult.GetLabelBytes),
		})
	}
	return parseArtifacts
}

// parseBuildThreadCostSummarySignatureLegacy preserves the previous concatenated thread-cost signature path for benchmark comparison.
func parseBuildThreadCostSummarySignatureLegacy(parseMessages []message, parseModels []modelOption) string {
	var parseBuilder strings.Builder
	parseUsedModelIDs := parseBuildUsedAssistantModelIDSet(parseMessages)
	for _, parseOption := range parseModels {
		if _, hasParseUsedModel := parseUsedModelIDs[parseOption.ID]; !hasParseUsedModel {
			continue
		}
		parseBuilder.WriteString(fmt.Sprintf("model|%s|%.6f|%.6f|%s\n", parseOption.ID, parseOption.Pricing.InputDollarsPerMillion, parseOption.Pricing.OutputDollarsPerMillion, parseOption.Pricing.Currency))
	}
	parseBuilder.WriteString("--\n")
	for parseMessageIndex, parseMessageItem := range parseMessages {
		if parseMessageItem.Role != roleAssistant || parseMessageItem.Pending {
			continue
		}
		if strings.TrimSpace(parseMessageItem.Content) == "" {
			continue
		}
		parseBuilder.WriteString(fmt.Sprintf("%d|%s|%d|%d\n", parseMessageIndex, parseMessageItem.ModelID, parseMessageItem.PromptTokens, parseMessageItem.CompletionTokens))
	}
	return parseBuilder.String()
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
		parseMessages = append(parseMessages, message{
			Role:             roleAssistant,
			Content:          "content",
			Pending:          false,
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
