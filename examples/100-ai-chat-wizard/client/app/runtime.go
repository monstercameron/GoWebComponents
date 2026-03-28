//go:build js && wasm

package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/examples/shared/renderworker"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/ui"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"
)

type grpcMonitorResult struct {
	Reason    string
	Sleep     bool
	Immediate bool
}

// useAppRuntime owns the worker lifecycle, gRPC connection, and markdown
// rendering side effects so App() can stay focused on state and composition.
func parseUseAppRuntime(
	parseApp ui.Reducer[appState, appAction],
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	parseMarkdownWorkerRef ui.Ref[*interop.Worker],
	parseMarkdownWorkerPoolRef ui.Ref[*interop.WorkerPool],
	parseMarkdownRenderInFlight ui.Ref[map[string]bool],
	parseMarkdownRenderVersion ui.State[int],
	parseMarkdownRenderTick int,
	parseCompletedMarkdownSignature string,
	parseAssistantMetadataSignature string,
	parseThreadCostSignature string,
	parseThoughtCacheByMessageState ui.State[map[int]renderWorkerThoughtCacheEntry],
	parseCanvasCacheByMessageState ui.State[map[int]renderWorkerCanvasCacheEntry],
	parseThreadCostSummaryState ui.State[threadCostSummary],
	parseThreadCostSummarySignatureState ui.State[string],
	parseOnConversationRefresh func(bool),
	parseOnProfileRefresh func(bool),
) {
	parseMarkdownWorkerSubscriptionRef := ui.UseRef[*interop.Subscription](nil)
	parseMetadataGenerationRef := ui.UseRef(uint64(0))
	parseThreadCostGenerationRef := ui.UseRef(uint64(0))

	ui.UseEffect(func() func() {
		if !parseApp.Get().AuthResolved || parseApp.Get().MarkdownWorkerFallback || parseMarkdownWorkerRef.Get() != nil || parseMarkdownWorkerPoolRef.Get() != nil {
			return nil
		}
		parseCtx, parseCancel := context.WithCancel(context.Background())

		// Worker startup blocks on a channel select waiting for the worker's
		// "ready" message.  Effects run synchronously on the main goroutine,
		// so blocking here would starve the JS event loop and prevent the
		// worker callback from ever firing.  Run it in a separate goroutine.
		go func() {
			buildWorker := func(parseReadyTimeout time.Duration) (interop.Worker, error) {
				return interop.OpenGoWASMWorker(parseCtx, interop.GoWASMWorkerOptions{
					RuntimeURL:   backgroundWorkerRuntimeURL,
					WASMURL:      backgroundWorkerWASMURL + parseCurrentWASMQuerySuffix(),
					Name:         "chat-background",
					Ready:        true,
					ReadyTimeout: parseReadyTimeout,
				})
			}

			parseWorker, parseErr := buildWorker(5 * time.Second)
			if parseErr != nil && interop.IsCode(parseErr, interop.CodeTimeout) {
				chatLog.Info("background worker startup timed out; retrying", logging.Fields{"error": parseErr})
				select {
				case <-parseCtx.Done():
					return
				case <-time.After(1200 * time.Millisecond):
				}
				parseWorker, parseErr = buildWorker(12 * time.Second)
			}
			if parseErr != nil && interop.IsCode(parseErr, interop.CodeTimeout) {
				chatLog.Info("background worker startup timed out; continuing without worker", logging.Fields{"error": parseErr})
				return
			}
			if parseErr != nil {
				parseApp.Dispatch(appAction{Type: appActionSetMarkdownWorkerFallback, MarkdownWorkerFallback: true})
				chatLog.Warn("background worker unavailable; falling back to main-thread work", logging.Fields{"error": parseErr})
				return
			}
			parseSub, parseSubErr := parseWorker.Subscribe(func(parseMsg2 interop.WorkerMessage, parseErr4 error) {
				if parseErr4 != nil {
					chatLog.Warn("background worker subscription error", logging.Fields{"error": parseErr4})
					return
				}
				if parseMsg2.Name == backgroundWorkerEventTick && parseOnConversationRefresh != nil {
					parseOnConversationRefresh(false)
				}
			})
			if parseSubErr != nil {
				parseApp.Dispatch(appAction{Type: appActionSetMarkdownWorkerFallback, MarkdownWorkerFallback: true})
				chatLog.Warn("background worker subscribe failed; falling back to main-thread work", logging.Fields{"error": parseSubErr})
				_ = parseWorker.Terminate()
				return
			}
			if parseCtx.Err() != nil {
				parseSub.Cancel()
				_ = parseWorker.Terminate()
				return
			}
			parseMarkdownWorkerSubscriptionRef.Set(&parseSub)
			chatLog.Info("worker ready", nil)
			parseMarkdownWorkerRef.Set(&parseWorker)
			parsePool, parsePoolErr := renderworker.BuildRenderWorkerPool(parseCtx, renderworker.PoolOptions{
				GetSize:         backgroundWorkerRenderPoolSize,
				GetQueueLimit:   backgroundWorkerRenderPoolSize,
				GetRuntimeURL:   backgroundWorkerRuntimeURL,
				GetWASMURL:      backgroundWorkerWASMURL + parseCurrentWASMQuerySuffix(),
				GetNamePrefix:   "chat-render",
				IsReady:         true,
				GetReadyTimeout: 5 * time.Second,
			})
			if parsePoolErr != nil {
				chatLog.Warn("background render worker pool unavailable; using single worker", logging.Fields{"error": parsePoolErr, "pool_size": backgroundWorkerRenderPoolSize})
			} else {
				parseMarkdownWorkerPoolRef.Set(&parsePool)
				chatLog.Info("background render worker pool ready", logging.Fields{"pool_size": backgroundWorkerRenderPoolSize})
			}
			// Nudge the markdown-render effect once the worker is available, but
			// do not let that state change retrigger worker startup itself.
			parseMarkdownRenderVersion.Update(func(parsePrevious int) int {
				return parsePrevious + 1
			})
		}()

		return func() {
			parseCancel()
			chatLog.Info("worker stop", nil)
			if parsePool := parseMarkdownWorkerPoolRef.Get(); parsePool != nil {
				_ = parsePool.Close()
				parseMarkdownWorkerPoolRef.Set(nil)
			}
			if parseSub := parseMarkdownWorkerSubscriptionRef.Get(); parseSub != nil {
				parseSub.Cancel()
				parseMarkdownWorkerSubscriptionRef.Set(nil)
			}
			if parseW := parseMarkdownWorkerRef.Get(); parseW != nil {
				parseMarkdownWorkerRef.Set(nil)
				_ = parseW.Terminate()
			}
		}
	}, parseApp.Get().AuthResolved, parseApp.Get().MarkdownWorkerFallback)

	ui.UseEffect(func() func() {
		parseCtx2, parseCancel2 := context.WithCancel(context.Background())
		parseWakeCh := make(chan string, 1)
		parseSleepCh := make(chan string, 1)
		parseReconnectCh := make(chan string, 1)

		parseSignal := func(parseCh chan string, parseReason string) {
			parseReason = strings.TrimSpace(parseReason)
			select {
			case parseCh <- parseReason:
			default:
			}
		}
		parseDisconnect := func(parseConn2 *grpc.ClientConn, parseReason2 string) {
			parseSyncGRPCReadyState(parseApp, parseChatClientRef, parseMarkdownWorkerRef, nil, false, parseReason2, parseOnConversationRefresh, parseOnProfileRefresh)
			if parseConn2 != nil {
				_ = parseConn2.Close()
			}
		}

		parseUnregisterReconnect := parseRegisterGRPCReconnectHandler(func(parseReason3 string) {
			parseSignal(parseReconnectCh, parseReason3)
		})

		parseWindow := js.Global().Get("window")
		parseDocument := js.Global().Get("document")
		parseStopHiddenSleep := func() {}
		var parseStartHiddenSleep func(time.Duration)
		parseStartHiddenSleep = func(parseDelay3 time.Duration) {
			parseStopHiddenSleep()
			if parseRuntimePageVisible() {
				return
			}
			parseTimer := time.AfterFunc(parseDelay3, func() {
				if parseRuntimePageVisible() {
					return
				}
				if parseApp.Get().Streaming {
					parseStartHiddenSleep(grpcSleepRetry)
					return
				}
				parseSignal(parseSleepCh, "hidden idle")
			})
			parseStopHiddenSleep = func() {
				if parseTimer != nil {
					parseTimer.Stop()
				}
			}
		}

		parseCleanupOnline := parseAttachWindowListener(parseWindow, "online", func() {
			parseStopHiddenSleep()
			parseSignal(parseWakeCh, "browser online")
		})
		parseCleanupOffline := parseAttachWindowListener(parseWindow, "offline", func() {
			parseStopHiddenSleep()
			parseSignal(parseSleepCh, "browser offline")
		})
		parseCleanupFocus := parseAttachWindowListener(parseWindow, "focus", func() {
			parseStopHiddenSleep()
			parseSignal(parseWakeCh, "window focus")
		})
		parseCleanupVisibility := parseAttachDocumentListener(parseDocument, "visibilitychange", func() {
			if parseRuntimePageVisible() {
				parseStopHiddenSleep()
				parseSignal(parseWakeCh, "document visible")
				return
			}
			parseStartHiddenSleep(grpcSleepAfter)
		})
		if !parseRuntimePageVisible() {
			parseStartHiddenSleep(grpcSleepAfter)
		}

		go func() {
			defer parseSyncGRPCReadyState(parseApp, parseChatClientRef, parseMarkdownWorkerRef, nil, false, "runtime stopped", parseOnConversationRefresh, parseOnProfileRefresh)

			isParseSleeping := false
			parseAttempt := 0
			for {
				if parseCtx2.Err() != nil {
					return
				}
				if isParseSleeping || !parseRuntimeNavigatorOnline() {
					parseSyncGRPCReadyState(parseApp, parseChatClientRef, parseMarkdownWorkerRef, nil, false, "bridge sleeping", parseOnConversationRefresh, parseOnProfileRefresh)
					parseWakeReason, parseOk := parseWaitForWakeSignal(parseCtx2, parseWakeCh)
					if !parseOk {
						return
					}
					isParseSleeping = false
					parseAttempt = 0
					if parseWakeReason != "" {
						chatLog.Info("grpc wake", logging.Fields{"reason": parseWakeReason})
					}
				}

				parseDialCtx, parseCancelDial := context.WithTimeout(parseCtx2, grpcDialTimeout)
				parseDialOptions := grpctunnel.ApplyTunnelInsecureCredentials([]grpc.DialOption{
					grpc.WithUnaryInterceptor(func(parseCtx3 context.Context, parseMethod string, parseReq interface{}, parseReply interface{}, parseCc *grpc.ClientConn, parseInvoker grpc.UnaryInvoker, parseOpts ...grpc.CallOption) error {
						return parseInvoker(parseAuthContextWithMetadata(parseCtx3), parseMethod, parseReq, parseReply, parseCc, parseOpts...)
					}),
					grpc.WithStreamInterceptor(func(parseCtx4 context.Context, parseDesc *grpc.StreamDesc, parseCc2 *grpc.ClientConn, parseMethod2 string, parseStreamer grpc.Streamer, parseOpts2 ...grpc.CallOption) (grpc.ClientStream, error) {
						return parseStreamer(parseAuthContextWithMetadata(parseCtx4), parseDesc, parseCc2, parseMethod2, parseOpts2...)
					}),
					grpc.WithBlock(),
				})
				parseConn, parseErr2 := grpctunnel.BuildTunnelConn(parseDialCtx, grpctunnel.TunnelConfig{
					Target:      grpcEndpoint,
					GRPCOptions: parseDialOptions,
				})
				parseCancelDial()
				if parseErr2 != nil {
					parseDelay := parseGrpcReconnectDelay(parseAttempt)
					parseAttempt++
					parseSyncGRPCReadyState(parseApp, parseChatClientRef, parseMarkdownWorkerRef, nil, false, "dial failed", parseOnConversationRefresh, parseOnProfileRefresh)
					chatLog.Warn("grpc dial failed", logging.Fields{"error": parseErr2, "retry_in_ms": int(parseDelay / time.Millisecond)})
					switch parseWaitForReconnectDelay(parseCtx2, parseDelay, parseWakeCh, parseSleepCh, parseReconnectCh) {
					case "sleep":
						isParseSleeping = true
					case "wake", "reconnect":
						parseAttempt = 0
					case "done":
						return
					}
					continue
				}

				parseAttempt = 0
				parseSyncGRPCReadyState(parseApp, parseChatClientRef, parseMarkdownWorkerRef, parseConn, true, "bridge connected", parseOnConversationRefresh, parseOnProfileRefresh)
				parseResult := parseMonitorGRPCConnection(parseCtx2, parseApp, parseChatClientRef, parseMarkdownWorkerRef, parseConn, parseWakeCh, parseSleepCh, parseReconnectCh)
				parseDisconnect(parseConn, parseResult.Reason)
				if parseCtx2.Err() != nil {
					return
				}
				if parseResult.Sleep {
					isParseSleeping = true
					continue
				}
				if parseResult.Immediate {
					continue
				}
				parseDelay2 := parseGrpcReconnectDelay(parseAttempt)
				parseAttempt++
				switch parseWaitForReconnectDelay(parseCtx2, parseDelay2, parseWakeCh, parseSleepCh, parseReconnectCh) {
				case "sleep":
					isParseSleeping = true
				case "wake", "reconnect":
					parseAttempt = 0
				case "done":
					return
				}
			}
		}()

		return func() {
			parseCancel2()
			parseStopHiddenSleep()
			parseCleanupVisibility()
			parseCleanupFocus()
			parseCleanupOffline()
			parseCleanupOnline()
			parseUnregisterReconnect()
		}
	}, true)

	ui.UseEffect(func() func() {
		if !parseApp.Get().GRPCReady {
			parseSetClientLogRelay(nil, "", false)
			return nil
		}
		parseClient := parseChatClientRef.Get()
		if parseClient == nil {
			return nil
		}
		parseSetClientLogRelay(parseClient, parseLoadPersistedClientIdentity(), true)
		parseEnsureClientIdentity(parseClient)
		return nil
	}, parseApp.Get().GRPCReady)

	ui.UseEffect(func() func() {
		if parseApp.Get().MarkdownWorkerFallback {
			return nil
		}
		var parseRequester interop.WorkerRequester
		parseBatchCount := 1
		if parsePool := parseMarkdownWorkerPoolRef.Get(); parsePool != nil {
			parseRequester = *parsePool
			parseBatchCount = parsePool.GetSize()
		} else if parseWorker := parseMarkdownWorkerRef.Get(); parseWorker != nil {
			parseRequester = *parseWorker
		}
		if parseRequester == nil {
			return nil
		}
		parseInFlight := parseMarkdownRenderInFlight.Get()
		parseSources := make([]string, 0, len(parseApp.Get().Messages))
		for _, parseMsg := range parseApp.Get().Messages {
			if parseMsg.Role != roleAssistant || parseMsg.Pending {
				continue
			}
			parseSource := strings.TrimSpace(parseMsg.Content)
			if parseSource == "" {
				continue
			}
			if _, parseOk2 := cachedRenderedMarkdown(parseSource); parseOk2 {
				continue
			}
			if parseInFlight[parseSource] {
				continue
			}
			parseInFlight[parseSource] = true
			parseSources = append(parseSources, parseSource)
		}
		if len(parseSources) == 0 {
			return nil
		}
		parseSourceBatches := parseBuildMarkdownSourceBatches(parseSources, parseBatchCount)
		go func() {
			type parseBatchResult struct {
				Results []markdownRenderResult
				Error   error
			}
			parseResultCh := make(chan parseBatchResult, len(parseSourceBatches))
			var parseWait sync.WaitGroup
			for _, parseBatchSources := range parseSourceBatches {
				parseBatchSourcesCopy := append([]string(nil), parseBatchSources...)
				parseWait.Add(1)
				go func(parseRequestSources []string) {
					defer parseWait.Done()
					parseBatchResponse, parseBatchErr := interop.RequestWorkerDecoded[markdownRenderBatchRequest, struct{}, markdownRenderBatchResult](
						context.Background(),
						parseRequester,
						backgroundWorkerRequestRenderMarkdownBatch,
						parseBuildMarkdownBinaryBatchRequest(parseRequestSources),
						nil,
					)
					if parseBatchErr != nil {
						parseResultCh <- parseBatchResult{Error: parseBatchErr}
						return
					}
					parseResultCh <- parseBatchResult{Results: parseBatchResponse.Results}
				}(parseBatchSourcesCopy)
			}
			parseWait.Wait()
			close(parseResultCh)

			parseRenderedBySource := map[string]string{}
			var parseBatchError error
			for parseBatch := range parseResultCh {
				if parseBatch.Error != nil && parseBatchError == nil {
					parseBatchError = parseBatch.Error
				}
				for _, parseRendered := range parseBatch.Results {
					parseRenderedBySource[parseBuildMarkdownSourceKey(parseRendered.GetSourceBytes)] = string(parseRendered.GetHTMLBytes)
				}
			}

			if parseBatchError != nil {
				for _, parseSource2 := range parseSources {
					delete(parseInFlight, parseSource2)
					cacheRenderedMarkdown(parseSource2, renderMarkdownSync(parseSource2))
				}
				parseApp.Dispatch(appAction{Type: appActionSetMarkdownWorkerFallback, MarkdownWorkerFallback: true})
				parseMarkdownRenderVersion.Set(parseMarkdownRenderVersion.Get() + 1)
				chatLog.Warn("markdown worker batch request failed; using main-thread fallback", logging.Fields{"error": parseBatchError, "sources": len(parseSources)})
				return
			}
			for _, parseSource3 := range parseSources {
				delete(parseInFlight, parseSource3)
				parseRenderedHTML, hasRenderedHTML := parseRenderedBySource[parseSource3]
				if !hasRenderedHTML {
					parseRenderedHTML = renderMarkdownSync(parseSource3)
				}
				cacheRenderedMarkdown(parseSource3, parseRenderedHTML)
			}
			parseMarkdownRenderVersion.Set(parseMarkdownRenderVersion.Get() + 1)
		}()
		return nil
	}, parseApp.Get().MarkdownWorkerFallback, parseCompletedMarkdownSignature, parseMarkdownRenderTick)

	ui.UseEffect(func() func() {
		if parseApp.Get().MarkdownWorkerFallback {
			return nil
		}
		parseRequester, parseBatchCount := parseResolveBackgroundRenderRequester(parseMarkdownWorkerRef, parseMarkdownWorkerPoolRef)
		if parseRequester == nil {
			parseThoughtCacheByMessage := map[int]renderWorkerThoughtCacheEntry{}
			parseCanvasCacheByMessage := map[int]renderWorkerCanvasCacheEntry{}
			for parseMessageIndex, parseMessageItem := range parseApp.Get().Messages {
				if parseMessageItem.Role != roleAssistant || parseMessageItem.Pending {
					continue
				}
				parseThoughtCacheByMessage[parseMessageIndex] = renderWorkerThoughtCacheEntry{
					GetThoughtText: parseMessageItem.Thought,
					GetSection:     parseThoughtSections(parseMessageIndex, parseMessageItem.Thought),
				}
				parseCanvasCacheByMessage[parseMessageIndex] = renderWorkerCanvasCacheEntry{
					GetContentText: parseMessageItem.Content,
					GetArtifact:    canvasArtifactsFromMarkdown(parseMessageIndex, parseMessageItem.Content),
				}
			}
			parseThoughtCacheByMessageState.Set(parseThoughtCacheByMessage)
			parseCanvasCacheByMessageState.Set(parseCanvasCacheByMessage)
			return nil
		}
		parseMessageItems := parseBuildAssistantMessageMetadataItems(parseApp.Get().Messages)
		parseGeneration := parseMetadataGenerationRef.Get() + 1
		parseMetadataGenerationRef.Set(parseGeneration)
		if len(parseMessageItems) == 0 {
			parseThoughtCacheByMessageState.Set(map[int]renderWorkerThoughtCacheEntry{})
			parseCanvasCacheByMessageState.Set(map[int]renderWorkerCanvasCacheEntry{})
			return nil
		}
		go func(parseExpectedGeneration uint64, parseItems []renderWorkerMessageMetadataMessageRequest, parsePoolBatchCount int) {
			parseThoughtCacheByMessage, parseCanvasCacheByMessage, parseErr := parseRequestAssistantMessageMetadata(context.Background(), parseRequester, parseExpectedGeneration, parseItems, parsePoolBatchCount)
			if parseErr != nil {
				chatLog.Warn("message metadata worker request failed; using synchronous fallback", logging.Fields{"error": parseErr, "messages": len(parseItems)})
				parseThoughtFallbackByMessage := map[int]renderWorkerThoughtCacheEntry{}
				parseCanvasFallbackByMessage := map[int]renderWorkerCanvasCacheEntry{}
				for _, parseItem := range parseItems {
					parseContentText := string(parseItem.GetContentBytes)
					parseThoughtText := string(parseItem.GetThoughtBytes)
					parseThoughtFallbackByMessage[parseItem.GetMessageIndex] = renderWorkerThoughtCacheEntry{
						GetThoughtText: parseThoughtText,
						GetSection:     parseThoughtSections(parseItem.GetMessageIndex, parseThoughtText),
					}
					parseCanvasFallbackByMessage[parseItem.GetMessageIndex] = renderWorkerCanvasCacheEntry{
						GetContentText: parseContentText,
						GetArtifact:    canvasArtifactsFromMarkdown(parseItem.GetMessageIndex, parseContentText),
					}
				}
				if parseMetadataGenerationRef.Get() == parseExpectedGeneration {
					parseThoughtCacheByMessageState.Set(parseThoughtFallbackByMessage)
					parseCanvasCacheByMessageState.Set(parseCanvasFallbackByMessage)
				}
				return
			}
			if parseMetadataGenerationRef.Get() != parseExpectedGeneration {
				return
			}
			parseThoughtCacheByMessageState.Set(parseThoughtCacheByMessage)
			parseCanvasCacheByMessageState.Set(parseCanvasCacheByMessage)
		}(parseGeneration, append([]renderWorkerMessageMetadataMessageRequest(nil), parseMessageItems...), parseBatchCount)
		return nil
	}, parseApp.Get().MarkdownWorkerFallback, parseAssistantMetadataSignature, parseMarkdownRenderTick)

	ui.UseEffect(func() func() {
		if parseApp.Get().MarkdownWorkerFallback {
			parseThreadCostSummarySignatureState.Set("")
			return nil
		}
		parseRequester, _ := parseResolveBackgroundRenderRequester(parseMarkdownWorkerRef, parseMarkdownWorkerPoolRef)
		if parseRequester == nil {
			parseThreadCostSummaryState.Set(parseDeriveThreadCostSummary(parseApp.Get().Messages, parseApp.Get().ModelOptions))
			parseThreadCostSummarySignatureState.Set(parseThreadCostSignature)
			return nil
		}
		parseGeneration := parseThreadCostGenerationRef.Get() + 1
		parseThreadCostGenerationRef.Set(parseGeneration)
		parseMessages := append([]message(nil), parseApp.Get().Messages...)
		parseModels := append([]modelOption(nil), parseApp.Get().ModelOptions...)
		go func(parseExpectedGeneration uint64, parseWorkerMessages []message, parseWorkerModels []modelOption, parseExpectedSignature string) {
			parseSummary, parseErr := parseRequestWorkerThreadCostSummary(context.Background(), parseRequester, parseExpectedGeneration, parseWorkerMessages, parseWorkerModels)
			if parseErr != nil {
				chatLog.Warn("thread cost worker request failed; keeping previous summary", logging.Fields{"error": parseErr})
				if parseThreadCostGenerationRef.Get() == parseExpectedGeneration {
					parseThreadCostSummaryState.Set(parseDeriveThreadCostSummary(parseWorkerMessages, parseWorkerModels))
					parseThreadCostSummarySignatureState.Set(parseExpectedSignature)
				}
				return
			}
			if parseThreadCostGenerationRef.Get() != parseExpectedGeneration {
				return
			}
			parseThreadCostSummaryState.Set(parseSummary)
			parseThreadCostSummarySignatureState.Set(parseExpectedSignature)
		}(parseGeneration, parseMessages, parseModels, parseThreadCostSignature)
		return nil
	}, parseApp.Get().MarkdownWorkerFallback, parseThreadCostSignature, parseMarkdownRenderTick)
}

