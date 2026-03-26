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
	parseScope, parseErr := interop.GetWorkerScope()
	if parseErr != nil {
		panic(parseErr)
	}
	if _, parseErr2 := parseScope.Subscribe(func(parseMessage interop.WorkerMessage, parseMessageErr error) {
		if parseMessageErr != nil {
			return
		}
		go handleMessage(parseScope, parseMessage)
	}); parseErr2 != nil {
		panic(parseErr2)
	}
	if parseErr3 := parseScope.Ready("bootstrap"); parseErr3 != nil {
		panic(parseErr3)
	}
	select {}
}

func handleMessage(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
	switch strings.TrimSpace(parseMessage.Phase) {
	case "message":
		handleCommand(parseScope, parseMessage)
	case "request":
		switch strings.TrimSpace(parseMessage.Name) {
		case backgroundWorkerRequestRenderMarkdown:
			handleRenderMarkdown(parseScope, parseMessage)
		case backgroundWorkerRequestRenderMarkdownBatch:
			handleRenderMarkdownBatch(parseScope, parseMessage)
		}
	}
}

func handleCommand(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
	switch strings.TrimSpace(parseMessage.Name) {
	case backgroundWorkerCommandStartTicker:
		var parseCommand tickerCommand
		if parseErr := interop.Decode(parseMessage.Payload, &parseCommand); parseErr != nil {
			return
		}
		parseStartTicker(parseScope, parseCommand.IntervalMs)
	case backgroundWorkerCommandStopTicker:
		parseStopTicker()
	}
}

func handleRenderMarkdown(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
	var parseRequest markdownRenderRequest
	if parseErr := interop.Decode(parseMessage.Payload, &parseRequest); parseErr != nil {
		_ = parseScope.Error(parseMessage.ID, backgroundWorkerRequestRenderMarkdown, parseErr.Error(), markdownRenderResult{Source: parseRequest.Source})
		return
	}
	parseHtml, parseErr2 := renderMarkdown(parseRequest.Source)
	if parseErr2 != nil {
		_ = parseScope.Error(parseMessage.ID, backgroundWorkerRequestRenderMarkdown, parseErr2.Error(), markdownRenderResult{Source: parseRequest.Source})
		return
	}
	_ = parseScope.Result(parseMessage.ID, backgroundWorkerRequestRenderMarkdown, markdownRenderResult{
		Source: parseRequest.Source,
		HTML:   parseHtml,
	})
}

func handleRenderMarkdownBatch(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
	var parseRequest markdownRenderBatchRequest
	if parseErr := interop.Decode(parseMessage.Payload, &parseRequest); parseErr != nil {
		_ = parseScope.Error(parseMessage.ID, backgroundWorkerRequestRenderMarkdownBatch, parseErr.Error(), markdownRenderBatchResult{})
		return
	}
	if len(parseRequest.Sources) == 0 {
		_ = parseScope.Result(parseMessage.ID, backgroundWorkerRequestRenderMarkdownBatch, markdownRenderBatchResult{})
		return
	}
	parseResults := make([]markdownRenderResult, 0, len(parseRequest.Sources))
	for _, parseSource := range parseRequest.Sources {
		parseHtml, parseErr2 := renderMarkdown(parseSource)
		if parseErr2 != nil {
			_ = parseScope.Error(parseMessage.ID, backgroundWorkerRequestRenderMarkdownBatch, parseErr2.Error(), markdownRenderBatchResult{Results: parseResults})
			return
		}
		parseResults = append(parseResults, markdownRenderResult{
			Source: parseSource,
			HTML:   parseHtml,
		})
	}
	_ = parseScope.Result(parseMessage.ID, backgroundWorkerRequestRenderMarkdownBatch, markdownRenderBatchResult{
		Results: parseResults,
	})
}

func renderMarkdown(parseSource string) (string, error) {
	return markdownrender.Render(parseSource)
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
