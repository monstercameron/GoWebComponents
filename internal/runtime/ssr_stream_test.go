package runtime

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestRenderToStreamMatchesRenderToStringWithoutSuspense(parseT *testing.T) {
	parseRoot := CreateElement("main", map[string]any{"id": "root"},
		CreateElement("h1", nil, "Hello"),
		CreateElement("p", nil, "stream"),
	)
	parseWant, parseErr := RenderToString(parseRoot)
	if parseErr != nil {
		parseT.Fatalf("RenderToString: %v", parseErr)
	}

	var parseBuffer bytes.Buffer
	var parseChunks []SSRStreamChunk
	if parseErr2 := RenderToStream(context.Background(), &parseBuffer, parseRoot, SSRStreamOptions{
		OnChunk: func(parseChunk SSRStreamChunk) {
			parseChunks = append(parseChunks, parseChunk)
		},
	}); parseErr2 != nil {
		parseT.Fatalf("RenderToStream: %v", parseErr2)
	}

	if parseBuffer.String() != parseWant {
		parseT.Fatalf("stream output diverged\nwant: %s\ngot:  %s", parseWant, parseBuffer.String())
	}
	if len(parseChunks) != 1 || parseChunks[0].Kind != SSRStreamChunkShell || parseChunks[0].HTML != parseWant {
		parseT.Fatalf("expected one shell chunk, got %+v", parseChunks)
	}
}

func TestRenderToStreamWritesShellBeforeSuspensionResolves(parseT *testing.T) {
	parseDone := make(chan struct{})
	parseRoot := CreateElement("main", nil,
		CreateElement("h1", nil, "Dashboard"),
		ssrStreamTestBoundary(parseDone, "loaded", "loading"),
	)

	parseChunks := make(chan SSRStreamChunk, 4)
	parseErrs := make(chan error, 1)
	go func() {
		var parseBuffer bytes.Buffer
		parseErrs <- RenderToStream(context.Background(), &parseBuffer, parseRoot, SSRStreamOptions{
			OnChunk: func(parseChunk SSRStreamChunk) {
				parseChunks <- parseChunk
			},
		})
	}()

	parseShell := receiveSSRStreamTestChunk(parseT, parseChunks)
	if parseShell.Kind != SSRStreamChunkShell {
		parseT.Fatalf("expected shell chunk first, got %+v", parseShell)
	}
	for _, parseExpected := range []string{
		"<h1>Dashboard</h1>",
		"loading",
		"<!--gwc-stream-boundary:gwc-stream-1:start-->",
		"<!--gwc-stream-boundary:gwc-stream-1:end-->",
	} {
		if !strings.Contains(parseShell.HTML, parseExpected) {
			parseT.Fatalf("shell missing %q: %s", parseExpected, parseShell.HTML)
		}
	}
	if strings.Contains(parseShell.HTML, "loaded") {
		parseT.Fatalf("shell must not include unresolved boundary content: %s", parseShell.HTML)
	}
	select {
	case parseErr := <-parseErrs:
		parseT.Fatalf("stream returned before suspension resolved: %v", parseErr)
	default:
	}

	close(parseDone)
	parseBoundary := receiveSSRStreamTestChunk(parseT, parseChunks)
	if parseBoundary.Kind != SSRStreamChunkBoundary || parseBoundary.BoundaryID != "gwc-stream-1" {
		parseT.Fatalf("expected boundary replacement chunk, got %+v", parseBoundary)
	}
	for _, parseExpected := range []string{`<template data-gwc-stream-boundary="gwc-stream-1">`, "<strong>loaded</strong>", "<script"} {
		if !strings.Contains(parseBoundary.HTML, parseExpected) {
			parseT.Fatalf("boundary chunk missing %q: %s", parseExpected, parseBoundary.HTML)
		}
	}

	if parseErr := receiveSSRStreamTestError(parseT, parseErrs); parseErr != nil {
		parseT.Fatalf("RenderToStream returned error: %v", parseErr)
	}
}