// parseBuildMarkdownSourceBatches splits one source list into balanced batches for multi-worker dispatch.
func parseBuildMarkdownSourceBatches(parseSources []string, parseBatchCount int) [][]string {
	if len(parseSources) == 0 {
		return nil
	}
	if parseBatchCount < 1 {
		parseBatchCount = 1
	}
	if parseBatchCount > len(parseSources) {
		parseBatchCount = len(parseSources)
	}
	parseBatches := make([][]string, parseBatchCount)
	for parseIndex, parseSource := range parseSources {
		parseBatchIndex := parseIndex % parseBatchCount
		parseBatches[parseBatchIndex] = append(parseBatches[parseBatchIndex], parseSource)
	}
	parseFilteredBatches := make([][]string, 0, len(parseBatches))
	for _, parseBatch := range parseBatches {
		if len(parseBatch) == 0 {
			continue
		}
		parseFilteredBatches = append(parseFilteredBatches, parseBatch)
	}
	return parseFilteredBatches
}

// parseBuildMarkdownBinaryBatchRequest converts one source batch into binary payload leaves for worker transfer.
func parseBuildMarkdownBinaryBatchRequest(parseSources []string) markdownRenderBatchRequest {
	parseSourceBytesList := make([][]byte, 0, len(parseSources))
	for _, parseSource := range parseSources {
		parseSourceBytesList = append(parseSourceBytesList, []byte(parseSource))
	}
	return markdownRenderBatchRequest{GetSourceBytesList: parseSourceBytesList}
}

