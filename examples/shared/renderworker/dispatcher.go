package renderworker

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/interop"
)

const getRenderWorkerFallbackRequestName = "render-worker-request"

// Handler handles one worker request routed through RenderWorkerDispatcher.
type Handler func(interop.WorkerScope, interop.WorkerMessage)

// DecodedHandlerOptions configures safety guards for typed decoded worker handlers.
type DecodedHandlerOptions struct {
	GetRequestTimeout  time.Duration
	ShouldRecoverPanic bool
}

// RenderWorkerDispatcher routes worker request messages to extensible named handlers.
type RenderWorkerDispatcher struct {
	storeMu                 sync.RWMutex
	storeHandlerByName      map[string]Handler
	storeDefaultRequestName string
}

// BuildRenderWorkerDispatcher builds one request dispatcher with an optional default request name.
func BuildRenderWorkerDispatcher(parseDefaultRequestName string) *RenderWorkerDispatcher {
	return &RenderWorkerDispatcher{
		storeHandlerByName:      map[string]Handler{},
		storeDefaultRequestName: strings.TrimSpace(parseDefaultRequestName),
	}
}

// SetRenderWorkerHandler stores one named worker request handler for later dispatch.
func (parseDispatcher *RenderWorkerDispatcher) SetRenderWorkerHandler(parseRequestName string, parseHandler Handler) error {
	if parseDispatcher == nil {
		return fmt.Errorf("renderworker: dispatcher is required")
	}
	parseRequestName = strings.TrimSpace(parseRequestName)
	if parseRequestName == "" {
		return fmt.Errorf("renderworker: request name is required")
	}
	if parseHandler == nil {
		return fmt.Errorf("renderworker: request handler is required")
	}
	parseDispatcher.storeMu.Lock()
	defer parseDispatcher.storeMu.Unlock()
	if _, hasHandler := parseDispatcher.storeHandlerByName[parseRequestName]; hasHandler {
		return fmt.Errorf("renderworker: request handler %q is already registered", parseRequestName)
	}
	parseDispatcher.storeHandlerByName[parseRequestName] = parseHandler
	return nil
}

// HandleRenderWorkerMessage dispatches one worker message to the configured request handler.
func (parseDispatcher *RenderWorkerDispatcher) HandleRenderWorkerMessage(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
	if parseDispatcher == nil || strings.TrimSpace(parseMessage.Phase) != "request" {
		return
	}
	parseRequestName := parseDispatcher.buildRenderWorkerRequestName(parseMessage.Name)
	parseHandler := parseDispatcher.getRenderWorkerHandler(parseRequestName)
	if parseHandler == nil {
		_ = parseScope.Error(parseMessage.ID, parseRequestName, "unknown render worker request", map[string]string{
			"request": parseRequestName,
		})
		return
	}
	parseHandler(parseScope, parseMessage)
}

// BuildRenderWorkerDecodedHandler builds one typed request handler that decodes payloads and posts typed results.
func BuildRenderWorkerDecodedHandler[Req any, Res any](parseRequestName string, parseHandle func(context.Context, Req) (Res, error)) Handler {
	return BuildRenderWorkerDecodedHandlerWithOptions(parseRequestName, parseHandle, DecodedHandlerOptions{ShouldRecoverPanic: true})
}

