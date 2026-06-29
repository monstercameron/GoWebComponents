//go:build js && wasm

package main

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/internal/markdownrender"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared/renderworker"
	"github.com/monstercameron/GoWebComponents/v4/interop"
	"github.com/monstercameron/GoWebComponents/v4/logging"
)

const backgroundWorkerRequestRenderMarkdown = "render-markdown"
const backgroundWorkerRequestRenderMarkdownBatch = "render-markdown-batch"
const backgroundWorkerRequestRenderMessageMetadataBatch = "render-message-metadata-batch"
const backgroundWorkerRequestRenderThreadCostSummary = "render-thread-cost-summary"
const backgroundWorkerRequestRenderSignatures = "render-signatures"
const backgroundWorkerCommandStartTicker = "start-ticker"
const backgroundWorkerCommandStopTicker = "stop-ticker"
const backgroundWorkerEventTick = "tick"

type markdownRenderRequest struct {
	GetSourceBytes []byte `json:"sourceBytes"`
}

type markdownRenderResult struct {
	GetSourceBytes []byte `json:"sourceBytes"`
	GetHTMLBytes   []byte `json:"htmlBytes"`
}

type markdownRenderBatchRequest struct {
	GetSourceBytesList [][]byte `json:"sourceBytesList"`
}

type markdownRenderBatchResult struct {
	Results []markdownRenderResult `json:"results"`
}

type tickerCommand struct {
	IntervalMs int64 `json:"intervalMs"`
}

var tickerState struct {
	mu     sync.Mutex
	stopCh chan struct{}
}

var maintenanceState struct {
	mu     sync.Mutex
	stopCh chan struct{}
}

var logBackgroundWorker = logging.New("background-worker")

func main() {
	parseScope, parseErr := interop.GetWorkerScope()
	if parseErr != nil {
		parseFailBackgroundWorkerBootstrap("scope", parseErr)
	}
	parseDispatcher := buildBackgroundWorkerDispatcher()
	if _, parseErr2 := parseScope.Subscribe(func(parseMessage interop.WorkerMessage, parseMessageErr error) {
		if parseMessageErr != nil {
			logBackgroundWorker.Warn("worker message receive failed", logging.Fields{"stage": "subscribe", "error": parseMessageErr})
			return
		}
		go handleMessage(parseScope, parseDispatcher, parseMessage)
	}); parseErr2 != nil {
		parseFailBackgroundWorkerBootstrap("subscribe", parseErr2)
	}
	if parseErr3 := parseScope.Ready("bootstrap"); parseErr3 != nil {
		parseFailBackgroundWorkerBootstrap("ready", parseErr3)
	}
	select {}
}

// parseFailBackgroundWorkerBootstrap logs one bootstrap failure before stopping the worker.
func parseFailBackgroundWorkerBootstrap(parseStage string, parseErr error) {
	logBackgroundWorker.Error("worker bootstrap failed", logging.Fields{
		"stage": strings.TrimSpace(parseStage),
		"error": parseErr,
	})
	panic(parseErr)
}

func handleMessage(parseScope interop.WorkerScope, parseDispatcher *renderworker.RenderWorkerDispatcher, parseMessage interop.WorkerMessage) {
	switch strings.TrimSpace(parseMessage.Phase) {
	case "message":
		handleCommand(parseScope, parseMessage)
	case "request":
		if parseDispatcher != nil {
			parseDispatcher.HandleRenderWorkerMessage(parseScope, parseMessage)
		}
	}
}