// parseBuildMarkdownSourceKey projects binary source payload bytes into the cache lookup key used by the app.
func parseBuildMarkdownSourceKey(parseSourceBytes []byte) string {
	return string(parseSourceBytes)
}

// parseResolveBackgroundRenderRequester selects one request-capable worker target and the desired worker-lane count.
func parseResolveBackgroundRenderRequester(parseMarkdownWorkerRef ui.Ref[*interop.Worker], parseMarkdownWorkerPoolRef ui.Ref[*interop.WorkerPool]) (interop.WorkerRequester, int) {
	if parsePool := parseMarkdownWorkerPoolRef.Get(); parsePool != nil {
		parsePoolSize := parsePool.GetSize()
		if parsePoolSize < 1 {
			parsePoolSize = 1
		}
		parseRequester := interop.WorkerRequester(*parsePool)
		return parseRequester, parsePoolSize
	}
	if parseWorker := parseMarkdownWorkerRef.Get(); parseWorker != nil {
		parseRequester := interop.WorkerRequester(*parseWorker)
		return parseRequester, 1
	}
	return nil, 0
}

// parseBuildAssistantMessageMetadataItems projects assistant messages into binary request payload leaves for worker metadata derivation.
func parseBuildAssistantMessageMetadataItems(parseMessages []message) []renderWorkerMessageMetadataMessageRequest {
	parseItems := make([]renderWorkerMessageMetadataMessageRequest, 0, len(parseMessages))
	for parseMessageIndex, parseMessageItem := range parseMessages {
		if parseMessageItem.Role != roleAssistant || parseMessageItem.Pending {
			continue
		}
		parseContentBytes := []byte(parseMessageItem.Content)
		parseThoughtBytes := []byte(parseMessageItem.Thought)
		if len(parseContentBytes) == 0 && len(parseThoughtBytes) == 0 {
			continue
		}
		parseItems = append(parseItems, renderWorkerMessageMetadataMessageRequest{
			GetMessageIndex: parseMessageIndex,
			GetContentBytes: parseContentBytes,
			GetThoughtBytes: parseThoughtBytes,
		})
	}
	return parseItems
}

