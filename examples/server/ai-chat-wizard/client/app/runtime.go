//go:build js && wasm

package app

import (
	"context"
	"strings"
	"sync"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/v5/examples/shared/renderworker"
	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/monstercameron/GoWebComponents/v5/logging"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"google.golang.org/grpc"
)

type grpcMonitorResult struct {
	Reason    string
	Sleep     bool
	Immediate bool
}

type backgroundWorkerMaintenanceBatchEvent struct {
	Tasks []struct {
		TaskType string `json:"taskType"`
		ScopeKey string `json:"scopeKey"`
		QueueKey string `json:"queueKey"`
		Reason   string `json:"reason"`
	} `json:"tasks"`
	EmittedAt string `json:"emittedAt"`
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
	parseMetadataSignatureRef := ui.UseRef("")
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
					return
				}
				if parseMsg2.Name == backgroundWorkerEventMaintenanceBatch {
					parseHandleBackgroundWorkerMaintenanceBatch(parseMsg2)
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
		if !parseShouldRefreshWorkerDerivedSignature(parseAssistantMetadataSignature, parseMetadataSignatureRef.Get()) {
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
			parseMetadataSignatureRef.Set(parseAssistantMetadataSignature)
			return nil
		}
		parseMessageItems, parseRetainedThoughtCacheByMessage, parseRetainedCanvasCacheByMessage := parseBuildAssistantMessageMetadataDelta(
			parseApp.Get().Messages,
			parseThoughtCacheByMessageState.Get(),
			parseCanvasCacheByMessageState.Get(),
		)
		parseGeneration := parseMetadataGenerationRef.Get() + 1
		parseMetadataGenerationRef.Set(parseGeneration)
		if len(parseMessageItems) == 0 {
			parseThoughtCacheByMessageState.Set(parseRetainedThoughtCacheByMessage)
			parseCanvasCacheByMessageState.Set(parseRetainedCanvasCacheByMessage)
			parseMetadataSignatureRef.Set(parseAssistantMetadataSignature)
			return nil
		}
		// Metadata delta items are rebuilt for each effect run, so the async worker
		// request can consume this snapshot directly without another copy.
		go func(parseExpectedGeneration uint64, parseExpectedSignature string, parseItems []renderWorkerMessageMetadataMessageRequest, parsePoolBatchCount int, parseRetainedThoughtCache map[int]renderWorkerThoughtCacheEntry, parseRetainedCanvasCache map[int]renderWorkerCanvasCacheEntry) {
			parseThoughtCacheByMessage, parseCanvasCacheByMessage, parseErr := parseRequestAssistantMessageMetadata(context.Background(), parseRequester, parseExpectedGeneration, parseItems, parsePoolBatchCount)
			if parseErr != nil {
				chatLog.Warn("message metadata worker request failed; using synchronous fallback", logging.Fields{"error": parseErr, "messages": len(parseItems)})
				parseThoughtCacheByMessage, parseCanvasCacheByMessage = parseBuildAssistantMessageMetadataFallback(parseItems)
				if parseMetadataGenerationRef.Get() == parseExpectedGeneration {
					parseThoughtMergedCacheByMessage, parseCanvasMergedCacheByMessage := parseMergeAssistantMessageMetadataCaches(parseRetainedThoughtCache, parseRetainedCanvasCache, parseThoughtCacheByMessage, parseCanvasCacheByMessage)
					parseThoughtCacheByMessageState.Set(parseThoughtMergedCacheByMessage)
					parseCanvasCacheByMessageState.Set(parseCanvasMergedCacheByMessage)
					parseMetadataSignatureRef.Set(parseExpectedSignature)
				}
				return
			}
			if parseMetadataGenerationRef.Get() != parseExpectedGeneration {
				return
			}
			parseThoughtMergedCacheByMessage, parseCanvasMergedCacheByMessage := parseMergeAssistantMessageMetadataCaches(parseRetainedThoughtCache, parseRetainedCanvasCache, parseThoughtCacheByMessage, parseCanvasCacheByMessage)
			parseThoughtCacheByMessageState.Set(parseThoughtMergedCacheByMessage)
			parseCanvasCacheByMessageState.Set(parseCanvasMergedCacheByMessage)
			parseMetadataSignatureRef.Set(parseExpectedSignature)
		}(parseGeneration, parseAssistantMetadataSignature, parseMessageItems, parseBatchCount, parseRetainedThoughtCacheByMessage, parseRetainedCanvasCacheByMessage)
		return nil
	}, parseApp.Get().MarkdownWorkerFallback, parseAssistantMetadataSignature)

	ui.UseEffect(func() func() {
		if parseApp.Get().MarkdownWorkerFallback {
			parseThreadCostSummarySignatureState.Set("")
			return nil
		}
		if !parseShouldRefreshWorkerDerivedSignature(parseThreadCostSignature, parseThreadCostSummarySignatureState.Get()) {
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
		// App state updates replace message/model slices, so this render snapshot is
		// safe to hand to the async worker request without another clone.
		parseMessages := parseApp.Get().Messages
		parseModels := parseApp.Get().ModelOptions
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
	}, parseApp.Get().MarkdownWorkerFallback, parseThreadCostSignature)
}