func handleCommand(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
	switch strings.TrimSpace(parseMessage.Name) {
	case backgroundWorkerCommandStartTicker:
		var parseCommand tickerCommand
		if parseErr := interop.Decode(parseMessage.Payload, &parseCommand); parseErr != nil {
			logBackgroundWorker.Warn("worker command decode failed", logging.Fields{
				"command": backgroundWorkerCommandStartTicker,
				"error":   parseErr,
			})
			return
		}
		parseStartTicker(parseScope, parseCommand.IntervalMs)
	case backgroundWorkerCommandStopTicker:
		parseStopTicker()
	case backgroundWorkerCommandStartMaintenanceLoop:
		var parseCommand backgroundWorkerMaintenanceCommand
		if parseErr := interop.Decode(parseMessage.Payload, &parseCommand); parseErr != nil {
			logBackgroundWorker.Warn("worker command decode failed", logging.Fields{
				"command": backgroundWorkerCommandStartMaintenanceLoop,
				"error":   parseErr,
			})
			return
		}
		parseStartMaintenanceLoop(parseScope, parseCommand)
	case backgroundWorkerCommandStopMaintenanceLoop:
		parseStopMaintenanceLoop()
	}
}

func renderMarkdown(parseSource string) (string, error) {
	return markdownrender.Render(parseSource)
}

// buildBackgroundWorkerDispatcher builds one extensible request dispatcher for background-worker render functions.
func buildBackgroundWorkerDispatcher() *renderworker.RenderWorkerDispatcher {
	parseDispatcher := renderworker.BuildRenderWorkerDispatcher(backgroundWorkerRequestRenderMarkdown)
	parseDecodedOptions := renderworker.DecodedHandlerOptions{
		ShouldRecoverPanic: true,
	}
	if parseErr := renderworker.SetRenderWorkerDecodedHandler(
		parseDispatcher,
		backgroundWorkerRequestRenderMarkdown,
		handleRenderMarkdownRequest,
		parseDecodedOptions,
	); parseErr != nil {
		parseFailBackgroundWorkerBootstrap(backgroundWorkerRequestRenderMarkdown, parseErr)
	}
	if parseErr := renderworker.SetRenderWorkerDecodedHandler(
		parseDispatcher,
		backgroundWorkerRequestRenderMarkdownBatch,
		handleRenderMarkdownBatchRequest,
		parseDecodedOptions,
	); parseErr != nil {
		parseFailBackgroundWorkerBootstrap(backgroundWorkerRequestRenderMarkdownBatch, parseErr)
	}
	if parseErr := renderworker.SetRenderWorkerDecodedHandler(
		parseDispatcher,
		backgroundWorkerRequestRenderMessageMetadataBatch,
		handleRenderMessageMetadataBatchRequest,
		parseDecodedOptions,
	); parseErr != nil {
		parseFailBackgroundWorkerBootstrap(backgroundWorkerRequestRenderMessageMetadataBatch, parseErr)
	}
	if parseErr := renderworker.SetRenderWorkerDecodedHandler(
		parseDispatcher,
		backgroundWorkerRequestRenderThreadCostSummary,
		handleRenderThreadCostSummaryRequest,
		parseDecodedOptions,
	); parseErr != nil {
		parseFailBackgroundWorkerBootstrap(backgroundWorkerRequestRenderThreadCostSummary, parseErr)
	}
	if parseErr := renderworker.SetRenderWorkerDecodedHandler(
		parseDispatcher,
		backgroundWorkerRequestRenderSignatures,
		handleRenderSignaturesRequest,
		parseDecodedOptions,
	); parseErr != nil {
		parseFailBackgroundWorkerBootstrap(backgroundWorkerRequestRenderSignatures, parseErr)
	}
	if parseErr := setBackgroundWorkerCustomHandlers(parseDispatcher); parseErr != nil {
		parseFailBackgroundWorkerBootstrap("custom-handlers", parseErr)
	}
	return parseDispatcher
}

// handleRenderMarkdownRequest renders one markdown source payload into HTML.
func handleRenderMarkdownRequest(parseCtx context.Context, parseRequest markdownRenderRequest) (markdownRenderResult, error) {
	_ = parseCtx
	parseSource := string(parseRequest.GetSourceBytes)
	parseHTML, parseErr := renderMarkdown(parseSource)
	if parseErr != nil {
		return markdownRenderResult{GetSourceBytes: append([]byte(nil), parseRequest.GetSourceBytes...)}, parseErr
	}
	return markdownRenderResult{
		GetSourceBytes: append([]byte(nil), parseRequest.GetSourceBytes...),
		GetHTMLBytes:   []byte(parseHTML),
	}, nil
}

