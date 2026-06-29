//go:build !js || !wasm

package runtime2

// DetectCapabilitySource reports runtime capability inputs for non-browser builds.
func DetectCapabilitySource() CapabilitySource {
	return CapabilitySource{}
}
