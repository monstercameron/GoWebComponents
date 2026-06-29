// Package agentbridge implements the live-session agent control surface for
// GWC apps: the gwc.agentbridge wire protocol spoken between a running
// js/wasm app (which dials out to the agent hub) and the hub that relays
// commands from MCP tools.
//
// This file set is layered like hotreload: the protocol and any other pure-Go
// logic are tag-free so they build, vet, and unit-test on both native and
// js/wasm targets; browser-coupled transport lives in _wasm.go files with
// _native.go stubs.
//
// Design rule for every command this protocol carries: mutations target the
// inputs the fiber tree is derived from (atoms, hook slots, events, route,
// mount roots) - never fibers directly, which the reconciler would overwrite.
package agentbridge