// handleRenderMarkdownBatchRequest renders one markdown batch payload into HTML results.
func handleRenderMarkdownBatchRequest(parseCtx context.Context, parseRequest markdownRenderBatchRequest) (markdownRenderBatchResult, error) {
	_ = parseCtx
	if len(parseRequest.GetSourceBytesList) == 0 {
		return markdownRenderBatchResult{}, nil
	}
	parseResults := make([]markdownRenderResult, 0, len(parseRequest.GetSourceBytesList))
	for _, parseSourceBytes := range parseRequest.GetSourceBytesList {
		parseSource := string(parseSourceBytes)
		parseHTML, parseErr := renderMarkdown(parseSource)
		if parseErr != nil {
			return markdownRenderBatchResult{Results: parseResults}, parseErr
		}
		parseResults = append(parseResults, markdownRenderResult{
			GetSourceBytes: append([]byte(nil), parseSourceBytes...),
			GetHTMLBytes:   []byte(parseHTML),
		})
	}
	return markdownRenderBatchResult{Results: parseResults}, nil
}

func parseStartTicker(parseScope interop.WorkerScope, parseIntervalMs int64) {
	if parseIntervalMs <= 0 {
		parseIntervalMs = 60000
	}
	parseStopTicker()
	parseStopCh := make(chan struct{})
	tickerState.mu.Lock()
	tickerState.stopCh = parseStopCh
	tickerState.mu.Unlock()
	go func() {
		parseTicker := time.NewTicker(time.Duration(parseIntervalMs) * time.Millisecond)
		defer parseTicker.Stop()
		for {
			select {
			case <-parseTicker.C:
				_ = parseScope.Message(backgroundWorkerEventTick, nil)
			case <-parseStopCh:
				return
			}
		}
	}()
}

func parseStopTicker() {
	tickerState.mu.Lock()
	parseStopCh := tickerState.stopCh
	tickerState.stopCh = nil
	tickerState.mu.Unlock()
	if parseStopCh != nil {
		close(parseStopCh)
	}
}

// parseStartMaintenanceLoop starts one background maintenance loop for cache/outbox upkeep tasks.
func parseStartMaintenanceLoop(parseScope interop.WorkerScope, parseCommand backgroundWorkerMaintenanceCommand) {
	parseCommand = parseNormalizeBackgroundWorkerMaintenanceCommand(parseCommand)
	parseStopMaintenanceLoop()
	parseStopCh := make(chan struct{})
	maintenanceState.mu.Lock()
	maintenanceState.stopCh = parseStopCh
	maintenanceState.mu.Unlock()
	go func() {
		parseTicker := time.NewTicker(time.Duration(parseCommand.IntervalMs) * time.Millisecond)
		defer parseTicker.Stop()
		for {
			select {
			case <-parseTicker.C:
				parseBatch := parseBuildBackgroundWorkerMaintenanceTaskBatch(parseCommand, time.Now().UTC())
				if parseErr := parseScope.Message(backgroundWorkerEventMaintenanceBatch, parseBatch); parseErr != nil {
					logBackgroundWorker.Warn("worker maintenance batch emit failed", logging.Fields{
						"event": backgroundWorkerEventMaintenanceBatch,
						"error": parseErr,
					})
				}
			case <-parseStopCh:
				return
			}
		}
	}()
}

// parseStopMaintenanceLoop stops one active background maintenance loop.
func parseStopMaintenanceLoop() {
	maintenanceState.mu.Lock()
	parseStopCh := maintenanceState.stopCh
	maintenanceState.stopCh = nil
	maintenanceState.mu.Unlock()
	if parseStopCh != nil {
		close(parseStopCh)
	}
}
