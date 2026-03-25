//go:build !js || !wasm
// +build !js !wasm

package main

func nowMillis() float64 {
	return 0
}

func writeMetric(parseElementID, parseValue string) {}
