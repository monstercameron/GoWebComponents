//go:build !js || !wasm || !gwcagent

package agentbridge

// EnableAgentBridge is a no-op stub for non-browser or non-agent builds.
// Applications call EnableAgentBridge unconditionally at startup; this stub
// compiles it out on all builds that do not have the js, wasm, and gwcagent
// build tags simultaneously active so release builds carry zero overhead.
func EnableAgentBridge() {}