func TestRenderToStreamBoundaryScriptThreadsNonce(parseT *testing.T) {
	parseDone := make(chan struct{})
	parseRoot := CreateElement(AsyncBoundaryNodeType, map[string]any{
		"fallback": CreateElement("em", nil, "loading"),
		"content": CreateElement(func() *Element {
			SuspendUntil(parseDone, "nonce boundary")
			return CreateElement("strong", nil, "loaded")
		}, nil),
	})

	var parseBuffer bytes.Buffer
	parseErrs := make(chan error, 1)
	go func() {
		parseErrs <- RenderToStream(context.Background(), &parseBuffer, parseRoot, SSRStreamOptions{ScriptNonce: `nonce"42`})
	}()
	waitForStreamContains(parseT, &parseBuffer, "<em>loading</em>")
	close(parseDone)
	if parseErr := <-parseErrs; parseErr != nil {
		parseT.Fatalf("RenderToStream returned error: %v", parseErr)
	}
	parseHTML := parseBuffer.String()
	if !strings.Contains(parseHTML, `nonce="nonce&#34;42"`) {
		parseT.Fatalf("expected nonce on streamed boundary script, got %q", parseHTML)
	}
}

func waitForStreamContains(parseT *testing.T, parseBuffer *bytes.Buffer, parseNeedle string) {
	parseT.Helper()
	parseDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) {
		if strings.Contains(parseBuffer.String(), parseNeedle) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	parseT.Fatalf("timed out waiting for stream to contain %q; got %q", parseNeedle, parseBuffer.String())
}

func TestRenderToStreamFlushesBoundariesOutOfOrder(parseT *testing.T) {
	parseFirstDone := make(chan struct{})
	parseSecondDone := make(chan struct{})
	parseRoot := CreateElement("section", nil,
		ssrStreamTestBoundary(parseFirstDone, "first ready", "first loading"),
		ssrStreamTestBoundary(parseSecondDone, "second ready", "second loading"),
	)

	parseChunks := make(chan SSRStreamChunk, 8)
	parseErrs := make(chan error, 1)
	go func() {
		var parseBuffer bytes.Buffer
		parseErrs <- RenderToStream(context.Background(), &parseBuffer, parseRoot, SSRStreamOptions{
			OnChunk: func(parseChunk SSRStreamChunk) {
				parseChunks <- parseChunk
			},
		})
	}()

	parseShell := receiveSSRStreamTestChunk(parseT, parseChunks)
	if parseShell.Kind != SSRStreamChunkShell || !strings.Contains(parseShell.HTML, "first loading") || !strings.Contains(parseShell.HTML, "second loading") {
		parseT.Fatalf("expected shell with both fallbacks, got %+v", parseShell)
	}

	close(parseSecondDone)
	parseSecond := receiveSSRStreamTestChunk(parseT, parseChunks)
	if parseSecond.BoundaryID != "gwc-stream-2" || !strings.Contains(parseSecond.HTML, "second ready") {
		parseT.Fatalf("expected second boundary to flush first, got %+v", parseSecond)
	}

	close(parseFirstDone)
	parseFirst := receiveSSRStreamTestChunk(parseT, parseChunks)
	if parseFirst.BoundaryID != "gwc-stream-1" || !strings.Contains(parseFirst.HTML, "first ready") {
		parseT.Fatalf("expected first boundary to flush after second, got %+v", parseFirst)
	}

	if parseErr := receiveSSRStreamTestError(parseT, parseErrs); parseErr != nil {
		parseT.Fatalf("RenderToStream returned error: %v", parseErr)
	}
}

func TestRenderToStreamContextCancellationStopsPendingBoundary(parseT *testing.T) {
	parseDone := make(chan struct{})
	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseRoot := CreateElement("main", nil, ssrStreamTestBoundary(parseDone, "ready", "loading"))

	parseChunks := make(chan SSRStreamChunk, 4)
	parseErrs := make(chan error, 1)
	go func() {
		var parseBuffer bytes.Buffer
		parseErrs <- RenderToStream(parseCtx, &parseBuffer, parseRoot, SSRStreamOptions{
			OnChunk: func(parseChunk SSRStreamChunk) {
				parseChunks <- parseChunk
			},
		})
	}()

	parseShell := receiveSSRStreamTestChunk(parseT, parseChunks)
	if parseShell.Kind != SSRStreamChunkShell || !strings.Contains(parseShell.HTML, "loading") {
		parseT.Fatalf("expected fallback shell before cancellation, got %+v", parseShell)
	}
	parseCancel()
	if parseErr := receiveSSRStreamTestError(parseT, parseErrs); parseErr != context.Canceled {
		parseT.Fatalf("expected context cancellation, got %v", parseErr)
	}
}

