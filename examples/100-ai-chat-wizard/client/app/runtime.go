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
func useAppRuntime(
	app ui.Reducer[appState, appAction],
	chatClientRef ui.Ref[chatpb.ChatServiceClient],
	markdownWorkerRef ui.Ref[*interop.Worker],
	markdownRenderInFlight ui.Ref[map[string]bool],
	markdownRenderVersion ui.State[int],
	completedMarkdownSignature string,
	onConversationRefresh func(bool),
	onProfileRefresh func(bool),
) {
	ui.UseEffect(func() func() {
		if !app.Get().AuthResolved || app.Get().MarkdownWorkerFallback || markdownWorkerRef.Get() != nil {
			return nil
		}
		ctx, cancel := context.WithCancel(context.Background())
		buildWorker := func(readyTimeout time.Duration) (interop.Worker, error) {
			return interop.NewGoWASMWorker(ctx, interop.GoWASMWorkerOptions{
				RuntimeURL:   backgroundWorkerRuntimeURL,
				WASMURL:      backgroundWorkerWASMURL + currentWASMQuerySuffix(),
				Name:         "chat-background",
				Ready:        true,
				ReadyTimeout: readyTimeout,
			})
		}

		worker, err := buildWorker(5 * time.Second)
		if err != nil && interop.IsCode(err, interop.CodeTimeout) {
			chatLog.Info("background worker startup timed out; retrying", logging.Fields{"error": err})
			select {
			case <-ctx.Done():
				return cancel
			case <-time.After(1200 * time.Millisecond):
			}
			worker, err = buildWorker(12 * time.Second)
		}
		if err != nil {
			app.Dispatch(appAction{Type: appActionSetMarkdownWorkerFallback, MarkdownWorkerFallback: true})
			chatLog.Warn("background worker unavailable; falling back to main-thread work", logging.Fields{"error": err})
			return cancel
		}
		sub, subErr := worker.Subscribe(func(msg interop.WorkerMessage, err error) {
			if err != nil {
				chatLog.Warn("background worker subscription error", logging.Fields{"error": err})
				return
			}
			if msg.Name == backgroundWorkerEventTick && onConversationRefresh != nil {
				onConversationRefresh(false)
			}
		})
		if subErr != nil {
			app.Dispatch(appAction{Type: appActionSetMarkdownWorkerFallback, MarkdownWorkerFallback: true})
			chatLog.Warn("background worker subscribe failed; falling back to main-thread work", logging.Fields{"error": subErr})
			_ = worker.Terminate()
			return cancel
		}
		chatLog.Info("worker ready", nil)
		markdownWorkerRef.Set(&worker)
		return func() {
			cancel()
			chatLog.Info("worker stop", nil)
			markdownWorkerRef.Set(nil)
			sub.Cancel()
			_ = worker.Terminate()
		}
	}, app.Get().AuthResolved, app.Get().MarkdownWorkerFallback)

	ui.UseEffect(func() func() {
		ctx, cancel := context.WithCancel(context.Background())
		wakeCh := make(chan string, 1)
		sleepCh := make(chan string, 1)
		reconnectCh := make(chan string, 1)

		signal := func(ch chan string, reason string) {
			reason = strings.TrimSpace(reason)
			select {
			case ch <- reason:
			default:
			}
		}
		disconnect := func(conn *grpc.ClientConn, reason string) {
			syncGRPCReadyState(app, chatClientRef, markdownWorkerRef, nil, false, reason, onConversationRefresh, onProfileRefresh)
			if conn != nil {
				_ = conn.Close()
			}
		}

		unregisterReconnect := registerGRPCReconnectHandler(func(reason string) {
			signal(reconnectCh, reason)
		})

		window := js.Global().Get("window")
		document := js.Global().Get("document")
		stopHiddenSleep := func() {}
		var startHiddenSleep func(time.Duration)
		startHiddenSleep = func(delay time.Duration) {
			stopHiddenSleep()
			if runtimePageVisible() {
				return
			}
			timer := time.AfterFunc(delay, func() {
				if runtimePageVisible() {
					return
				}
				if app.Get().Streaming {
					startHiddenSleep(grpcSleepRetry)
					return
				}
				signal(sleepCh, "hidden idle")
			})
			stopHiddenSleep = func() {
				if timer != nil {
					timer.Stop()
				}
			}
		}

		cleanupOnline := attachWindowListener(window, "online", func() {
			stopHiddenSleep()
			signal(wakeCh, "browser online")
		})
		cleanupOffline := attachWindowListener(window, "offline", func() {
			stopHiddenSleep()
			signal(sleepCh, "browser offline")
		})
		cleanupFocus := attachWindowListener(window, "focus", func() {
			stopHiddenSleep()
			signal(wakeCh, "window focus")
		})
		cleanupVisibility := attachDocumentListener(document, "visibilitychange", func() {
			if runtimePageVisible() {
				stopHiddenSleep()
				signal(wakeCh, "document visible")
				return
			}
			startHiddenSleep(grpcSleepAfter)
		})
		if !runtimePageVisible() {
			startHiddenSleep(grpcSleepAfter)
		}

		go func() {
			defer syncGRPCReadyState(app, chatClientRef, markdownWorkerRef, nil, false, "runtime stopped", onConversationRefresh, onProfileRefresh)

			sleeping := false
			attempt := 0
			for {
				if ctx.Err() != nil {
					return
				}
				if sleeping || !runtimeNavigatorOnline() {
					syncGRPCReadyState(app, chatClientRef, markdownWorkerRef, nil, false, "bridge sleeping", onConversationRefresh, onProfileRefresh)
					wakeReason, ok := waitForWakeSignal(ctx, wakeCh)
					if !ok {
						return
					}
					sleeping = false
					attempt = 0
					if wakeReason != "" {
						chatLog.Info("grpc wake", logging.Fields{"reason": wakeReason})
					}
				}

				dialCtx, cancelDial := context.WithTimeout(ctx, grpcDialTimeout)
				conn, err := grpctunnel.DialContext(
					dialCtx,
					grpcEndpoint,
					grpc.WithTransportCredentials(insecure.NewCredentials()),
					grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
						return invoker(authContextWithMetadata(ctx), method, req, reply, cc, opts...)
					}),
					grpc.WithStreamInterceptor(func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
						return streamer(authContextWithMetadata(ctx), desc, cc, method, opts...)
					}),
					grpc.WithBlock(),
				)
				cancelDial()
				if err != nil {
					delay := grpcReconnectDelay(attempt)
					attempt++
					syncGRPCReadyState(app, chatClientRef, markdownWorkerRef, nil, false, "dial failed", onConversationRefresh, onProfileRefresh)
					chatLog.Warn("grpc dial failed", logging.Fields{"error": err, "retry_in_ms": int(delay / time.Millisecond)})
					switch waitForReconnectDelay(ctx, delay, wakeCh, sleepCh, reconnectCh) {
					case "sleep":
						sleeping = true
					case "wake", "reconnect":
						attempt = 0
					case "done":
						return
					}
					continue
				}

				attempt = 0
				syncGRPCReadyState(app, chatClientRef, markdownWorkerRef, conn, true, "bridge connected", onConversationRefresh, onProfileRefresh)
				result := monitorGRPCConnection(ctx, app, chatClientRef, markdownWorkerRef, conn, wakeCh, sleepCh, reconnectCh)
				disconnect(conn, result.Reason)
				if ctx.Err() != nil {
					return
				}
				if result.Sleep {
					sleeping = true
					continue
				}
				if result.Immediate {
					continue
				}
				delay := grpcReconnectDelay(attempt)
				attempt++
				switch waitForReconnectDelay(ctx, delay, wakeCh, sleepCh, reconnectCh) {
				case "sleep":
					sleeping = true
				case "wake", "reconnect":
					attempt = 0
				case "done":
					return
				}
			}
		}()

		return func() {
			cancel()
			stopHiddenSleep()
			cleanupVisibility()
			cleanupFocus()
			cleanupOffline()
			cleanupOnline()
			unregisterReconnect()
		}
	}, true)

	ui.UseEffect(func() func() {
		if app.Get().MarkdownWorkerFallback {
			return nil
		}
		worker := markdownWorkerRef.Get()
		if worker == nil {
			return nil
		}
		inFlight := markdownRenderInFlight.Get()
		sources := make([]string, 0, len(app.Get().Messages))
		for _, msg := range app.Get().Messages {
			if msg.Role != roleAssistant || msg.Pending {
				continue
			}
			source := strings.TrimSpace(msg.Content)
			if source == "" {
				continue
			}
			if _, ok := cachedRenderedMarkdown(source); ok {
				continue
			}
			if inFlight[source] {
				continue
			}
			inFlight[source] = true
			sources = append(sources, source)
		}
		if len(sources) == 0 {
			return nil
		}
		go func() {
			result, err := interop.RequestWorkerDecoded[markdownRenderBatchRequest, struct{}, markdownRenderBatchResult](
				context.Background(),
				*worker,
				backgroundWorkerRequestRenderMarkdownBatch,
				markdownRenderBatchRequest{Sources: sources},
				nil,
			)
			if err != nil {
				for _, source := range sources {
					delete(inFlight, source)
					cacheRenderedMarkdown(source, renderMarkdownSync(source))
				}
				app.Dispatch(appAction{Type: appActionSetMarkdownWorkerFallback, MarkdownWorkerFallback: true})
				markdownRenderVersion.Set(markdownRenderVersion.Get() + 1)
				chatLog.Warn("markdown worker batch request failed; using main-thread fallback", logging.Fields{"error": err, "sources": len(sources)})
				return
			}
			for _, rendered := range result.Results {
				delete(inFlight, rendered.Source)
				cacheRenderedMarkdown(rendered.Source, rendered.HTML)
			}
			for _, source := range sources {
				delete(inFlight, source)
			}
			markdownRenderVersion.Set(markdownRenderVersion.Get() + 1)
		}()
		return nil
	}, app.Get().MarkdownWorkerFallback, completedMarkdownSignature)
}

