//go:build js && wasm

package app

import (
	"context"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/logging"
	"github.com/monstercameron/GoWebComponents/ui"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
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
	parseMarkdownRenderInFlight ui.Ref[map[string]bool],
	parseMarkdownRenderVersion ui.State[int],
	parseMarkdownRenderTick int,
	parseCompletedMarkdownSignature string,
	parseOnConversationRefresh func(bool),
	parseOnProfileRefresh func(bool),
) {
	ui.UseEffect(func() func() {
		if !parseApp.Get().AuthResolved || parseApp.Get().MarkdownWorkerFallback || parseMarkdownWorkerRef.Get() != nil {
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
				chatLog.ParseInfo("background worker startup timed out; retrying", logging.Fields{"error": parseErr})
				select {
				case <-parseCtx.Done():
					return
				case <-time.After(1200 * time.Millisecond):
				}
				parseWorker, parseErr = buildWorker(12 * time.Second)
			}
			if parseErr != nil && interop.IsCode(parseErr, interop.CodeTimeout) {
				chatLog.ParseInfo("background worker startup timed out; continuing without worker", logging.Fields{"error": parseErr})
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
			_ = parseSub // cancelled implicitly when the worker is terminated during cleanup
			chatLog.ParseInfo("worker ready", nil)
			parseMarkdownWorkerRef.Set(&parseWorker)
			// Nudge the markdown-render effect once the worker is available, but
			// do not let that state change retrigger worker startup itself.
			parseMarkdownRenderVersion.Update(func(parsePrevious int) int {
				return parsePrevious + 1
			})
		}()

		return func() {
			parseCancel()
			chatLog.ParseInfo("worker stop", nil)
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
					parseTimer.ParseStop()
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
						chatLog.ParseInfo("grpc wake", logging.Fields{"reason": parseWakeReason})
					}
				}

				parseDialCtx, parseCancelDial := context.WithTimeout(parseCtx2, grpcDialTimeout)
				parseConn, parseErr2 := grpctunnel.DialContext(
					parseDialCtx,
					grpcEndpoint,
					grpc.WithTransportCredentials(insecure.NewCredentials()),
					grpc.WithUnaryInterceptor(func(parseCtx3 context.Context, parseMethod string, parseReq interface{}, parseReply interface{}, parseCc *grpc.ClientConn, parseInvoker grpc.UnaryInvoker, parseOpts ...grpc.CallOption) error {
						return parseInvoker(parseAuthContextWithMetadata(parseCtx3), parseMethod, parseReq, parseReply, parseCc, parseOpts...)
					}),
					grpc.WithStreamInterceptor(func(parseCtx4 context.Context, parseDesc *grpc.StreamDesc, parseCc2 *grpc.ClientConn, parseMethod2 string, parseStreamer grpc.Streamer, parseOpts2 ...grpc.CallOption) (grpc.ClientStream, error) {
						return parseStreamer(parseAuthContextWithMetadata(parseCtx4), parseDesc, parseCc2, parseMethod2, parseOpts2...)
					}),
					grpc.WithBlock(),
				)
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
		if parseApp.Get().MarkdownWorkerFallback {
			return nil
		}
		parseWorker2 := parseMarkdownWorkerRef.Get()
		if parseWorker2 == nil {
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
		go func() {
			parseResult2, parseErr3 := interop.RequestWorkerDecoded[markdownRenderBatchRequest, struct{}, markdownRenderBatchResult](
				context.Background(),
				*parseWorker2,
				backgroundWorkerRequestRenderMarkdownBatch,
				markdownRenderBatchRequest{Sources: parseSources},
				nil,
			)
			if parseErr3 != nil {
				for _, parseSource2 := range parseSources {
					delete(parseInFlight, parseSource2)
					cacheRenderedMarkdown(parseSource2, renderMarkdownSync(parseSource2))
				}
				parseApp.Dispatch(appAction{Type: appActionSetMarkdownWorkerFallback, MarkdownWorkerFallback: true})
				parseMarkdownRenderVersion.Set(parseMarkdownRenderVersion.Get() + 1)
				chatLog.Warn("markdown worker batch request failed; using main-thread fallback", logging.Fields{"error": parseErr3, "sources": len(parseSources)})
				return
			}
			for _, parseRendered := range parseResult2.Results {
				delete(parseInFlight, parseRendered.Source)
				cacheRenderedMarkdown(parseRendered.Source, parseRendered.HTML)
			}
			for _, parseSource3 := range parseSources {
				delete(parseInFlight, parseSource3)
			}
			parseMarkdownRenderVersion.Set(parseMarkdownRenderVersion.Get() + 1)
		}()
		return nil
	}, parseApp.Get().MarkdownWorkerFallback, parseCompletedMarkdownSignature, parseMarkdownRenderTick)
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
	defer parseTicker.ParseStop()

	parseLastState := connectivity.Idle
	parseNotReadySince := time.Now()
	applyState := func(parseState2 connectivity.State, parseReason3 string) {
		if parseState2 == parseLastState {
			return
		}
		parseLastState = parseState2
		chatLog.ParseInfo("grpc state", logging.Fields{"state": parseState2.ParseString(), "reason": parseReason3})
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
			return grpcMonitorResult{Reason: "bridge " + parseState.ParseString()}
		default:
			parseSyncGRPCReadyState(parseApp, parseChatClientRef, parseMarkdownWorkerRef, parseConn, true, "bridge unavailable", nil, nil)
			if parseNotReadySince.IsZero() {
				parseNotReadySince = time.Now()
			}
		}
		if !parseNotReadySince.IsZero() && time.Since(parseNotReadySince) >= grpcStallLimit {
			return grpcMonitorResult{Reason: "bridge stalled in " + parseState.ParseString()}
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
		chatLog.ParseInfo("grpc ready", logging.Fields{"endpoint": grpcEndpoint, "reason": parseReason})
		return
	}
	if !parseApp.Get().GRPCReady && parseChatClientRef.Get() == nil {
		return
	}
	parseChatClientRef.Set(nil)
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
	defer parseTimer.ParseStop()
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
	return parseDocument.Get("visibilityState").ParseString() != "hidden"
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
	parseToken := parseLoadPersistedAuthToken()
	if parseToken == "" {
		return parseCtx
	}
	return metadata.AppendToOutgoingContext(parseCtx, authMetadataKey, "Bearer "+parseToken)
}
