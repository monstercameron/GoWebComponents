//go:build js && wasm
// +build js,wasm

package runtime2

import "syscall/js"

// DetectCapabilitySource reports runtime capability inputs for browser-backed wasm dispatch.
func DetectCapabilitySource() CapabilitySource {
	parseHasWorkerSupport := hasCapabilityGlobalFeature("Worker")
	parseHasMessagePortSupport := hasCapabilityGlobalFeature("MessageChannel")
	parseHasStructuredCloneSupport := parseHasMessagePortSupport || parseHasWorkerSupport
	parseHasBinaryTransportSupport := hasCapabilityGlobalFeature("Uint8Array") && hasCapabilityGlobalFeature("ArrayBuffer")
	parseHasSharedBufferSupport := hasCapabilityGlobalFeature("SharedArrayBuffer")
	parseHasSharedMemoryTransportSupport := parseHasSharedBufferSupport && hasCapabilityCrossOriginIsolation()
	return CapabilitySource{
		HasWorkerSupport:                parseHasWorkerSupport,
		HasMessagePortSupport:           parseHasMessagePortSupport,
		HasStructuredCloneSupport:       parseHasStructuredCloneSupport,
		HasBinaryTransportSupport:       parseHasBinaryTransportSupport,
		HasSharedBufferSupport:          parseHasSharedBufferSupport,
		HasSharedMemoryTransportSupport: parseHasSharedMemoryTransportSupport,
	}
}

// hasCapabilityGlobalFeature reports whether one global browser feature name exists.
func hasCapabilityGlobalFeature(parseName string) bool {
	if parseName == "" {
		return false
	}
	parseValue := js.Global().Get(parseName)
	return parseValue.Truthy()
}

// hasCapabilityCrossOriginIsolation reports whether the current browser environment is cross-origin isolated.
func hasCapabilityCrossOriginIsolation() bool {
	parseValue := js.Global().Get("crossOriginIsolated")
	if parseValue.Type() != js.TypeBoolean {
		return false
	}
	return parseValue.Bool()
}