func TestRenderToStreamDoesNotLeakPartialBoundaryMarkupOnSuspension(parseT *testing.T) {
	parseDone := make(chan struct{})
	parseSuspendingChild := CreateElement(func() *Element {
		SuspendUntil(parseDone, "late child")
		return CreateElement("em", nil, "late")
	}, nil)
	parseContent := CreateElement("div", nil, "before", parseSuspendingChild, "after")
	parseRoot := CreateElement(AsyncBoundaryNodeType, map[string]any{
		"content":  parseContent,
		"fallback": CreateElement("span", nil, "loading"),
	})

	parseChunks := make(chan SSRStreamChunk, 4)
	parseErrs := make(chan error, 1)
	go func() {
		var parseBuffer bytes.Buffer
		parseErrs <- RenderToStream(context.Background(), &parseBuffer, parseRoot, SSRStreamOptions{
			OnChunk: func(parseChunk SSRStreamChunk) {
				parseChunks <- parseChunk
			},
		})
	}()

	parseShell := receiveSSRStreamTestChunk(parseT, parseChunks)
	if parseShell.Kind != SSRStreamChunkShell {
		parseT.Fatalf("expected shell chunk, got %+v", parseShell)
	}
	if strings.Contains(parseShell.HTML, "before") || strings.Contains(parseShell.HTML, "after") {
		parseT.Fatalf("stream shell leaked partial boundary content: %s", parseShell.HTML)
	}
	if !strings.Contains(parseShell.HTML, "loading") {
		parseT.Fatalf("stream shell missing fallback: %s", parseShell.HTML)
	}

	close(parseDone)
	parseBoundary := receiveSSRStreamTestChunk(parseT, parseChunks)
	if !strings.Contains(parseBoundary.HTML, "before<em>late</em>after") {
		parseT.Fatalf("boundary replacement missing resolved content: %s", parseBoundary.HTML)
	}
	if parseErr := receiveSSRStreamTestError(parseT, parseErrs); parseErr != nil {
		parseT.Fatalf("RenderToStream returned error: %v", parseErr)
	}
}

func TestRenderToStreamDiscardsNestedPendingBoundaryWhenOuterSuspends(parseT *testing.T) {
	parseInnerDone := make(chan struct{})
	parseOuterDone := make(chan struct{})
	parseOuterSuspendingChild := CreateElement(func() *Element {
		SuspendUntil(parseOuterDone, "outer ready")
		return CreateElement("b", nil, "outer ready")
	}, nil)
	parseOuterContent := CreateElement("div", nil,
		ssrStreamTestBoundary(parseInnerDone, "inner ready", "inner loading"),
		parseOuterSuspendingChild,
	)
	parseRoot := CreateElement(AsyncBoundaryNodeType, map[string]any{
		"content":  parseOuterContent,
		"fallback": CreateElement("span", nil, "outer loading"),
	})

	parseCtx, parseCancel := context.WithTimeout(context.Background(), time.Second)
	defer parseCancel()
	parseChunks := make(chan SSRStreamChunk, 4)
	parseErrs := make(chan error, 1)
	go func() {
		var parseBuffer bytes.Buffer
		parseErrs <- RenderToStream(parseCtx, &parseBuffer, parseRoot, SSRStreamOptions{
			OnChunk: func(parseChunk SSRStreamChunk) {
				parseChunks <- parseChunk
			},
		})
	}()

	parseShell := receiveSSRStreamTestChunk(parseT, parseChunks)
	if parseShell.Kind != SSRStreamChunkShell {
		parseT.Fatalf("expected shell chunk, got %+v", parseShell)
	}
	for _, parseUnexpected := range []string{"inner loading", "inner ready"} {
		if strings.Contains(parseShell.HTML, parseUnexpected) {
			parseT.Fatalf("shell leaked discarded nested boundary content %q: %s", parseUnexpected, parseShell.HTML)
		}
	}
	if !strings.Contains(parseShell.HTML, "outer loading") || !strings.Contains(parseShell.HTML, "gwc-stream-1") {
		parseT.Fatalf("shell missing outer fallback boundary: %s", parseShell.HTML)
	}

	close(parseOuterDone)
	parseBoundary := receiveSSRStreamTestChunk(parseT, parseChunks)
	if parseBoundary.Kind != SSRStreamChunkBoundary || parseBoundary.BoundaryID != "gwc-stream-1" {
		parseT.Fatalf("expected outer replacement chunk only, got %+v", parseBoundary)
	}
	if !strings.Contains(parseBoundary.HTML, "outer ready") || !strings.Contains(parseBoundary.HTML, "inner loading") {
		parseT.Fatalf("outer replacement should render resolved outer content with nested fallback: %s", parseBoundary.HTML)
	}
	if parseErr := receiveSSRStreamTestError(parseT, parseErrs); parseErr != nil {
		parseT.Fatalf("RenderToStream returned error before unresolved inner boundary resolved: %v", parseErr)
	}
}

