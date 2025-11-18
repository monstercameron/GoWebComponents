//go:build !js || !wasm
// +build !js !wasm

package runtime

// GoUseFetch is a stub for non-WASM environments
func GoUseFetch(url string, options ...interface{}) (func() FetchState, func()) {
	// Return empty state and no-op refetch
	getter := func() FetchState {
		return FetchState{Data: nil, Error: "Not supported in this environment", Loading: false}
	}
	refetch := func() {}
	return getter, refetch
}
