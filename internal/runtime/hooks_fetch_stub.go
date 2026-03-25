//go:build !js || !wasm
// +build !js !wasm

package runtime

var unsupportedFetchState = FetchState{
	Data:    nil,
	Error:   "fetch API unavailable in this environment",
	Loading: false,
}

var unsupportedFetchGetter = func() FetchState {
	return unsupportedFetchState
}

var unsupportedFetchRefetch = func() {}

// GoUseFetch is a stub for non-WASM environments
func GoUseFetch(parseFetchURL string, parseFetchOptions ...interface{}) (func() FetchState, func()) {
	_ = parseFetchURL
	_ = parseFetchOptions
	return unsupportedFetchGetter, unsupportedFetchRefetch
}
