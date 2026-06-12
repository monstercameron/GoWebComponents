package runtime

import (
	"testing"
	"time"
)

func TestRenderToStringAsyncBoundarySuspensionFallback(parseT *testing.T) {
	parseDone := make(chan struct{})
	parseChild := func() *Element {
		SuspendUntil(parseDone, "load profile")
		return CreateElement("span", nil, "ready")
	}

	parseMarkup, parseErr := RenderToString(CreateElement(AsyncBoundaryNodeType, map[string]any{
		"fallback": CreateElement("p", nil, "loading"),
	}, CreateElement(parseChild, nil)))
	if parseErr != nil {
		parseT.Fatalf("unexpected async boundary render error: %v", parseErr)
	}
	if parseMarkup != `<p>loading</p>` {
		parseT.Fatalf("expected suspension fallback markup, got %q", parseMarkup)
	}
}

func TestAsyncBoundaryRetriesAfterSuspensionResolves(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseDone := make(chan struct{})
	isReady := false

	parseChild := func() *Element {
		if !isReady {
			SuspendUntil(parseDone, "load profile")
		}
		return CreateElement("span", map[string]any{"id": "ready"}, "ready")
	}

	parseRt.Render(CreateElement(AsyncBoundaryNodeType, map[string]any{
		"fallback": CreateElement("p", map[string]any{"id": "fallback"}, "loading"),
	}, CreateElement(parseChild, nil)), parseContainer)
	flushScheduledWork(parseScheduler)

	if parseGot := nodeTextContent(parseContainer); parseGot != "loading" {
		parseT.Fatalf("expected initial fallback text, got %q", parseGot)
	}

	isReady = true
	close(parseDone)
	waitForAsyncBoundaryRetry(parseT, parseScheduler)
	flushScheduledWork(parseScheduler)

	if parseGot2 := nodeTextContent(parseContainer); parseGot2 != "ready" {
		parseT.Fatalf("expected ready text after suspension resolved, got %q", parseGot2)
	}
}

func TestSuspendUntilResolvedChannelDoesNotPanic(parseT *testing.T) {
	parseDone := make(chan struct{})
	close(parseDone)

	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseT.Fatalf("SuspendUntil panicked for resolved channel: %v", parseRecovered)
		}
	}()
	SuspendUntil(parseDone, "already ready")
}

func waitForAsyncBoundaryRetry(parseT *testing.T, parseScheduler *testScheduler) {
	parseT.Helper()
	parseDeadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(parseDeadline) {
		if len(parseScheduler.timeouts) > 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	parseT.Fatal("timed out waiting for async boundary retry")
}
