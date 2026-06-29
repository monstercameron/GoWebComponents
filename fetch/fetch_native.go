//go:build !js || !wasm

package fetch

import (
	"context"
	"errors"
)

// Fetch performs an asynchronous HTTP fetch operation on non-browser builds.
func Fetch(parseUrl string, parseOptions Options) <-chan Result {
	_ = parseUrl
	_ = parseOptions
	parseCh := make(chan Result, 1)
	parseCh <- Result{Err: errors.New("fetch API unavailable in this environment")}
	return parseCh
}

// Upload performs an upload operation on non-browser builds.
func Upload(parseCtx context.Context, parseUrl string, parseOptions Options) <-chan UploadUpdate {
	_ = parseUrl
	_ = parseOptions
	parseCh := make(chan UploadUpdate, 1)
	parseErr := errors.New("XMLHttpRequest unavailable in this environment")
	if parseCtx != nil && parseCtx.Err() != nil {
		parseErr = parseCtx.Err()
	}
	parseCh <- UploadUpdate{Done: true, Result: Result{Err: parseErr}}
	close(parseCh)
	return parseCh
}