func TestRenderToStreamEdgeErrorsAndFlushFallbacks(parseT *testing.T) {
	if parseErr := RenderToStream(context.Background(), nil, CreateElement("div", nil), SSRStreamOptions{}); parseErr == nil {
		parseT.Fatal("RenderToStream should reject nil writer")
	}

	parseWriter := &ssrStreamFailingWriter{}
	parseErr := RenderToStream(nil, parseWriter, CreateElement("div", nil, "x"), SSRStreamOptions{})
	if !errors.Is(parseErr, errSSRStreamWrite) {
		parseT.Fatalf("RenderToStream writer error = %v", parseErr)
	}

	parseFlusher := &ssrStreamFlushWriter{}
	flushSSRStream(parseFlusher, SSRStreamOptions{})
	if parseFlusher.flushes != 1 {
		parseT.Fatalf("writer Flush() count = %d, want 1", parseFlusher.flushes)
	}
	parseOptionFlushes := 0
	flushSSRStream(parseFlusher, SSRStreamOptions{Flush: func() { parseOptionFlushes++ }})
	if parseOptionFlushes != 1 || parseFlusher.flushes != 1 {
		parseT.Fatalf("option Flush should take precedence, option=%d writer=%d", parseOptionFlushes, parseFlusher.flushes)
	}
}

func TestRenderToStreamShellCoversBoundaryAndElementBranches(parseT *testing.T) {
	parsePanicChild := CreateElement(func() *Element {
		panic("render failed")
	}, nil)
	parseOnErrorCalled := false
	parseRoot := CreateElement("main", nil,
		CreateElement(NewErrorBoundaryType(), map[string]any{
			"onError": func(error) { parseOnErrorCalled = true },
			"errorFallback": func(parseErr error, parseReset func()) *Element {
				return CreateElement("strong", nil, "caught:"+parseErr.Error())
			},
		}, parsePanicChild),
		CreateElement(AsyncBoundaryNodeType, map[string]any{
			"pending":  true,
			"fallback": CreateElement("em", nil, "pending"),
		}),
		CreateElement(AsyncBoundaryNodeType, map[string]any{
			"error": errors.New("async failed"),
			"errorFallback": func(parseErr error) *Element {
				return CreateElement("b", nil, "async:"+parseErr.Error())
			},
		}),
		CreateElement("br", nil, "ignored"),
		CreateElement("FRAGMENT", nil, "frag"),
		CreateElement(NewContextProviderType(NewContextDescriptor("ssr-stream-test")), nil, "ctx"),
		CreateElement(PortalNodeType, nil, "portal"),
		CreateElement(ReactiveTextNodeType, map[string]any{reactiveTextGetterProp: func() string { return "<live>" }}),
		CreateElement(ReactiveRegionNodeType, map[string]any{reactiveRegionRenderProp: func() *Element {
			return CreateElement("i", nil, "region")
		}}),
		nil,
		7,
	)

	var parseBuffer bytes.Buffer
	if parseErr := RenderToStream(context.Background(), &parseBuffer, parseRoot, SSRStreamOptions{}); parseErr != nil {
		parseT.Fatalf("RenderToStream edge shell error = %v", parseErr)
	}
	if !parseOnErrorCalled {
		parseT.Fatal("error boundary onError should run")
	}
	parseHTML := parseBuffer.String()
	for _, parseWant := range []string{
		"<strong>caught:render failed</strong>",
		"<em>pending</em>",
		"<b>async:async failed</b>",
		"<br>",
		"frag",
		"ctx",
		"portal",
		"&lt;live&gt;",
		"<i>region</i>",
		"7",
	} {
		if !strings.Contains(parseHTML, parseWant) {
			parseT.Fatalf("stream shell missing %q: %s", parseWant, parseHTML)
		}
	}
}

