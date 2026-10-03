package runtime

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// buildSSRSuspendingErrorBoundary wraps a suspending component in an error
// boundary that sits inside an async boundary.
func buildSSRSuspendingErrorBoundary(parseDone <-chan struct{}) *Element {
	parseChild := CreateElement(func() *Element {
		SuspendUntil(parseDone, "inner data")
		return CreateElement("strong", nil, "loaded")
	}, nil)
	parseGuard := CreateElement(NewErrorBoundaryType(), map[string]any{
		"fallback": CreateElement("em", nil, "ERROR"),
	}, parseChild)
	return CreateElement(AsyncBoundaryNodeType, map[string]any{
		"content":  parseGuard,
		"fallback": CreateElement("span", nil, "loading"),
	})
}

// TestRenderToStringErrorBoundaryDoesNotSwallowSuspension pins that a render
// suspension crosses an error boundary to the async boundary, as it does on the client.
func TestRenderToStringErrorBoundaryDoesNotSwallowSuspension(parseT *testing.T) {
	parseOut, parseErr := RenderToString(buildSSRSuspendingErrorBoundary(make(chan struct{})))
	if parseErr != nil {
		parseT.Fatalf("RenderToString: %v", parseErr)
	}
	if parseOut != "<span>loading</span>" {
		parseT.Fatalf("suspension was caught by the error boundary: %q", parseOut)
	}
}

// TestRenderToStreamErrorBoundaryDoesNotSwallowSuspension is the streaming twin:
// the boundary must stream its content later instead of baking in the error fallback.
func TestRenderToStreamErrorBoundaryDoesNotSwallowSuspension(parseT *testing.T) {
	parseDone := make(chan struct{})
	var parseBuffer bytes.Buffer
	parseChunks := make(chan SSRStreamChunk, 4)
	parseErrs := make(chan error, 1)
	go func() {
		parseErrs <- RenderToStream(context.Background(), &parseBuffer, buildSSRSuspendingErrorBoundary(parseDone), SSRStreamOptions{
			OnChunk: func(parseChunk SSRStreamChunk) { parseChunks <- parseChunk },
		})
	}()
	parseShell := receiveSSRStreamTestChunk(parseT, parseChunks)
	if strings.Contains(parseShell.HTML, "ERROR") || !strings.Contains(parseShell.HTML, "loading") {
		parseT.Fatalf("shell must show the async fallback, got %q", parseShell.HTML)
	}
	close(parseDone)
	parseBoundary := receiveSSRStreamTestChunk(parseT, parseChunks)
	if !strings.Contains(parseBoundary.HTML, "<strong>loaded</strong>") {
		parseT.Fatalf("boundary chunk missing resolved content: %q", parseBoundary.HTML)
	}
	if parseErr := receiveSSRStreamTestError(parseT, parseErrs); parseErr != nil {
		parseT.Fatalf("RenderToStream: %v", parseErr)
	}
}

// TestRenderToStreamBoundaryIDPrefixCannotBreakOutOfComment pins that a hostile
// BoundaryIDPrefix cannot terminate the marker comment and inject markup.
func TestRenderToStreamBoundaryIDPrefixCannotBreakOutOfComment(parseT *testing.T) {
	parseDone := make(chan struct{})
	close(parseDone)
	parseRoot := ssrStreamTestBoundary(make(chan struct{}), "loaded", "loading")
	var parseOut strings.Builder
	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseCancel()
	_ = RenderToStream(parseCtx, &parseOut, parseRoot, SSRStreamOptions{BoundaryIDPrefix: `x--><img src=x onerror=alert(1)><!--`})
	if strings.Contains(parseOut.String(), "<img") {
		parseT.Fatalf("hostile boundary prefix injected markup: %s", parseOut.String())
	}
}
