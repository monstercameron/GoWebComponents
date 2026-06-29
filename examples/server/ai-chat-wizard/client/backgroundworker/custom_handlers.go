//go:build js && wasm

package main

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v4/examples/shared/renderworker"
)

// setBackgroundWorkerCustomHandlers registers optional user-defined request handlers for local MT experiments.
func setBackgroundWorkerCustomHandlers(parseDispatcher *renderworker.RenderWorkerDispatcher) error {
	_ = parseDispatcher
	// Example:
	// return renderworker.SetRenderWorkerDecodedHandler(
	// 	parseDispatcher,
	// 	"custom-request",
	// 	handleCustomRequest,
	// 	renderworker.DecodedHandlerOptions{
	// 		GetRequestTimeout:  2 * time.Second,
	// 		ShouldRecoverPanic: true,
	// 	},
	// )
	return nil
}

// handleCustomRequest shows the expected handler signature for custom decoded worker requests.
func handleCustomRequest(parseCtx context.Context, parseRequest struct{}) (struct{}, error) {
	_ = parseCtx
	_ = parseRequest
	return struct{}{}, nil
}
