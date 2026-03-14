//go:build !js || !wasm
// +build !js !wasm

package runtime

var unsupportedFetchState = FetchState{
	Data:    nil,
	Error:   "fetch API unavailable in this environment",
	Loading: false,
}

func unsupportedFetchGetter() FetchState {
	return unsupportedFetchState
}

func unsupportedFetchRefetch() {}

// GoUseFetch is a stub for non-WASM environments
func GoUseFetch(url string, options ...interface{}) (func() FetchState, func()) {
	return unsupportedFetchGetter, unsupportedFetchRefetch
}