// parseBuildAssistantMessageMetadataWeights estimates chunking work weights from content and thought payload lengths.
func parseBuildAssistantMessageMetadataWeights(parseItems []renderWorkerMessageMetadataMessageRequest) []int {
	parseWeights := make([]int, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseWeight := len(parseItem.GetContentBytes) + len(parseItem.GetThoughtBytes)
		if parseWeight < 1 {
			parseWeight = 1
		}
		parseWeights = append(parseWeights, parseWeight)
	}
	return parseWeights
}

// parseBuildRenderWorkerRequesterList builds one lane-indexable requester list for batched fan-out APIs.
func parseBuildRenderWorkerRequesterList(parseRequester interop.WorkerRequester, parseCount int) []interop.WorkerRequester {
	if parseRequester == nil {
		return nil
	}
	if parseCount < 1 {
		parseCount = 1
	}
	parseRequesters := make([]interop.WorkerRequester, 0, parseCount)
	for parseIndex := 0; parseIndex < parseCount; parseIndex++ {
		parseRequesters = append(parseRequesters, parseRequester)
	}
	return parseRequesters
}

// parseRequestAssistantMessageMetadata requests worker-derived thought-section and canvas-label metadata and maps chunk results back by message index.
func parseRequestAssistantMessageMetadata(parseCtx context.Context, parseRequester interop.WorkerRequester, parseGeneration uint64, parseItems []renderWorkerMessageMetadataMessageRequest, parseBatchCount int) (map[int]renderWorkerThoughtCacheEntry, map[int]renderWorkerCanvasCacheEntry, error) {
	parseThoughtCacheByMessage := make(map[int]renderWorkerThoughtCacheEntry, len(parseItems))
	parseCanvasCacheByMessage := make(map[int]renderWorkerCanvasCacheEntry, len(parseItems))
	if len(parseItems) == 0 {
		return parseThoughtCacheByMessage, parseCanvasCacheByMessage, nil
	}
	parseWorkerCount := parseBatchCount
	if parseWorkerCount < 1 {
		parseWorkerCount = 1
	}
	parseChunkCount := parseWorkerCount * 2
	if parseChunkCount > len(parseItems) {
		parseChunkCount = len(parseItems)
	}
	parseWeights := parseBuildAssistantMessageMetadataWeights(parseItems)
	parseChunkBounds := renderworker.BuildRenderWorkerAdaptiveChunkBounds(parseWeights, parseChunkCount)
	parseChunkPlans := renderworker.BuildRenderWorkerChunkPlans(parseChunkBounds, parseWeights)
	parseLanePlans := renderworker.BuildRenderWorkerLanePlans(parseWorkerCount, parseChunkPlans)
	parseChunkResults, parseErr := renderworker.RequestRenderWorkerChunkBatches[renderWorkerMessageMetadataBatchRequest, renderWorkerMessageMetadataBatchResult, renderWorkerMessageMetadataChunkResult](
		renderworker.BatchDispatchOptions[renderWorkerMessageMetadataBatchRequest, renderWorkerMessageMetadataBatchResult, renderWorkerMessageMetadataChunkResult]{
			GetCtx:                parseCtx,
			GetRequestName:        backgroundWorkerRequestRenderMessageMetadataBatch,
			GetExpectedGeneration: parseGeneration,
			GetRequesters:         parseBuildRenderWorkerRequesterList(parseRequester, parseWorkerCount),
			GetLanePlans:          parseLanePlans,
			GetChunkCount:         len(parseChunkPlans),
			BuildBatchRequest: func(parseLanePlan renderworker.RenderWorkerLanePlan) renderWorkerMessageMetadataBatchRequest {
				return parseBuildMessageMetadataBatchRequest(parseGeneration, parseLanePlan, parseItems)
			},
			GetBatchChunks: func(parseBatchResult renderWorkerMessageMetadataBatchResult) []renderWorkerMessageMetadataChunkResult {
				return parseBatchResult.GetChunk
			},
			GetChunkIndex: func(parseChunkResult renderWorkerMessageMetadataChunkResult) int {
				return parseChunkResult.GetChunkIndex
			},
			GetBatchGeneration: func(parseBatchResult renderWorkerMessageMetadataBatchResult) uint64 {
				return parseBatchResult.GetGeneration
			},
			GetChunkGeneration: func(parseChunkResult renderWorkerMessageMetadataChunkResult) uint64 {
				return parseChunkResult.GetGeneration
			},
		},
	)
	if parseErr != nil {
		return nil, nil, parseErr
	}
	for _, parseChunkResult := range parseChunkResults {
		for _, parseMessageResult := range parseChunkResult.GetMessage {
			parseThoughtCacheByMessage[parseMessageResult.GetMessageIndex] = renderWorkerThoughtCacheEntry{
				GetThoughtText: string(parseMessageResult.GetThoughtBytes),
				GetSection:     parseBuildThoughtSectionResultsFromWorker(parseMessageResult.GetMessageIndex, parseMessageResult.GetThoughtSection),
			}
			parseCanvasCacheByMessage[parseMessageResult.GetMessageIndex] = renderWorkerCanvasCacheEntry{
				GetContentText: string(parseMessageResult.GetContentBytes),
				GetArtifact:    parseBuildCanvasArtifactResultsFromWorker(parseMessageResult.GetMessageIndex, parseMessageResult.GetCanvasArtifact),
			}
		}
	}
	return parseThoughtCacheByMessage, parseCanvasCacheByMessage, nil
}