func monitorGRPCConnection(
	ctx context.Context,
	app ui.Reducer[appState, appAction],
	chatClientRef ui.Ref[chatpb.ChatServiceClient],
	markdownWorkerRef ui.Ref[*interop.Worker],
	conn *grpc.ClientConn,
	wakeCh <-chan string,
	sleepCh <-chan string,
	reconnectCh <-chan string,
) grpcMonitorResult {
	ticker := time.NewTicker(grpcStatePoll)
	defer ticker.Stop()

	lastState := connectivity.Idle
	notReadySince := time.Now()
	applyState := func(state connectivity.State, reason string) {
		if state == lastState {
			return
		}
		lastState = state
		chatLog.Info("grpc state", logging.Fields{"state": state.String(), "reason": reason})
	}

	for {
		state := conn.GetState()
		applyState(state, "poll")
		switch state {
		case connectivity.Ready:
			notReadySince = time.Time{}
			syncGRPCReadyState(app, chatClientRef, markdownWorkerRef, conn, true, "bridge ready", nil, nil)
		case connectivity.Idle:
			notReadySince = time.Time{}
			syncGRPCReadyState(app, chatClientRef, markdownWorkerRef, conn, true, "bridge idle", nil, nil)
		case connectivity.Connecting:
			syncGRPCReadyState(app, chatClientRef, markdownWorkerRef, conn, true, "bridge reconnecting", nil, nil)
			if notReadySince.IsZero() {
				notReadySince = time.Now()
			}
		case connectivity.TransientFailure, connectivity.Shutdown:
			return grpcMonitorResult{Reason: "bridge " + state.String()}
		default:
			syncGRPCReadyState(app, chatClientRef, markdownWorkerRef, conn, true, "bridge unavailable", nil, nil)
			if notReadySince.IsZero() {
				notReadySince = time.Now()
			}
		}
		if !notReadySince.IsZero() && time.Since(notReadySince) >= grpcStallLimit {
			return grpcMonitorResult{Reason: "bridge stalled in " + state.String()}
		}

		select {
		case <-ctx.Done():
			return grpcMonitorResult{Reason: "runtime canceled"}
		case reason := <-sleepCh:
			return grpcMonitorResult{Reason: reason, Sleep: true}
		case reason := <-reconnectCh:
			return grpcMonitorResult{Reason: reason, Immediate: true}
		case <-wakeCh:
			if runtimeNavigatorOnline() {
				conn.Connect()
			}
		case <-ticker.C:
		}
	}
}