func TestRenderSSRStreamBoundaryChunkErrorsAndScriptOptions(parseT *testing.T) {
	parseMissingDone := renderSSRStreamBoundaryChunk(context.Background(), ssrStreamPendingBoundary{
		id:         "b1",
		content:    CreateElement("span", nil, "ready"),
		suspension: &Suspension{Reason: "missing done"},
	}, SSRStreamOptions{})
	if parseMissingDone.Err == nil || !strings.Contains(parseMissingDone.Err.Error(), "suspended without a completion channel") {
		parseT.Fatalf("missing done chunk = %+v", parseMissingDone)
	}

	parseDone := make(chan struct{})
	close(parseDone)
	parseResuspending := renderSSRStreamBoundaryChunk(context.Background(), ssrStreamPendingBoundary{
		id: "b2",
		content: CreateElement(func() *Element {
			SuspendUntil(make(chan struct{}), "again")
			return nil
		}, nil),
		suspension: &Suspension{Done: parseDone},
	}, SSRStreamOptions{})
	if parseResuspending.Err == nil || !strings.Contains(parseResuspending.Err.Error(), "suspended again after completion") {
		parseT.Fatalf("resuspending chunk = %+v", parseResuspending)
	}

	parsePatch := renderSSRStreamBoundaryPatch(`b"3`, "<span>ready</span>", false, "nonce")
	if strings.Contains(parsePatch, "<script") || !strings.Contains(parsePatch, `data-gwc-stream-boundary="b&#34;3"`) {
		parseT.Fatalf("disabled-script patch = %s", parsePatch)
	}

	parseState := &ssrStreamState{options: SSRStreamOptions{BoundaryIDPrefix: " custom- "}}
	if parseID := parseState.nextSSRStreamBoundaryID(); parseID != "custom-1" {
		parseT.Fatalf("custom boundary ID = %q", parseID)
	}
}

type ssrStreamFailingWriter struct{}

var errSSRStreamWrite = errors.New("write failed")

func (*ssrStreamFailingWriter) Write([]byte) (int, error) {
	return 0, errSSRStreamWrite
}

type ssrStreamFlushWriter struct {
	bytes.Buffer
	flushes int
}

func (parseWriter *ssrStreamFlushWriter) Flush() {
	parseWriter.flushes++
}

var _ io.Writer = (*ssrStreamFailingWriter)(nil)

func ssrStreamTestBoundary(parseDone <-chan struct{}, parseReady string, parseFallback string) *Element {
	parseContent := CreateElement(func() *Element {
		SuspendUntil(parseDone, parseReady)
		return CreateElement("strong", nil, parseReady)
	}, nil)
	return CreateElement(AsyncBoundaryNodeType, map[string]any{
		"content":  parseContent,
		"fallback": CreateElement("span", nil, parseFallback),
	})
}

func receiveSSRStreamTestChunk(parseT *testing.T, parseChunks <-chan SSRStreamChunk) SSRStreamChunk {
	parseT.Helper()
	select {
	case parseChunk := <-parseChunks:
		return parseChunk
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for SSR stream chunk")
		return SSRStreamChunk{}
	}
}

func receiveSSRStreamTestError(parseT *testing.T, parseErrs <-chan error) error {
	parseT.Helper()
	select {
	case parseErr := <-parseErrs:
		return parseErr
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for SSR stream completion")
		return nil
	}
}