// parseBuildMessageMetadataBatchRequest builds one lane-scoped chunk request payload from chunk-plan assignments.
func parseBuildMessageMetadataBatchRequest(parseGeneration uint64, parseLanePlan renderworker.RenderWorkerLanePlan, parseItems []renderWorkerMessageMetadataMessageRequest) renderWorkerMessageMetadataBatchRequest {
	parseChunkRequests := make([]renderWorkerMessageMetadataChunkRequest, 0, len(parseLanePlan.GetChunkPlans))
	for _, parseChunkPlan := range parseLanePlan.GetChunkPlans {
		parseChunkMessageItems := parseItems[parseChunkPlan.GetStart:parseChunkPlan.GetEnd]
		parseChunkRequests = append(parseChunkRequests, renderWorkerMessageMetadataChunkRequest{
			GetChunkIndex:   parseChunkPlan.GetChunkIndex,
			GetMessageItems: parseChunkMessageItems,
		})
	}
	return renderWorkerMessageMetadataBatchRequest{
		GetGeneration:   parseGeneration,
		GetChunkRequest: parseChunkRequests,
	}
}

// parseBuildThoughtSectionResultsFromWorker converts worker thought-section payloads into keyed UI thought-section records.
func parseBuildThoughtSectionResultsFromWorker(parseMessageIndex int, parseSectionResults []renderWorkerThoughtSectionResult) []thoughtSection {
	if len(parseSectionResults) == 0 {
		return nil
	}
	parseSections := make([]thoughtSection, 0, len(parseSectionResults))
	for parseSectionIndex, parseSectionResult := range parseSectionResults {
		parseHeading := string(parseSectionResult.GetHeadingBytes)
		parseSections = append(parseSections, thoughtSection{
			Key:     parseThoughtSectionKey(parseMessageIndex, parseSectionIndex, parseHeading),
			Heading: parseHeading,
			Body:    string(parseSectionResult.GetBodyBytes),
		})
	}
	return parseSections
}