// BuildRenderWorkerDecodedHandlerWithOptions builds one typed request handler with configurable guard rails for user-defined code.
func BuildRenderWorkerDecodedHandlerWithOptions[Req any, Res any](parseRequestName string, parseHandle func(context.Context, Req) (Res, error), parseOptions DecodedHandlerOptions) Handler {
	parseRequestName = strings.TrimSpace(parseRequestName)
	return func(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
		parseResultRequestName := buildRenderWorkerResultRequestName(parseRequestName, parseMessage.Name)
		if parseOptions.ShouldRecoverPanic {
			defer func() {
				if parseRecovered := recover(); parseRecovered != nil {
					var parseResult Res
					parsePanicText := formatRenderWorkerPanicText(parseRecovered)
					_ = parseScope.Error(parseMessage.ID, parseResultRequestName, parsePanicText, parseResult)
				}
			}()
		}
		var parseRequest Req
		if parseErr := interop.Decode(parseMessage.Payload, &parseRequest); parseErr != nil {
			var parseResult Res
			_ = parseScope.Error(parseMessage.ID, parseResultRequestName, parseErr.Error(), parseResult)
			return
		}
		if parseHandle == nil {
			var parseResult Res
			_ = parseScope.Error(parseMessage.ID, parseResultRequestName, "render worker handler is nil", parseResult)
			return
		}
		parseRequestCtx := context.Background()
		parseCancelRequest := func() {}
		if parseOptions.GetRequestTimeout > 0 {
			parseRequestCtx, parseCancelRequest = context.WithTimeout(context.Background(), parseOptions.GetRequestTimeout)
		}
		defer parseCancelRequest()
		parseResult, parseErr := parseHandle(parseRequestCtx, parseRequest)
		if parseErr != nil {
			_ = parseScope.Error(parseMessage.ID, parseResultRequestName, parseErr.Error(), parseResult)
			return
		}
		_ = parseScope.Result(parseMessage.ID, parseResultRequestName, parseResult)
	}
}

// SetRenderWorkerDecodedHandler builds and registers one typed decoded request handler on the dispatcher.
func SetRenderWorkerDecodedHandler[Req any, Res any](parseDispatcher *RenderWorkerDispatcher, parseRequestName string, parseHandle func(context.Context, Req) (Res, error), parseOptions DecodedHandlerOptions) error {
	if parseDispatcher == nil {
		return fmt.Errorf("renderworker: dispatcher is required")
	}
	parseHandler := BuildRenderWorkerDecodedHandlerWithOptions(parseRequestName, parseHandle, parseOptions)
	return parseDispatcher.SetRenderWorkerHandler(parseRequestName, parseHandler)
}

// buildRenderWorkerRequestName resolves one request name using dispatcher default fallback semantics.
func (parseDispatcher *RenderWorkerDispatcher) buildRenderWorkerRequestName(parseRequestName string) string {
	parseRequestName = strings.TrimSpace(parseRequestName)
	if parseRequestName != "" {
		return parseRequestName
	}
	parseDispatcher.storeMu.RLock()
	defer parseDispatcher.storeMu.RUnlock()
	parseFallbackRequestName := strings.TrimSpace(parseDispatcher.storeDefaultRequestName)
	if parseFallbackRequestName != "" {
		return parseFallbackRequestName
	}
	return getRenderWorkerFallbackRequestName
}

// getRenderWorkerHandler loads one named request handler.
func (parseDispatcher *RenderWorkerDispatcher) getRenderWorkerHandler(parseRequestName string) Handler {
	parseDispatcher.storeMu.RLock()
	defer parseDispatcher.storeMu.RUnlock()
	return parseDispatcher.storeHandlerByName[parseRequestName]
}

// buildRenderWorkerResultRequestName resolves the request name that should be written back on worker result/error messages.
func buildRenderWorkerResultRequestName(parseConfiguredRequestName string, parseMessageRequestName string) string {
	parseConfiguredRequestName = strings.TrimSpace(parseConfiguredRequestName)
	if parseConfiguredRequestName != "" {
		return parseConfiguredRequestName
	}
	parseMessageRequestName = strings.TrimSpace(parseMessageRequestName)
	if parseMessageRequestName != "" {
		return parseMessageRequestName
	}
	return getRenderWorkerFallbackRequestName
}

// formatRenderWorkerPanicText formats one panic payload into a stable error string for worker responses.
func formatRenderWorkerPanicText(parseRecovered any) string {
	parseRecoveredText := strings.TrimSpace(fmt.Sprint(parseRecovered))
	if parseRecoveredText == "" {
		parseRecoveredText = "unknown panic"
	}
	parseStackText := strings.TrimSpace(string(debug.Stack()))
	if parseStackText == "" {
		return "render worker panic: " + parseRecoveredText
	}
	return "render worker panic: " + parseRecoveredText + "\n" + parseStackText
}
