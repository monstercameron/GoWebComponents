package ui

import (
	"context"
	"io"
	"time"

	"github.com/monstercameron/GoWebComponents/v7/internal/runtime"
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

func renderToStreamObserved(ctx context.Context, writer io.Writer, root Node, observability SSRObservabilityOptions, options SSRStreamOptions) error {
	if !hasSSRObservers(observability) {
		return runtime.RenderToStream(ctx, writer, root, options)
	}
	start := time.Now()
	err := runtime.RenderToStream(ctx, writer, root, options)
	dispatchSSRObservation(observability, newSSRRenderObservation(observability, time.Since(start), err))
	return err
}
