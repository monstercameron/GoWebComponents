package ui

import (
	"context"
	"io"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

const (
	// SSRStreamChunkShell identifies the initial streaming SSR shell chunk.
	SSRStreamChunkShell = runtime.SSRStreamChunkShell
	// SSRStreamChunkBoundary identifies an async boundary replacement chunk.
	SSRStreamChunkBoundary = runtime.SSRStreamChunkBoundary
)

// SSRStreamChunk describes one emitted streaming SSR chunk.
type SSRStreamChunk = runtime.SSRStreamChunk

// SSRStreamOptions configures streaming SSR output.
type SSRStreamOptions = runtime.SSRStreamOptions

// RenderToStream writes an SSR shell immediately and then streams async
// boundary replacements as render-time suspensions resolve.
func RenderToStream(parseCtx context.Context, parseWriter io.Writer, parseRoot Node, parseOptions ...SSRStreamOptions) error {
	return renderToStreamObserved(parseCtx, parseWriter, parseRoot, SSRObservabilityOptions{}, firstSSRStreamOptions(parseOptions))
}

// RenderToStreamObserved streams an SSR tree and emits SSR render metrics.
func RenderToStreamObserved(parseCtx context.Context, parseWriter io.Writer, parseRoot Node, parseObservability SSRObservabilityOptions, parseOptions ...SSRStreamOptions) error {
	return renderToStreamObserved(parseCtx, parseWriter, parseRoot, parseObservability, firstSSRStreamOptions(parseOptions))
}

func firstSSRStreamOptions(parseOptions []SSRStreamOptions) SSRStreamOptions {
	if len(parseOptions) == 0 {
		return SSRStreamOptions{}
	}
	return parseOptions[0]
}

func renderToStreamObserved(parseCtx context.Context, parseWriter io.Writer, parseRoot Node, parseObservability SSRObservabilityOptions, parseOptions SSRStreamOptions) error {
	parseStart := time.Now()
	parseErr := runtime.RenderToStream(parseCtx, parseWriter, parseRoot, parseOptions)
	dispatchSSRObservation(parseObservability, newSSRRenderObservation(parseObservability, time.Since(parseStart), parseErr))
	return parseErr
}