func syncGRPCReadyState(
	app ui.Reducer[appState, appAction],
	chatClientRef ui.Ref[chatpb.ChatServiceClient],
	markdownWorkerRef ui.Ref[*interop.Worker],
	conn *grpc.ClientConn,
	ready bool,
	reason string,
	onConversationRefresh func(bool),
	onProfileRefresh func(bool),
) {
	if ready && conn != nil {
		if app.Get().GRPCReady && chatClientRef.Get() != nil {
			return
		}
		chatClientRef.Set(chatpb.NewChatServiceClient(conn))
		app.Dispatch(appAction{Type: appActionSetGRPCReady, GRPCReady: true})
		postBackgroundWorkerTicker(markdownWorkerRef, backgroundWorkerCommandStartTicker)
		if onConversationRefresh != nil {
			go onConversationRefresh(true)
		}
		if onProfileRefresh != nil {
			go onProfileRefresh(true)
		}
		chatLog.Info("grpc ready", logging.Fields{"endpoint": grpcEndpoint, "reason": reason})
		return
	}
	if !app.Get().GRPCReady && chatClientRef.Get() == nil {
		return
	}
	chatClientRef.Set(nil)
	app.Dispatch(appAction{Type: appActionSetGRPCReady, GRPCReady: false})
	postBackgroundWorkerTicker(markdownWorkerRef, backgroundWorkerCommandStopTicker)
	chatLog.Warn("grpc unavailable", logging.Fields{"reason": reason})
}

