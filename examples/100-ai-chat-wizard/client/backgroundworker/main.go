//go:build js && wasm

package main

import (
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/internal/markdownrender"
	"github.com/monstercameron/GoWebComponents/interop"
)

const backgroundWorkerRequestRenderMarkdown = "render-markdown"
const backgroundWorkerRequestRenderMarkdownBatch = "render-markdown-batch"
const backgroundWorkerCommandStartTicker = "start-ticker"
const backgroundWorkerCommandStopTicker = "stop-ticker"
const backgroundWorkerEventTick = "tick"

type markdownRenderRequest struct {
	Source string `json:"source"`
}

type markdownRenderResult struct {
	Source string `json:"source"`
	HTML   string `json:"html"`
}

type markdownRenderBatchRequest struct {
	Sources []string `json:"sources"`
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

func main() {
	scope, err := interop.CurrentWorkerScope()
	if err != nil {
		panic(err)
	}
	if _, err := scope.Subscribe(func(message interop.WorkerMessage, messageErr error) {
		if messageErr != nil {
			return
		}
		go handleMessage(scope, message)
	}); err != nil {
		panic(err)
	}
	if err := scope.Ready("bootstrap"); err != nil {
		panic(err)
	}
	select {}
}

func handleMessage(scope interop.WorkerScope, message interop.WorkerMessage) {
	switch strings.TrimSpace(message.Phase) {
	case "message":
		handleCommand(scope, message)
	case "request":
		switch strings.TrimSpace(message.Name) {
		case backgroundWorkerRequestRenderMarkdown:
			handleRenderMarkdown(scope, message)
		case backgroundWorkerRequestRenderMarkdownBatch:
			handleRenderMarkdownBatch(scope, message)
		}
	}
}

func handleCommand(scope interop.WorkerScope, message interop.WorkerMessage) {
	switch strings.TrimSpace(message.Name) {
	case backgroundWorkerCommandStartTicker:
		var command tickerCommand
		if err := interop.Decode(message.Payload, &command); err != nil {
			return
		}
		startTicker(scope, command.IntervalMs)
	case backgroundWorkerCommandStopTicker:
		stopTicker()
	}
}

func handleRenderMarkdown(scope interop.WorkerScope, message interop.WorkerMessage) {
	var request markdownRenderRequest
	if err := interop.Decode(message.Payload, &request); err != nil {
		_ = scope.Error(message.ID, backgroundWorkerRequestRenderMarkdown, err.Error(), markdownRenderResult{Source: request.Source})
		return
	}
	html, err := renderMarkdown(request.Source)
	if err != nil {
		_ = scope.Error(message.ID, backgroundWorkerRequestRenderMarkdown, err.Error(), markdownRenderResult{Source: request.Source})
		return
	}
	_ = scope.Result(message.ID, backgroundWorkerRequestRenderMarkdown, markdownRenderResult{
		Source: request.Source,
		HTML:   html,
	})
}

func handleRenderMarkdownBatch(scope interop.WorkerScope, message interop.WorkerMessage) {
	var request markdownRenderBatchRequest
	if err := interop.Decode(message.Payload, &request); err != nil {
		_ = scope.Error(message.ID, backgroundWorkerRequestRenderMarkdownBatch, err.Error(), markdownRenderBatchResult{})
		return
	}
	if len(request.Sources) == 0 {
		_ = scope.Result(message.ID, backgroundWorkerRequestRenderMarkdownBatch, markdownRenderBatchResult{})
		return
	}
	results := make([]markdownRenderResult, 0, len(request.Sources))
	for _, source := range request.Sources {
		html, err := renderMarkdown(source)
		if err != nil {
			_ = scope.Error(message.ID, backgroundWorkerRequestRenderMarkdownBatch, err.Error(), markdownRenderBatchResult{Results: results})
			return
		}
		results = append(results, markdownRenderResult{
			Source: source,
			HTML:   html,
		})
	}
	_ = scope.Result(message.ID, backgroundWorkerRequestRenderMarkdownBatch, markdownRenderBatchResult{
		Results: results,
	})
}

func renderMarkdown(source string) (string, error) {
	return markdownrender.Render(source)
}

func startTicker(scope interop.WorkerScope, intervalMs int64) {
	if intervalMs <= 0 {
		intervalMs = 60000
	}
	stopTicker()
	stopCh := make(chan struct{})
	tickerState.mu.Lock()
	tickerState.stopCh = stopCh
	tickerState.mu.Unlock()
	go func() {
		ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = scope.Message(backgroundWorkerEventTick, nil)
			case <-stopCh:
				return
			}
		}
	}()
}

func stopTicker() {
	tickerState.mu.Lock()
	stopCh := tickerState.stopCh
	tickerState.stopCh = nil
	tickerState.mu.Unlock()
	if stopCh != nil {
		close(stopCh)
	}
}
