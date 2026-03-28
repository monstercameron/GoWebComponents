//go:build !js || !wasm

package cachecore

import "errors"

// BrowserStorage is unavailable outside js/wasm builds.
type BrowserStorage struct{}

// BuildBrowserStorage returns one unsupported error outside browser runtimes.
func BuildBrowserStorage() (*BrowserStorage, error) {
	return nil, errors.New("cachecore browser storage: only available for js/wasm builds")
}