func postBackgroundWorkerTicker(markdownWorkerRef ui.Ref[*interop.Worker], command string) {
	worker := markdownWorkerRef.Get()
	if worker == nil {
		return
	}
	payload := map[string]any{}
	if command == backgroundWorkerCommandStartTicker {
		payload["intervalMs"] = int64(bgRefreshInterval / time.Millisecond)
	}
	if err := worker.Post(interop.WorkerMessage{Phase: "message", Name: command, Payload: payload}); err != nil {
		chatLog.Warn("background worker ticker command failed", logging.Fields{"error": err, "command": command})
	}
}

func waitForWakeSignal(ctx context.Context, wakeCh <-chan string) (string, bool) {
	for {
		select {
		case <-ctx.Done():
			return "", false
		case reason := <-wakeCh:
			if runtimeNavigatorOnline() {
				return reason, true
			}
		}
	}
}

func waitForReconnectDelay(ctx context.Context, delay time.Duration, wakeCh <-chan string, sleepCh <-chan string, reconnectCh <-chan string) string {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return "done"
	case <-timer.C:
		return "timer"
	case <-sleepCh:
		return "sleep"
	case <-wakeCh:
		return "wake"
	case <-reconnectCh:
		return "reconnect"
	}
}

func grpcReconnectDelay(attempt int) time.Duration {
	delay := grpcBackoffBase
	for i := 0; i < attempt; i++ {
		delay *= 2
		if delay >= grpcBackoffMax {
			return grpcBackoffMax
		}
	}
	if delay > grpcBackoffMax {
		return grpcBackoffMax
	}
	return delay
}

func runtimeNavigatorOnline() bool {
	window := js.Global().Get("window")
	if !window.Truthy() {
		return true
	}
	navigator := window.Get("navigator")
	if !navigator.Truthy() {
		return true
	}
	onLine := navigator.Get("onLine")
	if onLine.Type() == js.TypeBoolean {
		return onLine.Bool()
	}
	return true
}

func runtimePageVisible() bool {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return true
	}
	return document.Get("visibilityState").String() != "hidden"
}

func attachWindowListener(window js.Value, eventName string, handler func()) func() {
	if !window.Truthy() || window.Get("addEventListener").Type() != js.TypeFunction {
		return func() {}
	}
	listener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler()
		return nil
	})
	window.Call("addEventListener", eventName, listener)
	return func() {
		window.Call("removeEventListener", eventName, listener)
		listener.Release()
	}
}

func attachDocumentListener(document js.Value, eventName string, handler func()) func() {
	if !document.Truthy() || document.Get("addEventListener").Type() != js.TypeFunction {
		return func() {}
	}
	listener := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler()
		return nil
	})
	document.Call("addEventListener", eventName, listener)
	return func() {
		document.Call("removeEventListener", eventName, listener)
		listener.Release()
	}
}

func authContextWithMetadata(ctx context.Context) context.Context {
	token := loadPersistedAuthToken()
	if token == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, authMetadataKey, "Bearer "+token)
}
