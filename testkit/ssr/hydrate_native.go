//go:build !js || !wasm
// +build !js !wasm

package ssr

import "testing"

// HydrationOptions configures the hydration smoke harness.
type HydrationOptions struct{}

// HydrationHarness wraps one hydration smoke run.
type HydrationHarness struct{}

// SmokeHydrate requires a js/wasm test environment.
func SmokeHydrate(tb testing.TB, root interface{}, options ...HydrationOptions) *HydrationHarness {
	tb.Helper()
	tb.Fatalf("testkit/ssr SmokeHydrate requires js/wasm tests; run go test with a js/wasm executor such as .\\tools\\go_js_wasm_exec.bat on Windows")
	return nil
}

// RoundTripHydrate requires a js/wasm test environment.
func RoundTripHydrate(tb testing.TB, root interface{}, options ...HydrationOptions) *HydrationHarness {
	tb.Helper()
	tb.Fatalf("testkit/ssr RoundTripHydrate requires js/wasm tests; run go test with a js/wasm executor such as .\\tools\\go_js_wasm_exec.bat on Windows")
	return nil
}
