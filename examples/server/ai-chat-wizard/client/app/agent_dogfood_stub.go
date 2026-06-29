//go:build !js || !wasm || !gwcagent

package app

// parseEnableDogfoodAgentBridge is a no-op outside the dogfood agent wasm
// build.
func parseEnableDogfoodAgentBridge() {}