// parseBuildCanvasArtifactResultsFromWorker converts worker canvas-artifact payloads into lightweight UI metadata records.
func parseBuildCanvasArtifactResultsFromWorker(parseMessageIndex int, parseArtifactResults []renderWorkerCanvasArtifactResult) []canvasArtifact {
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

// parseRequestWorkerThreadCostSummary requests one off-thread assistant cost summary from the render worker.
func parseRequestWorkerThreadCostSummary(parseCtx context.Context, parseRequester interop.WorkerRequester, parseGeneration uint64, parseMessages []message, parseModels []modelOption) (threadCostSummary, error) {
	parseResult, parseErr := interop.RequestWorkerDecoded[renderWorkerThreadCostSummaryRequest, struct{}, renderWorkerThreadCostSummaryResult](
		parseCtx,
		parseRequester,
		backgroundWorkerRequestRenderThreadCostSummary,
		parseBuildWorkerThreadCostSummaryRequest(parseGeneration, parseMessages, parseModels),
		nil,
	)
	if parseErr != nil {
		return threadCostSummary{}, parseErr
	}
	return parseBuildWorkerThreadCostSummaryFromResult(parseResult), nil
}

// parseRequestWorkerRenderSignatures requests one off-thread render-signature bundle used by markdown, metadata, and thread-cost effects.
func parseRequestWorkerRenderSignatures(parseCtx context.Context, parseRequester interop.WorkerRequester, parseGeneration uint64, parseMessages []message, parseModels []modelOption) (renderWorkerSignatureState, error) {
	parseResult, parseErr := interop.RequestWorkerDecoded[renderWorkerSignatureRequest, struct{}, renderWorkerSignatureResult](
		parseCtx,
		parseRequester,
		backgroundWorkerRequestRenderSignatures,
		parseBuildWorkerRenderSignatureRequest(parseGeneration, parseMessages, parseModels),
		nil,
	)
	if parseErr != nil {
		return renderWorkerSignatureState{}, parseErr
	}
	return parseBuildWorkerSignatureStateFromResult(parseResult), nil
}

// parseBuildWorkerThreadCostSummaryRequest projects message/model state into one binary-heavy worker request payload.
func parseBuildWorkerThreadCostSummaryRequest(parseGeneration uint64, parseMessages []message, parseModels []modelOption) renderWorkerThreadCostSummaryRequest {
	parseWorkerMessages := make([]renderWorkerCostMessageRequest, 0, len(parseMessages))
	for parseMessageIndex, parseMessageItem := range parseMessages {
		parseWorkerMessages = append(parseWorkerMessages, renderWorkerCostMessageRequest{
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
	return renderWorkerThreadCostSummaryRequest{
		GetGeneration: parseGeneration,
		GetMessage:    parseWorkerMessages,
		GetModel:      parseWorkerModels,
	}
}

// parseBuildWorkerRenderSignatureRequest projects message/model state into one worker payload used for signature derivation.
func parseBuildWorkerRenderSignatureRequest(parseGeneration uint64, parseMessages []message, parseModels []modelOption) renderWorkerSignatureRequest {
	parseWorkerMessages := make([]renderWorkerSignatureMessageRequest, 0, len(parseMessages))
	for parseMessageIndex, parseMessageItem := range parseMessages {
		parseWorkerMessages = append(parseWorkerMessages, renderWorkerSignatureMessageRequest{
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
	return renderWorkerSignatureRequest{
		GetGeneration: parseGeneration,
		GetMessage:    parseWorkerMessages,
		GetModel:      parseWorkerModels,
	}
}

// parseBuildWorkerThreadCostSummaryFromResult converts one worker summary payload into the app's summary domain shape.
func parseBuildWorkerThreadCostSummaryFromResult(parseResult renderWorkerThreadCostSummaryResult) threadCostSummary {
	parseSummary := threadCostSummary{
		TotalCost:              parseResult.GetTotalCost,
		AssistantMessageCosts:  make(map[int]assistantMessageCost, len(parseResult.GetAssistantMessageCost)),
		HasAnyExactCosts:       parseResult.GetHasAnyExactCosts,
		AllAssistantCostsExact: parseResult.GetAllAssistantCostsExact,
	}
	for _, parseMessageCost := range parseResult.GetAssistantMessageCost {
		if !parseMessageCost.GetHasExactCost {
			continue
		}
		parseSummary.AssistantMessageCosts[parseMessageCost.GetMessageIndex] = assistantMessageCost{
			ModelID:          string(parseMessageCost.GetModelIDBytes),
			PromptTokens:     parseMessageCost.GetPromptTokens,
			CompletionTokens: parseMessageCost.GetCompletionTokens,
			Cost:             parseMessageCost.GetCost,
		}
	}
	return parseSummary
}

// parseBuildWorkerSignatureStateFromResult converts one worker signature result payload into UI-facing signature state.
func parseBuildWorkerSignatureStateFromResult(parseResult renderWorkerSignatureResult) renderWorkerSignatureState {
	return renderWorkerSignatureState{
		GetCompletedMarkdownSignature: string(parseResult.GetCompletedMarkdownSignatureBytes),
		GetAssistantMetadataSignature: string(parseResult.GetAssistantMetadataSignatureBytes),
		GetThreadCostSignature:        string(parseResult.GetThreadCostSignatureBytes),
	}
}

func parseMonitorGRPCConnection(
	parseCtx context.Context,
	parseApp ui.Reducer[appState, appAction],
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	parseMarkdownWorkerRef ui.Ref[*interop.Worker],
	parseConn *grpc.ClientConn,
	parseWakeCh <-chan string,
	parseSleepCh <-chan string,
	parseReconnectCh <-chan string,
) grpcMonitorResult {
	parseTicker := time.NewTicker(grpcStatePoll)
	defer parseTicker.Stop()

	parseLastState := connectivity.Idle
	parseNotReadySince := time.Now()
	applyState := func(parseState2 connectivity.State, parseReason3 string) {
		if parseState2 == parseLastState {
			return
		}
		parseLastState = parseState2
		chatLog.Info("grpc state", logging.Fields{"state": parseState2.String(), "reason": parseReason3})
	}

	for {
		parseState := parseConn.GetState()
		applyState(parseState, "poll")
		switch parseState {
		case connectivity.Ready:
			parseNotReadySince = time.Time{}
			parseSyncGRPCReadyState(parseApp, parseChatClientRef, parseMarkdownWorkerRef, parseConn, true, "bridge ready", nil, nil)
		case connectivity.Idle:
			parseNotReadySince = time.Time{}
			parseSyncGRPCReadyState(parseApp, parseChatClientRef, parseMarkdownWorkerRef, parseConn, true, "bridge idle", nil, nil)
		case connectivity.Connecting:
			parseSyncGRPCReadyState(parseApp, parseChatClientRef, parseMarkdownWorkerRef, parseConn, true, "bridge reconnecting", nil, nil)
			if parseNotReadySince.IsZero() {
				parseNotReadySince = time.Now()
			}
		case connectivity.TransientFailure, connectivity.Shutdown:
			return grpcMonitorResult{Reason: "bridge " + parseState.String()}
		default:
			parseSyncGRPCReadyState(parseApp, parseChatClientRef, parseMarkdownWorkerRef, parseConn, true, "bridge unavailable", nil, nil)
			if parseNotReadySince.IsZero() {
				parseNotReadySince = time.Now()
			}
		}
		if !parseNotReadySince.IsZero() && time.Since(parseNotReadySince) >= grpcStallLimit {
			return grpcMonitorResult{Reason: "bridge stalled in " + parseState.String()}
		}

		select {
		case <-parseCtx.Done():
			return grpcMonitorResult{Reason: "runtime canceled"}
		case parseReason := <-parseSleepCh:
			return grpcMonitorResult{Reason: parseReason, Sleep: true}
		case parseReason2 := <-parseReconnectCh:
			return grpcMonitorResult{Reason: parseReason2, Immediate: true}
		case <-parseWakeCh:
			if parseRuntimeNavigatorOnline() {
				parseConn.Connect()
			}
		case <-parseTicker.C:
		}
	}
}

func parseSyncGRPCReadyState(
	parseApp ui.Reducer[appState, appAction],
	parseChatClientRef ui.Ref[chatpb.ChatServiceClient],
	parseMarkdownWorkerRef ui.Ref[*interop.Worker],
	parseConn *grpc.ClientConn,
	isReady bool,
	parseReason string,
	parseOnConversationRefresh func(bool),
	parseOnProfileRefresh func(bool),
) {
	if isReady && parseConn != nil {
		if parseApp.Get().GRPCReady && parseChatClientRef.Get() != nil {
			return
		}
		parseChatClientRef.Set(chatpb.NewChatServiceClient(parseConn))
		parseSetClientLogRelay(parseChatClientRef.Get(), parseLoadPersistedClientIdentity(), true)
		parseEnsureClientIdentity(parseChatClientRef.Get())
		parseApp.Dispatch(appAction{Type: appActionSetGRPCReady, GRPCReady: true})
		parsePostBackgroundWorkerTicker(parseMarkdownWorkerRef, backgroundWorkerCommandStartTicker)
		if parseOnConversationRefresh != nil {
			// Respect the feature TTLs on reconnect so a flapping bridge does not
			// repeatedly force list/profile RPCs while still allowing the initial
			// post-connect load to run when nothing has been fetched yet.
			go parseOnConversationRefresh(false)
		}
		if parseOnProfileRefresh != nil {
			go parseOnProfileRefresh(false)
		}
		chatLog.Info("grpc ready", logging.Fields{"endpoint": grpcEndpoint, "reason": parseReason})
		return
	}
	if !parseApp.Get().GRPCReady && parseChatClientRef.Get() == nil {
		return
	}
	parseChatClientRef.Set(nil)
	parseSetClientLogRelay(nil, "", false)
	parseApp.Dispatch(appAction{Type: appActionSetGRPCReady, GRPCReady: false})
	parsePostBackgroundWorkerTicker(parseMarkdownWorkerRef, backgroundWorkerCommandStopTicker)
	chatLog.Warn("grpc unavailable", logging.Fields{"reason": parseReason})
}

func parsePostBackgroundWorkerTicker(parseMarkdownWorkerRef ui.Ref[*interop.Worker], parseCommand string) {
	parseWorker := parseMarkdownWorkerRef.Get()
	if parseWorker == nil {
		return
	}
	parsePayload := map[string]any{}
	if parseCommand == backgroundWorkerCommandStartTicker {
		parsePayload["intervalMs"] = int64(bgRefreshInterval / time.Millisecond)
	}
	if parseErr := parseWorker.Post(interop.WorkerMessage{Phase: "message", Name: parseCommand, Payload: parsePayload}); parseErr != nil {
		chatLog.Warn("background worker ticker command failed", logging.Fields{"error": parseErr, "command": parseCommand})
	}
}

func parseWaitForWakeSignal(parseCtx context.Context, parseWakeCh <-chan string) (string, bool) {
	for {
		select {
		case <-parseCtx.Done():
			return "", false
		case parseReason := <-parseWakeCh:
			if parseRuntimeNavigatorOnline() {
				return parseReason, true
			}
		}
	}
}

func parseWaitForReconnectDelay(parseCtx context.Context, parseDelay time.Duration, parseWakeCh <-chan string, parseSleepCh <-chan string, parseReconnectCh <-chan string) string {
	parseTimer := time.NewTimer(parseDelay)
	defer parseTimer.Stop()
	select {
	case <-parseCtx.Done():
		return "done"
	case <-parseTimer.C:
		return "timer"
	case <-parseSleepCh:
		return "sleep"
	case <-parseWakeCh:
		return "wake"
	case <-parseReconnectCh:
		return "reconnect"
	}
}

func parseGrpcReconnectDelay(parseAttempt int) time.Duration {
	parseDelay := grpcBackoffBase
	for parseI := 0; parseI < parseAttempt; parseI++ {
		parseDelay *= 2
		if parseDelay >= grpcBackoffMax {
			return grpcBackoffMax
		}
	}
	if parseDelay > grpcBackoffMax {
		return grpcBackoffMax
	}
	return parseDelay
}

func parseRuntimeNavigatorOnline() bool {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() {
		return true
	}
	parseNavigator := parseWindow.Get("navigator")
	if !parseNavigator.Truthy() {
		return true
	}
	parseOnLine := parseNavigator.Get("onLine")
	if parseOnLine.Type() == js.TypeBoolean {
		return parseOnLine.Bool()
	}
	return true
}

func parseRuntimePageVisible() bool {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return true
	}
	return parseDocument.Get("visibilityState").String() != "hidden"
}

func parseAttachWindowListener(parseWindow js.Value, parseEventName string, parseHandler func()) func() {
	if !parseWindow.Truthy() || parseWindow.Get("addEventListener").Type() != js.TypeFunction {
		return func() {}
	}
	parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseHandler()
		return nil
	})
	parseWindow.Call("addEventListener", parseEventName, parseListener)
	return func() {
		parseWindow.Call("removeEventListener", parseEventName, parseListener)
		parseListener.Release()
	}
}

func parseAttachDocumentListener(parseDocument js.Value, parseEventName string, parseHandler func()) func() {
	if !parseDocument.Truthy() || parseDocument.Get("addEventListener").Type() != js.TypeFunction {
		return func() {}
	}
	parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseHandler()
		return nil
	})
	parseDocument.Call("addEventListener", parseEventName, parseListener)
	return func() {
		parseDocument.Call("removeEventListener", parseEventName, parseListener)
		parseListener.Release()
	}
}

func parseAuthContextWithMetadata(parseCtx context.Context) context.Context {
	parseMetadataPairs := make([]string, 0, 14)
	parseToken := parseLoadPersistedAuthToken()
	if parseToken != "" {
		parseMetadataPairs = append(parseMetadataPairs, authMetadataKey, "Bearer "+parseToken)
	}
	parseClientID := parseLoadPersistedClientIdentity()
	if parseClientID != "" {
		parseMetadataPairs = append(parseMetadataPairs, clientMetadataKey, parseClientID)
	}
	parseCorrelationID := parseEnsurePersistedCorrelationIdentity()
	if parseCorrelationID != "" {
		parseMetadataPairs = append(parseMetadataPairs, correlationIDMetadataKey, parseCorrelationID)
	}
	parseRequestID := parseBuildRequestMetadataID()
	if parseRequestID != "" {
		parseMetadataPairs = append(parseMetadataPairs, requestIDMetadataKey, parseRequestID)
	}
	parseTraceParent := parseBuildTraceParentMetadata()
	if parseTraceParent != "" {
		parseMetadataPairs = append(parseMetadataPairs, traceParentMetadataKey, parseTraceParent)
	}
	parseTraceState := parseBuildTraceStateMetadata(parseCorrelationID)
	if parseTraceState != "" {
		parseMetadataPairs = append(parseMetadataPairs, traceStateMetadataKey, parseTraceState)
	}
	if len(parseMetadataPairs) == 0 {
		return parseCtx
	}
	return metadata.AppendToOutgoingContext(parseCtx, parseMetadataPairs...)
}

// parseBuildTraceParentMetadata creates one W3C traceparent value for bridge RPC correlation.
func parseBuildTraceParentMetadata() string {
	var parseTraceID [16]byte
	var parseSpanID [8]byte
	if _, parseErr := rand.Read(parseTraceID[:]); parseErr != nil {
		return ""
	}
	if _, parseErr2 := rand.Read(parseSpanID[:]); parseErr2 != nil {
		return ""
	}
	return "00-" + hex.EncodeToString(parseTraceID[:]) + "-" + hex.EncodeToString(parseSpanID[:]) + "-01"
}

// parseBuildRequestMetadataID creates one per-RPC request identifier for bridge/backend correlation.
func parseBuildRequestMetadataID() string {
	return parseBuildOpaqueMetadataID()
}

// parseBuildTraceStateMetadata creates one tracestate value that carries stable client correlation identity.
func parseBuildTraceStateMetadata(parseCorrelationID string) string {
	parseCorrelationID = strings.TrimSpace(parseCorrelationID)
	if parseCorrelationID == "" {
		return ""
	}
	return "gwc-correlation-id=" + parseCorrelationID
}

// parseBuildOpaqueMetadataID creates one opaque hexadecimal identifier used for metadata correlation fields.
func parseBuildOpaqueMetadataID() string {
	var parseIDBytes [16]byte
	if _, parseErr := rand.Read(parseIDBytes[:]); parseErr != nil {
		return ""
	}
	return hex.EncodeToString(parseIDBytes[:])
}
